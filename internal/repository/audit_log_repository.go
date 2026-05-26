package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) ([]model.AuditLog, int, error)
}

type AuditLogFilter struct {
	UserID   string
	Action   string
	Resource string
	Search   string
	Page     int
	PerPage  int
}

type auditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	log.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, user_email, role, tenant_id, action, resource, resource_id, method, path, ip_address, user_agent, status_code, detail, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW())`,
		log.ID, log.UserID, log.UserEmail, log.Role, log.TenantID,
		log.Action, log.Resource, log.ResourceID,
		log.Method, log.Path, log.IPAddress, log.UserAgent,
		log.StatusCode, log.Detail,
	)
	return err
}

func (r *auditLogRepository) List(ctx context.Context, filter AuditLogFilter) ([]model.AuditLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if filter.UserID != "" {
		where += fmt.Sprintf(" AND user_id = $%d", idx)
		args = append(args, filter.UserID)
		idx++
	}
	if filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", idx)
		args = append(args, filter.Action)
		idx++
	}
	if filter.Resource != "" {
		where += fmt.Sprintf(" AND resource = $%d", idx)
		args = append(args, filter.Resource)
		idx++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND (user_email ILIKE $%d OR path ILIKE $%d OR detail ILIKE $%d)", idx, idx, idx)
		args = append(args, "%"+filter.Search+"%")
		idx++
	}

	// Count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Paginate
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	query := fmt.Sprintf(`
		SELECT id, user_id, user_email, role, COALESCE(tenant_id,''), action, resource, COALESCE(resource_id,''),
		       method, path, ip_address, user_agent, status_code, COALESCE(detail,''), created_at
		FROM audit_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.UserEmail, &l.Role, &l.TenantID,
			&l.Action, &l.Resource, &l.ResourceID,
			&l.Method, &l.Path, &l.IPAddress, &l.UserAgent,
			&l.StatusCode, &l.Detail, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
