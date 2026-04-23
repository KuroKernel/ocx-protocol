// Dumps canonical CBOR hex for a fixed ReceiptCore, for cross-language byte-equality testing.
// Usage:  go run dump_canonical.go
// Output: hex string (one per line) for:
//   1. signed map (keys 1-7), no signature
//   2. transmitted map (keys 1-7 + key 8 = dummy signature of 64 zero bytes)
// The Python side must produce identical bytes when encoding the same data.

package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/fxamacker/cbor/v2"
	"ocx.local/pkg/receipt"
)

func main() {
	// Fixed fixture matching the Python parity script byte-for-byte.
	core := &receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	// (1) Signed bytes: canonical CBOR of map {1..7}
	signedBytes, err := receipt.CanonicalizeCore(core)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CanonicalizeCore failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SIGNED_HEX=%s\n", hex.EncodeToString(signedBytes))

	// (2) Transmitted bytes: same map + key 8 = fake 64-byte signature of zeros.
	// Built by hand so the Python side can mirror it: map with integer keys
	// 1..8 in canonical order.
	fakeSig := make([]byte, 64)
	encOpts := cbor.CanonicalEncOptions()
	em, err := encOpts.EncMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "EncMode failed: %v\n", err)
		os.Exit(1)
	}
	transmittedMap := map[uint64]interface{}{
		1: core.ProgramHash[:],
		2: core.InputHash[:],
		3: core.OutputHash[:],
		4: core.GasUsed,
		5: core.StartedAt,
		6: core.FinishedAt,
		7: core.IssuerID,
		8: fakeSig,
	}
	transmittedBytes, err := em.Marshal(transmittedMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Marshal failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("TRANSMITTED_HEX=%s\n", hex.EncodeToString(transmittedBytes))
}
