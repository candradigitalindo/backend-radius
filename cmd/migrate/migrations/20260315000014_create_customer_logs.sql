-- migrate:up
CREATE TABLE customer_logs (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    action          VARCHAR(30) NOT NULL,
    description     TEXT,
    metadata        JSONB,
    performed_by    CHAR(26) REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_customer_logs_customer ON customer_logs(customer_id);
CREATE INDEX idx_customer_logs_action ON customer_logs(tenant_id, action);

-- migrate:down
DROP TABLE IF EXISTS customer_logs;
