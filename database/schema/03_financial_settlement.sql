-- =============================================
-- FINANCIAL SETTLEMENT SYSTEM
-- =============================================

-- Escrow status types
CREATE TYPE escrow_status_type AS ENUM (
    'pending_confirmation', 'active', 'releasing', 'completed', 'disputed', 'refunded'
);

-- Escrow and payment tracking
CREATE TABLE escrow_accounts (
    escrow_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES compute_orders(order_id),
    requester_id UUID NOT NULL,
    
    -- Amounts in USDC (6 decimals)
    total_escrowed_usdc DECIMAL(18,6) NOT NULL,
    amount_released_usdc DECIMAL(18,6) DEFAULT 0,
    amount_disputed_usdc DECIMAL(18,6) DEFAULT 0,
    amount_refunded_usdc DECIMAL(18,6) DEFAULT 0,
    
    -- Blockchain integration
    escrow_contract_address TEXT NOT NULL,
    deposit_tx_hash TEXT NOT NULL,
    deposit_block_number BIGINT,
    deposit_confirmed_at TIMESTAMPTZ,
    
    -- Status tracking
    escrow_status escrow_status_type DEFAULT 'pending_confirmation',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_escrow_status (escrow_status, created_at DESC),
    INDEX idx_escrow_order (order_id)
);

-- Settlement status types
CREATE TYPE settlement_status_type AS ENUM (
    'pending', 'approved', 'released', 'disputed', 'cancelled'
);

-- Individual payments to providers
CREATE TABLE settlement_transactions (
    settlement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES compute_sessions(session_id),
    escrow_id UUID REFERENCES escrow_accounts(escrow_id),
    provider_id UUID REFERENCES providers(provider_id),
    
    -- Payment breakdown
    base_payment_usdc DECIMAL(18,6) NOT NULL,
    performance_bonus_usdc DECIMAL(18,6) DEFAULT 0,
    protocol_fee_usdc DECIMAL(18,6) NOT NULL,
    provider_net_usdc DECIMAL(18,6) NOT NULL,
    
    -- Settlement details
    settlement_status settlement_status_type DEFAULT 'pending',
    release_tx_hash TEXT,
    release_block_number BIGINT,
    settlement_timestamp TIMESTAMPTZ,
    
    -- Verification
    usage_proof_hash TEXT, -- IPFS hash of usage metrics
    provider_signature BYTEA, -- Provider signs off on payment
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_settlements_provider (provider_id, settlement_status, created_at DESC),
    INDEX idx_settlements_session (session_id)
);

-- Protocol fee types
CREATE TYPE protocol_fee_type AS ENUM (
    'marketplace_fee', 'settlement_fee', 'dispute_resolution_fee', 'staking_reward'
);

-- Protocol revenue tracking
CREATE TABLE protocol_revenue (
    revenue_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_id UUID REFERENCES settlement_transactions(settlement_id),
    fee_type protocol_fee_type NOT NULL,
    fee_amount_usdc DECIMAL(18,6) NOT NULL,
    fee_percentage DECIMAL(5,4), -- What % was charged
    
    -- Revenue allocation
    treasury_amount_usdc DECIMAL(18,6),
    burn_amount_usdc DECIMAL(18,6),
    rewards_pool_amount_usdc DECIMAL(18,6),
    
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_protocol_revenue_type_date (fee_type, collected_at DESC)
);

-- =============================================
-- DISPUTE RESOLUTION
-- =============================================

-- Dispute type enumeration
CREATE TYPE dispute_type_enum AS ENUM (
    'non_delivery', 'performance_below_sla', 'early_termination', 
    'payment_dispute', 'resource_misrepresentation', 'security_breach',
    'contract_violation', 'force_majeure'
);

-- Dispute status enumeration
CREATE TYPE dispute_status_type AS ENUM (
    'filed', 'acknowledged', 'evidence_collection', 'arbitration', 
    'resolved', 'appealed', 'final'
);

-- Dispute cases between parties
CREATE TABLE disputes (
    dispute_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES compute_sessions(session_id),
    plaintiff_id UUID NOT NULL, -- Who filed the dispute
    defendant_id UUID NOT NULL, -- Who is being disputed
    
    -- Dispute details
    dispute_type dispute_type_enum NOT NULL,
    disputed_amount_usdc DECIMAL(18,6) NOT NULL,
    claim_description TEXT NOT NULL,
    evidence_hashes TEXT[], -- IPFS hashes of evidence files
    
    -- Resolution process
    dispute_status dispute_status_type DEFAULT 'filed',
    assigned_arbitrator_id UUID,
    arbitration_fee_usdc DECIMAL(18,6),
    
    -- Timeline
    filed_at TIMESTAMPTZ DEFAULT NOW(),
    response_due_at TIMESTAMPTZ,
    resolution_target_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    
    -- Resolution outcome
    resolution_summary TEXT,
    awarded_amount_usdc DECIMAL(18,6),
    awarded_to_plaintiff BOOLEAN,
    
    INDEX idx_disputes_status (dispute_status, filed_at DESC),
    INDEX idx_disputes_arbitrator (assigned_arbitrator_id, dispute_status)
);
