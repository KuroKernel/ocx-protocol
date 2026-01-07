package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"ocx.local/pkg/receipt"
)

// FuzzReplayStore fuzzes the replay store nonce checking
func FuzzReplayStore(f *testing.F) {
	// Add seed corpus
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, uint64(time.Now().UnixNano()))
	f.Add(make([]byte, 16), uint64(0))
	f.Add(make([]byte, 16), uint64(1640995200000000000))

	f.Fuzz(func(t *testing.T, nonce []byte, timestamp uint64) {
		store := NewInMemoryReplayStore(time.Hour, 5*time.Minute)

		// Should not panic on any input
		ok, err := store.CheckAndStore(nonce, timestamp)

		// Verify behavior for valid nonces
		if len(nonce) == 16 {
			// Check timestamp bounds
			now := uint64(time.Now().UnixNano())
			clockSkew := uint64(5 * time.Minute.Nanoseconds())
			retention := uint64(time.Hour.Nanoseconds())

			isFuture := timestamp > now+clockSkew
			isTooOld := timestamp < now-retention

			if isFuture || isTooOld {
				if err == nil {
					// Expected error for out-of-bounds timestamp
					return
				}
			} else if err != nil {
				t.Fatalf("Unexpected error for valid nonce and timestamp: %v", err)
			}

			// If first check succeeded, second should fail (replay)
			if ok && err == nil {
				ok2, err2 := store.CheckAndStore(nonce, timestamp)
				if err2 != nil {
					t.Fatalf("Second check returned error: %v", err2)
				}
				if ok2 {
					t.Fatal("Replay not detected")
				}
			}
		} else {
			// Invalid nonce length should return error
			if err == nil {
				t.Fatalf("Expected error for nonce length %d", len(nonce))
			}
		}
	})
}

// FuzzBatchVerifier fuzzes batch verification with random receipts
func FuzzBatchVerifier(f *testing.F) {
	// Add seed corpus
	f.Add([]byte("valid receipt data"), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	f.Add([]byte{}, []byte{})
	f.Add([]byte{0xa2, 0x64}, make([]byte, 32)) // Truncated CBOR

	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 2})

	f.Fuzz(func(t *testing.T, receiptData, pubKey []byte) {
		batch := []ReceiptBatch{{
			ReceiptData: receiptData,
			PublicKey:   pubKey,
		}}

		// Should not panic on any input
		results, stats := bv.VerifyBatch(context.Background(), batch)

		// Should always return one result
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Stats should be consistent
		if stats.Valid+stats.Invalid != 1 {
			t.Fatalf("Stats inconsistent: valid=%d invalid=%d", stats.Valid, stats.Invalid)
		}

		// If invalid, should have error
		if !results[0].Valid && results[0].Error == nil {
			t.Fatal("Invalid result should have error")
		}
	})
}

// FuzzVerifyReceipt fuzzes single receipt verification
func FuzzVerifyReceipt(f *testing.F) {
	// Generate a valid receipt for seed
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

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
		IssuerID:    "fuzz-test",
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	coreBytes, _ := receipt.CanonicalizeCore(&core)
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	full := &receipt.ReceiptFull{
		Core:       core,
		Signature:  sig,
		HostCycles: 5000,
	}
	validReceipt, _ := receipt.CanonicalizeFull(full)

	f.Add(validReceipt, []byte(pub))
	f.Add([]byte{}, []byte{})
	f.Add([]byte("invalid"), make([]byte, 32))

	verifier := NewGoVerifier()

	f.Fuzz(func(t *testing.T, receiptData, pubKey []byte) {
		// Should not panic on any input
		_, _ = verifier.VerifyReceipt(receiptData, pubKey)
	})
}

// FuzzExtractReceiptFields fuzzes field extraction
func FuzzExtractReceiptFields(f *testing.F) {
	// Add seed corpus with various CBOR patterns
	f.Add([]byte{})
	f.Add([]byte{0xa0})        // Empty map
	f.Add([]byte{0xa1, 0x00})  // Truncated map
	f.Add([]byte{0xbf, 0xff})  // Indefinite map
	f.Add([]byte("not cbor"))

	// Add a valid receipt
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     1000,
		Nonce:       nonce,
		IssuerID:    "fuzz",
	}
	full := &receipt.ReceiptFull{
		Core:      core,
		Signature: make([]byte, 64),
	}
	validReceipt, _ := receipt.CanonicalizeFull(full)
	f.Add(validReceipt)

	verifier := NewGoVerifier()

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		fields, err := verifier.ExtractReceiptFields(data)

		if err == nil && fields == nil {
			t.Fatal("No error but nil fields")
		}
	})
}

// FuzzBatchVerifyMixed fuzzes batch verification with mixed valid/invalid receipts
func FuzzBatchVerifyMixed(f *testing.F) {
	f.Add(uint8(1), uint8(0))  // 1 valid, 0 invalid
	f.Add(uint8(0), uint8(1))  // 0 valid, 1 invalid
	f.Add(uint8(5), uint8(5))  // Mixed
	f.Add(uint8(10), uint8(0)) // All valid

	bv, _ := NewBatchVerifier(BatchVerifierConfig{Workers: 4})

	f.Fuzz(func(t *testing.T, validCount, invalidCount uint8) {
		// Limit sizes for fuzzing performance
		if validCount > 20 {
			validCount = 20
		}
		if invalidCount > 20 {
			invalidCount = 20
		}

		total := int(validCount) + int(invalidCount)
		if total == 0 {
			return
		}

		batches := make([]ReceiptBatch, total)

		// Create valid receipts
		for i := 0; i < int(validCount); i++ {
			pub, priv, _ := ed25519.GenerateKey(rand.Reader)

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
				GasUsed:     uint64(1000 + i),
				StartedAt:   now - 1000000,
				FinishedAt:  now - 500000,
				IssuerID:    "valid-issuer",
				KeyVersion:  1,
				Nonce:       nonce,
				IssuedAt:    now,
				FloatMode:   "disabled",
			}

			coreBytes, _ := receipt.CanonicalizeCore(&core)
			msg := append([]byte("OCXv1|receipt|"), coreBytes...)
			sig := ed25519.Sign(priv, msg)

			full := &receipt.ReceiptFull{
				Core:       core,
				Signature:  sig,
				HostCycles: 5000,
			}
			data, _ := receipt.CanonicalizeFull(full)

			batches[i] = ReceiptBatch{
				ReceiptData: data,
				PublicKey:   pub,
			}
		}

		// Create invalid receipts (wrong public key)
		for i := 0; i < int(invalidCount); i++ {
			_, priv, _ := ed25519.GenerateKey(rand.Reader)
			wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)

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
				GasUsed:     uint64(2000 + i),
				StartedAt:   now - 1000000,
				FinishedAt:  now - 500000,
				IssuerID:    "invalid-issuer",
				KeyVersion:  1,
				Nonce:       nonce,
				IssuedAt:    now,
				FloatMode:   "disabled",
			}

			coreBytes, _ := receipt.CanonicalizeCore(&core)
			msg := append([]byte("OCXv1|receipt|"), coreBytes...)
			sig := ed25519.Sign(priv, msg)

			full := &receipt.ReceiptFull{
				Core:       core,
				Signature:  sig,
				HostCycles: 5000,
			}
			data, _ := receipt.CanonicalizeFull(full)

			batches[int(validCount)+i] = ReceiptBatch{
				ReceiptData: data,
				PublicKey:   wrongPub, // Wrong key!
			}
		}

		// Verify batch
		results, stats := bv.VerifyBatch(context.Background(), batches)

		// Check result count
		if len(results) != total {
			t.Fatalf("Expected %d results, got %d", total, len(results))
		}

		// Check stats
		if stats.Valid != int(validCount) {
			t.Errorf("Expected %d valid, got %d", validCount, stats.Valid)
		}
		if stats.Invalid != int(invalidCount) {
			t.Errorf("Expected %d invalid, got %d", invalidCount, stats.Invalid)
		}

		// Verify order preserved
		for i := 0; i < int(validCount); i++ {
			if !results[i].Valid {
				t.Errorf("Result %d should be valid", i)
			}
		}
		for i := int(validCount); i < total; i++ {
			if results[i].Valid {
				t.Errorf("Result %d should be invalid", i)
			}
		}
	})
}

// FuzzReplayStoreConcurrent fuzzes concurrent access to replay store
func FuzzReplayStoreConcurrent(f *testing.F) {
	f.Add(uint8(10), uint8(5))  // 10 goroutines, 5 nonces each
	f.Add(uint8(50), uint8(10)) // More goroutines

	f.Fuzz(func(t *testing.T, numGoroutines, noncesPerGoroutine uint8) {
		if numGoroutines == 0 || noncesPerGoroutine == 0 {
			return
		}
		if numGoroutines > 50 {
			numGoroutines = 50
		}
		if noncesPerGoroutine > 20 {
			noncesPerGoroutine = 20
		}

		store := NewInMemoryReplayStore(time.Hour, time.Minute)
		now := uint64(time.Now().UnixNano())

		done := make(chan bool, numGoroutines)

		for i := uint8(0); i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				for j := uint8(0); j < noncesPerGoroutine; j++ {
					nonce := make([]byte, 16)
					rand.Read(nonce)

					// Should not panic
					_, _ = store.CheckAndStore(nonce, now)
				}
			}()
		}

		// Wait for all goroutines
		for i := uint8(0); i < numGoroutines; i++ {
			<-done
		}

		// Store size should be reasonable
		size := store.Size()
		maxSize := int(numGoroutines) * int(noncesPerGoroutine)
		if size > maxSize {
			t.Fatalf("Store size %d exceeds max %d", size, maxSize)
		}
	})
}
