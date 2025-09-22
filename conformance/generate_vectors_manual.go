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
	"sort"
	"time"
)

type Receipt struct {
	Version     string
	IssuerID    string
	TimestampMs int64
	ProgramHash []byte
	InputHash   []byte
	OutputHash  []byte
	Cycles      uint64
	Meta        map[string]interface{}
	SigAlg      string
	Signature   []byte
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
	coreBytes := encodeCanonicalCBOR(receipt)

	// Create signing message
	message := append([]byte("OCXv1|receipt|"), coreBytes...)
	messageHash := sha256.Sum256(message)

	// Sign the message
	signature := ed25519.Sign(privKey, message)
	receipt.Signature = signature

	// Encode complete receipt
	fullReceiptBytes := encodeCanonicalCBOR(receipt)

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

// encodeCanonicalCBOR manually creates canonical CBOR according to RFC 8949
func encodeCanonicalCBOR(receipt Receipt) []byte {
	// Create a map with all fields
	fields := make(map[string]interface{})
	
	// Add required fields in lexical order
	fields["cycles"] = receipt.Cycles
	fields["input_hash"] = receipt.InputHash
	fields["issuer_id"] = receipt.IssuerID
	fields["output_hash"] = receipt.OutputHash
	fields["program_hash"] = receipt.ProgramHash
	fields["sig_alg"] = receipt.SigAlg
	fields["signature"] = receipt.Signature
	fields["timestamp_ms"] = receipt.TimestampMs
	fields["version"] = receipt.Version
	
	// Add optional fields if present
	if receipt.Meta != nil && len(receipt.Meta) > 0 {
		fields["meta"] = receipt.Meta
	}

	// Sort keys lexically
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("DEBUG: Sorted keys: %v\n", keys)

	// Build CBOR map
	var result []byte
	
	// Map header (0xa0 + length)
	mapLen := len(keys)
	if mapLen <= 23 {
		result = append(result, 0xa0|byte(mapLen))
	} else if mapLen <= 0xff {
		result = append(result, 0xb8, byte(mapLen))
	} else if mapLen <= 0xffff {
		result = append(result, 0xb9, byte(mapLen>>8), byte(mapLen))
	} else {
		result = append(result, 0xba, byte(mapLen>>24), byte(mapLen>>16), byte(mapLen>>8), byte(mapLen))
	}

	// Add key-value pairs
	for _, key := range keys {
		value := fields[key]
		fmt.Printf("DEBUG: Encoding key '%s' with value type %T\n", key, value)
		
		// Encode key (text string)
		keyBytes := []byte(key)
		if len(keyBytes) <= 23 {
			result = append(result, 0x60|byte(len(keyBytes)))
		} else if len(keyBytes) <= 0xff {
			result = append(result, 0x78, byte(len(keyBytes)))
		} else if len(keyBytes) <= 0xffff {
			result = append(result, 0x79, byte(len(keyBytes)>>8), byte(len(keyBytes)))
		} else {
			result = append(result, 0x7a, byte(len(keyBytes)>>24), byte(len(keyBytes)>>16), byte(len(keyBytes)>>8), byte(len(keyBytes)))
		}
		result = append(result, keyBytes...)
		
		// Encode value
		result = append(result, encodeValue(value)...)
	}

	return result
}

// encodeValue encodes a CBOR value
func encodeValue(value interface{}) []byte {
	switch v := value.(type) {
	case string:
		// Text string
		bytes := []byte(v)
		if len(bytes) <= 23 {
			return append([]byte{0x60 | byte(len(bytes))}, bytes...)
		} else if len(bytes) <= 0xff {
			return append([]byte{0x78, byte(len(bytes))}, bytes...)
		} else if len(bytes) <= 0xffff {
			return append([]byte{0x79, byte(len(bytes)>>8), byte(len(bytes))}, bytes...)
		} else {
			return append([]byte{0x7a, byte(len(bytes)>>24), byte(len(bytes)>>16), byte(len(bytes)>>8), byte(len(bytes))}, bytes...)
		}
	case []byte:
		// Byte string
		if len(v) <= 23 {
			return append([]byte{0x40 | byte(len(v))}, v...)
		} else if len(v) <= 0xff {
			return append([]byte{0x58, byte(len(v))}, v...)
		} else if len(v) <= 0xffff {
			return append([]byte{0x59, byte(len(v)>>8), byte(len(v))}, v...)
		} else {
			return append([]byte{0x5a, byte(len(v)>>24), byte(len(v)>>16), byte(len(v)>>8), byte(len(v))}, v...)
		}
	case uint64:
		// Unsigned integer
		if v <= 23 {
			return []byte{byte(v)}
		} else if v <= 0xff {
			return []byte{0x18, byte(v)}
		} else if v <= 0xffff {
			return []byte{0x19, byte(v>>8), byte(v)}
		} else if v <= 0xffffffff {
			return []byte{0x1a, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)}
		} else {
			return []byte{0x1b, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v)}
		}
	case int64:
		// Signed integer
		if v >= 0 {
			return encodeValue(uint64(v))
		} else {
			// Negative integer
			abs := uint64(-v - 1)
			if abs <= 23 {
				return []byte{0x20 | byte(abs)}
			} else if abs <= 0xff {
				return []byte{0x38, byte(abs)}
			} else if abs <= 0xffff {
				return []byte{0x39, byte(abs>>8), byte(abs)}
			} else if abs <= 0xffffffff {
				return []byte{0x3a, byte(abs>>24), byte(abs>>16), byte(abs>>8), byte(abs)}
			} else {
				return []byte{0x3b, byte(abs>>56), byte(abs>>48), byte(abs>>40), byte(abs>>32), byte(abs>>24), byte(abs>>16), byte(abs>>8), byte(abs)}
			}
		}
	case map[string]interface{}:
		// Map
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		var result []byte
		mapLen := len(keys)
		if mapLen <= 23 {
			result = append(result, 0xa0|byte(mapLen))
		} else if mapLen <= 0xff {
			result = append(result, 0xb8, byte(mapLen))
		} else if mapLen <= 0xffff {
			result = append(result, 0xb9, byte(mapLen>>8), byte(mapLen))
		} else {
			result = append(result, 0xba, byte(mapLen>>24), byte(mapLen>>16), byte(mapLen>>8), byte(mapLen))
		}
		
		for _, key := range keys {
			// Encode key
			keyBytes := []byte(key)
			if len(keyBytes) <= 23 {
				result = append(result, 0x60|byte(len(keyBytes)))
			} else if len(keyBytes) <= 0xff {
				result = append(result, 0x78, byte(len(keyBytes)))
			} else if len(keyBytes) <= 0xffff {
				result = append(result, 0x79, byte(len(keyBytes)>>8), byte(len(keyBytes)))
			} else {
				result = append(result, 0x7a, byte(len(keyBytes)>>24), byte(len(keyBytes)>>16), byte(len(keyBytes)>>8), byte(len(keyBytes)))
			}
			result = append(result, keyBytes...)
			
			// Encode value
			result = append(result, encodeValue(v[key])...)
		}
		return result
	default:
		panic(fmt.Sprintf("unsupported type: %T", value))
	}
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
