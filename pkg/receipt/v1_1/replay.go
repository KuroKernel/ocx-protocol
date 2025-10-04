package v1_1

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// ReplayProtection manages nonce-based replay protection
type ReplayProtection struct {
	db            *sql.DB
	retention     time.Duration
	clockSkew     time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	mu            sync.RWMutex
}

// NewReplayProtection creates a new replay protection manager
func NewReplayProtection(db *sql.DB, retention, clockSkew time.Duration) *ReplayProtection {
	return &ReplayProtection{
		db:        db,
		retention: retention,
		clockSkew: clockSkew,
	}
}

// StartCleanup starts the automatic cleanup of expired nonces
func (rp *ReplayProtection) StartCleanup(ctx context.Context) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.cleanupTicker != nil {
		return fmt.Errorf("cleanup already started")
	}

	// Clean up expired entries every hour
	rp.cleanupTicker = time.NewTicker(time.Hour)
	rp.stopCleanup = make(chan struct{})

	go func() {
		for {
			select {
			case <-rp.cleanupTicker.C:
				if err := rp.cleanupExpired(ctx); err != nil {
					// Log error but continue
					fmt.Printf("Warning: failed to cleanup expired nonces: %v\n", err)
				}
			case <-rp.stopCleanup:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// StopCleanup stops the automatic cleanup
func (rp *ReplayProtection) StopCleanup() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.cleanupTicker != nil {
		rp.cleanupTicker.Stop()
		rp.cleanupTicker = nil
	}

	if rp.stopCleanup != nil {
		close(rp.stopCleanup)
		rp.stopCleanup = nil
	}
}

// CheckAndRecordNonce checks if a nonce has been used and records it if not
func (rp *ReplayProtection) CheckAndRecordNonce(ctx context.Context, issuerID string, nonce [16]byte, issuedAt time.Time) error {
	// Check clock skew
	now := time.Now()
	skew := now.Sub(issuedAt)
	if skew < -rp.clockSkew || skew > rp.clockSkew {
		return fmt.Errorf("clock skew too large: %v (max allowed: %v)", skew, rp.clockSkew)
	}

	// Check if nonce already exists
	exists, err := rp.nonceExists(ctx, issuerID, nonce)
	if err != nil {
		return fmt.Errorf("failed to check nonce existence: %w", err)
	}

	if exists {
		return fmt.Errorf("nonce already used: replay attack detected")
	}

	// Record the nonce
	expiresAt := now.Add(rp.retention)
	err = rp.recordNonce(ctx, issuerID, nonce, now, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to record nonce: %w", err)
	}

	return nil
}

// nonceExists checks if a nonce has already been used
func (rp *ReplayProtection) nonceExists(ctx context.Context, issuerID string, nonce [16]byte) (bool, error) {
	query := `
		SELECT COUNT(*) FROM ocx_replay_protection 
		WHERE issuer_id = $1 AND nonce = $2 AND expires_at > NOW()
	`

	var count int
	err := rp.db.QueryRowContext(ctx, query, issuerID, nonce[:]).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// recordNonce records a new nonce
func (rp *ReplayProtection) recordNonce(ctx context.Context, issuerID string, nonce [16]byte, usedAt, expiresAt time.Time) error {
	query := `
		INSERT INTO ocx_replay_protection (issuer_id, nonce, used_at, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (issuer_id, nonce) DO NOTHING
	`

	_, err := rp.db.ExecContext(ctx, query, issuerID, nonce[:], usedAt, expiresAt)
	return err
}

// cleanupExpired removes expired nonce entries
func (rp *ReplayProtection) cleanupExpired(ctx context.Context) error {
	query := `DELETE FROM ocx_replay_protection WHERE expires_at < NOW()`

	result, err := rp.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d expired nonce entries\n", rowsAffected)
	}

	return nil
}

// GetStats returns statistics about replay protection
func (rp *ReplayProtection) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total nonces
	var totalNonces int
	err := rp.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ocx_replay_protection WHERE expires_at > NOW()").Scan(&totalNonces)
	if err != nil {
		return nil, err
	}
	stats["total_active_nonces"] = totalNonces

	// Count nonces by issuer
	rows, err := rp.db.QueryContext(ctx, `
		SELECT issuer_id, COUNT(*) 
		FROM ocx_replay_protection 
		WHERE expires_at > NOW() 
		GROUP BY issuer_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issuerCounts := make(map[string]int)
	for rows.Next() {
		var issuerID string
		var count int
		if err := rows.Scan(&issuerID, &count); err != nil {
			return nil, err
		}
		issuerCounts[issuerID] = count
	}
	stats["nonces_by_issuer"] = issuerCounts

	// Get oldest and newest nonce timestamps
	var oldest, newest time.Time
	err = rp.db.QueryRowContext(ctx, `
		SELECT MIN(used_at), MAX(used_at) 
		FROM ocx_replay_protection 
		WHERE expires_at > NOW()
	`).Scan(&oldest, &newest)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if !oldest.IsZero() {
		stats["oldest_nonce"] = oldest
		stats["newest_nonce"] = newest
	}

	stats["retention_period"] = rp.retention
	stats["clock_skew_tolerance"] = rp.clockSkew

	return stats, nil
}

// CreateReplayProtectionTable creates the database table for replay protection
func CreateReplayProtectionTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS ocx_replay_protection (
			issuer_id VARCHAR(255) NOT NULL,
			nonce BYTEA NOT NULL,
			used_at TIMESTAMP WITH TIME ZONE NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (issuer_id, nonce)
		);
		
		CREATE INDEX IF NOT EXISTS idx_ocx_replay_protection_expires_at 
		ON ocx_replay_protection (expires_at);
		
		CREATE INDEX IF NOT EXISTS idx_ocx_replay_protection_issuer_id 
		ON ocx_replay_protection (issuer_id);
	`

	_, err := db.Exec(query)
	return err
}
