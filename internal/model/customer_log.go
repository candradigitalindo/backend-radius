package model

import (
	"encoding/json"
	"time"
)

type CustomerLog struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	CustomerID  string          `json:"customer_id"`
	Action      string          `json:"action"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	PerformedBy *string         `json:"performed_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`

	Customer *Customer `json:"customer,omitempty"`
}
