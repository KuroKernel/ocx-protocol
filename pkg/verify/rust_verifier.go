//go:build rust_verifier
// +build rust_verifier

package verify

/*
#cgo LDFLAGS: -L${SRCDIR}/../../libocx-verify/target/release -llibocx_verify -ldl -lm
#include "../../libocx-verify/ocx_verify.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"time"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"ocx.local/pkg/receipt"
)

// RustVerifier implements receipt verification using the Rust libocx-verify library
type RustVerifier struct {
	enabled bool
}

// NewRustVerifier creates a new Rust-based verifier
func NewRustVerifier() Verifier {
	return &RustVerifier{
		enabled: true,
	}
}

// VerifyReceipt verifies a receipt using the Rust implementation
func (rv *RustVerifier) VerifyReceipt(receiptData []byte, publicKey []byte) (*receipt.ReceiptCore, error) {
	if !rv.enabled {
		return nil, fmt.Errorf("rust verifier is disabled")
	}

	if len(receiptData) == 0 {
		return nil, fmt.Errorf("empty receipt data")
	}

	if len(publicKey) != 32 {
		return nil, fmt.Errorf("invalid public key length: expected 32 bytes, got %d", len(publicKey))
	}

	// Call Rust verification function with detailed error reporting
	var errorCode C.OcxErrorCode
	result := C.ocx_verify_receipt_detailed(
		(*C.uint8_t)(unsafe.Pointer(&receiptData[0])),
		C.size_t(len(receiptData)),
		(*C.uint8_t)(unsafe.Pointer(&publicKey[0])),
		&errorCode,
	)

	if !result {
		// Get human-readable error message
		errorMsg := rv.getErrorMessage(errorCode)
		return nil, fmt.Errorf("rust verification failed: %s (code: %d)", errorMsg, int(errorCode))
	}

	// Extract receipt core from the verified receipt
	core, err := rv.extractReceiptCore(receiptData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract receipt core: %w", err)
	}

	return core, nil
}

// VerifyReceiptSimple verifies a receipt using embedded key ID (for testing)
func (rv *RustVerifier) VerifyReceiptSimple(receiptData []byte) error {
	if !rv.enabled {
		return fmt.Errorf("rust verifier is disabled")
	}

	if len(receiptData) == 0 {
		return fmt.Errorf("empty receipt data")
	}

	result := C.ocx_verify_receipt_simple(
		(*C.uint8_t)(unsafe.Pointer(&receiptData[0])),
		C.size_t(len(receiptData)),
	)

	if !result {
		return fmt.Errorf("rust simple verification failed")
	}

	return nil
}

// ExtractReceiptFields extracts receipt fields into Go structures
func (rv *RustVerifier) ExtractReceiptFields(receiptData []byte) (*ReceiptFields, error) {
	if !rv.enabled {
		return nil, fmt.Errorf("rust verifier is disabled")
	}

	if len(receiptData) == 0 {
		return nil, fmt.Errorf("empty receipt data")
	}

	var fields C.OcxReceiptFields
	keyIdBuffer := make([]byte, 256) // Buffer for key ID
	sigBuffer := make([]byte, 64)    // Buffer for signature

	errorCode := C.ocx_extract_receipt_fields(
		(*C.uint8_t)(unsafe.Pointer(&receiptData[0])),
		C.size_t(len(receiptData)),
		&fields,
		(*C.char)(unsafe.Pointer(&keyIdBuffer[0])),
		C.size_t(len(keyIdBuffer)),
		(*C.uint8_t)(unsafe.Pointer(&sigBuffer[0])),
		C.size_t(len(sigBuffer)),
	)

	if errorCode != C.OCX_SUCCESS {
		errorMsg := rv.getErrorMessage(errorCode)
		return nil, fmt.Errorf("failed to extract fields: %s", errorMsg)
	}

	// Convert C structures to Go structures
	result := &ReceiptFields{
		ProgramHash: make([]byte, 32),
		InputHash:   make([]byte, 32),
		OutputHash:  make([]byte, 32),
		GasUsed:     uint64(fields.cycles_used),
		StartedAt:   uint64(fields.started_at),
		FinishedAt:  uint64(fields.finished_at),
		IssuerID:    string(keyIdBuffer[:fields.issuer_key_id_len]),
		Signature:   sigBuffer[:fields.signature_len],
	}

	copy(result.ProgramHash, (*[32]byte)(unsafe.Pointer(&fields.artifact_hash))[:])
	copy(result.InputHash, (*[32]byte)(unsafe.Pointer(&fields.input_hash))[:])
	copy(result.OutputHash, (*[32]byte)(unsafe.Pointer(&fields.output_hash))[:])

	return result, nil
}

// BatchVerify verifies multiple receipts efficiently
func (rv *RustVerifier) BatchVerify(receipts []ReceiptBatch) ([]bool, error) {
	if !rv.enabled {
		return nil, fmt.Errorf("rust verifier is disabled")
	}

	if len(receipts) == 0 {
		return nil, fmt.Errorf("no receipts provided")
	}

	// Prepare C structures
	cReceipts := make([]C.OcxReceiptData, len(receipts))
	results := make([]bool, len(receipts))
	cResults := make([]C.bool, len(receipts))

	for i, receipt := range receipts {
		if len(receipt.PublicKey) != 32 {
			return nil, fmt.Errorf("invalid public key length for receipt %d", i)
		}

		cReceipts[i] = C.OcxReceiptData{
			cbor_data:     (*C.uint8_t)(unsafe.Pointer(&receipt.ReceiptData[0])),
			cbor_data_len: C.size_t(len(receipt.ReceiptData)),
			public_key:    (*C.uint8_t)(unsafe.Pointer(&receipt.PublicKey[0])),
		}
	}

	// Call batch verification
	_ = C.ocx_verify_receipts_batch(
		(*C.OcxReceiptData)(unsafe.Pointer(&cReceipts[0])),
		C.size_t(len(cReceipts)),
		(*C.bool)(unsafe.Pointer(&cResults[0])),
	)

	// Convert results
	for i, cResult := range cResults {
		results[i] = bool(cResult)
	}

	return results, nil
}

// GetVersion returns the Rust library version
func (rv *RustVerifier) GetVersion() (string, error) {
	buffer := make([]byte, 64)
	length := C.ocx_get_version(
		(*C.char)(unsafe.Pointer(&buffer[0])),
		C.size_t(len(buffer)),
	)

	if length == 0 {
		return "", fmt.Errorf("failed to get version")
	}

	return string(buffer[:length-1]), nil // Exclude null terminator
}

// getErrorMessage converts error code to human-readable message
func (rv *RustVerifier) getErrorMessage(errorCode C.OcxErrorCode) string {
	buffer := make([]byte, 256)
	length := C.ocx_get_error_message(
		errorCode,
		(*C.char)(unsafe.Pointer(&buffer[0])),
		C.size_t(len(buffer)),
	)

	if length == 0 {
		return fmt.Sprintf("Unknown error (code: %d)", int(errorCode))
	}

	return string(buffer[:length-1]) // Exclude null terminator
}

// IsEnabled returns whether the Rust verifier is enabled
func (rv *RustVerifier) IsEnabled() bool {
	return rv.enabled
}

// Enable enables the Rust verifier
func (rv *RustVerifier) Enable() {
	rv.enabled = true
}

// Disable disables the Rust verifier
func (rv *RustVerifier) Disable() {
	rv.enabled = false
}

// extractReceiptCore extracts the core fields from a verified receipt
func (rv *RustVerifier) extractReceiptCore(receiptData []byte) (*receipt.ReceiptCore, error) {
	// Parse the receipt data to extract core fields using JSON or CBOR
	var receiptMap map[string]interface{}
	if err := json.Unmarshal(receiptData, &receiptMap); err != nil {
		// If JSON parsing fails, attempt CBOR parsing
		var cborMap map[string]interface{}
		if cborErr := cbor.Unmarshal(receiptData, &cborMap); cborErr != nil {
			return nil, fmt.Errorf("failed to parse receipt as JSON or CBOR: json=%v, cbor=%v", err, cborErr)
		}
		receiptMap = cborMap
	}

	// Extract fields from the parsed receipt
	core := &receipt.ReceiptCore{
		Version:     getString(receiptMap, "v", "OCXv1"),
		TxID:        getString(receiptMap, "tx_id", ""),
		ArtifactID:  getString(receiptMap, "artifact_id", ""),
		ExecutionID: getString(receiptMap, "execution_id", ""),
		Timestamp:   getInt64(receiptMap, "ts", time.Now().Unix()),
		ExitCode:    getInt(receiptMap, "exit_code", 0),
		GasUsed:     getUint64(receiptMap, "gas_used", 1000),
		OutputHash:  getString(receiptMap, "output_hash", ""),
	}

	return core, nil
}

// Helper functions for extracting typed values from map
func getString(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return defaultValue
}

func getInt64(m map[string]interface{}, key string, defaultValue int64) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return defaultValue
}

func getUint64(m map[string]interface{}, key string, defaultValue uint64) uint64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case uint64:
			return v
		case int:
			return uint64(v)
		case int64:
			return uint64(v)
		case float64:
			return uint64(v)
		}
	}
	return defaultValue
}
