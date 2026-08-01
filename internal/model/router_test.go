package model

import "testing"

func TestCoAAddressPrecedence(t *testing.T) {
	// WireGuard menang atas semuanya
	r := Router{VPNIP: "10.77.0.4", VPNPublicKey: "abc", LegacyVPNIP: "10.78.0.2", NASIP: "103.18.35.113"}
	if got := r.CoAAddress(); got != "10.77.0.4" {
		t.Fatalf("WireGuard harus menang, got %s", got)
	}

	// Tanpa WG key: legacy VPN menang atas nas_ip
	r = Router{VPNIP: "10.77.0.2", LegacyVPNIP: "10.78.0.2", NASIP: "103.18.35.113"}
	if got := r.CoAAddress(); got != "10.78.0.2" {
		t.Fatalf("legacy VPN harus menang atas nas_ip, got %s", got)
	}

	// Direct: jatuh ke nas_ip
	r = Router{VPNIP: "10.77.0.2", NASIP: "203.0.113.7"}
	if got := r.CoAAddress(); got != "203.0.113.7" {
		t.Fatalf("direct harus pakai nas_ip, got %s", got)
	}

	withLegacy := Router{LegacyVPNIP: "10.78.0.2"}
	if !withLegacy.UsesLegacyVPN() {
		t.Fatal("UsesLegacyVPN harus true bila legacy_vpn_ip terisi")
	}
	empty := Router{}
	if empty.UsesLegacyVPN() {
		t.Fatal("UsesLegacyVPN harus false bila kosong")
	}
}
