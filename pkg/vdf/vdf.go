// Package vdf provides Go bindings to the Rust VDF (Verifiable Delay Function)
// implementation via CGO/FFI. It wraps the Wesolowski VDF construction for
// computing and verifying temporal proofs in OCX receipts.
package vdf

/*
#cgo LDFLAGS: -L${SRCDIR}/../../libocx-verify/target/release -llibocx_verify -ldl -lm
#include "../../libocx-verify/ocx_verify.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Proof represents a VDF temporal proof result.
type Proof struct {
	Output    []byte // VDF output y = x^(2^T) mod N
	Proof     []byte // Wesolowski proof π
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

// Evaluate computes a VDF temporal proof for the given 32-byte receipt hash.
// This is intentionally slow (~1s for T=100,000) and non-parallelizable.
func Evaluate(receiptHash [32]byte, iterations uint64) (*Proof, error) {
	var cProof C.OcxVdfProof

	result := C.ocx_vdf_evaluate(
		(*C.uint8_t)(unsafe.Pointer(&receiptHash[0])),
		C.uint64_t(iterations),
		&cProof,
	)

	if result != C.OCX_SUCCESS {
		return nil, fmt.Errorf("VDF evaluate failed with error code %d", int(result))
	}

	// Extract output bytes (right-aligned in 256-byte buffer)
	outputLen := int(cProof.output_len)
	output := make([]byte, outputLen)
	copy(output, C.GoBytes(unsafe.Pointer(&cProof.output[256-outputLen]), C.int(outputLen)))

	// Extract proof bytes (right-aligned in 256-byte buffer)
	proofLen := int(cProof.proof_len)
	proof := make([]byte, proofLen)
	copy(proof, C.GoBytes(unsafe.Pointer(&cProof.proof[256-proofLen]), C.int(proofLen)))

	// Extract modulus ID (null-terminated string)
	modulusID := C.GoString((*C.char)(unsafe.Pointer(&cProof.modulus_id[0])))

	return &Proof{
		Output:     output,
		Proof:      proof,
		Iterations: uint64(cProof.iterations),
		ModulusID:  modulusID,
		DurationMs: uint64(cProof.duration_ms),
	}, nil
}

// Verify checks a VDF proof against a 32-byte receipt hash.
// This is fast (~10ms) regardless of the original iteration count.
func Verify(receiptHash [32]byte, p *Proof) (bool, error) {
	if p == nil {
		return false, fmt.Errorf("nil proof")
	}

	// Build the C proof struct
	var cProof C.OcxVdfProof

	// Copy output (right-aligned in 256-byte buffer)
	if len(p.Output) > 256 {
		return false, fmt.Errorf("output too large: %d bytes", len(p.Output))
	}
	for i, b := range p.Output {
		cProof.output[256-len(p.Output)+i] = C.uint8_t(b)
	}
	cProof.output_len = C.uint32_t(len(p.Output))

	// Copy proof (right-aligned in 256-byte buffer)
	if len(p.Proof) > 256 {
		return false, fmt.Errorf("proof too large: %d bytes", len(p.Proof))
	}
	for i, b := range p.Proof {
		cProof.proof[256-len(p.Proof)+i] = C.uint8_t(b)
	}
	cProof.proof_len = C.uint32_t(len(p.Proof))

	cProof.iterations = C.uint64_t(p.Iterations)

	// Copy modulus ID
	modulusBytes := []byte(p.ModulusID)
	if len(modulusBytes) >= 64 {
		return false, fmt.Errorf("modulus ID too long: %d bytes", len(modulusBytes))
	}
	for i, b := range modulusBytes {
		cProof.modulus_id[i] = C.uint8_t(b)
	}

	valid := C.ocx_vdf_verify(
		(*C.uint8_t)(unsafe.Pointer(&receiptHash[0])),
		&cProof,
	)

	return bool(valid), nil
}
