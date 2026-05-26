package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type FTTHHandler struct {
	ftthService *service.FTTHService
}

func NewFTTHHandler(ftthService *service.FTTHService) *FTTHHandler {
	return &FTTHHandler{ftthService: ftthService}
}

func (h *FTTHHandler) GetStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	stats, err := h.ftthService.GetStats(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil statistik FTTH"})
	}

	return c.JSON(stats)
}

func (h *FTTHHandler) GetMap(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	items, err := h.ftthService.GetMapItems(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data peta FTTH"})
	}

	return c.JSON(fiber.Map{"data": items})
}
