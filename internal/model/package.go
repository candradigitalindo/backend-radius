package model

import "time"

type Package struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	BandwidthUp     int                  `json:"bandwidth_up"`
	BandwidthDown   int                  `json:"bandwidth_down"`
	Price           int64                `json:"price"`
	BurstLimit      string               `json:"burst_limit"`
	AddressList     string               `json:"address_list"`
	IsActive        bool                 `json:"is_active"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}
