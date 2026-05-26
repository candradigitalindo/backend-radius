package model

import "time"

type Tenant struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	Email         string     `json:"email"`
	Phone         string     `json:"phone"`
	Address       string     `json:"address"`
	LogoURL       string     `json:"logo_url"`
	Timezone      string     `json:"timezone"`
	Currency      string     `json:"currency"`
	BillingCycle        int    `json:"billing_cycle"`
	DueDay              int    `json:"due_day"`
	IsolirDay           int    `json:"isolir_day"`
	GracePeriod         int    `json:"grace_period"`
	DefaultBillingType  string `json:"default_billing_type"`
	Plan          string     `json:"plan"`
	PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	MaxCustomers  int        `json:"max_customers"`
	WAAPIKey      string     `json:"-"`
	WASender      string     `json:"wa_sender"`
	PGProvider    string     `json:"pg_provider"`
	PGAPIKey      string     `json:"-"`
	PGSecretKey   string     `json:"-"`
	PGMerchantID  string     `json:"pg_merchant_id"`
	PGSandbox     bool       `json:"pg_sandbox"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TenantWithSecrets is used only by settings endpoints that need to expose integration keys.
type TenantWithSecrets struct {
	Tenant
	WAAPIKey    string `json:"wa_api_key"`
	PGAPIKey    string `json:"pg_api_key"`
	PGSecretKey string `json:"pg_secret_key"`
}

// ToWithSecrets returns a TenantWithSecrets from a Tenant, copying the hidden fields.
func (t *Tenant) ToWithSecrets() TenantWithSecrets {
	return TenantWithSecrets{
		Tenant:      *t,
		WAAPIKey:    t.WAAPIKey,
		PGAPIKey:    t.PGAPIKey,
		PGSecretKey: t.PGSecretKey,
	}
}
