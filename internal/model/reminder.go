package model

import "time"

type Reminder struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	DaysOffset      int       `json:"days_offset"`
	MessageTemplate string    `json:"message_template"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
