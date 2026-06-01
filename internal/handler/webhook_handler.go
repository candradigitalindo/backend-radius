package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/service"
)

// WebhookHandler handles payment gateway webhook callbacks.
type WebhookHandler struct {
	invoiceService      *service.InvoiceService
	voucherService      *service.VoucherService
	subscriptionService *service.SubscriptionService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(invoiceService *service.InvoiceService) *WebhookHandler {
	return &WebhookHandler{invoiceService: invoiceService}
}

func (h *WebhookHandler) WithVoucherService(voucherService *service.VoucherService) *WebhookHandler {
	h.voucherService = voucherService
	return h
}

func (h *WebhookHandler) WithSubscriptionService(subscriptionService *service.SubscriptionService) *WebhookHandler {
	h.subscriptionService = subscriptionService
	return h
}

// TripayCallback handles Tripay payment gateway webhook notifications.
// This endpoint is public (no JWT auth) but validated via HMAC signature inside the service.
func (h *WebhookHandler) TripayCallback(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var payload payment.TripayCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] tripay slug=%s reference=%s status=%s", slug, payload.Reference, payload.Status)

	if err := h.invoiceService.ProcessTripayWebhook(c.Context(), slug, payload); err != nil {
		log.Printf("[webhook] tripay process error: %v", err)
		// Always return 200 so Tripay does not keep retrying
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// MidtransCallback handles Midtrans HTTP notification (webhook) callbacks.
// This endpoint is public (no JWT auth). Signature is verified inside the service using
// the tenant's stored server key.
func (h *WebhookHandler) MidtransCallback(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var n payment.MidtransNotification
	if err := c.BodyParser(&n); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] midtrans slug=%s order_id=%s status=%s", slug, n.OrderID, n.TransactionStatus)

	if err := h.invoiceService.ProcessMidtransWebhook(c.Context(), slug, n); err != nil {
		log.Printf("[webhook] midtrans process error: %v", err)
		// Always return 200 so Midtrans does not keep retrying
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// TripayVoucherCallback handles Tripay webhook notifications for voucher purchases.
func (h *WebhookHandler) TripayVoucherCallback(c *fiber.Ctx) error {
	if h.voucherService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan voucher tidak dikonfigurasi"})
	}

	slug := c.Params("slug")
	var payload payment.TripayCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] tripay voucher slug=%s reference=%s status=%s", slug, payload.Reference, payload.Status)

	if err := h.voucherService.ProcessVoucherTripayWebhook(c.Context(), slug, payload); err != nil {
		log.Printf("[webhook] tripay voucher error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// MidtransVoucherCallback handles Midtrans webhook notifications for voucher purchases.
func (h *WebhookHandler) MidtransVoucherCallback(c *fiber.Ctx) error {
	if h.voucherService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan voucher tidak dikonfigurasi"})
	}

	slug := c.Params("slug")
	var n payment.MidtransNotification
	if err := c.BodyParser(&n); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] midtrans voucher slug=%s order_id=%s status=%s", slug, n.OrderID, n.TransactionStatus)

	if err := h.voucherService.ProcessVoucherMidtransWebhook(c.Context(), slug, n); err != nil {
		log.Printf("[webhook] midtrans voucher error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// TripaySubscriptionCallback handles Tripay webhook for subscription payments.
func (h *WebhookHandler) TripaySubscriptionCallback(c *fiber.Ctx) error {
	if h.subscriptionService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan subscription tidak dikonfigurasi"})
	}

	var payload payment.TripayCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] tripay subscription reference=%s status=%s", payload.Reference, payload.Status)

	if err := h.subscriptionService.ProcessSubTripayWebhook(c.Context(), payload); err != nil {
		log.Printf("[webhook] tripay subscription error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// MidtransSubscriptionCallback handles Midtrans webhook for subscription payments.
func (h *WebhookHandler) MidtransSubscriptionCallback(c *fiber.Ctx) error {
	if h.subscriptionService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan subscription tidak dikonfigurasi"})
	}

	var n payment.MidtransNotification
	if err := c.BodyParser(&n); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] midtrans subscription order_id=%s status=%s", n.OrderID, n.TransactionStatus)

	if err := h.subscriptionService.ProcessSubMidtransWebhook(c.Context(), n); err != nil {
		log.Printf("[webhook] midtrans subscription error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// XenditCallback handles Xendit Invoice webhook notifications.
// Xendit verifies via "x-callback-token" header instead of body signature.
func (h *WebhookHandler) XenditCallback(c *fiber.Ctx) error {
	slug := c.Params("slug")
	callbackToken := c.Get("x-callback-token")

	var payload payment.XenditCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] xendit slug=%s external_id=%s status=%s", slug, payload.ExternalID, payload.Status)

	if err := h.invoiceService.ProcessXenditWebhook(c.Context(), slug, callbackToken, payload); err != nil {
		log.Printf("[webhook] xendit process error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// XenditVoucherCallback handles Xendit webhook notifications for voucher purchases.
func (h *WebhookHandler) XenditVoucherCallback(c *fiber.Ctx) error {
	if h.voucherService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan voucher tidak dikonfigurasi"})
	}

	slug := c.Params("slug")
	callbackToken := c.Get("x-callback-token")

	var payload payment.XenditCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] xendit voucher slug=%s external_id=%s status=%s", slug, payload.ExternalID, payload.Status)

	if err := h.voucherService.ProcessVoucherXenditWebhook(c.Context(), slug, callbackToken, payload); err != nil {
		log.Printf("[webhook] xendit voucher error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// XenditSubscriptionCallback handles Xendit webhook for subscription payments.
func (h *WebhookHandler) XenditSubscriptionCallback(c *fiber.Ctx) error {
	if h.subscriptionService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": "Layanan subscription tidak dikonfigurasi"})
	}

	callbackToken := c.Get("x-callback-token")

	var payload payment.XenditCallbackPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	log.Printf("[webhook] xendit subscription external_id=%s status=%s", payload.ExternalID, payload.Status)

	if err := h.subscriptionService.ProcessSubXenditWebhook(c.Context(), callbackToken, payload); err != nil {
		log.Printf("[webhook] xendit subscription error: %v", err)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}
