package model

import "time"

type Package struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	BandwidthUp   int       `json:"bandwidth_up"`
	BandwidthDown int       `json:"bandwidth_down"`
	Price         int64     `json:"price"`
	BurstLimit    string    `json:"burst_limit"`
	AddressList   string    `json:"address_list"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AutoBurstMbps returns the burst-limit for a plan rate: 125% rounded up.
// Burst is always derived from the plan rate, never stored — RouterOS rejects
// the whole queue when burst-limit < max-limit, which kills the PPP session.
func AutoBurstMbps(rateMbps int) int {
	return (rateMbps*5 + 3) / 4
}

// BurstThresholdMbps returns the burst-threshold for a plan rate: 75%, min 1.
func BurstThresholdMbps(rateMbps int) int {
	t := rateMbps * 3 / 4
	if t < 1 {
		t = 1
	}
	return t
}
