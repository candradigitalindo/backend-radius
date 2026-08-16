package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/vpn"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrRouterNotFound   = errors.New("Router tidak ditemukan")
	ErrInvalidHeartbeat = errors.New("Token heartbeat tidak valid")
)

const (
	// Convention defaults surfaced to the MikroTik config UI so the frontend
	// never hardcodes them. CoAPort default applies only when a router has none.
	defaultCoAPort           = 3799
	defaultWGRouterPort      = "13231"   // router-side WireGuard listen port
	defaultHeartbeatInterval = "00:05:00" // RouterOS scheduler interval (hh:mm:ss)
)

type RouterService struct {
	routerRepo   repository.RouterRepository
	sessionRepo  repository.SessionRepository
	ifaceRepo    repository.RouterIfaceRepository
	snmpService  *SNMPService
	vpnManager   *vpn.Manager
	appURL       string
	radiusSecret string
	legacyVPN    config.LegacyVPNConfig
}

func NewRouterService(routerRepo repository.RouterRepository, sessionRepo repository.SessionRepository, vpnMgr *vpn.Manager) *RouterService {
	return &RouterService{routerRepo: routerRepo, sessionRepo: sessionRepo, vpnManager: vpnMgr}
}

func (s *RouterService) WithAppConfig(appURL, radiusSecret string) *RouterService {
	s.appURL = appURL
	s.radiusSecret = radiusSecret
	return s
}

func (s *RouterService) WithSNMP(snmp *SNMPService) *RouterService {
	s.snmpService = snmp
	return s
}

func (s *RouterService) WithIfaceRepo(repo repository.RouterIfaceRepository) *RouterService {
	s.ifaceRepo = repo
	return s
}

func (s *RouterService) WithLegacyVPN(cfg config.LegacyVPNConfig) *RouterService {
	s.legacyVPN = cfg
	return s
}

// IfaceStatInput is one interface's raw counters pushed by the router.
type IfaceStatInput struct {
	Name    string `json:"name"`
	RxBytes int64  `json:"rx_bytes"`
	TxBytes int64  `json:"tx_bytes"`
}

// RecordInterfaceStats stores interface counters pushed by a router (token-auth,
// same token as heartbeat). Works without any VPN — the router posts outbound.
func (s *RouterService) RecordInterfaceStats(ctx context.Context, token string, ifaces []IfaceStatInput) error {
	if s.ifaceRepo == nil {
		return errors.New("interface monitoring tidak dikonfigurasi")
	}
	router, err := s.routerRepo.FindByHeartbeatToken(ctx, token)
	if err != nil {
		return err
	}
	if router == nil {
		return ErrInvalidHeartbeat
	}
	for _, in := range ifaces {
		if in.Name == "" {
			continue
		}
		if err := s.ifaceRepo.RecordSample(ctx, router.ID, router.TenantID, in.Name, in.RxBytes, in.TxBytes); err != nil {
			log.Printf("[iface-stats] record %s/%s: %v", router.ID, in.Name, err)
		}
	}
	// Housekeeping: keep ~30 minutes of samples.
	_ = s.ifaceRepo.Prune(ctx, 30)
	return nil
}

// InterfaceStats is the response for the monitoring chart (pushed data, no SNMP).
type InterfaceStats struct {
	Interfaces []string                  `json:"interfaces"`
	Samples    []repository.IfaceSample  `json:"samples"`
}

// GetInterfaceStats returns recent pushed interface throughput for a router.
func (s *RouterService) GetInterfaceStats(ctx context.Context, tenantID, routerID string, minutes int) (*InterfaceStats, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	if s.ifaceRepo == nil {
		return &InterfaceStats{}, nil
	}
	if minutes <= 0 {
		minutes = 10
	}
	names, err := s.ifaceRepo.ListNames(ctx, routerID, minutes)
	if err != nil {
		return nil, err
	}
	samples, err := s.ifaceRepo.ListSamples(ctx, routerID, minutes)
	if err != nil {
		return nil, err
	}
	return &InterfaceStats{Interfaces: names, Samples: samples}, nil
}

func (s *RouterService) GetServerPublicIP() string {
	if s.vpnManager == nil || !s.vpnManager.IsAvailable() {
		return ""
	}
	info, err := s.vpnManager.GetServerInfo()
	if err != nil {
		return ""
	}
	return info.PublicIP
}

func (s *RouterService) GetServerPublicKey() string {
	if s.vpnManager == nil || !s.vpnManager.IsAvailable() {
		return ""
	}
	info, err := s.vpnManager.GetServerInfo()
	if err != nil {
		return ""
	}
	return info.PublicKey
}

func (s *RouterService) GetVPNStatus(vpnIP string) *vpn.PeerStatus {
	if s.vpnManager == nil || !s.vpnManager.IsAvailable() || vpnIP == "" {
		return &vpn.PeerStatus{}
	}
	status, err := s.vpnManager.GetPeerStatus(vpnIP)
	if err != nil {
		return &vpn.PeerStatus{}
	}
	return status
}

type CreateRouterInput struct {
	TenantID      string
	Name          string
	RouterType    string
	SNMPCommunity string
}

type UpdateRouterInput struct {
	Name          string
	RouterType    string
	SNMPCommunity string
	IsActive      bool
}

// validRouterTypes are the router brands with a tailored configuration guide.
var validRouterTypes = map[string]bool{
	"mikrotik": true, "cisco": true, "huawei": true,
	"juniper": true, "vyos": true, "ruijie": true,
}

// normalizeRouterType returns a supported router type, defaulting to mikrotik.
func normalizeRouterType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if validRouterTypes[t] {
		return t
	}
	return "mikrotik"
}

func (s *RouterService) Create(ctx context.Context, input CreateRouterInput) (*model.Router, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	radiusSecret, err := generateToken(16)
	if err != nil {
		return nil, err
	}

	var vpnIP string
	if s.vpnManager != nil && s.vpnManager.IsAvailable() {
		usedIPs, _ := s.getUsedVPNIPs(ctx)
		allocated, err := s.vpnManager.AllocateIP(usedIPs)
		if err != nil {
			return nil, fmt.Errorf("allocate VPN IP: %w", err)
		}
		vpnIP = allocated
	}

	community := input.SNMPCommunity
	if community == "" {
		community = "public"
	}

	router := &model.Router{
		TenantID:       input.TenantID,
		Name:           strings.ToUpper(input.Name),
		RouterType:     normalizeRouterType(input.RouterType),
		VPNIP:          vpnIP,
		RADIUSSecret:   radiusSecret,
		CoAPort:        defaultCoAPort,
		HeartbeatToken: token,
		SNMPCommunity:  community,
		IsActive:       true,
	}

	if err := s.routerRepo.Create(ctx, router); err != nil {
		return nil, err
	}

	return s.routerRepo.FindByID(ctx, router.TenantID, router.ID)
}

func (s *RouterService) GetByID(ctx context.Context, tenantID, routerID string) (*model.Router, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	return router, nil
}

func (s *RouterService) Update(ctx context.Context, tenantID, routerID string, input UpdateRouterInput) (*model.Router, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}

	router.Name = strings.ToUpper(input.Name)
	if input.SNMPCommunity != "" {
		router.SNMPCommunity = input.SNMPCommunity
	}
	if input.RouterType != "" {
		router.RouterType = normalizeRouterType(input.RouterType)
	}
	router.IsActive = input.IsActive

	if err := s.routerRepo.Update(ctx, router); err != nil {
		return nil, err
	}

	return s.routerRepo.FindByID(ctx, tenantID, routerID)
}

func (s *RouterService) Delete(ctx context.Context, tenantID, routerID string) error {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return err
	}
	if router == nil {
		return ErrRouterNotFound
	}

	if s.vpnManager != nil && s.vpnManager.IsAvailable() && router.VPNPublicKey != "" {
		if err := s.vpnManager.RemovePeer(router.VPNPublicKey); err != nil {
			log.Printf("WireGuard remove peer failed for router %s: %v", routerID, err)
		}
	}

	return s.routerRepo.Delete(ctx, tenantID, routerID)
}

func (s *RouterService) List(ctx context.Context, tenantID string, filter repository.RouterFilter) ([]model.Router, int, error) {
	return s.routerRepo.List(ctx, tenantID, filter)
}

func (s *RouterService) RegenerateToken(ctx context.Context, tenantID, routerID string) (string, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return "", err
	}
	if router == nil {
		return "", ErrRouterNotFound
	}

	token, err := generateToken(32)
	if err != nil {
		return "", err
	}

	router.HeartbeatToken = token
	if err := s.routerRepo.Update(ctx, router); err != nil {
		return "", err
	}

	return token, nil
}

// Heartbeat processes a heartbeat from a router.
// senderIP is the IP address of the request sender — saved as nas_ip for Direct mode CoA routing.
func (s *RouterService) Heartbeat(ctx context.Context, token string, info repository.HeartbeatInfo, senderIP string) error {
	router, err := s.routerRepo.FindByHeartbeatToken(ctx, token)
	if err != nil {
		return err
	}
	if router == nil {
		return ErrInvalidHeartbeat
	}

	wasOffline, err := s.routerRepo.UpdateHeartbeat(ctx, router.ID, info)
	if err != nil {
		return err
	}

	// Always update nas_ip from heartbeat sender — WAN IP can change dynamically
	// (PPPoE reconnect, DHCP). This keeps CoA routing accurate.
	if senderIP != "" {
		_ = s.routerRepo.UpdateNASIP(ctx, router.ID, senderIP)
	}

	if wasOffline {
		endpoint := senderIP
		if s.vpnManager != nil && s.vpnManager.IsAvailable() && router.VPNIP != "" {
			if status, err := s.vpnManager.GetPeerStatus(router.VPNIP); err == nil && status.Endpoint != "" {
				endpoint = status.Endpoint
			}
		}
		_ = s.routerRepo.CreateConnectionLog(ctx, &model.RouterConnectionLog{
			RouterID:    router.ID,
			RouterName:  router.Name,
			Event:       "connected",
			VPNIP:       router.VPNIP,
			Endpoint:    endpoint,
			Identity:    info.Identity,
			RouterOSVer: info.RouterOSVer,
			BoardName:   info.BoardName,
			Uptime:      info.Uptime,
			CPULoad:     &info.CPULoad,
			FreeMemory:  &info.FreeMemory,
			TotalMemory: &info.TotalMemory,
		})
	}

	return nil
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *RouterService) ListSessions(ctx context.Context, tenantID, routerID string, activeOnly bool, page, perPage int) ([]model.RadiusSession, int, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, 0, err
	}
	if router == nil {
		return nil, 0, ErrRouterNotFound
	}
	return s.sessionRepo.ListByRouter(ctx, tenantID, routerID, activeOnly, page, perPage)
}

func (s *RouterService) GetTraffic(ctx context.Context, tenantID, routerID string) (*model.RouterTraffic, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	return s.sessionRepo.TrafficByRouter(ctx, tenantID, routerID)
}

func (s *RouterService) TestConnection(ctx context.Context, tenantID, routerID string) (bool, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return false, err
	}
	if router == nil {
		return false, ErrRouterNotFound
	}

	// Test the address the server actually uses to reach this router: VPN IP for
	// WireGuard routers, WAN IP (nas_ip) for Direct / IP Publik routers.
	addr := net.JoinHostPort(router.CoAAddress(), fmt.Sprintf("%d", router.CoAPort))
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return false, nil
	}
	conn.Close()
	return true, nil
}

func (s *RouterService) SyncSessions(ctx context.Context, tenantID, routerID string) (int, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return 0, err
	}
	if router == nil {
		return 0, ErrRouterNotFound
	}
	return s.sessionRepo.CleanStaleSessions(ctx, tenantID, routerID)
}

func (s *RouterService) ListVPNPeers(ctx context.Context, tenantID string) ([]vpn.PeerInfo, error) {
	if s.vpnManager == nil || !s.vpnManager.IsAvailable() {
		return nil, errors.New("WireGuard tidak tersedia")
	}

	allPeers, err := s.vpnManager.ListPeers()
	if err != nil {
		return nil, err
	}

	routers, _, err := s.routerRepo.List(ctx, tenantID, repository.RouterFilter{Page: 1, PerPage: 10000})
	if err != nil {
		return nil, err
	}

	allowedIPs := make(map[string]bool, len(routers))
	for _, r := range routers {
		if r.VPNIP != "" {
			allowedIPs[r.VPNIP+"/32"] = true
		}
	}

	filtered := make([]vpn.PeerInfo, 0)
	for _, p := range allPeers {
		if allowedIPs[p.AllowedIPs] {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

func (s *RouterService) ListConnectionLogs(ctx context.Context, tenantID, routerID string, page, perPage int) ([]model.RouterConnectionLog, int, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, 0, err
	}
	if router == nil {
		return nil, 0, ErrRouterNotFound
	}
	return s.routerRepo.ListConnectionLogs(ctx, routerID, page, perPage)
}

func (s *RouterService) LogDisconnections(ctx context.Context, routerIDs []string) {
	for _, rid := range routerIDs {
		router, err := s.routerRepo.FindByIDOnly(ctx, rid)
		if err != nil || router == nil {
			_ = s.routerRepo.CreateConnectionLog(ctx, &model.RouterConnectionLog{
				RouterID: rid,
				Event:    "disconnected",
			})
			continue
		}

		var duration string
		if router.LastSeenAt != nil {
			d := time.Since(*router.LastSeenAt)
			if d < time.Hour {
				duration = fmt.Sprintf("%dm", int(d.Minutes()))
			} else if d < 24*time.Hour {
				duration = fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
			} else {
				days := int(d.Hours()) / 24
				hours := int(d.Hours()) % 24
				duration = fmt.Sprintf("%dd%dh", days, hours)
			}
		}

		_ = s.routerRepo.CreateConnectionLog(ctx, &model.RouterConnectionLog{
			RouterID:    rid,
			RouterName:  router.Name,
			Event:       "disconnected",
			VPNIP:       router.VPNIP,
			Identity:    router.Identity,
			RouterOSVer: stringVal(router.RouterOSVer),
			BoardName:   stringVal(router.BoardName),
			Uptime:      stringVal(router.Uptime),
			CPULoad:     router.CPULoad,
			FreeMemory:  router.FreeMemory,
			TotalMemory: router.TotalMemory,
			Duration:    duration,
		})
	}
}

func (s *RouterService) GetRealtimeInterfaceTraffic(ctx context.Context, router *model.Router, interfaceNames []string) ([]InterfaceBandwidth, error) {
	if s.snmpService == nil {
		return nil, errors.New("layanan SNMP tidak tersedia")
	}
	return s.snmpService.GetRouterInterfaceBandwidth(ctx, router.VPNIP, router.SNMPCommunity, interfaceNames)
}

func (s *RouterService) GetInterfaces(ctx context.Context, tenantID, routerID string) ([]MonitorInterface, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	if s.snmpService == nil {
		return nil, errors.New("layanan SNMP tidak tersedia")
	}
	return s.snmpService.GetRouterInterfaces(ctx, router.VPNIP, router.SNMPCommunity)
}

func stringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type MikroTikConfig struct {
	RouterName        string `json:"router_name"`
	Mode              string `json:"mode"` // "wireguard" | "legacy" | "direct"
	VPNIP             string `json:"vpn_ip"`
	RADIUSSecret      string `json:"radius_secret"`
	RADIUSAddress     string `json:"radius_address"` // address the router must point RADIUS to (mode-aware)
	HeartbeatToken    string `json:"heartbeat_token"`
	HeartbeatURL      string `json:"heartbeat_url"`
	HeartbeatInterval string `json:"heartbeat_interval"`
	CoAPort           int    `json:"coa_port"`
	ServerPublicKey   string `json:"server_public_key"`
	ServerEndpoint    string `json:"server_endpoint"`
	ServerVPNIP       string `json:"server_vpn_ip"`
	VPNSubnet         string `json:"vpn_subnet"`
	VPNListenPort     string `json:"vpn_listen_port"`
	WGRouterPort      string `json:"wg_router_port"`
	RADIUSAuthPort    string `json:"radius_auth_port"`
	RADIUSAcctPort    string `json:"radius_acct_port"`
	Script            string `json:"script"`

	// Legacy VPN (L2TP/SSTP via accel-ppp) — for RouterOS 6 / routers behind NAT.
	LegacyVPNAvailable bool   `json:"legacy_vpn_available"`
	LegacyVPNUsername  string `json:"legacy_vpn_username,omitempty"`
	LegacyVPNPassword  string `json:"legacy_vpn_password,omitempty"`
	LegacyVPNIP        string `json:"legacy_vpn_ip,omitempty"`
	LegacyVPNGateway   string `json:"legacy_vpn_gw,omitempty"`
	LegacyL2TPPort     string `json:"legacy_l2tp_port,omitempty"`
	LegacySSTPPort     string `json:"legacy_sstp_port,omitempty"`
}

func (s *RouterService) GetMikroTikConfig(ctx context.Context, tenantID, routerID string) (*MikroTikConfig, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}

	config := &MikroTikConfig{
		RouterName:        router.Name,
		VPNIP:             router.VPNIP,
		RADIUSSecret:      router.RADIUSSecret,
		HeartbeatToken:    router.HeartbeatToken,
		HeartbeatInterval: defaultHeartbeatInterval,
		CoAPort:           router.CoAPort,
		WGRouterPort:      defaultWGRouterPort,
		RADIUSAuthPort:    "1812",
		RADIUSAcctPort:    "1813",
	}
	if config.CoAPort == 0 {
		config.CoAPort = defaultCoAPort
	}

	if s.appURL != "" {
		config.HeartbeatURL = s.appURL + "/api/v1/routers/heartbeat"
	}

	// Static VPN parameters (subnet, server VPN IP, listen port, public IP) are
	// configuration values — populate them regardless of whether the WireGuard
	// interface is currently up, so the UI never falls back to placeholders.
	if s.vpnManager != nil {
		static := s.vpnManager.StaticConfig()
		config.ServerEndpoint = static.PublicIP
		config.ServerVPNIP = static.ServerIP
		config.VPNSubnet = static.Subnet
		config.VPNListenPort = static.ListenPort

		// The server's WireGuard public key can only be read from the live interface.
		if s.vpnManager.IsAvailable() {
			if serverInfo, err := s.vpnManager.GetServerInfo(); err == nil {
				config.ServerPublicKey = serverInfo.PublicKey
			}
		}
	}

	// Legacy VPN (L2TP/SSTP) values — surfaced whenever the concentrator is
	// enabled so the guide can offer the mode; credentials only once provisioned.
	config.LegacyVPNAvailable = s.legacyVPN.SecretsFile != ""
	config.LegacyVPNGateway = s.legacyVPN.GatewayIP
	config.LegacyL2TPPort = s.legacyVPN.L2TPPort
	config.LegacySSTPPort = s.legacyVPN.SSTPPort
	if router.UsesLegacyVPN() {
		config.LegacyVPNUsername = router.ID
		config.LegacyVPNPassword = router.VPNPassword
		config.LegacyVPNIP = router.LegacyVPNIP
	}

	// Determine connection mode dynamically. WireGuard (registered public key)
	// wins; then provisioned L2TP/SSTP credentials; otherwise Direct / IP Publik
	// and RADIUS must point at the server's public address.
	switch {
	case router.UsesVPN():
		config.Mode = "wireguard"
		config.RADIUSAddress = config.ServerVPNIP
	case router.UsesLegacyVPN():
		config.Mode = "legacy"
		config.RADIUSAddress = s.legacyVPN.GatewayIP
	default:
		config.Mode = "direct"
		config.RADIUSAddress = config.ServerEndpoint
	}

	config.Script = s.buildMikroTikScript(config)
	return config, nil
}

func (s *RouterService) buildMikroTikScript(cfg *MikroTikConfig) string {
	var b strings.Builder
	step := 1
	b.WriteString("# ===== MikroTik Configuration Script =====\n")
	b.WriteString(fmt.Sprintf("# Router: %s\n", cfg.RouterName))
	if cfg.Mode == "wireguard" {
		b.WriteString("# Mode: WireGuard VPN\n")
	} else {
		b.WriteString("# Mode: IP Publik (Direct) — buka UDP 1812/1813 & CoA di firewall server\n")
	}
	b.WriteString("# Generated by Radius Server\n\n")

	// WireGuard tunnel — only in VPN mode. Direct routers reach the server over
	// the public internet, so the RADIUS address is the server's public IP.
	if cfg.Mode == "wireguard" {
		b.WriteString(fmt.Sprintf("# --- %d. WireGuard VPN ---\n", step))
		b.WriteString(fmt.Sprintf("/interface wireguard add name=wg0 listen-port=%s\n", cfg.WGRouterPort))
		b.WriteString(fmt.Sprintf("/interface wireguard peers add interface=wg0 public-key=\"%s\" endpoint-address=%s endpoint-port=%s allowed-address=%s persistent-keepalive=25 comment=\"VPN Server\"\n",
			cfg.ServerPublicKey, cfg.ServerEndpoint, cfg.VPNListenPort, cfg.VPNSubnet))
		b.WriteString(fmt.Sprintf("/ip address add address=%s/24 interface=wg0 network=%s\n\n",
			cfg.VPNIP, strings.Split(cfg.VPNSubnet, "/")[0]))
		step++
	}

	b.WriteString(fmt.Sprintf("# --- %d. RADIUS ---\n", step))
	b.WriteString(fmt.Sprintf("/radius add service=hotspot,login,ppp address=%s secret=\"%s\" authentication-port=%s accounting-port=%s\n",
		cfg.RADIUSAddress, cfg.RADIUSSecret, cfg.RADIUSAuthPort, cfg.RADIUSAcctPort))
	b.WriteString(fmt.Sprintf("/radius incoming set accept=yes port=%d\n", cfg.CoAPort))
	b.WriteString("/ppp aaa set use-radius=yes accounting=yes interim-update=5m\n")
	// One session per host: saat pelanggan redial dari MAC yang sama, router
	// langsung memutus sesi lama — mencegah interface ganda <pppoe-user-1>.
	b.WriteString("/interface pppoe-server server set [find] one-session-per-host=yes\n\n")
	step++

	b.WriteString(fmt.Sprintf("# --- %d. Heartbeat Monitoring ---\n", step))
	b.WriteString(fmt.Sprintf(`:global heartbeatToken "%s"
:global heartbeatURL "%s"

/system scheduler add name=radius-heartbeat interval=%s on-event={
  :local cpuLoad [/system resource get cpu-load]
  :local freeMem [/system resource get free-memory]
  :local totalMem [/system resource get total-memory]
  :local uptime [/system resource get uptime]
  :local boardName [/system resource get board-name]
  :local osVer [/system resource get version]
  :local identity [/system identity get name]
  :local payload "{\"token\":\"$heartbeatToken\",\"identity\":\"$identity\",\"cpu_load\":$cpuLoad,\"free_memory\":$freeMem,\"total_memory\":$totalMem,\"uptime\":\"$uptime\",\"board_name\":\"$boardName\",\"router_os_ver\":\"$osVer\"}"
  /tool fetch url=$heartbeatURL http-method=post http-header-field="Content-Type: application/json" http-data=$payload output=none
} comment="Radius Server Heartbeat"
`, cfg.HeartbeatToken, cfg.HeartbeatURL, cfg.HeartbeatInterval))
	return b.String()
}

func (s *RouterService) RegisterVPNKey(ctx context.Context, tenantID, routerID, publicKey string) (*model.Router, error) {
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	if s.vpnManager == nil || !s.vpnManager.IsAvailable() {
		return nil, errors.New("WireGuard tidak tersedia")
	}
	if router.VPNIP == "" {
		return nil, errors.New("router belum memiliki VPN IP")
	}
	if router.VPNPublicKey != "" && router.VPNPublicKey != publicKey {
		if err := s.vpnManager.RemovePeer(router.VPNPublicKey); err != nil {
			log.Printf("WireGuard remove old peer failed for router %s: %v", routerID, err)
		}
	}
	if err := s.vpnManager.AddPeer(publicKey, router.VPNIP); err != nil {
		return nil, fmt.Errorf("gagal menambahkan WireGuard peer: %w", err)
	}
	router.VPNPublicKey = publicKey
	router.NASIP = "" // clear nas_ip — router now uses WireGuard, prevent stale CoA misrouting
	if err := s.routerRepo.Update(ctx, router); err != nil {
		_ = s.vpnManager.RemovePeer(publicKey)
		return nil, err
	}
	return s.routerRepo.FindByID(ctx, tenantID, routerID)
}

func (s *RouterService) getUsedVPNIPs(ctx context.Context) ([]string, error) {
	return s.routerRepo.ListAllVPNIPs(ctx)
}

// EnableLegacyVPN provisions L2TP/SSTP tunnel credentials for a router that
// cannot use WireGuard (RouterOS 6 / behind NAT): a static tunnel IP from the
// legacy subnet plus a generated password (tunnel username = router ID).
// Idempotent — returns the existing credentials when already provisioned.
func (s *RouterService) EnableLegacyVPN(ctx context.Context, tenantID, routerID string) (*model.Router, error) {
	if s.legacyVPN.SecretsFile == "" {
		return nil, errors.New("VPN L2TP/SSTP belum diaktifkan di server")
	}
	router, err := s.routerRepo.FindByID(ctx, tenantID, routerID)
	if err != nil {
		return nil, err
	}
	if router == nil {
		return nil, ErrRouterNotFound
	}
	if router.LegacyVPNIP != "" && router.VPNPassword != "" {
		return router, nil
	}

	password, err := generateToken(12)
	if err != nil {
		return nil, err
	}
	used, err := s.routerRepo.ListAllLegacyVPNIPs(ctx)
	if err != nil {
		return nil, err
	}
	ip, err := allocateLegacyVPNIP(s.legacyVPN.Subnet, s.legacyVPN.GatewayIP, used)
	if err != nil {
		return nil, err
	}
	if err := s.routerRepo.SetLegacyVPN(ctx, tenantID, routerID, password, ip); err != nil {
		return nil, err
	}
	if err := s.SyncLegacyVPNSecrets(ctx); err != nil {
		log.Printf("legacy VPN: sync chap-secrets failed: %v", err)
	}
	return s.routerRepo.FindByID(ctx, tenantID, routerID)
}

// allocateLegacyVPNIP picks the lowest free host address in the legacy VPN
// subnet, skipping the network, broadcast and gateway addresses.
func allocateLegacyVPNIP(subnet, gateway string, used []string) (string, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("subnet VPN legacy tidak valid: %w", err)
	}
	taken := make(map[string]bool, len(used)+1)
	for _, u := range used {
		taken[u] = true
	}
	taken[gateway] = true

	base4 := ipNet.IP.To4()
	if base4 == nil {
		return "", errors.New("subnet VPN legacy harus IPv4")
	}
	base := binary.BigEndian.Uint32(base4)
	ones, bits := ipNet.Mask.Size()
	size := uint32(1) << (bits - ones)
	// Skip network (+0), gateway convention (+1 covered via taken) and broadcast.
	for off := uint32(1); off < size-1; off++ {
		candidate := make(net.IP, 4)
		binary.BigEndian.PutUint32(candidate, base+off)
		ip := candidate.String()
		if !taken[ip] {
			return ip, nil
		}
	}
	return "", errors.New("tidak ada IP VPN legacy yang tersisa")
}

// SyncLegacyVPNSecrets rewrites the accel-ppp chap-secrets file from the
// database. accel-ppp reads it per authentication (plus the vpn container
// watches its mtime and issues a reload), so a plain atomic write is enough.
// Format per line: <username> * <password> <static-ip>
func (s *RouterService) SyncLegacyVPNSecrets(ctx context.Context) error {
	if s.legacyVPN.SecretsFile == "" {
		return nil
	}
	accounts, err := s.routerRepo.ListLegacyVPNAccounts(ctx)
	if err != nil {
		return err
	}
	content := buildChapSecrets(accounts)
	tmp := s.legacyVPN.SecretsFile + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.legacyVPN.SecretsFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.legacyVPN.SecretsFile)
}

// buildChapSecrets renders the accel-ppp chap-secrets file body.
func buildChapSecrets(accounts []repository.LegacyVPNAccount) string {
	var b strings.Builder
	b.WriteString("# Generated by radius app — do not edit; entries come from the routers table.\n")
	for _, a := range accounts {
		b.WriteString(fmt.Sprintf("%s * %s %s\n", a.RouterID, a.Password, a.IP))
	}
	return b.String()
}
