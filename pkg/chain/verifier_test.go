package chain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

// testCounter ensures unique hashes for each receipt
var testCounter uint64 = 0

// Helper to create a test receipt with unique hashes
func createTestReceipt(prevHash *[32]byte, startedAt, finishedAt uint64) *ChainedReceipt {
	testCounter++
	// Use counter and timestamps to ensure unique hashes
	artifactHash := sha256.Sum256([]byte(fmt.Sprintf("test-artifact-%d-%d", testCounter, startedAt)))
	inputHash := sha256.Sum256([]byte(fmt.Sprintf("test-input-%d-%d", testCounter, startedAt)))
	outputHash := sha256.Sum256([]byte(fmt.Sprintf("test-output-%d-%d", testCounter, finishedAt)))

	receipt := &ChainedReceipt{
		ArtifactHash:    artifactHash,
		InputHash:       inputHash,
		OutputHash:      outputHash,
		CyclesUsed:      1000,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		IssuerKeyID:     "test-issuer",
		Signature:       make([]byte, 64),
		PrevReceiptHash: prevHash,
		StoredAt:        time.Now(),
	}

	// Calculate unique receipt hash using all unique fields
	hashInput := append(receipt.ArtifactHash[:], receipt.InputHash[:]...)
	hashInput = append(hashInput, receipt.OutputHash[:]...)
	receipt.ReceiptHash = sha256.Sum256(hashInput)

	return receipt
}

func TestMemoryStore_SaveAndRetrieve(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	receipt := createTestReceipt(nil, 1000, 2000)

	// Save receipt
	err := store.SaveReceipt(ctx, receipt)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	// Retrieve receipt
	retrieved, err := store.GetReceiptByHash(ctx, receipt.ReceiptHash)
	if err != nil {
		t.Fatalf("GetReceiptByHash failed: %v", err)
	}

	if retrieved.ReceiptHash != receipt.ReceiptHash {
		t.Error("Retrieved receipt hash doesn't match")
	}
}

func TestMemoryStore_GetAncestors(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create a chain of 5 receipts
	var prevHash *[32]byte
	var receipts []*ChainedReceipt
	baseTime := uint64(1000)

	for i := 0; i < 5; i++ {
		receipt := createTestReceipt(prevHash, baseTime+uint64(i*100), baseTime+uint64(i*100)+50)
		receipt.ChainID = "test-chain"
		receipt.ChainSeq = uint64(i + 1)

		err := store.SaveReceipt(ctx, receipt)
		if err != nil {
			t.Fatalf("SaveReceipt failed at %d: %v", i, err)
		}

		receipts = append(receipts, receipt)
		hash := receipt.ReceiptHash
		prevHash = &hash
	}

	// Get ancestors from head
	head := receipts[len(receipts)-1]
	ancestors, err := store.GetAncestors(ctx, head.ReceiptHash, 0)
	if err != nil {
		t.Fatalf("GetAncestors failed: %v", err)
	}

	if len(ancestors) != 5 {
		t.Errorf("Expected 5 ancestors, got %d", len(ancestors))
	}

	// Verify order (head first, genesis last)
	if ancestors[0].ReceiptHash != head.ReceiptHash {
		t.Error("First ancestor should be head")
	}

	if ancestors[len(ancestors)-1].PrevReceiptHash != nil {
		t.Error("Last ancestor (genesis) should have nil prev_receipt_hash")
	}
}

func TestVerifier_VerifyChain_Valid(t *testing.T) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false // Skip signature verification for test
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create a valid chain
	var prevHash *[32]byte
	baseTime := uint64(time.Now().Unix() - 1000)

	for i := 0; i < 3; i++ {
		receipt := createTestReceipt(prevHash, baseTime+uint64(i*100), baseTime+uint64(i*100)+50)
		receipt.ChainID = "valid-chain"
		receipt.ChainSeq = uint64(i + 1)

		err := store.SaveReceipt(ctx, receipt)
		if err != nil {
			t.Fatalf("SaveReceipt failed: %v", err)
		}

		hash := receipt.ReceiptHash
		prevHash = &hash
	}

	// Verify the chain
	result, err := verifier.VerifyChain(ctx, *prevHash)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Chain should be valid, errors: %+v", result.Errors)
	}

	if result.ChainLength != 3 {
		t.Errorf("Expected chain length 3, got %d", result.ChainLength)
	}
}

func TestVerifier_VerifyChain_TimestampViolation(t *testing.T) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false
	policy.MaxClockSkew = 0 // No tolerance for this test
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create first receipt
	baseTime := uint64(time.Now().Unix() - 1000)
	receipt1 := createTestReceipt(nil, baseTime, baseTime+500) // finishes at baseTime+500
	receipt1.ChainID = "bad-chain"
	receipt1.ChainSeq = 1
	err := store.SaveReceipt(ctx, receipt1)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	// Create second receipt with timestamp clearly BEFORE first receipt finished
	// Previous finished at baseTime+500, this one starts at baseTime+100 (400 seconds before!)
	hash1 := receipt1.ReceiptHash
	receipt2 := createTestReceipt(&hash1, baseTime+100, baseTime+200) // starts at 100, but prev finished at 500
	receipt2.ChainID = "bad-chain"
	receipt2.ChainSeq = 2
	err = store.SaveReceipt(ctx, receipt2)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	// Verify should fail
	result, err := verifier.VerifyChain(ctx, receipt2.ReceiptHash)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if result.Valid {
		t.Error("Chain with timestamp violation should be invalid")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors for timestamp violation")
	}
}

func TestVerifier_VerifyChain_MissingAncestor(t *testing.T) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false
	policy.AllowMissingAncestors = false
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create a receipt that references a non-existent previous receipt
	baseTime := uint64(time.Now().Unix() - 1000)
	fakeHash := sha256.Sum256([]byte("non-existent"))
	receipt := createTestReceipt(&fakeHash, baseTime, baseTime+50)
	receipt.ChainID = "orphan-chain"

	err := store.SaveReceipt(ctx, receipt)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	// Verify should fail due to missing ancestor
	result, err := verifier.VerifyChain(ctx, receipt.ReceiptHash)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if result.Valid {
		t.Error("Chain with missing ancestor should be invalid")
	}
}

func TestVerifier_VerifyChain_WithDepthLimit(t *testing.T) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false
	policy.MaxChainDepth = 3 // Only verify last 3 receipts
	policy.AllowMissingAncestors = true
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create a chain of 10 receipts
	var prevHash *[32]byte
	baseTime := uint64(time.Now().Unix() - 2000)

	for i := 0; i < 10; i++ {
		receipt := createTestReceipt(prevHash, baseTime+uint64(i*100), baseTime+uint64(i*100)+50)
		receipt.ChainID = "long-chain"
		receipt.ChainSeq = uint64(i + 1)

		err := store.SaveReceipt(ctx, receipt)
		if err != nil {
			t.Fatalf("SaveReceipt failed: %v", err)
		}

		hash := receipt.ReceiptHash
		prevHash = &hash
	}

	// Verify - should only check last 3
	result, err := verifier.VerifyChain(ctx, *prevHash)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Chain should be valid with depth limit, errors: %+v", result.Errors)
	}

	if result.ChainLength != 3 {
		t.Errorf("Expected chain length 3 (depth limit), got %d", result.ChainLength)
	}
}

func TestVerifier_AppendToChain(t *testing.T) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create genesis receipt
	baseTime := uint64(time.Now().Unix() - 100)
	genesis := createTestReceipt(nil, baseTime, baseTime+10)
	genesis.ChainID = "append-test"

	err := verifier.CreateGenesisReceipt(ctx, genesis, "append-test")
	if err != nil {
		t.Fatalf("CreateGenesisReceipt failed: %v", err)
	}

	// Append second receipt
	hash1 := genesis.ReceiptHash
	receipt2 := createTestReceipt(&hash1, baseTime+20, baseTime+30)

	err = verifier.AppendToChain(ctx, receipt2)
	if err != nil {
		t.Fatalf("AppendToChain failed: %v", err)
	}

	// Verify chain
	result, err := verifier.VerifyChain(ctx, receipt2.ReceiptHash)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Appended chain should be valid, errors: %+v", result.Errors)
	}

	if result.ChainLength != 2 {
		t.Errorf("Expected chain length 2, got %d", result.ChainLength)
	}
}

func TestVerifier_GetChainProvenance(t *testing.T) {
	store := NewMemoryStore()
	policy := RelaxedValidationPolicy()
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create a chain
	var prevHash *[32]byte
	var head [32]byte
	baseTime := uint64(time.Now().Unix() - 500)

	for i := 0; i < 3; i++ {
		receipt := createTestReceipt(prevHash, baseTime+uint64(i*100), baseTime+uint64(i*100)+50)
		receipt.ChainID = "provenance-test"
		receipt.ChainSeq = uint64(i + 1)

		err := store.SaveReceipt(ctx, receipt)
		if err != nil {
			t.Fatalf("SaveReceipt failed: %v", err)
		}

		hash := receipt.ReceiptHash
		prevHash = &hash
		head = hash
	}

	// Get provenance
	provenance, err := verifier.GetChainProvenance(ctx, head)
	if err != nil {
		t.Fatalf("GetChainProvenance failed: %v", err)
	}

	if len(provenance) != 3 {
		t.Errorf("Expected 3 provenance entries, got %d", len(provenance))
	}

	// Each entry should contain position and hash info
	for i, entry := range provenance {
		if entry == "" {
			t.Errorf("Provenance entry %d is empty", i)
		}
	}
}

func TestChainedReceipt_IsGenesisReceipt(t *testing.T) {
	// Genesis receipt (no prev hash)
	genesis := createTestReceipt(nil, 1000, 2000)
	if !genesis.IsGenesisReceipt() {
		t.Error("Receipt without prev_receipt_hash should be genesis")
	}

	// Non-genesis receipt
	prevHash := sha256.Sum256([]byte("prev"))
	nonGenesis := createTestReceipt(&prevHash, 2000, 3000)
	if nonGenesis.IsGenesisReceipt() {
		t.Error("Receipt with prev_receipt_hash should not be genesis")
	}
}

func TestHashToHex_And_HexToHash(t *testing.T) {
	original := sha256.Sum256([]byte("test data"))

	// Convert to hex
	hexStr := HashToHex(original)
	if len(hexStr) != 64 {
		t.Errorf("Expected 64 char hex string, got %d", len(hexStr))
	}

	// Convert back
	recovered, err := HexToHash(hexStr)
	if err != nil {
		t.Fatalf("HexToHash failed: %v", err)
	}

	if recovered != original {
		t.Error("Round-trip hash conversion failed")
	}
}

func TestHexToHash_InvalidInput(t *testing.T) {
	// Invalid hex
	_, err := HexToHash("not-hex")
	if err == nil {
		t.Error("Expected error for invalid hex")
	}

	// Wrong length
	_, err = HexToHash("abcd")
	if err == nil {
		t.Error("Expected error for wrong length")
	}
}

func TestMemoryStore_HasReceipt(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	receipt := createTestReceipt(nil, 1000, 2000)

	// Before saving
	exists, err := store.HasReceipt(ctx, receipt.ReceiptHash)
	if err != nil {
		t.Fatalf("HasReceipt failed: %v", err)
	}
	if exists {
		t.Error("Receipt should not exist before saving")
	}

	// After saving
	err = store.SaveReceipt(ctx, receipt)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	exists, err = store.HasReceipt(ctx, receipt.ReceiptHash)
	if err != nil {
		t.Fatalf("HasReceipt failed: %v", err)
	}
	if !exists {
		t.Error("Receipt should exist after saving")
	}
}

func TestMemoryStore_CycleDetection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create two receipts that reference each other (cycle)
	receipt1 := createTestReceipt(nil, 1000, 2000)
	receipt2 := createTestReceipt(nil, 2000, 3000)

	// Make them reference each other
	hash1 := receipt1.ReceiptHash
	hash2 := receipt2.ReceiptHash
	receipt1.PrevReceiptHash = &hash2
	receipt2.PrevReceiptHash = &hash1

	err := store.SaveReceipt(ctx, receipt1)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}
	err = store.SaveReceipt(ctx, receipt2)
	if err != nil {
		t.Fatalf("SaveReceipt failed: %v", err)
	}

	// GetAncestors should detect the cycle
	_, err = store.GetAncestors(ctx, receipt1.ReceiptHash, 0)
	if err == nil {
		t.Error("Expected error for cycle detection")
	}
}

func BenchmarkVerifyChain(b *testing.B) {
	store := NewMemoryStore()
	policy := DefaultValidationPolicy()
	policy.VerifySignatures = false
	verifier := NewVerifier(store, policy)
	ctx := context.Background()

	// Create a chain of 100 receipts
	var prevHash *[32]byte
	var head [32]byte
	baseTime := uint64(time.Now().Unix() - 10000)

	for i := 0; i < 100; i++ {
		receipt := createTestReceipt(prevHash, baseTime+uint64(i*100), baseTime+uint64(i*100)+50)
		receipt.ChainID = "benchmark-chain"
		_ = store.SaveReceipt(ctx, receipt)
		hash := receipt.ReceiptHash
		prevHash = &hash
		head = hash
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyChain(ctx, head)
	}
}
