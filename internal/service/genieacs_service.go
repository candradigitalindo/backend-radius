package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/genieacs"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrGenieACSDeviceNotFound = errors.New("Perangkat tidak ditemukan di GenieACS")
)

// GenieACSService provides TR-069 auto-provisioning and device management.
type GenieACSService struct {
	client       *genieacs.Client
	ontRepo      repository.ONTRepository
	customerRepo repository.CustomerRepository
	waClient     *whatsapp.Client
	settingRepo  repository.SettingRepository
}

func NewGenieACSService(client *genieacs.Client, ontRepo repository.ONTRepository, customerRepo repository.CustomerRepository) *GenieACSService {
	return &GenieACSService{client: client, ontRepo: ontRepo, customerRepo: customerRepo}
}

func (s *GenieACSService) WithWAClient(wa *whatsapp.Client) *GenieACSService {
	s.waClient = wa
	return s
}

func (s *GenieACSService) WithSettingRepo(r repository.SettingRepository) *GenieACSService {
	s.settingRepo = r
	return s
}

// DeviceInfo holds relevant info from a GenieACS-managed device.
type ConnectedHost struct {
	IPAddress     string `json:"ip_address"`
	MACAddress    string `json:"mac_address"`
	HostName      string `json:"hostname"`
	InterfaceType string `json:"interface_type"` // e.g. "802.11" (WiFi) or "Ethernet"
	Active        bool   `json:"active"`
}

type DeviceInfo struct {
	DeviceID          string          `json:"device_id"`
	SerialNumber      string          `json:"serial_number"`
	Manufacturer      string          `json:"manufacturer"`
	ProductClass      string          `json:"product_class"`
	SoftwareVersion   string          `json:"software_version,omitempty"`
	Uptime            *int64          `json:"uptime,omitempty"`
	RxPower           *float64        `json:"rx_power,omitempty"`
	TxPower           *float64        `json:"tx_power,omitempty"`
	WiFiSSID          string          `json:"wifi_ssid,omitempty"`
	WiFiPassword      string          `json:"wifi_password"`       // tampilkan meski kosong, "" = belum ada password
	WiFiPasswordReported bool         `json:"wifi_password_reported"`
	WiFiPasswordSource string         `json:"wifi_password_source,omitempty"`
	WiFiSecurity      string          `json:"wifi_security,omitempty"`
	WiFiSecurityModes []string        `json:"wifi_security_modes,omitempty"`
	WANIP             string          `json:"wan_ip,omitempty"`
	PPPoEUsername     string          `json:"pppoe_username,omitempty"`
	LastInform        string          `json:"last_inform,omitempty"`
	DataRealtime      bool            `json:"data_realtime"`
	ConnectedHostCount int            `json:"connected_host_count"`
	ConnectedHostsComplete bool       `json:"connected_hosts_complete"`
	ConnectedHostsSource string       `json:"connected_hosts_source,omitempty"`
	ConnectedHosts    []ConnectedHost `json:"connected_hosts"`
}

// ONTDashboardItem represents one ONT in the dashboard view.
type ONTDashboardItem struct {
	ONTID        string   `json:"ont_id"`
	SerialNumber string   `json:"serial_number"`
	CustomerName string   `json:"customer_name,omitempty"`
	Status       string   `json:"status"`
	RxPower      *float64 `json:"rx_power,omitempty"`
	TxPower      *float64 `json:"tx_power,omitempty"`
}

// ListDevices lists all GenieACS-managed devices, optionally filtered.
func (s *GenieACSService) ListDevices(ctx context.Context, query string) ([]genieacs.Device, error) {
	return s.client.ListDevices(ctx, query)
}

// ListTasks lists tasks from GenieACS, optionally filtered by device.
func (s *GenieACSService) ListTasks(ctx context.Context, device string) ([]genieacs.Task, error) {
	query := ""
	if device != "" {
		query = fmt.Sprintf(`{"device":"%s"}`, device)
	}
	return s.client.ListTasks(ctx, query)
}

// GetDevice gets a single device from GenieACS.
func (s *GenieACSService) GetDevice(ctx context.Context, deviceID string) (*genieacs.Device, error) {
	device, err := s.client.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, ErrGenieACSDeviceNotFound
	}
	return device, nil
}

// SyncONT syncs optical power readings and metadata from GenieACS to the ONT record.
func (s *GenieACSService) SyncONT(ctx context.Context, tenantID, ontID string) (*model.ONT, error) {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return nil, err
	}
	if ont == nil {
		return nil, ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("genieacs sync: %w", err)
	}
	if device == nil {
		return nil, ErrGenieACSDeviceNotFound
	}

	// Extract optical power from TR-069 parameters if available
	if rxParam, ok := device.Parameters["InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.RXPower"]; ok {
		if val, ok := rxParam.Value.(float64); ok {
			ont.RxPower = &val
		}
	}
	if txParam, ok := device.Parameters["InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.TXPower"]; ok {
		if val, ok := txParam.Value.(float64); ok {
			ont.TxPower = &val
		}
	}

	if err := s.ontRepo.Update(ctx, ont); err != nil {
		return nil, err
	}
	return s.ontRepo.FindByID(ctx, tenantID, ontID)
}

// RebootDevice sends a reboot command to a device via GenieACS.
func (s *GenieACSService) RebootDevice(ctx context.Context, tenantID, ontID string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return err
	}
	if ont == nil {
		return ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return fmt.Errorf("genieacs reboot: %w", err)
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}

	_, err = s.client.Reboot(ctx, device.ID)
	return err
}

func normalizeWiFiSecurity(password, security string) string {
	switch security {
	case "None", "WPA", "11i", "WPAand11i":
		return security
	default:
		return "11i"
	}
}

// buildWiFiParams membangun daftar TR-069 parameter untuk konfigurasi WiFi.
// security: "None" (open/no password), "WPA", "11i" (WPA2), "WPAand11i". Default: "11i"
func buildWiFiParams(ssid, password, security string) []genieacs.TaskParam {
	const base = "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1."

	security = normalizeWiFiSecurity(password, security)

	params := []genieacs.TaskParam{
		{Name: base + "SSID", Value: ssid, Type: "xsd:string"},
		{Name: base + "BeaconType", Value: security, Type: "xsd:string"},
	}

	if security == "None" {
		// Open network — set BasicEncryptionModes ke None
		params = append(params, genieacs.TaskParam{
			Name: base + "BasicEncryptionModes", Value: "None", Type: "xsd:string",
		})
	} else if password != "" {
		// Kirim password sesuai mode
		params = append(params,
			genieacs.TaskParam{
				Name: base + "PreSharedKey.1.PreSharedKey", Value: password, Type: "xsd:string",
			},
			genieacs.TaskParam{
				Name: base + "IEEE11iEncryptionModes", Value: "AESEncryption", Type: "xsd:string",
			},
		)
	}

	return params
}

// SetWiFiSSID configures the WiFi SSID on a device.
// security: "None" (open), "WPA", "11i" (WPA2), "WPAand11i" (WPA/WPA2 Mixed). Default: "11i"
func (s *GenieACSService) SetWiFiSSID(ctx context.Context, tenantID, ontID, ssid, password, security string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return err
	}
	if ont == nil {
		return ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return fmt.Errorf("genieacs wifi: %w", err)
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}

	params := buildWiFiParams(ssid, password, security)
	_, err = s.client.SetParameterValues(ctx, device.ID, params)
	if err != nil {
		return err
	}
	if updateLastKnownWiFiFromSetRequest(ont, ssid, password, security) {
		if err := s.ontRepo.Update(ctx, ont); err != nil {
			return fmt.Errorf("genieacs wifi cache: %w", err)
		}
	}
	return nil
}

// ProvisionONT performs initial provisioning:
//  1. Tag perangkat dengan tenant ID (isolasi multi-tenant)
//  2. Kirim PPPoE username + password ke WAN interface (jika pelanggan terhubung)
//  3. Aktifkan koneksi WAN PPPoE
//  4. Refresh semua data TR-069 dari perangkat
func (s *GenieACSService) ProvisionONT(ctx context.Context, tenantID, ontID string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return err
	}
	if ont == nil {
		return ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return fmt.Errorf("genieacs provision: %w", err)
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}

	// Tag the device with the tenant ID for multi-tenant isolation
	if err := s.client.AddTag(ctx, device.ID, "tenant:"+tenantID); err != nil {
		return fmt.Errorf("genieacs tag: %w", err)
	}

	// Push PPPoE credentials to WAN interface if customer is linked
	if ont.Customer != nil && ont.Customer.PPPoEUsername != "" {
		params := []genieacs.TaskParam{
			{
				Name:  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
				Value: ont.Customer.PPPoEUsername,
				Type:  "xsd:string",
			},
			{
				Name:  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Password",
				Value: ont.Customer.PPPoEPassword,
				Type:  "xsd:string",
			},
			{
				Name:  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Enable",
				Value: true,
				Type:  "xsd:boolean",
			},
		}
		if _, err := s.client.SetParameterValues(ctx, device.ID, params); err != nil {
			return fmt.Errorf("genieacs provision pppoe: %w", err)
		}
	}

	// Auto-set WiFi SSID/password from customer data (if not already configured)
	defaultSSID, defaultPass := s.buildDefaultWiFi(ont)
	if defaultSSID != "" {
		if _, werr := s.client.SetParameterValues(ctx, device.ID, buildWiFiParams(defaultSSID, defaultPass, "11i")); werr == nil {
			updateLastKnownWiFiFromSetRequest(ont, defaultSSID, defaultPass, "11i")
		}
	}

	// Refresh all device data from TR-069
	if _, err = s.client.RefreshObject(ctx, device.ID, "InternetGatewayDevice."); err != nil {
		return fmt.Errorf("genieacs provision refresh: %w", err)
	}

	// Mark ONT as provisioned
	now := time.Now()
	ont.ProvisionedAt = &now
	if err := s.ontRepo.Update(ctx, ont); err != nil {
		return fmt.Errorf("genieacs provision save: %w", err)
	}

	// Notify customer via WA with WiFi credentials
	if defaultSSID != "" && ont.Customer != nil {
		go s.sendWiFiCredentialsWA(context.Background(), ont.TenantID, ont.Customer, defaultSSID, defaultPass)
	}

	return nil
}

// buildDefaultWiFi generates SSID from customer_code and a random 10-char password.
// Returns empty strings if customer is not linked.
func (s *GenieACSService) buildDefaultWiFi(ont *model.ONT) (ssid, password string) {
	if ont.Customer == nil || ont.Customer.CustomerCode == "" {
		return "", ""
	}
	// SSID = customer code (max 32 chars, WPA limit)
	ssid = ont.Customer.CustomerCode
	if len(ssid) > 32 {
		ssid = ssid[:32]
	}
	// If already has a cached password, keep it
	if ont.LastKnownWiFiPassword != nil && *ont.LastKnownWiFiPassword != "" {
		return ssid, *ont.LastKnownWiFiPassword
	}
	// Generate a random 10-char alphanumeric password
	const chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 10)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return ssid, string(b)
}

// sendWiFiCredentialsWA sends WiFi SSID and password to the customer via WhatsApp.
func (s *GenieACSService) sendWiFiCredentialsWA(ctx context.Context, tenantID string, customer *model.Customer, ssid, password string) {
	if s.waClient == nil || customer == nil || customer.Phone == "" {
		return
	}
	msg := fmt.Sprintf(
		"Selamat datang, %s!\n\nPerangkat WiFi Anda telah aktif:\n📶 Nama WiFi (SSID): *%s*\n🔑 Password: *%s*\n\nSilakan ubah password melalui portal pelanggan sesuai keinginan Anda.",
		customer.Name, ssid, password,
	)
	session := WASessionForTenant(ctx, tenantID, s.settingRepo)
	if _, err := s.waClient.SendMessage(ctx, session, customer.Phone, msg); err != nil {
		fmt.Printf("[genieacs] WA WiFi notify failed for %s: %v\n", customer.CustomerCode, err)
	}
}

// DeprovisionONT resets device configuration when an ONT is unlinked from a customer.
// Removes tenant tag and resets WiFi to prevent the new customer from using old credentials.
func (s *GenieACSService) DeprovisionONT(ctx context.Context, tenantID, ontID string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil || ont == nil {
		return nil // already gone, nothing to do
	}
	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil || device == nil {
		return nil
	}
	// Remove tenant tag
	_ = s.client.RemoveTag(ctx, device.ID, "tenant:"+tenantID)
	// Reset WiFi to a blank/random state so old credentials are gone
	newPass := "changeme" // placeholder — customer will set via portal
	_ = s.SetWiFiSSID(ctx, tenantID, ontID, ont.SerialNumber[:8], newPass, "11i")
	// Clear cached WiFi in DB
	ont.LastKnownWiFiSSID = nil
	ont.LastKnownWiFiPassword = nil
	ont.LastKnownWiFiSource = nil
	_ = s.ontRepo.Update(ctx, ont)
	return nil
}

// SetPPPoECredentials mendorong perubahan PPPoE username/password ke perangkat ONT.
// Dipanggil ketika admin mengubah kredensial pelanggan.
func (s *GenieACSService) SetPPPoECredentials(ctx context.Context, tenantID, ontID, username, password string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return err
	}
	if ont == nil {
		return ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return fmt.Errorf("genieacs set pppoe: %w", err)
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}

	params := []genieacs.TaskParam{
		{
			Name:  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
			Value: username,
			Type:  "xsd:string",
		},
		{
			Name:  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Password",
			Value: password,
			Type:  "xsd:string",
		},
	}
	_, err = s.client.SetParameterValues(ctx, device.ID, params)
	return err
}

// GetDeviceDiagnostics reads diagnostic data from a device.
func (s *GenieACSService) GetDeviceDiagnostics(ctx context.Context, tenantID, ontID string) (*DeviceInfo, error) {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return nil, err
	}
	if ont == nil {
		return nil, ErrONTNotFound
	}

	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("genieacs diagnostics: %w", err)
	}
	if device == nil {
		return nil, ErrGenieACSDeviceNotFound
	}

	info := &DeviceInfo{
		DeviceID:     device.ID,
		SerialNumber: ont.SerialNumber,
		Manufacturer: device.Manufacturer,
		ProductClass: device.ProductClass,
	}

	if sw, ok := device.Parameters["InternetGatewayDevice.DeviceInfo.SoftwareVersion"]; ok {
		if val, ok := sw.Value.(string); ok {
			info.SoftwareVersion = val
		}
	}
	if ut, ok := device.Parameters["InternetGatewayDevice.DeviceInfo.UpTime"]; ok {
		switch v := ut.Value.(type) {
		case float64:
			val := int64(v)
			info.Uptime = &val
		case int64:
			info.Uptime = &v
		}
	}
	if rx, ok := device.Parameters["InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.RXPower"]; ok {
		if val, ok := rx.Value.(float64); ok {
			info.RxPower = &val
		}
	}
	if tx, ok := device.Parameters["InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.TXPower"]; ok {
		if val, ok := tx.Value.(float64); ok {
			info.TxPower = &val
		}
	}

	return info, nil
}

// --- Serial-number-based methods (for /genieacs/ routes per roadmap) ---

// validateSerialOwnership verifies the serial number belongs to the given tenant.
func (s *GenieACSService) getOwnedONTBySerial(ctx context.Context, tenantID, serial string) (*model.ONT, error) {
	ont, err := s.ontRepo.FindBySerialNumber(ctx, tenantID, serial)
	if err != nil {
		return nil, err
	}
	if ont == nil {
		return nil, ErrONTNotFound
	}
	return ont, nil
}

func (s *GenieACSService) validateSerialOwnership(ctx context.Context, tenantID, serial string) error {
	_, err := s.getOwnedONTBySerial(ctx, tenantID, serial)
	return err
}

// GetDeviceBySerial retrieves a device from GenieACS by serial number.
func (s *GenieACSService) GetDeviceBySerial(ctx context.Context, tenantID, serial string) (*genieacs.Device, error) {
	if _, err := s.getOwnedONTBySerial(ctx, tenantID, serial); err != nil {
		return nil, err
	}
	device, err := s.client.FindDeviceBySerialNumber(ctx, serial)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, ErrGenieACSDeviceNotFound
	}
	return device, nil
}

// GetDeviceStatus returns status/diagnostics by serial number.
func (s *GenieACSService) GetDeviceStatus(ctx context.Context, tenantID, serial string) (*DeviceInfo, error) {
	ont, err := s.getOwnedONTBySerial(ctx, tenantID, serial)
	if err != nil {
		return nil, err
	}
	device, err := s.client.FindDeviceBySerialNumber(ctx, serial)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, ErrGenieACSDeviceNotFound
	}

	// Refresh semua parameter penting dari ONT secara realtime.
	// Jika ONT tidak bisa dijangkau (NAT/offline), task di-queue (202) dan
	// kita fallback ke data cache dari periodic inform terakhir.
	refreshCtx, refreshCancel := context.WithTimeout(ctx, 20*time.Second)
	defer refreshCancel()

	// Kirim refresh dengan path spesifik — lebih cepat, ONT tidak timeout
	dataRealtime := s.client.RefreshSpecificParams(refreshCtx, device.ID, []string{
		"InternetGatewayDevice.DeviceInfo.SoftwareVersion",
		"InternetGatewayDevice.DeviceInfo.UpTime",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.PreSharedKey",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_CT-COM_WPSKeyWord",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.PreSharedKey",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.X_CT-COM_WPSKeyWord",
		"InternetGatewayDevice.LANDevice.1.Hosts.HostNumberOfEntries",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.TXPower",
	}, 15000)

	// Jangan menambah queued task kedua ketika connection request pertama saja
	// sudah gagal. Saat ONT benar-benar merespons, refresh object host lalu
	// minta leaf value eksplisit agar firmware yang mendukung bisa mengisi detail host.
	if dataRealtime {
		hostsTask, _ := s.client.RefreshObject(refreshCtx, device.ID, "InternetGatewayDevice.LANDevice.1.Hosts.Host.")
		if hostsTask != nil && hostsTask.Completed {
			paths := buildHostRefreshPaths(getConnectedHostCount(device))
			paths = append(paths, buildAssociatedHostRefreshPaths()...)
			if len(paths) > 0 {
				s.client.RefreshSpecificParams(refreshCtx, device.ID, paths, 15000)
			}
		}
	}

	// Baca ulang device dari GenieACS dengan context baru
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	if refreshed, err2 := s.client.FindDeviceBySerialNumber(readCtx, serial); err2 == nil && refreshed != nil {
		device = refreshed
	}

	info := &DeviceInfo{
		DeviceID:       device.ID,
		SerialNumber:   serial,
		Manufacturer:   device.Manufacturer,
		ProductClass:   device.ProductClass,
		LastInform:     device.LastInform,
		DataRealtime:   dataRealtime,
		ConnectedHostsComplete: true,
		ConnectedHostsSource: "none",
		ConnectedHosts: []ConnectedHost{},
	}

	if sw := device.GetNestedValue("InternetGatewayDevice.DeviceInfo.SoftwareVersion"); sw != nil {
		if val, ok := sw.(string); ok {
			info.SoftwareVersion = val
		}
	}
	if ut := device.GetNestedValue("InternetGatewayDevice.DeviceInfo.UpTime"); ut != nil {
		switch v := ut.(type) {
		case float64:
			val := int64(v)
			info.Uptime = &val
		case int64:
			info.Uptime = &v
		}
	}

	// WiFi SSID & Security Mode dibaca dari GenieACS.
	// Password tidak diandalkan dari GenieACS karena firmware ONT tidak melaporkannya;
	// gunakan password terakhir yang pernah kita kirim dan simpan di database.
	const wlanBase = "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1."
	if ssid := device.GetNestedValue(wlanBase + "SSID"); ssid != nil {
		if val, ok := ssid.(string); ok {
			info.WiFiSSID = val
		}
	}
	if beacon := device.GetNestedValue(wlanBase + "BeaconType"); beacon != nil {
		if val, ok := beacon.(string); ok {
			info.WiFiSecurity = val
		}
	}
	if applyCachedWiFiPassword(ont, info) {
		// gunakan password terakhir yang pernah dikirim
	} else {
		info.WiFiPasswordSource = "unavailable"
	}

	// Deteksi mode security yang didukung ONT berdasarkan parameter yang ada
	info.WiFiSecurityModes = detectWiFiSecurityModes(device, wlanBase)

	// WAN IP + PPPoE username — coba semua kombinasi index
	for wanIdx := 1; wanIdx <= 4; wanIdx++ {
		for connIdx := 1; connIdx <= 4; connIdx++ {
			base := fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANPPPConnection.1", wanIdx, connIdx)
			if ip := device.GetNestedValue(base + ".ExternalIPAddress"); ip != nil {
				if val, ok := ip.(string); ok && val != "" {
					info.WANIP = val
				}
			}
			if user := device.GetNestedValue(base + ".Username"); user != nil {
				if val, ok := user.(string); ok && val != "" {
					info.PPPoEUsername = val
				}
			}
			if info.WANIP != "" {
				goto doneWAN
			}
		}
	}
doneWAN:

	// RX/TX optical power — coba semua index
	for wanIdx := 1; wanIdx <= 4; wanIdx++ {
		for connIdx := 1; connIdx <= 4; connIdx++ {
			base := fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANPONInterfaceConfig", wanIdx, connIdx)
			if rx := device.GetNestedValue(base + ".RXPower"); rx != nil {
				if val, ok := rx.(float64); ok {
					info.RxPower = &val
				}
			}
			if tx := device.GetNestedValue(base + ".TXPower"); tx != nil {
				if val, ok := tx.(float64); ok {
					info.TxPower = &val
				}
			}
			if info.RxPower != nil {
				goto donePower
			}
		}
	}
donePower:

	// Connected hosts — coba tabel Hosts.Host dulu, lalu merge dengan
	// AssociatedDevice (WiFi clients) jika tersedia.
	hostsFromTable := extractConnectedHosts(device)
	hostsFromAssociated := extractAssociatedHosts(device)
	info.ConnectedHosts = mergeConnectedHosts(hostsFromTable, hostsFromAssociated)
	info.ConnectedHostCount = len(info.ConnectedHosts)
	// Fallback ke HostNumberOfEntries jika detail host tidak ter-fetch (firmware limitation)
	if hostCount := getConnectedHostCount(device); hostCount > info.ConnectedHostCount {
		info.ConnectedHostCount = hostCount
	}
	switch {
	case len(hostsFromTable) > 0 && len(hostsFromAssociated) > 0:
		info.ConnectedHostsSource = "merged"
	case len(hostsFromTable) > 0:
		info.ConnectedHostsSource = "hosts"
	case len(hostsFromAssociated) > 0:
		info.ConnectedHostsSource = "associated_device"
	case info.ConnectedHostCount > 0:
		info.ConnectedHostsSource = "count_only"
	}
	if len(info.ConnectedHosts) > 0 {
		if updateLastKnownConnectedHostsFromReported(ont, info) {
			_ = s.ontRepo.Update(ctx, ont)
		}
	} else if info.ConnectedHostCount > 0 {
		applyCachedConnectedHosts(ont, info)
	}
	if strings.HasPrefix(info.ConnectedHostsSource, "cached_") {
		info.ConnectedHostsComplete = false
	} else {
		info.ConnectedHostsComplete = len(info.ConnectedHosts) == info.ConnectedHostCount
	}

	return info, nil
}

// detectWiFiSecurityModes mendeteksi mode security yang didukung ONT
// berdasarkan keberadaan parameter di data GenieACS (tanpa butuh _value).
// Mode yang dicek: None, WEP, WPA, WPA2 (11i), WPA/WPA2 Mixed (WPAand11i).
func detectWiFiSecurityModes(device *genieacs.Device, wlanBase string) []string {
	modes := []string{}

	// None / Open — selalu ada jika BeaconType bisa di-set
	if device.HasNestedParam(wlanBase + "BasicEncryptionModes") {
		modes = append(modes, "None")
	}

	// WEP — cek keberadaan WEPEncryptionLevel atau WEPKey
	if device.HasNestedParam(wlanBase+"WEPEncryptionLevel") || device.HasNestedParam(wlanBase+"WEPKey") {
		modes = append(modes, "WEP")
	}

	// WPA — cek keberadaan WPAEncryptionModes
	if device.HasNestedParam(wlanBase + "WPAEncryptionModes") {
		modes = append(modes, "WPA")
	}

	// WPA2 / 11i — cek keberadaan IEEE11iEncryptionModes
	if device.HasNestedParam(wlanBase + "IEEE11iEncryptionModes") {
		modes = append(modes, "11i")
	}

	// WPA/WPA2 Mixed — tersedia jika keduanya WPA dan 11i ada
	if device.HasNestedParam(wlanBase+"WPAEncryptionModes") && device.HasNestedParam(wlanBase+"IEEE11iEncryptionModes") {
		modes = append(modes, "WPAand11i")
	}

	return modes
}

// extractConnectedHosts membaca semua perangkat yang terhubung ke ONT dari
// InternetGatewayDevice.LANDevice.1.Hosts.Host.{N}.*
func extractConnectedHosts(device *genieacs.Device) []ConnectedHost {
	hosts := make([]ConnectedHost, 0)

	// Gunakan HostNumberOfEntries sebagai batas atas yang akurat
	maxHosts := getConnectedHostCount(device)
	if maxHosts == 0 || maxHosts > 64 {
		maxHosts = 64
	}

	for i := 1; i <= maxHosts; i++ {
		base := fmt.Sprintf("InternetGatewayDevice.LANDevice.1.Hosts.Host.%d", i)
		ip := genieStrVal(device.GetNestedValue(base + ".IPAddress"))
		mac := genieStrVal(device.GetNestedValue(base + ".MACAddress"))
		// Skip entry yang benar-benar kosong (nilai belum di-fetch firmware)
		if ip == "" && mac == "" {
			continue
		}
		active := false
		if v := device.GetNestedValue(base + ".Active"); v != nil {
			if b, ok := v.(bool); ok {
				active = b
			}
		}
		hosts = append(hosts, ConnectedHost{
			IPAddress:     ip,
			MACAddress:    mac,
			HostName:      genieStrVal(device.GetNestedValue(base + ".HostName")),
			InterfaceType: genieStrVal(device.GetNestedValue(base + ".InterfaceType")),
			Active:        active,
		})
	}
	return hosts
}

func extractAssociatedHosts(device *genieacs.Device) []ConnectedHost {
	hosts := make([]ConnectedHost, 0)
	for wlanIdx := 1; wlanIdx <= 2; wlanIdx++ {
		for assocIdx := 1; assocIdx <= 16; assocIdx++ {
			base := fmt.Sprintf("InternetGatewayDevice.LANDevice.1.WLANConfiguration.%d.AssociatedDevice.%d", wlanIdx, assocIdx)
			ip := genieStrVal(device.GetNestedValue(base + ".AssociatedDeviceIPAddress"))
			mac := genieStrVal(device.GetNestedValue(base + ".AssociatedDeviceMACAddress"))
			if ip == "" && mac == "" {
				continue
			}
			authState := strings.ToLower(genieStrVal(device.GetNestedValue(base + ".AssociatedDeviceAuthenticationState")))
			hosts = append(hosts, ConnectedHost{
				IPAddress:     ip,
				MACAddress:    mac,
				HostName:      "",
				InterfaceType: "802.11",
				Active:        authState == "" || authState == "true" || authState == "1" || authState == "authenticated" || authState == "associated",
			})
		}
	}
	return hosts
}

func mergeConnectedHosts(primary, secondary []ConnectedHost) []ConnectedHost {
	merged := make([]ConnectedHost, 0, len(primary)+len(secondary))
	seen := make(map[string]int)
	appendHost := func(host ConnectedHost) {
		key := connectedHostKey(host)
		if key == "" {
			merged = append(merged, host)
			return
		}
		if idx, ok := seen[key]; ok {
			if merged[idx].IPAddress == "" {
				merged[idx].IPAddress = host.IPAddress
			}
			if merged[idx].MACAddress == "" {
				merged[idx].MACAddress = host.MACAddress
			}
			if merged[idx].HostName == "" {
				merged[idx].HostName = host.HostName
			}
			if merged[idx].InterfaceType == "" {
				merged[idx].InterfaceType = host.InterfaceType
			}
			merged[idx].Active = merged[idx].Active || host.Active
			return
		}
		seen[key] = len(merged)
		merged = append(merged, host)
	}
	for _, host := range primary {
		appendHost(host)
	}
	for _, host := range secondary {
		appendHost(host)
	}
	return merged
}

func connectedHostKey(host ConnectedHost) string {
	if host.MACAddress != "" {
		return "mac:" + strings.ToLower(host.MACAddress)
	}
	if host.IPAddress != "" {
		return "ip:" + host.IPAddress
	}
	return ""
}

func getConnectedHostCount(device *genieacs.Device) int {
	if v := device.GetNestedValue("InternetGatewayDevice.LANDevice.1.Hosts.HostNumberOfEntries"); v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int64:
			return int(n)
		}
	}
	return 0
}

func buildHostRefreshPaths(hostCount int) []string {
	if hostCount <= 0 {
		return nil
	}
	if hostCount > 16 {
		hostCount = 16
	}
	paths := make([]string, 0, hostCount*5)
	props := []string{"IPAddress", "MACAddress", "HostName", "Active", "InterfaceType"}
	for i := 1; i <= hostCount; i++ {
		base := fmt.Sprintf("InternetGatewayDevice.LANDevice.1.Hosts.Host.%d", i)
		for _, prop := range props {
			paths = append(paths, base+"."+prop)
		}
	}
	return paths
}

func buildAssociatedHostRefreshPaths() []string {
	paths := make([]string, 0, 2*16*3)
	props := []string{"AssociatedDeviceIPAddress", "AssociatedDeviceMACAddress", "AssociatedDeviceAuthenticationState"}
	for wlanIdx := 1; wlanIdx <= 2; wlanIdx++ {
		for assocIdx := 1; assocIdx <= 16; assocIdx++ {
			base := fmt.Sprintf("InternetGatewayDevice.LANDevice.1.WLANConfiguration.%d.AssociatedDevice.%d", wlanIdx, assocIdx)
			for _, prop := range props {
				paths = append(paths, base+"."+prop)
			}
		}
	}
	return paths
}


func genieStrVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// RebootBySerial reboots a device by serial number.
func (s *GenieACSService) RebootBySerial(ctx context.Context, tenantID, serial string) error {
	ont, err := s.getOwnedONTBySerial(ctx, tenantID, serial)
	if err != nil {
		return err
	}
	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return err
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}
	_, err = s.client.Reboot(ctx, device.ID)
	return err
}

// SetWiFiBySerial sets WiFi SSID and password by serial number.
func (s *GenieACSService) SetWiFiBySerial(ctx context.Context, tenantID, serial, ssid, password, security string) error {
	ont, err := s.getOwnedONTBySerial(ctx, tenantID, serial)
	if err != nil {
		return err
	}
	device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
	if err != nil {
		return err
	}
	if device == nil {
		return ErrGenieACSDeviceNotFound
	}
	params := buildWiFiParams(ssid, password, security)
	_, err = s.client.SetParameterValues(ctx, device.ID, params)
	if err != nil {
		return err
	}
	if updateLastKnownWiFiFromSetRequest(ont, ssid, password, security) {
		if err := s.ontRepo.Update(ctx, ont); err != nil {
			return fmt.Errorf("genieacs wifi cache: %w", err)
		}
	}
	return nil
}

func updateLastKnownWiFiFromSetRequest(ont *model.ONT, ssid, password, security string) bool {
	if ont == nil {
		return false
	}
	changed := false
	security = normalizeWiFiSecurity(password, security)
	changed = setOptionalString(&ont.LastKnownWiFiSSID, ssid) || changed
	changed = setOptionalString(&ont.LastKnownWiFiSecurity, security) || changed
	if password != "" {
		changed = setOptionalString(&ont.LastKnownWiFiPassword, password) || changed
	} else if security == "None" {
		changed = setOptionalString(&ont.LastKnownWiFiPassword, "") || changed
	}
	changed = setOptionalString(&ont.LastKnownWiFiSource, "last_sent") || changed
	if changed {
		now := time.Now()
		setOptionalTime(&ont.LastKnownWiFiUpdatedAt, now)
	}
	return changed
}

func applyCachedWiFiPassword(ont *model.ONT, info *DeviceInfo) bool {
	if ont == nil || info == nil || ont.LastKnownWiFiPassword == nil {
		return false
	}
	if strings.TrimSpace(*ont.LastKnownWiFiPassword) == "" {
		return false
	}
	if info.WiFiSecurity == "None" {
		return false
	}
	if ont.LastKnownWiFiSSID != nil && info.WiFiSSID != "" && *ont.LastKnownWiFiSSID != info.WiFiSSID {
		return false
	}
	if ont.LastKnownWiFiSecurity != nil && info.WiFiSecurity != "" && *ont.LastKnownWiFiSecurity != "" && *ont.LastKnownWiFiSecurity != info.WiFiSecurity {
		return false
	}
	info.WiFiPassword = *ont.LastKnownWiFiPassword
	info.WiFiPasswordReported = false
	info.WiFiPasswordSource = lastKnownWiFiSourceLabel(ont.LastKnownWiFiSource)
	return true
}

func lastKnownWiFiSourceLabel(source *string) string {
	if source == nil {
		return "last_sent"
	}
	value := strings.TrimSpace(*source)
	if value == "" {
		return "last_sent"
	}
	switch value {
	case "set_request":
		return "last_sent"
	default:
		return value
	}
}

func updateLastKnownConnectedHostsFromReported(ont *model.ONT, info *DeviceInfo) bool {
	if ont == nil || info == nil || len(info.ConnectedHosts) == 0 {
		return false
	}
	payload, err := json.Marshal(info.ConnectedHosts)
	if err != nil {
		return false
	}
	changed := false
	changed = setOptionalString(&ont.LastKnownConnectedHosts, string(payload)) || changed
	changed = setOptionalInt(&ont.LastKnownConnectedHostCount, info.ConnectedHostCount) || changed
	if info.ConnectedHostsSource != "" {
		changed = setOptionalString(&ont.LastKnownConnectedHostsSource, info.ConnectedHostsSource) || changed
	}
	if changed {
		now := time.Now()
		setOptionalTime(&ont.LastKnownConnectedHostsUpdatedAt, now)
	}
	return changed
}

func applyCachedConnectedHosts(ont *model.ONT, info *DeviceInfo) bool {
	if ont == nil || info == nil || info.ConnectedHostCount <= 0 || ont.LastKnownConnectedHosts == nil {
		return false
	}
	raw := strings.TrimSpace(*ont.LastKnownConnectedHosts)
	if raw == "" {
		return false
	}
	var hosts []ConnectedHost
	if err := json.Unmarshal([]byte(raw), &hosts); err != nil || len(hosts) == 0 {
		return false
	}
	if info.ConnectedHostCount < len(hosts) {
		return false
	}
	info.ConnectedHosts = hosts
	if ont.LastKnownConnectedHostsSource != nil && *ont.LastKnownConnectedHostsSource != "" {
		info.ConnectedHostsSource = "cached_" + *ont.LastKnownConnectedHostsSource
	} else {
		info.ConnectedHostsSource = "cached_last_known"
	}
	return true
}

func setOptionalString(target **string, value string) bool {
	if *target != nil && **target == value {
		return false
	}
	copy := value
	*target = &copy
	return true
}

func setOptionalInt(target **int, value int) bool {
	if *target != nil && **target == value {
		return false
	}
	copy := value
	*target = &copy
	return true
}

func setOptionalTime(target **time.Time, value time.Time) bool {
	if *target != nil && (**target).Equal(value) {
		return false
	}
	copy := value
	*target = &copy
	return true
}

// GetONTDashboard returns an aggregated view of all ONTs for a tenant with their signal status.
func (s *GenieACSService) GetONTDashboard(ctx context.Context, tenantID string) ([]ONTDashboardItem, error) {
	onts, _, err := s.ontRepo.List(ctx, tenantID, repository.ONTFilter{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}

	items := make([]ONTDashboardItem, 0, len(onts))
	for _, ont := range onts {
		item := ONTDashboardItem{
			ONTID:        ont.ID,
			SerialNumber: ont.SerialNumber,
			Status:       ont.Status,
			RxPower:      ont.RxPower,
			TxPower:      ont.TxPower,
		}
		if ont.Customer != nil {
			item.CustomerName = ont.Customer.Name
		}
		items = append(items, item)
	}
	return items, nil
}

// BulkProvisionResult menyimpan hasil bulk provisioning.
type BulkProvisionResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// BulkProvision mencoba provisioning semua ONT aktif milik tenant yang belum terprovisi.
// Berguna untuk migrasi pelanggan lama ke GenieACS.
func (s *GenieACSService) BulkProvision(ctx context.Context, tenantID string) (*BulkProvisionResult, error) {
	onts, _, err := s.ontRepo.List(ctx, tenantID, repository.ONTFilter{
		Status:  "active",
		Page:    1,
		PerPage: 1000,
	})
	if err != nil {
		return nil, err
	}

	result := &BulkProvisionResult{Total: len(onts)}
	for _, ont := range onts {
		// Lewati yang sudah terprovisi
		if ont.ProvisionedAt != nil {
			result.Skipped++
			continue
		}
		if err := s.ProvisionONT(ctx, tenantID, ont.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %v", ont.SerialNumber, ont.ID, err))
			result.Skipped++
		} else {
			result.Success++
		}
	}
	return result, nil
}

// DiscoveryResult menyimpan hasil auto-discovery perangkat dari GenieACS.
type DiscoveryResult struct {
	TotalInGenieACS int      `json:"total_in_genieacs"`
	NewDiscovered   int      `json:"new_discovered"`
	AlreadyKnown    int      `json:"already_known"`
	Errors          []string `json:"errors,omitempty"`
}

// DiscoverONTs membandingkan semua perangkat di GenieACS dengan database lokal.
// Perangkat yang ada di GenieACS tetapi belum terdaftar di DB akan otomatis dibuat
// dengan status "discovered" (belum terhubung ke pelanggan).
// Admin kemudian bisa assign perangkat tersebut ke pelanggan melalui UI.
//
// Multi-tenant safety: serial number dicek secara GLOBAL (lintas semua tenant) sehingga
// device yang sama tidak pernah masuk dua kali, terlepas tenant mana yang memicu scan.
func (s *GenieACSService) DiscoverONTs(ctx context.Context, tenantID string) (*DiscoveryResult, error) {
	// Load semua serial number dari SEMUA tenant (O(1) lookup, mencegah duplikat lintas tenant)
	knownSerials, err := s.ontRepo.FindAllSerialNumbersGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover: load known serials: %w", err)
	}

	// Ambil semua perangkat dari GenieACS
	devices, err := s.client.ListDevices(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("discover: list genieacs devices: %w", err)
	}

	result := &DiscoveryResult{TotalInGenieACS: len(devices)}

	for _, dev := range devices {
		if dev.SerialNumber == "" {
			continue
		}

		// Skip device internal GenieACS (DISCOVERYSERVICE, probe, dll)
		if isGenieACSInternalDevice(dev) {
			continue
		}

		// Skip jika serial sudah ada di DB manapun (global check mencegah duplikat)
		if knownSerials[dev.SerialNumber] {
			result.AlreadyKnown++
			continue
		}

		// Buat ONT baru dengan status "discovered" — belum ada pelanggan
		ont := &model.ONT{
			TenantID:     tenantID,
			SerialNumber: dev.SerialNumber,
			Status:       "discovered",
		}
		if dev.Manufacturer != "" {
			ont.Vendor = &dev.Manufacturer
		}
		if dev.ProductClass != "" {
			ont.Model = &dev.ProductClass
		}
		if dev.LastInform != "" {
			if t, err := time.Parse(time.RFC3339Nano, dev.LastInform); err == nil {
				ont.LastOnlineAt = &t
			}
		}
		if err := s.ontRepo.Create(ctx, ont); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", dev.SerialNumber, err))
			continue
		}

		result.NewDiscovered++
		knownSerials[dev.SerialNumber] = true // hindari duplikat dalam satu run
	}

	return result, nil
}

// isGenieACSInternalDevice returns true for GenieACS internal/probe devices that are not real ONTs.
func isGenieACSInternalDevice(dev genieacs.Device) bool {
	// GenieACS CWMP discovery service creates fake entries
	if dev.Manufacturer == "DISCOVERYSERVICE" || dev.ProductClass == "DISCOVERYSERVICE" {
		return true
	}
	// Common probe/test serials
	if dev.SerialNumber == "000000000001" || dev.SerialNumber == "probe" {
		return true
	}
	// Empty manufacturer with random-looking short serial (not a real ONT serial)
	if dev.Manufacturer == "" && dev.ProductClass == "" && len(dev.SerialNumber) < 8 {
		return true
	}
	return false
}

// AutoMatchResult menyimpan hasil auto-match ONT ke pelanggan.
type AutoMatchResult struct {
	TotalUnlinked int      `json:"total_unlinked"`
	Matched       int      `json:"matched"`
	Unmatched     int      `json:"unmatched"`
	Errors        []string `json:"errors,omitempty"`
}

// findPPPoEUsername mencari PPPoE username di dalam data TR-069 perangkat.
// Index WANDevice (1–4) dan WANConnectionDevice (1–4) dicoba semua kombinasi
// karena nilainya bervariasi antar-perangkat/konfigurasi ISP.
func findPPPoEUsername(device *genieacs.Device) string {
	for wanIdx := 1; wanIdx <= 4; wanIdx++ {
		for connIdx := 1; connIdx <= 4; connIdx++ {
			path := fmt.Sprintf(
				"InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANPPPConnection.1.Username",
				wanIdx, connIdx,
			)
			val := device.GetNestedValue(path)
			if val == nil {
				continue
			}
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// findExternalIP membaca ExternalIPAddress dari data TR-069 perangkat.
// Digunakan sebagai fallback pencocokan ONT ke pelanggan via RADIUS session aktif.
func findExternalIP(device *genieacs.Device) string {
	for wanIdx := 1; wanIdx <= 4; wanIdx++ {
		for connIdx := 1; connIdx <= 4; connIdx++ {
			for _, connType := range []string{"WANPPPConnection", "WANIPConnection"} {
				path := fmt.Sprintf(
					"InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.%s.1.ExternalIPAddress",
					wanIdx, connIdx, connType,
				)
				val := device.GetNestedValue(path)
				if val == nil {
					continue
				}
				if ip, ok := val.(string); ok && ip != "" && ip != "0.0.0.0" {
					return ip
				}
			}
		}
	}
	return ""
}

// AutoMatchONTs mencoba menghubungkan ONT yang belum ter-assign ke pelanggan
// dengan cara membaca PPPoE username dari parameter TR-069 perangkat di GenieACS,
// lalu mencocokkan dengan field pppoe_username di tabel customers (lintas semua tenant).
//
// Dipanggil setelah DiscoverONTs atau secara periodik oleh worker (sekali — bukan per-tenant).
// Parameter tenantID diabaikan; search dilakukan secara global sehingga ONT yang terdaftar
// di tenant mana pun bisa dicocokkan dengan pelanggan di tenant mana pun.
func (s *GenieACSService) AutoMatchONTs(ctx context.Context, _ string) (*AutoMatchResult, error) {
	// Ambil semua ONT yang belum ada customer_id (lintas semua tenant)
	unlinked, err := s.ontRepo.FindUnlinkedGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("auto-match: find unlinked: %w", err)
	}

	result := &AutoMatchResult{TotalUnlinked: len(unlinked)}
	if len(unlinked) == 0 {
		return result, nil
	}

	for _, ont := range unlinked {
		// Cari perangkat di GenieACS berdasarkan serial number
		device, err := s.client.FindDeviceBySerialNumber(ctx, ont.SerialNumber)
		if err != nil || device == nil {
			result.Unmatched++
			continue
		}

		// Baca PPPoE username dari parameter TR-069 perangkat.
		// Index WANDevice dan WANConnectionDevice bisa bervariasi (1, 2, 3...) tergantung
		// konfigurasi ONT — coba semua kombinasi hingga menemukan username yang valid.
		pppoeUsername := findPPPoEUsername(device)

		// Jika Username belum ter-cache di GenieACS, kirim task refresh (non-blocking).
		// Pada run AutoMatch berikutnya (~5 menit) parameter sudah tersedia.
		if pppoeUsername == "" {
			refreshPaths := make([]string, 0, 8)
			for wanIdx := 1; wanIdx <= 2; wanIdx++ {
				for connIdx := 1; connIdx <= 2; connIdx++ {
					refreshPaths = append(refreshPaths,
						fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d.WANPPPConnection.1.Username", wanIdx, connIdx),
					)
				}
			}
			// timeout=0: task di-queue, tidak menunggu respon ONT
			_, _ = s.client.RefreshObjects(ctx, device.ID, refreshPaths, 0)
		}

		var customer *model.Customer
		if pppoeUsername != "" {
			// Cari pelanggan dengan PPPoE username ini — lintas semua tenant (global)
			customer, err = s.customerRepo.FindByPPPoEUsernameGlobal(ctx, pppoeUsername)
			if err != nil {
				customer = nil
			}
		}

		// Fallback: cocokkan via ExternalIPAddress → active RADIUS session
		if customer == nil {
			externalIP := findExternalIP(device)
			if externalIP != "" {
				customer, _ = s.customerRepo.FindByActiveSessionIP(ctx, externalIP)
			}
		}

		if customer == nil {
			result.Unmatched++
			continue
		}

		// Pastikan pelanggan belum punya ONT lain
		existing, _ := s.ontRepo.FindByCustomerID(ctx, customer.TenantID, customer.ID)
		if existing != nil {
			result.Unmatched++
			continue
		}

		// Link ONT ke pelanggan; tenant_id disesuaikan dengan tenant pelanggan
		if err := s.ontRepo.LinkToCustomer(ctx, ont.ID, customer.ID, customer.TenantID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: link failed: %v", ont.SerialNumber, err))
			continue
		}

		result.Matched++
	}

	return result, nil
}
