-- migrate:up
ALTER TABLE routers ADD COLUMN IF NOT EXISTS router_type VARCHAR(20) NOT NULL DEFAULT 'mikrotik';

-- migrate:down
ALTER TABLE routers DROP COLUMN IF EXISTS router_type;
