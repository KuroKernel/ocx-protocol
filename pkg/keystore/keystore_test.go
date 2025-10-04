package keystore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestKeystoreCreation(t *testing.T) {
	tests := []struct {
		name        string
		keyDir      string
		expectError bool
	}{
		{
			name:        "valid_directory",
			keyDir:      t.TempDir(),
			expectError: false,
		},
		{
			name:        "nonexistent_directory",
			keyDir:      "/nonexistent/path/that/does/not/exist",
			expectError: true,
		},
		{
			name:        "empty_directory",
			keyDir:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, err := New(tt.keyDir)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			if ks == nil {
				t.Errorf("Expected keystore instance for %s, got nil", tt.name)
			}
		})
	}
}

func TestKeyGeneration(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	t.Run("generate_new_key", func(t *testing.T) {
		err := ks.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}

		// Check that a key was generated
		activeKey := ks.GetActiveKey()
		if activeKey == nil {
			t.Error("Expected active key to be set after generation")
		}
	})

	t.Run("generate_multiple_keys", func(t *testing.T) {
		// Generate another key
		err := ks.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate second key: %v", err)
		}

		// Should still have an active key
		activeKey := ks.GetActiveKey()
		if activeKey == nil {
			t.Error("Expected active key to be set after second generation")
		}
	})
}

func TestKeyRetrieval(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}
	keyID := activeKey.ID

	t.Run("get_existing_key", func(t *testing.T) {
		key := ks.GetKey(keyID)
		if key == nil {
			t.Error("Expected to retrieve existing key")
		}
		if key.ID != keyID {
			t.Errorf("Expected key ID %s, got %s", keyID, key.ID)
		}
	})

	t.Run("get_nonexistent_key", func(t *testing.T) {
		key := ks.GetKey("nonexistent-key")
		if key != nil {
			t.Error("Expected nil for nonexistent key")
		}
	})

	t.Run("get_active_key", func(t *testing.T) {
		activeKey := ks.GetActiveKey()
		if activeKey == nil {
			t.Error("Expected active key to be set")
		}
		if activeKey.ID != keyID {
			t.Errorf("Expected active key ID %s, got %s", keyID, activeKey.ID)
		}
	})
}

func TestKeySigning(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}

	message := []byte("test message to sign")

	t.Run("sign_with_existing_key", func(t *testing.T) {
		signature, pubKeyHex, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign message: %v", err)
		}
		if len(signature) != ed25519.SignatureSize {
			t.Errorf("Expected signature size %d, got %d", ed25519.SignatureSize, len(signature))
		}
		if pubKeyHex == "" {
			t.Error("Expected non-empty public key hex")
		}
	})

	t.Run("sign_empty_message", func(t *testing.T) {
		signature, pubKeyHex, err := ks.Sign([]byte{})
		if err != nil {
			t.Fatalf("Failed to sign empty message: %v", err)
		}
		if len(signature) != ed25519.SignatureSize {
			t.Errorf("Expected signature size %d, got %d", ed25519.SignatureSize, len(signature))
		}
		if pubKeyHex == "" {
			t.Error("Expected non-empty public key hex")
		}
	})
}

func TestSignatureVerification(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	message := []byte("test message to sign")
	signature, keyID, err := ks.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Get the public key from the keystore
	key := ks.GetKey(keyID)
	if key == nil {
		t.Fatalf("Failed to get key: %s", keyID)
	}
	pubKeyHex := hex.EncodeToString(key.PublicKey)

	t.Run("verify_valid_signature", func(t *testing.T) {
		valid := VerifySignature(pubKeyHex, message, signature)
		if !valid {
			t.Error("Expected valid signature verification")
		}
	})

	t.Run("verify_invalid_signature", func(t *testing.T) {
		invalidSignature := make([]byte, ed25519.SignatureSize)
		rand.Read(invalidSignature)

		valid := VerifySignature(pubKeyHex, message, invalidSignature)
		if valid {
			t.Error("Expected invalid signature verification")
		}
	})

	t.Run("verify_signature_with_wrong_message", func(t *testing.T) {
		wrongMessage := []byte("wrong message")
		valid := VerifySignature(pubKeyHex, wrongMessage, signature)
		if valid {
			t.Error("Expected invalid signature verification with wrong message")
		}
	})

	t.Run("verify_signature_with_wrong_public_key", func(t *testing.T) {
		_, wrongPubKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate wrong public key: %v", err)
		}

		wrongPubKeyHex := hex.EncodeToString(wrongPubKey)
		valid := VerifySignature(wrongPubKeyHex, message, signature)
		if valid {
			t.Error("Expected invalid signature verification with wrong public key")
		}
	})
}

func TestLocalSigner(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}
	keyID := activeKey.ID

	signer := NewLocalSigner(ks)
	ctx := context.Background()
	message := []byte("test message to sign")

	t.Run("sign_with_local_signer", func(t *testing.T) {
		signature, pubKey, err := signer.Sign(ctx, keyID, message)
		if err != nil {
			t.Fatalf("Failed to sign with local signer: %v", err)
		}
		if len(signature) != ed25519.SignatureSize {
			t.Errorf("Expected signature size %d, got %d", ed25519.SignatureSize, len(signature))
		}
		if len(pubKey) != ed25519.PublicKeySize {
			t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(pubKey))
		}
	})

	t.Run("get_public_key", func(t *testing.T) {
		pubKey, err := signer.PublicKey(ctx, keyID)
		if err != nil {
			t.Fatalf("Failed to get public key: %v", err)
		}
		if len(pubKey) != ed25519.PublicKeySize {
			t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(pubKey))
		}
	})

	t.Run("sign_with_nonexistent_key", func(t *testing.T) {
		_, _, err := signer.Sign(ctx, "nonexistent-key", message)
		if err == nil {
			t.Error("Expected error when signing with nonexistent key")
		}
	})

	t.Run("get_public_key_nonexistent", func(t *testing.T) {
		_, err := signer.PublicKey(ctx, "nonexistent-key")
		if err == nil {
			t.Error("Expected error when getting public key for nonexistent key")
		}
	})
}

func TestDomainSeparator(t *testing.T) {
	t.Run("domain_separator_constant", func(t *testing.T) {
		expected := "OCXv1|receipt|"
		if DomainSeparator != expected {
			t.Errorf("Expected domain separator %q, got %q", expected, DomainSeparator)
		}
	})
}

func TestKeyPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create first keystore and generate a key
	ks1, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create first keystore: %v", err)
	}

	err = ks1.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks1.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}
	keyID := activeKey.ID

	// Create second keystore with same directory
	ks2, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second keystore: %v", err)
	}

	t.Run("key_persistence", func(t *testing.T) {
		key := ks2.GetKey(keyID)
		if key == nil {
			t.Error("Expected to retrieve persisted key")
		}
		if key.ID != keyID {
			t.Errorf("Expected key ID %s, got %s", keyID, key.ID)
		}
	})

	t.Run("signature_consistency", func(t *testing.T) {
		message := []byte("test message")

		// Sign with first keystore
		signature1, _, err := ks1.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign with first keystore: %v", err)
		}

		// Sign with second keystore (should be identical)
		signature2, _, err := ks2.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign with second keystore: %v", err)
		}

		// Signatures should be identical
		if string(signature1) != string(signature2) {
			t.Error("Expected identical signatures from both keystores")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}

	message := []byte("concurrent test message")
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	t.Run("concurrent_signing", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, _, err := ks.Sign(message)
				results <- err
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent signing failed: %v", err)
			}
		}
	})

	t.Run("concurrent_verification", func(t *testing.T) {
		signature, keyID, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to create signature for concurrent test: %v", err)
		}

		// Get the public key from the keystore
		key := ks.GetKey(keyID)
		if key == nil {
			t.Fatalf("Failed to get key: %s", keyID)
		}
		pubKeyHex := hex.EncodeToString(key.PublicKey)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				valid := VerifySignature(pubKeyHex, message, signature)
				if !valid {
					results <- fmt.Errorf("signature verification failed")
					return
				}
				results <- nil
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent verification failed: %v", err)
			}
		}
	})
}

func TestKeyRotation(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Generate first key
	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate first key: %v", err)
	}

	activeKey1 := ks.GetActiveKey()
	if activeKey1 == nil {
		t.Fatalf("Expected active key to be set")
	}

	// Generate second key
	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}

	activeKey2 := ks.GetActiveKey()
	if activeKey2 == nil {
		t.Fatalf("Expected active key to be set")
	}

	message := []byte("key rotation test message")

	t.Run("multiple_keys_different_signatures", func(t *testing.T) {
		// Note: Since the keystore only has one active key at a time,
		// we can't easily test multiple keys. This test is simplified.
		signature1, pubKeyHex1, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign with first key: %v", err)
		}

		// Generate another key
		err = ks.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate second key: %v", err)
		}

		signature2, pubKeyHex2, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign with second key: %v", err)
		}

		// Signatures should be different
		if string(signature1) == string(signature2) {
			t.Error("Expected different signatures from different keys")
		}

		// Public keys should be different
		if pubKeyHex1 == pubKeyHex2 {
			t.Error("Expected different public keys")
		}
	})

	t.Run("cross_key_verification_fails", func(t *testing.T) {
		signature1, _, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign with first key: %v", err)
		}

		// Generate another key
		err = ks.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate second key: %v", err)
		}

		// Get the new public key
		activeKey := ks.GetActiveKey()
		if activeKey == nil {
			t.Fatalf("Expected active key to be set")
		}
		pubKeyHex2 := hex.EncodeToString(activeKey.PublicKey)

		// Verify with wrong public key should fail
		valid := VerifySignature(pubKeyHex2, message, signature1)
		if valid {
			t.Error("Expected cross-key verification to fail")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	ks, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		t.Fatalf("Expected active key to be set")
	}

	t.Run("very_large_message", func(t *testing.T) {
		largeMessage := make([]byte, 1024*1024) // 1MB
		rand.Read(largeMessage)

		signature, keyID, err := ks.Sign(largeMessage)
		if err != nil {
			t.Fatalf("Failed to sign large message: %v", err)
		}

		// Get the public key from the keystore
		key := ks.GetKey(keyID)
		if key == nil {
			t.Fatalf("Failed to get key: %s", keyID)
		}
		pubKeyHex := hex.EncodeToString(key.PublicKey)

		valid := VerifySignature(pubKeyHex, largeMessage, signature)
		if !valid {
			t.Error("Expected valid signature for large message")
		}
	})

	t.Run("zero_length_signature", func(t *testing.T) {
		message := []byte("test message")
		_, keyID, err := ks.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign message: %v", err)
		}

		// Get the public key from the keystore
		key := ks.GetKey(keyID)
		if key == nil {
			t.Fatalf("Failed to get key: %s", keyID)
		}
		pubKeyHex := hex.EncodeToString(key.PublicKey)

		valid := VerifySignature(pubKeyHex, message, []byte{})
		if valid {
			t.Error("Expected invalid signature for zero-length signature")
		}
	})

	t.Run("nil_message", func(t *testing.T) {
		_, _, err := ks.Sign(nil)
		if err != nil {
			t.Fatalf("Failed to sign nil message: %v", err)
		}
	})
}

func BenchmarkKeyGeneration(b *testing.B) {
	ks, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create keystore: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ks.GenerateKey()
		if err != nil {
			b.Fatalf("Failed to generate key: %v", err)
		}
	}
}

func BenchmarkSigning(b *testing.B) {
	ks, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		b.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		b.Fatalf("Expected active key to be set")
	}

	message := []byte("benchmark message")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := ks.Sign(message)
		if err != nil {
			b.Fatalf("Failed to sign message: %v", err)
		}
	}
}

func BenchmarkVerification(b *testing.B) {
	ks, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		b.Fatalf("Failed to generate key: %v", err)
	}

	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		b.Fatalf("Expected active key to be set")
	}

	message := []byte("benchmark message")
	signature, pubKeyHex, err := ks.Sign(message)
	if err != nil {
		b.Fatalf("Failed to create signature: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		valid := VerifySignature(pubKeyHex, message, signature)
		if !valid {
			b.Fatalf("Failed to verify signature")
		}
	}
}
