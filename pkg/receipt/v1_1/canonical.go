package v1_1

import (
	"crypto/sha256"
	"fmt"

	cbor "github.com/fxamacker/cbor/v2"
)

// CanonicalEncoder provides deterministic CBOR encoding
type CanonicalEncoder struct {
	encMode cbor.EncMode
}

// NewCanonicalEncoder creates a new canonical CBOR encoder
func NewCanonicalEncoder() (*CanonicalEncoder, error) {
	opts := cbor.CanonicalEncOptions()
	opts.Time = cbor.TimeUnix                    // Use Unix timestamps
	opts.Sort = cbor.SortCanonical               // Sort map keys canonically
	opts.IndefLength = cbor.IndefLengthForbidden // Use definite lengths only

	encMode, err := opts.EncMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical encoder: %w", err)
	}

	return &CanonicalEncoder{encMode: encMode}, nil
}

// EncodeCore encodes the receipt core with canonical CBOR
func (ce *CanonicalEncoder) EncodeCore(core *ReceiptCore) ([]byte, error) {
	data, err := ce.encMode.Marshal(core)
	if err != nil {
		return nil, fmt.Errorf("failed to encode receipt core: %w", err)
	}
	return data, nil
}

// EncodeFull encodes the full receipt with canonical CBOR
func (ce *CanonicalEncoder) EncodeFull(receipt *ReceiptFull) ([]byte, error) {
	data, err := ce.encMode.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to encode full receipt: %w", err)
	}
	return data, nil
}

// DecodeCore decodes a receipt core from canonical CBOR
func (ce *CanonicalEncoder) DecodeCore(data []byte) (*ReceiptCore, error) {
	var core ReceiptCore
	err := cbor.Unmarshal(data, &core)
	if err != nil {
		return nil, fmt.Errorf("failed to decode receipt core: %w", err)
	}
	return &core, nil
}

// DecodeFull decodes a full receipt from canonical CBOR
func (ce *CanonicalEncoder) DecodeFull(data []byte) (*ReceiptFull, error) {
	var receipt ReceiptFull
	err := cbor.Unmarshal(data, &receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode full receipt: %w", err)
	}
	return &receipt, nil
}

// ComputeCoreHash computes the SHA-256 hash of the canonical CBOR encoding
func (ce *CanonicalEncoder) ComputeCoreHash(core *ReceiptCore) ([32]byte, error) {
	data, err := ce.EncodeCore(core)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}

// VerifyCanonicalEncoding verifies that the data is canonically encoded
func (ce *CanonicalEncoder) VerifyCanonicalEncoding(data []byte) error {
	// Re-encode and compare to ensure canonical form
	var core ReceiptCore
	if err := cbor.Unmarshal(data, &core); err != nil {
		return fmt.Errorf("invalid CBOR data: %w", err)
	}

	canonical, err := ce.EncodeCore(&core)
	if err != nil {
		return fmt.Errorf("failed to re-encode: %w", err)
	}

	if len(data) != len(canonical) {
		return fmt.Errorf("non-canonical encoding: length mismatch")
	}

	for i, b := range data {
		if b != canonical[i] {
			return fmt.Errorf("non-canonical encoding: byte mismatch at position %d", i)
		}
	}

	return nil
}
