package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type SettingRepository interface {
	Upsert(ctx context.Context, setting *model.Setting) error
	FindByKey(ctx context.Context, tenantID, key string) (*model.Setting, error)
	Delete(ctx context.Context, tenantID, key string) error
	List(ctx context.Context, tenantID string) ([]model.Setting, error)
	BulkUpsert(ctx context.Context, tenantID string, settings map[string]string) error
}

type settingRepository struct {
	db *pgxpool.Pool
}

func NewSettingRepository(db *pgxpool.Pool) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) Upsert(ctx context.Context, setting *model.Setting) error {
	setting.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO settings (id, tenant_id, key, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value
	`, setting.ID, setting.TenantID, setting.Key, setting.Value)
	return err
}

func (r *settingRepository) FindByKey(ctx context.Context, tenantID, key string) (*model.Setting, error) {
	var s model.Setting
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, key, value
		FROM settings
		WHERE tenant_id = $1 AND key = $2
	`, tenantID, key).Scan(&s.ID, &s.TenantID, &s.Key, &s.Value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *settingRepository) Delete(ctx context.Context, tenantID, key string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM settings WHERE tenant_id = $1 AND key = $2`, tenantID, key)
	return err
}

func (r *settingRepository) List(ctx context.Context, tenantID string) ([]model.Setting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, key, value
		FROM settings
		WHERE tenant_id = $1
		ORDER BY key ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []model.Setting
	for rows.Next() {
		var s model.Setting
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Key, &s.Value); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}

	return settings, nil
}

func (r *settingRepository) BulkUpsert(ctx context.Context, tenantID string, settings map[string]string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for key, value := range settings {
		newID := id.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO settings (id, tenant_id, key, value)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value
		`, newID, tenantID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
