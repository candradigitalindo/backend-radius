package service

import (
	"context"
	"errors"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrExpenseCategoryNotFound = errors.New("Kategori pengeluaran tidak ditemukan")
	ErrExpenseNotFound         = errors.New("Pengeluaran tidak ditemukan")
)

type ExpenseService struct {
	categoryRepo repository.ExpenseCategoryRepository
	expenseRepo  repository.ExpenseRepository
}

func NewExpenseService(
	categoryRepo repository.ExpenseCategoryRepository,
	expenseRepo repository.ExpenseRepository,
) *ExpenseService {
	return &ExpenseService{
		categoryRepo: categoryRepo,
		expenseRepo:  expenseRepo,
	}
}

// -- Category --

type CreateCategoryInput struct {
	TenantID string
	Name     string
	Color    string
}

type UpdateCategoryInput struct {
	Name  string
	Color string
}

func (s *ExpenseService) CreateCategory(ctx context.Context, input CreateCategoryInput) (*model.ExpenseCategory, error) {
	if input.Color == "" {
		input.Color = "#6B7280"
	}

	category := &model.ExpenseCategory{
		TenantID: input.TenantID,
		Name:     input.Name,
		Color:    input.Color,
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return s.categoryRepo.FindByID(ctx, input.TenantID, category.ID)
}

func (s *ExpenseService) GetCategory(ctx context.Context, tenantID, categoryID string) (*model.ExpenseCategory, error) {
	cat, err := s.categoryRepo.FindByID(ctx, tenantID, categoryID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrExpenseCategoryNotFound
	}
	return cat, nil
}

func (s *ExpenseService) UpdateCategory(ctx context.Context, tenantID, categoryID string, input UpdateCategoryInput) (*model.ExpenseCategory, error) {
	cat, err := s.categoryRepo.FindByID(ctx, tenantID, categoryID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrExpenseCategoryNotFound
	}

	cat.Name = input.Name
	cat.Color = input.Color

	if err := s.categoryRepo.Update(ctx, cat); err != nil {
		return nil, err
	}

	return s.categoryRepo.FindByID(ctx, tenantID, categoryID)
}

func (s *ExpenseService) DeleteCategory(ctx context.Context, tenantID, categoryID string) error {
	cat, err := s.categoryRepo.FindByID(ctx, tenantID, categoryID)
	if err != nil {
		return err
	}
	if cat == nil {
		return ErrExpenseCategoryNotFound
	}
	return s.categoryRepo.Delete(ctx, tenantID, categoryID)
}

func (s *ExpenseService) ListCategories(ctx context.Context, tenantID string) ([]model.ExpenseCategory, error) {
	return s.categoryRepo.List(ctx, tenantID)
}

// -- Expense --

type CreateExpenseInput struct {
	TenantID    string
	CategoryID  *string
	Description string
	Amount      int64
	ExpenseDate time.Time
	ReceiptURL  string
	CreatedBy   string
}

type UpdateExpenseInput struct {
	CategoryID  *string
	Description string
	Amount      int64
	ExpenseDate time.Time
	ReceiptURL  string
}

func (s *ExpenseService) Create(ctx context.Context, input CreateExpenseInput) (*model.Expense, error) {
	expense := &model.Expense{
		TenantID:    input.TenantID,
		CategoryID:  input.CategoryID,
		Description: input.Description,
		Amount:      input.Amount,
		ExpenseDate: input.ExpenseDate,
		ReceiptURL:  input.ReceiptURL,
		CreatedBy:   input.CreatedBy,
	}

	if err := s.expenseRepo.Create(ctx, expense); err != nil {
		return nil, err
	}

	return s.expenseRepo.FindByID(ctx, input.TenantID, expense.ID)
}

func (s *ExpenseService) GetByID(ctx context.Context, tenantID, expenseID string) (*model.Expense, error) {
	expense, err := s.expenseRepo.FindByID(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, ErrExpenseNotFound
	}
	return expense, nil
}

func (s *ExpenseService) Update(ctx context.Context, tenantID, expenseID string, input UpdateExpenseInput) (*model.Expense, error) {
	expense, err := s.expenseRepo.FindByID(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, ErrExpenseNotFound
	}

	expense.CategoryID = input.CategoryID
	expense.Description = input.Description
	expense.Amount = input.Amount
	expense.ExpenseDate = input.ExpenseDate
	expense.ReceiptURL = input.ReceiptURL

	if err := s.expenseRepo.Update(ctx, expense); err != nil {
		return nil, err
	}

	return s.expenseRepo.FindByID(ctx, tenantID, expenseID)
}

func (s *ExpenseService) Delete(ctx context.Context, tenantID, expenseID string) error {
	expense, err := s.expenseRepo.FindByID(ctx, tenantID, expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrExpenseNotFound
	}
	return s.expenseRepo.Delete(ctx, tenantID, expenseID)
}

func (s *ExpenseService) List(ctx context.Context, tenantID string, filter repository.ExpenseFilter) ([]model.Expense, int, error) {
	return s.expenseRepo.List(ctx, tenantID, filter)
}

func (s *ExpenseService) SumByDateRange(ctx context.Context, tenantID, startDate, endDate string) (int64, error) {
	return s.expenseRepo.SumByDateRange(ctx, tenantID, startDate, endDate)
}
