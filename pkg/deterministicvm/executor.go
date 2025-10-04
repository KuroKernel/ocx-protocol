package deterministicvm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ExecuteArtifact is the main entry point for deterministic artifact execution.
// It orchestrates the complete execution pipeline to ensure deterministic results.
func ExecuteArtifact(ctx context.Context, artifactHash [32]byte, input []byte) (*ExecutionResult, error) {
	// 1. RESOLVE ARTIFACT: Fetch the artifact from cache based on its hash
	artifactPath, err := resolveArtifactFromHash(artifactHash)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve artifact: %w", err)
	}

	// 2. PREPARE ENVIRONMENT: Create a temporary, deterministic execution environment
	execDir, cleanup, err := prepareExecutionEnvironment(artifactPath, input)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare environment: %w", err)
	}
	defer cleanup() // Ensure cleanup on any error or success

	// 3. VALIDATE ENVIRONMENT: Ensure the environment is properly set up
	if err := validateEnvironment(execDir); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	// 4. SET DETERMINISTIC FLAGS: Configure the VM for deterministic execution
	config := createDeterministicConfig(execDir, input)

	// 5. EXECUTE: Run the artifact inside the isolated VM
	result, err := defaultVM.Run(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("VM execution failed: %w", err)
	}

	// 6. VALIDATE RESULT: Check for common determinism pitfalls
	if err := validateExecutionResult(result, config); err != nil {
		return nil, fmt.Errorf("execution validation failed: %w", err)
	}

	// 7. ENRICH RESULT: Add additional metadata for receipts
	result = enrichExecutionResult(result, artifactHash, input)

	// 8. GENERATE EVIDENCE: Create audit trail for the execution
	generateExecutionEvidence(artifactHash, input, result, &config)

	return result, nil
}

// ExecuteArtifactWithConfig provides more control over execution parameters.
// This is useful for testing or when specific configurations are required.
func ExecuteArtifactWithConfig(ctx context.Context, artifactHash [32]byte, input []byte, customConfig *VMConfig) (*ExecutionResult, error) {
	// Resolve artifact
	artifactPath, err := resolveArtifactFromHash(artifactHash)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve artifact: %w", err)
	}

	// Prepare environment
	execDir, cleanup, err := prepareExecutionEnvironment(artifactPath, input)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare environment: %w", err)
	}
	defer cleanup()

	// Validate environment
	if err := validateEnvironment(execDir); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	// Use custom config or create default
	var config VMConfig
	if customConfig != nil {
		config = *customConfig
		// Override critical paths
		config.ArtifactPath = filepath.Join(execDir, "artifact")
		config.WorkingDir = execDir
		config.InputData = input
	} else {
		config = createDeterministicConfig(execDir, input)
	}

	// Execute
	result, err := defaultVM.Run(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("VM execution failed: %w", err)
	}

	// Validate result
	if err := validateExecutionResult(result, config); err != nil {
		return nil, fmt.Errorf("execution validation failed: %w", err)
	}

	// Enrich result
	result = enrichExecutionResult(result, artifactHash, input)

	return result, nil
}

// createDeterministicConfig builds a VMConfig with all deterministic settings.
func createDeterministicConfig(execDir string, input []byte) VMConfig {
	config := DefaultVMConfig()

	// Set execution-specific paths
	config.ArtifactPath = filepath.Join(execDir, "artifact")
	config.WorkingDir = execDir
	config.InputData = input

	// Enhance environment variables for maximum determinism
	config.Env = append(config.Env,
		// Disable hardware-specific optimizations that could affect determinism
		"OMP_NUM_THREADS=1",      // Force single-threaded OpenMP
		"OPENBLAS_NUM_THREADS=1", // Force single-threaded OpenBLAS
		"MKL_NUM_THREADS=1",      // Force single-threaded Intel MKL
		"NUMEXPR_NUM_THREADS=1",  // Force single-threaded NumExpr

		// Disable randomization features
		"PYTHONHASHSEED=0",  // Python hash seed
		"MALLOC_PERTURB_=0", // Disable glibc malloc perturbation

		// Set predictable temporary directories
		"TMPDIR="+filepath.Join(execDir, "tmp"),
		"TEMP="+filepath.Join(execDir, "tmp"),
		"TMP="+filepath.Join(execDir, "tmp"),

		// Disable profiling and debugging that could affect timing
		"CPUPROFILE=",
		"MEMPROFILE=",
		"GOMAXPROCS=1", // Force single-threaded Go

		// Set predictable cache directories
		"XDG_CACHE_HOME="+filepath.Join(execDir, "tmp", "cache"),
		"XDG_DATA_HOME="+filepath.Join(execDir, "tmp", "data"),
		"XDG_CONFIG_HOME="+filepath.Join(execDir, "tmp", "config"),
	)

	return config
}

// validateExecutionResult checks the execution result for common issues.
func validateExecutionResult(result *ExecutionResult, config VMConfig) error {
	// Check for successful execution (unless we expect failures)
	if result.ExitCode != 0 {
		// Log non-zero exit codes for debugging
		log.Printf("Warning: Process exited with code %d", result.ExitCode)

		// In strict mode, treat non-zero exit codes as failures
		if config.StrictMode {
			return &ExecutionError{
				Code:    ErrorCodeExecutionFailed,
				Message: fmt.Sprintf("Process failed with exit code %d (strict mode enabled)", result.ExitCode),
				Context: map[string]interface{}{
					"exit_code":   result.ExitCode,
					"stdout":      result.Stdout,
					"stderr":      result.Stderr,
					"strict_mode": true,
				},
			}
		}
	}

	// Validate cycle usage (using host cycles for limit checking)
	if result.HostCycles > config.CycleLimit {
		return &ExecutionError{
			Code:    ErrorCodeCycleLimitExceeded,
			Message: fmt.Sprintf("Execution exceeded cycle limit: %d > %d", result.HostCycles, config.CycleLimit),
			Context: map[string]interface{}{
				"cycles_used": result.HostCycles,
				"cycle_limit": config.CycleLimit,
			},
		}
	}

	// Validate memory usage
	if result.MemoryUsed > config.MemoryLimit {
		return &ExecutionError{
			Code:    ErrorCodeMemoryLimitExceeded,
			Message: fmt.Sprintf("Execution exceeded memory limit: %d > %d bytes", result.MemoryUsed, config.MemoryLimit),
			Context: map[string]interface{}{
				"memory_used":  result.MemoryUsed,
				"memory_limit": config.MemoryLimit,
			},
		}
	}

	// Validate execution time
	if result.Duration > config.Timeout {
		return &ExecutionError{
			Code:    ErrorCodeTimeout,
			Message: fmt.Sprintf("Execution exceeded timeout: %v > %v", result.Duration, config.Timeout),
			Context: map[string]interface{}{
				"duration": result.Duration,
				"timeout":  config.Timeout,
			},
		}
	}

	// Check for suspicious patterns that could indicate non-determinism
	if err := checkDeterminismWarnings(result); err != nil {
		// Log determinism warnings for analysis
		log.Printf("Determinism warning: %v", err)

		// In strict mode, treat determinism warnings as failures
		if config.StrictMode {
			return &ExecutionError{
				Code:    ErrorCodeNonDeterministic,
				Message: fmt.Sprintf("Non-deterministic execution detected (strict mode enabled): %v", err),
				Context: map[string]interface{}{
					"warning":     err.Error(),
					"stdout":      result.Stdout,
					"stderr":      result.Stderr,
					"strict_mode": true,
				},
			}
		}
	}

	return nil
}

// enrichExecutionResult adds additional metadata to the execution result.
func enrichExecutionResult(result *ExecutionResult, artifactHash [32]byte, input []byte) *ExecutionResult {
	// Calculate deterministic gas for the execution
	// This replaces the variable GasUsed with deterministic GasUsed

	// Get the artifact content to calculate deterministic gas
	artifactPath, err := resolveArtifactFromHash(artifactHash)
	if err != nil {
		// Fallback to a reasonable default if we can't read the artifact
		result.GasUsed = 1000
		return result
	}

	// Read artifact content
	artifactContent, err := os.ReadFile(artifactPath)
	if err != nil {
		// Fallback to a reasonable default
		result.GasUsed = 1000
		return result
	}

	// Calculate deterministic gas based on artifact content and input
	result.GasUsed = CalculateDeterministicGas(string(artifactContent), input)

	// Keep the original cycles as HostCycles for diagnostics
	// (HostCycles is already set by the VM execution)

	// Add host information
	result.HostInfo = getHostInfo()

	return result
}

// checkDeterminismWarnings looks for patterns that might indicate non-deterministic behavior.
func checkDeterminismWarnings(result *ExecutionResult) error {
	// Check for common non-deterministic patterns in output
	warnings := []string{}

	// Convert output to string for pattern matching
	stdout := string(result.Stdout)
	stderr := string(result.Stderr)

	// Check for timestamp patterns (basic heuristic)
	timestampPatterns := []string{
		"2024-", "2025-", // Year patterns
		" UTC", " GMT", // Timezone indicators
		"T00:", "T01:", "T02:", "T03:", "T04:", "T05:", // Hour patterns
		"timestamp", "time", "date", // Common time-related words
	}

	for _, pattern := range timestampPatterns {
		if containsIgnoreCase(stdout, pattern) || containsIgnoreCase(stderr, pattern) {
			warnings = append(warnings, fmt.Sprintf("Potential timestamp in output: %s", pattern))
		}
	}

	// Check for memory addresses (which should be ASLR-disabled but worth checking)
	if containsPattern(stdout, `0x[a-fA-F0-9]{8,}`) || containsPattern(stderr, `0x[a-fA-F0-9]{8,}`) {
		warnings = append(warnings, "Potential memory address in output")
	}

	// Check for process IDs
	if containsPattern(stdout, `pid[:\s]+\d+`) || containsPattern(stderr, `pid[:\s]+\d+`) {
		warnings = append(warnings, "Potential process ID in output")
	}

	if len(warnings) > 0 {
		return &ExecutionError{
			Code:    ErrorCodeUnknown, // This is just a warning
			Message: fmt.Sprintf("Determinism warnings: %v", warnings),
			Context: map[string]interface{}{
				"warnings": warnings,
			},
		}
	}

	return nil
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		containsPattern(s, `(?i)`+substr)
}

func containsPattern(s, pattern string) bool {
	// Use regexp for proper pattern matching
	matched, err := regexp.MatchString(pattern, s)
	if err != nil {
		// If regex fails, fall back to simple substring matching
		return strings.Contains(strings.ToLower(s), strings.ToLower(pattern))
	}
	return matched
}

// GetArtifactInfo retrieves metadata about an artifact without executing it.
func GetArtifactInfo(artifactHash [32]byte) (*ArtifactInfo, error) {
	artifactPath, err := resolveArtifactFromHash(artifactHash)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(artifactPath)
	if err != nil {
		return nil, &ExecutionError{
			Code:       ErrorCodeArtifactInvalid,
			Message:    "Failed to stat artifact",
			Underlying: err,
		}
	}

	return &ArtifactInfo{
		Hash:       artifactHash,
		Path:       artifactPath,
		Size:       info.Size(),
		Executable: isExecutable(info.Mode()),
		Format:     detectArtifactFormat(artifactPath),
	}, nil
}

// detectArtifactFormat attempts to identify the artifact type.
func detectArtifactFormat(path string) string {
	// Format detection based on file extension and magic numbers
	// Supports: WASM, JavaScript, Python, ELF binaries, and shell scripts
	ext := filepath.Ext(path)
	switch ext {
	case ".wasm":
		return "wasm"
	case ".js":
		return "javascript"
	case ".py":
		return "python"
	case ".sh":
		return "shell"
	case ".bash":
		return "bash"
	case ".zsh":
		return "zsh"
	case ".fish":
		return "fish"
	case ".exe":
		return "windows_executable"
	case ".dll":
		return "windows_library"
	case ".so":
		return "shared_object"
	case ".dylib":
		return "macos_library"
	default:
		// Try to detect binary formats by reading magic bytes
		if isELFBinary(path) {
			return "elf"
		}
		if isWASMBinaryFile(path) {
			return "wasm"
		}
		if isShellScript(path) {
			return "shell"
		}
		return "unknown"
	}
}

func isELFBinary(path string) bool {
	// Check for ELF magic number (0x7f 'E' 'L' 'F')
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return false
	}

	return magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
}

// isWASMBinaryFile checks if the file is a WebAssembly binary
func isWASMBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return false
	}

	// WASM magic number: 0x00 0x61 0x73 0x6d (".wasm")
	return magic[0] == 0x00 && magic[1] == 0x61 && magic[2] == 0x73 && magic[3] == 0x6d
}

// isShellScript checks if the file is a shell script by reading the shebang
func isShellScript(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first line to check for shebang
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil || n < 2 {
		return false
	}

	firstLine := string(buffer[:n])
	// Check for common shell shebangs
	shellShebangs := []string{
		"#!/bin/sh",
		"#!/bin/bash",
		"#!/bin/zsh",
		"#!/bin/fish",
		"#!/usr/bin/env sh",
		"#!/usr/bin/env bash",
		"#!/usr/bin/env zsh",
		"#!/usr/bin/env fish",
	}

	for _, shebang := range shellShebangs {
		if strings.HasPrefix(firstLine, shebang) {
			return true
		}
	}

	return false
}

// getHostInfo collects variable host system information for diagnostics
func getHostInfo() HostInfo {
	info := HostInfo{
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}

	// Try to get CPU model
	if cpuInfo, err := exec.Command("uname", "-m").Output(); err == nil {
		info.CPUModel = strings.TrimSpace(string(cpuInfo))
	}

	// Try to get kernel version
	if kernelInfo, err := exec.Command("uname", "-r").Output(); err == nil {
		info.KernelVer = strings.TrimSpace(string(kernelInfo))
	}

	// Try to get load average
	if loadInfo, err := exec.Command("uptime").Output(); err == nil {
		info.LoadAvg = strings.TrimSpace(string(loadInfo))
	}

	return info
}

// generateExecutionEvidence creates evidence for the execution
func generateExecutionEvidence(artifactHash [32]byte, input []byte, result *ExecutionResult, config *VMConfig) {
	// Create execution receipt for evidence generation
	receipt := &OCXReceipt{
		SpecHash:     [32]byte{}, // Would be set by caller
		ArtifactHash: artifactHash,
		InputHash:    sha256.Sum256(input),
		OutputHash:   sha256.Sum256(result.Stdout),
		GasUsed:      result.GasUsed,
		HostCycles:   result.HostCycles,
		StartedAt:    uint64(result.StartTime.Unix()),
		FinishedAt:   uint64(result.EndTime.Unix()),
		DurationMs:   uint64(result.Duration.Milliseconds()),
		MemoryUsed:   result.MemoryUsed,
		HostInfo:     result.HostInfo,
		IssuerID:     "ocx-dmvm-v1",
		Signature:    []byte{}, // Would be signed by caller
	}

	// Generate receipt hash
	receiptHash := fmt.Sprintf("sha256:%x", sha256.Sum256(receipt.toCanonicalCBOR()))

	// Create evidence
	evidence := CreateEvidence(
		fmt.Sprintf("artifact_%x", artifactHash),
		receiptHash,
		fmt.Sprintf("0x%08x", config.Seed),
		*config,
	)

	// Emit evidence
	emitEvidence(evidence)
}

// ExecutorConfig holds executor configuration
type ExecutorConfig struct {
	WorkingDir    string
	EnableSeccomp bool
	StrictMode    bool
	Timeout       int
	Environment   []string
}

// Executor executes artifacts deterministically
type Executor struct {
	config ExecutorConfig
}

// NewExecutor creates a new executor
func NewExecutor(config ExecutorConfig) (*Executor, error) {
	if config.WorkingDir == "" {
		config.WorkingDir = "/tmp"
	}

	return &Executor{
		config: config,
	}, nil
}

// Execute runs an artifact and returns results
func (e *Executor) Execute(ctx context.Context, artifactPath string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Apply seccomp if enabled
	if e.config.EnableSeccomp {
		seccompCfg := SeccompConfig{
			StrictSandbox: e.config.StrictMode,
			WorkingDir:    e.config.WorkingDir,
		}

		if err := ApplySeccompProfile(ctx, seccompCfg); err != nil {
			if e.config.StrictMode {
				return nil, fmt.Errorf("seccomp required but unavailable: %w", err)
			}
			// Continue without seccomp in non-strict mode
		}
	}

	// Execute artifact
	cmd := exec.CommandContext(ctx, artifactPath)
	cmd.Dir = e.config.WorkingDir
	cmd.Env = e.config.Environment

	stdout, err := cmd.Output()
	stderr := []byte{}
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			stderr = exitErr.Stderr
		} else {
			return nil, fmt.Errorf("execution failed: %w", err)
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	result := &ExecutionResult{
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		Duration:   duration,
		GasUsed:    uint64(calculateGasUsed(duration, len(stdout), len(stderr))),
		MemoryUsed: uint64(len(stdout) + len(stderr)),
		StartTime:  startTime,
		EndTime:    endTime,
		HostInfo: HostInfo{
			Platform: runtime.GOOS + "/" + runtime.GOARCH,
		},
	}

	return result, nil
}

// calculateGasUsed computes gas consumption
func calculateGasUsed(duration time.Duration, stdoutLen, stderrLen int) int64 {
	// Simplified gas calculation
	// 1 gas per millisecond + 1 gas per KB of output
	timeGas := duration.Milliseconds()
	outputGas := int64((stdoutLen + stderrLen) / 1024)
	return timeGas + outputGas
}
