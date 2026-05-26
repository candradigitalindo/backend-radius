package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	Version  string
	Name     string
	UpSQL    string
	DownSQL  string
	Filename string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db := database.Connect(cfg)
	defer db.Close()

	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [up|down|status]")
		os.Exit(1)
	}

	ctx := context.Background()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	switch os.Args[1] {
	case "up":
		if err := migrateUp(ctx, db, migrations); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	case "down":
		if err := migrateDown(ctx, db, migrations); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
	case "status":
		if err := migrateStatus(ctx, db, migrations); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func ensureMigrationsTable(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(14) PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading migrations dir: %w", err)
	}

	var migrations []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrationFS, "migrations/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}

		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}

		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		raw := string(content)
		up, down := parseMigration(raw)

		migrations = append(migrations, migration{
			Version:  version,
			Name:     name,
			UpSQL:    up,
			DownSQL:  down,
			Filename: e.Name(),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func parseMigration(raw string) (up, down string) {
	parts := strings.SplitN(raw, "-- migrate:down", 2)
	up = strings.TrimPrefix(parts[0], "-- migrate:up")
	up = strings.TrimSpace(up)
	if len(parts) == 2 {
		down = strings.TrimSpace(parts[1])
	}
	return
}

func migrateUp(ctx context.Context, db *pgxpool.Pool, migrations []migration) error {
	applied, err := getApplied(ctx, db)
	if err != nil {
		return err
	}

	count := 0
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		fmt.Printf("Applying %s (%s)...\n", m.Version, m.Name)

		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("executing %s: %w", m.Filename, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			m.Version, m.Name,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("recording %s: %w", m.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", m.Version, err)
		}

		fmt.Printf("  Applied %s\n", m.Version)
		count++
	}

	if count == 0 {
		fmt.Println("No new migrations to apply")
	} else {
		fmt.Printf("Applied %d migration(s)\n", count)
	}
	return nil
}

func migrateDown(ctx context.Context, db *pgxpool.Pool, migrations []migration) error {
	applied, err := getApplied(ctx, db)
	if err != nil {
		return err
	}

	// Find the last applied migration
	var last *migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if applied[migrations[i].Version] {
			last = &migrations[i]
			break
		}
	}

	if last == nil {
		fmt.Println("No migrations to rollback")
		return nil
	}

	if last.DownSQL == "" {
		return fmt.Errorf("migration %s has no down SQL", last.Version)
	}

	fmt.Printf("Rolling back %s (%s)...\n", last.Version, last.Name)

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.Exec(ctx, last.DownSQL); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("executing down %s: %w", last.Filename, err)
	}

	if _, err := tx.Exec(ctx,
		"DELETE FROM schema_migrations WHERE version = $1", last.Version,
	); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("removing %s: %w", last.Version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rollback %s: %w", last.Version, err)
	}

	fmt.Printf("Rolled back %s\n", last.Version)
	return nil
}

func migrateStatus(ctx context.Context, db *pgxpool.Pool, migrations []migration) error {
	applied, err := getAppliedWithTime(ctx, db)
	if err != nil {
		return err
	}

	fmt.Printf("%-14s %-40s %s\n", "VERSION", "NAME", "STATUS")
	fmt.Println(strings.Repeat("-", 80))

	for _, m := range migrations {
		if t, ok := applied[m.Version]; ok {
			fmt.Printf("%-14s %-40s Applied at %s\n", m.Version, m.Name, t.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("%-14s %-40s Pending\n", m.Version, m.Name)
		}
	}
	return nil
}

func getApplied(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func getAppliedWithTime(ctx context.Context, db *pgxpool.Pool) (map[string]time.Time, error) {
	rows, err := db.Query(ctx, "SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var v string
		var t time.Time
		if err := rows.Scan(&v, &t); err != nil {
			return nil, err
		}
		applied[v] = t
	}
	return applied, rows.Err()
}
