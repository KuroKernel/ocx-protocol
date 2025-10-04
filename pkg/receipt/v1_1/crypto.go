package v1_1

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// CryptoManager handles all cryptographic operations for receipts
type CryptoManager struct {
	encoder *CanonicalEncoder
}

// NewCryptoManager creates a new crypto manager
func NewCryptoManager() (*CryptoManager, error) {
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		return nil, err
	}

	return &CryptoManager{encoder: encoder}, nil
}

// GenerateKeyPair generates a new Ed25519 key pair
func (cm *CryptoManager) GenerateKeyPair(version uint32) (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Version:    version,
		CreatedAt:  time.Now(),
	}, nil
}

// SignReceipt signs a receipt core with the given private key
func (cm *CryptoManager) SignReceipt(core *ReceiptCore, privateKey ed25519.PrivateKey) ([64]byte, error) {
	// Encode the core with canonical CBOR
	coreData, err := cm.encoder.EncodeCore(core)
	if err != nil {
		return [64]byte{}, fmt.Errorf("failed to encode core: %w", err)
	}

	// Create the message to sign: domain separator + core data
	message := append([]byte(DomainSeparator), coreData...)

	// Sign with Ed25519
	signature := ed25519.Sign(privateKey, message)

	// Convert to fixed-size array
	var sigArray [64]byte
	copy(sigArray[:], signature)

	return sigArray, nil
}

// VerifyReceipt verifies a receipt signature
func (cm *CryptoManager) VerifyReceipt(core *ReceiptCore, signature [64]byte, publicKey ed25519.PublicKey) error {
	// Encode the core with canonical CBOR
	coreData, err := cm.encoder.EncodeCore(core)
	if err != nil {
		return fmt.Errorf("failed to encode core: %w", err)
	}

	// Create the message that was signed: domain separator + core data
	message := append([]byte(DomainSeparator), coreData...)

	// Verify the signature
	valid := ed25519.Verify(publicKey, message, signature[:])
	if !valid {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// GenerateNonce generates a cryptographically secure 16-byte nonce
func (cm *CryptoManager) GenerateNonce() ([16]byte, error) {
	var nonce [16]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		return [16]byte{}, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// CreateReceipt creates a complete receipt with signature
func (cm *CryptoManager) CreateReceipt(
	programHash, inputHash, outputHash [32]byte,
	gasUsed uint64,
	startedAt, finishedAt time.Time,
	issuerID string,
	keyPair *KeyPair,
	hostCycles uint64,
	hostInfo map[string]string,
) (*ReceiptFull, error) {
	// Generate nonce for replay protection
	nonce, err := cm.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create the receipt core
	core := &ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     gasUsed,
		StartedAt:   uint64(startedAt.UnixNano()),
		FinishedAt:  uint64(finishedAt.UnixNano()),
		IssuerID:    issuerID,
		KeyVersion:  keyPair.Version,
		Nonce:       nonce,
		IssuedAt:    uint64(time.Now().UnixNano()),
		FloatMode:   DefaultFloatMode,
	}

	// Sign the receipt
	signature, err := cm.SignReceipt(core, keyPair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign receipt: %w", err)
	}

	// Create the full receipt
	receipt := &ReceiptFull{
		Core:       *core,
		Signature:  signature,
		HostCycles: hostCycles,
		HostInfo:   hostInfo,
	}

	return receipt, nil
}

// PublicKeyToHex converts a public key to hex string
func (cm *CryptoManager) PublicKeyToHex(publicKey ed25519.PublicKey) string {
	return hex.EncodeToString(publicKey)
}

// PublicKeyFromHex converts a hex string to public key
func (cm *CryptoManager) PublicKeyFromHex(hexStr string) (ed25519.PublicKey, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(data))
	}

	return ed25519.PublicKey(data), nil
}

// ValidateReceipt performs comprehensive validation of a receipt
func (cm *CryptoManager) ValidateReceipt(receipt *ReceiptFull, publicKey ed25519.PublicKey, clockSkew time.Duration) (*VerificationInfo, error) {
	info := &VerificationInfo{
		IssuerID:       receipt.Core.IssuerID,
		PublicKey:      cm.PublicKeyToHex(publicKey),
		KeyVersion:     receipt.Core.KeyVersion,
		SignatureValid: false,
		ReplayValid:    false,
		ClockValid:     false,
	}

	// Verify signature
	err := cm.VerifyReceipt(&receipt.Core, receipt.Signature, publicKey)
	if err != nil {
		return info, fmt.Errorf("signature verification failed: %w", err)
	}
	info.SignatureValid = true

	// Verify clock skew
	now := time.Now()
	issuedAt := time.Unix(0, int64(receipt.Core.IssuedAt))
	skew := now.Sub(issuedAt)
	if skew < -clockSkew || skew > clockSkew {
		return info, fmt.Errorf("clock skew too large: %v (max allowed: %v)", skew, clockSkew)
	}
	info.ClockValid = true

	return info, nil
}
