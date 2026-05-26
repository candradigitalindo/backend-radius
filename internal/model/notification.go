package model

import "time"

type PushDevice struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	CustomerID string    `json:"customer_id"`
	DeviceType string    `json:"device_type"` // android, ios
	FCMToken   string    `json:"fcm_token"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Notification struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	CustomerID string     `json:"customer_id"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Data       string     `json:"data,omitempty"` // JSON extra data
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
