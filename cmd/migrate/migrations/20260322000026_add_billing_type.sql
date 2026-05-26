-- migrate:up
ALTER TABLE customers ADD COLUMN IF NOT EXISTS billing_type VARCHAR(10) DEFAULT 'fixed';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS billing_deadline SMALLINT DEFAULT 20;

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS billing_deadline;
ALTER TABLE customers DROP COLUMN IF EXISTS billing_type;
