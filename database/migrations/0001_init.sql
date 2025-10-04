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

-- Create comments for documentation
COMMENT ON TABLE ocx_keys IS 'Public keys for receipt verification';
COMMENT ON TABLE ocx_receipts IS 'Signed execution receipts with canonical CBOR';
COMMENT ON TABLE ocx_idempotency IS 'Request idempotency tracking';
COMMENT ON TABLE ocx_audit_log IS 'Security and compliance audit trail';
COMMENT ON TABLE ocx_metrics IS 'Performance and operational metrics';

COMMENT ON COLUMN ocx_receipts.receipt_id IS 'Unique receipt identifier';
COMMENT ON COLUMN ocx_receipts.issuer_id IS 'Receipt issuer identifier';
COMMENT ON COLUMN ocx_receipts.program_hash IS 'SHA256 hash of executed program';
COMMENT ON COLUMN ocx_receipts.input_hash IS 'SHA256 hash of program input';
COMMENT ON COLUMN ocx_receipts.output_hash IS 'SHA256 hash of program output';
COMMENT ON COLUMN ocx_receipts.gas_used IS 'Deterministic gas consumption';
COMMENT ON COLUMN ocx_receipts.signature IS 'Ed25519 signature of receipt core';
COMMENT ON COLUMN ocx_receipts.receipt_cbor IS 'Full canonical CBOR receipt';
