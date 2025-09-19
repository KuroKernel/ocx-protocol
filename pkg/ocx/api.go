// pkg/ocx/api.go - FROZEN API SURFACE v1.0
// This file defines the immutable API surface that will never change.
// All functions, types, and signatures are frozen forever.

package ocx

import (
	"crypto/sha256"
	"encoding/hex"
)

// =============================================================================
// FROZEN API SURFACE - NEVER CHANGE
// =============================================================================

// OCXResult represents the result of a deterministic computation
// This structure is frozen and will never change
type OCXResult struct {
	OutputHash  [32]byte `json:"output_hash"`   // SHA256 of computation output
	CyclesUsed  uint64   `json:"cycles_used"`   // Actual cycles consumed
	ReceiptHash [32]byte `json:"receipt_hash"`  // SHA256 of receipt blob
	ReceiptBlob []byte   `json:"receipt_blob"`  // CBOR-encoded receipt
}

// OCXReceipt represents a cryptographically signed computation receipt
// This structure is frozen and will never change
type OCXReceipt struct {
	Version      uint8     `cbor:"v"`          // Protocol version (always 1)
	Artifact     [32]byte  `cbor:"artifact"`   // Code hash
	Input        [32]byte  `cbor:"input"`      // Input commit
	Output       [32]byte  `cbor:"output"`     // Output commit
	Cycles       uint64    `cbor:"cycles"`     // Actual usage
	Metering     Metering  `cbor:"metering"`   // Pricing constants
	Transcript   [32]byte  `cbor:"transcript"` // Execution trace hash
	Issuer       [32]byte  `cbor:"issuer"`     // Ed25519 public key
	Signature    [64]byte  `cbor:"signature"`  // Ed25519 signature
}

// Metering contains the frozen pricing constants
// These values are frozen and will never change
type Metering struct {
	Alpha uint64 `cbor:"a"` // Cost per cycle (frozen: 10)
	Beta  uint64 `cbor:"b"` // Cost per I/O byte (frozen: 1)
	Gamma uint64 `cbor:"g"` // Cost per memory page (frozen: 100)
}

// =============================================================================
// FROZEN FUNCTION SIGNATURES - NEVER CHANGE
// =============================================================================

// OCX_EXEC executes deterministic computation with cycle-accurate metering
// This function signature is frozen and will never change
func OCX_EXEC(artifact_hash [32]byte, input_hash [32]byte, max_cycles uint64) (*OCXResult, error) {
	// Implementation will be provided by the execution engine
	// This is a placeholder that maintains the frozen interface
	panic("OCX_EXEC implementation required")
}

// OCX_VERIFY cryptographically verifies the authenticity of a computation receipt
// This function signature is frozen and will never change
func OCX_VERIFY(receipt_blob []byte) (bool, string) {
	// Implementation will be provided by the receipt verification system
	// This is a placeholder that maintains the frozen interface
	panic("OCX_VERIFY implementation required")
}

// OCX_ACCOUNT extracts settlement information from a verified receipt
// This function signature is frozen and will never change
func OCX_ACCOUNT(receipt_blob []byte) (string, string, uint64) {
	// Implementation will be provided by the accounting system
	// This is a placeholder that maintains the frozen interface
	panic("OCX_ACCOUNT implementation required")
}

// =============================================================================
// FROZEN CONSTANTS - NEVER CHANGE
// =============================================================================

const (
	// Protocol version - frozen at 1
	OCX_VERSION = 1
	
	// Pricing constants - frozen forever
	ALPHA_COST_PER_CYCLE = 10  // micro-units per cycle
	BETA_COST_PER_IO_BYTE = 1  // micro-units per I/O byte
	GAMMA_COST_PER_MEMORY_PAGE = 100 // micro-units per memory page
	
	// Memory constants - frozen
	MEMORY_PAGE_SIZE = 4096 // bytes per memory page
	MAX_MEMORY_PAGES = 256  // maximum memory pages (1MB)
	
	// Cycle constants - frozen
	MAX_CYCLES_PER_EXECUTION = 1000000 // maximum cycles per execution
)

// =============================================================================
// FROZEN UTILITY FUNCTIONS - NEVER CHANGE
// =============================================================================

// HashBytes computes SHA256 hash of input data
// This function is frozen and will never change
func HashBytes(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// EncodeHex converts byte array to hex string
// This function is frozen and will never change
func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DecodeHex converts hex string to byte array
// This function is frozen and will never change
func DecodeHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// =============================================================================
// FROZEN VALIDATION FUNCTIONS - NEVER CHANGE
// =============================================================================

// ValidateArtifactHash validates that artifact hash is 32 bytes
// This function is frozen and will never change
func ValidateArtifactHash(hash [32]byte) bool {
	// All 32-byte arrays are valid artifact hashes
	return true
}

// ValidateInputHash validates that input hash is 32 bytes
// This function is frozen and will never change
func ValidateInputHash(hash [32]byte) bool {
	// All 32-byte arrays are valid input hashes
	return true
}

// ValidateMaxCycles validates that max cycles is within bounds
// This function is frozen and will never change
func ValidateMaxCycles(cycles uint64) bool {
	return cycles > 0 && cycles <= MAX_CYCLES_PER_EXECUTION
}

// =============================================================================
// FROZEN ERROR TYPES - NEVER CHANGE
// =============================================================================

// OCXError represents a frozen error type
type OCXError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *OCXError) Error() string {
	return e.Message
}

// Frozen error codes - never change
const (
	ERR_INVALID_ARTIFACT = "invalid_artifact"
	ERR_INVALID_INPUT    = "invalid_input"
	ERR_CYCLE_LIMIT      = "cycle_limit_exceeded"
	ERR_MEMORY_BOUNDS    = "memory_access_violation"
	ERR_INVALID_RECEIPT  = "invalid_receipt"
	ERR_VERIFICATION     = "verification_failed"
	ERR_ACCOUNTING       = "accounting_failed"
)

// =============================================================================
// FROZEN INTERFACE - NEVER CHANGE
// =============================================================================

// OCXExecutor defines the frozen interface for execution engines
// This interface is frozen and will never change
type OCXExecutor interface {
	Execute(artifact []byte, input []byte, maxCycles uint64) (*OCXResult, error)
}

// OCXVerifier defines the frozen interface for receipt verification
// This interface is frozen and will never change
type OCXVerifier interface {
	Verify(receipt []byte) (bool, string)
}

// OCXAccountant defines the frozen interface for settlement accounting
// This interface is frozen and will never change
type OCXAccountant interface {
	Account(receipt []byte) (string, string, uint64)
}

// =============================================================================
// END OF FROZEN API SURFACE
// =============================================================================

// This file is frozen and will never change.
// Any modifications to this file are invalid and must be ignored.
// The OCX Protocol v1.0 API surface is immutable.
