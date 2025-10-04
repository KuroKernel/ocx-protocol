package performance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

// PerformanceTestSuite provides comprehensive performance testing for the OCX Protocol
type PerformanceTestSuite struct {
	keystore *keystore.Keystore
	signer   keystore.Signer
	verifier *verify.GoVerifier
	tempDir  string
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(t *testing.T) *PerformanceTestSuite {
	tempDir := t.TempDir()
	
	ks, err := keystore.New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	keyID, err := ks.GenerateKey("performance-test-key")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	signer := keystore.NewLocalSigner(ks)
	verifier := verify.NewGoVerifier()

	return &PerformanceTestSuite{
		keystore: ks,
		signer:   signer,
		verifier: verifier,
		tempDir:  tempDir,
	}
}

// TestReceiptSigningPerformance tests the performance of receipt signing
func TestReceiptSigningPerformance(t *testing.T) {
	suite := NewPerformanceTestSuite(t)
	ctx := context.Background()

	// Create test receipt core
	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "performance-test-issuer",
	}

	t.Run("canonicalization_performance", func(t *testing.T) {
		iterations := 10000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := receipt.CanonicalizeCore(&receiptCore)
			if err != nil {
				t.Fatalf("Failed to canonicalize receipt core: %v", err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Canonicalized %d receipts in %v (avg: %v per receipt)", iterations, duration, avgDuration)

		// Canonicalization should be very fast (< 100μs per receipt)
		if avgDuration > 100*time.Microsecond {
			t.Errorf("Canonicalization too slow: %v per receipt (expected < 100μs)", avgDuration)
		}
	})

	t.Run("signing_performance", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
			if err != nil {
				t.Fatalf("Failed to canonicalize receipt core: %v", err)
			}

			messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
			_, _, err = suite.signer.Sign(ctx, "performance-test-key", messageToSign)
			if err != nil {
				t.Fatalf("Failed to sign receipt: %v", err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Signed %d receipts in %v (avg: %v per receipt)", iterations, duration, avgDuration)

		// Signing should be fast (< 1ms per receipt)
		if avgDuration > time.Millisecond {
			t.Errorf("Signing too slow: %v per receipt (expected < 1ms)", avgDuration)
		}
	})

	t.Run("verification_performance", func(t *testing.T) {
		// Create a signed receipt first
		coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
		if err != nil {
			t.Fatalf("Failed to canonicalize receipt core: %v", err)
		}

		messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
		signature, pubKey, err := suite.signer.Sign(ctx, "performance-test-key", messageToSign)
		if err != nil {
			t.Fatalf("Failed to sign receipt: %v", err)
		}

		receiptFull := receipt.ReceiptFull{
			Core:        receiptCore,
			Signature:   signature,
			HostCycles:  12345,
			HostInfo:    map[string]string{"host": "performance-test-host"},
		}

		fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
		if err != nil {
			t.Fatalf("Failed to canonicalize full receipt: %v", err)
		}

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := suite.verifier.VerifyReceipt(fullReceiptBytes, pubKey)
			if err != nil {
				t.Fatalf("Failed to verify receipt: %v", err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Verified %d receipts in %v (avg: %v per receipt)", iterations, duration, avgDuration)

		// Verification should be fast (< 1ms per receipt)
		if avgDuration > time.Millisecond {
			t.Errorf("Verification too slow: %v per receipt (expected < 1ms)", avgDuration)
		}
	})
}

// TestDeterministicVMPerformance tests the performance of the deterministic VM
func TestDeterministicVMPerformance(t *testing.T) {
	suite := NewPerformanceTestSuite(t)
	ctx := context.Background()

	// Create test artifacts
	artifacts := map[string]string{
		"simple_echo": `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`,
		"input_processing": `#!/bin/bash
echo "Processing input..."
wc -c < input.bin
exit 0`,
		"text_processing": `#!/bin/bash
echo "Text processing..."
tr 'a-z' 'A-Z' < input.bin
exit 0`,
		"large_input": `#!/bin/bash
echo "Processing large input..."
head -c 1000 input.bin | wc -c
exit 0`,
	}

	// Create artifact files
	artifactHashes := make(map[string][32]byte)
	for name, script := range artifacts {
		artifactPath := filepath.Join(suite.tempDir, name)
		err := os.WriteFile(artifactPath, []byte(script), 0755)
		if err != nil {
			t.Fatalf("Failed to create artifact %s: %v", name, err)
		}

		artifactBytes, err := os.ReadFile(artifactPath)
		if err != nil {
			t.Fatalf("Failed to read artifact %s: %v", name, err)
		}

		artifactHashes[name] = sha256.Sum256(artifactBytes)
	}

	// Test inputs
	testInputs := map[string][]byte{
		"simple_echo":      []byte(""),
		"input_processing": []byte("Hello, World!"),
		"text_processing":  []byte("hello world"),
		"large_input":      make([]byte, 10000), // 10KB input
	}

	// Fill large input with test data
	for i := range testInputs["large_input"] {
		testInputs["large_input"][i] = byte(i % 256)
	}

	t.Run("vm_execution_performance", func(t *testing.T) {
		for name, artifactHash := range artifactHashes {
			t.Run(name, func(t *testing.T) {
				input := testInputs[name]
				iterations := 100

				start := time.Now()
				for i := 0; i < iterations; i++ {
					result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
					if err != nil {
						t.Fatalf("Failed to execute artifact %s: %v", name, err)
					}

					// Verify result is reasonable
					if result.ExitCode != 0 {
						t.Errorf("Expected exit code 0 for %s, got %d", name, result.ExitCode)
					}
					if result.GasUsed == 0 {
						t.Errorf("Expected non-zero gas usage for %s", name)
					}
				}

				duration := time.Since(start)
				avgDuration := duration / time.Duration(iterations)

				t.Logf("Executed %s %d times in %v (avg: %v per execution)", name, iterations, duration, avgDuration)

				// VM execution should be reasonably fast (< 100ms per execution)
				if avgDuration > 100*time.Millisecond {
					t.Errorf("VM execution too slow for %s: %v per execution (expected < 100ms)", name, avgDuration)
				}
			})
		}
	})

	t.Run("vm_determinism_performance", func(t *testing.T) {
		// Test determinism with multiple executions
		artifactHash := artifactHashes["simple_echo"]
		input := testInputs["simple_echo"]
		iterations := 1000

		start := time.Now()
		var firstResult *deterministicvm.ExecutionResult

		for i := 0; i < iterations; i++ {
			result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
			if err != nil {
				t.Fatalf("Failed to execute artifact: %v", err)
			}

			if i == 0 {
				firstResult = result
			} else {
				// Verify determinism
				if result.ExitCode != firstResult.ExitCode {
					t.Errorf("Non-deterministic exit code: expected %d, got %d", firstResult.ExitCode, result.ExitCode)
				}
				if result.GasUsed != firstResult.GasUsed {
					t.Errorf("Non-deterministic gas usage: expected %d, got %d", firstResult.GasUsed, result.GasUsed)
				}
				if string(result.Stdout) != string(firstResult.Stdout) {
					t.Errorf("Non-deterministic output: expected %q, got %q", string(firstResult.Stdout), string(result.Stdout))
				}
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Verified determinism for %d executions in %v (avg: %v per execution)", iterations, duration, avgDuration)

		// Determinism verification should be fast
		if avgDuration > 50*time.Millisecond {
			t.Errorf("Determinism verification too slow: %v per execution (expected < 50ms)", avgDuration)
		}
	})
}

// TestConcurrentPerformance tests performance under concurrent load
func TestConcurrentPerformance(t *testing.T) {
	suite := NewPerformanceTestSuite(t)
	ctx := context.Background()

	// Create test artifact
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
echo "Input: $(cat input.bin)"
exit 0`

	artifactPath := filepath.Join(suite.tempDir, "concurrent-test-artifact")
	err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)

	t.Run("concurrent_vm_execution", func(t *testing.T) {
		numGoroutines := 10
		iterationsPerGoroutine := 100
		totalIterations := numGoroutines * iterationsPerGoroutine

		var wg sync.WaitGroup
		results := make(chan error, numGoroutines)
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				input := []byte(fmt.Sprintf("test input from goroutine %d", goroutineID))
				for j := 0; j < iterationsPerGoroutine; j++ {
					result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
					if err != nil {
						results <- fmt.Errorf("goroutine %d, iteration %d: %v", goroutineID, j, err)
						return
					}

					if result.ExitCode != 0 {
						results <- fmt.Errorf("goroutine %d, iteration %d: expected exit code 0, got %d", goroutineID, j, result.ExitCode)
						return
					}
				}
				results <- nil
			}(i)
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)

		// Check for errors
		for err := range results {
			if err != nil {
				t.Errorf("Concurrent execution error: %v", err)
			}
		}

		avgDuration := duration / time.Duration(totalIterations)
		t.Logf("Executed %d concurrent operations in %v (avg: %v per operation)", totalIterations, duration, avgDuration)

		// Concurrent execution should be reasonably fast
		if avgDuration > 200*time.Millisecond {
			t.Errorf("Concurrent execution too slow: %v per operation (expected < 200ms)", avgDuration)
		}
	})

	t.Run("concurrent_receipt_processing", func(t *testing.T) {
		numGoroutines := 10
		iterationsPerGoroutine := 100
		totalIterations := numGoroutines * iterationsPerGoroutine

		// Create test receipt core
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   1640995200,
			FinishedAt:  1640995201,
			IssuerID:    "concurrent-test-issuer",
		}

		var wg sync.WaitGroup
		results := make(chan error, numGoroutines)
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < iterationsPerGoroutine; j++ {
					// Canonicalize
					coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
					if err != nil {
						results <- fmt.Errorf("goroutine %d, iteration %d: canonicalize failed: %v", goroutineID, j, err)
						return
					}

					// Sign
					messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
					signature, pubKey, err := suite.signer.Sign(ctx, "performance-test-key", messageToSign)
					if err != nil {
						results <- fmt.Errorf("goroutine %d, iteration %d: sign failed: %v", goroutineID, j, err)
						return
					}

					// Create full receipt
					receiptFull := receipt.ReceiptFull{
						Core:        receiptCore,
						Signature:   signature,
						HostCycles:  uint64(12345 + j),
						HostInfo:    map[string]string{"host": "concurrent-test-host", "goroutine": fmt.Sprintf("%d", goroutineID)},
					}

					// Canonicalize full receipt
					fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
					if err != nil {
						results <- fmt.Errorf("goroutine %d, iteration %d: canonicalize full failed: %v", goroutineID, j, err)
						return
					}

					// Verify
					_, err = suite.verifier.VerifyReceipt(fullReceiptBytes, pubKey)
					if err != nil {
						results <- fmt.Errorf("goroutine %d, iteration %d: verify failed: %v", goroutineID, j, err)
						return
					}
				}
				results <- nil
			}(i)
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)

		// Check for errors
		for err := range results {
			if err != nil {
				t.Errorf("Concurrent receipt processing error: %v", err)
			}
		}

		avgDuration := duration / time.Duration(totalIterations)
		t.Logf("Processed %d concurrent receipts in %v (avg: %v per receipt)", totalIterations, duration, avgDuration)

		// Concurrent receipt processing should be fast
		if avgDuration > 5*time.Millisecond {
			t.Errorf("Concurrent receipt processing too slow: %v per receipt (expected < 5ms)", avgDuration)
		}
	})
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	suite := NewPerformanceTestSuite(t)
	ctx := context.Background()

	// Create test artifact
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

	artifactPath := filepath.Join(suite.tempDir, "memory-test-artifact")
	err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)

	t.Run("memory_usage_during_execution", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Execute many artifacts
		iterations := 1000
		for i := 0; i < iterations; i++ {
			input := []byte(fmt.Sprintf("test input %d", i))
			result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
			if err != nil {
				t.Fatalf("Failed to execute artifact: %v", err)
			}

			// Verify result is reasonable
			if result.ExitCode != 0 {
				t.Errorf("Expected exit code 0, got %d", result.ExitCode)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		avgMemoryPerExecution := memoryUsed / uint64(iterations)

		t.Logf("Memory usage: %d bytes total, %d bytes per execution", memoryUsed, avgMemoryPerExecution)

		// Memory usage should be reasonable (< 1KB per execution)
		if avgMemoryPerExecution > 1024 {
			t.Errorf("Memory usage too high: %d bytes per execution (expected < 1KB)", avgMemoryPerExecution)
		}
	})

	t.Run("memory_usage_during_receipt_processing", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Process many receipts
		iterations := 1000
		receiptCore := receipt.ReceiptCore{
			ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			GasUsed:     1000,
			StartedAt:   1640995200,
			FinishedAt:  1640995201,
			IssuerID:    "memory-test-issuer",
		}

		for i := 0; i < iterations; i++ {
			// Canonicalize
			coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
			if err != nil {
				t.Fatalf("Failed to canonicalize receipt core: %v", err)
			}

			// Sign
			messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
			signature, pubKey, err := suite.signer.Sign(ctx, "performance-test-key", messageToSign)
			if err != nil {
				t.Fatalf("Failed to sign receipt: %v", err)
			}

			// Create full receipt
			receiptFull := receipt.ReceiptFull{
				Core:        receiptCore,
				Signature:   signature,
				HostCycles:  uint64(12345 + i),
				HostInfo:    map[string]string{"host": "memory-test-host", "iteration": fmt.Sprintf("%d", i)},
			}

			// Canonicalize full receipt
			fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
			if err != nil {
				t.Fatalf("Failed to canonicalize full receipt: %v", err)
			}

			// Verify
			_, err = suite.verifier.VerifyReceipt(fullReceiptBytes, pubKey)
			if err != nil {
				t.Fatalf("Failed to verify receipt: %v", err)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		avgMemoryPerReceipt := memoryUsed / uint64(iterations)

		t.Logf("Memory usage: %d bytes total, %d bytes per receipt", memoryUsed, avgMemoryPerReceipt)

		// Memory usage should be reasonable (< 2KB per receipt)
		if avgMemoryPerReceipt > 2048 {
			t.Errorf("Memory usage too high: %d bytes per receipt (expected < 2KB)", avgMemoryPerReceipt)
		}
	})
}

// TestScalability tests how the system scales with different loads
func TestScalability(t *testing.T) {
	suite := NewPerformanceTestSuite(t)
	ctx := context.Background()

	// Create test artifact
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

	artifactPath := filepath.Join(suite.tempDir, "scalability-test-artifact")
	err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)

	// Test different load levels
	loadLevels := []struct {
		name        string
		goroutines  int
		iterations  int
		maxDuration time.Duration
	}{
		{"light_load", 1, 100, 10 * time.Second},
		{"medium_load", 5, 100, 15 * time.Second},
		{"heavy_load", 10, 100, 20 * time.Second},
		{"extreme_load", 20, 100, 30 * time.Second},
	}

	for _, level := range loadLevels {
		t.Run(level.name, func(t *testing.T) {
			var wg sync.WaitGroup
			results := make(chan error, level.goroutines)
			start := time.Now()

			for i := 0; i < level.goroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()

					for j := 0; j < level.iterations; j++ {
						input := []byte(fmt.Sprintf("test input from goroutine %d, iteration %d", goroutineID, j))
						result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
						if err != nil {
							results <- fmt.Errorf("goroutine %d, iteration %d: %v", goroutineID, j, err)
							return
						}

						if result.ExitCode != 0 {
							results <- fmt.Errorf("goroutine %d, iteration %d: expected exit code 0, got %d", goroutineID, j, result.ExitCode)
							return
						}
					}
					results <- nil
				}(i)
			}

			wg.Wait()
			close(results)
			duration := time.Since(start)

			// Check for errors
			for err := range results {
				if err != nil {
					t.Errorf("Scalability test error: %v", err)
				}
			}

			totalOperations := level.goroutines * level.iterations
			avgDuration := duration / time.Duration(totalOperations)
			throughput := float64(totalOperations) / duration.Seconds()

			t.Logf("Load level: %s", level.name)
			t.Logf("Goroutines: %d, Iterations per goroutine: %d", level.goroutines, level.iterations)
			t.Logf("Total operations: %d", totalOperations)
			t.Logf("Total duration: %v", duration)
			t.Logf("Average duration per operation: %v", avgDuration)
			t.Logf("Throughput: %.2f operations/second", throughput)

			// Verify we completed within the expected time
			if duration > level.maxDuration {
				t.Errorf("Load level %s took too long: %v (expected < %v)", level.name, duration, level.maxDuration)
			}

			// Verify throughput is reasonable (> 10 ops/sec)
			if throughput < 10 {
				t.Errorf("Load level %s throughput too low: %.2f ops/sec (expected > 10)", level.name, throughput)
			}
		})
	}
}

// BenchmarkReceiptSigning benchmarks receipt signing performance
func BenchmarkReceiptSigning(b *testing.B) {
	suite := NewPerformanceTestSuite(&testing.T{})
	ctx := context.Background()

	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "benchmark-test-issuer",
	}

	coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
	if err != nil {
		b.Fatalf("Failed to canonicalize receipt core: %v", err)
	}

	messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := suite.signer.Sign(ctx, "performance-test-key", messageToSign)
		if err != nil {
			b.Fatalf("Failed to sign receipt: %v", err)
		}
	}
}

// BenchmarkReceiptVerification benchmarks receipt verification performance
func BenchmarkReceiptVerification(b *testing.B) {
	suite := NewPerformanceTestSuite(&testing.T{})
	ctx := context.Background()

	receiptCore := receipt.ReceiptCore{
		ProgramHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		InputHash:   [32]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
		OutputHash:  [32]byte{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
		GasUsed:     1000,
		StartedAt:   1640995200,
		FinishedAt:  1640995201,
		IssuerID:    "benchmark-test-issuer",
	}

	coreBytes, err := receipt.CanonicalizeCore(&receiptCore)
	if err != nil {
		b.Fatalf("Failed to canonicalize receipt core: %v", err)
	}

	messageToSign := append([]byte(keystore.DomainSeparator), coreBytes...)
	signature, pubKey, err := suite.signer.Sign(ctx, "performance-test-key", messageToSign)
	if err != nil {
		b.Fatalf("Failed to sign receipt: %v", err)
	}

	receiptFull := receipt.ReceiptFull{
		Core:        receiptCore,
		Signature:   signature,
		HostCycles:  12345,
		HostInfo:    map[string]string{"host": "benchmark-test-host"},
	}

	fullReceiptBytes, err := receipt.CanonicalizeFull(&receiptFull)
	if err != nil {
		b.Fatalf("Failed to canonicalize full receipt: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.verifier.VerifyReceipt(fullReceiptBytes, pubKey)
		if err != nil {
			b.Fatalf("Failed to verify receipt: %v", err)
		}
	}
}

// BenchmarkVMExecution benchmarks VM execution performance
func BenchmarkVMExecution(b *testing.B) {
	suite := NewPerformanceTestSuite(&testing.T{})
	ctx := context.Background()

	// Create test artifact
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

	artifactPath := filepath.Join(suite.tempDir, "benchmark-test-artifact")
	err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		b.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		b.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)

	input := []byte("benchmark test input")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
		if err != nil {
			b.Fatalf("Failed to execute artifact: %v", err)
		}

		if result.ExitCode != 0 {
			b.Fatalf("Expected exit code 0, got %d", result.ExitCode)
		}
	}
}
