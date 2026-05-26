package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type SettingHandler struct {
	settingService *service.SettingService
}

func NewSettingHandler(settingService *service.SettingService) *SettingHandler {
	return &SettingHandler{settingService: settingService}
}

type setSettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type bulkSetRequest struct {
	Settings map[string]string `json:"settings"`
}

func (h *SettingHandler) Set(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req setSettingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Key wajib diisi"})
	}

	if err := h.settingService.Set(c.Context(), tenantID, req.Key, req.Value); err != nil {
		if errors.Is(err, service.ErrSettingKeyInvalid) || errors.Is(err, service.ErrSettingValueLimit) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan pengaturan"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan tersimpan", "key": req.Key})
}

func (h *SettingHandler) Get(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	key := c.Params("key")

	setting, err := h.settingService.Get(c.Context(), tenantID, key)
	if err != nil {
		if errors.Is(err, service.ErrSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(setting)
}

func (h *SettingHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	key := c.Params("key")

	if err := h.settingService.Delete(c.Context(), tenantID, key); err != nil {
		if errors.Is(err, service.ErrSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan dihapus"})
}

func (h *SettingHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	settings, err := h.settingService.GetAsMap(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"data": settings})
}

func (h *SettingHandler) BulkSet(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req bulkSetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if len(req.Settings) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data pengaturan wajib diisi"})
	}

	if err := h.settingService.BulkSet(c.Context(), tenantID, req.Settings); err != nil {
		if errors.Is(err, service.ErrSettingKeyInvalid) || errors.Is(err, service.ErrSettingValueLimit) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan pengaturan"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan tersimpan"})
}
