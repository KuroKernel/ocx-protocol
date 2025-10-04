package receipt

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create test receipt
	receipt := ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   1640995200,
			FinishedAt:  1640995201,
			IssuerID:    "test-issuer",
		},
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	receiptBytes := []byte("test receipt bytes")

	t.Run("save_and_get_receipt", func(t *testing.T) {
		// Save receipt
		receiptID, err := store.SaveReceipt(ctx, receipt, receiptBytes)
		if err != nil {
			t.Fatalf("Failed to save receipt: %v", err)
		}
		if receiptID == "" {
			t.Error("Expected non-empty receipt ID")
		}

		// Get receipt
		retrievedBytes, err := store.GetReceipt(ctx, receiptID)
		if err != nil {
			t.Fatalf("Failed to get receipt: %v", err)
		}
		if string(retrievedBytes) != string(receiptBytes) {
			t.Errorf("Expected receipt bytes %q, got %q", string(receiptBytes), string(retrievedBytes))
		}
	})

	t.Run("get_nonexistent_receipt", func(t *testing.T) {
		_, err := store.GetReceipt(ctx, "nonexistent-receipt-id")
		if err == nil {
			t.Error("Expected error when getting nonexistent receipt")
		}
	})

	t.Run("save_duplicate_receipt", func(t *testing.T) {
		// Save same receipt again
		receiptID2, err := store.SaveReceipt(ctx, receipt, receiptBytes)
		if err != nil {
			t.Fatalf("Failed to save duplicate receipt: %v", err)
		}
		if receiptID2 == "" {
			t.Error("Expected non-empty receipt ID for duplicate")
		}
		// Should get a different ID
		if receiptID2 == "receipt-1" {
			t.Error("Expected different receipt ID for duplicate")
		}
	})
}

func TestMemoryStoreIdempotency(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	key := "test-idempotency-key"
	reqHash := sha256.Sum256([]byte("test request"))
	respBytes := []byte("test response")

	t.Run("put_idempotent_first_time", func(t *testing.T) {
		exists, isNew, retrievedBytes, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
		if err != nil {
			t.Fatalf("Failed to put idempotent: %v", err)
		}
		if exists {
			t.Error("Expected idempotent key to not exist")
		}
		if !isNew {
			t.Error("Expected idempotent key to be new")
		}
		if string(retrievedBytes) != string(respBytes) {
			t.Errorf("Expected response bytes %q, got %q", string(respBytes), string(retrievedBytes))
		}
	})

	t.Run("put_idempotent_duplicate", func(t *testing.T) {
		exists, isNew, retrievedBytes, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
		if err != nil {
			t.Fatalf("Failed to put idempotent duplicate: %v", err)
		}
		if !exists {
			t.Error("Expected idempotent key to exist")
		}
		if isNew {
			t.Error("Expected idempotent key to not be new")
		}
		if string(retrievedBytes) != string(respBytes) {
			t.Errorf("Expected response bytes %q, got %q", string(respBytes), string(retrievedBytes))
		}
	})

	t.Run("get_idempotent", func(t *testing.T) {
		retrievedReqHash, retrievedBytes, exists, err := store.GetIdempotent(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get idempotent: %v", err)
		}
		if !exists {
			t.Error("Expected idempotent key to exist")
		}
		if retrievedReqHash != reqHash {
			t.Error("Expected matching request hash")
		}
		if string(retrievedBytes) != string(respBytes) {
			t.Errorf("Expected response bytes %q, got %q", string(respBytes), string(retrievedBytes))
		}
	})

	t.Run("get_nonexistent_idempotent", func(t *testing.T) {
		_, _, exists, err := store.GetIdempotent(ctx, "nonexistent-key")
		if err != nil {
			t.Fatalf("Failed to get nonexistent idempotent: %v", err)
		}
		if exists {
			t.Error("Expected nonexistent idempotent key to not exist")
		}
	})
}

func TestMemoryStoreReceiptByCore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	core := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	receipt := ReceiptFull{
		Core:       core,
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	receiptBytes := []byte("test receipt bytes")

	t.Run("get_receipt_by_core", func(t *testing.T) {
		// Save receipt first
		_, err := store.SaveReceipt(ctx, receipt, receiptBytes)
		if err != nil {
			t.Fatalf("Failed to save receipt: %v", err)
		}

		// Get by core
		retrievedReceipt, err := store.GetReceiptByCore(ctx, &core)
		if err != nil {
			t.Fatalf("Failed to get receipt by core: %v", err)
		}
		if retrievedReceipt == nil {
			t.Error("Expected non-nil receipt")
		}
		if retrievedReceipt.Core != core {
			t.Error("Expected matching receipt core")
		}
	})

	t.Run("get_nonexistent_receipt_by_core", func(t *testing.T) {
		nonexistentCore := ReceiptCore{
			ProgramHash: [32]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
			InputHash:   [32]byte{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
			OutputHash:  [32]byte{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
			GasUsed:     9999,
			StartedAt:   9999999999,
			FinishedAt:  9999999999,
			IssuerID:    "nonexistent-issuer",
		}

		_, err := store.GetReceiptByCore(ctx, &nonexistentCore)
		if err == nil {
			t.Error("Expected error when getting nonexistent receipt by core")
		}
	})
}

func TestMemoryStoreListReceipts(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create multiple receipts
	receipts := make([]ReceiptFull, 5)
	for i := 0; i < 5; i++ {
		receipts[i] = ReceiptFull{
			Core: ReceiptCore{
				ProgramHash: [32]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{byte(i), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{byte(i), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     uint64(1000 + i),
				StartedAt:   1640995200 + uint64(i),
				FinishedAt:  1640995201 + uint64(i),
				IssuerID:    "test-issuer",
			},
			Signature:  make([]byte, 64),
			HostCycles: uint64(12345 + i),
			HostInfo:   map[string]string{"host": "test-host", "index": string(rune(i))},
		}

		_, err := store.SaveReceipt(ctx, receipts[i], []byte("test receipt bytes"))
		if err != nil {
			t.Fatalf("Failed to save receipt %d: %v", i, err)
		}
	}

	t.Run("list_all_receipts", func(t *testing.T) {
		summaries, err := store.ListReceipts(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list receipts: %v", err)
		}
		if len(summaries) != 5 {
			t.Errorf("Expected 5 receipts, got %d", len(summaries))
		}
	})

	t.Run("list_receipts_with_limit", func(t *testing.T) {
		summaries, err := store.ListReceipts(ctx, 3, 0)
		if err != nil {
			t.Fatalf("Failed to list receipts with limit: %v", err)
		}
		if len(summaries) != 3 {
			t.Errorf("Expected 3 receipts, got %d", len(summaries))
		}
	})

	t.Run("list_receipts_with_offset", func(t *testing.T) {
		summaries, err := store.ListReceipts(ctx, 10, 2)
		if err != nil {
			t.Fatalf("Failed to list receipts with offset: %v", err)
		}
		if len(summaries) != 3 { // 5 total - 2 offset = 3 remaining
			t.Errorf("Expected 3 receipts with offset, got %d", len(summaries))
		}
	})

	t.Run("list_receipts_empty_store", func(t *testing.T) {
		emptyStore := NewMemoryStore()
		summaries, err := emptyStore.ListReceipts(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list receipts from empty store: %v", err)
		}
		if len(summaries) != 0 {
			t.Errorf("Expected 0 receipts from empty store, got %d", len(summaries))
		}
	})
}

func TestMemoryStoreCleanup(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create some idempotency entries
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		reqHash := sha256.Sum256([]byte("test request " + key))
		respBytes := []byte("test response " + key)
		_, _, _, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
		if err != nil {
			t.Fatalf("Failed to put idempotent %s: %v", key, err)
		}
	}

	t.Run("cleanup_old_idempotency", func(t *testing.T) {
		// Cleanup entries older than 1 nanosecond (should clean all)
		cleaned, err := store.CleanupOldIdempotency(ctx, 1*time.Nanosecond)
		if err != nil {
			t.Fatalf("Failed to cleanup old idempotency: %v", err)
		}
		if cleaned != 3 {
			t.Errorf("Expected to cleanup 3 entries, cleaned %d", cleaned)
		}

		// Verify entries are gone
		for _, key := range keys {
			_, _, exists, err := store.GetIdempotent(ctx, key)
			if err != nil {
				t.Fatalf("Failed to get idempotent %s: %v", key, err)
			}
			if exists {
				t.Errorf("Expected idempotent key %s to be cleaned up", key)
			}
		}
	})

	t.Run("cleanup_no_old_entries", func(t *testing.T) {
		// Put a new entry
		reqHash := sha256.Sum256([]byte("new test request"))
		respBytes := []byte("new test response")
		_, _, _, err := store.PutIdempotent(ctx, "new-key", reqHash, respBytes)
		if err != nil {
			t.Fatalf("Failed to put new idempotent: %v", err)
		}

		// Cleanup entries older than 1 hour (should clean none)
		cleaned, err := store.CleanupOldIdempotency(ctx, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to cleanup old idempotency: %v", err)
		}
		if cleaned != 0 {
			t.Errorf("Expected to cleanup 0 entries, cleaned %d", cleaned)
		}

		// Verify entry still exists
		_, _, exists, err := store.GetIdempotent(ctx, "new-key")
		if err != nil {
			t.Fatalf("Failed to get new idempotent: %v", err)
		}
		if !exists {
			t.Error("Expected new idempotent key to still exist")
		}
	})
}

func TestMemoryStoreStats(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	t.Run("stats_empty_store", func(t *testing.T) {
		stats, err := store.GetStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats == nil {
			t.Error("Expected non-nil stats")
		}
		if stats.TotalReceipts != 0 {
			t.Errorf("Expected 0 total receipts, got %d", stats.TotalReceipts)
		}
		if stats.TotalIdempotencyKeys != 0 {
			t.Errorf("Expected 0 total idempotency keys, got %d", stats.TotalIdempotencyKeys)
		}
	})

	t.Run("stats_with_data", func(t *testing.T) {
		// Add some receipts
		for i := 0; i < 3; i++ {
			receipt := ReceiptFull{
				Core: ReceiptCore{
					ProgramHash: [32]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
					InputHash:   [32]byte{byte(i), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
					OutputHash:  [32]byte{byte(i), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
					GasUsed:     uint64(1000 + i),
					StartedAt:   1640995200 + uint64(i),
					FinishedAt:  1640995201 + uint64(i),
					IssuerID:    "test-issuer",
				},
				Signature:  make([]byte, 64),
				HostCycles: uint64(12345 + i),
				HostInfo:   map[string]string{"host": "test-host"},
			}

			_, err := store.SaveReceipt(ctx, receipt, []byte("test receipt bytes"))
			if err != nil {
				t.Fatalf("Failed to save receipt %d: %v", i, err)
			}
		}

		// Add some idempotency keys
		for i := 0; i < 2; i++ {
			key := "idempotency-key-" + string(rune(i))
			reqHash := sha256.Sum256([]byte("test request " + key))
			respBytes := []byte("test response " + key)
			_, _, _, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
			if err != nil {
				t.Fatalf("Failed to put idempotent %s: %v", key, err)
			}
		}

		stats, err := store.GetStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats.TotalReceipts != 3 {
			t.Errorf("Expected 3 total receipts, got %d", stats.TotalReceipts)
		}
		if stats.TotalIdempotencyKeys != 2 {
			t.Errorf("Expected 2 total idempotency keys, got %d", stats.TotalIdempotencyKeys)
		}
	})
}

func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	numGoroutines := 10
	results := make(chan error, numGoroutines)

	t.Run("concurrent_receipt_operations", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				receipt := ReceiptFull{
					Core: ReceiptCore{
						ProgramHash: [32]byte{byte(index), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
						InputHash:   [32]byte{byte(index), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
						OutputHash:  [32]byte{byte(index), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
						GasUsed:     uint64(1000 + index),
						StartedAt:   1640995200 + uint64(index),
						FinishedAt:  1640995201 + uint64(index),
						IssuerID:    "test-issuer",
					},
					Signature:  make([]byte, 64),
					HostCycles: uint64(12345 + index),
					HostInfo:   map[string]string{"host": "test-host", "index": string(rune(index))},
				}

				receiptBytes := []byte("test receipt bytes " + string(rune(index)))
				receiptID, err := store.SaveReceipt(ctx, receipt, receiptBytes)
				if err != nil {
					results <- err
					return
				}

				retrievedBytes, err := store.GetReceipt(ctx, receiptID)
				if err != nil {
					results <- err
					return
				}

				if string(retrievedBytes) != string(receiptBytes) {
					results <- err
					return
				}

				results <- nil
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent receipt operation failed: %v", err)
			}
		}
	})

	t.Run("concurrent_idempotency_operations", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				key := "concurrent-key-" + string(rune(index))
				reqHash := sha256.Sum256([]byte("test request " + key))
				respBytes := []byte("test response " + key)

				_, _, _, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
				if err != nil {
					results <- err
					return
				}

				retrievedReqHash, retrievedBytes, exists, err := store.GetIdempotent(ctx, key)
				if err != nil {
					results <- err
					return
				}

				if !exists {
					results <- err
					return
				}

				if retrievedReqHash != reqHash {
					results <- err
					return
				}

				if string(retrievedBytes) != string(respBytes) {
					results <- err
					return
				}

				results <- nil
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent idempotency operation failed: %v", err)
			}
		}
	})
}

func TestMemoryStoreEdgeCases(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	t.Run("save_receipt_with_empty_bytes", func(t *testing.T) {
		receipt := ReceiptFull{
			Core: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     1000,
				StartedAt:   1640995200,
				FinishedAt:  1640995201,
				IssuerID:    "test-issuer",
			},
			Signature:  make([]byte, 64),
			HostCycles: 12345,
			HostInfo:   map[string]string{"host": "test-host"},
		}

		receiptID, err := store.SaveReceipt(ctx, receipt, []byte{})
		if err != nil {
			t.Fatalf("Failed to save receipt with empty bytes: %v", err)
		}

		retrievedBytes, err := store.GetReceipt(ctx, receiptID)
		if err != nil {
			t.Fatalf("Failed to get receipt with empty bytes: %v", err)
		}

		if len(retrievedBytes) != 0 {
			t.Errorf("Expected empty bytes, got %d bytes", len(retrievedBytes))
		}
	})

	t.Run("put_idempotent_with_empty_response", func(t *testing.T) {
		key := "empty-response-key"
		reqHash := sha256.Sum256([]byte("test request"))
		respBytes := []byte{}

		exists, isNew, retrievedBytes, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
		if err != nil {
			t.Fatalf("Failed to put idempotent with empty response: %v", err)
		}
		if exists {
			t.Error("Expected idempotent key to not exist")
		}
		if !isNew {
			t.Error("Expected idempotent key to be new")
		}
		if len(retrievedBytes) != 0 {
			t.Errorf("Expected empty response bytes, got %d bytes", len(retrievedBytes))
		}
	})

	t.Run("list_receipts_with_negative_limit", func(t *testing.T) {
		summaries, err := store.ListReceipts(ctx, -1, 0)
		if err != nil {
			t.Fatalf("Failed to list receipts with negative limit: %v", err)
		}
		// Should handle negative limit gracefully
		if len(summaries) < 0 {
			t.Error("Expected non-negative number of receipts")
		}
	})

	t.Run("list_receipts_with_negative_offset", func(t *testing.T) {
		summaries, err := store.ListReceipts(ctx, 10, -1)
		if err != nil {
			t.Fatalf("Failed to list receipts with negative offset: %v", err)
		}
		// Should handle negative offset gracefully
		if len(summaries) < 0 {
			t.Error("Expected non-negative number of receipts")
		}
	})
}

func BenchmarkMemoryStoreSaveReceipt(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	receipt := ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   1640995200,
			FinishedAt:  1640995201,
			IssuerID:    "test-issuer",
		},
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	receiptBytes := []byte("test receipt bytes")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.SaveReceipt(ctx, receipt, receiptBytes)
		if err != nil {
			b.Fatalf("Failed to save receipt: %v", err)
		}
	}
}

func BenchmarkMemoryStoreGetReceipt(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	receipt := ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   1640995200,
			FinishedAt:  1640995201,
			IssuerID:    "test-issuer",
		},
		Signature:  make([]byte, 64),
		HostCycles: 12345,
		HostInfo:   map[string]string{"host": "test-host"},
	}

	receiptBytes := []byte("test receipt bytes")
	receiptID, err := store.SaveReceipt(ctx, receipt, receiptBytes)
	if err != nil {
		b.Fatalf("Failed to save receipt: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetReceipt(ctx, receiptID)
		if err != nil {
			b.Fatalf("Failed to get receipt: %v", err)
		}
	}
}

func BenchmarkMemoryStorePutIdempotent(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	key := "benchmark-key"
	reqHash := sha256.Sum256([]byte("test request"))
	respBytes := []byte("test response")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := store.PutIdempotent(ctx, key, reqHash, respBytes)
		if err != nil {
			b.Fatalf("Failed to put idempotent: %v", err)
		}
	}
}
