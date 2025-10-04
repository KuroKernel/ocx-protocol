package receipt

// ReceiptCore represents the signed core fields of a receipt
type ReceiptCore struct {
	ProgramHash [32]byte `cbor:"1,keyasint"` // key 1
	InputHash   [32]byte `cbor:"2,keyasint"` // key 2
	OutputHash  [32]byte `cbor:"3,keyasint"` // key 3
	GasUsed     uint64   `cbor:"4,keyasint"` // key 4
	StartedAt   uint64   `cbor:"5,keyasint"` // key 5
	FinishedAt  uint64   `cbor:"6,keyasint"` // key 6
	IssuerID    string   `cbor:"7,keyasint"` // key 7
}

// ReceiptFull represents the complete receipt with metadata
type ReceiptFull struct {
	Core       ReceiptCore       `cbor:"core"`
	Signature  []byte            `cbor:"signature"` // 64B
	HostCycles uint64            `cbor:"host_cycles"`
	HostInfo   map[string]string `cbor:"host_info"`
	// Optional chaining fields can be added here
}
