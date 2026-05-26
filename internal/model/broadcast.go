package model

import "time"

type Broadcast struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	ImageURL      string    `json:"image_url"`
	Target        string    `json:"target"`
	Status        string    `json:"status"`
	TotalSent     int       `json:"total_sent"`
	TotalSuccess  int       `json:"total_success"`
	TotalFailed   int       `json:"total_failed"`
	TotalPending  int       `json:"total_pending"`
	PendingPhones string    `json:"pending_phones,omitempty"`
	SentBy        string    `json:"sent_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
