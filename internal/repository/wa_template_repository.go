package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type WABroadcastTemplate struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	ImageURL  string    `json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WABroadcastTemplateRepository interface {
	Create(ctx context.Context, t *WABroadcastTemplate) error
	Update(ctx context.Context, t *WABroadcastTemplate) error
	FindByID(ctx context.Context, tenantID, templateID string) (*WABroadcastTemplate, error)
	Delete(ctx context.Context, tenantID, templateID string) error
	List(ctx context.Context, tenantID, category string, page, perPage int) ([]WABroadcastTemplate, int, error)
}

type waBroadcastTemplateRepository struct {
	db *pgxpool.Pool
}

func NewWABroadcastTemplateRepository(db *pgxpool.Pool) WABroadcastTemplateRepository {
	return &waBroadcastTemplateRepository{db: db}
}

func (r *waBroadcastTemplateRepository) Create(ctx context.Context, t *WABroadcastTemplate) error {
	t.ID = id.New()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	_, err := r.db.Exec(ctx, `
		INSERT INTO wa_broadcast_templates (id, tenant_id, name, category, message, image_url, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, t.ID, t.TenantID, t.Name, t.Category, t.Message, t.ImageURL, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *waBroadcastTemplateRepository) Update(ctx context.Context, t *WABroadcastTemplate) error {
	t.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE wa_broadcast_templates
		SET name=$1, category=$2, message=$3, image_url=$4, updated_at=$5
		WHERE id=$6 AND tenant_id=$7
	`, t.Name, t.Category, t.Message, t.ImageURL, t.UpdatedAt, t.ID, t.TenantID)
	return err
}

func (r *waBroadcastTemplateRepository) FindByID(ctx context.Context, tenantID, templateID string) (*WABroadcastTemplate, error) {
	var t WABroadcastTemplate
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, category, message, image_url, created_at, updated_at
		FROM wa_broadcast_templates WHERE id=$1 AND tenant_id=$2
	`, templateID, tenantID).Scan(&t.ID, &t.TenantID, &t.Name, &t.Category, &t.Message, &t.ImageURL, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *waBroadcastTemplateRepository) Delete(ctx context.Context, tenantID, templateID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wa_broadcast_templates WHERE id=$1 AND tenant_id=$2`, templateID, tenantID)
	return err
}

func (r *waBroadcastTemplateRepository) List(ctx context.Context, tenantID, category string, page, perPage int) ([]WABroadcastTemplate, int, error) {
	args := []interface{}{tenantID}
	where := "WHERE tenant_id = $1"
	argIdx := 2

	if category != "" {
		where += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM wa_broadcast_templates "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, category, message, image_url, created_at, updated_at
		FROM wa_broadcast_templates %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []WABroadcastTemplate
	for rows.Next() {
		var t WABroadcastTemplate
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Category, &t.Message, &t.ImageURL, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		templates = append(templates, t)
	}
	return templates, total, nil
}
