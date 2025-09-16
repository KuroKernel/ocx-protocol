-- OCX Protocol: Production Database Schema
-- PostgreSQL + TimescaleDB for time-series metrics
-- Version: 1.0.0

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- =============================================
-- CORE PROVIDER & HARDWARE REGISTRY
-- =============================================

-- Provider status enumeration
CREATE TYPE provider_status AS ENUM (
    'pending_verification', 'active', 'maintenance', 
    'slashed', 'voluntary_exit', 'terminated'
);

-- Physical compute resources in the network
CREATE TABLE providers (
    provider_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key BYTEA NOT NULL UNIQUE, -- Ed25519 for signing
    operator_address TEXT NOT NULL,
    geographic_region TEXT NOT NULL, -- ISO 3166-1 alpha-2
    data_center_tier INTEGER CHECK (data_center_tier BETWEEN 1 AND 4),
    network_bandwidth_gbps DECIMAL(8,2),
    power_cost_per_kwh DECIMAL(6,4),
    carbon_intensity_g_co2_per_kwh INTEGER,
    compliance_certifications JSONB, -- SOC2, ISO27001, etc
    reputation_score DECIMAL(4,3) DEFAULT 0.500,
    collateral_staked_usdc DECIMAL(18,6) DEFAULT 0,
    total_revenue_earned DECIMAL(18,6) DEFAULT 0,
    slashing_incidents INTEGER DEFAULT 0,
    last_heartbeat TIMESTAMPTZ,
    registration_timestamp TIMESTAMPTZ DEFAULT NOW(),
    status provider_status DEFAULT 'pending_verification',
    
    -- Indexes for geographic and reputation queries
    INDEX idx_providers_region_reputation (geographic_region, reputation_score DESC),
    INDEX idx_providers_status_heartbeat (status, last_heartbeat DESC)
);

-- Hardware type enumerations
CREATE TYPE compute_hardware_type AS ENUM (
    'gpu_training', 'gpu_inference', 'cpu_general', 'fpga_custom'
);

CREATE TYPE storage_type AS ENUM (
    'nvme_ssd', 'sata_ssd', 'hdd', 'network_attached'
);

CREATE TYPE availability_status AS ENUM (
    'available', 'reserved', 'in_use', 'maintenance', 'offline'
);

-- Individual GPU/compute units owned by providers
CREATE TABLE compute_units (
    unit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID REFERENCES providers(provider_id) ON DELETE CASCADE,
    hardware_type compute_hardware_type NOT NULL,
    gpu_model TEXT, -- 'RTX_4090', 'H100_SXM5', 'A100_80GB'
    gpu_memory_gb INTEGER,
    gpu_compute_capability TEXT, -- '8.9', '9.0'
    cpu_cores INTEGER,
    ram_gb INTEGER,
    storage_type storage_type,
    storage_capacity_gb INTEGER,
    network_interface_speed_gbps DECIMAL(6,2),
    pcie_generation INTEGER,
    cooling_solution TEXT,
    power_draw_watts INTEGER,
    base_price_per_hour_usdc DECIMAL(10,6),
    current_availability availability_status DEFAULT 'available',
    total_uptime_hours DECIMAL(12,2) DEFAULT 0,
    successful_jobs_completed INTEGER DEFAULT 0,
    last_benchmark_score INTEGER,
    last_benchmark_timestamp TIMESTAMPTZ,
    provisioning_time_seconds INTEGER DEFAULT 180,
    
    -- Composite index for matching queries
    INDEX idx_compute_units_matching (hardware_type, gpu_model, current_availability, base_price_per_hour_usdc),
    INDEX idx_compute_units_provider_status (provider_id, current_availability)
);

-- =============================================
-- ORDER MANAGEMENT SYSTEM
-- =============================================

-- Workload classification
CREATE TYPE workload_classification AS ENUM (
    'llm_training', 'llm_inference', 'diffusion_training', 'diffusion_inference',
    'rl_training', 'data_processing', 'simulation', 'rendering', 'general_compute'
);

-- Order status types
CREATE TYPE order_status_type AS ENUM (
    'pending_matching', 'matched', 'provisioning', 'active', 'completed', 
    'cancelled', 'failed', 'disputed'
);

-- Compute resource requests from users
CREATE TABLE compute_orders (
    order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID NOT NULL, -- Links to external identity system
    requester_public_key BYTEA NOT NULL,
    
    -- Resource requirements (exact matching)
    required_hardware_type compute_hardware_type NOT NULL,
    required_gpu_model TEXT, -- NULL means any GPU of the type
    required_gpu_memory_gb INTEGER,
    required_cpu_cores INTEGER,
    required_ram_gb INTEGER,
    required_storage_gb INTEGER,
    
    -- Execution parameters
    estimated_duration_hours DECIMAL(8,2) NOT NULL,
    max_price_per_hour_usdc DECIMAL(10,6) NOT NULL,
    workload_type workload_classification,
    container_image_uri TEXT,
    startup_script TEXT,
    environment_variables JSONB,
    
    -- Geographic and quality preferences
    preferred_regions TEXT[] DEFAULT '{}',
    min_provider_reputation DECIMAL(4,3) DEFAULT 0.300,
    max_provisioning_time_seconds INTEGER DEFAULT 300,
    require_compliance_certs TEXT[] DEFAULT '{}',
    
    -- Order lifecycle
    order_status order_status_type DEFAULT 'pending_matching',
    total_budget_usdc DECIMAL(18,6) NOT NULL,
    escrow_tx_hash TEXT, -- Blockchain transaction for escrow
    placed_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    matched_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    
    -- Constraints
    CONSTRAINT valid_duration CHECK (estimated_duration_hours > 0 AND estimated_duration_hours <= 720), -- Max 30 days
    CONSTRAINT valid_budget CHECK (total_budget_usdc >= max_price_per_hour_usdc * estimated_duration_hours),
    CONSTRAINT valid_expiry CHECK (expires_at > placed_at),
    
    INDEX idx_orders_matching (order_status, required_hardware_type, max_price_per_hour_usdc),
    INDEX idx_orders_requester_status (requester_id, order_status, placed_at DESC)
);

-- Match status types
CREATE TYPE match_status_type AS ENUM (
    'proposed', 'provider_accepted', 'requester_accepted', 'confirmed', 'rejected', 'expired'
);

-- Order matching results (many-to-many: orders can match multiple units)
CREATE TABLE order_matches (
    match_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES compute_orders(order_id),
    unit_id UUID REFERENCES compute_units(unit_id),
    provider_id UUID REFERENCES providers(provider_id),
    matched_price_per_hour_usdc DECIMAL(10,6) NOT NULL,
    matching_score DECIMAL(5,4), -- Algorithm confidence 0-1
    matched_at TIMESTAMPTZ DEFAULT NOW(),
    match_status match_status_type DEFAULT 'proposed',
    provider_accepted_at TIMESTAMPTZ,
    requester_accepted_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    UNIQUE(order_id, unit_id),
    INDEX idx_matches_order_status (order_id, match_status),
    INDEX idx_matches_provider_pending (provider_id, match_status, matched_at DESC)
);

-- =============================================
-- ACTIVE SESSIONS & MONITORING
-- =============================================

-- Session status types
CREATE TYPE session_status_type AS ENUM (
    'provisioning', 'active', 'completed', 'terminated_user', 
    'terminated_provider', 'failed', 'disputed'
);

-- Live compute sessions
CREATE TABLE compute_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES compute_orders(order_id),
    unit_id UUID REFERENCES compute_units(unit_id),
    provider_id UUID REFERENCES providers(provider_id),
    
    -- Session parameters
    agreed_price_per_hour_usdc DECIMAL(10,6) NOT NULL,
    estimated_end_time TIMESTAMPTZ NOT NULL,
    session_status session_status_type DEFAULT 'provisioning',
    
    -- Access details (encrypted with requester's public key)
    connection_details_encrypted BYTEA, -- SSH, API endpoint, etc
    session_token TEXT, -- For API access
    
    -- Resource allocation
    allocated_gpu_devices INTEGER[],
    allocated_cpu_cores INTEGER[],
    allocated_ram_gb INTEGER,
    allocated_storage_path TEXT,
    
    -- Lifecycle timestamps
    provisioning_started_at TIMESTAMPTZ DEFAULT NOW(),
    session_started_at TIMESTAMPTZ,
    session_ended_at TIMESTAMPTZ,
    
    -- Cost tracking
    base_cost_usdc DECIMAL(18,6) DEFAULT 0,
    usage_premiums_usdc DECIMAL(18,6) DEFAULT 0, -- High utilization bonuses
    total_cost_usdc DECIMAL(18,6) DEFAULT 0,
    
    INDEX idx_sessions_active (session_status, session_started_at DESC),
    INDEX idx_sessions_provider_active (provider_id, session_status),
    INDEX idx_sessions_billing (session_ended_at, total_cost_usdc) WHERE session_ended_at IS NOT NULL
);

-- Create TimescaleDB hypertable for metrics
SELECT create_hypertable('session_metrics', 'timestamp');

-- Real-time metrics (TimescaleDB hypertable)
CREATE TABLE session_metrics (
    session_id UUID REFERENCES compute_sessions(session_id),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- GPU metrics
    gpu_utilization_percent INTEGER CHECK (gpu_utilization_percent BETWEEN 0 AND 100),
    gpu_memory_used_mb INTEGER,
    gpu_temperature_celsius INTEGER,
    gpu_power_draw_watts INTEGER,
    gpu_clock_speed_mhz INTEGER,
    
    -- System metrics  
    cpu_utilization_percent INTEGER CHECK (cpu_utilization_percent BETWEEN 0 AND 100),
    ram_used_gb DECIMAL(8,3),
    disk_io_read_mbps DECIMAL(8,2),
    disk_io_write_mbps DECIMAL(8,2),
    network_rx_mbps DECIMAL(8,2),
    network_tx_mbps DECIMAL(8,2),
    
    -- Performance metrics
    training_steps_per_second DECIMAL(10,2),
    inference_tokens_per_second DECIMAL(10,2),
    batch_size_processed INTEGER,
    memory_peak_mb INTEGER,
    
    PRIMARY KEY (session_id, timestamp)
);

-- Optimize for recent metrics queries
CREATE INDEX idx_session_metrics_recent 
ON session_metrics (session_id, timestamp DESC);

-- Aggregate hourly stats for billing
CREATE MATERIALIZED VIEW session_hourly_stats AS
SELECT 
    session_id,
    date_trunc('hour', timestamp) AS hour,
    AVG(gpu_utilization_percent) AS avg_gpu_util,
    MAX(gpu_utilization_percent) AS peak_gpu_util,
    AVG(gpu_memory_used_mb) AS avg_gpu_memory,
    AVG(cpu_utilization_percent) AS avg_cpu_util,
    SUM(CASE WHEN gpu_utilization_percent > 90 THEN 1 ELSE 0 END) AS high_util_minutes,
    COUNT(*) AS total_samples
FROM session_metrics
GROUP BY session_id, date_trunc('hour', timestamp);

CREATE UNIQUE INDEX idx_session_hourly_stats_pk 
ON session_hourly_stats (session_id, hour);
