-- migrate:up
-- VPN legacy (L2TP/SSTP via accel-ppp) untuk router yang tidak bisa WireGuard
-- (RouterOS 6) dan tidak punya IP publik di WAN. Username tunnel = router.id,
-- password di vpn_password, IP statis tunnel di legacy_vpn_ip (subnet 10.78.0.0/24).
ALTER TABLE routers ADD COLUMN IF NOT EXISTS vpn_password TEXT;
ALTER TABLE routers ADD COLUMN IF NOT EXISTS legacy_vpn_ip TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_routers_legacy_vpn_ip ON routers(legacy_vpn_ip) WHERE legacy_vpn_ip IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_routers_legacy_vpn_ip;
ALTER TABLE routers DROP COLUMN IF EXISTS legacy_vpn_ip;
ALTER TABLE routers DROP COLUMN IF EXISTS vpn_password;
