-- migrate:up
ALTER TABLE odps ADD COLUMN splitter_ratio VARCHAR(50);

-- migrate:down
ALTER TABLE odps DROP COLUMN IF EXISTS splitter_ratio;
