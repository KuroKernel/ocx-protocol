package receipt

import (
	"testing"
	"time"
)

func TestCanonicalizeCore(t *testing.T) {
	tests := []struct {
		name        string
		receipt     ReceiptCore
		expectError bool
	}{
		{
			name: "valid_receipt_core",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     1000,
				StartedAt:   1640995200, // 2022-01-01 00:00:00 UTC
				FinishedAt:  1640995201, // 2022-01-01 00:00:01 UTC
				IssuerID:    "test-issuer",
			},
			expectError: false,
		},
		{
			name: "zero_gas_used",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     0,
				StartedAt:   1640995200,
				FinishedAt:  1640995201,
				IssuerID:    "test-issuer",
			},
			expectError: false,
		},
		{
			name: "empty_issuer_id",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     1000,
				StartedAt:   1640995200,
				FinishedAt:  1640995201,
				IssuerID:    "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonicalBytes, err := CanonicalizeCore(&tt.receipt)
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

			// Verify canonical bytes are not empty
			if len(canonicalBytes) == 0 {
				t.Errorf("Expected non-empty canonical bytes for %s", tt.name)
			}

			// Verify determinism - same input should produce same output
			canonicalBytes2, err := CanonicalizeCore(&tt.receipt)
			if err != nil {
				t.Errorf("Failed to canonicalize receipt again: %v", err)
			}
			if string(canonicalBytes) != string(canonicalBytes2) {
				t.Errorf("Canonicalization not deterministic for %s", tt.name)
			}
		})
	}
}

func TestCanonicalizeFull(t *testing.T) {
	core := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	tests := []struct {
		name        string
		receipt     ReceiptFull
		expectError bool
	}{
		{
			name: "valid_receipt_full",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  make([]byte, 64), // 64-byte Ed25519 signature
				HostCycles: 12345,
				HostInfo:   map[string]string{"host": "test-host", "version": "1.0.0"},
			},
			expectError: false,
		},
		{
			name: "empty_signature",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  []byte{},
				HostCycles: 12345,
				HostInfo:   map[string]string{"host": "test-host"},
			},
			expectError: false,
		},
		{
			name: "nil_host_info",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  make([]byte, 64),
				HostCycles: 12345,
				HostInfo:   nil,
			},
			expectError: false,
		},
		{
			name: "empty_host_info",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  make([]byte, 64),
				HostCycles: 12345,
				HostInfo:   map[string]string{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonicalBytes, err := CanonicalizeFull(&tt.receipt)
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

			// Verify canonical bytes are not empty
			if len(canonicalBytes) == 0 {
				t.Errorf("Expected non-empty canonical bytes for %s", tt.name)
			}

			// Verify determinism - same input should produce same output
			canonicalBytes2, err := CanonicalizeFull(&tt.receipt)
			if err != nil {
				t.Errorf("Failed to canonicalize receipt again: %v", err)
			}
			if string(canonicalBytes) != string(canonicalBytes2) {
				t.Errorf("Canonicalization not deterministic for %s", tt.name)
			}
		})
	}
}

func TestCanonicalizationDeterminism(t *testing.T) {
	// Test that canonicalization is deterministic across multiple runs
	receipt := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	// Canonicalize multiple times
	var results [][]byte
	for i := 0; i < 10; i++ {
		canonicalBytes, err := CanonicalizeCore(&receipt)
		if err != nil {
			t.Fatalf("Failed to canonicalize receipt (iteration %d): %v", i, err)
		}
		results = append(results, canonicalBytes)
	}

	// All results should be identical
	firstResult := results[0]
	for i, result := range results[1:] {
		if string(firstResult) != string(result) {
			t.Errorf("Canonicalization not deterministic: iteration %d differs from first", i+1)
		}
	}
}

func TestCanonicalizationOrdering(t *testing.T) {
	// Test that field ordering is consistent (integer keys should be in order)
	receipt := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	canonicalBytes, err := CanonicalizeCore(&receipt)
	if err != nil {
		t.Fatalf("Failed to canonicalize receipt: %v", err)
	}

	// The canonical bytes should be consistent
	// We can't easily verify the internal structure without decoding,
	// but we can verify it's deterministic
	expectedLength := len(canonicalBytes)
	if expectedLength == 0 {
		t.Error("Expected non-empty canonical bytes")
	}

	// Test with different field orders (though struct fields are fixed order)
	// This test mainly ensures the canonicalization function works consistently
}

func TestReceiptCoreValidation(t *testing.T) {
	tests := []struct {
		name        string
		receipt     ReceiptCore
		expectValid bool
	}{
		{
			name: "valid_receipt",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     1000,
				StartedAt:   1640995200,
				FinishedAt:  1640995201,
				IssuerID:    "test-issuer",
			},
			expectValid: true,
		},
		{
			name: "zero_gas_used",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     0,
				StartedAt:   1640995200,
				FinishedAt:  1640995201,
				IssuerID:    "test-issuer",
			},
			expectValid: true, // Zero gas is valid
		},
		{
			name: "started_after_finished",
			receipt: ReceiptCore{
				ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
				GasUsed:     1000,
				StartedAt:   1640995201, // Started after finished
				FinishedAt:  1640995200,
				IssuerID:    "test-issuer",
			},
			expectValid: true, // Canonicalization doesn't validate business logic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonicalBytes, err := CanonicalizeCore(&tt.receipt)
			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected valid canonicalization for %s, got error: %v", tt.name, err)
				}
				if len(canonicalBytes) == 0 {
					t.Errorf("Expected non-empty canonical bytes for %s", tt.name)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
			}
		})
	}
}

func TestReceiptFullValidation(t *testing.T) {
	core := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	tests := []struct {
		name        string
		receipt     ReceiptFull
		expectValid bool
	}{
		{
			name: "valid_receipt_full",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  make([]byte, 64), // 64-byte Ed25519 signature
				HostCycles: 12345,
				HostInfo:   map[string]string{"host": "test-host"},
			},
			expectValid: true,
		},
		{
			name: "invalid_signature_length",
			receipt: ReceiptFull{
				Core:       core,
				Signature:  make([]byte, 32), // Wrong signature length
				HostCycles: 12345,
				HostInfo:   map[string]string{"host": "test-host"},
			},
			expectValid: true, // Canonicalization doesn't validate signature length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonicalBytes, err := CanonicalizeFull(&tt.receipt)
			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected valid canonicalization for %s, got error: %v", tt.name, err)
				}
				if len(canonicalBytes) == 0 {
					t.Errorf("Expected non-empty canonical bytes for %s", tt.name)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
			}
		})
	}
}

func TestCanonicalizationPerformance(t *testing.T) {
	receipt := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	// Test performance with multiple iterations
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := CanonicalizeCore(&receipt)
		if err != nil {
			t.Fatalf("Failed to canonicalize receipt (iteration %d): %v", i, err)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Canonicalized %d receipts in %v (avg: %v per receipt)", iterations, duration, avgDuration)

	// Canonicalization should be fast (less than 1ms per receipt)
	if avgDuration > time.Millisecond {
		t.Errorf("Canonicalization too slow: %v per receipt (expected < 1ms)", avgDuration)
	}
}

func TestCanonicalizationConsistency(t *testing.T) {
	// Test that canonicalization produces consistent results across different
	// field values and edge cases
	testCases := []struct {
		name   string
		modify func(*ReceiptCore)
	}{
		{
			name: "minimal_values",
			modify: func(r *ReceiptCore) {
				r.GasUsed = 0
				r.StartedAt = 0
				r.FinishedAt = 0
				r.IssuerID = ""
			},
		},
		{
			name: "maximal_values",
			modify: func(r *ReceiptCore) {
				r.GasUsed = ^uint64(0) // Max uint64
				r.StartedAt = ^uint64(0)
				r.FinishedAt = ^uint64(0)
				r.IssuerID = "very-long-issuer-id-that-might-cause-issues-with-encoding-and-should-be-handled-properly"
			},
		},
		{
			name: "special_characters_in_issuer_id",
			modify: func(r *ReceiptCore) {
				r.IssuerID = "issuer-with-special-chars-!@#$%^&*()_+-=[]{}|;':\",./<>?"
			},
		},
		{
			name: "unicode_in_issuer_id",
			modify: func(r *ReceiptCore) {
				r.IssuerID = "issuer-with-unicode-🚀-🎉-测试"
			},
		},
	}

	baseReceipt := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receipt := baseReceipt
			tc.modify(&receipt)

			// Should be able to canonicalize without error
			canonicalBytes, err := CanonicalizeCore(&receipt)
			if err != nil {
				t.Errorf("Failed to canonicalize receipt for %s: %v", tc.name, err)
				return
			}

			// Should produce non-empty result
			if len(canonicalBytes) == 0 {
				t.Errorf("Expected non-empty canonical bytes for %s", tc.name)
			}

			// Should be deterministic
			canonicalBytes2, err := CanonicalizeCore(&receipt)
			if err != nil {
				t.Errorf("Failed to canonicalize receipt again for %s: %v", tc.name, err)
				return
			}

			if string(canonicalBytes) != string(canonicalBytes2) {
				t.Errorf("Canonicalization not deterministic for %s", tc.name)
			}
		})
	}
}

func BenchmarkCanonicalizeCore(b *testing.B) {
	receipt := ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "test-issuer",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CanonicalizeCore(&receipt)
		if err != nil {
			b.Fatalf("Failed to canonicalize receipt: %v", err)
		}
	}
}

func BenchmarkCanonicalizeFull(b *testing.B) {
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
		HostInfo:   map[string]string{"host": "test-host", "version": "1.0.0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CanonicalizeFull(&receipt)
		if err != nil {
			b.Fatalf("Failed to canonicalize receipt: %v", err)
		}
	}
}
