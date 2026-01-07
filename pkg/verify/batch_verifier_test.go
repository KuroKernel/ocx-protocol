package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"ocx.local/pkg/receipt"
)

// createTestReceipt creates a signed test receipt
func createTestReceipt(t *testing.T) (*receipt.ReceiptFull, ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()

	// Generate key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create receipt core with nonce
	var programHash, inputHash, outputHash [32]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])

	var nonce [16]byte
	rand.Read(nonce[:])

	now := uint64(time.Now().UnixNano())

	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	// Canonicalize and sign
	coreBytes, err := receipt.CanonicalizeCore(&core)
	if err != nil {
		t.Fatalf("failed to canonicalize: %v", err)
	}

	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	full := &receipt.ReceiptFull{
		Core:       core,
		Signature:  sig,
		HostCycles: 5000,
		HostInfo:   map[string]string{"platform": "test"},
	}

	return full, pub, priv
}

// TestBatchVerifier_SingleReceipt tests verifying a single receipt
func TestBatchVerifier_SingleReceipt(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: 2})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	rcpt, pub, _ := createTestReceipt(t)

	// Serialize receipt
	data, err := receipt.CanonicalizeFull(rcpt)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	batch := []ReceiptBatch{{ReceiptData: data, PublicKey: pub}}

	ctx := context.Background()
	results, stats := bv.VerifyBatch(ctx, batch)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Valid {
		t.Errorf("expected valid receipt, got error: %v", results[0].Error)
	}

	if stats.Valid != 1 {
		t.Errorf("expected 1 valid, got %d", stats.Valid)
	}

	if stats.Invalid != 0 {
		t.Errorf("expected 0 invalid, got %d", stats.Invalid)
	}
}

// TestBatchVerifier_InvalidSignature tests signature rejection
func TestBatchVerifier_InvalidSignature(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	rcpt, _, _ := createTestReceipt(t)

	// Use wrong public key
	wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)

	data, _ := receipt.CanonicalizeFull(rcpt)
	batch := []ReceiptBatch{{ReceiptData: data, PublicKey: wrongPub}}

	results, stats := bv.VerifyBatch(context.Background(), batch)

	if results[0].Valid {
		t.Error("expected invalid for wrong public key")
	}

	if stats.Invalid != 1 {
		t.Errorf("expected 1 invalid, got %d", stats.Invalid)
	}
}

// TestBatchVerifier_CorruptedData tests handling of corrupted data
func TestBatchVerifier_CorruptedData(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	_, pub, _ := createTestReceipt(t)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"random_bytes", []byte("not valid cbor data at all")},
		{"truncated", []byte{0xa2, 0x64}}, // Partial CBOR
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			batch := []ReceiptBatch{{ReceiptData: tc.data, PublicKey: pub}}
			results, _ := bv.VerifyBatch(context.Background(), batch)

			if results[0].Valid {
				t.Error("expected invalid for corrupted data")
			}

			if results[0].Error == nil {
				t.Error("expected error for corrupted data")
			}
		})
	}
}

// TestBatchVerifier_InvalidPublicKeyLength tests public key validation
func TestBatchVerifier_InvalidPublicKeyLength(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	rcpt, _, _ := createTestReceipt(t)
	data, _ := receipt.CanonicalizeFull(rcpt)

	testCases := []struct {
		name   string
		keyLen int
	}{
		{"empty", 0},
		{"too_short", 16},
		{"too_long", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.keyLen)
			rand.Read(key)

			batch := []ReceiptBatch{{ReceiptData: data, PublicKey: key}}
			results, _ := bv.VerifyBatch(context.Background(), batch)

			if results[0].Valid {
				t.Error("expected invalid for wrong key length")
			}
		})
	}
}

// TestBatchVerifier_LargeBatch tests batch verification performance
func TestBatchVerifier_LargeBatch(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: 8})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	const batchSize = 100

	batches := make([]ReceiptBatch, batchSize)
	for i := 0; i < batchSize; i++ {
		rcpt, pub, _ := createTestReceipt(t)
		data, _ := receipt.CanonicalizeFull(rcpt)
		batches[i] = ReceiptBatch{ReceiptData: data, PublicKey: pub}
	}

	ctx := context.Background()
	start := time.Now()
	results, stats := bv.VerifyBatch(ctx, batches)
	elapsed := time.Since(start)

	if len(results) != batchSize {
		t.Fatalf("expected %d results, got %d", batchSize, len(results))
	}

	if stats.Valid != batchSize {
		t.Errorf("expected %d valid, got %d valid, %d invalid", batchSize, stats.Valid, stats.Invalid)
		for i, r := range results {
			if !r.Valid {
				t.Logf("  result %d: %v", i, r.Error)
			}
		}
	}

	t.Logf("Verified %d receipts in %v (%.1f receipts/sec)", batchSize, elapsed, stats.Throughput)

	// Should achieve > 50 receipts/sec
	if stats.Throughput < 50 {
		t.Errorf("throughput too low: %.1f receipts/sec", stats.Throughput)
	}
}

// TestBatchVerifier_ContextCancellation tests cancellation handling
func TestBatchVerifier_ContextCancellation(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: 2})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	// Create a large batch
	const batchSize = 1000
	batches := make([]ReceiptBatch, batchSize)
	for i := 0; i < batchSize; i++ {
		rcpt, pub, _ := createTestReceipt(t)
		data, _ := receipt.CanonicalizeFull(rcpt)
		batches[i] = ReceiptBatch{ReceiptData: data, PublicKey: pub}
	}

	// Cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, _ := bv.VerifyBatch(ctx, batches)

	// Should still return all results (might have context errors)
	if len(results) != batchSize {
		t.Errorf("expected %d results, got %d", batchSize, len(results))
	}
}

// TestBatchVerifier_ZeroNonce tests zero nonce rejection
func TestBatchVerifier_ZeroNonce(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	// Create receipt with zero nonce
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	var programHash, inputHash, outputHash [32]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])

	now := uint64(time.Now().UnixNano())

	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "test-issuer",
		KeyVersion:  1,
		Nonce:       [16]byte{}, // Zero nonce
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, _ := receipt.CanonicalizeCore(&core)
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	full := &receipt.ReceiptFull{Core: core, Signature: sig}
	data, _ := receipt.CanonicalizeFull(full)

	batch := []ReceiptBatch{{ReceiptData: data, PublicKey: pub}}
	results, _ := bv.VerifyBatch(context.Background(), batch)

	if results[0].Valid {
		t.Error("expected invalid for zero nonce")
	}
}

// TestBatchVerifier_MixedBatch tests batch with valid and invalid receipts
func TestBatchVerifier_MixedBatch(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{})
	if err != nil {
		t.Fatalf("failed to create batch verifier: %v", err)
	}

	batches := make([]ReceiptBatch, 10)

	// 5 valid receipts
	for i := 0; i < 5; i++ {
		rcpt, pub, _ := createTestReceipt(t)
		data, _ := receipt.CanonicalizeFull(rcpt)
		batches[i] = ReceiptBatch{ReceiptData: data, PublicKey: pub}
	}

	// 5 invalid receipts (wrong public key)
	for i := 5; i < 10; i++ {
		rcpt, _, _ := createTestReceipt(t)
		wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)
		data, _ := receipt.CanonicalizeFull(rcpt)
		batches[i] = ReceiptBatch{ReceiptData: data, PublicKey: wrongPub}
	}

	results, stats := bv.VerifyBatch(context.Background(), batches)

	if stats.Valid != 5 {
		t.Errorf("expected 5 valid, got %d", stats.Valid)
	}

	if stats.Invalid != 5 {
		t.Errorf("expected 5 invalid, got %d", stats.Invalid)
	}

	// Check order preserved
	for i := 0; i < 5; i++ {
		if !results[i].Valid {
			t.Errorf("result %d should be valid", i)
		}
		if results[i].Index != i {
			t.Errorf("result %d has wrong index %d", i, results[i].Index)
		}
	}

	for i := 5; i < 10; i++ {
		if results[i].Valid {
			t.Errorf("result %d should be invalid", i)
		}
	}
}

// BenchmarkBatchVerifier benchmarks batch verification
func BenchmarkBatchVerifier(b *testing.B) {
	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 8})

	// Pre-create receipts
	batches := make([]ReceiptBatch, 100)
	for i := 0; i < 100; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)

		var programHash, inputHash, outputHash [32]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])

		var nonce [16]byte
		rand.Read(nonce[:])

		now := uint64(time.Now().UnixNano())
		core := receipt.ReceiptCore{
			ProgramHash: programHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     1000,
			StartedAt:   now - 1000000,
			FinishedAt:  now - 500000,
			IssuerID:    "test-issuer",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    now,
			FloatMode:   "disabled",
		}

		coreBytes, _ := receipt.CanonicalizeCore(&core)
		msg := append([]byte("OCXv1|receipt|"), coreBytes...)
		sig := ed25519.Sign(priv, msg)

		full := &receipt.ReceiptFull{Core: core, Signature: sig}
		data, _ := receipt.CanonicalizeFull(full)
		batches[i] = ReceiptBatch{ReceiptData: data, PublicKey: pub}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bv.VerifyBatch(context.Background(), batches)
	}
}
