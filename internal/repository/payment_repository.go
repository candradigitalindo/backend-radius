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

type PaymentRepository interface {
	Create(ctx context.Context, payment *model.Payment) error
	FindByID(ctx context.Context, tenantID, paymentID string) (*model.Payment, error)
	FindByGatewayTrxID(ctx context.Context, gatewayTrxID string) (*model.Payment, error)
	FindByGatewayTrxIDForTenant(ctx context.Context, tenantID, gatewayTrxID string) (*model.Payment, error)
	UpdateStatus(ctx context.Context, paymentID, status string) error
	Delete(ctx context.Context, tenantID, paymentID string) error
	ListByInvoice(ctx context.Context, tenantID, invoiceID string) ([]model.Payment, error)
	ListExpiredPending(ctx context.Context) ([]model.Payment, error)
	List(ctx context.Context, tenantID string, filter PaymentFilter) ([]model.Payment, int, error)
	// FindActivePaymentURL returns the payment URL of the latest non-expired pending payment
	// for the given invoice, extracting it from the JSONB gateway_response column.
	// Returns an empty string if no active payment with a URL exists.
	FindActivePaymentURL(ctx context.Context, tenantID, invoiceID string) (string, error)
}

type PaymentFilter struct {
	Search        string
	InvoiceID     string
	Status        string
	PaymentMethod string
	Page          int
	PerPage       int
}

type paymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	payment.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO payments (
			id, tenant_id, invoice_id, amount, payment_method, gateway,
			gateway_trx_id, gateway_status, gateway_response, status,
			paid_at, expired_at, collected_by, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`,
		payment.ID, payment.TenantID, payment.InvoiceID, payment.Amount,
		payment.PaymentMethod, payment.Gateway, payment.GatewayTrxID,
		payment.GatewayStatus, payment.GatewayResponse, payment.Status,
		payment.PaidAt, payment.ExpiredAt, payment.CollectedBy, payment.Notes,
	)
	return err
}

func (r *paymentRepository) FindByID(ctx context.Context, tenantID, paymentID string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, invoice_id, amount, payment_method, COALESCE(gateway,''),
		       COALESCE(gateway_trx_id,''), COALESCE(gateway_status,''), gateway_response, status,
		       paid_at, expired_at, collected_by, COALESCE(notes,''), created_at
		FROM payments
		WHERE id = $1 AND tenant_id = $2
	`, paymentID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
		&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
		&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) FindByGatewayTrxID(ctx context.Context, gatewayTrxID string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, invoice_id, amount, payment_method, COALESCE(gateway,''),
		       COALESCE(gateway_trx_id,''), COALESCE(gateway_status,''), gateway_response, status,
		       paid_at, expired_at, collected_by, COALESCE(notes,''), created_at
		FROM payments
		WHERE gateway_trx_id = $1
	`, gatewayTrxID).Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
		&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
		&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// FindByGatewayTrxIDForTenant scopes the lookup to one tenant. Gateway trx IDs
// (a per-tenant date+sequence) are not globally unique, so a webhook must match
// within the tenant it authenticated as, otherwise it can hit another tenant's row.
func (r *paymentRepository) FindByGatewayTrxIDForTenant(ctx context.Context, tenantID, gatewayTrxID string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, invoice_id, amount, payment_method, COALESCE(gateway,''),
		       COALESCE(gateway_trx_id,''), COALESCE(gateway_status,''), gateway_response, status,
		       paid_at, expired_at, collected_by, COALESCE(notes,''), created_at
		FROM payments
		WHERE gateway_trx_id = $1 AND tenant_id = $2
	`, gatewayTrxID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
		&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
		&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, paymentID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE payments SET status = $1 WHERE id = $2`, status, paymentID)
	return err
}

func (r *paymentRepository) Delete(ctx context.Context, tenantID, paymentID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM payments WHERE id = $1 AND tenant_id = $2`, paymentID, tenantID)
	return err
}

func (r *paymentRepository) ListByInvoice(ctx context.Context, tenantID, invoiceID string) ([]model.Payment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, invoice_id, amount, payment_method, COALESCE(gateway,''),
		       COALESCE(gateway_trx_id,''), COALESCE(gateway_status,''), gateway_response, status,
		       paid_at, expired_at, collected_by, COALESCE(notes,''), created_at
		FROM payments
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY created_at DESC
	`, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []model.Payment
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
			&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
			&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	return payments, nil
}

func (r *paymentRepository) List(ctx context.Context, tenantID string, filter PaymentFilter) ([]model.Payment, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("p.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.InvoiceID != "" {
		conditions = append(conditions, fmt.Sprintf("p.invoice_id = $%d", argIdx))
		args = append(args, filter.InvoiceID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.PaymentMethod != "" {
		conditions = append(conditions, fmt.Sprintf("p.payment_method = $%d", argIdx))
		args = append(args, filter.PaymentMethod)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(p.gateway_trx_id ILIKE $%d OR p.notes ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM payments p "+where, args...).Scan(&total); err != nil {
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
		SELECT p.id, p.tenant_id, p.invoice_id, p.amount, p.payment_method, COALESCE(p.gateway,''),
		       COALESCE(p.gateway_trx_id,''), COALESCE(p.gateway_status,''), p.gateway_response, p.status,
		       p.paid_at, p.expired_at, p.collected_by, COALESCE(p.notes,''), p.created_at
		FROM payments p
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []model.Payment
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
			&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
			&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		payments = append(payments, p)
	}

	return payments, total, nil
}

// ListExpiredPending returns gateway payments that are still pending but whose expired_at has passed.
func (r *paymentRepository) FindActivePaymentURL(ctx context.Context, tenantID, invoiceID string) (string, error) {
	// Each gateway stores the URL under a different key inside gateway_response JSONB:
	//   Tripay  → payment_url
	//   Midtrans → redirect_url
	//   Xendit   → invoice_url
	var url string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(
			gateway_response->>'payment_url',
			gateway_response->>'redirect_url',
			gateway_response->>'invoice_url',
			''
		)
		FROM payments
		WHERE tenant_id = $1
		  AND invoice_id = $2
		  AND status = 'pending'
		  AND (expired_at IS NULL OR expired_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID, invoiceID).Scan(&url)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return url, nil
}

func (r *paymentRepository) ListExpiredPending(ctx context.Context) ([]model.Payment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, invoice_id, amount, payment_method, COALESCE(gateway,''),
			   COALESCE(gateway_trx_id,''), COALESCE(gateway_status,''), gateway_response, status,
			   paid_at, expired_at, collected_by, COALESCE(notes,''), created_at
		FROM payments
		WHERE status = 'pending'
		  AND gateway != ''
		  AND expired_at IS NOT NULL
		  AND expired_at < NOW()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []model.Payment
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.InvoiceID, &p.Amount, &p.PaymentMethod, &p.Gateway,
			&p.GatewayTrxID, &p.GatewayStatus, &p.GatewayResponse, &p.Status,
			&p.PaidAt, &p.ExpiredAt, &p.CollectedBy, &p.Notes, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}
