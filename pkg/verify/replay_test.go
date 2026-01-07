package verify

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"
)

// TestInMemoryReplayStore_Basic tests basic nonce checking
func TestInMemoryReplayStore_Basic(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)

	// Generate a nonce
	nonce := make([]byte, 16)
	rand.Read(nonce)
	now := uint64(time.Now().UnixNano())

	// First check should succeed
	ok, err := store.CheckAndStore(nonce, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("first check should succeed")
	}

	// Second check with same nonce should fail (replay)
	ok, err = store.CheckAndStore(nonce, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("second check should fail (replay detected)")
	}

	// Different nonce should succeed
	nonce2 := make([]byte, 16)
	rand.Read(nonce2)
	ok, err = store.CheckAndStore(nonce2, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("different nonce should succeed")
	}
}

// TestInMemoryReplayStore_InvalidNonceLength tests nonce length validation
func TestInMemoryReplayStore_InvalidNonceLength(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	now := uint64(time.Now().UnixNano())

	testCases := []struct {
		name   string
		length int
	}{
		{"empty", 0},
		{"too_short_1", 1},
		{"too_short_8", 8},
		{"too_short_15", 15},
		{"too_long_17", 17},
		{"too_long_32", 32},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nonce := make([]byte, tc.length)
			rand.Read(nonce)
			_, err := store.CheckAndStore(nonce, now)
			if err == nil {
				t.Errorf("expected error for nonce length %d", tc.length)
			}
		})
	}
}

// TestInMemoryReplayStore_TimestampValidation tests timestamp bounds
func TestInMemoryReplayStore_TimestampValidation(t *testing.T) {
	clockSkew := 5 * time.Minute
	retention := time.Hour
	store := NewInMemoryReplayStore(retention, clockSkew)

	nonce := make([]byte, 16)
	rand.Read(nonce)
	now := uint64(time.Now().UnixNano())

	// Too far in future
	t.Run("future_timestamp", func(t *testing.T) {
		futureNonce := make([]byte, 16)
		rand.Read(futureNonce)
		future := now + uint64(10*time.Minute.Nanoseconds())
		_, err := store.CheckAndStore(futureNonce, future)
		if err == nil {
			t.Error("expected error for future timestamp")
		}
	})

	// Too old
	t.Run("old_timestamp", func(t *testing.T) {
		oldNonce := make([]byte, 16)
		rand.Read(oldNonce)
		old := now - uint64(2*time.Hour.Nanoseconds())
		_, err := store.CheckAndStore(oldNonce, old)
		if err == nil {
			t.Error("expected error for old timestamp")
		}
	})

	// Within bounds (just inside clock skew)
	t.Run("within_skew", func(t *testing.T) {
		skewNonce := make([]byte, 16)
		rand.Read(skewNonce)
		withinSkew := now + uint64(4*time.Minute.Nanoseconds())
		ok, err := store.CheckAndStore(skewNonce, withinSkew)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("timestamp within skew should succeed")
		}
	})
}

// TestInMemoryReplayStore_Cleanup tests nonce expiration
func TestInMemoryReplayStore_Cleanup(t *testing.T) {
	// Short retention for testing
	retention := 100 * time.Millisecond
	store := NewInMemoryReplayStore(retention, time.Minute)

	nonce := make([]byte, 16)
	rand.Read(nonce)
	now := uint64(time.Now().UnixNano())

	// Store nonce
	ok, _ := store.CheckAndStore(nonce, now)
	if !ok {
		t.Fatal("first check should succeed")
	}

	// Verify it's stored
	if store.Size() != 1 {
		t.Fatalf("expected size 1, got %d", store.Size())
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Cleanup
	store.Cleanup()

	// Should be empty now
	if store.Size() != 0 {
		t.Fatalf("expected size 0 after cleanup, got %d", store.Size())
	}
}

// TestInMemoryReplayStore_Concurrent tests thread safety
func TestInMemoryReplayStore_Concurrent(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	now := uint64(time.Now().UnixNano())

	const numGoroutines = 100
	const noncesPer = 100

	var wg sync.WaitGroup
	successCount := make([]int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < noncesPer; j++ {
				nonce := make([]byte, 16)
				rand.Read(nonce)
				ok, err := store.CheckAndStore(nonce, now)
				if err == nil && ok {
					successCount[id]++
				}
			}
		}(i)
	}

	wg.Wait()

	// All unique nonces should succeed
	total := 0
	for _, c := range successCount {
		total += c
	}

	expected := numGoroutines * noncesPer
	if total != expected {
		t.Errorf("expected %d successes, got %d", expected, total)
	}

	// Store should have all nonces
	if store.Size() != expected {
		t.Errorf("expected store size %d, got %d", expected, store.Size())
	}
}

// TestInMemoryReplayStore_ReplayConcurrent tests concurrent replay detection
func TestInMemoryReplayStore_ReplayConcurrent(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	now := uint64(time.Now().UnixNano())

	// Single nonce that all goroutines will try
	sharedNonce := make([]byte, 16)
	rand.Read(sharedNonce)

	const numGoroutines = 100
	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := store.CheckAndStore(sharedNonce, now)
			if err == nil && ok {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only ONE goroutine should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 success for shared nonce, got %d", successCount)
	}
}

// TestInMemoryReplayStore_CleanupLoop tests the background cleanup
func TestInMemoryReplayStore_CleanupLoop(t *testing.T) {
	retention := 50 * time.Millisecond
	store := NewInMemoryReplayStore(retention, time.Minute)

	// Add some nonces
	for i := 0; i < 10; i++ {
		nonce := make([]byte, 16)
		rand.Read(nonce)
		store.CheckAndStore(nonce, uint64(time.Now().UnixNano()))
	}

	if store.Size() != 10 {
		t.Fatalf("expected 10 nonces, got %d", store.Size())
	}

	// Start cleanup loop with short interval
	stopCh := store.StartCleanupLoop(60 * time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Should be cleaned up
	if store.Size() != 0 {
		t.Errorf("expected 0 nonces after cleanup loop, got %d", store.Size())
	}

	// Stop the cleanup loop
	close(stopCh)
}

// BenchmarkInMemoryReplayStore_CheckAndStore benchmarks nonce checking
func BenchmarkInMemoryReplayStore_CheckAndStore(b *testing.B) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	now := uint64(time.Now().UnixNano())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nonce := make([]byte, 16)
		rand.Read(nonce)
		store.CheckAndStore(nonce, now)
	}
}

// BenchmarkInMemoryReplayStore_Concurrent benchmarks concurrent access
func BenchmarkInMemoryReplayStore_Concurrent(b *testing.B) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	now := uint64(time.Now().UnixNano())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			nonce := make([]byte, 16)
			rand.Read(nonce)
			store.CheckAndStore(nonce, now)
		}
	})
}
