package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type VoucherHandler struct {
	voucherService *service.VoucherService
}

func NewVoucherHandler(voucherService *service.VoucherService) *VoucherHandler {
	return &VoucherHandler{voucherService: voucherService}
}

// -- Product requests --

type createVoucherProductRequest struct {
	Name          string  `json:"name"`
	Duration      int     `json:"duration"`
	BandwidthUp   int     `json:"bandwidth_up"`
	BandwidthDown int     `json:"bandwidth_down"`
	Price         int64   `json:"price"`
	ProfileName   string  `json:"profile_name"`
	RouterID      *string `json:"router_id"`
}

type updateVoucherProductRequest struct {
	Name          string  `json:"name"`
	Duration      int     `json:"duration"`
	BandwidthUp   int     `json:"bandwidth_up"`
	BandwidthDown int     `json:"bandwidth_down"`
	Price         int64   `json:"price"`
	ProfileName   string  `json:"profile_name"`
	RouterID      *string `json:"router_id"`
	IsActive      bool    `json:"is_active"`
}

type generateVouchersRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Prefix    string `json:"prefix"`
}

type sellVoucherRequest struct {
	BuyerPhone string `json:"buyer_phone"`
}

// -- Product handlers --

func (h *VoucherHandler) CreateProduct(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createVoucherProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Duration <= 0 || req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, durasi, dan harga wajib diisi"})
	}

	product, err := h.voucherService.CreateProduct(c.Context(), service.CreateVoucherProductInput{
		TenantID:      tenantID,
		Name:          req.Name,
		Duration:      req.Duration,
		BandwidthUp:   req.BandwidthUp,
		BandwidthDown: req.BandwidthDown,
		Price:         req.Price,
		ProfileName:   req.ProfileName,
		RouterID:      req.RouterID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat produk voucher"})
	}

	return c.Status(fiber.StatusCreated).JSON(product)
}

func (h *VoucherHandler) GetProduct(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	productID := c.Params("id")

	product, err := h.voucherService.GetProduct(c.Context(), tenantID, productID)
	if err != nil {
		if errors.Is(err, service.ErrVoucherProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(product)
}

func (h *VoucherHandler) UpdateProduct(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	productID := c.Params("id")

	var req updateVoucherProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Duration <= 0 || req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, durasi, dan harga wajib diisi"})
	}

	product, err := h.voucherService.UpdateProduct(c.Context(), tenantID, productID, service.UpdateVoucherProductInput{
		Name:          req.Name,
		Duration:      req.Duration,
		BandwidthUp:   req.BandwidthUp,
		BandwidthDown: req.BandwidthDown,
		Price:         req.Price,
		ProfileName:   req.ProfileName,
		RouterID:      req.RouterID,
		IsActive:      req.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrVoucherProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(product)
}

func (h *VoucherHandler) DeleteProduct(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	productID := c.Params("id")

	if err := h.voucherService.DeleteProduct(c.Context(), tenantID, productID); err != nil {
		if errors.Is(err, service.ErrVoucherProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Produk voucher berhasil dihapus"})
}

func (h *VoucherHandler) ListProducts(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.VoucherProductFilter{
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	products, total, err := h.voucherService.ListProducts(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar produk voucher"})
	}

	return c.JSON(fiber.Map{
		"data":     products,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// -- Voucher handlers --

func (h *VoucherHandler) Generate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req generateVouchersRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.ProductID == "" || req.Quantity <= 0 || req.Quantity > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "product_id dan quantity (1-1000) wajib diisi"})
	}

	count, err := h.voucherService.Generate(c.Context(), service.GenerateVouchersInput{
		TenantID:  tenantID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Prefix:    req.Prefix,
	})
	if err != nil {
		if errors.Is(err, service.ErrVoucherProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat voucher"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Voucher berhasil dibuat",
		"count":   count,
	})
}

func (h *VoucherHandler) GetVoucher(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	voucherID := c.Params("id")

	voucher, err := h.voucherService.GetVoucher(c.Context(), tenantID, voucherID)
	if err != nil {
		if errors.Is(err, service.ErrVoucherNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(voucher)
}

func (h *VoucherHandler) DeleteVoucher(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	voucherID := c.Params("id")

	if err := h.voucherService.DeleteVoucher(c.Context(), tenantID, voucherID); err != nil {
		if errors.Is(err, service.ErrVoucherNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Voucher berhasil dihapus"})
}

func (h *VoucherHandler) ListVouchers(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.VoucherFilter{
		Search:    c.Query("search"),
		ProductID: c.Query("product_id"),
		Status:    c.Query("status"),
		Page:      page,
		PerPage:   perPage,
	}

	vouchers, total, err := h.voucherService.ListVouchers(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar voucher"})
	}

	return c.JSON(fiber.Map{
		"data":     vouchers,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

