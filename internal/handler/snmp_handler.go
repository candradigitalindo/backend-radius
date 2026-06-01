package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type SNMPHandler struct {
	snmpService *service.SNMPService
	oltService  *service.OLTService
}

func (h *SNMPHandler) WithOLTService(s *service.OLTService) *SNMPHandler {
	h.oltService = s
	return h
}

func NewSNMPHandler(snmpService *service.SNMPService) *SNMPHandler {
	return &SNMPHandler{snmpService: snmpService}
}

// SyncOLT polls SNMP and updates the OLT record with live system info (name, vendor, uptime).
func (h *SNMPHandler) SyncOLT(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	result, err := h.snmpService.MonitorOLT(c.Context(), tenantID, oltID)
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data SNMP: " + err.Error()})
	}

	if result.Status == "offline" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "OLT tidak dapat dijangkau via SNMP"})
	}

	if h.oltService != nil {
		_ = h.oltService.SyncFromSNMP(c.Context(), tenantID, oltID, result)
	}

	return c.JSON(fiber.Map{"data": result, "synced": true})
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
