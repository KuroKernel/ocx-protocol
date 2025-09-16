-- =============================================
-- REPUTATION & TRUST SYSTEM
-- =============================================

-- Trust relationships between entities
CREATE TABLE trust_relationships (
    relationship_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluator_id UUID NOT NULL, -- Who is giving the rating
    target_id UUID NOT NULL, -- Who is being rated (provider or requester)
    
    -- Trust components
    reliability_score DECIMAL(4,3) CHECK (reliability_score BETWEEN 0 AND 1),
    performance_score DECIMAL(4,3) CHECK (performance_score BETWEEN 0 AND 1), 
    communication_score DECIMAL(4,3) CHECK (communication_score BETWEEN 0 AND 1),
    overall_trust_score DECIMAL(4,3) CHECK (overall_trust_score BETWEEN 0 AND 1),
    
    -- Supporting evidence
    interaction_count INTEGER DEFAULT 0,
    total_transaction_value_usdc DECIMAL(18,6) DEFAULT 0,
    dispute_count INTEGER DEFAULT 0,
    successful_completions INTEGER DEFAULT 0,
    
    -- Temporal decay
    last_interaction_at TIMESTAMPTZ,
    trust_decay_factor DECIMAL(4,3) DEFAULT 1.0,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(evaluator_id, target_id),
    INDEX idx_trust_target_score (target_id, overall_trust_score DESC),
    INDEX idx_trust_updated (updated_at DESC)
);

-- Reputation event types
CREATE TYPE reputation_event_type AS ENUM (
    'session_completed_successfully',
    'session_terminated_early', 
    'performance_exceeded_sla',
    'performance_below_sla',
    'dispute_resolved_in_favor',
    'dispute_resolved_against',
    'uptime_penalty',
    'security_incident',
    'compliance_violation',
    'exceptional_service',
    'referral_bonus',
    'staking_bonus',
    'slashing_penalty'
);

-- Provider reputation events (immutable log)
CREATE TABLE reputation_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID REFERENCES providers(provider_id),
    session_id UUID REFERENCES compute_sessions(session_id),
    event_type reputation_event_type NOT NULL,
    
    -- Event details
    impact_score DECIMAL(6,4), -- Can be negative for penalties
    event_description TEXT NOT NULL,
    evidence_hash TEXT, -- IPFS hash of supporting evidence
    
    -- Verification
    verified_by UUID, -- Admin or oracle that verified this event
    verification_timestamp TIMESTAMPTZ,
    verification_signature BYTEA,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_reputation_events_provider (provider_id, created_at DESC),
    INDEX idx_reputation_events_type (event_type, created_at DESC)
);

-- Reputation algorithm configuration
CREATE TABLE reputation_weights (
    weight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type reputation_event_type UNIQUE NOT NULL,
    base_weight DECIMAL(6,4) NOT NULL,
    decay_half_life_days INTEGER DEFAULT 30,
    max_impact_per_day DECIMAL(6,4),
    requires_verification BOOLEAN DEFAULT false,
    active_from TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_reputation_weights_active (active_from DESC)
);

-- Precomputed reputation scores (updated by background job)
CREATE TABLE provider_reputation_cache (
    provider_id UUID PRIMARY KEY REFERENCES providers(provider_id),
    overall_score DECIMAL(4,3) NOT NULL,
    reliability_component DECIMAL(4,3) NOT NULL,
    performance_component DECIMAL(4,3) NOT NULL,
    availability_component DECIMAL(4,3) NOT NULL,
    dispute_resolution_component DECIMAL(4,3) NOT NULL,
    
    -- Statistical confidence
    confidence_interval DECIMAL(4,3), -- Standard error
    sample_size INTEGER, -- Number of interactions
    
    -- Decay tracking
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    next_decay_update TIMESTAMPTZ,
    
    INDEX idx_reputation_cache_score (overall_score DESC)
);

-- Initialize default reputation weights
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
