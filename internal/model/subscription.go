package model

import "time"

// SubscriptionPlan represents a plan available for purchase.
type SubscriptionPlan struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	Price          int64     `json:"price"`
	DurationMonths int       `json:"duration_months"`
	MaxCustomers   int       `json:"max_customers"`
	MaxRouters     int       `json:"max_routers"`
	Features       []string  `json:"features"`
	IsPopular      bool      `json:"is_popular"`
	IsActive       bool      `json:"is_active"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SubscriptionOrder represents a tenant's subscription purchase/history.
type SubscriptionOrder struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	PlanID         string     `json:"plan_id"`
	PlanName       string     `json:"plan_name"`
	Amount         int64      `json:"amount"`
	DurationMonths int        `json:"duration_months"`
	Status         string     `json:"status"` // pending, paid, expired, cancelled
	PaymentMethod  string     `json:"payment_method"`
	PaymentURL     string     `json:"payment_url"`
	PaymentRef     string     `json:"payment_ref"`
	SnapToken      string     `json:"snap_token,omitempty"`
	UniqueCode     int        `json:"unique_code,omitempty"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	StartsAt       *time.Time `json:"starts_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Notes          string     `json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
