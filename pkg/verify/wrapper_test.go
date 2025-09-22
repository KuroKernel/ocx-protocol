package verify

import (
	"testing"
)

func TestGoVerifier(t *testing.T) {
	// Test the Go verifier (fallback)
	verifier := &GoVerifier{}
	
	// Test with empty receipt
	err := verifier.VerifyReceipt([]byte{}, []byte{})
	if err == nil {
		t.Error("Expected error for empty receipt")
	}
	
	// Test with invalid receipt
	err = verifier.VerifyReceipt([]byte{0x01, 0x02, 0x03}, []byte{})
	if err == nil {
		t.Error("Expected error for invalid receipt")
	}
}

func TestNewVerifier(t *testing.T) {
	// Test default verifier
	verifier := NewVerifier()
	if verifier == nil {
		t.Error("Expected verifier to not be nil")
	}
}

func TestVerifierInterface(t *testing.T) {
	// Test unified interface
	verifier := NewVerifier()
	err := verifier.VerifyReceipt([]byte{}, []byte{})
	if err == nil {
		t.Error("Expected error for empty receipt")
	}
}
