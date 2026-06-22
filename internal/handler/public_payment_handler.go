package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

// PublicPaymentHandler handles unauthenticated public payment page endpoints.
type PublicPaymentHandler struct {
	invoiceService *service.InvoiceService
}

func NewPublicPaymentHandler(invoiceService *service.InvoiceService) *PublicPaymentHandler {
	return &PublicPaymentHandler{invoiceService: invoiceService}
}

// GetCustomerInfo handles GET /public/pay/:tenant_id/:customer_code
// Returns limited customer info and unpaid invoices for the public payment page.
func (h *PublicPaymentHandler) GetCustomerInfo(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")
	customerCode := c.Params("customer_code")

	info, invoices, err := h.invoiceService.GetPublicCustomerInfo(c.Context(), tenantID, customerCode)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Pelanggan tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"customer": info, "invoices": invoices}})
}

// CreatePayment handles POST /public/pay/:tenant_id/:customer_code
// Creates a gateway payment for a specific invoice (no auth required).
func (h *PublicPaymentHandler) CreatePayment(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")
	customerCode := c.Params("customer_code")

	var req struct {
		InvoiceID     string `json:"invoice_id"`
		PaymentMethod string `json:"payment_method"`
		CustomerName  string `json:"customer_name"`
		CustomerEmail string `json:"customer_email"`
		CustomerPhone string `json:"customer_phone"`
		ReturnURL     string `json:"return_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Format request tidak valid"})
	}
	if req.InvoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invoice_id wajib diisi"})
	}

	// Verify that the invoice belongs to this customer
	if err := h.invoiceService.VerifyInvoiceOwnership(c.Context(), tenantID, customerCode, req.InvoiceID); err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Pelanggan tidak ditemukan"})
		}
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Faktur tidak ditemukan"})
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "Faktur bukan milik pelanggan ini"})
	}

	result, err := h.invoiceService.CreateGatewayPayment(c.Context(), service.PaymentGatewayInput{
		TenantID:      tenantID,
		InvoiceID:     req.InvoiceID,
		PaymentMethod: req.PaymentMethod,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		ReturnURL:     req.ReturnURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Faktur tidak ditemukan"})
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "Faktur sudah dibayar"})
		}
		if errors.Is(err, service.ErrPaymentGatewayInactive) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Pembayaran online belum aktif. Hubungi admin Anda."})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": result})
}

// CheckPaymentStatus handles GET /public/pay/check/:trx_id
// Returns the current status of a gateway payment by transaction ID.
func (h *PublicPaymentHandler) CheckPaymentStatus(c *fiber.Ctx) error {
	trxID := c.Params("trx_id")

	p, err := h.invoiceService.CheckPaymentStatus(c.Context(), trxID)
	if err != nil {
		if errors.Is(err, service.ErrPaymentNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Pembayaran tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"payment_id":     p.ID,
		"gateway_trx_id": p.GatewayTrxID,
		"status":         p.Status,
		"amount":         p.Amount,
		"gateway":        p.Gateway,
		"payment_method": p.PaymentMethod,
		"paid_at":        p.PaidAt,
		"expired_at":     p.ExpiredAt,
	}})
}
