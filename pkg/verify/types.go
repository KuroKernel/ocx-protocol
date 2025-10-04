package verify

import "ocx.local/pkg/receipt"

// ReceiptFields represents extracted receipt data
type ReceiptFields struct {
	ProgramHash []byte            `json:"program_hash"`
	InputHash   []byte            `json:"input_hash"`
	OutputHash  []byte            `json:"output_hash"`
	GasUsed     uint64            `json:"gas_used"`
	StartedAt   uint64            `json:"started_at"`
	FinishedAt  uint64            `json:"finished_at"`
	IssuerID    string            `json:"issuer_id"`
	Signature   []byte            `json:"signature"`
	HostCycles  uint64            `json:"host_cycles"`
	HostInfo    map[string]string `json:"host_info"`
}

// ReceiptBatch represents a receipt for batch verification
type ReceiptBatch struct {
	ReceiptData []byte `json:"receipt_data"`
	PublicKey   []byte `json:"public_key"`
}

// Verifier interface for both Go and Rust implementations
type Verifier interface {
	VerifyReceipt(receiptData []byte, publicKey []byte) (*receipt.ReceiptCore, error)
	VerifyReceiptSimple(receiptData []byte) error
	ExtractReceiptFields(receiptData []byte) (*ReceiptFields, error)
	BatchVerify(receipts []ReceiptBatch) ([]bool, error)
	GetVersion() (string, error)
}
