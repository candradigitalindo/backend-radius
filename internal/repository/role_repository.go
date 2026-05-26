package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type RoleRepository interface {
	List(ctx context.Context, tenantID string) ([]*model.Role, error)
	FindByID(ctx context.Context, roleID string) (*model.Role, error)
	FindBySlug(ctx context.Context, tenantID, slug string) (*model.Role, error)
	Create(ctx context.Context, role *model.Role) error
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, tenantID, roleID string) error
	CountByTenant(ctx context.Context, tenantID string) (int, error)
}

type roleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) List(ctx context.Context, tenantID string) ([]*model.Role, error) {
	query := `
		SELECT id, tenant_id, name, slug, COALESCE(description,''), is_system, COALESCE(permissions,'{}'), created_at, updated_at
		FROM roles
		WHERE tenant_id = $1
		ORDER BY is_system DESC, name ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
			&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}
	return roles, nil
}

func (r *roleRepository) FindByID(ctx context.Context, roleID string) (*model.Role, error) {
	query := `
		SELECT id, tenant_id, name, slug, COALESCE(description,''), is_system, COALESCE(permissions,'{}'), created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	var role model.Role
	err := r.db.QueryRow(ctx, query, roleID).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
		&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindBySlug(ctx context.Context, tenantID, slug string) (*model.Role, error) {
	query := `
		SELECT id, tenant_id, name, slug, COALESCE(description,''), is_system, COALESCE(permissions,'{}'), created_at, updated_at
		FROM roles
		WHERE tenant_id = $1 AND slug = $2
	`

	var role model.Role
	err := r.db.QueryRow(ctx, query, tenantID, slug).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
		&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	role.ID = id.New()
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	query := `
		INSERT INTO roles (id, tenant_id, name, slug, description, is_system, permissions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, slug) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query,
		role.ID, role.TenantID, role.Name, role.Slug, role.Description,
		role.IsSystem, role.Permissions, role.CreatedAt, role.UpdatedAt,
	)
	return err
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	role.UpdatedAt = time.Now()

	query := `
		UPDATE roles
		SET name = $1, description = $2, permissions = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`

	_, err := r.db.Exec(ctx, query,
		role.Name, role.Description, role.Permissions, role.UpdatedAt,
		role.ID, role.TenantID,
	)
	return err
}

func (r *roleRepository) Delete(ctx context.Context, tenantID, roleID string) error {
	query := `DELETE FROM roles WHERE id = $1 AND tenant_id = $2 AND is_system = false`
	_, err := r.db.Exec(ctx, query, roleID, tenantID)
	return err
}

func (r *roleRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM roles WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}
