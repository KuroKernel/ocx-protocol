package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ocx.local/pkg/receipt"
)

// Property: Replay store always rejects duplicate nonces
func TestProperty_ReplayStoreRejectsDuplicates(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		nonce := make([]byte, 16)
		rand.Read(nonce)

		timestamp := uint64(time.Now().UnixNano())

		// First attempt should succeed
		ok1, err1 := store.CheckAndStore(nonce, timestamp)
		if err1 != nil {
			t.Fatalf("First store failed: %v", err1)
		}
		if !ok1 {
			t.Fatal("First store should return true")
		}

		// Second attempt with same nonce should fail
		ok2, err2 := store.CheckAndStore(nonce, timestamp)
		if err2 != nil {
			t.Fatalf("Second store error: %v", err2)
		}
		if ok2 {
			t.Fatal("Second store should return false (duplicate)")
		}

		// Third attempt should also fail
		ok3, _ := store.CheckAndStore(nonce, timestamp)
		if ok3 {
			t.Fatal("Third store should return false (duplicate)")
		}

		_ = rng // Silence unused warning
	}
}

// Property: Different nonces are always accepted
func TestProperty_ReplayStoreAcceptsDifferentNonces(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)

	timestamp := uint64(time.Now().UnixNano())

	for i := 0; i < 1000; i++ {
		nonce := make([]byte, 16)
		rand.Read(nonce)

		ok, err := store.CheckAndStore(nonce, timestamp)
		if err != nil {
			t.Fatalf("Store failed at iteration %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("Unique nonce rejected at iteration %d", i)
		}
	}

	// Verify store size
	if store.Size() != 1000 {
		t.Errorf("Expected 1000 nonces, got %d", store.Size())
	}
}

// Property: Replay store rejects future timestamps
func TestProperty_ReplayStoreRejectsFuture(t *testing.T) {
	clockSkew := time.Minute
	store := NewInMemoryReplayStore(time.Hour, clockSkew)

	futureTime := uint64(time.Now().Add(clockSkew * 2).UnixNano())

	nonce := make([]byte, 16)
	rand.Read(nonce)

	ok, err := store.CheckAndStore(nonce, futureTime)

	// Should either return error or false
	if ok && err == nil {
		t.Fatal("Future timestamp should be rejected")
	}
}

// Property: Replay store rejects old timestamps
func TestProperty_ReplayStoreRejectsOld(t *testing.T) {
	retention := time.Hour
	store := NewInMemoryReplayStore(retention, time.Minute)

	oldTime := uint64(time.Now().Add(-retention * 2).UnixNano())

	nonce := make([]byte, 16)
	rand.Read(nonce)

	ok, err := store.CheckAndStore(nonce, oldTime)

	// Should either return error or false
	if ok && err == nil {
		t.Fatal("Old timestamp should be rejected")
	}
}

// Property: Batch verifier returns correct count
func TestProperty_BatchVerifierCount(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: 2})
	if err != nil {
		t.Fatalf("Failed to create batch verifier: %v", err)
	}

	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	for batchSize := 1; batchSize <= 50; batchSize++ {
		batches := make([]ReceiptBatch, batchSize)

		for i := 0; i < batchSize; i++ {
			data := make([]byte, 100)
			pubKey := make([]byte, 32)
			rng.Read(data)
			rng.Read(pubKey)

			batches[i] = ReceiptBatch{
				ReceiptData: data,
				PublicKey:   pubKey,
			}
		}

		results, stats := bv.VerifyBatch(context.Background(), batches)

		// Must return exact number of results
		if len(results) != batchSize {
			t.Fatalf("Expected %d results, got %d", batchSize, len(results))
		}

		// Stats must match
		if stats.Valid+stats.Invalid != batchSize {
			t.Fatalf("Stats mismatch: valid=%d invalid=%d total=%d",
				stats.Valid, stats.Invalid, batchSize)
		}
	}
}

// Property: Batch verifier preserves order
func TestProperty_BatchVerifierOrder(t *testing.T) {
	bv, err := NewBatchVerifier(BatchVerifierConfig{Workers: 4})
	if err != nil {
		t.Fatalf("Failed to create batch verifier: %v", err)
	}

	// Create batch with identifiable receipts
	batchSize := 20
	batches := make([]ReceiptBatch, batchSize)
	markers := make([][]byte, batchSize)

	for i := 0; i < batchSize; i++ {
		// Use index as marker in receipt data
		markers[i] = []byte{byte(i)}
		batches[i] = ReceiptBatch{
			ReceiptData: markers[i],
			PublicKey:   make([]byte, 32),
		}
	}

	results, _ := bv.VerifyBatch(context.Background(), batches)

	// Results must be in same order
	for i := 0; i < batchSize; i++ {
		if results[i].Index != i {
			t.Fatalf("Result order wrong: expected index %d, got %d", i, results[i].Index)
		}
	}
}

// Property: Valid signatures always verify
func TestProperty_ValidSignatureAlwaysVerifies(t *testing.T) {
	verifier := NewGoVerifier()

	for i := 0; i < 50; i++ {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}

		// Create valid receipt
		receiptFull := createSignedReceipt(priv)
		receiptData, err := receipt.CanonicalizeFull(receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize receipt: %v", err)
		}

		// Verify - returns (*ReceiptCore, error), nil core means invalid
		core, err := verifier.VerifyReceipt(receiptData, pub)
		if err != nil {
			t.Fatalf("Verification error at iteration %d: %v", i, err)
		}
		if core == nil {
			t.Fatalf("Valid receipt rejected at iteration %d", i)
		}
	}
}

// Property: Invalid signatures always fail
func TestProperty_InvalidSignatureAlwaysFails(t *testing.T) {
	verifier := NewGoVerifier()

	for i := 0; i < 50; i++ {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)

		// Create receipt signed with priv
		receiptFull := createSignedReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)

		// Verify with wrong public key - should return error or nil core
		core, err := verifier.VerifyReceipt(receiptData, wrongPub)
		if err == nil && core != nil {
			t.Fatalf("Invalid receipt accepted at iteration %d", i)
		}
	}
}

// Property: Verification is deterministic
func TestProperty_VerificationDeterminism(t *testing.T) {
	verifier := NewGoVerifier()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	receiptFull := createSignedReceipt(priv)
	receiptData, _ := receipt.CanonicalizeFull(receiptFull)

	// Verify 100 times
	for i := 0; i < 100; i++ {
		core, err := verifier.VerifyReceipt(receiptData, pub)
		if err != nil {
			t.Fatalf("Verification error at iteration %d: %v", i, err)
		}
		if core == nil {
			t.Fatalf("Verification result inconsistent at iteration %d", i)
		}
	}
}

// Property: Field extraction is deterministic
func TestProperty_FieldExtractionDeterminism(t *testing.T) {
	verifier := NewGoVerifier()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)

	receiptFull := createSignedReceipt(priv)
	receiptData, _ := receipt.CanonicalizeFull(receiptFull)

	// Extract fields multiple times
	fields1, err1 := verifier.ExtractReceiptFields(receiptData)
	fields2, err2 := verifier.ExtractReceiptFields(receiptData)

	if (err1 != nil) != (err2 != nil) {
		t.Fatal("Field extraction error inconsistent")
	}

	if err1 == nil {
		if fields1.GasUsed != fields2.GasUsed {
			t.Fatal("GasUsed extraction not deterministic")
		}
		if fields1.IssuerID != fields2.IssuerID {
			t.Fatal("IssuerID extraction not deterministic")
		}
	}
}

// Property: Concurrent replay store access is safe
func TestProperty_ReplayStoreConcurrencySafe(t *testing.T) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)

	var wg sync.WaitGroup
	var successCount int64
	var duplicateCount int64

	// Use same nonces across goroutines to test concurrent access
	nonces := make([][]byte, 100)
	for i := range nonces {
		nonces[i] = make([]byte, 16)
		rand.Read(nonces[i])
	}

	timestamp := uint64(time.Now().UnixNano())

	// Launch many goroutines trying to store same nonces
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, nonce := range nonces {
				ok, err := store.CheckAndStore(nonce, timestamp)
				if err == nil {
					if ok {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&duplicateCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	// Exactly 100 should succeed (one per unique nonce)
	if successCount != 100 {
		t.Errorf("Expected exactly 100 successes, got %d", successCount)
	}

	// Store should have exactly 100 nonces
	if store.Size() != 100 {
		t.Errorf("Expected store size 100, got %d", store.Size())
	}
}

// Property: Batch verification is consistent across workers
func TestProperty_BatchVerificationConsistency(t *testing.T) {
	// Create same batch and verify with different worker counts
	batches := make([]ReceiptBatch, 20)
	for i := 0; i < 20; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)

		receiptFull := createSignedReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)

		batches[i] = ReceiptBatch{
			ReceiptData: receiptData,
			PublicKey:   pub,
		}
	}

	// Verify with 1 worker
	bv1, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 1})
	results1, _ := bv1.VerifyBatch(context.Background(), batches)

	// Verify with 4 workers
	bv4, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 4})
	results4, _ := bv4.VerifyBatch(context.Background(), batches)

	// Results must match
	for i := 0; i < 20; i++ {
		if results1[i].Valid != results4[i].Valid {
			t.Fatalf("Results differ at index %d: 1 worker=%v, 4 workers=%v",
				i, results1[i].Valid, results4[i].Valid)
		}
	}
}

// Helper: Create a signed receipt
func createSignedReceipt(priv ed25519.PrivateKey) *receipt.ReceiptFull {
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	now := uint64(time.Now().UnixNano())
	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     1000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "property-test",
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, _ := receipt.CanonicalizeCore(&core)
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	return &receipt.ReceiptFull{
		Core:       core,
		Signature:  sig,
		HostCycles: 5000,
	}
}

// Benchmark property tests
func BenchmarkProperty_ReplayStore(b *testing.B) {
	store := NewInMemoryReplayStore(time.Hour, time.Minute)
	timestamp := uint64(time.Now().UnixNano())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nonce := make([]byte, 16)
		rand.Read(nonce)
		store.CheckAndStore(nonce, timestamp)
	}
}

func BenchmarkProperty_VerifyReceipt(b *testing.B) {
	verifier := NewGoVerifier()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	receiptFull := createSignedReceipt(priv)
	receiptData, _ := receipt.CanonicalizeFull(receiptFull)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.VerifyReceipt(receiptData, pub)
	}
}

func BenchmarkProperty_BatchVerify20(b *testing.B) {
	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 4})

	batches := make([]ReceiptBatch, 20)
	for i := 0; i < 20; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		receiptFull := createSignedReceipt(priv)
		receiptData, _ := receipt.CanonicalizeFull(receiptFull)
		batches[i] = ReceiptBatch{
			ReceiptData: receiptData,
			PublicKey:   pub,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bv.VerifyBatch(context.Background(), batches)
	}
}
