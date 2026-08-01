package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

func TestAllocateLegacyVPNIP(t *testing.T) {
	// Alokasi pertama: .1 (gateway) dilewati, dapat .2
	ip, err := allocateLegacyVPNIP("10.78.0.0/24", "10.78.0.1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ip != "10.78.0.2" {
		t.Fatalf("want 10.78.0.2, got %s", ip)
	}

	// IP terpakai dilewati, lubang di tengah diisi
	ip, err = allocateLegacyVPNIP("10.78.0.0/24", "10.78.0.1", []string{"10.78.0.2", "10.78.0.4"})
	if err != nil {
		t.Fatal(err)
	}
	if ip != "10.78.0.3" {
		t.Fatalf("want 10.78.0.3, got %s", ip)
	}

	// Broadcast & network tidak pernah dialokasikan
	var used []string
	for i := 2; i <= 254; i++ {
		used = append(used, "10.78.0."+strconv.Itoa(i))
	}
	if _, err = allocateLegacyVPNIP("10.78.0.0/24", "10.78.0.1", used); err == nil {
		t.Fatal("subnet penuh seharusnya error, bukan mengalokasikan .0/.255")
	}

	// Subnet tidak valid
	if _, err = allocateLegacyVPNIP("bukan-cidr", "10.78.0.1", nil); err == nil {
		t.Fatal("subnet tidak valid seharusnya error")
	}
}

func TestBuildChapSecrets(t *testing.T) {
	got := buildChapSecrets([]repository.LegacyVPNAccount{
		{RouterID: "01ABC", Password: "rahasia1", IP: "10.78.0.2"},
		{RouterID: "01DEF", Password: "rahasia2", IP: "10.78.0.3"},
	})
	if !strings.Contains(got, "01ABC * rahasia1 10.78.0.2\n") {
		t.Fatalf("baris pertama salah:\n%s", got)
	}
	if !strings.Contains(got, "01DEF * rahasia2 10.78.0.3\n") {
		t.Fatalf("baris kedua salah:\n%s", got)
	}
	if !strings.HasPrefix(got, "#") {
		t.Fatalf("harus diawali komentar:\n%s", got)
	}
}
