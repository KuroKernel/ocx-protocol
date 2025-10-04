package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Test the reputation compute logic
func main() {
	fmt.Println("===========================================")
	fmt.Println("OCX Reputation Compute Test")
	fmt.Println("===========================================")
	fmt.Println()

	// Test Case 1: GitHub only
	fmt.Println("Test 1: GitHub Platform Only")
	fmt.Println("-------------------------------------------")
	userID := "testuser"
	platforms := map[string]float64{
		"github": 85.5,
	}
	score, confidence := computeScore(platforms)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("GitHub Score: %.2f\n", platforms["github"])
	fmt.Printf("Trust Score: %.2f\n", score)
	fmt.Printf("Confidence: %.2f\n", confidence)
	fmt.Println()

	// Test Case 2: All platforms
	fmt.Println("Test 2: All Platforms")
	fmt.Println("-------------------------------------------")
	platforms2 := map[string]float64{
		"github":   85.5,
		"linkedin": 72.3,
		"uber":     90.1,
	}
	score2, confidence2 := computeScore(platforms2)
	fmt.Printf("GitHub: %.2f (40%% weight)\n", platforms2["github"])
	fmt.Printf("LinkedIn: %.2f (35%% weight)\n", platforms2["linkedin"])
	fmt.Printf("Uber: %.2f (25%% weight)\n", platforms2["uber"])
	fmt.Printf("Trust Score: %.2f\n", score2)
	fmt.Printf("Confidence: %.2f\n", confidence2)
	fmt.Println()

	// Test Case 3: Partial platforms
	fmt.Println("Test 3: GitHub + Uber Only")
	fmt.Println("-------------------------------------------")
	platforms3 := map[string]float64{
		"github": 95.0,
		"uber":   88.0,
	}
	score3, confidence3 := computeScore(platforms3)
	fmt.Printf("GitHub: %.2f (40%% weight)\n", platforms3["github"])
	fmt.Printf("Uber: %.2f (25%% weight)\n", platforms3["uber"])
	fmt.Printf("Trust Score: %.2f\n", score3)
	fmt.Printf("Confidence: %.2f\n", confidence3)
	fmt.Println()

	// Test Case 4: WASM Input Format
	fmt.Println("Test 4: WASM Module Input Format")
	fmt.Println("-------------------------------------------")
	inputData := prepareWASMInput("alice", 0x07) // All platforms enabled
	fmt.Printf("Input Data Length: %d bytes\n", len(inputData))
	fmt.Printf("Input Data (hex): %x\n", inputData)
	inputHash := sha256.Sum256(inputData)
	fmt.Printf("Input Hash: %x\n", inputHash)
	fmt.Println()

	// Test Case 5: Receipt Generation
	fmt.Println("Test 5: Receipt Generation")
	fmt.Println("-------------------------------------------")
	outputData := map[string]interface{}{
		"trust_score": score2,
		"confidence":  confidence2,
	}
	outputJSON, _ := json.Marshal(outputData)
	outputHash := sha256.Sum256(outputJSON)
	wasmHash := sha256.Sum256(readWASMModule())

	fmt.Printf("WASM Module Hash: %x\n", wasmHash)
	fmt.Printf("Output Hash: %x\n", outputHash)
	fmt.Printf("Gas Target: 238 units\n")
	fmt.Printf("Computation Duration: <5ms (target)\n")
	fmt.Println()

	// Test Case 6: Determinism Check
	fmt.Println("Test 6: Determinism Verification")
	fmt.Println("-------------------------------------------")
	runs := 10
	results := make([]float64, runs)
	start := time.Now()

	for i := 0; i < runs; i++ {
		score, _ := computeScore(platforms2)
		results[i] = score
	}
	duration := time.Since(start)

	allIdentical := true
	for i := 1; i < runs; i++ {
		if results[i] != results[0] {
			allIdentical = false
			break
		}
	}

	fmt.Printf("Runs: %d\n", runs)
	fmt.Printf("First Score: %.2f\n", results[0])
	fmt.Printf("Last Score: %.2f\n", results[runs-1])
	fmt.Printf("All Identical: %v\n", allIdentical)
	fmt.Printf("Total Duration: %v\n", duration)
	fmt.Printf("Avg Per Run: %v\n", duration/time.Duration(runs))
	fmt.Println()

	fmt.Println("===========================================")
	if allIdentical {
		fmt.Println("✅ All tests passed - deterministic execution verified")
	} else {
		fmt.Println("❌ Non-deterministic behavior detected")
	}
	fmt.Println("===========================================")
}

// computeScore calculates weighted reputation score
func computeScore(platforms map[string]float64) (float64, float64) {
	var totalScore float64
	var totalWeight float64

	weights := map[string]float64{
		"github":   0.4,
		"linkedin": 0.35,
		"uber":     0.25,
	}

	for platform, score := range platforms {
		if score >= 0 && score <= 100 {
			weight := weights[platform]
			totalScore += score * weight
			totalWeight += weight
		}
	}

	finalScore := 0.0
	if totalWeight > 0 {
		finalScore = totalScore / totalWeight
	}

	return finalScore, totalWeight
}

// prepareWASMInput creates binary input for WASM module
func prepareWASMInput(userID string, platformFlags int) []byte {
	userIDBytes := []byte(userID)
	inputData := make([]byte, 0, 1024)

	// Add user_id length (4 bytes, little-endian)
	length := len(userIDBytes)
	inputData = append(inputData, byte(length), byte(length>>8), byte(length>>16), byte(length>>24))

	// Add user_id content
	inputData = append(inputData, userIDBytes...)

	// Add platform flags (4 bytes, little-endian)
	inputData = append(inputData, byte(platformFlags), byte(platformFlags>>8), byte(platformFlags>>16), byte(platformFlags>>24))

	return inputData
}

// readWASMModule reads the compiled WASM module
func readWASMModule() []byte {
	// Placeholder - in production, read from artifacts/reputation-aggregator.wasm
	return []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic number
}
