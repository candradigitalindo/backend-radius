-- migrate:up
CREATE TABLE routers (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    identity        VARCHAR(100),
    vpn_ip          VARCHAR(45) NOT NULL,
    vpn_public_key  VARCHAR(100),
    radius_secret   VARCHAR(255) NOT NULL,
    coa_port        INTEGER DEFAULT 3799,
    heartbeat_token VARCHAR(255),
    is_online       BOOLEAN DEFAULT FALSE,
    last_seen_at    TIMESTAMPTZ,
    router_os_ver   VARCHAR(20),
    board_name      VARCHAR(50),
    uptime          VARCHAR(50),
    cpu_load        SMALLINT,
    free_memory     BIGINT,
    total_memory    BIGINT,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_routers_tenant ON routers(tenant_id);
CREATE INDEX idx_routers_vpn_ip ON routers(vpn_ip);

-- migrate:down
DROP TABLE IF EXISTS routers;
