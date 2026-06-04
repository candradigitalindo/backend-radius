package model

import "time"

type RadiusSession struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenant_id"`
	CustomerID          *string    `json:"customer_id,omitempty"`
	RouterID            *string    `json:"router_id,omitempty"`
	SessionID           string     `json:"session_id"`
	Username            string     `json:"username"`
	NASIPAddress        string     `json:"nas_ip_address"`
	FramedIP            string     `json:"framed_ip"`
	CallerID            string     `json:"caller_id"`
	InputOctets         int64      `json:"input_octets"`
	OutputOctets        int64      `json:"output_octets"`
	StartedAt           time.Time  `json:"started_at"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
	EndedAt             *time.Time `json:"ended_at,omitempty"`
	SessionTime         int        `json:"session_time"`
	Status              string     `json:"status"`
	TerminateCause      string     `json:"terminate_cause"`
	RealtimeDownloadBps int64      `json:"-"`
	RealtimeUploadBps   int64      `json:"-"`
	RealtimeSampledAt   *time.Time `json:"-"`
}

type RouterTraffic struct {
	RouterID         string `json:"router_id"`
	ActiveSessions   int    `json:"active_sessions"`
	TotalInputBytes  int64  `json:"total_input_bytes"`
	TotalOutputBytes int64  `json:"total_output_bytes"`
}
