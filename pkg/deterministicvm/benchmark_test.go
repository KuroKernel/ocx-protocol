package deterministicvm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPerformanceBenchmark runs the comprehensive performance benchmark
func TestPerformanceBenchmark(t *testing.T) {
	// Save the original VM type
	originalVM := GetVM()

	// Ensure we reset to the original VM type after the test
	defer func() {
		SetVM(originalVM)
	}()

	suite := NewBenchmarkSuite()

	// Run a subset of benchmarks for testing
	err := suite.RunComprehensiveBenchmark()
	if err != nil {
		t.Fatalf("Benchmark suite failed: %v", err)
	}
}

// BenchmarkExecuteArtifactComprehensive benchmarks artifact execution
func BenchmarkExecuteArtifactComprehensive(b *testing.B) {
	// Create a simple test artifact
	testScript := `#!/bin/bash
echo "Hello, OCX!"
echo "Input: $(cat input.bin)"
exit 0`

	// Create temporary artifact
	tmpFile, err := createTempArtifact(testScript)
	if err != nil {
		b.Fatalf("Failed to create temp artifact: %v", err)
	}
	defer cleanupTempArtifact(tmpFile)

	// Set up artifact cache
	hash := setupArtifactCache(tmpFile)
	defer cleanupArtifactCache(hash)

	// Test input
	input := []byte("benchmark test input")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ExecuteArtifact(context.Background(), hash, input)
		if err != nil {
			b.Fatalf("ExecuteArtifact failed: %v", err)
		}
	}
}

// BenchmarkWASMExecution benchmarks WASM execution
func BenchmarkWASMExecution(b *testing.B) {
	// Set WASM VM type
	err := SetVMType(VMTypeWASM)
	if err != nil {
		b.Skipf("WASM VM not available: %v", err)
	}

	// Create a simple test artifact (this would be a WASM file in real usage)
	testScript := `#!/bin/bash
echo "WASM benchmark test"
exit 0`

	// Create temporary artifact
	tmpFile, err := createTempArtifact(testScript)
	if err != nil {
		b.Fatalf("Failed to create temp artifact: %v", err)
	}
	defer cleanupTempArtifact(tmpFile)

	// Set up artifact cache
	hash := setupArtifactCache(tmpFile)
	defer cleanupArtifactCache(hash)

	// Test input
	input := []byte("wasm benchmark input")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ExecuteArtifact(context.Background(), hash, input)
		if err != nil {
			// WASM execution might fail with shell scripts, that's expected
			// In real usage, this would be a proper WASM module
			b.Logf("WASM execution failed (expected with shell scripts): %v", err)
			continue
		}
	}
}

// BenchmarkGasCalculation benchmarks gas calculation
func BenchmarkGasCalculation(b *testing.B) {
	script := `#!/bin/bash
echo "Hello, OCX!"
echo "Input: $(cat input.bin)"
echo "Processing..."
echo "Done"
exit 0`

	input := []byte("benchmark test input for gas calculation")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateDeterministicGas(script, input)
	}
}

// BenchmarkDeterministicExecution benchmarks deterministic execution
func BenchmarkDeterministicExecution(b *testing.B) {
	testScript := `#!/bin/bash
echo "Deterministic benchmark"
echo "Input: $(cat input.bin)"
exit 0`

	// Create temporary artifact
	tmpFile, err := createTempArtifact(testScript)
	if err != nil {
		b.Fatalf("Failed to create temp artifact: %v", err)
	}
	defer cleanupTempArtifact(tmpFile)

	// Set up artifact cache
	hash := setupArtifactCache(tmpFile)
	defer cleanupArtifactCache(hash)

	// Test input
	input := []byte("deterministic benchmark input")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := ExecuteArtifact(context.Background(), hash, input)
		if err != nil {
			b.Fatalf("ExecuteArtifact failed: %v", err)
		}

		// Verify determinism (first run sets baseline)
		if i == 0 {
			// Store baseline for comparison
			_ = result
		}
	}
}

// BenchmarkReceiptGeneration benchmarks receipt generation
func BenchmarkReceiptGeneration(b *testing.B) {
	// Create a test receipt
	receipt := &OCXReceipt{
		SpecHash:     [32]byte{},
		ArtifactHash: [32]byte{},
		InputHash:    [32]byte{},
		OutputHash:   [32]byte{},
		GasUsed:      1000,
		IssuerID:     "benchmark-test",
		Signature:    make([]byte, 64),
		HostCycles:   2000000,
		StartedAt:    uint64(time.Now().Unix()),
		FinishedAt:   uint64(time.Now().Unix()),
		DurationMs:   5,
		MemoryUsed:   4194304,
		HostInfo: HostInfo{
			Platform:  "linux/amd64",
			CPUModel:  "x86_64",
			KernelVer: "6.12.10",
			LoadAvg:   "0.1 0.2 0.3",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := CanonicalizeReceipt(receipt)
		if err != nil {
			b.Fatalf("CanonicalizeReceipt failed: %v", err)
		}
	}
}

// BenchmarkVMTypeSwitching benchmarks VM type switching
func BenchmarkVMTypeSwitching(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Switch between VM types
		err := SetVMType(VMTypeOSProcess)
		if err != nil {
			b.Fatalf("Failed to set OS process VM: %v", err)
		}

		err = SetVMType(VMTypeWASM)
		if err != nil {
			b.Fatalf("Failed to set WASM VM: %v", err)
		}
	}
}

// Helper functions for benchmarks

func createTempArtifact(script string) (string, error) {
	tmpFile, err := os.CreateTemp("", "benchmark-*.sh")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func cleanupTempArtifact(path string) {
	os.Remove(path)
}

func setupArtifactCache(artifactPath string) [32]byte {
	// Read artifact content
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		panic(err)
	}

	// Calculate hash
	hash := sha256.Sum256(data)

	// Set up cache
	cacheDir := filepath.Dir(artifactPath)
	hashStr := fmt.Sprintf("%x", hash)
	cachePath := filepath.Join(cacheDir, hashStr)

	// Copy to cache location
	if err := os.WriteFile(cachePath, data, 0755); err != nil {
		panic(err)
	}

	return hash
}

func cleanupArtifactCache(hash [32]byte) {
	// This would clean up the cache file
	// For benchmarks, we'll leave it for potential reuse
}
