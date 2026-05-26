-- migrate:up
CREATE TABLE olts (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    hostname        VARCHAR(100),
    ip_address      VARCHAR(45) NOT NULL,
    vendor          VARCHAR(50),
    model           VARCHAR(50),
    serial_number   VARCHAR(50),
    total_pon_ports SMALLINT NOT NULL DEFAULT 16,
    latitude        DECIMAL(10, 8),
    longitude       DECIMAL(11, 8),
    snmp_community  VARCHAR(50) DEFAULT 'public',
    status          VARCHAR(20) DEFAULT 'active',
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_olts_tenant ON olts(tenant_id);

CREATE TABLE pon_ports (
    id              CHAR(26) PRIMARY KEY,
    olt_id          CHAR(26) NOT NULL REFERENCES olts(id) ON DELETE CASCADE,
    port_number     SMALLINT NOT NULL,
    description     VARCHAR(100),
    status          VARCHAR(20) DEFAULT 'active',
    UNIQUE(olt_id, port_number)
);

CREATE INDEX idx_pon_ports_olt ON pon_ports(olt_id);

CREATE TABLE splitters (
    id                  CHAR(26) PRIMARY KEY,
    tenant_id           CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pon_port_id         CHAR(26) REFERENCES pon_ports(id),
    parent_splitter_id  CHAR(26) REFERENCES splitters(id),
    name                VARCHAR(100) NOT NULL,
    splitter_type       VARCHAR(10) NOT NULL,
    latitude            DECIMAL(10, 8),
    longitude           DECIMAL(11, 8),
    notes               TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_splitters_tenant ON splitters(tenant_id);
CREATE INDEX idx_splitters_pon_port ON splitters(pon_port_id);
CREATE INDEX idx_splitters_parent ON splitters(parent_splitter_id);

ALTER TABLE odps ADD COLUMN IF NOT EXISTS olt_id CHAR(26) REFERENCES olts(id);
ALTER TABLE odps ADD COLUMN IF NOT EXISTS splitter_id CHAR(26) REFERENCES splitters(id);

CREATE TABLE onts (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) REFERENCES customers(id),
    odp_port_id     CHAR(26) REFERENCES odp_ports(id),
    serial_number   VARCHAR(50) NOT NULL,
    model           VARCHAR(50),
    vendor          VARCHAR(50),
    mac_address     VARCHAR(17),
    ip_address      VARCHAR(45),
    rx_power        DECIMAL(6, 2),
    tx_power        DECIMAL(6, 2),
    status          VARCHAR(20) DEFAULT 'active',
    notes           TEXT,
    last_online_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, serial_number)
);

CREATE INDEX idx_onts_tenant ON onts(tenant_id);
CREATE INDEX idx_onts_customer ON onts(customer_id);
CREATE INDEX idx_onts_odp_port ON onts(odp_port_id);

-- migrate:down
DROP TABLE IF EXISTS onts;
ALTER TABLE odps DROP COLUMN IF EXISTS splitter_id;
ALTER TABLE odps DROP COLUMN IF EXISTS olt_id;
DROP TABLE IF EXISTS splitters;
DROP TABLE IF EXISTS pon_ports;
DROP TABLE IF EXISTS olts;
