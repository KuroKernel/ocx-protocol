//go:build !cgo

package vdf

import "fmt"

// Evaluate is a stub when built without CGO (no Rust FFI available).
// Returns an error indicating VDF is not available in this build.
func Evaluate(receiptHash [32]byte, iterations uint64) (*Proof, error) {
	return nil, fmt.Errorf("VDF not available: built without CGO/Rust FFI")
}

// Verify is a stub when built without CGO (no Rust FFI available).
// Returns an error indicating VDF is not available in this build.
func Verify(receiptHash [32]byte, p *Proof) (bool, error) {
	return false, fmt.Errorf("VDF not available: built without CGO/Rust FFI")
}
