package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type createGatewayPaymentRequest struct {
	PaymentMethod string `json:"payment_method"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	CustomerPhone string `json:"customer_phone"`
	ReturnURL     string `json:"return_url"`
}

// CreateGatewayPayment creates a payment link for an invoice via the tenant's payment gateway.
func (h *InvoiceHandler) CreateGatewayPayment(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	invoiceID := c.Params("id")

	var req createGatewayPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PaymentMethod == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Metode pembayaran wajib diisi"})
	}

	result, err := h.invoiceService.CreateGatewayPayment(c.Context(), service.PaymentGatewayInput{
		TenantID:      tenantID,
		InvoiceID:     invoiceID,
		PaymentMethod: req.PaymentMethod,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		ReturnURL:     req.ReturnURL,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Faktur tidak ditemukan"})
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Faktur sudah dibayar"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": result})
}
