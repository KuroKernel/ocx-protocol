package receipt

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"
)

// createTestReceiptFull creates a test receipt for compression tests
func createTestReceiptFull(t *testing.T) *ReceiptFull {
	t.Helper()

	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	now := uint64(time.Now().UnixNano())

	return &ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: programHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     12345,
			StartedAt:   now - 1000000,
			FinishedAt:  now - 500000,
			IssuerID:    "test-issuer-id-12345",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    now,
			FloatMode:   "disabled",
		},
		Signature:  make([]byte, 64),
		HostCycles: 1000000,
		HostInfo: map[string]string{
			"platform": "linux/amd64",
			"version":  "1.0.0",
		},
	}
}

// TestCompressor_RoundTrip tests compression/decompression roundtrip
func TestCompressor_RoundTrip(t *testing.T) {
	compressor, err := NewCompressor(CompressionDefault)
	if err != nil {
		t.Fatalf("failed to create compressor: %v", err)
	}
	defer compressor.Close()

	original := createTestReceiptFull(t)

	// Compress
	compressed, err := compressor.CompressReceipt(original)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	// Verify magic header
	if !IsCompressed(compressed) {
		t.Error("compressed data should have magic header")
	}

	// Decompress
	decompressed, err := compressor.DecompressReceipt(compressed)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	// Compare
	if decompressed.Core.IssuerID != original.Core.IssuerID {
		t.Error("issuer ID mismatch")
	}

	if decompressed.Core.GasUsed != original.Core.GasUsed {
		t.Error("gas used mismatch")
	}

	if decompressed.Core.Nonce != original.Core.Nonce {
		t.Error("nonce mismatch")
	}

	if !bytes.Equal(decompressed.Signature, original.Signature) {
		t.Error("signature mismatch")
	}
}

// TestCompressor_CompressionRatio tests that compression actually reduces size
func TestCompressor_CompressionRatio(t *testing.T) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	receipt := createTestReceiptFull(t)

	ratio, err := compressor.CompressRatio(receipt)
	if err != nil {
		t.Fatalf("failed to get compression ratio: %v", err)
	}

	t.Logf("Compression ratio: %.2fx", ratio)

	// Should achieve at least some compression
	if ratio < 1.0 {
		t.Errorf("expected ratio > 1.0, got %.2f", ratio)
	}
}

// TestCompressor_Levels tests different compression levels
func TestCompressor_Levels(t *testing.T) {
	levels := []CompressionLevel{CompressionFastest, CompressionDefault, CompressionBest}
	receipt := createTestReceiptFull(t)

	for _, level := range levels {
		t.Run(levelName(level), func(t *testing.T) {
			compressor, err := NewCompressor(level)
			if err != nil {
				t.Fatalf("failed to create compressor: %v", err)
			}
			defer compressor.Close()

			compressed, err := compressor.CompressReceipt(receipt)
			if err != nil {
				t.Fatalf("failed to compress: %v", err)
			}

			decompressed, err := compressor.DecompressReceipt(compressed)
			if err != nil {
				t.Fatalf("failed to decompress: %v", err)
			}

			if decompressed.Core.IssuerID != receipt.Core.IssuerID {
				t.Error("roundtrip failed")
			}
		})
	}
}

func levelName(level CompressionLevel) string {
	switch level {
	case CompressionFastest:
		return "fastest"
	case CompressionDefault:
		return "default"
	case CompressionBest:
		return "best"
	default:
		return "unknown"
	}
}

// TestCompressor_UncompressedData tests handling of uncompressed data
func TestCompressor_UncompressedData(t *testing.T) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	receipt := createTestReceiptFull(t)

	// Get uncompressed CBOR
	cbor, err := CanonicalizeFull(receipt)
	if err != nil {
		t.Fatalf("failed to canonicalize: %v", err)
	}

	// Should not have magic header
	if IsCompressed(cbor) {
		t.Error("uncompressed data should not have magic header")
	}

	// DecompressReceipt should handle uncompressed data
	decompressed, err := compressor.DecompressReceipt(cbor)
	if err != nil {
		t.Fatalf("failed to decompress uncompressed data: %v", err)
	}

	if decompressed.Core.IssuerID != receipt.Core.IssuerID {
		t.Error("roundtrip failed for uncompressed data")
	}
}

// TestCompressor_CorruptedData tests handling of corrupted compressed data
func TestCompressor_CorruptedData(t *testing.T) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	testCases := []struct {
		name string
		data []byte
	}{
		{"magic_only", CompressedMagic},
		{"magic_plus_short", append(CompressedMagic, 0x00, 0x00)},
		{"invalid_zstd", append(append(CompressedMagic, 0x00, 0x00, 0x00, 0x10), []byte("not zstd data")...)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compressor.DecompressReceipt(tc.data)
			if err == nil {
				t.Error("expected error for corrupted data")
			}
		})
	}
}

// TestCompressor_BatchRoundTrip tests batch compression/decompression
func TestCompressor_BatchRoundTrip(t *testing.T) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	// Create batch of receipts
	receipts := make([]*ReceiptFull, 10)
	for i := 0; i < 10; i++ {
		receipts[i] = createTestReceiptFull(t)
	}

	// Compress batch
	compressed, err := compressor.CompressReceiptBatch(receipts)
	if err != nil {
		t.Fatalf("failed to compress batch: %v", err)
	}

	// Decompress batch
	decompressed, err := compressor.DecompressReceiptBatch(compressed)
	if err != nil {
		t.Fatalf("failed to decompress batch: %v", err)
	}

	// Verify count
	if len(decompressed) != len(receipts) {
		t.Fatalf("expected %d receipts, got %d", len(receipts), len(decompressed))
	}

	// Verify each receipt
	for i := 0; i < len(receipts); i++ {
		if decompressed[i].Core.Nonce != receipts[i].Core.Nonce {
			t.Errorf("receipt %d nonce mismatch", i)
		}
	}
}

// TestCompressor_Stats tests compression statistics
func TestCompressor_Stats(t *testing.T) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

	receipt := createTestReceiptFull(t)

	stats, err := compressor.GetStats(receipt)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2fx, Algorithm: %s",
		stats.OriginalSize, stats.CompressedSize, stats.Ratio, stats.Algorithm)

	if stats.OriginalSize <= 0 {
		t.Error("original size should be positive")
	}

	if stats.CompressedSize <= 0 {
		t.Error("compressed size should be positive")
	}

	if stats.Algorithm != "zstd" {
		t.Errorf("expected algorithm zstd, got %s", stats.Algorithm)
	}
}

// TestDefaultCompressor tests the singleton compressor
func TestDefaultCompressor(t *testing.T) {
	c1 := DefaultCompressor()
	c2 := DefaultCompressor()

	if c1 != c2 {
		t.Error("DefaultCompressor should return singleton")
	}

	receipt := createTestReceiptFull(t)
	compressed, err := c1.CompressReceipt(receipt)
	if err != nil {
		t.Fatalf("failed to compress: %v", err)
	}

	if !IsCompressed(compressed) {
		t.Error("should be compressed")
	}
}

// BenchmarkCompressor_Compress benchmarks compression
func BenchmarkCompressor_Compress(b *testing.B) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

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
			GasUsed:     12345,
			StartedAt:   uint64(time.Now().UnixNano()),
			FinishedAt:  uint64(time.Now().UnixNano()),
			IssuerID:    "benchmark-issuer",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    uint64(time.Now().UnixNano()),
			FloatMode:   "disabled",
		},
		Signature:  make([]byte, 64),
		HostCycles: 1000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.CompressReceipt(receipt)
	}
}

// BenchmarkCompressor_Decompress benchmarks decompression
func BenchmarkCompressor_Decompress(b *testing.B) {
	compressor, _ := NewCompressor(CompressionDefault)
	defer compressor.Close()

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
			GasUsed:     12345,
			StartedAt:   uint64(time.Now().UnixNano()),
			FinishedAt:  uint64(time.Now().UnixNano()),
			IssuerID:    "benchmark-issuer",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    uint64(time.Now().UnixNano()),
			FloatMode:   "disabled",
		},
		Signature:  make([]byte, 64),
		HostCycles: 1000000,
	}

	compressed, _ := compressor.CompressReceipt(receipt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.DecompressReceipt(compressed)
	}
}
