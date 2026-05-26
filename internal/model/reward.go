package model

import "time"

// Reward represents a reward/referral program for a tenant.
type Reward struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Type        string     `json:"type"`         // referral, loyalty, promo
	Value       int64      `json:"value"`        // reward amount in smallest currency unit
	ValueType   string     `json:"value_type"`   // fixed, percentage
	MinInvoices int        `json:"min_invoices"` // minimum paid invoices to qualify (loyalty)
	IsActive    bool       `json:"is_active"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Referral tracks customer referrals.
type Referral struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	ReferrerID   string     `json:"referrer_id"` // customer who referred
	ReferredID   string     `json:"referred_id"` // new customer
	RewardID     string     `json:"reward_id"`
	ReferralCode string     `json:"referral_code"`
	Status       string     `json:"status"` // pending, qualified, rewarded, expired
	RewardAmount int64      `json:"reward_amount"`
	QualifiedAt  *time.Time `json:"qualified_at,omitempty"`
	RewardedAt   *time.Time `json:"rewarded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	// Relations
	ReferrerName string `json:"referrer_name,omitempty"`
	ReferredName string `json:"referred_name,omitempty"`
}

// RewardClaim represents a redeemed reward (discount on next invoice, balance credit, etc.)
type RewardClaim struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	CustomerID string     `json:"customer_id"`
	RewardID   string     `json:"reward_id"`
	ReferralID *string    `json:"referral_id,omitempty"`
	Amount     int64      `json:"amount"`
	Type       string     `json:"type"`   // invoice_discount, balance_credit
	Status     string     `json:"status"` // pending, applied, expired
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`

	// Relations
	CustomerName string `json:"customer_name,omitempty"`
	RewardName   string `json:"reward_name,omitempty"`
}
