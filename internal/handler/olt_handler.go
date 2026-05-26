package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type OLTHandler struct {
	oltService *service.OLTService
}

func NewOLTHandler(oltService *service.OLTService) *OLTHandler {
	return &OLTHandler{oltService: oltService}
}

type createOLTRequest struct {
	RouterID      *string  `json:"router_id"`
	Name          string   `json:"name"`
	IPAddress     string   `json:"ip_address"`
	Vendor        *string  `json:"vendor"`
	Model         *string  `json:"model"`
	SerialNumber  *string  `json:"serial_number"`
	TotalPONPorts int      `json:"total_pon_ports"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	SNMPCommunity string   `json:"snmp_community"`
	Notes         *string  `json:"notes"`
}

type updateOLTRequest struct {
	RouterID      *string  `json:"router_id"`
	Name          string   `json:"name"`
	IPAddress     string   `json:"ip_address"`
	Vendor        *string  `json:"vendor"`
	Model         *string  `json:"model"`
	SerialNumber  *string  `json:"serial_number"`
	TotalPONPorts int      `json:"total_pon_ports"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	SNMPCommunity string   `json:"snmp_community"`
	Status        string   `json:"status"`
	Notes         *string  `json:"notes"`
}

func (h *OLTHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createOLTRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.IPAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan ip_address wajib diisi"})
	}

	olt, err := h.oltService.Create(c.Context(), service.CreateOLTInput{
		TenantID:      tenantID,
		RouterID:      req.RouterID,
		Name:          req.Name,
		IPAddress:     req.IPAddress,
		Vendor:        req.Vendor,
		Model:         req.Model,
		SerialNumber:  req.SerialNumber,
		TotalPONPorts: req.TotalPONPorts,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		SNMPCommunity: req.SNMPCommunity,
		Notes:         req.Notes,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat OLT"})
	}

	return c.Status(fiber.StatusCreated).JSON(olt)
}

func (h *OLTHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	olt, err := h.oltService.GetByID(c.Context(), tenantID, oltID)
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(olt)
}

func (h *OLTHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	var req updateOLTRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.IPAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan ip_address wajib diisi"})
	}

	olt, err := h.oltService.Update(c.Context(), tenantID, oltID, service.UpdateOLTInput{
		RouterID:      req.RouterID,
		Name:          req.Name,
		IPAddress:     req.IPAddress,
		Vendor:        req.Vendor,
		Model:         req.Model,
		SerialNumber:  req.SerialNumber,
		TotalPONPorts: req.TotalPONPorts,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		SNMPCommunity: req.SNMPCommunity,
		Status:        req.Status,
		Notes:         req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(olt)
}

func (h *OLTHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	if err := h.oltService.Delete(c.Context(), tenantID, oltID); err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(fiber.Map{"message": "OLT berhasil dihapus"})
}

func (h *OLTHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.OLTFilter{
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		Page:    page,
		PerPage: perPage,
	}

	olts, total, err := h.oltService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar OLT"})
	}

	return c.JSON(fiber.Map{
		"data":     olts,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// PON Port handlers

type createPONPortRequest struct {
	PortNumber  int      `json:"port_number"`
	Description *string  `json:"description"`
	SFPRxPower  *float64 `json:"sfp_rx_power"`
	SFPTxPower  *float64 `json:"sfp_tx_power"`
}

type updatePONPortRequest struct {
	Description *string  `json:"description"`
	Status      string   `json:"status"`
	SFPRxPower  *float64 `json:"sfp_rx_power"`
	SFPTxPower  *float64 `json:"sfp_tx_power"`
}

func (h *OLTHandler) CreatePONPort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	var req createPONPortRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PortNumber <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "port_number wajib diisi dan harus positif"})
	}

	port, err := h.oltService.CreatePONPort(c.Context(), tenantID, service.CreatePONPortInput{
		OLTID:       oltID,
		PortNumber:  req.PortNumber,
		Description: req.Description,
		SFPRxPower:  req.SFPRxPower,
		SFPTxPower:  req.SFPTxPower,
	})
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat port PON"})
	}

	return c.Status(fiber.StatusCreated).JSON(port)
}

func (h *OLTHandler) ListPONPorts(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")

	ports, err := h.oltService.ListPONPorts(c.Context(), tenantID, oltID)
	if err != nil {
		if errors.Is(err, service.ErrOLTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar port PON"})
	}

	return c.JSON(fiber.Map{"data": ports})
}

func (h *OLTHandler) UpdatePONPort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")
	portID := c.Params("portId")

	var req updatePONPortRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	port, err := h.oltService.UpdatePONPort(c.Context(), tenantID, oltID, portID, service.UpdatePONPortInput{
		Description: req.Description,
		Status:      req.Status,
		SFPRxPower:  req.SFPRxPower,
		SFPTxPower:  req.SFPTxPower,
	})
	if err != nil {
		if errors.Is(err, service.ErrPONPortNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(port)
}

func (h *OLTHandler) DeletePONPort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	oltID := c.Params("id")
	portID := c.Params("portId")

	if err := h.oltService.DeletePONPort(c.Context(), tenantID, oltID, portID); err != nil {
		if errors.Is(err, service.ErrPONPortNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(fiber.Map{"message": "Port PON berhasil dihapus"})
}
