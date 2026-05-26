-- migrate:up
ALTER TABLE customers ADD COLUMN IF NOT EXISTS billing_type VARCHAR(10) DEFAULT 'fixed';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS billing_deadline SMALLINT DEFAULT 20;
ALTER TABLE odps ADD COLUMN IF NOT EXISTS splitter_ratio VARCHAR(50);

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS billing_type;
ALTER TABLE customers DROP COLUMN IF EXISTS billing_deadline;
ALTER TABLE odps DROP COLUMN IF EXISTS splitter_ratio;
