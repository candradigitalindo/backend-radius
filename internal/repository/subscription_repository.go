package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type SubscriptionRepository interface {
	// Plans
	ListPlans(ctx context.Context, activeOnly bool) ([]model.SubscriptionPlan, error)
	FindPlanByID(ctx context.Context, planID string) (*model.SubscriptionPlan, error)
	FindPlanBySlug(ctx context.Context, slug string) (*model.SubscriptionPlan, error)
	CreatePlan(ctx context.Context, plan *model.SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *model.SubscriptionPlan) error
	DeletePlan(ctx context.Context, planID string) error

	// Orders
	CreateOrder(ctx context.Context, order *model.SubscriptionOrder) error
	FindOrderByID(ctx context.Context, orderID string) (*model.SubscriptionOrder, error)
	FindOrderByPaymentRef(ctx context.Context, paymentRef string) (*model.SubscriptionOrder, error)
	UpdateOrder(ctx context.Context, order *model.SubscriptionOrder) error
	ListOrders(ctx context.Context, tenantID string, filter OrderFilter) ([]model.SubscriptionOrder, int, error)
}

type OrderFilter struct {
	Status  string
	Page    int
	PerPage int
}

type subscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

// --- Plans ---

func (r *subscriptionRepository) ListPlans(ctx context.Context, activeOnly bool) ([]model.SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, COALESCE(description,''), price, duration_months,
		       max_customers, max_routers, COALESCE(features,'[]'), is_popular, is_active,
		       sort_order, created_at, updated_at
		FROM subscription_plans
	`
	if activeOnly {
		query += " WHERE is_active = TRUE"
	}
	query += " ORDER BY sort_order ASC"

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []model.SubscriptionPlan
	for rows.Next() {
		var p model.SubscriptionPlan
		var featuresJSON []byte
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.DurationMonths,
			&p.MaxCustomers, &p.MaxRouters, &featuresJSON, &p.IsPopular, &p.IsActive,
			&p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
			p.Features = []string{}
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *subscriptionRepository) FindPlanByID(ctx context.Context, planID string) (*model.SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, COALESCE(description,''), price, duration_months,
		       max_customers, max_routers, COALESCE(features,'[]'), is_popular, is_active,
		       sort_order, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`
	var p model.SubscriptionPlan
	var featuresJSON []byte
	err := r.db.QueryRow(ctx, query, planID).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.DurationMonths,
		&p.MaxCustomers, &p.MaxRouters, &featuresJSON, &p.IsPopular, &p.IsActive,
		&p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
		p.Features = []string{}
	}
	return &p, nil
}

func (r *subscriptionRepository) FindPlanBySlug(ctx context.Context, slug string) (*model.SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, COALESCE(description,''), price, duration_months,
		       max_customers, max_routers, COALESCE(features,'[]'), is_popular, is_active,
		       sort_order, created_at, updated_at
		FROM subscription_plans
		WHERE slug = $1
	`
	var p model.SubscriptionPlan
	var featuresJSON []byte
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.DurationMonths,
		&p.MaxCustomers, &p.MaxRouters, &featuresJSON, &p.IsPopular, &p.IsActive,
		&p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
		p.Features = []string{}
	}
	return &p, nil
}

func (r *subscriptionRepository) CreatePlan(ctx context.Context, plan *model.SubscriptionPlan) error {
	plan.ID = id.New()
	now := time.Now()
	plan.CreatedAt = now
	plan.UpdatedAt = now

	featuresJSON, err := json.Marshal(plan.Features)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO subscription_plans
			(id, name, slug, description, price, duration_months, max_customers, max_routers,
			 features, is_popular, is_active, sort_order, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		plan.ID, plan.Name, plan.Slug, plan.Description, plan.Price, plan.DurationMonths,
		plan.MaxCustomers, plan.MaxRouters, featuresJSON, plan.IsPopular, plan.IsActive,
		plan.SortOrder, plan.CreatedAt, plan.UpdatedAt,
	)
	return err
}

func (r *subscriptionRepository) UpdatePlan(ctx context.Context, plan *model.SubscriptionPlan) error {
	plan.UpdatedAt = time.Now()

	featuresJSON, err := json.Marshal(plan.Features)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		UPDATE subscription_plans SET
			name=$1, slug=$2, description=$3, price=$4, duration_months=$5,
			max_customers=$6, max_routers=$7, features=$8, is_popular=$9,
			is_active=$10, sort_order=$11, updated_at=$12
		WHERE id=$13`,
		plan.Name, plan.Slug, plan.Description, plan.Price, plan.DurationMonths,
		plan.MaxCustomers, plan.MaxRouters, featuresJSON, plan.IsPopular,
		plan.IsActive, plan.SortOrder, plan.UpdatedAt, plan.ID,
	)
	return err
}

func (r *subscriptionRepository) DeletePlan(ctx context.Context, planID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM subscription_plans WHERE id=$1`, planID)
	return err
}

// --- Orders ---

func (r *subscriptionRepository) CreateOrder(ctx context.Context, order *model.SubscriptionOrder) error {
	order.ID = id.New()
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	query := `
		INSERT INTO subscription_orders (
			id, tenant_id, plan_id, plan_name, amount, duration_months,
			status, payment_method, payment_url, payment_ref,
			paid_at, starts_at, expires_at, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`
	_, err := r.db.Exec(ctx, query,
		order.ID, order.TenantID, order.PlanID, order.PlanName,
		order.Amount, order.DurationMonths,
		order.Status, order.PaymentMethod, order.PaymentURL, order.PaymentRef,
		order.PaidAt, order.StartsAt, order.ExpiresAt, order.Notes,
		order.CreatedAt, order.UpdatedAt,
	)
	return err
}

func (r *subscriptionRepository) FindOrderByID(ctx context.Context, orderID string) (*model.SubscriptionOrder, error) {
	query := `
		SELECT id, tenant_id, plan_id, plan_name, amount, duration_months,
		       status, COALESCE(payment_method,''), COALESCE(payment_url,''), COALESCE(payment_ref,''),
		       paid_at, starts_at, expires_at, COALESCE(notes,''), created_at, updated_at
		FROM subscription_orders
		WHERE id = $1
	`
	var o model.SubscriptionOrder
	err := r.db.QueryRow(ctx, query, orderID).Scan(
		&o.ID, &o.TenantID, &o.PlanID, &o.PlanName, &o.Amount, &o.DurationMonths,
		&o.Status, &o.PaymentMethod, &o.PaymentURL, &o.PaymentRef,
		&o.PaidAt, &o.StartsAt, &o.ExpiresAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *subscriptionRepository) FindOrderByPaymentRef(ctx context.Context, paymentRef string) (*model.SubscriptionOrder, error) {
	query := `
		SELECT id, tenant_id, plan_id, plan_name, amount, duration_months,
		       status, COALESCE(payment_method,''), COALESCE(payment_url,''), COALESCE(payment_ref,''),
		       paid_at, starts_at, expires_at, COALESCE(notes,''), created_at, updated_at
		FROM subscription_orders
		WHERE payment_ref = $1
	`
	var o model.SubscriptionOrder
	err := r.db.QueryRow(ctx, query, paymentRef).Scan(
		&o.ID, &o.TenantID, &o.PlanID, &o.PlanName, &o.Amount, &o.DurationMonths,
		&o.Status, &o.PaymentMethod, &o.PaymentURL, &o.PaymentRef,
		&o.PaidAt, &o.StartsAt, &o.ExpiresAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *subscriptionRepository) UpdateOrder(ctx context.Context, order *model.SubscriptionOrder) error {
	order.UpdatedAt = time.Now()
	query := `
		UPDATE subscription_orders SET
			status = $1, payment_method = $2, payment_url = $3, payment_ref = $4,
			paid_at = $5, starts_at = $6, expires_at = $7, notes = $8, updated_at = $9
		WHERE id = $10
	`
	_, err := r.db.Exec(ctx, query,
		order.Status, order.PaymentMethod, order.PaymentURL, order.PaymentRef,
		order.PaidAt, order.StartsAt, order.ExpiresAt, order.Notes, order.UpdatedAt,
		order.ID,
	)
	return err
}

func (r *subscriptionRepository) ListOrders(ctx context.Context, tenantID string, filter OrderFilter) ([]model.SubscriptionOrder, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM subscription_orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	query := fmt.Sprintf(`
		SELECT id, tenant_id, plan_id, plan_name, amount, duration_months,
		       status, COALESCE(payment_method,''), COALESCE(payment_url,''), COALESCE(payment_ref,''),
		       paid_at, starts_at, expires_at, COALESCE(notes,''), created_at, updated_at
		FROM subscription_orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []model.SubscriptionOrder
	for rows.Next() {
		var o model.SubscriptionOrder
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.PlanID, &o.PlanName, &o.Amount, &o.DurationMonths,
			&o.Status, &o.PaymentMethod, &o.PaymentURL, &o.PaymentRef,
			&o.PaidAt, &o.StartsAt, &o.ExpiresAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}
