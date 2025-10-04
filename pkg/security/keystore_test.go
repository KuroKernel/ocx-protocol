package security

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestNewKeyStore(t *testing.T) {
	tmpDir := t.TempDir()
	
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	if ks.keyDir != tmpDir {
		t.Errorf("Expected keyDir %s, got %s", tmpDir, ks.keyDir)
	}
}

func TestGenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	pub, priv, err := ks.GenerateKey("test-key")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Verify key sizes
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("Invalid public key size: %d", len(pub))
	}
	
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("Invalid private key size: %d", len(priv))
	}
	
	// Verify files were created
	if !ks.KeyExists("test-key") {
		t.Error("Key files were not created")
	}
}

func TestGenerateKeyWithInvalidDir(t *testing.T) {
	// Try to create keystore in unwritable location
	_, err := NewKeyStore("/proc/invalid-location/keys")
	if err == nil {
		t.Fatal("Expected error when creating keystore in invalid location")
	}
}

func TestSaveAndLoadKeys(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Generate test keys
	pub, priv, err := ks.GenerateKey("test-save-load")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Load keys back
	loadedPriv, err := ks.LoadPrivateKey("test-save-load")
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}
	
	loadedPub, err := ks.LoadPublicKey("test-save-load")
	if err != nil {
		t.Fatalf("Failed to load public key: %v", err)
	}
	
	// Verify keys match
	if string(priv) != string(loadedPriv) {
		t.Error("Loaded private key doesn't match original")
	}
	
	if string(pub) != string(loadedPub) {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestLoadNonexistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	_, err = ks.LoadPrivateKey("nonexistent")
	if err == nil {
		t.Fatal("Expected error when loading nonexistent key")
	}
}

func TestLoadCorruptedKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Write corrupted key file
	corruptedPath := filepath.Join(tmpDir, "corrupted.priv")
	if err := os.WriteFile(corruptedPath, []byte("not-hex-data!"), 0600); err != nil {
		t.Fatalf("Failed to write corrupted key: %v", err)
	}
	
	_, err = ks.LoadPrivateKey("corrupted")
	if err == nil {
		t.Fatal("Expected error when loading corrupted key")
	}
}

func TestKeyExists(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Should not exist initially
	if ks.KeyExists("test-exists") {
		t.Error("Key should not exist initially")
	}
	
	// Generate key
	_, _, err = ks.GenerateKey("test-exists")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Should exist now
	if !ks.KeyExists("test-exists") {
		t.Error("Key should exist after generation")
	}
}

func TestDeleteKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Generate key
	_, _, err = ks.GenerateKey("test-delete")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Delete key
	if err := ks.DeleteKey("test-delete"); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}
	
	// Verify deletion
	if ks.KeyExists("test-delete") {
		t.Error("Key should not exist after deletion")
	}
}

func TestDeleteNonexistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Should not error when deleting nonexistent key
	if err := ks.DeleteKey("nonexistent"); err != nil {
		t.Errorf("Unexpected error when deleting nonexistent key: %v", err)
	}
}

func TestListKeys(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := NewKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	// Should be empty initially
	keys, err := ks.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
	
	// Generate multiple keys
	keyNames := []string{"key1", "key2", "key3"}
	for _, name := range keyNames {
		if _, _, err := ks.GenerateKey(name); err != nil {
			t.Fatalf("Failed to generate key %s: %v", name, err)
		}
	}
	
	// List keys
	keys, err = ks.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	
	if len(keys) != len(keyNames) {
		t.Errorf("Expected %d keys, got %d", len(keyNames), len(keys))
	}
	
	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	
	for _, name := range keyNames {
		if !keyMap[name] {
			t.Errorf("Key %s not found in list", name)
		}
	}
}