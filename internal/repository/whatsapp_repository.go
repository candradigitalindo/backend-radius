package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type WhatsAppConfig struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	APIURL       string    `json:"api_url"`
	APIKey       string    `json:"api_key"`
	SenderNumber string    `json:"sender_number"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type WhatsAppLog struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Phone     string    `json:"phone"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WhatsAppRepository interface {
	GetConfig(ctx context.Context, tenantID string) (*WhatsAppConfig, error)
	UpsertConfig(ctx context.Context, cfg *WhatsAppConfig) error
	ListLogs(ctx context.Context, tenantID string, page, perPage int) ([]WhatsAppLog, int, error)
	CreateLog(ctx context.Context, log *WhatsAppLog) error
}

type whatsAppRepository struct {
	db *pgxpool.Pool
}

func NewWhatsAppRepository(db *pgxpool.Pool) WhatsAppRepository {
	return &whatsAppRepository{db: db}
}

func (r *whatsAppRepository) GetConfig(ctx context.Context, tenantID string) (*WhatsAppConfig, error) {
	var c WhatsAppConfig
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, api_url, api_key, sender_number, is_active, created_at, updated_at
		FROM whatsapp_configs WHERE tenant_id = $1
	`, tenantID).Scan(&c.ID, &c.TenantID, &c.APIURL, &c.APIKey, &c.SenderNumber, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &WhatsAppConfig{TenantID: tenantID}, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *whatsAppRepository) UpsertConfig(ctx context.Context, cfg *WhatsAppConfig) error {
	cfg.UpdatedAt = time.Now()
	if cfg.ID == "" {
		cfg.ID = id.New()
		cfg.CreatedAt = cfg.UpdatedAt
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO whatsapp_configs (id, tenant_id, api_url, api_key, sender_number, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id) DO UPDATE SET
			api_url = EXCLUDED.api_url,
			api_key = EXCLUDED.api_key,
			sender_number = EXCLUDED.sender_number,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`, cfg.ID, cfg.TenantID, cfg.APIURL, cfg.APIKey, cfg.SenderNumber, cfg.IsActive, cfg.CreatedAt, cfg.UpdatedAt)
	return err
}

func (r *whatsAppRepository) ListLogs(ctx context.Context, tenantID string, page, perPage int) ([]WhatsAppLog, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM whatsapp_logs WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, phone, message, status, error_msg, created_at
		FROM whatsapp_logs WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []WhatsAppLog
	for rows.Next() {
		var l WhatsAppLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.Phone, &l.Message, &l.Status, &l.ErrorMsg, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (r *whatsAppRepository) CreateLog(ctx context.Context, log *WhatsAppLog) error {
	if log.ID == "" {
		log.ID = id.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO whatsapp_logs (id, tenant_id, phone, message, status, error_msg, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.ID, log.TenantID, log.Phone, log.Message, log.Status, log.ErrorMsg, log.CreatedAt)
	return err
}
