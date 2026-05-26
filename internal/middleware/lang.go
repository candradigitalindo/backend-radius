package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/pkg/i18n"
)

// LangMiddleware extracts the language preference from the request.
// Priority: 1) ?lang= query param, 2) Accept-Language header, 3) default "id"
func LangMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		lang := c.Query("lang")
		if lang == "" {
			lang = c.Get("Accept-Language")
		}

		// Normalize — only keep supported language prefix
		switch {
		case len(lang) >= 2 && (lang[:2] == "en" || lang[:2] == "EN"):
			lang = i18n.LangEN
		default:
			lang = i18n.LangID
		}

		c.Locals("lang", lang)
		return c.Next()
	}
}

// GetLang retrieves the language from fiber context.
func GetLang(c *fiber.Ctx) string {
	if lang, ok := c.Locals("lang").(string); ok && lang != "" {
		return lang
	}
	return i18n.LangID
}
