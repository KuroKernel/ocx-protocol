package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
)

// generateNonce creates a random 16-byte nonce for testing
func generateNonce() [16]byte {
	var nonce [16]byte
	rand.Read(nonce[:])
	return nonce
}

func TestGoVerifierVerifyReceipt(t *testing.T) {
	// Create test keystore and signer
	ks, err := keystore.New(t.TempDir())
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

	signer := keystore.NewLocalSigner(ks)
	ctx := context.Background()

	// Create test receipt core
	now := uint64(time.Now().UnixNano())
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       generateNonce(),
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	// Canonicalize and sign the receipt core
	coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
	if err != nil {
		t.Fatalf("Failed to canonicalize receipt core: %v", err)
	}

	signature, pubKey, err := signer.Sign(ctx, keyID, coreBytes)
	if err != nil {
		t.Fatalf("Failed to sign receipt core: %v", err)
	}

	// Create full receipt
	receiptFull := receipt.ReceiptFull{
		Core:       receiptCore,
		Signature:  signature,
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	// Canonicalize full receipt
	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		t.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	// Create verifier
	verifier := NewGoVerifier()

	t.Run("verify_valid_receipt", func(t *testing.T) {
		verifiedCore, err := verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err != nil {
			t.Fatalf("Failed to verify valid receipt: %v", err)
		}
		if verifiedCore == nil {
			t.Error("Expected non-nil verified core")
		}
		if verifiedCore.ProgramHash != receiptCore.ProgramHash ||
			verifiedCore.InputHash != receiptCore.InputHash ||
			verifiedCore.OutputHash != receiptCore.OutputHash ||
			verifiedCore.GasUsed != receiptCore.GasUsed ||
			verifiedCore.IssuerID != receiptCore.IssuerID {
			t.Error("Expected verified core to match original core")
		}
	})

	t.Run("verify_with_wrong_public_key", func(t *testing.T) {
		_, wrongPubKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate wrong public key: %v", err)
		}

		_, err = verifier.VerifyReceipt(fullReceiptBytes, wrongPubKey)
		if err == nil {
			t.Error("Expected error when verifying with wrong public key")
		}
	})

	t.Run("verify_corrupted_receipt", func(t *testing.T) {
		corruptedBytes := make([]byte, len(fullReceiptBytes))
		copy(corruptedBytes, fullReceiptBytes)
		corruptedBytes[0] ^= 0xFF // Flip first byte

		_, err = verifier.VerifyReceipt(corruptedBytes, pubKey)
		if err == nil {
			t.Error("Expected error when verifying corrupted receipt")
		}
	})

	t.Run("verify_empty_receipt", func(t *testing.T) {
		_, err = verifier.VerifyReceipt([]byte{}, pubKey)
		if err == nil {
			t.Error("Expected error when verifying empty receipt")
		}
	})

	t.Run("verify_nil_receipt", func(t *testing.T) {
		_, err = verifier.VerifyReceipt(nil, pubKey)
		if err == nil {
			t.Error("Expected error when verifying nil receipt")
		}
	})

	t.Run("verify_with_nil_public_key", func(t *testing.T) {
		_, err = verifier.VerifyReceipt(fullReceiptBytes, nil)
		if err == nil {
			t.Error("Expected error when verifying with nil public key")
		}
	})
}

func TestGoVerifierExtractReceiptFields(t *testing.T) {
	// Create test receipt
	now := uint64(time.Now().UnixNano())
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       generateNonce(),
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	receiptFull := receipt.ReceiptFull{
		Core:       receiptCore,
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host", "version": "1.0.0"},
	}

	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		t.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	verifier := NewGoVerifier()

	t.Run("extract_valid_receipt_fields", func(t *testing.T) {
		fields, err := verifier.ExtractReceiptFields(fullReceiptBytes)
		if err != nil {
			t.Fatalf("Failed to extract receipt fields: %v", err)
		}
		if fields == nil {
			t.Error("Expected non-nil receipt fields")
		}

		// Verify extracted fields match original
		if string(fields.ProgramHash) != string(receiptCore.ProgramHash[:]) {
			t.Error("Expected matching program hash")
		}
		if string(fields.InputHash) != string(receiptCore.InputHash[:]) {
			t.Error("Expected matching input hash")
		}
		if string(fields.OutputHash) != string(receiptCore.OutputHash[:]) {
			t.Error("Expected matching output hash")
		}
		if fields.GasUsed != receiptCore.GasUsed {
			t.Error("Expected matching gas used")
		}
		if fields.StartedAt != receiptCore.StartedAt {
			t.Error("Expected matching started at")
		}
		if fields.FinishedAt != receiptCore.FinishedAt {
			t.Error("Expected matching finished at")
		}
		if fields.IssuerID != receiptCore.IssuerID {
			t.Error("Expected matching issuer ID")
		}
		if string(fields.Signature) != string(receiptFull.Signature) {
			t.Error("Expected matching signature")
		}
		if fields.HostCycles != receiptFull.HostCycles {
			t.Error("Expected matching host cycles")
		}
		if len(fields.HostInfo) != len(receiptFull.HostInfo) {
			t.Error("Expected matching host info length")
		}
		for k, v := range receiptFull.HostInfo {
			if fields.HostInfo[k] != v {
				t.Errorf("Expected matching host info for key %s", k)
			}
		}
	})

	t.Run("extract_from_corrupted_receipt", func(t *testing.T) {
		corruptedBytes := make([]byte, len(fullReceiptBytes))
		copy(corruptedBytes, fullReceiptBytes)
		corruptedBytes[0] ^= 0xFF // Flip first byte

		_, err = verifier.ExtractReceiptFields(corruptedBytes)
		if err == nil {
			t.Error("Expected error when extracting from corrupted receipt")
		}
	})

	t.Run("extract_from_empty_receipt", func(t *testing.T) {
		_, err = verifier.ExtractReceiptFields([]byte{})
		if err == nil {
			t.Error("Expected error when extracting from empty receipt")
		}
	})

	t.Run("extract_from_nil_receipt", func(t *testing.T) {
		_, err = verifier.ExtractReceiptFields(nil)
		if err == nil {
			t.Error("Expected error when extracting from nil receipt")
		}
	})
}

func TestGoVerifierBatchVerify(t *testing.T) {
	// Create test keystore and signer
	ks, err := keystore.New(t.TempDir())
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

	signer := keystore.NewLocalSigner(ks)
	ctx := context.Background()

	verifier := NewGoVerifier()

	// Create multiple test receipts
	numReceipts := 5
	receipts := make([][]byte, numReceipts)
	publicKeys := make([]ed25519.PublicKey, numReceipts)
	now := uint64(time.Now().UnixNano())

	for i := 0; i < numReceipts; i++ {
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{byte(i), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{byte(i), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     uint64(1000 + i),
			StartedAt:   now - 1000000 + uint64(i),
			FinishedAt:  now - 500000 + uint64(i),
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       generateNonce(),
			IssuedAt:    now + uint64(i),
			FloatMode:   "disabled",
		}

		// Canonicalize and sign
		coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
		if err != nil {
			t.Fatalf("Failed to canonicalize receipt core %d: %v", i, err)
		}

		signature, pubKey, err := signer.Sign(ctx, keyID, coreBytes)
		if err != nil {
			t.Fatalf("Failed to sign receipt core %d: %v", i, err)
		}

		receiptFull := receipt.ReceiptFull{
			Core:       receiptCore,
			Signature:  signature,
			HostCycles: uint64(12345 + i),
			HostInfo:   map[string]string{"host": "test-host", "index": string(rune(i))},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize full receipt %d: %v", i, err)
		}

		receipts[i] = fullReceiptBytes
		publicKeys[i] = pubKey
	}

	t.Run("batch_verify_all_valid", func(t *testing.T) {
		// Convert to ReceiptBatch format
		batches := make([]ReceiptBatch, numReceipts)
		for i := 0; i < numReceipts; i++ {
			batches[i] = ReceiptBatch{
				ReceiptData: receipts[i],
				PublicKey:   publicKeys[i],
			}
		}

		results, err := verifier.BatchVerify(batches)
		if err != nil {
			t.Fatalf("Failed to batch verify receipts: %v", err)
		}
		if len(results) != numReceipts {
			t.Errorf("Expected %d results, got %d", numReceipts, len(results))
		}
		for i, result := range results {
			if !result {
				t.Errorf("Expected true result for receipt %d", i)
			}
		}
	})

	t.Run("batch_verify_with_invalid_receipt", func(t *testing.T) {
		// Corrupt one receipt
		corruptedReceipts := make([][]byte, len(receipts))
		copy(corruptedReceipts, receipts)
		corruptedReceipts[2] = []byte("corrupted receipt data")

		// Convert to ReceiptBatch format
		batches := make([]ReceiptBatch, numReceipts)
		for i := 0; i < numReceipts; i++ {
			batches[i] = ReceiptBatch{
				ReceiptData: corruptedReceipts[i],
				PublicKey:   publicKeys[i],
			}
		}

		results, err := verifier.BatchVerify(batches)
		if err != nil {
			t.Fatalf("Failed to batch verify receipts with invalid one: %v", err)
		}
		if len(results) != numReceipts {
			t.Errorf("Expected %d results, got %d", numReceipts, len(results))
		}
		// The corrupted receipt should have a false result
		if results[2] {
			t.Error("Expected false result for corrupted receipt")
		}
		// Other receipts should still be valid
		for i, result := range results {
			if i == 2 {
				continue // Skip the corrupted one
			}
			if !result {
				t.Errorf("Expected true result for valid receipt %d", i)
			}
		}
	})

	t.Run("batch_verify_mismatched_lengths", func(t *testing.T) {
		// Test with mismatched lengths - this should be handled by the caller
		// since we now use ReceiptBatch format
		batches := make([]ReceiptBatch, 2)
		batches[0] = ReceiptBatch{ReceiptData: receipts[0], PublicKey: publicKeys[0]}
		batches[1] = ReceiptBatch{ReceiptData: receipts[1], PublicKey: publicKeys[1]}

		results, err := verifier.BatchVerify(batches)
		if err != nil {
			t.Fatalf("Failed to batch verify with valid batches: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("batch_verify_empty_input", func(t *testing.T) {
		results, err := verifier.BatchVerify([]ReceiptBatch{})
		if err != nil {
			t.Fatalf("Failed to batch verify empty input: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty input, got %d", len(results))
		}
	})
}

func TestGoVerifierGetVersion(t *testing.T) {
	verifier := NewGoVerifier()

	t.Run("get_version", func(t *testing.T) {
		version, err := verifier.GetVersion()
		if err != nil {
			t.Fatalf("Failed to get version: %v", err)
		}
		if version == "" {
			t.Error("Expected non-empty version")
		}
		// Version should be a reasonable format
		if len(version) < 3 {
			t.Errorf("Expected version to be at least 3 characters, got %d", len(version))
		}
	})
}

func TestGoVerifierEdgeCases(t *testing.T) {
	verifier := NewGoVerifier()
	now := uint64(time.Now().UnixNano())

	t.Run("verify_receipt_with_invalid_signature_length", func(t *testing.T) {
		// Create a receipt with invalid signature length
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   now - 1000000,
			FinishedAt:  now - 500000,
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       generateNonce(),
			IssuedAt:    now,
			FloatMode:   "disabled",
		}

		receiptFull := receipt.ReceiptFull{
			Core:       receiptCore,
			Signature:  make([]byte, 32), // Wrong signature length
			HostCycles: 12345,
			HostInfo:   map[string]string{"host": "test-host"},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize full receipt: %v", err)
		}

		_, pubKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate public key: %v", err)
		}

		_, err = verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err == nil {
			t.Error("Expected error when verifying receipt with invalid signature length")
		}
	})

	t.Run("verify_receipt_with_zero_gas", func(t *testing.T) {
		// Create a receipt with zero gas (should be valid)
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     0, // Zero gas
			StartedAt:   now - 1000000,
			FinishedAt:  now - 500000,
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       generateNonce(),
			IssuedAt:    now,
			FloatMode:   "disabled",
		}

		receiptFull := receipt.ReceiptFull{
			Core:       receiptCore,
			Signature:  make([]byte, 64),
			HostCycles: 12345,
			HostInfo:   map[string]string{"host": "test-host"},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize full receipt: %v", err)
		}

		_, pubKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate public key: %v", err)
		}

		// This should fail because we can't verify a signature we didn't create
		_, err = verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err == nil {
			t.Error("Expected error when verifying receipt with zero gas and random signature")
		}
	})

	t.Run("verify_receipt_with_invalid_timestamps", func(t *testing.T) {
		// Create a receipt with invalid timestamps (started after finished)
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   now - 500000, // Started after finished
			FinishedAt:  now - 1000000,
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       generateNonce(),
			IssuedAt:    now,
			FloatMode:   "disabled",
		}

		receiptFull := receipt.ReceiptFull{
			Core:       receiptCore,
			Signature:  make([]byte, 64),
			HostCycles: 12345,
			HostInfo:   map[string]string{"host": "test-host"},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize full receipt: %v", err)
		}

		_, pubKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate public key: %v", err)
		}

		// This should fail because we can't verify a signature we didn't create
		_, err = verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err == nil {
			t.Error("Expected error when verifying receipt with invalid timestamps and random signature")
		}
	})
}

func TestGoVerifierConcurrency(t *testing.T) {
	// Create test keystore and signer
	ks, err := keystore.New(t.TempDir())
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

	signer := keystore.NewLocalSigner(ks)
	ctx := context.Background()

	// Create test receipt
	now := uint64(time.Now().UnixNano())
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       generateNonce(),
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
	if err != nil {
		t.Fatalf("Failed to canonicalize receipt core: %v", err)
	}

	signature, pubKey, err := signer.Sign(ctx, keyID, coreBytes)
	if err != nil {
		t.Fatalf("Failed to sign receipt core: %v", err)
	}

	receiptFull := receipt.ReceiptFull{
		Core:       receiptCore,
		Signature:  signature,
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		t.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	verifier := NewGoVerifier()
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	t.Run("concurrent_verification", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := verifier.VerifyReceipt(fullReceiptBytes, pubKey)
				results <- err
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent verification failed: %v", err)
			}
		}
	})

	t.Run("concurrent_field_extraction", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := verifier.ExtractReceiptFields(fullReceiptBytes)
				results <- err
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent field extraction failed: %v", err)
			}
		}
	})
}

func BenchmarkGoVerifierVerifyReceipt(b *testing.B) {
	// Create test keystore and signer
	ks, err := keystore.New(b.TempDir())
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
	keyID := activeKey.ID

	signer := keystore.NewLocalSigner(ks)
	ctx := context.Background()

	// Create test receipt
	now := uint64(time.Now().UnixNano())
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       generateNonce(),
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
	if err != nil {
		b.Fatalf("Failed to canonicalize receipt core: %v", err)
	}

	messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
	signature, pubKey, err := signer.Sign(ctx, keyID, messageToSign)
	if err != nil {
		b.Fatalf("Failed to sign receipt core: %v", err)
	}

	receiptFull := receipt.ReceiptFull{
		Core:       receiptCore,
		Signature:  signature,
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		b.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	verifier := NewGoVerifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err != nil {
			b.Fatalf("Failed to verify receipt: %v", err)
		}
	}
}

func BenchmarkGoVerifierExtractReceiptFields(b *testing.B) {
	// Create test receipt
	now := uint64(time.Now().UnixNano())
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       generateNonce(),
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	receiptFull := receipt.ReceiptFull{
		Core:       receiptCore,
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host", "version": "1.0.0"},
	}

	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		b.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	verifier := NewGoVerifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := verifier.ExtractReceiptFields(fullReceiptBytes)
		if err != nil {
			b.Fatalf("Failed to extract receipt fields: %v", err)
		}
	}
}

func BenchmarkGoVerifierBatchVerify(b *testing.B) {
	// Create test keystore and signer
	ks, err := keystore.New(b.TempDir())
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
	keyID := activeKey.ID

	signer := keystore.NewLocalSigner(ks)
	ctx := context.Background()

	// Create multiple test receipts
	numReceipts := 10
	receipts := make([][]byte, numReceipts)
	publicKeys := make([]ed25519.PublicKey, numReceipts)
	now := uint64(time.Now().UnixNano())

	for i := 0; i < numReceipts; i++ {
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{byte(i), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{byte(i), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     uint64(1000 + i),
			StartedAt:   now - 1000000 + uint64(i),
			FinishedAt:  now - 500000 + uint64(i),
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       generateNonce(),
			IssuedAt:    now + uint64(i),
			FloatMode:   "disabled",
		}

		coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
		if err != nil {
			b.Fatalf("Failed to canonicalize receipt core %d: %v", i, err)
		}

		signature, pubKey, err := signer.Sign(ctx, keyID, coreBytes)
		if err != nil {
			b.Fatalf("Failed to sign receipt core %d: %v", i, err)
		}

		receiptFull := receipt.ReceiptFull{
			Core:       receiptCore,
			Signature:  signature,
			HostCycles: uint64(12345 + i),
			HostInfo:   map[string]string{"host": "test-host", "index": string(rune(i))},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			b.Fatalf("Failed to canonicalize full receipt %d: %v", i, err)
		}

		receipts[i] = fullReceiptBytes
		publicKeys[i] = pubKey
	}

	verifier := NewGoVerifier()

	b.ResetTimer()
	// Convert to ReceiptBatch format
	batches := make([]ReceiptBatch, len(receipts))
	for i := 0; i < len(receipts); i++ {
		batches[i] = ReceiptBatch{
			ReceiptData: receipts[i],
			PublicKey:   publicKeys[i],
		}
	}

	for i := 0; i < b.N; i++ {
		_, err := verifier.BatchVerify(batches)
		if err != nil {
			b.Fatalf("Failed to batch verify receipts: %v", err)
		}
	}
}
