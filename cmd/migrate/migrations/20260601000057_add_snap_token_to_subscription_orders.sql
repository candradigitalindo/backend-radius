-- migrate:up
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS snap_token TEXT NOT NULL DEFAULT '';

-- migrate:down
ALTER TABLE subscription_orders DROP COLUMN IF EXISTS snap_token;
