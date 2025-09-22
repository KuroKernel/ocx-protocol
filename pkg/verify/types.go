package verify

// ReceiptFields represents extracted receipt data
type ReceiptFields struct {
	ArtifactHash []byte `json:"artifact_hash"`
	InputHash    []byte `json:"input_hash"`
	OutputHash   []byte `json:"output_hash"`
	CyclesUsed   uint64 `json:"cycles_used"`
	StartedAt    uint64 `json:"started_at"`
	FinishedAt   uint64 `json:"finished_at"`
	IssuerKeyID  string `json:"issuer_key_id"`
	Signature    []byte `json:"signature"`
}

// ReceiptBatch represents a receipt for batch verification
type ReceiptBatch struct {
	ReceiptData []byte `json:"receipt_data"`
	PublicKey   []byte `json:"public_key"`
}

// Verifier interface for both Go and Rust implementations
type Verifier interface {
	VerifyReceipt(receiptData []byte, publicKey []byte) error
	VerifyReceiptSimple(receiptData []byte) error
	ExtractReceiptFields(receiptData []byte) (*ReceiptFields, error)
	BatchVerify(receipts []ReceiptBatch) ([]bool, error)
	GetVersion() (string, error)
}
