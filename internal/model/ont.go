package model

import "time"

type ONTConnectedHost struct {
	IPAddress     string `json:"ip_address"`
	MACAddress    string `json:"mac_address"`
	HostName      string `json:"hostname"`
	InterfaceType string `json:"interface_type"`
	Active        bool   `json:"active"`
}

type ONTAvailableActions struct {
	CanRestart    bool `json:"can_restart"`
	CanManageWiFi bool `json:"can_manage_wifi"`
}

type ONTStatusDetail struct {
	DeviceID               string             `json:"device_id,omitempty"`
	SerialNumber           string             `json:"serial_number,omitempty"`
	Manufacturer           string             `json:"manufacturer,omitempty"`
	ProductClass           string             `json:"product_class,omitempty"`
	SoftwareVersion        string             `json:"software_version,omitempty"`
	Uptime                 *int64             `json:"uptime,omitempty"`
	RxPower                *float64           `json:"rx_power,omitempty"`
	TxPower                *float64           `json:"tx_power,omitempty"`
	WiFiSSID               string             `json:"wifi_ssid,omitempty"`
	WiFiPassword           string             `json:"wifi_password"`
	WiFiPasswordReported   bool               `json:"wifi_password_reported"`
	WiFiPasswordSource     string             `json:"wifi_password_source,omitempty"`
	WiFiSecurity           string             `json:"wifi_security,omitempty"`
	WiFiSecurityModes      []string           `json:"wifi_security_modes,omitempty"`
	WANIP                  string             `json:"wan_ip,omitempty"`
	PPPoEUsername          string             `json:"pppoe_username,omitempty"`
	LastInform             string             `json:"last_inform,omitempty"`
	DataRealtime           bool               `json:"data_realtime"`
	ConnectedHostCount     int                `json:"connected_host_count"`
	ConnectedHostsComplete bool               `json:"connected_hosts_complete"`
	ConnectedHostsSource   string             `json:"connected_hosts_source,omitempty"`
	ConnectedHosts         []ONTConnectedHost `json:"connected_hosts"`
}

type ONT struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	CustomerID   *string    `json:"customer_id,omitempty"`
	ODPPortID    *string    `json:"odp_port_id,omitempty"`
	SerialNumber string     `json:"serial_number"`
	Model        *string    `json:"model,omitempty"`
	Vendor       *string    `json:"vendor,omitempty"`
	MACAddress   *string    `json:"mac_address,omitempty"`
	IPAddress    *string    `json:"ip_address,omitempty"`
	LastKnownWiFiSSID      *string    `json:"last_known_wifi_ssid,omitempty"`
	LastKnownWiFiPassword  *string    `json:"last_known_wifi_password,omitempty"`
	LastKnownWiFiSecurity  *string    `json:"last_known_wifi_security,omitempty"`
	LastKnownWiFiSource    *string    `json:"last_known_wifi_source,omitempty"`
	LastKnownWiFiUpdatedAt *time.Time `json:"last_known_wifi_updated_at,omitempty"`
	LastKnownConnectedHosts          *string    `json:"last_known_connected_hosts,omitempty"`
	LastKnownConnectedHostCount      *int       `json:"last_known_connected_host_count,omitempty"`
	LastKnownConnectedHostsSource    *string    `json:"last_known_connected_hosts_source,omitempty"`
	LastKnownConnectedHostsUpdatedAt *time.Time `json:"last_known_connected_hosts_updated_at,omitempty"`
	RxPower      *float64   `json:"rx_power,omitempty"`
	TxPower      *float64   `json:"tx_power,omitempty"`
	Status        string     `json:"status"`
	Notes         *string    `json:"notes,omitempty"`
	LastOnlineAt  *time.Time `json:"last_online_at,omitempty"`
	ProvisionedAt *time.Time `json:"provisioned_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Customer     *Customer           `json:"customer,omitempty"`
	StatusDetail *ONTStatusDetail    `json:"status_detail,omitempty"`
	Actions      *ONTAvailableActions `json:"actions,omitempty"`
}
