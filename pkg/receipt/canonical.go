package receipt

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// CanonicalizeCore encodes only the signed core with canonical CBOR options
func CanonicalizeCore(r *ReceiptCore) ([]byte, error) {
	// Use canonical encoding options
	encOpts := cbor.CanonicalEncOptions()
	encOpts.Time = cbor.TimeUnix // Use Unix time for timestamps
	em, err := encOpts.EncMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical encoder: %w", err)
	}

	// Build map manually with integer keys to guarantee canonical ordering
	m := make(map[uint64]interface{})
	m[1] = r.ProgramHash[:]
	m[2] = r.InputHash[:]
	m[3] = r.OutputHash[:]
	m[4] = r.GasUsed
	m[5] = r.StartedAt
	m[6] = r.FinishedAt
	m[7] = r.IssuerID

	// Include VDF fields if present (v1.2 temporal proof — covered by signature)
	if len(r.VdfOutput) > 0 {
		m[12] = r.VdfOutput
	}
	if len(r.VdfProof) > 0 {
		m[13] = r.VdfProof
	}
	if r.VdfIter > 0 {
		m[14] = r.VdfIter
	}
	if r.VdfModulusID != "" {
		m[15] = r.VdfModulusID
	}

	return em.Marshal(m)
}

// CanonicalizeFull encodes the full receipt (core + metadata) with canonical CBOR
func CanonicalizeFull(r *ReceiptFull) ([]byte, error) {
	// Use canonical encoding options
	encOpts := cbor.CanonicalEncOptions()
	encOpts.Time = cbor.TimeUnix // Use Unix time for timestamps
	em, err := encOpts.EncMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical encoder: %w", err)
	}

	return em.Marshal(r)
}
