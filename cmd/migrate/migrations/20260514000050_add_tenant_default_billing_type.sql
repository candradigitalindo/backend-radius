-- migrate:up
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_billing_type VARCHAR(10) NOT NULL DEFAULT 'fixed';

-- migrate:down
ALTER TABLE tenants DROP COLUMN IF EXISTS default_billing_type;
