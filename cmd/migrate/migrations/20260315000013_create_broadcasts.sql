-- migrate:up
CREATE TABLE broadcasts (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type            VARCHAR(20) NOT NULL,
    title           VARCHAR(100) NOT NULL,
    message         TEXT NOT NULL,
    target          VARCHAR(20) DEFAULT 'all',
    total_sent      INTEGER DEFAULT 0,
    total_success   INTEGER DEFAULT 0,
    total_failed    INTEGER DEFAULT 0,
    sent_by         CHAR(26) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_broadcasts_tenant ON broadcasts(tenant_id);

-- migrate:down
DROP TABLE IF EXISTS broadcasts;
