-- migrate:up
ALTER TABLE customers ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS password_hash;
