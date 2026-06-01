-- migrate:up
ALTER TABLE routers ADD COLUMN IF NOT EXISTS snmp_community VARCHAR(50) DEFAULT 'public';

-- migrate:down
ALTER TABLE routers DROP COLUMN IF EXISTS snmp_community;
