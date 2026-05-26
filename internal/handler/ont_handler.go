package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type ONTHandler struct {
	ontService *service.ONTService
}

func NewONTHandler(ontService *service.ONTService) *ONTHandler {
	return &ONTHandler{ontService: ontService}
}

type createONTRequest struct {
	SerialNumber string  `json:"serial_number"`
	CustomerID   *string `json:"customer_id"`
	Vendor       *string `json:"vendor"`
	Model        *string `json:"model"`
	Notes        *string `json:"notes"`
}

type updateONTRequest struct {
	SerialNumber string  `json:"serial_number"`
	CustomerID   *string `json:"customer_id"`
	Vendor       *string `json:"vendor"`
	Model        *string `json:"model"`
	Status       string  `json:"status"`
	Notes        *string `json:"notes"`
}

func (h *ONTHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createONTRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.SerialNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "serial_number wajib diisi"})
	}

	ont, err := h.ontService.Create(c.Context(), service.CreateONTInput{
		TenantID:     tenantID,
		SerialNumber: req.SerialNumber,
		CustomerID:   req.CustomerID,
		Vendor:       req.Vendor,
		Model:        req.Model,
		Notes:        req.Notes,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat ONT"})
	}

	return c.Status(fiber.StatusCreated).JSON(ont)
}

func (h *ONTHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	ont, err := h.ontService.GetByID(c.Context(), tenantID, ontID)
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(ont)
}

func (h *ONTHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")
	userID, _ := c.Locals("user_id").(string)

	var req updateONTRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.SerialNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "serial_number wajib diisi"})
	}

	var performedBy *string
	if userID != "" {
		performedBy = &userID
	}

	ont, err := h.ontService.Update(c.Context(), tenantID, ontID, service.UpdateONTInput{
		SerialNumber: req.SerialNumber,
		CustomerID:   req.CustomerID,
		Vendor:       req.Vendor,
		Model:        req.Model,
		Status:       req.Status,
		Notes:        req.Notes,
		PerformedBy:  performedBy,
	})
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(ont)
}

func (h *ONTHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	if err := h.ontService.Delete(c.Context(), tenantID, ontID); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "ONT berhasil dihapus"})
}

func (h *ONTHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.ONTFilter{
		Search:     c.Query("search"),
		Status:     c.Query("status"),
		CustomerID: c.Query("customer_id"),
		Page:       page,
		PerPage:    perPage,
	}

	onts, total, err := h.ontService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar ONT"})
	}

	return c.JSON(fiber.Map{
		"data":     onts,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *ONTHandler) GetByCustomer(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	ont, err := h.ontService.GetByCustomerID(c.Context(), tenantID, customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	if ont == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ONT tidak ditemukan untuk pelanggan ini"})
	}

	return c.JSON(ont)
}
