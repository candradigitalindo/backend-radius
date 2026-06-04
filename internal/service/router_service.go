package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/vpn"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrRouterNotFound   = errors.New("Router tidak ditemukan")
	ErrInvalidHeartbeat = errors.New("Token heartbeat tidak valid")
)

type RouterService struct {
	routerRepo   repository.RouterRepository
	sessionRepo  repository.SessionRepository
	snmpService  *SNMPService
	vpnManager   *vpn.Manager
	appURL       string
	radiusSecret string
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
	SNMPCommunity string
}

type UpdateRouterInput struct {
	Name          string
	SNMPCommunity string
	IsActive      bool
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
		VPNIP:          vpnIP,
		RADIUSSecret:   radiusSecret,
		CoAPort:        3799,
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

	addr := net.JoinHostPort(router.VPNIP, fmt.Sprintf("%d", router.CoAPort))
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
	RouterName      string `json:"router_name"`
	VPNIP           string `json:"vpn_ip"`
	RADIUSSecret    string `json:"radius_secret"`
	HeartbeatToken  string `json:"heartbeat_token"`
	HeartbeatURL    string `json:"heartbeat_url"`
	CoAPort         int    `json:"coa_port"`
	ServerPublicKey string `json:"server_public_key"`
	ServerEndpoint  string `json:"server_endpoint"`
	ServerVPNIP     string `json:"server_vpn_ip"`
	VPNSubnet       string `json:"vpn_subnet"`
	VPNListenPort   string `json:"vpn_listen_port"`
	RADIUSAuthPort  string `json:"radius_auth_port"`
	RADIUSAcctPort  string `json:"radius_acct_port"`
	Script          string `json:"script"`
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
		RouterName:     router.Name,
		VPNIP:          router.VPNIP,
		RADIUSSecret:   router.RADIUSSecret,
		HeartbeatToken: router.HeartbeatToken,
		CoAPort:        router.CoAPort,
		RADIUSAuthPort: "1812",
		RADIUSAcctPort: "1813",
	}

	if s.appURL != "" {
		config.HeartbeatURL = s.appURL + "/api/v1/routers/heartbeat"
	}

	if s.vpnManager != nil && s.vpnManager.IsAvailable() {
		serverInfo, err := s.vpnManager.GetServerInfo()
		if err == nil {
			config.ServerPublicKey = serverInfo.PublicKey
			config.ServerEndpoint = serverInfo.PublicIP
			config.ServerVPNIP = serverInfo.ServerIP
			config.VPNSubnet = serverInfo.Subnet
			config.VPNListenPort = serverInfo.ListenPort
		}
	}

	config.Script = s.buildMikroTikScript(config)
	return config, nil
}

func (s *RouterService) buildMikroTikScript(cfg *MikroTikConfig) string {
	var b strings.Builder
	b.WriteString("# ===== MikroTik Configuration Script =====\n")
	b.WriteString(fmt.Sprintf("# Router: %s\n", cfg.RouterName))
	b.WriteString("# Generated by Radius Server\n\n")
	b.WriteString("# --- 1. WireGuard VPN ---\n")
	b.WriteString("/interface wireguard add name=wg0 listen-port=13231\n")
	b.WriteString(fmt.Sprintf("/interface wireguard peers add interface=wg0 public-key=\"%s\" endpoint-address=%s endpoint-port=%s allowed-address=%s persistent-keepalive=25 comment=\"VPN Server\"\n",
		cfg.ServerPublicKey, cfg.ServerEndpoint, cfg.VPNListenPort, cfg.VPNSubnet))
	b.WriteString(fmt.Sprintf("/ip address add address=%s/24 interface=wg0 network=%s\n\n",
		cfg.VPNIP, strings.Split(cfg.VPNSubnet, "/")[0]))
	b.WriteString("# --- 2. RADIUS ---\n")
	b.WriteString(fmt.Sprintf("/radius add service=hotspot,login,ppp address=%s secret=\"%s\" authentication-port=%s accounting-port=%s\n",
		cfg.ServerVPNIP, cfg.RADIUSSecret, cfg.RADIUSAuthPort, cfg.RADIUSAcctPort))
	b.WriteString("/radius incoming set accept=yes port=3799\n\n")
	b.WriteString("# --- 3. Heartbeat Monitoring ---\n")
	b.WriteString(fmt.Sprintf(`:global heartbeatToken "%s"
:global heartbeatURL "%s"

/system scheduler add name=radius-heartbeat interval=5m on-event={
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
`, cfg.HeartbeatToken, cfg.HeartbeatURL))
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
