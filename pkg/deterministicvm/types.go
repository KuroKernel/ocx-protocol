// Package deterministicvm provides a secure, deterministic execution environment
// for the OCX protocol. It ensures identical artifacts with identical inputs
// produce byte-for-byte identical results across different architectures.
package deterministicvm

import (
	"context"
	"time"
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
