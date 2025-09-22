// Package deterministicvm provides a secure, deterministic execution environment
// for the OCX protocol. It ensures identical artifacts with identical inputs
// produce byte-for-byte identical results across different architectures.
package deterministicvm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// VM defines the interface for different virtual machine backends.
// This allows for future expansion to different execution environments
// (e.g., gVisor, Firecracker, WebAssembly) while maintaining the same API.
type VM interface {
	Run(ctx context.Context, config VMConfig) (*ExecutionResult, error)
}

// VMConfig contains all parameters needed to execute an artifact
// in a deterministic environment.
type VMConfig struct {
	// ArtifactPath is the path to the executable artifact
	ArtifactPath string
	
	// InputData is the raw input bytes to be provided to the artifact
	InputData []byte
	
	// WorkingDir is the directory where execution takes place
	WorkingDir string
	
	// Timeout specifies the maximum execution time allowed
	Timeout time.Duration
	
	// CycleLimit is the maximum number of computational cycles allowed
	CycleLimit uint64
	
	// Env contains environment variables for deterministic execution
	Env []string
	
	// Network controls whether network access is allowed (should be false for determinism)
	Network bool
	
	// MemoryLimit specifies the maximum memory usage in bytes
	MemoryLimit uint64
	
	// Seed for deterministic random number generation
	Seed uint32
}

// ExecutionResult contains all outputs and metadata from artifact execution.
type ExecutionResult struct {
	// ExitCode is the process exit code
	ExitCode int
	
	// Stdout contains all data written to standard output
	Stdout []byte
	
	// Stderr contains all data written to standard error
	Stderr []byte
	
	// CyclesUsed tracks computational cycles consumed
	CyclesUsed uint64
	
	// MemoryUsed tracks peak memory consumption in bytes
	MemoryUsed uint64
	
	// Duration is the wall-clock time taken for execution
	Duration time.Duration
	
	// StartTime is when execution began (UTC)
	StartTime time.Time
	
	// EndTime is when execution completed (UTC)
	EndTime time.Time
}

// ArtifactInfo contains metadata about a resolved artifact.
type ArtifactInfo struct {
	// Hash is the SHA256 hash of the artifact
	Hash [32]byte
	
	// Path is the local filesystem path to the artifact
	Path string
	
	// Size is the artifact size in bytes
	Size int64
	
	// Executable indicates if the artifact has execute permissions
	Executable bool
	
	// Format describes the artifact type (e.g., "elf", "wasm", "script")
	Format string
}

// ExecutionError represents errors that occur during artifact execution.
type ExecutionError struct {
	// Code categorizes the type of error
	Code ErrorCode
	
	// Message provides a human-readable description
	Message string
	
	// Underlying wraps the original error if applicable
	Underlying error
	
	// Context provides additional debugging information
	Context map[string]interface{}
}

func (e *ExecutionError) Error() string {
	if e.Underlying != nil {
		return e.Message + ": " + e.Underlying.Error()
	}
	return e.Message
}

// ErrorCode categorizes different types of execution errors.
type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeArtifactNotFound
	ErrorCodeArtifactInvalid
	ErrorCodeEnvironmentSetup
	ErrorCodeExecution
	ErrorCodeTimeout
	ErrorCodeCycleLimitExceeded
	ErrorCodeMemoryLimitExceeded
	ErrorCodePermissionDenied
	ErrorCodeNetworkViolation
)

// DefaultVMConfig returns a VMConfig with secure, deterministic defaults.
func DefaultVMConfig() VMConfig {
	return VMConfig{
		Timeout:     30 * time.Second,
		CycleLimit:  10_000_000, // Increased for shell script execution
		MemoryLimit: 64 * 1024 * 1024, // 64MB
		Network:     false,
		Env: []string{
			"LC_ALL=C.UTF-8",    // Force consistent locale
			"TZ=UTC",            // Force UTC timezone
			"HOME=/tmp",         // Consistent home directory
			"USER=nobody",       // Consistent user
			"PATH=/usr/bin:/bin", // Minimal, consistent PATH
			"LANG=C.UTF-8",      // Consistent language
		},
	}
}

// =============================================================================
// CANONICAL CBOR UTILITIES
// =============================================================================

// CanonicalCBORMode creates deterministic CBOR encoding
var CanonicalCBORMode, _ = cbor.EncOptions{
	Sort:          cbor.SortCanonical,     // Sort map keys canonically
	ShortestFloat: cbor.ShortestFloat16,   // Use shortest float encoding
	NaNConvert:    cbor.NaNConvert7e00,    // Convert NaN consistently
	InfConvert:    cbor.InfConvertFloat16, // Convert Inf consistently
	IndefLength:   cbor.IndefLengthForbidden, // No indefinite lengths
}.EncMode()

// OCXReceipt represents the receipt structure for canonical CBOR encoding
type OCXReceipt struct {
	SpecHash      [32]byte `cbor:"1,keyasint"`
	ArtifactHash  [32]byte `cbor:"2,keyasint"`
	InputHash     [32]byte `cbor:"3,keyasint"`
	OutputHash    [32]byte `cbor:"4,keyasint"`
	CyclesUsed    uint64   `cbor:"5,keyasint"`
	StartedAt     uint64   `cbor:"6,keyasint"`
	FinishedAt    uint64   `cbor:"7,keyasint"`
	IssuerID      string   `cbor:"8,keyasint"`
	Signature     []byte   `cbor:"9,keyasint"`
}

// CanonicalizeReceipt ensures receipt is in canonical CBOR form
func CanonicalizeReceipt(receipt *OCXReceipt) ([]byte, error) {
	// Encode to canonical CBOR
	canonical, err := CanonicalCBORMode.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("canonical encoding failed: %w", err)
	}
	
	// Verify idempotence: decode and re-encode should be identical
	var decoded OCXReceipt
	if err := cbor.Unmarshal(canonical, &decoded); err != nil {
		return nil, fmt.Errorf("canonical verification failed: %w", err)
	}
	
	reencoded, err := CanonicalCBORMode.Marshal(&decoded)
	if err != nil {
		return nil, fmt.Errorf("re-encoding failed: %w", err)
	}
	
	if !bytes.Equal(canonical, reencoded) {
		return nil, fmt.Errorf("CBOR encoding is not idempotent")
	}
	
	return canonical, nil
}

// toCanonicalCBOR returns the canonical CBOR representation of the receipt
func (r *OCXReceipt) toCanonicalCBOR() []byte {
	canonical, err := CanonicalizeReceipt(r)
	if err != nil {
		// Return empty bytes if canonicalization fails
		return []byte{}
	}
	return canonical
}

// =============================================================================
// DETERMINISTIC RNG UTILITIES
// =============================================================================

// DeterministicRNG provides seeded random number generation
type DeterministicRNG struct {
	seed uint64
	rand *rand.Rand
}

// NewDeterministicRNG creates a new seeded RNG
func NewDeterministicRNG(seed uint64) *DeterministicRNG {
	source := rand.NewSource(int64(seed))
	return &DeterministicRNG{
		seed: seed,
		rand: rand.New(source),
	}
}

// Seed returns the current seed
func (r *DeterministicRNG) Seed() uint64 {
	return r.seed
}

// calculateSeed creates a deterministic seed from artifact and input
func calculateSeed(artifactPath string, input []byte) uint64 {
	h := sha256.New()
	h.Write([]byte(artifactPath))
	h.Write(input)
	h.Write([]byte("OCX-DETERMINISTIC-SEED-V1"))
	
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// =============================================================================
// ENVIRONMENT HASH UTILITIES
// =============================================================================

// calculateEnvironmentHash creates a hash of environment factors that affect determinism
func calculateEnvironmentHash() string {
	h := sha256.New()
	
	// Hash platform information
	h.Write([]byte(runtime.GOOS))
	h.Write([]byte(runtime.GOARCH))
	h.Write([]byte(runtime.Version()))
	h.Write([]byte(getKernelVersionSafe()))
	
	// Hash environment variables that affect execution
	envVars := []string{"PATH", "LC_ALL", "TZ", "HOME", "USER"}
	for _, envVar := range envVars {
		h.Write([]byte(envVar + "=" + os.Getenv(envVar)))
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

