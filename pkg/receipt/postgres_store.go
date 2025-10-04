package receipt

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore provides PostgreSQL-based receipt storage using pgx/v5
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL store using pgx/v5
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		pool: pool,
	}
}

// SaveReceipt stores a receipt in PostgreSQL
func (s *PostgresStore) SaveReceipt(ctx context.Context, r ReceiptFull, fullCBOR []byte) (string, error) {
	// Generate cryptographically secure receipt ID
	receiptID, err := s.generateReceiptID()
	if err != nil {
		return "", fmt.Errorf("failed to generate receipt ID: %w", err)
	}

	// Insert receipt into database
	const query = `
		INSERT INTO ocx_receipts (
			receipt_id, issuer_id, 
			program_hash, input_hash, output_hash, gas_used,
			started_at, finished_at, signature, host_cycles, host_info, receipt_cbor, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
	`

	// Convert host info to JSON
	hostInfoJSON := "{}"
	if r.HostInfo != nil {
		hostInfoBytes, err := json.Marshal(r.HostInfo)
		if err == nil {
			hostInfoJSON = string(hostInfoBytes)
		}
	}

	_, err = s.pool.Exec(ctx, query,
		receiptID,
		r.Core.IssuerID,
		r.Core.ProgramHash[:],
		r.Core.InputHash[:],
		r.Core.OutputHash[:],
		r.Core.GasUsed,
		r.Core.StartedAt,
		r.Core.FinishedAt,
		r.Signature,
		r.HostCycles,
		hostInfoJSON,
		fullCBOR,
	)

	if err != nil {
		return "", fmt.Errorf("failed to save receipt: %w", err)
	}

	return receiptID, nil
}

// generateReceiptID generates a cryptographically secure receipt ID as UUID
func (s *PostgresStore) generateReceiptID() (string, error) {
	// Generate 16 bytes of randomness (128 bits for UUID)
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Set version (4) and variant bits according to RFC 4122
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant bits

	// Format as UUID string
	receiptID := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])

	return receiptID, nil
}

// GetReceipt retrieves a receipt from PostgreSQL
func (s *PostgresStore) GetReceipt(ctx context.Context, id string) ([]byte, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	const query = `SELECT receipt_data FROM ocx_receipts WHERE receipt_id = $1`
	var receiptData []byte
	err := s.pool.QueryRow(ctx, query, id).Scan(&receiptData)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("receipt not found: %s", id)
		}
		return nil, fmt.Errorf("failed to query receipt: %w", err)
	}

	return receiptData, nil
}

// DeleteReceipt removes a receipt from PostgreSQL
func (s *PostgresStore) DeleteReceipt(ctx context.Context, id string) error {
	if s.pool == nil {
		return fmt.Errorf("database connection not available")
	}

	const query = `DELETE FROM ocx_receipts WHERE receipt_id = $1`
	tag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("receipt not found: %s", id)
	}

	return nil
}

// PutIdempotent stores idempotency data in PostgreSQL
func (s *PostgresStore) PutIdempotent(ctx context.Context, key string, reqHash [32]byte, respCBOR []byte) (bool, bool, []byte, error) {
	if s.pool == nil {
		return false, false, nil, fmt.Errorf("database connection not available")
	}

	// Check if key already exists
	const checkQuery = `SELECT request_hash, response_cbor FROM ocx_idempotency WHERE idempotency_key = $1`
	var existingReqHash [32]byte
	var existingRespCBOR []byte
	err := s.pool.QueryRow(ctx, checkQuery, key).Scan(&existingReqHash, &existingRespCBOR)

	if err == nil {
		// Key exists, check if request hash matches
		if existingReqHash == reqHash {
			return true, true, existingRespCBOR, nil // Found existing response
		}
		return true, false, nil, fmt.Errorf("idempotency key exists with different request hash")
	}

	// Key doesn't exist, insert new entry
	const insertQuery = `INSERT INTO ocx_idempotency (idempotency_key, request_hash, response_cbor, created_at) VALUES ($1, $2, $3, NOW())`
	_, err = s.pool.Exec(ctx, insertQuery, key, reqHash, respCBOR)
	if err != nil {
		return false, false, nil, fmt.Errorf("failed to store idempotency data: %w", err)
	}

	return false, false, nil, nil // New entry created
}

// GetIdempotent retrieves idempotency data from PostgreSQL
func (s *PostgresStore) GetIdempotent(ctx context.Context, key string) ([32]byte, []byte, bool, error) {
	if s.pool == nil {
		var emptyHash [32]byte
		return emptyHash, nil, false, fmt.Errorf("database connection not available")
	}

	const query = `SELECT request_hash, response_cbor FROM ocx_idempotency WHERE idempotency_key = $1`
	var reqHash [32]byte
	var respCBOR []byte
	err := s.pool.QueryRow(ctx, query, key).Scan(&reqHash, &respCBOR)

	if err != nil {
		var emptyHash [32]byte
		return emptyHash, nil, false, nil // Not found
	}

	return reqHash, respCBOR, true, nil
}

// GetReceiptByCore retrieves a receipt by its core fields from PostgreSQL
func (s *PostgresStore) GetReceiptByCore(ctx context.Context, core *ReceiptCore) (*ReceiptFull, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query by core fields (program_hash, input_hash, output_hash)
	const query = `SELECT receipt_id, issuer_id, program_hash, input_hash, output_hash, gas_used, started_at, finished_at, signature, created_at FROM ocx_receipts WHERE program_hash = $1 AND input_hash = $2 AND output_hash = $3 LIMIT 1`

	var receiptID, issuerID string
	var programHash, inputHash, outputHash []byte
	var gasUsed uint64
	var startedAt, finishedAt, createdAt time.Time
	var signature []byte

	err := s.pool.QueryRow(ctx, query, core.ProgramHash[:], core.InputHash[:], core.OutputHash[:]).Scan(
		&receiptID, &issuerID,
		&programHash, &inputHash, &outputHash, &gasUsed,
		&startedAt, &finishedAt, &signature, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to query receipt by core: %w", err)
	}

	// Reconstruct the receipt
	receiptCore := &ReceiptCore{
		ProgramHash: [32]byte(programHash),
		InputHash:   [32]byte(inputHash),
		OutputHash:  [32]byte(outputHash),
		GasUsed:     gasUsed,
		StartedAt:   uint64(startedAt.Unix()),
		FinishedAt:  uint64(finishedAt.Unix()),
		IssuerID:    issuerID,
	}

	return &ReceiptFull{
		Core:      *receiptCore,
		Signature: signature,
	}, nil
}

// ListReceipts lists receipts with pagination from PostgreSQL
func (s *PostgresStore) ListReceipts(ctx context.Context, limit, offset int) ([]ReceiptSummary, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	const query = `SELECT receipt_id, issuer_id, program_hash, gas_used, started_at, finished_at, created_at FROM ocx_receipts ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipts: %w", err)
	}
	defer rows.Close()

	var receipts []ReceiptSummary
	for rows.Next() {
		var summary ReceiptSummary
		var programHash []byte
		err := rows.Scan(&summary.ID, &summary.IssuerID, &programHash, &summary.GasUsed, &summary.StartedAt, &summary.FinishedAt, &summary.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan receipt: %w", err)
		}
		copy(summary.ProgramHash[:], programHash)
		receipts = append(receipts, summary)
	}

	return receipts, nil
}

// CleanupOldIdempotency cleans up old idempotency entries in PostgreSQL
func (s *PostgresStore) CleanupOldIdempotency(ctx context.Context, olderThan time.Duration) (int64, error) {
	if s.pool == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	const query = `DELETE FROM ocx_idempotency WHERE created_at < NOW() - INTERVAL '%d seconds'`
	tag, err := s.pool.Exec(ctx, fmt.Sprintf(query, int(olderThan.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old idempotency entries: %w", err)
	}

	return tag.RowsAffected(), nil
}

// GetStats returns store statistics from PostgreSQL
func (s *PostgresStore) GetStats(ctx context.Context) (*StoreStats, error) {
	if s.pool == nil {
		return &StoreStats{
			TotalReceipts:        0,
			TotalIdempotencyKeys: 0,
			OldestReceipt:        time.Now(),
			NewestReceipt:        time.Now(),
		}, fmt.Errorf("database connection not available")
	}

	stats := &StoreStats{}

	// Get receipt count
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ocx_receipts").Scan(&stats.TotalReceipts)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt count: %w", err)
	}

	// Get idempotency key count
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ocx_idempotency").Scan(&stats.TotalIdempotencyKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key count: %w", err)
	}

	// Get oldest and newest receipt timestamps
	err = s.pool.QueryRow(ctx, "SELECT MIN(created_at), MAX(created_at) FROM ocx_receipts").Scan(&stats.OldestReceipt, &stats.NewestReceipt)
	if err != nil {
		// If no receipts exist, use current time
		now := time.Now()
		stats.OldestReceipt = now
		stats.NewestReceipt = now
	}

	return stats, nil
}

// ReceiptSummary represents a summary of a receipt for listing
type ReceiptSummary struct {
	ID          string    `json:"id"`
	IssuerID    string    `json:"issuer_id"`
	ProgramHash [32]byte  `json:"program_hash"`
	GasUsed     uint64    `json:"gas_used"`
	StartedAt   uint64    `json:"started_at"`
	FinishedAt  uint64    `json:"finished_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// StoreStats represents database statistics
type StoreStats struct {
	TotalReceipts        int       `json:"total_receipts"`
	TotalIdempotencyKeys int       `json:"total_idempotency_keys"`
	OldestReceipt        time.Time `json:"oldest_receipt"`
	NewestReceipt        time.Time `json:"newest_receipt"`
}
