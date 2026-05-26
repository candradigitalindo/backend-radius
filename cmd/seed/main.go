package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db := database.Connect(cfg)
	defer db.Close()

	ctx := context.Background()

	fmt.Println("Starting database seeder...")

	if err := seed(ctx, db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	fmt.Println("Database seeded successfully!")
}

func seed(ctx context.Context, db *pgxpool.Pool) error {
	var count int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&count); err != nil {
		return fmt.Errorf("checking tenants: %w", err)
	}
	if count > 0 {
		fmt.Println("Database already has data, skipping seed.")
		return nil
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// ==================== TENANT ====================
	tenantID := id.New()
	fmt.Printf("  Creating tenant (ID: %s)...\n", tenantID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, email, phone, address, timezone, currency,
			billing_cycle, due_day, isolir_day, grace_period, plan, max_customers, is_active,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		tenantID, "ISP Demo", "isp-demo", "admin@ispdemo.com", "08123456789",
		"Jl. Contoh No. 123, Jakarta", "Asia/Jakarta", "IDR",
		1, 20, 21, 3, "free", 50, true,
		now, now,
	); err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}

	// ==================== ADMIN USER ====================
	adminID := id.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	fmt.Println("  Creating admin user (admin@ispdemo.com / password123)...")
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, tenant_id, name, email, password_hash, role, phone, is_active,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		adminID, tenantID, "Administrator", "admin@ispdemo.com", string(hash),
		"admin", "08123456789", true,
		now, now,
	); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	// ==================== TECHNICIAN USER ====================
	techID := id.New()
	techHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	fmt.Println("  Creating technician user (teknisi@ispdemo.com / password123)...")
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, tenant_id, name, email, password_hash, role, phone, is_active,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		techID, tenantID, "Teknisi", "teknisi@ispdemo.com", string(techHash),
		"technician", "08198765432", true,
		now, now,
	); err != nil {
		return fmt.Errorf("insert technician: %w", err)
	}

	// suppress unused warnings
	_ = adminID
	_ = techID

	// ==================== PACKAGES ====================
	pkgBasicID := id.New()
	pkgStdID := id.New()
	pkgPremiumID := id.New()

	packages := []struct {
		id, name, desc string
		up, down       int
		price          int64
	}{
		{pkgBasicID, "Basic 10 Mbps", "Paket internet 10 Mbps", 10, 10, 150000},
		{pkgStdID, "Standard 20 Mbps", "Paket internet 20 Mbps", 20, 20, 250000},
		{pkgPremiumID, "Premium 50 Mbps", "Paket internet 50 Mbps", 50, 50, 450000},
	}

	fmt.Println("  Creating packages...")
	for _, p := range packages {
		if _, err := tx.Exec(ctx, `
			INSERT INTO packages (id, tenant_id, name, description, bandwidth_up, bandwidth_down,
				price, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			p.id, tenantID, p.name, p.desc, p.up, p.down, p.price, true, now, now,
		); err != nil {
			return fmt.Errorf("insert package %s: %w", p.name, err)
		}
	}

	// ==================== ROUTERS ====================
	routerID := id.New()
	fmt.Println("  Creating router...")
	if _, err := tx.Exec(ctx, `
		INSERT INTO routers (id, tenant_id, name, identity, vpn_ip, vpn_public_key,
			radius_secret, coa_port, heartbeat_token, is_online, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		routerID, tenantID, "Router Utama", "MikroTik-Main",
		"10.10.0.2", "", "sharedsecret123", 3799,
		"demo-heartbeat-token", false, true,
		now, now,
	); err != nil {
		return fmt.Errorf("insert router: %w", err)
	}

	// ==================== CUSTOMERS ====================
	customers := []struct {
		id, code, name, phone, user, pass string
		pkgID                             string
		billingDate                       int
	}{
		{id.New(), "CUST-001", "Budi Santoso", "08111111111", "budi.santoso", "pppoe123", pkgBasicID, 1},
		{id.New(), "CUST-002", "Siti Rahayu", "08122222222", "siti.rahayu", "pppoe123", pkgStdID, 5},
		{id.New(), "CUST-003", "Ahmad Wijaya", "08133333333", "ahmad.wijaya", "pppoe123", pkgPremiumID, 10},
		{id.New(), "CUST-004", "Dewi Lestari", "08144444444", "dewi.lestari", "pppoe123", pkgStdID, 15},
		{id.New(), "CUST-005", "Rudi Hermawan", "08155555555", "rudi.hermawan", "pppoe123", pkgBasicID, 20},
	}

	fmt.Println("  Creating customers...")
	for _, c := range customers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO customers (id, tenant_id, customer_code, name, phone, connection_type,
				pppoe_username, pppoe_password, package_id, router_id, billing_date, status,
				join_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`,
			c.id, tenantID, c.code, c.name, c.phone, "pppoe",
			c.user, c.pass, c.pkgID, routerID, c.billingDate, "active",
			now, now, now,
		); err != nil {
			return fmt.Errorf("insert customer %s: %w", c.code, err)
		}
	}

	// ==================== EXPENSE CATEGORIES ====================
	expCategories := []struct {
		id, name, color string
	}{
		{id.New(), "Peralatan Jaringan", "#3B82F6"},
		{id.New(), "Listrik & Internet", "#EF4444"},
		{id.New(), "Gaji Karyawan", "#10B981"},
		{id.New(), "Transportasi", "#F59E0B"},
		{id.New(), "Lain-lain", "#6B7280"},
	}

	fmt.Println("  Creating expense categories...")
	for _, ec := range expCategories {
		if _, err := tx.Exec(ctx, `
			INSERT INTO expense_categories (id, tenant_id, name, color)
			VALUES ($1, $2, $3, $4)
		`, ec.id, tenantID, ec.name, ec.color); err != nil {
			return fmt.Errorf("insert expense category %s: %w", ec.name, err)
		}
	}

	// ==================== VOUCHER PRODUCTS ====================
	vprod1 := id.New()
	vprod2 := id.New()

	voucherProducts := []struct {
		id, name, profile  string
		duration, up, down int
		price              int64
	}{
		{vprod1, "Voucher 1 Jam", "hotspot-1jam", 60, 5, 5, 3000},
		{vprod2, "Voucher 3 Jam", "hotspot-3jam", 180, 10, 10, 7000},
	}

	fmt.Println("  Creating voucher products...")
	for _, vp := range voucherProducts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO voucher_products (id, tenant_id, name, duration, bandwidth_up, bandwidth_down,
				price, profile_name, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			vp.id, tenantID, vp.name, vp.duration, vp.up, vp.down,
			vp.price, vp.profile, true, now, now,
		); err != nil {
			return fmt.Errorf("insert voucher product %s: %w", vp.name, err)
		}
	}

	// ==================== SETTINGS ====================
	settings := []struct {
		key, value string
	}{
		{"company_name", "ISP Demo"},
		{"company_phone", "08123456789"},
		{"company_address", "Jl. Contoh No. 123, Jakarta"},
		{"invoice_prefix", "INV"},
		{"invoice_footer", "Terima kasih atas pembayaran Anda."},
		{"isolir_enabled", "true"},
		{"wa_notification_enabled", "false"},
	}

	fmt.Println("  Creating settings...")
	for _, s := range settings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO settings (id, tenant_id, key, value)
			VALUES ($1, $2, $3, $4)
		`, id.New(), tenantID, s.key, s.value); err != nil {
			return fmt.Errorf("insert setting %s: %w", s.key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Println()
	fmt.Println("  Seed Summary:")
	fmt.Println("  -------------------------------------------")
	fmt.Println("  Tenant:             ISP Demo")
	fmt.Println("  Admin Login:        admin@ispdemo.com / password123")
	fmt.Println("  Technician Login:   teknisi@ispdemo.com / password123")
	fmt.Printf("  Packages:           %d\n", len(packages))
	fmt.Println("  Router:             1 (Router Utama)")
	fmt.Printf("  Customers:          %d\n", len(customers))
	fmt.Printf("  Expense Categories: %d\n", len(expCategories))
	fmt.Printf("  Voucher Products:   %d\n", len(voucherProducts))
	fmt.Printf("  Settings:           %d\n", len(settings))
	fmt.Println("  -------------------------------------------")

	return nil
}
