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

type VoucherPaymentRepository interface {
	Create(ctx context.Context, payment *model.VoucherPayment) error
	FindByID(ctx context.Context, tenantID, paymentID string) (*model.VoucherPayment, error)
	FindByGatewayTrxID(ctx context.Context, gatewayTrxID string) (*model.VoucherPayment, error)
	FindByGatewayTrxIDForTenant(ctx context.Context, tenantID, gatewayTrxID string) (*model.VoucherPayment, error)
	UpdateStatus(ctx context.Context, paymentID, status string) error
	List(ctx context.Context, tenantID string, filter VoucherPaymentFilter) ([]model.VoucherPayment, int, error)
}

type VoucherPaymentFilter struct {
	Search  string
	Status  string
	Gateway string
	Page    int
	PerPage int
}

type voucherPaymentRepository struct {
	db *pgxpool.Pool
}

func NewVoucherPaymentRepository(db *pgxpool.Pool) VoucherPaymentRepository {
	return &voucherPaymentRepository{db: db}
}

func (r *voucherPaymentRepository) Create(ctx context.Context, payment *model.VoucherPayment) error {
	payment.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO voucher_payments (
			id, tenant_id, voucher_id, buyer_name, buyer_phone,
			amount, gateway, gateway_trx_id, status, paid_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`,
		payment.ID, payment.TenantID, payment.VoucherID, payment.BuyerName,
		payment.BuyerPhone, payment.Amount, payment.Gateway, payment.GatewayTrxID,
		payment.Status, payment.PaidAt,
	)
	return err
}

func (r *voucherPaymentRepository) FindByID(ctx context.Context, tenantID, paymentID string) (*model.VoucherPayment, error) {
	var p model.VoucherPayment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, voucher_id, COALESCE(buyer_name,''), buyer_phone,
		       amount, gateway, COALESCE(gateway_trx_id,''), status, paid_at, created_at
		FROM voucher_payments
		WHERE id = $1 AND tenant_id = $2
	`, paymentID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.VoucherID, &p.BuyerName, &p.BuyerPhone,
		&p.Amount, &p.Gateway, &p.GatewayTrxID, &p.Status, &p.PaidAt, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *voucherPaymentRepository) FindByGatewayTrxID(ctx context.Context, gatewayTrxID string) (*model.VoucherPayment, error) {
	var p model.VoucherPayment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, voucher_id, COALESCE(buyer_name,''), buyer_phone,
		       amount, gateway, COALESCE(gateway_trx_id,''), status, paid_at, created_at
		FROM voucher_payments
		WHERE gateway_trx_id = $1
	`, gatewayTrxID).Scan(
		&p.ID, &p.TenantID, &p.VoucherID, &p.BuyerName, &p.BuyerPhone,
		&p.Amount, &p.Gateway, &p.GatewayTrxID, &p.Status, &p.PaidAt, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// FindByGatewayTrxIDForTenant scopes the lookup to one tenant — gateway trx IDs
// are a per-tenant date+sequence and are not globally unique, so a webhook must
// match within the tenant it authenticated as.
func (r *voucherPaymentRepository) FindByGatewayTrxIDForTenant(ctx context.Context, tenantID, gatewayTrxID string) (*model.VoucherPayment, error) {
	var p model.VoucherPayment
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, voucher_id, COALESCE(buyer_name,''), buyer_phone,
		       amount, gateway, COALESCE(gateway_trx_id,''), status, paid_at, created_at
		FROM voucher_payments
		WHERE gateway_trx_id = $1 AND tenant_id = $2
	`, gatewayTrxID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.VoucherID, &p.BuyerName, &p.BuyerPhone,
		&p.Amount, &p.Gateway, &p.GatewayTrxID, &p.Status, &p.PaidAt, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *voucherPaymentRepository) UpdateStatus(ctx context.Context, paymentID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE voucher_payments SET status = $1 WHERE id = $2`, status, paymentID)
	return err
}

func (r *voucherPaymentRepository) List(ctx context.Context, tenantID string, filter VoucherPaymentFilter) ([]model.VoucherPayment, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("vp.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(vp.buyer_name ILIKE $%d OR vp.buyer_phone ILIKE $%d OR vp.gateway_trx_id ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("vp.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Gateway != "" {
		conditions = append(conditions, fmt.Sprintf("vp.gateway = $%d", argIdx))
		args = append(args, filter.Gateway)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM voucher_payments vp "+where, args...).Scan(&total); err != nil {
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
		SELECT vp.id, vp.tenant_id, vp.voucher_id, COALESCE(vp.buyer_name,''), vp.buyer_phone,
		       vp.amount, vp.gateway, COALESCE(vp.gateway_trx_id,''), vp.status, vp.paid_at, vp.created_at
		FROM voucher_payments vp
		%s
		ORDER BY vp.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []model.VoucherPayment
	for rows.Next() {
		var p model.VoucherPayment
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.VoucherID, &p.BuyerName, &p.BuyerPhone,
			&p.Amount, &p.Gateway, &p.GatewayTrxID, &p.Status, &p.PaidAt, &p.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		payments = append(payments, p)
	}

	return payments, total, nil
}
