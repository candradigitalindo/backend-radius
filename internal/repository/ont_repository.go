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

type ONTRepository interface {
	Create(ctx context.Context, ont *model.ONT) error
	FindByID(ctx context.Context, tenantID, ontID string) (*model.ONT, error)
	Update(ctx context.Context, ont *model.ONT) error
	Delete(ctx context.Context, tenantID, ontID string) error
	List(ctx context.Context, tenantID string, filter ONTFilter) ([]model.ONT, int, error)
	FindByCustomerID(ctx context.Context, tenantID, customerID string) (*model.ONT, error)
	FindBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*model.ONT, error)
	FindByODPPortID(ctx context.Context, odpPortID string) (*model.ONT, error)
	UpdateCustomerByODPPortID(ctx context.Context, odpPortID string, customerID *string) error
	// FindUnprovisioned returns all active ONTs with a linked customer but not yet provisioned via GenieACS.
	FindUnprovisioned(ctx context.Context) ([]model.ONT, error)
	// FindAllSerialNumbers returns a set of all serial numbers for a tenant (used for discovery deduplication).
	FindAllSerialNumbers(ctx context.Context, tenantID string) (map[string]bool, error)
	// FindAllSerialNumbersGlobal returns all known serial numbers across ALL tenants.
	// Used so discovery never creates duplicate ONTs regardless of which tenant triggers the scan.
	FindAllSerialNumbersGlobal(ctx context.Context) (map[string]bool, error)
	// FindUnlinked returns all ONTs without a customer_id for a given tenant (discovered but not yet assigned).
	FindUnlinked(ctx context.Context, tenantID string) ([]model.ONT, error)
	// FindUnlinkedGlobal returns all ONTs without a customer_id across ALL tenants.
	FindUnlinkedGlobal(ctx context.Context) ([]model.ONT, error)
	// LinkToCustomer assigns an ONT to a customer and updates its tenant to match the customer.
	// Does NOT filter by tenant_id in WHERE — safe to call even when tenant may differ.
	LinkToCustomer(ctx context.Context, ontID, customerID, tenantID string) error
}

type ONTFilter struct {
	Search     string
	Status     string
	CustomerID string
	Page       int
	PerPage    int
}

type ontRepository struct {
	db *pgxpool.Pool
}

func NewONTRepository(db *pgxpool.Pool) ONTRepository {
	return &ontRepository{db: db}
}

func (r *ontRepository) Create(ctx context.Context, ont *model.ONT) error {
	ont.ID = id.New()
	now := time.Now()
	ont.CreatedAt = now
	ont.UpdatedAt = now

	query := `
		INSERT INTO onts (
			id, tenant_id, customer_id, odp_port_id, serial_number, model, vendor,
			mac_address, ip_address, last_known_wifi_ssid, last_known_wifi_password,
			last_known_wifi_security, last_known_wifi_source, last_known_wifi_updated_at,
			last_known_connected_hosts, last_known_connected_host_count,
			last_known_connected_hosts_source, last_known_connected_hosts_updated_at,
			rx_power, tx_power, status, notes, last_online_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
	`

	_, err := r.db.Exec(ctx, query,
		ont.ID, ont.TenantID, ont.CustomerID, ont.ODPPortID,
		ont.SerialNumber, ont.Model, ont.Vendor,
		ont.MACAddress, ont.IPAddress, ont.LastKnownWiFiSSID, ont.LastKnownWiFiPassword,
		ont.LastKnownWiFiSecurity, ont.LastKnownWiFiSource, ont.LastKnownWiFiUpdatedAt,
		ont.LastKnownConnectedHosts, ont.LastKnownConnectedHostCount,
		ont.LastKnownConnectedHostsSource, ont.LastKnownConnectedHostsUpdatedAt,
		ont.RxPower, ont.TxPower, ont.Status, ont.Notes,
		ont.LastOnlineAt, ont.CreatedAt, ont.UpdatedAt,
	)
	return err
}

func (r *ontRepository) FindByID(ctx context.Context, tenantID, ontID string) (*model.ONT, error) {
	query := `
		SELECT o.id, o.tenant_id, o.customer_id, o.odp_port_id, o.serial_number,
		       o.model, o.vendor, o.mac_address, o.ip_address, o.last_known_wifi_ssid,
		       o.last_known_wifi_password, o.last_known_wifi_security, o.last_known_wifi_source,
		       o.last_known_wifi_updated_at, o.last_known_connected_hosts,
		       o.last_known_connected_host_count, o.last_known_connected_hosts_source,
		       o.last_known_connected_hosts_updated_at, o.rx_power, o.tx_power,
		       o.status, o.notes, o.last_online_at, o.provisioned_at, o.created_at, o.updated_at,
		       c.id, c.tenant_id, c.customer_code, c.name, c.nik, c.phone, c.email, c.address,
		       c.latitude, c.longitude, c.connection_type, c.pppoe_username, c.pppoe_password,
		       c.ip_address, c.package_id, c.router_id, c.odp_port_id, c.join_date,
		       c.billing_date, c.billing_type, c.billing_deadline, c.custom_price,
		       c.discount, c.additional_fee, c.fee_description, c.status,
		       c.isolated_at, c.notes, c.created_at, c.updated_at
		FROM onts o
		LEFT JOIN customers c ON c.id = o.customer_id
		WHERE o.id = $1 AND o.tenant_id = $2
		LIMIT 1
	`

	var ont model.ONT
	var cust model.Customer
	var custID *string
	err := r.db.QueryRow(ctx, query, ontID, tenantID).Scan(
		&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
		&ont.SerialNumber, &ont.Model, &ont.Vendor,
		&ont.MACAddress, &ont.IPAddress, &ont.LastKnownWiFiSSID,
		&ont.LastKnownWiFiPassword, &ont.LastKnownWiFiSecurity, &ont.LastKnownWiFiSource,
		&ont.LastKnownWiFiUpdatedAt, &ont.LastKnownConnectedHosts,
		&ont.LastKnownConnectedHostCount, &ont.LastKnownConnectedHostsSource,
		&ont.LastKnownConnectedHostsUpdatedAt, &ont.RxPower, &ont.TxPower,
		&ont.Status, &ont.Notes, &ont.LastOnlineAt, &ont.ProvisionedAt,
		&ont.CreatedAt, &ont.UpdatedAt,
		&custID, &cust.TenantID, &cust.CustomerCode, &cust.Name, &cust.NIK,
		&cust.Phone, &cust.Email, &cust.Address, &cust.Latitude, &cust.Longitude,
		&cust.ConnectionType, &cust.PPPoEUsername, &cust.PPPoEPassword,
		&cust.IPAddress, &cust.PackageID, &cust.RouterID, &cust.ODPPortID,
		&cust.JoinDate, &cust.BillingDate, &cust.BillingType, &cust.BillingDeadline,
		&cust.CustomPrice, &cust.Discount, &cust.AdditionalFee, &cust.FeeDescription,
		&cust.Status, &cust.IsolatedAt, &cust.Notes, &cust.CreatedAt, &cust.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if custID != nil {
		cust.ID = *custID
		ont.Customer = &cust
	}
	return &ont, nil
}

func (r *ontRepository) Update(ctx context.Context, ont *model.ONT) error {
	ont.UpdatedAt = time.Now()

	query := `
		UPDATE onts SET
			customer_id = $1, odp_port_id = $2, serial_number = $3, model = $4,
			vendor = $5, mac_address = $6, ip_address = $7, last_known_wifi_ssid = $8,
			last_known_wifi_password = $9, last_known_wifi_security = $10,
			last_known_wifi_source = $11, last_known_wifi_updated_at = $12,
			last_known_connected_hosts = $13, last_known_connected_host_count = $14,
			last_known_connected_hosts_source = $15, last_known_connected_hosts_updated_at = $16,
			rx_power = $17, tx_power = $18, status = $19, notes = $20,
			last_online_at = $21, provisioned_at = $22, updated_at = $23
		WHERE id = $24 AND tenant_id = $25
	`

	_, err := r.db.Exec(ctx, query,
		ont.CustomerID, ont.ODPPortID, ont.SerialNumber, ont.Model,
		ont.Vendor, ont.MACAddress, ont.IPAddress, ont.LastKnownWiFiSSID,
		ont.LastKnownWiFiPassword, ont.LastKnownWiFiSecurity, ont.LastKnownWiFiSource,
		ont.LastKnownWiFiUpdatedAt, ont.LastKnownConnectedHosts, ont.LastKnownConnectedHostCount,
		ont.LastKnownConnectedHostsSource, ont.LastKnownConnectedHostsUpdatedAt,
		ont.RxPower, ont.TxPower, ont.Status, ont.Notes,
		ont.LastOnlineAt, ont.ProvisionedAt, ont.UpdatedAt,
		ont.ID, ont.TenantID,
	)
	return err
}

func (r *ontRepository) Delete(ctx context.Context, tenantID, ontID string) error {
	query := `DELETE FROM onts WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, ontID, tenantID)
	return err
}

func (r *ontRepository) List(ctx context.Context, tenantID string, filter ONTFilter) ([]model.ONT, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("o.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(o.serial_number ILIKE $%d OR o.mac_address ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.CustomerID != "" {
		conditions = append(conditions, fmt.Sprintf("o.customer_id = $%d", argIdx))
		args = append(args, filter.CustomerID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM onts o LEFT JOIN customers c ON c.id = o.customer_id %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	dataQuery := fmt.Sprintf(`
		SELECT o.id, o.tenant_id, o.customer_id, o.odp_port_id, o.serial_number,
		       o.model, o.vendor, o.mac_address, o.ip_address, o.last_known_wifi_ssid,
		       o.last_known_wifi_password, o.last_known_wifi_security, o.last_known_wifi_source,
		       o.last_known_wifi_updated_at, o.last_known_connected_hosts,
		       o.last_known_connected_host_count, o.last_known_connected_hosts_source,
		       o.last_known_connected_hosts_updated_at, o.rx_power, o.tx_power,
		       o.status, o.notes, o.last_online_at, o.provisioned_at, o.created_at, o.updated_at,
		       c.id, c.tenant_id, c.customer_code, c.name, c.nik, c.phone, c.email, c.address,
		       c.latitude, c.longitude, c.connection_type, c.pppoe_username, c.pppoe_password,
		       c.ip_address, c.package_id, c.router_id, c.odp_port_id, c.join_date,
		       c.billing_date, c.billing_type, c.billing_deadline, c.custom_price,
		       c.discount, c.additional_fee, c.fee_description, c.status,
		       c.isolated_at, c.notes, c.created_at, c.updated_at
		FROM onts o
		LEFT JOIN customers c ON c.id = o.customer_id
		%s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var onts []model.ONT
	for rows.Next() {
		var ont model.ONT
		var cust model.Customer
		var custID *string
		if err := rows.Scan(
			&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
			&ont.SerialNumber, &ont.Model, &ont.Vendor,
			&ont.MACAddress, &ont.IPAddress, &ont.LastKnownWiFiSSID,
			&ont.LastKnownWiFiPassword, &ont.LastKnownWiFiSecurity, &ont.LastKnownWiFiSource,
			&ont.LastKnownWiFiUpdatedAt, &ont.LastKnownConnectedHosts,
			&ont.LastKnownConnectedHostCount, &ont.LastKnownConnectedHostsSource,
			&ont.LastKnownConnectedHostsUpdatedAt, &ont.RxPower, &ont.TxPower,
			&ont.Status, &ont.Notes, &ont.LastOnlineAt, &ont.ProvisionedAt,
			&ont.CreatedAt, &ont.UpdatedAt,
			&custID, &cust.TenantID, &cust.CustomerCode, &cust.Name, &cust.NIK,
			&cust.Phone, &cust.Email, &cust.Address, &cust.Latitude, &cust.Longitude,
			&cust.ConnectionType, &cust.PPPoEUsername, &cust.PPPoEPassword,
			&cust.IPAddress, &cust.PackageID, &cust.RouterID, &cust.ODPPortID,
			&cust.JoinDate, &cust.BillingDate, &cust.BillingType, &cust.BillingDeadline,
			&cust.CustomPrice, &cust.Discount, &cust.AdditionalFee, &cust.FeeDescription,
			&cust.Status, &cust.IsolatedAt, &cust.Notes, &cust.CreatedAt, &cust.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if custID != nil {
			cust.ID = *custID
			ont.Customer = &cust
		}
		onts = append(onts, ont)
	}

	return onts, total, nil
}

func (r *ontRepository) FindByCustomerID(ctx context.Context, tenantID, customerID string) (*model.ONT, error) {
	query := `
		SELECT id, tenant_id, customer_id, odp_port_id, serial_number, model, vendor,
		       mac_address, ip_address, last_known_wifi_ssid, last_known_wifi_password,
		       last_known_wifi_security, last_known_wifi_source, last_known_wifi_updated_at,
		       last_known_connected_hosts, last_known_connected_host_count,
		       last_known_connected_hosts_source, last_known_connected_hosts_updated_at,
		       rx_power, tx_power, status, notes, last_online_at, provisioned_at,
		       created_at, updated_at
		FROM onts
		WHERE customer_id = $1 AND tenant_id = $2
		LIMIT 1
	`

	var ont model.ONT
	err := r.db.QueryRow(ctx, query, customerID, tenantID).Scan(
		&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
		&ont.SerialNumber, &ont.Model, &ont.Vendor,
		&ont.MACAddress, &ont.IPAddress, &ont.LastKnownWiFiSSID,
		&ont.LastKnownWiFiPassword, &ont.LastKnownWiFiSecurity, &ont.LastKnownWiFiSource,
		&ont.LastKnownWiFiUpdatedAt, &ont.LastKnownConnectedHosts,
		&ont.LastKnownConnectedHostCount, &ont.LastKnownConnectedHostsSource,
		&ont.LastKnownConnectedHostsUpdatedAt, &ont.RxPower, &ont.TxPower,
		&ont.Status, &ont.Notes, &ont.LastOnlineAt, &ont.ProvisionedAt,
		&ont.CreatedAt, &ont.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ont, nil
}

func (r *ontRepository) FindBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*model.ONT, error) {
	query := `
		SELECT id, tenant_id, customer_id, odp_port_id, serial_number, model, vendor,
		       mac_address, ip_address, last_known_wifi_ssid, last_known_wifi_password,
		       last_known_wifi_security, last_known_wifi_source, last_known_wifi_updated_at,
		       last_known_connected_hosts, last_known_connected_host_count,
		       last_known_connected_hosts_source, last_known_connected_hosts_updated_at,
		       rx_power, tx_power, status, notes, last_online_at, provisioned_at,
		       created_at, updated_at
		FROM onts
		WHERE serial_number = $1 AND tenant_id = $2
		LIMIT 1
	`

	var ont model.ONT
	err := r.db.QueryRow(ctx, query, serialNumber, tenantID).Scan(
		&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
		&ont.SerialNumber, &ont.Model, &ont.Vendor,
		&ont.MACAddress, &ont.IPAddress, &ont.LastKnownWiFiSSID,
		&ont.LastKnownWiFiPassword, &ont.LastKnownWiFiSecurity, &ont.LastKnownWiFiSource,
		&ont.LastKnownWiFiUpdatedAt, &ont.LastKnownConnectedHosts,
		&ont.LastKnownConnectedHostCount, &ont.LastKnownConnectedHostsSource,
		&ont.LastKnownConnectedHostsUpdatedAt, &ont.RxPower, &ont.TxPower,
		&ont.Status, &ont.Notes, &ont.LastOnlineAt, &ont.ProvisionedAt,
		&ont.CreatedAt, &ont.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ont, nil
}

func (r *ontRepository) FindByODPPortID(ctx context.Context, odpPortID string) (*model.ONT, error) {
	query := `
		SELECT id, tenant_id, customer_id, odp_port_id, serial_number, model, vendor,
		       mac_address, ip_address, last_known_wifi_ssid, last_known_wifi_password,
		       last_known_wifi_security, last_known_wifi_source, last_known_wifi_updated_at,
		       last_known_connected_hosts, last_known_connected_host_count,
		       last_known_connected_hosts_source, last_known_connected_hosts_updated_at,
		       rx_power, tx_power, status, notes, last_online_at, provisioned_at,
		       created_at, updated_at
		FROM onts
		WHERE odp_port_id = $1
		LIMIT 1
	`

	var ont model.ONT
	err2 := r.db.QueryRow(ctx, query, odpPortID).Scan(
		&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
		&ont.SerialNumber, &ont.Model, &ont.Vendor,
		&ont.MACAddress, &ont.IPAddress, &ont.LastKnownWiFiSSID,
		&ont.LastKnownWiFiPassword, &ont.LastKnownWiFiSecurity, &ont.LastKnownWiFiSource,
		&ont.LastKnownWiFiUpdatedAt, &ont.LastKnownConnectedHosts,
		&ont.LastKnownConnectedHostCount, &ont.LastKnownConnectedHostsSource,
		&ont.LastKnownConnectedHostsUpdatedAt, &ont.RxPower, &ont.TxPower,
		&ont.Status, &ont.Notes, &ont.LastOnlineAt, &ont.ProvisionedAt,
		&ont.CreatedAt, &ont.UpdatedAt,
	)
	if err2 != nil {
		if err2 == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err2
	}
	return &ont, nil
}

func (r *ontRepository) UpdateCustomerByODPPortID(ctx context.Context, odpPortID string, customerID *string) error {
	query := `UPDATE onts SET customer_id = $1, updated_at = NOW() WHERE odp_port_id = $2`
	_, err := r.db.Exec(ctx, query, customerID, odpPortID)
	return err
}

// FindUnprovisioned returns all active ONTs linked to a customer but not yet provisioned via GenieACS.
func (r *ontRepository) FindUnprovisioned(ctx context.Context) ([]model.ONT, error) {
	query := `
		SELECT id, tenant_id, serial_number
		FROM onts
		WHERE customer_id IS NOT NULL
		  AND provisioned_at IS NULL
		  AND status = 'active'
		ORDER BY created_at ASC
		LIMIT 200
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var onts []model.ONT
	for rows.Next() {
		var ont model.ONT
		if err := rows.Scan(&ont.ID, &ont.TenantID, &ont.SerialNumber); err != nil {
			return nil, err
		}
		onts = append(onts, ont)
	}
	return onts, nil
}

// FindAllSerialNumbers returns a set of all serial numbers registered for a tenant.
// Used for O(1) deduplication during auto-discovery.
func (r *ontRepository) FindAllSerialNumbers(ctx context.Context, tenantID string) (map[string]bool, error) {
	query := `SELECT serial_number FROM onts WHERE tenant_id = $1`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var serial string
		if err := rows.Scan(&serial); err != nil {
			return nil, err
		}
		result[serial] = true
	}
	return result, nil
}

// FindUnlinked returns ONTs that have no customer_id yet (status="discovered").
// Used for auto-matching by PPPoE username.
func (r *ontRepository) FindUnlinked(ctx context.Context, tenantID string) ([]model.ONT, error) {
	query := `
		SELECT id, tenant_id, serial_number
		FROM onts
		WHERE tenant_id = $1 AND customer_id IS NULL
		ORDER BY created_at ASC
		LIMIT 500
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var onts []model.ONT
	for rows.Next() {
		var ont model.ONT
		if err := rows.Scan(&ont.ID, &ont.TenantID, &ont.SerialNumber); err != nil {
			return nil, err
		}
		onts = append(onts, ont)
	}
	return onts, nil
}

// FindAllSerialNumbersGlobal returns all known serial numbers across ALL tenants.
// Prevents duplicate ONT entries when the same device is discovered by multiple tenants.
func (r *ontRepository) FindAllSerialNumbersGlobal(ctx context.Context) (map[string]bool, error) {
	query := `SELECT serial_number FROM onts`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var serial string
		if err := rows.Scan(&serial); err != nil {
			return nil, err
		}
		result[serial] = true
	}
	return result, nil
}

// FindUnlinkedGlobal returns all ONTs without a customer_id across ALL tenants.
// Used by global AutoMatchONTs so cross-tenant ONTs are not missed.
func (r *ontRepository) FindUnlinkedGlobal(ctx context.Context) ([]model.ONT, error) {
	query := `
		SELECT id, tenant_id, serial_number
		FROM onts
		WHERE customer_id IS NULL
		ORDER BY created_at ASC
		LIMIT 500
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var onts []model.ONT
	for rows.Next() {
		var ont model.ONT
		if err := rows.Scan(&ont.ID, &ont.TenantID, &ont.SerialNumber); err != nil {
			return nil, err
		}
		onts = append(onts, ont)
	}
	return onts, nil
}

// LinkToCustomer assigns an ONT to a customer and corrects its tenant_id to match the customer.
// Does NOT use tenant_id in the WHERE clause — safe even when the ONT was discovered under a
// different tenant (which can happen when discovery races across tenants).
func (r *ontRepository) LinkToCustomer(ctx context.Context, ontID, customerID, tenantID string) error {
	query := `
		UPDATE onts
		SET customer_id = $1, tenant_id = $2, status = 'active', updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, customerID, tenantID, ontID)
	return err
}
