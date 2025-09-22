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

func TestExecuteArtifact(t *testing.T) {
	// Create a simple test artifact (shell script)
	testScript := `#!/bin/bash
echo "Hello, deterministic world!"
echo "Input received: $(cat input.bin)"
exit 0`
	
	artifact, cleanup := createTestArtifact(t, testScript)
	defer cleanup()
	
	// Calculate artifact hash
	artifactHash := calculateFileHash(t, artifact)
	
	// Test input
	input := []byte("test input data")
	
	// Execute the artifact
	result, err := ExecuteArtifact(context.Background(), artifactHash, input)
	if err != nil {
		t.Fatalf("ExecuteArtifact failed: %v", err)
	}
	
	// Validate results
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	expectedOutput := "Hello, deterministic world!\nInput received: test input data\n"
	if string(result.Stdout) != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, string(result.Stdout))
	}
	
	if result.CyclesUsed == 0 {
		t.Error("Expected non-zero cycles used")
	}
	
	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestDeterministicExecution(t *testing.T) {
	// Create a test artifact
	testScript := `#!/bin/bash
# Output some potentially non-deterministic things that we should handle
echo "Hello from script"
echo "Environment: $LC_ALL $TZ"
echo "Input: $(cat input.bin)"
# Output the input hash to verify we got the same input
cat input.bin | sha256sum
exit 0`
	
	artifact, cleanup := createTestArtifact(t, testScript)
	defer cleanup()
	
	artifactHash := calculateFileHash(t, artifact)
	input := []byte("deterministic test input")
	
	// Execute the same artifact multiple times
	const numRuns = 3
	results := make([]*ExecutionResult, numRuns)
	
	for i := 0; i < numRuns; i++ {
		result, err := ExecuteArtifact(context.Background(), artifactHash, input)
		if err != nil {
			t.Fatalf("Run %d failed: %v", i+1, err)
		}
		results[i] = result
	}
	
	// Verify all results are identical (deterministic)
	for i := 1; i < numRuns; i++ {
		if results[i].ExitCode != results[0].ExitCode {
			t.Errorf("Run %d exit code differs: %d vs %d", i+1, results[i].ExitCode, results[0].ExitCode)
		}
		
		if string(results[i].Stdout) != string(results[0].Stdout) {
			t.Errorf("Run %d stdout differs:\nExpected: %q\nGot: %q", 
				i+1, string(results[0].Stdout), string(results[i].Stdout))
		}
		
		if string(results[i].Stderr) != string(results[0].Stderr) {
			t.Errorf("Run %d stderr differs: %q vs %q", 
				i+1, string(results[0].Stderr), string(results[i].Stderr))
		}
	}
}

func TestExecutionLimits(t *testing.T) {
	// Test timeout limit
	t.Run("TimeoutLimit", func(t *testing.T) {
		longScript := `#!/bin/bash
sleep 60  # Sleep longer than timeout
echo "Should not reach here"`
		
		artifact, cleanup := createTestArtifact(t, longScript)
		defer cleanup()
		
		artifactHash := calculateFileHash(t, artifact)
		
		// Create config with short timeout
		config := DefaultVMConfig()
		config.Timeout = 1 * time.Second
		
		_, err := ExecuteArtifactWithConfig(context.Background(), artifactHash, []byte("test"), &config)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
		
		if execErr, ok := err.(*ExecutionError); ok {
			if execErr.Code != ErrorCodeTimeout {
				t.Errorf("Expected timeout error, got %v", execErr.Code)
			}
		}
	})
	
	// Test cycle limit
	t.Run("CycleLimit", func(t *testing.T) {
		// Create a script that should exceed cycle limits
		intensiveScript := `#!/bin/bash
# Simulate intensive computation
for i in {1..1000}; do
  echo "Computing $i" > /dev/null
done
echo "Done"`
		
		artifact, cleanup := createTestArtifact(t, intensiveScript)
		defer cleanup()
		
		artifactHash := calculateFileHash(t, artifact)
		
		// Create config with very low cycle limit
		config := DefaultVMConfig()
		config.CycleLimit = 1000 // Very low limit
		
		_, err := ExecuteArtifactWithConfig(context.Background(), artifactHash, []byte("test"), &config)
		if err == nil {
			t.Error("Expected cycle limit error, got nil")
		}
		
		if execErr, ok := err.(*ExecutionError); ok {
			if execErr.Code != ErrorCodeCycleLimitExceeded {
				t.Errorf("Expected cycle limit error, got %v", execErr.Code)
			}
		}
	})
}

func TestArtifactNotFound(t *testing.T) {
	// Test with non-existent artifact hash
	fakeHash := sha256.Sum256([]byte("non-existent-artifact"))
	
	_, err := ExecuteArtifact(context.Background(), fakeHash, []byte("test"))
	if err == nil {
		t.Error("Expected artifact not found error, got nil")
	}
	
	if execErr, ok := err.(*ExecutionError); ok {
		if execErr.Code != ErrorCodeArtifactNotFound {
			t.Errorf("Expected artifact not found error, got %v", execErr.Code)
		}
	}
}

func TestEnvironmentSetup(t *testing.T) {
	testInput := []byte("environment test data")
	
	// Create a simple artifact for environment testing
	artifact, cleanup := createTestArtifact(t, `#!/bin/bash
echo "LC_ALL=$LC_ALL"
echo "TZ=$TZ"
echo "HOME=$HOME"
echo "USER=$USER"
echo "PWD=$PWD"
ls -la
cat input.bin`)
	defer cleanup()
	
	execDir, cleanupEnv, err := prepareExecutionEnvironment(artifact, testInput)
	if err != nil {
		t.Fatalf("prepareExecutionEnvironment failed: %v", err)
	}
	defer cleanupEnv()
	
	// Verify directory structure
	expectedPaths := []string{
		"input.bin",
		"artifact",
		"tmp",
		"dev/null",
		"dev/zero",
		"etc/passwd",
	}
	
	for _, path := range expectedPaths {
		fullPath := filepath.Join(execDir, path)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("Expected path %s not found: %v", path, err)
		}
	}
	
	// Verify input file content
	inputPath := filepath.Join(execDir, "input.bin")
	actualInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}
	
	if string(actualInput) != string(testInput) {
		t.Errorf("Input file content mismatch: expected %q, got %q", 
			string(testInput), string(actualInput))
	}
	
	// Verify artifact is executable
	artifactPath := filepath.Join(execDir, "artifact")
	info, err := os.Stat(artifactPath)
	if err != nil {
		t.Fatalf("Failed to stat artifact: %v", err)
	}
	
	if !isExecutable(info.Mode()) {
		t.Error("Artifact is not executable")
	}
}

func TestGetArtifactInfo(t *testing.T) {
	testScript := `#!/bin/bash
echo "test artifact"`
	
	artifact, cleanup := createTestArtifact(t, testScript)
	defer cleanup()
	
	artifactHash := calculateFileHash(t, artifact)
	
	info, err := GetArtifactInfo(artifactHash)
	if err != nil {
		t.Fatalf("GetArtifactInfo failed: %v", err)
	}
	
	if info.Hash != artifactHash {
		t.Errorf("Hash mismatch: expected %x, got %x", artifactHash, info.Hash)
	}
	
	if !info.Executable {
		t.Error("Expected artifact to be marked as executable")
	}
	
	if info.Size == 0 {
		t.Error("Expected non-zero artifact size")
	}
}

// Helper functions for testing

func createTestArtifact(t *testing.T, script string) (string, func()) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-artifact-*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// Write script content
	if _, err := tmpFile.WriteString(script); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}
	tmpFile.Close()
	
	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}
	
	// Set up artifact cache directory for testing
	cacheDir := filepath.Dir(tmpFile.Name())
	hash := calculateFileHash(t, tmpFile.Name())
	hashStr := fmt.Sprintf("%x", hash)
	cachePath := filepath.Join(cacheDir, hashStr)
	
	// Copy to cache location
	if err := copyFile(tmpFile.Name(), cachePath); err != nil {
		t.Fatalf("Failed to copy to cache: %v", err)
	}
	
	cleanup := func() {
		os.Remove(tmpFile.Name())
		os.Remove(cachePath)
	}
	
	return tmpFile.Name(), cleanup
}

func calculateFileHash(t *testing.T, path string) [32]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file for hash: %v", err)
	}
	return sha256.Sum256(data)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	
	return os.WriteFile(dst, data, 0755)
}

// Benchmark tests

func BenchmarkExecuteArtifact(b *testing.B) {
	testScript := `#!/bin/bash
echo "benchmark test"
echo "input: $(cat input.bin)"`
	
	artifact, cleanup := createTestArtifactBench(b, testScript)
	defer cleanup()
	
	artifactHash := calculateFileHashBench(b, artifact)
	input := []byte("benchmark input")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := ExecuteArtifact(context.Background(), artifactHash, input)
		if err != nil {
			b.Fatalf("ExecuteArtifact failed: %v", err)
		}
	}
}

func BenchmarkEnvironmentSetup(b *testing.B) {
	testScript := `#!/bin/bash
echo "test"`
	
	artifact, cleanup := createTestArtifactBench(b, testScript)
	defer cleanup()
	
	input := []byte("test input")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		execDir, cleanupEnv, err := prepareExecutionEnvironment(artifact, input)
		if err != nil {
			b.Fatalf("prepareExecutionEnvironment failed: %v", err)
		}
		cleanupEnv()
		_ = execDir
	}
}

// createTestArtifact for benchmarks (similar to test version but with different signature)
func createTestArtifactBench(b *testing.B, script string) (string, func()) {
	// Similar implementation to the test version
	tmpFile, err := os.CreateTemp("", "bench-artifact-*.sh")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	
	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatalf("Failed to write script: %v", err)
	}
	tmpFile.Close()
	
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		b.Fatalf("Failed to make script executable: %v", err)
	}
	
	cacheDir := filepath.Dir(tmpFile.Name())
	hash := calculateFileHashBench(b, tmpFile.Name())
	hashStr := fmt.Sprintf("%x", hash)
	cachePath := filepath.Join(cacheDir, hashStr)
	
	if err := copyFile(tmpFile.Name(), cachePath); err != nil {
		b.Fatalf("Failed to copy to cache: %v", err)
	}
	
	cleanup := func() {
		os.Remove(tmpFile.Name())
		os.Remove(cachePath)
	}
	
	return tmpFile.Name(), cleanup
}

func calculateFileHashBench(b *testing.B, path string) [32]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("Failed to read file for hash: %v", err)
	}
	return sha256.Sum256(data)
}
