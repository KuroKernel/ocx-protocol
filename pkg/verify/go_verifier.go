//go:build !rust_verifier
// +build !rust_verifier

package verify

import (
	"crypto/ed25519"
	"fmt"
	"ocx.local/pkg/receipt"

	"github.com/fxamacker/cbor/v2"
)

// GoVerifier implements receipt verification using pure Go
type GoVerifier struct {
	// Implementation uses Go-based verification logic
}

// NewGoVerifier creates a new Go-based verifier
func NewGoVerifier() Verifier {
	return &GoVerifier{}
}

// VerifyReceipt verifies a receipt using Go implementation
func (gv *GoVerifier) VerifyReceipt(receiptData []byte, publicKey []byte) (*receipt.ReceiptCore, error) {
	core, err := gv.verifyReceiptInternal(receiptData, publicKey)
	if err != nil {
		return nil, err
	}

	// Validate invariants
	if err := gv.validateInvariants(core); err != nil {
		return nil, fmt.Errorf("invariant validation failed: %w", err)
	}

	return core, nil
}

// VerifyReceiptSimple verifies a receipt using embedded key ID
func (gv *GoVerifier) VerifyReceiptSimple(receiptData []byte) error {
	// This method requires a public key to be provided
	// Note: Key ID extraction from receipt would be implemented for full verification
	return fmt.Errorf("VerifyReceiptSimple requires public key extraction - use VerifyReceipt")
}

// ExtractReceiptFields extracts receipt fields using Go implementation
func (gv *GoVerifier) ExtractReceiptFields(receiptData []byte) (*ReceiptFields, error) {
	var receiptFull receipt.ReceiptFull

	// Decode full CBOR
	decOpts := cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}
	dm, err := decOpts.DecMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	err = dm.Unmarshal(receiptData, &receiptFull)
	if err != nil {
		return nil, fmt.Errorf("failed to decode receipt: %w", err)
	}

	return &ReceiptFields{
		ProgramHash: receiptFull.Core.ProgramHash[:],
		InputHash:   receiptFull.Core.InputHash[:],
		OutputHash:  receiptFull.Core.OutputHash[:],
		GasUsed:     receiptFull.Core.GasUsed,
		StartedAt:   receiptFull.Core.StartedAt,
		FinishedAt:  receiptFull.Core.FinishedAt,
		IssuerID:    receiptFull.Core.IssuerID,
		Signature:   receiptFull.Signature,
		HostCycles:  receiptFull.HostCycles,
		HostInfo:    receiptFull.HostInfo,
	}, nil
}

// BatchVerify verifies multiple receipts
func (gv *GoVerifier) BatchVerify(receipts []ReceiptBatch) ([]bool, error) {
	results := make([]bool, len(receipts))

	for i, batch := range receipts {
		_, err := gv.VerifyReceipt(batch.ReceiptData, batch.PublicKey)
		results[i] = (err == nil)
	}

	return results, nil
}

// GetVersion returns the Go implementation version
func (gv *GoVerifier) GetVersion() (string, error) {
	return "go-1.0.0", nil
}

// verifyReceiptInternal performs the core verification logic
func (gv *GoVerifier) verifyReceiptInternal(receiptData []byte, publicKey []byte) (*receipt.ReceiptCore, error) {
	var receiptFull receipt.ReceiptFull

	// Decode full CBOR
	decOpts := cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}
	dm, err := decOpts.DecMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	err = dm.Unmarshal(receiptData, &receiptFull)
	if err != nil {
		return nil, fmt.Errorf("failed to decode receipt: %w", err)
	}

	// Validate public key length
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	// Validate signature length
	if len(receiptFull.Signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid signature length: expected %d, got %d", ed25519.SignatureSize, len(receiptFull.Signature))
	}

	// Reconstruct core bytes for verification
	coreBytes, err := receipt.CanonicalizeCore(&receiptFull.Core)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize core: %w", err)
	}

	// Create signing message with domain separator
	coreBytesToVerify := append([]byte("OCXv1|receipt|"), coreBytes...)

	// Verify Ed25519 signature
	pubKey := ed25519.PublicKey(publicKey)
	if !ed25519.Verify(pubKey, coreBytesToVerify, receiptFull.Signature) {
		return nil, fmt.Errorf("signature verification failed")
	}

	return &receiptFull.Core, nil
}

// validateInvariants validates receipt invariants
func (gv *GoVerifier) validateInvariants(core *receipt.ReceiptCore) error {
	// Validate timestamps are monotonic
	if core.FinishedAt < core.StartedAt {
		return fmt.Errorf("finished_at (%d) must be >= started_at (%d)", core.FinishedAt, core.StartedAt)
	}

	// Validate hash lengths
	if len(core.ProgramHash) != 32 {
		return fmt.Errorf("program_hash must be 32 bytes, got %d", len(core.ProgramHash))
	}
	if len(core.InputHash) != 32 {
		return fmt.Errorf("input_hash must be 32 bytes, got %d", len(core.InputHash))
	}
	if len(core.OutputHash) != 32 {
		return fmt.Errorf("output_hash must be 32 bytes, got %d", len(core.OutputHash))
	}

	// Validate gas usage
	if core.GasUsed == 0 {
		return fmt.Errorf("gas_used must be > 0")
	}

	// Validate issuer ID
	if core.IssuerID == "" {
		return fmt.Errorf("issuer_id must not be empty")
	}

	return nil
}
