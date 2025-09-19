// conformance.go — OCX Conformance Test Vectors
// Ensures deterministic execution across all implementations

package executor

import (
	"encoding/binary"
	"testing"
)

// ConformanceTest represents a test case for deterministic execution
type ConformanceTest struct {
	Name     string    `json:"name"`
	Input    OCXInput  `json:"input"`
	Expected uint64    `json:"expected"`
}

// ConformanceVectors contains all test cases
var ConformanceVectors = []ConformanceTest{
	{
		Name: "simple_addition",
		Input: OCXInput{
			Code: []byte{
				byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
				byte(OP_LOAD), 8, 0, 0, 0, // LOAD from address 8
				byte(OP_ADD),              // ADD
				byte(OP_STORE), 16, 0, 0, 0, // STORE to address 16
				byte(OP_HALT),             // HALT
			},
			Data:      makeTestData(5, 7),
			MaxCycles: 1000,
		},
		Expected: 12,
	},
	{
		Name: "multiplication",
		Input: OCXInput{
			Code: []byte{
				byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
				byte(OP_LOAD), 8, 0, 0, 0, // LOAD from address 8
				byte(OP_MUL),              // MUL
				byte(OP_STORE), 16, 0, 0, 0, // STORE to address 16
				byte(OP_HALT),             // HALT
			},
			Data:      makeTestData(6, 7),
			MaxCycles: 1000,
		},
		Expected: 42,
	},
	{
		Name: "division",
		Input: OCXInput{
			Code: []byte{
				byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
				byte(OP_LOAD), 8, 0, 0, 0, // LOAD from address 8
				byte(OP_DIV),              // DIV
				byte(OP_STORE), 16, 0, 0, 0, // STORE to address 16
				byte(OP_HALT),             // HALT
			},
			Data:      makeTestData(84, 2),
			MaxCycles: 1000,
		},
		Expected: 42,
	},
	{
		Name: "hash_operation",
		Input: OCXInput{
			Code: []byte{
				byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
				byte(OP_HASH),             // HASH
				byte(OP_STORE), 8, 0, 0, 0, // STORE to address 8
				byte(OP_HALT),             // HALT
			},
			Data:      makeTestData(42, 0),
			MaxCycles: 1000,
		},
		Expected: 0, // Hash result (will be calculated)
	},
}

// TestConformanceVectors runs all conformance tests
func TestConformanceVectors(t *testing.T) {
	for _, test := range ConformanceVectors {
		t.Run(test.Name, func(t *testing.T) {
			result, err := OCX_EXEC(test.Input)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}
			
			if !OCX_VERIFY(result.Receipt) {
				t.Fatal("Receipt verification failed")
			}
			
			if len(result.Output) < 24 {
				t.Fatal("Output too short")
			}
			
			actual := binary.LittleEndian.Uint64(result.Output[16:24])
			if test.Expected != 0 && actual != test.Expected {
				t.Fatalf("Expected %d, got %d", test.Expected, actual)
			}
			
			// Verify deterministic pricing
			expectedPrice := calculatePrice(
				result.Receipt.CyclesUsed,
				uint64(len(test.Input.Data) + len(result.Output)),
				uint64(len(result.Output) / 4096),
			)
			if result.Receipt.Price != expectedPrice {
				t.Fatalf("Price mismatch: expected %d, got %d", expectedPrice, result.Receipt.Price)
			}
		})
	}
}

// TestDeterminism ensures identical inputs produce identical receipts
func TestDeterminism(t *testing.T) {
	input := OCXInput{
		Code: []byte{
			byte(OP_LOAD), 0, 0, 0, 0,
			byte(OP_LOAD), 8, 0, 0, 0,
			byte(OP_ADD),
			byte(OP_STORE), 16, 0, 0, 0,
			byte(OP_HALT),
		},
		Data:      makeTestData(5, 7),
		MaxCycles: 1000,
	}
	
	// Run multiple times
	results := make([]*OCXResult, 5)
	for i := 0; i < 5; i++ {
		result, err := OCX_EXEC(input)
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}
		results[i] = result
	}
	
	// All results should be identical
	firstReceipt := results[0].Receipt
	for i := 1; i < len(results); i++ {
		if results[i].Receipt.ArtifactHash != firstReceipt.ArtifactHash {
			t.Fatal("Artifact hash not deterministic")
		}
		if results[i].Receipt.InputCommit != firstReceipt.InputCommit {
			t.Fatal("Input commit not deterministic")
		}
		if results[i].Receipt.OutputHash != firstReceipt.OutputHash {
			t.Fatal("Output hash not deterministic")
		}
		if results[i].Receipt.CyclesUsed != firstReceipt.CyclesUsed {
			t.Fatal("Cycles used not deterministic")
		}
		if results[i].Receipt.Price != firstReceipt.Price {
			t.Fatal("Price not deterministic")
		}
	}
}

// TestCycleLimits ensures cycle limits are enforced
func TestCycleLimits(t *testing.T) {
	// Create infinite loop code
	code := []byte{
		byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
		byte(OP_LOAD), 8, 0, 0, 0, // LOAD from address 8
		byte(OP_ADD),              // ADD
		byte(OP_JUMP), 0, 0, 0, 0, // JUMP to start (infinite loop)
	}
	
	input := OCXInput{
		Code:      code,
		Data:      makeTestData(5, 7),
		MaxCycles: 100, // Very low limit
	}
	
	_, err := OCX_EXEC(input)
	if err == nil {
		t.Fatal("Expected cycle limit exceeded error")
	}
	if err.Error() != "cycle limit exceeded" {
		t.Fatalf("Expected 'cycle limit exceeded', got: %v", err)
	}
}

// TestMemoryBounds ensures memory access is bounds-checked
func TestMemoryBounds(t *testing.T) {
	code := []byte{
		byte(OP_LOAD), 0xFF, 0xFF, 0xFF, 0xFF, // LOAD from invalid address
		byte(OP_HALT),
	}
	
	input := OCXInput{
		Code:      code,
		Data:      makeTestData(5, 7),
		MaxCycles: 1000,
	}
	
	_, err := OCX_EXEC(input)
	if err == nil {
		t.Fatal("Expected memory access error")
	}
	if err.Error() != "memory access out of bounds" {
		t.Fatalf("Expected 'memory access out of bounds', got: %v", err)
	}
}

// TestDivisionByZero ensures division by zero is handled
func TestDivisionByZero(t *testing.T) {
	code := []byte{
		byte(OP_LOAD), 0, 0, 0, 0, // LOAD from address 0
		byte(OP_LOAD), 8, 0, 0, 0, // LOAD from address 8
		byte(OP_DIV),              // DIV
		byte(OP_HALT),
	}
	
	input := OCXInput{
		Code:      code,
		Data:      makeTestData(5, 0), // Second value is 0
		MaxCycles: 1000,
	}
	
	_, err := OCX_EXEC(input)
	if err == nil {
		t.Fatal("Expected division by zero error")
	}
	if err.Error() != "division by zero" {
		t.Fatalf("Expected 'division by zero', got: %v", err)
	}
}

// Helper function to create test data
func makeTestData(a, b uint64) []byte {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], a)
	binary.LittleEndian.PutUint64(data[8:16], b)
	return data
}
