// OCX Protocol CLI - Command-line tool for receipt operations
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

const (
	version = "1.0.0"
	banner  = `
╔═══════════════════════════════════════════════════╗
║          OCX Protocol CLI v%s                  ║
║     Verifiable Computation Receipt Toolkit        ║
╚═══════════════════════════════════════════════════╝
`
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("ocx version %s\n", version)

	case "help", "-h", "--help":
		printUsage()

	case "keygen":
		cmdKeygen(os.Args[2:])

	case "sign":
		cmdSign(os.Args[2:])

	case "verify":
		cmdVerify(os.Args[2:])

	case "batch":
		cmdBatch(os.Args[2:])

	case "merkle":
		cmdMerkle(os.Args[2:])

	case "inspect":
		cmdInspect(os.Args[2:])

	case "hash":
		cmdHash(os.Args[2:])

	case "demo":
		cmdDemo()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(banner, version)
	fmt.Println(`
USAGE:
    ocx <command> [options] [arguments]

COMMANDS:
    keygen     Generate Ed25519 keypair for signing receipts
    sign       Create and sign a new receipt
    verify     Verify a receipt signature
    batch      Batch verify multiple receipts
    merkle     Build Merkle tree and generate proofs
    inspect    Inspect a receipt file
    hash       Calculate deterministic hash of a file
    demo       Run interactive demo
    version    Print version information
    help       Show this help message

EXAMPLES:
    # Generate a new keypair
    ocx keygen -o keys/

    # Create a signed receipt
    ocx sign -k keys/private.key -p program.wasm -i input.bin -o output.bin

    # Verify a receipt
    ocx verify -k keys/public.key receipt.ocx

    # Batch verify receipts
    ocx batch -k keys/public.key receipts/*.ocx

    # Build Merkle tree
    ocx merkle -o tree.json receipts/*.ocx

    # Inspect receipt
    ocx inspect receipt.ocx

    # Hash a file
    ocx hash program.wasm

For more information, visit: https://github.com/KuroKernel/ocx-protocol
`)
}

// cmdKeygen generates a new Ed25519 keypair
func cmdKeygen(args []string) {
	outputDir := "."
	prefix := "ocx"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		case "-p", "--prefix":
			if i+1 < len(args) {
				prefix = args[i+1]
				i++
			}
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Generate keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating keypair: %v\n", err)
		os.Exit(1)
	}

	// Write private key
	privPath := filepath.Join(outputDir, prefix+".key")
	if err := os.WriteFile(privPath, []byte(hex.EncodeToString(priv)), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing private key: %v\n", err)
		os.Exit(1)
	}

	// Write public key
	pubPath := filepath.Join(outputDir, prefix+".pub")
	if err := os.WriteFile(pubPath, []byte(hex.EncodeToString(pub)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing public key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated keypair:\n")
	fmt.Printf("  Private key: %s\n", privPath)
	fmt.Printf("  Public key:  %s\n", pubPath)
	fmt.Printf("  Public key (hex): %s\n", hex.EncodeToString(pub))
}

// cmdSign creates and signs a receipt
func cmdSign(args []string) {
	var keyPath, programPath, inputPath, outputPath, receiptPath, issuerID string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k", "--key":
			if i+1 < len(args) {
				keyPath = args[i+1]
				i++
			}
		case "-p", "--program":
			if i+1 < len(args) {
				programPath = args[i+1]
				i++
			}
		case "-i", "--input":
			if i+1 < len(args) {
				inputPath = args[i+1]
				i++
			}
		case "-o", "--output":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		case "-r", "--receipt":
			if i+1 < len(args) {
				receiptPath = args[i+1]
				i++
			}
		case "--issuer":
			if i+1 < len(args) {
				issuerID = args[i+1]
				i++
			}
		}
	}

	if keyPath == "" || programPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: ocx sign -k <key> -p <program> [-i input] [-o output] [-r receipt] [--issuer id]\n")
		os.Exit(1)
	}

	// Read private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading key: %v\n", err)
		os.Exit(1)
	}
	privBytes, err := hex.DecodeString(strings.TrimSpace(string(keyData)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding key: %v\n", err)
		os.Exit(1)
	}
	priv := ed25519.PrivateKey(privBytes)

	// Hash program
	programData, err := os.ReadFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading program: %v\n", err)
		os.Exit(1)
	}
	programHash := sha256.Sum256(programData)

	// Hash input
	var inputHash [32]byte
	if inputPath != "" {
		inputData, err := os.ReadFile(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		inputHash = sha256.Sum256(inputData)
	}

	// Hash output
	var outputHash [32]byte
	if outputPath != "" {
		outputData, err := os.ReadFile(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading output: %v\n", err)
			os.Exit(1)
		}
		outputHash = sha256.Sum256(outputData)
	}

	// Generate nonce
	var nonce [16]byte
	rand.Read(nonce[:])

	if issuerID == "" {
		issuerID = "ocx-cli"
	}

	now := uint64(time.Now().UnixNano())
	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     uint64(len(programData) + len(programData)/10),
		StartedAt:   now - 1000000,
		FinishedAt:  now,
		IssuerID:    issuerID,
		KeyVersion:  1,
		Nonce:       nonce,
		IssuedAt:    now,
		FloatMode:   "disabled",
	}

	// Sign
	coreBytes, _ := receipt.CanonicalizeCore(&core)
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	sig := ed25519.Sign(priv, msg)

	receiptFull := &receipt.ReceiptFull{
		Core:       core,
		Signature:  sig,
		HostCycles: core.GasUsed * 5,
	}

	// Serialize
	data, err := receipt.CanonicalizeFull(receiptFull)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing receipt: %v\n", err)
		os.Exit(1)
	}

	// Write receipt
	if receiptPath == "" {
		receiptPath = strings.TrimSuffix(programPath, filepath.Ext(programPath)) + ".ocx"
	}
	if err := os.WriteFile(receiptPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing receipt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created receipt: %s\n", receiptPath)
	fmt.Printf("  Program hash: %s\n", hex.EncodeToString(programHash[:8])+"...")
	fmt.Printf("  Gas used:     %d\n", core.GasUsed)
	fmt.Printf("  Issuer:       %s\n", issuerID)
	fmt.Printf("  Size:         %d bytes\n", len(data))
}

// cmdVerify verifies a receipt
func cmdVerify(args []string) {
	var keyPath string
	var receiptPaths []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k", "--key":
			if i+1 < len(args) {
				keyPath = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				receiptPaths = append(receiptPaths, args[i])
			}
		}
	}

	if keyPath == "" || len(receiptPaths) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ocx verify -k <public_key> <receipt.ocx> [...]\n")
		os.Exit(1)
	}

	// Read public key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading key: %v\n", err)
		os.Exit(1)
	}
	pubBytes, err := hex.DecodeString(strings.TrimSpace(string(keyData)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding key: %v\n", err)
		os.Exit(1)
	}

	verifier := verify.NewGoVerifier()

	allValid := true
	for _, path := range receiptPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("  %s: ERROR (cannot read: %v)\n", path, err)
			allValid = false
			continue
		}

		result, err := verifier.VerifyReceipt(data, pubBytes)
		if err != nil || result == nil {
			fmt.Printf("  %s: INVALID\n", path)
			allValid = false
		} else {
			fmt.Printf("  %s: VALID (gas=%d, issuer=%s)\n", path, result.GasUsed, result.IssuerID)
		}
	}

	if allValid {
		fmt.Println("\nAll receipts verified successfully!")
	} else {
		fmt.Println("\nSome receipts failed verification!")
		os.Exit(1)
	}
}

// cmdBatch performs batch verification
func cmdBatch(args []string) {
	var keyPath string
	var receiptPaths []string
	workers := 4

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k", "--key":
			if i+1 < len(args) {
				keyPath = args[i+1]
				i++
			}
		case "-w", "--workers":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &workers)
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				// Expand glob patterns
				matches, err := filepath.Glob(args[i])
				if err == nil && len(matches) > 0 {
					receiptPaths = append(receiptPaths, matches...)
				} else {
					receiptPaths = append(receiptPaths, args[i])
				}
			}
		}
	}

	if keyPath == "" || len(receiptPaths) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ocx batch -k <public_key> [-w workers] <receipts...>\n")
		os.Exit(1)
	}

	// Read public key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading key: %v\n", err)
		os.Exit(1)
	}
	pubBytes, err := hex.DecodeString(strings.TrimSpace(string(keyData)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding key: %v\n", err)
		os.Exit(1)
	}

	// Build batch
	batches := make([]verify.ReceiptBatch, 0, len(receiptPaths))
	for _, path := range receiptPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", path, err)
			continue
		}
		batches = append(batches, verify.ReceiptBatch{
			ReceiptData: data,
			PublicKey:   pubBytes,
		})
	}

	fmt.Printf("Batch verifying %d receipts with %d workers...\n\n", len(batches), workers)

	bv, _ := verify.NewBatchVerifier(verify.BatchVerifierConfig{Workers: workers})

	start := time.Now()
	results, stats := bv.VerifyBatch(context.Background(), batches)
	elapsed := time.Since(start)

	// Print results
	for i, result := range results {
		status := "VALID"
		if !result.Valid {
			status = "INVALID"
		}
		fmt.Printf("  [%d] %s: %s\n", i+1, receiptPaths[i], status)
	}

	throughput := float64(len(batches)) / elapsed.Seconds()

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Valid:      %d\n", stats.Valid)
	fmt.Printf("  Invalid:    %d\n", stats.Invalid)
	fmt.Printf("  Duration:   %v\n", elapsed)
	fmt.Printf("  Throughput: %.0f receipts/sec\n", throughput)
}

// cmdMerkle builds a Merkle tree
func cmdMerkle(args []string) {
	var outputPath string
	var receiptPaths []string
	proofIndex := -1

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		case "-p", "--proof":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &proofIndex)
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				matches, err := filepath.Glob(args[i])
				if err == nil && len(matches) > 0 {
					receiptPaths = append(receiptPaths, matches...)
				} else {
					receiptPaths = append(receiptPaths, args[i])
				}
			}
		}
	}

	if len(receiptPaths) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ocx merkle [-o output.json] [-p proof_index] <receipts...>\n")
		os.Exit(1)
	}

	// Load receipts
	receipts := make([]*receipt.ReceiptFull, 0, len(receiptPaths))
	for _, path := range receiptPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read %s\n", path)
			continue
		}

		// Parse receipt (simplified)
		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		receiptFull := &receipt.ReceiptFull{
			Core: receipt.ReceiptCore{
				ProgramHash: programHash,
				InputHash:   inputHash,
				OutputHash:  outputHash,
				Nonce:       nonce,
			},
			Signature: make([]byte, 64),
		}
		// Use actual hash of data for simplicity
		hash := sha256.Sum256(data)
		copy(receiptFull.Core.ProgramHash[:], hash[:])

		receipts = append(receipts, receiptFull)
	}

	if len(receipts) == 0 {
		fmt.Fprintf(os.Stderr, "No valid receipts found\n")
		os.Exit(1)
	}

	// Build tree
	tree, err := receipt.NewMerkleTree(receipts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building tree: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merkle Tree:\n")
	fmt.Printf("  Leaves:     %d\n", len(receipts))
	fmt.Printf("  Root:       %s\n", hex.EncodeToString(tree.Root[:]))

	// Generate proof if requested
	if proofIndex >= 0 && proofIndex < len(receipts) {
		proof, err := tree.GenerateProof(proofIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating proof: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nProof for index %d:\n", proofIndex)
		fmt.Printf("  Siblings:   %d\n", len(proof.Siblings))
		fmt.Printf("  Valid:      %v\n", proof.Verify())

		serialized := proof.Serialize()
		fmt.Printf("  Size:       %d bytes\n", len(serialized))
	}

	// Write output
	if outputPath != "" {
		output := map[string]interface{}{
			"root":   hex.EncodeToString(tree.Root[:]),
			"leaves": len(receipts),
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		os.WriteFile(outputPath, data, 0644)
		fmt.Printf("\nTree saved to: %s\n", outputPath)
	}
}

// cmdInspect inspects a receipt
func cmdInspect(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ocx inspect <receipt.ocx>\n")
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	verifier := verify.NewGoVerifier()
	fields, err := verifier.ExtractReceiptFields(data)

	hash := sha256.Sum256(data)
	fmt.Printf("Receipt: %s\n", args[0])
	fmt.Printf("  Size:         %d bytes\n", len(data))
	fmt.Printf("  SHA256:       %s\n", hex.EncodeToString(hash[:]))

	if err == nil && fields != nil {
		fmt.Printf("\nCore Fields:\n")
		fmt.Printf("  Program Hash: %s\n", hex.EncodeToString(fields.ProgramHash[:]))
		fmt.Printf("  Input Hash:   %s\n", hex.EncodeToString(fields.InputHash[:]))
		fmt.Printf("  Output Hash:  %s\n", hex.EncodeToString(fields.OutputHash[:]))
		fmt.Printf("  Gas Used:     %d\n", fields.GasUsed)
		fmt.Printf("  Issuer ID:    %s\n", fields.IssuerID)
		fmt.Printf("  Float Mode:   %s\n", fields.FloatMode)
	} else {
		fmt.Printf("\n  (Could not parse receipt fields)\n")
	}
}

// cmdHash hashes a file
func cmdHash(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ocx hash <file>\n")
		os.Exit(1)
	}

	for _, path := range args {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", path, err)
			continue
		}

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
			continue
		}
		f.Close()

		hash := h.Sum(nil)
		fmt.Printf("%s  %s\n", hex.EncodeToString(hash), path)
	}
}

// cmdDemo runs the interactive demo
func cmdDemo() {
	fmt.Printf(banner, version)
	fmt.Println("Running OCX Protocol Demo...")
	fmt.Println()

	// This will import and run the demo from the demo package
	fmt.Println("Use 'go run demo/determinism_demo.go' for full demo")
}
