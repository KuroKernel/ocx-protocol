// Package conformance provides test vectors and conformance testing for OCX Protocol v1.0.0-rc.1
package conformance

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestVector represents a conformance test vector
type TestVector struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	ArtifactHex         string `json:"artifact_hex"`
	InputHex            string `json:"input_hex"`
	MaxCycles           uint64 `json:"max_cycles"`
	ExpectedReceiptHash string `json:"expected_receipt_hash"`
	ExpectedCyclesUsed  uint64 `json:"expected_cycles_used"`
	ExpectedOutputHash  string `json:"expected_output_hash"`
}

// ExecutionResult represents the result of CLI execution
type ExecutionResult struct {
	Status      string `json:"status"`
	Result      string `json:"result"`
	CyclesUsed  uint64 `json:"cycles_used"`
	ReceiptHash string `json:"receipt_hash"`
	OutputHash  string `json:"output_hash"`
}

// TestConfig holds configuration for conformance tests
type TestConfig struct {
	CLIPath     string
	ServerURL   string
	GoldenDir   string
	Timeout     time.Duration
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		CLIPath:   "./minimal-cli",
		ServerURL: "http://localhost:9000",
		GoldenDir: "./conformance/golden",
		Timeout:   30 * time.Second,
	}
}

// LoadTestVectors loads all test vectors from the golden directory
func LoadTestVectors(goldenDir string) ([]TestVector, error) {
	var vectors []TestVector
	
	// Find all JSON files in the golden directory
	pattern := filepath.Join(goldenDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob test vector files: %w", err)
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no test vector files found in %s", goldenDir)
	}
	
	// Load each test vector file
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("failed to read test vector file %s: %w", match, err)
		}
		
		var vector TestVector
		if err := json.Unmarshal(data, &vector); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test vector file %s: %w", match, err)
		}
		
		vectors = append(vectors, vector)
	}
	
	return vectors, nil
}

// ExecuteCLI executes the minimal-cli with the given parameters
func ExecuteCLI(config *TestConfig, vector TestVector) (*ExecutionResult, error) {
	// Convert hex strings to actual data
	artifact, err := hex.DecodeString(vector.ArtifactHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode artifact hex: %w", err)
	}
	
	input, err := hex.DecodeString(vector.InputHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode input hex: %w", err)
	}
	
	// Build command arguments
	args := []string{
		"-command", "execute",
		"-server", config.ServerURL,
		"-artifact", string(artifact),
		"-input", string(input),
		"-max-cycles", fmt.Sprintf("%d", vector.MaxCycles),
		"-lease-id", fmt.Sprintf("test-%s", vector.Name),
	}
	
	// Execute CLI command
	cmd := exec.Command(config.CLIPath, args...)
	cmd.Timeout = config.Timeout
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute CLI: %w", err)
	}
	
	// Parse JSON output
	var result ExecutionResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CLI output: %w", err)
	}
	
	return &result, nil
}

// ValidateResult validates the execution result against expected values
func ValidateResult(result *ExecutionResult, vector TestVector) error {
	var errors []string
	
	// Validate status
	if result.Status != "success" {
		errors = append(errors, fmt.Sprintf("expected status 'success', got '%s'", result.Status))
	}
	
	// Validate cycles used
	if result.CyclesUsed != vector.ExpectedCyclesUsed {
		errors = append(errors, fmt.Sprintf("expected cycles used %d, got %d", vector.ExpectedCyclesUsed, result.CyclesUsed))
	}
	
	// Validate receipt hash
	if result.ReceiptHash != vector.ExpectedReceiptHash {
		errors = append(errors, fmt.Sprintf("expected receipt hash '%s', got '%s'", vector.ExpectedReceiptHash, result.ReceiptHash))
	}
	
	// Validate output hash
	if result.OutputHash != vector.ExpectedOutputHash {
		errors = append(errors, fmt.Sprintf("expected output hash '%s', got '%s'", vector.ExpectedOutputHash, result.OutputHash))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// TestConformance runs all conformance tests
func TestConformance(t *testing.T) {
	config := DefaultTestConfig()
	
	// Load test vectors
	vectors, err := LoadTestVectors(config.GoldenDir)
	if err != nil {
		t.Fatalf("Failed to load test vectors: %v", err)
	}
	
	if len(vectors) == 0 {
		t.Fatal("No test vectors found")
	}
	
	t.Logf("Loaded %d test vectors", len(vectors))
	
	// Check if CLI exists
	if _, err := os.Stat(config.CLIPath); os.IsNotExist(err) {
		t.Fatalf("CLI not found at %s", config.CLIPath)
	}
	
	// Run each test vector
	var passed, failed int
	
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			t.Logf("Running test: %s - %s", vector.Name, vector.Description)
			
			// Execute CLI
			result, err := ExecuteCLI(config, vector)
			if err != nil {
				t.Errorf("Failed to execute CLI: %v", err)
				failed++
				return
			}
			
			// Validate result
			if err := ValidateResult(result, vector); err != nil {
				t.Errorf("Validation failed: %v", err)
				failed++
				return
			}
			
			t.Logf("Test passed: %s", vector.Name)
			passed++
		})
	}
	
	// Print summary
	t.Logf("Conformance test summary: %d passed, %d failed", passed, failed)
	
	if failed > 0 {
		t.Fatalf("Conformance tests failed: %d out of %d tests failed", failed, len(vectors))
	}
}

// TestConformanceWithServer tests conformance with a running server
func TestConformanceWithServer(t *testing.T) {
	config := DefaultTestConfig()
	
	// Check if server is running
	cmd := exec.Command("curl", "-s", config.ServerURL+"/health")
	if err := cmd.Run(); err != nil {
		t.Skipf("Server not running at %s, skipping server tests", config.ServerURL)
	}
	
	// Run conformance tests
	TestConformance(t)
}

// BenchmarkConformance benchmarks conformance test execution
func BenchmarkConformance(b *testing.B) {
	config := DefaultTestConfig()
	
	// Load test vectors
	vectors, err := LoadTestVectors(config.GoldenDir)
	if err != nil {
		b.Fatalf("Failed to load test vectors: %v", err)
	}
	
	if len(vectors) == 0 {
		b.Fatal("No test vectors found")
	}
	
	// Use first test vector for benchmarking
	vector := vectors[0]
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := ExecuteCLI(config, vector)
		if err != nil {
			b.Fatalf("Failed to execute CLI: %v", err)
		}
	}
}

// TestDeterminism tests that the same input produces the same output
func TestDeterminism(t *testing.T) {
	config := DefaultTestConfig()
	
	// Load test vectors
	vectors, err := LoadTestVectors(config.GoldenDir)
	if err != nil {
		t.Fatalf("Failed to load test vectors: %v", err)
	}
	
	if len(vectors) == 0 {
		t.Fatal("No test vectors found")
	}
	
	// Test each vector multiple times
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			var results []*ExecutionResult
			
			// Execute multiple times
			for i := 0; i < 3; i++ {
				result, err := ExecuteCLI(config, vector)
				if err != nil {
					t.Errorf("Failed to execute CLI (attempt %d): %v", i+1, err)
					return
				}
				results = append(results, result)
			}
			
			// Compare results
			first := results[0]
			for i, result := range results[1:] {
				if result.ReceiptHash != first.ReceiptHash {
					t.Errorf("Receipt hash mismatch (attempt %d): expected %s, got %s", i+2, first.ReceiptHash, result.ReceiptHash)
				}
				if result.OutputHash != first.OutputHash {
					t.Errorf("Output hash mismatch (attempt %d): expected %s, got %s", i+2, first.OutputHash, result.OutputHash)
				}
				if result.CyclesUsed != first.CyclesUsed {
					t.Errorf("Cycles used mismatch (attempt %d): expected %d, got %d", i+2, first.CyclesUsed, result.CyclesUsed)
				}
			}
		})
	}
}
