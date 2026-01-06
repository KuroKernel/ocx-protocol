// Package chain provides receipt chain verification for OCX Protocol.
//
// Receipt chains enable cryptographic linking of receipts, creating verifiable
// audit trails from raw transactions to complex decisions (e.g., invoice → GST match
// → bank confirmation → AI credit score → loan disbursement).
package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ChainedReceipt represents a receipt with chaining metadata
type ChainedReceipt struct {
	// Core receipt data
	ReceiptHash   [32]byte `json:"receipt_hash"`   // SHA-256 of canonical CBOR
	ArtifactHash  [32]byte `json:"artifact_hash"`  // Program/computation hash
	InputHash     [32]byte `json:"input_hash"`     // Input data hash
	OutputHash    [32]byte `json:"output_hash"`    // Output data hash
	CyclesUsed    uint64   `json:"cycles_used"`    // Computational cycles
	StartedAt     uint64   `json:"started_at"`     // Unix timestamp (seconds)
	FinishedAt    uint64   `json:"finished_at"`    // Unix timestamp (seconds)
	IssuerKeyID   string   `json:"issuer_key_id"`  // Issuer identifier
	Signature     []byte   `json:"signature"`      // Ed25519 signature (64 bytes)

	// Chain fields
	PrevReceiptHash   *[32]byte `json:"prev_receipt_hash,omitempty"`   // Previous receipt in chain
	RequestDigest     *[32]byte `json:"request_digest,omitempty"`      // Original request binding
	WitnessSignatures [][]byte  `json:"witness_signatures,omitempty"`  // Multi-party witnesses

	// Storage metadata
	StoredAt  time.Time `json:"stored_at"`
	ChainID   string    `json:"chain_id,omitempty"`   // Logical chain identifier
	ChainSeq  uint64    `json:"chain_seq,omitempty"`  // Sequence number in chain
}

// ChainVerificationResult contains the result of verifying a receipt chain
type ChainVerificationResult struct {
	Valid           bool                   `json:"valid"`
	ChainLength     int                    `json:"chain_length"`
	GenesisHash     [32]byte               `json:"genesis_hash"`     // First receipt in chain
	HeadHash        [32]byte               `json:"head_hash"`        // Last receipt (the one verified)
	Receipts        []ChainedReceipt       `json:"receipts"`         // All receipts in chain order
	Errors          []ChainVerificationError `json:"errors,omitempty"`
	VerifiedAt      time.Time              `json:"verified_at"`
	VerificationMs  int64                  `json:"verification_ms"`
}

// ChainVerificationError represents an error in chain verification
type ChainVerificationError struct {
	ReceiptHash [32]byte `json:"receipt_hash"`
	Position    int      `json:"position"`    // Position in chain (0 = genesis)
	ErrorType   string   `json:"error_type"`  // "missing", "timestamp", "signature", "hash_mismatch"
	Message     string   `json:"message"`
}

// ChainStats contains statistics about receipt chains
type ChainStats struct {
	TotalChains      int64     `json:"total_chains"`
	TotalReceipts    int64     `json:"total_receipts"`
	LongestChain     int       `json:"longest_chain"`
	AvgChainLength   float64   `json:"avg_chain_length"`
	LastVerifiedAt   time.Time `json:"last_verified_at"`
}

// HashToHex converts a 32-byte hash to hex string
func HashToHex(hash [32]byte) string {
	return hex.EncodeToString(hash[:])
}

// HexToHash converts a hex string to 32-byte hash
func HexToHash(hexStr string) ([32]byte, error) {
	var hash [32]byte
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return hash, fmt.Errorf("invalid hex string: %w", err)
	}
	if len(bytes) != 32 {
		return hash, fmt.Errorf("invalid hash length: expected 32, got %d", len(bytes))
	}
	copy(hash[:], bytes)
	return hash, nil
}

// CalculateReceiptHash computes SHA-256 hash of receipt's canonical CBOR
func CalculateReceiptHash(canonicalCBOR []byte) [32]byte {
	return sha256.Sum256(canonicalCBOR)
}

// IsGenesisReceipt returns true if receipt has no previous receipt
func (r *ChainedReceipt) IsGenesisReceipt() bool {
	return r.PrevReceiptHash == nil
}

// GetPrevHashHex returns hex string of previous receipt hash, or empty string
func (r *ChainedReceipt) GetPrevHashHex() string {
	if r.PrevReceiptHash == nil {
		return ""
	}
	return HashToHex(*r.PrevReceiptHash)
}

// ChainValidationPolicy configures chain verification behavior
type ChainValidationPolicy struct {
	// MaxChainDepth limits how far back to verify (0 = unlimited)
	MaxChainDepth int `json:"max_chain_depth"`

	// RequireContiguousTimestamps enforces prev.finished_at <= curr.started_at
	RequireContiguousTimestamps bool `json:"require_contiguous_timestamps"`

	// AllowMissingAncestors permits verification even if some ancestors aren't stored
	AllowMissingAncestors bool `json:"allow_missing_ancestors"`

	// VerifySignatures re-verifies Ed25519 signatures for each receipt
	VerifySignatures bool `json:"verify_signatures"`

	// MaxClockSkew allows some tolerance in timestamp ordering (seconds)
	MaxClockSkew uint64 `json:"max_clock_skew"`
}

// DefaultValidationPolicy returns a strict validation policy
func DefaultValidationPolicy() ChainValidationPolicy {
	return ChainValidationPolicy{
		MaxChainDepth:               0,     // Unlimited
		RequireContiguousTimestamps: true,
		AllowMissingAncestors:       false,
		VerifySignatures:            true,
		MaxClockSkew:                300,   // 5 minutes
	}
}

// RelaxedValidationPolicy returns a more permissive policy for queries
func RelaxedValidationPolicy() ChainValidationPolicy {
	return ChainValidationPolicy{
		MaxChainDepth:               100,
		RequireContiguousTimestamps: false,
		AllowMissingAncestors:       true,
		VerifySignatures:            false,
		MaxClockSkew:                3600, // 1 hour
	}
}
