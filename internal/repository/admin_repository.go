package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SuperAdmin Dashboard Stats
type AdminDashboardStats struct {
	TotalTenants       int              `json:"total_tenants"`
	ActiveTenants      int              `json:"active_tenants"`
	FreePlanTenants    int              `json:"free_plan_tenants"`
	ProPlanTenants     int              `json:"pro_plan_tenants"`
	EnterpriseTenants  int              `json:"enterprise_tenants"`
	TotalCustomers     int              `json:"total_customers"`
	ActiveCustomers    int              `json:"active_customers"`
	InactiveCustomers  int              `json:"inactive_customers"`
	TotalRouters       int              `json:"total_routers"`
	OnlineRouters      int              `json:"online_routers"`
	TotalRevenue       int64            `json:"total_revenue"`
	SubscriberCount    int              `json:"subscriber_count"`
	TenantStats        []TenantStat     `json:"tenant_stats"`
}

type TenantStat struct {
	TenantID          string `json:"tenant_id"`
	TenantName        string `json:"tenant_name"`
	Slug              string `json:"slug"`
	Plan              string `json:"plan"`
	IsActive          bool   `json:"is_active"`
	TotalCustomers    int    `json:"total_customers"`
	ActiveCustomers   int    `json:"active_customers"`
	InactiveCustomers int    `json:"inactive_customers"`
	TotalRouters      int    `json:"total_routers"`
	OnlineRouters     int    `json:"online_routers"`
}

type TenantRouterStat struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	RouterID   string `json:"router_id"`
	RouterName string `json:"router_name"`
	Host       string `json:"host"`
	IsActive   bool   `json:"is_active"`
	LastSeen   *string `json:"last_seen,omitempty"`
}

type AdminRepository interface {
	GetDashboardStats(ctx context.Context) (*AdminDashboardStats, error)
	GetTenantStats(ctx context.Context) ([]TenantStat, error)
	GetAllRouters(ctx context.Context, page, perPage int) ([]TenantRouterStat, int, error)
	GetTenantCustomerCounts(ctx context.Context) ([]TenantStat, error)
}

type adminRepository struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) GetDashboardStats(ctx context.Context) (*AdminDashboardStats, error) {
	var s AdminDashboardStats

	// Tenant counts
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE is_active = true),
			COUNT(*) FILTER (WHERE plan = 'free' OR plan = '' OR plan IS NULL),
			COUNT(*) FILTER (WHERE plan = 'pro'),
			COUNT(*) FILTER (WHERE plan = 'enterprise')
		FROM tenants
	`).Scan(&s.TotalTenants, &s.ActiveTenants, &s.FreePlanTenants, &s.ProPlanTenants, &s.EnterpriseTenants)
	if err != nil {
		return nil, err
	}

	// Global customer counts
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'active'),
			COUNT(*) FILTER (WHERE status != 'active')
		FROM customers
	`).Scan(&s.TotalCustomers, &s.ActiveCustomers, &s.InactiveCustomers)
	if err != nil {
		return nil, err
	}

	// Global router counts
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE is_active = true)
		FROM routers
	`).Scan(&s.TotalRouters, &s.OnlineRouters)
	if err != nil {
		return nil, err
	}

	// Total revenue (all paid invoices current month)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(paid_amount), 0) FROM invoices
		WHERE status = 'paid'
		  AND period_month = EXTRACT(MONTH FROM NOW())
		  AND period_year = EXTRACT(YEAR FROM NOW())
	`).Scan(&s.TotalRevenue)
	if err != nil {
		return nil, err
	}

	// Active subscribers
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM subscription_orders WHERE status = 'paid'
	`).Scan(&s.SubscriberCount)
	if err != nil {
		return nil, err
	}

	// Per-tenant stats
	s.TenantStats, err = r.GetTenantStats(ctx)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *adminRepository) GetTenantStats(ctx context.Context) ([]TenantStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			t.id, t.name, t.slug, COALESCE(t.plan, 'free'), t.is_active,
			COALESCE(c.total, 0), COALESCE(c.active, 0), COALESCE(c.total - c.active, 0),
			COALESCE(r.total_routers, 0), COALESCE(r.online_routers, 0)
		FROM tenants t
		LEFT JOIN (
			SELECT tenant_id,
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'active') as active
			FROM customers GROUP BY tenant_id
		) c ON c.tenant_id = t.id
		LEFT JOIN (
			SELECT tenant_id,
				COUNT(*) as total_routers,
				COUNT(*) FILTER (WHERE is_active = true) as online_routers
			FROM routers GROUP BY tenant_id
		) r ON r.tenant_id = t.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TenantStat
	for rows.Next() {
		var ts TenantStat
		if err := rows.Scan(&ts.TenantID, &ts.TenantName, &ts.Slug, &ts.Plan, &ts.IsActive,
			&ts.TotalCustomers, &ts.ActiveCustomers, &ts.InactiveCustomers,
			&ts.TotalRouters, &ts.OnlineRouters); err != nil {
			return nil, err
		}
		result = append(result, ts)
	}
	return result, nil
}

func (r *adminRepository) GetAllRouters(ctx context.Context, page, perPage int) ([]TenantRouterStat, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM routers`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, r.id, r.name, r.host, r.is_active, r.updated_at::text
		FROM routers r
		JOIN tenants t ON t.id = r.tenant_id
		ORDER BY r.updated_at DESC
		LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []TenantRouterStat
	for rows.Next() {
		var rs TenantRouterStat
		if err := rows.Scan(&rs.TenantID, &rs.TenantName, &rs.RouterID, &rs.RouterName,
			&rs.Host, &rs.IsActive, &rs.LastSeen); err != nil {
			return nil, 0, err
		}
		result = append(result, rs)
	}
	return result, total, nil
}

func (r *adminRepository) GetTenantCustomerCounts(ctx context.Context) ([]TenantStat, error) {
	return r.GetTenantStats(ctx)
}