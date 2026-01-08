package receipt

import (
	"crypto/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// StressTest: Concurrent Merkle tree construction
func TestStress_MerkleTreeConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	numGoroutines := runtime.NumCPU() * 2
	treesPerGoroutine := 20

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < treesPerGoroutine; i++ {
				receipts := generateStressReceipts(16)
				tree, err := NewMerkleTree(receipts)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				// Verify all proofs
				allValid := true
				for j := 0; j < len(receipts); j++ {
					proof, err := tree.GenerateProof(j)
					if err != nil {
						allValid = false
						break
					}
					if !proof.Verify() {
						allValid = false
						break
					}
				}

				if allValid {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * treesPerGoroutine)
	t.Logf("Stress test: %d/%d trees built and verified successfully", successCount, expected)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d (errors: %d)", expected, successCount, errorCount)
	}
}

// StressTest: Concurrent compression/decompression
func TestStress_CompressionConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	numGoroutines := runtime.NumCPU() * 2
	operationsPerGoroutine := 100

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each goroutine gets its own compressor
			compressor, err := NewCompressor(CompressionDefault)
			if err != nil {
				atomic.AddInt64(&errorCount, int64(operationsPerGoroutine))
				return
			}
			defer compressor.Close()

			for i := 0; i < operationsPerGoroutine; i++ {
				receipts := generateStressReceipts(1)
				receipt := receipts[0]

				compressed, err := compressor.CompressReceipt(receipt)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				decompressed, err := compressor.DecompressReceipt(compressed)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				// Verify roundtrip
				if decompressed.Core.GasUsed != receipt.Core.GasUsed {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * operationsPerGoroutine)
	t.Logf("Stress test: %d/%d compress/decompress cycles successful", successCount, expected)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d (errors: %d)", expected, successCount, errorCount)
	}
}

// StressTest: Concurrent canonicalization
func TestStress_CanonicalizationConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var wg sync.WaitGroup
	var successCount int64

	numGoroutines := runtime.NumCPU() * 4
	operationsPerGoroutine := 500

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				receipts := generateStressReceipts(1)
				receipt := receipts[0]

				data1, err1 := CanonicalizeFull(receipt)
				data2, err2 := CanonicalizeFull(receipt)

				if err1 == nil && err2 == nil && len(data1) == len(data2) {
					// Verify determinism
					match := true
					for j := range data1 {
						if data1[j] != data2[j] {
							match = false
							break
						}
					}
					if match {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * operationsPerGoroutine)
	t.Logf("Stress test: %d/%d canonicalizations consistent", successCount, expected)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d", expected, successCount)
	}
}

// StressTest: BatchRoot concurrency
func TestStress_BatchRootConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var wg sync.WaitGroup
	var successCount int64

	numGoroutines := runtime.NumCPU() * 2
	operationsPerGoroutine := 50

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				receipts := generateStressReceipts(8)

				// BatchRoot should be deterministic
				root1, err1 := BatchRoot(receipts)
				root2, err2 := BatchRoot(receipts)

				if err1 == nil && err2 == nil && root1 == root2 {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * operationsPerGoroutine)
	t.Logf("Stress test: %d/%d BatchRoot calculations consistent", successCount, expected)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d", expected, successCount)
	}
}

// StressTest: Merkle proof generation under load
func TestStress_MerkleProofGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Build a large tree once
	receipts := generateStressReceipts(256)
	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	var wg sync.WaitGroup
	var successCount int64

	numGoroutines := runtime.NumCPU() * 4
	proofsPerGoroutine := 100

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < proofsPerGoroutine; i++ {
				idx := i % 256

				proof, err := tree.GenerateProof(idx)
				if err != nil {
					continue
				}

				if proof.Verify() {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * proofsPerGoroutine)
	t.Logf("Stress test: %d/%d proof generations succeeded", successCount, expected)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d", expected, successCount)
	}
}

// StressTest: Proof serialization throughput
func TestStress_ProofSerialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Build tree and generate proofs
	receipts := generateStressReceipts(64)
	tree, _ := NewMerkleTree(receipts)

	proofs := make([]*MerkleProof, 64)
	for i := 0; i < 64; i++ {
		proofs[i], _ = tree.GenerateProof(i)
	}

	var wg sync.WaitGroup
	var successCount int64

	numGoroutines := runtime.NumCPU() * 4
	operationsPerGoroutine := 200

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < operationsPerGoroutine; i++ {
				proof := proofs[i%64]

				// Serialize
				serialized := proof.Serialize()

				// Deserialize
				deserialized, err := DeserializeProof(serialized)
				if err != nil {
					continue
				}

				// Verify roundtrip
				if deserialized.LeafIndex == proof.LeafIndex {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	elapsed := time.Since(start)
	throughput := float64(successCount) / elapsed.Seconds()

	expected := int64(numGoroutines * operationsPerGoroutine)
	t.Logf("Stress test: %d/%d serialization roundtrips (%.2f ops/sec)", successCount, expected, throughput)

	if successCount != expected {
		t.Errorf("Expected %d successes, got %d", expected, successCount)
	}
}

// StressTest: Memory under repeated tree construction
func TestStress_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Force GC
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	iterations := 100
	for i := 0; i < iterations; i++ {
		receipts := generateStressReceipts(32)
		tree, _ := NewMerkleTree(receipts)

		// Generate all proofs
		for j := 0; j < 32; j++ {
			tree.GenerateProof(j)
		}

		// Compute batch root
		BatchRoot(receipts)
	}

	// Check memory
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	allocGrowth := memAfter.Alloc - memBefore.Alloc
	t.Logf("Memory: before=%d KB, after=%d KB, growth=%d KB",
		memBefore.Alloc/1024, memAfter.Alloc/1024, allocGrowth/1024)

	// Should not grow excessively (less than 20MB)
	maxGrowth := uint64(20 * 1024 * 1024)
	if allocGrowth > maxGrowth {
		t.Errorf("Memory growth %d exceeds limit %d", allocGrowth, maxGrowth)
	}
}

// StressTest: Large tree construction
func TestStress_LargeTree(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		start := time.Now()

		receipts := generateStressReceipts(size)
		tree, err := NewMerkleTree(receipts)
		if err != nil {
			t.Errorf("Failed to create tree of size %d: %v", size, err)
			continue
		}

		// Verify random proofs
		for i := 0; i < 10; i++ {
			idx := i * size / 10
			proof, err := tree.GenerateProof(idx)
			if err != nil {
				t.Errorf("Failed to generate proof %d for tree size %d", idx, size)
				continue
			}
			if !proof.Verify() {
				t.Errorf("Proof %d failed for tree size %d", idx, size)
			}
		}

		elapsed := time.Since(start)
		t.Logf("Tree size %d: construction + 10 proofs in %v", size, elapsed)
	}
}

// Helper: Generate stress test receipts
func generateStressReceipts(count int) []*ReceiptFull {
	receipts := make([]*ReceiptFull, count)
	for i := 0; i < count; i++ {
		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])
		rand.Read(nonce[:])

		receipts[i] = &ReceiptFull{
			Core: ReceiptCore{
				ProgramHash: programHash,
				InputHash:   inputHash,
				OutputHash:  outputHash,
				GasUsed:     uint64(1000 + i),
				StartedAt:   uint64(time.Now().UnixNano()) - 1000000,
				FinishedAt:  uint64(time.Now().UnixNano()),
				IssuerID:    "stress-test",
				KeyVersion:  1,
				Nonce:       nonce,
				IssuedAt:    uint64(time.Now().UnixNano()),
				FloatMode:   "disabled",
			},
			Signature:  make([]byte, 64),
			HostCycles: uint64(5000 + i),
		}
	}
	return receipts
}

// Benchmark stress scenarios
func BenchmarkStress_MerkleTree(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			receipts := generateStressReceipts(16)
			NewMerkleTree(receipts)
		}
	})
}

func BenchmarkStress_Compression(b *testing.B) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	receipts := generateStressReceipts(1)
	receipt := receipts[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressed, _ := compressor.CompressReceipt(receipt)
		compressor.DecompressReceipt(compressed)
	}
}

func BenchmarkStress_Canonicalization(b *testing.B) {
	receipts := generateStressReceipts(1)
	receipt := receipts[0]

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			CanonicalizeFull(receipt)
		}
	})
}
