-- migrate:up
ALTER TABLE resellers ADD COLUMN IF NOT EXISTS company_name VARCHAR(150) DEFAULT '';

-- migrate:down
ALTER TABLE resellers DROP COLUMN IF EXISTS company_name;
