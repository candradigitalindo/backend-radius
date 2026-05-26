package model

import "time"

type AuditLog struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	UserEmail  string    `json:"user_email"`
	Role       string    `json:"role"`
	TenantID   string    `json:"tenant_id,omitempty"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id,omitempty"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	StatusCode int       `json:"status_code"`
	Detail     string    `json:"detail,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
