package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type ODPHandler struct {
	odpService *service.ODPService
}

func NewODPHandler(odpService *service.ODPService) *ODPHandler {
	return &ODPHandler{odpService: odpService}
}

// ODP requests

type createODPRequest struct {
	OLTID         *string  `json:"olt_id"`
	PONPortID     *string  `json:"pon_port_id"`
	SplitterRatio *string  `json:"splitter_ratio"`
	Name          string   `json:"name"`
	Address       *string  `json:"address"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	TotalPorts    int      `json:"total_ports"`
	Sequence      int      `json:"sequence"`
	CableLengthM  float64  `json:"cable_length_m"`
	RatioPercent  float64  `json:"ratio_percent"`
	SplitterType  *string  `json:"splitter_type"`
	Status        string   `json:"status"`
	Notes         *string  `json:"notes"`
}

type updateODPRequest struct {
	OLTID         *string  `json:"olt_id"`
	PONPortID     *string  `json:"pon_port_id"`
	SplitterRatio *string  `json:"splitter_ratio"`
	Name          string   `json:"name"`
	Address       *string  `json:"address"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	TotalPorts    int      `json:"total_ports"`
	Sequence      int      `json:"sequence"`
	CableLengthM  float64  `json:"cable_length_m"`
	RatioPercent  float64  `json:"ratio_percent"`
	SplitterType  *string  `json:"splitter_type"`
	Status        string   `json:"status"`
	Notes         *string  `json:"notes"`
}

func (h *ODPHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createODPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	odp, err := h.odpService.Create(c.Context(), service.CreateODPInput{
		TenantID:      tenantID,
		OLTID:         req.OLTID,
		PONPortID:     req.PONPortID,
		SplitterRatio: req.SplitterRatio,
		Name:          req.Name,
		Address:       req.Address,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		TotalPorts:    req.TotalPorts,
		Sequence:      req.Sequence,
		CableLengthM:  req.CableLengthM,
		RatioPercent:  req.RatioPercent,
		SplitterType:  req.SplitterType,
		Status:        req.Status,
		Notes:         req.Notes,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat ODP"})
	}

	return c.Status(fiber.StatusCreated).JSON(odp)
}

func (h *ODPHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	odpID := c.Params("id")

	odp, err := h.odpService.GetByID(c.Context(), tenantID, odpID)
	if err != nil {
		if errors.Is(err, service.ErrODPNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(odp)
}

func (h *ODPHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	odpID := c.Params("id")

	var req updateODPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	odp, err := h.odpService.Update(c.Context(), tenantID, odpID, service.UpdateODPInput{
		OLTID:         req.OLTID,
		PONPortID:     req.PONPortID,
		SplitterRatio: req.SplitterRatio,
		Name:          req.Name,
		Address:       req.Address,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		TotalPorts:    req.TotalPorts,
		Sequence:      req.Sequence,
		CableLengthM:  req.CableLengthM,
		RatioPercent:  req.RatioPercent,
		SplitterType:  req.SplitterType,
		Status:        req.Status,
		Notes:         req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrODPNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(odp)
}

func (h *ODPHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	odpID := c.Params("id")

	if err := h.odpService.Delete(c.Context(), tenantID, odpID); err != nil {
		if errors.Is(err, service.ErrODPNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(fiber.Map{"message": "ODP berhasil dihapus"})
}

func (h *ODPHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.ODPFilter{
		Search:  c.Query("search"),
		OLTID:   c.Query("olt_id"),
		Page:    page,
		PerPage: perPage,
	}

	odps, total, err := h.odpService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar ODP"})
	}

	return c.JSON(fiber.Map{
		"data":     odps,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// ODP Port handlers

type createODPPortRequest struct {
	PortNumber int     `json:"port_number"`
	CustomerID *string `json:"customer_id"`
	Notes      *string `json:"notes"`
}

type updateODPPortRequest struct {
	CustomerID *string `json:"customer_id"`
	Status     string  `json:"status"`
	Notes      *string `json:"notes"`
}

func (h *ODPHandler) CreatePort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	odpID := c.Params("id")

	var req createODPPortRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PortNumber <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "port_number wajib diisi dan harus positif"})
	}

	port, err := h.odpService.CreatePort(c.Context(), tenantID, service.CreateODPPortInput{
		ODPID:      odpID,
		PortNumber: req.PortNumber,
		CustomerID: req.CustomerID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrODPNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat port ODP"})
	}

	return c.Status(fiber.StatusCreated).JSON(port)
}

func (h *ODPHandler) ListPorts(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	odpID := c.Params("id")

	ports, err := h.odpService.ListPorts(c.Context(), tenantID, odpID)
	if err != nil {
		if errors.Is(err, service.ErrODPNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar port ODP"})
	}

	return c.JSON(fiber.Map{"data": ports})
}

func (h *ODPHandler) UpdatePort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	portID := c.Params("portId")

	var req updateODPPortRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	port, err := h.odpService.UpdatePort(c.Context(), tenantID, portID, service.UpdateODPPortInput{
		CustomerID: req.CustomerID,
		Status:     req.Status,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrODPPortNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(port)
}

func (h *ODPHandler) DeletePort(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	portID := c.Params("portId")

	if err := h.odpService.DeletePort(c.Context(), tenantID, portID); err != nil {
		if errors.Is(err, service.ErrODPPortNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(fiber.Map{"message": "Port ODP berhasil dihapus"})
}

// Splitter handlers

type createSplitterRequest struct {
	PONPortID        *string  `json:"pon_port_id"`
	ParentSplitterID *string  `json:"parent_splitter_id"`
	Name             string   `json:"name"`
	SplitterType     string   `json:"splitter_type"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	Notes            *string  `json:"notes"`
}

type updateSplitterRequest struct {
	PONPortID        *string  `json:"pon_port_id"`
	ParentSplitterID *string  `json:"parent_splitter_id"`
	Name             string   `json:"name"`
	SplitterType     string   `json:"splitter_type"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	Notes            *string  `json:"notes"`
}

func (h *ODPHandler) CreateSplitter(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createSplitterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.SplitterType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan splitter_type wajib diisi"})
	}

	splitter, err := h.odpService.CreateSplitter(c.Context(), service.CreateSplitterInput{
		TenantID:         tenantID,
		PONPortID:        req.PONPortID,
		ParentSplitterID: req.ParentSplitterID,
		Name:             req.Name,
		SplitterType:     req.SplitterType,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Notes:            req.Notes,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat splitter"})
	}

	return c.Status(fiber.StatusCreated).JSON(splitter)
}

func (h *ODPHandler) GetSplitter(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	splitterID := c.Params("id")

	splitter, err := h.odpService.GetSplitterByID(c.Context(), tenantID, splitterID)
	if err != nil {
		if errors.Is(err, service.ErrSplitterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(splitter)
}

func (h *ODPHandler) UpdateSplitter(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	splitterID := c.Params("id")

	var req updateSplitterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.SplitterType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan splitter_type wajib diisi"})
	}

	splitter, err := h.odpService.UpdateSplitter(c.Context(), tenantID, splitterID, service.UpdateSplitterInput{
		PONPortID:        req.PONPortID,
		ParentSplitterID: req.ParentSplitterID,
		Name:             req.Name,
		SplitterType:     req.SplitterType,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Notes:            req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrSplitterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(splitter)
}

func (h *ODPHandler) DeleteSplitter(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	splitterID := c.Params("id")

	if err := h.odpService.DeleteSplitter(c.Context(), tenantID, splitterID); err != nil {
		if errors.Is(err, service.ErrSplitterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	return c.JSON(fiber.Map{"message": "Splitter berhasil dihapus"})
}

func (h *ODPHandler) ListSplitters(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.SplitterFilter{
		Search:    c.Query("search"),
		PONPortID: c.Query("pon_port_id"),
		ParentID:  c.Query("parent_id"),
		Page:      page,
		PerPage:   perPage,
	}

	splitters, total, err := h.odpService.ListSplitters(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar splitter"})
	}

	return c.JSON(fiber.Map{
		"data":     splitters,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
