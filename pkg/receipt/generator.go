package receipt

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Receipt represents a cryptographic execution receipt
type Receipt struct {
	Version     string            `cbor:"v"`
	TxID        string            `cbor:"tx_id"`
	ArtifactID  string            `cbor:"artifact_id"`
	ExecutionID string            `cbor:"execution_id"`
	Timestamp   int64             `cbor:"ts"`
	ExitCode    int               `cbor:"exit_code"`
	GasUsed     uint64            `cbor:"gas_used"`
	OutputHash  string            `cbor:"output_hash"`
	Evidence    Evidence          `cbor:"evidence"`
	Metadata    map[string]string `cbor:"metadata"`
	Signature   []byte            `cbor:"sig"`
}

// Evidence contains execution proof data
type Evidence struct {
	StdoutHash    string   `cbor:"stdout_hash"`
	StderrHash    string   `cbor:"stderr_hash"`
	ResourceUsage Resource `cbor:"resources"`
	Environment   []string `cbor:"env"`
	StartTime     int64    `cbor:"start_ts"`
	EndTime       int64    `cbor:"end_ts"`
}

// Resource tracks execution resource consumption
type Resource struct {
	CPUTimeMs      int64 `cbor:"cpu_ms"`
	MemoryBytes    int64 `cbor:"mem_bytes"`
	DiskReadBytes  int64 `cbor:"disk_read"`
	DiskWriteBytes int64 `cbor:"disk_write"`
}

// Generator creates cryptographic receipts
type Generator struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	issuer     string
}

// NewGenerator creates a new receipt generator
func NewGenerator(privateKey ed25519.PrivateKey, issuer string) *Generator {
	return &Generator{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
		issuer:     issuer,
	}
}

// Generate creates a signed receipt from execution data
func (g *Generator) Generate(
	txID string,
	artifactID string,
	executionID string,
	exitCode int,
	gasUsed uint64,
	stdout, stderr []byte,
	resources Resource,
	env []string,
	startTime, endTime time.Time,
	metadata map[string]string,
) (*Receipt, error) {
	// Compute output hashes
	outputHash := computeHash(stdout)
	stdoutHash := computeHash(stdout)
	stderrHash := computeHash(stderr)

	// Build receipt
	receipt := &Receipt{
		Version:     "OCXv1",
		TxID:        txID,
		ArtifactID:  artifactID,
		ExecutionID: executionID,
		Timestamp:   time.Now().Unix(),
		ExitCode:    exitCode,
		GasUsed:     gasUsed,
		OutputHash:  outputHash,
		Evidence: Evidence{
			StdoutHash:    stdoutHash,
			StderrHash:    stderrHash,
			ResourceUsage: resources,
			Environment:   env,
			StartTime:     startTime.Unix(),
			EndTime:       endTime.Unix(),
		},
		Metadata: metadata,
	}

	// Add issuer to metadata
	if receipt.Metadata == nil {
		receipt.Metadata = make(map[string]string)
	}
	receipt.Metadata["issuer"] = g.issuer

	// Create receipt core for signing (matching standalone verifier format)
	core := &ReceiptCore{
		ProgramHash: computeHashBytes(stdout),             // Use stdout hash as program hash
		InputHash:   computeHashBytes([]byte(artifactID)), // Use artifact ID as input hash
		OutputHash:  computeHashBytes(stdout),             // Use stdout hash as output hash
		GasUsed:     gasUsed,
		StartedAt:   uint64(startTime.Unix()),
		FinishedAt:  uint64(endTime.Unix()),
		IssuerID:    g.issuer,
	}

	// Encode core with canonical CBOR
	canonicalCBOR, err := CanonicalizeCore(core)
	if err != nil {
		return nil, fmt.Errorf("failed to encode receipt core: %w", err)
	}

	// Add domain separation and sign
	domainSeparator := []byte("OCXv1|receipt|")
	message := append(domainSeparator, canonicalCBOR...)
	signature := ed25519.Sign(g.privateKey, message)

	// Convert to the old format for compatibility
	receipt.Signature = signature

	return receipt, nil
}

// Verify checks receipt signature validity
func (g *Generator) Verify(receipt *Receipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt is nil")
	}

	// Extract signature
	signature := receipt.Signature
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: %d", len(signature))
	}

	// Encode receipt without signature
	receiptForVerification := *receipt
	receiptForVerification.Signature = nil

	canonicalCBOR, err := cbor.Marshal(receiptForVerification)
	if err != nil {
		return fmt.Errorf("failed to encode receipt for verification: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(g.publicKey, canonicalCBOR, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// EncodeCBOR encodes receipt to canonical CBOR
func (g *Generator) EncodeCBOR(receipt *Receipt) ([]byte, error) {
	return cbor.Marshal(receipt)
}

// EncodeStandaloneFormat encodes receipt in the format expected by standalone verifier
func (g *Generator) EncodeStandaloneFormat(receipt *Receipt) ([]byte, error) {
	// Convert hex strings to byte arrays
	programHash, err := hex.DecodeString(receipt.Evidence.StdoutHash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode program hash: %w", err)
	}

	inputHash := computeHashBytes([]byte(receipt.ArtifactID))

	outputHash, err := hex.DecodeString(receipt.OutputHash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode output hash: %w", err)
	}

	// Create the format expected by standalone verifier
	standaloneReceipt := map[uint64]interface{}{
		1: programHash,                        // ProgramHash (32 bytes)
		2: inputHash[:],                       // InputHash (32 bytes)
		3: outputHash,                         // OutputHash (32 bytes)
		4: receipt.GasUsed,                    // GasUsed
		5: uint64(receipt.Evidence.StartTime), // StartedAt
		6: uint64(receipt.Evidence.EndTime),   // FinishedAt
		7: receipt.Metadata["issuer"],         // IssuerID
		8: receipt.Signature,                  // Signature
	}

	// Use canonical encoding
	encOpts := cbor.CanonicalEncOptions()
	em, err := encOpts.EncMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical encoder: %w", err)
	}

	return em.Marshal(standaloneReceipt)
}

// DecodeCBOR decodes receipt from CBOR
func (g *Generator) DecodeCBOR(data []byte) (*Receipt, error) {
	var receipt Receipt
	if err := cbor.Unmarshal(data, &receipt); err != nil {
		return nil, fmt.Errorf("failed to decode receipt: %w", err)
	}
	return &receipt, nil
}

// computeHash computes SHA256 hash of data
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// computeHashBytes computes SHA256 hash of data and returns as byte array
func computeHashBytes(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// GetPublicKey returns the generator's public key
func (g *Generator) GetPublicKey() ed25519.PublicKey {
	return g.publicKey
}

// CreateReceipt is a convenience function for creating receipts
func CreateReceipt(
	txID string,
	artifactID string,
	executionID string,
	exitCode int,
	gasUsed uint64,
	stdout, stderr []byte,
	resources Resource,
	env []string,
	startTime, endTime time.Time,
	metadata map[string]string,
	privateKey ed25519.PrivateKey,
	issuer string,
) ([]byte, error) {
	generator := NewGenerator(privateKey, issuer)
	receipt, err := generator.Generate(
		txID, artifactID, executionID, exitCode, gasUsed,
		stdout, stderr, resources, env, startTime, endTime, metadata,
	)
	if err != nil {
		return nil, err
	}

	return generator.EncodeCBOR(receipt)
}

// Verify is a convenience function for verifying receipts
func Verify(receiptData []byte, publicKey ed25519.PublicKey) (*VerifyResult, error) {
	generator := &Generator{publicKey: publicKey}
	receipt, err := generator.DecodeCBOR(receiptData)
	if err != nil {
		return &VerifyResult{
			Valid: false,
			Error: fmt.Sprintf("failed to decode receipt: %v", err),
		}, nil
	}

	err = generator.Verify(receipt)
	if err != nil {
		return &VerifyResult{
			Valid: false,
			Error: fmt.Sprintf("verification failed: %v", err),
		}, nil
	}

	return &VerifyResult{
		Valid:     true,
		Receipt:   receipt,
		Timestamp: time.Unix(receipt.Timestamp, 0).Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// VerifyResult represents the result of receipt verification
type VerifyResult struct {
	Valid     bool     `json:"valid"`
	Error     string   `json:"error,omitempty"`
	Receipt   *Receipt `json:"receipt,omitempty"`
	Timestamp string   `json:"timestamp,omitempty"`
}
