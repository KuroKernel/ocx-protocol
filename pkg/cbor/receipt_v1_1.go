// pkg/cbor/receipt_v11.go - OCX CBOR v1.1 Standard Enhancement
// This file enhances the existing receipt format with moat fields (backward compatible)

package cbor

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// WitnessSignature represents a witness signature with metadata
type WitnessSignature struct {
	WitnessID string    `cbor:"1,keyasint" json:"witness_id"`
	PublicKey []byte    `cbor:"2,keyasint" json:"public_key"`
	Signature []byte    `cbor:"3,keyasint" json:"signature"`
	Timestamp time.Time `cbor:"4,keyasint" json:"timestamp"`
	Domain    string    `cbor:"5,keyasint,omitempty" json:"domain,omitempty"`
}

// ReceiptV11 represents OCX-CBOR v1.1 with enhanced fields
type ReceiptV11 struct {
	// Core fields (v1.0 compatibility)
	Artifact []byte `cbor:"1,keyasint" json:"artifact"`
	Input    []byte `cbor:"2,keyasint" json:"input"`
	Cycles   uint64 `cbor:"3,keyasint" json:"cycles"`

	// Enhanced fields (v1.1)
	PrevReceiptHash    *[32]byte           `cbor:"4,keyasint,omitempty" json:"prev_receipt_hash,omitempty"`
	RequestDigest      *[32]byte           `cbor:"5,keyasint,omitempty" json:"request_digest,omitempty"`
	WitnessSignatures  []WitnessSignature  `cbor:"6,keyasint,omitempty" json:"witness_signatures,omitempty"`
	
	// Metadata
	Version     string    `cbor:"7,keyasint" json:"version"`
	CreatedAt   time.Time `cbor:"8,keyasint" json:"created_at"`
	IssuerKeyID string    `cbor:"9,keyasint" json:"issuer_key_id"`
	Signature   []byte    `cbor:"10,keyasint" json:"signature"`
}

// CreateRequestDigest generates SHA-256 hash of raw HTTP request body
func CreateRequestDigest(requestBody []byte) [32]byte {
	return sha256.Sum256(requestBody)
}

// CreateReceiptHash generates SHA-256 hash of entire receipt CBOR for chaining
func (r *ReceiptV11) CreateReceiptHash() ([32]byte, error) {
	cborData, err := r.MarshalCBOR()
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(cborData), nil
}

// ValidateWitnessSignatures verifies all witness signatures against trusted witness set
func (r *ReceiptV11) ValidateWitnessSignatures(trustedWitnesses map[string][]byte) error {
	if len(r.WitnessSignatures) == 0 {
		return nil // No witnesses to validate
	}

	signedData, err := r.GetSignedData()
	if err != nil {
		return err
	}

	for _, witness := range r.WitnessSignatures {
		trustedKey, exists := trustedWitnesses[witness.WitnessID]
		if !exists {
			return fmt.Errorf("untrusted witness: %s", witness.WitnessID)
		}

		if !bytes.Equal(witness.PublicKey, trustedKey) {
			return fmt.Errorf("witness public key mismatch: %s", witness.WitnessID)
		}

		if !ed25519.Verify(witness.PublicKey, signedData, witness.Signature) {
			return fmt.Errorf("invalid witness signature: %s", witness.WitnessID)
		}
	}

	return nil
}

// GetSignedData returns the data that should be signed (all fields except signature)
func (r *ReceiptV11) GetSignedData() ([]byte, error) {
	// Create a copy without the signature field
	signature := r.Signature
	r.Signature = nil
	
	// Marshal to JSON for signing
	data, err := json.Marshal(r)
	
	// Restore signature
	r.Signature = signature
	
	return data, err
}

// MarshalCBOR converts the receipt to CBOR bytes
func (r *ReceiptV11) MarshalCBOR() ([]byte, error) {
	// This would use a CBOR library like fxamacker/cbor
	// For now, return JSON marshaled data as placeholder
	return json.Marshal(r)
}

// VerifySignature verifies the Ed25519 signature
func (r *ReceiptV11) VerifySignature(publicKey []byte) bool {
	signedData, err := r.GetSignedData()
	if err != nil {
		return false
	}
	
	return ed25519.Verify(publicKey, signedData, r.Signature)
}

// IsChained returns true if this receipt is part of a chain
func (r *ReceiptV11) IsChained() bool {
	return r.PrevReceiptHash != nil
}

// HasWitness returns true if this receipt has witness signatures
func (r *ReceiptV11) HasWitness() bool {
	return len(r.WitnessSignatures) > 0
}

// GetChainDepth returns the depth of the receipt chain
func (r *ReceiptV11) GetChainDepth() int {
	if !r.IsChained() {
		return 1
	}
	// In a real implementation, this would traverse the chain
	return 2
}
