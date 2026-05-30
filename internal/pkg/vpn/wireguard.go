package vpn

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Manager handles WireGuard VPN peer management using wgctrl-go (pure Go, no CLI dependency).
type Manager struct {
	iface      string
	subnet     string
	serverIP   string
	listenPort string
	publicIP   string
	mu         sync.Mutex
}

func NewManager(iface, subnet, serverIP, listenPort, publicIP string) *Manager {
	if publicIP == "" {
		publicIP = detectPublicIP()
	}
	return &Manager{
		iface:      iface,
		subnet:     subnet,
		serverIP:   serverIP,
		listenPort: listenPort,
		publicIP:   publicIP,
	}
}

// detectPublicIP auto-detects the VPS public IP address.
func detectPublicIP() string {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, url := range services {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if net.ParseIP(ip) != nil {
			log.Printf("[VPN] Auto-detected public IP: %s", ip)
			return ip
		}
	}
	log.Println("[VPN] WARNING: Could not auto-detect public IP, set VPN_PUBLIC_IP in .env")
	return ""
}

// RestorePeers re-adds all known peers to the WireGuard interface.
// Should be called on startup to restore peers lost after restart.
func (m *Manager) RestorePeers(peers []struct{ PublicKey, VPNIP string }) {
	if !m.IsAvailable() {
		log.Println("[VPN] WireGuard not available, skipping peer restore")
		return
	}
	for _, p := range peers {
		if err := m.AddPeer(p.PublicKey, p.VPNIP); err != nil {
			log.Printf("[VPN] Failed to restore peer %s (%s): %v", p.VPNIP, p.PublicKey[:8]+"...", err)
		}
	}
	log.Printf("[VPN] Peer restore complete: %d peers processed", len(peers))
}

// AddPeer adds a WireGuard peer with the given public key and allowed IP.
func (m *Manager) AddPeer(publicKey, allowedIP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if publicKey == "" || allowedIP == "" {
		return fmt.Errorf("public key and allowed IP are required")
	}

	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	// Ensure allowedIP has /32 suffix
	if !strings.Contains(allowedIP, "/") {
		allowedIP = allowedIP + "/32"
	}
	_, ipNet, err := net.ParseCIDR(allowedIP)
	if err != nil {
		return fmt.Errorf("parse allowed IP %s: %w", allowedIP, err)
	}

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	err = client.ConfigureDevice(m.iface, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:         key,
				ReplaceAllowedIPs: true,
				AllowedIPs:        []net.IPNet{*ipNet},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("configure device add peer: %w", err)
	}

	return nil
}

// AddPeerAllowedIP adds an additional allowed IP/subnet to an existing peer identified by its VPN IP.
func (m *Manager) AddPeerAllowedIP(vpnIP, extraSubnet string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	device, err := client.Device(m.iface)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	target := vpnIP + "/32"
	for _, p := range device.Peers {
		for _, ipNet := range p.AllowedIPs {
			if ipNet.String() == target {
				// Check if subnet already in allowed IPs
				for _, existing := range p.AllowedIPs {
					if existing.String() == extraSubnet {
						return nil // already exists
					}
				}

				// Parse new subnet
				_, newNet, err := net.ParseCIDR(extraSubnet)
				if err != nil {
					return fmt.Errorf("parse subnet %s: %w", extraSubnet, err)
				}

				// Build new allowed IPs list
				newAllowed := make([]net.IPNet, len(p.AllowedIPs)+1)
				copy(newAllowed, p.AllowedIPs)
				newAllowed[len(p.AllowedIPs)] = *newNet

				err = client.ConfigureDevice(m.iface, wgtypes.Config{
					Peers: []wgtypes.PeerConfig{
						{
							PublicKey:         p.PublicKey,
							ReplaceAllowedIPs: true,
							AllowedIPs:        newAllowed,
						},
					},
				})
				if err != nil {
					return fmt.Errorf("update peer allowed IPs: %w", err)
				}

				log.Printf("[VPN] Added %s to peer %s allowed IPs", extraSubnet, vpnIP)
				return nil
			}
		}
	}

	return fmt.Errorf("peer with VPN IP %s not found", vpnIP)
}

// RemovePeer removes a WireGuard peer by public key.
func (m *Manager) RemovePeer(publicKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if publicKey == "" {
		return nil
	}

	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	err = client.ConfigureDevice(m.iface, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: key,
				Remove:    true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("configure device remove peer: %w", err)
	}

	return nil
}

// ListPeers returns the current WireGuard peers.
func (m *Manager) ListPeers() ([]PeerInfo, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	device, err := client.Device(m.iface)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", m.iface, err)
	}

	peers := make([]PeerInfo, 0, len(device.Peers))
	for _, p := range device.Peers {
		var allowedIPs []string
		for _, ipNet := range p.AllowedIPs {
			allowedIPs = append(allowedIPs, ipNet.String())
		}

		endpoint := ""
		if p.Endpoint != nil {
			endpoint = p.Endpoint.String()
		}

		var handshake string
		if !p.LastHandshakeTime.IsZero() {
			handshake = p.LastHandshakeTime.Format(time.RFC3339)
		}

		peers = append(peers, PeerInfo{
			PublicKey:       p.PublicKey.String(),
			Endpoint:        endpoint,
			AllowedIPs:      strings.Join(allowedIPs, ","),
			LatestHandshake: handshake,
			TransferRx:      fmt.Sprintf("%d", p.ReceiveBytes),
			TransferTx:      fmt.Sprintf("%d", p.TransmitBytes),
		})
	}
	return peers, nil
}

// AllocateIP finds the next available IP in the VPN subnet.
func (m *Manager) AllocateIP(usedIPs []string) (string, error) {
	_, ipNet, err := net.ParseCIDR(m.subnet)
	if err != nil {
		return "", fmt.Errorf("parse subnet %s: %w", m.subnet, err)
	}

	used := make(map[string]bool)
	used[m.serverIP] = true
	for _, ip := range usedIPs {
		used[ip] = true
	}

	// Iterate through subnet to find available IP
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)

	for ip = nextIP(ip); ipNet.Contains(ip); ip = nextIP(ip) {
		candidate := ip.String()
		if !used[candidate] && !isBroadcast(ip, ipNet) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no available IP in subnet %s", m.subnet)
}

// IsAvailable checks if the WireGuard kernel module / userspace is accessible.
func (m *Manager) IsAvailable() bool {
	client, err := wgctrl.New()
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// GetServerInfo returns the WireGuard server's public key and connection details.
func (m *Manager) GetServerInfo() (*ServerInfo, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	device, err := client.Device(m.iface)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", m.iface, err)
	}

	return &ServerInfo{
		PublicKey:  device.PublicKey.String(),
		ListenPort: m.listenPort,
		ServerIP:   m.serverIP,
		Subnet:     m.subnet,
		Interface:  m.iface,
		PublicIP:   m.publicIP,
	}, nil
}

// ServerInfo holds WireGuard server configuration details.
type ServerInfo struct {
	PublicKey  string `json:"public_key"`
	ListenPort string `json:"listen_port"`
	ServerIP   string `json:"server_ip"`
	Subnet     string `json:"subnet"`
	Interface  string `json:"interface"`
	PublicIP   string `json:"public_ip"`
}

// PeerInfo holds information about a WireGuard peer.
type PeerInfo struct {
	PublicKey       string `json:"public_key"`
	Endpoint        string `json:"endpoint"`
	AllowedIPs      string `json:"allowed_ips"`
	LatestHandshake string `json:"latest_handshake,omitempty"`
	TransferRx      string `json:"transfer_rx,omitempty"`
	TransferTx      string `json:"transfer_tx,omitempty"`
}

// PeerStatus holds real-time VPN connection info for a single peer.
type PeerStatus struct {
	Connected       bool   `json:"connected"`
	Endpoint        string `json:"endpoint,omitempty"`
	LatestHandshake string `json:"latest_handshake,omitempty"`
	TransferRx      int64  `json:"transfer_rx"`
	TransferTx      int64  `json:"transfer_tx"`
}

// GetPeerStatus returns real-time VPN status for a peer identified by its allowed IP.
func (m *Manager) GetPeerStatus(vpnIP string) (*PeerStatus, error) {
	if !m.IsAvailable() {
		return &PeerStatus{}, nil
	}

	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("wgctrl client: %w", err)
	}
	defer client.Close()

	device, err := client.Device(m.iface)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", m.iface, err)
	}

	target := vpnIP + "/32"
	for _, p := range device.Peers {
		for _, ipNet := range p.AllowedIPs {
			if ipNet.String() == target {
				connected := !p.LastHandshakeTime.IsZero() && time.Since(p.LastHandshakeTime) < 3*time.Minute
				status := &PeerStatus{
					Connected:  connected,
					TransferRx: p.ReceiveBytes,
					TransferTx: p.TransmitBytes,
				}
				if p.Endpoint != nil {
					status.Endpoint = p.Endpoint.String()
				}
				if !p.LastHandshakeTime.IsZero() {
					status.LatestHandshake = p.LastHandshakeTime.Format(time.RFC3339)
				}
				return status, nil
			}
		}
	}

	return &PeerStatus{}, nil
}

func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] > 0 {
			break
		}
	}
	return next
}

func isBroadcast(ip net.IP, ipNet *net.IPNet) bool {
	for i := range ip {
		if ip[i] != ipNet.IP[i]|^ipNet.Mask[i] {
			return false
		}
	}
	return true
}
