package deterministicvm

import (
	"testing"
	"time"
)

// TestDeterministicRNG tests deterministic RNG functionality
func TestDeterministicRNG(t *testing.T) {
	seed := []byte("test-seed-123")
	rng1 := NewDeterministicRNG(seed)
	rng2 := NewDeterministicRNG(seed)
	
	// Test that same seed produces same sequence
	for i := 0; i < 100; i++ {
		val1 := rng1.Uint32()
		val2 := rng2.Uint32()
		
		if val1 != val2 {
			t.Errorf("Mismatch at iteration %d: %d != %d", i, val1, val2)
		}
	}
}

// TestDeterministicRNGFromExecution tests RNG from execution context
func TestDeterministicRNGFromExecution(t *testing.T) {
	executionID := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	timestamp := uint64(time.Now().Unix())
	artifactHash := []byte("artifact-hash-123")
	
	rng1 := NewDeterministicRNGFromExecution(executionID, timestamp, artifactHash)
	rng2 := NewDeterministicRNGFromExecution(executionID, timestamp, artifactHash)
	
	// Test that same context produces same sequence
	for i := 0; i < 50; i++ {
		val1 := rng1.Uint64()
		val2 := rng2.Uint64()
		
		if val1 != val2 {
			t.Errorf("Mismatch at iteration %d: %d != %d", i, val1, val2)
		}
	}
}

// TestDeterministicRNGDifferentSeeds tests that different seeds produce different sequences
func TestDeterministicRNGDifferentSeeds(t *testing.T) {
	seed1 := []byte("seed-1")
	seed2 := []byte("seed-2")
	
	rng1 := NewDeterministicRNG(seed1)
	rng2 := NewDeterministicRNG(seed2)
	
	// Test that different seeds produce different sequences
	allSame := true
	for i := 0; i < 100; i++ {
		val1 := rng1.Uint32()
		val2 := rng2.Uint32()
		
		if val1 != val2 {
			allSame = false
			break
		}
	}
	
	if allSame {
		t.Error("Different seeds produced same sequence")
	}
}

// TestDeterministicRNGTypes tests different RNG types
func TestDeterministicRNGTypes(t *testing.T) {
	seed := []byte("test-seed")
	rng := NewDeterministicRNG(seed)
	
	// Test Uint32
	val32 := rng.Uint32()
	if val32 == 0 {
		t.Error("Uint32 returned 0")
	}
	
	// Test Uint64
	val64 := rng.Uint64()
	if val64 == 0 {
		t.Error("Uint64 returned 0")
	}
	
	// Test Int32
	valInt32 := rng.Int32()
	if valInt32 == 0 {
		t.Error("Int32 returned 0")
	}
	
	// Test Int64
	valInt64 := rng.Int64()
	if valInt64 == 0 {
		t.Error("Int64 returned 0")
	}
	
	// Test Float32
	valFloat32 := rng.Float32()
	if valFloat32 < 0 || valFloat32 >= 1 {
		t.Errorf("Float32 out of range [0, 1): %f", valFloat32)
	}
	
	// Test Float64
	valFloat64 := rng.Float64()
	if valFloat64 < 0 || valFloat64 >= 1 {
		t.Errorf("Float64 out of range [0, 1): %f", valFloat64)
	}
}

// TestDeterministicRNGIntn tests Intn function
func TestDeterministicRNGIntn(t *testing.T) {
	seed := []byte("test-seed")
	rng := NewDeterministicRNG(seed)
	
	// Test Intn with various ranges
	for n := 1; n <= 100; n++ {
		val := rng.Intn(n)
		if val < 0 || val >= n {
			t.Errorf("Intn(%d) returned %d, expected [0, %d)", n, val, n)
		}
	}
	
	// Test Intn with 0
	val := rng.Intn(0)
	if val != 0 {
		t.Errorf("Intn(0) returned %d, expected 0", val)
	}
	
	// Test Intn with negative
	val = rng.Intn(-5)
	if val != 0 {
		t.Errorf("Intn(-5) returned %d, expected 0", val)
	}
}

// TestDeterministicRNGIntRange tests IntRange function
func TestDeterministicRNGIntRange(t *testing.T) {
	seed := []byte("test-seed")
	rng := NewDeterministicRNG(seed)
	
	// Test IntRange with various ranges
	for min := 0; min < 10; min++ {
		for max := min + 1; max < min + 20; max++ {
			val := rng.IntRange(min, max)
			if val < min || val >= max {
				t.Errorf("IntRange(%d, %d) returned %d, expected [%d, %d)", min, max, val, min, max)
			}
		}
	}
	
	// Test IntRange with min >= max
	val := rng.IntRange(10, 5)
	if val != 10 {
		t.Errorf("IntRange(10, 5) returned %d, expected 10", val)
	}
}

// TestDeterministicRNGShuffle tests Shuffle function
func TestDeterministicRNGShuffle(t *testing.T) {
	seed := []byte("test-seed")
	rng := NewDeterministicRNG(seed)
	
	// Test shuffle with small slice
	slice := []int{1, 2, 3, 4, 5}
	original := make([]int, len(slice))
	copy(original, slice)
	
	rng.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
	
	// Check that all elements are still present
	if len(slice) != len(original) {
		t.Error("Shuffle changed slice length")
	}
	
	// Check that elements are shuffled (very unlikely to be in same order)
	sameOrder := true
	for i := 0; i < len(slice); i++ {
		if slice[i] != original[i] {
			sameOrder = false
			break
		}
	}
	
	if sameOrder {
		t.Error("Shuffle did not change order")
	}
}

// TestDeterministicRNGReset tests Reset function
func TestDeterministicRNGReset(t *testing.T) {
	seed := []byte("test-seed")
	rng := NewDeterministicRNG(seed)
	
	// Generate some values
	val1 := rng.Uint32()
	val2 := rng.Uint32()
	val3 := rng.Uint32()
	
	// Reset
	rng.Reset()
	
	// Generate same values again
	val1Reset := rng.Uint32()
	val2Reset := rng.Uint32()
	val3Reset := rng.Uint32()
	
	// Check that sequence restarted
	if val1 != val1Reset || val2 != val2Reset || val3 != val3Reset {
		t.Error("Reset did not restart sequence")
	}
}

// TestRNGProvider tests RNG provider functionality
func TestRNGProvider(t *testing.T) {
	provider := NewRNGProvider()
	
	// Test getting RNG for domain
	seed := []byte("test-seed")
	rng1 := provider.GetRNG("domain1", seed)
	rng2 := provider.GetRNG("domain1", seed)
	
	// Should return same instance
	if rng1 != rng2 {
		t.Error("GetRNG returned different instances for same domain")
	}
	
	// Test different domains
	rng3 := provider.GetRNG("domain2", seed)
	if rng1 == rng3 {
		t.Error("GetRNG returned same instance for different domains")
	}
	
	// Test GetRNGFromExecution
	executionID := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	timestamp := uint64(time.Now().Unix())
	artifactHash := []byte("artifact-hash")
	
	rng4 := provider.GetRNGFromExecution("domain3", executionID, timestamp, artifactHash)
	if rng4 == nil {
		t.Error("GetRNGFromExecution returned nil")
	}
	
	// Test Clear
	provider.Clear()
	domains := provider.GetDomains()
	if len(domains) != 0 {
		t.Error("Clear did not remove all domains")
	}
}

// TestMockRNG tests mock RNG functionality
func TestMockRNG(t *testing.T) {
	values := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	mockRNG := NewMockRNG(values)
	
	// Test Read
	buf := make([]byte, 5)
	n, err := mockRNG.Read(buf)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Read returned %d, expected 5", n)
	}
	
	// Test Uint32
	mockRNG.Reset()
	val32 := mockRNG.Uint32()
	if val32 == 0 {
		t.Error("Uint32 returned 0")
	}
	
	// Test Uint64
	mockRNG.Reset()
	val64 := mockRNG.Uint64()
	if val64 == 0 {
		t.Error("Uint64 returned 0")
	}
	
	// Test Intn
	mockRNG.Reset()
	val := mockRNG.Intn(5)
	if val < 0 || val >= 5 {
		t.Errorf("Intn(5) returned %d, expected [0, 5)", val)
	}
	
	// Test Reset
	mockRNG.Reset()
	state := mockRNG.GetState()
	if state != 0 {
		t.Errorf("Reset did not reset state, got %d", state)
	}
	
	// Test SetState
	mockRNG.SetState(5)
	state = mockRNG.GetState()
	if state != 5 {
		t.Errorf("SetState did not set state, got %d", state)
	}
}
