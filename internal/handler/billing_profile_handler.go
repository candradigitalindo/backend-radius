package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type BillingProfileHandler struct {
	svc *service.BillingProfileService
}

func NewBillingProfileHandler(svc *service.BillingProfileService) *BillingProfileHandler {
	return &BillingProfileHandler{svc: svc}
}

func (h *BillingProfileHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	profiles, err := h.svc.List(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat profil billing"})
	}
	return c.JSON(fiber.Map{"data": profiles})
}

func (h *BillingProfileHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	p, err := h.svc.GetByID(c.Context(), tenantID, c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrBillingProfileNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat profil billing"})
	}
	return c.JSON(fiber.Map{"data": p})
}

func (h *BillingProfileHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var input service.BillingProfileInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	p, err := h.svc.Create(c.Context(), tenantID, input)
	if err != nil {
		if isValidationErr(err) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat profil billing"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": p})
}

func (h *BillingProfileHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var input service.BillingProfileInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	p, err := h.svc.Update(c.Context(), tenantID, c.Params("id"), input)
	if err != nil {
		if errors.Is(err, service.ErrBillingProfileNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if isValidationErr(err) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui profil billing"})
	}
	return c.JSON(fiber.Map{"data": p})
}

func (h *BillingProfileHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if err := h.svc.Delete(c.Context(), tenantID, c.Params("id")); err != nil {
		if errors.Is(err, service.ErrBillingProfileNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus profil billing"})
	}
	return c.JSON(fiber.Map{"message": "Profil billing dihapus"})
}

func (h *BillingProfileHandler) SetDefault(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if err := h.svc.SetDefault(c.Context(), tenantID, c.Params("id")); err != nil {
		if errors.Is(err, service.ErrBillingProfileNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal set profil default"})
	}
	return c.JSON(fiber.Map{"message": "Profil billing diset sebagai default"})
}

func isValidationErr(err error) bool {
	return errors.Is(err, service.ErrBillingProfileNameEmpty) ||
		errors.Is(err, service.ErrBillingModelInvalid) ||
		errors.Is(err, service.ErrPaymentTimingInvalid)
}
