-- migrate:up
CREATE TABLE tenants (
    id              CHAR(26) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(50) UNIQUE NOT NULL,
    email           VARCHAR(100) NOT NULL,
    phone           VARCHAR(20),
    address         TEXT,
    logo_url        VARCHAR(255),
    timezone        VARCHAR(50) DEFAULT 'Asia/Jakarta',
    currency        VARCHAR(5) DEFAULT 'IDR',
    billing_cycle   SMALLINT DEFAULT 1,
    due_day         SMALLINT DEFAULT 20,
    isolir_day      SMALLINT DEFAULT 21,
    grace_period    SMALLINT DEFAULT 3,
    plan            VARCHAR(20) DEFAULT 'trial',
    plan_expires_at TIMESTAMPTZ,
    max_customers   INTEGER DEFAULT 100,
    wa_api_key      VARCHAR(255),
    wa_sender       VARCHAR(20),
    pg_provider     VARCHAR(20),
    pg_api_key      VARCHAR(255),
    pg_secret_key   VARCHAR(255),
    pg_merchant_id  VARCHAR(100),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);

-- migrate:down
DROP TABLE IF EXISTS tenants;
