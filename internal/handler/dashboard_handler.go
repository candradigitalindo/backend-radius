package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	stats, err := h.dashboardService.GetStats(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": stats})
}

func (h *DashboardHandler) GetMonthlyRevenue(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.dashboardService.GetMonthlyRevenue(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": data, "year": year})
}

func (h *DashboardHandler) GetRollingRevenue(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	data, err := h.dashboardService.GetRollingMonthlyRevenue(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": data})
}
