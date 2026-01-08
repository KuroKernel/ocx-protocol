// +build ignore

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           OCX Protocol - Verifiable Computation Demo          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Demo 1: Deterministic Execution
	demoDeterminism()

	// Demo 2: Receipt Creation & Verification
	demoReceiptVerification()

	// Demo 3: Replay Protection
	demoReplayProtection()

	// Demo 4: Batch Verification Performance
	demoBatchVerification()

	// Demo 5: Merkle Proofs
	demoMerkleProofs()

	fmt.Println("\n✅ All demos completed successfully!")
}

func demoDeterminism() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEMO 1: Deterministic Execution                             │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	script := `#!/bin/bash
echo "Computing 2 + 2..."
echo "Result: $((2 + 2))"
echo "Hello from OCX!"
`
	input := []byte("test input data")

	// Create temp script
	tmpDir, _ := os.MkdirTemp("", "ocx-demo")
	defer os.RemoveAll(tmpDir)
	scriptPath := filepath.Join(tmpDir, "demo.sh")
	os.WriteFile(scriptPath, []byte(script), 0755)

	vm := &deterministicvm.OSProcessVM{}
	config := deterministicvm.VMConfig{
		ArtifactPath: scriptPath,
		WorkingDir:   tmpDir,
		InputData:    input,
		Timeout:      10 * time.Second,
		CycleLimit:   100000000,
		MemoryLimit:  64 * 1024 * 1024,
		Env:          []string{"PATH=/bin:/usr/bin"},
	}

	fmt.Println("\n  Running same program 3 times...")
	fmt.Println()

	var hashes [][32]byte
	for i := 1; i <= 3; i++ {
		result, err := vm.Run(context.Background(), config)
		if err != nil {
			fmt.Printf("  ❌ Run %d failed: %v\n", i, err)
			return
		}

		outputHash := sha256.Sum256(result.Stdout)
		hashes = append(hashes, outputHash)

		fmt.Printf("  Run %d:\n", i)
		fmt.Printf("    Output Hash: %s\n", hex.EncodeToString(outputHash[:8])+"...")
		fmt.Printf("    Gas Used:    %d\n", result.GasUsed)
	}

	// Verify all identical
	allSame := hashes[0] == hashes[1] && hashes[1] == hashes[2]
	if allSame {
		fmt.Println("\n  ✅ All 3 runs produced IDENTICAL output hashes!")
		fmt.Println("     This proves deterministic execution.")
	} else {
		fmt.Println("\n  ❌ Outputs differ - not deterministic!")
	}
	fmt.Println()
}

func demoReceiptVerification() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEMO 2: Receipt Creation & Signature Verification           │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Generate keypair
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	// Create receipt
	var programHash, inputHash, outputHash [32]byte
	var nonce [16]byte
	rand.Read(programHash[:])
	rand.Read(inputHash[:])
	rand.Read(outputHash[:])
	rand.Read(nonce[:])

	now := uint64(time.Now().UnixNano())
	core := receipt.ReceiptCore{
		ProgramHash: programHash,
		InputHash:   inputHash,
		OutputHash:  outputHash,
		GasUsed:     42000,
		StartedAt:   now - 1000000,
		FinishedAt:  now - 500000,
		IssuerID:    "demo-issuer",
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
		HostCycles: 210000,
	}

	receiptData, _ := receipt.CanonicalizeFull(receiptFull)

	fmt.Println("\n  Created signed receipt:")
	fmt.Printf("    Program Hash: %s...\n", hex.EncodeToString(programHash[:8]))
	fmt.Printf("    Gas Used:     %d\n", core.GasUsed)
	fmt.Printf("    Issuer:       %s\n", core.IssuerID)
	fmt.Printf("    Signature:    %s...\n", hex.EncodeToString(sig[:16]))

	// Verify with correct key
	verifier := verify.NewGoVerifier()
	result, err := verifier.VerifyReceipt(receiptData, pub)

	fmt.Println("\n  Verifying with correct public key...")
	if err == nil && result != nil {
		fmt.Println("  ✅ Signature VALID!")
	} else {
		fmt.Println("  ❌ Verification failed")
	}

	// Verify with wrong key
	wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)
	result2, _ := verifier.VerifyReceipt(receiptData, wrongPub)

	fmt.Println("\n  Verifying with WRONG public key...")
	if result2 == nil {
		fmt.Println("  ✅ Correctly REJECTED invalid signature!")
	} else {
		fmt.Println("  ❌ Should have been rejected")
	}
	fmt.Println()
}

func demoReplayProtection() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEMO 3: Replay Attack Protection                            │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	store := verify.NewInMemoryReplayStore(time.Hour, time.Minute)

	nonce := make([]byte, 16)
	rand.Read(nonce)
	timestamp := uint64(time.Now().UnixNano())

	fmt.Printf("\n  Nonce: %s\n", hex.EncodeToString(nonce))

	// First attempt
	ok1, _ := store.CheckAndStore(nonce, timestamp)
	fmt.Printf("\n  First use:  ")
	if ok1 {
		fmt.Println("✅ ACCEPTED (new nonce)")
	} else {
		fmt.Println("❌ Rejected")
	}

	// Second attempt (replay attack)
	ok2, _ := store.CheckAndStore(nonce, timestamp)
	fmt.Printf("  Second use: ")
	if !ok2 {
		fmt.Println("✅ REJECTED (replay detected!)")
	} else {
		fmt.Println("❌ Should have been rejected")
	}

	// Third attempt
	ok3, _ := store.CheckAndStore(nonce, timestamp)
	fmt.Printf("  Third use:  ")
	if !ok3 {
		fmt.Println("✅ REJECTED (replay detected!)")
	} else {
		fmt.Println("❌ Should have been rejected")
	}

	fmt.Printf("\n  Store size: %d nonces tracked\n", store.Size())
	fmt.Println()
}

func demoBatchVerification() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEMO 4: High-Performance Batch Verification                 │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	bv, _ := verify.NewBatchVerifier(verify.BatchVerifierConfig{Workers: 4})

	// Create batch of receipts
	batchSize := 100
	fmt.Printf("\n  Creating %d signed receipts...\n", batchSize)

	batches := make([]verify.ReceiptBatch, batchSize)
	for i := 0; i < batchSize; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)

		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])
		rand.Read(nonce[:])

		now := uint64(time.Now().UnixNano())
		core := receipt.ReceiptCore{
			ProgramHash: programHash,
			InputHash:   inputHash,
			OutputHash:  outputHash,
			GasUsed:     uint64(1000 + i),
			StartedAt:   now - 1000000,
			FinishedAt:  now - 500000,
			IssuerID:    "batch-demo",
			KeyVersion:  1,
			Nonce:       nonce,
			IssuedAt:    now,
			FloatMode:   "disabled",
		}

		coreBytes, _ := receipt.CanonicalizeCore(&core)
		msg := append([]byte("OCXv1|receipt|"), coreBytes...)
		sig := ed25519.Sign(priv, msg)

		receiptFull := &receipt.ReceiptFull{Core: core, Signature: sig, HostCycles: 5000}
		data, _ := receipt.CanonicalizeFull(receiptFull)

		batches[i] = verify.ReceiptBatch{ReceiptData: data, PublicKey: pub}
	}

	// Time the batch verification
	start := time.Now()
	_, stats := bv.VerifyBatch(context.Background(), batches)
	elapsed := time.Since(start)

	throughput := float64(batchSize) / elapsed.Seconds()

	fmt.Printf("\n  Verified %d receipts in %v\n", batchSize, elapsed)
	fmt.Printf("  Throughput: %.0f verifications/second\n", throughput)
	fmt.Printf("  Valid: %d, Invalid: %d\n", stats.Valid, stats.Invalid)

	if stats.Valid == batchSize {
		fmt.Println("\n  ✅ All receipts verified successfully!")
	}
	fmt.Println()
}

func demoMerkleProofs() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ DEMO 5: Merkle Tree Proofs                                  │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Create receipts
	count := 16
	fmt.Printf("\n  Building Merkle tree with %d receipts...\n", count)

	receipts := make([]*receipt.ReceiptFull, count)
	for i := 0; i < count; i++ {
		var programHash, inputHash, outputHash [32]byte
		var nonce [16]byte
		rand.Read(programHash[:])
		rand.Read(inputHash[:])
		rand.Read(outputHash[:])
		rand.Read(nonce[:])

		receipts[i] = &receipt.ReceiptFull{
			Core: receipt.ReceiptCore{
				ProgramHash: programHash,
				InputHash:   inputHash,
				OutputHash:  outputHash,
				GasUsed:     uint64(1000 + i),
				IssuerID:    "merkle-demo",
				Nonce:       nonce,
				FloatMode:   "disabled",
			},
			Signature: make([]byte, 64),
		}
	}

	tree, _ := receipt.NewMerkleTree(receipts)

	fmt.Printf("\n  Merkle Root: %s...\n", hex.EncodeToString(tree.Root[:8]))

	// Generate and verify a proof
	proofIdx := 7
	proof, _ := tree.GenerateProof(proofIdx)

	fmt.Printf("\n  Generating proof for receipt #%d...\n", proofIdx)
	fmt.Printf("    Proof size: %d sibling hashes\n", len(proof.Siblings))

	serialized := proof.Serialize()
	fmt.Printf("    Serialized: %d bytes\n", len(serialized))

	if proof.Verify() {
		fmt.Println("\n  ✅ Proof verified! Receipt is in the tree.")
	} else {
		fmt.Println("\n  ❌ Proof invalid")
	}

	// Show that proof is compact
	fmt.Printf("\n  💡 With %d receipts, proof is only %d bytes\n", count, len(serialized))
	fmt.Printf("     (O(log n) = %d hashes needed)\n", len(proof.Siblings))
	fmt.Println()
}
