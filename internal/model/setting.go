package model

type Setting struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}
