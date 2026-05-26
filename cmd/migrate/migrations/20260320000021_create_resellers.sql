-- migrate:up
CREATE TABLE resellers (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(150) NOT NULL,
    phone           VARCHAR(30) NOT NULL,
    address         TEXT DEFAULT '',
    commission_rate NUMERIC(5,2) DEFAULT 0,
    balance         BIGINT DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    notes           TEXT DEFAULT '',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_resellers_tenant ON resellers(tenant_id);
CREATE INDEX idx_resellers_status ON resellers(tenant_id, status);

CREATE TABLE reseller_commissions (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reseller_id     CHAR(26) NOT NULL REFERENCES resellers(id) ON DELETE CASCADE,
    invoice_id      CHAR(26) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    amount          BIGINT NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reseller_commissions_reseller ON reseller_commissions(tenant_id, reseller_id);
CREATE INDEX idx_reseller_commissions_status ON reseller_commissions(tenant_id, reseller_id, status);

-- migrate:down
DROP TABLE IF EXISTS reseller_commissions;
DROP TABLE IF EXISTS resellers;
