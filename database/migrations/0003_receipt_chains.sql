-- OCX Protocol Receipt Chains Database Schema
-- Migration: 0003_receipt_chains.sql
-- Description: Adds support for receipt chaining (Level 3: Composable Trust Chains)
--
-- Receipt chains enable cryptographic linking of receipts, creating verifiable
-- audit trails: invoice → GST match → bank confirmation → AI credit score → loan disbursement

-- Add chain-related columns to existing receipts table
ALTER TABLE ocx_receipts_v1_1
    ADD COLUMN IF NOT EXISTS receipt_hash BYTEA,
    ADD COLUMN IF NOT EXISTS prev_receipt_hash BYTEA,
    ADD COLUMN IF NOT EXISTS request_digest BYTEA,
    ADD COLUMN IF NOT EXISTS witness_signatures BYTEA[],
    ADD COLUMN IF NOT EXISTS chain_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS chain_seq BIGINT DEFAULT 0;

-- Add constraints for new columns
ALTER TABLE ocx_receipts_v1_1
    ADD CONSTRAINT chk_receipt_hash_length
        CHECK (receipt_hash IS NULL OR OCTET_LENGTH(receipt_hash) = 32),
    ADD CONSTRAINT chk_prev_receipt_hash_length
        CHECK (prev_receipt_hash IS NULL OR OCTET_LENGTH(prev_receipt_hash) = 32),
    ADD CONSTRAINT chk_request_digest_length
        CHECK (request_digest IS NULL OR OCTET_LENGTH(request_digest) = 32);

-- Create index for receipt hash lookup (critical for chain verification)
CREATE UNIQUE INDEX IF NOT EXISTS idx_receipts_receipt_hash
    ON ocx_receipts_v1_1 (receipt_hash)
    WHERE receipt_hash IS NOT NULL;

-- Create index for chain traversal (finding children of a receipt)
CREATE INDEX IF NOT EXISTS idx_receipts_prev_receipt_hash
    ON ocx_receipts_v1_1 (prev_receipt_hash)
    WHERE prev_receipt_hash IS NOT NULL;

-- Create index for chain queries
CREATE INDEX IF NOT EXISTS idx_receipts_chain_id
    ON ocx_receipts_v1_1 (chain_id)
    WHERE chain_id IS NOT NULL;

-- Create index for chain ordering
CREATE INDEX IF NOT EXISTS idx_receipts_chain_seq
    ON ocx_receipts_v1_1 (chain_id, chain_seq)
    WHERE chain_id IS NOT NULL;

-- Create dedicated chain metadata table for chain-level information
CREATE TABLE IF NOT EXISTS ocx_receipt_chains (
    chain_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    genesis_receipt_hash BYTEA NOT NULL CHECK (OCTET_LENGTH(genesis_receipt_hash) = 32),
    head_receipt_hash BYTEA NOT NULL CHECK (OCTET_LENGTH(head_receipt_hash) = 32),
    chain_length BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Create indexes for chain metadata
CREATE INDEX IF NOT EXISTS idx_receipt_chains_genesis
    ON ocx_receipt_chains (genesis_receipt_hash);

CREATE INDEX IF NOT EXISTS idx_receipt_chains_head
    ON ocx_receipt_chains (head_receipt_hash);

CREATE INDEX IF NOT EXISTS idx_receipt_chains_created
    ON ocx_receipt_chains (created_at);

-- Create chain verification cache for performance
CREATE TABLE IF NOT EXISTS ocx_chain_verification_cache (
    receipt_hash BYTEA PRIMARY KEY CHECK (OCTET_LENGTH(receipt_hash) = 32),
    chain_id VARCHAR(255),
    chain_length INTEGER,
    genesis_hash BYTEA CHECK (OCTET_LENGTH(genesis_hash) = 32),
    is_valid BOOLEAN,
    last_verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    verification_ms INTEGER,
    error_message TEXT
);

-- Create index for verification cache
CREATE INDEX IF NOT EXISTS idx_chain_verification_cache_chain
    ON ocx_chain_verification_cache (chain_id);

CREATE INDEX IF NOT EXISTS idx_chain_verification_cache_verified
    ON ocx_chain_verification_cache (last_verified_at);

-- Create function to get chain ancestors
CREATE OR REPLACE FUNCTION get_chain_ancestors(
    p_receipt_hash BYTEA,
    p_max_depth INTEGER DEFAULT 100
)
RETURNS TABLE (
    receipt_hash BYTEA,
    prev_receipt_hash BYTEA,
    chain_seq BIGINT,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    depth INTEGER
) AS $$
WITH RECURSIVE chain AS (
    -- Base case: start with the given receipt
    SELECT
        r.receipt_hash,
        r.prev_receipt_hash,
        r.chain_seq,
        r.started_at,
        r.finished_at,
        1 AS depth
    FROM ocx_receipts_v1_1 r
    WHERE r.receipt_hash = p_receipt_hash

    UNION ALL

    -- Recursive case: follow prev_receipt_hash
    SELECT
        r.receipt_hash,
        r.prev_receipt_hash,
        r.chain_seq,
        r.started_at,
        r.finished_at,
        c.depth + 1
    FROM ocx_receipts_v1_1 r
    INNER JOIN chain c ON r.receipt_hash = c.prev_receipt_hash
    WHERE c.depth < p_max_depth
)
SELECT * FROM chain ORDER BY depth;
$$ LANGUAGE SQL;

-- Create function to verify chain integrity
CREATE OR REPLACE FUNCTION verify_chain_integrity(
    p_receipt_hash BYTEA
)
RETURNS JSON AS $$
DECLARE
    v_chain_length INTEGER;
    v_genesis_hash BYTEA;
    v_errors TEXT[] := '{}';
    v_prev_finished TIMESTAMP WITH TIME ZONE;
    v_current RECORD;
    v_is_valid BOOLEAN := true;
BEGIN
    -- Get all ancestors
    FOR v_current IN
        SELECT * FROM get_chain_ancestors(p_receipt_hash, 1000) ORDER BY depth
    LOOP
        -- Check timestamp ordering
        IF v_prev_finished IS NOT NULL AND v_current.started_at < v_prev_finished THEN
            v_is_valid := false;
            v_errors := array_append(v_errors,
                format('Timestamp violation at depth %s: started_at < prev finished_at', v_current.depth));
        END IF;

        v_prev_finished := v_current.finished_at;
        v_chain_length := v_current.depth;

        -- Genesis is the receipt with no prev_receipt_hash
        IF v_current.prev_receipt_hash IS NULL THEN
            v_genesis_hash := v_current.receipt_hash;
        END IF;
    END LOOP;

    -- Cache the result
    INSERT INTO ocx_chain_verification_cache (
        receipt_hash, chain_length, genesis_hash, is_valid,
        last_verified_at, error_message
    ) VALUES (
        p_receipt_hash, v_chain_length, v_genesis_hash, v_is_valid,
        NOW(), array_to_string(v_errors, '; ')
    )
    ON CONFLICT (receipt_hash) DO UPDATE SET
        chain_length = EXCLUDED.chain_length,
        genesis_hash = EXCLUDED.genesis_hash,
        is_valid = EXCLUDED.is_valid,
        last_verified_at = EXCLUDED.last_verified_at,
        error_message = EXCLUDED.error_message;

    RETURN json_build_object(
        'valid', v_is_valid,
        'chain_length', v_chain_length,
        'genesis_hash', encode(v_genesis_hash, 'hex'),
        'head_hash', encode(p_receipt_hash, 'hex'),
        'errors', v_errors
    );
END;
$$ LANGUAGE plpgsql;

-- Create function to append receipt to chain
CREATE OR REPLACE FUNCTION append_to_chain(
    p_chain_id VARCHAR(255),
    p_receipt_hash BYTEA,
    p_prev_receipt_hash BYTEA
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_head BYTEA;
    v_new_seq BIGINT;
BEGIN
    -- Verify the prev_receipt_hash is the current chain head
    SELECT head_receipt_hash INTO v_current_head
    FROM ocx_receipt_chains
    WHERE chain_id = p_chain_id;

    IF v_current_head IS NULL THEN
        -- Chain doesn't exist, create it
        INSERT INTO ocx_receipt_chains (
            chain_id, genesis_receipt_hash, head_receipt_hash, chain_length
        ) VALUES (
            p_chain_id, p_receipt_hash, p_receipt_hash, 1
        );
        v_new_seq := 1;
    ELSIF v_current_head != p_prev_receipt_hash THEN
        -- prev_receipt_hash doesn't match current head
        RAISE EXCEPTION 'Chain head mismatch: expected %, got %',
            encode(v_current_head, 'hex'), encode(p_prev_receipt_hash, 'hex');
    ELSE
        -- Update chain head
        UPDATE ocx_receipt_chains
        SET head_receipt_hash = p_receipt_hash,
            chain_length = chain_length + 1,
            updated_at = NOW()
        WHERE chain_id = p_chain_id
        RETURNING chain_length INTO v_new_seq;
    END IF;

    -- Update receipt with chain info
    UPDATE ocx_receipts_v1_1
    SET chain_id = p_chain_id,
        chain_seq = v_new_seq,
        prev_receipt_hash = p_prev_receipt_hash
    WHERE receipt_hash = p_receipt_hash;

    -- Invalidate verification cache for this chain
    DELETE FROM ocx_chain_verification_cache
    WHERE chain_id = p_chain_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Create view for chain statistics
CREATE OR REPLACE VIEW ocx_chain_stats AS
SELECT
    chain_id,
    COUNT(*) as receipt_count,
    MIN(chain_seq) as min_seq,
    MAX(chain_seq) as max_seq,
    MIN(started_at) as chain_start,
    MAX(finished_at) as chain_end,
    SUM(gas_used) as total_gas,
    COUNT(DISTINCT issuer_id) as unique_issuers
FROM ocx_receipts_v1_1
WHERE chain_id IS NOT NULL
GROUP BY chain_id;

-- Create trigger to update chain head on receipt insert
CREATE OR REPLACE FUNCTION update_chain_head_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.chain_id IS NOT NULL AND NEW.receipt_hash IS NOT NULL THEN
        -- Update or insert chain metadata
        INSERT INTO ocx_receipt_chains (
            chain_id, genesis_receipt_hash, head_receipt_hash, chain_length
        ) VALUES (
            NEW.chain_id, NEW.receipt_hash, NEW.receipt_hash, 1
        )
        ON CONFLICT (chain_id) DO UPDATE SET
            head_receipt_hash = NEW.receipt_hash,
            chain_length = ocx_receipt_chains.chain_length + 1,
            updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger (drop first if exists to allow re-running migration)
DROP TRIGGER IF EXISTS trg_update_chain_head ON ocx_receipts_v1_1;
CREATE TRIGGER trg_update_chain_head
    AFTER INSERT ON ocx_receipts_v1_1
    FOR EACH ROW
    EXECUTE FUNCTION update_chain_head_trigger();

-- Add comments
COMMENT ON COLUMN ocx_receipts_v1_1.receipt_hash IS 'SHA-256 hash of canonical CBOR for chain linking';
COMMENT ON COLUMN ocx_receipts_v1_1.prev_receipt_hash IS 'Reference to previous receipt in chain';
COMMENT ON COLUMN ocx_receipts_v1_1.request_digest IS 'Hash of original request for binding';
COMMENT ON COLUMN ocx_receipts_v1_1.witness_signatures IS 'Additional witness signatures for multi-party verification';
COMMENT ON COLUMN ocx_receipts_v1_1.chain_id IS 'Logical chain identifier';
COMMENT ON COLUMN ocx_receipts_v1_1.chain_seq IS 'Sequence number within the chain';

COMMENT ON TABLE ocx_receipt_chains IS 'Chain-level metadata for receipt chains';
COMMENT ON TABLE ocx_chain_verification_cache IS 'Cached chain verification results for performance';

COMMENT ON FUNCTION get_chain_ancestors IS 'Recursively retrieves all ancestors of a receipt up to max_depth';
COMMENT ON FUNCTION verify_chain_integrity IS 'Verifies timestamp ordering and chain integrity, caches result';
COMMENT ON FUNCTION append_to_chain IS 'Appends a receipt to an existing chain with validation';
