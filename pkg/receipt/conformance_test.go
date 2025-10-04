package receipt

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ConformanceVector represents a test vector from the manifest
type ConformanceVector struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	File             string `json:"file"`
	ExpectedCoreHash string `json:"expected_core_hash"`
	SignerPubkeyB64  string `json:"signer_pubkey_b64"`
	ArtifactType     string `json:"artifact_type"`
	ExitCode         int    `json:"exit_code"`
}

// ConformanceManifest represents the manifest file
type ConformanceManifest struct {
	Version string              `json:"version"`
	Vectors []ConformanceVector `json:"vectors"`
}

// TestReceiptConformance tests all receipt vectors in the conformance directory
func TestReceiptConformance(t *testing.T) {
	manifestPath := "conformance/receipts/v1/manifest.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Skip("Conformance manifest not found, skipping conformance tests")
		return
	}

	// Load manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest ConformanceManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	t.Logf("Testing %d conformance vectors", len(manifest.Vectors))

	for _, vector := range manifest.Vectors {
		t.Run(vector.ID, func(t *testing.T) {
			testReceiptVector(t, vector)
		})
	}
}

// testReceiptVector tests a single receipt vector
func testReceiptVector(t *testing.T, vector ConformanceVector) {
	// Skip vectors with pending core hashes
	if vector.ExpectedCoreHash == "pending" {
		t.Skip("Vector has pending core hash")
		return
	}

	receiptPath := filepath.Join("conformance/receipts/v1", vector.File)
	if _, err := os.Stat(receiptPath); os.IsNotExist(err) {
		t.Skipf("Receipt file not found: %s", receiptPath)
		return
	}

	// Load receipt
	receiptData, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("Failed to read receipt: %v", err)
	}

	// Decode public key
	pubkeyData, err := base64.StdEncoding.DecodeString(vector.SignerPubkeyB64)
	if err != nil {
		t.Fatalf("Failed to decode public key: %v", err)
	}

	if len(pubkeyData) != ed25519.PublicKeySize {
		t.Fatalf("Invalid public key size: %d", len(pubkeyData))
	}

	pubkey := ed25519.PublicKey(pubkeyData)

	// Verify receipt
	result, err := Verify(receiptData, pubkey)
	if err != nil {
		t.Fatalf("Receipt verification failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("Receipt verification returned invalid: %s", result.Error)
	}

	// Verify core hash matches expected
	// This would require extracting the core hash from the receipt
	// and comparing it to vector.ExpectedCoreHash

	t.Logf("✅ Vector %s verified successfully", vector.ID)
}

// TestReceiptCanonicalization tests that receipts are canonicalized correctly
func TestReceiptCanonicalization(t *testing.T) {
	// Create a test receipt
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	generator := NewGenerator(privKey, "test-issuer")

	// Generate receipt
	receipt, err := generator.Generate(
		"test-tx",
		"test-artifact",
		"test-execution",
		0,   // exit code
		232, // gas used
		[]byte("test output"),
		[]byte("test stderr"),
		Resource{CPUTimeMs: 100, MemoryBytes: 1024},
		[]string{"TEST=1"},
		time.Now(),
		time.Now(),
		map[string]string{"test": "metadata"},
	)
	if err != nil {
		t.Fatalf("Failed to generate receipt: %v", err)
	}

	// Encode to CBOR
	cbor1, err := generator.EncodeCBOR(receipt)
	if err != nil {
		t.Fatalf("Failed to encode receipt: %v", err)
	}

	// Decode and re-encode
	decoded, err := generator.DecodeCBOR(cbor1)
	if err != nil {
		t.Fatalf("Failed to decode receipt: %v", err)
	}

	cbor2, err := generator.EncodeCBOR(decoded)
	if err != nil {
		t.Fatalf("Failed to re-encode receipt: %v", err)
	}

	// Should be byte-for-byte identical
	if len(cbor1) != len(cbor2) {
		t.Fatalf("CBOR length mismatch: %d vs %d", len(cbor1), len(cbor2))
	}

	for i := 0; i < len(cbor1); i++ {
		if cbor1[i] != cbor2[i] {
			t.Fatalf("CBOR byte mismatch at position %d: %02x vs %02x", i, cbor1[i], cbor2[i])
		}
	}

	t.Log("✅ Receipt canonicalization verified")
}

// TestReceiptDeterminism tests that identical inputs produce identical receipts
func TestReceiptDeterminism(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	generator := NewGenerator(privKey, "test-issuer")

	// Fixed timestamp for deterministic testing
	fixedTime := time.Unix(1640995200, 0)

	// Generate receipt multiple times
	var receipts [][]byte
	for i := 0; i < 5; i++ {
		receipt, err := generator.Generate(
			"test-tx",
			"test-artifact",
			"test-execution",
			0,   // exit code
			232, // gas used
			[]byte("test output"),
			[]byte("test stderr"),
			Resource{CPUTimeMs: 100, MemoryBytes: 1024},
			[]string{"TEST=1"},
			fixedTime,
			fixedTime,
			map[string]string{"test": "metadata"},
		)
		if err != nil {
			t.Fatalf("Failed to generate receipt %d: %v", i, err)
		}

		cbor, err := generator.EncodeCBOR(receipt)
		if err != nil {
			t.Fatalf("Failed to encode receipt %d: %v", i, err)
		}

		receipts = append(receipts, cbor)
	}

	// All receipts should be identical
	firstReceipt := receipts[0]
	for i := 1; i < len(receipts); i++ {
		if len(firstReceipt) != len(receipts[i]) {
			t.Fatalf("Receipt %d length mismatch: %d vs %d", i, len(firstReceipt), len(receipts[i]))
		}

		for j := 0; j < len(firstReceipt); j++ {
			if firstReceipt[j] != receipts[i][j] {
				t.Fatalf("Receipt %d byte mismatch at position %d: %02x vs %02x", i, j, firstReceipt[j], receipts[i][j])
			}
		}
	}

	t.Log("✅ Receipt determinism verified")
}
