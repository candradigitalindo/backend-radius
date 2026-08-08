package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type PackageHandler struct {
	packageService *service.PackageService
}

func NewPackageHandler(packageService *service.PackageService) *PackageHandler {
	return &PackageHandler{packageService: packageService}
}

type createPackageRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	BandwidthUp   int    `json:"bandwidth_up"`
	BandwidthDown int    `json:"bandwidth_down"`
	Price         int64  `json:"price"`
	AddressList   string `json:"address_list"`
}

type updatePackageRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	BandwidthUp   int    `json:"bandwidth_up"`
	BandwidthDown int    `json:"bandwidth_down"`
	Price         int64  `json:"price"`
	AddressList   string `json:"address_list"`
	IsActive      bool   `json:"is_active"`
}

func (h *PackageHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createPackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.BandwidthUp <= 0 || req.BandwidthDown <= 0 || req.Price < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, bandwidth_up, bandwidth_down, dan harga wajib diisi"})
	}

	pkg, err := h.packageService.Create(c.Context(), service.CreatePackageInput{
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		BandwidthUp:   req.BandwidthUp,
		BandwidthDown: req.BandwidthDown,
		Price:         req.Price,
		AddressList:   req.AddressList,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat paket"})
	}

	return c.Status(fiber.StatusCreated).JSON(pkg)
}

func (h *PackageHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	packageID := c.Params("id")

	pkg, err := h.packageService.GetByID(c.Context(), tenantID, packageID)
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(pkg)
}

func (h *PackageHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	packageID := c.Params("id")

	var req updatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.BandwidthUp <= 0 || req.BandwidthDown <= 0 || req.Price < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, bandwidth_up, bandwidth_down, dan harga wajib diisi"})
	}

	pkg, err := h.packageService.Update(c.Context(), tenantID, packageID, service.UpdatePackageInput{
		Name:          req.Name,
		Description:   req.Description,
		BandwidthUp:   req.BandwidthUp,
		BandwidthDown: req.BandwidthDown,
		Price:         req.Price,
		AddressList:   req.AddressList,
		IsActive:      req.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(pkg)
}

func (h *PackageHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	packageID := c.Params("id")

	if err := h.packageService.Delete(c.Context(), tenantID, packageID); err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Paket berhasil dihapus"})
}

func (h *PackageHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.PackageFilter{
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	packages, total, err := h.packageService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar paket"})
	}

	return c.JSON(fiber.Map{
		"data":     packages,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
