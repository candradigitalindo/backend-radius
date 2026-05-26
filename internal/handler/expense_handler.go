package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type ExpenseHandler struct {
	expenseService *service.ExpenseService
}

func NewExpenseHandler(expenseService *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseService: expenseService}
}

// -- Category requests --

type createCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type updateCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// -- Category handlers --

func (h *ExpenseHandler) CreateCategory(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	cat, err := h.expenseService.CreateCategory(c.Context(), service.CreateCategoryInput{
		TenantID: tenantID,
		Name:     req.Name,
		Color:    req.Color,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat kategori pengeluaran"})
	}

	return c.Status(fiber.StatusCreated).JSON(cat)
}

func (h *ExpenseHandler) GetCategory(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	categoryID := c.Params("id")

	cat, err := h.expenseService.GetCategory(c.Context(), tenantID, categoryID)
	if err != nil {
		if errors.Is(err, service.ErrExpenseCategoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(cat)
}

func (h *ExpenseHandler) UpdateCategory(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	categoryID := c.Params("id")

	var req updateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	cat, err := h.expenseService.UpdateCategory(c.Context(), tenantID, categoryID, service.UpdateCategoryInput{
		Name:  req.Name,
		Color: req.Color,
	})
	if err != nil {
		if errors.Is(err, service.ErrExpenseCategoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(cat)
}

func (h *ExpenseHandler) DeleteCategory(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	categoryID := c.Params("id")

	if err := h.expenseService.DeleteCategory(c.Context(), tenantID, categoryID); err != nil {
		if errors.Is(err, service.ErrExpenseCategoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Kategori pengeluaran dihapus"})
}

func (h *ExpenseHandler) ListCategories(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	cats, err := h.expenseService.ListCategories(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": cats})
}

// -- Expense requests --

type createExpenseRequest struct {
	CategoryID  *string `json:"category_id"`
	Description string  `json:"description"`
	Amount      int64   `json:"amount"`
	ExpenseDate string  `json:"expense_date"`
	ReceiptURL  string  `json:"receipt_url"`
}

type updateExpenseRequest struct {
	CategoryID  *string `json:"category_id"`
	Description string  `json:"description"`
	Amount      int64   `json:"amount"`
	ExpenseDate string  `json:"expense_date"`
	ReceiptURL  string  `json:"receipt_url"`
}

// -- Expense handlers --

func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)

	var req createExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Description == "" || req.Amount <= 0 || req.ExpenseDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Deskripsi, jumlah, dan tanggal pengeluaran wajib diisi"})
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format expense_date tidak valid, gunakan YYYY-MM-DD"})
	}

	expense, err := h.expenseService.Create(c.Context(), service.CreateExpenseInput{
		TenantID:    tenantID,
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Amount:      req.Amount,
		ExpenseDate: expenseDate,
		ReceiptURL:  req.ReceiptURL,
		CreatedBy:   userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pengeluaran"})
	}

	return c.Status(fiber.StatusCreated).JSON(expense)
}

func (h *ExpenseHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	expenseID := c.Params("id")

	expense, err := h.expenseService.GetByID(c.Context(), tenantID, expenseID)
	if err != nil {
		if errors.Is(err, service.ErrExpenseNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(expense)
}

func (h *ExpenseHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	expenseID := c.Params("id")

	var req updateExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Description == "" || req.Amount <= 0 || req.ExpenseDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Deskripsi, jumlah, dan tanggal pengeluaran wajib diisi"})
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format expense_date tidak valid, gunakan YYYY-MM-DD"})
	}

	expense, err := h.expenseService.Update(c.Context(), tenantID, expenseID, service.UpdateExpenseInput{
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Amount:      req.Amount,
		ExpenseDate: expenseDate,
		ReceiptURL:  req.ReceiptURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrExpenseNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(expense)
}

func (h *ExpenseHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	expenseID := c.Params("id")

	if err := h.expenseService.Delete(c.Context(), tenantID, expenseID); err != nil {
		if errors.Is(err, service.ErrExpenseNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Pengeluaran dihapus"})
}

func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.ExpenseFilter{
		Search:     c.Query("search"),
		CategoryID: c.Query("category_id"),
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		Page:       page,
		PerPage:    perPage,
	}

	expenses, total, err := h.expenseService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"data":     expenses,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *ExpenseHandler) Summary(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "start_date dan end_date wajib diisi"})
	}

	// Validate date format (YYYY-MM-DD)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format start_date tidak valid (YYYY-MM-DD)"})
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format end_date tidak valid (YYYY-MM-DD)"})
	}

	total, err := h.expenseService.SumByDateRange(c.Context(), tenantID, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"start_date":    startDate,
		"end_date":      endDate,
		"total_expense": total,
	})
}
