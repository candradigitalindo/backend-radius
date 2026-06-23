package handler

import (
	"crypto/subtle"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

// InternalHandler exposes machine-to-machine endpoints (e.g. for the n8n CS bot),
// guarded by a shared API secret rather than a tenant JWT.
type InternalHandler struct {
	tenantRepo   repository.TenantRepository
	customerRepo repository.CustomerRepository
	apiSecret    string
}

func NewInternalHandler(tenantRepo repository.TenantRepository, customerRepo repository.CustomerRepository, apiSecret string) *InternalHandler {
	return &InternalHandler{tenantRepo: tenantRepo, customerRepo: customerRepo, apiSecret: apiSecret}
}

var nonDigit = regexp.MustCompile(`\D`)

// authorize checks the shared secret from "Authorization: Bearer <secret>" or "X-API-Key".
func (h *InternalHandler) authorize(c *fiber.Ctx) bool {
	if h.apiSecret == "" {
		return false
	}
	got := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if got == "" {
		got = c.Get("X-API-Key")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.apiSecret)) == 1
}

// TenantByPhone resolves a WhatsApp sender phone to its tenant + subscription summary.
// Used by the n8n CS bot so it can answer tenant questions ("kapan langganan habis", dst).
func (h *InternalHandler) TenantByPhone(c *fiber.Ctx) error {
	if !h.authorize(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	phone := nonDigit.ReplaceAllString(c.Query("phone"), "")
	if len(phone) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone tidak valid"})
	}

	tenant, err := h.tenantRepo.FindByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "lookup gagal"})
	}
	if tenant == nil {
		return c.JSON(fiber.Map{"found": false})
	}

	activeCustomers, _ := h.customerRepo.CountActive(c.Context(), tenant.ID)

	var expiresAt, expiresHuman string
	daysLeft := 0
	expired := false
	if tenant.PlanExpiresAt != nil {
		expiresAt = tenant.PlanExpiresAt.Format(time.RFC3339)
		expiresHuman = tenant.PlanExpiresAt.In(time.FixedZone("WIB", 7*3600)).Format("02 January 2006")
		daysLeft = int(time.Until(*tenant.PlanExpiresAt).Hours() / 24)
		expired = daysLeft < 0
	}

	return c.JSON(fiber.Map{
		"found":            true,
		"tenant_id":        tenant.ID,
		"name":             tenant.Name,
		"slug":             tenant.Slug,
		"email":            tenant.Email,
		"phone":            tenant.Phone,
		"plan":             tenant.Plan,
		"plan_expires_at":  expiresAt,
		"plan_expires_human": expiresHuman,
		"days_left":        daysLeft,
		"expired":          expired,
		"status":           tenant.Status,
		"is_active":        tenant.IsActive,
		"max_customers":    tenant.MaxCustomers,
		"active_customers": activeCustomers,
	})
}
