package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

var ErrDuplicateIPPool = errors.New("pool dengan nama atau network yang sama sudah ada")

type IPAMRepository interface {
	// Pool operations
	CreatePool(ctx context.Context, pool *model.IPPool) error
	GetPool(ctx context.Context, tenantID, poolID string) (*model.IPPool, error)
	UpdatePool(ctx context.Context, pool *model.IPPool) error
	DeletePool(ctx context.Context, tenantID, poolID string) error
	ListPools(ctx context.Context, tenantID string, page, perPage int) ([]model.IPPool, int, error)

	// IP Address operations
	CreateAddress(ctx context.Context, addr *model.IPAddress) error
	CreateAddressBatch(ctx context.Context, addrs []model.IPAddress) error
	GetAddress(ctx context.Context, tenantID, addrID string) (*model.IPAddress, error)
	UpdateAddress(ctx context.Context, addr *model.IPAddress) error
	DeleteAddress(ctx context.Context, tenantID, addrID string) error
	ListAddresses(ctx context.Context, tenantID, poolID string, filter IPAddressFilter) ([]model.IPAddress, int, error)
	AssignAddress(ctx context.Context, tenantID, addrID, customerID string) error
	ReleaseAddress(ctx context.Context, tenantID, addrID string) error
	FindAvailable(ctx context.Context, tenantID, poolID string) (*model.IPAddress, error)
	FindByCustomerID(ctx context.Context, customerID string) (*model.IPAddress, error)
	FindByCustomerAndPool(ctx context.Context, customerID, poolID string) (*model.IPAddress, error)
	ReleaseAllByCustomerID(ctx context.Context, customerID string) error
	ReleaseAllByCustomerIDExcept(ctx context.Context, customerID, exceptAddrID string) error
	ReleaseAndAssign(ctx context.Context, tenantID, poolID, customerID string) (*model.IPAddress, error)
	FindPoolByRouterID(ctx context.Context, routerID string) (*model.IPPool, error)
	GetPoolStats(ctx context.Context, tenantID, poolID string) (*IPPoolStats, error)
}

type IPAddressFilter struct {
	Status  string
	Search  string
	Page    int
	PerPage int
}

type IPPoolStats struct {
	Total     int `json:"total"`
	Available int `json:"available"`
	Assigned  int `json:"assigned"`
	Reserved  int `json:"reserved"`
	Disabled  int `json:"disabled"`
}

type ipamRepository struct {
	db *pgxpool.Pool
}

func NewIPAMRepository(db *pgxpool.Pool) IPAMRepository {
	return &ipamRepository{db: db}
}

func (r *ipamRepository) CreatePool(ctx context.Context, pool *model.IPPool) error {
	pool.ID = id.New()
	now := time.Now()
	pool.CreatedAt = now
	pool.UpdatedAt = now

	query := `
		INSERT INTO ip_pools (id, tenant_id, name, network, gateway, dns_primary, dns_secondary, router_id, total_ips, used_ips, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		pool.ID, pool.TenantID, pool.Name, pool.Network, pool.Gateway,
		pool.DNSPrimary, pool.DNSSecondary, pool.RouterID,
		pool.TotalIPs, pool.UsedIPs, pool.Notes, pool.CreatedAt, pool.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateIPPool
		}
		return err
	}
	return nil
}

func (r *ipamRepository) GetPool(ctx context.Context, tenantID, poolID string) (*model.IPPool, error) {
	query := `
		SELECT TRIM(id), TRIM(tenant_id), name, network, COALESCE(gateway,''), COALESCE(dns_primary,''), COALESCE(dns_secondary,''), TRIM(router_id),
		       total_ips, used_ips, COALESCE(notes,''), created_at, updated_at
		FROM ip_pools WHERE id = $1::character(26) AND tenant_id = $2::character(26)
	`
	var p model.IPPool
	err := r.db.QueryRow(ctx, query, poolID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Network, &p.Gateway,
		&p.DNSPrimary, &p.DNSSecondary, &p.RouterID,
		&p.TotalIPs, &p.UsedIPs, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ipamRepository) UpdatePool(ctx context.Context, pool *model.IPPool) error {
	pool.UpdatedAt = time.Now()
	query := `
		UPDATE ip_pools SET name = $1, network = $2, gateway = $3, dns_primary = $4,
		       dns_secondary = $5, router_id = $6, notes = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10
	`
	_, err := r.db.Exec(ctx, query,
		pool.Name, pool.Network, pool.Gateway, pool.DNSPrimary,
		pool.DNSSecondary, pool.RouterID, pool.Notes, pool.UpdatedAt,
		pool.ID, pool.TenantID,
	)
	return err
}

func (r *ipamRepository) DeletePool(ctx context.Context, tenantID, poolID string) error {
	query := `DELETE FROM ip_pools WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, poolID, tenantID)
	return err
}

func (r *ipamRepository) ListPools(ctx context.Context, tenantID string, page, perPage int) ([]model.IPPool, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ip_pools WHERE tenant_id = $1::character(26)`, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT TRIM(id), TRIM(tenant_id), name, network, COALESCE(gateway,''), COALESCE(dns_primary,''), COALESCE(dns_secondary,''), TRIM(router_id),
		       total_ips, used_ips, COALESCE(notes,''), created_at, updated_at
		FROM ip_pools WHERE tenant_id = $1::character(26)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pools []model.IPPool
	for rows.Next() {
		var p model.IPPool
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Network, &p.Gateway,
			&p.DNSPrimary, &p.DNSSecondary, &p.RouterID,
			&p.TotalIPs, &p.UsedIPs, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		pools = append(pools, p)
	}
	return pools, total, nil
}

func (r *ipamRepository) CreateAddress(ctx context.Context, addr *model.IPAddress) error {
	addr.ID = id.New()
	now := time.Now()
	addr.CreatedAt = now
	addr.UpdatedAt = now
	if addr.Status == "" {
		addr.Status = "available"
	}

	query := `
		INSERT INTO ip_addresses (id, tenant_id, pool_id, address, customer_id, status, notes, assigned_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		addr.ID, addr.TenantID, addr.PoolID, addr.Address, addr.CustomerID,
		addr.Status, addr.Notes, addr.AssignedAt, addr.CreatedAt, addr.UpdatedAt,
	)
	return err
}

func (r *ipamRepository) CreateAddressBatch(ctx context.Context, addrs []model.IPAddress) error {
	if len(addrs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(`INSERT INTO ip_addresses (id, tenant_id, pool_id, address, status, notes, created_at, updated_at) VALUES `)
	args := make([]interface{}, 0, len(addrs)*8)
	now := time.Now()

	poolID := addrs[0].PoolID

	for i := range addrs {
		if addrs[i].ID == "" {
			addrs[i].ID = id.New()
		}
		addrs[i].CreatedAt = now
		addrs[i].UpdatedAt = now
		if addrs[i].Status == "" {
			addrs[i].Status = "available"
		}

		if i > 0 {
			sb.WriteString(", ")
		}
		base := i*8 + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6, base+7))
		args = append(args, addrs[i].ID, addrs[i].TenantID, addrs[i].PoolID, addrs[i].Address,
			addrs[i].Status, addrs[i].Notes, addrs[i].CreatedAt, addrs[i].UpdatedAt)
	}

	sb.WriteString(" ON CONFLICT (tenant_id, pool_id, address) DO NOTHING")
	if _, err := r.db.Exec(ctx, sb.String(), args...); err != nil {
		return err
	}

	// Update total_ips counter to reflect actual row count
	_, err := r.db.Exec(ctx, `
		UPDATE ip_pools SET total_ips = (
			SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1
		), updated_at = NOW()
		WHERE id = $1
	`, poolID)
	return err
}

func (r *ipamRepository) GetAddress(ctx context.Context, tenantID, addrID string) (*model.IPAddress, error) {
	query := `
		SELECT TRIM(id), TRIM(tenant_id), TRIM(pool_id), address, TRIM(customer_id), status, COALESCE(notes,''), assigned_at, created_at, updated_at
		FROM ip_addresses WHERE id = $1::character(26) AND tenant_id = $2::character(26)
	`
	var a model.IPAddress
	err := r.db.QueryRow(ctx, query, addrID, tenantID).Scan(
		&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
		&a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ipamRepository) UpdateAddress(ctx context.Context, addr *model.IPAddress) error {
	addr.UpdatedAt = time.Now()
	query := `
		UPDATE ip_addresses SET status = $1, notes = $2, updated_at = $3
		WHERE id = $4 AND tenant_id = $5
	`
	_, err := r.db.Exec(ctx, query, addr.Status, addr.Notes, addr.UpdatedAt, addr.ID, addr.TenantID)
	return err
}

func (r *ipamRepository) DeleteAddress(ctx context.Context, tenantID, addrID string) error {
	query := `DELETE FROM ip_addresses WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, addrID, tenantID)
	return err
}

func (r *ipamRepository) ListAddresses(ctx context.Context, tenantID, poolID string, filter IPAddressFilter) ([]model.IPAddress, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("a.tenant_id = $%d::character(26)", argIdx))
	args = append(args, strings.TrimSpace(tenantID))
	argIdx++

	conditions = append(conditions, fmt.Sprintf("a.pool_id = $%d::character(26)", argIdx))
	args = append(args, strings.TrimSpace(poolID))
	argIdx++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("a.address ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ip_addresses a "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT TRIM(a.id), TRIM(a.tenant_id), TRIM(a.pool_id), a.address, TRIM(a.customer_id),
		       COALESCE(c.name, '') as customer_name,
		       a.status, COALESCE(a.notes, ''), a.assigned_at, a.created_at, a.updated_at
		FROM ip_addresses a
		LEFT JOIN customers c ON c.id = a.customer_id
		%s
			ORDER BY inet(a.address)
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	addrs := make([]model.IPAddress, 0, total)
	for rows.Next() {
		var a model.IPAddress
		if err := rows.Scan(&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
			&a.CustomerName, &a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		addrs = append(addrs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return addrs, total, nil
}

func (r *ipamRepository) AssignAddress(ctx context.Context, tenantID, addrID, customerID string) error {
	now := time.Now()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		UPDATE ip_addresses SET customer_id = $1, status = 'assigned', assigned_at = $2, updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND status = 'available'
	`, customerID, now, addrID, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("ip address not available for assignment")
	}

	// Update pool used_ips counter within the same transaction
	_, _ = tx.Exec(ctx, `
		UPDATE ip_pools SET used_ips = (
			SELECT COUNT(*) FROM ip_addresses WHERE pool_id = (SELECT pool_id FROM ip_addresses WHERE id = $1) AND status = 'assigned'
		) WHERE id = (SELECT pool_id FROM ip_addresses WHERE id = $1)
	`, addrID)

	return tx.Commit(ctx)
}

func (r *ipamRepository) ReleaseAddress(ctx context.Context, tenantID, addrID string) error {
	now := time.Now()
	query := `
		UPDATE ip_addresses SET customer_id = NULL, status = 'available', assigned_at = NULL, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND status = 'assigned'
	`
	_, err := r.db.Exec(ctx, query, now, addrID, tenantID)
	if err != nil {
		return err
	}

	// Update pool used_ips counter
	_, _ = r.db.Exec(ctx, `
		UPDATE ip_pools SET used_ips = (
			SELECT COUNT(*) FROM ip_addresses WHERE pool_id = (SELECT pool_id FROM ip_addresses WHERE id = $1) AND status = 'assigned'
		) WHERE id = (SELECT pool_id FROM ip_addresses WHERE id = $1)
	`, addrID)

	return nil
}

func (r *ipamRepository) FindAvailable(ctx context.Context, tenantID, poolID string) (*model.IPAddress, error) {
	// FOR UPDATE SKIP LOCKED requires a transaction to hold the lock past the query.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var a model.IPAddress
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, pool_id, address, customer_id, status, notes, assigned_at, created_at, updated_at
		FROM ip_addresses WHERE tenant_id = $1::character(26) AND pool_id = $2::character(26) AND status = 'available'
		  AND address NOT IN (SELECT framed_ip FROM radius_sessions WHERE status = 'active' AND framed_ip IS NOT NULL AND framed_ip != '')
		ORDER BY inet(address) LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, tenantID, poolID).Scan(
		&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
		&a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = tx.Commit(ctx)
	return &a, nil
}

func (r *ipamRepository) GetPoolStats(ctx context.Context, tenantID, poolID string) (*IPPoolStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'available') as available,
			COUNT(*) FILTER (WHERE status = 'assigned') as assigned,
			COUNT(*) FILTER (WHERE status = 'reserved') as reserved,
			COUNT(*) FILTER (WHERE status = 'disabled') as disabled
		FROM ip_addresses
		WHERE tenant_id = $1::character(26) AND pool_id = $2::character(26)
	`
	var s IPPoolStats
	err := r.db.QueryRow(ctx, query, tenantID, poolID).Scan(
		&s.Total, &s.Available, &s.Assigned, &s.Reserved, &s.Disabled,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ipamRepository) FindByCustomerID(ctx context.Context, customerID string) (*model.IPAddress, error) {
	// ORDER BY: IPs from pool with most recent assignment first, ensuring deterministic result.
	// This also means if a customer somehow has 2 IPs, we always pick the most recently assigned one.
	query := `
		SELECT id, tenant_id, pool_id, address, customer_id, status, notes, assigned_at, created_at, updated_at
		FROM ip_addresses WHERE customer_id = $1 AND status = 'assigned'
		ORDER BY assigned_at DESC NULLS LAST
		LIMIT 1
	`
	var a model.IPAddress
	err := r.db.QueryRow(ctx, query, customerID).Scan(
		&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
		&a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ipamRepository) FindByCustomerAndPool(ctx context.Context, customerID, poolID string) (*model.IPAddress, error) {
	query := `
		SELECT id, tenant_id, pool_id, address, customer_id, status, notes, assigned_at, created_at, updated_at
		FROM ip_addresses WHERE customer_id = $1 AND pool_id = $2 AND status = 'assigned'
		ORDER BY assigned_at DESC NULLS LAST
		LIMIT 1
	`
	var a model.IPAddress
	err := r.db.QueryRow(ctx, query, customerID, poolID).Scan(
		&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
		&a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ipamRepository) ReleaseAllByCustomerID(ctx context.Context, customerID string) error {
	now := time.Now()

	// Find affected pool IDs before releasing
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT pool_id FROM ip_addresses WHERE customer_id = $1 AND status = 'assigned'
	`, customerID)
	if err != nil {
		return err
	}
	var poolIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		poolIDs = append(poolIDs, pid)
	}
	rows.Close()

	if len(poolIDs) == 0 {
		return nil
	}

	query := `
		UPDATE ip_addresses SET customer_id = NULL, status = 'available', assigned_at = NULL, updated_at = $1
		WHERE customer_id = $2 AND status = 'assigned'
	`
	_, err = r.db.Exec(ctx, query, now, customerID)
	if err != nil {
		return err
	}

	// Update pool used_ips counters only for affected pools
	for _, pid := range poolIDs {
		_, _ = r.db.Exec(ctx, `
			UPDATE ip_pools SET used_ips = (
				SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1 AND status = 'assigned'
			) WHERE id = $1
		`, pid)
	}

	return nil
}

// ReleaseAllByCustomerIDExcept releases all IPs for a customer except one specific address ID.
// Used after reusing an existing IP to clean up stale assignments in other pools.
func (r *ipamRepository) ReleaseAllByCustomerIDExcept(ctx context.Context, customerID, exceptAddrID string) error {
	now := time.Now()

	// Find affected pool IDs (excluding the one we're keeping)
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT pool_id FROM ip_addresses
		WHERE customer_id = $1 AND status = 'assigned' AND id != $2
	`, customerID, exceptAddrID)
	if err != nil {
		return err
	}
	var poolIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		poolIDs = append(poolIDs, pid)
	}
	rows.Close()

	if len(poolIDs) == 0 {
		return nil
	}

	_, err = r.db.Exec(ctx, `
		UPDATE ip_addresses SET customer_id = NULL, status = 'available', assigned_at = NULL, updated_at = $1
		WHERE customer_id = $2 AND status = 'assigned' AND id != $3
	`, now, customerID, exceptAddrID)
	if err != nil {
		return err
	}

	for _, pid := range poolIDs {
		_, _ = r.db.Exec(ctx, `
			UPDATE ip_pools SET used_ips = (
				SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1 AND status = 'assigned'
			) WHERE id = $1
		`, pid)
	}
	return nil
}

// ReleaseAndAssign atomically releases ANY previously assigned IPs for the customer
// across ALL pools, then assigns the next available IP from the target pool.
// This prevents IP accumulation when a router reconnects and RADIUS re-authenticates.
func (r *ipamRepository) ReleaseAndAssign(ctx context.Context, tenantID, poolID, customerID string) (*model.IPAddress, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Collect all pool IDs that will be affected by the release (for counter update).
	affectedRows, err := tx.Query(ctx, `
		SELECT DISTINCT pool_id FROM ip_addresses
		WHERE customer_id = $1 AND status = 'assigned'
	`, customerID)
	if err != nil {
		return nil, fmt.Errorf("collect affected pools: %w", err)
	}
	var affectedPools []string
	for affectedRows.Next() {
		var pid string
		if err := affectedRows.Scan(&pid); err != nil {
			affectedRows.Close()
			return nil, err
		}
		affectedPools = append(affectedPools, pid)
	}
	affectedRows.Close()

	// Release ALL previously assigned IPs for this customer across ALL pools.
	_, err = tx.Exec(ctx, `
		UPDATE ip_addresses SET customer_id = NULL, status = 'available', assigned_at = NULL, updated_at = $1
		WHERE customer_id = $2 AND status = 'assigned'
	`, now, customerID)
	if err != nil {
		return nil, fmt.Errorf("release old ips: %w", err)
	}

	// Update used_ips counters for all previously affected pools.
	for _, pid := range affectedPools {
		_, _ = tx.Exec(ctx, `
			UPDATE ip_pools SET used_ips = (
				SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1 AND status = 'assigned'
			) WHERE id = $1
		`, pid)
	}

	// Find and lock the next available IP atomically
	// Exclude IPs currently in use by active RADIUS sessions to prevent duplicates
	var a model.IPAddress
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, pool_id, address, customer_id, status, notes, assigned_at, created_at, updated_at
		FROM ip_addresses
		WHERE tenant_id = $1 AND pool_id = $2 AND status = 'available'
		  AND address NOT IN (SELECT framed_ip FROM radius_sessions WHERE status = 'active' AND framed_ip IS NOT NULL AND framed_ip != '')
		ORDER BY inet(address)
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, tenantID, poolID).Scan(
		&a.ID, &a.TenantID, &a.PoolID, &a.Address, &a.CustomerID,
		&a.Status, &a.Notes, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No available IPs; commit the release and return nil
			_ = tx.Commit(ctx)
			return nil, nil
		}
		return nil, fmt.Errorf("find available: %w", err)
	}

	// Assign the IP to the customer
	_, err = tx.Exec(ctx, `
		UPDATE ip_addresses SET customer_id = $1, status = 'assigned', assigned_at = $2, updated_at = $2
		WHERE id = $3
	`, customerID, now, a.ID)
	if err != nil {
		return nil, fmt.Errorf("assign ip: %w", err)
	}

	// Update pool counter
	_, _ = tx.Exec(ctx, `
		UPDATE ip_pools SET used_ips = (
			SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1 AND status = 'assigned'
		) WHERE id = $1
	`, poolID)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	a.CustomerID = &customerID
	a.Status = "assigned"
	a.AssignedAt = &now
	return &a, nil
}

func (r *ipamRepository) FindPoolByRouterID(ctx context.Context, routerID string) (*model.IPPool, error) {
	query := `
		SELECT id, tenant_id, name, network, COALESCE(gateway,''), COALESCE(dns_primary,''), COALESCE(dns_secondary,''),
		       router_id, total_ips, used_ips, COALESCE(notes,''), created_at, updated_at
		FROM ip_pools WHERE router_id = $1
		LIMIT 1
	`
	var p model.IPPool
	err := r.db.QueryRow(ctx, query, routerID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Network, &p.Gateway, &p.DNSPrimary, &p.DNSSecondary,
		&p.RouterID, &p.TotalIPs, &p.UsedIPs, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
