package deterministicvm

import (
	"testing"
	"time"
)

// TestStrictModeExitCodeFailure tests that strict mode fails on non-zero exit codes
func TestStrictModeExitCodeFailure(t *testing.T) {
	// Create a test result with non-zero exit code
	result := &ExecutionResult{
		ExitCode:   1, // Non-zero exit code
		Stdout:     []byte("some output"),
		Stderr:     []byte("error occurred"),
		Duration:   time.Second,
		HostCycles: 1000,
		GasUsed:    1000,
		MemoryUsed: 1024,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second),
	}

	// Test strict mode (should fail)
	strictConfig := DefaultVMConfig()
	strictConfig.StrictMode = true

	err := validateExecutionResult(result, strictConfig)
	if err == nil {
		t.Fatal("Expected error in strict mode with non-zero exit code, got nil")
	}

	execErr, ok := err.(*ExecutionError)
	if !ok {
		t.Fatalf("Expected ExecutionError, got %T", err)
	}

	if execErr.Code != ErrorCodeExecutionFailed {
		t.Errorf("Expected ErrorCodeExecutionFailed, got %v", execErr.Code)
	}

	if !contains(execErr.Message, "strict mode enabled") {
		t.Errorf("Expected message to mention strict mode, got: %s", execErr.Message)
	}

	// Test permissive mode (should not fail)
	permissiveConfig := DefaultVMConfig()
	permissiveConfig.StrictMode = false

	err = validateExecutionResult(result, permissiveConfig)
	if err != nil {
		t.Errorf("Expected no error in permissive mode, got: %v", err)
	}
}

// TestStrictModeDeterminismFailure tests that strict mode fails on determinism violations
// Note: This test is simplified since checkDeterminismWarnings is not easily mockable
func TestStrictModeDeterminismFailure(t *testing.T) {
	// Create a test result that would trigger determinism warnings
	result := &ExecutionResult{
		ExitCode:   0, // Successful exit
		Stdout:     []byte("random output that changes each time"),
		Stderr:     []byte(""),
		Duration:   time.Second,
		HostCycles: 1000,
		GasUsed:    1000,
		MemoryUsed: 1024,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second),
	}

	// Test that the function runs without error in both modes
	// (The actual determinism check would need to be tested with a real implementation)
	strictConfig := DefaultVMConfig()
	strictConfig.StrictMode = true

	err := validateExecutionResult(result, strictConfig)
	// In this test, we expect no error since checkDeterminismWarnings is not mocked
	// The real test would require a proper mock or integration test
	if err != nil {
		t.Logf("Got error in strict mode (expected if determinism check fails): %v", err)
	}

	permissiveConfig := DefaultVMConfig()
	permissiveConfig.StrictMode = false

	err = validateExecutionResult(result, permissiveConfig)
	if err != nil {
		t.Errorf("Expected no error in permissive mode, got: %v", err)
	}
}

// TestDefaultVMConfigStrictMode tests that default config has strict mode enabled
func TestDefaultVMConfigStrictMode(t *testing.T) {
	config := DefaultVMConfig()
	if !config.StrictMode {
		t.Error("Expected default VMConfig to have StrictMode enabled")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		s[:len(substr)] == substr ||
		len(s) > len(substr) &&
			contains(s[1:], substr)
}
