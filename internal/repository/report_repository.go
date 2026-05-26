package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository interface {
	GetRevenueReport(ctx context.Context, tenantID string, year int) ([]MonthlyRevenueReport, error)
	GetCustomerGrowth(ctx context.Context, tenantID string, year int) ([]MonthlyCustomerGrowth, error)
	GetPaymentMethodBreakdown(ctx context.Context, tenantID string, month, year int) ([]PaymentMethodStat, error)
	GetCollectionRate(ctx context.Context, tenantID string, month, year int) (*CollectionRateStat, error)
	GetProfitLoss(ctx context.Context, tenantID string, month, year int) (*ProfitLossStat, error)
	GetVoucherSalesReport(ctx context.Context, tenantID string, month, year int) (*VoucherSalesStat, error)
}

type MonthlyRevenueReport struct {
	Month         int   `json:"month"`
	Revenue       int64 `json:"revenue"`
	Expenses      int64 `json:"expenses"`
	Profit        int64 `json:"profit"`
	InvoicesPaid  int   `json:"invoices_paid"`
	InvoicesTotal int   `json:"invoices_total"`
}

type MonthlyCustomerGrowth struct {
	Month       int `json:"month"`
	NewJoined   int `json:"new_joined"`
	TotalActive int `json:"total_active"`
	TotalAll    int `json:"total_all"`
	Churned     int `json:"churned"`
}

type PaymentMethodStat struct {
	Method string `json:"method"`
	Count  int    `json:"count"`
	Amount int64  `json:"amount"`
}

type CollectionRateStat struct {
	TotalInvoices  int     `json:"total_invoices"`
	PaidInvoices   int     `json:"paid_invoices"`
	PaidOnTime     int     `json:"paid_on_time"`
	PaidLate       int     `json:"paid_late"`
	Unpaid         int     `json:"unpaid"`
	CollectionRate float64 `json:"collection_rate"`
	OnTimeRate     float64 `json:"on_time_rate"`
	TotalBilled    int64   `json:"total_billed"`
	TotalCollected int64   `json:"total_collected"`
}

type ProfitLossStat struct {
	Revenue      int64 `json:"revenue"`
	Expenses     int64 `json:"expenses"`
	Profit       int64 `json:"profit"`
	VoucherSales int64 `json:"voucher_sales"`
	GrandTotal   int64 `json:"grand_total"`
}

type VoucherSalesStat struct {
	TotalSold   int                 `json:"total_sold"`
	TotalAmount int64               `json:"total_amount"`
	ByGateway   []PaymentMethodStat `json:"by_gateway"`
}

type reportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetRevenueReport(ctx context.Context, tenantID string, year int) ([]MonthlyRevenueReport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.month,
		       COALESCE(i.revenue, 0) AS revenue,
		       COALESCE(e.expenses, 0) AS expenses,
		       COALESCE(i.revenue, 0) - COALESCE(e.expenses, 0) AS profit,
		       COALESCE(i.paid_count, 0) AS invoices_paid,
		       COALESCE(i.total_count, 0) AS invoices_total
		FROM generate_series(1, 12) AS m(month)
		LEFT JOIN (
			SELECT period_month,
			       SUM(paid_amount) FILTER (WHERE status = 'paid') AS revenue,
			       COUNT(*) FILTER (WHERE status = 'paid') AS paid_count,
			       COUNT(*) AS total_count
			FROM invoices
			WHERE tenant_id = $1 AND period_year = $2
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

	var result []MonthlyRevenueReport
	for rows.Next() {
		var mr MonthlyRevenueReport
		if err := rows.Scan(&mr.Month, &mr.Revenue, &mr.Expenses, &mr.Profit, &mr.InvoicesPaid, &mr.InvoicesTotal); err != nil {
			return nil, err
		}
		result = append(result, mr)
	}
	return result, nil
}

func (r *reportRepository) GetCustomerGrowth(ctx context.Context, tenantID string, year int) ([]MonthlyCustomerGrowth, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.month,
		       COALESCE(j.new_joined, 0) AS new_joined,
		       (SELECT COUNT(*) FROM customers
		        WHERE tenant_id = $1 AND status = 'active'
		          AND EXTRACT(YEAR FROM join_date) * 12 + EXTRACT(MONTH FROM join_date) <= $2 * 12 + m.month
		       ) AS total_active,
		       (SELECT COUNT(*) FROM customers
		        WHERE tenant_id = $1
		          AND EXTRACT(YEAR FROM join_date) * 12 + EXTRACT(MONTH FROM join_date) <= $2 * 12 + m.month
		       ) AS total_all,
		       COALESCE(c.churned, 0) AS churned
		FROM generate_series(1, 12) AS m(month)
		LEFT JOIN (
			SELECT EXTRACT(MONTH FROM join_date)::int AS month, COUNT(*) AS new_joined
			FROM customers
			WHERE tenant_id = $1 AND EXTRACT(YEAR FROM join_date) = $2
			GROUP BY 1
		) j ON j.month = m.month
		LEFT JOIN (
			SELECT EXTRACT(MONTH FROM isolated_at)::int AS month, COUNT(*) AS churned
			FROM customers
			WHERE tenant_id = $1 AND isolated_at IS NOT NULL
			  AND EXTRACT(YEAR FROM isolated_at) = $2
			GROUP BY 1
		) c ON c.month = m.month
		ORDER BY m.month
	`, tenantID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthlyCustomerGrowth
	for rows.Next() {
		var mg MonthlyCustomerGrowth
		if err := rows.Scan(&mg.Month, &mg.NewJoined, &mg.TotalActive, &mg.TotalAll, &mg.Churned); err != nil {
			return nil, err
		}
		result = append(result, mg)
	}
	return result, nil
}

func (r *reportRepository) GetPaymentMethodBreakdown(ctx context.Context, tenantID string, month, year int) ([]PaymentMethodStat, error) {
	query := `
		SELECT COALESCE(NULLIF(payment_method, ''), 'unknown') AS method,
		       COUNT(*) AS count,
		       COALESCE(SUM(amount), 0) AS amount
		FROM payments
		WHERE tenant_id = $1 AND status = 'paid'
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if month > 0 && year > 0 {
		query += fmt.Sprintf(" AND EXTRACT(MONTH FROM paid_at) = $%d AND EXTRACT(YEAR FROM paid_at) = $%d", argIdx, argIdx+1)
		args = append(args, month, year)
	}

	query += " GROUP BY method ORDER BY amount DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PaymentMethodStat
	for rows.Next() {
		var ps PaymentMethodStat
		if err := rows.Scan(&ps.Method, &ps.Count, &ps.Amount); err != nil {
			return nil, err
		}
		result = append(result, ps)
	}
	return result, nil
}

func (r *reportRepository) GetCollectionRate(ctx context.Context, tenantID string, month, year int) (*CollectionRateStat, error) {
	var s CollectionRateStat

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_invoices,
			COUNT(*) FILTER (WHERE status = 'paid') AS paid_invoices,
			COUNT(*) FILTER (WHERE status = 'paid' AND paid_at <= due_date) AS paid_on_time,
			COUNT(*) FILTER (WHERE status = 'paid' AND paid_at > due_date) AS paid_late,
			COUNT(*) FILTER (WHERE status IN ('unpaid', 'overdue')) AS unpaid,
			COALESCE(SUM(total_amount), 0) AS total_billed,
			COALESCE(SUM(paid_amount) FILTER (WHERE status = 'paid'), 0) AS total_collected
		FROM invoices
		WHERE tenant_id = $1 AND period_month = $2 AND period_year = $3
	`, tenantID, month, year).Scan(
		&s.TotalInvoices, &s.PaidInvoices, &s.PaidOnTime, &s.PaidLate,
		&s.Unpaid, &s.TotalBilled, &s.TotalCollected,
	)
	if err != nil {
		return nil, err
	}

	if s.TotalInvoices > 0 {
		s.CollectionRate = float64(s.PaidInvoices) / float64(s.TotalInvoices) * 100
	}
	if s.PaidInvoices > 0 {
		s.OnTimeRate = float64(s.PaidOnTime) / float64(s.PaidInvoices) * 100
	}

	return &s, nil
}

func (r *reportRepository) GetProfitLoss(ctx context.Context, tenantID string, month, year int) (*ProfitLossStat, error) {
	var s ProfitLossStat

	// Invoice revenue
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(paid_amount), 0) FROM invoices
		WHERE tenant_id = $1 AND period_month = $2 AND period_year = $3 AND status = 'paid'
	`, tenantID, month, year).Scan(&s.Revenue)
	if err != nil {
		return nil, err
	}

	// Expenses
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM expenses
		WHERE tenant_id = $1
		  AND EXTRACT(MONTH FROM expense_date) = $2
		  AND EXTRACT(YEAR FROM expense_date) = $3
	`, tenantID, month, year).Scan(&s.Expenses)
	if err != nil {
		return nil, err
	}

	// Voucher sales
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM voucher_payments
		WHERE tenant_id = $1 AND status = 'paid'
		  AND EXTRACT(MONTH FROM paid_at) = $2
		  AND EXTRACT(YEAR FROM paid_at) = $3
	`, tenantID, month, year).Scan(&s.VoucherSales)
	if err != nil {
		return nil, err
	}

	s.Profit = s.Revenue - s.Expenses
	s.GrandTotal = s.Revenue + s.VoucherSales - s.Expenses

	return &s, nil
}

func (r *reportRepository) GetVoucherSalesReport(ctx context.Context, tenantID string, month, year int) (*VoucherSalesStat, error) {
	var s VoucherSalesStat

	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM voucher_payments
		WHERE tenant_id = $1 AND status = 'paid'
		  AND EXTRACT(MONTH FROM paid_at) = $2
		  AND EXTRACT(YEAR FROM paid_at) = $3
	`, tenantID, month, year).Scan(&s.TotalSold, &s.TotalAmount)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(gateway, 'unknown') AS method,
		       COUNT(*) AS count,
		       COALESCE(SUM(amount), 0) AS amount
		FROM voucher_payments
		WHERE tenant_id = $1 AND status = 'paid'
		  AND EXTRACT(MONTH FROM paid_at) = $2
		  AND EXTRACT(YEAR FROM paid_at) = $3
		GROUP BY gateway
		ORDER BY amount DESC
	`, tenantID, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ps PaymentMethodStat
		if err := rows.Scan(&ps.Method, &ps.Count, &ps.Amount); err != nil {
			return nil, err
		}
		s.ByGateway = append(s.ByGateway, ps)
	}

	return &s, nil
}
