package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type CustomerRepository interface {
	FindByPPPoEUsername(ctx context.Context, tenantID, username string) (*model.Customer, error)
	FindByPPPoEUsernameGlobal(ctx context.Context, username string) (*model.Customer, error)
	FindByCode(ctx context.Context, tenantID, customerCode string) (*model.Customer, error)
	FindByCodeGlobal(ctx context.Context, customerCode string) ([]model.Customer, error)
	FindByPhone(ctx context.Context, tenantID, phone string) (*model.Customer, error)
	FindByEmail(ctx context.Context, tenantID, email string) (*model.Customer, error)
	UpdateStatus(ctx context.Context, tenantID, id, status string) error
	Create(ctx context.Context, customer *model.Customer) error
	FindByID(ctx context.Context, tenantID, customerID string) (*model.Customer, error)
	Update(ctx context.Context, customer *model.Customer) error
	Delete(ctx context.Context, tenantID, customerID string) error
	List(ctx context.Context, tenantID string, filter CustomerFilter) ([]model.Customer, int, error)
	SetIsolated(ctx context.Context, tenantID, customerID string, isolatedAt *time.Time) error
	CountByCodePrefix(ctx context.Context, tenantID, prefix string) (int, error)
	CountByPPPoEPrefix(ctx context.Context, tenantID, prefix string) (int, error)
	CountActive(ctx context.Context, tenantID string) (int, error)
	UpdateODPPortID(ctx context.Context, customerID string, odpPortID *string) error
	ClearODPPortID(ctx context.Context, odpPortID string) error
	UpdatePasswordHash(ctx context.Context, tenantID, customerID, hash string) error
	FindByReferralCode(ctx context.Context, tenantID, referralCode string) (*model.Customer, error)
	// FindByActiveSessionIP returns the customer that currently has an active RADIUS session
	// with the given framed_ip. Used as fallback when GenieACS doesn't store PPPoE username.
	FindByActiveSessionIP(ctx context.Context, ip string) (*model.Customer, error)
}

type CustomerFilter struct {
	Search   string
	Status   string
	RouterID string
	Page     int
	PerPage  int
}

type customerRepository struct {
	db *pgxpool.Pool
}

func NewCustomerRepository(db *pgxpool.Pool) CustomerRepository {
	return &customerRepository{db: db}
}

func generateCustomerReferralCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (r *customerRepository) Create(ctx context.Context, customer *model.Customer) error {
	customer.ID = id.New()
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now
	if customer.ReferralCode == "" {
		customer.ReferralCode = generateCustomerReferralCode()
	}

	query := `
		INSERT INTO customers (
			id, tenant_id, customer_code, name, nik, phone, email, address,
			latitude, longitude, connection_type, pppoe_username, pppoe_password,
			ip_address, package_id, router_id, odp_port_id, join_date, billing_date,
			billing_type, billing_deadline, billing_profile_id,
			pending_billing_profile_id, previous_package_price, package_changed_at,
			custom_price, discount, additional_fee, fee_description, status, notes,
			referral_code, created_at, updated_at, reseller_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21, $22,
			$23, $24, $25,
			$26, $27, $28, $29, $30, $31,
			$32, $33, $34, $35
		)
	`

	_, err := r.db.Exec(ctx, query,
		customer.ID, customer.TenantID, customer.CustomerCode, customer.Name,
		customer.NIK, customer.Phone, customer.Email, customer.Address,
		customer.Latitude, customer.Longitude, customer.ConnectionType,
		customer.PPPoEUsername, customer.PPPoEPassword,
		customer.IPAddress, customer.PackageID, customer.RouterID, customer.ODPPortID,
		customer.JoinDate, customer.BillingDate,
		customer.BillingType, customer.BillingDeadline, customer.BillingProfileID,
		customer.PendingBillingProfileID, customer.PreviousPackagePrice, customer.PackageChangedAt,
		customer.CustomPrice, customer.Discount, customer.AdditionalFee,
		customer.FeeDescription, customer.Status, customer.Notes,
		customer.ReferralCode,
		customer.CreatedAt, customer.UpdatedAt, customer.ResellerID,
	)
	return err
}

func (r *customerRepository) FindByID(ctx context.Context, tenantID, customerID string) (*model.Customer, error) {
	query := `
		SELECT c.id, c.tenant_id, c.customer_code, c.name, COALESCE(c.nik,''), COALESCE(c.phone,''), COALESCE(c.email,''),
		       COALESCE(c.address,''), c.latitude, c.longitude,
		       c.connection_type, COALESCE(c.pppoe_username,''), COALESCE(c.pppoe_password,''), COALESCE(c.ip_address,''),
		       c.package_id, c.router_id, c.odp_port_id, c.join_date, c.billing_date,
		       COALESCE(c.billing_type,'fixed'), COALESCE(c.billing_deadline,20), c.billing_profile_id,
		       c.pending_billing_profile_id, COALESCE(c.previous_package_price,0), c.package_changed_at,
		       c.custom_price, c.discount, c.additional_fee, COALESCE(c.fee_description,''),
		       c.status, c.isolated_at, COALESCE(c.notes,''), COALESCE(c.password_hash,''),
		       COALESCE(c.referral_code,''), c.created_at, c.updated_at, c.reseller_id,
		       p.id, p.name, p.bandwidth_up, p.bandwidth_down, p.price,
		       p.burst_limit, p.address_list,
		       rt.id, rt.name, rt.vpn_ip, COALESCE(rt.vpn_public_key,''), COALESCE(rt.legacy_vpn_ip,''), COALESCE(rt.nas_ip,''), rt.radius_secret, rt.coa_port,
		       dp.odp_id, dp.port_number,
		       EXISTS(SELECT 1 FROM radius_sessions rs WHERE rs.customer_id = c.id AND rs.status = 'active') AS is_online
		FROM customers c
		LEFT JOIN packages p ON c.package_id = p.id
		LEFT JOIN routers rt ON c.router_id = rt.id
		LEFT JOIN odp_ports dp ON c.odp_port_id = dp.id
		WHERE c.id = $1 AND c.tenant_id = $2
		LIMIT 1
	`

	var c model.Customer
	var pkgID, rtID, odpPortID *string
	var pkgJoinID, pkgName, pkgBurstLimit, pkgAddressList *string
	var pkgBandwidthUp, pkgBandwidthDown *int
	var pkgPrice *int64
	var rtJoinID, rtName, rtVPNIP, rtWGKey, rtLegacyIP, rtNASIP, rtRADIUSSecret *string
	var rtCoAPort *int
	var dpODPID *string
	var dpPortNumber *int
	var isOnline bool

	err := r.db.QueryRow(ctx, query, customerID, tenantID).Scan(
		&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.NIK, &c.Phone, &c.Email,
		&c.Address, &c.Latitude, &c.Longitude,
		&c.ConnectionType, &c.PPPoEUsername, &c.PPPoEPassword, &c.IPAddress,
		&pkgID, &rtID, &odpPortID, &c.JoinDate, &c.BillingDate,
		&c.BillingType, &c.BillingDeadline, &c.BillingProfileID,
		&c.PendingBillingProfileID, &c.PreviousPackagePrice, &c.PackageChangedAt,
		&c.CustomPrice, &c.Discount, &c.AdditionalFee, &c.FeeDescription,
		&c.Status, &c.IsolatedAt, &c.Notes, &c.PasswordHash,
		&c.ReferralCode, &c.CreatedAt, &c.UpdatedAt, &c.ResellerID,
		&pkgJoinID, &pkgName, &pkgBandwidthUp, &pkgBandwidthDown, &pkgPrice,
		&pkgBurstLimit, &pkgAddressList,
		&rtJoinID, &rtName, &rtVPNIP, &rtWGKey, &rtLegacyIP, &rtNASIP, &rtRADIUSSecret, &rtCoAPort,
		&dpODPID, &dpPortNumber,
		&isOnline,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	c.PackageID = pkgID
	c.RouterID = rtID
	c.ODPPortID = odpPortID
	if odpPortID != nil && dpODPID != nil {
		c.ODPPort = &model.ODPPort{ID: *odpPortID, ODPID: *dpODPID}
		if dpPortNumber != nil {
			c.ODPPort.PortNumber = *dpPortNumber
		}
	}

	if c.Status == "isolated" {
		c.ConnectionStatus = "isolated"
	} else if isOnline {
		c.ConnectionStatus = "online"
	} else {
		c.ConnectionStatus = "offline"
	}

	if pkgID != nil && pkgJoinID != nil {
		c.Package = &model.Package{ID: *pkgJoinID}
		if pkgName != nil {
			c.Package.Name = *pkgName
		}
		if pkgBandwidthUp != nil {
			c.Package.BandwidthUp = *pkgBandwidthUp
		}
		if pkgBandwidthDown != nil {
			c.Package.BandwidthDown = *pkgBandwidthDown
		}
		if pkgPrice != nil {
			c.Package.Price = *pkgPrice
		}
		if pkgBurstLimit != nil {
			c.Package.BurstLimit = *pkgBurstLimit
		}
		if pkgAddressList != nil {
			c.Package.AddressList = *pkgAddressList
		}
	}
	if rtID != nil && rtJoinID != nil {
		c.Router = &model.Router{ID: *rtJoinID}
		if rtName != nil {
			c.Router.Name = *rtName
		}
		if rtVPNIP != nil {
			c.Router.VPNIP = *rtVPNIP
		}
		if rtWGKey != nil {
			c.Router.VPNPublicKey = *rtWGKey
		}
		if rtLegacyIP != nil {
			c.Router.LegacyVPNIP = *rtLegacyIP
		}
		if rtNASIP != nil {
			c.Router.NASIP = *rtNASIP
		}
		if rtRADIUSSecret != nil {
			c.Router.RADIUSSecret = *rtRADIUSSecret
		}
		if rtCoAPort != nil {
			c.Router.CoAPort = *rtCoAPort
		}
	}

	return &c, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *model.Customer) error {
	customer.UpdatedAt = time.Now()
	query := `
		UPDATE customers SET
			customer_code = $1, name = $2, nik = $3, phone = $4, email = $5, address = $6,
			latitude = $7, longitude = $8, connection_type = $9,
			pppoe_username = $10, pppoe_password = $11, ip_address = $12,
			package_id = $13, router_id = $14, odp_port_id = $15,
			join_date = $16, billing_date = $17, billing_type = $18, billing_deadline = $19,
			billing_profile_id = $20, pending_billing_profile_id = $21,
			previous_package_price = $22, package_changed_at = $23,
			custom_price = $24, discount = $25,
			additional_fee = $26, fee_description = $27, notes = $28,
			updated_at = $29, reseller_id = $30
		WHERE id = $31 AND tenant_id = $32
	`
	_, err := r.db.Exec(ctx, query,
		customer.CustomerCode, customer.Name, customer.NIK, customer.Phone, customer.Email, customer.Address,
		customer.Latitude, customer.Longitude, customer.ConnectionType,
		customer.PPPoEUsername, customer.PPPoEPassword, customer.IPAddress,
		customer.PackageID, customer.RouterID, customer.ODPPortID,
		customer.JoinDate, customer.BillingDate, customer.BillingType, customer.BillingDeadline,
		customer.BillingProfileID, customer.PendingBillingProfileID,
		customer.PreviousPackagePrice, customer.PackageChangedAt,
		customer.CustomPrice, customer.Discount,
		customer.AdditionalFee, customer.FeeDescription, customer.Notes,
		customer.UpdatedAt, customer.ResellerID,
		customer.ID, customer.TenantID,
	)
	return err
}

func (r *customerRepository) Delete(ctx context.Context, tenantID, customerID string) error {
	query := `DELETE FROM customers WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, customerID, tenantID)
	return err
}

func (r *customerRepository) List(ctx context.Context, tenantID string, filter CustomerFilter) ([]model.Customer, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1
	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(c.name ILIKE $%d OR c.customer_code ILIKE $%d OR c.pppoe_username ILIKE $%d OR c.phone ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.RouterID != "" {
		conditions = append(conditions, fmt.Sprintf("c.router_id = $%d", argIdx))
		args = append(args, filter.RouterID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	countQuery := "SELECT COUNT(*) FROM customers c " + where
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
		SELECT c.id, c.tenant_id, c.customer_code, c.name, COALESCE(c.nik,''), COALESCE(c.phone,''), COALESCE(c.email,''),
		       COALESCE(c.address,''), c.latitude, c.longitude,
		       c.connection_type, COALESCE(c.pppoe_username,''), COALESCE(c.pppoe_password,''), COALESCE(c.ip_address,''),
		       c.package_id, c.router_id, c.odp_port_id, c.join_date, c.billing_date,
		       COALESCE(c.billing_type,'fixed'), COALESCE(c.billing_deadline,20), c.billing_profile_id,
		       c.pending_billing_profile_id, COALESCE(c.previous_package_price,0), c.package_changed_at,
		       c.custom_price, c.discount, c.additional_fee, COALESCE(c.fee_description,''),
		       c.status, c.isolated_at, COALESCE(c.notes,''), c.created_at, c.updated_at,
		       p.id, p.name, p.bandwidth_up, p.bandwidth_down, p.price,
		       p.burst_limit, p.address_list, p.is_active,
		       rt.id, rt.name, rt.vpn_ip, rt.radius_secret, rt.coa_port, rt.is_active,
		       EXISTS(SELECT 1 FROM radius_sessions rs WHERE rs.customer_id = c.id AND rs.status = 'active') AS is_online
		FROM customers c
		LEFT JOIN packages p ON c.package_id = p.id
		LEFT JOIN routers rt ON c.router_id = rt.id
		%s
		ORDER BY c.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var c model.Customer
		var pkgID, rtID, odpPortID *string
		var pkgJoinID, pkgName, pkgBurstLimit, pkgAddressList *string
		var pkgBandwidthUp, pkgBandwidthDown *int
		var pkgPrice *int64
		var pkgIsActive *bool
		var rtJoinID, rtName, rtVPNIP, rtRADIUSSecret *string
		var rtCoAPort *int
		var rtIsActive *bool
		var isOnline bool
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.NIK, &c.Phone, &c.Email,
			&c.Address, &c.Latitude, &c.Longitude,
			&c.ConnectionType, &c.PPPoEUsername, &c.PPPoEPassword, &c.IPAddress,
			&pkgID, &rtID, &odpPortID, &c.JoinDate, &c.BillingDate,
			&c.BillingType, &c.BillingDeadline, &c.BillingProfileID,
			&c.PendingBillingProfileID, &c.PreviousPackagePrice, &c.PackageChangedAt,
			&c.CustomPrice, &c.Discount, &c.AdditionalFee, &c.FeeDescription,
			&c.Status, &c.IsolatedAt, &c.Notes, &c.CreatedAt, &c.UpdatedAt,
			&pkgJoinID, &pkgName, &pkgBandwidthUp, &pkgBandwidthDown, &pkgPrice,
			&pkgBurstLimit, &pkgAddressList, &pkgIsActive,
			&rtJoinID, &rtName, &rtVPNIP, &rtRADIUSSecret, &rtCoAPort, &rtIsActive,
			&isOnline,
		); err != nil {
			return nil, 0, err
		}

		if c.Status == "isolated" {
			c.ConnectionStatus = "isolated"
		} else if isOnline {
			c.ConnectionStatus = "online"
		} else {
			c.ConnectionStatus = "offline"
		}
		c.PackageID = pkgID
		c.RouterID = rtID
		c.ODPPortID = odpPortID
		if pkgID != nil && pkgJoinID != nil {
			c.Package = &model.Package{ID: *pkgJoinID}
			if pkgName != nil {
				c.Package.Name = *pkgName
			}
			if pkgBandwidthUp != nil {
				c.Package.BandwidthUp = *pkgBandwidthUp
			}
			if pkgBandwidthDown != nil {
				c.Package.BandwidthDown = *pkgBandwidthDown
			}
			if pkgPrice != nil {
				c.Package.Price = *pkgPrice
			}
			if pkgBurstLimit != nil {
				c.Package.BurstLimit = *pkgBurstLimit
			}
			if pkgAddressList != nil {
				c.Package.AddressList = *pkgAddressList
			}
			if pkgIsActive != nil {
				c.Package.IsActive = *pkgIsActive
			}
		}
		if rtID != nil && rtJoinID != nil {
			c.Router = &model.Router{ID: *rtJoinID}
			if rtName != nil {
				c.Router.Name = *rtName
			}
			if rtVPNIP != nil {
				c.Router.VPNIP = *rtVPNIP
			}
			if rtRADIUSSecret != nil {
				c.Router.RADIUSSecret = *rtRADIUSSecret
			}
			if rtCoAPort != nil {
				c.Router.CoAPort = *rtCoAPort
			}
			if rtIsActive != nil {
				c.Router.IsActive = *rtIsActive
			}
		}
		customers = append(customers, c)
	}
	return customers, total, nil
}

func (r *customerRepository) SetIsolated(ctx context.Context, tenantID, customerID string, isolatedAt *time.Time) error {
	status := "active"
	if isolatedAt != nil {
		status = "isolated"
	}
	query := `UPDATE customers SET status = $1, isolated_at = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`
	_, err := r.db.Exec(ctx, query, status, isolatedAt, customerID, tenantID)
	return err
}

func (r *customerRepository) FindByPPPoEUsername(ctx context.Context, tenantID, username string) (*model.Customer, error) {
	query := `
		SELECT c.id, c.tenant_id, c.customer_code, c.name, COALESCE(c.phone,''), COALESCE(c.email,''),
		       c.connection_type, COALESCE(c.pppoe_username,''), COALESCE(c.pppoe_password,''), COALESCE(c.ip_address,''),
		       c.package_id, c.router_id, c.status, c.isolated_at,
		       p.id, p.name, p.bandwidth_up, p.bandwidth_down, p.price,
		       p.burst_limit, p.address_list, p.is_active,
		       rt.id, rt.vpn_ip, COALESCE(rt.vpn_public_key,''), COALESCE(rt.legacy_vpn_ip,''), COALESCE(rt.nas_ip,''), rt.radius_secret, rt.coa_port
		FROM customers c
		LEFT JOIN packages p ON c.package_id = p.id
		LEFT JOIN routers rt ON c.router_id = rt.id
		WHERE c.pppoe_username = $1 AND c.tenant_id = $2
		LIMIT 1
	`
	var c model.Customer
	var pkgID, rtID, pkgName, pkgBurstLimit, pkgAddressList, rtVPNIP, rtWGKey, rtLegacyIP, rtNASIP, rtSecret *string
	var pkgBandwidthUp, pkgBandwidthDown, rtCoAPort *int
	var pkgPrice *int64
	var pkgIsActive *bool
	var pID, rID *string

	err := r.db.QueryRow(ctx, query, username, tenantID).Scan(
		&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.Phone, &c.Email,
		&c.ConnectionType, &c.PPPoEUsername, &c.PPPoEPassword, &c.IPAddress,
		&pID, &rID, &c.Status, &c.IsolatedAt,
		&pkgID, &pkgName, &pkgBandwidthUp, &pkgBandwidthDown, &pkgPrice,
		&pkgBurstLimit, &pkgAddressList, &pkgIsActive,
		&rtID, &rtVPNIP, &rtWGKey, &rtLegacyIP, &rtNASIP, &rtSecret, &rtCoAPort,
	)
	if err != nil {
		return nil, err
	}
	c.PackageID = pID
	c.RouterID = rID
	if pID != nil && pkgID != nil {
		c.Package = &model.Package{ID: *pkgID, Name: *pkgName, BandwidthUp: *pkgBandwidthUp, BandwidthDown: *pkgBandwidthDown, Price: *pkgPrice, BurstLimit: *pkgBurstLimit, AddressList: *pkgAddressList, IsActive: *pkgIsActive}
	}
	if rID != nil && rtID != nil {
		c.Router = &model.Router{ID: *rtID, VPNIP: *rtVPNIP, VPNPublicKey: *rtWGKey, LegacyVPNIP: *rtLegacyIP, NASIP: *rtNASIP, RADIUSSecret: *rtSecret, CoAPort: *rtCoAPort}
	}
	return &c, nil
}

func (r *customerRepository) FindByPPPoEUsernameGlobal(ctx context.Context, username string) (*model.Customer, error) {
	query := `SELECT id, tenant_id, customer_code, name, pppoe_username, status FROM customers WHERE pppoe_username = $1 LIMIT 1`
	var c model.Customer
	err := r.db.QueryRow(ctx, query, username).Scan(&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.PPPoEUsername, &c.Status)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *customerRepository) FindByActiveSessionIP(ctx context.Context, ip string) (*model.Customer, error) {
	query := `
		SELECT c.id, c.tenant_id, c.customer_code, c.name, c.pppoe_username, c.status
		FROM radius_sessions rs
		JOIN customers c ON c.id = rs.customer_id
		WHERE rs.framed_ip = $1 AND rs.ended_at IS NULL
		ORDER BY rs.started_at DESC
		LIMIT 1
	`
	var c model.Customer
	err := r.db.QueryRow(ctx, query, ip).Scan(&c.ID, &c.TenantID, &c.CustomerCode, &c.Name, &c.PPPoEUsername, &c.Status)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *customerRepository) FindByCode(ctx context.Context, tenantID, customerCode string) (*model.Customer, error) {
	query := `SELECT c.id, c.tenant_id, c.customer_code, c.name,
		COALESCE(c.phone, ''), COALESCE(c.email, ''), COALESCE(c.status, ''), COALESCE(c.password_hash, '')
		FROM customers c WHERE c.customer_code = $1 AND c.tenant_id = $2 LIMIT 1`
	var c model.Customer
	err := r.db.QueryRow(ctx, query, customerCode, tenantID).Scan(
		&c.ID, &c.TenantID, &c.CustomerCode, &c.Name,
		&c.Phone, &c.Email, &c.Status, &c.PasswordHash,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *customerRepository) FindByCodeGlobal(ctx context.Context, customerCode string) ([]model.Customer, error) {
	return nil, nil
}
func (r *customerRepository) FindByPhone(ctx context.Context, tenantID, phone string) (*model.Customer, error) {
	return nil, nil
}
func (r *customerRepository) FindByEmail(ctx context.Context, tenantID, email string) (*model.Customer, error) {
	return nil, nil
}
func (r *customerRepository) UpdateStatus(ctx context.Context, tenantID, id, status string) error {
	return nil
}
func (r *customerRepository) CountByCodePrefix(ctx context.Context, tenantID, prefix string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM customers WHERE tenant_id = $1 AND customer_code LIKE $2`,
		tenantID, prefix+"%",
	).Scan(&count)
	return count, err
}

func (r *customerRepository) CountByPPPoEPrefix(ctx context.Context, tenantID, prefix string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM customers WHERE tenant_id = $1 AND pppoe_username LIKE $2`,
		tenantID, prefix+"%",
	).Scan(&count)
	return count, err
}

func (r *customerRepository) CountActive(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM customers WHERE tenant_id = $1 AND status != 'deleted'`,
		tenantID,
	).Scan(&count)
	return count, err
}
func (r *customerRepository) UpdateODPPortID(ctx context.Context, customerID string, odpPortID *string) error {
	return nil
}
func (r *customerRepository) ClearODPPortID(ctx context.Context, odpPortID string) error { return nil }
func (r *customerRepository) UpdatePasswordHash(ctx context.Context, tenantID, customerID, hash string) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE customers SET password_hash = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		hash, customerID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *customerRepository) FindByReferralCode(ctx context.Context, tenantID, referralCode string) (*model.Customer, error) {
	var customerID string
	err := r.db.QueryRow(ctx,
		`SELECT id FROM customers WHERE tenant_id = $1 AND referral_code = $2 AND status != 'deleted' LIMIT 1`,
		tenantID, referralCode,
	).Scan(&customerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.FindByID(ctx, tenantID, customerID)
}
