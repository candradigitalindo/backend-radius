package model

import "time"

type ExpenseCategory struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
}

type Expense struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	CategoryID  *string   `json:"category_id,omitempty"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	ExpenseDate time.Time `json:"expense_date"`
	ReceiptURL  string    `json:"receipt_url"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`

	Category *ExpenseCategory `json:"category,omitempty"`
}
