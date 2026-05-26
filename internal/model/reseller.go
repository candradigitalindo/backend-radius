package model

import "time"

type Reseller struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	CompanyName    string    `json:"company_name"`
	Address        string    `json:"address,omitempty"`
	CommissionRate float64   `json:"commission_rate"` // percentage e.g. 10.5 = 10.5%
	Balance        int64     `json:"balance"`         // accumulated unpaid commission
	Status         string    `json:"status"`          // active, inactive, suspended
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ResellerCommission struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	ResellerID   string     `json:"reseller_id"`
	InvoiceID    string     `json:"invoice_id"`
	CustomerID   string     `json:"customer_id"`
	CustomerName string     `json:"customer_name,omitempty"`
	Amount       int64      `json:"amount"`
	Status       string     `json:"status"` // pending, paid, cancelled
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
