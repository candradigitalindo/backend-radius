package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// GetDashboardStats returns superadmin dashboard overview
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	stats, err := h.adminService.GetDashboardStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"data": stats})
}

// GetTenantStats returns per-tenant customer/router counts
func (h *AdminHandler) GetTenantStats(c *fiber.Ctx) error {
	stats, err := h.adminService.GetTenantStats(c.Context())
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

	routers, total, err := h.adminService.GetAllRouters(c.Context(), page, perPage)
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
	stats, err := h.adminService.GetTenantCustomerCounts(c.Context())
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
