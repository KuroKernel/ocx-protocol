package receipt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
)

// Magic bytes for compressed receipts
var CompressedMagic = []byte{0x4F, 0x43, 0x58, 0x5A} // "OCXZ"

// CompressionLevel defines compression speed/ratio tradeoff
type CompressionLevel int

const (
	CompressionFastest CompressionLevel = iota // Fastest, larger size
	CompressionDefault                         // Balanced
	CompressionBest                            // Best ratio, slower
)

// Compressor handles receipt compression/decompression
type Compressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	encMu   sync.Mutex
	decMu   sync.Mutex
}

var (
	defaultCompressor *Compressor
	initOnce          sync.Once
)

// DefaultCompressor returns the shared compressor instance
func DefaultCompressor() *Compressor {
	initOnce.Do(func() {
		var err error
		defaultCompressor, err = NewCompressor(CompressionDefault)
		if err != nil {
			panic(fmt.Sprintf("failed to create default compressor: %v", err))
		}
	})
	return defaultCompressor
}

// NewCompressor creates a new compressor with specified level
func NewCompressor(level CompressionLevel) (*Compressor, error) {
	var encLevel zstd.EncoderLevel
	switch level {
	case CompressionFastest:
		encLevel = zstd.SpeedFastest
	case CompressionDefault:
		encLevel = zstd.SpeedDefault
	case CompressionBest:
		encLevel = zstd.SpeedBestCompression
	}

	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(encLevel),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	decoder, err := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderMaxMemory(64<<20), // 64MB max
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	return &Compressor{
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// CompressReceipt compresses a ReceiptFull to bytes
// Format: OCXZ + uint32(uncompressed_size) + zstd(cbor)
func (c *Compressor) CompressReceipt(receipt *ReceiptFull) ([]byte, error) {
	// Encode to CBOR first
	cborData, err := cbor.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("CBOR encode failed: %w", err)
	}

	c.encMu.Lock()
	compressed := c.encoder.EncodeAll(cborData, nil)
	c.encMu.Unlock()

	// Build output: magic + original_size + compressed_data
	var buf bytes.Buffer
	buf.Write(CompressedMagic)

	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(len(cborData)))
	buf.Write(sizeBytes)

	buf.Write(compressed)

	return buf.Bytes(), nil
}

// DecompressReceipt decompresses bytes to a ReceiptFull
func (c *Compressor) DecompressReceipt(data []byte) (*ReceiptFull, error) {
	// Check if compressed (has magic header)
	if !bytes.HasPrefix(data, CompressedMagic) {
		// Not compressed, try direct CBOR decode
		var receipt ReceiptFull
		if err := cbor.Unmarshal(data, &receipt); err != nil {
			return nil, fmt.Errorf("CBOR decode failed: %w", err)
		}
		return &receipt, nil
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("compressed data too short")
	}

	// Parse header
	originalSize := binary.BigEndian.Uint32(data[4:8])
	compressed := data[8:]

	// Decompress
	c.decMu.Lock()
	decompressed, err := c.decoder.DecodeAll(compressed, make([]byte, 0, originalSize))
	c.decMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("zstd decompress failed: %w", err)
	}

	// Decode CBOR
	var receipt ReceiptFull
	if err := cbor.Unmarshal(decompressed, &receipt); err != nil {
		return nil, fmt.Errorf("CBOR decode failed: %w", err)
	}

	return &receipt, nil
}

// CompressRatio returns the compression ratio for given data
func (c *Compressor) CompressRatio(receipt *ReceiptFull) (float64, error) {
	original, err := cbor.Marshal(receipt)
	if err != nil {
		return 0, err
	}

	compressed, err := c.CompressReceipt(receipt)
	if err != nil {
		return 0, err
	}

	return float64(len(original)) / float64(len(compressed)), nil
}

// IsCompressed checks if data has compression magic header
func IsCompressed(data []byte) bool {
	return bytes.HasPrefix(data, CompressedMagic)
}

// CompressReceiptBatch compresses multiple receipts efficiently
func (c *Compressor) CompressReceiptBatch(receipts []*ReceiptFull) ([]byte, error) {
	// Encode batch as CBOR array
	cborData, err := cbor.Marshal(receipts)
	if err != nil {
		return nil, fmt.Errorf("CBOR encode batch failed: %w", err)
	}

	c.encMu.Lock()
	compressed := c.encoder.EncodeAll(cborData, nil)
	c.encMu.Unlock()

	// Build output with batch magic
	var buf bytes.Buffer
	buf.Write([]byte{0x4F, 0x43, 0x58, 0x42}) // "OCXB" for batch

	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(len(cborData)))
	buf.Write(sizeBytes)

	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(receipts)))
	buf.Write(countBytes)

	buf.Write(compressed)

	return buf.Bytes(), nil
}

// DecompressReceiptBatch decompresses a batch of receipts
func (c *Compressor) DecompressReceiptBatch(data []byte) ([]*ReceiptFull, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("batch data too short")
	}

	// Check batch magic
	if !bytes.HasPrefix(data, []byte{0x4F, 0x43, 0x58, 0x42}) {
		return nil, fmt.Errorf("invalid batch magic")
	}

	originalSize := binary.BigEndian.Uint32(data[4:8])
	// count := binary.BigEndian.Uint32(data[8:12]) // unused but available
	compressed := data[12:]

	c.decMu.Lock()
	decompressed, err := c.decoder.DecodeAll(compressed, make([]byte, 0, originalSize))
	c.decMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("zstd decompress failed: %w", err)
	}

	var receipts []*ReceiptFull
	if err := cbor.Unmarshal(decompressed, &receipts); err != nil {
		return nil, fmt.Errorf("CBOR decode failed: %w", err)
	}

	return receipts, nil
}

// Close releases compressor resources
func (c *Compressor) Close() error {
	c.encoder.Close()
	c.decoder.Close()
	return nil
}

// Stats returns compression statistics
type CompressionStats struct {
	OriginalSize   int
	CompressedSize int
	Ratio          float64
	Algorithm      string
}

// GetStats returns compression stats for a receipt
func (c *Compressor) GetStats(receipt *ReceiptFull) (*CompressionStats, error) {
	original, err := cbor.Marshal(receipt)
	if err != nil {
		return nil, err
	}

	compressed, err := c.CompressReceipt(receipt)
	if err != nil {
		return nil, err
	}

	return &CompressionStats{
		OriginalSize:   len(original),
		CompressedSize: len(compressed),
		Ratio:          float64(len(original)) / float64(len(compressed)),
		Algorithm:      "zstd",
	}, nil
}
