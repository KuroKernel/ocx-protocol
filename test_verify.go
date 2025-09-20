package main

import (
	"encoding/base64"
	"fmt"
	"ocx.local/pkg/ocx"
	"ocx.local/pkg/receipt"
)

func main() {
	// Create a proper receipt with correct metering constants
	ocxReceipt := ocx.OCXReceipt{
		Version: 1,
		Artifact: [32]byte{44, 242, 77, 186, 95, 176, 163, 14, 38, 232, 59, 42, 197, 185, 226, 158, 27, 22, 30, 92, 31, 167, 66, 94, 115, 4, 51, 98, 147, 139, 152, 36},
		Input: [32]byte{72, 110, 164, 98, 36, 209, 187, 79, 182, 128, 243, 79, 124, 154, 217, 106, 143, 36, 236, 136, 190, 115, 234, 142, 90, 108, 101, 38, 14, 156, 184, 167},
		Output: [32]byte{180, 136, 127, 138, 236, 168, 186, 128, 138, 92, 211, 123, 212, 0, 98, 142, 111, 146, 235, 121, 118, 31, 20, 59, 174, 151, 123, 240, 80, 175, 42, 166},
		Cycles: 1000,
		Metering: ocx.Metering{
			Alpha: 10,
			Beta:  1,
			Gamma: 100,
		},
		Issuer: [32]byte{1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Signature: [64]byte{1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Transcript: [32]byte{129, 127, 18, 215, 55, 250, 227, 128, 113, 239, 157, 47, 13, 14, 50, 67, 190, 154, 88, 167, 252, 197, 121, 19, 58, 125, 83, 226, 128, 67, 137, 61},
	}
	
	// Create receipt wrapper
	receiptWrapper := &receipt.Receipt{OCXReceipt: &ocxReceipt}
	
	// Serialize the receipt
	blob, err := receiptWrapper.Serialize()
	if err != nil {
		fmt.Printf("Serialize error: %v\n", err)
		return
	}
	
	// Encode as base64
	base64Blob := base64.StdEncoding.EncodeToString(blob)
	fmt.Printf("Generated receipt blob: %s\n", base64Blob)
	
	// Test verification
	valid, reason := receiptWrapper.Verify()
	fmt.Printf("Verification result: valid=%v, reason=%s\n", valid, reason)
}
