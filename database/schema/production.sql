-- OCX Protocol Production Database Schema
-- PostgreSQL schema for immutable receipt storage

-- =============================================================================
-- RECEIPTS TABLE (IMMUTABLE)
-- =============================================================================

CREATE TABLE receipts (
    receipt_hash BYTEA PRIMARY KEY,
    receipt_body BYTEA NOT NULL,
    artifact_hash BYTEA NOT NULL,
    input_hash BYTEA NOT NULL,
    output_hash BYTEA NOT NULL,
    cycles_used BIGINT NOT NULL,
    price_micro_units BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Immutability constraint
    CONSTRAINT body_hash_match 
    CHECK (digest(receipt_body, 'sha256') = receipt_hash),
    
    -- Performance indexes
    INDEX idx_receipts_artifact (artifact_hash),
    INDEX idx_receipts_input (input_hash),
    INDEX idx_receipts_output (output_hash),
    INDEX idx_receipts_created (created_at),
    INDEX idx_receipts_cycles (cycles_used),
    INDEX idx_receipts_price (price_micro_units)
);

-- Prevent updates/deletes (append-only)
CREATE RULE no_receipt_updates AS ON UPDATE TO receipts DO NOTHING;
CREATE RULE no_receipt_deletes AS ON DELETE TO receipts DO NOTHING;

-- =============================================================================
-- ARTIFACTS TABLE (CODE REGISTRY)
-- =============================================================================

CREATE TABLE artifacts (
    artifact_hash BYTEA PRIMARY KEY,
    artifact_code BYTEA NOT NULL,
    artifact_size BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    usage_count BIGINT NOT NULL DEFAULT 0,
    
    -- Performance indexes
    INDEX idx_artifacts_size (artifact_size),
    INDEX idx_artifacts_created (created_at),
    INDEX idx_artifacts_last_used (last_used_at),
    INDEX idx_artifacts_usage (usage_count)
);

-- =============================================================================
-- EXECUTIONS TABLE (EXECUTION LOG)
-- =============================================================================

CREATE TABLE executions (
    execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_hash BYTEA NOT NULL REFERENCES receipts(receipt_hash),
    artifact_hash BYTEA NOT NULL,
    input_hash BYTEA NOT NULL,
    max_cycles BIGINT NOT NULL,
    actual_cycles BIGINT NOT NULL,
    execution_time_ms BIGINT NOT NULL,
    memory_pages_used BIGINT NOT NULL,
    io_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Performance indexes
    INDEX idx_executions_receipt (receipt_hash),
    INDEX idx_executions_artifact (artifact_hash),
    INDEX idx_executions_created (created_at),
    INDEX idx_executions_cycles (actual_cycles),
    INDEX idx_executions_time (execution_time_ms)
);

-- =============================================================================
-- SETTLEMENTS TABLE (PAYMENT LOG)
-- =============================================================================

CREATE TABLE settlements (
    settlement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_hash BYTEA NOT NULL REFERENCES receipts(receipt_hash),
    payer_id VARCHAR(64) NOT NULL,
    payee_id VARCHAR(64) NOT NULL,
    amount_micro_units BIGINT NOT NULL,
    currency VARCHAR(16) NOT NULL DEFAULT 'USD_MICRO',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    transaction_hash VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    
    -- Performance indexes
    INDEX idx_settlements_receipt (receipt_hash),
    INDEX idx_settlements_payer (payer_id),
    INDEX idx_settlements_payee (payee_id),
    INDEX idx_settlements_status (status),
    INDEX idx_settlements_created (created_at),
    INDEX idx_settlements_amount (amount_micro_units)
);

-- =============================================================================
-- CONFORMANCE TESTS TABLE (TEST RESULTS)
-- =============================================================================

CREATE TABLE conformance_tests (
    test_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_name VARCHAR(128) NOT NULL,
    test_category VARCHAR(32) NOT NULL,
    test_vector JSONB NOT NULL,
    expected_result JSONB NOT NULL,
    actual_result JSONB,
    passed BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    execution_time_ms BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Performance indexes
    INDEX idx_conformance_name (test_name),
    INDEX idx_conformance_category (test_category),
    INDEX idx_conformance_passed (passed),
    INDEX idx_conformance_created (created_at)
);

-- =============================================================================
-- BENCHMARKS TABLE (PERFORMANCE DATA)
-- =============================================================================

CREATE TABLE benchmarks (
    benchmark_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    benchmark_name VARCHAR(128) NOT NULL,
    benchmark_type VARCHAR(32) NOT NULL,
    ops_per_sec DOUBLE PRECISION NOT NULL,
    ns_per_op DOUBLE PRECISION NOT NULL,
    cycles_avg BIGINT NOT NULL,
    memory_avg BIGINT NOT NULL,
    test_environment JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Performance indexes
    INDEX idx_benchmarks_name (benchmark_name),
    INDEX idx_benchmarks_type (benchmark_type),
    INDEX idx_benchmarks_ops (ops_per_sec),
    INDEX idx_benchmarks_created (created_at)
);

-- =============================================================================
-- VIEWS FOR ANALYTICS
-- =============================================================================

-- Receipt summary view
CREATE VIEW receipt_summary AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_receipts,
    SUM(cycles_used) as total_cycles,
    SUM(price_micro_units) as total_revenue,
    AVG(cycles_used) as avg_cycles,
    AVG(price_micro_units) as avg_price
FROM receipts
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Artifact usage view
CREATE VIEW artifact_usage AS
SELECT 
    a.artifact_hash,
    a.artifact_size,
    a.usage_count,
    a.last_used_at,
    COUNT(r.receipt_hash) as receipt_count,
    SUM(r.cycles_used) as total_cycles,
    SUM(r.price_micro_units) as total_revenue
FROM artifacts a
LEFT JOIN receipts r ON a.artifact_hash = r.artifact_hash
GROUP BY a.artifact_hash, a.artifact_size, a.usage_count, a.last_used_at
ORDER BY total_revenue DESC;

-- Performance trends view
CREATE VIEW performance_trends AS
SELECT 
    DATE(created_at) as date,
    AVG(execution_time_ms) as avg_execution_time,
    AVG(actual_cycles) as avg_cycles,
    AVG(memory_pages_used) as avg_memory_pages,
    COUNT(*) as execution_count
FROM executions
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- =============================================================================
-- FUNCTIONS FOR DATA INTEGRITY
-- =============================================================================

-- Function to validate receipt hash
CREATE OR REPLACE FUNCTION validate_receipt_hash(receipt_body BYTEA)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN digest(receipt_body, 'sha256') = (
        SELECT receipt_hash FROM receipts 
        WHERE receipt_body = $1
    );
END;
$$ LANGUAGE plpgsql;

-- Function to calculate total revenue
CREATE OR REPLACE FUNCTION calculate_total_revenue(start_date TIMESTAMPTZ, end_date TIMESTAMPTZ)
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE(
        (SELECT SUM(price_micro_units) 
         FROM receipts 
         WHERE created_at BETWEEN start_date AND end_date), 
        0
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get top artifacts by usage
CREATE OR REPLACE FUNCTION get_top_artifacts(limit_count INTEGER)
RETURNS TABLE (
    artifact_hash BYTEA,
    usage_count BIGINT,
    total_revenue BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.artifact_hash,
        a.usage_count,
        COALESCE(SUM(r.price_micro_units), 0) as total_revenue
    FROM artifacts a
    LEFT JOIN receipts r ON a.artifact_hash = r.artifact_hash
    GROUP BY a.artifact_hash, a.usage_count
    ORDER BY total_revenue DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TRIGGERS FOR DATA CONSISTENCY
-- =============================================================================

-- Trigger to update artifact usage count
CREATE OR REPLACE FUNCTION update_artifact_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE artifacts 
    SET 
        usage_count = usage_count + 1,
        last_used_at = NEW.created_at
    WHERE artifact_hash = NEW.artifact_hash;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_artifact_usage
    AFTER INSERT ON receipts
    FOR EACH ROW
    EXECUTE FUNCTION update_artifact_usage();

-- Trigger to validate receipt data
CREATE OR REPLACE FUNCTION validate_receipt_data()
RETURNS TRIGGER AS $$
BEGIN
    -- Validate hash length
    IF length(NEW.receipt_hash) != 32 THEN
        RAISE EXCEPTION 'Invalid receipt hash length: %', length(NEW.receipt_hash);
    END IF;
    
    -- Validate other hash lengths
    IF length(NEW.artifact_hash) != 32 THEN
        RAISE EXCEPTION 'Invalid artifact hash length: %', length(NEW.artifact_hash);
    END IF;
    
    IF length(NEW.input_hash) != 32 THEN
        RAISE EXCEPTION 'Invalid input hash length: %', length(NEW.input_hash);
    END IF;
    
    IF length(NEW.output_hash) != 32 THEN
        RAISE EXCEPTION 'Invalid output hash length: %', length(NEW.output_hash);
    END IF;
    
    -- Validate positive values
    IF NEW.cycles_used <= 0 THEN
        RAISE EXCEPTION 'Invalid cycles used: %', NEW.cycles_used;
    END IF;
    
    IF NEW.price_micro_units < 0 THEN
        RAISE EXCEPTION 'Invalid price: %', NEW.price_micro_units;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_receipt_data
    BEFORE INSERT ON receipts
    FOR EACH ROW
    EXECUTE FUNCTION validate_receipt_data();

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================

-- Composite indexes for common queries
CREATE INDEX idx_receipts_artifact_created ON receipts(artifact_hash, created_at);
CREATE INDEX idx_receipts_cycles_created ON receipts(cycles_used, created_at);
CREATE INDEX idx_receipts_price_created ON receipts(price_micro_units, created_at);

-- Partial indexes for active data
CREATE INDEX idx_receipts_recent ON receipts(created_at) WHERE created_at > NOW() - INTERVAL '30 days';
CREATE INDEX idx_executions_recent ON executions(created_at) WHERE created_at > NOW() - INTERVAL '7 days';

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE receipts IS 'Immutable receipt storage - append-only';
COMMENT ON TABLE artifacts IS 'Code artifact registry with usage tracking';
COMMENT ON TABLE executions IS 'Execution log with performance metrics';
COMMENT ON TABLE settlements IS 'Payment settlement tracking';
COMMENT ON TABLE conformance_tests IS 'Conformance test results';
COMMENT ON TABLE benchmarks IS 'Performance benchmark data';

COMMENT ON COLUMN receipts.receipt_hash IS 'SHA256 hash of receipt_body (primary key)';
COMMENT ON COLUMN receipts.receipt_body IS 'CBOR-encoded receipt data';
COMMENT ON COLUMN receipts.price_micro_units IS 'Price in micro-units (1/1,000,000 of base currency)';

-- =============================================================================
-- END OF PRODUCTION SCHEMA
-- =============================================================================
