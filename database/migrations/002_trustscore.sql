-- ============================================================================
-- OCX TrustScore Database Schema
-- Migration: 002_trustscore.sql
-- Description: Add reputation verification and platform connection tables
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Reputation Verifications Table
-- ============================================================================
-- Stores reputation verification results with cryptographic receipts
CREATE TABLE IF NOT EXISTS reputation_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    trust_score DECIMAL(5,2) NOT NULL CHECK (trust_score >= 0 AND trust_score <= 100),
    confidence DECIMAL(4,3) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    components JSONB NOT NULL DEFAULT '{}',
    receipt_id VARCHAR(255),
    algorithm_version VARCHAR(50) NOT NULL DEFAULT 'trustscore-v1.0.0',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_refreshed_at TIMESTAMP WITH TIME ZONE,

    -- Indexes for performance
    CONSTRAINT fk_receipt FOREIGN KEY (receipt_id)
        REFERENCES receipts(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reputation_user_id
    ON reputation_verifications(user_id);

CREATE INDEX IF NOT EXISTS idx_reputation_created_at
    ON reputation_verifications(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reputation_expires_at
    ON reputation_verifications(expires_at);

CREATE INDEX IF NOT EXISTS idx_reputation_trust_score
    ON reputation_verifications(trust_score DESC);

-- Partial index for active (non-expired) verifications
CREATE INDEX IF NOT EXISTS idx_reputation_active
    ON reputation_verifications(user_id, created_at DESC)
    WHERE expires_at > NOW();

-- ============================================================================
-- Platform Connections Table
-- ============================================================================
-- Tracks user connections to various platforms (GitHub, LinkedIn, etc.)
CREATE TABLE IF NOT EXISTS platform_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    platform_type VARCHAR(50) NOT NULL,
    platform_username VARCHAR(255) NOT NULL,
    platform_user_id VARCHAR(255),
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    verification_method VARCHAR(50),
    last_checked TIMESTAMP WITH TIME ZONE,
    last_score DECIMAL(5,2),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint: one platform connection per user per platform
    CONSTRAINT unique_user_platform UNIQUE(user_id, platform_type)
);

CREATE INDEX IF NOT EXISTS idx_platform_user_id
    ON platform_connections(user_id);

CREATE INDEX IF NOT EXISTS idx_platform_type
    ON platform_connections(platform_type);

CREATE INDEX IF NOT EXISTS idx_platform_verified
    ON platform_connections(verified)
    WHERE verified = TRUE;

CREATE INDEX IF NOT EXISTS idx_platform_last_checked
    ON platform_connections(last_checked);

-- ============================================================================
-- Reputation History Table
-- ============================================================================
-- Stores historical reputation snapshots for trend analysis
CREATE TABLE IF NOT EXISTS reputation_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    trust_score DECIMAL(5,2) NOT NULL,
    confidence DECIMAL(4,3) NOT NULL,
    snapshot_date DATE NOT NULL,
    platform_count INTEGER NOT NULL DEFAULT 0,
    algorithm_version VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint: one snapshot per user per day
    CONSTRAINT unique_user_snapshot UNIQUE(user_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_history_user_id
    ON reputation_history(user_id);

CREATE INDEX IF NOT EXISTS idx_history_snapshot_date
    ON reputation_history(snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_history_user_date
    ON reputation_history(user_id, snapshot_date DESC);

-- ============================================================================
-- Platform API Usage Table
-- ============================================================================
-- Tracks API usage for rate limiting and cost management
CREATE TABLE IF NOT EXISTS platform_api_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform_type VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    total_requests INTEGER NOT NULL DEFAULT 0,
    successful_requests INTEGER NOT NULL DEFAULT 0,
    failed_requests INTEGER NOT NULL DEFAULT 0,
    rate_limited_requests INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint: one record per platform per day
    CONSTRAINT unique_platform_date UNIQUE(platform_type, date)
);

CREATE INDEX IF NOT EXISTS idx_api_usage_platform
    ON platform_api_usage(platform_type, date DESC);

-- ============================================================================
-- Functions and Triggers
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for platform_connections
DROP TRIGGER IF EXISTS update_platform_connections_updated_at ON platform_connections;
CREATE TRIGGER update_platform_connections_updated_at
    BEFORE UPDATE ON platform_connections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to create daily reputation snapshots
CREATE OR REPLACE FUNCTION create_reputation_snapshot()
RETURNS void AS $$
BEGIN
    -- Insert daily snapshot for users with recent verifications
    INSERT INTO reputation_history (user_id, trust_score, confidence, snapshot_date, platform_count, algorithm_version)
    SELECT DISTINCT ON (rv.user_id)
        rv.user_id,
        rv.trust_score,
        rv.confidence,
        CURRENT_DATE,
        COALESCE((
            SELECT COUNT(*)
            FROM platform_connections pc
            WHERE pc.user_id = rv.user_id AND pc.verified = TRUE
        ), 0),
        rv.algorithm_version
    FROM reputation_verifications rv
    WHERE rv.created_at::date = CURRENT_DATE - INTERVAL '1 day'
    ORDER BY rv.user_id, rv.created_at DESC
    ON CONFLICT (user_id, snapshot_date)
    DO UPDATE SET
        trust_score = EXCLUDED.trust_score,
        confidence = EXCLUDED.confidence,
        platform_count = EXCLUDED.platform_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Views for Common Queries
-- ============================================================================

-- View: Latest reputation for each user
CREATE OR REPLACE VIEW latest_reputation AS
SELECT DISTINCT ON (user_id)
    user_id,
    trust_score,
    confidence,
    components,
    algorithm_version,
    created_at,
    expires_at
FROM reputation_verifications
WHERE expires_at > NOW()
ORDER BY user_id, created_at DESC;

-- View: Active platform connections count per user
CREATE OR REPLACE VIEW user_platform_stats AS
SELECT
    user_id,
    COUNT(*) as total_connections,
    COUNT(*) FILTER (WHERE verified = TRUE) as verified_connections,
    ARRAY_AGG(platform_type ORDER BY platform_type) FILTER (WHERE verified = TRUE) as verified_platforms
FROM platform_connections
GROUP BY user_id;

-- View: Reputation trends (30-day moving average)
CREATE OR REPLACE VIEW reputation_trends AS
SELECT
    user_id,
    snapshot_date,
    trust_score,
    AVG(trust_score) OVER (
        PARTITION BY user_id
        ORDER BY snapshot_date
        ROWS BETWEEN 29 PRECEDING AND CURRENT ROW
    ) as ma30_score,
    trust_score - LAG(trust_score, 1) OVER (
        PARTITION BY user_id
        ORDER BY snapshot_date
    ) as day_change
FROM reputation_history
ORDER BY user_id, snapshot_date DESC;

-- ============================================================================
-- Sample Data for Testing (commented out for production)
-- ============================================================================
/*
-- Insert test platform connections
INSERT INTO platform_connections (user_id, platform_type, platform_username, verified, verified_at)
VALUES
    ('alice@example.com', 'github', 'alice', TRUE, NOW()),
    ('alice@example.com', 'linkedin', 'alice-linkedin', TRUE, NOW()),
    ('bob@example.com', 'github', 'bob', TRUE, NOW());

-- Insert test reputation verification
INSERT INTO reputation_verifications (user_id, trust_score, confidence, components, created_at, expires_at)
VALUES
    ('alice@example.com', 87.5, 0.92, '{"github": {"score": 90, "weight": 0.5}, "linkedin": {"score": 85, "weight": 0.5}}'::jsonb, NOW(), NOW() + INTERVAL '30 days');
*/

-- ============================================================================
-- Grants (adjust based on your user roles)
-- ============================================================================
-- GRANT SELECT, INSERT, UPDATE, DELETE ON reputation_verifications TO ocx_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON platform_connections TO ocx_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON reputation_history TO ocx_app;
-- GRANT SELECT, INSERT, UPDATE ON platform_api_usage TO ocx_app;
-- GRANT SELECT ON latest_reputation TO ocx_app;
-- GRANT SELECT ON user_platform_stats TO ocx_app;
-- GRANT SELECT ON reputation_trends TO ocx_app;

-- ============================================================================
-- Migration Complete
-- ============================================================================
SELECT 'TrustScore schema migration complete' AS status;
