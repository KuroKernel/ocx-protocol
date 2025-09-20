package keystore

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DomainSeparator = "OCX\x00RECEIPT\x00v1"

type KeyMetadata struct {
	ID        string    `json:"id"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // "active" or "revoked"
}

type Keystore struct {
	dir        string
	activeKeys map[string]*Key
}

type Key struct {
	ID         string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	Metadata   KeyMetadata
}

func New(dir string) (*Keystore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %v", err)
	}
	
	ks := &Keystore{
		dir:        dir,
		activeKeys: make(map[string]*Key),
	}
	
	if err := ks.loadKeys(); err != nil {
		return nil, fmt.Errorf("failed to load keys: %v", err)
	}
	
	// Create default key if none exist
	if len(ks.activeKeys) == 0 {
		if err := ks.GenerateKey(); err != nil {
			return nil, fmt.Errorf("failed to generate default key: %v", err)
		}
	}
	
	return ks, nil
}

func (ks *Keystore) GenerateKey() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}
	
	keyID := hex.EncodeToString(publicKey[:8]) // First 8 bytes as ID
	
	key := &Key{
		ID:         keyID,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Metadata: KeyMetadata{
			ID:        keyID,
			PublicKey: hex.EncodeToString(publicKey),
			CreatedAt: time.Now().UTC(),
			Status:    "active",
		},
	}
	
	if err := ks.saveKey(key); err != nil {
		return fmt.Errorf("failed to save key: %v", err)
	}
	
	ks.activeKeys[keyID] = key
	return nil
}

func (ks *Keystore) GetActiveKey() *Key {
	// Return the most recently created active key
	var newest *Key
	for _, key := range ks.activeKeys {
		if key.Metadata.Status == "active" {
			if newest == nil || key.Metadata.CreatedAt.After(newest.Metadata.CreatedAt) {
				newest = key
			}
		}
	}
	return newest
}

func (ks *Keystore) GetKey(id string) *Key {
	return ks.activeKeys[id]
}

func (ks *Keystore) RevokeKey(id string) error {
	key := ks.activeKeys[id]
	if key == nil {
		return fmt.Errorf("key not found: %s", id)
	}
	
	key.Metadata.Status = "revoked"
	
	if err := ks.saveKey(key); err != nil {
		return fmt.Errorf("failed to save revoked key: %v", err)
	}
	
	delete(ks.activeKeys, id)
	return nil
}

func (ks *Keystore) ListKeys() []KeyMetadata {
	var keys []KeyMetadata
	
	// Load all keys from disk
	files, err := filepath.Glob(filepath.Join(ks.dir, "*.json"))
	if err != nil {
		return keys
	}
	
	for _, file := range files {
		var metadata KeyMetadata
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}
		
		keys = append(keys, metadata)
	}
	
	return keys
}

func (ks *Keystore) Sign(message []byte) ([]byte, string, error) {
	key := ks.GetActiveKey()
	if key == nil {
		return nil, "", fmt.Errorf("no active key available")
	}
	
	// Add domain separator
	toSign := append([]byte(DomainSeparator), message...)
	signature := ed25519.Sign(key.PrivateKey, toSign)
	
	return signature, key.ID, nil
}

func VerifySignature(publicKeyHex string, message []byte, signature []byte) bool {
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false
	}
	
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	
	// Add domain separator
	toVerify := append([]byte(DomainSeparator), message...)
	
	return ed25519.Verify(publicKey, toVerify, signature)
}

func (ks *Keystore) saveKey(key *Key) error {
	metadataPath := filepath.Join(ks.dir, fmt.Sprintf("%s.json", key.ID))
	keyPath := filepath.Join(ks.dir, fmt.Sprintf("%s.key", key.ID))
	
	// Save metadata
	metadataData, err := json.MarshalIndent(key.Metadata, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(metadataPath, metadataData, 0600); err != nil {
		return err
	}
	
	// Save private key
	keyData := hex.EncodeToString(key.PrivateKey)
	if err := os.WriteFile(keyPath, []byte(keyData), 0600); err != nil {
		return err
	}
	
	return nil
}

func (ks *Keystore) loadKeys() error {
	files, err := filepath.Glob(filepath.Join(ks.dir, "*.json"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		var metadata KeyMetadata
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}
		
		if metadata.Status != "active" {
			continue
		}
		
		// Load private key
		keyPath := filepath.Join(ks.dir, fmt.Sprintf("%s.key", metadata.ID))
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		
		privateKey, err := hex.DecodeString(string(keyData))
		if err != nil {
			continue
		}
		
		publicKey, err := hex.DecodeString(metadata.PublicKey)
		if err != nil {
			continue
		}
		
		key := &Key{
			ID:         metadata.ID,
			PublicKey:  publicKey,
			PrivateKey: privateKey,
			Metadata:   metadata,
		}
		
		ks.activeKeys[metadata.ID] = key
	}
	
	return nil
}
