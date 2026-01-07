package verify

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ReplayStore provides nonce tracking for replay protection
type ReplayStore interface {
	// CheckAndStore returns true if nonce is new (not seen before)
	// and stores it atomically. Returns false if nonce was already seen.
	CheckAndStore(nonce []byte, issuedAt uint64) (bool, error)
	// Cleanup removes expired nonces (older than retention period)
	Cleanup() error
}

// InMemoryReplayStore is a thread-safe in-memory replay protection store
type InMemoryReplayStore struct {
	mu        sync.RWMutex
	seen      map[string]uint64 // nonce hex -> issued_at timestamp
	retention time.Duration     // how long to keep nonces
	clockSkew time.Duration     // allowed clock skew
}

// NewInMemoryReplayStore creates a new in-memory replay store
func NewInMemoryReplayStore(retention, clockSkew time.Duration) *InMemoryReplayStore {
	if retention == 0 {
		retention = 7 * 24 * time.Hour // Default 7 days
	}
	if clockSkew == 0 {
		clockSkew = 5 * time.Minute // Default 5 minutes
	}
	return &InMemoryReplayStore{
		seen:      make(map[string]uint64),
		retention: retention,
		clockSkew: clockSkew,
	}
}

// CheckAndStore checks if nonce is new and stores it atomically
func (s *InMemoryReplayStore) CheckAndStore(nonce []byte, issuedAt uint64) (bool, error) {
	if len(nonce) != 16 {
		return false, fmt.Errorf("invalid nonce length: expected 16, got %d", len(nonce))
	}

	// Validate timestamp is within acceptable bounds
	now := uint64(time.Now().UnixNano())
	maxSkew := uint64(s.clockSkew.Nanoseconds())

	// Check if issuedAt is too far in the future
	if issuedAt > now+maxSkew {
		return false, fmt.Errorf("issued_at is too far in the future: %d > %d + %d", issuedAt, now, maxSkew)
	}

	// Check if issuedAt is too old (beyond retention)
	retentionNs := uint64(s.retention.Nanoseconds())
	if issuedAt < now-retentionNs {
		return false, fmt.Errorf("receipt is too old: issued_at %d is beyond retention window", issuedAt)
	}

	nonceHex := hex.EncodeToString(nonce)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if nonce already seen
	if _, exists := s.seen[nonceHex]; exists {
		return false, nil // Replay detected
	}

	// Store nonce with timestamp
	s.seen[nonceHex] = issuedAt
	return true, nil
}

// Cleanup removes expired nonces
func (s *InMemoryReplayStore) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixNano())
	retentionNs := uint64(s.retention.Nanoseconds())
	cutoff := now - retentionNs

	for nonce, issuedAt := range s.seen {
		if issuedAt < cutoff {
			delete(s.seen, nonce)
		}
	}

	return nil
}

// Size returns the number of tracked nonces (for monitoring)
func (s *InMemoryReplayStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.seen)
}

// StartCleanupLoop starts a background goroutine that periodically cleans up expired nonces
func (s *InMemoryReplayStore) StartCleanupLoop(interval time.Duration) chan struct{} {
	if interval == 0 {
		interval = 1 * time.Hour
	}
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-stopCh:
				return
			}
		}
	}()
	return stopCh
}
