package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fxamacker/cbor/v2"
)

type Receipt struct {
	Version     string                 `cbor:"version"`
	IssuerID    string                 `cbor:"issuer_id"`
	TimestampMs int64                  `cbor:"timestamp_ms"`
	ProgramHash []byte                 `cbor:"program_hash"`
	InputHash   []byte                 `cbor:"input_hash"`
	OutputHash  []byte                 `cbor:"output_hash"`
	Cycles      uint64                 `cbor:"cycles"`
	Meta        map[string]interface{} `cbor:"meta,omitempty"`
	SigAlg      string                 `cbor:"sig_alg"`
	Signature   []byte                 `cbor:"signature"`
}

func main() {
	// Create output directory
	os.MkdirAll("conformance/receipts/v1", 0755)

	// Generate test vectors
	vectors := []struct {
		name        string
		programHash [32]byte
		inputHash   [32]byte
		outputHash  [32]byte
		cycles      uint64
		meta        map[string]interface{}
	}{
		{
			name:        "minimal",
			programHash: sha256.Sum256([]byte("hello world")),
			inputHash:   sha256.Sum256([]byte("input")),
			outputHash:  sha256.Sum256([]byte("output")),
			cycles:      1000,
			meta:        nil,
		},
		{
			name:        "with_meta",
			programHash: sha256.Sum256([]byte("complex program")),
			inputHash:   sha256.Sum256([]byte("complex input")),
			outputHash:  sha256.Sum256([]byte("complex output")),
			cycles:      50000,
			meta: map[string]interface{}{
				"build_id": "abc123",
				"compiler": "rustc 1.70.0",
			},
		},
	}

	for i, vector := range vectors {
		generateVector(i, vector.name, vector.programHash, vector.inputHash,
			vector.outputHash, vector.cycles, vector.meta)
	}

	fmt.Println("Generated", len(vectors), "conformance vectors")
}

func generateVector(index int, name string, programHash, inputHash, outputHash [32]byte,
	cycles uint64, meta map[string]interface{}) {

	// Generate Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	// Create receipt without signature
	now := time.Now().UnixMilli()
	receipt := Receipt{
		Version:     "ocx-1",
		IssuerID:    fmt.Sprintf("issuer-%d", index),
		TimestampMs: now,
		ProgramHash: programHash[:],
		InputHash:   inputHash[:],
		OutputHash:  outputHash[:],
		Cycles:      cycles,
		Meta:        meta,
		SigAlg:      "ed25519",
	}

	// Encode core (without signature) in canonical CBOR
	// Use lexical sorting instead of length-first sorting
	opts := cbor.CanonicalEncOptions()
	opts.Sort = cbor.SortNone // No sorting, we'll handle it manually
	enc, err := opts.EncMode()
	if err != nil {
		panic(err)
	}

	coreBytes, err := enc.Marshal(receipt)
	if err != nil {
		panic(err)
	}

	// Create signing message
	message := append([]byte("OCXv1|receipt|"), coreBytes...)
	messageHash := sha256.Sum256(message)

	// Sign the message
	signature := ed25519.Sign(privKey, message)
	receipt.Signature = signature

	// Encode complete receipt
	fullReceiptBytes, err := enc.Marshal(receipt)
	if err != nil {
		panic(err)
	}

	// Create vector directory
	vectorDir := fmt.Sprintf("conformance/receipts/v1/%03d_%s", index, name)
	os.MkdirAll(vectorDir, 0755)

	// Write all vector files
	writeFile(vectorDir, "receipt.cbor", fullReceiptBytes)
	writeFile(vectorDir, "core.cbor", coreBytes)
	writeFile(vectorDir, "message.bin", message)
	writeFile(vectorDir, "message.sha256", []byte(hex.EncodeToString(messageHash[:])))
	writeFile(vectorDir, "pubkey.bin", pubKey)
	writeFile(vectorDir, "signature.bin", signature)

	// Generate diagnostic text
	diag := generateDiagnostic(receipt)
	writeFile(vectorDir, "receipt.diag", []byte(diag))

	fmt.Printf("Generated vector %03d_%s\n", index, name)
}

func writeFile(dir, name string, data []byte) {
	path := filepath.Join(dir, name)
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func generateDiagnostic(receipt Receipt) string {
	return fmt.Sprintf(`{
  "version": "%s",
  "issuer_id": "%s", 
  "timestamp_ms": %d,
  "program_hash": h'%x',
  "input_hash": h'%x',
  "output_hash": h'%x',
  "cycles": %d,
  "sig_alg": "%s",
  "signature": h'%x'
}`, receipt.Version, receipt.IssuerID, receipt.TimestampMs,
		receipt.ProgramHash, receipt.InputHash, receipt.OutputHash,
		receipt.Cycles, receipt.SigAlg, receipt.Signature)
}
