package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

// AuditLogger returns a middleware that records audit log entries for every request.
// It resolves the action and resource from the HTTP method and route path.
func AuditLogger(repo repository.AuditLogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Execute the handler first
		err := c.Next()

		// Only log mutating requests + GET on sensitive resources
		method := c.Method()
		if method == "OPTIONS" || method == "HEAD" {
			return err
		}

		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			return err
		}

		role, _ := c.Locals("role").(string)
		tenantID, _ := c.Locals("tenant_id").(string)
		userEmail, _ := c.Locals("user_email").(string)

		path := c.Path()
		action, resource, resourceID := resolveAction(method, path)

		entry := &model.AuditLog{
			UserID:     userID,
			UserEmail:  userEmail,
			Role:       role,
			TenantID:   tenantID,
			Action:     action,
			Resource:   resource,
			ResourceID: resourceID,
			Method:     method,
			Path:       path,
			IPAddress:  c.IP(),
			UserAgent:  c.Get("User-Agent"),
			StatusCode: c.Response().StatusCode(),
		}

		// Log asynchronously to not slow down the response
		go func() {
			_ = repo.Create(context.Background(), entry)
		}()

		return err
	}
}

// AuditUserLoader is a lightweight middleware that loads the user email
// into c.Locals for the audit logger, querying the database once per request.
func AuditUserLoader(userRepo repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		if userID != "" {
			if u, err := userRepo.FindByID(c.Context(), userID); err == nil && u != nil {
				c.Locals("user_email", u.Email)
			}
		}
		return c.Next()
	}
}

// resolveAction maps HTTP method + path into human-readable action, resource, and resourceID.
func resolveAction(method, path string) (action, resource, resourceID string) {
	// Strip query string and prefix
	path = strings.SplitN(path, "?", 2)[0]
	path = strings.TrimPrefix(path, "/api/v1/")

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return method, "unknown", ""
	}

	resource = parts[0]

	// Detect resourceID (ULID-like or UUID-like)
	if len(parts) >= 2 && len(parts[1]) >= 20 {
		resourceID = parts[1]
	}

	// Sub-action (e.g., /orders/:id/confirm)
	subAction := ""
	for i := 2; i < len(parts); i++ {
		if len(parts[i]) < 20 {
			subAction = parts[i]
		}
	}

	switch method {
	case "GET":
		if resourceID != "" {
			action = "view"
		} else {
			action = "list"
		}
	case "POST":
		action = "create"
		if subAction != "" {
			action = subAction
		}
	case "PUT", "PATCH":
		action = "update"
		if subAction != "" {
			action = subAction
		}
	case "DELETE":
		action = "delete"
	default:
		action = method
	}

	if subAction != "" && method != "POST" && method != "PUT" {
		action = fmt.Sprintf("%s_%s", action, subAction)
	}

	return action, resource, resourceID
}
