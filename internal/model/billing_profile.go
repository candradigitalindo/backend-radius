package model

import "time"

// BillingProfile adalah template konfigurasi billing yang bisa dipilih saat create customer.
type BillingProfile struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	// BillingModel: "cycle" | "fixed"
	//   cycle → InvoiceDay & DueDay tetap setiap bulan (misal generate tgl 24, JT tgl 1)
	//   fixed → JT = join_date.Day() customer, invoice = InvoiceDay hari sebelum JT
	BillingModel string `json:"billing_model"`

	// InvoiceDay: untuk cycle = tanggal generate invoice (1-28)
	//             untuk fixed = berapa hari sebelum JT invoice digenerate (0 = auto 7 hari)
	InvoiceDay int `json:"invoice_day"`

	// DueDay: untuk cycle = tanggal jatuh tempo (1-28)
	//         untuk fixed = tidak dipakai (JT = join_date.Day())
	DueDay int `json:"due_day"`

	// IsolirDay: hari setelah JT sebelum customer diisolir (0 = isolir pada hari JT)
	IsolirDay int `json:"isolir_day"`

	// GracePeriod: hari toleransi setelah status isolir sebelum diputus total
	GracePeriod int `json:"grace_period"`

	// PaymentTiming: "due_date" = bayar saat/sebelum JT
	//                "advance"  = bayar di muka saat aktivasi
	PaymentTiming string `json:"payment_timing"`

	IsDefault bool `json:"is_default"`
	IsActive  bool `json:"is_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
