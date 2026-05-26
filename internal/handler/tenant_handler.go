package handler

import (
	"errors"
	"net/mail"
	"regexp"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type TenantHandler struct {
	tenantService *service.TenantService
	webhookURLs   map[string]string
}

func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

func (h *TenantHandler) WithWebhookURLs(urls map[string]string) *TenantHandler {
	h.webhookURLs = urls
	return h
}

type createTenantRequest struct {
	Name                string `json:"name"`
	Slug                string `json:"slug"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	Address             string `json:"address"`
	Timezone            string `json:"timezone"`
	Currency            string `json:"currency"`
	BillingCycle        int    `json:"billing_cycle"`
	DueDay              int    `json:"due_day"`
	IsolirDay           int    `json:"isolir_day"`
	GracePeriod         int    `json:"grace_period"`
	DefaultBillingType  string `json:"default_billing_type"`
	Plan                string `json:"plan"`
	MaxCustomers        int    `json:"max_customers"`
}

type updateTenantRequest struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	Address             string `json:"address"`
	LogoURL             string `json:"logo_url"`
	Timezone            string `json:"timezone"`
	Currency            string `json:"currency"`
	BillingCycle        int    `json:"billing_cycle"`
	DueDay              int    `json:"due_day"`
	IsolirDay           int    `json:"isolir_day"`
	GracePeriod         int    `json:"grace_period"`
	DefaultBillingType  string `json:"default_billing_type"`
	IsActive            bool   `json:"is_active"`
}

type updateSettingsRequest struct {
	WAAPIKey     string `json:"wa_api_key"`
	WASender     string `json:"wa_sender"`
	PGProvider   string `json:"pg_provider"`
	PGAPIKey     string `json:"pg_api_key"`
	PGSecretKey  string `json:"pg_secret_key"`
	PGMerchantID string `json:"pg_merchant_id"`
}

func (h *TenantHandler) Create(c *fiber.Ctx) error {
	var req createTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Slug == "" || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, slug, dan email wajib diisi"})
	}

	if len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama tenant maksimal 100 karakter"})
	}

	if !slugRegex.MatchString(req.Slug) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug hanya boleh huruf kecil, angka, dan tanda hubung"})
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format email tidak valid"})
	}

	if req.BillingCycle < 0 || req.DueDay < 0 || req.DueDay > 31 || req.IsolirDay < 0 || req.GracePeriod < 0 || req.MaxCustomers < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nilai numerik tidak valid"})
	}

	tenant, err := h.tenantService.Create(c.Context(), service.CreateTenantInput{
		Name:                req.Name,
		Slug:                req.Slug,
		Email:               req.Email,
		Phone:               req.Phone,
		Address:             req.Address,
		Timezone:            req.Timezone,
		Currency:            req.Currency,
		BillingCycle:        req.BillingCycle,
		DueDay:              req.DueDay,
		IsolirDay:           req.IsolirDay,
		GracePeriod:         req.GracePeriod,
		DefaultBillingType:  req.DefaultBillingType,
		Plan:                req.Plan,
		MaxCustomers:        req.MaxCustomers,
	})
	if err != nil {
		if errors.Is(err, service.ErrSlugAlreadyUsed) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat tenant"})
	}

	return c.Status(fiber.StatusCreated).JSON(tenant)
}

func (h *TenantHandler) GetByID(c *fiber.Ctx) error {
	tenantID := c.Params("id")

	tenant, err := h.tenantService.GetByID(c.Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data tenant"})
	}

	return c.JSON(tenant)
}

func (h *TenantHandler) GetCurrent(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	tenant, err := h.tenantService.GetByID(c.Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data tenant"})
	}

	// Owner sees their own secrets for settings configuration
	return c.JSON(tenant.ToWithSecrets())
}

func (h *TenantHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req updateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan email wajib diisi"})
	}

	tenant, err := h.tenantService.Update(c.Context(), tenantID, service.UpdateTenantInput{
		Name:                req.Name,
		Email:               req.Email,
		Phone:               req.Phone,
		Address:             req.Address,
		LogoURL:             req.LogoURL,
		Timezone:            req.Timezone,
		Currency:            req.Currency,
		BillingCycle:        req.BillingCycle,
		DueDay:              req.DueDay,
		IsolirDay:           req.IsolirDay,
		GracePeriod:         req.GracePeriod,
		DefaultBillingType:  req.DefaultBillingType,
		IsActive:            req.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui tenant"})
	}

	return c.JSON(tenant)
}

func (h *TenantHandler) UpdateSettings(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req updateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	err := h.tenantService.UpdateSettings(c.Context(), tenantID, service.UpdateSettingsInput{
		WAAPIKey:     req.WAAPIKey,
		WASender:     req.WASender,
		PGProvider:   req.PGProvider,
		PGAPIKey:     req.PGAPIKey,
		PGSecretKey:  req.PGSecretKey,
		PGMerchantID: req.PGMerchantID,
	})
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pengaturan"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan berhasil diperbarui"})
}

// GetWebhookURLs returns the payment gateway callback URLs that the tenant
// should register in their Tripay / Midtrans dashboard.
func (h *TenantHandler) GetWebhookURLs(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data":    h.webhookURLs,
	})
}

func (h *TenantHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.TenantFilter{
		Search:  c.Query("search"),
		Plan:    c.Query("plan"),
		Page:    page,
		PerPage: perPage,
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	tenants, total, err := h.tenantService.List(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar tenant"})
	}

	return c.JSON(fiber.Map{
		"data":     tenants,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
