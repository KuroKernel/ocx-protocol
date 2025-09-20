package main

import (
	"encoding/base64"
	"fmt"
	"ocx.local/pkg/receipt"
)

func main() {
	receiptBlob := "qWF2AWVpbnB1dFggSG6kYiTRu0+2gPNPfJrZao8k7Ii+c+qOWmxlJg6cuKdmY3ljbGVzGQPoZmlzc3VlclggAQIDBAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABmb3V0cHV0WCC0iH+K7Ki6gIpc03vUAGKOb5LreXYfFDuul3vwUK8qpmhhcnRpZmFjdFggLPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCRobWV0ZXJpbmejYWEKYWIBYWcBaXNpZ25hdHVyZVhAAQIDBAUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGp0cmFuc2NyaXB0WCCBfxLXN/rjgHHvnS8NDjJDvppYp/zFeRM6fVPigEOJPQ=="
	
	// Server logic
	receiptData := []byte(receiptBlob)
	
	// Try to decode as base64 first
	if decoded, err := base64.StdEncoding.DecodeString(receiptBlob); err == nil {
		receiptData = decoded
		fmt.Printf("Base64 decoded successfully, length: %d\n", len(receiptData))
	} else {
		fmt.Printf("Base64 decode failed: %v\n", err)
	}
	
	// Deserialize receipt
	receipt, err := receipt.Deserialize(receiptData)
	if err != nil {
		fmt.Printf("Deserialize error: %v\n", err)
		return
	}
	
	fmt.Printf("Deserialized successfully: %+v\n", receipt)
}
