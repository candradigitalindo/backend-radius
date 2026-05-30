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
		{"wa_notification_sender", "own"},
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

	// ==================== CUSTOMER REMINDERS ====================
	fmt.Println("  Creating customer reminder templates...")
	customerReminders := []struct {
		name, rtype, template string
		daysOffset            int
	}{
		{
			name:  "Tagihan Baru Dibuat",
			rtype: "invoice_created", daysOffset: 0,
			template: "Selamat {salam} 🙏\n\nHalo *{nama}*,\n\nTagihan internet Anda untuk periode *{periode}* telah dibuat.\n\n📋 *Detail Tagihan:*\n• No. Invoice: *{nomor_invoice}*\n• Paket: *{paket}*\n• Total: *Rp{jumlah}*\n• Jatuh Tempo: *{jatuh_tempo}*\n\nMohon segera lakukan pembayaran sebelum jatuh tempo untuk menghindari pemutusan layanan.\n\nTerima kasih atas kepercayaan Anda. 🙏",
		},
		{
			name:  "Pengingat H-7 Sebelum Jatuh Tempo",
			rtype: "before_due", daysOffset: 7,
			template: "Selamat {salam} 🙏\n\nHalo *{nama}*,\n\nKami mengingatkan bahwa tagihan internet Anda akan jatuh tempo dalam *7 hari lagi* pada *{jatuh_tempo}*.\n\n📋 *Detail Tagihan:*\n• No. Invoice: *{nomor_invoice}*\n• Paket: *{paket}*\n• Total: *Rp{jumlah}*\n\nMohon segera lakukan pembayaran agar layanan internet Anda tetap aktif.\n\nTerima kasih. 🙏",
		},
		{
			name:  "Pengingat H-1 Sebelum Jatuh Tempo",
			rtype: "before_due", daysOffset: 1,
			template: "⚠️ *Pengingat: Tagihan Jatuh Tempo Besok!*\n\nSelamat {salam} 🙏\n\nHalo *{nama}*,\n\nTagihan internet Anda akan jatuh tempo *BESOK, {jatuh_tempo}*.\n\n📋 *Detail Tagihan:*\n• No. Invoice: *{nomor_invoice}*\n• Paket: *{paket}*\n• Total: *Rp{jumlah}*\n\nSegera lakukan pembayaran untuk menghindari pemutusan layanan.\n\nTerima kasih. 🙏",
		},
		{
			name:  "Tagihan Jatuh Tempo Hari Ini",
			rtype: "on_due", daysOffset: 0,
			template: "🔔 *Pengingat: Tagihan Jatuh Tempo Hari Ini*\n\nSelamat {salam},\n\nHalo *{nama}*,\n\nTagihan internet Anda jatuh tempo *HARI INI, {jatuh_tempo}*.\n\n📋 *Detail Tagihan:*\n• No. Invoice: *{nomor_invoice}*\n• Paket: *{paket}*\n• Total: *Rp{jumlah}*\n\nHarap segera lakukan pembayaran agar layanan tidak terputus.\n\nTerima kasih. 🙏",
		},
		{
			name:  "Notifikasi Layanan Diisolir",
			rtype: "isolir", daysOffset: 0,
			template: "🔴 *Pemberitahuan: Layanan Internet Diputus*\n\nSelamat {salam},\n\nHalo *{nama}*,\n\nKami memberitahukan bahwa layanan internet Anda *telah diisolir* karena tagihan *{nomor_invoice}* senilai *Rp{jumlah}* (jatuh tempo {jatuh_tempo}) belum dibayar.\n\nUntuk mengaktifkan kembali layanan, silakan segera lakukan pembayaran dan hubungi kami.\n\nTerima kasih atas pengertian Anda. 🙏",
		},
		{
			name:  "Konfirmasi Pembayaran Berhasil",
			rtype: "payment", daysOffset: 0,
			template: "✅ *Pembayaran Berhasil!*\n\nSelamat {salam} 🎉\n\nHalo *{nama}*,\n\nPembayaran tagihan Anda telah *berhasil* dikonfirmasi.\n\n📋 *Detail Pembayaran:*\n• No. Invoice: *{nomor_invoice}*\n• Periode: *{periode}*\n• Paket: *{paket}*\n• Jumlah Bayar: *Rp{jumlah}*\n• Kode Pelanggan: *{kode_pelanggan}*\n\nLayanan internet Anda kini aktif dan siap digunakan. Terima kasih! 🙏",
		},
	}
	for _, r := range customerReminders {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reminders (id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,true,$7,$8)
		`, id.New(), tenantID, r.name, r.rtype, r.daysOffset, r.template, now, now); err != nil {
			return fmt.Errorf("insert customer reminder %s: %w", r.name, err)
		}
	}

	// ==================== SUBSCRIPTION REMINDER TEMPLATES (system-level) ====================
	fmt.Println("  Creating subscription reminder templates...")
	subReminders := []struct {
		name, rtype, template string
	}{
		{
			name:  "Notifikasi H-7 Sebelum Berakhir",
			rtype: "sub_expiry_h7",
			template: "Selamat {salam} 🙏\n\nKepada Yth. Tim *{nama_isp}*,\n\nKami menginformasikan bahwa langganan layanan *D Radius* dengan paket *{nama_paket}* Anda akan berakhir dalam *7 hari lagi*, tepatnya pada *{tanggal_berakhir}*.\n\nAgar operasional bisnis ISP Anda tidak terganggu, mohon segera lakukan perpanjangan.\n\n💡 *Cara Perpanjang:*\nMasuk ke Panel D Radius → Menu Beranda → Klik Perpanjang Langganan → Pilih Paket → Lakukan Pembayaran\n\nJika memerlukan bantuan, jangan ragu menghubungi tim support kami.\n\nTerima kasih atas kepercayaan Anda menggunakan D Radius. 🙏",
		},
		{
			name:  "Notifikasi H-1 Sebelum Berakhir",
			rtype: "sub_expiry_h1",
			template: "⚠️ *Peringatan Penting!*\n\nSelamat {salam} 🙏\n\nKepada Yth. Tim *{nama_isp}*,\n\nLangganan paket *{nama_paket}* untuk layanan D Radius Anda akan berakhir *BESOK, {tanggal_berakhir}*.\n\nHarap segera lakukan perpanjangan agar seluruh fitur dan layanan tetap dapat digunakan tanpa gangguan.\n\n🔗 Akses sekarang: Panel D Radius → Perpanjang Langganan\n\nHubungi kami jika memerlukan bantuan atau pertanyaan seputar perpanjangan.\n\nTerima kasih. 🙏",
		},
		{
			name:  "Notifikasi Hari H Berakhir",
			rtype: "sub_expiry_h0",
			template: "🔴 *Pemberitahuan: Langganan Telah Berakhir*\n\nSelamat {salam},\n\nKepada Yth. Tim *{nama_isp}*,\n\nKami menginformasikan bahwa langganan paket *{nama_paket}* untuk layanan D Radius Anda telah *berakhir* pada *{tanggal_berakhir}*.\n\nAkses panel D Radius Anda saat ini *dibatasi*. Untuk mengaktifkan kembali seluruh fitur, silakan lakukan pembayaran perpanjangan secepatnya.\n\n🔗 Cara Perpanjang: Buka D Radius → Layar Perpanjangan → Pilih Paket → Bayar\n\nJika ada kendala atau pertanyaan, segera hubungi tim support kami.\n\nTerima kasih atas perhatian Anda. 🙏",
		},
		{
			name:  "Konfirmasi Pembayaran Langganan",
			rtype: "sub_payment",
			template: "✅ *Pembayaran Langganan Berhasil!*\n\nSelamat {salam} 🎉\n\nKepada Yth. Tim *{nama_isp}*,\n\nTerima kasih! Pembayaran langganan D Radius Anda telah berhasil kami terima dan diproses.\n\n📋 *Detail Pembayaran:*\n• ISP: *{nama_isp}*\n• Paket: *{nama_paket}*\n• Durasi: *{durasi}*\n• Total Bayar: *Rp{jumlah}*\n• Aktif Hingga: *{tanggal_berakhir}*\n\nSeluruh fitur D Radius kini telah aktif dan siap digunakan. Semoga bisnis ISP Anda semakin berkembang! 🚀\n\nTerima kasih atas kepercayaan Anda. 🙏",
		},
		{
			name:  "OTP Reset Password",
			rtype: "otp_reset_password",
			template: "🔐 *Kode OTP Reset Password*\n\nHalo *{nama}*,\n\nKami menerima permintaan reset password untuk akun *{nama_isp}* Anda.\n\n🔑 *Kode OTP Anda:*\n*─────────────*\n*    {kode_otp}    *\n*─────────────*\n\n⏱ Berlaku selama *{durasi}*\n⚠️ Jangan bagikan kode ini kepada siapapun demi keamanan akun Anda.\n\nJika Anda tidak merasa meminta reset password, abaikan pesan ini.\n\nTerima kasih,\n_Tim D Radius_",
		},
		{
			name:  "OTP Registrasi",
			rtype: "otp_registration",
			template: "🔐 *Kode Verifikasi D Radius*\n\nHalo,\n\nKode verifikasi (OTP) untuk pendaftaran akun Anda adalah:\n\n*─────────────*\n*    {kode_otp}    *\n*─────────────*\n\nKode ini berlaku selama 5 menit. Jangan bagikan kode ini kepada siapapun.\n\nTerima kasih,\n_Tim Support D Radius_",
		},
		{
			name:  "Pendaftaran Berhasil (Welcome)",
			rtype: "welcome_tenant",
			template: "🚀 *Selamat Datang di D Radius!*\n\nHalo *{nama_isp}*,\n\nPendaftaran ISP Anda telah berhasil! Berikut adalah detail akun panel manajemen Anda:\n\n🌐 *Panel URL:* {panel_url}\n📧 *Email:* {email}\n🔑 *Password:* `{password}` (8 digit angka)\n\n⚠️ *Penting:*\nSegera ganti password Anda setelah login pertama kali demi keamanan akun.\n\nTerima kasih telah bergabung dengan D Radius!\n_Tim Support D Radius_",
		},
		{
			name:  "Password Baru",
			rtype: "reset_password_new",
			template: "🔐 *Password Baru D Radius*\n\nHalo *{nama_isp}*,\n\nPassword akun panel manajemen Anda telah direset oleh SuperAdmin.\n\n📧 *Email:* {email}\n🔑 *Password Baru:* `{password}` (8 digit angka)\n\nSilakan gunakan password baru ini untuk login. Segera ganti password Anda demi keamanan.\n\nTerima kasih,\n_Tim Support D Radius_",
		},
	}
	// Lookup superadmin tenant ID to store subscription reminder templates
	var superadminTenantID string
	_ = tx.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = 'superadmin' LIMIT 1`).Scan(&superadminTenantID)
	if superadminTenantID != "" {
		for _, r := range subReminders {
			if _, err := tx.Exec(ctx, `
				INSERT INTO reminders (id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at)
				VALUES ($1,$2,$3,$4,0,$5,true,$6,$7)
			`, id.New(), superadminTenantID, r.name, r.rtype, r.template, now, now); err != nil {
				return fmt.Errorf("insert subscription reminder %s: %w", r.name, err)
			}
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
	fmt.Printf("  Customer Reminders: %d\n", len(customerReminders))
	fmt.Printf("  Sub. Reminders:     %d\n", len(subReminders))
	fmt.Println("  -------------------------------------------")

	return nil
}
