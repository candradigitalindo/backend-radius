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

type CustomerLogRepository interface {
	Create(ctx context.Context, log *model.CustomerLog) error
	FindByID(ctx context.Context, tenantID, logID string) (*model.CustomerLog, error)
	List(ctx context.Context, tenantID string, filter CustomerLogFilter) ([]model.CustomerLog, int, error)
	ListByCustomer(ctx context.Context, tenantID, customerID string, filter CustomerLogFilter) ([]model.CustomerLog, int, error)
}

type CustomerLogFilter struct {
	Action  string
	Search  string
	Page    int
	PerPage int
}

type customerLogRepository struct {
	db *pgxpool.Pool
}

func NewCustomerLogRepository(db *pgxpool.Pool) CustomerLogRepository {
	return &customerLogRepository{db: db}
}

func (r *customerLogRepository) Create(ctx context.Context, log *model.CustomerLog) error {
	log.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO customer_logs (id, tenant_id, customer_id, action, description, metadata, performed_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, log.ID, log.TenantID, log.CustomerID, log.Action, log.Description, log.Metadata, log.PerformedBy)
	return err
}

func (r *customerLogRepository) FindByID(ctx context.Context, tenantID, logID string) (*model.CustomerLog, error) {
	var l model.CustomerLog
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, action, COALESCE(description,''), metadata, performed_by, created_at
		FROM customer_logs
		WHERE id = $1 AND tenant_id = $2
	`, logID, tenantID).Scan(
		&l.ID, &l.TenantID, &l.CustomerID, &l.Action, &l.Description,
		&l.Metadata, &l.PerformedBy, &l.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (r *customerLogRepository) List(ctx context.Context, tenantID string, filter CustomerLogFilter) ([]model.CustomerLog, int, error) {
	return r.listWithConditions(ctx, tenantID, "", filter)
}

func (r *customerLogRepository) ListByCustomer(ctx context.Context, tenantID, customerID string, filter CustomerLogFilter) ([]model.CustomerLog, int, error) {
	return r.listWithConditions(ctx, tenantID, customerID, filter)
}

func (r *customerLogRepository) listWithConditions(ctx context.Context, tenantID, customerID string, filter CustomerLogFilter) ([]model.CustomerLog, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("cl.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if customerID != "" {
		conditions = append(conditions, fmt.Sprintf("cl.customer_id = $%d", argIdx))
		args = append(args, customerID)
		argIdx++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("cl.action = $%d", argIdx))
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("cl.description ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM customer_logs cl "+where, args...).Scan(&total); err != nil {
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
		SELECT cl.id, cl.tenant_id, cl.customer_id, cl.action, COALESCE(cl.description,''),
		       cl.metadata, cl.performed_by, cl.created_at, c.name, c.customer_code
		FROM customer_logs cl
		JOIN customers c ON c.id = cl.customer_id
		%s
		ORDER BY cl.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.CustomerLog
	for rows.Next() {
		var l model.CustomerLog
		var custName, custCode string
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.CustomerID, &l.Action, &l.Description,
			&l.Metadata, &l.PerformedBy, &l.CreatedAt, &custName, &custCode,
		); err != nil {
			return nil, 0, err
		}
		l.Customer = &model.Customer{ID: l.CustomerID, Name: custName, CustomerCode: custCode}
		logs = append(logs, l)
	}

	return logs, total, nil
}
