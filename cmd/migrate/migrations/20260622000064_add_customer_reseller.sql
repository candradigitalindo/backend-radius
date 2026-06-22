-- migrate:up
ALTER TABLE customers ADD COLUMN IF NOT EXISTS reseller_id VARCHAR(40);
CREATE INDEX IF NOT EXISTS idx_customers_reseller ON customers (reseller_id) WHERE reseller_id IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_customers_reseller;
ALTER TABLE customers DROP COLUMN IF EXISTS reseller_id;
