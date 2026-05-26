-- migrate:up
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_wifi_ssid VARCHAR(255) DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_wifi_password TEXT DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_wifi_security VARCHAR(32) DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_wifi_source VARCHAR(32) DEFAULT NULL;
ALTER TABLE onts ADD COLUMN IF NOT EXISTS last_known_wifi_updated_at TIMESTAMPTZ DEFAULT NULL;

-- migrate:down
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_wifi_updated_at;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_wifi_source;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_wifi_security;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_wifi_password;
ALTER TABLE onts DROP COLUMN IF EXISTS last_known_wifi_ssid;