package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	TaskGenerateInvoices  = "billing:generate_invoices"
	TaskAutoIsolir        = "billing:auto_isolir"
	TaskTriggerReminders  = "reminder:trigger"
	TaskProcessPayment    = "payment:process_webhook"
	TaskExpirePayments    = "payment:expire_stale"
	TaskRouterMonitor     = "router:monitor"
	TaskCleanSessions     = "session:clean_stale"
	TaskONTProvisionRetry = "genieacs:provision_retry"
	TaskONTDiscover       = "genieacs:ont_discover"
	TaskONTAutoMatch      = "genieacs:ont_auto_match"
	TaskExpireRewardClaims = "reward:expire_claims"
)

type GenerateInvoicesPayload struct {
	TenantID string `json:"tenant_id"`
	Month    int    `json:"month"`
	Year     int    `json:"year"`
	DueDay   int    `json:"due_day"`
}

type AutoIsolirPayload struct {
	TenantID string `json:"tenant_id"`
}

type TriggerRemindersPayload struct {
	TenantID string `json:"tenant_id"`
}

type ProcessPaymentPayload struct {
	Gateway      string `json:"gateway"`
	GatewayTrxID string `json:"gateway_trx_id"`
	Status       string `json:"status"`
	RawBody      []byte `json:"raw_body"`
}

type ExpirePaymentsPayload struct{}

type RouterMonitorPayload struct {
	TenantID string `json:"tenant_id"`
}

func NewGenerateInvoicesTask(p GenerateInvoicesPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskGenerateInvoices, data), nil
}

func NewAutoIsolirTask(p AutoIsolirPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskAutoIsolir, data), nil
}

func NewTriggerRemindersTask(p TriggerRemindersPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTriggerReminders, data), nil
}

func NewProcessPaymentTask(p ProcessPaymentPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcessPayment, data), nil
}

func NewExpirePaymentsTask() (*asynq.Task, error) {
	return asynq.NewTask(TaskExpirePayments, nil), nil
}

func NewRouterMonitorTask(p RouterMonitorPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskRouterMonitor, data), nil
}

type CleanSessionsPayload struct {
	TenantID string `json:"tenant_id"`
}

func NewCleanSessionsTask(p CleanSessionsPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskCleanSessions, data), nil
}

func NewONTProvisionRetryTask() (*asynq.Task, error) {
	return asynq.NewTask(TaskONTProvisionRetry, nil), nil
}

func NewONTDiscoverTask() (*asynq.Task, error) {
	return asynq.NewTask(TaskONTDiscover, nil), nil
}

func NewONTAutoMatchTask() (*asynq.Task, error) {
	return asynq.NewTask(TaskONTAutoMatch, nil), nil
}
