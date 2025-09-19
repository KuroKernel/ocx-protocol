// OCX Killer Applications Demo
// Demonstrates the power of OCX protocol with real-world applications

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ocx.local/pkg/executor"
	"ocx.local/pkg/programs"
)

func main() {
	fmt.Println("🚀 OCX Killer Applications Demo")
	fmt.Println("================================")
	fmt.Println()

	// Run all killer programs
	for _, program := range programs.KillerPrograms {
		fmt.Printf("Running: %s\n", program.Name)
		fmt.Printf("Description: %s\n", program.Description)
		fmt.Println("----------------------------------------")
		
		// Execute the program
		start := time.Now()
		result, err := runKillerProgram(program)
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Success! Execution time: %v\n", duration)
			fmt.Printf("Cycles used: %d\n", result.Receipt.CyclesUsed)
			fmt.Printf("Price: %d micro-units\n", result.Receipt.Price)
			fmt.Printf("Output size: %d bytes\n", len(result.Output))
			
			// Show first 16 bytes of output as hex
			if len(result.Output) > 0 {
				fmt.Printf("Output (first 16 bytes): %x\n", result.Output[:min(16, len(result.Output))])
			}
			
			// Verify the receipt
			verified := executor.OCX_VERIFY(result.Receipt)
			fmt.Printf("Receipt verified: %t\n", verified)
			
			// Get accounting info
			payer, payee, amount := executor.OCX_ACCOUNT(result.Receipt)
			fmt.Printf("Accounting - Payer: %s, Payee: %s, Amount: %d\n", payer, payee, amount)
		}
		
		fmt.Println()
	}
	
	fmt.Println("🎉 All killer applications completed!")
}

func runKillerProgram(program programs.KillerProgram) (*executor.OCXResult, error) {
	// Create input
	input := executor.OCXInput{
		Code:      program.Bytecode,
		Data:      program.TestData,
		MaxCycles: program.Expected.CyclesMax,
	}
	
	// Execute
	result, err := executor.OCX_EXEC(input)
	if err != nil {
		return nil, err
	}
	
	// Validate output size
	if len(result.Output) != program.Expected.OutputSize {
		return nil, fmt.Errorf("output size mismatch: expected %d, got %d", 
			program.Expected.OutputSize, len(result.Output))
	}
	
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Pretty print JSON for debugging
func prettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("Error marshaling: %v", err)
		return
	}
	fmt.Println(string(b))
}
