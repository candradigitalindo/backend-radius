package model

import "time"

type Ticket struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	CustomerID      string               `json:"customer_id"`
	TicketNumber    string               `json:"ticket_number"`
	Subject         string               `json:"subject"`
	Description     string               `json:"description"`
	Category        string               `json:"category"`
	Priority        string               `json:"priority"`
	Status          string               `json:"status"`
	AssignedTo      *string              `json:"assigned_to,omitempty"`
	ResolvedAt      *time.Time           `json:"resolved_at,omitempty"`
	ClosedAt        *time.Time           `json:"closed_at,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`

	Customer        *Customer            `json:"customer,omitempty"`
	Messages        []TicketMessage      `json:"messages"`
}

type TicketMessage struct {
	ID              string               `json:"id"`
	TicketID        string               `json:"ticket_id"`
	SenderType      string               `json:"sender_type"`
	SenderID        string               `json:"sender_id"`
	Message         string               `json:"message"`
	AttachmentURL   string               `json:"attachment_url"`
	CreatedAt       time.Time            `json:"created_at"`
}
