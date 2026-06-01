-- migrate:up
-- nas_ip menyimpan WAN IP router untuk mode Direct (tanpa WireGuard).
-- Diisi otomatis dari IP pengirim saat heartbeat diterima.
ALTER TABLE routers ADD COLUMN IF NOT EXISTS nas_ip VARCHAR(45) DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_routers_nas_ip ON routers(nas_ip) WHERE nas_ip != '';

-- migrate:down
DROP INDEX IF EXISTS idx_routers_nas_ip;
ALTER TABLE routers DROP COLUMN IF EXISTS nas_ip;
