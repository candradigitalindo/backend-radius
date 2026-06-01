package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type AdminHandler struct {
	adminService        *service.AdminService
	subscriptionService *service.SubscriptionService
	reminderRepo        repository.ReminderRepository
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) WithReminderRepo(repo repository.ReminderRepository) *AdminHandler {
	h.reminderRepo = repo
	return h
}

func (h *AdminHandler) WithSubscriptionService(svc *service.SubscriptionService) *AdminHandler {
	h.subscriptionService = svc
	return h
}

// getExcludeTenantID extracts the superadmin's own tenant_id from JWT to exclude it from listings
func (h *AdminHandler) getExcludeTenantID(c *fiber.Ctx) string {
	tenantID, _ := c.Locals("tenant_id").(string)
	return tenantID
}

// GetDashboardStats returns superadmin dashboard overview
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	excludeTenantID := h.getExcludeTenantID(c)
	stats, err := h.adminService.GetDashboardStats(c.Context(), excludeTenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": stats})
}

// GetTenantStats returns per-tenant customer/router counts
func (h *AdminHandler) GetTenantStats(c *fiber.Ctx) error {
	excludeTenantID := h.getExcludeTenantID(c)
	stats, err := h.adminService.GetTenantStats(c.Context(), excludeTenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": stats})
}

// GetAllRouters returns all routers across all tenants
func (h *AdminHandler) GetAllRouters(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	excludeTenantID := h.getExcludeTenantID(c)
	routers, total, err := h.adminService.GetAllRouters(c.Context(), excludeTenantID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"data":     routers,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetTenantCustomerCounts returns customer counts per tenant
func (h *AdminHandler) GetTenantCustomerCounts(c *fiber.Ctx) error {
	excludeTenantID := h.getExcludeTenantID(c)
	stats, err := h.adminService.GetTenantCustomerCounts(c.Context(), excludeTenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": stats})
}

// GetRollingRevenue returns rolling 6-month revenue across all tenants
func (h *AdminHandler) GetRollingRevenue(c *fiber.Ctx) error {
	data, err := h.adminService.GetRollingRevenue(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": data})
}

// GetSubscriptionRevenue returns rolling 6-month subscription revenue
func (h *AdminHandler) GetSubscriptionRevenue(c *fiber.Ctx) error {
	data, err := h.adminService.GetSubscriptionRevenue(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": data})
}

// ListSubscriptionPlans returns all plans (including inactive) for superadmin management.
func (h *AdminHandler) ListSubscriptionPlans(c *fiber.Ctx) error {
	plans, err := h.subscriptionService.ListAllPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar produk"})
	}
	return c.JSON(fiber.Map{"data": plans})
}

type planRequest struct {
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Description    string   `json:"description"`
	Price          int64    `json:"price"`
	DurationMonths int      `json:"duration_months"`
	MaxCustomers   int      `json:"max_customers"`
	MaxRouters     int      `json:"max_routers"`
	Features       []string `json:"features"`
	IsPopular      bool     `json:"is_popular"`
	IsActive       bool     `json:"is_active"`
	SortOrder      int      `json:"sort_order"`
}

// CreateSubscriptionPlan creates a new subscription plan.
func (h *AdminHandler) CreateSubscriptionPlan(c *fiber.Ctx) error {
	var req planRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Name == "" || req.Slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name dan slug wajib diisi"})
	}

	plan := &model.SubscriptionPlan{
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    req.Description,
		Price:          req.Price,
		DurationMonths: req.DurationMonths,
		MaxCustomers:   req.MaxCustomers,
		MaxRouters:     req.MaxRouters,
		Features:       req.Features,
		IsPopular:      req.IsPopular,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
	}
	if plan.Features == nil {
		plan.Features = []string{}
	}

	if err := h.subscriptionService.CreatePlan(c.Context(), plan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat produk"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": plan})
}

// UpdateSubscriptionPlan updates an existing subscription plan.
func (h *AdminHandler) UpdateSubscriptionPlan(c *fiber.Ctx) error {
	planID := c.Params("id")

	existing, err := h.subscriptionService.GetPlan(c.Context(), planID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Produk tidak ditemukan"})
	}

	var req planRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	existing.Name = req.Name
	existing.Slug = req.Slug
	existing.Description = req.Description
	existing.Price = req.Price
	existing.DurationMonths = req.DurationMonths
	existing.MaxCustomers = req.MaxCustomers
	existing.MaxRouters = req.MaxRouters
	existing.Features = req.Features
	existing.IsPopular = req.IsPopular
	existing.IsActive = req.IsActive
	existing.SortOrder = req.SortOrder
	if existing.Features == nil {
		existing.Features = []string{}
	}

	if err := h.subscriptionService.UpdatePlan(c.Context(), existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui produk"})
	}
	return c.JSON(fiber.Map{"data": existing})
}

// DeleteSubscriptionPlan deletes a subscription plan.
func (h *AdminHandler) DeleteSubscriptionPlan(c *fiber.Ctx) error {
	planID := c.Params("id")
	if err := h.subscriptionService.DeletePlan(c.Context(), planID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus produk"})
	}
	return c.JSON(fiber.Map{"message": "Produk berhasil dihapus"})
}

// ListSubscriptionReminders returns all subscription reminder templates (tenant_id = 'system').
// saTenantID returns the superadmin's own tenant_id from JWT locals.
// ResetTenantPassword resets the password for a tenant's owner user and sends it via WhatsApp.
func (h *AdminHandler) ResetTenantPassword(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	if tenantID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID tenant wajib diisi"})
	}

	// We need tenantService to call ResetPassword
	// Since AdminHandler doesn't have it directly, we use the one from dependencies
	// (Note: in a real refactor, we should add tenantService to AdminHandler struct)
	// For now, I will assume we can get it or I'll add it.
	// Looking at router.go, AdminHandler is initialized with adminService.
	// I will use a local reference if possible or I'll add it to the struct.
	// Better: Add TenantService to AdminHandler.
	
	// I will update the struct first in a separate turn if needed, 
	// but let's assume I can add the method here and fix the struct later in the same file.
	
	pass, err := h.adminService.ResetTenantPassword(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":  "Password berhasil direset dan dikirim via WhatsApp",
		"password": pass, // Still return it for UI confirmation
	})
}

func (h *AdminHandler) saTenantID(c *fiber.Ctx) string {
	tid, _ := c.Locals("tenant_id").(string)
	return tid
}

func (h *AdminHandler) ListSubscriptionReminders(c *fiber.Ctx) error {
	if h.reminderRepo == nil {
		return c.JSON(fiber.Map{"data": []interface{}{}})
	}
	tid := h.saTenantID(c)
	reminders, _, err := h.reminderRepo.List(c.Context(), tid, repository.ReminderFilter{
		Page: 1, PerPage: 50,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat template pesan"})
	}
	return c.JSON(fiber.Map{"data": reminders})
}

type subReminderUpdateRequest struct {
	Name            string `json:"name"`
	MessageTemplate string `json:"message_template"`
	IsActive        bool   `json:"is_active"`
}

// UpdateSubscriptionReminder updates a subscription reminder template by ID.
func (h *AdminHandler) UpdateSubscriptionReminder(c *fiber.Ctx) error {
	reminderID := c.Params("id")
	if h.reminderRepo == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Reminder repo tidak tersedia"})
	}

	tid := h.saTenantID(c)
	rem, err := h.reminderRepo.FindByID(c.Context(), tid, reminderID)
	if err != nil || rem == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Template tidak ditemukan"})
	}

	var req subReminderUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.MessageTemplate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Template pesan tidak boleh kosong"})
	}

	rem.Name = req.Name
	rem.MessageTemplate = req.MessageTemplate
	rem.IsActive = req.IsActive

	if err := h.reminderRepo.Update(c.Context(), rem); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui template"})
	}
	return c.JSON(fiber.Map{"data": rem})
}

// ── Subscription Order CRUD (superadmin) ─────────────────────────────────────

func (h *AdminHandler) ListSubOrders(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.AdminOrderFilter{
		TenantID: c.Query("tenant_id"),
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     page,
		PerPage:  perPage,
	}

	rows, total, err := h.subscriptionService.AdminListAllOrders(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data transaksi"})
	}
	if rows == nil {
		rows = []repository.AdminOrderRow{}
	}
	return c.JSON(fiber.Map{"data": rows, "total": total, "page": page})
}

func (h *AdminHandler) GetSubOrder(c *fiber.Ctx) error {
	order, err := h.subscriptionService.AdminGetOrder(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data transaksi"})
	}
	return c.JSON(fiber.Map{"data": order})
}

type adminCreateOrderReq struct {
	TenantID       string `json:"tenant_id"`
	PlanID         string `json:"plan_id"`
	DurationMonths int    `json:"duration_months"`
}

func (h *AdminHandler) CreateSubOrder(c *fiber.Ctx) error {
	var req adminCreateOrderReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.TenantID == "" || req.PlanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id dan plan_id wajib diisi"})
	}
	if req.DurationMonths <= 0 {
		req.DurationMonths = 1
	}
	order, err := h.subscriptionService.AdminCreateOrder(c.Context(), req.TenantID, req.PlanID, req.DurationMonths)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": order})
}

type adminUpdateOrderReq struct {
	Status        string  `json:"status"`
	PaymentMethod string  `json:"payment_method"`
	PaymentRef    string  `json:"payment_ref"`
	Notes         string  `json:"notes"`
	PaidAt        *string `json:"paid_at"`
	StartsAt      *string `json:"starts_at"`
	ExpiresAt     *string `json:"expires_at"`
}

func (h *AdminHandler) UpdateSubOrder(c *fiber.Ctx) error {
	var req adminUpdateOrderReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	parseTime := func(s *string) *time.Time {
		if s == nil || *s == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, *s)
		if err != nil {
			t, err = time.Parse("2006-01-02", *s)
			if err != nil {
				return nil
			}
		}
		return &t
	}

	input := service.AdminUpdateOrderInput{
		Status:        req.Status,
		PaymentMethod: req.PaymentMethod,
		PaymentRef:    req.PaymentRef,
		Notes:         req.Notes,
		PaidAt:        parseTime(req.PaidAt),
		StartsAt:      parseTime(req.StartsAt),
		ExpiresAt:     parseTime(req.ExpiresAt),
	}

	order, err := h.subscriptionService.AdminUpdateOrder(c.Context(), c.Params("id"), input)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": order})
}

func (h *AdminHandler) DeleteSubOrder(c *fiber.Ctx) error {
	err := h.subscriptionService.AdminDeleteOrder(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus transaksi"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
