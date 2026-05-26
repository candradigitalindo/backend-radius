package handler

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

// IsolirPageHandler serves the custom isolation landing page per tenant.
type IsolirPageHandler struct {
	settingService  *service.SettingService
	tenantService   *service.TenantService
	customerService *service.CustomerService
}

func NewIsolirPageHandler(settingService *service.SettingService, tenantService *service.TenantService, customerService *service.CustomerService) *IsolirPageHandler {
	return &IsolirPageHandler{
		settingService:  settingService,
		tenantService:   tenantService,
		customerService: customerService,
	}
}

// GetPage serves the isolation page for a customer by tenant slug and customer code.
// GET /isolir/:tenant_slug/:customer_code
func (h *IsolirPageHandler) GetPage(c *fiber.Ctx) error {
	tenantSlug := c.Params("tenant_slug")
	customerCode := c.Params("customer_code")

	tenant, err := h.tenantService.GetBySlug(c.Context(), tenantSlug)
	if err != nil || tenant == nil {
		return c.Status(fiber.StatusNotFound).SendString("Halaman tidak ditemukan")
	}

	customer, err := h.customerService.GetByCode(c.Context(), tenant.ID, customerCode)
	if err != nil || customer == nil {
		return c.Status(fiber.StatusNotFound).SendString("Pelanggan tidak ditemukan")
	}

	if customer.Status != "isolated" {
		return c.Status(fiber.StatusOK).SendString("Akun Anda aktif, tidak ada masalah.")
	}

	// Get custom settings
	settings, _ := h.settingService.GetAsMap(c.Context(), tenant.ID)

	title := getOrDefault(settings, "isolir_page_title", "Akun Anda Terisolir")
	message := getOrDefault(settings, "isolir_page_message", "Layanan internet Anda sementara dinonaktifkan karena tagihan belum dibayar. Silakan lakukan pembayaran untuk mengaktifkan kembali.")
	bgColor := getOrDefault(settings, "isolir_page_bg_color", "#f8d7da")
	textColor := getOrDefault(settings, "isolir_page_text_color", "#721c24")
	logoURL := getOrDefault(settings, "isolir_page_logo", tenant.LogoURL)
	contactPhone := getOrDefault(settings, "isolir_page_contact_phone", tenant.Phone)
	contactWA := getOrDefault(settings, "isolir_page_contact_wa", tenant.Phone)
	paymentURL := getOrDefault(settings, "isolir_page_payment_url", "")
	customCSS := getOrDefault(settings, "isolir_page_custom_css", "")
	customHTML := getOrDefault(settings, "isolir_page_custom_html", "")
	darkMode := getOrDefault(settings, "isolir_page_dark_mode", "auto")

	// If tenant provides full custom HTML, serve with CSP to block all scripts
	if customHTML != "" {
		safe := stripScriptTags(customHTML)
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Content-Security-Policy", "script-src 'none'; object-src 'none'")
		return c.SendString(safe)
	}

	// Build default isolation page
	page := buildIsolirPage(
		html.EscapeString(tenant.Name),
		html.EscapeString(title),
		html.EscapeString(message),
		html.EscapeString(customer.Name),
		html.EscapeString(customer.CustomerCode),
		html.EscapeString(bgColor),
		html.EscapeString(textColor),
		html.EscapeString(logoURL),
		html.EscapeString(contactPhone),
		html.EscapeString(contactWA),
		html.EscapeString(paymentURL),
		sanitizeCSS(customCSS),
		html.EscapeString(darkMode),
	)

	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Content-Security-Policy", "script-src 'none'; object-src 'none'")
	return c.SendString(page)
}

// GetConfig returns the isolir page configuration for admin editing.
// GET /v1/settings/isolir-page
func (h *IsolirPageHandler) GetConfig(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	settings, _ := h.settingService.GetAsMap(c.Context(), tenantID)

	config := map[string]string{
		"isolir_page_title":         getOrDefault(settings, "isolir_page_title", "Akun Anda Terisolir"),
		"isolir_page_message":       getOrDefault(settings, "isolir_page_message", "Layanan internet Anda sementara dinonaktifkan karena tagihan belum dibayar."),
		"isolir_page_bg_color":      getOrDefault(settings, "isolir_page_bg_color", "#f8d7da"),
		"isolir_page_text_color":    getOrDefault(settings, "isolir_page_text_color", "#721c24"),
		"isolir_page_logo":          getOrDefault(settings, "isolir_page_logo", ""),
		"isolir_page_contact_phone": getOrDefault(settings, "isolir_page_contact_phone", ""),
		"isolir_page_contact_wa":    getOrDefault(settings, "isolir_page_contact_wa", ""),
		"isolir_page_payment_url":   getOrDefault(settings, "isolir_page_payment_url", ""),
		"isolir_page_custom_css":    getOrDefault(settings, "isolir_page_custom_css", ""),
		"isolir_page_custom_html":   getOrDefault(settings, "isolir_page_custom_html", ""),
		"isolir_page_dark_mode":     getOrDefault(settings, "isolir_page_dark_mode", "auto"),
	}

	return c.JSON(fiber.Map{"data": config})
}

// UpdateConfig bulk-updates the isolir page settings.
// PUT /v1/settings/isolir-page
func (h *IsolirPageHandler) UpdateConfig(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req map[string]string
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	// Only allow isolir_page_ prefixed keys
	allowed := make(map[string]string)
	for k, v := range req {
		if len(k) > 12 && k[:12] == "isolir_page_" {
			allowed[k] = v
		}
	}

	if len(allowed) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak ada pengaturan yang valid"})
	}

	if err := h.settingService.BulkSet(c.Context(), tenantID, allowed); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pengaturan"})
	}

	return c.JSON(fiber.Map{"message": "Pengaturan halaman isolir diperbarui"})
}

func getOrDefault(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

// sanitizeCSS strips sequences that could break out of a <style> tag.
func sanitizeCSS(css string) string {
	// Remove </style> variants (case-insensitive) to prevent tag breakout
	css = strings.ReplaceAll(css, "</style>", "")
	css = strings.ReplaceAll(css, "</Style>", "")
	css = strings.ReplaceAll(css, "</STYLE>", "")
	// Remove <script tags and HTML open/close tags
	css = strings.ReplaceAll(css, "<script", "")
	css = strings.ReplaceAll(css, "<Script", "")
	css = strings.ReplaceAll(css, "<SCRIPT", "")
	css = strings.ReplaceAll(css, "</", "")
	return css
}

// stripScriptTags removes <script>, <iframe>, <object>, <embed>, <svg> tags
// and event handler attributes (case-insensitive) from HTML.
var (
	reScriptTag  = regexp.MustCompile(`(?i)<\s*/?(?:script|iframe|object|embed|applet|meta)[^>]*>`)
	reEventAttrs = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]*)`)
	reSvgOnload  = regexp.MustCompile(`(?i)<\s*svg[^>]*\bon\w+\s*=[^>]*>`)
	reJavascript = regexp.MustCompile(`(?i)(?:href|src|action)\s*=\s*(?:"javascript:[^"]*"|'javascript:[^']*')`)
)

func stripScriptTags(input string) string {
	s := reScriptTag.ReplaceAllString(input, "")
	s = reEventAttrs.ReplaceAllString(s, "")
	s = reSvgOnload.ReplaceAllString(s, "<svg>")
	s = reJavascript.ReplaceAllString(s, "")
	return s
}

func buildIsolirPage(tenantName, title, message, customerName, customerCode, bgColor, textColor, logoURL, contactPhone, contactWA, paymentURL, customCSS, darkMode string) string {
	logo := ""
	if logoURL != "" {
		logo = fmt.Sprintf(`<img src="%s" alt="%s" style="max-width:180px;margin-bottom:20px;">`, logoURL, tenantName)
	}

	paymentSection := ""
	if paymentURL != "" {
		paymentSection = fmt.Sprintf(`<a href="%s" style="display:inline-block;padding:12px 30px;background:%s;color:%s;border-radius:8px;text-decoration:none;font-weight:bold;margin-top:15px;border:2px solid %s;">Bayar Sekarang</a>`, paymentURL, textColor, bgColor, textColor)
	}

	contactSection := ""
	if contactPhone != "" {
		contactSection = fmt.Sprintf(`<p style="margin-top:20px;">Hubungi kami: <a href="tel:%s" style="color:%s;font-weight:bold;">%s</a></p>`, contactPhone, textColor, contactPhone)
	}
	if contactWA != "" {
		contactSection += fmt.Sprintf(`<p><a href="https://wa.me/%s" style="color:%s;font-weight:bold;" target="_blank" rel="noopener">Chat WhatsApp</a></p>`, contactWA, textColor)
	}

	// Dark mode CSS
	darkCSS := ""
	darkStyles := `
body.dark-mode{background:#1a1a2e;color:#e0e0e0}
body.dark-mode .container{background:#16213e;box-shadow:0 4px 20px rgba(0,0,0,0.4)}
body.dark-mode .customer-info{background:#0f3460;color:#a0a0c0}
body.dark-mode .message{color:#c0c0d0}
body.dark-mode a{color:#4fc3f7}
`
	switch darkMode {
	case "on":
		darkCSS = darkStyles + `body{background:#1a1a2e;color:#e0e0e0}.container{background:#16213e;box-shadow:0 4px 20px rgba(0,0,0,0.4)}.customer-info{background:#0f3460;color:#a0a0c0}.message{color:#c0c0d0}a{color:#4fc3f7}`
	case "auto":
		darkCSS = fmt.Sprintf(`@media(prefers-color-scheme:dark){%s}`, darkStyles)
	}

	bodyClass := ""
	if darkMode == "on" {
		bodyClass = ` class="dark-mode"`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - %s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:%s;color:%s;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:20px;transition:background .3s,color .3s}
.container{background:white;border-radius:16px;padding:40px;max-width:500px;width:100%%;text-align:center;box-shadow:0 4px 20px rgba(0,0,0,0.1);transition:background .3s,box-shadow .3s}
h1{font-size:1.5em;margin-bottom:10px}
.customer-info{background:#f8f9fa;border-radius:8px;padding:12px;margin:15px 0;font-size:0.9em;color:#666;transition:background .3s,color .3s}
.message{font-size:1em;line-height:1.6;margin:15px 0;color:#333;transition:color .3s}
%s
%s
</style>
</head>
<body%s>
<div class="container">
%s
<h1>%s</h1>
<div class="customer-info">
<strong>%s</strong><br>Kode: %s
</div>
<p class="message">%s</p>
%s
%s
<p style="margin-top:30px;font-size:0.8em;color:#999;">&copy; %s</p>
</div>
</body>
</html>`, title, tenantName, bgColor, textColor, customCSS, darkCSS, bodyClass, logo, title, customerName, customerCode, message, paymentSection, contactSection, tenantName)
}
