-- migrate:up
CREATE TABLE odps (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    address         TEXT,
    latitude        DECIMAL(10, 8) NOT NULL,
    longitude       DECIMAL(11, 8) NOT NULL,
    total_ports     SMALLINT NOT NULL DEFAULT 8,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_odps_tenant ON odps(tenant_id);

CREATE TABLE odp_ports (
    id              CHAR(26) PRIMARY KEY,
    odp_id          CHAR(26) NOT NULL REFERENCES odps(id) ON DELETE CASCADE,
    port_number     SMALLINT NOT NULL,
    customer_id     CHAR(26),
    status          VARCHAR(10) DEFAULT 'available',
    notes           VARCHAR(100),
    UNIQUE(odp_id, port_number)
);

CREATE INDEX idx_odp_ports_odp ON odp_ports(odp_id);

-- migrate:down
DROP TABLE IF EXISTS odp_ports;
DROP TABLE IF EXISTS odps;
