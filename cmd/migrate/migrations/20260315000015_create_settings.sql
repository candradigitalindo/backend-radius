-- migrate:up
CREATE TABLE settings (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key             VARCHAR(50) NOT NULL,
    value           TEXT,
    UNIQUE(tenant_id, key)
);

CREATE INDEX idx_settings_tenant ON settings(tenant_id);

-- migrate:down
DROP TABLE IF EXISTS settings;
