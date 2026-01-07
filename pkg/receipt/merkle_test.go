package receipt

import (
	"crypto/rand"
	"testing"
	"time"
)

// createTestReceipts creates n test receipts
func createTestReceipts(t *testing.T, n int) []*ReceiptFull {
	t.Helper()

	receipts := make([]*ReceiptFull, n)
	for i := 0; i < n; i++ {
		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])
		rand.Read(nonce[:])

		now := uint64(time.Now().UnixNano())

		receipts[i] = &ReceiptFull{
			Core: ReceiptCore{
				ProgramHash: programHash,
				InputHash:   inputHash,
				OutputHash:  outputHash,
				GasUsed:     uint64(1000 + i),
				StartedAt:   now - 1000000,
				FinishedAt:  now - 500000,
				IssuerID:    "test-issuer",
				KeyVersion:  1,
				Nonce:       nonce,
				IssuedAt:    now,
				FloatMode:   "disabled",
			},
			Signature:  make([]byte, 64),
			HostCycles: uint64(5000 + i),
		}
	}

	return receipts
}

// TestMerkleTree_Build tests basic tree construction
func TestMerkleTree_Build(t *testing.T) {
	receipts := createTestReceipts(t, 8)

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("failed to build tree: %v", err)
	}

	// Should have log2(8) + 1 = 4 levels
	if len(tree.Levels) != 4 {
		t.Errorf("expected 4 levels, got %d", len(tree.Levels))
	}

	// Root should be a single hash
	if len(tree.Levels[len(tree.Levels)-1]) != 1 {
		t.Error("root level should have exactly 1 hash")
	}

	// Leaves should match input count
	if len(tree.Leaves) != 8 {
		t.Errorf("expected 8 leaves, got %d", len(tree.Leaves))
	}

	// Root should not be zero
	var zeroHash [32]byte
	if tree.Root == zeroHash {
		t.Error("root should not be zero")
	}

	t.Logf("Merkle root: %s", tree.RootHex())
}

// TestMerkleTree_Empty tests empty input handling
func TestMerkleTree_Empty(t *testing.T) {
	_, err := NewMerkleTree([]*ReceiptFull{})
	if err == nil {
		t.Error("expected error for empty receipts")
	}
}

// TestMerkleTree_SingleReceipt tests tree with one receipt
func TestMerkleTree_SingleReceipt(t *testing.T) {
	receipts := createTestReceipts(t, 1)

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("failed to build tree: %v", err)
	}

	// Root should equal the single leaf
	if tree.Root != tree.Leaves[0] {
		t.Error("single receipt tree: root should equal leaf")
	}
}

// TestMerkleTree_OddCount tests tree with odd number of receipts
func TestMerkleTree_OddCount(t *testing.T) {
	for _, count := range []int{3, 5, 7, 9, 15} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			receipts := createTestReceipts(t, count)

			tree, err := NewMerkleTree(receipts)
			if err != nil {
				t.Fatalf("failed to build tree with %d receipts: %v", count, err)
			}

			if len(tree.Leaves) != count {
				t.Errorf("expected %d leaves, got %d", count, len(tree.Leaves))
			}

			// Verify all proofs work
			for i := 0; i < count; i++ {
				proof, err := tree.GenerateProof(i)
				if err != nil {
					t.Errorf("failed to generate proof for index %d: %v", i, err)
					continue
				}

				if !proof.Verify() {
					t.Errorf("proof verification failed for index %d", i)
				}
			}
		})
	}
}

// TestMerkleTree_ProofGeneration tests proof generation
func TestMerkleTree_ProofGeneration(t *testing.T) {
	receipts := createTestReceipts(t, 8)

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("failed to build tree: %v", err)
	}

	for i := 0; i < 8; i++ {
		proof, err := tree.GenerateProof(i)
		if err != nil {
			t.Errorf("failed to generate proof for index %d: %v", i, err)
			continue
		}

		// Proof should have log2(8) = 3 siblings
		if len(proof.Siblings) != 3 {
			t.Errorf("expected 3 siblings, got %d for index %d", len(proof.Siblings), i)
		}

		if proof.LeafIndex != i {
			t.Errorf("leaf index mismatch: expected %d, got %d", i, proof.LeafIndex)
		}

		if proof.Root != tree.Root {
			t.Error("proof root should match tree root")
		}
	}
}

// TestMerkleTree_ProofVerification tests proof verification
func TestMerkleTree_ProofVerification(t *testing.T) {
	receipts := createTestReceipts(t, 16)

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("failed to build tree: %v", err)
	}

	for i := 0; i < 16; i++ {
		proof, _ := tree.GenerateProof(i)

		// Should verify successfully
		if !proof.Verify() {
			t.Errorf("proof verification failed for index %d", i)
		}

		// Verify receipt is in batch
		valid, err := VerifyReceiptInBatch(receipts[i], proof)
		if err != nil {
			t.Errorf("VerifyReceiptInBatch error for index %d: %v", i, err)
		}
		if !valid {
			t.Errorf("VerifyReceiptInBatch failed for index %d", i)
		}
	}
}

// TestMerkleTree_InvalidProof tests that invalid proofs are rejected
func TestMerkleTree_InvalidProof(t *testing.T) {
	receipts := createTestReceipts(t, 8)

	tree, _ := NewMerkleTree(receipts)
	proof, _ := tree.GenerateProof(0)

	// Tamper with leaf hash
	t.Run("tampered_leaf", func(t *testing.T) {
		tampered := *proof
		tampered.LeafHash[0] ^= 0xff
		if tampered.Verify() {
			t.Error("tampered leaf should fail verification")
		}
	})

	// Tamper with sibling
	t.Run("tampered_sibling", func(t *testing.T) {
		tampered := *proof
		tampered.Siblings[0][0] ^= 0xff
		if tampered.Verify() {
			t.Error("tampered sibling should fail verification")
		}
	})

	// Tamper with root
	t.Run("tampered_root", func(t *testing.T) {
		tampered := *proof
		tampered.Root[0] ^= 0xff
		if tampered.Verify() {
			t.Error("tampered root should fail verification")
		}
	})

	// Wrong receipt
	t.Run("wrong_receipt", func(t *testing.T) {
		otherReceipts := createTestReceipts(t, 1)
		valid, _ := VerifyReceiptInBatch(otherReceipts[0], proof)
		if valid {
			t.Error("wrong receipt should fail verification")
		}
	})
}

// TestMerkleTree_ProofSerialization tests proof serialization/deserialization
func TestMerkleTree_ProofSerialization(t *testing.T) {
	receipts := createTestReceipts(t, 8)

	tree, _ := NewMerkleTree(receipts)
	proof, _ := tree.GenerateProof(3)

	// Serialize
	data := proof.Serialize()

	t.Logf("Proof size: %d bytes for tree of 8 receipts", len(data))

	// Deserialize
	deserialized, err := DeserializeProof(data)
	if err != nil {
		t.Fatalf("failed to deserialize proof: %v", err)
	}

	// Verify deserialized proof
	if !deserialized.Verify() {
		t.Error("deserialized proof should verify")
	}

	if deserialized.LeafIndex != proof.LeafIndex {
		t.Error("leaf index mismatch after deserialization")
	}

	if len(deserialized.Siblings) != len(proof.Siblings) {
		t.Error("siblings count mismatch after deserialization")
	}
}

// TestMerkleTree_Determinism tests that same receipts produce same root
func TestMerkleTree_Determinism(t *testing.T) {
	// Create fixed receipts (same data)
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	copy(programHash[:], "program hash here 12345678901234")
	copy(inputHash[:], "input hash here 123456789012345")
	copy(outputHash[:], "output hash here 12345678901234")
	copy(nonce[:], "nonce here 12345")

	receipt := &ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: programHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     1000,
			StartedAt:   1000000,
			FinishedAt:  2000000,
			IssuerID:    "determinism-test",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    3000000,
			FloatMode:   "disabled",
		},
		Signature:  make([]byte, 64),
		HostCycles: 5000,
	}

	// Build tree twice
	tree1, _ := NewMerkleTree([]*ReceiptFull{receipt})
	tree2, _ := NewMerkleTree([]*ReceiptFull{receipt})

	if tree1.Root != tree2.Root {
		t.Error("same receipts should produce same root")
	}
}

// TestMerkleTree_BatchRoot tests convenience function
func TestMerkleTree_BatchRoot(t *testing.T) {
	receipts := createTestReceipts(t, 10)

	root, err := BatchRoot(receipts)
	if err != nil {
		t.Fatalf("BatchRoot failed: %v", err)
	}

	var zeroHash [32]byte
	if root == zeroHash {
		t.Error("root should not be zero")
	}

	// Verify it matches tree root
	tree, _ := NewMerkleTree(receipts)
	if root != tree.Root {
		t.Error("BatchRoot should match tree root")
	}
}

// TestMerkleTree_LargeTree tests larger tree
func TestMerkleTree_LargeTree(t *testing.T) {
	receipts := createTestReceipts(t, 1000)

	tree, err := NewMerkleTree(receipts)
	if err != nil {
		t.Fatalf("failed to build large tree: %v", err)
	}

	// Proof size should be log2(1000) ~ 10
	proof, _ := tree.GenerateProof(500)
	t.Logf("Proof size for 1000 receipts: %d siblings", len(proof.Siblings))

	if len(proof.Siblings) > 11 {
		t.Errorf("proof should have <= 11 siblings for 1000 receipts, got %d", len(proof.Siblings))
	}

	// Verify
	if !proof.Verify() {
		t.Error("large tree proof should verify")
	}
}

// TestMerkleTree_MultiProof tests multi-proof generation
func TestMerkleTree_MultiProof(t *testing.T) {
	receipts := createTestReceipts(t, 16)

	tree, _ := NewMerkleTree(receipts)

	// Generate multi-proof for indices 0, 5, 10, 15
	indices := []int{0, 5, 10, 15}
	multiProof, err := tree.GenerateMultiProof(indices)
	if err != nil {
		t.Fatalf("failed to generate multi-proof: %v", err)
	}

	if len(multiProof.LeafIndices) != len(indices) {
		t.Errorf("expected %d indices, got %d", len(indices), len(multiProof.LeafIndices))
	}

	if len(multiProof.LeafHashes) != len(indices) {
		t.Errorf("expected %d leaf hashes, got %d", len(indices), len(multiProof.LeafHashes))
	}

	t.Logf("Multi-proof for %d receipts: %d proof hashes (vs %d for individual proofs)",
		len(indices), len(multiProof.Proof), len(indices)*4)
}

// TestMerkleTree_InvalidIndex tests invalid index handling
func TestMerkleTree_InvalidIndex(t *testing.T) {
	receipts := createTestReceipts(t, 8)
	tree, _ := NewMerkleTree(receipts)

	testCases := []struct {
		name  string
		index int
	}{
		{"negative", -1},
		{"equal_to_count", 8},
		{"greater_than_count", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tree.GenerateProof(tc.index)
			if err == nil {
				t.Errorf("expected error for index %d", tc.index)
			}
		})
	}
}

// BenchmarkMerkleTree_Build benchmarks tree construction
func BenchmarkMerkleTree_Build(b *testing.B) {
	receipts := make([]*ReceiptFull, 1000)
	for i := 0; i < 1000; i++ {
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
				GasUsed:     1000,
				Nonce:       nonce,
				IssuerID:    "bench",
			},
			Signature: make([]byte, 64),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMerkleTree(receipts)
	}
}

// BenchmarkMerkleTree_Verify benchmarks proof verification
func BenchmarkMerkleTree_Verify(b *testing.B) {
	receipts := make([]*ReceiptFull, 1000)
	for i := 0; i < 1000; i++ {
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
				GasUsed:     1000,
				Nonce:       nonce,
				IssuerID:    "bench",
			},
			Signature: make([]byte, 64),
		}
	}

	tree, _ := NewMerkleTree(receipts)
	proof, _ := tree.GenerateProof(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proof.Verify()
	}
}
