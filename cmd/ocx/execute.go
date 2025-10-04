package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
)

var (
	outputReceipt string
	enableSeccomp bool
	strictMode    bool
	timeout       int
	workingDir    string
	envVars       []string
)

var executeCmd = &cobra.Command{
	Use:   "execute <artifact>",
	Short: "Execute an artifact in the deterministic VM",
	Long: `Execute an artifact in the OCX deterministic virtual machine.
	
The artifact will be executed in an isolated, deterministic environment.
Optionally generate a cryptographic receipt proving the execution.`,
	Args: cobra.ExactArgs(1),
	RunE: runExecute,
}

func init() {
	// Execute command flags
	executeCmd.Flags().StringVarP(&outputReceipt, "output", "o", "", "Output receipt file path")
	executeCmd.Flags().BoolVar(&enableSeccomp, "seccomp", false, "Enable seccomp sandboxing")
	executeCmd.Flags().BoolVar(&strictMode, "strict", false, "Strict seccomp mode (fail if unavailable)")
	executeCmd.Flags().IntVar(&timeout, "timeout", 30, "Execution timeout in seconds")
	executeCmd.Flags().StringVar(&workingDir, "workdir", "", "Working directory for execution")
	executeCmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variables (format: KEY=VALUE)")

	// Add command to root
	rootCmd.AddCommand(executeCmd)
}

func runExecute(cmd *cobra.Command, args []string) error {
	artifactPath := args[0]

	// Verify artifact exists
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("artifact not found: %w", err)
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Create VM config for direct execution
	config := deterministicvm.VMConfig{
		ArtifactPath: artifactPath,
		InputData:    []byte{},
		WorkingDir:   workingDir,
		Timeout:      time.Duration(timeout) * time.Second,
		CycleLimit:   1000000000,        // 1 billion cycles
		MemoryLimit:  100 * 1024 * 1024, // 100MB
		Env:          parseEnvVars(envVars),
		Network:      false,
		StrictMode:   enableSeccomp && strictMode,
	}

	// Execute artifact
	fmt.Fprintf(os.Stderr, "Executing: %s\n", artifactPath)
	if enableSeccomp {
		fmt.Fprintf(os.Stderr, "Seccomp: enabled (strict=%v)\n", strictMode)
	} else {
		fmt.Fprintf(os.Stderr, "Seccomp: disabled\n")
	}

	// Use deterministic execution path
	// First, create a hash of the artifact for deterministic execution
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	artifactHash := sha256.Sum256(artifactData)

	// Cache the artifact in temp directory for resolution
	hashStr := fmt.Sprintf("%x", artifactHash)
	tempCachePath := filepath.Join(os.TempDir(), hashStr)
	if err := os.WriteFile(tempCachePath, artifactData, 0755); err != nil {
		return fmt.Errorf("failed to cache artifact: %w", err)
	}
	defer os.Remove(tempCachePath) // Clean up after execution

	// Execute using the deterministic path
	result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, config.InputData)
	if err != nil {
		// Check if this is a seccomp error (expected behavior)
		if enableSeccomp && isSeccompError(err) {
			fmt.Fprintf(os.Stderr, "⚠️  Execution blocked by seccomp (security working correctly): %v\n", err)
			result = &deterministicvm.ExecutionResult{
				ExitCode: 159, // SIGSYS
				Stderr:   []byte(fmt.Sprintf("Killed by seccomp: %v", err)),
			}
		} else {
			return fmt.Errorf("execution failed: %w", err)
		}
	}

	// Output results (only to stderr to avoid affecting determinism)
	fmt.Fprintf(os.Stderr, "Exit code: %d\n", result.ExitCode)
	// For deterministic execution, report logical time (gas) instead of wall-clock time
	fmt.Fprintf(os.Stderr, "Logical time: %d gas units\n", result.GasUsed)
	fmt.Fprintf(os.Stderr, "Wall-clock duration: %v\n", result.Duration)

	// Print stdout/stderr
	if len(result.Stdout) > 0 {
		fmt.Fprintf(os.Stderr, "\n--- STDOUT ---\n")
		os.Stdout.Write(result.Stdout)
	}

	if len(result.Stderr) > 0 {
		fmt.Fprintf(os.Stderr, "\n--- STDERR ---\n")
		os.Stderr.Write(result.Stderr)
	}

	// Generate receipt if requested
	if outputReceipt != "" {
		if err := generateReceipt(artifactPath, result, outputReceipt); err != nil {
			return fmt.Errorf("failed to generate receipt: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\n✓ Receipt written to: %s\n", outputReceipt)
	}

	// Exit with same code as artifact
	os.Exit(result.ExitCode)
	return nil
}

func setupArtifactCache(artifactPath string) ([32]byte, error) {
	// Read artifact content
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to read artifact: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(data)

	// Set up cache directory in /tmp
	cacheDir := "/tmp/ocx-cache"
	os.MkdirAll(cacheDir, 0755)
	hashStr := fmt.Sprintf("%x", hash)
	cachePath := filepath.Join(cacheDir, hashStr)

	// Copy to cache location if not exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		if err := os.WriteFile(cachePath, data, 0755); err != nil {
			return [32]byte{}, fmt.Errorf("failed to cache artifact: %w", err)
		}
	}

	return hash, nil
}

func generateReceipt(artifactPath string, result *deterministicvm.ExecutionResult, outputPath string) error {
	// Load signing key from environment or generate random key
	var privateKey ed25519.PrivateKey
	var err error

	if os.Getenv("OCX_SIGNING_KEY_PEM") != "" {
		// Load from PEM file
		signer, err := keystore.LoadSignerFromEnv()
		if err != nil {
			return fmt.Errorf("failed to load signer from env: %w", err)
		}
		// Get the public key to verify the signer loaded correctly
		pubKey, err := signer.PublicKey(context.Background(), "default")
		if err != nil {
			return fmt.Errorf("failed to get public key from signer: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Loaded signer with public key: %x\n", pubKey)

		// Extract the private key from the PEM file directly
		// since the signer interface doesn't expose it
		privateKey, err = loadPrivateKeyFromPEM(os.Getenv("OCX_SIGNING_KEY_PEM"))
		if err != nil {
			return fmt.Errorf("failed to load private key from PEM: %w", err)
		}
	} else {
		// Generate random key (fallback)
		_, privateKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
	}

	// Create receipt generator
	generator := receipt.NewGenerator(privateKey, "ocx-cli")

	// Generate receipt
	rcpt, err := generator.Generate(
		generateTxID(),
		computeArtifactID(artifactPath),
		generateExecutionID(),
		result.ExitCode,
		uint64(result.GasUsed),
		result.Stdout,
		result.Stderr,
		receipt.Resource{
			CPUTimeMs:      result.Duration.Milliseconds(),
			MemoryBytes:    int64(result.MemoryUsed),
			DiskReadBytes:  0, // Not available in ExecutionResult
			DiskWriteBytes: 0, // Not available in ExecutionResult
		},
		[]string{}, // Environment not available in ExecutionResult
		result.StartTime,
		result.EndTime,
		map[string]string{
			"artifact":      filepath.Base(artifactPath),
			"deterministic": "true",
			"version":       "OCXv1",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to generate receipt: %w", err)
	}

	// Encode to standalone format (compatible with standalone verifier)
	data, err := generator.EncodeStandaloneFormat(rcpt)
	if err != nil {
		return fmt.Errorf("failed to encode receipt: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write receipt: %w", err)
	}

	// Also output JSON for debugging
	jsonPath := outputPath + ".json"
	jsonData, _ := json.MarshalIndent(rcpt, "", "  ")
	os.WriteFile(jsonPath, jsonData, 0644)

	return nil
}

func parseEnvVars(envs []string) []string {
	result := make([]string, 0, len(envs))
	for _, env := range envs {
		result = append(result, env)
	}
	return result
}

func isSeccompError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	patterns := []string{
		"bad system call",
		"operation not permitted",
		"killed by seccomp",
		"SIGSYS",
		"seccomp",
	}
	for _, pattern := range patterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// loadPrivateKeyFromPEM loads an Ed25519 private key from a PEM file
func loadPrivateKeyFromPEM(pemPath string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PEM file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ed25519Key, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not Ed25519")
	}

	return ed25519Key, nil
}

func generateTxID() string {
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}

func computeArtifactID(path string) string {
	// Future enhancement: compute actual hash
	return fmt.Sprintf("artifact_%s", filepath.Base(path))
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
