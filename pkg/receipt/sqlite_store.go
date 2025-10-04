package receipt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements ReceiptStore using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed receipt store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to configure SQLite: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Create tables
	if err := store.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

// createTables creates the necessary tables
func (s *SQLiteStore) createTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS ocx_receipts (
		receipt_id TEXT PRIMARY KEY,
		issuer_id TEXT NOT NULL,
		program_hash BLOB NOT NULL CHECK (length(program_hash) = 32),
		input_hash BLOB NOT NULL CHECK (length(input_hash) = 32),
		output_hash BLOB NOT NULL CHECK (length(output_hash) = 32),
		gas_used INTEGER NOT NULL,
		started_at INTEGER NOT NULL,
		finished_at INTEGER NOT NULL,
		signature BLOB NOT NULL CHECK (length(signature) = 64),
		host_cycles INTEGER NOT NULL,
		host_info TEXT NOT NULL DEFAULT '{}',
		receipt_cbor BLOB NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_ocx_receipts_issuer_id ON ocx_receipts(issuer_id);
	CREATE INDEX IF NOT EXISTS idx_ocx_receipts_created_at ON ocx_receipts(created_at);
	CREATE INDEX IF NOT EXISTS idx_ocx_receipts_program_hash ON ocx_receipts(program_hash);
	`

	_, err := s.db.Exec(createTableSQL)
	return err
}

// SaveReceipt saves a receipt to the database
func (s *SQLiteStore) SaveReceipt(ctx context.Context, r ReceiptFull, fullCBOR []byte) (string, error) {
	// Generate receipt ID
	receiptID := s.generateReceiptID()

	const query = `
		INSERT INTO ocx_receipts (
			receipt_id, issuer_id, 
			program_hash, input_hash, output_hash, gas_used,
			started_at, finished_at, signature, receipt_cbor, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		receiptID,
		"ocx-server-v1", // Default issuer ID
		r.Core.ProgramHash[:],
		r.Core.InputHash[:],
		r.Core.OutputHash[:],
		r.Core.GasUsed,
		r.Core.StartedAt,
		r.Core.FinishedAt,
		r.Signature,
		fullCBOR,
		time.Now(),
	)

	if err != nil {
		return "", fmt.Errorf("failed to save receipt: %w", err)
	}

	return receiptID, nil
}

// GetReceipt retrieves a receipt by ID
func (s *SQLiteStore) GetReceipt(ctx context.Context, receiptID string) ([]byte, error) {
	const query = `SELECT receipt_cbor FROM ocx_receipts WHERE receipt_id = ?`

	var receiptData []byte
	err := s.db.QueryRowContext(ctx, query, receiptID).Scan(&receiptData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("receipt not found: %s", receiptID)
		}
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return receiptData, nil
}

// GetReceiptByCore retrieves a receipt by its core components
func (s *SQLiteStore) GetReceiptByCore(ctx context.Context, core *ReceiptCore) (*ReceiptFull, error) {
	const query = `SELECT receipt_id, issuer_id, program_hash, input_hash, output_hash, gas_used, started_at, finished_at, signature, created_at FROM ocx_receipts WHERE program_hash = ? AND input_hash = ? AND output_hash = ? LIMIT 1`

	var receiptID, issuerID string
	var programHash, inputHash, outputHash []byte
	var gasUsed uint64
	var startedAt, finishedAt uint64
	var createdAt time.Time
	var signature []byte

	err := s.db.QueryRowContext(ctx, query, core.ProgramHash[:], core.InputHash[:], core.OutputHash[:]).Scan(
		&receiptID, &issuerID,
		&programHash, &inputHash, &outputHash, &gasUsed,
		&startedAt, &finishedAt, &signature, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("receipt not found")
		}
		return nil, fmt.Errorf("failed to get receipt by core: %w", err)
	}

	// Reconstruct the receipt
	receipt := &ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: [32]byte(programHash),
			InputHash:   [32]byte(inputHash),
			OutputHash:  [32]byte(outputHash),
			GasUsed:     gasUsed,
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
		},
		Signature: signature,
	}

	return receipt, nil
}

// ListReceipts lists receipts with pagination
func (s *SQLiteStore) ListReceipts(ctx context.Context, limit, offset int) ([]ReceiptSummary, error) {
	const query = `SELECT receipt_id, issuer_id, program_hash, gas_used, started_at, finished_at, created_at FROM ocx_receipts ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list receipts: %w", err)
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate receipts: %w", err)
	}

	return receipts, nil
}

// DeleteReceipt deletes a receipt by ID
func (s *SQLiteStore) DeleteReceipt(ctx context.Context, receiptID string) error {
	const query = `DELETE FROM ocx_receipts WHERE receipt_id = ?`

	result, err := s.db.ExecContext(ctx, query, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("receipt not found: %s", receiptID)
	}

	return nil
}

// generateReceiptID generates a receipt ID as UUID
func (s *SQLiteStore) generateReceiptID() string {
	bytes := make([]byte, 16) // 16 bytes for UUID
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return fmt.Sprintf("receipt-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits according to RFC 4122
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant bits

	// Format as UUID string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

// PutIdempotent stores idempotent request/response pairs
func (s *SQLiteStore) PutIdempotent(ctx context.Context, key string, reqHash [32]byte, respCBOR []byte) (bool, bool, []byte, error) {
	// For now, just return false (not implemented)
	return false, false, nil, nil
}

// GetIdempotent retrieves idempotent request/response pairs
func (s *SQLiteStore) GetIdempotent(ctx context.Context, key string) ([32]byte, []byte, bool, error) {
	// For now, just return false (not implemented)
	return [32]byte{}, nil, false, nil
}

// CleanupOldIdempotency cleans up old idempotency records
func (s *SQLiteStore) CleanupOldIdempotency(ctx context.Context, olderThan time.Duration) (int64, error) {
	// For now, just return 0 (not implemented)
	return 0, nil
}

// GetStats returns database statistics
func (s *SQLiteStore) GetStats(ctx context.Context) (*StoreStats, error) {
	const query = `SELECT COUNT(*) FROM ocx_receipts`

	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &StoreStats{
		TotalReceipts:        count,
		TotalIdempotencyKeys: 0,
		OldestReceipt:        time.Now(),
		NewestReceipt:        time.Now(),
	}, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
