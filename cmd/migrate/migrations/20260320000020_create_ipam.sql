-- migrate:up
CREATE TABLE ip_pools (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    network         VARCHAR(50) NOT NULL,
    gateway         VARCHAR(50),
    dns_primary     VARCHAR(50),
    dns_secondary   VARCHAR(50),
    router_id       CHAR(26) REFERENCES routers(id) ON DELETE SET NULL,
    total_ips       INTEGER DEFAULT 0,
    used_ips        INTEGER DEFAULT 0,
    notes           TEXT DEFAULT '',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ip_pools_tenant ON ip_pools(tenant_id);

CREATE TABLE ip_addresses (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pool_id         CHAR(26) NOT NULL REFERENCES ip_pools(id) ON DELETE CASCADE,
    address         VARCHAR(50) NOT NULL,
    customer_id     CHAR(26) REFERENCES customers(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'available',
    notes           TEXT DEFAULT '',
    assigned_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ip_addresses_pool ON ip_addresses(tenant_id, pool_id);
CREATE INDEX idx_ip_addresses_status ON ip_addresses(tenant_id, pool_id, status);
CREATE INDEX idx_ip_addresses_customer ON ip_addresses(customer_id) WHERE customer_id IS NOT NULL;
CREATE UNIQUE INDEX idx_ip_addresses_unique ON ip_addresses(tenant_id, pool_id, address);

-- migrate:down
DROP TABLE IF EXISTS ip_addresses;
DROP TABLE IF EXISTS ip_pools;
