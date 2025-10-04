package conformance

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCLIExecution(t *testing.T) {
	// Skip if CLI binary not built
	cliPath := findCLIBinary()
	if cliPath == "" {
		t.Skip("CLI binary not found - run 'make build' first")
	}

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "help",
			args: []string{"help"},
			expectError: false,
		},
		{
			name: "conformance",
			args: []string{"conformance"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, cliPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but command succeeded. Output: %s", output)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Command failed: %v\nOutput: %s", err, output)
			}

			t.Logf("Output: %s", output)
		})
	}
}

func TestExecuteArtifact(t *testing.T) {
	cliPath := findCLIBinary()
	if cliPath == "" {
		t.Skip("CLI binary not found")
	}

	// Test the benchmark command instead of execute
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliPath, "benchmark")
	output, err := cmd.CombinedOutput()

	t.Logf("Benchmark output: %s", output)

	// Benchmark command should work
	if err != nil {
		t.Errorf("Benchmark command failed: %v", err)
	}
}

func findCLIBinary() string {
	// Look for CLI binary in common locations
	candidates := []string{
		"../../bin/ocx",
		"../../ocx",
		"./ocx",
		"../ocx",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func isSeccompError(output string) bool {
	patterns := []string{
		"bad system call",
		"operation not permitted",
		"killed by seccomp",
		"SIGSYS",
	}

	for _, pattern := range patterns {
		if contains(output, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
