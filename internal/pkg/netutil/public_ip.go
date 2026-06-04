package netutil

import (
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	cachedIP string
	cachedAt time.Time
	cacheMu  sync.Mutex
	cacheTTL = 10 * time.Minute
)

// GetPublicIP returns the server's public IP address.
// Result is cached for 10 minutes. Uses UDP trick first;
// if result is a private IP, falls back to querying ipify.org.
func GetPublicIP() string {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cachedIP != "" && time.Since(cachedAt) < cacheTTL {
		return cachedIP
	}

	ip := detectOutboundIP()
	if isPrivateIP(ip) {
		ip = fetchPublicIPExternal()
	}

	if ip != "" {
		cachedIP = ip
		cachedAt = time.Now()
	}
	return ip
}

// detectOutboundIP uses a UDP dial to determine the local outbound IP.
func detectOutboundIP() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 3*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

// fetchPublicIPExternal queries ipify.org to get the public IP.
func fetchPublicIPExternal() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// isPrivateIP checks if an IP string is a private/loopback address.
func isPrivateIP(ipStr string) bool {
	if ipStr == "" {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}
	for _, cidr := range privateRanges {
		_, block, _ := net.ParseCIDR(cidr)
		if block != nil && block.Contains(ip) {
			return true
		}
	}
	return false
}
