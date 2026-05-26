-- migrate:up
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_connected_hosts TEXT DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_connected_host_count INTEGER DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_connected_hosts_source VARCHAR(32) DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_connected_hosts_updated_at TIMESTAMPTZ DEFAULT NULL;

-- migrate:down
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_connected_hosts_updated_at;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_connected_hosts_source;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_connected_host_count;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_connected_hosts;