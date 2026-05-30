package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

func SubscriptionGuard(tenantRepo repository.TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if role == "superadmin" || role == "customer" || role == "" {
			return c.Next()
		}

		path := c.Path()
		// Allow subscription and basic tenant info routes
		if strings.HasPrefix(path, "/api/v1/subscription") ||
			path == "/api/v1/tenant" ||
			strings.HasPrefix(path, "/api/v1/auth") {
			return c.Next()
		}

		tenantID, _ := c.Locals("tenant_id").(string)
		if tenantID == "" {
			return c.Next()
		}

		tenant, err := tenantRepo.FindByID(c.Context(), tenantID)
		if err != nil || tenant == nil {
			return c.Next()
		}

		if isExpired(tenant) {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"code":    "subscription_expired",
				"message": "Langganan Anda telah berakhir. Silakan perpanjang untuk melanjutkan.",
				"tenant": fiber.Map{
					"id":              tenant.ID,
					"name":            tenant.Name,
					"plan":            tenant.Plan,
					"plan_expires_at": tenant.PlanExpiresAt,
				},
			})
		}

		return c.Next()
	}
}

func isExpired(t *model.Tenant) bool {
	if t.PlanExpiresAt == nil {
		return false
	}
	plan := strings.ToLower(t.Plan)
	if plan == "gratis" || plan == "free" || plan == "trial" || plan == "" {
		return false
	}
	return t.PlanExpiresAt.Before(time.Now())
}
