package model

import "time"

type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionGroup groups related permissions for UI display.
type PermissionGroup struct {
	Key         string       `json:"key"`
	Label       string       `json:"label"`
	Section     string       `json:"section"` // sidebar category: dashboard, pelanggan, keuangan, jaringan, layanan, marketing, administrasi
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func AllPermissionGroups() []PermissionGroup {
	return []PermissionGroup{
		// ── Dashboard ──────────────────────────────────────────────────────
		{Key: "dashboard", Label: "Dashboard", Section: "dashboard", Permissions: []Permission{
			{Key: "dashboard.view", Label: "Lihat dashboard"},
		}},

		// ── Pelanggan ──────────────────────────────────────────────────────
		{Key: "customers", Label: "Data Pelanggan", Section: "pelanggan", Permissions: []Permission{
			{Key: "customers.view", Label: "Lihat pelanggan"},
			{Key: "customers.create", Label: "Tambah pelanggan"},
			{Key: "customers.edit", Label: "Edit pelanggan"},
			{Key: "customers.delete", Label: "Hapus pelanggan"},
			{Key: "customers.isolate", Label: "Isolir / aktifkan"},
		}},
		{Key: "packages", Label: "Paket Internet", Section: "pelanggan", Permissions: []Permission{
			{Key: "packages.view", Label: "Lihat paket"},
			{Key: "packages.create", Label: "Tambah paket"},
			{Key: "packages.edit", Label: "Edit paket"},
			{Key: "packages.delete", Label: "Hapus paket"},
		}},

		// ── Keuangan ───────────────────────────────────────────────────────
		{Key: "invoices", Label: "Tagihan", Section: "keuangan", Permissions: []Permission{
			{Key: "invoices.view", Label: "Lihat tagihan"},
			{Key: "invoices.create", Label: "Buat tagihan"},
			{Key: "invoices.edit", Label: "Edit tagihan"},
			{Key: "invoices.delete", Label: "Hapus tagihan"},
			{Key: "invoices.pay", Label: "Catat pembayaran"},
		}},
		{Key: "expenses", Label: "Pengeluaran", Section: "keuangan", Permissions: []Permission{
			{Key: "expenses.view", Label: "Lihat pengeluaran"},
			{Key: "expenses.create", Label: "Tambah pengeluaran"},
			{Key: "expenses.edit", Label: "Edit pengeluaran"},
			{Key: "expenses.delete", Label: "Hapus pengeluaran"},
		}},
		{Key: "reports", Label: "Laporan", Section: "keuangan", Permissions: []Permission{
			{Key: "reports.view", Label: "Lihat laporan"},
			{Key: "reports.export", Label: "Export laporan"},
		}},

		// ── Jaringan ───────────────────────────────────────────────────────
		{Key: "routers", Label: "Router", Section: "jaringan", Permissions: []Permission{
			{Key: "routers.view", Label: "Lihat router"},
			{Key: "routers.create", Label: "Tambah router"},
			{Key: "routers.edit", Label: "Edit router"},
			{Key: "routers.delete", Label: "Hapus router"},
		}},
		{Key: "olts", Label: "OLT", Section: "jaringan", Permissions: []Permission{
			{Key: "olts.view", Label: "Lihat OLT"},
			{Key: "olts.create", Label: "Tambah OLT"},
			{Key: "olts.edit", Label: "Edit OLT"},
			{Key: "olts.delete", Label: "Hapus OLT"},
		}},
		{Key: "odps", Label: "ODP", Section: "jaringan", Permissions: []Permission{
			{Key: "odps.view", Label: "Lihat ODP"},
			{Key: "odps.create", Label: "Tambah ODP"},
			{Key: "odps.edit", Label: "Edit ODP"},
			{Key: "odps.delete", Label: "Hapus ODP"},
		}},
		{Key: "onts", Label: "ONT / Perangkat", Section: "jaringan", Permissions: []Permission{
			{Key: "onts.view", Label: "Lihat ONT"},
			{Key: "onts.create", Label: "Tambah ONT"},
			{Key: "onts.edit", Label: "Edit ONT"},
			{Key: "onts.delete", Label: "Hapus ONT"},
		}},
		{Key: "ftth", Label: "Peta Jaringan", Section: "jaringan", Permissions: []Permission{
			{Key: "ftth.view", Label: "Lihat peta jaringan"},
		}},
		{Key: "ip_pools", Label: "IPAM (IP Pool)", Section: "jaringan", Permissions: []Permission{
			{Key: "ip_pools.view", Label: "Lihat IP pool"},
			{Key: "ip_pools.create", Label: "Tambah IP pool"},
			{Key: "ip_pools.edit", Label: "Edit IP pool"},
			{Key: "ip_pools.delete", Label: "Hapus IP pool"},
		}},
		{Key: "bandwidth", Label: "Monitoring Bandwidth", Section: "jaringan", Permissions: []Permission{
			{Key: "bandwidth.view", Label: "Lihat bandwidth & saturasi koneksi"},
		}},
		{Key: "genieacs", Label: "GenieACS (TR-069)", Section: "jaringan", Permissions: []Permission{
			{Key: "genieacs.view", Label: "Lihat perangkat TR-069"},
			{Key: "genieacs.manage", Label: "Kelola perangkat TR-069"},
		}},

		// ── Layanan ────────────────────────────────────────────────────────
		{Key: "tickets", Label: "Tiket", Section: "layanan", Permissions: []Permission{
			{Key: "tickets.view", Label: "Lihat tiket"},
			{Key: "tickets.create", Label: "Buat tiket"},
			{Key: "tickets.edit", Label: "Edit tiket"},
			{Key: "tickets.delete", Label: "Hapus tiket"},
			{Key: "tickets.assign", Label: "Assign tiket"},
		}},
		{Key: "vouchers", Label: "Voucher", Section: "layanan", Permissions: []Permission{
			{Key: "vouchers.view", Label: "Lihat voucher"},
			{Key: "vouchers.create", Label: "Buat voucher"},
			{Key: "vouchers.edit", Label: "Edit voucher"},
			{Key: "vouchers.delete", Label: "Hapus voucher"},
		}},
		{Key: "notifications", Label: "Notifikasi", Section: "layanan", Permissions: []Permission{
			{Key: "notifications.view", Label: "Lihat notifikasi"},
			{Key: "notifications.send", Label: "Kirim notifikasi"},
		}},
		{Key: "broadcasts", Label: "Broadcast WA", Section: "layanan", Permissions: []Permission{
			{Key: "broadcasts.view", Label: "Lihat broadcast"},
			{Key: "broadcasts.create", Label: "Kirim broadcast"},
			{Key: "broadcasts.delete", Label: "Hapus broadcast"},
		}},

		// ── Marketing ──────────────────────────────────────────────────────
		{Key: "rewards", Label: "Reward", Section: "marketing", Permissions: []Permission{
			{Key: "rewards.view", Label: "Lihat reward & dashboard"},
			{Key: "rewards.create", Label: "Tambah reward"},
			{Key: "rewards.edit", Label: "Edit reward"},
			{Key: "rewards.delete", Label: "Hapus reward"},
		}},
		{Key: "resellers", Label: "Reseller & Referral", Section: "marketing", Permissions: []Permission{
			{Key: "resellers.view", Label: "Lihat reseller & referral"},
			{Key: "resellers.create", Label: "Tambah reseller"},
			{Key: "resellers.edit", Label: "Edit reseller"},
			{Key: "resellers.delete", Label: "Hapus reseller"},
		}},

		// ── Administrasi ───────────────────────────────────────────────────
		{Key: "users", Label: "Pengguna", Section: "administrasi", Permissions: []Permission{
			{Key: "users.view", Label: "Lihat pengguna"},
			{Key: "users.create", Label: "Tambah pengguna"},
			{Key: "users.edit", Label: "Edit pengguna"},
			{Key: "users.delete", Label: "Hapus pengguna"},
		}},
		{Key: "roles", Label: "Role & Permission", Section: "administrasi", Permissions: []Permission{
			{Key: "roles.view", Label: "Lihat role"},
			{Key: "roles.create", Label: "Tambah role"},
			{Key: "roles.edit", Label: "Edit role"},
			{Key: "roles.delete", Label: "Hapus role"},
		}},
		{Key: "whatsapp", Label: "WhatsApp", Section: "administrasi", Permissions: []Permission{
			{Key: "whatsapp.view", Label: "Lihat WhatsApp"},
			{Key: "whatsapp.configure", Label: "Konfigurasi WhatsApp"},
			{Key: "whatsapp.send", Label: "Kirim pesan manual"},
		}},
		{Key: "tenant", Label: "Profil Tenant", Section: "administrasi", Permissions: []Permission{
			{Key: "tenant.view", Label: "Lihat profil tenant"},
			{Key: "tenant.edit", Label: "Edit profil tenant"},
		}},
		{Key: "subscription", Label: "Langganan Platform", Section: "administrasi", Permissions: []Permission{
			{Key: "subscription.view", Label: "Lihat langganan"},
			{Key: "subscription.manage", Label: "Kelola langganan"},
		}},
		{Key: "settings", Label: "Pengaturan", Section: "administrasi", Permissions: []Permission{
			{Key: "settings.view", Label: "Lihat pengaturan & panduan"},
			{Key: "settings.edit", Label: "Edit pengaturan"},
		}},
		{Key: "reminders", Label: "Pengingat Tagihan", Section: "administrasi", Permissions: []Permission{
			{Key: "reminders.view", Label: "Lihat pengingat"},
			{Key: "reminders.create", Label: "Tambah pengingat"},
			{Key: "reminders.edit", Label: "Edit pengingat"},
			{Key: "reminders.delete", Label: "Hapus pengingat"},
		}},
	}
}

func AllPermissionKeys() []string {
	groups := AllPermissionGroups()
	keys := make([]string, 0, 80)
	for _, g := range groups {
		for _, p := range g.Permissions {
			keys = append(keys, p.Key)
		}
	}
	return keys
}

func DefaultAdminPermissions() []string {
	excluded := map[string]bool{
		"users.create": true, "users.edit": true, "users.delete": true,
		"roles.create": true, "roles.edit": true, "roles.delete": true,
		"tenant.edit": true,
	}
	all := AllPermissionKeys()
	perms := make([]string, 0, len(all))
	for _, k := range all {
		if !excluded[k] {
			perms = append(perms, k)
		}
	}
	return perms
}

func DefaultTechnicianPermissions() []string {
	return []string{
		"dashboard.view",
		"customers.view",
		"tickets.view", "tickets.create", "tickets.edit",
		"routers.view",
		"olts.view",
		"odps.view",
		"onts.view",
		"ftth.view",
		"bandwidth.view",
	}
}
