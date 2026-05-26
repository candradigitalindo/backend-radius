-- migrate:up
CREATE TABLE IF NOT EXISTS rewards (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL DEFAULT 'referral',
    value BIGINT NOT NULL DEFAULT 0,
    value_type VARCHAR(20) NOT NULL DEFAULT 'fixed',
    min_invoices INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rewards_tenant ON rewards(tenant_id);
CREATE INDEX idx_rewards_active ON rewards(tenant_id, is_active);

CREATE TABLE IF NOT EXISTS referrals (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    referrer_id CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    referred_id CHAR(26) REFERENCES customers(id) ON DELETE SET NULL,
    reward_id CHAR(26) REFERENCES rewards(id) ON DELETE SET NULL,
    referral_code VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reward_amount BIGINT NOT NULL DEFAULT 0,
    qualified_at TIMESTAMPTZ,
    rewarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_referrals_code ON referrals(tenant_id, referral_code);
CREATE INDEX idx_referrals_tenant ON referrals(tenant_id);
CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);
CREATE INDEX idx_referrals_referred ON referrals(referred_id);

CREATE TABLE IF NOT EXISTS reward_claims (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    reward_id CHAR(26) REFERENCES rewards(id) ON DELETE SET NULL,
    referral_id CHAR(26) REFERENCES referrals(id) ON DELETE SET NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    type VARCHAR(30) NOT NULL DEFAULT 'balance_credit',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    applied_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reward_claims_tenant ON reward_claims(tenant_id);
CREATE INDEX idx_reward_claims_customer ON reward_claims(customer_id);
CREATE INDEX idx_reward_claims_status ON reward_claims(tenant_id, status);

-- migrate:down
DROP TABLE IF EXISTS reward_claims;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS rewards;
