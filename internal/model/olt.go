package model

import "time"

type OLT struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	RouterID      *string   `json:"router_id,omitempty"`
	Name          string    `json:"name"`
	IPAddress     string    `json:"ip_address"`
	Vendor        *string   `json:"vendor,omitempty"`
	Model         *string   `json:"model,omitempty"`
	SerialNumber  *string   `json:"serial_number,omitempty"`
	TotalPONPorts int       `json:"total_pon_ports"`
	Latitude      *float64  `json:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty"`
	SNMPCommunity string    `json:"snmp_community"`
	Status        string    `json:"status"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	PONPorts []PONPort `json:"pon_ports,omitempty"`
	Router   *Router   `json:"router,omitempty"`
}

type PONPort struct {
	ID             string     `json:"id"`
	OLTID          string     `json:"olt_id"`
	PortNumber     int        `json:"port_number"`
	Description    *string    `json:"description,omitempty"`
	Status         string     `json:"status"`
	IfIndex        *int       `json:"if_index,omitempty"`
	AdminStatus    string     `json:"admin_status,omitempty"`
	OperStatus     string     `json:"oper_status,omitempty"`
	InOctets       int64      `json:"in_octets"`
	OutOctets      int64      `json:"out_octets"`
	SFPVendor      *string    `json:"sfp_vendor,omitempty"`
	SFPModel       *string    `json:"sfp_model,omitempty"`
	SFPRxPower     *float64   `json:"sfp_rx_power,omitempty"`
	SFPTxPower     *float64   `json:"sfp_tx_power,omitempty"`
	SFPTemperature *float64   `json:"sfp_temperature,omitempty"`
	SFPVoltage     *float64   `json:"sfp_voltage,omitempty"`
	LastPolledAt   *time.Time `json:"last_polled_at,omitempty"`
}
