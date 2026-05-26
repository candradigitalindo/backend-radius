package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, tenantID, email string) (*model.User, error)
	FindByEmailOnly(ctx context.Context, email string) ([]*model.User, error)
	FindByID(ctx context.Context, userID string) (*model.User, error)
	FindByPhone(ctx context.Context, phone string) ([]*model.User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, tenantID, userID string) error
	CountByRole(ctx context.Context, tenantID, role string) (int, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	user.ID = id.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, tenant_id, name, email, password_hash, role, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID, user.TenantID, user.Name, user.Email, user.PasswordHash,
		user.Role, user.Phone, user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepository) FindByEmail(ctx context.Context, tenantID, email string) (*model.User, error) {
	query := `
		SELECT id, tenant_id, name, email, password_hash, role, COALESCE(phone,''), is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2
		LIMIT 1
	`

	var u model.User
	err := r.db.QueryRow(ctx, query, tenantID, email).Scan(
		&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash,
		&u.Role, &u.Phone, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByEmailOnly(ctx context.Context, email string) ([]*model.User, error) {
	query := `
		SELECT id, tenant_id, name, email, password_hash, role, COALESCE(phone,''), is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	rows, err := r.db.Query(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash,
			&u.Role, &u.Phone, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *userRepository) FindByPhone(ctx context.Context, phone string) ([]*model.User, error) {
	query := `
		SELECT id, tenant_id, name, email, password_hash, role, COALESCE(phone,''), is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE phone = $1 AND is_active = true
	`

	rows, err := r.db.Query(ctx, query, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash,
			&u.Role, &u.Phone, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *userRepository) FindByID(ctx context.Context, userID string) (*model.User, error) {
	query := `
		SELECT id, tenant_id, name, email, password_hash, role, COALESCE(phone,''), is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	var u model.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash,
		&u.Role, &u.Phone, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *userRepository) ListByTenant(ctx context.Context, tenantID string) ([]*model.User, error) {
	query := `
		SELECT id, tenant_id, name, email, password_hash, role, COALESCE(phone,''), is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash,
			&u.Role, &u.Phone, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	query := `
		UPDATE users
		SET name = $1, email = $2, role = $3, phone = $4, is_active = $5, password_hash = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`
	_, err := r.db.Exec(ctx, query,
		user.Name, user.Email, user.Role, user.Phone, user.IsActive, user.PasswordHash, user.UpdatedAt,
		user.ID, user.TenantID,
	)
	return err
}

func (r *userRepository) Delete(ctx context.Context, tenantID, userID string) error {
	query := `DELETE FROM users WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, userID, tenantID)
	return err
}

func (r *userRepository) CountByRole(ctx context.Context, tenantID, role string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND role = $2`, tenantID, role).Scan(&count)
	return count, err
}
