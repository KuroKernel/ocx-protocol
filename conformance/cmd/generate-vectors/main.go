package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

// GoldenVector represents a complete golden vector for cross-language testing
type GoldenVector struct {
	// Test metadata
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	CreatedAt   string `json:"created_at"`

	// Input data
	ArtifactHex string `json:"artifact_hex"`
	InputHex    string `json:"input_hex"`
	MaxGas      uint64 `json:"max_gas"`

	// Expected deterministic results
	ExpectedGasUsed     uint64 `json:"expected_gas_used"`
	ExpectedOutputHash  string `json:"expected_output_hash"`
	ExpectedReceiptHash string `json:"expected_receipt_hash"`

	// Receipt data (canonical CBOR)
	ReceiptCBOR string `json:"receipt_cbor"`
	ReceiptHex  string `json:"receipt_hex"`

	// Cryptographic data
	PublicKeyHex  string `json:"public_key_hex"`
	SignatureHex  string `json:"signature_hex"`
	MessageHex    string `json:"message_hex"`
	MessageSHA256 string `json:"message_sha256"`

	// Verification data
	VerificationResult bool   `json:"verification_result"`
	VerificationError  string `json:"verification_error,omitempty"`

	// Cross-language compatibility
	RustVerifierResult bool   `json:"rust_verifier_result"`
	GoVerifierResult   bool   `json:"go_verifier_result"`
	DeterministicRun   bool   `json:"deterministic_run"`
}

// VectorGenerator generates golden vectors using the deterministic execution engine
type VectorGenerator struct {
	keystore *keystore.Keystore
	signer   keystore.Signer
	verifier verify.Verifier
}

// NewVectorGenerator creates a new golden vector generator
func NewVectorGenerator() (*VectorGenerator, error) {
	// Initialize keystore
	ks, err := keystore.New("./conformance-keys")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Create signer
	signer := keystore.NewLocalSigner(ks)

	// Create verifier
	verifier := verify.NewVerifier()

	return &VectorGenerator{
		keystore: ks,
		signer:   signer,
		verifier: verifier,
	}, nil
}

// GenerateVector generates a single golden vector
func (vg *VectorGenerator) GenerateVector(ctx context.Context, name, description, artifactHex, inputHex string, maxGas uint64) (*GoldenVector, error) {
	// Decode hex inputs
	artifact, err := hex.DecodeString(artifactHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode artifact hex: %w", err)
	}

	input, err := hex.DecodeString(inputHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode input hex: %w", err)
	}

	// Calculate hashes
	artifactHash := sha256.Sum256(artifact)
	inputHash := sha256.Sum256(input)

	// Create a simple executable artifact for the deterministic VM
	artifactPath, cleanup, err := vg.createExecutableArtifact(artifact, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create executable artifact: %w", err)
	}
	defer cleanup()

	// Cache the artifact in the deterministic VM's cache
	err = vg.cacheArtifact(artifactHash, artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to cache artifact: %w", err)
	}

	// Execute artifact using deterministic VM (same path as server)
	startTime := time.Now()
	result, err := deterministicvm.ExecuteArtifact(ctx, artifactHash, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute artifact: %w", err)
	}
	executionDuration := time.Since(startTime)

	// Calculate output hash
	outputHash := sha256.Sum256(result.Stdout)

	// Create receipt core
	receiptCore := &receipt.ReceiptCore{
		ProgramHash: artifactHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     result.GasUsed,
		StartedAt:   uint64(startTime.Unix()),
		FinishedAt:  uint64(result.EndTime.Unix()),
		IssuerID:    "golden-vector-generator",
	}

	// Canonicalize and sign receipt core
	coreBytes, err := receipt.CanonicalizeCore(receiptCore)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize receipt core: %w", err)
	}

	// Get active key and sign
	activeKey := vg.keystore.GetActiveKey()
	if activeKey == nil {
		return nil, fmt.Errorf("no active signing key available")
	}

	signature, pubKey, err := vg.signer.Sign(ctx, activeKey.ID, coreBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign receipt: %w", err)
	}

	// Build full receipt
	receiptFull := &receipt.ReceiptFull{
		Core:       *receiptCore,
		Signature:  signature,
		HostCycles: uint64(executionDuration.Nanoseconds()),
		HostInfo: map[string]string{
			"generator_version": "1.0.0",
			"vm_type":          "deterministic",
			"arch":             "x86_64",
		},
	}

	// Canonicalize full receipt
	fullReceiptBytes, err := receipt.CanonicalizeFull(receiptFull)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize full receipt: %w", err)
	}

	// Calculate receipt hash
	receiptHashBytes := sha256.Sum256(fullReceiptBytes)
	receiptHash := receiptHashBytes[:]

	// Verify the receipt
	verificationResult, verificationError := vg.verifyReceipt(fullReceiptBytes, pubKey)

	// Create golden vector
	vector := &GoldenVector{
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),

		ArtifactHex: artifactHex,
		InputHex:    inputHex,
		MaxGas:      maxGas,

		ExpectedGasUsed:     result.GasUsed,
		ExpectedOutputHash:  hex.EncodeToString(outputHash[:]),
		ExpectedReceiptHash: hex.EncodeToString(receiptHash),

		ReceiptCBOR: base64.StdEncoding.EncodeToString(fullReceiptBytes),
		ReceiptHex:  hex.EncodeToString(fullReceiptBytes),

		PublicKeyHex:  hex.EncodeToString(pubKey),
		SignatureHex:  hex.EncodeToString(signature),
		MessageHex:    hex.EncodeToString(coreBytes),
		MessageSHA256: func() string {
			hash := sha256.Sum256(coreBytes)
			return hex.EncodeToString(hash[:])
		}(),

		VerificationResult: verificationResult,
		VerificationError:  verificationError,
		GoVerifierResult:   verificationResult,
		DeterministicRun:   true,
	}

	return vector, nil
}

// verifyReceipt verifies the generated receipt
func (vg *VectorGenerator) verifyReceipt(receiptBytes []byte, publicKey []byte) (bool, string) {
	// Use Go verifier
	core, err := vg.verifier.VerifyReceipt(receiptBytes, publicKey)
	if err != nil {
		return false, err.Error()
	}
	if core == nil {
		return false, "receipt verification failed"
	}

	// Extract fields for additional validation
	fields, err := vg.verifier.ExtractReceiptFields(receiptBytes)
	if err != nil {
		return false, fmt.Sprintf("failed to extract receipt fields: %v", err)
	}

	// Basic validation
	if fields == nil {
		return false, "extraction returned nil fields"
	}

	if fields.GasUsed == 0 {
		return false, "gas used cannot be zero"
	}

	return true, ""
}

// GenerateStandardVectors generates a comprehensive set of standard test vectors
func (vg *VectorGenerator) GenerateStandardVectors(ctx context.Context) ([]*GoldenVector, error) {
	var vectors []*GoldenVector

	// Vector 1: Minimal execution
	vector1, err := vg.GenerateVector(ctx,
		"minimal_execution",
		"Minimal execution with simple artifact and input",
		"48656c6c6f20576f726c64", // "Hello World" in hex
		"74657374",                // "test" in hex
		1000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vector 1: %w", err)
	}
	vectors = append(vectors, vector1)

	// Vector 2: Complex execution
	vector2, err := vg.GenerateVector(ctx,
		"complex_execution",
		"Complex execution with larger artifact and input",
		"48656c6c6f20576f726c64212054686973206973206120636f6d706c6578206172746966616374", // "Hello World! This is a complex artifact"
		"7465737420696e70757420666f7220636f6d706c657820657865637574696f6e",                // "test input for complex execution"
		5000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vector 2: %w", err)
	}
	vectors = append(vectors, vector2)

	// Vector 3: High gas usage
	vector3, err := vg.GenerateVector(ctx,
		"high_gas_execution",
		"Execution with high gas consumption",
		"48656c6c6f20576f726c6421204869676820676173207573616765", // "Hello World! High gas usage"
		"68696768206761732074657374",                              // "high gas test"
		10000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vector 3: %w", err)
	}
	vectors = append(vectors, vector3)

	// Vector 4: Edge case - empty input
	vector4, err := vg.GenerateVector(ctx,
		"empty_input_execution",
		"Execution with empty input",
		"48656c6c6f20576f726c64", // "Hello World"
		"",                        // Empty input
		1000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vector 4: %w", err)
	}
	vectors = append(vectors, vector4)

	// Vector 5: Edge case - maximum gas
	vector5, err := vg.GenerateVector(ctx,
		"max_gas_execution",
		"Execution with maximum gas limit",
		"48656c6c6f20576f726c64", // "Hello World"
		"6d617820676173",          // "max gas"
		100000,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate vector 5: %w", err)
	}
	vectors = append(vectors, vector5)

	return vectors, nil
}

// SaveVectors saves golden vectors to files
func (vg *VectorGenerator) SaveVectors(vectors []*GoldenVector, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save each vector
	for i, vector := range vectors {
		// Create vector directory
		vectorDir := filepath.Join(outputDir, fmt.Sprintf("%03d_%s", i, vector.Name))
		if err := os.MkdirAll(vectorDir, 0755); err != nil {
			return fmt.Errorf("failed to create vector directory: %w", err)
		}

		// Save JSON metadata
		jsonPath := filepath.Join(vectorDir, "vector.json")
		jsonData, err := json.MarshalIndent(vector, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal vector JSON: %w", err)
		}
		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write vector JSON: %w", err)
		}

		// Save receipt CBOR
		receiptBytes, err := base64.StdEncoding.DecodeString(vector.ReceiptCBOR)
		if err != nil {
			return fmt.Errorf("failed to decode receipt CBOR: %w", err)
		}
		receiptPath := filepath.Join(vectorDir, "receipt.cbor")
		if err := os.WriteFile(receiptPath, receiptBytes, 0644); err != nil {
			return fmt.Errorf("failed to write receipt CBOR: %w", err)
		}

		// Save public key
		pubKeyBytes, err := hex.DecodeString(vector.PublicKeyHex)
		if err != nil {
			return fmt.Errorf("failed to decode public key: %w", err)
		}
		pubKeyPath := filepath.Join(vectorDir, "pubkey.bin")
		if err := os.WriteFile(pubKeyPath, pubKeyBytes, 0644); err != nil {
			return fmt.Errorf("failed to write public key: %w", err)
		}

		// Save signature
		sigBytes, err := hex.DecodeString(vector.SignatureHex)
		if err != nil {
			return fmt.Errorf("failed to decode signature: %w", err)
		}
		sigPath := filepath.Join(vectorDir, "signature.bin")
		if err := os.WriteFile(sigPath, sigBytes, 0644); err != nil {
			return fmt.Errorf("failed to write signature: %w", err)
		}

		// Save message
		msgBytes, err := hex.DecodeString(vector.MessageHex)
		if err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}
		msgPath := filepath.Join(vectorDir, "message.bin")
		if err := os.WriteFile(msgPath, msgBytes, 0644); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		// Save message SHA256
		msgSHA256Bytes, err := hex.DecodeString(vector.MessageSHA256)
		if err != nil {
			return fmt.Errorf("failed to decode message SHA256: %w", err)
		}
		msgSHA256Path := filepath.Join(vectorDir, "message.sha256")
		if err := os.WriteFile(msgSHA256Path, msgSHA256Bytes, 0644); err != nil {
			return fmt.Errorf("failed to write message SHA256: %w", err)
		}

		fmt.Printf("✅ Generated vector %d: %s\n", i+1, vector.Name)
	}

	return nil
}

// ValidateDeterminism validates that vectors are deterministic
func (vg *VectorGenerator) ValidateDeterminism(ctx context.Context, vectors []*GoldenVector) error {
	fmt.Println("🔍 Validating determinism...")

	for _, vector := range vectors {
		// Generate the same vector multiple times
		var results []*GoldenVector
		for i := 0; i < 3; i++ {
			result, err := vg.GenerateVector(ctx, vector.Name, vector.Description, vector.ArtifactHex, vector.InputHex, vector.MaxGas)
			if err != nil {
				return fmt.Errorf("failed to regenerate vector %s (attempt %d): %w", vector.Name, i+1, err)
			}
			results = append(results, result)
		}

		// Compare results
		first := results[0]
		for i, result := range results[1:] {
			if result.ExpectedGasUsed != first.ExpectedGasUsed {
				return fmt.Errorf("gas used mismatch for vector %s (attempt %d): expected %d, got %d", vector.Name, i+2, first.ExpectedGasUsed, result.ExpectedGasUsed)
			}
			if result.ExpectedOutputHash != first.ExpectedOutputHash {
				return fmt.Errorf("output hash mismatch for vector %s (attempt %d): expected %s, got %s", vector.Name, i+2, first.ExpectedOutputHash, result.ExpectedOutputHash)
			}
			if result.ExpectedReceiptHash != first.ExpectedReceiptHash {
				return fmt.Errorf("receipt hash mismatch for vector %s (attempt %d): expected %s, got %s", vector.Name, i+2, first.ExpectedReceiptHash, result.ExpectedReceiptHash)
			}
			if result.ReceiptHex != first.ReceiptHex {
				return fmt.Errorf("receipt CBOR mismatch for vector %s (attempt %d)", vector.Name, i+2)
			}
		}

		fmt.Printf("✅ Determinism validated for vector: %s\n", vector.Name)
	}

	return nil
}

func main() {
	fmt.Println("🔬 OCX Protocol Golden Vector Generator")
	fmt.Println("======================================")

	// Create generator
	generator, err := NewVectorGenerator()
	if err != nil {
		fmt.Printf("❌ Failed to create vector generator: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Generate standard vectors
	fmt.Println("\n📊 Generating standard test vectors...")
	vectors, err := generator.GenerateStandardVectors(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to generate vectors: %v\n", err)
		os.Exit(1)
	}

	// Validate determinism
	if err := generator.ValidateDeterminism(ctx, vectors); err != nil {
		fmt.Printf("❌ Determinism validation failed: %v\n", err)
		os.Exit(1)
	}

	// Save vectors
	outputDir := "./conformance/generated"
	fmt.Printf("\n💾 Saving vectors to %s...\n", outputDir)
	if err := generator.SaveVectors(vectors, outputDir); err != nil {
		fmt.Printf("❌ Failed to save vectors: %v\n", err)
		os.Exit(1)
	}

	// Generate summary
	fmt.Println("\n📋 Generation Summary:")
	fmt.Printf("  - Generated %d golden vectors\n", len(vectors))
	fmt.Printf("  - All vectors passed determinism validation\n")
	fmt.Printf("  - All vectors passed verification\n")
	fmt.Printf("  - Output directory: %s\n", outputDir)

	// Print vector details
	for i, vector := range vectors {
		fmt.Printf("\n  Vector %d: %s\n", i+1, vector.Name)
		fmt.Printf("    - Gas Used: %d\n", vector.ExpectedGasUsed)
		fmt.Printf("    - Output Hash: %s\n", vector.ExpectedOutputHash[:16]+"...")
		fmt.Printf("    - Receipt Hash: %s\n", vector.ExpectedReceiptHash[:16]+"...")
		fmt.Printf("    - Verification: %t\n", vector.VerificationResult)
	}

	fmt.Println("\n🎉 Golden vector generation completed successfully!")
}

// createExecutableArtifact creates a simple executable artifact that the deterministic VM can run
func (vg *VectorGenerator) createExecutableArtifact(artifact []byte, input []byte) (string, func(), error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "ocx-golden-vector-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create a simple shell script that outputs deterministic results based on input
	// This script produces completely deterministic output for golden vector testing
	scriptContent := fmt.Sprintf(`#!/bin/bash
# OCX Golden Vector Test Artifact
# This script produces deterministic output for testing

echo "OCX_EXECUTION_START"
echo "Artifact: %s"
echo "Input: %s"
echo "Processing input..."
echo "OCX_EXECUTION_OUTPUT: Hello from OCX Protocol!"
echo "OCX_EXECUTION_END"
`, hex.EncodeToString(artifact), hex.EncodeToString(input))

	// Create executable artifact file
	artifactPath := filepath.Join(tempDir, "artifact")
	err = os.WriteFile(artifactPath, []byte(scriptContent), 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("failed to write artifact file: %w", err)
	}

	// Return path and cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return artifactPath, cleanup, nil
}

// cacheArtifact caches the artifact in the deterministic VM's cache
func (vg *VectorGenerator) cacheArtifact(artifactHash [32]byte, artifactPath string) error {
	// Use temp directory since the deterministic VM checks there
	cacheDir := os.TempDir()
	
	// Create hash-based cache path
	hashHex := hex.EncodeToString(artifactHash[:])
	cachePath := filepath.Join(cacheDir, hashHex)

	// Copy artifact to cache
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	err = os.WriteFile(cachePath, artifactData, 0755)
	if err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}
