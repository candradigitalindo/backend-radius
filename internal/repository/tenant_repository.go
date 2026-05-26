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

type TenantRepository interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	FindByID(ctx context.Context, tenantID string) (*model.Tenant, error)
	FindBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) error
	List(ctx context.Context, filter TenantFilter) ([]model.Tenant, int, error)
	UpdateSettings(ctx context.Context, tenant *model.Tenant) error
	UpdatePlan(ctx context.Context, tenant *model.Tenant) error
	// ListActive returns all active tenants. Used by background workers.
	ListActive(ctx context.Context) ([]model.Tenant, error)
}

type TenantFilter struct {
	Search  string
	Plan    string
	Active  *bool
	Page    int
	PerPage int
}

type tenantRepository struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	tenant.ID = id.New()
	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	query := `
		INSERT INTO tenants (
			id, name, slug, email, phone, address, logo_url,
			timezone, currency, billing_cycle, due_day, isolir_day, grace_period, default_billing_type,
			plan, plan_expires_at, max_customers, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := r.db.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Email, tenant.Phone,
		tenant.Address, tenant.LogoURL,
		tenant.Timezone, tenant.Currency, tenant.BillingCycle,
		tenant.DueDay, tenant.IsolirDay, tenant.GracePeriod, tenant.DefaultBillingType,
		tenant.Plan, tenant.PlanExpiresAt, tenant.MaxCustomers,
		tenant.IsActive, tenant.CreatedAt, tenant.UpdatedAt,
	)
	return err
}

func (r *tenantRepository) FindByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	query := `
		SELECT id, name, slug, email, COALESCE(phone,''), COALESCE(address,''), COALESCE(logo_url,''),
		       timezone, currency, billing_cycle, due_day, isolir_day, grace_period, COALESCE(default_billing_type,'fixed'),
		       plan, plan_expires_at, max_customers,
		       COALESCE(wa_api_key,''), COALESCE(wa_sender,''), COALESCE(pg_provider,''), COALESCE(pg_api_key,''), COALESCE(pg_secret_key,''), COALESCE(pg_merchant_id,''),
		       COALESCE(pg_sandbox, true),
		       is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1
		LIMIT 1
	`

	var t model.Tenant
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
		&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay, &t.GracePeriod, &t.DefaultBillingType,
		&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
		&t.WAAPIKey, &t.WASender, &t.PGProvider, &t.PGAPIKey, &t.PGSecretKey, &t.PGMerchantID,
		&t.PGSandbox,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *tenantRepository) FindBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	query := `
		SELECT id, name, slug, email, COALESCE(phone,''), COALESCE(address,''), COALESCE(logo_url,''),
		       timezone, currency, billing_cycle, due_day, isolir_day, grace_period, COALESCE(default_billing_type,'fixed'),
		       plan, plan_expires_at, max_customers, is_active, created_at, updated_at
		FROM tenants
		WHERE slug = $1
		LIMIT 1
	`

	var t model.Tenant
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
		&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay, &t.GracePeriod, &t.DefaultBillingType,
		&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	tenant.UpdatedAt = time.Now()

	query := `
		UPDATE tenants SET
			name = $1, email = $2, phone = $3, address = $4, logo_url = $5,
			timezone = $6, currency = $7, billing_cycle = $8,
			due_day = $9, isolir_day = $10, grace_period = $11,
			default_billing_type = $12,
			is_active = $13, updated_at = $14
		WHERE id = $15
	`

	_, err := r.db.Exec(ctx, query,
		tenant.Name, tenant.Email, tenant.Phone, tenant.Address, tenant.LogoURL,
		tenant.Timezone, tenant.Currency, tenant.BillingCycle,
		tenant.DueDay, tenant.IsolirDay, tenant.GracePeriod,
		tenant.DefaultBillingType,
		tenant.IsActive, tenant.UpdatedAt,
		tenant.ID,
	)
	return err
}

func (r *tenantRepository) UpdateSettings(ctx context.Context, tenant *model.Tenant) error {
	tenant.UpdatedAt = time.Now()

	query := `
		UPDATE tenants SET
			wa_api_key = $1, wa_sender = $2,
			pg_provider = $3, pg_api_key = $4, pg_secret_key = $5, pg_merchant_id = $6,
			pg_sandbox = $7,
			updated_at = $8
		WHERE id = $9
	`

	_, err := r.db.Exec(ctx, query,
		tenant.WAAPIKey, tenant.WASender,
		tenant.PGProvider, tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGMerchantID,
		tenant.PGSandbox,
		tenant.UpdatedAt,
		tenant.ID,
	)
	return err
}

func (r *tenantRepository) List(ctx context.Context, filter TenantFilter) ([]model.Tenant, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR slug ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Plan != "" {
		conditions = append(conditions, fmt.Sprintf("plan = $%d", argIdx))
		args = append(args, filter.Plan)
		argIdx++
	}

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	dataQuery := fmt.Sprintf(`
		SELECT id, name, slug, email, COALESCE(phone,''), COALESCE(address,''), COALESCE(logo_url,''),
		       timezone, currency, billing_cycle, due_day, isolir_day, grace_period, COALESCE(default_billing_type,'fixed'),
		       plan, plan_expires_at, max_customers, is_active, created_at, updated_at
		FROM tenants
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
			&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay, &t.GracePeriod, &t.DefaultBillingType,
			&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
			&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, t)
	}

	return tenants, total, nil
}

func (r *tenantRepository) UpdatePlan(ctx context.Context, tenant *model.Tenant) error {
	tenant.UpdatedAt = time.Now()
	query := `
		UPDATE tenants SET
			plan = $1, plan_expires_at = $2, max_customers = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query,
		tenant.Plan, tenant.PlanExpiresAt, tenant.MaxCustomers, tenant.UpdatedAt,
		tenant.ID,
	)
	return err
}

func (r *tenantRepository) ListActive(ctx context.Context) ([]model.Tenant, error) {
	query := `SELECT id FROM tenants WHERE is_active = true ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}
