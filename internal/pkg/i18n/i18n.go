package i18n

const (
	LangID = "id"
	LangEN = "en"
)

type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Messages holds all translations keyed by language then by message key.
var Messages = map[string]map[string]string{
	LangID: {
		// General
		"success":            "Berhasil",
		"error.internal":     "Terjadi kesalahan internal",
		"error.not_found":    "Data tidak ditemukan",
		"error.bad_request":  "Permintaan tidak valid",
		"error.unauthorized": "Tidak memiliki akses",
		"error.forbidden":    "Akses ditolak",
		"error.validation":   "Validasi gagal",
		"error.duplicate":    "Data sudah ada",
		"error.conflict":     "Konflik data",

		// Auth
		"auth.login_success":    "Login berhasil",
		"auth.login_failed":     "Email atau password salah",
		"auth.token_expired":    "Token sudah kadaluarsa",
		"auth.token_invalid":    "Token tidak valid",
		"auth.otp_sent":         "Kode OTP telah dikirim",
		"auth.otp_invalid":      "Kode OTP tidak valid",
		"auth.password_changed": "Password berhasil diubah",

		// Customer
		"customer.created":   "Pelanggan berhasil dibuat",
		"customer.updated":   "Pelanggan berhasil diperbarui",
		"customer.deleted":   "Pelanggan berhasil dihapus",
		"customer.activated": "Pelanggan berhasil diaktifkan",
		"customer.isolated":  "Pelanggan berhasil diisolir",
		"customer.not_found": "Pelanggan tidak ditemukan",

		// Invoice
		"invoice.created":      "Invoice berhasil dibuat",
		"invoice.paid":         "Pembayaran berhasil",
		"invoice.cancelled":    "Invoice berhasil dibatalkan",
		"invoice.not_found":    "Invoice tidak ditemukan",
		"invoice.already_paid": "Invoice sudah dibayar",

		// Ticket
		"ticket.created":   "Tiket berhasil dibuat",
		"ticket.updated":   "Tiket berhasil diperbarui",
		"ticket.closed":    "Tiket berhasil ditutup",
		"ticket.not_found": "Tiket tidak ditemukan",

		// Package
		"package.created":   "Paket berhasil dibuat",
		"package.updated":   "Paket berhasil diperbarui",
		"package.deleted":   "Paket berhasil dihapus",
		"package.not_found": "Paket tidak ditemukan",

		// Router
		"router.created":   "Router berhasil ditambahkan",
		"router.updated":   "Router berhasil diperbarui",
		"router.deleted":   "Router berhasil dihapus",
		"router.not_found": "Router tidak ditemukan",

		// Payment
		"payment.success":   "Pembayaran berhasil",
		"payment.failed":    "Pembayaran gagal",
		"payment.pending":   "Pembayaran menunggu konfirmasi",
		"payment.not_found": "Pembayaran tidak ditemukan",

		// Notification
		"notification.sent":      "Notifikasi berhasil dikirim",
		"notification.broadcast": "Broadcast berhasil dikirim",

		// Reward
		"reward.created":       "Reward berhasil dibuat",
		"reward.updated":       "Reward berhasil diperbarui",
		"reward.deleted":       "Reward berhasil dihapus",
		"reward.not_found":     "Reward tidak ditemukan",
		"referral.created":     "Referral berhasil dibuat",
		"referral.rewarded":    "Referral berhasil diberi reward",
		"reward_claim.created": "Klaim reward berhasil dibuat",
		"reward_claim.applied": "Klaim reward berhasil diterapkan",

		// Isolir page
		"isolir.page_updated": "Halaman isolir berhasil diperbarui",
		"isolir.not_isolated": "Pelanggan tidak dalam status isolir",

		// Reports / Export
		"export.success": "Export berhasil",
		"export.failed":  "Export gagal",

		// Settings
		"setting.updated": "Pengaturan berhasil diperbarui",
		"setting.deleted": "Pengaturan berhasil dihapus",

		// IPAM
		"ipam.pool_created":     "IP Pool berhasil dibuat",
		"ipam.pool_deleted":     "IP Pool berhasil dihapus",
		"ipam.address_assigned": "Alamat IP berhasil dialokasikan",
		"ipam.address_released": "Alamat IP berhasil dilepaskan",

		// Reseller
		"reseller.created":          "Reseller berhasil dibuat",
		"reseller.updated":          "Reseller berhasil diperbarui",
		"reseller.deleted":          "Reseller berhasil dihapus",
		"reseller.commission_added": "Komisi berhasil ditambahkan",
		"reseller.commission_paid":  "Komisi berhasil dibayarkan",

		// Theme / Dark mode
		"theme.updated":             "Tema berhasil diperbarui",
		"theme.preferences_updated": "Preferensi berhasil diperbarui",
		"theme.invalid":             "Tema harus light, dark, atau system",
	},
	LangEN: {
		// General
		"success":            "Success",
		"error.internal":     "Internal server error",
		"error.not_found":    "Data not found",
		"error.bad_request":  "Invalid request",
		"error.unauthorized": "Unauthorized",
		"error.forbidden":    "Access denied",
		"error.validation":   "Validation failed",
		"error.duplicate":    "Data already exists",
		"error.conflict":     "Data conflict",

		// Auth
		"auth.login_success":    "Login successful",
		"auth.login_failed":     "Invalid email or password",
		"auth.token_expired":    "Token has expired",
		"auth.token_invalid":    "Invalid token",
		"auth.otp_sent":         "OTP code has been sent",
		"auth.otp_invalid":      "Invalid OTP code",
		"auth.password_changed": "Password changed successfully",

		// Customer
		"customer.created":   "Customer created successfully",
		"customer.updated":   "Customer updated successfully",
		"customer.deleted":   "Customer deleted successfully",
		"customer.activated": "Customer activated successfully",
		"customer.isolated":  "Customer isolated successfully",
		"customer.not_found": "Customer not found",

		// Invoice
		"invoice.created":      "Invoice created successfully",
		"invoice.paid":         "Payment successful",
		"invoice.cancelled":    "Invoice cancelled successfully",
		"invoice.not_found":    "Invoice not found",
		"invoice.already_paid": "Invoice already paid",

		// Ticket
		"ticket.created":   "Ticket created successfully",
		"ticket.updated":   "Ticket updated successfully",
		"ticket.closed":    "Ticket closed successfully",
		"ticket.not_found": "Ticket not found",

		// Package
		"package.created":   "Package created successfully",
		"package.updated":   "Package updated successfully",
		"package.deleted":   "Package deleted successfully",
		"package.not_found": "Package not found",

		// Router
		"router.created":   "Router added successfully",
		"router.updated":   "Router updated successfully",
		"router.deleted":   "Router deleted successfully",
		"router.not_found": "Router not found",

		// Payment
		"payment.success":   "Payment successful",
		"payment.failed":    "Payment failed",
		"payment.pending":   "Payment pending confirmation",
		"payment.not_found": "Payment not found",

		// Notification
		"notification.sent":      "Notification sent successfully",
		"notification.broadcast": "Broadcast sent successfully",

		// Reward
		"reward.created":       "Reward created successfully",
		"reward.updated":       "Reward updated successfully",
		"reward.deleted":       "Reward deleted successfully",
		"reward.not_found":     "Reward not found",
		"referral.created":     "Referral created successfully",
		"referral.rewarded":    "Referral rewarded successfully",
		"reward_claim.created": "Reward claim created successfully",
		"reward_claim.applied": "Reward claim applied successfully",

		// Isolir page
		"isolir.page_updated": "Isolation page updated successfully",
		"isolir.not_isolated": "Customer is not in isolated status",

		// Reports / Export
		"export.success": "Export successful",
		"export.failed":  "Export failed",

		// Settings
		"setting.updated": "Setting updated successfully",
		"setting.deleted": "Setting deleted successfully",

		// IPAM
		"ipam.pool_created":     "IP Pool created successfully",
		"ipam.pool_deleted":     "IP Pool deleted successfully",
		"ipam.address_assigned": "IP address assigned successfully",
		"ipam.address_released": "IP address released successfully",

		// Reseller
		"reseller.created":          "Reseller created successfully",
		"reseller.updated":          "Reseller updated successfully",
		"reseller.deleted":          "Reseller deleted successfully",
		"reseller.commission_added": "Commission added successfully",
		"reseller.commission_paid":  "Commission paid successfully",

		// Theme / Dark mode
		"theme.updated":             "Theme updated successfully",
		"theme.preferences_updated": "Preferences updated successfully",
		"theme.invalid":             "Theme must be light, dark, or system",
	},
}

// T translates a message key for the given language.
// Falls back to Indonesian if the key is not found in the requested language.
func T(lang, key string) string {
	if msgs, ok := Messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	// Fallback to Indonesian
	if msgs, ok := Messages[LangID]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	return key
}

// SupportedLanguages returns a list of supported languages.
func SupportedLanguages() []Language {
	return []Language{
		{Code: LangID, Name: "Bahasa Indonesia"},
		{Code: LangEN, Name: "English"},
	}
}
