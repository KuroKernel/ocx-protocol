package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	cbor "github.com/fxamacker/cbor/v2"
)

const domain = "OCXv1|receipt|"

// loadPubKey reads a file that can be either base64 of 32 bytes or hex of 32 bytes.
// It returns the raw 32-byte Ed25519 public key.
func loadPubKey(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pubkey: %w", err)
	}
	s := strings.TrimSpace(string(b))

	// Try base64 first
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil && len(raw) == ed25519.PublicKeySize {
		return raw, nil
	}

	// Fallback: hex
	if hexDecoded, err := hex.DecodeString(s); err == nil && len(hexDecoded) == ed25519.PublicKeySize {
		return hexDecoded, nil
	}

	return nil, fmt.Errorf("invalid pubkey format: expected base64 or hex of %d bytes", ed25519.PublicKeySize)
}

type receiptCore struct {
	ProgramHash [32]byte `cbor:"1,keyasint"`
	InputHash   [32]byte `cbor:"2,keyasint"`
	OutputHash  [32]byte `cbor:"3,keyasint"`
	GasUsed     uint64   `cbor:"4,keyasint"`
	StartedAt   uint64   `cbor:"5,keyasint"`
	FinishedAt  uint64   `cbor:"6,keyasint"`
	IssuerID    string   `cbor:"7,keyasint"`
}

type receiptFull struct {
	Core       receiptCore       `cbor:"core"`
	Signature  []byte            `cbor:"signature"`
	HostCycles uint64            `cbor:"host_cycles"`
	HostInfo   map[string]string `cbor:"host_info"`
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <receipt.cbor> <pubkey.b64>", os.Args[0])
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	pub, err := loadPubKey(os.Args[2])
	if err != nil {
		log.Fatalf("pubkey load: %v", err)
	}

	// Strict length check
	if len(pub) != ed25519.PublicKeySize {
		log.Fatalf("pubkey wrong length: got %d, want %d", len(pub), ed25519.PublicKeySize)
	}

	var rf receiptFull
	decm, _ := cbor.DecOptions{TimeTag: cbor.DecTagRequired}.DecMode()
	if err := decm.Unmarshal(b, &rf); err != nil {
		log.Fatalf("decode cbor: %v", err)
	}

	// Use the same canonical encoding as the server
	encOpts := cbor.CanonicalEncOptions()
	encm, err := encOpts.EncMode()
	if err != nil {
		log.Fatalf("failed to create canonical encoder: %v", err)
	}
	coreCBOR, err := encm.Marshal(rf.Core)
	if err != nil {
		log.Fatal(err)
	}

	msg := append([]byte(domain), coreCBOR...)
	ok := ed25519.Verify(ed25519.PublicKey(pub), msg, rf.Signature)
	fmt.Printf("verified=%v\nsha256(core)=%x\n", ok, sha256.Sum256(coreCBOR))
	if !ok {
		os.Exit(1)
	}
}
