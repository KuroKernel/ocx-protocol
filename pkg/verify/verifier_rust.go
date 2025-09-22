//go:build rust_verifier
// +build rust_verifier

package verify

import (
	"os"
	"strings"
)

// NewVerifier creates the appropriate verifier based on build tags and environment
func NewVerifier() Verifier {
	// Check environment variable to override build-time decision
	if useRust := os.Getenv("OCX_USE_RUST_VERIFIER"); useRust != "" {
		if strings.ToLower(useRust) == "true" || useRust == "1" {
			return NewRustVerifier()
		}
	}

	// Default to Rust verifier when built with rust_verifier tag
	return NewRustVerifier()
}
