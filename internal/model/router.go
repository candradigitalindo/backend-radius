package model

import "time"

type Router struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Name           string     `json:"name"`
	RouterType     string     `json:"router_type"` // mikrotik | cisco | huawei | juniper | vyos | ruijie
	Identity       string     `json:"identity"`
	VPNIP          string     `json:"vpn_ip"`
	VPNPublicKey   string     `json:"vpn_public_key"`
	RADIUSSecret   string     `json:"radius_secret"`
	CoAPort        int        `json:"coa_port"`
	HeartbeatToken string     `json:"heartbeat_token"`
	IsOnline       bool       `json:"is_online"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	RouterOSVer    *string    `json:"router_os_ver,omitempty"`
	BoardName      *string    `json:"board_name,omitempty"`
	Uptime         *string    `json:"uptime,omitempty"`
	CPULoad        *int       `json:"cpu_load"`
	FreeMemory     *int64     `json:"free_memory"`
	TotalMemory    *int64     `json:"total_memory"`
	SNMPCommunity  string     `json:"snmp_community"`
	NASIP          string     `json:"nas_ip"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RouterConnectionLog struct {
	ID          string    `json:"id"`
	RouterID    string    `json:"router_id"`
	RouterName  string    `json:"router_name"`
	Event       string    `json:"event"`
	VPNIP       string    `json:"vpn_ip"`
	Endpoint    string    `json:"endpoint"`
	Identity    string    `json:"identity,omitempty"`
	RouterOSVer string    `json:"router_os_ver,omitempty"`
	BoardName   string    `json:"board_name,omitempty"`
	Uptime      string    `json:"uptime,omitempty"`
	CPULoad     *int      `json:"cpu_load,omitempty"`
	FreeMemory  *int64    `json:"free_memory,omitempty"`
	TotalMemory *int64    `json:"total_memory,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
