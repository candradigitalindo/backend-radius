package main

import "testing"

// Sanity check: every embedded migration file must parse without error and
// produce a non-empty up section.
func TestAllEmbeddedMigrationsParse(t *testing.T) {
	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	if len(migs) == 0 {
		t.Fatal("no migrations loaded")
	}
	for _, m := range migs {
		if m.UpSQL == "" {
			t.Errorf("%s: empty up SQL", m.Filename)
		}
	}
	t.Logf("parsed %d migrations OK", len(migs))
}

// The goose format must be rejected.
func TestGooseFormatRejected(t *testing.T) {
	_, _, err := parseMigration("-- +goose Up\nALTER TABLE x ADD COLUMN y INT;\n-- +goose Down\nALTER TABLE x DROP COLUMN y;")
	if err == nil {
		t.Fatal("expected goose format to be rejected, got nil error")
	}
	t.Logf("goose rejected with: %v", err)
}

// Legacy plain SQL (no markers) must be treated as up-only.
func TestLegacyPlainSQLUpOnly(t *testing.T) {
	up, down, err := parseMigration("ALTER TABLE x ADD COLUMN y INT;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if up == "" {
		t.Fatal("expected non-empty up")
	}
	if down != "" {
		t.Fatalf("expected empty down, got %q", down)
	}
}
