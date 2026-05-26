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

type VoucherRepository interface {
	Create(ctx context.Context, voucher *model.Voucher) error
	CreateBatch(ctx context.Context, vouchers []model.Voucher) error
	FindByID(ctx context.Context, tenantID, voucherID string) (*model.Voucher, error)
	FindByUsername(ctx context.Context, username string) (*model.Voucher, error)
	FindByUsernameWithProduct(ctx context.Context, username string) (*model.Voucher, error)
	UpdateStatus(ctx context.Context, voucherID, status string) error
	Activate(ctx context.Context, voucherID string, activatedAt, expiresAt time.Time) error
	MarkSold(ctx context.Context, voucher *model.Voucher) error
	Delete(ctx context.Context, tenantID, voucherID string) error
	List(ctx context.Context, tenantID string, filter VoucherFilter) ([]model.Voucher, int, error)
	CountByProduct(ctx context.Context, tenantID, productID, status string) (int, error)
}

type VoucherFilter struct {
	Search    string
	ProductID string
	Status    string
	Page      int
	PerPage   int
}

type voucherRepository struct {
	db *pgxpool.Pool
}

func NewVoucherRepository(db *pgxpool.Pool) VoucherRepository {
	return &voucherRepository{db: db}
}

func (r *voucherRepository) Create(ctx context.Context, voucher *model.Voucher) error {
	voucher.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO vouchers (id, tenant_id, product_id, username, password, status)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, voucher.ID, voucher.TenantID, voucher.ProductID, voucher.Username, voucher.Password, voucher.Status)
	return err
}

func (r *voucherRepository) CreateBatch(ctx context.Context, vouchers []model.Voucher) error {
	if len(vouchers) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for i := range vouchers {
		vouchers[i].ID = id.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO vouchers (id, tenant_id, product_id, username, password, status)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, vouchers[i].ID, vouchers[i].TenantID, vouchers[i].ProductID,
			vouchers[i].Username, vouchers[i].Password, vouchers[i].Status)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *voucherRepository) FindByID(ctx context.Context, tenantID, voucherID string) (*model.Voucher, error) {
	var v model.Voucher
	var prodName string
	var prodPrice int64
	var prodDuration int
	err := r.db.QueryRow(ctx, `
		SELECT v.id, v.tenant_id, v.product_id, v.username, v.password, v.status,
		       v.buyer_phone, v.sold_at, v.activated_at, v.expires_at, v.created_at,
		       p.name, p.price, p.duration
		FROM vouchers v
		JOIN voucher_products p ON p.id = v.product_id
		WHERE v.id = $1 AND v.tenant_id = $2
	`, voucherID, tenantID).Scan(
		&v.ID, &v.TenantID, &v.ProductID, &v.Username, &v.Password, &v.Status,
		&v.BuyerPhone, &v.SoldAt, &v.ActivatedAt, &v.ExpiresAt, &v.CreatedAt,
		&prodName, &prodPrice, &prodDuration,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	v.Product = &model.VoucherProduct{ID: v.ProductID, Name: prodName, Price: prodPrice, Duration: prodDuration}
	return &v, nil
}

func (r *voucherRepository) FindByUsername(ctx context.Context, username string) (*model.Voucher, error) {
	var v model.Voucher
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, product_id, username, password, status,
		       buyer_phone, sold_at, activated_at, expires_at, created_at
		FROM vouchers
		WHERE username = $1
	`, username).Scan(
		&v.ID, &v.TenantID, &v.ProductID, &v.Username, &v.Password, &v.Status,
		&v.BuyerPhone, &v.SoldAt, &v.ActivatedAt, &v.ExpiresAt, &v.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *voucherRepository) FindByUsernameWithProduct(ctx context.Context, username string) (*model.Voucher, error) {
	var v model.Voucher
	var prod model.VoucherProduct
	err := r.db.QueryRow(ctx, `
		SELECT v.id, v.tenant_id, v.product_id, v.username, v.password, v.status,
		       v.buyer_phone, v.sold_at, v.activated_at, v.expires_at, v.created_at,
		       p.id, p.name, p.duration, p.bandwidth_up, p.bandwidth_down, p.price, COALESCE(p.profile_name,'')
		FROM vouchers v
		JOIN voucher_products p ON p.id = v.product_id
		WHERE v.username = $1
	`, username).Scan(
		&v.ID, &v.TenantID, &v.ProductID, &v.Username, &v.Password, &v.Status,
		&v.BuyerPhone, &v.SoldAt, &v.ActivatedAt, &v.ExpiresAt, &v.CreatedAt,
		&prod.ID, &prod.Name, &prod.Duration, &prod.BandwidthUp, &prod.BandwidthDown,
		&prod.Price, &prod.ProfileName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	v.Product = &prod
	return &v, nil
}

func (r *voucherRepository) UpdateStatus(ctx context.Context, voucherID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE vouchers SET status = $1 WHERE id = $2`, status, voucherID)
	return err
}

func (r *voucherRepository) Activate(ctx context.Context, voucherID string, activatedAt, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vouchers SET status = 'active', activated_at = $1, expires_at = $2
		WHERE id = $3
	`, activatedAt, expiresAt, voucherID)
	return err
}

func (r *voucherRepository) MarkSold(ctx context.Context, voucher *model.Voucher) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vouchers SET status = 'sold', buyer_phone = $1, sold_at = NOW()
		WHERE id = $2
	`, voucher.BuyerPhone, voucher.ID)
	return err
}

func (r *voucherRepository) Delete(ctx context.Context, tenantID, voucherID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vouchers WHERE id = $1 AND tenant_id = $2`, voucherID, tenantID)
	return err
}

func (r *voucherRepository) List(ctx context.Context, tenantID string, filter VoucherFilter) ([]model.Voucher, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("v.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(v.username ILIKE $%d OR v.buyer_phone ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.ProductID != "" {
		conditions = append(conditions, fmt.Sprintf("v.product_id = $%d", argIdx))
		args = append(args, filter.ProductID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM vouchers v " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
		SELECT v.id, v.tenant_id, v.product_id, v.username, v.password, v.status,
		       v.buyer_phone, v.sold_at, v.activated_at, v.expires_at, v.created_at,
		       p.name, p.price, p.duration
		FROM vouchers v
		JOIN voucher_products p ON p.id = v.product_id
		%s
		ORDER BY v.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var vouchers []model.Voucher
	for rows.Next() {
		var v model.Voucher
		var prodName string
		var prodPrice int64
		var prodDuration int
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.ProductID, &v.Username, &v.Password, &v.Status,
			&v.BuyerPhone, &v.SoldAt, &v.ActivatedAt, &v.ExpiresAt, &v.CreatedAt,
			&prodName, &prodPrice, &prodDuration,
		); err != nil {
			return nil, 0, err
		}
		v.Product = &model.VoucherProduct{ID: v.ProductID, Name: prodName, Price: prodPrice, Duration: prodDuration}
		vouchers = append(vouchers, v)
	}

	return vouchers, total, nil
}

func (r *voucherRepository) CountByProduct(ctx context.Context, tenantID, productID, status string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM vouchers WHERE tenant_id = $1 AND product_id = $2 AND status = $3
	`, tenantID, productID, status).Scan(&count)
	return count, err
}
