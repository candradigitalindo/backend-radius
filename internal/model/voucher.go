package model

import "time"

type VoucherProduct struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Duration      int       `json:"duration"`
	BandwidthUp   int       `json:"bandwidth_up"`
	BandwidthDown int       `json:"bandwidth_down"`
	Price         int64     `json:"price"`
	ProfileName   string    `json:"profile_name"`
	RouterID      *string   `json:"router_id,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Voucher struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	ProductID   string     `json:"product_id"`
	Username    string     `json:"username"`
	Password    string     `json:"password"`
	Status      string     `json:"status"`
	BuyerPhone  *string    `json:"buyer_phone,omitempty"`
	SoldAt      *time.Time `json:"sold_at,omitempty"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	Product *VoucherProduct `json:"product,omitempty"`
}

type VoucherPayment struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	VoucherID    string     `json:"voucher_id"`
	BuyerName    string     `json:"buyer_name"`
	BuyerPhone   string     `json:"buyer_phone"`
	Amount       int64      `json:"amount"`
	Gateway      string     `json:"gateway"`
	GatewayTrxID string     `json:"gateway_trx_id"`
	Status       string     `json:"status"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	Voucher *Voucher `json:"voucher,omitempty"`
}
