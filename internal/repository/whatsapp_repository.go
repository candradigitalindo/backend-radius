package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

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
	ListLogs(ctx context.Context, tenantID string, page, perPage int) ([]WhatsAppLog, int, error)
	CreateLog(ctx context.Context, log *WhatsAppLog) error
}

type whatsAppRepository struct {
	db *pgxpool.Pool
}

func NewWhatsAppRepository(db *pgxpool.Pool) WhatsAppRepository {
	return &whatsAppRepository{db: db}
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
