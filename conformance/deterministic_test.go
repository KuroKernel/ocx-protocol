// +build ignore

package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
)

// DeterministicTestSuite tests the deterministic execution engine
type DeterministicTestSuite struct {
	keystore *keystore.Keystore
}

// NewDeterministicTestSuite creates a new deterministic test suite
func NewDeterministicTestSuite() (*DeterministicTestSuite, error) {
	// Initialize keystore
	ks, err := keystore.New("./conformance-keys")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keystore: %w", err)
	}

	return &DeterministicTestSuite{
		keystore: ks,
	}, nil
}

// DeterministicTestResult represents the result of a deterministic test
type DeterministicTestResult struct {
	TestName      string        `json:"test_name"`
	Runs          int           `json:"runs"`
	GasUsed       []uint64      `json:"gas_used"`
	OutputHashes  []string      `json:"output_hashes"`
	ReceiptHashes []string      `json:"receipt_hashes"`
	ExecutionTimes []time.Duration `json:"execution_times"`
	Deterministic bool          `json:"deterministic"`
	Error         string        `json:"error,omitempty"`
}

// TestDeterministicExecution tests that the same input produces identical outputs
func (dts *DeterministicTestSuite) TestDeterministicExecution(ctx context.Context, testName, artifactHex, inputHex string, maxGas uint64, runs int) (*DeterministicTestResult, error) {
	// Decode inputs
	artifact, err := hex.DecodeString(artifactHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode artifact hex: %w", err)
	}

	input, err := hex.DecodeString(inputHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode input hex: %w", err)
	}

	// Calculate hashes
	artifactHash := sha256.Sum256(artifact)
	inputHash := sha256.Sum256(input)

	result := &DeterministicTestResult{
		TestName:      testName,
		Runs:          runs,
		GasUsed:       make([]uint64, runs),
		OutputHashes:  make([]string, runs),
		ReceiptHashes: make([]string, runs),
		ExecutionTimes: make([]time.Duration, runs),
	}

	// Execute multiple times
	for i := 0; i < runs; i++ {
		startTime := time.Now()
		
		// Execute artifact
		execResult, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
		if err != nil {
			return nil, fmt.Errorf("failed to execute artifact (run %d): %w", i+1, err)
		}
		
		executionTime := time.Since(startTime)
		result.ExecutionTimes[i] = executionTime

		// Calculate output hash
		outputHash := sha256.Sum256(execResult.Stdout)
		result.OutputHashes[i] = hex.EncodeToString(outputHash[:])

		// Create receipt
		receiptCore := &receipt.ReceiptCore{
			ProgramHash: artifactHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     execResult.GasUsed,
			StartedAt:   uint64(startTime.Unix()),
			FinishedAt:  uint64(execResult.EndTime.Unix()),
			IssuerID:    "deterministic-test",
		}

		// Canonicalize receipt
		coreBytes, err := receipt.CanonicalizeCore(receiptCore)
		if err != nil {
			return nil, fmt.Errorf("failed to canonicalize receipt (run %d): %w", i+1, err)
		}

		// Calculate receipt hash
		receiptHash := sha256.Sum256(coreBytes)
		result.ReceiptHashes[i] = hex.EncodeToString(receiptHash[:])

		// Store gas used
		result.GasUsed[i] = execResult.GasUsed
	}

	// Check determinism
	result.Deterministic = dts.checkDeterminism(result)

	return result, nil
}

// checkDeterminism checks if all runs produced identical results
func (dts *DeterministicTestSuite) checkDeterminism(result *DeterministicTestResult) bool {
	if len(result.GasUsed) < 2 {
		return true
	}

	// Check gas used
	firstGas := result.GasUsed[0]
	for _, gas := range result.GasUsed[1:] {
		if gas != firstGas {
			return false
		}
	}

	// Check output hashes
	firstOutputHash := result.OutputHashes[0]
	for _, hash := range result.OutputHashes[1:] {
		if hash != firstOutputHash {
			return false
		}
	}

	// Check receipt hashes
	firstReceiptHash := result.ReceiptHashes[0]
	for _, hash := range result.ReceiptHashes[1:] {
		if hash != firstReceiptHash {
			return false
		}
	}

	return true
}

// TestDeterministicExecution tests deterministic execution
func TestDeterministicExecution(t *testing.T) {
	// Create test suite
	suite, err := NewDeterministicTestSuite()
	if err != nil {
		t.Fatalf("Failed to create deterministic test suite: %v", err)
	}

	ctx := context.Background()

	// Test cases
	testCases := []struct {
		name        string
		artifactHex string
		inputHex    string
		maxGas      uint64
		runs        int
	}{
		{
			name:        "minimal_execution",
			artifactHex: "48656c6c6f20576f726c64", // "Hello World"
			inputHex:    "74657374",                // "test"
			maxGas:      1000,
			runs:        5,
		},
		{
			name:        "complex_execution",
			artifactHex: "48656c6c6f20576f726c64212054686973206973206120636f6d706c6578206172746966616374", // "Hello World! This is a complex artifact"
			inputHex:    "7465737420696e70757420666f7220636f6d706c657820657865637574696f6e",                // "test input for complex execution"
			maxGas:      5000,
			runs:        3,
		},
		{
			name:        "high_gas_execution",
			artifactHex: "48656c6c6f20576f726c6421204869676820676173207573616765", // "Hello World! High gas usage"
			inputHex:    "68696768206761732074657374",                              // "high gas test"
			maxGas:      10000,
			runs:        3,
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := suite.TestDeterministicExecution(ctx, tc.name, tc.artifactHex, tc.inputHex, tc.maxGas, tc.runs)
			if err != nil {
				t.Fatalf("Failed to run deterministic test: %v", err)
			}

			// Validate result
			if !result.Deterministic {
				t.Errorf("Test %s failed determinism check:", tc.name)
				t.Errorf("  Gas used: %v", result.GasUsed)
				t.Errorf("  Output hashes: %v", result.OutputHashes)
				t.Errorf("  Receipt hashes: %v", result.ReceiptHashes)
			}

			// Log execution times
			var totalTime time.Duration
			for _, execTime := range result.ExecutionTimes {
				totalTime += execTime
			}
			avgTime := totalTime / time.Duration(len(result.ExecutionTimes))

			t.Logf("Test %s: %d runs, deterministic=%t, avg_time=%v", tc.name, tc.runs, result.Deterministic, avgTime)
		})
	}
}

// TestDeterministicExecutionStress tests deterministic execution under stress
func TestDeterministicExecutionStress(t *testing.T) {
	// Create test suite
	suite, err := NewDeterministicTestSuite()
	if err != nil {
		t.Fatalf("Failed to create deterministic test suite: %v", err)
	}

	ctx := context.Background()

	// Stress test with many runs
	artifactHex := "48656c6c6f20576f726c64" // "Hello World"
	inputHex := "74657374"                // "test"
	maxGas := uint64(1000)
	runs := 100 // High number of runs

	t.Logf("Running stress test with %d executions...", runs)

	result, err := suite.TestDeterministicExecution(ctx, "stress_test", artifactHex, inputHex, maxGas, runs)
	if err != nil {
		t.Fatalf("Failed to run stress test: %v", err)
	}

	// Validate determinism
	if !result.Deterministic {
		t.Errorf("Stress test failed determinism check:")
		t.Errorf("  Gas used: %v", result.GasUsed)
		t.Errorf("  Output hashes: %v", result.OutputHashes)
		t.Errorf("  Receipt hashes: %v", result.ReceiptHashes)
	}

	// Calculate statistics
	var totalTime time.Duration
	var minTime, maxTime time.Duration
	for i, execTime := range result.ExecutionTimes {
		totalTime += execTime
		if i == 0 {
			minTime = execTime
			maxTime = execTime
		} else {
			if execTime < minTime {
				minTime = execTime
			}
			if execTime > maxTime {
				maxTime = execTime
			}
		}
	}
	avgTime := totalTime / time.Duration(len(result.ExecutionTimes))

	t.Logf("Stress test results:")
	t.Logf("  Runs: %d", runs)
	t.Logf("  Deterministic: %t", result.Deterministic)
	t.Logf("  Avg time: %v", avgTime)
	t.Logf("  Min time: %v", minTime)
	t.Logf("  Max time: %v", maxTime)
	t.Logf("  Gas used: %d", result.GasUsed[0])
}

// BenchmarkDeterministicExecution benchmarks deterministic execution
func BenchmarkDeterministicExecution(b *testing.B) {
	// Create test suite
	_, err := NewDeterministicTestSuite()
	if err != nil {
		b.Fatalf("Failed to create deterministic test suite: %v", err)
	}

	ctx := context.Background()

	// Benchmark parameters
	artifactHex := "48656c6c6f20576f726c64" // "Hello World"
	inputHex := "74657374"                // "test"
	_ = uint64(1000) // maxGas placeholder

	// Decode inputs
	artifact, err := hex.DecodeString(artifactHex)
	if err != nil {
		b.Fatalf("Failed to decode artifact hex: %v", err)
	}

	input, err := hex.DecodeString(inputHex)
	if err != nil {
		b.Fatalf("Failed to decode input hex: %v", err)
	}

	// Calculate hashes
	artifactHash := sha256.Sum256(artifact)
	_ = sha256.Sum256(input) // inputHash placeholder

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Execute artifact
		_, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
		if err != nil {
			b.Fatalf("Failed to execute artifact: %v", err)
		}
	}
}

// TestDeterministicExecutionWithGoldenVectors tests deterministic execution with golden vectors
func TestDeterministicExecutionWithGoldenVectors(t *testing.T) {
	// Create test suite
	suite, err := NewDeterministicTestSuite()
	if err != nil {
		t.Fatalf("Failed to create deterministic test suite: %v", err)
	}

	ctx := context.Background()

	// Check if golden vectors exist
	vectorsDir := "./conformance/generated"
	if _, err := os.Stat(vectorsDir); os.IsNotExist(err) {
		t.Skipf("Golden vectors directory not found at %s, skipping golden vector tests", vectorsDir)
	}

	// Find all vector directories
	entries, err := os.ReadDir(vectorsDir)
	if err != nil {
		t.Fatalf("Failed to read vectors directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No golden vectors found")
	}

	// Test each golden vector
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		vectorPath := filepath.Join(vectorsDir, entry.Name())
		jsonPath := filepath.Join(vectorPath, "vector.json")

		// Check if vector.json exists
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			continue
		}

		// Load vector
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Errorf("Failed to read vector JSON %s: %v", jsonPath, err)
			continue
		}

		var vector GoldenVector
		if err := json.Unmarshal(data, &vector); err != nil {
			t.Errorf("Failed to unmarshal vector JSON %s: %v", jsonPath, err)
			continue
		}

		t.Run(vector.Name, func(t *testing.T) {
			// Test deterministic execution
			result, err := suite.TestDeterministicExecution(ctx, vector.Name, vector.ArtifactHex, vector.InputHex, vector.MaxGas, 3)
			if err != nil {
				t.Errorf("Failed to run deterministic test: %v", err)
				return
			}

			// Validate determinism
			if !result.Deterministic {
				t.Errorf("Golden vector %s failed determinism check:", vector.Name)
				t.Errorf("  Gas used: %v", result.GasUsed)
				t.Errorf("  Output hashes: %v", result.OutputHashes)
				t.Errorf("  Receipt hashes: %v", result.ReceiptHashes)
			}

			// Validate against expected values
			if result.GasUsed[0] != vector.ExpectedGasUsed {
				t.Errorf("Gas used mismatch for vector %s: expected %d, got %d", vector.Name, vector.ExpectedGasUsed, result.GasUsed[0])
			}

			if result.OutputHashes[0] != vector.ExpectedOutputHash {
				t.Errorf("Output hash mismatch for vector %s: expected %s, got %s", vector.Name, vector.ExpectedOutputHash, result.OutputHashes[0])
			}

			t.Logf("Golden vector %s: deterministic=%t, gas=%d", vector.Name, result.Deterministic, result.GasUsed[0])
		})
	}
}
