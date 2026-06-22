package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/service"
)

const saTenantID = "tnt-superadmin" // FK ke tenants.id
const saWASession = "superadmin"    // identifier sesi WA (bukan FK)

type SASettingsHandler struct {
	tenantService  *service.TenantService
	settingService *service.SettingService
	waClient       *whatsapp.Client
	baseURL        string
}

func NewSASettingsHandler(tenantService *service.TenantService, settingService *service.SettingService, waClient *whatsapp.Client) *SASettingsHandler {
	return &SASettingsHandler{tenantService: tenantService, settingService: settingService, waClient: waClient}
}

func (h *SASettingsHandler) WithBaseURL(url string) *SASettingsHandler {
	h.baseURL = url
	return h
}

// saSettingsResponse combines settings map.
type saSettingsResponse struct {
	PGProvider        string `json:"pg_provider"`
	PGMerchantID      string `json:"pg_merchant_id"`
	PGAPIKey          string `json:"pg_api_key"`
	PGSecretKey       string `json:"pg_secret_key"`
	PGSandbox         bool   `json:"pg_sandbox"`
	Timezone          string `json:"timezone"`
	DefaultTheme      string `json:"default_theme"`
	WANotifySubscribe bool   `json:"wa_notify_subscribe"`
	WANotifyPayment   bool   `json:"wa_notify_payment"`
	WANotifyDueDate   bool   `json:"wa_notify_due_date"`
	WANotifyOTP       bool   `json:"wa_notify_otp"`
	WANotifyBroadcast bool   `json:"wa_notify_broadcast"`
}

// GetSettings returns all superadmin settings.
func (h *SASettingsHandler) GetSettings(c *fiber.Ctx) error {
	settings, _ := h.settingService.GetAsMap(c.Context(), saTenantID)

	resp := saSettingsResponse{
		PGProvider:        getSettingDefault(settings, "sa_pg_provider", "midtrans"),
		PGMerchantID:      getSettingDefault(settings, "sa_pg_merchant_id", ""),
		PGAPIKey:          getSettingDefault(settings, "sa_pg_api_key", ""),
		PGSecretKey:       getSettingDefault(settings, "sa_pg_secret_key", ""),
		PGSandbox:         getSettingBool(settings, "sa_pg_sandbox", false),
		Timezone:          getSettingDefault(settings, "sa_timezone", "Asia/Jakarta"),
		DefaultTheme:      getSettingDefault(settings, "sa_app_theme", "light"),
		WANotifySubscribe: getSettingBool(settings, "sa_wa_notify_subscribe", true),
		WANotifyPayment:   getSettingBool(settings, "sa_wa_notify_payment", true),
		WANotifyDueDate:   getSettingBool(settings, "sa_wa_notify_due_date", true),
		WANotifyOTP:       getSettingBool(settings, "sa_wa_notify_otp", true),
		WANotifyBroadcast: getSettingBool(settings, "sa_wa_notify_broadcast", true),
	}

	return c.JSON(fiber.Map{"data": resp})
}

type updateSASettingsRequest struct {
	// WA notification toggles
	WANotifySubscribe *bool `json:"wa_notify_subscribe"`
	WANotifyPayment   *bool `json:"wa_notify_payment"`
	WANotifyDueDate   *bool `json:"wa_notify_due_date"`
	WANotifyOTP       *bool `json:"wa_notify_otp"`
	WANotifyBroadcast *bool `json:"wa_notify_broadcast"`
	// PG
	PGProvider   *string `json:"pg_provider"`
	PGMerchantID *string `json:"pg_merchant_id"`
	PGAPIKey     *string `json:"pg_api_key"`
	PGSecretKey  *string `json:"pg_secret_key"`
	PGSandbox    *bool   `json:"pg_sandbox"`
	// General
	Timezone     *string `json:"timezone"`
	DefaultTheme *string `json:"default_theme"`
}

// UpdateSettings updates all superadmin settings.
func (h *SASettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	var req updateSASettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	settingsUpdates := map[string]string{}

	if req.PGProvider != nil {
		settingsUpdates["sa_pg_provider"] = *req.PGProvider
	}
	if req.PGMerchantID != nil {
		settingsUpdates["sa_pg_merchant_id"] = *req.PGMerchantID
	}
	if req.PGAPIKey != nil {
		settingsUpdates["sa_pg_api_key"] = *req.PGAPIKey
	}
	if req.PGSecretKey != nil {
		settingsUpdates["sa_pg_secret_key"] = *req.PGSecretKey
	}
	if req.PGSandbox != nil {
		settingsUpdates["sa_pg_sandbox"] = boolToStr(*req.PGSandbox)
	}
	if req.Timezone != nil && *req.Timezone != "" {
		settingsUpdates["sa_timezone"] = *req.Timezone
	}
	if req.DefaultTheme != nil {
		switch *req.DefaultTheme {
		case "light", "dark":
			settingsUpdates["sa_app_theme"] = *req.DefaultTheme
		}
	}
	if req.WANotifySubscribe != nil {
		settingsUpdates["sa_wa_notify_subscribe"] = boolToStr(*req.WANotifySubscribe)
	}
	if req.WANotifyPayment != nil {
		settingsUpdates["sa_wa_notify_payment"] = boolToStr(*req.WANotifyPayment)
	}
	if req.WANotifyDueDate != nil {
		settingsUpdates["sa_wa_notify_due_date"] = boolToStr(*req.WANotifyDueDate)
	}
	if req.WANotifyOTP != nil {
		settingsUpdates["sa_wa_notify_otp"] = boolToStr(*req.WANotifyOTP)
	}
	if req.WANotifyBroadcast != nil {
		settingsUpdates["sa_wa_notify_broadcast"] = boolToStr(*req.WANotifyBroadcast)
	}

	if len(settingsUpdates) > 0 {
		_ = h.settingService.BulkSet(c.Context(), saTenantID, settingsUpdates)
	}

	return c.JSON(fiber.Map{"message": "Pengaturan berhasil disimpan"})
}

// GetWebhookURLs returns the subscription payment gateway callback URLs for the superadmin dashboard.
func (h *SASettingsHandler) GetWebhookURLs(c *fiber.Ctx) error {
	base := h.baseURL
	return c.JSON(fiber.Map{
		"success": true,
		"data": map[string]string{
			"tripay":   base + "/api/v1/webhooks/subscription/tripay",
			"midtrans": base + "/api/v1/webhooks/subscription/midtrans",
			"xendit":   base + "/api/v1/webhooks/subscription/xendit",
		},
	})
}

// TestPGConnection actually verifies the superadmin payment-gateway credentials by
// calling the provider's API. Credentials may be sent in the body (to test the
// values currently on screen) or fall back to the saved superadmin settings.
func (h *SASettingsHandler) TestPGConnection(c *fiber.Ctx) error {
	var req struct {
		Provider   string `json:"pg_provider"`
		APIKey     string `json:"pg_api_key"`
		SecretKey  string `json:"pg_secret_key"`
		MerchantID string `json:"pg_merchant_id"`
		Sandbox    *bool  `json:"pg_sandbox"`
	}
	_ = c.BodyParser(&req)

	settings, _ := h.settingService.GetAsMap(c.Context(), saTenantID)
	pick := func(v, key, def string) string {
		if v != "" {
			return v
		}
		return getSettingDefault(settings, key, def)
	}
	provider := pick(req.Provider, "sa_pg_provider", "midtrans")
	apiKey := pick(req.APIKey, "sa_pg_api_key", "")
	secretKey := pick(req.SecretKey, "sa_pg_secret_key", "")
	merchantID := pick(req.MerchantID, "sa_pg_merchant_id", "")
	sandbox := getSettingBool(settings, "sa_pg_sandbox", false)
	if req.Sandbox != nil {
		sandbox = *req.Sandbox
	}

	bad := func(msg string) error {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": msg})
	}

	var err error
	switch provider {
	case "midtrans":
		if secretKey == "" {
			return bad("Server Key (Secret Key) Midtrans belum diisi")
		}
		err = payment.NewMidtransClient(secretKey, sandbox).TestConnection(c.Context())
	case "xendit":
		if apiKey == "" {
			return bad("Secret API Key Xendit belum diisi")
		}
		err = payment.NewXenditClient(apiKey, secretKey, sandbox).TestConnection(c.Context())
	case "tripay":
		if apiKey == "" {
			return bad("API Key Tripay belum diisi")
		}
		err = payment.NewTripayClient(apiKey, secretKey, merchantID, sandbox).TestConnection(c.Context())
	default:
		return bad("Provider payment gateway tidak dikenal")
	}

	if err != nil {
		return c.JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Koneksi payment gateway berhasil"})
}

// ─── WhatsApp Session endpoints for superadmin ───

// StartWASession starts a WhatsApp session for superadmin.
func (h *SASettingsHandler) StartWASession(c *fiber.Ctx) error {
	if h.waClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "WhatsApp service tidak dikonfigurasi"})
	}

	result, err := h.waClient.StartSession(c.Context(), saWASession)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "WhatsApp service tidak tersedia: " + err.Error()})
	}

	return c.JSON(result)
}

// GetWASessionStatus returns the WhatsApp session status for superadmin.
func (h *SASettingsHandler) GetWASessionStatus(c *fiber.Ctx) error {
	if h.waClient == nil {
		return c.JSON(fiber.Map{"status": "disconnected", "message": "WhatsApp service tidak dikonfigurasi"})
	}

	result, err := h.waClient.GetSessionStatus(c.Context(), saWASession)
	if err != nil {
		return c.JSON(fiber.Map{"status": "disconnected", "message": "WhatsApp service tidak tersedia"})
	}

	return c.JSON(result)
}

// GetWAQR returns the QR code for superadmin WhatsApp pairing.
func (h *SASettingsHandler) GetWAQR(c *fiber.Ctx) error {
	if h.waClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "WhatsApp service tidak dikonfigurasi"})
	}

	result, err := h.waClient.GetQR(c.Context(), saWASession)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "Gagal memuat QR: " + err.Error()})
	}

	return c.JSON(result)
}

// StopWASession stops the WhatsApp session for superadmin.
func (h *SASettingsHandler) StopWASession(c *fiber.Ctx) error {
	if h.waClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "WhatsApp service tidak dikonfigurasi"})
	}

	result, err := h.waClient.StopSession(c.Context(), saWASession)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "Gagal menghentikan sesi: " + err.Error()})
	}

	return c.JSON(result)
}

func getSettingDefault(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

func getSettingBool(m map[string]string, key string, def bool) bool {
	v, ok := m[key]
	if !ok {
		return def
	}
	return v == "true"
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
