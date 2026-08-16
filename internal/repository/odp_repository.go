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

type ODPRepository interface {
	Create(ctx context.Context, odp *model.ODP) error
	FindByID(ctx context.Context, tenantID, odpID string) (*model.ODP, error)
	Update(ctx context.Context, odp *model.ODP) error
	Delete(ctx context.Context, tenantID, odpID string) error
	List(ctx context.Context, tenantID string, filter ODPFilter) ([]model.ODP, int, error)
	ListByPONPort(ctx context.Context, ponPortID string) ([]model.ODP, error)
	GetPONPortSFPRxPower(ctx context.Context, ponPortID string) (*float64, error)
	CreatePort(ctx context.Context, port *model.ODPPort) error
	ListPorts(ctx context.Context, odpID string) ([]model.ODPPort, error)
	FindPortByID(ctx context.Context, portID string) (*model.ODPPort, error)
	UpdatePort(ctx context.Context, port *model.ODPPort) error
	DeletePort(ctx context.Context, portID string) error
	CreateSplitter(ctx context.Context, s *model.Splitter) error
	FindSplitterByID(ctx context.Context, tenantID, splitterID string) (*model.Splitter, error)
	UpdateSplitter(ctx context.Context, s *model.Splitter) error
	DeleteSplitter(ctx context.Context, tenantID, splitterID string) error
	ListSplitters(ctx context.Context, tenantID string, filter SplitterFilter) ([]model.Splitter, int, error)
}

type ODPFilter struct {
	Search  string
	OLTID   string
	Page    int
	PerPage int
}

type SplitterFilter struct {
	Search    string
	PONPortID string
	ParentID  string
	Page      int
	PerPage   int
}

type odpRepository struct {
	db *pgxpool.Pool
}

func NewODPRepository(db *pgxpool.Pool) ODPRepository {
	return &odpRepository{db: db}
}

func (r *odpRepository) Create(ctx context.Context, odp *model.ODP) error {
	odp.ID = id.New()
	now := time.Now()
	odp.CreatedAt = now
	odp.UpdatedAt = now

	query := `
		INSERT INTO odps (
			id, tenant_id, olt_id, splitter_id, pon_port_id, splitter_ratio, name, address,
			latitude, longitude, total_ports, sequence, cable_length_m,
			ratio_percent, splitter_type, power_level_dbm, status, notes,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
	`

	_, err := r.db.Exec(ctx, query,
		odp.ID, odp.TenantID, odp.OLTID, odp.SplitterID, odp.PONPortID, odp.SplitterRatio,
		odp.Name, odp.Address,
		odp.Latitude, odp.Longitude, odp.TotalPorts,
		odp.Sequence, odp.CableLengthM, odp.RatioPercent, odp.SplitterType,
		odp.PowerLevelDBm, odp.Status, odp.Notes,
		odp.CreatedAt, odp.UpdatedAt,
	)
	return err
}

func (r *odpRepository) FindByID(ctx context.Context, tenantID, odpID string) (*model.ODP, error) {
	query := `
		SELECT o.id, o.tenant_id, o.olt_id, o.splitter_id, o.pon_port_id, o.splitter_ratio,
		       o.name, o.address,
		       o.latitude, o.longitude, o.total_ports,
		       o.sequence, COALESCE(o.cable_length_m,0), COALESCE(o.ratio_percent,0),
		       o.splitter_type, o.power_level_dbm,
		       COALESCE(o.status,'draft'), o.notes, o.created_at, o.updated_at,
		       olt.name, pp.sfp_rx_power, pp.sfp_tx_power, pp.port_number
		FROM odps o
		LEFT JOIN olts olt ON olt.id = o.olt_id
		LEFT JOIN pon_ports pp ON pp.id = o.pon_port_id
		WHERE o.id = $1 AND o.tenant_id = $2
		LIMIT 1
	`

	var odp model.ODP
	var oltName *string
	err := r.db.QueryRow(ctx, query, odpID, tenantID).Scan(
		&odp.ID, &odp.TenantID, &odp.OLTID, &odp.SplitterID, &odp.PONPortID, &odp.SplitterRatio,
		&odp.Name, &odp.Address,
		&odp.Latitude, &odp.Longitude, &odp.TotalPorts,
		&odp.Sequence, &odp.CableLengthM, &odp.RatioPercent,
		&odp.SplitterType, &odp.PowerLevelDBm,
		&odp.Status, &odp.Notes, &odp.CreatedAt, &odp.UpdatedAt,
		&oltName, &odp.PONPortSFPRxPower, &odp.PONPortSFPTxPower, &odp.PONPortNumber,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if odp.OLTID != nil && oltName != nil {
		odp.OLT = &model.OLT{ID: *odp.OLTID, Name: *oltName}
	}

	ports, err := r.ListPorts(ctx, odp.ID)
	if err != nil {
		return nil, err
	}
	odp.Ports = ports

	return &odp, nil
}

func (r *odpRepository) Update(ctx context.Context, odp *model.ODP) error {
	odp.UpdatedAt = time.Now()

	query := `
		UPDATE odps SET
			olt_id = $1, splitter_id = $2, pon_port_id = $3, splitter_ratio = $4, name = $5, address = $6,
			latitude = $7, longitude = $8, total_ports = $9,
			sequence = $10, cable_length_m = $11, ratio_percent = $12,
			splitter_type = $13, power_level_dbm = $14, status = $15, notes = $16,
			updated_at = $17
		WHERE id = $18 AND tenant_id = $19
	`

	_, err := r.db.Exec(ctx, query,
		odp.OLTID, odp.SplitterID, odp.PONPortID, odp.SplitterRatio, odp.Name, odp.Address,
		odp.Latitude, odp.Longitude, odp.TotalPorts,
		odp.Sequence, odp.CableLengthM, odp.RatioPercent,
		odp.SplitterType, odp.PowerLevelDBm, odp.Status, odp.Notes,
		odp.UpdatedAt, odp.ID, odp.TenantID,
	)
	return err
}

func (r *odpRepository) Delete(ctx context.Context, tenantID, odpID string) error {
	query := `DELETE FROM odps WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, odpID, tenantID)
	return err
}

func (r *odpRepository) List(ctx context.Context, tenantID string, filter ODPFilter) ([]model.ODP, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("o.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(o.name ILIKE $%d OR o.address ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.OLTID != "" {
		conditions = append(conditions, fmt.Sprintf("o.olt_id = $%d", argIdx))
		args = append(args, filter.OLTID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM odps o "+where, args...).Scan(&total); err != nil {
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
		SELECT o.id, o.tenant_id, o.olt_id, o.splitter_id, o.pon_port_id, o.splitter_ratio,
		       o.name, o.address,
		       o.latitude, o.longitude, o.total_ports,
		       o.sequence, COALESCE(o.cable_length_m,0), COALESCE(o.ratio_percent,0),
		       o.splitter_type, o.power_level_dbm,
		       COALESCE(o.status,'draft'), o.notes, o.created_at, o.updated_at,
		       olt.name, pp.sfp_rx_power, pp.sfp_tx_power, pp.port_number
		FROM odps o
		LEFT JOIN olts olt ON olt.id = o.olt_id
		LEFT JOIN pon_ports pp ON pp.id = o.pon_port_id
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

	var odps []model.ODP
	for rows.Next() {
		var odp model.ODP
		var oltName *string
		if err := rows.Scan(
			&odp.ID, &odp.TenantID, &odp.OLTID, &odp.SplitterID, &odp.PONPortID, &odp.SplitterRatio,
			&odp.Name, &odp.Address,
			&odp.Latitude, &odp.Longitude, &odp.TotalPorts,
			&odp.Sequence, &odp.CableLengthM, &odp.RatioPercent,
			&odp.SplitterType, &odp.PowerLevelDBm,
			&odp.Status, &odp.Notes, &odp.CreatedAt, &odp.UpdatedAt,
			&oltName, &odp.PONPortSFPRxPower, &odp.PONPortSFPTxPower, &odp.PONPortNumber,
		); err != nil {
			return nil, 0, err
		}
		if odp.OLTID != nil && oltName != nil {
			odp.OLT = &model.OLT{ID: *odp.OLTID, Name: *oltName}
		}
		odps = append(odps, odp)
	}

	return odps, total, nil
}

func (r *odpRepository) ListByPONPort(ctx context.Context, ponPortID string) ([]model.ODP, error) {
	query := `
		SELECT o.id, o.tenant_id, o.olt_id, o.pon_port_id, o.splitter_ratio,
		       o.name, o.address,
		       o.latitude, o.longitude, o.total_ports,
		       o.sequence, COALESCE(o.cable_length_m,0), COALESCE(o.ratio_percent,0),
		       o.splitter_type, o.power_level_dbm,
		       COALESCE(o.status,'draft'), o.notes, o.created_at, o.updated_at,
		       pp.sfp_rx_power, pp.sfp_tx_power
		FROM odps o
		LEFT JOIN pon_ports pp ON pp.id = o.pon_port_id
		WHERE o.pon_port_id = $1
		ORDER BY o.sequence ASC
	`

	rows, err := r.db.Query(ctx, query, ponPortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var odps []model.ODP
	for rows.Next() {
		var odp model.ODP
		if err := rows.Scan(
			&odp.ID, &odp.TenantID, &odp.OLTID, &odp.PONPortID, &odp.SplitterRatio,
			&odp.Name, &odp.Address,
			&odp.Latitude, &odp.Longitude, &odp.TotalPorts,
			&odp.Sequence, &odp.CableLengthM, &odp.RatioPercent,
			&odp.SplitterType, &odp.PowerLevelDBm,
			&odp.Status, &odp.Notes, &odp.CreatedAt, &odp.UpdatedAt,
			&odp.PONPortSFPRxPower, &odp.PONPortSFPTxPower,
		); err != nil {
			return nil, err
		}
		odps = append(odps, odp)
	}
	return odps, nil
}

func (r *odpRepository) GetPONPortSFPRxPower(ctx context.Context, ponPortID string) (*float64, error) {
	var power *float64
	err := r.db.QueryRow(ctx, "SELECT sfp_rx_power FROM pon_ports WHERE id = $1", ponPortID).Scan(&power)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return power, nil
}

func (r *odpRepository) CreatePort(ctx context.Context, port *model.ODPPort) error {
	port.ID = id.New()

	query := `
		INSERT INTO odp_ports (id, odp_id, port_number, customer_id, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		port.ID, port.ODPID, port.PortNumber, port.CustomerID, port.Status, port.Notes,
	)
	return err
}

func (r *odpRepository) ListPorts(ctx context.Context, odpID string) ([]model.ODPPort, error) {
	query := `
		SELECT p.id, p.odp_id, p.port_number, p.customer_id, p.status, p.notes,
		       c.name, c.customer_code,
		       ont.id, ont.serial_number, ont.vendor, ont.model, ont.status
		FROM odp_ports p
		LEFT JOIN customers c ON c.id = p.customer_id
		LEFT JOIN onts ont ON ont.odp_port_id = p.id
		WHERE p.odp_id = $1
		ORDER BY p.port_number
	`

	rows, err := r.db.Query(ctx, query, odpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []model.ODPPort
	for rows.Next() {
		var port model.ODPPort
		var custName, custCode *string
		var ontID, ontSN, ontVendor, ontModel, ontStatus *string
		if err := rows.Scan(&port.ID, &port.ODPID, &port.PortNumber, &port.CustomerID, &port.Status, &port.Notes,
			&custName, &custCode,
			&ontID, &ontSN, &ontVendor, &ontModel, &ontStatus); err != nil {
			return nil, err
		}
		if port.CustomerID != nil && custName != nil {
			port.Customer = &model.Customer{ID: *port.CustomerID, Name: *custName}
			if custCode != nil {
				port.Customer.CustomerCode = *custCode
			}
		}
		if ontID != nil {
			port.ONT = &model.ONT{ID: *ontID}
			if ontSN != nil {
				port.ONT.SerialNumber = *ontSN
			}
			port.ONT.Vendor = ontVendor
			port.ONT.Model = ontModel
			if ontStatus != nil {
				port.ONT.Status = *ontStatus
			}
		}
		ports = append(ports, port)
	}

	return ports, nil
}

func (r *odpRepository) FindPortByID(ctx context.Context, portID string) (*model.ODPPort, error) {
	query := `
		SELECT id, odp_id, port_number, customer_id, status, notes
		FROM odp_ports
		WHERE id = $1
		LIMIT 1
	`

	var port model.ODPPort
	err := r.db.QueryRow(ctx, query, portID).Scan(
		&port.ID, &port.ODPID, &port.PortNumber, &port.CustomerID, &port.Status, &port.Notes,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &port, nil
}

func (r *odpRepository) UpdatePort(ctx context.Context, port *model.ODPPort) error {
	query := `
		UPDATE odp_ports SET customer_id = $1, status = $2, notes = $3
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, port.CustomerID, port.Status, port.Notes, port.ID)
	return err
}

func (r *odpRepository) DeletePort(ctx context.Context, portID string) error {
	query := `DELETE FROM odp_ports WHERE id = $1`
	_, err := r.db.Exec(ctx, query, portID)
	return err
}

// Splitter operations

func (r *odpRepository) CreateSplitter(ctx context.Context, s *model.Splitter) error {
	s.ID = id.New()
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	query := `
		INSERT INTO splitters (
			id, tenant_id, pon_port_id, parent_splitter_id, name, splitter_type,
			latitude, longitude, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`

	_, err := r.db.Exec(ctx, query,
		s.ID, s.TenantID, s.PONPortID, s.ParentSplitterID,
		s.Name, s.SplitterType,
		s.Latitude, s.Longitude, s.Notes,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *odpRepository) FindSplitterByID(ctx context.Context, tenantID, splitterID string) (*model.Splitter, error) {
	query := `
		SELECT id, tenant_id, pon_port_id, parent_splitter_id, name, splitter_type,
		       latitude, longitude, notes, created_at, updated_at
		FROM splitters
		WHERE id = $1 AND tenant_id = $2
		LIMIT 1
	`

	var s model.Splitter
	err := r.db.QueryRow(ctx, query, splitterID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.PONPortID, &s.ParentSplitterID,
		&s.Name, &s.SplitterType,
		&s.Latitude, &s.Longitude, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *odpRepository) UpdateSplitter(ctx context.Context, s *model.Splitter) error {
	s.UpdatedAt = time.Now()

	query := `
		UPDATE splitters SET
			pon_port_id = $1, parent_splitter_id = $2, name = $3, splitter_type = $4,
			latitude = $5, longitude = $6, notes = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10
	`

	_, err := r.db.Exec(ctx, query,
		s.PONPortID, s.ParentSplitterID, s.Name, s.SplitterType,
		s.Latitude, s.Longitude, s.Notes, s.UpdatedAt,
		s.ID, s.TenantID,
	)
	return err
}

func (r *odpRepository) DeleteSplitter(ctx context.Context, tenantID, splitterID string) error {
	query := `DELETE FROM splitters WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, splitterID, tenantID)
	return err
}

func (r *odpRepository) ListSplitters(ctx context.Context, tenantID string, filter SplitterFilter) ([]model.Splitter, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.PONPortID != "" {
		conditions = append(conditions, fmt.Sprintf("pon_port_id = $%d", argIdx))
		args = append(args, filter.PONPortID)
		argIdx++
	}

	if filter.ParentID != "" {
		conditions = append(conditions, fmt.Sprintf("parent_splitter_id = $%d", argIdx))
		args = append(args, filter.ParentID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM splitters "+where, args...).Scan(&total); err != nil {
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
		SELECT id, tenant_id, pon_port_id, parent_splitter_id, name, splitter_type,
		       latitude, longitude, notes, created_at, updated_at
		FROM splitters
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

	var splitters []model.Splitter
	for rows.Next() {
		var s model.Splitter
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.PONPortID, &s.ParentSplitterID,
			&s.Name, &s.SplitterType,
			&s.Latitude, &s.Longitude, &s.Notes,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		splitters = append(splitters, s)
	}

	return splitters, total, nil
}
