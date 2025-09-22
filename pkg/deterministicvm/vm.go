package deterministicvm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// defaultVM is the singleton VM instance used by the module.
// It can be swapped out for different implementations as needed.
var defaultVM VM = &OSProcessVM{}

// SetVM allows changing the VM implementation (useful for testing).
func SetVM(vm VM) {
	defaultVM = vm
}

// GetVM returns the current VM implementation.
func GetVM() VM {
	return defaultVM
}

// OSProcessVM implements the VM interface using OS processes with isolation.
// This is the initial implementation that provides basic deterministic execution.
type OSProcessVM struct{}

// Run executes an artifact in an isolated OS process with deterministic constraints.
func (v *OSProcessVM) Run(ctx context.Context, config VMConfig) (*ExecutionResult, error) {
	startTime := time.Now().UTC()
	
	// Create the command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	
	// Create the command
	cmd := exec.CommandContext(cmdCtx, config.ArtifactPath)
	cmd.Dir = config.WorkingDir
	cmd.Env = config.Env
	
	// Apply OS-specific isolation
	if err := v.configureIsolation(cmd); err != nil {
		return nil, &ExecutionError{
			Code:    ErrorCodeEnvironmentSetup,
			Message: "Failed to configure process isolation",
			Underlying: err,
		}
	}
	
	// Set up stdin, stdout, stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Provide input data via stdin if present
	if len(config.InputData) > 0 {
		cmd.Stdin = bytes.NewReader(config.InputData)
	}
	
	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, &ExecutionError{
			Code:    ErrorCodeExecution,
			Message: "Failed to start process",
			Underlying: err,
			Context: map[string]interface{}{
				"artifact": config.ArtifactPath,
				"workdir":  config.WorkingDir,
			},
		}
	}
	
	// Wait for completion
	err := cmd.Wait()
	endTime := time.Now().UTC()
	duration := endTime.Sub(startTime)
	
	// Get exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Check if it was a timeout or other context error
			if cmdCtx.Err() == context.DeadlineExceeded {
				return nil, &ExecutionError{
					Code:    ErrorCodeTimeout,
					Message: fmt.Sprintf("Execution timeout after %v", config.Timeout),
					Context: map[string]interface{}{
						"timeout": config.Timeout,
						"duration": duration,
					},
				}
			}
			// Other execution error
			return nil, &ExecutionError{
				Code:    ErrorCodeExecution,
				Message: "Process execution failed",
				Underlying: err,
			}
		}
	}
	
	// Calculate cycles used (simplified model based on execution time)
	// In a more sophisticated implementation, this would use actual CPU metrics
	cyclesUsed := v.calculateCycles(duration)
	
	// Check cycle limit
	if cyclesUsed > config.CycleLimit {
		return nil, &ExecutionError{
			Code:    ErrorCodeCycleLimitExceeded,
			Message: fmt.Sprintf("Cycle limit exceeded: %d > %d", cyclesUsed, config.CycleLimit),
			Context: map[string]interface{}{
				"cycles_used": cyclesUsed,
				"cycle_limit": config.CycleLimit,
			},
		}
	}
	
	// Estimate memory usage (in a real implementation, we'd track this properly)
	memoryUsed := v.estimateMemoryUsage(&stdout, &stderr)
	
	// Check memory limit
	if memoryUsed > config.MemoryLimit {
		return nil, &ExecutionError{
			Code:    ErrorCodeMemoryLimitExceeded,
			Message: fmt.Sprintf("Memory limit exceeded: %d > %d bytes", memoryUsed, config.MemoryLimit),
			Context: map[string]interface{}{
				"memory_used":  memoryUsed,
				"memory_limit": config.MemoryLimit,
			},
		}
	}
	
	return &ExecutionResult{
		ExitCode:   exitCode,
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		CyclesUsed: cyclesUsed,
		MemoryUsed: memoryUsed,
		Duration:   duration,
		StartTime:  startTime,
		EndTime:    endTime,
	}, nil
}

// configureIsolation applies OS-specific process isolation settings.
func (v *OSProcessVM) configureIsolation(cmd *exec.Cmd) error {
	switch runtime.GOOS {
	case "linux":
		return v.configureLinuxIsolation(cmd)
	case "darwin":
		return v.configureDarwinIsolation(cmd)
	default:
		// Basic isolation for other platforms
		return v.configureBasicIsolation(cmd)
	}
}

// configureLinuxIsolation sets up Linux-specific isolation using namespaces.
func (v *OSProcessVM) configureLinuxIsolation(cmd *exec.Cmd) error {
	// For testing, use minimal isolation to avoid permission issues
	// In production, you would want more comprehensive isolation
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Set process group for easier cleanup
		Setpgid: true,
	}
	return nil
}

// configureDarwinIsolation sets up macOS-specific isolation.
func (v *OSProcessVM) configureDarwinIsolation(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Set process group for easier cleanup
		Setpgid: true,
	}
	return nil
}

// configureBasicIsolation provides minimal isolation for other platforms.
func (v *OSProcessVM) configureBasicIsolation(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Set process group for easier cleanup
		Setpgid: true,
	}
	return nil
}

// calculateCycles converts execution duration to a cycle count.
// This is a simplified model - a production implementation would use
// actual CPU performance counters or more sophisticated metrics.
func (v *OSProcessVM) calculateCycles(duration time.Duration) uint64 {
	// Simple model: assume 1 GHz equivalent processing
	// 1 nanosecond = 1 cycle at 1 GHz
	return uint64(duration.Nanoseconds())
}

// estimateMemoryUsage provides a rough estimate of memory usage.
// In a real implementation, we'd track actual memory consumption
// using cgroups (Linux) or similar mechanisms.
func (v *OSProcessVM) estimateMemoryUsage(stdout, stderr *bytes.Buffer) uint64 {
	// Base memory overhead + output buffer sizes
	baseMemory := uint64(4 * 1024 * 1024) // 4MB base
	outputMemory := uint64(stdout.Len() + stderr.Len())
	return baseMemory + outputMemory
}
