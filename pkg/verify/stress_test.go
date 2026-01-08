package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ocx.local/pkg/receipt"
)

// StressTest: High concurrency replay store operations
func TestStress_ReplayStoreConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	timestamp := uint64(time.Now().UnixNano())

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	numGoroutines := runtime.NumCPU() * 4
	operationsPerGoroutine := 1000

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				nonce := make([]byte, 16)
				rand.Read(nonce)

				ok, err := store.CheckAndStore(nonce, timestamp)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else if ok {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * operationsPerGoroutine)
	total := successCount + errorCount

	t.Logf("Stress test: %d goroutines x %d ops = %d total", numGoroutines, operationsPerGoroutine, expected)
	t.Logf("Results: %d successes, %d errors, store size: %d", successCount, errorCount, store.Size())

	// All unique nonces should succeed
	if successCount != expected {
		t.Errorf("Expected %d successes, got %d", expected, successCount)
	}

	if total != expected {
		t.Errorf("Total operations mismatch: expected %d, got %d", expected, total)
	}
}

// StressTest: Batch verification under load
func TestStress_BatchVerifierConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: runtime.NumCPU()})
	if err != nil {
		t.Fatalf("Failed to create batch verifier: %v", err)
	}

	var wg sync.WaitGroup
	var totalVerified int64

	numGoroutines := runtime.NumCPU() * 2
	batchesPerGoroutine := 10
	batchSize := 20

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for b := 0; b < batchesPerGoroutine; b++ {
				batches := make([]ReceiptBatch, batchSize)
				for i := 0; i < batchSize; i++ {
					pub, priv, _ := ed25519.GenerateKey(rand.Reader)
					receiptFull := createStressReceipt(priv)
					receiptData, _ := receipt.CanonicalizeFull(receiptFull)

					batches[i] = ReceiptBatch{
						ReceiptData: receiptData,
						PublicKey:   pub,
					}
				}

				results, stats := bv.VerifyBatch(context.Background(), batches)

				if len(results) != batchSize {
					t.Errorf("Result count mismatch: expected %d, got %d", batchSize, len(results))
				}

				atomic.AddInt64(&totalVerified, int64(stats.Valid))
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * batchesPerGoroutine * batchSize)
	t.Logf("Stress test: verified %d receipts across %d goroutines", totalVerified, numGoroutines)

	if totalVerified != expected {
		t.Errorf("Expected %d verified receipts, got %d", expected, totalVerified)
	}
}

// StressTest: Mixed valid/invalid receipts under load
func TestStress_MixedBatchVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: runtime.NumCPU()})

	var wg sync.WaitGroup
	var totalValid int64
	var totalInvalid int64

	numGoroutines := runtime.NumCPU()
	batchesPerGoroutine := 5

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for b := 0; b < batchesPerGoroutine; b++ {
				// Create 10 valid, 10 invalid
				batches := make([]ReceiptBatch, 20)

				// Valid receipts
				for i := 0; i < 10; i++ {
					pub, priv, _ := ed25519.GenerateKey(rand.Reader)
					receiptFull := createStressReceipt(priv)
					receiptData, _ := receipt.CanonicalizeFull(receiptFull)

					batches[i] = ReceiptBatch{
						ReceiptData: receiptData,
						PublicKey:   pub,
					}
				}

				// Invalid receipts (wrong key)
				for i := 10; i < 20; i++ {
					_, priv, _ := ed25519.GenerateKey(rand.Reader)
					wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)
					receiptFull := createStressReceipt(priv)
					receiptData, _ := receipt.CanonicalizeFull(receiptFull)

					batches[i] = ReceiptBatch{
						ReceiptData: receiptData,
						PublicKey:   wrongPub,
					}
				}

				_, stats := bv.VerifyBatch(context.Background(), batches)

				atomic.AddInt64(&totalValid, int64(stats.Valid))
				atomic.AddInt64(&totalInvalid, int64(stats.Invalid))
			}
		}()
	}

	wg.Wait()

	expectedValid := int64(numGoroutines * batchesPerGoroutine * 10)
	expectedInvalid := int64(numGoroutines * batchesPerGoroutine * 10)

	t.Logf("Stress test: %d valid, %d invalid", totalValid, totalInvalid)

	if totalValid != expectedValid {
		t.Errorf("Expected %d valid, got %d", expectedValid, totalValid)
	}
	if totalInvalid != expectedInvalid {
		t.Errorf("Expected %d invalid, got %d", expectedInvalid, totalInvalid)
	}
}

// StressTest: Replay store with duplicate detection
func TestStress_ReplayStoreDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	timestamp := uint64(time.Now().UnixNano())

	// Pre-generate shared nonces
	sharedNonces := make([][]byte, 100)
	for i := range sharedNonces {
		sharedNonces[i] = make([]byte, 16)
		rand.Read(sharedNonces[i])
	}

	var wg sync.WaitGroup
	var acceptedCount int64

	numGoroutines := 20

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, nonce := range sharedNonces {
				ok, err := store.CheckAndStore(nonce, timestamp)
				if err == nil && ok {
					atomic.AddInt64(&acceptedCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	// Exactly 100 should be accepted (one per unique nonce)
	if acceptedCount != 100 {
		t.Errorf("Expected exactly 100 acceptances (one per unique nonce), got %d", acceptedCount)
	}

	// Store should have exactly 100 entries
	if store.Size() != 100 {
		t.Errorf("Store size should be 100, got %d", store.Size())
	}
}

// StressTest: Verifier throughput measurement
func TestStress_VerifierThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	verifier := NewGoVerifier()

	// Pre-create receipts
	numReceipts := 1000
	receipts := make([]struct {
		data []byte
		pub  []byte
	}, numReceipts)

	for i := 0; i < numReceipts; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		receiptFull := createStressReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)
		receipts[i] = struct {
			data []byte
			pub  []byte
		}{data: receiptData, pub: pub}
	}

	// Measure throughput
	start := time.Now()

	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU()
	receiptsPerWorker := numReceipts / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		startIdx := w * receiptsPerWorker
		endIdx := startIdx + receiptsPerWorker
		if w == numWorkers-1 {
			endIdx = numReceipts
		}

		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				verifier.VerifyReceipt(receipts[i].data, receipts[i].pub)
			}
		}(startIdx, endIdx)
	}

	wg.Wait()

	elapsed := time.Since(start)
	throughput := float64(numReceipts) / elapsed.Seconds()

	t.Logf("Throughput: %.2f verifications/second with %d workers", throughput, numWorkers)

	// Minimum expected throughput (very conservative)
	minThroughput := 100.0
	if throughput < minThroughput {
		t.Errorf("Throughput %.2f below minimum %.2f", throughput, minThroughput)
	}
}

// StressTest: Batch verifier scaling
func TestStress_BatchVerifierScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Test different worker counts
	workerCounts := []int{1, 2, 4, runtime.NumCPU()}
	batchSize := 100

	// Create test batch
	batches := make([]ReceiptBatch, batchSize)
	for i := 0; i < batchSize; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		receiptFull := createStressReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)
		batches[i] = ReceiptBatch{
			ReceiptData: receiptData,
			PublicKey:   pub,
		}
	}

	var lastDuration time.Duration

	for _, workers := range workerCounts {
		bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: workers})

		start := time.Now()
		iterations := 10
		for i := 0; i < iterations; i++ {
			bv.VerifyBatch(context.Background(), batches)
		}
		duration := time.Since(start) / time.Duration(iterations)

		t.Logf("Workers=%d: avg %.2fms per batch of %d", workers, float64(duration.Microseconds())/1000, batchSize)

		// More workers should generally not be slower (with some tolerance)
		if lastDuration > 0 && duration > lastDuration*2 {
			t.Logf("Warning: more workers (%d) took longer than fewer workers", workers)
		}
		lastDuration = duration
	}
}

// StressTest: Memory pressure during batch operations
func TestStress_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: runtime.NumCPU()})

	// Force GC to get baseline
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Run many batch verifications
	iterations := 100
	batchSize := 50

	for iter := 0; iter < iterations; iter++ {
		batches := make([]ReceiptBatch, batchSize)
		for i := 0; i < batchSize; i++ {
			pub, priv, _ := ed25519.GenerateKey(rand.Reader)
			receiptFull := createStressReceipt(priv)
			receiptData, _ := receipt.CanonicalizeFull(receiptFull)
			batches[i] = ReceiptBatch{
				ReceiptData: receiptData,
				PublicKey:   pub,
			}
		}
		bv.VerifyBatch(context.Background(), batches)
	}

	// Check memory usage
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	allocGrowth := memAfter.Alloc - memBefore.Alloc
	t.Logf("Memory: before=%d KB, after=%d KB, growth=%d KB",
		memBefore.Alloc/1024, memAfter.Alloc/1024, allocGrowth/1024)

	// Memory growth should be reasonable (less than 50MB)
	maxGrowth := uint64(50 * 1024 * 1024)
	if allocGrowth > maxGrowth {
		t.Errorf("Memory growth %d exceeds limit %d", allocGrowth, maxGrowth)
	}
}

// StressTest: Concurrent context cancellation
func TestStress_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: runtime.NumCPU()})

	var wg sync.WaitGroup
	var cancelled int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())

			// Cancel almost immediately
			go func() {
				time.Sleep(time.Microsecond * 100)
				cancel()
			}()

			batches := make([]ReceiptBatch, 100)
			for j := 0; j < 100; j++ {
				pub, priv, _ := ed25519.GenerateKey(rand.Reader)
				receiptFull := createStressReceipt(priv)
				receiptData, _ := receipt.CanonicalizeFull(receiptFull)
				batches[j] = ReceiptBatch{
					ReceiptData: receiptData,
					PublicKey:   pub,
				}
			}

			_, stats := bv.VerifyBatch(ctx, batches)
			if stats.Valid+stats.Invalid < 100 {
				atomic.AddInt64(&cancelled, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Context cancellations that took effect: %d/20", cancelled)
	// Some should have been cancelled (but not all - timing dependent)
}

// Helper for stress tests
func createStressReceipt(priv ed25519.PrivateKey) *receipt.ReceiptFull {
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	now := uint64(time.Now().UnixNano())
	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "stress-test",
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, _ := receipt.CanonicalizeCore(&core)
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	return &receipt.ReceiptFull{
		Core:       core,
		Signature:  sig,
		HostCycles: 5000,
	}
}

// Benchmark stress scenarios
func BenchmarkStress_ReplayStore(b *testing.B) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	timestamp := uint64(time.Now().UnixNano())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			nonce := make([]byte, 16)
			rand.Read(nonce)
			store.CheckAndStore(nonce, timestamp)
		}
	})
}

func BenchmarkStress_BatchVerifier(b *testing.B) {
	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: runtime.NumCPU()})

	// Pre-create batch
	batchSize := 20
	batches := make([]ReceiptBatch, batchSize)
	for i := 0; i < batchSize; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		receiptFull := createStressReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)
		batches[i] = ReceiptBatch{
			ReceiptData: receiptData,
			PublicKey:   pub,
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bv.VerifyBatch(context.Background(), batches)
		}
	})
}
