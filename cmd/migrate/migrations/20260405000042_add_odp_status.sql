-- migrate:up
ALTER TABLE odps ADD COLUMN IF NOT EXISTS status VARCHAR(10) DEFAULT 'draft';

-- migrate:down
ALTER TABLE odps DROP COLUMN IF EXISTS status;
