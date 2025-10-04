-- OCX Protocol Database Initialization Script
-- This script sets up the production database schema

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create custom types
CREATE TYPE receipt_status AS ENUM ('pending', 'completed', 'failed', 'expired');
CREATE TYPE execution_status AS ENUM ('queued', 'running', 'completed', 'failed', 'timeout');

-- Create tables
CREATE TABLE IF NOT EXISTS ocx_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    receipt_hash VARCHAR(64) UNIQUE NOT NULL,
    receipt_data BYTEA NOT NULL,
    artifact_hash VARCHAR(64) NOT NULL,
    input_hash VARCHAR(64) NOT NULL,
    output_hash VARCHAR(64) NOT NULL,
    gas_used BIGINT NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL,
    issuer_id VARCHAR(64) NOT NULL,
    signature BYTEA NOT NULL,
    status receipt_status DEFAULT 'completed',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ocx_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    artifact_hash VARCHAR(64) NOT NULL,
    input_data BYTEA NOT NULL,
    output_data BYTEA,
    gas_used BIGINT,
    status execution_status DEFAULT 'queued',
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ocx_idempotency (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    request_hash VARCHAR(64) NOT NULL,
    response_data BYTEA,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS ocx_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    permissions TEXT[] DEFAULT '{}',
    rate_limit_rps INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_receipts_hash ON ocx_receipts(receipt_hash);
CREATE INDEX IF NOT EXISTS idx_receipts_artifact ON ocx_receipts(artifact_hash);
CREATE INDEX IF NOT EXISTS idx_receipts_created_at ON ocx_receipts(created_at);
CREATE INDEX IF NOT EXISTS idx_receipts_status ON ocx_receipts(status);

CREATE INDEX IF NOT EXISTS idx_executions_artifact ON ocx_executions(artifact_hash);
CREATE INDEX IF NOT EXISTS idx_executions_status ON ocx_executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON ocx_executions(created_at);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON ocx_idempotency(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON ocx_api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON ocx_api_keys(is_active);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_receipts_updated_at BEFORE UPDATE ON ocx_receipts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_executions_updated_at BEFORE UPDATE ON ocx_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to clean up expired idempotency keys
CREATE OR REPLACE FUNCTION cleanup_expired_idempotency()
RETURNS void AS $$
BEGIN
    DELETE FROM ocx_idempotency WHERE expires_at < NOW();
END;
$$ language 'plpgsql';

-- Create function to clean up old executions
CREATE OR REPLACE FUNCTION cleanup_old_executions()
RETURNS void AS $$
BEGIN
    DELETE FROM ocx_executions 
    WHERE created_at < NOW() - INTERVAL '30 days' 
    AND status IN ('completed', 'failed', 'timeout');
END;
$$ language 'plpgsql';

-- Insert default API key (admin:supersecretkey)
-- Hash is SHA256 of "admin:supersecretkey"
INSERT INTO ocx_api_keys (key_name, key_hash, permissions, rate_limit_rps, expires_at)
VALUES (
    'admin',
    'a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456',
    ARRAY['execute', 'verify', 'read', 'admin'],
    1000,
    NOW() + INTERVAL '1 year'
) ON CONFLICT (key_hash) DO NOTHING;

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ocx;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ocx;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO ocx;
