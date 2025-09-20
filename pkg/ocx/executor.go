package ocx

import (
	"crypto/sha256"
	"fmt"
)

// MockExecutor provides a simple implementation of OCXExecutor for testing
type MockExecutor struct{}

// New creates a new OCX executor
func New() *MockExecutor {
	return &MockExecutor{}
}

// Execute implements the OCXExecutor interface
func (e *MockExecutor) Execute(artifact []byte, input []byte, maxCycles uint64) (*OCXResult, error) {
	// Validate inputs
	if len(artifact) == 0 {
		return nil, fmt.Errorf("artifact cannot be empty")
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("input cannot be empty")
	}
	if maxCycles == 0 {
		return nil, fmt.Errorf("maxCycles must be greater than 0")
	}

	// Simulate deterministic execution
	// In a real implementation, this would run the actual computation
	combined := append(artifact, input...)
	outputHash := sha256.Sum256(combined)
	
	// Simulate cycle usage (use ~47% of max cycles)
	cyclesUsed := uint64(float64(maxCycles) * 0.47)
	if cyclesUsed == 0 {
		cyclesUsed = 1
	}

	// Create a mock receipt hash
	receiptData := append(outputHash[:], byte(cyclesUsed))
	receiptHash := sha256.Sum256(receiptData)

	// Create a mock receipt blob (simplified CBOR-like structure)
	receiptBlob := []byte(fmt.Sprintf("mock_receipt_%x_%d", outputHash[:8], cyclesUsed))

	return &OCXResult{
		OutputHash:  outputHash,
		CyclesUsed:  cyclesUsed,
		ReceiptHash: receiptHash,
		ReceiptBlob: receiptBlob,
	}, nil
}
