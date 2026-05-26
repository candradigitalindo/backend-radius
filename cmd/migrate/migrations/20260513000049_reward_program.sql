-- Reward program tables and customer referral_code field
-- Migration: 20260513000049

-- Add referral_code to customers (unique per tenant)
ALTER TABLE customers ADD COLUMN IF NOT EXISTS referral_code VARCHAR(16);
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_referral_code ON customers (tenant_id, referral_code) WHERE referral_code IS NOT NULL;

-- Back-fill referral_code for existing customers (random 8-char uppercase)
UPDATE customers SET referral_code = UPPER(substring(replace(gen_random_uuid()::text, '-', ''), 1, 8)) WHERE referral_code IS NULL OR referral_code = '';

-- Rewards master table
CREATE TABLE IF NOT EXISTS rewards (
    id              VARCHAR(26)  PRIMARY KEY,
    tenant_id       VARCHAR(26)  NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    type            VARCHAR(32)  NOT NULL CHECK (type IN ('referral','loyalty','promo')),
    value           BIGINT       NOT NULL DEFAULT 0,
    value_type      VARCHAR(16)  NOT NULL CHECK (value_type IN ('fixed','percentage')),
    min_invoices    INT          NOT NULL DEFAULT 0,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    start_date      TIMESTAMPTZ,
    end_date        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rewards_tenant ON rewards (tenant_id);
CREATE INDEX IF NOT EXISTS idx_rewards_tenant_type ON rewards (tenant_id, type, is_active);

-- Referrals table
CREATE TABLE IF NOT EXISTS referrals (
    id              VARCHAR(26)  PRIMARY KEY,
    tenant_id       VARCHAR(26)  NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    referrer_id     VARCHAR(26)  NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    referred_id     VARCHAR(26)  NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    reward_id       VARCHAR(26)  NOT NULL REFERENCES rewards(id) ON DELETE CASCADE,
    referral_code   VARCHAR(16)  NOT NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','qualified','rewarded','expired')),
    reward_amount   BIGINT       NOT NULL DEFAULT 0,
    qualified_at    TIMESTAMPTZ,
    rewarded_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_referrals_tenant ON referrals (tenant_id);
CREATE INDEX IF NOT EXISTS idx_referrals_referred ON referrals (tenant_id, referred_id, status);
CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals (tenant_id, referrer_id);

-- Reward claims table
CREATE TABLE IF NOT EXISTS reward_claims (
    id          VARCHAR(26)  PRIMARY KEY,
    tenant_id   VARCHAR(26)  NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id VARCHAR(26)  NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    reward_id   VARCHAR(26)  NOT NULL REFERENCES rewards(id) ON DELETE CASCADE,
    referral_id VARCHAR(26)  REFERENCES referrals(id) ON DELETE SET NULL,
    amount      BIGINT       NOT NULL DEFAULT 0,
    type        VARCHAR(32)  NOT NULL CHECK (type IN ('invoice_discount','balance_credit')),
    status      VARCHAR(16)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','applied','expired')),
    applied_at  TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reward_claims_tenant ON reward_claims (tenant_id);
CREATE INDEX IF NOT EXISTS idx_reward_claims_customer ON reward_claims (tenant_id, customer_id, status);
CREATE INDEX IF NOT EXISTS idx_reward_claims_expires ON reward_claims (status, expires_at) WHERE status = 'pending';
