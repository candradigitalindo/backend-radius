package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

type createInvoiceRequest struct {
	CustomerID     string `json:"customer_id"`
	PeriodMonth    int    `json:"period_month"`
	PeriodYear     int    `json:"period_year"`
	PackagePrice   int64  `json:"package_price"`
	Discount       int64  `json:"discount"`
	AdditionalFee  int64  `json:"additional_fee"`
	FeeDescription string `json:"fee_description"`
	DueDate        string `json:"due_date"`
	Notes          string `json:"notes"`
}

type updateInvoiceRequest struct {
	PackagePrice   int64  `json:"package_price"`
	Discount       int64  `json:"discount"`
	AdditionalFee  int64  `json:"additional_fee"`
	FeeDescription string `json:"fee_description"`
	DueDate        string `json:"due_date"`
	Notes          string `json:"notes"`
}

type recordPaymentRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Notes         string `json:"notes"`
}

type generateInvoiceRequest struct {
	PeriodMonth int `json:"period_month"`
	PeriodYear  int `json:"period_year"`
	DueDay      int `json:"due_day"`
}

func (h *InvoiceHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerID == "" || req.PeriodMonth <= 0 || req.PeriodYear <= 0 || req.PackagePrice <= 0 || req.DueDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id, period_month, period_year, package_price, dan due_date wajib diisi"})
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format due_date tidak valid, gunakan YYYY-MM-DD"})
	}

	invoice, err := h.invoiceService.Create(c.Context(), service.CreateInvoiceInput{
		TenantID:       tenantID,
		CustomerID:     req.CustomerID,
		PeriodMonth:    req.PeriodMonth,
		PeriodYear:     req.PeriodYear,
		PackagePrice:   req.PackagePrice,
		Discount:       req.Discount,
		AdditionalFee:  req.AdditionalFee,
		FeeDescription: req.FeeDescription,
		DueDate:        dueDate,
		Notes:          req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat faktur"})
	}

	return c.Status(fiber.StatusCreated).JSON(invoice)
}

func (h *InvoiceHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	invoice, err := h.invoiceService.GetByID(c.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(invoice)
}

func (h *InvoiceHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	var req updateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PackagePrice <= 0 || req.DueDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "package_price dan due_date wajib diisi"})
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format due_date tidak valid, gunakan YYYY-MM-DD"})
	}

	invoice, err := h.invoiceService.Update(c.Context(), tenantID, invoiceID, service.UpdateInvoiceInput{
		PackagePrice:   req.PackagePrice,
		Discount:       req.Discount,
		AdditionalFee:  req.AdditionalFee,
		FeeDescription: req.FeeDescription,
		DueDate:        dueDate,
		Notes:          req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(invoice)
}

func (h *InvoiceHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	if err := h.invoiceService.Delete(c.Context(), tenantID, invoiceID); err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Faktur berhasil dihapus"})
}

func (h *InvoiceHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	periodMonth, _ := strconv.Atoi(c.Query("period_month"))
	periodYear, _ := strconv.Atoi(c.Query("period_year"))

	filter := repository.InvoiceFilter{
		Search:      c.Query("search"),
		CustomerID:  c.Query("customer_id"),
		Status:      c.Query("status"),
		PeriodMonth: periodMonth,
		PeriodYear:  periodYear,
		Page:        page,
		PerPage:     perPage,
	}

	invoices, total, err := h.invoiceService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar faktur"})
	}

	return c.JSON(fiber.Map{
		"data":     invoices,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *InvoiceHandler) Generate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req generateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PeriodMonth <= 0 || req.PeriodMonth > 12 || req.PeriodYear <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "period_month (1-12) dan period_year yang valid wajib diisi"})
	}

	if req.DueDay <= 0 || req.DueDay > 28 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "due_day harus antara 1 dan 28"})
	}

	count, err := h.invoiceService.GenerateMonthly(c.Context(), tenantID, req.PeriodMonth, req.PeriodYear, req.DueDay)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat faktur otomatis"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Faktur berhasil dibuat",
		"count":   count,
	})
}

func (h *InvoiceHandler) RecordPayment(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)
	invoiceID := c.Params("id")

	var req recordPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Amount <= 0 || req.PaymentMethod == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount dan payment_method wajib diisi"})
	}

	payment, err := h.invoiceService.RecordPayment(c.Context(), service.RecordPaymentInput{
		TenantID:      tenantID,
		InvoiceID:     invoiceID,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		CollectedBy:   userID,
		Notes:         req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to record payment"})
	}

	return c.Status(fiber.StatusCreated).JSON(payment)
}

func (h *InvoiceHandler) GetPayments(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	payments, err := h.invoiceService.GetPayments(c.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": payments})
}

func (h *InvoiceHandler) ListPayments(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.PaymentFilter{
		Search:        c.Query("search"),
		InvoiceID:     c.Query("invoice_id"),
		Status:        c.Query("status"),
		PaymentMethod: c.Query("payment_method"),
		Page:          page,
		PerPage:       perPage,
	}

	payments, total, err := h.invoiceService.ListPayments(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list payments"})
	}

	return c.JSON(fiber.Map{
		"data":     payments,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *InvoiceHandler) ListByCustomer(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	invoices, total, err := h.invoiceService.ListByCustomer(c.Context(), tenantID, customerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar faktur"})
	}

	return c.JSON(fiber.Map{
		"data":     invoices,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
