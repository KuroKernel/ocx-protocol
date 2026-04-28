-- =====================================================================
-- OCX API — Postgres schema
-- Provider-agnostic. Stores customers, subscriptions, usage, and
-- webhook idempotency for whichever payment provider is wired up
-- (currently LemonSqueezy).
-- =====================================================================

-- Idempotency table for inbound webhook events. We MUST de-dupe events
-- because providers retry. Provider name is part of the row so two
-- providers cannot collide on a numeric id.
CREATE TABLE IF NOT EXISTS provider_events (
    event_id        TEXT PRIMARY KEY,
    provider        TEXT NOT NULL,           -- 'lemonsqueezy' | 'razorpay' | …
    event_type      TEXT NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS provider_events_provider_idx ON provider_events (provider);

-- One row per paying account.
-- API keys are stored as SHA-256 hashes; the plaintext is shown ONCE
-- to the customer (in the post-checkout email) and never again.
CREATE TABLE IF NOT EXISTS customers (
    id                       BIGSERIAL PRIMARY KEY,
    provider                 TEXT NOT NULL,
    provider_customer_id     TEXT UNIQUE NOT NULL,
    email                    TEXT NOT NULL,
    api_key_hash             TEXT UNIQUE NOT NULL,   -- SHA-256 hex of the raw key
    api_key_prefix           TEXT NOT NULL,          -- first 12 chars for display ("ocx_live_ab12...")
    tier                     TEXT NOT NULL,          -- 'starter' | 'growth' | 'scale' | 'enterprise'
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS customers_email_idx ON customers (email);
CREATE INDEX IF NOT EXISTS customers_tier_idx  ON customers (tier);

-- One row per active subscription. Multiple subs per customer is rare
-- but supported (e.g., during a tier change before the old sub
-- cancels). `portal_url` is provider-supplied (LemonSqueezy gives a
-- per-subscription customer-portal link; if your provider doesn't,
-- leave it NULL).
CREATE TABLE IF NOT EXISTS subscriptions (
    id                          BIGSERIAL PRIMARY KEY,
    provider                    TEXT NOT NULL,
    provider_subscription_id    TEXT UNIQUE NOT NULL,
    customer_id                 BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    status                      TEXT NOT NULL,      -- active | past_due | cancelled | …
    variant_id                  TEXT NOT NULL,      -- provider's price/variant identifier
    current_period_start        TIMESTAMPTZ NOT NULL,
    current_period_end          TIMESTAMPTZ NOT NULL,
    cancel_at_period_end        BOOLEAN NOT NULL DEFAULT FALSE,
    portal_url                  TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS subscriptions_customer_idx ON subscriptions (customer_id);
CREATE INDEX IF NOT EXISTS subscriptions_status_idx   ON subscriptions (status);

-- Per-customer monthly usage counter. Reset when current_period_start
-- changes on the corresponding subscription.
CREATE TABLE IF NOT EXISTS usage (
    id                       BIGSERIAL PRIMARY KEY,
    customer_id              BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    period_start             TIMESTAMPTZ NOT NULL,
    period_end               TIMESTAMPTZ NOT NULL,
    receipts_verified        BIGINT NOT NULL DEFAULT 0,
    bytes_processed          BIGINT NOT NULL DEFAULT 0,
    last_request_at          TIMESTAMPTZ,
    UNIQUE (customer_id, period_start)
);

CREATE INDEX IF NOT EXISTS usage_customer_period_idx ON usage (customer_id, period_start DESC);
