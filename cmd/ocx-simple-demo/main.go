// OCX Simple Demo - Shows the basic concept without complex bytecode
package main

import (
	"fmt"
	"time"
	"ocx.local/pkg/executor"
)

func main() {
	fmt.Println("🚀 OCX Simple Demo")
	fmt.Println("==================")
	fmt.Println()

	// Demo 1: Simple computation
	fmt.Println("Demo 1: Simple Computation")
	fmt.Println("--------------------------")
	
	// Create a simple program that just returns the input data
	code := []byte{
		byte(executor.OP_HALT), // Just halt immediately
	}
	
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	input := executor.OCXInput{
		Code:      code,
		Data:      data,
		MaxCycles: 100,
	}
	
	start := time.Now()
	result, err := executor.OCX_EXEC(input)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Success! Execution time: %v\n", duration)
		fmt.Printf("Cycles used: %d\n", result.Receipt.CyclesUsed)
		fmt.Printf("Price: %d micro-units\n", result.Receipt.Price)
		fmt.Printf("Output size: %d bytes\n", len(result.Output))
		fmt.Printf("Output: %x\n", result.Output)
		
		// Verify the receipt
		verified := executor.OCX_VERIFY(result.Receipt)
		fmt.Printf("Receipt verified: %t\n", verified)
		
		// Get accounting info
		payer, payee, amount := executor.OCX_ACCOUNT(result.Receipt)
		fmt.Printf("Accounting - Payer: %s, Payee: %s, Amount: %d\n", payer, payee, amount)
	}
	
	fmt.Println()
	
	// Demo 2: Different input sizes
	fmt.Println("Demo 2: Different Input Sizes")
	fmt.Println("-----------------------------")
	
	testSizes := []int{16, 32, 64, 128, 256}
	
	for _, size := range testSizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		
		input := executor.OCXInput{
			Code:      code,
			Data:      data,
			MaxCycles: 100,
		}
		
		start := time.Now()
		result, err := executor.OCX_EXEC(input)
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("Size %d: ❌ Error: %v\n", size, err)
		} else {
			fmt.Printf("Size %d: ✅ Success! Time: %v, Cycles: %d, Price: %d\n", 
				size, duration, result.Receipt.CyclesUsed, result.Receipt.Price)
		}
	}
	
	fmt.Println()
	fmt.Println("🎉 Demo completed!")
	fmt.Println()
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("- Deterministic execution")
	fmt.Println("- Cryptographic receipts for every computation")
	fmt.Println("- Cycle-accurate metering and pricing")
	fmt.Println("- Verifiable computation results")
	fmt.Println("- Accounting information extraction")
}
