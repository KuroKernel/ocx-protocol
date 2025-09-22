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
)

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
	}{
		{
			name:        "minimal",
			programHash: sha256.Sum256([]byte("hello world")),
			inputHash:   sha256.Sum256([]byte("input")),
			outputHash:  sha256.Sum256([]byte("output")),
			cycles:      1000,
		},
		{
			name:        "complex",
			programHash: sha256.Sum256([]byte("complex program")),
			inputHash:   sha256.Sum256([]byte("complex input")),
			outputHash:  sha256.Sum256([]byte("complex output")),
			cycles:      50000,
		},
	}

	for i, vector := range vectors {
		generateVector(i, vector.name, vector.programHash, vector.inputHash,
			vector.outputHash, vector.cycles)
	}

	fmt.Println("Generated", len(vectors), "conformance vectors")
}

func generateVector(index int, name string, programHash, inputHash, outputHash [32]byte,
	cycles uint64) {

	// Generate Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	// Create receipt without signature
	now := time.Now().Unix()

	// Encode core (without signature) in canonical CBOR with integer keys
	coreBytes := encodeReceiptCore(programHash[:], inputHash[:], outputHash[:],
		cycles, uint64(now), uint64(now+1), fmt.Sprintf("issuer-%d", index))

	// Create signing message
	message := append([]byte("OCXv1|receipt|"), coreBytes...)
	messageHash := sha256.Sum256(message)

	// Sign the message
	signature := ed25519.Sign(privKey, message)

	// Encode complete receipt with signature
	fullReceiptBytes := encodeReceipt(programHash[:], inputHash[:], outputHash[:],
		cycles, uint64(now), uint64(now+1), fmt.Sprintf("issuer-%d", index), signature)

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
	diag := generateDiagnostic(programHash[:], inputHash[:], outputHash[:],
		cycles, uint64(now), uint64(now+1), fmt.Sprintf("issuer-%d", index), signature)
	writeFile(vectorDir, "receipt.diag", []byte(diag))

	fmt.Printf("Generated vector %03d_%s\n", index, name)
}

// encodeReceiptCore creates canonical CBOR for receipt without signature
func encodeReceiptCore(programHash, inputHash, outputHash []byte, cycles, startedAt, finishedAt uint64, issuerID string) []byte {
	var cbor []byte

	// Map header: map(7) - 7 fields without signature
	cbor = append(cbor, 0xa7)

	// Key 1: program_hash (32 bytes)
	cbor = append(cbor, 0x01)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, programHash...)

	// Key 2: input_hash (32 bytes)
	cbor = append(cbor, 0x02)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, inputHash...)

	// Key 3: output_hash (32 bytes)
	cbor = append(cbor, 0x03)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, outputHash...)

	// Key 4: cycles (uint64) - use minimal encoding
	cbor = append(cbor, 0x04)
	if cycles <= 0x17 {
		cbor = append(cbor, byte(cycles))
	} else if cycles <= 0xff {
		cbor = append(cbor, 0x18, byte(cycles))
	} else if cycles <= 0xffff {
		cbor = append(cbor, 0x19, byte(cycles>>8), byte(cycles))
	} else if cycles <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(cycles>>24), byte(cycles>>16), byte(cycles>>8), byte(cycles))
	} else {
		cbor = append(cbor, 0x1b, byte(cycles>>56), byte(cycles>>48), byte(cycles>>40), byte(cycles>>32),
			byte(cycles>>24), byte(cycles>>16), byte(cycles>>8), byte(cycles))
	}

	// Key 5: started_at (uint64) - use minimal encoding
	cbor = append(cbor, 0x05)
	if startedAt <= 0x17 {
		cbor = append(cbor, byte(startedAt))
	} else if startedAt <= 0xff {
		cbor = append(cbor, 0x18, byte(startedAt))
	} else if startedAt <= 0xffff {
		cbor = append(cbor, 0x19, byte(startedAt>>8), byte(startedAt))
	} else if startedAt <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(startedAt>>24), byte(startedAt>>16), byte(startedAt>>8), byte(startedAt))
	} else {
		cbor = append(cbor, 0x1b, byte(startedAt>>56), byte(startedAt>>48), byte(startedAt>>40), byte(startedAt>>32),
			byte(startedAt>>24), byte(startedAt>>16), byte(startedAt>>8), byte(startedAt))
	}

	// Key 6: finished_at (uint64) - use minimal encoding
	cbor = append(cbor, 0x06)
	if finishedAt <= 0x17 {
		cbor = append(cbor, byte(finishedAt))
	} else if finishedAt <= 0xff {
		cbor = append(cbor, 0x18, byte(finishedAt))
	} else if finishedAt <= 0xffff {
		cbor = append(cbor, 0x19, byte(finishedAt>>8), byte(finishedAt))
	} else if finishedAt <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(finishedAt>>24), byte(finishedAt>>16), byte(finishedAt>>8), byte(finishedAt))
	} else {
		cbor = append(cbor, 0x1b, byte(finishedAt>>56), byte(finishedAt>>48), byte(finishedAt>>40), byte(finishedAt>>32),
			byte(finishedAt>>24), byte(finishedAt>>16), byte(finishedAt>>8), byte(finishedAt))
	}

	// Key 7: issuer_id (text string)
	cbor = append(cbor, 0x07)
	issuerBytes := []byte(issuerID)
	if len(issuerBytes) <= 0x17 {
		cbor = append(cbor, byte(0x60+len(issuerBytes)))
	} else if len(issuerBytes) <= 0xff {
		cbor = append(cbor, 0x78, byte(len(issuerBytes)))
	} else if len(issuerBytes) <= 0xffff {
		cbor = append(cbor, 0x79, byte(len(issuerBytes)>>8), byte(len(issuerBytes)))
	} else {
		cbor = append(cbor, 0x7a, byte(len(issuerBytes)>>24), byte(len(issuerBytes)>>16), byte(len(issuerBytes)>>8), byte(len(issuerBytes)))
	}
	cbor = append(cbor, issuerBytes...)

	return cbor
}

// encodeReceipt creates canonical CBOR for complete receipt with signature
func encodeReceipt(programHash, inputHash, outputHash []byte, cycles, startedAt, finishedAt uint64, issuerID string, signature []byte) []byte {
	var cbor []byte

	// Map header: map(8) - 8 fields including signature
	cbor = append(cbor, 0xa8)

	// Key 1: program_hash (32 bytes)
	cbor = append(cbor, 0x01)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, programHash...)

	// Key 2: input_hash (32 bytes)
	cbor = append(cbor, 0x02)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, inputHash...)

	// Key 3: output_hash (32 bytes)
	cbor = append(cbor, 0x03)
	cbor = append(cbor, 0x58, 0x20) // bytes(32)
	cbor = append(cbor, outputHash...)

	// Key 4: cycles (uint64) - use minimal encoding
	cbor = append(cbor, 0x04)
	if cycles <= 0x17 {
		cbor = append(cbor, byte(cycles))
	} else if cycles <= 0xff {
		cbor = append(cbor, 0x18, byte(cycles))
	} else if cycles <= 0xffff {
		cbor = append(cbor, 0x19, byte(cycles>>8), byte(cycles))
	} else if cycles <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(cycles>>24), byte(cycles>>16), byte(cycles>>8), byte(cycles))
	} else {
		cbor = append(cbor, 0x1b, byte(cycles>>56), byte(cycles>>48), byte(cycles>>40), byte(cycles>>32),
			byte(cycles>>24), byte(cycles>>16), byte(cycles>>8), byte(cycles))
	}

	// Key 5: started_at (uint64) - use minimal encoding
	cbor = append(cbor, 0x05)
	if startedAt <= 0x17 {
		cbor = append(cbor, byte(startedAt))
	} else if startedAt <= 0xff {
		cbor = append(cbor, 0x18, byte(startedAt))
	} else if startedAt <= 0xffff {
		cbor = append(cbor, 0x19, byte(startedAt>>8), byte(startedAt))
	} else if startedAt <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(startedAt>>24), byte(startedAt>>16), byte(startedAt>>8), byte(startedAt))
	} else {
		cbor = append(cbor, 0x1b, byte(startedAt>>56), byte(startedAt>>48), byte(startedAt>>40), byte(startedAt>>32),
			byte(startedAt>>24), byte(startedAt>>16), byte(startedAt>>8), byte(startedAt))
	}

	// Key 6: finished_at (uint64) - use minimal encoding
	cbor = append(cbor, 0x06)
	if finishedAt <= 0x17 {
		cbor = append(cbor, byte(finishedAt))
	} else if finishedAt <= 0xff {
		cbor = append(cbor, 0x18, byte(finishedAt))
	} else if finishedAt <= 0xffff {
		cbor = append(cbor, 0x19, byte(finishedAt>>8), byte(finishedAt))
	} else if finishedAt <= 0xffffffff {
		cbor = append(cbor, 0x1a, byte(finishedAt>>24), byte(finishedAt>>16), byte(finishedAt>>8), byte(finishedAt))
	} else {
		cbor = append(cbor, 0x1b, byte(finishedAt>>56), byte(finishedAt>>48), byte(finishedAt>>40), byte(finishedAt>>32),
			byte(finishedAt>>24), byte(finishedAt>>16), byte(finishedAt>>8), byte(finishedAt))
	}

	// Key 7: issuer_id (text string)
	cbor = append(cbor, 0x07)
	issuerBytes := []byte(issuerID)
	if len(issuerBytes) <= 0x17 {
		cbor = append(cbor, byte(0x60+len(issuerBytes)))
	} else if len(issuerBytes) <= 0xff {
		cbor = append(cbor, 0x78, byte(len(issuerBytes)))
	} else if len(issuerBytes) <= 0xffff {
		cbor = append(cbor, 0x79, byte(len(issuerBytes)>>8), byte(len(issuerBytes)))
	} else {
		cbor = append(cbor, 0x7a, byte(len(issuerBytes)>>24), byte(len(issuerBytes)>>16), byte(len(issuerBytes)>>8), byte(len(issuerBytes)))
	}
	cbor = append(cbor, issuerBytes...)

	// Key 8: signature (64 bytes)
	cbor = append(cbor, 0x08)
	cbor = append(cbor, 0x58, 0x40) // bytes(64)
	cbor = append(cbor, signature...)

	return cbor
}

func writeFile(dir, name string, data []byte) {
	path := filepath.Join(dir, name)
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func generateDiagnostic(programHash, inputHash, outputHash []byte, cycles, startedAt, finishedAt uint64, issuerID string, signature []byte) string {
	return fmt.Sprintf(`{
  "program_hash": h'%x',
  "input_hash": h'%x',
  "output_hash": h'%x',
  "cycles": %d,
  "started_at": %d,
  "finished_at": %d,
  "issuer_id": "%s",
  "signature": h'%x'
}`, programHash, inputHash, outputHash, cycles, startedAt, finishedAt, issuerID, signature)
}
