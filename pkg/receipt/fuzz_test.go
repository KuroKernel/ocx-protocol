package receipt

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"
)

// FuzzCanonicalizeFull fuzzes the CanonicalizeFull function
func FuzzCanonicalizeFull(f *testing.F) {
	// Add seed corpus
	f.Add(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		[]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		[]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		uint64(1000),
		uint64(1640995200),
		uint64(1640995201),
		"test-issuer",
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	)

	f.Fuzz(func(t *testing.T, programHash, inputHash, outputHash []byte, gasUsed, startedAt, finishedAt uint64, issuerID string, nonce []byte) {
		// Skip if hashes are wrong length
		if len(programHash) != 32 || len(inputHash) != 32 || len(outputHash) != 32 {
			return
		}
		if len(nonce) != 16 {
			return
		}

		var ph, ih, oh [32]byte
		var n [16]byte
		copy(ph[:], programHash)
		copy(ih[:], inputHash)
		copy(oh[:], outputHash)
		copy(n[:], nonce)

		core := ReceiptCore{
			ProgramHash: ph,
			InputHash:   ih,
			OutputHash:  oh,
			GasUsed:     gasUsed,
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
			IssuerID:    issuerID,
			KeyVersion:  1,
			Nonce:       n,
			IssuedAt:    uint64(time.Now().UnixNano()),
			FloatMode:   "disabled",
		}

		full := &ReceiptFull{
			Core:       core,
			Signature:  make([]byte, 64),
			HostCycles: gasUsed * 5,
		}

		// Canonicalization should not panic
		data, err := CanonicalizeFull(full)
		if err != nil {
			return // Expected for some invalid inputs
		}

		// Verify determinism - same input should produce same output
		data2, err := CanonicalizeFull(full)
		if err != nil {
			t.Fatalf("Second canonicalization failed: %v", err)
		}

		if !bytes.Equal(data, data2) {
			t.Fatal("Canonicalization is not deterministic")
		}
	})
}

// FuzzCanonicalizeCore fuzzes the CanonicalizeCore function
func FuzzCanonicalizeCore(f *testing.F) {
	f.Add(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		uint64(1000),
		"test-issuer",
	)

	f.Fuzz(func(t *testing.T, hash []byte, gasUsed uint64, issuerID string) {
		if len(hash) < 32 {
			return
		}

		var h [32]byte
		copy(h[:], hash[:32])

		var nonce [16]byte
		rand.Read(nonce[:])

		core := &ReceiptCore{
			ProgramHash: h,
			InputHash:   h,
			OutputHash:  h,
			GasUsed:     gasUsed,
			StartedAt:   uint64(time.Now().UnixNano()) - 1000000,
			FinishedAt:  uint64(time.Now().UnixNano()),
			IssuerID:    issuerID,
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    uint64(time.Now().UnixNano()),
			FloatMode:   "disabled",
		}

		// Should not panic
		data, err := CanonicalizeCore(core)
		if err != nil {
			return
		}

		// Verify non-empty
		if len(data) == 0 {
			t.Fatal("Canonicalization produced empty output")
		}
	})
}

// FuzzCompression fuzzes the compression/decompression cycle
func FuzzCompression(f *testing.F) {
	// Add seed corpus with various data patterns
	f.Add([]byte("simple test data"))
	f.Add([]byte{0x00, 0x00, 0x00, 0x00}) // Zeros
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}) // All ones
	f.Add(bytes.Repeat([]byte("A"), 1000)) // Repetitive
	f.Add([]byte{})                         // Empty

	compressor, err := NewCompressor(CompressionDefault)
	if err != nil {
		f.Fatalf("Failed to create compressor: %v", err)
	}
	defer compressor.Close()

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create a receipt with the fuzz data as part of host info
		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])
		rand.Read(nonce[:])

		receipt := &ReceiptFull{
			Core: ReceiptCore{
				ProgramHash: programHash,
				InputHash:   inputHash,
				OutputHash:  outputHash,
				GasUsed:     1000,
				StartedAt:   uint64(time.Now().UnixNano()) - 1000000,
				FinishedAt:  uint64(time.Now().UnixNano()),
				IssuerID:    hex.EncodeToString(data), // Use hex-encoded fuzz data as issuer (valid UTF-8)
				KeyVersion:  1,
				Nonce:       nonce,
				IssuedAt:    uint64(time.Now().UnixNano()),
				FloatMode:   "disabled",
			},
			Signature:  make([]byte, 64),
			HostCycles: 5000,
		}

		// Compress should not panic
		compressed, err := compressor.CompressReceipt(receipt)
		if err != nil {
			return // Some inputs may be invalid
		}

		// Decompress should not panic
		decompressed, err := compressor.DecompressReceipt(compressed)
		if err != nil {
			t.Fatalf("Decompression failed after successful compression: %v", err)
		}

		// Verify roundtrip
		if decompressed.Core.IssuerID != receipt.Core.IssuerID {
			t.Fatal("Roundtrip failed: issuer ID mismatch")
		}
		if decompressed.Core.GasUsed != receipt.Core.GasUsed {
			t.Fatal("Roundtrip failed: gas used mismatch")
		}
	})
}

// FuzzDecompressCorrupted fuzzes decompression with potentially corrupted data
func FuzzDecompressCorrupted(f *testing.F) {
	// Add seed corpus with various corrupted patterns
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add(CompressedMagic)
	f.Add(append(CompressedMagic, 0x00, 0x00, 0x00, 0x00))
	f.Add([]byte("not cbor data"))
	f.Add([]byte{0xa2, 0x64}) // Truncated CBOR

	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		_, _ = compressor.DecompressReceipt(data)
	})
}

// FuzzMerkleTree fuzzes Merkle tree construction
func FuzzMerkleTree(f *testing.F) {
	f.Add(uint8(1))
	f.Add(uint8(2))
	f.Add(uint8(3))
	f.Add(uint8(7))
	f.Add(uint8(8))
	f.Add(uint8(15))
	f.Add(uint8(16))
	f.Add(uint8(100))

	f.Fuzz(func(t *testing.T, count uint8) {
		if count == 0 {
			// Empty should return error
			_, err := NewMerkleTree([]*ReceiptFull{})
			if err == nil {
				t.Fatal("Expected error for empty receipts")
			}
			return
		}

		// Limit to reasonable size for fuzzing
		if count > 100 {
			count = 100
		}

		receipts := make([]*ReceiptFull, count)
		for i := uint8(0); i < count; i++ {
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
					GasUsed:     uint64(1000) + uint64(i),
					StartedAt:   uint64(time.Now().UnixNano()) - 1000000,
					FinishedAt:  uint64(time.Now().UnixNano()),
					IssuerID:    "fuzz-issuer",
					KeyVersion:  1,
					Nonce:       nonce,
					IssuedAt:    uint64(time.Now().UnixNano()),
					FloatMode:   "disabled",
				},
				Signature:  make([]byte, 64),
				HostCycles: uint64(5000) + uint64(i),
			}
		}

		// Build tree should not panic
		tree, err := NewMerkleTree(receipts)
		if err != nil {
			t.Fatalf("Failed to build tree: %v", err)
		}

		// Root should not be zero
		var zeroHash [32]byte
		if tree.Root == zeroHash {
			t.Fatal("Root should not be zero")
		}

		// Verify all proofs
		for i := 0; i < int(count); i++ {
			proof, err := tree.GenerateProof(i)
			if err != nil {
				t.Fatalf("Failed to generate proof for index %d: %v", i, err)
			}

			if !proof.Verify() {
				t.Fatalf("Proof verification failed for index %d", i)
			}
		}
	})
}

// FuzzMerkleProofSerialization fuzzes proof serialization/deserialization
func FuzzMerkleProofSerialization(f *testing.F) {
	// Create a valid proof and use its serialized form as seed
	receipts := make([]*ReceiptFull, 8)
	for i := 0; i < 8; i++ {
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
				IssuerID:    "seed",
			},
			Signature: make([]byte, 64),
		}
	}
	tree, _ := NewMerkleTree(receipts)
	proof, _ := tree.GenerateProof(3)
	seedData := proof.Serialize()

	f.Add(seedData)
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		proof, err := DeserializeProof(data)
		if err != nil {
			return // Expected for invalid data
		}

		// If deserialization succeeded, verify should not panic
		_ = proof.Verify()

		// Re-serialization should be deterministic
		reserialized := proof.Serialize()
		proof2, err := DeserializeProof(reserialized)
		if err != nil {
			t.Fatalf("Failed to deserialize re-serialized proof: %v", err)
		}

		if proof.LeafIndex != proof2.LeafIndex {
			t.Fatal("Leaf index mismatch after roundtrip")
		}
	})
}

// FuzzBatchRoot fuzzes the BatchRoot function
func FuzzBatchRoot(f *testing.F) {
	f.Add(uint8(1))
	f.Add(uint8(10))
	f.Add(uint8(50))

	f.Fuzz(func(t *testing.T, count uint8) {
		if count == 0 {
			_, err := BatchRoot([]*ReceiptFull{})
			if err == nil {
				t.Fatal("Expected error for empty receipts")
			}
			return
		}

		if count > 50 {
			count = 50
		}

		receipts := make([]*ReceiptFull, count)
		for i := uint8(0); i < count; i++ {
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
					GasUsed:     uint64(1000) + uint64(i),
					Nonce:       nonce,
					IssuerID:    "fuzz",
				},
				Signature: make([]byte, 64),
			}
		}

		// BatchRoot should not panic
		root, err := BatchRoot(receipts)
		if err != nil {
			t.Fatalf("BatchRoot failed: %v", err)
		}

		var zeroHash [32]byte
		if root == zeroHash {
			t.Fatal("Root should not be zero")
		}

		// Verify determinism
		root2, _ := BatchRoot(receipts)
		if root != root2 {
			t.Fatal("BatchRoot is not deterministic")
		}
	})
}
