package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type ResellerRepository interface {
	Create(ctx context.Context, reseller *model.Reseller) error
	GetByID(ctx context.Context, tenantID, resellerID string) (*model.Reseller, error)
	Update(ctx context.Context, reseller *model.Reseller) error
	Delete(ctx context.Context, tenantID, resellerID string) error
	List(ctx context.Context, tenantID string, filter ResellerFilter) ([]model.Reseller, int, error)
	ListCustomers(ctx context.Context, tenantID, resellerID string) ([]model.Customer, error)
	AddCommission(ctx context.Context, comm *model.ResellerCommission) error
	ListCommissions(ctx context.Context, tenantID, resellerID string, page, perPage int) ([]model.ResellerCommission, int, error)
	PayCommission(ctx context.Context, tenantID, commissionID string) error
	PayAllPending(ctx context.Context, tenantID, resellerID string) (int, error)
	GetCommissionSummary(ctx context.Context, tenantID, resellerID string) (*CommissionSummary, error)
}

type ResellerFilter struct {
	Status  string
	Search  string
	Page    int
	PerPage int
}

type CommissionSummary struct {
	TotalEarned   int64 `json:"total_earned"`
	TotalPaid     int64 `json:"total_paid"`
	TotalPending  int64 `json:"total_pending"`
	CustomerCount int   `json:"customer_count"`
}

type resellerRepository struct {
	db *pgxpool.Pool
}

func NewResellerRepository(db *pgxpool.Pool) ResellerRepository {
	return &resellerRepository{db: db}
}

func (r *resellerRepository) Create(ctx context.Context, reseller *model.Reseller) error {
	reseller.ID = id.New()
	now := time.Now()
	reseller.CreatedAt = now
	reseller.UpdatedAt = now
	if reseller.Status == "" {
		reseller.Status = "active"
	}

	query := `
		INSERT INTO resellers (id, tenant_id, name, email, phone, company_name, address, commission_rate, balance, status, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		reseller.ID, reseller.TenantID, reseller.Name, reseller.Email, reseller.Phone,
		reseller.CompanyName, reseller.Address, reseller.CommissionRate, reseller.Balance, reseller.Status,
		reseller.Notes, reseller.CreatedAt, reseller.UpdatedAt,
	)
	return err
}

func (r *resellerRepository) GetByID(ctx context.Context, tenantID, resellerID string) (*model.Reseller, error) {
	query := `
		SELECT id, tenant_id, name, email, phone, company_name, address, commission_rate, balance, status, notes, created_at, updated_at
		FROM resellers WHERE id = $1 AND tenant_id = $2
	`
	var res model.Reseller
	err := r.db.QueryRow(ctx, query, resellerID, tenantID).Scan(
		&res.ID, &res.TenantID, &res.Name, &res.Email, &res.Phone,
		&res.CompanyName, &res.Address, &res.CommissionRate, &res.Balance, &res.Status,
		&res.Notes, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *resellerRepository) Update(ctx context.Context, reseller *model.Reseller) error {
	reseller.UpdatedAt = time.Now()
	query := `
		UPDATE resellers SET name = $1, email = $2, phone = $3, company_name = $4, address = $5,
		       commission_rate = $6, status = $7, notes = $8, updated_at = $9
		WHERE id = $10 AND tenant_id = $11
	`
	_, err := r.db.Exec(ctx, query,
		reseller.Name, reseller.Email, reseller.Phone, reseller.CompanyName, reseller.Address,
		reseller.CommissionRate, reseller.Status, reseller.Notes, reseller.UpdatedAt,
		reseller.ID, reseller.TenantID,
	)
	return err
}

func (r *resellerRepository) Delete(ctx context.Context, tenantID, resellerID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM resellers WHERE id = $1 AND tenant_id = $2`, resellerID, tenantID)
	return err
}

func (r *resellerRepository) List(ctx context.Context, tenantID string, filter ResellerFilter) ([]model.Reseller, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM resellers "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, email, phone, company_name, address, commission_rate, balance, status, notes, created_at, updated_at
		FROM resellers %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var resellers []model.Reseller
	for rows.Next() {
		var res model.Reseller
		if err := rows.Scan(
			&res.ID, &res.TenantID, &res.Name, &res.Email, &res.Phone,
			&res.CompanyName, &res.Address, &res.CommissionRate, &res.Balance, &res.Status,
			&res.Notes, &res.CreatedAt, &res.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		resellers = append(resellers, res)
	}
	return resellers, total, nil
}

func (r *resellerRepository) AddCommission(ctx context.Context, comm *model.ResellerCommission) error {
	comm.ID = id.New()
	comm.CreatedAt = time.Now()
	if comm.Status == "" {
		comm.Status = "pending"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO reseller_commissions (id, tenant_id, reseller_id, invoice_id, customer_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if _, err := tx.Exec(ctx, query,
		comm.ID, comm.TenantID, comm.ResellerID, comm.InvoiceID, comm.CustomerID,
		comm.Amount, comm.Status, comm.CreatedAt,
	); err != nil {
		return err
	}

	// Update reseller balance
	if _, err := tx.Exec(ctx,
		`UPDATE resellers SET balance = balance + $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		comm.Amount, comm.ResellerID, comm.TenantID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *resellerRepository) ListCommissions(ctx context.Context, tenantID, resellerID string, page, perPage int) ([]model.ResellerCommission, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM reseller_commissions WHERE tenant_id = $1 AND reseller_id = $2`,
		tenantID, resellerID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT rc.id, rc.tenant_id, rc.reseller_id, rc.invoice_id, rc.customer_id,
		       COALESCE(c.name, '') as customer_name,
		       rc.amount, rc.status, rc.paid_at, rc.created_at
		FROM reseller_commissions rc
		LEFT JOIN customers c ON c.id = rc.customer_id
		WHERE rc.tenant_id = $1 AND rc.reseller_id = $2
		ORDER BY rc.created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, tenantID, resellerID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comms []model.ResellerCommission
	for rows.Next() {
		var c model.ResellerCommission
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.ResellerID, &c.InvoiceID, &c.CustomerID,
			&c.CustomerName, &c.Amount, &c.Status, &c.PaidAt, &c.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		comms = append(comms, c)
	}
	return comms, total, nil
}

func (r *resellerRepository) PayCommission(ctx context.Context, tenantID, commissionID string) error {
	now := time.Now()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get commission details
	var amount int64
	var resellerID string
	err = tx.QueryRow(ctx,
		`SELECT reseller_id, amount FROM reseller_commissions WHERE id = $1 AND tenant_id = $2 AND status = 'pending'`,
		commissionID, tenantID,
	).Scan(&resellerID, &amount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("commission not found or already paid")
		}
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE reseller_commissions SET status = 'paid', paid_at = $1 WHERE id = $2 AND tenant_id = $3`,
		now, commissionID, tenantID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE resellers SET balance = balance - $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		amount, now, resellerID, tenantID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *resellerRepository) PayAllPending(ctx context.Context, tenantID, resellerID string) (int, error) {
	now := time.Now()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE reseller_commissions SET status = 'paid', paid_at = $1 WHERE tenant_id = $2 AND reseller_id = $3 AND status = 'pending'`,
		now, tenantID, resellerID,
	)
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE resellers SET balance = 0, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		now, resellerID, tenantID,
	); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func (r *resellerRepository) GetCommissionSummary(ctx context.Context, tenantID, resellerID string) (*CommissionSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(amount), 0) as total_earned,
			COALESCE(SUM(amount) FILTER (WHERE status = 'paid'), 0) as total_paid,
			COALESCE(SUM(amount) FILTER (WHERE status = 'pending'), 0) as total_pending,
			COUNT(DISTINCT customer_id) as customer_count
		FROM reseller_commissions
		WHERE tenant_id = $1 AND reseller_id = $2
	`
	var s CommissionSummary
	err := r.db.QueryRow(ctx, query, tenantID, resellerID).Scan(
		&s.TotalEarned, &s.TotalPaid, &s.TotalPending, &s.CustomerCount,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *resellerRepository) ListCustomers(ctx context.Context, tenantID, resellerID string) ([]model.Customer, error) {
	query := `
		SELECT DISTINCT c.id, c.tenant_id, c.customer_code, c.name, c.phone, c.email, c.address, c.status
		FROM customers c
		INNER JOIN reseller_commissions rc ON rc.customer_id = c.id
		WHERE rc.tenant_id = $1 AND rc.reseller_id = $2
		ORDER BY c.name
	`
	rows, err := r.db.Query(ctx, query, tenantID, resellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var c model.Customer
		if err := rows.Scan(&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.Phone, &c.Email, &c.Address, &c.Status); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}
