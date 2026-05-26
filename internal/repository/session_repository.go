package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

// InterimResult holds the previous octets so the caller can compute deltas.
type InterimResult struct {
	TenantID       string
	CustomerID     *string
	PrevInput      int64
	PrevOutput     int64
	PrevSessionTime int
}

type SessionRepository interface {
	Create(ctx context.Context, session *model.RadiusSession) error
	UpdateUsage(ctx context.Context, sessionID string, inputOctets, outputOctets uint32, sessionTime uint32) (*InterimResult, error)
	EndSession(ctx context.Context, sessionID string, terminateCause string, inputOctets, outputOctets uint32, sessionTime uint32) error
	FindActiveByUsername(ctx context.Context, tenantID, username string) (*model.RadiusSession, error)
	ListByRouter(ctx context.Context, tenantID, routerID string, activeOnly bool, page, perPage int) ([]model.RadiusSession, int, error)
	ListByCustomer(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.RadiusSession, int, error)
	TrafficByRouter(ctx context.Context, tenantID, routerID string) (*model.RouterTraffic, error)
	CleanStaleSessions(ctx context.Context, tenantID, routerID string) (int, error)
	CleanAllStaleSessions(ctx context.Context, tenantID string) (int, error)
	CleanAllStaleSessionsWithCustomers(ctx context.Context, tenantID string) (int, []string, error)
	InsertBandwidthSample(ctx context.Context, tenantID string, customerID *string, sessionID string, intervalSec int, downloadBps, uploadBps int64) error
}

type sessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *model.RadiusSession) error {
	session.ID = id.New()
	session.Status = "active"

	query := `
		INSERT INTO radius_sessions (
			id, tenant_id, customer_id, router_id,
			session_id, username, nas_ip_address, framed_ip, caller_id,
			started_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		session.ID, session.TenantID, session.CustomerID, session.RouterID,
		session.SessionID, session.Username, session.NASIPAddress, session.FramedIP, session.CallerID,
		session.StartedAt, session.Status,
	)
	return err
}

func (r *sessionRepository) UpdateUsage(ctx context.Context, sessionID string, inputOctets, outputOctets uint32, sessionTime uint32) (*InterimResult, error) {
	// CTE captures old values before the UPDATE applies
	query := `
		WITH old AS (
			SELECT tenant_id, customer_id, COALESCE(input_octets,0) AS prev_in, COALESCE(output_octets,0) AS prev_out, COALESCE(session_time,0) AS prev_time
			FROM radius_sessions WHERE session_id = $4 AND status = 'active'
		)
		UPDATE radius_sessions rs
		SET input_octets = $1, output_octets = $2, session_time = $3, updated_at = NOW()
		FROM old
		WHERE rs.session_id = $4 AND rs.status = 'active'
		RETURNING old.tenant_id, old.customer_id, old.prev_in, old.prev_out, old.prev_time
	`

	var res InterimResult
	err := r.db.QueryRow(ctx, query, int64(inputOctets), int64(outputOctets), int(sessionTime), sessionID).Scan(
		&res.TenantID, &res.CustomerID, &res.PrevInput, &res.PrevOutput, &res.PrevSessionTime,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *sessionRepository) EndSession(ctx context.Context, sessionID string, terminateCause string, inputOctets, outputOctets uint32, sessionTime uint32) error {
	now := time.Now()
	query := `
		UPDATE radius_sessions
		SET status = 'stopped', ended_at = $1, terminate_cause = $2,
		    input_octets = $3, output_octets = $4, session_time = $5, updated_at = $1
		WHERE session_id = $6 AND status = 'active'
	`
	_, err := r.db.Exec(ctx, query, now, terminateCause, int64(inputOctets), int64(outputOctets), int(sessionTime), sessionID)
	return err
}

func (r *sessionRepository) FindActiveByUsername(ctx context.Context, tenantID, username string) (*model.RadiusSession, error) {
	query := `
		SELECT id, tenant_id, customer_id, router_id ,
		       session_id, username, nas_ip_address, COALESCE(framed_ip,''), COALESCE(caller_id,''),
		       input_octets, output_octets, started_at, updated_at, ended_at,
		       session_time, status, COALESCE(terminate_cause,'')
		FROM radius_sessions
		WHERE tenant_id = $1 AND username = $2 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1
	`

	var s model.RadiusSession
	err := r.db.QueryRow(ctx, query, tenantID, username).Scan(
		&s.ID, &s.TenantID, &s.CustomerID, &s.RouterID,
		&s.SessionID, &s.Username, &s.NASIPAddress, &s.FramedIP, &s.CallerID,
		&s.InputOctets, &s.OutputOctets, &s.StartedAt, &s.UpdatedAt, &s.EndedAt,
		&s.SessionTime, &s.Status, &s.TerminateCause,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) ListByRouter(ctx context.Context, tenantID, routerID string, activeOnly bool, page, perPage int) ([]model.RadiusSession, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	statusFilter := ""
	if activeOnly {
		statusFilter = " AND status = 'active'"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM radius_sessions WHERE tenant_id = $1 AND router_id = $2%s`, statusFilter)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, tenantID, routerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, customer_id, router_id,
		       session_id, username, nas_ip_address, COALESCE(framed_ip,''), COALESCE(caller_id,''),
		       input_octets, output_octets, started_at, updated_at, ended_at,
		       session_time, status, COALESCE(terminate_cause,'')
		FROM radius_sessions
		WHERE tenant_id = $1 AND router_id = $2%s
		ORDER BY started_at DESC
		LIMIT $3 OFFSET $4
	`, statusFilter)

	rows, err := r.db.Query(ctx, dataQuery, tenantID, routerID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []model.RadiusSession
	for rows.Next() {
		var s model.RadiusSession
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.CustomerID, &s.RouterID,
			&s.SessionID, &s.Username, &s.NASIPAddress, &s.FramedIP, &s.CallerID,
			&s.InputOctets, &s.OutputOctets, &s.StartedAt, &s.UpdatedAt, &s.EndedAt,
			&s.SessionTime, &s.Status, &s.TerminateCause,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, s)
	}

	return sessions, total, nil
}

func (r *sessionRepository) ListByCustomer(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.RadiusSession, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM radius_sessions WHERE tenant_id = $1 AND customer_id = $2`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, tenantID, customerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQuery := `
		SELECT id, tenant_id, customer_id, router_id,
		       session_id, username, nas_ip_address, COALESCE(framed_ip,''), COALESCE(caller_id,''),
		       input_octets, output_octets, started_at, updated_at, ended_at,
		       session_time, status, COALESCE(terminate_cause,'')
		FROM radius_sessions
		WHERE tenant_id = $1 AND customer_id = $2
		ORDER BY started_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(ctx, dataQuery, tenantID, customerID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []model.RadiusSession
	for rows.Next() {
		var s model.RadiusSession
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.CustomerID, &s.RouterID,
			&s.SessionID, &s.Username, &s.NASIPAddress, &s.FramedIP, &s.CallerID,
			&s.InputOctets, &s.OutputOctets, &s.StartedAt, &s.UpdatedAt, &s.EndedAt,
			&s.SessionTime, &s.Status, &s.TerminateCause,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, s)
	}

	return sessions, total, nil
}

func (r *sessionRepository) TrafficByRouter(ctx context.Context, tenantID, routerID string) (*model.RouterTraffic, error) {
	query := `
		SELECT COUNT(*),
		       COALESCE(SUM(input_octets), 0),
		       COALESCE(SUM(output_octets), 0)
		FROM radius_sessions
		WHERE tenant_id = $1 AND router_id = $2 AND status = 'active'
	`

	var t model.RouterTraffic
	t.RouterID = routerID
	err := r.db.QueryRow(ctx, query, tenantID, routerID).Scan(
		&t.ActiveSessions, &t.TotalInputBytes, &t.TotalOutputBytes,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *sessionRepository) CleanStaleSessions(ctx context.Context, tenantID, routerID string) (int, error) {
	// Mark sessions as stopped if no update in 10 minutes
	threshold := time.Now().Add(-10 * time.Minute)
	query := `
		UPDATE radius_sessions
		SET status = 'stopped', ended_at = NOW(), terminate_cause = 'Admin-Reset'
		WHERE tenant_id = $1 AND router_id = $2 AND status = 'active'
		  AND ((updated_at IS NULL AND started_at < $3) OR updated_at < $3)
	`

	tag, err := r.db.Exec(ctx, query, tenantID, routerID, threshold)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *sessionRepository) CleanAllStaleSessions(ctx context.Context, tenantID string) (int, error) {
	threshold := time.Now().Add(-10 * time.Minute)
	query := `
		UPDATE radius_sessions
		SET status = 'stopped', ended_at = NOW(), terminate_cause = 'NAS-Timeout'
		WHERE tenant_id = $1 AND status = 'active'
		  AND ((updated_at IS NULL AND started_at < $2) OR updated_at < $2)
	`

	tag, err := r.db.Exec(ctx, query, tenantID, threshold)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// CleanAllStaleSessionsWithCustomers is like CleanAllStaleSessions but also returns
// the customer IDs of terminated sessions so IPAM IPs can be released.
func (r *sessionRepository) CleanAllStaleSessionsWithCustomers(ctx context.Context, tenantID string) (int, []string, error) {
	threshold := time.Now().Add(-10 * time.Minute)
	query := `
		UPDATE radius_sessions
		SET status = 'stopped', ended_at = NOW(), terminate_cause = 'NAS-Timeout'
		WHERE tenant_id = $1 AND status = 'active'
		  AND ((updated_at IS NULL AND started_at < $2) OR updated_at < $2)
		RETURNING customer_id
	`

	rows, err := r.db.Query(ctx, query, tenantID, threshold)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var customerIDs []string
	seen := make(map[string]bool)
	count := 0
	for rows.Next() {
		var cid *string
		if err := rows.Scan(&cid); err != nil {
			return 0, nil, err
		}
		count++
		if cid != nil && !seen[*cid] {
			seen[*cid] = true
			customerIDs = append(customerIDs, *cid)
		}
	}
	return count, customerIDs, nil
}

func (r *sessionRepository) InsertBandwidthSample(ctx context.Context, tenantID string, customerID *string, sessionID string, intervalSec int, downloadBps, uploadBps int64) error {
	query := `
		INSERT INTO bandwidth_samples (id, tenant_id, customer_id, session_id, sampled_at, interval_sec, download_bps, upload_bps)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, id.New(), tenantID, customerID, sessionID, intervalSec, downloadBps, uploadBps)
	return err
}
