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
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func AllPermissionGroups() []PermissionGroup {
	return []PermissionGroup{
		{Key: "dashboard", Label: "Dashboard", Permissions: []Permission{
			{Key: "dashboard.view", Label: "Lihat dashboard"},
		}},
		{Key: "customers", Label: "Pelanggan", Permissions: []Permission{
			{Key: "customers.view", Label: "Lihat pelanggan"},
			{Key: "customers.create", Label: "Tambah pelanggan"},
			{Key: "customers.edit", Label: "Edit pelanggan"},
			{Key: "customers.delete", Label: "Hapus pelanggan"},
			{Key: "customers.isolate", Label: "Isolir/aktifkan"},
		}},
		{Key: "packages", Label: "Paket", Permissions: []Permission{
			{Key: "packages.view", Label: "Lihat paket"},
			{Key: "packages.create", Label: "Tambah paket"},
			{Key: "packages.edit", Label: "Edit paket"},
			{Key: "packages.delete", Label: "Hapus paket"},
		}},
		{Key: "invoices", Label: "Tagihan", Permissions: []Permission{
			{Key: "invoices.view", Label: "Lihat tagihan"},
			{Key: "invoices.create", Label: "Buat tagihan"},
			{Key: "invoices.edit", Label: "Edit tagihan"},
			{Key: "invoices.delete", Label: "Hapus tagihan"},
			{Key: "invoices.pay", Label: "Catat pembayaran"},
		}},
		{Key: "tickets", Label: "Tiket", Permissions: []Permission{
			{Key: "tickets.view", Label: "Lihat tiket"},
			{Key: "tickets.create", Label: "Buat tiket"},
			{Key: "tickets.edit", Label: "Edit tiket"},
			{Key: "tickets.delete", Label: "Hapus tiket"},
			{Key: "tickets.assign", Label: "Assign tiket"},
		}},
		{Key: "routers", Label: "Router", Permissions: []Permission{
			{Key: "routers.view", Label: "Lihat router"},
			{Key: "routers.create", Label: "Tambah router"},
			{Key: "routers.edit", Label: "Edit router"},
			{Key: "routers.delete", Label: "Hapus router"},
		}},
		{Key: "olts", Label: "OLT", Permissions: []Permission{
			{Key: "olts.view", Label: "Lihat OLT"},
			{Key: "olts.create", Label: "Tambah OLT"},
			{Key: "olts.edit", Label: "Edit OLT"},
			{Key: "olts.delete", Label: "Hapus OLT"},
		}},
		{Key: "odps", Label: "ODP", Permissions: []Permission{
			{Key: "odps.view", Label: "Lihat ODP"},
			{Key: "odps.create", Label: "Tambah ODP"},
			{Key: "odps.edit", Label: "Edit ODP"},
			{Key: "odps.delete", Label: "Hapus ODP"},
		}},
		{Key: "onts", Label: "ONT", Permissions: []Permission{
			{Key: "onts.view", Label: "Lihat ONT"},
			{Key: "onts.create", Label: "Tambah ONT"},
			{Key: "onts.edit", Label: "Edit ONT"},
			{Key: "onts.delete", Label: "Hapus ONT"},
		}},
		{Key: "ftth", Label: "FTTH", Permissions: []Permission{
			{Key: "ftth.view", Label: "Lihat FTTH"},
		}},
		{Key: "vouchers", Label: "Voucher", Permissions: []Permission{
			{Key: "vouchers.view", Label: "Lihat voucher"},
			{Key: "vouchers.create", Label: "Buat voucher"},
			{Key: "vouchers.delete", Label: "Hapus voucher"},
		}},
		{Key: "expenses", Label: "Pengeluaran", Permissions: []Permission{
			{Key: "expenses.view", Label: "Lihat pengeluaran"},
			{Key: "expenses.create", Label: "Tambah pengeluaran"},
			{Key: "expenses.edit", Label: "Edit pengeluaran"},
			{Key: "expenses.delete", Label: "Hapus pengeluaran"},
		}},
		{Key: "broadcasts", Label: "Broadcast", Permissions: []Permission{
			{Key: "broadcasts.view", Label: "Lihat broadcast"},
			{Key: "broadcasts.create", Label: "Kirim broadcast"},
			{Key: "broadcasts.delete", Label: "Hapus broadcast"},
		}},
		{Key: "reports", Label: "Laporan", Permissions: []Permission{
			{Key: "reports.view", Label: "Lihat laporan"},
			{Key: "reports.export", Label: "Export laporan"},
		}},
		{Key: "bandwidth", Label: "Bandwidth", Permissions: []Permission{
			{Key: "bandwidth.view", Label: "Lihat bandwidth"},
		}},
		{Key: "rewards", Label: "Reward", Permissions: []Permission{
			{Key: "rewards.view", Label: "Lihat reward"},
			{Key: "rewards.create", Label: "Tambah reward"},
			{Key: "rewards.edit", Label: "Edit reward"},
			{Key: "rewards.delete", Label: "Hapus reward"},
		}},
		{Key: "resellers", Label: "Reseller", Permissions: []Permission{
			{Key: "resellers.view", Label: "Lihat reseller"},
			{Key: "resellers.create", Label: "Tambah reseller"},
			{Key: "resellers.edit", Label: "Edit reseller"},
			{Key: "resellers.delete", Label: "Hapus reseller"},
		}},
		{Key: "ip_pools", Label: "IPAM", Permissions: []Permission{
			{Key: "ip_pools.view", Label: "Lihat IP pool"},
			{Key: "ip_pools.create", Label: "Tambah IP pool"},
			{Key: "ip_pools.edit", Label: "Edit IP pool"},
			{Key: "ip_pools.delete", Label: "Hapus IP pool"},
		}},
		{Key: "notifications", Label: "Notifikasi", Permissions: []Permission{
			{Key: "notifications.view", Label: "Lihat notifikasi"},
			{Key: "notifications.send", Label: "Kirim notifikasi"},
		}},
		{Key: "whatsapp", Label: "WhatsApp", Permissions: []Permission{
			{Key: "whatsapp.view", Label: "Lihat WhatsApp"},
			{Key: "whatsapp.configure", Label: "Konfigurasi WhatsApp"},
			{Key: "whatsapp.send", Label: "Kirim pesan"},
		}},
		{Key: "genieacs", Label: "GenieACS", Permissions: []Permission{
			{Key: "genieacs.view", Label: "Lihat perangkat"},
			{Key: "genieacs.manage", Label: "Kelola perangkat"},
		}},
		{Key: "settings", Label: "Pengaturan", Permissions: []Permission{
			{Key: "settings.view", Label: "Lihat pengaturan"},
			{Key: "settings.edit", Label: "Edit pengaturan"},
		}},
		{Key: "users", Label: "Pengguna", Permissions: []Permission{
			{Key: "users.view", Label: "Lihat pengguna"},
			{Key: "users.create", Label: "Tambah pengguna"},
			{Key: "users.edit", Label: "Edit pengguna"},
			{Key: "users.delete", Label: "Hapus pengguna"},
		}},
		{Key: "roles", Label: "Role & Permission", Permissions: []Permission{
			{Key: "roles.view", Label: "Lihat role"},
			{Key: "roles.create", Label: "Tambah role"},
			{Key: "roles.edit", Label: "Edit role"},
			{Key: "roles.delete", Label: "Hapus role"},
		}},
		{Key: "tenant", Label: "Tenant", Permissions: []Permission{
			{Key: "tenant.view", Label: "Lihat tenant"},
			{Key: "tenant.edit", Label: "Edit tenant"},
		}},
		{Key: "subscription", Label: "Langganan", Permissions: []Permission{
			{Key: "subscription.view", Label: "Lihat langganan"},
			{Key: "subscription.manage", Label: "Kelola langganan"},
		}},
		{Key: "reminders", Label: "Pengingat", Permissions: []Permission{
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
