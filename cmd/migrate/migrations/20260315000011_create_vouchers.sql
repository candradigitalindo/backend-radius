-- migrate:up
CREATE TABLE voucher_products (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    duration        INTEGER NOT NULL,
    bandwidth_up    INTEGER,
    bandwidth_down  INTEGER,
    price           BIGINT NOT NULL,
    profile_name    VARCHAR(50),
    router_id       CHAR(26) REFERENCES routers(id),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_voucher_products_tenant ON voucher_products(tenant_id);

CREATE TABLE vouchers (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id      CHAR(26) NOT NULL REFERENCES voucher_products(id),
    username        VARCHAR(50) NOT NULL,
    password        VARCHAR(50) NOT NULL,
    status          VARCHAR(10) DEFAULT 'available',
    buyer_phone     VARCHAR(20),
    sold_at         TIMESTAMPTZ,
    activated_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_vouchers_tenant ON vouchers(tenant_id);
CREATE INDEX idx_vouchers_product ON vouchers(product_id);
CREATE INDEX idx_vouchers_status ON vouchers(tenant_id, status);

CREATE TABLE voucher_payments (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    voucher_id      CHAR(26) NOT NULL REFERENCES vouchers(id),
    buyer_name      VARCHAR(100),
    buyer_phone     VARCHAR(20) NOT NULL,
    amount          BIGINT NOT NULL,
    gateway         VARCHAR(20) NOT NULL,
    gateway_trx_id  VARCHAR(100),
    status          VARCHAR(20) DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_voucher_payments_tenant ON voucher_payments(tenant_id);

-- migrate:down
DROP TABLE IF EXISTS voucher_payments;
DROP TABLE IF EXISTS vouchers;
DROP TABLE IF EXISTS voucher_products;
