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

type ExpenseCategoryRepository interface {
	Create(ctx context.Context, category *model.ExpenseCategory) error
	FindByID(ctx context.Context, tenantID, categoryID string) (*model.ExpenseCategory, error)
	Update(ctx context.Context, category *model.ExpenseCategory) error
	Delete(ctx context.Context, tenantID, categoryID string) error
	List(ctx context.Context, tenantID string) ([]model.ExpenseCategory, error)
}

type expenseCategoryRepository struct {
	db *pgxpool.Pool
}

func NewExpenseCategoryRepository(db *pgxpool.Pool) ExpenseCategoryRepository {
	return &expenseCategoryRepository{db: db}
}

func (r *expenseCategoryRepository) Create(ctx context.Context, category *model.ExpenseCategory) error {
	category.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO expense_categories (id, tenant_id, name, color)
		VALUES ($1,$2,$3,$4)
	`, category.ID, category.TenantID, category.Name, category.Color)
	return err
}

func (r *expenseCategoryRepository) FindByID(ctx context.Context, tenantID, categoryID string) (*model.ExpenseCategory, error) {
	var c model.ExpenseCategory
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, color
		FROM expense_categories
		WHERE id = $1 AND tenant_id = $2
	`, categoryID, tenantID).Scan(&c.ID, &c.TenantID, &c.Name, &c.Color)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *expenseCategoryRepository) Update(ctx context.Context, category *model.ExpenseCategory) error {
	_, err := r.db.Exec(ctx, `
		UPDATE expense_categories SET name = $1, color = $2
		WHERE id = $3 AND tenant_id = $4
	`, category.Name, category.Color, category.ID, category.TenantID)
	return err
}

func (r *expenseCategoryRepository) Delete(ctx context.Context, tenantID, categoryID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1 AND tenant_id = $2`, categoryID, tenantID)
	return err
}

func (r *expenseCategoryRepository) List(ctx context.Context, tenantID string) ([]model.ExpenseCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, color
		FROM expense_categories
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.ExpenseCategory
	for rows.Next() {
		var c model.ExpenseCategory
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Color); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// -- Expense Repository --

type ExpenseRepository interface {
	Create(ctx context.Context, expense *model.Expense) error
	FindByID(ctx context.Context, tenantID, expenseID string) (*model.Expense, error)
	Update(ctx context.Context, expense *model.Expense) error
	Delete(ctx context.Context, tenantID, expenseID string) error
	List(ctx context.Context, tenantID string, filter ExpenseFilter) ([]model.Expense, int, error)
	SumByDateRange(ctx context.Context, tenantID, startDate, endDate string) (int64, error)
}

type ExpenseFilter struct {
	Search     string
	CategoryID string
	StartDate  string
	EndDate    string
	Page       int
	PerPage    int
}

type expenseRepository struct {
	db *pgxpool.Pool
}

func NewExpenseRepository(db *pgxpool.Pool) ExpenseRepository {
	return &expenseRepository{db: db}
}

func (r *expenseRepository) Create(ctx context.Context, expense *model.Expense) error {
	expense.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO expenses (id, tenant_id, category_id, description, amount, expense_date, receipt_url, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, expense.ID, expense.TenantID, expense.CategoryID, expense.Description,
		expense.Amount, expense.ExpenseDate, expense.ReceiptURL, expense.CreatedBy)
	return err
}

func (r *expenseRepository) FindByID(ctx context.Context, tenantID, expenseID string) (*model.Expense, error) {
	var e model.Expense
	var catID *string
	var catName, catColor *string
	err := r.db.QueryRow(ctx, `
		SELECT e.id, e.tenant_id, e.category_id, e.description, e.amount,
		       e.expense_date, COALESCE(e.receipt_url,''), e.created_by, e.created_at,
		       ec.name, ec.color
		FROM expenses e
		LEFT JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.id = $1 AND e.tenant_id = $2
	`, expenseID, tenantID).Scan(
		&e.ID, &e.TenantID, &catID, &e.Description, &e.Amount,
		&e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt,
		&catName, &catColor,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	e.CategoryID = catID
	if catID != nil && catName != nil {
		e.Category = &model.ExpenseCategory{ID: *catID, Name: *catName, Color: *catColor}
	}
	return &e, nil
}

func (r *expenseRepository) Update(ctx context.Context, expense *model.Expense) error {
	_, err := r.db.Exec(ctx, `
		UPDATE expenses SET
			category_id = $1, description = $2, amount = $3,
			expense_date = $4, receipt_url = $5
		WHERE id = $6 AND tenant_id = $7
	`, expense.CategoryID, expense.Description, expense.Amount,
		expense.ExpenseDate, expense.ReceiptURL,
		expense.ID, expense.TenantID)
	return err
}

func (r *expenseRepository) Delete(ctx context.Context, tenantID, expenseID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM expenses WHERE id = $1 AND tenant_id = $2`, expenseID, tenantID)
	return err
}

func (r *expenseRepository) List(ctx context.Context, tenantID string, filter ExpenseFilter) ([]model.Expense, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("e.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("e.description ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("e.category_id = $%d", argIdx))
		args = append(args, filter.CategoryID)
		argIdx++
	}

	if filter.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("e.expense_date >= $%d", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}

	if filter.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("e.expense_date <= $%d", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM expenses e " + where
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
		SELECT e.id, e.tenant_id, e.category_id, e.description, e.amount,
		       e.expense_date, COALESCE(e.receipt_url,''), e.created_by, e.created_at,
		       ec.name, ec.color
		FROM expenses e
		LEFT JOIN expense_categories ec ON ec.id = e.category_id
		%s
		ORDER BY e.expense_date DESC, e.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		var e model.Expense
		var catID *string
		var catName, catColor *string
		if err := rows.Scan(
			&e.ID, &e.TenantID, &catID, &e.Description, &e.Amount,
			&e.ExpenseDate, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt,
			&catName, &catColor,
		); err != nil {
			return nil, 0, err
		}
		e.CategoryID = catID
		if catID != nil && catName != nil {
			e.Category = &model.ExpenseCategory{ID: *catID, Name: *catName, Color: *catColor}
		}
		expenses = append(expenses, e)
	}

	return expenses, total, nil
}

func (r *expenseRepository) SumByDateRange(ctx context.Context, tenantID, startDate, endDate string) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM expenses
		WHERE tenant_id = $1 AND expense_date >= $2 AND expense_date <= $3
	`, tenantID, startDate, endDate).Scan(&total)
	return total, err
}
