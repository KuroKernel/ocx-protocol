package deterministicvm

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMEngine implements the VM interface using WebAssembly for deterministic execution
type WASMEngine struct {
	runtime wazero.Runtime
	config  wazero.ModuleConfig
}

// NewWASMEngine creates a new WASM engine with deterministic configuration
func NewWASMEngine() VM {
	ctx := context.Background()
	
	// Create runtime with deterministic configuration
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(false). // Don't close on context done
		WithMemoryLimitPages(1024))    // 64MB memory limit
	
	// Create module config with deterministic settings
	config := wazero.NewModuleConfig().
		WithSysNanosleep().           // Controlled time
		WithSysNanotime().            // Monotonic time only
		WithEnv("DETERMINISTIC", "1"). // Mark as deterministic
		WithEnv("OCX_MODE", "wasm")   // OCX-specific mode
	
	return &WASMEngine{
		runtime: runtime,
		config:  config,
	}
}

// Run executes a WASM module with deterministic constraints
func (w *WASMEngine) Run(ctx context.Context, config VMConfig) (*ExecutionResult, error) {
	// Read WASM module from artifact path
	wasmBytes, err := os.ReadFile(config.ArtifactPath)
	if err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "Failed to read WASM module",
			Underlying: err,
		}
	}
	
	// Validate WASM module
	if !isWASMBinary(wasmBytes) {
		return nil, &ExecutionError{
			Code:    ErrorCodeExecution,
			Message: "Artifact is not a valid WASM file",
		}
	}

	// Validate floating-point operations in strict mode
	// In strict mode (production), reject ALL floating-point operations
	strictMode := isStrictModeEnabled()
	fpValidator := NewFPValidator(false, false, strictMode)
	if err := fpValidator.ValidateModule(wasmBytes); err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "WASM module contains disallowed floating-point operations",
			Underlying: err,
		}
	}

	// Compile module with deterministic settings
	mod, err := w.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "WASM compilation failed",
			Underlying: err,
		}
	}
	defer mod.Close(ctx)
	
	// Create deterministic module config
	moduleConfig := w.config.
		WithName("deterministic-exec").
		WithStartFunctions() // Disable WASI _start
	
	// Instantiate module
	instance, err := w.runtime.InstantiateModule(ctx, mod, moduleConfig)
	if err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "WASM instantiation failed",
			Underlying: err,
		}
	}
	defer instance.Close(ctx)
	
	// Execute main function
	startTime := time.Now()
	
	// Try to call main function first
	mainFunc := instance.ExportedFunction("main")
	if mainFunc != nil {
		results, err := mainFunc.Call(ctx)
		if err != nil {
			return nil, &ExecutionError{
				Code:       ErrorCodeExecution,
				Message:    "WASM main function execution failed",
				Underlying: err,
			}
		}
		
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		
		// Convert results to bytes
		var output []byte
		if len(results) > 0 {
			output = []byte(fmt.Sprintf("%v", results[0]))
		}
		
		// Calculate resource usage (deterministic - based on bytecode, not time)
		memoryUsed := w.getMemoryUsage(instance)
		gasUsed := w.estimateGasUsage(wasmBytes, memoryUsed)

		return &ExecutionResult{
			ExitCode:   0,
			Stdout:     output,
			Stderr:     nil,
			GasUsed:    gasUsed,
			HostCycles: uint64(duration.Nanoseconds()), // Diagnostic only, not in signed core
			MemoryUsed: memoryUsed,
			Duration:   duration,
			StartTime:  startTime,
			EndTime:    endTime,
			HostInfo: HostInfo{
				Platform: "wasm/deterministic",
			},
		}, nil
	}

	// Fallback: try _start function
	startFunc := instance.ExportedFunction("_start")
	if startFunc != nil {
		results, err := startFunc.Call(ctx)
		if err != nil {
			return nil, &ExecutionError{
				Code:       ErrorCodeExecution,
				Message:    "WASM _start function execution failed",
				Underlying: err,
			}
		}

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Convert results to bytes
		var output []byte
		if len(results) > 0 {
			output = []byte(fmt.Sprintf("%v", results[0]))
		}

		// Calculate resource usage (deterministic - based on bytecode, not time)
		memoryUsed := w.getMemoryUsage(instance)
		gasUsed := w.estimateGasUsage(wasmBytes, memoryUsed)
		
		return &ExecutionResult{
			ExitCode:   0,
			Stdout:     output,
			Stderr:     nil,
			GasUsed:    gasUsed,
			HostCycles: uint64(duration.Nanoseconds()),
			MemoryUsed: memoryUsed,
			Duration:   duration,
			StartTime:  startTime,
			EndTime:    endTime,
			HostInfo: HostInfo{
				Platform: "wasm/deterministic",
			},
		}, nil
	}
	
	// No entry point found
	return nil, &ExecutionError{
		Code:    ErrorCodeExecution,
		Message: "No entry point found in WASM module (main or _start)",
	}
}

// getMemoryUsage returns the current memory usage of the WASM instance
func (w *WASMEngine) getMemoryUsage(instance api.Module) uint64 {
	memory := instance.Memory()
	if memory == nil {
		return 0
	}
	
	// Return the size of the linear memory in bytes
	return uint64(memory.Size())
}

// calculateStaticWASMGas calculates deterministic gas by counting WASM instructions
// This is a static analysis approach - same bytecode always yields same gas
func calculateStaticWASMGas(wasmBytes []byte) uint64 {
	if len(wasmBytes) < 8 {
		return 100 // Minimum gas
	}

	// Parse WASM and count instructions in code section
	gas := uint64(100) // Base cost

	// Find code section (section ID 10)
	offset := 8 // Skip magic and version
	for offset < len(wasmBytes) {
		if offset >= len(wasmBytes) {
			break
		}
		sectionID := wasmBytes[offset]
		offset++

		// Read section size (LEB128)
		sectionSize, bytesRead := readLEB128(wasmBytes[offset:])
		offset += bytesRead

		if sectionID == 10 { // Code section
			// Count instructions in code section
			gas += countCodeSectionInstructions(wasmBytes[offset : offset+int(sectionSize)])
		}

		offset += int(sectionSize)
	}

	// Add cost for module size (1 gas per 100 bytes)
	gas += uint64(len(wasmBytes)) / 100

	return gas
}

// readLEB128 reads an unsigned LEB128 value
func readLEB128(data []byte) (uint64, int) {
	var result uint64
	var shift uint
	bytesRead := 0

	for i := 0; i < len(data) && i < 10; i++ {
		b := data[i]
		bytesRead++
		result |= uint64(b&0x7f) << shift

		if b&0x80 == 0 {
			break
		}
		shift += 7
	}

	return result, bytesRead
}

// countCodeSectionInstructions counts instructions in a WASM code section
func countCodeSectionInstructions(codeSection []byte) uint64 {
	if len(codeSection) == 0 {
		return 0
	}

	// Read number of functions
	numFunctions, bytesRead := readLEB128(codeSection)
	offset := bytesRead

	var totalInstructions uint64

	for i := uint64(0); i < numFunctions && offset < len(codeSection); i++ {
		// Read function body size
		bodySize, br := readLEB128(codeSection[offset:])
		offset += br

		// Count instructions in function body
		// Simple approach: count non-zero bytes as potential instructions
		// More accurate would be full opcode parsing
		endOffset := offset + int(bodySize)
		if endOffset > len(codeSection) {
			endOffset = len(codeSection)
		}

		for j := offset; j < endOffset; j++ {
			opcode := codeSection[j]
			totalInstructions += opcodeGasCost(opcode)
		}

		offset = endOffset
	}

	return totalInstructions
}

// opcodeGasCost returns the gas cost for a WASM opcode
func opcodeGasCost(opcode byte) uint64 {
	switch {
	// Control flow (expensive)
	case opcode == 0x00: // unreachable
		return 1
	case opcode == 0x01: // nop
		return 1
	case opcode >= 0x02 && opcode <= 0x04: // block, loop, if
		return 2
	case opcode == 0x05: // else
		return 1
	case opcode == 0x0b: // end
		return 1
	case opcode >= 0x0c && opcode <= 0x0e: // br, br_if, br_table
		return 3
	case opcode == 0x0f: // return
		return 2
	case opcode == 0x10: // call
		return 5
	case opcode == 0x11: // call_indirect
		return 10

	// Memory operations
	case opcode >= 0x28 && opcode <= 0x3e: // load/store variants
		return 3
	case opcode == 0x3f: // memory.size
		return 2
	case opcode == 0x40: // memory.grow
		return 100

	// Constants
	case opcode == 0x41: // i32.const
		return 1
	case opcode == 0x42: // i64.const
		return 1
	case opcode == 0x43: // f32.const
		return 1
	case opcode == 0x44: // f64.const
		return 1

	// Comparison operations
	case opcode >= 0x45 && opcode <= 0x66:
		return 1

	// Arithmetic operations
	case opcode >= 0x67 && opcode <= 0x8a:
		return 2

	// Division/remainder (more expensive)
	case opcode >= 0x6d && opcode <= 0x71: // div, rem variants
		return 5

	// Float operations (if allowed)
	case opcode >= 0x8b && opcode <= 0xc4:
		return 3

	default:
		return 1
	}
}

// estimateGasUsage is kept for backward compatibility but now uses static analysis
func (w *WASMEngine) estimateGasUsage(wasmBytes []byte, memoryUsed uint64) uint64 {
	// Use static instruction counting instead of time-based estimation
	staticGas := calculateStaticWASMGas(wasmBytes)

	// Add memory cost (deterministic based on allocated memory)
	memoryCost := memoryUsed / 1024 // 1 gas per KB

	return staticGas + memoryCost
}

// Close cleans up the WASM runtime
func (w *WASMEngine) Close(ctx context.Context) error {
	return w.runtime.Close(ctx)
}

// isWASMBinary checks if the given bytes represent a valid WASM binary
func isWASMBinary(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	
	// Check WASM magic number: 0x6d736100 ("\0asm")
	magic := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	if magic != 0x6d736100 {
		return false
	}
	
	// Check version (should be 1)
	version := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	return version == 1
}