-- migrate:up
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS pg_sandbox BOOLEAN NOT NULL DEFAULT true;

-- migrate:down
ALTER TABLE tenants DROP COLUMN IF EXISTS pg_sandbox;
