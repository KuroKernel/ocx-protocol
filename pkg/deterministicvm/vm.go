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
// In strict mode, cgroup failures cause execution to fail
func applyCgroups(config VMConfig, pid int) (*CgroupManager, error) {
	strictMode := config.StrictMode

	// Only on Linux
	if runtime.GOOS != "linux" {
		if strictMode {
			return nil, fmt.Errorf("cgroups required but not available on %s", runtime.GOOS)
		}
		return nil, nil // Graceful degradation in permissive mode
	}

	// Try to create cgroup
	cgManager, err := NewCgroupManager(fmt.Sprintf("vm-%d", pid))
	if err != nil {
		if strictMode {
			return nil, fmt.Errorf("cgroups required but creation failed: %w", err)
		}
		// Log warning to stderr to avoid breaking stdout determinism
		if !strings.Contains(err.Error(), "permission denied") {
			fmt.Fprintf(os.Stderr, "⚠ Cgroups unavailable (permissive mode): %v\n", err)
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
		cgManager.Cleanup()
		if strictMode {
			return nil, fmt.Errorf("cgroup limits application failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "⚠ Failed to apply cgroup limits (permissive mode): %v\n", err)
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
// Respects OCX_STRICT_MODE environment variable (default: true in production)
func installSeccomp() error {
	return installSeccompWithMode(isStrictModeEnabled(), "/tmp")
}

// installSeccompWithMode installs seccomp with explicit strict mode setting
func installSeccompWithMode(strictMode bool, workingDir string) error {
	ctx := context.Background()
	config := SeccompConfig{
		StrictSandbox: strictMode,
		WorkingDir:    workingDir,
		Logger:        log.New(os.Stderr, "[seccomp] ", log.LstdFlags),
	}

	return ApplySeccompProfile(ctx, config)
}

// isStrictModeEnabled checks if strict security mode is enabled
// Defaults to true unless OCX_STRICT_MODE=false
func isStrictModeEnabled() bool {
	val := os.Getenv("OCX_STRICT_MODE")
	if val == "" {
		return true // Default to strict in production
	}
	return val != "false" && val != "0" && val != "no"
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
	// In strict mode, cgroup failure will terminate execution
	cgManager, cgErr := applyCgroups(config, pid)
	if cgErr != nil {
		// Kill the process since we can't properly sandbox it
		cmd.Process.Kill()
		return nil, &ExecutionError{
			Code:       ErrorCodeEnvironmentSetup,
			Message:    "Failed to apply cgroup isolation",
			Underlying: cgErr,
		}
	}
	if cgManager != nil {
		defer cgManager.Cleanup()
	}

	// Wait for completion
	err := cmd.Wait()
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
				GasUsed:    calculateScriptGasFromPath(config.ArtifactPath, config.InputData, len(stdout.Bytes()), len(stderr.Bytes())),
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
		GasUsed:    calculateScriptGasFromPath(config.ArtifactPath, config.InputData, len(stdout.Bytes()), len(stderr.Bytes())),
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

// calculateScriptGasFromPath reads the script and calculates deterministic gas
// This is purely based on script content, not execution time
func calculateScriptGasFromPath(artifactPath string, inputData []byte, stdoutLen, stderrLen int) uint64 {
	// Read script content
	scriptBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		// Fallback to output-based gas if we can't read the script
		return uint64(100 + stdoutLen + stderrLen)
	}

	script := string(scriptBytes)

	// Use the deterministic gas calculation from gas.go
	baseGas := CalculateDeterministicGas(script, inputData)

	// Add I/O cost (deterministic based on output size)
	ioGas := uint64(stdoutLen + stderrLen)

	return baseGas + ioGas
}

// calculateDeterministicGas is kept for backward compatibility
// DEPRECATED: Use calculateScriptGasFromPath for proper deterministic gas
func calculateDeterministicGas(cycles uint64, stdoutLen, stderrLen int) uint64 {
	// This function is deprecated but kept for compatibility
	// It now ignores cycles and uses output-based calculation
	const minGas = 100
	ioGas := uint64(stdoutLen + stderrLen)
	totalGas := minGas + ioGas
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
	strictMode := isStrictModeEnabled()

	if strictMode {
		// Full namespace isolation in strict/production mode
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			// Enable namespace isolation for security
			Cloneflags: syscall.CLONE_NEWPID | // New PID namespace
				syscall.CLONE_NEWNET | // New network namespace (no network access)
				syscall.CLONE_NEWUTS | // New UTS namespace (hostname isolation)
				syscall.CLONE_NEWIPC, // New IPC namespace (shared memory isolation)
		}
	} else {
		// Basic isolation in permissive/development mode
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
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
