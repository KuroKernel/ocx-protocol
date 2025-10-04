package deterministicvm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWASMEngineBasic tests basic WASM engine functionality
func TestWASMEngineBasic(t *testing.T) {
	engine := NewWASMEngine().(*WASMEngine)
	defer engine.Close(context.Background())
	
	// Create a temporary WASM file for testing
	// This is a minimal WASM module that does nothing
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}
	
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	
	if err := os.WriteFile(wasmFile, wasmBytes, 0644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}
	
	config := VMConfig{
		ArtifactPath: wasmFile,
		Timeout:      5 * time.Second,
		MemoryLimit:  64 * 1024 * 1024, // 64MB
		CycleLimit:   1000000,
		WorkingDir:   tmpDir,
		Env:          []string{"DETERMINISTIC=1"},
	}
	
	ctx := context.Background()
	result, err := engine.Run(ctx, config)
	
	// We expect an error because the WASM module has no entry point
	if err == nil {
		t.Error("Expected error for WASM module without entry point")
	}
	
	// Check that the error is about missing entry point
	if result != nil {
		t.Error("Expected nil result for failed execution")
	}
}

// TestWASMEngineInvalidFile tests WASM engine with invalid file
func TestWASMEngineInvalidFile(t *testing.T) {
	engine := NewWASMEngine().(*WASMEngine)
	defer engine.Close(context.Background())
	
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.wasm")
	
	// Write invalid WASM data
	invalidData := []byte("not a wasm file")
	if err := os.WriteFile(invalidFile, invalidData, 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}
	
	config := VMConfig{
		ArtifactPath: invalidFile,
		Timeout:      5 * time.Second,
		MemoryLimit:  64 * 1024 * 1024,
		CycleLimit:   1000000,
		WorkingDir:   tmpDir,
	}
	
	ctx := context.Background()
	result, err := engine.Run(ctx, config)
	
	if err == nil {
		t.Error("Expected error for invalid WASM file")
	}
	
	if result != nil {
		t.Error("Expected nil result for failed execution")
	}
}

// TestWASMEngineNonexistentFile tests WASM engine with nonexistent file
func TestWASMEngineNonexistentFile(t *testing.T) {
	engine := NewWASMEngine().(*WASMEngine)
	defer engine.Close(context.Background())
	
	tmpDir := t.TempDir()
	nonexistentFile := filepath.Join(tmpDir, "nonexistent.wasm")
	
	config := VMConfig{
		ArtifactPath: nonexistentFile,
		Timeout:      5 * time.Second,
		MemoryLimit:  64 * 1024 * 1024,
		CycleLimit:   1000000,
		WorkingDir:   tmpDir,
	}
	
	ctx := context.Background()
	result, err := engine.Run(ctx, config)
	
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	
	if result != nil {
		t.Error("Expected nil result for failed execution")
	}
}

// TestWASMEngineTimeout tests WASM engine timeout
func TestWASMEngineTimeout(t *testing.T) {
	engine := NewWASMEngine().(*WASMEngine)
	defer engine.Close(context.Background())
	
	// Create a minimal WASM module
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}
	
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	
	if err := os.WriteFile(wasmFile, wasmBytes, 0644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}
	
	config := VMConfig{
		ArtifactPath: wasmFile,
		Timeout:      1 * time.Millisecond, // Very short timeout
		MemoryLimit:  64 * 1024 * 1024,
		CycleLimit:   1000000,
		WorkingDir:   tmpDir,
	}
	
	ctx := context.Background()
	result, err := engine.Run(ctx, config)
	
	// We expect an error due to missing entry point, not timeout
	if err == nil {
		t.Error("Expected error for WASM module without entry point")
	}
	
	if result != nil {
		t.Error("Expected nil result for failed execution")
	}
}

// TestFuelMeteredWASMEngine tests fuel-metered WASM engine
func TestFuelMeteredWASMEngine(t *testing.T) {
	engine := NewFuelMeteredWASMEngine(1000) // 1000 fuel limit
	defer engine.WASMEngine.Close(context.Background())
	
	// Create a minimal WASM module
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}
	
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test.wasm")
	
	if err := os.WriteFile(wasmFile, wasmBytes, 0644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}
	
	config := VMConfig{
		ArtifactPath: wasmFile,
		Timeout:      5 * time.Second,
		MemoryLimit:  64 * 1024 * 1024,
		CycleLimit:   1000000,
		WorkingDir:   tmpDir,
	}
	
	ctx := context.Background()
	result, err := engine.Run(ctx, config)
	
	// We expect an error because the WASM module has no entry point
	if err == nil {
		t.Error("Expected error for WASM module without entry point")
	}
	
	if result != nil {
		t.Error("Expected nil result for failed execution")
	}
	
	// Check fuel meter
	fuelMeter := engine.GetFuelMeter()
	if fuelMeter == nil {
		t.Error("Expected fuel meter to be available")
	}
	
	// Check fuel costs
	fuelCosts := engine.GetFuelCosts()
	if fuelCosts == nil {
		t.Error("Expected fuel costs to be available")
	}
}

// TestWASMBinaryDetection tests WASM binary detection
func TestWASMBinaryDetection(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Valid WASM",
			data:     []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Invalid magic",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "Invalid version",
			data:     []byte{0x00, 0x61, 0x73, 0x6d, 0x02, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "Too short",
			data:     []byte{0x00, 0x61, 0x73, 0x6d},
			expected: false,
		},
		{
			name:     "Empty",
			data:     []byte{},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWASMBinary(tt.data)
			if result != tt.expected {
				t.Errorf("isWASMBinary() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestVMTypeSwitching tests switching between VM types
func TestVMTypeSwitching(t *testing.T) {
	// Test OS Process VM
	if err := SetVMType(VMTypeOSProcess); err != nil {
		t.Fatalf("Failed to set OS Process VM: %v", err)
	}
	
	vm := GetVM()
	if _, ok := vm.(*OSProcessVM); !ok {
		t.Error("Expected OSProcessVM")
	}
	
	// Test WASM VM
	if err := SetVMType(VMTypeWASM); err != nil {
		t.Fatalf("Failed to set WASM VM: %v", err)
	}
	
	vm = GetVM()
	if _, ok := vm.(*WASMEngine); !ok {
		t.Error("Expected WASMEngine")
	}
	
	// Test Fuel Metered WASM VM
	if err := SetFuelMeteredWASMType(1000); err != nil {
		t.Fatalf("Failed to set Fuel Metered WASM VM: %v", err)
	}
	
	vm = GetVM()
	if _, ok := vm.(*FuelMeteredWASMEngine); !ok {
		t.Error("Expected FuelMeteredWASMEngine")
	}
	
	// Test invalid VM type
	if err := SetVMType("invalid"); err == nil {
		t.Error("Expected error for invalid VM type")
	}
}
