-- migrate:up

-- Subscription plans catalog
CREATE TABLE subscription_plans (
    id              CHAR(26) PRIMARY KEY,
    name            VARCHAR(50) NOT NULL,
    slug            VARCHAR(50) UNIQUE NOT NULL,
    description     TEXT,
    price           BIGINT NOT NULL DEFAULT 0,
    duration_months INTEGER NOT NULL DEFAULT 1,
    max_customers   INTEGER NOT NULL DEFAULT 50,
    max_routers     INTEGER NOT NULL DEFAULT 99999,
    features        JSONB DEFAULT '[]',
    is_popular      BOOLEAN DEFAULT FALSE,
    is_active       BOOLEAN DEFAULT TRUE,
    sort_order      INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Subscription orders/payments history
CREATE TABLE subscription_orders (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id),
    plan_id         CHAR(26) NOT NULL REFERENCES subscription_plans(id),
    plan_name       VARCHAR(50) NOT NULL,
    amount          BIGINT NOT NULL,
    duration_months INTEGER NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_method  VARCHAR(50),
    payment_url     VARCHAR(500),
    payment_ref     VARCHAR(100),
    paid_at         TIMESTAMPTZ,
    starts_at       TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscription_orders_tenant ON subscription_orders(tenant_id);
CREATE INDEX idx_subscription_orders_status ON subscription_orders(status);

-- Seed subscription plans for RT/RW Net and ISP
INSERT INTO subscription_plans (id, name, slug, description, price, duration_months, max_customers, max_routers, features, is_popular, sort_order) VALUES
('PLANFREE000000000000000000', 'Gratis', 'gratis', 'Cocok untuk coba semua fitur tanpa batas waktu. Upgrade kapan saja.', 0, 0, 20, 99999,
 '["Semua Fitur Terbuka", "Maks. 20 Pelanggan Aktif"]'::jsonb,
 FALSE, 1),

('PLANSTARTER000000000000000', 'Starter', 'starter', 'Untuk RT/RW Net yang baru memulai dengan fitur lengkap.', 99000, 1, 100, 99999,
 '["Semua Fitur Terbuka", "Maks. 100 Pelanggan Aktif"]'::jsonb,
 FALSE, 2),

('PLANGROWTH0000000000000000', 'Growth', 'growth', 'Pilihan terbaik untuk RT/RW Net yang sedang berkembang.', 199000, 1, 300, 99999,
 '["Semua Fitur Terbuka", "Maks. 300 Pelanggan Aktif"]'::jsonb,
 TRUE, 3),

('PLANPRO0000000000000000000', 'Professional', 'professional', 'Untuk ISP skala menengah dengan banyak pelanggan.', 399000, 1, 1000, 99999,
 '["Semua Fitur Terbuka", "Maks. 1.000 Pelanggan Aktif"]'::jsonb,
 FALSE, 4),

('PLANENTERPRISE000000000000', 'Enterprise', 'enterprise', 'Untuk ISP besar tanpa batasan pelanggan.', 699000, 1, 99999, 99999,
 '["Semua Fitur Terbuka", "Pelanggan Unlimited"]'::jsonb,
 FALSE, 5);

-- migrate:down
DROP TABLE IF EXISTS subscription_orders;
DROP TABLE IF EXISTS subscription_plans;
