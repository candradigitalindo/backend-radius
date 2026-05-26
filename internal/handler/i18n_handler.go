package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/middleware"
	"github.com/candrasyahputra/radius-server/internal/pkg/i18n"
)

type I18nHandler struct{}

func NewI18nHandler() *I18nHandler {
	return &I18nHandler{}
}

// GetLanguages returns the list of supported languages.
func (h *I18nHandler) GetLanguages(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"languages":    i18n.SupportedLanguages(),
		"current_lang": middleware.GetLang(c),
	})
}

// GetTranslations returns all translations for the current language.
func (h *I18nHandler) GetTranslations(c *fiber.Ctx) error {
	lang := middleware.GetLang(c)

	translations := make(map[string]string, len(i18n.Messages))
	for key := range i18n.Messages {
		translations[key] = i18n.T(lang, key)
	}

	return c.JSON(fiber.Map{
		"lang":         lang,
		"translations": translations,
	})
}
