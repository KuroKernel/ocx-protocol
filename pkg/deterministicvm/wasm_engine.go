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
		
		// Calculate resource usage
		memoryUsed := w.getMemoryUsage(instance)
		gasUsed := w.estimateGasUsage(duration, memoryUsed)
		
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
		
		// Calculate resource usage
		memoryUsed := w.getMemoryUsage(instance)
		gasUsed := w.estimateGasUsage(duration, memoryUsed)
		
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

// estimateGasUsage estimates gas usage based on execution time and memory
func (w *WASMEngine) estimateGasUsage(duration time.Duration, memoryUsed uint64) uint64 {
	// Simple gas estimation model
	// Base cost + time cost + memory cost
	baseCost := uint64(1000)
	timeCost := uint64(duration.Nanoseconds()) / 1000 // 1 gas per microsecond
	memoryCost := memoryUsed / 1024                   // 1 gas per KB
	
	return baseCost + timeCost + memoryCost
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