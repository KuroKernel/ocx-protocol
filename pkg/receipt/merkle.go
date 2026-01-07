package receipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// MerkleTree represents a Merkle tree of receipt hashes
type MerkleTree struct {
	Root   [32]byte   // Root hash
	Leaves [][32]byte // Leaf hashes (receipt hashes)
	Levels [][][32]byte // All tree levels for proof generation
}

// MerkleProof is a compact proof that a receipt is in a batch
type MerkleProof struct {
	LeafHash  [32]byte   // Hash of the receipt
	LeafIndex int        // Position in the tree
	Siblings  [][32]byte // Sibling hashes along the path
	Root      [32]byte   // Root hash to verify against
}

// NewMerkleTree builds a Merkle tree from receipts
func NewMerkleTree(receipts []*ReceiptFull) (*MerkleTree, error) {
	if len(receipts) == 0 {
		return nil, fmt.Errorf("cannot create tree from empty receipts")
	}

	// Compute leaf hashes
	leaves := make([][32]byte, len(receipts))
	for i, r := range receipts {
		hash, err := HashReceipt(r)
		if err != nil {
			return nil, fmt.Errorf("failed to hash receipt %d: %w", i, err)
		}
		leaves[i] = hash
	}

	// Build tree
	tree := &MerkleTree{
		Leaves: leaves,
		Levels: make([][][32]byte, 0),
	}

	// First level is leaves
	currentLevel := leaves
	tree.Levels = append(tree.Levels, currentLevel)

	// Build up to root
	for len(currentLevel) > 1 {
		nextLevel := make([][32]byte, (len(currentLevel)+1)/2)

		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			var right [32]byte
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			} else {
				right = left // Duplicate last if odd
			}
			nextLevel[i/2] = hashPair(left, right)
		}

		tree.Levels = append(tree.Levels, nextLevel)
		currentLevel = nextLevel
	}

	tree.Root = currentLevel[0]
	return tree, nil
}

// hashPair computes SHA256(left || right) in canonical order
func hashPair(left, right [32]byte) [32]byte {
	// Always hash in deterministic order (smaller first)
	var data [64]byte
	if bytes.Compare(left[:], right[:]) <= 0 {
		copy(data[:32], left[:])
		copy(data[32:], right[:])
	} else {
		copy(data[:32], right[:])
		copy(data[32:], left[:])
	}
	return sha256.Sum256(data[:])
}

// GenerateProof generates a Merkle proof for a receipt at given index
func (t *MerkleTree) GenerateProof(index int) (*MerkleProof, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, fmt.Errorf("index %d out of range [0, %d)", index, len(t.Leaves))
	}

	proof := &MerkleProof{
		LeafHash:  t.Leaves[index],
		LeafIndex: index,
		Root:      t.Root,
		Siblings:  make([][32]byte, 0),
	}

	currentIndex := index
	for level := 0; level < len(t.Levels)-1; level++ {
		levelHashes := t.Levels[level]

		// Get sibling index
		var siblingIndex int
		if currentIndex%2 == 0 {
			siblingIndex = currentIndex + 1
		} else {
			siblingIndex = currentIndex - 1
		}

		// Get sibling hash (use self if no sibling)
		var sibling [32]byte
		if siblingIndex < len(levelHashes) {
			sibling = levelHashes[siblingIndex]
		} else {
			sibling = levelHashes[currentIndex]
		}

		proof.Siblings = append(proof.Siblings, sibling)
		currentIndex = currentIndex / 2
	}

	return proof, nil
}

// Verify verifies a Merkle proof
func (p *MerkleProof) Verify() bool {
	current := p.LeafHash
	index := p.LeafIndex

	for _, sibling := range p.Siblings {
		if index%2 == 0 {
			current = hashPair(current, sibling)
		} else {
			current = hashPair(sibling, current)
		}
		index = index / 2
	}

	return current == p.Root
}

// VerifyReceipt verifies a receipt is included in a batch with given root
func VerifyReceiptInBatch(receipt *ReceiptFull, proof *MerkleProof) (bool, error) {
	// Hash the receipt
	hash, err := HashReceipt(receipt)
	if err != nil {
		return false, fmt.Errorf("failed to hash receipt: %w", err)
	}

	// Check leaf hash matches
	if hash != proof.LeafHash {
		return false, nil
	}

	// Verify proof
	return proof.Verify(), nil
}

// HashReceipt computes the canonical hash of a receipt
func HashReceipt(receipt *ReceiptFull) ([32]byte, error) {
	// Use canonical CBOR of core for consistency
	coreBytes, err := CanonicalizeCore(&receipt.Core)
	if err != nil {
		return [32]byte{}, err
	}

	// Include signature in hash
	toHash := append(coreBytes, receipt.Signature...)
	return sha256.Sum256(toHash), nil
}

// MerkleRoot returns the hex-encoded root hash
func (t *MerkleTree) RootHex() string {
	return hex.EncodeToString(t.Root[:])
}

// ProofSize returns the number of siblings in proof (log2(n))
func (p *MerkleProof) ProofSize() int {
	return len(p.Siblings)
}

// Serialize serializes a proof to bytes
func (p *MerkleProof) Serialize() []byte {
	// Format: leafHash(32) + root(32) + index(4) + numSiblings(4) + siblings(32*n)
	size := 32 + 32 + 4 + 4 + (32 * len(p.Siblings))
	buf := make([]byte, size)

	copy(buf[0:32], p.LeafHash[:])
	copy(buf[32:64], p.Root[:])

	buf[64] = byte(p.LeafIndex >> 24)
	buf[65] = byte(p.LeafIndex >> 16)
	buf[66] = byte(p.LeafIndex >> 8)
	buf[67] = byte(p.LeafIndex)

	buf[68] = byte(len(p.Siblings) >> 24)
	buf[69] = byte(len(p.Siblings) >> 16)
	buf[70] = byte(len(p.Siblings) >> 8)
	buf[71] = byte(len(p.Siblings))

	offset := 72
	for _, sibling := range p.Siblings {
		copy(buf[offset:offset+32], sibling[:])
		offset += 32
	}

	return buf
}

// DeserializeProof deserializes a proof from bytes
func DeserializeProof(data []byte) (*MerkleProof, error) {
	if len(data) < 72 {
		return nil, fmt.Errorf("proof data too short")
	}

	proof := &MerkleProof{}
	copy(proof.LeafHash[:], data[0:32])
	copy(proof.Root[:], data[32:64])

	proof.LeafIndex = int(data[64])<<24 | int(data[65])<<16 | int(data[66])<<8 | int(data[67])
	numSiblings := int(data[68])<<24 | int(data[69])<<16 | int(data[70])<<8 | int(data[71])

	expectedLen := 72 + (32 * numSiblings)
	if len(data) < expectedLen {
		return nil, fmt.Errorf("proof data too short for %d siblings", numSiblings)
	}

	proof.Siblings = make([][32]byte, numSiblings)
	offset := 72
	for i := 0; i < numSiblings; i++ {
		copy(proof.Siblings[i][:], data[offset:offset+32])
		offset += 32
	}

	return proof, nil
}

// BatchRoot computes the Merkle root for a batch of receipts
// Convenience function when you just need the root
func BatchRoot(receipts []*ReceiptFull) ([32]byte, error) {
	tree, err := NewMerkleTree(receipts)
	if err != nil {
		return [32]byte{}, err
	}
	return tree.Root, nil
}

// MultiProof is an optimized proof for multiple receipts in same batch
type MultiProof struct {
	LeafIndices []int        // Indices of receipts being proven
	LeafHashes  [][32]byte   // Hashes of receipts
	Proof       [][32]byte   // Deduplicated sibling hashes
	Root        [32]byte     // Root to verify against
}

// GenerateMultiProof generates a proof for multiple receipts
func (t *MerkleTree) GenerateMultiProof(indices []int) (*MultiProof, error) {
	// Validate indices
	for _, idx := range indices {
		if idx < 0 || idx >= len(t.Leaves) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
	}

	// Collect leaf hashes
	leafHashes := make([][32]byte, len(indices))
	for i, idx := range indices {
		leafHashes[i] = t.Leaves[idx]
	}

	// For simplicity, generate individual proofs and deduplicate
	// A production implementation would use a more efficient algorithm
	seenHashes := make(map[[32]byte]bool)
	var proofHashes [][32]byte

	for _, idx := range indices {
		proof, _ := t.GenerateProof(idx)
		for _, sibling := range proof.Siblings {
			if !seenHashes[sibling] {
				seenHashes[sibling] = true
				proofHashes = append(proofHashes, sibling)
			}
		}
	}

	return &MultiProof{
		LeafIndices: indices,
		LeafHashes:  leafHashes,
		Proof:       proofHashes,
		Root:        t.Root,
	}, nil
}
