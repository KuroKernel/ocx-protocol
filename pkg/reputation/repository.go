package reputation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ocx.local/pkg/reputation/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound is returned when a record is not found
	ErrNotFound = errors.New("record not found")

	// ErrExpired is returned when a verification has expired
	ErrExpired = errors.New("verification expired")

	// ErrDuplicate is returned when attempting to create a duplicate record
	ErrDuplicate = errors.New("duplicate record")
)

// Repository handles database operations for reputation data
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new reputation repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SaveVerification stores a new reputation verification
func (r *Repository) SaveVerification(ctx context.Context, v types.Verification) (string, error) {
	componentsJSON, err := json.Marshal(v.Components)
	if err != nil {
		return "", fmt.Errorf("failed to marshal components: %w", err)
	}

	query := `
		INSERT INTO reputation_verifications
		(user_id, trust_score, confidence, components, receipt_id, algorithm_version, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id string
	err = r.db.QueryRow(ctx, query,
		v.UserID,
		v.TrustScore,
		v.Confidence,
		componentsJSON,
		v.ReceiptID,
		v.AlgorithmVersion,
		v.CreatedAt,
		v.ExpiresAt,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to save verification: %w", err)
	}

	return id, nil
}

// GetVerification retrieves a verification by ID
func (r *Repository) GetVerification(ctx context.Context, id string) (*types.Verification, error) {
	query := `
		SELECT id, user_id, trust_score, confidence, components, receipt_id,
		       algorithm_version, created_at, expires_at, last_refreshed_at
		FROM reputation_verifications
		WHERE id = $1
	`

	var v types.Verification
	var componentsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID,
		&v.UserID,
		&v.TrustScore,
		&v.Confidence,
		&componentsJSON,
		&v.ReceiptID,
		&v.AlgorithmVersion,
		&v.CreatedAt,
		&v.ExpiresAt,
		&v.LastRefreshedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	if err := json.Unmarshal(componentsJSON, &v.Components); err != nil {
		return nil, fmt.Errorf("failed to unmarshal components: %w", err)
	}

	// Check if expired
	if time.Now().After(v.ExpiresAt) {
		return &v, ErrExpired
	}

	return &v, nil
}

// GetLatestVerification retrieves the latest non-expired verification for a user
func (r *Repository) GetLatestVerification(ctx context.Context, userID string) (*types.Verification, error) {
	query := `
		SELECT id, user_id, trust_score, confidence, components, receipt_id,
		       algorithm_version, created_at, expires_at, last_refreshed_at
		FROM reputation_verifications
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var v types.Verification
	var componentsJSON []byte

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&v.ID,
		&v.UserID,
		&v.TrustScore,
		&v.Confidence,
		&componentsJSON,
		&v.ReceiptID,
		&v.AlgorithmVersion,
		&v.CreatedAt,
		&v.ExpiresAt,
		&v.LastRefreshedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest verification: %w", err)
	}

	if err := json.Unmarshal(componentsJSON, &v.Components); err != nil {
		return nil, fmt.Errorf("failed to unmarshal components: %w", err)
	}

	return &v, nil
}

// GetVerificationHistory retrieves verification history for a user
func (r *Repository) GetVerificationHistory(ctx context.Context, userID string, limit int) ([]types.Verification, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, user_id, trust_score, confidence, components, receipt_id,
		       algorithm_version, created_at, expires_at, last_refreshed_at
		FROM reputation_verifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var results []types.Verification
	for rows.Next() {
		var v types.Verification
		var componentsJSON []byte

		err := rows.Scan(
			&v.ID,
			&v.UserID,
			&v.TrustScore,
			&v.Confidence,
			&componentsJSON,
			&v.ReceiptID,
			&v.AlgorithmVersion,
			&v.CreatedAt,
			&v.ExpiresAt,
			&v.LastRefreshedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(componentsJSON, &v.Components); err != nil {
			continue
		}

		results = append(results, v)
	}

	return results, nil
}

// SavePlatformConnection stores a platform connection
func (r *Repository) SavePlatformConnection(ctx context.Context, pc types.PlatformConnection) (string, error) {
	metadataJSON, _ := json.Marshal(pc.Metadata)

	query := `
		INSERT INTO platform_connections
		(user_id, platform_type, platform_username, platform_user_id, verified, verified_at,
		 verification_method, last_checked, last_score, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, platform_type)
		DO UPDATE SET
			platform_username = EXCLUDED.platform_username,
			platform_user_id = EXCLUDED.platform_user_id,
			verified = EXCLUDED.verified,
			verified_at = EXCLUDED.verified_at,
			verification_method = EXCLUDED.verification_method,
			last_checked = EXCLUDED.last_checked,
			last_score = EXCLUDED.last_score,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(ctx, query,
		pc.UserID,
		pc.PlatformType,
		pc.PlatformUsername,
		pc.PlatformUserID,
		pc.Verified,
		pc.VerifiedAt,
		pc.VerificationMethod,
		pc.LastChecked,
		pc.LastScore,
		metadataJSON,
		pc.CreatedAt,
		pc.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to save platform connection: %w", err)
	}

	return id, nil
}

// GetPlatformConnections retrieves all platform connections for a user
func (r *Repository) GetPlatformConnections(ctx context.Context, userID string) ([]types.PlatformConnection, error) {
	query := `
		SELECT id, user_id, platform_type, platform_username, platform_user_id, verified,
		       verified_at, verification_method, last_checked, last_score, metadata,
		       created_at, updated_at
		FROM platform_connections
		WHERE user_id = $1
		ORDER BY platform_type
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform connections: %w", err)
	}
	defer rows.Close()

	var results []types.PlatformConnection
	for rows.Next() {
		var pc types.PlatformConnection
		var metadataJSON []byte

		err := rows.Scan(
			&pc.ID,
			&pc.UserID,
			&pc.PlatformType,
			&pc.PlatformUsername,
			&pc.PlatformUserID,
			&pc.Verified,
			&pc.VerifiedAt,
			&pc.VerificationMethod,
			&pc.LastChecked,
			&pc.LastScore,
			&metadataJSON,
			&pc.CreatedAt,
			&pc.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &pc.Metadata)
		}

		results = append(results, pc)
	}

	return results, nil
}

// GetStats retrieves repository statistics
func (r *Repository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total verifications
	var totalVerifications int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM reputation_verifications").Scan(&totalVerifications)
	if err == nil {
		stats["total_verifications"] = totalVerifications
	}

	// Active verifications (non-expired)
	var activeVerifications int64
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM reputation_verifications WHERE expires_at > NOW()").Scan(&activeVerifications)
	if err == nil {
		stats["active_verifications"] = activeVerifications
	}

	// Unique users
	var uniqueUsers int64
	err = r.db.QueryRow(ctx, "SELECT COUNT(DISTINCT user_id) FROM reputation_verifications").Scan(&uniqueUsers)
	if err == nil {
		stats["unique_users"] = uniqueUsers
	}

	// Platform connections
	var platformConnections int64
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM platform_connections WHERE verified = TRUE").Scan(&platformConnections)
	if err == nil {
		stats["verified_connections"] = platformConnections
	}

	// Average trust score
	var avgTrustScore float64
	err = r.db.QueryRow(ctx, `
		SELECT AVG(trust_score)
		FROM reputation_verifications
		WHERE expires_at > NOW()
	`).Scan(&avgTrustScore)
	if err == nil {
		stats["avg_trust_score"] = avgTrustScore
	}

	return stats, nil
}
