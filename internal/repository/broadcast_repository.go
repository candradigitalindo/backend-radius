package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type BroadcastRepository interface {
	Create(ctx context.Context, broadcast *model.Broadcast) error
	FindByID(ctx context.Context, tenantID, broadcastID string) (*model.Broadcast, error)
	UpdateStats(ctx context.Context, broadcast *model.Broadcast) error
	UpdateProgress(ctx context.Context, broadcast *model.Broadcast) error
	ListPending(ctx context.Context) ([]model.Broadcast, error)
	Delete(ctx context.Context, tenantID, broadcastID string) error
	List(ctx context.Context, tenantID string, filter BroadcastFilter) ([]model.Broadcast, int, error)
}

type BroadcastFilter struct {
	Type    string
	Search  string
	Page    int
	PerPage int
}

type broadcastRepository struct {
	db *pgxpool.Pool
}

func NewBroadcastRepository(db *pgxpool.Pool) BroadcastRepository {
	return &broadcastRepository{db: db}
}

func (r *broadcastRepository) Create(ctx context.Context, broadcast *model.Broadcast) error {
	broadcast.ID = id.New()
	if broadcast.Status == "" {
		broadcast.Status = "sending"
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO broadcasts (id, tenant_id, type, title, message, image_url, target, status, total_sent, total_success, total_failed, total_pending, pending_phones, sent_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, broadcast.ID, broadcast.TenantID, broadcast.Type, broadcast.Title, broadcast.Message,
		broadcast.ImageURL, broadcast.Target, broadcast.Status, broadcast.TotalSent, broadcast.TotalSuccess, broadcast.TotalFailed, broadcast.TotalPending, broadcast.PendingPhones, broadcast.SentBy)
	return err
}

func (r *broadcastRepository) FindByID(ctx context.Context, tenantID, broadcastID string) (*model.Broadcast, error) {
	var b model.Broadcast
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, type, title, message, COALESCE(image_url,''), target,
		       COALESCE(status,'completed'), total_sent, total_success, total_failed,
		       COALESCE(total_pending,0), COALESCE(pending_phones,''), sent_by, created_at, COALESCE(updated_at, created_at)
		FROM broadcasts
		WHERE id = $1 AND tenant_id = $2
	`, broadcastID, tenantID).Scan(
		&b.ID, &b.TenantID, &b.Type, &b.Title, &b.Message, &b.ImageURL, &b.Target,
		&b.Status, &b.TotalSent, &b.TotalSuccess, &b.TotalFailed,
		&b.TotalPending, &b.PendingPhones, &b.SentBy, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *broadcastRepository) UpdateStats(ctx context.Context, broadcast *model.Broadcast) error {
	_, err := r.db.Exec(ctx, `
		UPDATE broadcasts SET total_sent = $1, total_success = $2, total_failed = $3,
		       status = $4, total_pending = $5, pending_phones = $6, updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
	`, broadcast.TotalSent, broadcast.TotalSuccess, broadcast.TotalFailed,
		broadcast.Status, broadcast.TotalPending, broadcast.PendingPhones, broadcast.ID, broadcast.TenantID)
	return err
}

func (r *broadcastRepository) UpdateProgress(ctx context.Context, broadcast *model.Broadcast) error {
	_, err := r.db.Exec(ctx, `
		UPDATE broadcasts SET total_success = total_success + $1, total_failed = total_failed + $2,
		       total_pending = $3, pending_phones = $4, status = $5, updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
	`, broadcast.TotalSuccess, broadcast.TotalFailed,
		broadcast.TotalPending, broadcast.PendingPhones, broadcast.Status, broadcast.ID, broadcast.TenantID)
	return err
}

func (r *broadcastRepository) ListPending(ctx context.Context) ([]model.Broadcast, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, type, title, message, COALESCE(image_url,''), target,
		       COALESCE(status,'completed'), total_sent, total_success, total_failed,
		       COALESCE(total_pending,0), COALESCE(pending_phones,''), sent_by, created_at, COALESCE(updated_at, created_at)
		FROM broadcasts
		WHERE status = 'pending' AND total_pending > 0
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var broadcasts []model.Broadcast
	for rows.Next() {
		var b model.Broadcast
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.Type, &b.Title, &b.Message, &b.ImageURL, &b.Target,
			&b.Status, &b.TotalSent, &b.TotalSuccess, &b.TotalFailed,
			&b.TotalPending, &b.PendingPhones, &b.SentBy, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		broadcasts = append(broadcasts, b)
	}
	return broadcasts, nil
}

func (r *broadcastRepository) Delete(ctx context.Context, tenantID, broadcastID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM broadcasts WHERE id = $1 AND tenant_id = $2`, broadcastID, tenantID)
	return err
}

func (r *broadcastRepository) List(ctx context.Context, tenantID string, filter BroadcastFilter) ([]model.Broadcast, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR message ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM broadcasts "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	query := fmt.Sprintf(`
		SELECT id, tenant_id, type, title, message, COALESCE(image_url,''), target,
		       COALESCE(status,'completed'), total_sent, total_success, total_failed,
		       COALESCE(total_pending,0), COALESCE(pending_phones,''), sent_by, created_at, COALESCE(updated_at, created_at)
		FROM broadcasts %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var broadcasts []model.Broadcast
	for rows.Next() {
		var b model.Broadcast
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.Type, &b.Title, &b.Message, &b.ImageURL, &b.Target,
			&b.Status, &b.TotalSent, &b.TotalSuccess, &b.TotalFailed,
			&b.TotalPending, &b.PendingPhones, &b.SentBy, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		broadcasts = append(broadcasts, b)
	}

	return broadcasts, total, nil
}
