-- migrate:up
ALTER TABLE customers ADD COLUMN IF NOT EXISTS nik VARCHAR(20);

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS nik;
