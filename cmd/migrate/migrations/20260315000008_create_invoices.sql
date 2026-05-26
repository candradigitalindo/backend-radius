-- migrate:up
CREATE TABLE invoices (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    invoice_number  VARCHAR(30) NOT NULL,
    period_month    SMALLINT NOT NULL,
    period_year     SMALLINT NOT NULL,
    package_price   BIGINT NOT NULL,
    discount        BIGINT DEFAULT 0,
    additional_fee  BIGINT DEFAULT 0,
    fee_description VARCHAR(100),
    total_amount    BIGINT NOT NULL,
    status          VARCHAR(20) DEFAULT 'unpaid',
    due_date        DATE NOT NULL,
    paid_at         TIMESTAMPTZ,
    paid_amount     BIGINT DEFAULT 0,
    payment_method  VARCHAR(30),
    notes           TEXT,
    auto_generated  BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, customer_id, period_month, period_year)
);

CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_customer ON invoices(customer_id);
CREATE INDEX idx_invoices_status ON invoices(tenant_id, status);
CREATE INDEX idx_invoices_period ON invoices(tenant_id, period_year, period_month);
CREATE INDEX idx_invoices_due ON invoices(tenant_id, due_date) WHERE status = 'unpaid';

-- migrate:down
DROP TABLE IF EXISTS invoices;
