package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type CustomerLogHandler struct {
	customerLogService *service.CustomerLogService
}

func NewCustomerLogHandler(customerLogService *service.CustomerLogService) *CustomerLogHandler {
	return &CustomerLogHandler{customerLogService: customerLogService}
}

type createCustomerLogRequest struct {
	CustomerID  string                 `json:"customer_id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (h *CustomerLogHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)

	var req createCustomerLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerID == "" || req.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id dan action wajib diisi"})
	}

	log, err := h.customerLogService.Create(c.Context(), service.CreateCustomerLogInput{
		TenantID:    tenantID,
		CustomerID:  req.CustomerID,
		Action:      req.Action,
		Description: req.Description,
		Metadata:    req.Metadata,
		PerformedBy: &userID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat log pelanggan"})
	}

	return c.Status(fiber.StatusCreated).JSON(log)
}

func (h *CustomerLogHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	logID := c.Params("id")

	log, err := h.customerLogService.GetByID(c.Context(), tenantID, logID)
	if err != nil {
		if errors.Is(err, service.ErrCustomerLogNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(log)
}

func (h *CustomerLogHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.CustomerLogFilter{
		Action:  c.Query("action"),
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	logs, total, err := h.customerLogService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"data":  logs,
		"total": total,
		"page":  page,
	})
}

func (h *CustomerLogHandler) ListByCustomer(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.CustomerLogFilter{
		Action:  c.Query("action"),
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	logs, total, err := h.customerLogService.ListByCustomer(c.Context(), tenantID, customerID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"data":  logs,
		"total": total,
		"page":  page,
	})
}
