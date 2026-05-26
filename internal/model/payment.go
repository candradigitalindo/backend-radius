package model

import "time"

type Payment struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	InvoiceID       string               `json:"invoice_id"`
	Amount          int64                `json:"amount"`
	PaymentMethod   string               `json:"payment_method"`
	Gateway         string               `json:"gateway"`
	GatewayTrxID    string               `json:"gateway_trx_id"`
	GatewayStatus   string               `json:"gateway_status"`
	GatewayResponse []byte               `json:"gateway_response"`
	Status          string               `json:"status"`
	PaidAt          *time.Time           `json:"paid_at,omitempty"`
	ExpiredAt       *time.Time           `json:"expired_at,omitempty"`
	CollectedBy     *string              `json:"collected_by,omitempty"`
	Notes           string               `json:"notes"`
	CreatedAt       time.Time            `json:"created_at"`

	Invoice         *Invoice             `json:"invoice,omitempty"`
}
