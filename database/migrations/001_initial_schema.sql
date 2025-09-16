-- Migration: Initial Production Schema
-- Description: Creates the complete production database schema
-- Version: 1.0.0
-- Dependencies: PostgreSQL 13+, TimescaleDB 2.0+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Run all schema files in order
\i database/schema/01_core_tables.sql
\i database/schema/02_reputation_system.sql
\i database/schema/03_financial_settlement.sql

-- Create additional indexes for performance
CREATE INDEX CONCURRENTLY idx_compute_units_price_availability 
ON compute_units (base_price_per_hour_usdc, current_availability) 
WHERE current_availability = 'available';

CREATE INDEX CONCURRENTLY idx_orders_expiry_status 
ON compute_orders (expires_at, order_status) 
WHERE order_status = 'pending_matching';

CREATE INDEX CONCURRENTLY idx_sessions_provider_time 
ON compute_sessions (provider_id, session_started_at DESC);

-- Create materialized views for common queries
CREATE MATERIALIZED VIEW provider_summary AS
SELECT 
    p.provider_id,
    p.geographic_region,
    p.reputation_score,
    p.status,
    COUNT(cu.unit_id) as total_units,
    COUNT(CASE WHEN cu.current_availability = 'available' THEN 1 END) as available_units,
    AVG(cu.base_price_per_hour_usdc) as avg_price_per_hour,
    SUM(cu.successful_jobs_completed) as total_jobs_completed
FROM providers p
LEFT JOIN compute_units cu ON p.provider_id = cu.provider_id
GROUP BY p.provider_id, p.geographic_region, p.reputation_score, p.status;

CREATE UNIQUE INDEX idx_provider_summary_pk ON provider_summary (provider_id);

-- Create refresh function for materialized views
CREATE OR REPLACE FUNCTION refresh_provider_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY provider_summary;
END;
$$ LANGUAGE plpgsql;

-- Create function to update reputation scores
CREATE OR REPLACE FUNCTION update_provider_reputation(p_provider_id UUID)
RETURNS void AS $$
DECLARE
    v_reputation_score DECIMAL(4,3);
    v_reliability DECIMAL(4,3);
    v_performance DECIMAL(4,3);
    v_availability DECIMAL(4,3);
    v_communication DECIMAL(4,3);
    v_economic DECIMAL(4,3);
    v_confidence_interval DECIMAL(4,3);
    v_sample_size INTEGER;
BEGIN
    -- Calculate reputation components (simplified algorithm)
    -- In production, this would call the Go reputation engine
    
    -- Reliability: based on successful sessions vs total sessions
    SELECT 
        COALESCE(
            SUM(CASE WHEN cs.session_status = 'completed' THEN 1 ELSE 0 END)::DECIMAL / 
            NULLIF(COUNT(*), 0), 
            0.5
        ) INTO v_reliability
    FROM compute_sessions cs
    WHERE cs.provider_id = p_provider_id
    AND cs.session_started_at > NOW() - INTERVAL '90 days';
    
    -- Performance: based on SLA compliance
    SELECT 
        COALESCE(
            AVG(CASE 
                WHEN cs.session_status = 'completed' 
                AND cs.session_ended_at <= cs.estimated_end_time 
                THEN 1.0 
                ELSE 0.0 
            END), 
            0.5
        ) INTO v_performance
    FROM compute_sessions cs
    WHERE cs.provider_id = p_provider_id
    AND cs.session_started_at > NOW() - INTERVAL '90 days';
    
    -- Availability: based on uptime
    SELECT 
        COALESCE(
            AVG(CASE 
                WHEN cu.current_availability = 'available' THEN 1.0 
                ELSE 0.0 
            END), 
            0.5
        ) INTO v_availability
    FROM compute_units cu
    WHERE cu.provider_id = p_provider_id;
    
    -- Communication: based on dispute resolution
    SELECT 
        COALESCE(
            1.0 - (COUNT(d.dispute_id)::DECIMAL / NULLIF(COUNT(cs.session_id), 0)), 
            0.5
        ) INTO v_communication
    FROM compute_sessions cs
    LEFT JOIN disputes d ON cs.session_id = d.session_id 
        AND d.defendant_id = p_provider_id
        AND d.dispute_status = 'resolved'
        AND d.awarded_to_plaintiff = true
    WHERE cs.provider_id = p_provider_id
    AND cs.session_started_at > NOW() - INTERVAL '90 days';
    
    -- Economic: based on payment reliability
    SELECT 
        COALESCE(
            AVG(CASE 
                WHEN st.settlement_status = 'released' 
                AND st.settlement_timestamp <= cs.session_ended_at + INTERVAL '24 hours'
                THEN 1.0 
                ELSE 0.0 
            END), 
            0.5
        ) INTO v_economic
    FROM compute_sessions cs
    LEFT JOIN settlement_transactions st ON cs.session_id = st.session_id
    WHERE cs.provider_id = p_provider_id
    AND cs.session_started_at > NOW() - INTERVAL '90 days';
    
    -- Calculate overall score (weighted average)
    v_reputation_score := (
        v_reliability * 0.30 +
        v_performance * 0.25 +
        v_availability * 0.20 +
        v_communication * 0.10 +
        v_economic * 0.15
    );
    
    -- Calculate confidence interval (simplified)
    SELECT COUNT(*) INTO v_sample_size
    FROM compute_sessions cs
    WHERE cs.provider_id = p_provider_id
    AND cs.session_started_at > NOW() - INTERVAL '90 days';
    
    v_confidence_interval := CASE 
        WHEN v_sample_size < 5 THEN 0.5
        WHEN v_sample_size < 20 THEN 0.2
        ELSE 0.1
    END;
    
    -- Update or insert reputation cache
    INSERT INTO provider_reputation_cache (
        provider_id,
        overall_score,
        reliability_component,
        performance_component,
        availability_component,
        dispute_resolution_component,
        confidence_interval,
        sample_size,
        last_updated,
        next_decay_update
    ) VALUES (
        p_provider_id,
        v_reputation_score,
        v_reliability,
        v_performance,
        v_availability,
        v_communication,
        v_confidence_interval,
        v_sample_size,
        NOW(),
        NOW() + INTERVAL '1 day'
    )
    ON CONFLICT (provider_id) DO UPDATE SET
        overall_score = EXCLUDED.overall_score,
        reliability_component = EXCLUDED.reliability_component,
        performance_component = EXCLUDED.performance_component,
        availability_component = EXCLUDED.availability_component,
        dispute_resolution_component = EXCLUDED.dispute_resolution_component,
        confidence_interval = EXCLUDED.confidence_interval,
        sample_size = EXCLUDED.sample_size,
        last_updated = EXCLUDED.last_updated,
        next_decay_update = EXCLUDED.next_decay_update;
    
    -- Update provider table
    UPDATE providers 
    SET reputation_score = v_reputation_score
    WHERE provider_id = p_provider_id;
    
END;
$$ LANGUAGE plpgsql;

-- Create function to clean up expired orders
CREATE OR REPLACE FUNCTION cleanup_expired_orders()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- Cancel expired orders
    UPDATE compute_orders 
    SET 
        order_status = 'cancelled',
        cancelled_at = NOW(),
        cancellation_reason = 'expired'
    WHERE order_status = 'pending_matching'
    AND expires_at < NOW();
    
    GET DIAGNOSTICS v_count = ROW_COUNT;
    
    -- Clean up associated matches
    DELETE FROM order_matches om
    WHERE om.order_id IN (
        SELECT order_id FROM compute_orders 
        WHERE order_status = 'cancelled'
        AND cancellation_reason = 'expired'
    );
    
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to calculate session costs
CREATE OR REPLACE FUNCTION calculate_session_cost(
    p_session_id UUID,
    p_hours DECIMAL(8,2)
)
RETURNS TABLE(
    base_cost DECIMAL(18,6),
    usage_premium DECIMAL(18,6),
    total_cost DECIMAL(18,6)
) AS $$
DECLARE
    v_base_price DECIMAL(10,6);
    v_avg_utilization INTEGER;
    v_utilization_premium DECIMAL(18,6);
BEGIN
    -- Get base price and average utilization
    SELECT 
        cs.agreed_price_per_hour_usdc,
        COALESCE(AVG(sm.gpu_utilization_percent), 0)
    INTO v_base_price, v_avg_utilization
    FROM compute_sessions cs
    LEFT JOIN session_metrics sm ON cs.session_id = sm.session_id
    WHERE cs.session_id = p_session_id
    GROUP BY cs.agreed_price_per_hour_usdc;
    
    -- Calculate base cost
    v_base_price := COALESCE(v_base_price, 0);
    
    -- Calculate utilization premium (bonus for high utilization)
    v_utilization_premium := CASE 
        WHEN v_avg_utilization > 90 THEN v_base_price * p_hours * 0.1  -- 10% bonus
        WHEN v_avg_utilization > 80 THEN v_base_price * p_hours * 0.05 -- 5% bonus
        ELSE 0
    END;
    
    RETURN QUERY SELECT 
        v_base_price * p_hours as base_cost,
        v_utilization_premium as usage_premium,
        (v_base_price * p_hours) + v_utilization_premium as total_cost;
END;
$$ LANGUAGE plpgsql;

-- Create scheduled jobs (requires pg_cron extension)
-- These would be set up in production with pg_cron

-- Refresh materialized views every hour
-- SELECT cron.schedule('refresh-provider-summary', '0 * * * *', 'SELECT refresh_provider_summary();');

-- Update reputation scores every 6 hours
-- SELECT cron.schedule('update-reputation', '0 */6 * * *', 'SELECT update_provider_reputation(provider_id) FROM providers WHERE status = ''active'';');

-- Clean up expired orders every 15 minutes
-- SELECT cron.schedule('cleanup-orders', '*/15 * * * *', 'SELECT cleanup_expired_orders();');

-- Create audit triggers for important tables
CREATE OR REPLACE FUNCTION audit_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- In production, this would log changes to an audit table
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers
CREATE TRIGGER audit_providers
    AFTER INSERT OR UPDATE OR DELETE ON providers
    FOR EACH ROW EXECUTE FUNCTION audit_trigger();

CREATE TRIGGER audit_compute_orders
    AFTER INSERT OR UPDATE OR DELETE ON compute_orders
    FOR EACH ROW EXECUTE FUNCTION audit_trigger();

CREATE TRIGGER audit_compute_sessions
    AFTER INSERT OR UPDATE OR DELETE ON compute_sessions
    FOR EACH ROW EXECUTE FUNCTION audit_trigger();

-- Create views for common queries
CREATE VIEW active_sessions AS
SELECT 
    cs.session_id,
    cs.order_id,
    cs.provider_id,
    p.operator_address,
    p.geographic_region,
    cs.agreed_price_per_hour_usdc,
    cs.session_status,
    cs.session_started_at,
    cs.estimated_end_time,
    EXTRACT(EPOCH FROM (NOW() - cs.session_started_at))/3600 as hours_running
FROM compute_sessions cs
JOIN providers p ON cs.provider_id = p.provider_id
WHERE cs.session_status = 'active';

CREATE VIEW available_units AS
SELECT 
    cu.unit_id,
    cu.provider_id,
    p.operator_address,
    p.geographic_region,
    p.reputation_score,
    cu.hardware_type,
    cu.gpu_model,
    cu.gpu_memory_gb,
    cu.cpu_cores,
    cu.ram_gb,
    cu.base_price_per_hour_usdc,
    cu.provisioning_time_seconds
FROM compute_units cu
JOIN providers p ON cu.provider_id = p.provider_id
WHERE cu.current_availability = 'available'
AND p.status = 'active'
AND p.last_heartbeat > NOW() - INTERVAL '5 minutes';

-- Insert initial data
INSERT INTO reputation_weights (event_type, base_weight, decay_half_life_days, max_impact_per_day, requires_verification) VALUES
('session_completed_successfully', 0.1000, 30, 0.5000, false),
('session_terminated_early', -0.2000, 30, -1.0000, false),
('performance_exceeded_sla', 0.1500, 60, 0.3000, false),
('performance_below_sla', -0.1000, 30, -0.5000, false),
('dispute_resolved_in_favor', 0.0500, 90, 0.2000, true),
('dispute_resolved_against', -0.3000, 90, -1.0000, true),
('uptime_penalty', -0.0500, 7, -0.2000, false),
('security_incident', -0.5000, 365, -2.0000, true),
('compliance_violation', -0.4000, 180, -1.5000, true),
('exceptional_service', 0.2000, 60, 0.4000, false),
('referral_bonus', 0.0250, 30, 0.1000, false),
('staking_bonus', 0.0100, 7, 0.0500, false),
('slashing_penalty', -1.0000, 365, -5.0000, true);

-- Create initial admin user (for testing)
INSERT INTO providers (
    provider_id,
    public_key,
    operator_address,
    geographic_region,
    data_center_tier,
    reputation_score,
    status,
    registration_timestamp
) VALUES (
    gen_random_uuid(),
    decode('0000000000000000000000000000000000000000000000000000000000000000', 'hex'),
    'admin@ocx.world',
    'US',
    4,
    1.000,
    'active',
    NOW()
);

COMMIT;
