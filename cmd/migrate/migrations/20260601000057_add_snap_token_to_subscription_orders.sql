-- +goose Up
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS snap_token TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE subscription_orders DROP COLUMN IF EXISTS snap_token;
