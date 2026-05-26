-- migrate:up
CREATE TABLE packages (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    bandwidth_up    INTEGER NOT NULL,
    bandwidth_down  INTEGER NOT NULL,
    price           BIGINT NOT NULL,
    burst_limit     VARCHAR(50),
    address_list    VARCHAR(50),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_packages_tenant ON packages(tenant_id);

-- migrate:down
DROP TABLE IF EXISTS packages;
