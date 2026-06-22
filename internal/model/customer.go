package model

import "time"

type Customer struct {
	ID                      string     `json:"id"`
	TenantID                string     `json:"tenant_id"`
	CustomerCode            string     `json:"customer_code"`
	Name                    string     `json:"name"`
	NIK                     string     `json:"nik"`
	Phone                   string     `json:"phone"`
	Email                   string     `json:"email"`
	Address                 string     `json:"address"`
	Latitude                *float64   `json:"latitude,omitempty"`
	Longitude               *float64   `json:"longitude,omitempty"`
	ConnectionType          string     `json:"connection_type"`
	PPPoEUsername           string     `json:"pppoe_username"`
	PPPoEPassword           string     `json:"pppoe_password"`
	IPAddress               string     `json:"ip_address"`
	PackageID               *string    `json:"package_id,omitempty"`
	RouterID                *string    `json:"router_id,omitempty"`
	ODPPortID               *string    `json:"odp_port_id,omitempty"`
	ResellerID              *string    `json:"reseller_id,omitempty"`
	JoinDate                time.Time  `json:"join_date"`
	BillingDate             int        `json:"billing_date"`
	BillingType             string     `json:"billing_type"`
	BillingDeadline         int        `json:"billing_deadline"`
	BillingProfileID        *string    `json:"billing_profile_id,omitempty"`
	PendingBillingProfileID *string    `json:"pending_billing_profile_id,omitempty"`
	PreviousPackagePrice    int64      `json:"previous_package_price"`
	PackageChangedAt        *time.Time `json:"package_changed_at,omitempty"`
	CustomPrice             *int64     `json:"custom_price,omitempty"`
	Discount                int64      `json:"discount"`
	AdditionalFee           int64      `json:"additional_fee"`
	FeeDescription          string     `json:"fee_description"`
	Status                  string     `json:"status"`
	ConnectionStatus        string     `json:"connection_status"`
	IsolatedAt              *time.Time `json:"isolated_at,omitempty"`
	Notes                   string     `json:"notes"`
	ReferralCode            string     `json:"referral_code,omitempty"`
	PasswordHash            string     `json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`

	Package *Package `json:"package,omitempty"`
	Router  *Router  `json:"router,omitempty"`
	ONT     *ONT     `json:"ont,omitempty"`

	// Computed fields - not stored in DB
	ACSURL         *string        `json:"acs_url,omitempty"`
	CurrentInvoice *Invoice       `json:"current_invoice,omitempty"`
	ActiveSession  *RadiusSession `json:"active_session,omitempty"`
}
