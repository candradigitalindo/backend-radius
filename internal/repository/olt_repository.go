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

type OLTRepository interface {
	Create(ctx context.Context, olt *model.OLT) error
	FindByID(ctx context.Context, tenantID, oltID string) (*model.OLT, error)
	Update(ctx context.Context, olt *model.OLT) error
	Delete(ctx context.Context, tenantID, oltID string) error
	List(ctx context.Context, tenantID string, filter OLTFilter) ([]model.OLT, int, error)
	ListAllWithRouter(ctx context.Context) ([]model.OLT, error)
	CreatePONPort(ctx context.Context, port *model.PONPort) error
	ListPONPorts(ctx context.Context, oltID string) ([]model.PONPort, error)
	FindPONPortByID(ctx context.Context, portID string) (*model.PONPort, error)
	UpdatePONPort(ctx context.Context, port *model.PONPort) error
	DeletePONPort(ctx context.Context, portID string) error
}

type OLTFilter struct {
	Search  string
	Status  string
	Page    int
	PerPage int
}

type oltRepository struct {
	db *pgxpool.Pool
}

func NewOLTRepository(db *pgxpool.Pool) OLTRepository {
	return &oltRepository{db: db}
}

func (r *oltRepository) Create(ctx context.Context, olt *model.OLT) error {
	olt.ID = id.New()
	now := time.Now()
	olt.CreatedAt = now
	olt.UpdatedAt = now

	query := `
		INSERT INTO olts (
			id, tenant_id, router_id, name, ip_address, vendor, model, serial_number,
			total_pon_ports, latitude, longitude, snmp_community, status, notes,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`

	_, err := r.db.Exec(ctx, query,
		olt.ID, olt.TenantID, olt.RouterID, olt.Name, olt.IPAddress,
		olt.Vendor, olt.Model, olt.SerialNumber,
		olt.TotalPONPorts, olt.Latitude, olt.Longitude,
		olt.SNMPCommunity, olt.Status, olt.Notes,
		olt.CreatedAt, olt.UpdatedAt,
	)
	return err
}

func (r *oltRepository) FindByID(ctx context.Context, tenantID, oltID string) (*model.OLT, error) {
	query := `
		SELECT o.id, o.tenant_id, o.router_id, o.name, o.ip_address, o.vendor, o.model, o.serial_number,
		       o.total_pon_ports, o.latitude, o.longitude, COALESCE(o.snmp_community,''), o.status, o.notes,
		       o.created_at, o.updated_at,
		       r.id, r.name, r.vpn_ip, r.is_online
		FROM olts o
		LEFT JOIN routers r ON r.id = o.router_id
		WHERE o.id = $1 AND o.tenant_id = $2
		LIMIT 1
	`

	var olt model.OLT
	var rID, rName, rVPNIP *string
	var rOnline *bool
	err := r.db.QueryRow(ctx, query, oltID, tenantID).Scan(
		&olt.ID, &olt.TenantID, &olt.RouterID, &olt.Name, &olt.IPAddress,
		&olt.Vendor, &olt.Model, &olt.SerialNumber,
		&olt.TotalPONPorts, &olt.Latitude, &olt.Longitude,
		&olt.SNMPCommunity, &olt.Status, &olt.Notes,
		&olt.CreatedAt, &olt.UpdatedAt,
		&rID, &rName, &rVPNIP, &rOnline,
	)
	if err == nil && rID != nil {
		olt.Router = &model.Router{ID: *rID, Name: *rName, VPNIP: *rVPNIP, IsOnline: *rOnline}
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	ports, err := r.ListPONPorts(ctx, olt.ID)
	if err != nil {
		return nil, err
	}
	olt.PONPorts = ports

	return &olt, nil
}

func (r *oltRepository) Update(ctx context.Context, olt *model.OLT) error {
	olt.UpdatedAt = time.Now()

	query := `
		UPDATE olts SET
			router_id = $1, name = $2, ip_address = $3, vendor = $4, model = $5,
			serial_number = $6, total_pon_ports = $7, latitude = $8, longitude = $9,
			snmp_community = $10, status = $11, notes = $12, updated_at = $13
		WHERE id = $14 AND tenant_id = $15
	`

	_, err := r.db.Exec(ctx, query,
		olt.RouterID, olt.Name, olt.IPAddress, olt.Vendor, olt.Model,
		olt.SerialNumber, olt.TotalPONPorts, olt.Latitude, olt.Longitude,
		olt.SNMPCommunity, olt.Status, olt.Notes, olt.UpdatedAt,
		olt.ID, olt.TenantID,
	)
	return err
}

func (r *oltRepository) Delete(ctx context.Context, tenantID, oltID string) error {
	query := `DELETE FROM olts WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, oltID, tenantID)
	return err
}

func (r *oltRepository) List(ctx context.Context, tenantID string, filter OLTFilter) ([]model.OLT, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("o.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(o.name ILIKE $%d OR o.ip_address ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM olts o "+where, args...).Scan(&total); err != nil {
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
		SELECT o.id, o.tenant_id, o.router_id, o.name, o.ip_address, o.vendor, o.model, o.serial_number,
		       o.total_pon_ports, o.latitude, o.longitude, COALESCE(o.snmp_community,''), o.status, o.notes,
		       o.created_at, o.updated_at,
		       r.id, r.name
		FROM olts o
		LEFT JOIN routers r ON r.id = o.router_id
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

	var olts []model.OLT
	for rows.Next() {
		var olt model.OLT
		var rID, rName *string
		if err := rows.Scan(
			&olt.ID, &olt.TenantID, &olt.RouterID, &olt.Name, &olt.IPAddress,
			&olt.Vendor, &olt.Model, &olt.SerialNumber,
			&olt.TotalPONPorts, &olt.Latitude, &olt.Longitude,
			&olt.SNMPCommunity, &olt.Status, &olt.Notes,
			&olt.CreatedAt, &olt.UpdatedAt,
			&rID, &rName,
		); err != nil {
			return nil, 0, err
		}
		if rID != nil {
			olt.Router = &model.Router{ID: *rID, Name: *rName}
		}
		olts = append(olts, olt)
	}

	return olts, total, nil
}

func (r *oltRepository) CreatePONPort(ctx context.Context, port *model.PONPort) error {
	port.ID = id.New()

	query := `
		INSERT INTO pon_ports (id, olt_id, port_number, description, status, sfp_rx_power, sfp_tx_power)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		port.ID, port.OLTID, port.PortNumber, port.Description, port.Status,
		port.SFPRxPower, port.SFPTxPower,
	)
	return err
}

// ListAllWithRouter returns all active OLTs that have a router attached — used for route restore on startup.
func (r *oltRepository) ListAllWithRouter(ctx context.Context) ([]model.OLT, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.id, o.tenant_id, o.ip_address, o.name,
		       rt.id, rt.vpn_ip
		FROM olts o
		JOIN routers rt ON o.router_id = rt.id
		WHERE o.status = 'active' AND rt.vpn_ip != '' AND rt.is_active = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var olts []model.OLT
	for rows.Next() {
		var o model.OLT
		var rt model.Router
		if err := rows.Scan(&o.ID, &o.TenantID, &o.IPAddress, &o.Name, &rt.ID, &rt.VPNIP); err != nil {
			return nil, err
		}
		o.Router = &rt
		olts = append(olts, o)
	}
	return olts, nil
}

func (r *oltRepository) ListPONPorts(ctx context.Context, oltID string) ([]model.PONPort, error) {
	query := `
		SELECT id, olt_id, port_number, description, status,
		       if_index, COALESCE(admin_status,'unknown'), COALESCE(oper_status,'unknown'),
		       COALESCE(in_octets,0), COALESCE(out_octets,0),
		       sfp_vendor, sfp_model, sfp_rx_power, sfp_tx_power,
		       sfp_temperature, sfp_voltage, last_polled_at
		FROM pon_ports
		WHERE olt_id = $1
		ORDER BY port_number
	`

	rows, err := r.db.Query(ctx, query, oltID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []model.PONPort
	for rows.Next() {
		var port model.PONPort
		if err := rows.Scan(
			&port.ID, &port.OLTID, &port.PortNumber, &port.Description, &port.Status,
			&port.IfIndex, &port.AdminStatus, &port.OperStatus,
			&port.InOctets, &port.OutOctets,
			&port.SFPVendor, &port.SFPModel, &port.SFPRxPower, &port.SFPTxPower,
			&port.SFPTemperature, &port.SFPVoltage, &port.LastPolledAt,
		); err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}

	return ports, nil
}

func (r *oltRepository) FindPONPortByID(ctx context.Context, portID string) (*model.PONPort, error) {
	query := `
		SELECT id, olt_id, port_number, description, status, sfp_rx_power, sfp_tx_power
		FROM pon_ports
		WHERE id = $1
		LIMIT 1
	`

	var port model.PONPort
	err := r.db.QueryRow(ctx, query, portID).Scan(
		&port.ID, &port.OLTID, &port.PortNumber, &port.Description, &port.Status,
		&port.SFPRxPower, &port.SFPTxPower,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &port, nil
}

func (r *oltRepository) UpdatePONPort(ctx context.Context, port *model.PONPort) error {
	query := `
		UPDATE pon_ports SET description = $1, status = $2, sfp_rx_power = $3, sfp_tx_power = $4
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query, port.Description, port.Status, port.SFPRxPower, port.SFPTxPower, port.ID)
	return err
}

func (r *oltRepository) DeletePONPort(ctx context.Context, portID string) error {
	query := `DELETE FROM pon_ports WHERE id = $1`
	_, err := r.db.Exec(ctx, query, portID)
	return err
}
