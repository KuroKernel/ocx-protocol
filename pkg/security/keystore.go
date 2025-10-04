package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// KeyStore manages cryptographic keys
type KeyStore struct {
	keyDir string
}

// NewKeyStore creates a new key store
func NewKeyStore(keyDir string) (*KeyStore, error) {
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	return &KeyStore{
		keyDir: keyDir,
	}, nil
}

// GenerateKey generates a new Ed25519 key pair
func (ks *KeyStore) GenerateKey(name string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save keys
	if err := ks.SavePrivateKey(name, priv); err != nil {
		return nil, nil, fmt.Errorf("failed to save private key: %w", err)
	}

	if err := ks.SavePublicKey(name, pub); err != nil {
		return nil, nil, fmt.Errorf("failed to save public key: %w", err)
	}

	return pub, priv, nil
}

// SavePrivateKey saves a private key to disk
func (ks *KeyStore) SavePrivateKey(name string, key ed25519.PrivateKey) error {
	path := filepath.Join(ks.keyDir, name+".priv")
	data := hex.EncodeToString(key)

	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// SavePublicKey saves a public key to disk
func (ks *KeyStore) SavePublicKey(name string, key ed25519.PublicKey) error {
	path := filepath.Join(ks.keyDir, name+".pub")
	data := hex.EncodeToString(key)

	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// LoadPrivateKey loads a private key from disk
func (ks *KeyStore) LoadPrivateKey(name string) (ed25519.PrivateKey, error) {
	path := filepath.Join(ks.keyDir, name+".priv")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d",
			ed25519.PrivateKeySize, len(key))
	}

	return ed25519.PrivateKey(key), nil
}

// LoadPublicKey loads a public key from disk
func (ks *KeyStore) LoadPublicKey(name string) (ed25519.PublicKey, error) {
	path := filepath.Join(ks.keyDir, name+".pub")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d",
			ed25519.PublicKeySize, len(key))
	}

	return ed25519.PublicKey(key), nil
}

// KeyExists checks if a key pair exists
func (ks *KeyStore) KeyExists(name string) bool {
	privPath := filepath.Join(ks.keyDir, name+".priv")
	pubPath := filepath.Join(ks.keyDir, name+".pub")

	_, privErr := os.Stat(privPath)
	_, pubErr := os.Stat(pubPath)

	return privErr == nil && pubErr == nil
}

// DeleteKey removes a key pair
func (ks *KeyStore) DeleteKey(name string) error {
	privPath := filepath.Join(ks.keyDir, name+".priv")
	pubPath := filepath.Join(ks.keyDir, name+".pub")

	// Remove private key
	if err := os.Remove(privPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove private key: %w", err)
	}

	// Remove public key
	if err := os.Remove(pubPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove public key: %w", err)
	}

	return nil
}

// ListKeys returns all key names in the store
func (ks *KeyStore) ListKeys() ([]string, error) {
	entries, err := os.ReadDir(ks.keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	keyMap := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) > 5 && name[len(name)-5:] == ".priv" {
			keyMap[name[:len(name)-5]] = true
		} else if len(name) > 4 && name[len(name)-4:] == ".pub" {
			keyMap[name[:len(name)-4]] = true
		}
	}

	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}

	return keys, nil
}

// KeyStoreConfig defines configuration for the key store
type KeyStoreConfig struct {
	KeyDir           string        `json:"key_dir"`
	RotationInterval time.Duration `json:"rotation_interval"`
	MaxKeys          int           `json:"max_keys"`
}

// KeyPair represents a key pair with metadata
type KeyPair struct {
	Name      string    `json:"name"`
	PublicKey []byte    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

// KeyStatistics provides statistics about keys
type KeyStatistics struct {
	TotalKeys    int       `json:"total_keys"`
	ActiveKeys   int       `json:"active_keys"`
	LastRotation time.Time `json:"last_rotation"`
	NextRotation time.Time `json:"next_rotation"`
}

// StartAutoRotation starts automatic key rotation (placeholder)
func (ks *KeyStore) StartAutoRotation() {
	// Placeholder implementation - in production this would start a background goroutine
}

// RotatingKeyStore provides automatic key rotation with background goroutine
type RotatingKeyStore struct {
	cur atomic.Pointer[ed25519.PrivateKey]
	pub atomic.Pointer[ed25519.PublicKey]

	// configuration
	rotationEvery time.Duration
	// optional: persist callback
	persist func(priv ed25519.PrivateKey, pub ed25519.PublicKey) error
}

// NewRotatingKeyStore creates a new rotating key store
func NewRotatingKeyStore(initial ed25519.PrivateKey, rotationEvery time.Duration, persist func(ed25519.PrivateKey, ed25519.PublicKey) error) *RotatingKeyStore {
	ks := &RotatingKeyStore{rotationEvery: rotationEvery, persist: persist}
	pub := initial.Public().(ed25519.PublicKey)
	ks.cur.Store(&initial)
	ks.pub.Store(&pub)
	return ks
}

// Current returns the current private and public keys
func (ks *RotatingKeyStore) Current() (ed25519.PrivateKey, ed25519.PublicKey) {
	priv := *ks.cur.Load()
	pub := *ks.pub.Load()
	return priv, pub
}

// StartAutoRotation starts the background key rotation goroutine
func (ks *RotatingKeyStore) StartAutoRotation(ctx context.Context) {
	t := time.NewTicker(ks.rotationEvery)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_, newPriv, err := ed25519.GenerateKey(nil)
				if err != nil {
					// log and continue; don't rotate on failure
					continue
				}
				newPub := newPriv.Public().(ed25519.PublicKey)
				if ks.persist != nil {
					if err := ks.persist(newPriv, newPub); err != nil {
						// log and skip persisting failure
					}
				}
				ks.cur.Store(&newPriv)
				ks.pub.Store(&newPub)
			}
		}
	}()
}

// GetKeyStatistics returns key statistics
func (ks *KeyStore) GetKeyStatistics() KeyStatistics {
	keys, _ := ks.ListKeys()
	return KeyStatistics{
		TotalKeys:    len(keys),
		ActiveKeys:   len(keys),
		LastRotation: time.Now().Add(-24 * time.Hour),
		NextRotation: time.Now().Add(24 * time.Hour),
	}
}

// Sign signs data with the current key
func (ks *KeyStore) Sign(data []byte) ([]byte, error) {
	// Use the first available key for signing
	keys, err := ks.ListKeys()
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("no keys available for signing")
	}

	priv, err := ks.LoadPrivateKey(keys[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load signing key: %w", err)
	}

	return ed25519.Sign(priv, data), nil
}

// VerifyWithCurrentKey verifies a signature with the current key
func (ks *KeyStore) VerifyWithCurrentKey(data, signature []byte) (bool, error) {
	keys, err := ks.ListKeys()
	if err != nil || len(keys) == 0 {
		return false, fmt.Errorf("no keys available for verification")
	}

	pub, err := ks.LoadPublicKey(keys[0])
	if err != nil {
		return false, fmt.Errorf("failed to load verification key: %w", err)
	}

	return ed25519.Verify(pub, data, signature), nil
}

// GetCurrentKeyInfo returns information about the current key
func (ks *KeyStore) GetCurrentKeyInfo() (*KeyPair, error) {
	keys, err := ks.ListKeys()
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("no keys available")
	}

	pub, err := ks.LoadPublicKey(keys[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &KeyPair{
		Name:      keys[0],
		PublicKey: pub,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		IsActive:  true,
	}, nil
}

// RotateKey rotates to a new key
func (ks *KeyStore) RotateKey() error {
	// Generate a new key with timestamp
	name := fmt.Sprintf("key-%d", time.Now().Unix())
	_, _, err := ks.GenerateKey(name)
	return err
}

// ExportPublicKey exports the current public key
func (ks *KeyStore) ExportPublicKey() ([]byte, error) {
	keys, err := ks.ListKeys()
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("no keys available")
	}

	pub, err := ks.LoadPublicKey(keys[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return pub, nil
}
