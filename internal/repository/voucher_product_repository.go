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

type VoucherProductRepository interface {
	Create(ctx context.Context, product *model.VoucherProduct) error
	FindByID(ctx context.Context, tenantID, productID string) (*model.VoucherProduct, error)
	Update(ctx context.Context, product *model.VoucherProduct) error
	Delete(ctx context.Context, tenantID, productID string) error
	List(ctx context.Context, tenantID string, filter VoucherProductFilter) ([]model.VoucherProduct, int, error)
}

type VoucherProductFilter struct {
	Search  string
	Active  *bool
	Page    int
	PerPage int
}

type voucherProductRepository struct {
	db *pgxpool.Pool
}

func NewVoucherProductRepository(db *pgxpool.Pool) VoucherProductRepository {
	return &voucherProductRepository{db: db}
}

func (r *voucherProductRepository) Create(ctx context.Context, product *model.VoucherProduct) error {
	product.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO voucher_products (
			id, tenant_id, name, duration, bandwidth_up, bandwidth_down,
			price, profile_name, router_id, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`,
		product.ID, product.TenantID, product.Name, product.Duration,
		product.BandwidthUp, product.BandwidthDown, product.Price,
		product.ProfileName, product.RouterID, product.IsActive,
	)
	return err
}

func (r *voucherProductRepository) FindByID(ctx context.Context, tenantID, productID string) (*model.VoucherProduct, error) {
	var p model.VoucherProduct
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, duration, bandwidth_up, bandwidth_down,
		       price, COALESCE(profile_name,''), router_id, is_active, created_at, updated_at
		FROM voucher_products
		WHERE id = $1 AND tenant_id = $2
	`, productID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Duration, &p.BandwidthUp, &p.BandwidthDown,
		&p.Price, &p.ProfileName, &p.RouterID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *voucherProductRepository) Update(ctx context.Context, product *model.VoucherProduct) error {
	_, err := r.db.Exec(ctx, `
		UPDATE voucher_products SET
			name = $1, duration = $2, bandwidth_up = $3, bandwidth_down = $4,
			price = $5, profile_name = $6, router_id = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9 AND tenant_id = $10
	`,
		product.Name, product.Duration, product.BandwidthUp, product.BandwidthDown,
		product.Price, product.ProfileName, product.RouterID, product.IsActive,
		product.ID, product.TenantID,
	)
	return err
}

func (r *voucherProductRepository) Delete(ctx context.Context, tenantID, productID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM voucher_products WHERE id = $1 AND tenant_id = $2`, productID, tenantID)
	return err
}

func (r *voucherProductRepository) List(ctx context.Context, tenantID string, filter VoucherProductFilter) ([]model.VoucherProduct, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM voucher_products "+where, args...).Scan(&total); err != nil {
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
		SELECT id, tenant_id, name, duration, bandwidth_up, bandwidth_down,
		       price, COALESCE(profile_name,''), router_id, is_active, created_at, updated_at
		FROM voucher_products
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

	var products []model.VoucherProduct
	for rows.Next() {
		var p model.VoucherProduct
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Duration, &p.BandwidthUp, &p.BandwidthDown,
			&p.Price, &p.ProfileName, &p.RouterID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, total, nil
}
