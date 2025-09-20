package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	"ocx.local/pkg/receipt"
)

func verifyCommand(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	receiptFlag := fs.String("receipt", "", "Base64-encoded receipt")
	fileFlag := fs.String("file", "", "Path to CBOR receipt file")
	
	fs.Parse(args)

	var receiptBytes []byte
	var err error

	switch {
	case *receiptFlag != "":
		receiptBytes, err = base64.StdEncoding.DecodeString(*receiptFlag)
		if err != nil {
			return fmt.Errorf("invalid base64 receipt: %v", err)
		}

	case *fileFlag != "":
		file, err := os.Open(*fileFlag)
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}
		defer file.Close()

		receiptBytes, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}

	default:
		return fmt.Errorf("either --receipt or --file must be specified")
	}

	// Verify receipt
	result, err := receipt.Verify(receiptBytes)
	if err != nil {
		fmt.Printf("❌ Invalid receipt: %v\n", err)
		return nil
	}

	fmt.Printf("✅ Valid receipt\n")
	fmt.Printf("   Issuer: %s\n", result.IssuerID)
	fmt.Printf("   Cycles: %d\n", result.Cycles)
	fmt.Printf("   Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05 UTC"))
	
	return nil
}
