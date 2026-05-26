package model

import "time"

type IPPool struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Network      string    `json:"network"`     // e.g. "10.0.0.0/24"
	Gateway      string    `json:"gateway"`     // e.g. "10.0.0.1"
	DNSPrimary   string    `json:"dns_primary"` // e.g. "8.8.8.8"
	DNSSecondary string    `json:"dns_secondary"`
	RouterID     *string   `json:"router_id,omitempty"`
	TotalIPs     int       `json:"total_ips"`
	UsedIPs      int       `json:"used_ips"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type IPAddress struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	PoolID       string     `json:"pool_id"`
	Address      string     `json:"address"` // e.g. "10.0.0.5"
	CustomerID   *string    `json:"customer_id,omitempty"`
	CustomerName string     `json:"customer_name,omitempty"`
	Status       string     `json:"status"` // available, assigned, reserved, disabled
	Notes        string     `json:"notes,omitempty"`
	AssignedAt   *time.Time `json:"assigned_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
