package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type VPNPeerInfo struct {
	PublicKey string
	VPNIP     string
}

type RouterRepository interface {
	FindByVPNIP(ctx context.Context, vpnIP string) (*model.Router, error)
	ListAllVPNIPs(ctx context.Context) ([]string, error)
	ListLegacyVPNAccounts(ctx context.Context) ([]LegacyVPNAccount, error)
	ListAllLegacyVPNIPs(ctx context.Context) ([]string, error)
	SetLegacyVPN(ctx context.Context, tenantID, routerID, password, ip string) error
	ListVPNPeers(ctx context.Context) ([]VPNPeerInfo, error)
	Create(ctx context.Context, router *model.Router) error
	FindByID(ctx context.Context, tenantID, routerID string) (*model.Router, error)
	FindByIDOnly(ctx context.Context, routerID string) (*model.Router, error)
	Update(ctx context.Context, router *model.Router) error
	Delete(ctx context.Context, tenantID, routerID string) error
	List(ctx context.Context, tenantID string, filter RouterFilter) ([]model.Router, int, error)
	UpdateHeartbeat(ctx context.Context, routerID string, info HeartbeatInfo) (wasOffline bool, err error)
	UpdateNASIP(ctx context.Context, routerID, nasIP string) error
	FindByHeartbeatToken(ctx context.Context, token string) (*model.Router, error)
	MarkStaleOffline(ctx context.Context, tenantID string, staleThreshold time.Time) ([]string, error)
	ListStaleWithVPN(ctx context.Context, tenantID string, staleThreshold time.Time) ([]model.Router, error)
	CreateConnectionLog(ctx context.Context, log *model.RouterConnectionLog) error
	ListConnectionLogs(ctx context.Context, routerID string, page, perPage int) ([]model.RouterConnectionLog, int, error)
}

type RouterFilter struct {
	Search  string
	Status  string // "online", "offline", or ""
	Page    int
	PerPage int
}

type HeartbeatInfo struct {
	Identity    string
	RouterOSVer string
	BoardName   string
	Uptime      string
	CPULoad     int
	FreeMemory  int64
	TotalMemory int64
}

type routerRepository struct {
	db *pgxpool.Pool
}

func NewRouterRepository(db *pgxpool.Pool) RouterRepository {
	return &routerRepository{db: db}
}

func (r *routerRepository) Create(ctx context.Context, router *model.Router) error {
	router.ID = id.New()
	now := time.Now()
	router.CreatedAt = now
	router.UpdatedAt = now

	if router.SNMPCommunity == "" {
		router.SNMPCommunity = "public"
	}
	if router.RouterType == "" {
		router.RouterType = "mikrotik"
	}

	query := `
		INSERT INTO routers (
			id, tenant_id, name, router_type, identity, vpn_ip, vpn_public_key,
			radius_secret, coa_port, heartbeat_token, snmp_community, is_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(ctx, query,
		router.ID, router.TenantID, router.Name, router.RouterType, router.Identity,
		router.VPNIP, router.VPNPublicKey,
		router.RADIUSSecret, router.CoAPort, router.HeartbeatToken, router.SNMPCommunity, router.IsActive,
		router.CreatedAt, router.UpdatedAt,
	)
	return err
}

func (r *routerRepository) FindByID(ctx context.Context, tenantID, routerID string) (*model.Router, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(router_type,'mikrotik'), COALESCE(identity,''), vpn_ip, COALESCE(vpn_public_key,''),
		       COALESCE(vpn_password,''), COALESCE(legacy_vpn_ip,''),
		       radius_secret, coa_port, COALESCE(heartbeat_token,''),
		       is_online, last_seen_at, router_os_ver, board_name, uptime,
		       cpu_load, free_memory, total_memory, snmp_community, COALESCE(nas_ip,''),
		       is_active, created_at, updated_at
		FROM routers
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1
	`

	var rt model.Router
	err := r.db.QueryRow(ctx, query, routerID, tenantID).Scan(
		&rt.ID, &rt.TenantID, &rt.Name, &rt.RouterType, &rt.Identity,
		&rt.VPNIP, &rt.VPNPublicKey,
		&rt.VPNPassword, &rt.LegacyVPNIP,
		&rt.RADIUSSecret, &rt.CoAPort, &rt.HeartbeatToken,
		&rt.IsOnline, &rt.LastSeenAt, &rt.RouterOSVer, &rt.BoardName, &rt.Uptime,
		&rt.CPULoad, &rt.FreeMemory, &rt.TotalMemory, &rt.SNMPCommunity, &rt.NASIP,
		&rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *routerRepository) Update(ctx context.Context, router *model.Router) error {
	router.UpdatedAt = time.Now()

	query := `
		UPDATE routers SET
			name = $1, identity = $2, vpn_ip = $3, vpn_public_key = $4,
			radius_secret = $5, coa_port = $6, heartbeat_token = $7, snmp_community = $8,
			nas_ip = $9, is_active = $10, router_type = $11, updated_at = $12
		WHERE id = $13 AND tenant_id = $14
	`

	_, err := r.db.Exec(ctx, query,
		router.Name, router.Identity, router.VPNIP, router.VPNPublicKey,
		router.RADIUSSecret, router.CoAPort, router.HeartbeatToken, router.SNMPCommunity,
		router.NASIP, router.IsActive, router.RouterType, router.UpdatedAt,
		router.ID, router.TenantID,
	)
	return err
}

func (r *routerRepository) Delete(ctx context.Context, tenantID, routerID string) error {
	query := `DELETE FROM routers WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, routerID, tenantID)
	return err
}

func (r *routerRepository) List(ctx context.Context, tenantID string, filter RouterFilter) ([]model.Router, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR identity ILIKE $%d OR vpn_ip ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Status == "online" {
		conditions = append(conditions, "is_online = TRUE")
	} else if filter.Status == "offline" {
		conditions = append(conditions, "is_online = FALSE")
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM routers "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Pagination
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, COALESCE(router_type,'mikrotik'), COALESCE(identity,''), vpn_ip, COALESCE(vpn_public_key,''),
		       radius_secret, coa_port, COALESCE(heartbeat_token,''),
		       is_online, last_seen_at, router_os_ver, board_name, uptime,
		       cpu_load, free_memory, total_memory, snmp_community,
		       is_active, created_at, updated_at
		FROM routers
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var routers []model.Router
	for rows.Next() {
		var rt model.Router
		if err := rows.Scan(
			&rt.ID, &rt.TenantID, &rt.Name, &rt.RouterType, &rt.Identity,
			&rt.VPNIP, &rt.VPNPublicKey,
			&rt.RADIUSSecret, &rt.CoAPort, &rt.HeartbeatToken,
			&rt.IsOnline, &rt.LastSeenAt, &rt.RouterOSVer, &rt.BoardName, &rt.Uptime,
			&rt.CPULoad, &rt.FreeMemory, &rt.TotalMemory, &rt.SNMPCommunity,
			&rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		routers = append(routers, rt)
	}

	return routers, total, nil
}

func (r *routerRepository) UpdateHeartbeat(ctx context.Context, routerID string, info HeartbeatInfo) (bool, error) {
	var wasOffline bool
	err := r.db.QueryRow(ctx, `SELECT NOT is_online FROM routers WHERE id = $1`, routerID).Scan(&wasOffline)
	if err != nil {
		return false, err
	}

	query := `
		UPDATE routers SET
			is_online = TRUE, last_seen_at = NOW(),
			identity = CASE WHEN $1 = '' THEN identity ELSE $1 END,
			router_os_ver = CASE WHEN $2 = '' THEN router_os_ver ELSE $2 END,
			board_name = CASE WHEN $3 = '' THEN board_name ELSE $3 END,
			uptime = CASE WHEN $4 = '' THEN uptime ELSE $4 END,
			cpu_load = $5, free_memory = $6, total_memory = $7,
			updated_at = NOW()
		WHERE id = $8
	`
	_, err = r.db.Exec(ctx, query,
		info.Identity, info.RouterOSVer, info.BoardName, info.Uptime,
		info.CPULoad, info.FreeMemory, info.TotalMemory,
		routerID,
	)
	return wasOffline, err
}

func (r *routerRepository) FindByHeartbeatToken(ctx context.Context, token string) (*model.Router, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(identity,''), vpn_ip, COALESCE(vpn_public_key,''),
		       radius_secret, coa_port, COALESCE(heartbeat_token,''),
		       is_online, last_seen_at, is_active
		FROM routers
		WHERE heartbeat_token = $1 AND is_active = TRUE
		LIMIT 1
	`

	var rt model.Router
	err := r.db.QueryRow(ctx, query, token).Scan(
		&rt.ID, &rt.TenantID, &rt.Name, &rt.Identity,
		&rt.VPNIP, &rt.VPNPublicKey,
		&rt.RADIUSSecret, &rt.CoAPort, &rt.HeartbeatToken,
		&rt.IsOnline, &rt.LastSeenAt, &rt.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *routerRepository) FindByIDOnly(ctx context.Context, routerID string) (*model.Router, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(identity,''), vpn_ip, COALESCE(vpn_public_key,''),
		       COALESCE(legacy_vpn_ip,''), radius_secret, coa_port, is_online, last_seen_at,
		       router_os_ver, board_name, uptime, cpu_load, free_memory, total_memory, snmp_community
		FROM routers
		WHERE id = $1
		LIMIT 1
	`

	var rt model.Router
	err := r.db.QueryRow(ctx, query, routerID).Scan(
		&rt.ID, &rt.TenantID, &rt.Name, &rt.Identity,
		&rt.VPNIP, &rt.VPNPublicKey, &rt.LegacyVPNIP,
		&rt.RADIUSSecret, &rt.CoAPort, &rt.IsOnline, &rt.LastSeenAt,
		&rt.RouterOSVer, &rt.BoardName, &rt.Uptime, &rt.CPULoad, &rt.FreeMemory, &rt.TotalMemory, &rt.SNMPCommunity,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

// FindByVPNIP resolves the registered router that a RADIUS packet came from,
// by its source/NAS IP. It matches the WireGuard VPN IP (VPN mode), the static
// L2TP/SSTP tunnel IP (legacy VPN mode), or the WAN IP / nas_ip (Direct /
// IP Publik mode), so authentication works regardless of how the router is
// connected. Exact tunnel-IP matches are always preferred so a stale nas_ip can
// never shadow a live tunneled router.
func (r *routerRepository) FindByVPNIP(ctx context.Context, vpnIP string) (*model.Router, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(router_type,'mikrotik'), COALESCE(identity,''), vpn_ip, COALESCE(vpn_public_key,''),
		       COALESCE(legacy_vpn_ip,''), radius_secret, coa_port, is_online, is_active, COALESCE(nas_ip,'')
		FROM routers
		WHERE (vpn_ip = $1 OR legacy_vpn_ip = $1 OR nas_ip = $1) AND is_active = TRUE
		ORDER BY (vpn_ip = $1) DESC, (legacy_vpn_ip = $1) DESC
		LIMIT 1
	`

	var router model.Router
	err := r.db.QueryRow(ctx, query, vpnIP).Scan(
		&router.ID, &router.TenantID, &router.Name, &router.RouterType, &router.Identity,
		&router.VPNIP, &router.VPNPublicKey, &router.LegacyVPNIP, &router.RADIUSSecret,
		&router.CoAPort, &router.IsOnline, &router.IsActive, &router.NASIP,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &router, nil
}

// LegacyVPNAccount is one row of the accel-ppp chap-secrets file: the tunnel
// login (router ID), its password and the static tunnel IP.
type LegacyVPNAccount struct {
	RouterID string
	Password string
	IP       string
}

// ListLegacyVPNAccounts returns every active router with provisioned L2TP/SSTP
// credentials, for regenerating the accel-ppp chap-secrets file.
func (r *routerRepository) ListLegacyVPNAccounts(ctx context.Context) ([]LegacyVPNAccount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(vpn_password,''), legacy_vpn_ip
		FROM routers
		WHERE legacy_vpn_ip IS NOT NULL AND legacy_vpn_ip != '' AND is_active = TRUE
		ORDER BY legacy_vpn_ip
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []LegacyVPNAccount
	for rows.Next() {
		var a LegacyVPNAccount
		if err := rows.Scan(&a.RouterID, &a.Password, &a.IP); err != nil {
			return nil, err
		}
		if a.Password != "" {
			accounts = append(accounts, a)
		}
	}
	return accounts, rows.Err()
}

// ListAllLegacyVPNIPs returns every allocated L2TP/SSTP tunnel IP (including
// inactive routers, so a disabled router's IP is never handed to another one).
func (r *routerRepository) ListAllLegacyVPNIPs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT legacy_vpn_ip FROM routers WHERE legacy_vpn_ip IS NOT NULL AND legacy_vpn_ip != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

// SetLegacyVPN stores the generated L2TP/SSTP credentials for a router.
func (r *routerRepository) SetLegacyVPN(ctx context.Context, tenantID, routerID, password, ip string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE routers SET vpn_password = $1, legacy_vpn_ip = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`, password, ip, routerID, tenantID)
	return err
}

// UpdateNASIP saves the router's WAN IP (used in Direct mode without WireGuard).
func (r *routerRepository) UpdateNASIP(ctx context.Context, routerID, nasIP string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE routers SET nas_ip = $1, updated_at = NOW() WHERE id = $2
	`, nasIP, routerID)
	return err
}

func (r *routerRepository) ListAllVPNIPs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT vpn_ip FROM routers WHERE vpn_ip != '' AND vpn_ip IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

func (r *routerRepository) ListVPNPeers(ctx context.Context) ([]VPNPeerInfo, error) {
	rows, err := r.db.Query(ctx, `SELECT vpn_public_key, vpn_ip FROM routers WHERE vpn_public_key != '' AND vpn_public_key IS NOT NULL AND vpn_ip != '' AND vpn_ip IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []VPNPeerInfo
	for rows.Next() {
		var p VPNPeerInfo
		if err := rows.Scan(&p.PublicKey, &p.VPNIP); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

func (r *routerRepository) MarkStaleOffline(ctx context.Context, tenantID string, staleThreshold time.Time) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id FROM routers
		WHERE tenant_id = $1
		  AND is_online = TRUE
		  AND is_active = TRUE
		  AND (last_seen_at IS NULL OR last_seen_at < $2)
	`, tenantID, staleThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, err
		}
		ids = append(ids, rid)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	_, err = r.db.Exec(ctx, `
		UPDATE routers
		SET is_online = FALSE, updated_at = NOW()
		WHERE tenant_id = $1
		  AND is_online = TRUE
		  AND is_active = TRUE
		  AND (last_seen_at IS NULL OR last_seen_at < $2)
	`, tenantID, staleThreshold)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// ListStaleWithVPN returns active routers that haven't sent a heartbeat since staleThreshold
// and have a VPN IP + SNMP community configured — these are candidates for SNMP polling.
func (r *routerRepository) ListStaleWithVPN(ctx context.Context, tenantID string, staleThreshold time.Time) ([]model.Router, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, vpn_ip, COALESCE(snmp_community, 'public')
		FROM routers
		WHERE tenant_id = $1
		  AND is_active = TRUE
		  AND vpn_ip != ''
		  AND COALESCE(snmp_community, '') != ''
		  AND (last_seen_at IS NULL OR last_seen_at < $2)
		ORDER BY last_seen_at ASC NULLS FIRST
	`, tenantID, staleThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []model.Router
	for rows.Next() {
		var rt model.Router
		if err := rows.Scan(&rt.ID, &rt.VPNIP, &rt.SNMPCommunity); err != nil {
			return nil, err
		}
		routers = append(routers, rt)
	}
	return routers, nil
}

func (r *routerRepository) CreateConnectionLog(ctx context.Context, logEntry *model.RouterConnectionLog) error {
	logEntry.ID = id.New()
	if logEntry.CreatedAt.IsZero() {
		logEntry.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO router_connection_logs (id, router_id, router_name, event, vpn_ip, endpoint, identity, router_os_ver, board_name, uptime, cpu_load, free_memory, total_memory, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(ctx, query,
		logEntry.ID, logEntry.RouterID, logEntry.RouterName, logEntry.Event, logEntry.VPNIP, logEntry.Endpoint,
		logEntry.Identity, logEntry.RouterOSVer, logEntry.BoardName, logEntry.Uptime,
		logEntry.CPULoad, logEntry.FreeMemory, logEntry.TotalMemory, logEntry.CreatedAt,
	)
	return err
}

func (r *routerRepository) ListConnectionLogs(ctx context.Context, routerID string, page, perPage int) ([]model.RouterConnectionLog, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM router_connection_logs WHERE router_id = $1`, routerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT id, router_id, COALESCE(router_name,''), event, COALESCE(vpn_ip,''), COALESCE(endpoint,''),
		       COALESCE(identity,''), COALESCE(router_os_ver,''), COALESCE(board_name,''), COALESCE(uptime,''),
		       cpu_load, free_memory, total_memory, COALESCE(duration,''), created_at
		FROM router_connection_logs
		WHERE router_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, routerID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.RouterConnectionLog
	for rows.Next() {
		var l model.RouterConnectionLog
		if err := rows.Scan(&l.ID, &l.RouterID, &l.RouterName, &l.Event, &l.VPNIP, &l.Endpoint,
			&l.Identity, &l.RouterOSVer, &l.BoardName, &l.Uptime,
			&l.CPULoad, &l.FreeMemory, &l.TotalMemory, &l.Duration, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
