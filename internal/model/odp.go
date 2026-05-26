package model

import "time"

type ODP struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	OLTID         *string   `json:"olt_id,omitempty"`
	PONPortID     *string   `json:"pon_port_id,omitempty"`
	SplitterRatio *string   `json:"splitter_ratio,omitempty"`
	Name          string    `json:"name"`
	Address       *string   `json:"address,omitempty"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	TotalPorts    int       `json:"total_ports"`
	Sequence      int       `json:"sequence"`
	CableLengthM  float64   `json:"cable_length_m"`
	RatioPercent  float64   `json:"ratio_percent"`
	SplitterType  *string   `json:"splitter_type,omitempty"`
	PowerLevelDBm *float64  `json:"power_level_dbm,omitempty"`
	Status        string    `json:"status"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Computed / joined
	PONPortSFPRxPower *float64 `json:"pon_port_sfp_rx_power,omitempty"`
	PONPortSFPTxPower *float64 `json:"pon_port_sfp_tx_power"`
	PONPortNumber     *int     `json:"pon_port_number,omitempty"`
	SignalAttenuationDB *float64 `json:"signal_attenuation_db,omitempty"`

	Ports []ODPPort `json:"ports,omitempty"`
	OLT   *OLT      `json:"olt,omitempty"`
}

type ODPPort struct {
	ID         string  `json:"id"`
	ODPID      string  `json:"odp_id"`
	PortNumber int     `json:"port_number"`
	CustomerID *string `json:"customer_id,omitempty"`
	Status     string  `json:"status"`
	Notes      *string `json:"notes,omitempty"`

	Customer *Customer `json:"customer,omitempty"`
	ONT      *ONT      `json:"ont,omitempty"`
}

type Splitter struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	PONPortID        *string   `json:"pon_port_id,omitempty"`
	ParentSplitterID *string   `json:"parent_splitter_id,omitempty"`
	Name             string    `json:"name"`
	SplitterType     string    `json:"splitter_type"`
	Latitude         *float64  `json:"latitude,omitempty"`
	Longitude        *float64  `json:"longitude,omitempty"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	PONPort        *PONPort  `json:"pon_port,omitempty"`
	ParentSplitter *Splitter `json:"parent_splitter,omitempty"`
}
