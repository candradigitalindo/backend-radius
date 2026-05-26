package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

// PublicVoucherHandler handles unauthenticated public voucher store endpoints.
type PublicVoucherHandler struct {
	voucherService *service.VoucherService
	tenantService  *service.TenantService
}

func NewPublicVoucherHandler(voucherService *service.VoucherService, tenantService *service.TenantService) *PublicVoucherHandler {
	return &PublicVoucherHandler{
		voucherService: voucherService,
		tenantService:  tenantService,
	}
}

// GetStoreProducts handles GET /public/store/:tenant_slug
// Returns active voucher products available for purchase.
func (h *PublicVoucherHandler) GetStoreProducts(c *fiber.Ctx) error {
	slug := c.Params("tenant_slug")

	tenant, err := h.tenantService.GetBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Toko tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	products, err := h.voucherService.GetPublicStoreProducts(c.Context(), tenant.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"tenant_name": tenant.Name,
			"products":    products,
		},
	})
}

// PurchaseVoucher handles POST /public/store/:tenant_slug/buy
// Purchases a voucher via payment gateway.
func (h *PublicVoucherHandler) PurchaseVoucher(c *fiber.Ctx) error {
	slug := c.Params("tenant_slug")

	tenant, err := h.tenantService.GetBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Toko tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	var req struct {
		ProductID     string `json:"product_id"`
		BuyerName     string `json:"buyer_name"`
		BuyerPhone    string `json:"buyer_phone"`
		PaymentMethod string `json:"payment_method"`
		ReturnURL     string `json:"return_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}
	if req.ProductID == "" || req.BuyerPhone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "product_id dan nomor telepon pembeli wajib diisi"})
	}

	result, err := h.voucherService.PurchaseVoucher(c.Context(), service.PurchaseVoucherInput{
		TenantID:      tenant.ID,
		ProductID:     req.ProductID,
		BuyerName:     req.BuyerName,
		BuyerPhone:    req.BuyerPhone,
		PaymentMethod: req.PaymentMethod,
		ReturnURL:     req.ReturnURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrVoucherNotAvailable) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "Tidak ada voucher tersedia untuk produk ini"})
		}
		if errors.Is(err, service.ErrVoucherProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Produk tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": result})
}
