-- migrate:up
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS unique_code SMALLINT NOT NULL DEFAULT 0;

-- migrate:down
ALTER TABLE subscription_orders DROP COLUMN IF EXISTS unique_code;
