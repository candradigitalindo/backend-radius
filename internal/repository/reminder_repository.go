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

type ReminderRepository interface {
	Create(ctx context.Context, reminder *model.Reminder) error
	FindByID(ctx context.Context, tenantID, reminderID string) (*model.Reminder, error)
	FindActiveByType(ctx context.Context, tenantID, reminderType string) (*model.Reminder, error)
	Update(ctx context.Context, reminder *model.Reminder) error
	Delete(ctx context.Context, tenantID, reminderID string) error
	List(ctx context.Context, tenantID string, filter ReminderFilter) ([]model.Reminder, int, error)
	ListActive(ctx context.Context, tenantID string) ([]model.Reminder, error)
}

type ReminderFilter struct {
	Type    string
	Search  string
	Page    int
	PerPage int
}

type reminderRepository struct {
	db *pgxpool.Pool
}

func NewReminderRepository(db *pgxpool.Pool) ReminderRepository {
	return &reminderRepository{db: db}
}

func (r *reminderRepository) Create(ctx context.Context, reminder *model.Reminder) error {
	reminder.ID = id.New()
	now := time.Now()
	reminder.CreatedAt = now
	reminder.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO reminders (id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, reminder.ID, reminder.TenantID, reminder.Name, reminder.Type, reminder.DaysOffset,
		reminder.MessageTemplate, reminder.IsActive, reminder.CreatedAt, reminder.UpdatedAt)
	return err
}

func (r *reminderRepository) FindByID(ctx context.Context, tenantID, reminderID string) (*model.Reminder, error) {
	var rem model.Reminder
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at
		FROM reminders
		WHERE id = $1 AND tenant_id = $2
	`, reminderID, tenantID).Scan(
		&rem.ID, &rem.TenantID, &rem.Name, &rem.Type, &rem.DaysOffset,
		&rem.MessageTemplate, &rem.IsActive, &rem.CreatedAt, &rem.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rem, nil
}

func (r *reminderRepository) FindActiveByType(ctx context.Context, tenantID, reminderType string) (*model.Reminder, error) {
	var rem model.Reminder
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at
		FROM reminders
		WHERE tenant_id = $1 AND type = $2 AND is_active = true
		LIMIT 1
	`, tenantID, reminderType).Scan(
		&rem.ID, &rem.TenantID, &rem.Name, &rem.Type, &rem.DaysOffset,
		&rem.MessageTemplate, &rem.IsActive, &rem.CreatedAt, &rem.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rem, nil
}

func (r *reminderRepository) Update(ctx context.Context, reminder *model.Reminder) error {
	reminder.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE reminders SET name=$1, type=$2, days_offset=$3, message_template=$4, is_active=$5, updated_at=$6
		WHERE id=$7 AND tenant_id=$8
	`, reminder.Name, reminder.Type, reminder.DaysOffset, reminder.MessageTemplate,
		reminder.IsActive, reminder.UpdatedAt, reminder.ID, reminder.TenantID)
	return err
}

func (r *reminderRepository) Delete(ctx context.Context, tenantID, reminderID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM reminders WHERE id=$1 AND tenant_id=$2`, reminderID, tenantID)
	return err
}

func (r *reminderRepository) List(ctx context.Context, tenantID string, filter ReminderFilter) ([]model.Reminder, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR message_template ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM reminders "+where, args...).Scan(&total); err != nil {
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
		SELECT id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at
		FROM reminders %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reminders []model.Reminder
	for rows.Next() {
		var rem model.Reminder
		if err := rows.Scan(
			&rem.ID, &rem.TenantID, &rem.Name, &rem.Type, &rem.DaysOffset,
			&rem.MessageTemplate, &rem.IsActive, &rem.CreatedAt, &rem.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		reminders = append(reminders, rem)
	}

	return reminders, total, nil
}

func (r *reminderRepository) ListActive(ctx context.Context, tenantID string) ([]model.Reminder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, type, days_offset, message_template, is_active, created_at, updated_at
		FROM reminders
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY days_offset ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []model.Reminder
	for rows.Next() {
		var rem model.Reminder
		if err := rows.Scan(
			&rem.ID, &rem.TenantID, &rem.Name, &rem.Type, &rem.DaysOffset,
			&rem.MessageTemplate, &rem.IsActive, &rem.CreatedAt, &rem.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}

	return reminders, nil
}
