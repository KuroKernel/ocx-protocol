-- OCX Protocol Receipt v1.1 Database Schema
-- This migration adds support for the new receipt format with enhanced features

-- Create the new receipts table with v1.1 fields
CREATE TABLE IF NOT EXISTS ocx_receipts_v1_1 (
    id VARCHAR(255) PRIMARY KEY,
    issuer_id VARCHAR(255) NOT NULL,
    key_version INTEGER NOT NULL,
    program_hash BYTEA NOT NULL,
    input_hash BYTEA NOT NULL,
    output_hash BYTEA NOT NULL,
    gas_used BIGINT NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    nonce BYTEA NOT NULL,
    float_mode VARCHAR(50) DEFAULT 'disabled',
    signature BYTEA NOT NULL,
    host_cycles BIGINT,
    host_info JSONB,
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_issuer_id ON ocx_receipts_v1_1 (issuer_id);
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_key_version ON ocx_receipts_v1_1 (key_version);
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_program_hash ON ocx_receipts_v1_1 (program_hash);
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_created_at ON ocx_receipts_v1_1 (created_at);
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_verified ON ocx_receipts_v1_1 (verified);
CREATE INDEX IF NOT EXISTS idx_ocx_receipts_v1_1_issuer_nonce ON ocx_receipts_v1_1 (issuer_id, nonce);

-- Create the replay protection table
CREATE TABLE IF NOT EXISTS ocx_replay_protection (
    issuer_id VARCHAR(255) NOT NULL,
    nonce BYTEA NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (issuer_id, nonce)
);

-- Create indexes for replay protection
CREATE INDEX IF NOT EXISTS idx_ocx_replay_protection_expires_at ON ocx_replay_protection (expires_at);
CREATE INDEX IF NOT EXISTS idx_ocx_replay_protection_issuer_id ON ocx_replay_protection (issuer_id);
CREATE INDEX IF NOT EXISTS idx_ocx_replay_protection_used_at ON ocx_replay_protection (used_at);

-- Create the audit log table for comprehensive logging
CREATE TABLE IF NOT EXISTS ocx_audit_log (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    event_type VARCHAR(100) NOT NULL,
    issuer_id VARCHAR(255),
    receipt_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    result VARCHAR(50) NOT NULL,
    details TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for audit log
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_timestamp ON ocx_audit_log (timestamp);
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_event_type ON ocx_audit_log (event_type);
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_issuer_id ON ocx_audit_log (issuer_id);
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_receipt_id ON ocx_audit_log (receipt_id);
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_action ON ocx_audit_log (action);
CREATE INDEX IF NOT EXISTS idx_ocx_audit_log_result ON ocx_audit_log (result);

-- Create the key management table
CREATE TABLE IF NOT EXISTS ocx_keys (
    id SERIAL PRIMARY KEY,
    key_id VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL,
    public_key BYTEA NOT NULL,
    algorithm VARCHAR(50) DEFAULT 'Ed25519',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(key_id, version)
);

-- Create indexes for key management
CREATE INDEX IF NOT EXISTS idx_ocx_keys_key_id ON ocx_keys (key_id);
CREATE INDEX IF NOT EXISTS idx_ocx_keys_version ON ocx_keys (version);
CREATE INDEX IF NOT EXISTS idx_ocx_keys_created_at ON ocx_keys (created_at);
CREATE INDEX IF NOT EXISTS idx_ocx_keys_expires_at ON ocx_keys (expires_at);

-- Create the system metrics table for monitoring
CREATE TABLE IF NOT EXISTS ocx_system_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    metric_unit VARCHAR(50),
    tags JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for system metrics
CREATE INDEX IF NOT EXISTS idx_ocx_system_metrics_timestamp ON ocx_system_metrics (timestamp);
CREATE INDEX IF NOT EXISTS idx_ocx_system_metrics_name ON ocx_system_metrics (metric_name);
CREATE INDEX IF NOT EXISTS idx_ocx_system_metrics_timestamp_name ON ocx_system_metrics (timestamp, metric_name);

-- Create views for common queries
CREATE OR REPLACE VIEW ocx_receipt_summary AS
SELECT 
    issuer_id,
    COUNT(*) as total_receipts,
    COUNT(*) FILTER (WHERE verified = true) as verified_receipts,
    COUNT(*) FILTER (WHERE verified = false) as unverified_receipts,
    AVG(EXTRACT(EPOCH FROM (finished_at - started_at)) * 1000) as avg_execution_time_ms,
    MAX(created_at) as last_activity,
    COUNT(DISTINCT key_version) as key_versions_used
FROM ocx_receipts_v1_1
GROUP BY issuer_id;

CREATE OR REPLACE VIEW ocx_daily_stats AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_receipts,
    COUNT(*) FILTER (WHERE verified = true) as verified_receipts,
    COUNT(*) FILTER (WHERE verified = false) as unverified_receipts,
    COUNT(DISTINCT issuer_id) as unique_issuers,
    AVG(EXTRACT(EPOCH FROM (finished_at - started_at)) * 1000) as avg_execution_time_ms
FROM ocx_receipts_v1_1
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Create functions for common operations
CREATE OR REPLACE FUNCTION cleanup_expired_nonces()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM ocx_replay_protection WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Keep audit logs for 90 days
    DELETE FROM ocx_audit_log WHERE created_at < NOW() - INTERVAL '90 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_old_metrics()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Keep metrics for 30 days
    DELETE FROM ocx_system_metrics WHERE created_at < NOW() - INTERVAL '30 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_ocx_receipts_v1_1_updated_at
    BEFORE UPDATE ON ocx_receipts_v1_1
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create a function to log audit events
CREATE OR REPLACE FUNCTION log_audit_event(
    p_event_type VARCHAR(100),
    p_issuer_id VARCHAR(255),
    p_receipt_id VARCHAR(255),
    p_action VARCHAR(100),
    p_result VARCHAR(50),
    p_details TEXT,
    p_ip_address INET,
    p_user_agent TEXT
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO ocx_audit_log (
        event_type, issuer_id, receipt_id, action, result, 
        details, ip_address, user_agent
    ) VALUES (
        p_event_type, p_issuer_id, p_receipt_id, p_action, p_result,
        p_details, p_ip_address, p_user_agent
    );
END;
$$ LANGUAGE plpgsql;

-- Create a function to check for replay attacks
CREATE OR REPLACE FUNCTION check_replay_attack(
    p_issuer_id VARCHAR(255),
    p_nonce BYTEA
)
RETURNS BOOLEAN AS $$
DECLARE
    nonce_exists BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM ocx_replay_protection 
        WHERE issuer_id = p_issuer_id 
        AND nonce = p_nonce 
        AND expires_at > NOW()
    ) INTO nonce_exists;
    
    RETURN nonce_exists;
END;
$$ LANGUAGE plpgsql;

-- Create a function to record nonce usage
CREATE OR REPLACE FUNCTION record_nonce_usage(
    p_issuer_id VARCHAR(255),
    p_nonce BYTEA,
    p_retention_hours INTEGER DEFAULT 168 -- 7 days
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO ocx_replay_protection (issuer_id, nonce, used_at, expires_at)
    VALUES (
        p_issuer_id, 
        p_nonce, 
        NOW(), 
        NOW() + (p_retention_hours || ' hours')::INTERVAL
    )
    ON CONFLICT (issuer_id, nonce) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Create a function to get system health
CREATE OR REPLACE FUNCTION get_system_health()
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'database_connected', true,
        'total_receipts', (SELECT COUNT(*) FROM ocx_receipts_v1_1),
        'verified_receipts', (SELECT COUNT(*) FROM ocx_receipts_v1_1 WHERE verified = true),
        'unverified_receipts', (SELECT COUNT(*) FROM ocx_receipts_v1_1 WHERE verified = false),
        'active_nonces', (SELECT COUNT(*) FROM ocx_replay_protection WHERE expires_at > NOW()),
        'replay_attacks', (SELECT COUNT(*) FROM ocx_audit_log WHERE event_type = 'replay_attack'),
        'unique_issuers', (SELECT COUNT(DISTINCT issuer_id) FROM ocx_receipts_v1_1),
        'last_activity', (SELECT MAX(created_at) FROM ocx_receipts_v1_1),
        'database_size_mb', (SELECT pg_size_pretty(pg_database_size(current_database()))),
        'check_time', NOW()
    ) INTO result;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ocx_receipts_v1_1 TO ocx_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ocx_replay_protection TO ocx_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ocx_audit_log TO ocx_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ocx_keys TO ocx_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ocx_system_metrics TO ocx_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ocx_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO ocx_user;

-- Create a comment explaining the schema
COMMENT ON TABLE ocx_receipts_v1_1 IS 'OCX Protocol Receipt v1.1 - Enhanced receipt format with replay protection and comprehensive metadata';
COMMENT ON TABLE ocx_replay_protection IS 'Replay protection for OCX receipts - stores used nonces to prevent replay attacks';
COMMENT ON TABLE ocx_audit_log IS 'Comprehensive audit logging for all OCX Protocol operations';
COMMENT ON TABLE ocx_keys IS 'Key management for OCX Protocol - stores public keys with versioning';
COMMENT ON TABLE ocx_system_metrics IS 'System metrics and monitoring data for OCX Protocol';
