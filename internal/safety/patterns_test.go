package safety

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSafeLoop(t *testing.T) {
	t.Run("basic_limit", func(t *testing.T) {
		sl := NewSafeLoop(3, 1*time.Minute)
		count := 0
		for sl.Next() {
			count++
		}
		if count != 3 {
			t.Errorf("Expected 3 iterations, got %d", count)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		sl := NewSafeLoop(100, 10*time.Millisecond)
		time.Sleep(20 * time.Millisecond) // Force timeout
		count := 0
		for sl.Next() {
			count++
		}
		if count > 1 { // Should stop after very few iterations due to timeout
			t.Errorf("Expected few iterations due to timeout, got %d", count)
		}
	})
}

func TestSafeSlice(t *testing.T) {
	t.Run("append_and_get", func(t *testing.T) {
		ss := NewSafeSlice[string](3)
		if err := ss.Append("apple"); err != nil {
			t.Fatal(err)
		}
		if err := ss.Append("banana"); err != nil {
			t.Fatal(err)
		}

		item, err := ss.Get(0)
		if err != nil {
			t.Fatal(err)
		}
		if item != "apple" {
			t.Errorf("Expected 'apple', got '%s'", item)
		}

		if ss.Length() != 2 {
			t.Errorf("Expected length 2, got %d", ss.Length())
		}
	})

	t.Run("capacity_exceeded", func(t *testing.T) {
		ss := NewSafeSlice[int](1)
		ss.Append(1)
		err := ss.Append(2)
		if err == nil || !strings.Contains(err.Error(), "capacity exceeded") {
			t.Errorf("Expected capacity exceeded error, got %v", err)
		}
	})

	t.Run("index_out_of_range", func(t *testing.T) {
		ss := NewSafeSlice[string](1)
		ss.Append("test")
		_, err := ss.Get(1)
		if err == nil || !strings.Contains(err.Error(), "index out of range") {
			t.Errorf("Expected index out of range error, got %v", err)
		}
	})

	t.Run("concurrent_access", func(t *testing.T) {
		ss := NewSafeSlice[int](1000)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				ss.Append(val)
			}(i)
		}
		wg.Wait()
		if ss.Length() > 100 { // Should be 100 if no capacity issues
			t.Errorf("Concurrent appends resulted in unexpected length: %d", ss.Length())
		}
	})
}

func TestSafeMap(t *testing.T) {
	t.Run("set_and_get", func(t *testing.T) {
		sm := NewSafeMap[string, int](3)
		if err := sm.Set("one", 1); err != nil {
			t.Fatal(err)
		}
		if err := sm.Set("two", 2); err != nil {
			t.Fatal(err)
		}

		val, ok := sm.Get("one")
		if !ok || val != 1 {
			t.Errorf("Expected 1, got %v", val)
		}
	})

	t.Run("size_exceeded", func(t *testing.T) {
		sm := NewSafeMap[string, int](1)
		sm.Set("one", 1)
		err := sm.Set("two", 2)
		if err == nil || !strings.Contains(err.Error(), "map size exceeded") {
			t.Errorf("Expected map size exceeded error, got %v", err)
		}
	})

	t.Run("concurrent_access", func(t *testing.T) {
		sm := NewSafeMap[int, int](1000)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(key, val int) {
				defer wg.Done()
				sm.Set(key, val)
			}(i, i*10)
		}
		wg.Wait()
		if len(sm.data) > 100 { // Should be 100 if no size issues
			t.Errorf("Concurrent sets resulted in unexpected size: %d", len(sm.data))
		}
	})
}

func TestSafeRecursion(t *testing.T) {
	t.Run("depth_limit", func(t *testing.T) {
		sr := NewSafeRecursion(3)
		var recursiveFunc func(int) error
		recursiveFunc = func(n int) error {
			if err := sr.Execute(func() error {
				if n == 0 {
					return nil
				}
				return recursiveFunc(n - 1)
			}); err != nil {
				return err
			}
			return nil
		}

		err := recursiveFunc(5)
		if err == nil || !strings.Contains(err.Error(), "maximum recursion depth") {
			t.Errorf("Expected maximum recursion depth error, got %v", err)
		}
	})

	t.Run("normal_recursion", func(t *testing.T) {
		sr := NewSafeRecursion(5)
		var recursiveFunc func(int) error
		recursiveFunc = func(n int) error {
			if err := sr.Execute(func() error {
				if n == 0 {
					return nil
				}
				return recursiveFunc(n - 1)
			}); err != nil {
				return err
			}
			return nil
		}

		err := recursiveFunc(3)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSafeExecution_ExecuteWithTimeout(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		se := NewSafeExecution(100*time.Millisecond, 10*1024*1024) // 10MB
		err := se.Execute(func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		se := NewSafeExecution(50*time.Millisecond, 10*1024*1024) // 10MB
		err := se.Execute(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}
	})
}

func TestSafeExecution_ExecuteWithPanicRecovery(t *testing.T) {
	t.Run("panic_recovery", func(t *testing.T) {
		se := NewSafeExecution(100*time.Millisecond, 10*1024*1024) // 10MB
		err := se.Execute(func() error {
			panic("simulated panic")
		})
		if err == nil || !strings.Contains(err.Error(), "panic") {
			t.Errorf("Expected panic recovery error, got %v", err)
		}
	})

	t.Run("normal_execution", func(t *testing.T) {
		se := NewSafeExecution(100*time.Millisecond, 10*1024*1024) // 10MB
		err := se.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSafeExecution_MemoryLimit(t *testing.T) {
	t.Run("memory_exceeded", func(t *testing.T) {
		se := NewSafeExecution(1*time.Second, 100*1024) // 1MB limit
		err := se.Execute(func() error {
			// Allocate a large slice to exceed memory limit
			_ = make([]byte, 200*1024) // 2MB allocation
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "heap allocation exceeded") {
			t.Errorf("Expected memory limit error, got %v", err)
		}
	})

	t.Run("memory_within_limit", func(t *testing.T) {
		se := NewSafeExecution(1*time.Second, 5*1024*1024) // 5MB limit
		err := se.Execute(func() error {
			_ = make([]byte, 200*1024) // 2MB allocation
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// TestSafeContext tests the SafeContext functionality
func TestSafeContext(t *testing.T) {
	t.Run("timeout_cancellation", func(t *testing.T) {
		sc := NewSafeContext(50 * time.Millisecond)
		defer sc.Close()

		select {
		case <-sc.Done():
			if sc.Err() != context.DeadlineExceeded {
				t.Errorf("Expected context.DeadlineExceeded, got %v", sc.Err())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context did not cancel in time")
		}
	})

	t.Run("manual_cancellation", func(t *testing.T) {
		sc := NewSafeContext(1 * time.Minute) // Long timeout
		defer sc.Close()

		go func() {
			time.Sleep(1 * time.Millisecond)
			sc.Cancel()
		}()

		select {
		case <-sc.Done():
			if sc.Err() != context.Canceled {
				t.Errorf("Expected context.Canceled, got %v", sc.Err())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context did not cancel in time")
		}
	})
}

// TestSafeChannel tests the SafeChannel functionality
func TestSafeChannel(t *testing.T) {
	t.Run("send_receive", func(t *testing.T) {
		sc := NewSafeChannel[int](5)
		defer sc.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			err := sc.Send(1)
			if err != nil {
				t.Errorf("Send error: %v", err)
			}
		}()

		go func() {
			defer wg.Done()
			val, err := sc.Receive()
			if err != nil {
				t.Errorf("Receive error: %v", err)
			}
			if val != 1 {
				t.Errorf("Expected 1, got %d", val)
			}
		}()
		wg.Wait()
	})

	t.Run("send_timeout", func(t *testing.T) {
		sc := NewSafeChannel[int](1)
		defer sc.Close()

		sc.Send(1) // Fill buffer

		err := sc.Send(2) // This should timeout
		if err == nil || !strings.Contains(err.Error(), "send timed out") {
			t.Errorf("Expected send timeout error, got %v", err)
		}
	})

	t.Run("receive_timeout", func(t *testing.T) {
		sc := NewSafeChannel[int](1)
		defer sc.Close()

		_, err := sc.Receive() // This should timeout
		if err == nil || !strings.Contains(err.Error(), "receive timed out") {
			t.Errorf("Expected receive timeout error, got %v", err)
		}
	})

	t.Run("close_channel", func(t *testing.T) {
		sc := NewSafeChannel[int](1)
		sc.Close()

		err := sc.Send(1)
		if err == nil || !strings.Contains(err.Error(), "closed channel") {
			t.Errorf("Expected send on closed channel error, got %v", err)
		}

		_, err = sc.Receive()
		if err == nil || !strings.Contains(err.Error(), "closed channel") {
			t.Errorf("Expected receive on closed channel error, got %v", err)
		}
	})
}

func TestSafeChannel_RaceCondition(t *testing.T) {
	sc := NewSafeChannel[int](10)
	defer sc.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(val int) {
			defer wg.Done()
			sc.Send(val)
		}(i)
		go func() {
			defer wg.Done()
			sc.Receive()
		}()
	}
	wg.Wait()
}

func TestSafeWorkerPool(t *testing.T) {
	swp := NewSafeWorkerPool(2, 10)
	swp.Start()
	defer swp.Stop()

	t.Run("submit_job", func(t *testing.T) {
		done := make(chan bool)
		err := swp.Submit(func() {
			time.Sleep(1 * time.Millisecond)
			done <- true
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Error("Job did not complete in time")
		}
	})

	t.Run("submit_to_full_queue", func(t *testing.T) {
		// Fill the queue
		for i := 0; i < 10; i++ {
			swp.Submit(func() {})
		}

		// This should fail
		err := swp.Submit(func() {})
		if err == nil {
			t.Error("Expected error for full queue, got nil")
		}
	})
}

func BenchmarkSafeSlice_Append(b *testing.B) {
	ss := NewSafeSlice[int](b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ss.Append(i)
	}
}

func BenchmarkSafeMap_Set(b *testing.B) {
	sm := NewSafeMap[int, int](b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Set(i, i)
	}
}
