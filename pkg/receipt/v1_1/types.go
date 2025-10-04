package v1_1

import (
	"crypto/ed25519"
	"time"
)

// ReceiptCore represents the core data that gets cryptographically signed
type ReceiptCore struct {
	ProgramHash [32]byte `cbor:"1,keyasint"`  // SHA-256 of the executed program
	InputHash   [32]byte `cbor:"2,keyasint"`  // SHA-256 of the input data
	OutputHash  [32]byte `cbor:"3,keyasint"`  // SHA-256 of the output data
	GasUsed     uint64   `cbor:"4,keyasint"`  // Gas units consumed
	StartedAt   uint64   `cbor:"5,keyasint"`  // Unix timestamp (nanoseconds)
	FinishedAt  uint64   `cbor:"6,keyasint"`  // Unix timestamp (nanoseconds)
	IssuerID    string   `cbor:"7,keyasint"`  // Issuer identifier
	KeyVersion  uint32   `cbor:"8,keyasint"`  // Key version for rotation
	Nonce       [16]byte `cbor:"9,keyasint"`  // 16-byte nonce for replay protection
	IssuedAt    uint64   `cbor:"10,keyasint"` // Unix timestamp when receipt was issued
	FloatMode   string   `cbor:"11,keyasint"` // Optional: "disabled", "soft", "hard"
}

// ReceiptFull represents the complete receipt with signature and metadata
type ReceiptFull struct {
	Core       ReceiptCore       `cbor:"core"`
	Signature  [64]byte          `cbor:"signature"` // Ed25519 signature
	HostCycles uint64            `cbor:"host_cycles"`
	HostInfo   map[string]string `cbor:"host_info"`
}

// ReceiptMetadata contains additional information not part of the signed core
type ReceiptMetadata struct {
	ReceiptID    string            `json:"receipt_id"`
	CreatedAt    time.Time         `json:"created_at"`
	Verified     bool              `json:"verified"`
	Verification *VerificationInfo `json:"verification,omitempty"`
}

// VerificationInfo contains verification details
type VerificationInfo struct {
	IssuerID       string `json:"issuer_id"`
	PublicKey      string `json:"public_key"` // Base64 encoded
	KeyVersion     uint32 `json:"key_version"`
	SignatureValid bool   `json:"signature_valid"`
	ReplayValid    bool   `json:"replay_valid"`
	ClockValid     bool   `json:"clock_valid"`
}

// KeyPair represents an Ed25519 key pair with versioning
type KeyPair struct {
	PrivateKey ed25519.PrivateKey `json:"-"`
	PublicKey  ed25519.PublicKey  `json:"public_key"`
	Version    uint32             `json:"version"`
	CreatedAt  time.Time          `json:"created_at"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
}

// ReplayEntry represents a stored nonce for replay protection
type ReplayEntry struct {
	IssuerID  string    `json:"issuer_id"`
	Nonce     [16]byte  `json:"nonce"`
	UsedAt    time.Time `json:"used_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Domain separator for cryptographic signing
const DomainSeparator = "OCXv1|receipt|"

// Default values
const (
	DefaultReplayRetention = 7 * 24 * time.Hour // 7 days
	DefaultClockSkew       = 5 * time.Minute    // 5 minutes
	DefaultFloatMode       = "disabled"
)
