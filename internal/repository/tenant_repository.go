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
	// ListExpiringSoon returns tenants whose plan expires within maxDays days (including already-expired up to 3 days).
	ListExpiringSoon(ctx context.Context, maxDays int) ([]model.Tenant, error)
	// Anti-cheating checks
	CountByFingerprint(ctx context.Context, fingerprint string) (int, error)
	CountByIP(ctx context.Context, ip string) (int, error)
	Approve(ctx context.Context, tenantID string) error
	Delete(ctx context.Context, tenantID string) error
	StampSubscriptionOrders(ctx context.Context, tenantID, tenantName, tenantSlug string) error
}

type TenantFilter struct {
	Search          string
	Plan            string
	Active          *bool
	ExcludeTenantID string
	Page            int
	PerPage         int
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
	if tenant.Status == "" {
		tenant.Status = "active"
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, email, phone, address, logo_url,
			timezone, currency, billing_cycle, due_day, isolir_day, grace_period, default_billing_type,
			plan, plan_expires_at, max_customers, is_active, status, fingerprint, registration_ip, risk_score, risk_reasons,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`

	_, err := r.db.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Email, tenant.Phone,
		tenant.Address, tenant.LogoURL,
		tenant.Timezone, tenant.Currency, tenant.BillingCycle,
		tenant.DueDay, tenant.IsolirDay, tenant.GracePeriod, tenant.DefaultBillingType,
		tenant.Plan, tenant.PlanExpiresAt, tenant.MaxCustomers,
		tenant.IsActive, tenant.Status, tenant.Fingerprint, tenant.RegistrationIP, tenant.RiskScore, tenant.RiskReasons,
		tenant.CreatedAt, tenant.UpdatedAt,
	)
	return err
}

func (r *tenantRepository) FindByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	query := `
		SELECT id, name, slug, email, COALESCE(phone,''), COALESCE(address,''), COALESCE(logo_url,''),
		       timezone, currency, billing_cycle, due_day, isolir_day, grace_period,
		       COALESCE(default_billing_type,'fixed'), COALESCE(default_payment_timing,'due_date'),
		       plan, plan_expires_at, max_customers,
		       COALESCE(wa_api_key,''), COALESCE(wa_sender,''), COALESCE(pg_provider,''), COALESCE(pg_api_key,''), COALESCE(pg_secret_key,''), COALESCE(pg_merchant_id,''),
		       COALESCE(pg_sandbox, true),
		       is_active, COALESCE(status, 'active'), COALESCE(fingerprint, ''), COALESCE(registration_ip, ''),
		       risk_score, COALESCE(risk_reasons, ''),
		       created_at, updated_at
		FROM tenants
		WHERE id = $1
		LIMIT 1
	`

	var t model.Tenant
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
		&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay, &t.GracePeriod,
		&t.DefaultBillingType, &t.DefaultPaymentTiming,
		&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
		&t.WAAPIKey, &t.WASender, &t.PGProvider, &t.PGAPIKey, &t.PGSecretKey, &t.PGMerchantID,
		&t.PGSandbox,
		&t.IsActive, &t.Status, &t.Fingerprint, &t.RegistrationIP, &t.RiskScore, &t.RiskReasons,
		&t.CreatedAt, &t.UpdatedAt,
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
		       plan, plan_expires_at, max_customers, is_active, COALESCE(status, 'active'), created_at, updated_at
		FROM tenants
		WHERE slug = $1
		LIMIT 1
	`

	var t model.Tenant
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
		&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay, &t.GracePeriod, &t.DefaultBillingType,
		&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
		&t.IsActive, &t.Status, &t.CreatedAt, &t.UpdatedAt,
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
			default_billing_type = $12, default_payment_timing = $13,
			is_active = $14, status = $15, updated_at = $16
		WHERE id = $17
	`

	_, err := r.db.Exec(ctx, query,
		tenant.Name, tenant.Email, tenant.Phone, tenant.Address, tenant.LogoURL,
		tenant.Timezone, tenant.Currency, tenant.BillingCycle,
		tenant.DueDay, tenant.IsolirDay, tenant.GracePeriod,
		tenant.DefaultBillingType, tenant.DefaultPaymentTiming,
		tenant.IsActive, tenant.Status, tenant.UpdatedAt,
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
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR slug ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
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

	if filter.ExcludeTenantID != "" {
		conditions = append(conditions, fmt.Sprintf("id != $%d", argIdx))
		args = append(args, filter.ExcludeTenantID)
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
		       plan, plan_expires_at, max_customers, is_active, COALESCE(status, 'active'), risk_score, created_at, updated_at
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
			&t.IsActive, &t.Status, &t.RiskScore, &t.CreatedAt, &t.UpdatedAt,
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

func (r *tenantRepository) ListExpiringSoon(ctx context.Context, maxDays int) ([]model.Tenant, error) {
	query := `
		SELECT id, name, slug, email, COALESCE(phone,''), COALESCE(address,''), COALESCE(logo_url,''),
		       timezone, currency, billing_cycle, due_day, isolir_day, grace_period, COALESCE(default_billing_type,'fixed'),
		       plan, plan_expires_at, max_customers,
		       COALESCE(wa_api_key,''), COALESCE(wa_sender,''), COALESCE(pg_provider,''), COALESCE(pg_api_key,''), COALESCE(pg_secret_key,''), COALESCE(pg_merchant_id,''),
		       COALESCE(pg_sandbox, true),
		       is_active, COALESCE(status, 'active'), created_at, updated_at
		FROM tenants
		WHERE plan_expires_at IS NOT NULL
		  AND plan NOT IN ('gratis', 'free', 'trial', '')
		  AND is_active = TRUE
		  AND plan_expires_at BETWEEN NOW() - INTERVAL '1 day' AND NOW() + INTERVAL '8 days'
		ORDER BY plan_expires_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.Email, &t.Phone, &t.Address, &t.LogoURL,
			&t.Timezone, &t.Currency, &t.BillingCycle, &t.DueDay, &t.IsolirDay,
			&t.GracePeriod, &t.DefaultBillingType,
			&t.Plan, &t.PlanExpiresAt, &t.MaxCustomers,
			&t.WAAPIKey, &t.WASender, &t.PGProvider, &t.PGAPIKey, &t.PGSecretKey, &t.PGMerchantID,
			&t.PGSandbox,
			&t.IsActive, &t.Status, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (r *tenantRepository) CountByFingerprint(ctx context.Context, fingerprint string) (int, error) {
	if fingerprint == "" {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE fingerprint = $1", fingerprint).Scan(&count)
	return count, err
}

func (r *tenantRepository) CountByIP(ctx context.Context, ip string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE registration_ip = $1", ip).Scan(&count)
	return count, err
}

func (r *tenantRepository) Approve(ctx context.Context, tenantID string) error {
	_, err := r.db.Exec(ctx, "UPDATE tenants SET status = 'active', is_active = true, updated_at = NOW() WHERE id = $1", tenantID)
	return err
}

func (r *tenantRepository) Delete(ctx context.Context, tenantID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
	return err
}

func (r *tenantRepository) StampSubscriptionOrders(ctx context.Context, tenantID, tenantName, tenantSlug string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE subscription_orders SET tenant_name = $1, tenant_slug = $2 WHERE tenant_id = $3 AND tenant_name IS NULL`,
		tenantName, tenantSlug, tenantID,
	)
	return err
}
