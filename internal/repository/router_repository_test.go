package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Use a disposable PostgreSQL instance via ROUTER_TEST_DATABASE_URL. A temporary
// table on a single connection keeps every operation isolated from real data.
func TestDirectRouterPersistenceAndSNMPCandidates(t *testing.T) {
	dsn := os.Getenv("ROUTER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ROUTER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(ctx, `CREATE TEMP TABLE routers (
		id text PRIMARY KEY, tenant_id text, name text, router_type text,
		identity text, vpn_ip text, vpn_public_key text, legacy_vpn_ip text,
		radius_secret text, coa_port int, heartbeat_token text, snmp_community text,
		nas_ip text, is_active boolean, is_online boolean DEFAULT false,
		last_seen_at timestamptz, created_at timestamptz, updated_at timestamptz
	)`)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRouterRepository(db)
	router := &model.Router{
		TenantID: "tenant", Name: "Cisco", RouterType: "cisco",
		VPNIP: "10.77.0.2", NASIP: "203.0.113.7", RADIUSSecret: "test-secret",
		CoAPort: 3799, SNMPCommunity: "monitor", IsActive: true,
	}
	if err := repo.Create(ctx, router); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindByVPNIP(ctx, router.NASIP)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != router.ID || found.RADIUSSecret != router.RADIUSSecret {
		t.Fatalf("created public NAS cannot be resolved for RADIUS: %#v", found)
	}

	for _, tc := range []struct {
		name, tenant, vpnIP, key, legacyIP, nasIP, community string
		active, fresh                                        bool
	}{
		{name: "wireguard", tenant: "tenant", vpnIP: "10.77.0.3", key: "public-key", community: "monitor", active: true},
		{name: "legacy", tenant: "tenant", legacyIP: "10.78.0.2", community: "monitor", active: true},
		{name: "unconnected", tenant: "tenant", vpnIP: "10.77.0.4", community: "monitor", active: true},
		{name: "no-community", tenant: "tenant", nasIP: "203.0.113.8", active: true},
		{name: "inactive", tenant: "tenant", nasIP: "203.0.113.9", community: "monitor"},
		{name: "fresh", tenant: "tenant", nasIP: "203.0.113.10", community: "monitor", active: true, fresh: true},
		{name: "other-tenant", tenant: "other", nasIP: "203.0.113.11", community: "monitor", active: true},
	} {
		var lastSeen *time.Time
		if tc.fresh {
			now := time.Now()
			lastSeen = &now
		}
		_, err := db.Exec(ctx, `INSERT INTO routers
			(id, tenant_id, vpn_ip, vpn_public_key, legacy_vpn_ip, nas_ip, snmp_community, is_active, last_seen_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			tc.name, tc.tenant, tc.vpnIP, tc.key, tc.legacyIP, tc.nasIP, tc.community, tc.active, lastSeen)
		if err != nil {
			t.Fatal(err)
		}
	}
	stale, err := repo.ListStaleForSNMP(ctx, "tenant", time.Now().Add(-10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{router.ID: router.NASIP, "wireguard": "10.77.0.3", "legacy": "10.78.0.2"}
	if len(stale) != len(want) {
		t.Fatalf("got %d SNMP candidates, want %d: %#v", len(stale), len(want), stale)
	}
	for _, rt := range stale {
		addr, ok := want[rt.ID]
		if !ok || rt.CoAAddress() != addr || rt.SNMPCommunity != "monitor" {
			t.Errorf("unexpected SNMP candidate or destination: %#v", rt)
		}
	}
}
