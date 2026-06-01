package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
)

var ErrBillingProfileNotFound = errors.New("profil billing tidak ditemukan")

type BillingProfileRepository interface {
	Create(ctx context.Context, p *model.BillingProfile) error
	FindByID(ctx context.Context, tenantID, id string) (*model.BillingProfile, error)
	List(ctx context.Context, tenantID string) ([]model.BillingProfile, error)
	Update(ctx context.Context, p *model.BillingProfile) error
	Delete(ctx context.Context, tenantID, id string) error
	SetDefault(ctx context.Context, tenantID, id string) error
}

type billingProfileRepository struct{ db *pgxpool.Pool }

func NewBillingProfileRepository(db *pgxpool.Pool) BillingProfileRepository {
	return &billingProfileRepository{db: db}
}

const bpCols = `id, tenant_id, name, description, billing_model,
	invoice_day, due_day, isolir_day, grace_period, payment_timing,
	is_default, is_active, created_at, updated_at`

func scanBP(row pgx.Row) (*model.BillingProfile, error) {
	var p model.BillingProfile
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description, &p.BillingModel,
		&p.InvoiceDay, &p.DueDay, &p.IsolirDay, &p.GracePeriod, &p.PaymentTiming,
		&p.IsDefault, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *billingProfileRepository) Create(ctx context.Context, p *model.BillingProfile) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO billing_profiles
		  (id, tenant_id, name, description, billing_model,
		   invoice_day, due_day, isolir_day, grace_period, payment_timing,
		   is_default, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		p.ID, p.TenantID, p.Name, p.Description, p.BillingModel,
		p.InvoiceDay, p.DueDay, p.IsolirDay, p.GracePeriod, p.PaymentTiming,
		p.IsDefault, p.IsActive,
	)
	return err
}

func (r *billingProfileRepository) FindByID(ctx context.Context, tenantID, id string) (*model.BillingProfile, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+bpCols+` FROM billing_profiles WHERE tenant_id=$1 AND id=$2`,
		tenantID, id,
	)
	return scanBP(row)
}

func (r *billingProfileRepository) List(ctx context.Context, tenantID string) ([]model.BillingProfile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+bpCols+` FROM billing_profiles WHERE tenant_id=$1 ORDER BY is_default DESC, name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []model.BillingProfile
	for rows.Next() {
		var p model.BillingProfile
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Description, &p.BillingModel,
			&p.InvoiceDay, &p.DueDay, &p.IsolirDay, &p.GracePeriod, &p.PaymentTiming,
			&p.IsDefault, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *billingProfileRepository) Update(ctx context.Context, p *model.BillingProfile) error {
	_, err := r.db.Exec(ctx, `
		UPDATE billing_profiles SET
		  name=$1, description=$2, billing_model=$3,
		  invoice_day=$4, due_day=$5, isolir_day=$6, grace_period=$7,
		  payment_timing=$8, is_active=$9, updated_at=NOW()
		WHERE tenant_id=$10 AND id=$11`,
		p.Name, p.Description, p.BillingModel,
		p.InvoiceDay, p.DueDay, p.IsolirDay, p.GracePeriod,
		p.PaymentTiming, p.IsActive, p.TenantID, p.ID,
	)
	return err
}

func (r *billingProfileRepository) Delete(ctx context.Context, tenantID, id string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM billing_profiles WHERE tenant_id=$1 AND id=$2`,
		tenantID, id,
	)
	return err
}

// SetDefault sets one profile as default and clears all others for the tenant (atomic).
func (r *billingProfileRepository) SetDefault(ctx context.Context, tenantID, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE billing_profiles SET is_default=FALSE WHERE tenant_id=$1`, tenantID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE billing_profiles SET is_default=TRUE WHERE tenant_id=$1 AND id=$2`, tenantID, id,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
