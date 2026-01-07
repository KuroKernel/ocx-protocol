package receipt

import (
	"crypto/rand"
	"time"
)

// ReceiptCore represents the signed core fields of a receipt
// ALL these fields are included in the signature - changing any field invalidates the signature
type ReceiptCore struct {
	ProgramHash [32]byte `cbor:"1,keyasint"`  // key 1 - SHA256 of executed program/artifact
	InputHash   [32]byte `cbor:"2,keyasint"`  // key 2 - SHA256 of input data
	OutputHash  [32]byte `cbor:"3,keyasint"`  // key 3 - SHA256 of output data
	GasUsed     uint64   `cbor:"4,keyasint"`  // key 4 - Deterministic gas consumed
	StartedAt   uint64   `cbor:"5,keyasint"`  // key 5 - Execution start (unix nanos)
	FinishedAt  uint64   `cbor:"6,keyasint"`  // key 6 - Execution end (unix nanos)
	IssuerID    string   `cbor:"7,keyasint"`  // key 7 - Issuer identifier
	KeyVersion  uint32   `cbor:"8,keyasint"`  // key 8 - Signing key version for rotation
	Nonce       [16]byte `cbor:"9,keyasint"`  // key 9 - 16-byte nonce for replay protection
	IssuedAt    uint64   `cbor:"10,keyasint"` // key 10 - When receipt was issued (unix nanos)
	FloatMode   string   `cbor:"11,keyasint"` // key 11 - Float handling: "disabled", "soft", "hard"
}

// ReceiptFull represents the complete receipt with metadata
type ReceiptFull struct {
	Core       ReceiptCore       `cbor:"core"`
	Signature  []byte            `cbor:"signature"` // 64B Ed25519 signature
	HostCycles uint64            `cbor:"host_cycles"`
	HostInfo   map[string]string `cbor:"host_info"`
}

// GenerateNonce generates a cryptographically secure 16-byte nonce
func GenerateNonce() ([16]byte, error) {
	var nonce [16]byte
	_, err := rand.Read(nonce[:])
	return nonce, err
}

// NewReceiptCore creates a new ReceiptCore with proper nonce and timestamps
func NewReceiptCore(
	programHash, inputHash, outputHash [32]byte,
	gasUsed uint64,
	startedAt, finishedAt time.Time,
	issuerID string,
	keyVersion uint32,
	floatMode string,
) (*ReceiptCore, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	if floatMode == "" {
		floatMode = "disabled" // Default to disabled for determinism
	}

	return &ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     gasUsed,
		StartedAt:   uint64(startedAt.UnixNano()),
		FinishedAt:  uint64(finishedAt.UnixNano()),
		IssuerID:    issuerID,
		KeyVersion:  keyVersion,
		Nonce:       nonce,
		IssuedAt:    uint64(time.Now().UnixNano()),
		FloatMode:   floatMode,
	}, nil
}

// Default values for replay protection
const (
	DefaultReplayRetention = 7 * 24 * time.Hour // 7 days
	DefaultClockSkew       = 5 * time.Minute    // 5 minutes tolerance
	DefaultFloatMode       = "disabled"
)
