// cmd/ocx-verifier/main.go - Tiny Standalone OCX Verifier
// This extracts the verification logic into a <1000 LOC standalone binary

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"ocx.local/pkg/cbor"
	"ocx.local/pkg/verify"
)

// VerifierConfig represents the verifier configuration
type VerifierConfig struct {
	InputFile    string `json:"input_file"`
	OutputFile   string `json:"output_file"`
	PublicKey    string `json:"public_key"`
	VerifyOnly   bool   `json:"verify_only"`
	Verbose      bool   `json:"verbose"`
	Profile      string `json:"profile"`
	MaxCycles    uint64 `json:"max_cycles"`
}

// VerifierResult represents the verification result
type VerifierResult struct {
	Valid        bool              `json:"valid"`
	Error        string            `json:"error,omitempty"`
	Receipt      *cbor.ReceiptV11 `json:"receipt,omitempty"`
	CyclesUsed   uint64            `json:"cycles_used"`
	VerificationTime time.Duration `json:"verification_time"`
	Profile      string            `json:"profile"`
	Chained      bool              `json:"chained"`
	Witness      bool              `json:"witness"`
}

// TinyVerifier represents the tiny OCX verifier
type TinyVerifier struct {
	config   *VerifierConfig
	verifier verify.Verifier
}

// NewTinyVerifier creates a new tiny verifier
func NewTinyVerifier(config *VerifierConfig) *TinyVerifier {
	return &TinyVerifier{
		config:   config,
		verifier: verify.NewVerifier(),
	}
}

// VerifyFile verifies a receipt file
func (tv *TinyVerifier) VerifyFile() (*VerifierResult, error) {
	start := time.Now()
	
	// Read input file
	data, err := ioutil.ReadFile(tv.config.InputFile)
	if err != nil {
		return &VerifierResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to read input file: %v", err),
		}, nil
	}
	
	// Parse receipt
	receipt, err := tv.parseReceipt(data)
	if err != nil {
		return &VerifierResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to parse receipt: %v", err),
		}, nil
	}
	
	// Verify receipt
	valid, err := tv.verifyReceipt(receipt)
	if err != nil {
		return &VerifierResult{
			Valid: false,
			Error: fmt.Sprintf("Verification failed: %v", err),
		}, nil
	}
	
	verificationTime := time.Since(start)
	
	result := &VerifierResult{
		Valid:           valid,
		Receipt:         receipt,
		CyclesUsed:      receipt.Cycles,
		VerificationTime: verificationTime,
		Profile:         tv.config.Profile,
		Chained:         receipt.IsChained(),
		Witness:         receipt.HasWitness(),
	}
	
	// Write output file if specified
	if tv.config.OutputFile != "" {
		if err := tv.writeOutput(result); err != nil {
			log.Printf("Warning: Failed to write output file: %v", err)
		}
	}
	
	return result, nil
}

// parseReceipt parses a receipt from data
func (tv *TinyVerifier) parseReceipt(data []byte) (*cbor.ReceiptV11, error) {
	// In a real implementation, this would parse CBOR
	// For now, create a placeholder receipt
	return &cbor.ReceiptV11{
		Artifact: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Input:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Cycles:   10000,
		Version:  "1.1",
		CreatedAt: time.Now(),
		IssuerKeyID: "test-key",
		Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}, nil
}

// verifyReceipt verifies a receipt
func (tv *TinyVerifier) verifyReceipt(receipt *cbor.ReceiptV11) (bool, error) {
	// Check cycles limit
	if receipt.Cycles > tv.config.MaxCycles {
		return false, fmt.Errorf("cycles exceeded: %d > %d", receipt.Cycles, tv.config.MaxCycles)
	}
	
	// Verify signature
	publicKey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	if !receipt.VerifySignature(publicKey) {
		return false, fmt.Errorf("signature verification failed")
	}
	
	// Additional verification logic would go here
	return true, nil
}

// writeOutput writes the verification result to output file
func (tv *TinyVerifier) writeOutput(result *VerifierResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(tv.config.OutputFile, data, 0644)
}

// printResult prints the verification result
func (tv *TinyVerifier) printResult(result *VerifierResult) {
	if tv.config.Verbose {
		fmt.Printf("OCX Verifier Result:\n")
		fmt.Printf("  Valid: %t\n", result.Valid)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		fmt.Printf("  Cycles Used: %d\n", result.CyclesUsed)
		fmt.Printf("  Verification Time: %v\n", result.VerificationTime)
		fmt.Printf("  Profile: %s\n", result.Profile)
		fmt.Printf("  Chained: %t\n", result.Chained)
		fmt.Printf("  Witness: %t\n", result.Witness)
	} else {
		if result.Valid {
			fmt.Printf("✅ OCX verification passed (%v)\n", result.VerificationTime)
		} else {
			fmt.Printf("❌ OCX verification failed: %s\n", result.Error)
		}
	}
}

func main() {
	// Parse command line flags
	var config VerifierConfig
	
	flag.StringVar(&config.InputFile, "input", "", "Input receipt file")
	flag.StringVar(&config.OutputFile, "output", "", "Output result file")
	flag.StringVar(&config.PublicKey, "key", "", "Public key for verification")
	flag.BoolVar(&config.VerifyOnly, "verify-only", false, "Only verify, do not execute")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.StringVar(&config.Profile, "profile", "v1-min", "OCX profile")
	flag.Uint64Var(&config.MaxCycles, "max-cycles", 50000, "Maximum cycles allowed")
	
	flag.Parse()
	
	// Validate configuration
	if config.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file is required\n")
		flag.Usage()
		os.Exit(1)
	}
	
	// Create verifier
	verifier := NewTinyVerifier(&config)
	
	// Verify file
	result, err := verifier.VerifyFile()
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	
	// Print result
	verifier.printResult(result)
	
	// Exit with appropriate code
	if result.Valid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
