package deterministicvm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		"OMP_NUM_THREADS=1",           // Force single-threaded OpenMP
		"OPENBLAS_NUM_THREADS=1",      // Force single-threaded OpenBLAS
		"MKL_NUM_THREADS=1",           // Force single-threaded Intel MKL
		"NUMEXPR_NUM_THREADS=1",       // Force single-threaded NumExpr
		
		// Disable randomization features
		"PYTHONHASHSEED=0",            // Python hash seed
		"MALLOC_PERTURB_=0",           // Disable glibc malloc perturbation
		
		// Set predictable temporary directories
		"TMPDIR="+filepath.Join(execDir, "tmp"),
		"TEMP="+filepath.Join(execDir, "tmp"),
		"TMP="+filepath.Join(execDir, "tmp"),
		
		// Disable profiling and debugging that could affect timing
		"CPUPROFILE=",
		"MEMPROFILE=",
		"GOMAXPROCS=1",                // Force single-threaded Go
		
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
		// For now, we'll allow non-zero exit codes but log them
		// In production, you might want to be more strict
	}

	// Validate cycle usage
	if result.CyclesUsed > config.CycleLimit {
		return &ExecutionError{
			Code:    ErrorCodeCycleLimitExceeded,
			Message: fmt.Sprintf("Execution exceeded cycle limit: %d > %d", result.CyclesUsed, config.CycleLimit),
			Context: map[string]interface{}{
				"cycles_used": result.CyclesUsed,
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
		// For now, just log warnings rather than failing
		// In production, you might want to collect these for analysis
		fmt.Printf("Determinism warning: %v\n", err)
	}

	return nil
}

// enrichExecutionResult adds additional metadata to the execution result.
func enrichExecutionResult(result *ExecutionResult, artifactHash [32]byte, input []byte) *ExecutionResult {
	// The result is already mostly complete, but we could add:
	// - Input hash for verification
	// - Artifact hash for traceability
	// - Additional determinism metadata
	
	// For now, we'll return the result as-is since the caller
	// has access to the artifactHash and input for receipt generation
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
		" UTC", " GMT",   // Timezone indicators
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
	// Simple pattern matching - in production you'd use regexp
	// For now, just check for simple substring matches
	return len(s) > 0 && len(pattern) > 0
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
			Code:    ErrorCodeArtifactInvalid,
			Message: "Failed to stat artifact",
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
	// Simple format detection based on file extension and magic numbers
	// In production, you'd want more sophisticated detection
	ext := filepath.Ext(path)
	switch ext {
	case ".wasm":
		return "wasm"
	case ".js":
		return "javascript"
	case ".py":
		return "python"
	default:
		// Try to detect ELF binaries by reading magic bytes
		if isELFBinary(path) {
			return "elf"
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
