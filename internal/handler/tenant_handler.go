package handler

import (
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type TenantHandler struct {
	tenantService  *service.TenantService
	authService    *service.AuthService
	auditLogRepo   repository.AuditLogRepository
	webhookBaseURL string
}

func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

func (h *TenantHandler) WithWebhookBaseURL(baseURL string) *TenantHandler {
	h.webhookBaseURL = baseURL
	return h
}

func (h *TenantHandler) WithAuthService(authService *service.AuthService) *TenantHandler {
	h.authService = authService
	return h
}

func (h *TenantHandler) WithAuditLogRepo(repo repository.AuditLogRepository) *TenantHandler {
	h.auditLogRepo = repo
	return h
}

type createTenantRequest struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Address            string `json:"address"`
	Timezone           string `json:"timezone"`
	Currency           string `json:"currency"`
	BillingCycle       int    `json:"billing_cycle"`
	DueDay             int    `json:"due_day"`
	IsolirDay          int    `json:"isolir_day"`
	GracePeriod        int    `json:"grace_period"`
	DefaultBillingType string `json:"default_billing_type"`
	Plan               string `json:"plan"`
	MaxCustomers       int    `json:"max_customers"`
	Fingerprint        string `json:"fingerprint"`
}

type updateTenantRequest struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Address              string `json:"address"`
	LogoURL              string `json:"logo_url"`
	Timezone             string `json:"timezone"`
	Currency             string `json:"currency"`
	BillingCycle         int    `json:"billing_cycle"`
	DueDay               int    `json:"due_day"`
	IsolirDay            int    `json:"isolir_day"`
	GracePeriod          int    `json:"grace_period"`
	DefaultBillingType   string `json:"default_billing_type"`
	DefaultPaymentTiming string `json:"default_payment_timing"`
	IsActive             bool   `json:"is_active"`
}

type updateSettingsRequest struct {
	WAAPIKey     string `json:"wa_api_key"`
	WASender     string `json:"wa_sender"`
	PGProvider   string `json:"pg_provider"`
	PGAPIKey     string `json:"pg_api_key"`
	PGSecretKey  string `json:"pg_secret_key"`
	PGMerchantID string `json:"pg_merchant_id"`
	PGSandbox    bool   `json:"pg_sandbox"`
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
		Name:               req.Name,
		Slug:               req.Slug,
		Email:              req.Email,
		Phone:              req.Phone,
		Address:            req.Address,
		Timezone:           req.Timezone,
		Currency:           req.Currency,
		BillingCycle:       req.BillingCycle,
		DueDay:             req.DueDay,
		IsolirDay:          req.IsolirDay,
		GracePeriod:        req.GracePeriod,
		DefaultBillingType: req.DefaultBillingType,
		Plan:               req.Plan,
		MaxCustomers:       req.MaxCustomers,
		Fingerprint:        req.Fingerprint,
		RegistrationIP:     c.IP(),
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
		Name:                 req.Name,
		Email:                req.Email,
		Phone:                req.Phone,
		Address:              req.Address,
		LogoURL:              req.LogoURL,
		Timezone:             req.Timezone,
		Currency:             req.Currency,
		BillingCycle:         req.BillingCycle,
		DueDay:               req.DueDay,
		IsolirDay:            req.IsolirDay,
		GracePeriod:          req.GracePeriod,
		DefaultBillingType:   req.DefaultBillingType,
		DefaultPaymentTiming: req.DefaultPaymentTiming,
		IsActive:             req.IsActive,
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
		PGSandbox:    req.PGSandbox,
	})
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pengaturan"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan berhasil diperbarui"})
}

// testPGRequest holds credentials to test directly (without saving to DB first).
type testPGRequest struct {
	PGProvider   string `json:"pg_provider"`
	PGAPIKey     string `json:"pg_api_key"`
	PGSecretKey  string `json:"pg_secret_key"`
	PGMerchantID string `json:"pg_merchant_id"`
	PGSandbox    bool   `json:"pg_sandbox"`
}

// TestPGConnection tests the payment gateway credentials.
// If the request body contains a pg_provider, those credentials are tested directly.
// Otherwise the credentials stored in the database are used.
func (h *TenantHandler) TestPGConnection(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req testPGRequest
	_ = c.BodyParser(&req)

	var override *service.PGCredentials
	if req.PGProvider != "" {
		override = &service.PGCredentials{
			Provider:   req.PGProvider,
			APIKey:     req.PGAPIKey,
			SecretKey:  req.PGSecretKey,
			MerchantID: req.PGMerchantID,
			Sandbox:    req.PGSandbox,
		}
	}

	if err := h.tenantService.TestPGConnection(c.Context(), tenantID, override); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Koneksi payment gateway berhasil",
	})
}

// GetWebhookURLs returns the per-tenant payment gateway callback URLs that the
// tenant should register in their Tripay / Midtrans / Xendit dashboard.
func (h *TenantHandler) GetWebhookURLs(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	tenant, err := h.tenantService.GetByID(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data tenant"})
	}
	slug := tenant.Slug
	base := h.webhookBaseURL
	return c.JSON(fiber.Map{
		"success": true,
		"data": map[string]string{
			"tripay":                base + "/api/v1/webhooks/tripay/" + slug,
			"midtrans":              base + "/api/v1/webhooks/midtrans/" + slug,
			"xendit":                base + "/api/v1/webhooks/xendit/" + slug,
			"tripay_voucher":        base + "/api/v1/webhooks/tripay/" + slug + "/voucher",
			"midtrans_voucher":      base + "/api/v1/webhooks/midtrans/" + slug + "/voucher",
			"xendit_voucher":        base + "/api/v1/webhooks/xendit/" + slug + "/voucher",
			"tripay_subscription":   base + "/api/v1/webhooks/subscription/tripay",
			"midtrans_subscription": base + "/api/v1/webhooks/subscription/midtrans",
			"xendit_subscription":   base + "/api/v1/webhooks/subscription/xendit",
		},
	})
}

type adminUpdateTenantRequest struct {
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Address       string  `json:"address"`
	IsActive      bool    `json:"is_active"`
	Status        string  `json:"status"`
	Plan          string  `json:"plan"`
	PlanExpiresAt *string `json:"plan_expires_at"`
	MaxCustomers  int     `json:"max_customers"`
}

func (h *TenantHandler) AdminUpdate(c *fiber.Ctx) error {
	tenantID := c.Params("id")

	var req adminUpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan email wajib diisi"})
	}

	// Paket "trial" hanya diberikan saat pembuatan tenant — tidak bisa di-assign ulang
	if req.Plan == "trial" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Paket trial tidak dapat di-assign ulang. Pilih paket gratis atau paket berbayar."})
	}

	var planExpiresAt *time.Time
	if req.PlanExpiresAt != nil && *req.PlanExpiresAt != "" {
		t, err := time.Parse("2006-01-02", *req.PlanExpiresAt)
		if err == nil {
			planExpiresAt = &t
		}
	}

	tenant, err := h.tenantService.AdminUpdate(c.Context(), tenantID, service.AdminUpdateTenantInput{
		Name:          req.Name,
		Email:         req.Email,
		Phone:         req.Phone,
		Address:       req.Address,
		IsActive:      req.IsActive,
		Status:        req.Status,
		Plan:          req.Plan,
		PlanExpiresAt: planExpiresAt,
		MaxCustomers:  req.MaxCustomers,
	})
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui tenant"})
	}

	return c.JSON(fiber.Map{"data": tenant})
}

func (h *TenantHandler) Delete(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	if err := h.tenantService.Delete(c.Context(), tenantID); err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus tenant"})
	}
	return c.JSON(fiber.Map{"message": "Tenant berhasil dihapus"})
}

// POST /api/v1/tenants/:id/impersonate — superadmin logs in as this tenant's
// owner to browse the app exactly as the tenant would see it.
func (h *TenantHandler) Impersonate(c *fiber.Ctx) error {
	if h.authService == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Layanan autentikasi tidak tersedia"})
	}

	tenantID := c.Params("id")
	actorID, _ := c.Locals("user_id").(string)

	resp, err := h.authService.ImpersonateTenant(c.Context(), tenantID, actorID)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if h.auditLogRepo != nil {
		actorEmail := ""
		if actor, aerr := h.authService.GetCurrentUser(c.Context(), actorID, "superadmin", ""); aerr == nil && actor != nil {
			actorEmail = actor.Email
		}
		_ = h.auditLogRepo.Create(c.Context(), &model.AuditLog{
			UserID:     actorID,
			UserEmail:  actorEmail,
			Role:       "superadmin",
			TenantID:   tenantID,
			Action:     "impersonate",
			Resource:   "tenant",
			ResourceID: tenantID,
			Method:     c.Method(),
			Path:       c.Path(),
			IPAddress:  c.IP(),
			UserAgent:  c.Get("User-Agent"),
			StatusCode: fiber.StatusOK,
		})
	}

	return c.JSON(resp)
}

// POST /api/v1/tenants/:id/approve
func (h *TenantHandler) Approve(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.tenantService.Approve(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyetujui tenant"})
	}
	return c.JSON(fiber.Map{"message": "Tenant berhasil disetujui"})
}

func (h *TenantHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	// Hide the superadmin's own tenant from the list
	excludeTenantID, _ := c.Locals("tenant_id").(string)

	filter := repository.TenantFilter{
		Search:          c.Query("search"),
		Plan:            c.Query("plan"),
		ExcludeTenantID: excludeTenantID,
		Page:            page,
		PerPage:         perPage,
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
