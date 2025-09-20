package main

import (
	"encoding/base64"
	"fmt"
	"ocx.local/pkg/receipt"
)

func main() {
	// Test the CBOR receipt blob
	receiptBlob := "qWF2AWVpbnB1dFggSG6kYiTRu0+2gPNPfJrZao8k7Ii+c+qOWmxlJg6cuKdmY3ljbGVzGQPoZmlzc3VlclggAQIDBAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABmb3V0cHV0WCC0iH+K7Ki6gIpc03vUAGKOb5LreXYfFDuul3vwUK8qpmhhcnRpZmFjdFggLPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCRobWV0ZXJpbmejYWEKYWIBYWcBaXNpZ25hdHVyZVhAAQIDBAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGp0cmFuc2NyaXB0WCCBfxLXN/rjgHHvnS8NDjJDvppYp/zFeRM6fVPigEOJPQ=="
	
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(receiptBlob)
	if err != nil {
		fmt.Printf("Base64 decode error: %v\n", err)
		return
	}
	
	fmt.Printf("Decoded data length: %d bytes\n", len(data))
	fmt.Printf("First 16 bytes: %x\n", data[:16])
	
	// Try to deserialize
	receipt, err := receipt.Deserialize(data)
	if err != nil {
		fmt.Printf("Deserialize error: %v\n", err)
		return
	}
	
	fmt.Printf("Deserialized successfully: %+v\n", receipt)
}
