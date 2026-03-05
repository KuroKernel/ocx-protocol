// Package vdf provides VDF (Verifiable Delay Function) temporal proofs for OCX receipts.
// The actual implementation is in vdf_cgo.go (requires Rust FFI) or vdf_stub.go (no-op fallback).
package vdf

// Proof represents a VDF temporal proof result.
type Proof struct {
	Output     []byte // VDF output y = x^(2^T) mod N
	Proof      []byte // Wesolowski proof π
	Iterations uint64 // Number of sequential squarings T
	ModulusID  string // Modulus identifier (e.g., "ocx-vdf-v1")
	DurationMs uint64 // Wall-clock evaluation time in milliseconds
}

// Config controls VDF behavior for the server.
type Config struct {
	Enabled    bool   // Master switch for VDF computation
	Iterations uint64 // Default T (e.g., 100_000 for ~1s)
	ModulusID  string // Which modulus to use (default: "ocx-vdf-v1")
	FailOpen   bool   // Continue without VDF on failure
}

// DefaultConfig returns a reasonable default VDF configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:    false, // Off by default until battle-tested
		Iterations: 100_000,
		ModulusID:  "ocx-vdf-v1",
		FailOpen:   true,
	}
}
