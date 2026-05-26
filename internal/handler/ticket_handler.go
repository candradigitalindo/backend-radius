package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type TicketHandler struct {
	ticketService *service.TicketService
}

func NewTicketHandler(ticketService *service.TicketService) *TicketHandler {
	return &TicketHandler{ticketService: ticketService}
}

type createTicketRequest struct {
	CustomerID  string `json:"customer_id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

type updateTicketRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type assignTicketRequest struct {
	AssignedTo string `json:"assigned_to"`
}

type addMessageRequest struct {
	Message       string `json:"message"`
	AttachmentURL string `json:"attachment_url"`
}

func (h *TicketHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerID == "" || req.Subject == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id dan subjek wajib diisi"})
	}

	ticket, err := h.ticketService.Create(c.Context(), service.CreateTicketInput{
		TenantID:    tenantID,
		CustomerID:  req.CustomerID,
		Subject:     req.Subject,
		Description: req.Description,
		Category:    req.Category,
		Priority:    req.Priority,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat tiket"})
	}

	return c.Status(fiber.StatusCreated).JSON(ticket)
}

func (h *TicketHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	ticket, err := h.ticketService.GetByID(c.Context(), tenantID, ticketID)
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(ticket)
}

func (h *TicketHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	var req updateTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Subject == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Subjek wajib diisi"})
	}

	ticket, err := h.ticketService.Update(c.Context(), tenantID, ticketID, service.UpdateTicketInput{
		Subject:     req.Subject,
		Description: req.Description,
		Category:    req.Category,
		Priority:    req.Priority,
	})
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrTicketAlreadyClosed) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(ticket)
}

func (h *TicketHandler) UpdateStatus(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	var req updateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	validStatuses := map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true}
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Status harus salah satu dari: open, in_progress, resolved, closed"})
	}

	err := h.ticketService.UpdateStatus(c.Context(), tenantID, ticketID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrTicketAlreadyClosed) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Status berhasil diperbarui"})
}

func (h *TicketHandler) Assign(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	var req assignTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.AssignedTo == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "assigned_to wajib diisi"})
	}

	err := h.ticketService.Assign(c.Context(), tenantID, ticketID, req.AssignedTo)
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Tiket berhasil ditugaskan"})
}

func (h *TicketHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	if err := h.ticketService.Delete(c.Context(), tenantID, ticketID); err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Tiket berhasil dihapus"})
}

func (h *TicketHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.TicketFilter{
		Search:     c.Query("search"),
		CustomerID: c.Query("customer_id"),
		Status:     c.Query("status"),
		Category:   c.Query("category"),
		Priority:   c.Query("priority"),
		AssignedTo: c.Query("assigned_to"),
		Page:       page,
		PerPage:    perPage,
	}

	tickets, total, err := h.ticketService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar tiket"})
	}

	return c.JSON(fiber.Map{
		"data":     tickets,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *TicketHandler) AddMessage(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("id")

	var req addMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan wajib diisi"})
	}

	msg, err := h.ticketService.AddMessage(c.Context(), service.AddMessageInput{
		TenantID:      tenantID,
		TicketID:      ticketID,
		SenderType:    "staff",
		SenderID:      userID,
		Message:       req.Message,
		AttachmentURL: req.AttachmentURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrTicketAlreadyClosed) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan pesan"})
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

func (h *TicketHandler) GetMessages(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	messages, err := h.ticketService.GetMessages(c.Context(), tenantID, ticketID)
	if err != nil {
		if errors.Is(err, service.ErrTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": messages})
}
