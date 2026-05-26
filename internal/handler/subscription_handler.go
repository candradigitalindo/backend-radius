package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

// ListPlans returns all active subscription plans.
func (h *SubscriptionHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.subscriptionService.ListPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar paket"})
	}
	return c.JSON(fiber.Map{"data": plans})
}

// GetPlan returns a single plan by ID.
func (h *SubscriptionHandler) GetPlan(c *fiber.Ctx) error {
	planID := c.Params("id")
	plan, err := h.subscriptionService.GetPlan(c.Context(), planID)
	if err != nil {
		if errors.Is(err, service.ErrPlanNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data paket"})
	}
	return c.JSON(fiber.Map{"data": plan})
}

type subscribeRequest struct {
	PlanID         string `json:"plan_id"`
	DurationMonths int    `json:"duration_months"`
}

// Subscribe creates a subscription order for the current tenant.
func (h *SubscriptionHandler) Subscribe(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req subscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.PlanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "plan_id wajib diisi"})
	}

	order, err := h.subscriptionService.Subscribe(c.Context(), service.SubscribeInput{
		TenantID:       tenantID,
		PlanID:         req.PlanID,
		DurationMonths: req.DurationMonths,
	})
	if err != nil {
		if errors.Is(err, service.ErrPlanNotFound) || errors.Is(err, service.ErrPlanInactive) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrPendingOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pesanan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": order})
}

type confirmPaymentRequest struct {
	PaymentMethod string `json:"payment_method"`
	PaymentRef    string `json:"payment_ref"`
}

// ConfirmPayment confirms a pending order payment (manual confirmation by superadmin or webhook).
func (h *SubscriptionHandler) ConfirmPayment(c *fiber.Ctx) error {
	orderID := c.Params("id")

	var req confirmPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	order, err := h.subscriptionService.ConfirmPayment(c.Context(), orderID, req.PaymentMethod, req.PaymentRef)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengonfirmasi pembayaran"})
	}

	return c.JSON(fiber.Map{"data": order})
}

// ListOrders returns subscription order history for the current tenant.
func (h *SubscriptionHandler) ListOrders(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.OrderFilter{
		Status:  c.Query("status"),
		Page:    page,
		PerPage: perPage,
	}

	orders, total, err := h.subscriptionService.ListOrders(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat riwayat order"})
	}

	return c.JSON(fiber.Map{
		"data":  orders,
		"total": total,
		"page":  page,
	})
}

// GetOrder returns a single order by ID.
func (h *SubscriptionHandler) GetOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	tenantID, _ := c.Locals("tenant_id").(string)

	order, err := h.subscriptionService.GetOrder(c.Context(), orderID, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data order"})
	}

	return c.JSON(fiber.Map{"data": order})
}

type createSubPaymentRequest struct {
	ReturnURL string `json:"return_url"`
}

// CreatePayment generates a payment gateway link for a pending subscription order.
func (h *SubscriptionHandler) CreatePayment(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	orderID := c.Params("id")

	var req createSubPaymentRequest
	_ = c.BodyParser(&req)

	result, err := h.subscriptionService.CreatePayment(c.Context(), orderID, tenantID, req.ReturnURL)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": result})
}
