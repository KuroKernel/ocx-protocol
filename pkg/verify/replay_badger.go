package verify

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// BadgerReplayStore is a persistent replay protection store using BadgerDB
// Survives restarts, handles millions of nonces efficiently
type BadgerReplayStore struct {
	db        *badger.DB
	retention time.Duration
	clockSkew time.Duration
}

// BadgerReplayStoreConfig contains configuration for BadgerReplayStore
type BadgerReplayStoreConfig struct {
	Path      string        // Database directory path
	Retention time.Duration // How long to keep nonces (default 7 days)
	ClockSkew time.Duration // Allowed clock skew (default 5 minutes)
	InMemory  bool          // Use in-memory mode (for testing)
}

// NewBadgerReplayStore creates a new persistent replay store
func NewBadgerReplayStore(cfg BadgerReplayStoreConfig) (*BadgerReplayStore, error) {
	if cfg.Retention == 0 {
		cfg.Retention = 7 * 24 * time.Hour
	}
	if cfg.ClockSkew == 0 {
		cfg.ClockSkew = 5 * time.Minute
	}

	opts := badger.DefaultOptions(cfg.Path).
		WithLoggingLevel(badger.WARNING).
		WithNumVersionsToKeep(1).
		WithCompactL0OnClose(true).
		WithValueLogFileSize(64 << 20) // 64MB value log files

	if cfg.InMemory {
		opts = opts.WithInMemory(true)
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	store := &BadgerReplayStore{
		db:        db,
		retention: cfg.Retention,
		clockSkew: cfg.ClockSkew,
	}

	// Start background GC
	go store.runGC()

	return store, nil
}

// CheckAndStore checks if nonce is new and stores it atomically
func (s *BadgerReplayStore) CheckAndStore(nonce []byte, issuedAt uint64) (bool, error) {
	if len(nonce) != 16 {
		return false, fmt.Errorf("invalid nonce length: expected 16, got %d", len(nonce))
	}

	// Validate timestamp bounds
	now := uint64(time.Now().UnixNano())
	maxSkew := uint64(s.clockSkew.Nanoseconds())

	if issuedAt > now+maxSkew {
		return false, fmt.Errorf("issued_at is too far in the future")
	}

	retentionNs := uint64(s.retention.Nanoseconds())
	if issuedAt < now-retentionNs {
		return false, fmt.Errorf("receipt is too old: beyond retention window")
	}

	key := s.nonceKey(nonce)

	// Use transaction for atomicity
	err := s.db.Update(func(txn *badger.Txn) error {
		// Check if exists
		_, err := txn.Get(key)
		if err == nil {
			// Key exists - replay detected
			return fmt.Errorf("REPLAY")
		}
		if err != badger.ErrKeyNotFound {
			return err
		}

		// Store with TTL
		value := make([]byte, 8)
		binary.BigEndian.PutUint64(value, issuedAt)

		entry := badger.NewEntry(key, value).WithTTL(s.retention)
		return txn.SetEntry(entry)
	})

	if err != nil {
		if err.Error() == "REPLAY" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Cleanup is a no-op for BadgerDB as TTL handles expiration automatically
func (s *BadgerReplayStore) Cleanup() error {
	return nil
}

// Close closes the database
func (s *BadgerReplayStore) Close() error {
	return s.db.Close()
}

// Size returns approximate number of keys (for monitoring)
func (s *BadgerReplayStore) Size() (int64, error) {
	var count int64
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("nonce:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

// nonceKey creates the database key for a nonce
func (s *BadgerReplayStore) nonceKey(nonce []byte) []byte {
	return []byte("nonce:" + hex.EncodeToString(nonce))
}

// runGC runs BadgerDB garbage collection periodically
func (s *BadgerReplayStore) runGC() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		for {
			// Run GC until it returns nil (no more to clean)
			err := s.db.RunValueLogGC(0.5)
			if err != nil {
				break
			}
		}
	}
}

// BatchCheckAndStore checks multiple nonces efficiently in a single transaction
// Returns a slice of booleans indicating which nonces are new
func (s *BadgerReplayStore) BatchCheckAndStore(nonces [][]byte, issuedAts []uint64) ([]bool, error) {
	if len(nonces) != len(issuedAts) {
		return nil, fmt.Errorf("nonces and issuedAts length mismatch")
	}

	results := make([]bool, len(nonces))
	now := uint64(time.Now().UnixNano())
	maxSkew := uint64(s.clockSkew.Nanoseconds())
	retentionNs := uint64(s.retention.Nanoseconds())

	err := s.db.Update(func(txn *badger.Txn) error {
		for i, nonce := range nonces {
			if len(nonce) != 16 {
				results[i] = false
				continue
			}

			issuedAt := issuedAts[i]

			// Validate timestamp
			if issuedAt > now+maxSkew || issuedAt < now-retentionNs {
				results[i] = false
				continue
			}

			key := s.nonceKey(nonce)

			// Check if exists
			_, err := txn.Get(key)
			if err == nil {
				// Replay
				results[i] = false
				continue
			}
			if err != badger.ErrKeyNotFound {
				return err
			}

			// Store
			value := make([]byte, 8)
			binary.BigEndian.PutUint64(value, issuedAt)
			entry := badger.NewEntry(key, value).WithTTL(s.retention)
			if err := txn.SetEntry(entry); err != nil {
				return err
			}

			results[i] = true
		}
		return nil
	})

	return results, err
}

// Stats returns database statistics for monitoring
func (s *BadgerReplayStore) Stats() map[string]interface{} {
	lsm, vlog := s.db.Size()
	return map[string]interface{}{
		"lsm_size_bytes":   lsm,
		"vlog_size_bytes":  vlog,
		"total_size_bytes": lsm + vlog,
	}
}
