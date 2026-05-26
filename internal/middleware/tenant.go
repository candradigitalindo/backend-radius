package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

// PermissionLoaderFunc is a function type that loads permissions for a role.
type PermissionLoaderFunc func(ctx context.Context, tenantID, roleSlug string) []string

// NewPermissionLoader creates a middleware that loads permissions into c.Locals("permissions").
func NewPermissionLoader(roleRepo repository.RoleRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		tenantID, _ := c.Locals("tenant_id").(string)

		if role == "owner" || role == "superadmin" {
			c.Locals("permissions", model.AllPermissionKeys())
			return c.Next()
		}

		if tenantID != "" && role != "" && role != "customer" {
			r, err := roleRepo.FindBySlug(c.Context(), tenantID, role)
			if err == nil && r != nil {
				c.Locals("permissions", r.Permissions)
			} else {
				c.Locals("permissions", []string{})
			}
		}

		return c.Next()
	}
}

func TenantGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID, ok := c.Locals("tenant_id").(string)
		if !ok || tenantID == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Tenant context not found",
			})
		}
		return c.Next()
	}
}

func RoleGuard(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Role not found",
			})
		}

		for _, r := range roles {
			if r == userRole {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// StaffGuard blocks customer and superadmin roles from accessing staff-only routes.
func StaffGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, _ := c.Locals("role").(string)
		if userRole == "customer" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Akses ditolak",
			})
		}
		return c.Next()
	}
}

// PermissionGuard checks if the user's permissions (set by PermissionLoader) include
// at least one of the required permissions. Owner and superadmin always pass.
func PermissionGuard(required ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, _ := c.Locals("role").(string)
		// Owner and superadmin bypass permission checks
		if userRole == "owner" || userRole == "superadmin" {
			return c.Next()
		}

		perms, _ := c.Locals("permissions").([]string)
		for _, rp := range required {
			for _, up := range perms {
				if rp == up {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}
