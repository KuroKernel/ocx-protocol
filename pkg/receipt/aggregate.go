package receipt

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// AggregateReceipt represents multiple receipts combined with a single aggregate signature
// This allows verifying N receipts with O(1) signature verifications
type AggregateReceipt struct {
	Version       uint8           `cbor:"1,keyasint"`  // Aggregate format version
	Count         uint32          `cbor:"2,keyasint"`  // Number of receipts
	MerkleRoot    [32]byte        `cbor:"3,keyasint"`  // Root of receipt Merkle tree
	AggregateHash [32]byte        `cbor:"4,keyasint"`  // Hash of all receipt cores
	IssuerID      string          `cbor:"5,keyasint"`  // Issuer that created aggregate
	IssuedAt      uint64          `cbor:"6,keyasint"`  // When aggregate was created
	Signature     []byte          `cbor:"7,keyasint"`  // Ed25519 signature over aggregate
	ReceiptHashes [][32]byte      `cbor:"8,keyasint"`  // Individual receipt hashes (for proofs)
}

// AggregateBuilder builds aggregate receipts
type AggregateBuilder struct {
	receipts    []*ReceiptFull
	issuerID    string
	privateKey  ed25519.PrivateKey
}

// NewAggregateBuilder creates a new aggregate builder
func NewAggregateBuilder(issuerID string, privateKey ed25519.PrivateKey) *AggregateBuilder {
	return &AggregateBuilder{
		receipts:   make([]*ReceiptFull, 0),
		issuerID:   issuerID,
		privateKey: privateKey,
	}
}

// Add adds a receipt to the aggregate
func (ab *AggregateBuilder) Add(receipt *ReceiptFull) error {
	// Validate receipt has signature
	if len(receipt.Signature) != ed25519.SignatureSize {
		return fmt.Errorf("receipt must have valid signature")
	}
	ab.receipts = append(ab.receipts, receipt)
	return nil
}

// AddBatch adds multiple receipts
func (ab *AggregateBuilder) AddBatch(receipts []*ReceiptFull) error {
	for _, r := range receipts {
		if err := ab.Add(r); err != nil {
			return err
		}
	}
	return nil
}

// Build creates the aggregate receipt
func (ab *AggregateBuilder) Build() (*AggregateReceipt, error) {
	if len(ab.receipts) == 0 {
		return nil, fmt.Errorf("no receipts to aggregate")
	}

	// Compute individual receipt hashes
	hashes := make([][32]byte, len(ab.receipts))
	for i, r := range ab.receipts {
		hash, err := HashReceipt(r)
		if err != nil {
			return nil, fmt.Errorf("failed to hash receipt %d: %w", i, err)
		}
		hashes[i] = hash
	}

	// Build Merkle tree
	tree, err := NewMerkleTree(ab.receipts)
	if err != nil {
		return nil, fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Compute aggregate hash (hash of all receipt hashes in order)
	aggregateHash := computeAggregateHash(hashes)

	// Create aggregate receipt
	agg := &AggregateReceipt{
		Version:       1,
		Count:         uint32(len(ab.receipts)),
		MerkleRoot:    tree.Root,
		AggregateHash: aggregateHash,
		IssuerID:      ab.issuerID,
		IssuedAt:      uint64(time.Now().UnixNano()),
		ReceiptHashes: hashes,
	}

	// Sign the aggregate
	toSign := ab.aggregateSigningBytes(agg)
	agg.Signature = ed25519.Sign(ab.privateKey, toSign)

	return agg, nil
}

// aggregateSigningBytes creates the bytes to sign for an aggregate
func (ab *AggregateBuilder) aggregateSigningBytes(agg *AggregateReceipt) []byte {
	// Domain separator + version + count + merkle_root + aggregate_hash + issuer_id + issued_at
	var buf []byte
	buf = append(buf, []byte("OCXv1|aggregate|")...)
	buf = append(buf, agg.Version)

	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, agg.Count)
	buf = append(buf, countBytes...)

	buf = append(buf, agg.MerkleRoot[:]...)
	buf = append(buf, agg.AggregateHash[:]...)
	buf = append(buf, []byte(agg.IssuerID)...)

	issuedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(issuedBytes, agg.IssuedAt)
	buf = append(buf, issuedBytes...)

	return buf
}

// computeAggregateHash computes hash of all receipt hashes
func computeAggregateHash(hashes [][32]byte) [32]byte {
	// Concatenate all hashes and hash the result
	data := make([]byte, len(hashes)*32)
	for i, h := range hashes {
		copy(data[i*32:(i+1)*32], h[:])
	}
	return sha256.Sum256(data)
}

// VerifyAggregate verifies an aggregate receipt
func VerifyAggregate(agg *AggregateReceipt, publicKey ed25519.PublicKey) error {
	// Validate structure
	if agg.Version != 1 {
		return fmt.Errorf("unsupported aggregate version: %d", agg.Version)
	}

	if int(agg.Count) != len(agg.ReceiptHashes) {
		return fmt.Errorf("count mismatch: %d vs %d hashes", agg.Count, len(agg.ReceiptHashes))
	}

	if len(agg.Signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length")
	}

	// Recompute aggregate hash
	computedHash := computeAggregateHash(agg.ReceiptHashes)
	if computedHash != agg.AggregateHash {
		return fmt.Errorf("aggregate hash mismatch")
	}

	// Verify signature
	builder := &AggregateBuilder{issuerID: agg.IssuerID}
	toVerify := builder.aggregateSigningBytes(agg)

	if !ed25519.Verify(publicKey, toVerify, agg.Signature) {
		return fmt.Errorf("aggregate signature verification failed")
	}

	return nil
}

// VerifyReceiptInAggregate verifies a single receipt is included in an aggregate
func VerifyReceiptInAggregate(receipt *ReceiptFull, agg *AggregateReceipt, index int) error {
	if index < 0 || index >= len(agg.ReceiptHashes) {
		return fmt.Errorf("index out of range")
	}

	// Hash the receipt
	hash, err := HashReceipt(receipt)
	if err != nil {
		return fmt.Errorf("failed to hash receipt: %w", err)
	}

	// Check if hash matches
	if hash != agg.ReceiptHashes[index] {
		return fmt.Errorf("receipt hash mismatch at index %d", index)
	}

	return nil
}

// AggregateStats contains statistics about an aggregate
type AggregateStats struct {
	Count         int
	TotalGasUsed  uint64
	FirstIssuedAt uint64
	LastIssuedAt  uint64
	IssuerID      string
}

// GetStats returns statistics about the receipts in an aggregate
func (agg *AggregateReceipt) GetStats(receipts []*ReceiptFull) (*AggregateStats, error) {
	if len(receipts) != int(agg.Count) {
		return nil, fmt.Errorf("receipt count mismatch")
	}

	stats := &AggregateStats{
		Count:    len(receipts),
		IssuerID: agg.IssuerID,
	}

	for i, r := range receipts {
		stats.TotalGasUsed += r.Core.GasUsed

		if i == 0 || r.Core.IssuedAt < stats.FirstIssuedAt {
			stats.FirstIssuedAt = r.Core.IssuedAt
		}
		if r.Core.IssuedAt > stats.LastIssuedAt {
			stats.LastIssuedAt = r.Core.IssuedAt
		}
	}

	return stats, nil
}

// SerializeAggregate serializes an aggregate to bytes
func SerializeAggregate(agg *AggregateReceipt) ([]byte, error) {
	// Use CBOR for serialization
	return CanonicalizeFull(&ReceiptFull{
		Core: ReceiptCore{
			IssuerID: agg.IssuerID,
			IssuedAt: agg.IssuedAt,
		},
		Signature: agg.Signature,
	})
}

// AggregateProof is a proof that a receipt is in an aggregate
type AggregateProof struct {
	AggregateHash [32]byte     // Hash of the aggregate
	Index         int          // Position in aggregate
	MerkleProof   *MerkleProof // Merkle inclusion proof
}

// GenerateAggregateProof generates a proof for a receipt in an aggregate
func GenerateAggregateProof(receipts []*ReceiptFull, index int) (*AggregateProof, error) {
	tree, err := NewMerkleTree(receipts)
	if err != nil {
		return nil, err
	}

	merkleProof, err := tree.GenerateProof(index)
	if err != nil {
		return nil, err
	}

	hashes := make([][32]byte, len(receipts))
	for i, r := range receipts {
		h, _ := HashReceipt(r)
		hashes[i] = h
	}

	return &AggregateProof{
		AggregateHash: computeAggregateHash(hashes),
		Index:         index,
		MerkleProof:   merkleProof,
	}, nil
}

// VerifyAggregateProof verifies an aggregate inclusion proof
func VerifyAggregateProof(receipt *ReceiptFull, proof *AggregateProof) (bool, error) {
	return VerifyReceiptInBatch(receipt, proof.MerkleProof)
}
