package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type SNMPHandler struct {
	snmpService *service.SNMPService
}

func NewSNMPHandler(snmpService *service.SNMPService) *SNMPHandler {
	return &SNMPHandler{snmpService: snmpService}
}

// MonitorOLT polls comprehensive OLT data: status, resources, interfaces, ONUs.
func (h *SNMPHandler) MonitorOLT(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	result, err := h.snmpService.MonitorOLT(c.Context(), tenantID, oltID)
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": result})
}
