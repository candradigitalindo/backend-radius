-- migrate:up
CREATE TABLE payments (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id      CHAR(26) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    amount          BIGINT NOT NULL,
    payment_method  VARCHAR(30) NOT NULL,
    gateway         VARCHAR(20),
    gateway_trx_id  VARCHAR(100),
    gateway_status  VARCHAR(30),
    gateway_response JSONB,
    status          VARCHAR(20) DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    expired_at      TIMESTAMPTZ,
    collected_by    CHAR(26) REFERENCES users(id),
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payments_tenant ON payments(tenant_id);
CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_gateway_trx ON payments(gateway_trx_id);

-- migrate:down
DROP TABLE IF EXISTS payments;
