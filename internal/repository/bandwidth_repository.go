package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BandwidthRepository interface {
	GetCustomerUsage(ctx context.Context, tenantID, customerID string, month, year int) (*BandwidthUsage, error)
	GetCustomerUsageHistory(ctx context.Context, tenantID, customerID string, year int) ([]MonthlyBandwidth, error)
	GetTopUsers(ctx context.Context, tenantID string, month, year, limit int) ([]TopBandwidthUser, error)
	GetUsageSummary(ctx context.Context, tenantID string, month, year int) (*BandwidthSummary, error)
	GetSaturationReport(ctx context.Context, tenantID string, hours, thresholdPct, limit int) ([]SaturationEntry, error)
	GetLatestSampleBySession(ctx context.Context, tenantID, sessionID string) (*BandwidthSample, error)
	GetSessionTransferTotals(ctx context.Context, tenantID string, sessionIDs []string) (map[string]SessionTransferTotal, error)
}

type BandwidthSample struct {
	SessionID   string    `json:"session_id"`
	SampledAt   time.Time `json:"sampled_at"`
	IntervalSec int       `json:"interval_sec"`
	DownloadBps int64     `json:"download_bps"`
	UploadBps   int64     `json:"upload_bps"`
}

type SessionTransferTotal struct {
	SessionID     string `json:"session_id"`
	TotalDownload int64  `json:"total_download"`
	TotalUpload   int64  `json:"total_upload"`
}

type BandwidthUsage struct {
	CustomerID    string `json:"customer_id"`
	TotalUpload   int64  `json:"total_upload"`
	TotalDownload int64  `json:"total_download"`
	TotalBytes    int64  `json:"total_bytes"`
	SessionCount  int    `json:"session_count"`
	TotalTime     int    `json:"total_time_seconds"`
}

type MonthlyBandwidth struct {
	Month         int   `json:"month"`
	TotalUpload   int64 `json:"total_upload"`
	TotalDownload int64 `json:"total_download"`
	TotalBytes    int64 `json:"total_bytes"`
	SessionCount  int   `json:"session_count"`
}

type TopBandwidthUser struct {
	CustomerID    string `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	Username      string `json:"username"`
	TotalUpload   int64  `json:"total_upload"`
	TotalDownload int64  `json:"total_download"`
	TotalBytes    int64  `json:"total_bytes"`
}

type BandwidthSummary struct {
	TotalUpload    int64 `json:"total_upload"`
	TotalDownload  int64 `json:"total_download"`
	TotalBytes     int64 `json:"total_bytes"`
	ActiveSessions int   `json:"active_sessions"`
	TotalSessions  int   `json:"total_sessions"`
	UniqueUsers    int   `json:"unique_users"`
}

type bandwidthRepository struct {
	db *pgxpool.Pool
}

func NewBandwidthRepository(db *pgxpool.Pool) BandwidthRepository {
	return &bandwidthRepository{db: db}
}

func (r *bandwidthRepository) GetCustomerUsage(ctx context.Context, tenantID, customerID string, month, year int) (*BandwidthUsage, error) {
	query := `
		SELECT
			COALESCE(SUM(input_octets), 0) as total_upload,
			COALESCE(SUM(output_octets), 0) as total_download,
			COALESCE(SUM(input_octets + output_octets), 0) as total_bytes,
			COUNT(*) as session_count,
			COALESCE(SUM(session_time), 0) as total_time
		FROM radius_sessions
		WHERE tenant_id = $1
		  AND customer_id = $2
		  AND EXTRACT(MONTH FROM started_at) = $3
		  AND EXTRACT(YEAR FROM started_at) = $4
	`

	var u BandwidthUsage
	u.CustomerID = customerID
	err := r.db.QueryRow(ctx, query, tenantID, customerID, month, year).Scan(
		&u.TotalUpload, &u.TotalDownload, &u.TotalBytes, &u.SessionCount, &u.TotalTime,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *bandwidthRepository) GetCustomerUsageHistory(ctx context.Context, tenantID, customerID string, year int) ([]MonthlyBandwidth, error) {
	query := `
		SELECT
			m.month,
			COALESCE(SUM(rs.input_octets), 0) as total_upload,
			COALESCE(SUM(rs.output_octets), 0) as total_download,
			COALESCE(SUM(rs.input_octets + rs.output_octets), 0) as total_bytes,
			COUNT(rs.id) as session_count
		FROM generate_series(1, 12) AS m(month)
		LEFT JOIN radius_sessions rs
			ON EXTRACT(MONTH FROM rs.started_at) = m.month
			AND EXTRACT(YEAR FROM rs.started_at) = $3
			AND rs.tenant_id = $1
			AND rs.customer_id = $2
		GROUP BY m.month
		ORDER BY m.month
	`

	rows, err := r.db.Query(ctx, query, tenantID, customerID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthlyBandwidth
	for rows.Next() {
		var mb MonthlyBandwidth
		if err := rows.Scan(&mb.Month, &mb.TotalUpload, &mb.TotalDownload, &mb.TotalBytes, &mb.SessionCount); err != nil {
			return nil, err
		}
		result = append(result, mb)
	}
	return result, nil
}

func (r *bandwidthRepository) GetTopUsers(ctx context.Context, tenantID string, month, year, limit int) ([]TopBandwidthUser, error) {
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf(`
		SELECT
			rs.customer_id,
			COALESCE(c.name, rs.username) as customer_name,
			rs.username,
			SUM(rs.input_octets) as total_upload,
			SUM(rs.output_octets) as total_download,
			SUM(rs.input_octets + rs.output_octets) as total_bytes
		FROM radius_sessions rs
		LEFT JOIN customers c ON rs.customer_id = c.id
		WHERE rs.tenant_id = $1
		  AND EXTRACT(MONTH FROM rs.started_at) = $2
		  AND EXTRACT(YEAR FROM rs.started_at) = $3
		  AND rs.customer_id IS NOT NULL
		GROUP BY rs.customer_id, c.name, rs.username
		ORDER BY total_bytes DESC
		LIMIT %d
	`, limit)

	rows, err := r.db.Query(ctx, query, tenantID, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TopBandwidthUser
	for rows.Next() {
		var t TopBandwidthUser
		if err := rows.Scan(&t.CustomerID, &t.CustomerName, &t.Username, &t.TotalUpload, &t.TotalDownload, &t.TotalBytes); err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, nil
}

func (r *bandwidthRepository) GetUsageSummary(ctx context.Context, tenantID string, month, year int) (*BandwidthSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(input_octets), 0) as total_upload,
			COALESCE(SUM(output_octets), 0) as total_download,
			COALESCE(SUM(input_octets + output_octets), 0) as total_bytes,
			COUNT(*) FILTER (WHERE status = 'active') as active_sessions,
			COUNT(*) as total_sessions,
			COUNT(DISTINCT username) as unique_users
		FROM radius_sessions
		WHERE tenant_id = $1
		  AND EXTRACT(MONTH FROM started_at) = $2
		  AND EXTRACT(YEAR FROM started_at) = $3
	`

	var s BandwidthSummary
	err := r.db.QueryRow(ctx, query, tenantID, month, year).Scan(
		&s.TotalUpload, &s.TotalDownload, &s.TotalBytes,
		&s.ActiveSessions, &s.TotalSessions, &s.UniqueUsers,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type SaturationEntry struct {
	CustomerID      string `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	Username        string `json:"username"`
	PackageName     string `json:"package_name"`
	BandwidthDownMb int    `json:"bandwidth_down_mbps"`
	BandwidthUpMb   int    `json:"bandwidth_up_mbps"`
	TotalSamples    int    `json:"total_samples"`
	SaturatedDown   int    `json:"saturated_download"`
	SaturatedUp     int    `json:"saturated_upload"`
	SaturationPct   int    `json:"saturation_pct"`
	PeakDownloadBps int64  `json:"peak_download_bps"`
	PeakUploadBps   int64  `json:"peak_upload_bps"`
	AvgDownloadBps  int64  `json:"avg_download_bps"`
	AvgUploadBps    int64  `json:"avg_upload_bps"`
}

func (r *bandwidthRepository) GetSaturationReport(ctx context.Context, tenantID string, hours, thresholdPct, limit int) ([]SaturationEntry, error) {
	if hours <= 0 {
		hours = 24
	}
	if thresholdPct <= 0 {
		thresholdPct = 80
	}
	if limit <= 0 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT
			c.id AS customer_id,
			c.name AS customer_name,
			c.pppoe_username AS username,
			COALESCE(p.name, '') AS package_name,
			COALESCE(p.bandwidth_down, 0) AS bandwidth_down,
			COALESCE(p.bandwidth_up, 0) AS bandwidth_up,
			COUNT(*) AS total_samples,
			COUNT(*) FILTER (WHERE bs.download_bps >= p.bandwidth_down * 1000000 * %d / 100) AS saturated_down,
			COUNT(*) FILTER (WHERE bs.upload_bps >= p.bandwidth_up * 1000000 * %d / 100) AS saturated_up,
			GREATEST(
				COUNT(*) FILTER (WHERE bs.download_bps >= p.bandwidth_down * 1000000 * %d / 100),
				COUNT(*) FILTER (WHERE bs.upload_bps >= p.bandwidth_up * 1000000 * %d / 100)
			) * 100 / GREATEST(COUNT(*), 1) AS saturation_pct,
			MAX(bs.download_bps) AS peak_download_bps,
			MAX(bs.upload_bps) AS peak_upload_bps,
			AVG(bs.download_bps)::bigint AS avg_download_bps,
			AVG(bs.upload_bps)::bigint AS avg_upload_bps
		FROM bandwidth_samples bs
		JOIN customers c ON bs.customer_id = c.id
		LEFT JOIN packages p ON c.package_id = p.id
		WHERE bs.tenant_id = $1
		  AND bs.sampled_at > NOW() - make_interval(hours => $2)
		  AND p.bandwidth_down > 0
		GROUP BY c.id, c.name, c.pppoe_username, p.name, p.bandwidth_down, p.bandwidth_up
		HAVING COUNT(*) >= 3
		ORDER BY saturation_pct DESC, peak_download_bps DESC
		LIMIT $3
	`, thresholdPct, thresholdPct, thresholdPct, thresholdPct)

	rows, err := r.db.Query(ctx, query, tenantID, hours, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SaturationEntry
	for rows.Next() {
		var e SaturationEntry
		if err := rows.Scan(
			&e.CustomerID, &e.CustomerName, &e.Username,
			&e.PackageName, &e.BandwidthDownMb, &e.BandwidthUpMb,
			&e.TotalSamples, &e.SaturatedDown, &e.SaturatedUp, &e.SaturationPct,
			&e.PeakDownloadBps, &e.PeakUploadBps, &e.AvgDownloadBps, &e.AvgUploadBps,
		); err != nil {
			return nil, err
		}
		result = append(result, e)
	}

	return result, nil
}

func (r *bandwidthRepository) GetLatestSampleBySession(ctx context.Context, tenantID, sessionID string) (*BandwidthSample, error) {
	query := `
		SELECT session_id, sampled_at, interval_sec, download_bps, upload_bps
		FROM bandwidth_samples
		WHERE tenant_id = $1 AND session_id = $2
		ORDER BY sampled_at DESC
		LIMIT 1
	`

	var sample BandwidthSample
	err := r.db.QueryRow(ctx, query, tenantID, sessionID).Scan(
		&sample.SessionID,
		&sample.SampledAt,
		&sample.IntervalSec,
		&sample.DownloadBps,
		&sample.UploadBps,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &sample, nil
}

func (r *bandwidthRepository) GetSessionTransferTotals(ctx context.Context, tenantID string, sessionIDs []string) (map[string]SessionTransferTotal, error) {
	if len(sessionIDs) == 0 {
		return map[string]SessionTransferTotal{}, nil
	}

	query := `
		SELECT
			session_id,
			COALESCE(SUM((download_bps::numeric * interval_sec) / 8), 0)::bigint AS total_download,
			COALESCE(SUM((upload_bps::numeric * interval_sec) / 8), 0)::bigint AS total_upload
		FROM bandwidth_samples
		WHERE tenant_id = $1
		  AND session_id = ANY($2)
		GROUP BY session_id
	`

	rows, err := r.db.Query(ctx, query, tenantID, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totals := make(map[string]SessionTransferTotal, len(sessionIDs))
	for rows.Next() {
		var total SessionTransferTotal
		if err := rows.Scan(&total.SessionID, &total.TotalDownload, &total.TotalUpload); err != nil {
			return nil, err
		}
		totals[total.SessionID] = total
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return totals, nil
}
