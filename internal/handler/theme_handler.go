package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

// ThemeHandler handles user theme/dark mode preferences.
type ThemeHandler struct {
	settingService *service.SettingService
}

func NewThemeHandler(settingService *service.SettingService) *ThemeHandler {
	return &ThemeHandler{settingService: settingService}
}

type themePreferencesResponse struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

type updateThemeRequest struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

func userThemeKey(userID string) string {
	return fmt.Sprintf("user_theme_%s", userID)
}

func userLangKey(userID string) string {
	return fmt.Sprintf("user_lang_%s", userID)
}

// GetPreferences returns the current user's UI preferences (theme, language).
// GET /v1/user/preferences
func (h *ThemeHandler) GetPreferences(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	tenantID, _ := c.Locals("tenant_id").(string)

	settings, _ := h.settingService.GetAsMap(c.Context(), tenantID)

	theme := getOrDefault(settings, userThemeKey(userID), "")
	if theme == "" {
		theme = getOrDefault(settings, "app_theme", "light")
	}

	lang := getOrDefault(settings, userLangKey(userID), "")
	if lang == "" {
		lang = getOrDefault(settings, "app_language", "id")
	}

	return c.JSON(fiber.Map{
		"data": themePreferencesResponse{
			Theme:    theme,
			Language: lang,
		},
	})
}

// UpdatePreferences updates the current user's UI preferences.
// PUT /v1/user/preferences
func (h *ThemeHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	tenantID, _ := c.Locals("tenant_id").(string)

	var req updateThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	updates := make(map[string]string)

	if req.Theme != "" {
		switch req.Theme {
		case "light", "dark", "system":
			updates[userThemeKey(userID)] = req.Theme
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tema harus light, dark, atau system"})
		}
	}

	if req.Language != "" {
		switch req.Language {
		case "id", "en":
			updates[userLangKey(userID)] = req.Language
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bahasa tidak didukung"})
		}
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada preferensi yang valid"})
	}

	if err := h.settingService.BulkSet(c.Context(), tenantID, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui preferensi"})
	}

	return c.JSON(fiber.Map{"message": "Preferensi diperbarui"})
}

// GetTenantTheme returns the tenant-level default theme.
// GET /v1/settings/theme
func (h *ThemeHandler) GetTenantTheme(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	settings, _ := h.settingService.GetAsMap(c.Context(), tenantID)

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"app_theme":    getOrDefault(settings, "app_theme", "light"),
			"app_language": getOrDefault(settings, "app_language", "id"),
		},
	})
}

// UpdateTenantTheme updates the tenant-level default theme.
// PUT /v1/settings/theme
func (h *ThemeHandler) UpdateTenantTheme(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req updateThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	updates := make(map[string]string)

	if req.Theme != "" {
		switch req.Theme {
		case "light", "dark", "system":
			updates["app_theme"] = req.Theme
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tema harus light, dark, atau system"})
		}
	}

	if req.Language != "" {
		switch req.Language {
		case "id", "en":
			updates["app_language"] = req.Language
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bahasa tidak didukung"})
		}
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada pengaturan yang valid"})
	}

	if err := h.settingService.BulkSet(c.Context(), tenantID, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pengaturan tema"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan tema diperbarui"})
}
