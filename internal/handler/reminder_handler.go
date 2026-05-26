package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type ReminderHandler struct {
	reminderService *service.ReminderService
}

func NewReminderHandler(reminderService *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{reminderService: reminderService}
}

type createReminderRequest struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	DaysOffset      int    `json:"days_offset"`
	MessageTemplate string `json:"message_template"`
	IsActive        bool   `json:"is_active"`
}

type updateReminderRequest struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	DaysOffset      int    `json:"days_offset"`
	MessageTemplate string `json:"message_template"`
	IsActive        bool   `json:"is_active"`
}

func (h *ReminderHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createReminderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Type == "" || req.MessageTemplate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama, tipe, dan message_template wajib diisi"})
	}

	validTypes := map[string]bool{"before_due": true, "on_due": true, "after_due": true}
	if !validTypes[req.Type] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tipe harus before_due, on_due, atau after_due"})
	}

	reminder, err := h.reminderService.Create(c.Context(), service.CreateReminderInput{
		TenantID:        tenantID,
		Name:            req.Name,
		Type:            req.Type,
		DaysOffset:      req.DaysOffset,
		MessageTemplate: req.MessageTemplate,
		IsActive:        req.IsActive,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pengingat"})
	}

	return c.Status(fiber.StatusCreated).JSON(reminder)
}

func (h *ReminderHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	reminderID := c.Params("id")

	reminder, err := h.reminderService.GetByID(c.Context(), tenantID, reminderID)
	if err != nil {
		if errors.Is(err, service.ErrReminderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data pengingat"})
	}

	return c.JSON(reminder)
}

func (h *ReminderHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	reminderID := c.Params("id")

	var req updateReminderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Type == "" || req.MessageTemplate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama, tipe, dan message_template wajib diisi"})
	}

	validTypes := map[string]bool{"before_due": true, "on_due": true, "after_due": true}
	if !validTypes[req.Type] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tipe harus before_due, on_due, atau after_due"})
	}

	reminder, err := h.reminderService.Update(c.Context(), tenantID, reminderID, service.UpdateReminderInput{
		Name:            req.Name,
		Type:            req.Type,
		DaysOffset:      req.DaysOffset,
		MessageTemplate: req.MessageTemplate,
		IsActive:        req.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrReminderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pengingat"})
	}

	return c.JSON(reminder)
}

func (h *ReminderHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	reminderID := c.Params("id")

	if err := h.reminderService.Delete(c.Context(), tenantID, reminderID); err != nil {
		if errors.Is(err, service.ErrReminderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus pengingat"})
	}

	return c.JSON(fiber.Map{"message": "Pengingat dihapus"})
}

func (h *ReminderHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.ReminderFilter{
		Type:    c.Query("type"),
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	reminders, total, err := h.reminderService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar pengingat"})
	}

	return c.JSON(fiber.Map{
		"data":     reminders,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// Trigger processes all active reminders for the tenant and sends WhatsApp messages.
func (h *ReminderHandler) Trigger(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.reminderService.TriggerReminders(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memicu pengingat"})
	}

	return c.JSON(result)
}
