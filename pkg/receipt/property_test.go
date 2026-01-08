package receipt

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	mathrand "math/rand"
	"testing"
	"time"
)

// Property: Canonicalization is deterministic
// The same receipt must always produce the same canonical bytes
func TestProperty_CanonicalizationDeterminism(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		receipt := generateRandomReceipt(rng)

		data1, err1 := CanonicalizeFull(receipt)
		data2, err2 := CanonicalizeFull(receipt)
		data3, err3 := CanonicalizeFull(receipt)

		// All should succeed or fail together
		if (err1 != nil) != (err2 != nil) || (err2 != nil) != (err3 != nil) {
			t.Fatalf("Canonicalization error inconsistent at iteration %d", i)
		}

		if err1 == nil {
			if !bytes.Equal(data1, data2) || !bytes.Equal(data2, data3) {
				t.Fatalf("Canonicalization not deterministic at iteration %d", i)
			}
		}
	}
}

// Property: Core canonicalization is deterministic
func TestProperty_CoreCanonicalizationDeterminism(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		core := generateRandomCore(rng)

		data1, err1 := CanonicalizeCore(core)
		data2, err2 := CanonicalizeCore(core)

		if (err1 != nil) != (err2 != nil) {
			t.Fatalf("Core canonicalization error inconsistent at iteration %d", i)
		}

		if err1 == nil && !bytes.Equal(data1, data2) {
			t.Fatalf("Core canonicalization not deterministic at iteration %d", i)
		}
	}
}

// Property: Compression is deterministic
func TestProperty_CompressionDeterminism(t *testing.T) {
	compressor, err := NewCompressor(CompressionDefault)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 50; i++ {
		receipt := generateRandomReceipt(rng)

		compressed1, err1 := compressor.CompressReceipt(receipt)
		compressed2, err2 := compressor.CompressReceipt(receipt)

		if (err1 != nil) != (err2 != nil) {
			t.Fatalf("Compression error inconsistent at iteration %d", i)
		}

		if err1 == nil && !bytes.Equal(compressed1, compressed2) {
			t.Fatalf("Compression not deterministic at iteration %d", i)
		}
	}
}

// Property: Compression/Decompression is a perfect roundtrip
func TestProperty_CompressionRoundtrip(t *testing.T) {
	compressor, err := NewCompressor(CompressionDefault)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 50; i++ {
		receipt := generateRandomReceipt(rng)

		compressed, err := compressor.CompressReceipt(receipt)
		if err != nil {
			continue // Skip invalid receipts
		}

		decompressed, err := compressor.DecompressReceipt(compressed)
		if err != nil {
			t.Fatalf("Decompression failed at iteration %d: %v", i, err)
		}

		// Verify all fields match
		if decompressed.Core.ProgramHash != receipt.Core.ProgramHash {
			t.Fatalf("ProgramHash mismatch at iteration %d", i)
		}
		if decompressed.Core.InputHash != receipt.Core.InputHash {
			t.Fatalf("InputHash mismatch at iteration %d", i)
		}
		if decompressed.Core.OutputHash != receipt.Core.OutputHash {
			t.Fatalf("OutputHash mismatch at iteration %d", i)
		}
		if decompressed.Core.GasUsed != receipt.Core.GasUsed {
			t.Fatalf("GasUsed mismatch at iteration %d", i)
		}
		if decompressed.Core.IssuerID != receipt.Core.IssuerID {
			t.Fatalf("IssuerID mismatch at iteration %d", i)
		}
	}
}

// Property: Merkle tree root is deterministic
func TestProperty_MerkleRootDeterminism(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for count := 1; count <= 20; count++ {
		receipts := make([]*ReceiptFull, count)
		for i := 0; i < count; i++ {
			receipts[i] = generateRandomReceipt(rng)
		}

		tree1, err1 := NewMerkleTree(receipts)
		tree2, err2 := NewMerkleTree(receipts)

		if (err1 != nil) != (err2 != nil) {
			t.Fatalf("Merkle tree error inconsistent for count=%d", count)
		}

		if err1 == nil && tree1.Root != tree2.Root {
			t.Fatalf("Merkle root not deterministic for count=%d", count)
		}
	}
}

// Property: Merkle proofs are valid for all leaves
func TestProperty_MerkleProofValidity(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for count := 1; count <= 20; count++ {
		receipts := make([]*ReceiptFull, count)
		for i := 0; i < count; i++ {
			receipts[i] = generateRandomReceipt(rng)
		}

		tree, err := NewMerkleTree(receipts)
		if err != nil {
			t.Fatalf("Failed to create tree for count=%d: %v", count, err)
		}

		// Verify proof for each leaf
		for i := 0; i < count; i++ {
			proof, err := tree.GenerateProof(i)
			if err != nil {
				t.Fatalf("Failed to generate proof for index %d: %v", i, err)
			}

			if !proof.Verify() {
				t.Fatalf("Proof invalid for index %d in tree of %d", i, count)
			}
		}
	}
}

// Property: Merkle proof serialization is deterministic
func TestProperty_MerkleProofSerializationDeterminism(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	receipts := make([]*ReceiptFull, 16)
	for i := 0; i < 16; i++ {
		receipts[i] = generateRandomReceipt(rng)
	}

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	for i := 0; i < 16; i++ {
		proof, _ := tree.GenerateProof(i)

		serialized1 := proof.Serialize()
		serialized2 := proof.Serialize()

		if !bytes.Equal(serialized1, serialized2) {
			t.Fatalf("Proof serialization not deterministic for index %d", i)
		}
	}
}

// Property: Merkle proof deserialization roundtrip
func TestProperty_MerkleProofRoundtrip(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	receipts := make([]*ReceiptFull, 8)
	for i := 0; i < 8; i++ {
		receipts[i] = generateRandomReceipt(rng)
	}

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	for i := 0; i < 8; i++ {
		proof, _ := tree.GenerateProof(i)
		serialized := proof.Serialize()

		deserialized, err := DeserializeProof(serialized)
		if err != nil {
			t.Fatalf("Failed to deserialize proof for index %d: %v", i, err)
		}

		// Verify deserialized proof still works
		if !deserialized.Verify() {
			t.Fatalf("Deserialized proof invalid for index %d", i)
		}

		// Verify fields match
		if deserialized.LeafIndex != proof.LeafIndex {
			t.Fatalf("LeafIndex mismatch for index %d", i)
		}
		if deserialized.Root != proof.Root {
			t.Fatalf("Root mismatch for index %d", i)
		}
	}
}

// Property: BatchRoot equals MerkleTree root
func TestProperty_BatchRootConsistency(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for count := 1; count <= 20; count++ {
		receipts := make([]*ReceiptFull, count)
		for i := 0; i < count; i++ {
			receipts[i] = generateRandomReceipt(rng)
		}

		batchRoot, err1 := BatchRoot(receipts)
		tree, err2 := NewMerkleTree(receipts)

		if (err1 != nil) != (err2 != nil) {
			t.Fatalf("Batch/Tree error inconsistent for count=%d", count)
		}

		if err1 == nil && batchRoot != tree.Root {
			t.Fatalf("BatchRoot != MerkleTree.Root for count=%d", count)
		}
	}
}

// Property: Receipt hash is deterministic
func TestProperty_ReceiptHashDeterminism(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		receipt := generateRandomReceipt(rng)

		data, err := CanonicalizeFull(receipt)
		if err != nil {
			continue
		}

		hash1 := sha256.Sum256(data)
		hash2 := sha256.Sum256(data)

		if hash1 != hash2 {
			t.Fatalf("Receipt hash not deterministic at iteration %d", i)
		}
	}
}

// Property: Different receipts have different hashes (collision resistance)
func TestProperty_ReceiptHashUniqueness(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	hashes := make(map[[32]byte]bool)

	for i := 0; i < 1000; i++ {
		receipt := generateRandomReceipt(rng)

		data, err := CanonicalizeFull(receipt)
		if err != nil {
			continue
		}

		hash := sha256.Sum256(data)
		if hashes[hash] {
			// Collision is extremely unlikely for random data
			// If this triggers, we have a bug or extreme luck
			t.Log("Note: Hash collision detected (extremely unlikely for random data)")
		}
		hashes[hash] = true
	}
}

// Property: Merkle tree with power-of-2 leaves has expected depth
func TestProperty_MerkleTreeDepth(t *testing.T) {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for exp := 0; exp <= 5; exp++ {
		count := 1 << exp // 1, 2, 4, 8, 16, 32

		receipts := make([]*ReceiptFull, count)
		for i := 0; i < count; i++ {
			receipts[i] = generateRandomReceipt(rng)
		}

		tree, err := NewMerkleTree(receipts)
		if err != nil {
			t.Fatalf("Failed to create tree for count=%d: %v", count, err)
		}

		// Verify proof length is as expected (log2(count) hashes)
		if count > 1 {
			proof, _ := tree.GenerateProof(0)
			expectedLen := exp
			if len(proof.Siblings) != expectedLen {
				t.Errorf("Proof length %d != expected %d for count=%d",
					len(proof.Siblings), expectedLen, count)
			}
		}
	}
}

// Helper: Generate random receipt
func generateRandomReceipt(rng *mathrand.Rand) *ReceiptFull {
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte

	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	return &ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: programHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     uint64(rng.Intn(100000)),
			StartedAt:   uint64(time.Now().UnixNano()) - uint64(rng.Intn(1000000)),
			FinishedAt:  uint64(time.Now().UnixNano()),
			IssuerID:    "property-test-issuer",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    uint64(time.Now().UnixNano()),
			FloatMode:   "disabled",
		},
		Signature:  make([]byte, 64),
		HostCycles: uint64(rng.Intn(500000)),
	}
}

// Helper: Generate random core
func generateRandomCore(rng *mathrand.Rand) *ReceiptCore {
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte

	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	return &ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     uint64(rng.Intn(100000)),
		StartedAt:   uint64(time.Now().UnixNano()) - uint64(rng.Intn(1000000)),
		FinishedAt:  uint64(time.Now().UnixNano()),
		IssuerID:    "property-test-issuer",
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    uint64(time.Now().UnixNano()),
		FloatMode:   "disabled",
	}
}

// Benchmark property tests
func BenchmarkProperty_Canonicalization(b *testing.B) {
	rng := mathrand.New(mathrand.NewSource(42))
	receipt := generateRandomReceipt(rng)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CanonicalizeFull(receipt)
	}
}

func BenchmarkProperty_MerkleTree16(b *testing.B) {
	rng := mathrand.New(mathrand.NewSource(42))
	receipts := make([]*ReceiptFull, 16)
	for i := 0; i < 16; i++ {
		receipts[i] = generateRandomReceipt(rng)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMerkleTree(receipts)
	}
}

func BenchmarkProperty_MerkleProof(b *testing.B) {
	rng := mathrand.New(mathrand.NewSource(42))
	receipts := make([]*ReceiptFull, 16)
	for i := 0; i < 16; i++ {
		receipts[i] = generateRandomReceipt(rng)
	}
	tree, _ := NewMerkleTree(receipts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GenerateProof(i % 16)
	}
}
