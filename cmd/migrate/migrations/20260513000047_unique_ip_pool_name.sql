-- migrate:up
CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_pools_unique_name ON ip_pools(tenant_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_pools_unique_network ON ip_pools(tenant_id, network);

-- migrate:down
DROP INDEX IF EXISTS idx_ip_pools_unique_name;
DROP INDEX IF EXISTS idx_ip_pools_unique_network;
