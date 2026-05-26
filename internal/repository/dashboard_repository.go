package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardStats struct {
	TotalCustomers    int   `json:"total_customers"`
	ActiveCustomers   int   `json:"active_customers"`
	IsolatedCustomers int   `json:"isolated_customers"`
	OnlineCustomers   int   `json:"online_customers"`
	OfflineCustomers  int   `json:"offline_customers"`
	TotalRevenue      int64 `json:"total_revenue"`
	PaidInvoices      int   `json:"paid_invoices"`
	UnpaidInvoices    int   `json:"unpaid_invoices"`
	TotalExpenses     int64 `json:"total_expenses"`
	ActiveSessions    int   `json:"active_sessions"`
	OpenTickets       int   `json:"open_tickets"`
}

type DashboardRepository interface {
	GetStats(ctx context.Context, tenantID string, periodMonth, periodYear int) (*DashboardStats, error)
	GetMonthlyRevenue(ctx context.Context, tenantID string, year int) ([]MonthlyRevenue, error)
}

type MonthlyRevenue struct {
	Month    int   `json:"month"`
	Revenue  int64 `json:"revenue"`
	Expenses int64 `json:"expenses"`
}

type dashboardRepository struct {
	db *pgxpool.Pool
}

func NewDashboardRepository(db *pgxpool.Pool) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetStats(ctx context.Context, tenantID string, periodMonth, periodYear int) (*DashboardStats, error) {
	var s DashboardStats

	// Customer counts
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'active'),
			COUNT(*) FILTER (WHERE status = 'isolated')
		FROM customers WHERE tenant_id = $1
	`, tenantID).Scan(&s.TotalCustomers, &s.ActiveCustomers, &s.IsolatedCustomers)
	if err != nil {
		return nil, err
	}

	// Invoice stats for period
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(paid_amount) FILTER (WHERE status = 'paid'), 0),
			COUNT(*) FILTER (WHERE status = 'paid'),
			COUNT(*) FILTER (WHERE status IN ('unpaid','overdue'))
		FROM invoices
		WHERE tenant_id = $1 AND period_month = $2 AND period_year = $3
	`, tenantID, periodMonth, periodYear).Scan(&s.TotalRevenue, &s.PaidInvoices, &s.UnpaidInvoices)
	if err != nil {
		return nil, err
	}

	// Total expenses for period
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM expenses
		WHERE tenant_id = $1
		  AND EXTRACT(MONTH FROM expense_date) = $2
		  AND EXTRACT(YEAR FROM expense_date) = $3
	`, tenantID, periodMonth, periodYear).Scan(&s.TotalExpenses)
	if err != nil {
		return nil, err
	}

	// Active RADIUS sessions + online customer count
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT customer_id) FROM radius_sessions
		WHERE tenant_id = $1 AND ended_at IS NULL
	`, tenantID).Scan(&s.ActiveSessions, &s.OnlineCustomers)
	if err != nil {
		return nil, err
	}
	s.OfflineCustomers = s.ActiveCustomers - s.OnlineCustomers
	if s.OfflineCustomers < 0 {
		s.OfflineCustomers = 0
	}

	// Open tickets
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tickets
		WHERE tenant_id = $1 AND status IN ('open','in_progress')
	`, tenantID).Scan(&s.OpenTickets)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *dashboardRepository) GetMonthlyRevenue(ctx context.Context, tenantID string, year int) ([]MonthlyRevenue, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.month,
		       COALESCE(i.revenue, 0) AS revenue,
		       COALESCE(e.expenses, 0) AS expenses
		FROM generate_series(1, 12) AS m(month)
		LEFT JOIN (
			SELECT period_month, SUM(paid_amount) AS revenue
			FROM invoices
			WHERE tenant_id = $1 AND period_year = $2 AND status = 'paid'
			GROUP BY period_month
		) i ON i.period_month = m.month
		LEFT JOIN (
			SELECT EXTRACT(MONTH FROM expense_date)::int AS month, SUM(amount) AS expenses
			FROM expenses
			WHERE tenant_id = $1 AND EXTRACT(YEAR FROM expense_date) = $2
			GROUP BY 1
		) e ON e.month = m.month
		ORDER BY m.month
	`, tenantID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthlyRevenue
	for rows.Next() {
		var mr MonthlyRevenue
		if err := rows.Scan(&mr.Month, &mr.Revenue, &mr.Expenses); err != nil {
			return nil, err
		}
		result = append(result, mr)
	}

	return result, nil
}
