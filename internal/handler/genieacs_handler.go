package handler

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type GenieACSHandler struct {
	genieacsService *service.GenieACSService
}

func NewGenieACSHandler(genieacsService *service.GenieACSService) *GenieACSHandler {
	return &GenieACSHandler{genieacsService: genieacsService}
}

// mongoRegexSanitizer strips characters that have special meaning in MongoDB regex/JSON.
var mongoRegexSanitizer = regexp.MustCompile(`[^\w\-.]`)

func (h *GenieACSHandler) ListDevices(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		if search := c.Query("search"); search != "" {
			safe := mongoRegexSanitizer.ReplaceAllString(search, "")
			if safe == "" {
				return c.JSON(fiber.Map{"data": []interface{}{}})
			}
			query = fmt.Sprintf(`{"_deviceId._SerialNumber":{"$regex":"%s"}}`, safe)
		}
	}

	devices, err := h.genieacsService.ListDevices(c.Context(), query)
	if err != nil {
		// Return empty array when GenieACS is unreachable
		return c.JSON(fiber.Map{"data": []interface{}{}})
	}

	return c.JSON(fiber.Map{"data": devices})
}

func (h *GenieACSHandler) ListTasks(c *fiber.Ctx) error {
	device := c.Query("device")

	tasks, err := h.genieacsService.ListTasks(c.Context(), device)
	if err != nil {
		return c.JSON(fiber.Map{"data": []interface{}{}})
	}

	return c.JSON(fiber.Map{"data": tasks})
}

func (h *GenieACSHandler) GetDevice(c *fiber.Ctx) error {
	deviceID := c.Params("id")

	device, err := h.genieacsService.GetDevice(c.Context(), deviceID)
	if err != nil {
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data perangkat"})
	}

	return c.JSON(device)
}

func (h *GenieACSHandler) SyncONT(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	ont, err := h.genieacsService.SyncONT(c.Context(), tenantID, ontID)
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to sync ONT"})
	}

	return c.JSON(ont)
}

func (h *GenieACSHandler) RebootDevice(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	if err := h.genieacsService.RebootDevice(c.Context(), tenantID, ontID); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal me-reboot perangkat"})
	}

	return c.JSON(fiber.Map{"message": "Perintah reboot terkirim"})
}

type setWiFiRequest struct {
	SSID     string `json:"ssid" form:"ssid"`
	Password string `json:"password" form:"password"`
	// Security: "None" (open), "WPA" (WPA), "11i" (WPA2), "WPAand11i" (WPA/WPA2 Mixed)
	// Default: "11i" jika tidak diisi
	Security string `json:"security" form:"security"`

	WiFiSSID         string `json:"wifi_ssid" form:"wifi_ssid"`
	WiFiSSIDCamel    string `json:"wifiSSID" form:"wifiSSID"`
	WiFiPassword     string `json:"wifi_password" form:"wifi_password"`
	WiFiPasswordCamel string `json:"wifiPassword" form:"wifiPassword"`
	Passphrase       string `json:"passphrase" form:"passphrase"`
	WiFiSecurity     string `json:"wifi_security" form:"wifi_security"`
	WiFiSecurityCamel string `json:"wifiSecurity" form:"wifiSecurity"`
}

func (r *setWiFiRequest) normalize() {
	if r.SSID == "" {
		switch {
		case r.WiFiSSID != "":
			r.SSID = r.WiFiSSID
		case r.WiFiSSIDCamel != "":
			r.SSID = r.WiFiSSIDCamel
		}
	}

	if r.Password == "" {
		switch {
		case r.WiFiPassword != "":
			r.Password = r.WiFiPassword
		case r.WiFiPasswordCamel != "":
			r.Password = r.WiFiPasswordCamel
		case r.Passphrase != "":
			r.Password = r.Passphrase
		}
	}

	if r.Security == "" {
		switch {
		case r.WiFiSecurity != "":
			r.Security = r.WiFiSecurity
		case r.WiFiSecurityCamel != "":
			r.Security = r.WiFiSecurityCamel
		}
	}
}

func (r setWiFiRequest) validate() error {
	if r.SSID == "" {
		return errors.New("SSID wajib diisi")
	}

	if r.Security == "None" {
		return nil
	}

	if r.Password == "" {
		return nil
	}

	if length := len(r.Password); length < 8 || length > 63 {
		return errors.New("Password WiFi WPA/WPA2 harus 8-63 karakter")
	}

	return nil
}

func (h *GenieACSHandler) SetWiFi(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	var req setWiFiRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.normalize()

	if err := req.validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.genieacsService.SetWiFiSSID(c.Context(), tenantID, ontID, req.SSID, req.Password, req.Security); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengkonfigurasi WiFi"})
	}

	return c.JSON(fiber.Map{"message": "Konfigurasi WiFi terkirim"})
}

func (h *GenieACSHandler) ProvisionONT(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	if err := h.genieacsService.ProvisionONT(c.Context(), tenantID, ontID); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memprovisioning ONT"})
	}

	return c.JSON(fiber.Map{"message": "ONT berhasil diprovisioning"})
}

func (h *GenieACSHandler) GetDiagnostics(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	info, err := h.genieacsService.GetDeviceDiagnostics(c.Context(), tenantID, ontID)
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menjalankan diagnostik"})
	}

	return c.JSON(info)
}

// --- Serial-number-based endpoints (per roadmap: /genieacs/devices/:serial) ---

func (h *GenieACSHandler) GetDeviceBySerial(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	serial := c.Params("serial")

	device, err := h.genieacsService.GetDeviceBySerial(c.Context(), tenantID, serial)
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data perangkat"})
	}

	return c.JSON(device)
}

func (h *GenieACSHandler) GetDeviceStatus(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	serial := c.Params("serial")

	info, err := h.genieacsService.GetDeviceStatus(c.Context(), tenantID, serial)
	if err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get device status"})
	}

	return c.JSON(info)
}

func (h *GenieACSHandler) RebootBySerial(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	serial := c.Params("serial")

	if err := h.genieacsService.RebootBySerial(c.Context(), tenantID, serial); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal me-reboot perangkat"})
	}

	return c.JSON(fiber.Map{"message": "Perintah reboot terkirim"})
}

func (h *GenieACSHandler) SetWiFiBySerial(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	serial := c.Params("serial")

	var req setWiFiRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.normalize()
	if err := req.validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.genieacsService.SetWiFiBySerial(c.Context(), tenantID, serial, req.SSID, req.Password, req.Security); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengkonfigurasi WiFi"})
	}

	return c.JSON(fiber.Map{"message": "Konfigurasi WiFi terkirim"})
}

func (h *GenieACSHandler) ONTDashboard(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	items, err := h.genieacsService.GetONTDashboard(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil dashboard perangkat"})
	}

	return c.JSON(fiber.Map{"data": items, "total": len(items)})
}

type setPPPoERequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetPPPoE mendorong PPPoE username & password baru ke perangkat ONT via TR-069.
func (h *GenieACSHandler) SetPPPoE(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	ontID := c.Params("id")

	var req setPPPoERequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username dan password wajib diisi"})
	}

	if err := h.genieacsService.SetPPPoECredentials(c.Context(), tenantID, ontID, req.Username, req.Password); err != nil {
		if errors.Is(err, service.ErrONTNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrGenieACSDeviceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perangkat tidak ditemukan di TR-069 ACS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengkonfigurasi PPPoE"})
	}

	return c.JSON(fiber.Map{"message": "Konfigurasi PPPoE terkirim ke perangkat"})
}

// BulkProvision memprovisioning semua ONT aktif tenant yang belum terprovisi.
// Digunakan untuk migrasi pelanggan lama ke GenieACS.
func (h *GenieACSHandler) BulkProvision(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.genieacsService.BulkProvision(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menjalankan bulk provisioning"})
	}

	return c.JSON(result)
}

// DiscoverONTs memindai semua perangkat di GenieACS dan mendaftarkan yang belum ada di database
// sebagai ONT dengan status "discovered". Admin kemudian bisa assign ke pelanggan.
func (h *GenieACSHandler) DiscoverONTs(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.genieacsService.DiscoverONTs(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menjalankan discovery perangkat"})
	}

	return c.JSON(result)
}

// AutoMatchONTs mencoba menghubungkan ONT yang belum ter-assign ke pelanggan
// menggunakan PPPoE username dari parameter TR-069 perangkat.
func (h *GenieACSHandler) AutoMatchONTs(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.genieacsService.AutoMatchONTs(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menjalankan auto-match perangkat"})
	}

	return c.JSON(result)
}
