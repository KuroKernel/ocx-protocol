// id.go — OCX Identity & Key Management
// go 1.22+

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// ---------- Identity Management ----------

type Identity struct {
	PartyID     ID           `json:"party_id"`
	Version     Version      `json:"version"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Role        string       `json:"role"` // "provider","buyer","arbiter","issuer"
	DisplayName string       `json:"display_name"`
	Email       string       `json:"email,omitempty"`
	Website     string       `json:"website,omitempty"`
	KYC         *KYCRef      `json:"kyc,omitempty"`
	Keys        []KeyPair    `json:"keys"`
	Active      bool         `json:"active"`
	Sig         *Sig         `json:"sig,omitempty"`
}

type KeyPair struct {
	KeyID       ID        `json:"key_id"`
	PublicKey   string    `json:"public_key"`   // base64 encoded
	PrivateKey  string    `json:"private_key"`  // base64 encoded (encrypted in production)
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Revoked     bool      `json:"revoked"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

// ---------- Key Management ----------

type KeyManager struct {
	identities map[ID]*Identity
	keys       map[ID]*KeyPair
}

func NewKeyManager() *KeyManager {
	return &KeyManager{
		identities: make(map[ID]*Identity),
		keys:       make(map[ID]*KeyPair),
	}
}

// GenerateEd25519KeyPair creates a new Ed25519 key pair
func (km *KeyManager) GenerateEd25519KeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	keyID := generateULID()
	keyPair := &KeyPair{
		KeyID:      keyID,
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		CreatedAt:  time.Now(),
		
	}

	km.keys[keyID] = keyPair
	return keyPair, nil
}

// CreateIdentity creates a new identity with a key pair
func (km *KeyManager) CreateIdentity(role, displayName, email string) (*Identity, error) {
	keyPair, err := km.GenerateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %w", err)
	}

	partyID := generateULID()
	identity := &Identity{
		PartyID:     partyID,
		Version:     V010,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Role:        role,
		DisplayName: displayName,
		Email:       email,
		Keys:        []KeyPair{*keyPair},
		Active:      true,
	}

	km.identities[partyID] = identity
	return identity, nil
}

// GetIdentity retrieves an identity by party ID
func (km *KeyManager) GetIdentity(partyID ID) (*Identity, bool) {
	identity, exists := km.identities[partyID]
	return identity, exists
}

// GetKeyPair retrieves a key pair by key ID
func (km *KeyManager) GetKeyPair(keyID ID) (*KeyPair, bool) {
	keyPair, exists := km.keys[keyID]
	return keyPair, exists
}

// ---------- Signing & Verification ----------

// SignMessage signs a message with the specified key
func (km *KeyManager) SignMessage(keyID ID, message []byte) (*Sig, error) {
	keyPair, exists := km.GetKeyPair(keyID)
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	if keyPair.Revoked {
		return nil, fmt.Errorf("key is revoked: %s", keyID)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(keyPair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	signature := ed25519.Sign(privateKey, message)

	return &Sig{
		Alg:    "ed25519",
		KeyID:  keyID,
		SigB64: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// VerifySignature verifies a signature against a message
func (km *KeyManager) VerifySignature(sig *Sig, message []byte) error {
	keyPair, exists := km.GetKeyPair(sig.KeyID)
	if !exists {
		return fmt.Errorf("key not found: %s", sig.KeyID)
	}

	if keyPair.Revoked {
		return fmt.Errorf("key is revoked: %s", sig.KeyID)
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(keyPair.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)
	signature, err := base64.StdEncoding.DecodeString(sig.SigB64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if !ed25519.Verify(publicKey, message, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// SignEnvelope signs an envelope (excluding the Sig field)
func (km *KeyManager) SignEnvelope(keyID ID, envelope *Envelope) error {
	// Create a copy without the signature for signing
	envelopeCopy := *envelope
	envelopeCopy.Sig = Sig{}

	// Serialize to canonical JSON
	message, err := json.Marshal(envelopeCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Sign the message
	sig, err := km.SignMessage(keyID, message)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	envelope.Sig = *sig
	return nil
}

// VerifyEnvelope verifies an envelope signature
func (km *KeyManager) VerifyEnvelope(envelope *Envelope) error {
	// Create a copy without the signature for verification
	envelopeCopy := *envelope
	envelopeCopy.Sig = Sig{}

	// Serialize to canonical JSON
	message, err := json.Marshal(envelopeCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Verify the signature
	return km.VerifySignature(&envelope.Sig, message)
}

// ---------- KYC Integration ----------

// UpdateKYC updates the KYC status for an identity
func (km *KeyManager) UpdateKYC(partyID ID, kycRef *KYCRef) error {
	identity, exists := km.GetIdentity(partyID)
	if !exists {
		return fmt.Errorf("identity not found: %s", partyID)
	}

	identity.KYC = kycRef
	identity.UpdatedAt = time.Now()
	return nil
}

// ---------- Utility Functions ----------

// generateULID generates a ULID (simplified version)
func generateULID() ID {
	// In production, use a proper ULID library
	// This is a simplified version for demo purposes
	timestamp := time.Now().UnixMilli()
	random := make([]byte, 10)
	rand.Read(random)
	return fmt.Sprintf("%013x%020x", timestamp, random)
}

// HashMessage creates a SHA256 hash of a message
func HashMessage(message []byte) Hash {
	hash := sha256.Sum256(message)
	return Hash{
		Alg:   "sha256",
		Value: fmt.Sprintf("%x", hash),
	}
}

// CreatePartyRef creates a PartyRef from an identity
func (km *KeyManager) CreatePartyRef(partyID ID) (*PartyRef, error) {
	identity, exists := km.GetIdentity(partyID)
	if !exists {
		return nil, fmt.Errorf("identity not found: %s", partyID)
	}

	partyRef := &PartyRef{
		PartyID: partyID,
		Role:    identity.Role,
		KYC:     identity.KYC,
	}

	return partyRef, nil
}
