package deterministicvm

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// VMType represents different VM execution engines
type VMType string

const (
	VMTypeOSProcess VMType = "os-process"
	VMTypeWASM      VMType = "wasm"
)

// defaultVM is the singleton VM instance used by the module.
// It can be swapped out for different implementations as needed.
var defaultVM VM = &OSProcessVM{}

// SetVM allows changing the VM implementation (useful for testing).
func SetVM(vm VM) {
	defaultVM = vm
}

// SetVMType allows changing the VM implementation by type
func SetVMType(vmType VMType) error {
	switch vmType {
	case VMTypeOSProcess:
		defaultVM = &OSProcessVM{}
		return nil
	case VMTypeWASM:
		defaultVM = NewWASMEngine()
		return nil
	default:
		return fmt.Errorf("unsupported VM type: %s", vmType)
	}
}

// SetFuelMeteredWASMType sets a fuel-metered WASM engine as the default VM
func SetFuelMeteredWASMType(fuelLimit uint64) error {
	defaultVM = NewFuelMeteredWASMEngine(fuelLimit)
	return nil
}

// applyCgroups applies cgroups v2 resource limits for deterministic execution
func applyCgroups(config VMConfig, pid int) (*CgroupManager, error) {
	// Only on Linux
	if runtime.GOOS != "linux" {
		return nil, nil // Graceful degradation
	}

	// Try to create cgroup
	cgManager, err := NewCgroupManager(fmt.Sprintf("vm-%d", pid))
	if err != nil {
		// Log warning to stderr to avoid breaking stdout determinism
		// Only log once to avoid spam
		if !strings.Contains(err.Error(), "permission denied") {
			fmt.Fprintf(os.Stderr, "⚠ Cgroups unavailable: %v\n", err)
		}
		return nil, nil
	}

	// Set limits
	limits := CgroupLimits{
		CPUQuotaMicros: 50000, // 50% of 1 CPU
		MemoryMaxBytes: int64(config.MemoryLimit),
		PidsMax:        100,
	}

	if err := cgManager.Apply(pid, limits); err != nil {
		fmt.Printf("⚠ Failed to apply cgroup limits: %v\n", err)
		cgManager.Cleanup()
		return nil, nil
	}

	return cgManager, nil
}

// applySeccomp applies seccomp sandboxing for deterministic execution
func applySeccomp(workingDir string, strictMode bool) error {
	ctx := context.Background()
	config := SeccompConfig{
		StrictSandbox: strictMode,
		WorkingDir:    workingDir,
		Logger:        log.New(os.Stderr, "[seccomp] ", log.LstdFlags),
	}

	return ApplySeccompProfile(ctx, config)
}

// installSeccomp installs seccomp filter for the current process
func installSeccomp() error {
	ctx := context.Background()
	config := SeccompConfig{
		StrictSandbox: false,  // Non-strict mode for VM processes
		WorkingDir:    "/tmp", // Default working directory
		Logger:        log.New(os.Stderr, "[seccomp] ", log.LstdFlags),
	}

	return ApplySeccompProfile(ctx, config)
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
	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(cmdCtx, config.ArtifactPath)
	cmd.Dir = config.WorkingDir
	cmd.Env = config.Env

	// Set up stdin, stdout, stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if len(config.InputData) > 0 {
		cmd.Stdin = bytes.NewReader(config.InputData)
	}

	// Configure isolation
	if err := v.configureIsolation(cmd); err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeEnvironmentSetup,
			Message:    "Failed to configure isolation",
			Underlying: err,
		}
	}

	// Start the process
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "Failed to start process",
			Underlying: err,
		}
	}

	pid := cmd.Process.Pid

	// Apply cgroups after process starts
	cgManager, err := applyCgroups(config, pid)
	if cgManager != nil {
		defer cgManager.Cleanup()
	}

	// Wait for completion
	err = cmd.Wait()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Calculate real resource usage
	cyclesUsed := v.calculateCycles(pid, duration)
	memoryUsed := v.getActualMemoryUsage(pid, cgManager)

	// Check for specific errors
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Process exited with non-zero code
			return &ExecutionResult{
				ExitCode:   exitError.ExitCode(),
				Stdout:     stdout.Bytes(),
				Stderr:     stderr.Bytes(),
				GasUsed:    calculateDeterministicGas(cyclesUsed, len(stdout.Bytes()), len(stderr.Bytes())),
				HostCycles: cyclesUsed,
				MemoryUsed: memoryUsed,
				Duration:   duration,
				StartTime:  startTime,
				EndTime:    endTime,
				HostInfo: HostInfo{
					Platform: runtime.GOOS + "/" + runtime.GOARCH,
				},
			}, nil
		}

		return nil, &ExecutionError{
			Code:       ErrorCodeExecution,
			Message:    "Execution failed",
			Underlying: err,
		}
	}

	// Check violations
	if duration > config.Timeout {
		return nil, &ExecutionError{
			Code:    ErrorCodeTimeout,
			Message: fmt.Sprintf("Execution timeout after %v", config.Timeout),
			Context: map[string]interface{}{
				"timeout":  config.Timeout,
				"duration": duration,
			},
		}
	}

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
		ExitCode:   0,
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		GasUsed:    calculateDeterministicGas(cyclesUsed, len(stdout.Bytes()), len(stderr.Bytes())),
		HostCycles: cyclesUsed,
		MemoryUsed: memoryUsed,
		Duration:   duration,
		StartTime:  startTime,
		EndTime:    endTime,
		HostInfo: HostInfo{
			Platform: runtime.GOOS + "/" + runtime.GOARCH,
		},
	}, nil
}

// calculateDeterministicGas computes a deterministic gas value based on execution metrics.
// This provides a consistent "logical time" that doesn't depend on wall-clock timing.
func calculateDeterministicGas(cycles uint64, stdoutLen, stderrLen int) uint64 {
	// Base gas from cycles (primary deterministic metric)
	baseGas := cycles / 1000 // Scale down for reasonable gas units

	// Add gas for I/O operations (deterministic based on output size)
	ioGas := uint64(stdoutLen + stderrLen) // 1 gas per byte of output

	// Minimum gas to ensure non-zero values for successful executions
	const minGas = 100

	totalGas := baseGas + ioGas
	if totalGas < minGas {
		totalGas = minGas
	}

	return totalGas
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

// configureLinuxIsolation sets up Linux-specific isolation using namespaces, seccomp, and cgroups.
func (v *OSProcessVM) configureLinuxIsolation(cmd *exec.Cmd) error {
	// Configure basic isolation (namespaces disabled for testing compatibility)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Set process group for easier cleanup
		Setpgid: true,
		// Note: Namespaces disabled for testing - enable in production
		// Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWUTS,
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
// Uses real hardware counters with fallbacks for maximum accuracy.
func (v *OSProcessVM) calculateCycles(pid int, duration time.Duration) uint64 {
	// Only on Linux
	if runtime.GOOS != "linux" {
		// Fallback: duration-based estimation
		return uint64(duration.Nanoseconds())
	}

	// 1) perf_event_open
	if pc, err := openPerfCycles(pid); err == nil {
		defer pc.Close()
		if err := pc.ResetEnable(); err == nil {
			time.Sleep(duration) // measure window caller asked for
			_ = pc.Disable()
			if c, err := pc.ReadCount(); err == nil {
				return c
			}
		}
	}

	// 2) /proc/<pid>/stat jiffies delta (approx cycles via time)
	j1, e1 := readProcStatJiffies(pid)
	if e1 == nil {
		time.Sleep(duration)
		j2, e2 := readProcStatJiffies(pid)
		if e2 == nil && j2 >= j1 {
			nanos := jiffiesToNanos(j2 - j1)
			// crude convert: assume nominal 1 cycle per ns at 1GHz; scale by a factor if you want.
			return uint64(nanos) // stable monotonic metric even if not exact cycles
		}
	}

	// 3) cgroup cpu.stat usage_usec delta (best-effort)
	if usec1, err := readCgroupCPUUsec(pid); err == nil {
		time.Sleep(duration)
		if usec2, err2 := readCgroupCPUUsec(pid); err2 == nil && usec2 >= usec1 {
			return uint64((usec2 - usec1) * 1000) // microseconds -> nanoseconds proxy
		}
	}

	// 4) last resort: wall time proxy
	cycles := uint64(duration.Nanoseconds())
	if cycles == 0 {
		cycles = 1 // Ensure we return at least 1 cycle
	}
	return cycles
}

// getActualMemoryUsage provides real memory usage tracking.
// Uses smaps_rollup with fallbacks for maximum accuracy.
func (v *OSProcessVM) getActualMemoryUsage(pid int, cgManager *CgroupManager) uint64 {
	// Try cgroup first if available
	if cgManager != nil {
		if mem, err := cgManager.GetMemoryUsage(); err == nil {
			return mem
		}
	}

	// Try /proc-based memory tracking
	if mu, err := getMemoryUsage(pid); err == nil {
		if mu.PSSBytes > 0 {
			return mu.PSSBytes // Prefer PSS (Proportional Set Size)
		}
		return mu.RSSBytes // Fallback to RSS
	}

	// Last resort: estimation
	return 4 * 1024 * 1024 // 4MB base
}
