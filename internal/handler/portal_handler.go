package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/token"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

// PortalHandler serves customer portal web endpoints.
// Customer authentication is fully separated from the staff/admin users table.
type PortalHandler struct {
	customerRepo   repository.CustomerRepository
	invoiceRepo    repository.InvoiceRepository
	ticketRepo     repository.TicketRepository
	paymentRepo    repository.PaymentRepository
	tenantRepo     repository.TenantRepository
	reminderRepo   repository.ReminderRepository
	ontRepo        repository.ONTRepository
	packageRepo    repository.PackageRepository
	rewardRepo     repository.RewardRepository
	invoiceService *service.InvoiceService
	genieacsSvc    GenieACSProvisioner
	tokenManager   *token.Manager
	redis          *redis.Client
	waClient       *whatsapp.Client
}

// GenieACSProvisioner is a minimal interface for portal device management.
type GenieACSProvisioner interface {
	GetDeviceStatus(ctx context.Context, tenantID, ontID string) (*service.DeviceInfo, error)
	SetWiFiSSID(ctx context.Context, tenantID, ontID, ssid, password, security string) error
	RebootDevice(ctx context.Context, tenantID, ontID string) error
}

func NewPortalHandler(
	customerRepo repository.CustomerRepository,
	invoiceRepo repository.InvoiceRepository,
	ticketRepo repository.TicketRepository,
	paymentRepo repository.PaymentRepository,
	tenantRepo repository.TenantRepository,
	tokenManager *token.Manager,
) *PortalHandler {
	return &PortalHandler{
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
		ticketRepo:   ticketRepo,
		paymentRepo:  paymentRepo,
		tenantRepo:   tenantRepo,
		tokenManager: tokenManager,
	}
}

// WithResetDeps injects Redis and WhatsApp client for password reset feature.
func (h *PortalHandler) WithResetDeps(rdb *redis.Client, wa *whatsapp.Client) *PortalHandler {
	h.redis = rdb
	h.waClient = wa
	return h
}

// WithDeviceService injects ONT repo and GenieACS service for portal device endpoints.
func (h *PortalHandler) WithDeviceService(ontRepo repository.ONTRepository, genie GenieACSProvisioner) *PortalHandler {
	h.ontRepo = ontRepo
	h.genieacsSvc = genie
	return h
}

// WithPackageRepo injects the package repository for listing packages in portal.
func (h *PortalHandler) WithPackageRepo(repo repository.PackageRepository) *PortalHandler {
	h.packageRepo = repo
	return h
}

// WithRewardRepo injects the reward repository for referral info in portal.
func (h *PortalHandler) WithRewardRepo(repo repository.RewardRepository) *PortalHandler {
	h.rewardRepo = repo
	return h
}

// WithInvoiceService injects the invoice service for gateway payments in portal.
func (h *PortalHandler) WithInvoiceService(svc *service.InvoiceService) *PortalHandler {
	h.invoiceService = svc
	return h
}

// GetDevice returns the customer's ONT status including WiFi info and connected hosts.
func (h *PortalHandler) GetDevice(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	if h.ontRepo == nil {
		return c.JSON(fiber.Map{"data": nil})
	}
	tenantID, _ := c.Locals("tenant_id").(string)
	ont, err := h.ontRepo.FindByCustomerID(c.Context(), tenantID, customer.ID)
	if err != nil || ont == nil {
		return c.JSON(fiber.Map{"data": nil})
	}
	if h.genieacsSvc == nil {
		return c.JSON(fiber.Map{"data": ont})
	}
	status, err := h.genieacsSvc.GetDeviceStatus(c.Context(), tenantID, ont.ID)
	if err != nil || status == nil {
		return c.JSON(fiber.Map{"data": ont})
	}
	return c.JSON(fiber.Map{"data": ont, "status": status})
}

// SetDeviceWiFi allows a customer to change their WiFi SSID and password.
func (h *PortalHandler) SetDeviceWiFi(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	tenantID, _ := c.Locals("tenant_id").(string)

	var req struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.SSID == "" || len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "SSID wajib diisi dan password minimal 8 karakter"})
	}

	ont, err := h.ontRepo.FindByCustomerID(c.Context(), tenantID, customer.ID)
	if err != nil || ont == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
	}
	if h.genieacsSvc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Layanan perangkat tidak tersedia"})
	}
	if err := h.genieacsSvc.SetWiFiSSID(c.Context(), tenantID, ont.ID, req.SSID, req.Password, "11i"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah WiFi: " + err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Pengaturan WiFi berhasil diperbarui"})
}

// RebootDevice allows a customer to restart their ONT/router.
func (h *PortalHandler) RebootDevice(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	tenantID, _ := c.Locals("tenant_id").(string)

	ont, err := h.ontRepo.FindByCustomerID(c.Context(), tenantID, customer.ID)
	if err != nil || ont == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
	}
	if h.genieacsSvc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Layanan perangkat tidak tersedia"})
	}
	if err := h.genieacsSvc.RebootDevice(c.Context(), tenantID, ont.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal merestart perangkat: " + err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Perangkat sedang direstart, mohon tunggu beberapa saat"})
}

func (h *PortalHandler) WithReminderRepo(repo repository.ReminderRepository) *PortalHandler {
	h.reminderRepo = repo
	return h
}

// otpTemplate returns the OTP WA message template from superadmin tenant reminders.
func (h *PortalHandler) otpTemplate(ctx context.Context) string {
	defaultTpl := "🔐 *Kode OTP Reset Password - D Radius*\n\n" +
		"Halo *{nama}*,\n\n" +
		"Kami menerima permintaan reset password untuk akun pelanggan *{nama_isp}* Anda.\n\n" +
		"🔑 Kode OTP Anda:\n\n" +
		"*{kode_otp}*\n\n" +
		"⏱ Berlaku selama *{durasi}*\n" +
		"⚠️ Jangan bagikan kode ini kepada siapapun.\n\n" +
		"Jika Anda tidak merasa meminta reset password, abaikan pesan ini.\n\n" +
		"Terima kasih,\n" +
		"_Tim D Radius_"

	if h.reminderRepo == nil || h.tenantRepo == nil {
		return defaultTpl
	}
	saT, err := h.tenantRepo.FindBySlug(ctx, "superadmin")
	if err != nil || saT == nil {
		return defaultTpl
	}
	rem, err := h.reminderRepo.FindActiveByType(ctx, saT.ID, "otp_reset_password")
	if err != nil || rem == nil {
		return defaultTpl
	}
	return rem.MessageTemplate
}

// resolveTenant looks up a tenant by slug first, then falls back to ID lookup.
func (h *PortalHandler) resolveTenant(c *fiber.Ctx, slugOrID string) (*model.Tenant, error) {
	tenant, err := h.tenantRepo.FindBySlug(c.Context(), slugOrID)
	if err == nil && tenant != nil {
		return tenant, nil
	}
	return h.tenantRepo.FindByID(c.Context(), slugOrID)
}

// GetTenantInfo returns public tenant info (name, logo, slug) for the portal login page.
// GET /api/v1/public/portal/:slug
func (h *PortalHandler) GetTenantInfo(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug tidak valid"})
	}

	tenant, err := h.resolveTenant(c, slug)
	if err != nil || tenant == nil || !tenant.IsActive {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Halaman tidak ditemukan"})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"name":     tenant.Name,
			"slug":     tenant.Slug,
			"logo_url": tenant.LogoURL,
		},
	})
}

// PortalLogin authenticates a customer-role user via tenant slug.
// POST /api/v1/public/portal/:slug/login
func (h *PortalHandler) PortalLogin(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug tidak valid"})
	}

	tenant, err := h.resolveTenant(c, slug)
	if err != nil || tenant == nil || !tenant.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Nomor pelanggan atau password salah"})
	}

	var req struct {
		CustomerCode string `json:"customer_code"`
		Password     string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.CustomerCode == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor pelanggan dan password wajib diisi"})
	}

	// Look up customer by customer code within tenant
	customer, err := h.customerRepo.FindByCode(c.Context(), tenant.ID, req.CustomerCode)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Nomor pelanggan atau password salah"})
	}

	// Customer must have a password set
	if customer.PasswordHash == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Akun belum diaktifkan, silakan gunakan fitur Lupa Password"})
	}

	// Verify password against customer's own password_hash
	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Nomor pelanggan atau password salah"})
	}

	// Generate JWT token with customer identity (role = "customer")
	pair, err := h.tokenManager.GeneratePair(token.Claims{
		UserID:   customer.ID,
		TenantID: customer.TenantID,
		Role:     "customer",
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat token"})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":            customer.ID,
			"tenant_id":     customer.TenantID,
			"name":          customer.Name,
			"email":         customer.Email,
			"phone":         customer.Phone,
			"role":          "customer",
			"customer_code": customer.CustomerCode,
		},
		"token": pair,
	})
}

// resolveCustomer finds the customer from the JWT token claims.
// In portal context, user_id IS the customer ID.
func (h *PortalHandler) resolveCustomer(c *fiber.Ctx) (*model.Customer, error) {
	customerID, _ := c.Locals("user_id").(string)
	tenantID, _ := c.Locals("tenant_id").(string)

	customer, err := h.customerRepo.FindByID(c.Context(), tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, fmt.Errorf("data pelanggan tidak ditemukan")
	}

	return customer, nil
}

// GetProfile returns the portal customer's profile data.
func (h *PortalHandler) GetProfile(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":            customer.ID,
			"tenant_id":     customer.TenantID,
			"name":          customer.Name,
			"email":         customer.Email,
			"phone":         customer.Phone,
			"role":          "customer",
			"customer_code": customer.CustomerCode,
			"customer":      customer,
		},
	})
}

// GetCustomer returns the customer record linked to the current user.
func (h *PortalHandler) GetCustomer(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"data": customer})
}

// ListInvoices returns invoices for the current customer.
func (h *PortalHandler) ListInvoices(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))

	invoices, total, err := h.invoiceRepo.ListByCustomer(c.Context(), tenantID, customer.ID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat tagihan"})
	}

	return c.JSON(fiber.Map{
		"data":     invoices,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetInvoice returns a single invoice detail (only if it belongs to the current customer).
func (h *PortalHandler) GetInvoice(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	invoice, err := h.invoiceRepo.FindByID(c.Context(), tenantID, invoiceID)
	if err != nil || invoice == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tagihan tidak ditemukan"})
	}

	// Ownership check
	if invoice.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	return c.JSON(fiber.Map{"data": invoice})
}

// GetInvoicePayments returns payment history for an invoice (only if it belongs to the current customer).
func (h *PortalHandler) GetInvoicePayments(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	// Verify invoice belongs to customer
	invoice, err := h.invoiceRepo.FindByID(c.Context(), tenantID, invoiceID)
	if err != nil || invoice == nil || invoice.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	payments, err := h.paymentRepo.ListByInvoice(c.Context(), tenantID, invoiceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pembayaran"})
	}

	return c.JSON(fiber.Map{"data": payments})
}

// ListTickets returns tickets for the current customer.
func (h *PortalHandler) ListTickets(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))

	tickets, total, err := h.ticketRepo.List(c.Context(), tenantID, repository.TicketFilter{
		CustomerID: customer.ID,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat tiket"})
	}

	return c.JSON(fiber.Map{
		"data":     tickets,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetTicket returns a single ticket with messages (only if it belongs to the current customer).
func (h *PortalHandler) GetTicket(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	ticket, err := h.ticketRepo.FindByID(c.Context(), tenantID, ticketID)
	if err != nil || ticket == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tiket tidak ditemukan"})
	}

	// Ownership check
	if ticket.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	msgs, err := h.ticketRepo.ListMessages(c.Context(), ticketID)
	if err == nil {
		ticket.Messages = msgs
	}

	return c.JSON(fiber.Map{"data": ticket})
}

// CreateTicket creates a ticket for the current customer.
func (h *PortalHandler) CreateTicket(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)

	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
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

	ticket := &model.Ticket{
		TenantID:    tenantID,
		CustomerID:  customer.ID,
		Subject:     req.Subject,
		Description: req.Description,
		Category:    "general",
		Priority:    priority,
		Status:      "open",
	}

	if err := h.ticketRepo.Create(c.Context(), ticket); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat tiket"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": ticket})
}

// GetTicketMessages returns messages for a ticket (only if it belongs to the current customer).
func (h *PortalHandler) GetTicketMessages(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	ticket, err := h.ticketRepo.FindByID(c.Context(), tenantID, ticketID)
	if err != nil || ticket == nil || ticket.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	msgs, err := h.ticketRepo.ListMessages(c.Context(), ticketID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pesan"})
	}

	return c.JSON(fiber.Map{"data": msgs})
}

// ReplyTicket adds a message to a ticket.
func (h *PortalHandler) ReplyTicket(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	ticketID := c.Params("id")

	// Ownership check
	ticket, err := h.ticketRepo.FindByID(c.Context(), tenantID, ticketID)
	if err != nil || ticket == nil || ticket.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	if ticket.Status == "closed" || ticket.Status == "resolved" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tiket sudah ditutup"})
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan wajib diisi"})
	}

	msg := &model.TicketMessage{
		TicketID:   ticketID,
		SenderType: "customer",
		SenderID:   customer.ID,
		Message:    req.Message,
	}

	if err := h.ticketRepo.AddMessage(c.Context(), msg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim pesan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": msg})
}

// ChangePassword allows the customer to change their login password.
func (h *PortalHandler) ChangePassword(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi"})
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.NewPassword == "" || len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru minimal 6 karakter"})
	}

	// Verify current password against customer's own password_hash
	if customer.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password saat ini salah"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah password"})
	}

	if err := h.customerRepo.UpdatePasswordHash(c.Context(), customer.TenantID, customer.ID, string(hash)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah password"})
	}

	return c.JSON(fiber.Map{"message": "Password berhasil diubah"})
}

// generatePIN creates a cryptographically secure 6-digit PIN.
func generatePIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// RequestResetPIN generates a 6-digit PIN, stores it in Redis, and sends it via WhatsApp.
// POST /api/v1/public/portal/:slug/reset-pin
func (h *PortalHandler) RequestResetPIN(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug tidak valid"})
	}

	tenant, err := h.resolveTenant(c, slug)
	if err != nil || tenant == nil || !tenant.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tenant tidak ditemukan"})
	}

	var req struct {
		CustomerCode string `json:"customer_code"`
	}
	if err := c.BodyParser(&req); err != nil || req.CustomerCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor pelanggan wajib diisi"})
	}

	customer, err := h.customerRepo.FindByCode(c.Context(), tenant.ID, req.CustomerCode)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor pelanggan tidak ditemukan"})
	}

	if customer.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor WhatsApp belum terdaftar pada akun Anda. Hubungi admin untuk menambahkan nomor HP."})
	}

	// Email is optional: portal login only needs customer_code + password, and the
	// OTP is delivered via WhatsApp. ISP customers often have no email, so do not
	// gate password reset on it.

	// Rate limit: check if a PIN was already sent recently
	redisKey := fmt.Sprintf("portal_reset:%s:%s", tenant.ID, req.CustomerCode)
	existing, _ := h.redis.Get(c.Context(), redisKey).Result()
	if existing != "" {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "PIN sudah dikirim, tunggu beberapa menit sebelum meminta ulang",
		})
	}

	ctx := context.Background()

	pin, err := generatePIN()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat PIN"})
	}

	// Store PIN in Redis with 5-minute TTL
	if err := h.redis.Set(c.Context(), redisKey, pin, 5*time.Minute).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan PIN"})
	}

	// Build message from template — always send via superadmin WA
	tpl := h.otpTemplate(ctx)
	msg := strings.ReplaceAll(tpl, "{nama}", customer.Name)
	msg = strings.ReplaceAll(msg, "{kode_otp}", pin)
	msg = strings.ReplaceAll(msg, "{nama_isp}", tenant.Name)
	msg = strings.ReplaceAll(msg, "{durasi}", "5 menit")

	_, waErr := h.waClient.SendMessage(ctx, "superadmin", customer.Phone, msg)
	if waErr != nil {
		h.redis.Del(c.Context(), redisKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim OTP melalui WhatsApp"})
	}

	// Mask phone number for display: show first 4 and last 4 digits
	phone := customer.Phone
	masked := phone
	if len(phone) > 8 {
		masked = phone[:4] + strings.Repeat("*", len(phone)-8) + phone[len(phone)-4:]
	}

	return c.JSON(fiber.Map{
		"message": "PIN telah dikirim melalui WhatsApp",
		"phone":   masked,
	})
}

// ResetPasswordWithPIN verifies the PIN and resets the customer's password.
// POST /api/v1/public/portal/:slug/reset-password
func (h *PortalHandler) ResetPasswordWithPIN(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug tidak valid"})
	}

	tenant, err := h.resolveTenant(c, slug)
	if err != nil || tenant == nil || !tenant.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tenant tidak ditemukan"})
	}

	var req struct {
		CustomerCode string `json:"customer_code"`
		PIN          string `json:"pin"`
		NewPassword  string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.CustomerCode == "" || req.PIN == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Semua field wajib diisi"})
	}
	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru minimal 6 karakter"})
	}

	// Verify PIN from Redis
	redisKey := fmt.Sprintf("portal_reset:%s:%s", tenant.ID, req.CustomerCode)
	attemptsKey := redisKey + ":attempts"

	// Check brute-force: max 5 attempts
	attempts, _ := h.redis.Get(c.Context(), attemptsKey).Int()
	if attempts >= 5 {
		// Too many failed attempts — delete PIN entirely
		h.redis.Del(c.Context(), redisKey)
		h.redis.Del(c.Context(), attemptsKey)
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Terlalu banyak percobaan. PIN telah dibatalkan. Silakan minta PIN baru.",
		})
	}

	storedPIN, err := h.redis.Get(c.Context(), redisKey).Result()
	if err != nil || storedPIN == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "PIN tidak valid atau sudah kedaluwarsa"})
	}

	if storedPIN != req.PIN {
		// Increment failed attempts counter (same TTL as PIN)
		h.redis.Incr(c.Context(), attemptsKey)
		h.redis.Expire(c.Context(), attemptsKey, 5*time.Minute)
		remaining := 4 - attempts
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("PIN salah. Sisa percobaan: %d", remaining),
		})
	}

	// PIN matched — clean up attempts counter
	h.redis.Del(c.Context(), attemptsKey)

	// Look up customer
	customer, err := h.customerRepo.FindByCode(c.Context(), tenant.ID, req.CustomerCode)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pelanggan tidak ditemukan"})
	}

	// Hash new password and update customer record directly
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mereset password"})
	}
	if err := h.customerRepo.UpdatePasswordHash(c.Context(), tenant.ID, customer.ID, string(hash)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mereset password"})
	}

	// Delete the used PIN
	h.redis.Del(c.Context(), redisKey)

	return c.JSON(fiber.Map{"message": "Password berhasil direset. Silakan login dengan password baru."})
}

// GetAvailablePackages returns all active packages the customer can upgrade/downgrade to.
// GET /portal/packages
func (h *PortalHandler) GetAvailablePackages(c *fiber.Ctx) error {
	if h.packageRepo == nil {
		return c.JSON(fiber.Map{"data": []interface{}{}})
	}
	tenantID, _ := c.Locals("tenant_id").(string)
	active := true
	pkgs, _, err := h.packageRepo.List(c.Context(), tenantID, repository.PackageFilter{
		Active:  &active,
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat paket"})
	}
	return c.JSON(fiber.Map{"data": pkgs})
}

// RequestPackageChange submits a package change request (upgrade/downgrade) as a ticket.
// POST /portal/change-package
func (h *PortalHandler) RequestPackageChange(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	tenantID, _ := c.Locals("tenant_id").(string)

	var req struct {
		PackageID   string `json:"package_id"`
		PackageName string `json:"package_name"`
		ChangeType  string `json:"change_type"` // upgrade or downgrade
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.PackageID == "" || req.PackageName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Paket tujuan wajib dipilih"})
	}
	changeTypeLabel := "Upgrade"
	if req.ChangeType == "downgrade" {
		changeTypeLabel = "Downgrade"
	}
	subject := fmt.Sprintf("[Permintaan %s Paket] ke %s", changeTypeLabel, req.PackageName)
	desc := fmt.Sprintf("Pelanggan mengajukan permintaan %s paket ke: %s (ID: %s).",
		strings.ToLower(changeTypeLabel), req.PackageName, req.PackageID)
	if req.Notes != "" {
		desc += "\n\nCatatan: " + req.Notes
	}

	ticket := &model.Ticket{
		TenantID:    tenantID,
		CustomerID:  customer.ID,
		Subject:     subject,
		Description: desc,
		Category:    "billing",
		Priority:    "medium",
		Status:      "open",
	}
	if err := h.ticketRepo.Create(c.Context(), ticket); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat permintaan perubahan paket"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Permintaan perubahan paket telah dikirim. Tim kami akan menghubungi Anda segera.",
		"ticket":  ticket,
	})
}

// GetReferralInfo returns the customer's referral code, referral history, and reward balance.
// GET /portal/referral
func (h *PortalHandler) GetReferralInfo(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	tenantID, _ := c.Locals("tenant_id").(string)

	resp := fiber.Map{
		"referral_code": customer.ReferralCode,
		"referrals":     []interface{}{},
		"balance":       int64(0),
		"claims":        []interface{}{},
	}

	if h.rewardRepo == nil {
		return c.JSON(fiber.Map{"data": resp})
	}

	refs, _, _ := h.rewardRepo.ListReferrals(c.Context(), tenantID, repository.ReferralFilter{
		CustomerID: customer.ID,
		Page:       1,
		PerPage:    50,
	})
	if refs != nil {
		resp["referrals"] = refs
	}

	balance, _ := h.rewardRepo.GetCustomerBalance(c.Context(), tenantID, customer.ID)
	resp["balance"] = balance

	claims, _, _ := h.rewardRepo.ListClaims(c.Context(), tenantID, repository.ClaimFilter{
		CustomerID: customer.ID,
		Page:       1,
		PerPage:    20,
	})
	if claims != nil {
		resp["claims"] = claims
	}

	return c.JSON(fiber.Map{"data": resp})
}

// PayInvoiceGateway initiates an online payment for an invoice via the tenant's payment gateway.
// POST /portal/invoices/:id/pay-gateway
func (h *PortalHandler) PayInvoiceGateway(c *fiber.Ctx) error {
	customer, err := h.resolveCustomer(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data pelanggan tidak ditemukan"})
	}
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	invoice, err := h.invoiceRepo.FindByID(c.Context(), tenantID, invoiceID)
	if err != nil || invoice == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tagihan tidak ditemukan"})
	}
	if invoice.CustomerID != customer.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}
	if invoice.Status == "paid" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Tagihan sudah dibayar"})
	}
	if h.invoiceService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Layanan pembayaran online tidak tersedia"})
	}

	var req struct {
		PaymentMethod string `json:"payment_method"`
		ReturnURL     string `json:"return_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "qris"
	}

	result, err := h.invoiceService.CreateGatewayPayment(c.Context(), service.PaymentGatewayInput{
		TenantID:      tenantID,
		InvoiceID:     invoiceID,
		PaymentMethod: req.PaymentMethod,
		CustomerName:  customer.Name,
		CustomerEmail: customer.Email,
		CustomerPhone: customer.Phone,
		ReturnURL:     req.ReturnURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrPaymentGatewayInactive) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pembayaran online belum aktif untuk tenant ini"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat link pembayaran: " + err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": result})
}

// GetPaymentConfig returns whether online payment is available for this tenant.
// Available = gateway configured (PGAPIKey set) AND in production mode (PGSandbox = false).
// GET /portal/payment-config
func (h *PortalHandler) GetPaymentConfig(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	tenant, err := h.tenantRepo.FindByID(c.Context(), tenantID)
	if err != nil || tenant == nil {
		return c.JSON(fiber.Map{"data": fiber.Map{"available": false, "provider": ""}})
	}
	return c.JSON(fiber.Map{"data": fiber.Map{
		"available": tenant.IsOnlinePaymentActive(),
		"provider":  tenant.PGProvider,
		"sandbox":   tenant.PGSandbox,
	}})
}
