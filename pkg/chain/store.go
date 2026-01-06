package chain

import (
	"context"
	"fmt"
	"sync"
)

// Store defines the interface for receipt chain storage
type Store interface {
	// SaveReceipt stores a chained receipt
	SaveReceipt(ctx context.Context, receipt *ChainedReceipt) error

	// GetReceiptByHash retrieves a receipt by its hash
	GetReceiptByHash(ctx context.Context, hash [32]byte) (*ChainedReceipt, error)

	// GetChainHead retrieves the most recent receipt in a chain
	GetChainHead(ctx context.Context, chainID string) (*ChainedReceipt, error)

	// GetChainReceipts retrieves all receipts in a chain
	GetChainReceipts(ctx context.Context, chainID string, limit int) ([]ChainedReceipt, error)

	// GetAncestors retrieves receipts from hash back to genesis (or limit)
	GetAncestors(ctx context.Context, hash [32]byte, limit int) ([]ChainedReceipt, error)

	// GetStats returns chain statistics
	GetStats(ctx context.Context) (*ChainStats, error)

	// HasReceipt checks if a receipt exists
	HasReceipt(ctx context.Context, hash [32]byte) (bool, error)
}

// MemoryStore implements Store with in-memory storage (for testing/development)
type MemoryStore struct {
	mu       sync.RWMutex
	receipts map[[32]byte]*ChainedReceipt
	chains   map[string][32]byte // chainID -> head receipt hash
}

// NewMemoryStore creates a new in-memory chain store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		receipts: make(map[[32]byte]*ChainedReceipt),
		chains:   make(map[string][32]byte),
	}
}

// SaveReceipt stores a chained receipt in memory
func (s *MemoryStore) SaveReceipt(ctx context.Context, receipt *ChainedReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the receipt
	s.receipts[receipt.ReceiptHash] = receipt

	// Update chain head if chain ID is specified
	if receipt.ChainID != "" {
		s.chains[receipt.ChainID] = receipt.ReceiptHash
	}

	return nil
}

// GetReceiptByHash retrieves a receipt by its hash
func (s *MemoryStore) GetReceiptByHash(ctx context.Context, hash [32]byte) (*ChainedReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	receipt, exists := s.receipts[hash]
	if !exists {
		return nil, fmt.Errorf("receipt not found: %s", HashToHex(hash))
	}

	return receipt, nil
}

// GetChainHead retrieves the most recent receipt in a chain
func (s *MemoryStore) GetChainHead(ctx context.Context, chainID string) (*ChainedReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	headHash, exists := s.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	receipt, exists := s.receipts[headHash]
	if !exists {
		return nil, fmt.Errorf("chain head receipt not found: %s", HashToHex(headHash))
	}

	return receipt, nil
}

// GetChainReceipts retrieves all receipts in a chain
func (s *MemoryStore) GetChainReceipts(ctx context.Context, chainID string, limit int) ([]ChainedReceipt, error) {
	head, err := s.GetChainHead(ctx, chainID)
	if err != nil {
		return nil, err
	}

	return s.GetAncestors(ctx, head.ReceiptHash, limit)
}

// GetAncestors retrieves receipts from hash back to genesis (or limit)
func (s *MemoryStore) GetAncestors(ctx context.Context, hash [32]byte, limit int) ([]ChainedReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ancestors []ChainedReceipt
	currentHash := hash
	visited := make(map[[32]byte]bool)

	for {
		// Check for cycles
		if visited[currentHash] {
			return nil, fmt.Errorf("cycle detected in chain at %s", HashToHex(currentHash))
		}
		visited[currentHash] = true

		// Get current receipt
		receipt, exists := s.receipts[currentHash]
		if !exists {
			// Can't continue - ancestor not found
			break
		}

		ancestors = append(ancestors, *receipt)

		// Check if we've reached genesis or limit
		if receipt.PrevReceiptHash == nil {
			break
		}
		if limit > 0 && len(ancestors) >= limit {
			break
		}

		currentHash = *receipt.PrevReceiptHash
	}

	return ancestors, nil
}

// GetStats returns chain statistics
func (s *MemoryStore) GetStats(ctx context.Context) (*ChainStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &ChainStats{
		TotalChains:   int64(len(s.chains)),
		TotalReceipts: int64(len(s.receipts)),
	}

	// Calculate chain lengths
	var totalLength int64
	for chainID := range s.chains {
		receipts, err := s.GetChainReceipts(ctx, chainID, 0)
		if err == nil {
			chainLen := len(receipts)
			totalLength += int64(chainLen)
			if chainLen > stats.LongestChain {
				stats.LongestChain = chainLen
			}
		}
	}

	if stats.TotalChains > 0 {
		stats.AvgChainLength = float64(totalLength) / float64(stats.TotalChains)
	}

	return stats, nil
}

// HasReceipt checks if a receipt exists
func (s *MemoryStore) HasReceipt(ctx context.Context, hash [32]byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.receipts[hash]
	return exists, nil
}

// Clear removes all receipts (for testing)
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.receipts = make(map[[32]byte]*ChainedReceipt)
	s.chains = make(map[string][32]byte)
}

// Count returns the number of stored receipts (for testing)
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.receipts)
}
