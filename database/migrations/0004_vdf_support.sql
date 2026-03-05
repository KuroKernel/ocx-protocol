-- Migration 0004: VDF (Verifiable Delay Function) support
-- Adds VDF temporal proof columns to receipt tables and a modulus registry.

-- Add VDF columns to receipts
ALTER TABLE ocx_receipts_v1_1 ADD COLUMN IF NOT EXISTS vdf_output BYTEA;
ALTER TABLE ocx_receipts_v1_1 ADD COLUMN IF NOT EXISTS vdf_proof BYTEA;
ALTER TABLE ocx_receipts_v1_1 ADD COLUMN IF NOT EXISTS vdf_iterations BIGINT;
ALTER TABLE ocx_receipts_v1_1 ADD COLUMN IF NOT EXISTS vdf_modulus_id VARCHAR(50);
ALTER TABLE ocx_receipts_v1_1 ADD COLUMN IF NOT EXISTS vdf_verified BOOLEAN;

-- VDF modulus registry (for key rotation)
CREATE TABLE IF NOT EXISTS ocx_vdf_moduli (
    modulus_id VARCHAR(50) PRIMARY KEY,
    modulus_hex TEXT NOT NULL,
    bit_length INTEGER NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    retired_at TIMESTAMPTZ
);

-- Seed initial RSA-2048 modulus
INSERT INTO ocx_vdf_moduli (modulus_id, modulus_hex, bit_length)
VALUES ('ocx-vdf-v1', 'RSA-2048-FACTORING-CHALLENGE', 2048)
ON CONFLICT DO NOTHING;
