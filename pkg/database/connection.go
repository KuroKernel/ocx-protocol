package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
)

// Note: This file requires Go 1.21+ due to pgx/v5 dependencies
// For Go 1.18 compatibility, we'll provide fallback implementations

// Config holds database configuration
type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// LoadConfig loads database configuration from environment variables
func LoadConfig() *Config {
	// Check both DATABASE_URL and OCX_DB_DSN for compatibility
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = os.Getenv("OCX_DB_DSN")
	}

	config := &Config{
		URL:             url,
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: time.Minute * 30,
	}

	// Set default DATABASE_URL if not provided
	if config.URL == "" {
		// Default to PostgreSQL, but server will fall back to in-memory store if connection fails
		config.URL = "postgres://ocx:ocx@localhost:5432/ocx?sslmode=disable"
	}

	return config
}

// Connect creates a new database connection pool
func Connect(ctx context.Context, config *Config) (*pgxpool.Pool, error) {
	if config == nil {
		config = LoadConfig()
	}

	// Check if it's a SQLite URL
	if strings.HasPrefix(config.URL, "file:") {
		// For SQLite, we'll return a mock pool that uses sql.DB internally
		// This is a temporary solution until we refactor to use a unified interface
		return nil, fmt.Errorf("SQLite not supported with pgx pool, use in-memory store")
	}

	// Create connection pool with pgx/v5 for PostgreSQL
	pool, err := pgxpool.New(ctx, config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// MockConnection represents a mock database connection for Go 1.18
type MockConnection struct {
	URL string
}

// DBHealth performs a health check on the database connection
func DBHealth(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// Set a short timeout for health check
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Execute a simple query to verify connectivity
	var result int
	err := pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

// Migrate runs database migrations
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// Execute embedded SQL migrations
	migrationSQL := `
		-- OCX Protocol Database Schema
		-- Migration: 0001_init.sql
		-- Description: Initial schema for receipts, keys, and idempotency

		-- Create keys table for public key storage
		CREATE TABLE IF NOT EXISTS ocx_keys (
			key_id TEXT PRIMARY KEY,
			public_key BYTEA NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create receipts table for receipt storage
		CREATE TABLE IF NOT EXISTS ocx_receipts (
			receipt_id UUID PRIMARY KEY,
			issuer_id TEXT NOT NULL,
			program_hash BYTEA NOT NULL CHECK (OCTET_LENGTH(program_hash) = 32),
			input_hash BYTEA NOT NULL CHECK (OCTET_LENGTH(input_hash) = 32),
			output_hash BYTEA NOT NULL CHECK (OCTET_LENGTH(output_hash) = 32),
			gas_used BIGINT NOT NULL,
			started_at BIGINT NOT NULL,
			finished_at BIGINT NOT NULL,
			signature BYTEA NOT NULL CHECK (OCTET_LENGTH(signature) = 64),
			host_cycles BIGINT NOT NULL,
			host_info JSONB NOT NULL DEFAULT '{}',
			receipt_cbor BYTEA NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create unique index for receipt core fields to prevent duplicates
		CREATE UNIQUE INDEX IF NOT EXISTS idx_receipts_core
			ON ocx_receipts (issuer_id, program_hash, input_hash, output_hash, gas_used, started_at, finished_at);

		-- Create index for receipt lookup by issuer
		CREATE INDEX IF NOT EXISTS idx_receipts_issuer
			ON ocx_receipts (issuer_id);

		-- Create index for receipt lookup by program hash
		CREATE INDEX IF NOT EXISTS idx_receipts_program
			ON ocx_receipts (program_hash);

		-- Create index for receipt lookup by creation time
		CREATE INDEX IF NOT EXISTS idx_receipts_created
			ON ocx_receipts (created_at);

		-- Create idempotency table for request deduplication
		CREATE TABLE IF NOT EXISTS ocx_idempotency (
			idem_key TEXT PRIMARY KEY,
			request_sha256 BYTEA NOT NULL CHECK (OCTET_LENGTH(request_sha256) = 32),
			response_cbor BYTEA NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create index for idempotency cleanup (old entries)
		CREATE INDEX IF NOT EXISTS idx_idempotency_created
			ON ocx_idempotency (created_at);

		-- Create audit log table for security and compliance
		CREATE TABLE IF NOT EXISTS ocx_audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_type TEXT NOT NULL,
			api_key TEXT,
			ip_address INET,
			user_agent TEXT,
			request_path TEXT,
			request_method TEXT,
			response_status INTEGER,
			error_code TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create index for audit log queries
		CREATE INDEX IF NOT EXISTS idx_audit_log_created
			ON ocx_audit_log (created_at);

		CREATE INDEX IF NOT EXISTS idx_audit_log_api_key
			ON ocx_audit_log (api_key);

		CREATE INDEX IF NOT EXISTS idx_audit_log_event_type
			ON ocx_audit_log (event_type);

		-- Create performance metrics table
		CREATE TABLE IF NOT EXISTS ocx_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			metric_name TEXT NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL,
			labels JSONB NOT NULL DEFAULT '{}',
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		-- Create index for metrics queries
		CREATE INDEX IF NOT EXISTS idx_metrics_name_timestamp
			ON ocx_metrics (metric_name, timestamp);

		CREATE INDEX IF NOT EXISTS idx_metrics_timestamp
			ON ocx_metrics (timestamp);

		-- Create cleanup function for old idempotency entries
		CREATE OR REPLACE FUNCTION cleanup_old_idempotency()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			-- Delete idempotency entries older than 24 hours
			DELETE FROM ocx_idempotency 
			WHERE created_at < NOW() - INTERVAL '24 hours';
			
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql;

		-- Create cleanup function for old audit logs
		CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			-- Delete audit logs older than 30 days
			DELETE FROM ocx_audit_log 
			WHERE created_at < NOW() - INTERVAL '30 days';
			
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql;

		-- Create cleanup function for old metrics
		CREATE OR REPLACE FUNCTION cleanup_old_metrics()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			-- Delete metrics older than 7 days
			DELETE FROM ocx_metrics 
			WHERE timestamp < NOW() - INTERVAL '7 days';
			
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql;

		-- Insert initial system configuration
		INSERT INTO ocx_keys (key_id, public_key) 
		VALUES ('system-init', '\x0000000000000000000000000000000000000000000000000000000000000000')
		ON CONFLICT (key_id) DO NOTHING;
	`

	// Execute the migration SQL
	_, err := pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}
