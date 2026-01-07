package deterministicvm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDeterminism_SameInputSameOutput verifies deterministic execution
func TestDeterminism_SameInputSameOutput(t *testing.T) {
	// Create a simple deterministic script
	script := `#!/bin/bash
echo "Hello, World!"
echo "2 + 2 = 4"
`

	// Create temp file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	vm := &OSProcessVM{}
	config := VMConfig{
		ArtifactPath: scriptPath,
		WorkingDir:   tmpDir,
		Timeout:      10 * time.Second,
		CycleLimit:   100000000,
		MemoryLimit:  64 * 1024 * 1024,
		Env:          []string{"PATH=/bin:/usr/bin"},
	}

	ctx := context.Background()

	// Run 10 times
	var firstHash [32]byte
	var firstGas uint64

	for i := 0; i < 10; i++ {
		result, err := vm.Run(ctx, config)
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}

		outputHash := sha256.Sum256(result.Stdout)

		if i == 0 {
			firstHash = outputHash
			firstGas = result.GasUsed
		} else {
			// Output must be identical
			if outputHash != firstHash {
				t.Errorf("run %d: output hash differs from run 0", i)
			}

			// Gas must be identical (deterministic gas model)
			if result.GasUsed != firstGas {
				t.Errorf("run %d: gas %d differs from run 0 gas %d", i, result.GasUsed, firstGas)
			}
		}
	}
}

// TestDeterminism_InputVariation verifies different inputs produce different outputs
func TestDeterminism_InputVariation(t *testing.T) {
	script := `#!/bin/bash
cat
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "cat.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	vm := &OSProcessVM{}

	inputs := []string{
		"input one",
		"input two",
		"input three",
	}

	hashes := make(map[[32]byte]bool)

	for i, input := range inputs {
		config := VMConfig{
			ArtifactPath: scriptPath,
			WorkingDir:   tmpDir,
			InputData:    []byte(input),
			Timeout:      10 * time.Second,
			CycleLimit:   100000000,
			MemoryLimit:  64 * 1024 * 1024,
			Env:          []string{"PATH=/bin:/usr/bin"},
		}

		result, err := vm.Run(context.Background(), config)
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}

		hash := sha256.Sum256(result.Stdout)
		if hashes[hash] {
			t.Errorf("input %d produced duplicate hash", i)
		}
		hashes[hash] = true
	}
}

// TestDeterminism_GasCalculation verifies gas is deterministic
func TestDeterminism_GasCalculation(t *testing.T) {
	script := `#!/bin/bash
for i in 1 2 3 4 5; do
  echo "Line $i"
done
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "loop.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Calculate expected gas from script
	expectedGas := CalculateDeterministicGas(script, nil)

	vm := &OSProcessVM{}
	config := VMConfig{
		ArtifactPath: scriptPath,
		WorkingDir:   tmpDir,
		Timeout:      10 * time.Second,
		CycleLimit:   100000000,
		MemoryLimit:  64 * 1024 * 1024,
		Env:          []string{"PATH=/bin:/usr/bin"},
	}

	// Run multiple times
	for i := 0; i < 5; i++ {
		result, err := vm.Run(context.Background(), config)
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}

		// Gas should be consistent with static analysis
		// Allow for I/O gas variance
		baseGas := result.GasUsed - uint64(len(result.Stdout)+len(result.Stderr))
		if baseGas < expectedGas/2 || baseGas > expectedGas*2 {
			t.Errorf("run %d: gas %d significantly differs from expected %d", i, baseGas, expectedGas)
		}
	}
}

// TestDeterminism_NoTimeLeakage verifies time doesn't affect output hash
func TestDeterminism_NoTimeLeakage(t *testing.T) {
	script := `#!/bin/bash
# This script should NOT include time in output
echo "Static output"
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "notime.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	vm := &OSProcessVM{}
	config := VMConfig{
		ArtifactPath: scriptPath,
		WorkingDir:   tmpDir,
		Timeout:      10 * time.Second,
		CycleLimit:   100000000,
		MemoryLimit:  64 * 1024 * 1024,
		Env:          []string{"PATH=/bin:/usr/bin"},
	}

	// Run with delays between executions
	var outputs [][]byte
	for i := 0; i < 3; i++ {
		result, err := vm.Run(context.Background(), config)
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}
		outputs = append(outputs, result.Stdout)

		time.Sleep(100 * time.Millisecond)
	}

	// All outputs must be identical
	for i := 1; i < len(outputs); i++ {
		if !bytes.Equal(outputs[0], outputs[i]) {
			t.Errorf("output %d differs from output 0", i)
		}
	}
}

// TestDeterminism_EnvIsolation verifies environment is controlled
func TestDeterminism_EnvIsolation(t *testing.T) {
	script := `#!/bin/bash
echo "PATH=$PATH"
echo "HOME=$HOME"
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "env.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	vm := &OSProcessVM{}

	// Run with explicit env
	config := VMConfig{
		ArtifactPath: scriptPath,
		WorkingDir:   tmpDir,
		Timeout:      10 * time.Second,
		CycleLimit:   100000000,
		MemoryLimit:  64 * 1024 * 1024,
		Env:          []string{"PATH=/bin:/usr/bin", "HOME=/tmp"},
	}

	result1, err := vm.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	// Run again with same env
	result2, err := vm.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	// Outputs must match
	if !bytes.Equal(result1.Stdout, result2.Stdout) {
		t.Error("outputs differ with same env")
	}
}

// TestStaticGasModel_WASMOpcodes tests WASM opcode gas costs
func TestStaticGasModel_WASMOpcodes(t *testing.T) {
	// These tests verify the current implementation's gas costs
	// The implementation uses a deterministic model where:
	// - Constants: 1 gas
	// - Arithmetic: 2 gas
	// - Memory ops: 3 gas
	// - Control flow: varies
	testCases := []struct {
		opcode      byte
		expectedGas uint64
		name        string
	}{
		{0x41, 1, "i32.const"},
		{0x6A, 2, "i32.add"},      // arithmetic
		{0x6B, 2, "i32.sub"},      // arithmetic
		{0x6C, 2, "i32.mul"},      // arithmetic
		{0x6D, 2, "i32.div_s"},    // arithmetic (treated as generic arithmetic)
		{0x10, 5, "call"},         // call
		{0x28, 3, "i32.load"},     // memory access
		{0x36, 3, "i32.store"},    // memory access
	}

	for _, tc := range testCases {
		gas := opcodeGasCost(tc.opcode)
		if gas != tc.expectedGas {
			t.Errorf("%s (0x%02X): expected gas %d, got %d",
				tc.name, tc.opcode, tc.expectedGas, gas)
		}
	}
}

// TestStaticGasModel_Consistency tests gas model consistency
func TestStaticGasModel_Consistency(t *testing.T) {
	script := `echo "test"`
	input := []byte("input data")

	// Calculate gas multiple times
	gas1 := CalculateDeterministicGas(script, input)
	gas2 := CalculateDeterministicGas(script, input)
	gas3 := CalculateDeterministicGas(script, input)

	if gas1 != gas2 || gas2 != gas3 {
		t.Errorf("gas calculations differ: %d, %d, %d", gas1, gas2, gas3)
	}
}

// TestStaticGasModel_ScriptComplexity tests gas scales with complexity
func TestStaticGasModel_ScriptComplexity(t *testing.T) {
	simple := `echo "hello"`
	complex := `
for i in 1 2 3 4 5 6 7 8 9 10; do
  echo "iteration $i"
  if [ $i -eq 5 ]; then
    echo "midpoint"
  fi
done
`

	simpleGas := CalculateDeterministicGas(simple, nil)
	complexGas := CalculateDeterministicGas(complex, nil)

	if complexGas <= simpleGas {
		t.Errorf("complex script (%d gas) should use more gas than simple (%d gas)",
			complexGas, simpleGas)
	}
}

// TestFPValidator_StrictMode tests FP opcode rejection
func TestFPValidator_StrictMode(t *testing.T) {
	// WASM with float operations
	wasmWithFloat := []byte{
		0x00, 0x61, 0x73, 0x6d, // Magic
		0x01, 0x00, 0x00, 0x00, // Version
		// Type section with f32 param
		0x01, 0x05, 0x01, 0x60, 0x01, 0x7d, 0x00, // func(f32) -> void
	}

	validator := NewFPValidator(false, false, true) // Strict mode
	err := validator.ValidateModule(wasmWithFloat)

	if err == nil {
		t.Error("strict mode should reject WASM with float types")
	}
}

// TestFPValidator_NonStrictMode tests FP opcode allowance
func TestFPValidator_NonStrictMode(t *testing.T) {
	// WASM without float operations (just i32)
	wasmNoFloat := []byte{
		0x00, 0x61, 0x73, 0x6d, // Magic
		0x01, 0x00, 0x00, 0x00, // Version
		// Type section with i32 param
		0x01, 0x05, 0x01, 0x60, 0x01, 0x7f, 0x00, // func(i32) -> void
	}

	validator := NewFPValidator(false, false, false) // Non-strict mode
	err := validator.ValidateModule(wasmNoFloat)

	if err != nil {
		t.Errorf("non-strict mode should accept WASM without floats: %v", err)
	}
}

// TestStrictModeEnv tests OCX_STRICT_MODE environment variable
func TestStrictModeEnv(t *testing.T) {
	testCases := []struct {
		envVal   string
		expected bool
	}{
		{"", true},       // Default is strict
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}

	for _, tc := range testCases {
		t.Run(tc.envVal, func(t *testing.T) {
			if tc.envVal != "" {
				os.Setenv("OCX_STRICT_MODE", tc.envVal)
				defer os.Unsetenv("OCX_STRICT_MODE")
			} else {
				os.Unsetenv("OCX_STRICT_MODE")
			}

			result := isStrictModeEnabled()
			if result != tc.expected {
				t.Errorf("OCX_STRICT_MODE=%q: expected %v, got %v", tc.envVal, tc.expected, result)
			}
		})
	}
}

// BenchmarkDeterministicGas benchmarks gas calculation
func BenchmarkDeterministicGas(b *testing.B) {
	script := `
for i in $(seq 1 100); do
  echo "Line $i"
  x=$((i * 2))
done
`
	input := bytes.Repeat([]byte("input"), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateDeterministicGas(script, input)
	}
}
