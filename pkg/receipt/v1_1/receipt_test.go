package v1_1

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite for testing
	db, err := sql.Open("postgres", "postgres://ocx:ocx@localhost/ocx_test?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	return db
}

// TestCryptoManager tests the cryptographic operations
func TestCryptoManager(t *testing.T) {
	crypto, err := NewCryptoManager()
	if err != nil {
		t.Fatalf("Failed to create crypto manager: %v", err)
	}

	// Test key generation
	keyPair, err := crypto.GenerateKeyPair(1)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if len(keyPair.PrivateKey) != 64 {
		t.Errorf("Expected private key length 64, got %d", len(keyPair.PrivateKey))
	}

	if len(keyPair.PublicKey) != 32 {
		t.Errorf("Expected public key length 32, got %d", len(keyPair.PublicKey))
	}

	// Test receipt creation
	programHash := sha256.Sum256([]byte("test program"))
	inputHash := sha256.Sum256([]byte("test input"))
	outputHash := sha256.Sum256([]byte("test output"))

	startedAt := time.Now()
	finishedAt := startedAt.Add(100 * time.Millisecond)

	receipt, err := crypto.CreateReceipt(
		programHash, inputHash, outputHash,
		1000, startedAt, finishedAt,
		"test-issuer", keyPair, 5000,
		map[string]string{"hostname": "test-host"},
	)
	if err != nil {
		t.Fatalf("Failed to create receipt: %v", err)
	}

	// Test signature verification
	err = crypto.VerifyReceipt(&receipt.Core, receipt.Signature, keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Receipt verification failed: %v", err)
	}

	// Test nonce generation
	nonce, err := crypto.GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	if len(nonce) != 16 {
		t.Errorf("Expected nonce length 16, got %d", len(nonce))
	}

	// Test public key conversion
	hexKey := crypto.PublicKeyToHex(keyPair.PublicKey)
	recoveredKey, err := crypto.PublicKeyFromHex(hexKey)
	if err != nil {
		t.Fatalf("Failed to recover public key: %v", err)
	}

	if len(recoveredKey) != len(keyPair.PublicKey) {
		t.Errorf("Recovered key length mismatch")
	}
}

// TestCanonicalEncoder tests the canonical CBOR encoding
func TestCanonicalEncoder(t *testing.T) {
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		t.Fatalf("Failed to create canonical encoder: %v", err)
	}

	// Test core encoding
	core := &ReceiptCore{
		ProgramHash: sha256.Sum256([]byte("test")),
		InputHash:   sha256.Sum256([]byte("input")),
		OutputHash:  sha256.Sum256([]byte("output")),
		GasUsed:     1000,
		StartedAt:   uint64(time.Now().UnixNano()),
		FinishedAt:  uint64(time.Now().UnixNano()),
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		IssuedAt:    uint64(time.Now().UnixNano()),
		FloatMode:   "disabled",
	}

	// Encode
	data, err := encoder.EncodeCore(core)
	if err != nil {
		t.Fatalf("Failed to encode core: %v", err)
	}

	// Decode
	decodedCore, err := encoder.DecodeCore(data)
	if err != nil {
		t.Fatalf("Failed to decode core: %v", err)
	}

	// Verify round-trip
	if decodedCore.ProgramHash != core.ProgramHash {
		t.Errorf("Program hash mismatch")
	}

	if decodedCore.IssuerID != core.IssuerID {
		t.Errorf("Issuer ID mismatch")
	}

	// Test canonical encoding verification
	err = encoder.VerifyCanonicalEncoding(data)
	if err != nil {
		t.Errorf("Canonical encoding verification failed: %v", err)
	}
}

// TestReplayProtection tests the replay protection system
func TestReplayProtection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create replay protection table
	err := CreateReplayProtectionTable(db)
	if err != nil {
		t.Fatalf("Failed to create replay protection table: %v", err)
	}

	replay := NewReplayProtection(db, 1*time.Hour, 5*time.Minute)

	ctx := context.Background()
	issuerID := "test-issuer"
	nonce := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	issuedAt := time.Now()

	// First use should succeed
	err = replay.CheckAndRecordNonce(ctx, issuerID, nonce, issuedAt)
	if err != nil {
		t.Fatalf("First nonce use should succeed: %v", err)
	}

	// Second use should fail (replay attack)
	err = replay.CheckAndRecordNonce(ctx, issuerID, nonce, issuedAt)
	if err == nil {
		t.Fatalf("Second nonce use should fail (replay attack)")
	}

	// Different issuer should succeed
	err = replay.CheckAndRecordNonce(ctx, "different-issuer", nonce, issuedAt)
	if err != nil {
		t.Fatalf("Different issuer should succeed: %v", err)
	}

	// Get stats
	stats, err := replay.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_active_nonces"].(int) < 2 {
		t.Errorf("Expected at least 2 active nonces, got %v", stats["total_active_nonces"])
	}
}

// TestKMSManager tests the key management system
func TestKMSManager(t *testing.T) {
	kms := NewKMSManager()

	// Register local provider
	localProvider := NewLocalKMSProvider()
	kms.RegisterProvider("local", localProvider)
	kms.SetDefaultProvider("local")

	ctx := context.Background()
	keyID := "test-key"
	version := uint32(1)

	// Get provider
	provider, err := kms.GetProvider("local")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	// Generate key
	_, err = provider.GenerateKey(ctx, keyID, version)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Get public key
	publicKey, err := provider.GetPublicKey(ctx, keyID, version)
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}

	if len(publicKey) != 32 {
		t.Errorf("Expected public key length 32, got %d", len(publicKey))
	}

	// Test signing
	data := []byte("test data")
	signature, err := provider.Sign(ctx, keyID, version, data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Test verification
	err = provider.Verify(ctx, keyID, version, data, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	// List keys
	keys, err := provider.ListKeys(ctx)
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	if keys[0].KeyID != keyID {
		t.Errorf("Expected key ID %s, got %s", keyID, keys[0].KeyID)
	}
}

// TestReceiptManager tests the complete receipt manager
func TestReceiptManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create tables
	err := CreateReplayProtectionTable(db)
	if err != nil {
		t.Fatalf("Failed to create replay protection table: %v", err)
	}

	manager, err := NewReceiptManager(db)
	if err != nil {
		t.Fatalf("Failed to create receipt manager: %v", err)
	}

	ctx := context.Background()

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()

	// Create a receipt
	programHash := sha256.Sum256([]byte("test program"))
	inputHash := sha256.Sum256([]byte("test input"))
	outputHash := sha256.Sum256([]byte("test output"))

	startedAt := time.Now()
	finishedAt := startedAt.Add(100 * time.Millisecond)

	receipt, err := manager.CreateReceipt(
		ctx,
		programHash, inputHash, outputHash,
		1000, startedAt, finishedAt,
		"test-issuer", "test-key", 1,
		5000, map[string]string{"hostname": "test-host"},
	)
	if err != nil {
		t.Fatalf("Failed to create receipt: %v", err)
	}

	// Verify the receipt
	verification, err := manager.VerifyReceipt(ctx, receipt, "test-key", 1)
	if err != nil {
		t.Fatalf("Failed to verify receipt: %v", err)
	}

	if !verification.SignatureValid {
		t.Errorf("Signature should be valid")
	}

	if !verification.ClockValid {
		t.Errorf("Clock should be valid")
	}

	// Test replay attack
	_, err = manager.CreateReceipt(
		ctx,
		programHash, inputHash, outputHash,
		1000, startedAt, finishedAt,
		"test-issuer", "test-key", 1,
		5000, map[string]string{"hostname": "test-host"},
	)
	if err == nil {
		t.Fatalf("Replay attack should be detected")
	}

	// Get dashboard stats
	stats := manager.GetDashboard().GetStats()
	if stats.TotalReceipts < 1 {
		t.Errorf("Expected at least 1 receipt in stats")
	}
}

// TestCrossArchitectureCompatibility tests that receipts are compatible across architectures
func TestCrossArchitectureCompatibility(t *testing.T) {
	crypto, err := NewCryptoManager()
	if err != nil {
		t.Fatalf("Failed to create crypto manager: %v", err)
	}

	// Create a receipt
	keyPair, err := crypto.GenerateKeyPair(1)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	programHash := sha256.Sum256([]byte("test program"))
	inputHash := sha256.Sum256([]byte("test input"))
	outputHash := sha256.Sum256([]byte("test output"))

	startedAt := time.Unix(1609459200, 0) // Fixed timestamp for reproducibility
	finishedAt := startedAt.Add(100 * time.Millisecond)

	receipt, err := crypto.CreateReceipt(
		programHash, inputHash, outputHash,
		1000, startedAt, finishedAt,
		"test-issuer", keyPair, 5000,
		map[string]string{"hostname": "test-host"},
	)
	if err != nil {
		t.Fatalf("Failed to create receipt: %v", err)
	}

	// Encode to canonical CBOR
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	receiptData, err := encoder.EncodeFull(receipt)
	if err != nil {
		t.Fatalf("Failed to encode receipt: %v", err)
	}

	// Verify the encoded data is deterministic
	receiptData2, err := encoder.EncodeFull(receipt)
	if err != nil {
		t.Fatalf("Failed to encode receipt again: %v", err)
	}

	if len(receiptData) != len(receiptData2) {
		t.Errorf("Encoded data length mismatch")
	}

	for i, b := range receiptData {
		if b != receiptData2[i] {
			t.Errorf("Encoded data mismatch at position %d", i)
		}
	}

	// Test that the receipt can be decoded and verified
	decodedReceipt, err := encoder.DecodeFull(receiptData)
	if err != nil {
		t.Fatalf("Failed to decode receipt: %v", err)
	}

	err = crypto.VerifyReceipt(&decodedReceipt.Core, decodedReceipt.Signature, keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Decoded receipt verification failed: %v", err)
	}

	// Print the receipt data for manual verification
	t.Logf("Receipt data (hex): %s", hex.EncodeToString(receiptData))
	t.Logf("Receipt data (base64): %s", base64.StdEncoding.EncodeToString(receiptData))
}

// BenchmarkReceiptCreation benchmarks receipt creation performance
func BenchmarkReceiptCreation(b *testing.B) {
	crypto, err := NewCryptoManager()
	if err != nil {
		b.Fatalf("Failed to create crypto manager: %v", err)
	}

	keyPair, err := crypto.GenerateKeyPair(1)
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	programHash := sha256.Sum256([]byte("test program"))
	inputHash := sha256.Sum256([]byte("test input"))
	outputHash := sha256.Sum256([]byte("test output"))

	startedAt := time.Now()
	finishedAt := startedAt.Add(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := crypto.CreateReceipt(
			programHash, inputHash, outputHash,
			1000, startedAt, finishedAt,
			"test-issuer", keyPair, 5000,
			map[string]string{"hostname": "test-host"},
		)
		if err != nil {
			b.Fatalf("Failed to create receipt: %v", err)
		}
	}
}

// BenchmarkReceiptVerification benchmarks receipt verification performance
func BenchmarkReceiptVerification(b *testing.B) {
	crypto, err := NewCryptoManager()
	if err != nil {
		b.Fatalf("Failed to create crypto manager: %v", err)
	}

	keyPair, err := crypto.GenerateKeyPair(1)
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	programHash := sha256.Sum256([]byte("test program"))
	inputHash := sha256.Sum256([]byte("test input"))
	outputHash := sha256.Sum256([]byte("test output"))

	startedAt := time.Now()
	finishedAt := startedAt.Add(100 * time.Millisecond)

	receipt, err := crypto.CreateReceipt(
		programHash, inputHash, outputHash,
		1000, startedAt, finishedAt,
		"test-issuer", keyPair, 5000,
		map[string]string{"hostname": "test-host"},
	)
	if err != nil {
		b.Fatalf("Failed to create receipt: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := crypto.VerifyReceipt(&receipt.Core, receipt.Signature, keyPair.PublicKey)
		if err != nil {
			b.Fatalf("Receipt verification failed: %v", err)
		}
	}
}
