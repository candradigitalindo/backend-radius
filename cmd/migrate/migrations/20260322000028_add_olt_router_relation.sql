-- migrate:up
ALTER TABLE olts ADD COLUMN router_id CHAR(26) REFERENCES routers(id) ON DELETE SET NULL;
ALTER TABLE olts DROP COLUMN IF EXISTS hostname;
CREATE INDEX idx_olts_router ON olts(router_id);

-- migrate:down
DROP INDEX IF EXISTS idx_olts_router;
ALTER TABLE olts ADD COLUMN hostname VARCHAR(100);
ALTER TABLE olts DROP COLUMN IF EXISTS router_id;
