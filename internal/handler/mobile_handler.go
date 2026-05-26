package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type MobileHandler struct {
	mobileService *service.MobileService
}

func NewMobileHandler(mobileService *service.MobileService) *MobileHandler {
	return &MobileHandler{mobileService: mobileService}
}

type mobileLoginRequest struct {
	CustomerCode string `json:"customer_code"`
	Password     string `json:"password"`
}

// Login authenticates a customer by customer_code and password.
func (h *MobileHandler) Login(c *fiber.Ctx) error {
	var req mobileLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerCode == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor pelanggan dan password wajib diisi"})
	}

	tokens, customer, err := h.mobileService.Login(c.Context(), req.CustomerCode, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrMobileInvalidPassword) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Nomor pelanggan atau password salah"})
		}
		if errors.Is(err, service.ErrMobileAccountNotActive) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akun belum diaktivasi, silakan hubungi admin atau gunakan lupa password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Login gagal"})
	}

	return c.JSON(fiber.Map{
		"tokens":   tokens,
		"customer": customer,
	})
}

// GetProfile returns the authenticated customer's profile.
func (h *MobileHandler) GetProfile(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	customer, err := h.mobileService.GetProfile(c.Context(), tenantID, customerID)
	if err != nil {
		if errors.Is(err, service.ErrMobileCustomerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pelanggan tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil profil"})
	}

	return c.JSON(fiber.Map{"customer": customer})
}

// GetInvoices returns the authenticated customer's invoices.
func (h *MobileHandler) GetInvoices(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	invoices, total, err := h.mobileService.GetInvoices(c.Context(), tenantID, customerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data faktur"})
	}

	return c.JSON(fiber.Map{
		"invoices": invoices,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetPackages returns available packages for the tenant.
func (h *MobileHandler) GetPackages(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	packages, err := h.mobileService.GetPackages(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data paket"})
	}

	return c.JSON(fiber.Map{"packages": packages})
}

// GetTickets returns the authenticated customer's tickets.
func (h *MobileHandler) GetTickets(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	tickets, total, err := h.mobileService.GetTickets(c.Context(), tenantID, customerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data tiket"})
	}

	return c.JSON(fiber.Map{
		"tickets":  tickets,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetTicket returns a single ticket with its messages.
func (h *MobileHandler) GetTicket(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("id")

	ticket, err := h.mobileService.GetTicket(c.Context(), tenantID, customerID, ticketID)
	if err != nil {
		if errors.Is(err, service.ErrMobileTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tiket tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data tiket"})
	}

	return c.JSON(fiber.Map{"ticket": ticket})
}

type createMobileTicketRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

// CreateTicket creates a support ticket from the mobile app.
func (h *MobileHandler) CreateTicket(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	var req createMobileTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Subject == "" || req.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Subjek dan deskripsi wajib diisi"})
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	category := req.Category
	if category == "" {
		category = "general"
	}

	ticket := &model.Ticket{
		TenantID:    tenantID,
		CustomerID:  customerID,
		Subject:     req.Subject,
		Description: req.Description,
		Category:    category,
		Priority:    priority,
		Status:      "open",
	}

	if err := h.mobileService.CreateTicket(c.Context(), ticket); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat tiket"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ticket": ticket})
}

type addMobileMessageRequest struct {
	Message string `json:"message"`
}

// AddTicketMessage adds a message to a ticket.
func (h *MobileHandler) AddTicketMessage(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("id")

	var req addMobileMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan wajib diisi"})
	}

	msg := &model.TicketMessage{
		TicketID:   ticketID,
		SenderType: "customer",
		SenderID:   customerID,
		Message:    req.Message,
	}

	if err := h.mobileService.AddTicketMessage(c.Context(), tenantID, customerID, msg); err != nil {
		if errors.Is(err, service.ErrMobileTicketNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tiket tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan pesan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": msg})
}

type upgradeRequest struct {
	PackageID string `json:"package_id"`
	Notes     string `json:"notes"`
}

// SubmitUpgradeRequest creates a ticket for package upgrade.
func (h *MobileHandler) SubmitUpgradeRequest(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	var req upgradeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PackageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "package_id wajib diisi"})
	}

	if err := h.mobileService.SubmitUpgradeRequest(c.Context(), tenantID, customerID, req.PackageID, req.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim permintaan upgrade"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Permintaan upgrade terkirim"})
}
