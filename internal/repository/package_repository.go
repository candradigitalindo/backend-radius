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

type PackageRepository interface {
	Create(ctx context.Context, pkg *model.Package) error
	FindByID(ctx context.Context, tenantID, packageID string) (*model.Package, error)
	Update(ctx context.Context, pkg *model.Package) error
	Delete(ctx context.Context, tenantID, packageID string) error
	List(ctx context.Context, tenantID string, filter PackageFilter) ([]model.Package, int, error)
}

type PackageFilter struct {
	Search  string
	Active  *bool
	Page    int
	PerPage int
}

type packageRepository struct {
	db *pgxpool.Pool
}

func NewPackageRepository(db *pgxpool.Pool) PackageRepository {
	return &packageRepository{db: db}
}

func (r *packageRepository) Create(ctx context.Context, pkg *model.Package) error {
	pkg.ID = id.New()
	now := time.Now()
	pkg.CreatedAt = now
	pkg.UpdatedAt = now

	query := `
		INSERT INTO packages (
			id, tenant_id, name, description, bandwidth_up, bandwidth_down,
			price, burst_limit, address_list, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Exec(ctx, query,
		pkg.ID, pkg.TenantID, pkg.Name, pkg.Description,
		pkg.BandwidthUp, pkg.BandwidthDown,
		pkg.Price, pkg.BurstLimit, pkg.AddressList,
		pkg.IsActive, pkg.CreatedAt, pkg.UpdatedAt,
	)
	return err
}

func (r *packageRepository) FindByID(ctx context.Context, tenantID, packageID string) (*model.Package, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(description,''), bandwidth_up, bandwidth_down,
		       price, COALESCE(burst_limit,''), COALESCE(address_list,''), is_active, created_at, updated_at
		FROM packages
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1
	`

	var pkg model.Package
	err := r.db.QueryRow(ctx, query, packageID, tenantID).Scan(
		&pkg.ID, &pkg.TenantID, &pkg.Name, &pkg.Description,
		&pkg.BandwidthUp, &pkg.BandwidthDown,
		&pkg.Price, &pkg.BurstLimit, &pkg.AddressList,
		&pkg.IsActive, &pkg.CreatedAt, &pkg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *packageRepository) Update(ctx context.Context, pkg *model.Package) error {
	pkg.UpdatedAt = time.Now()

	query := `
		UPDATE packages SET
			name = $1, description = $2, bandwidth_up = $3, bandwidth_down = $4,
			price = $5, burst_limit = $6, address_list = $7,
			is_active = $8, updated_at = $9
		WHERE id = $10 AND tenant_id = $11
	`

	_, err := r.db.Exec(ctx, query,
		pkg.Name, pkg.Description, pkg.BandwidthUp, pkg.BandwidthDown,
		pkg.Price, pkg.BurstLimit, pkg.AddressList,
		pkg.IsActive, pkg.UpdatedAt,
		pkg.ID, pkg.TenantID,
	)
	return err
}

func (r *packageRepository) Delete(ctx context.Context, tenantID, packageID string) error {
	query := `DELETE FROM packages WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, packageID, tenantID)
	return err
}

func (r *packageRepository) List(ctx context.Context, tenantID string, filter PackageFilter) ([]model.Package, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
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
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM packages "+where, args...).Scan(&total); err != nil {
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
		SELECT id, tenant_id, name, COALESCE(description,''), bandwidth_up, bandwidth_down,
		       price, COALESCE(burst_limit,''), COALESCE(address_list,''), is_active, created_at, updated_at
		FROM packages
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

	var packages []model.Package
	for rows.Next() {
		var pkg model.Package
		if err := rows.Scan(
			&pkg.ID, &pkg.TenantID, &pkg.Name, &pkg.Description,
			&pkg.BandwidthUp, &pkg.BandwidthDown,
			&pkg.Price, &pkg.BurstLimit, &pkg.AddressList,
			&pkg.IsActive, &pkg.CreatedAt, &pkg.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		packages = append(packages, pkg)
	}

	return packages, total, nil
}
