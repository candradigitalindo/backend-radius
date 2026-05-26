package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
)

type FTTHStats struct {
	TotalOLTs      int         `json:"total_olts"`
	TotalPONPorts  int         `json:"total_pon_ports"`
	TotalSplitters int         `json:"total_splitters"`
	TotalODPs      int         `json:"total_odps"`
	TotalODPPorts  int         `json:"total_odp_ports"`
	UsedODPPorts   int         `json:"used_odp_ports"`
	TotalONTs      int         `json:"total_onts"`
	ActiveONTs     int         `json:"active_onts"`
	RecentONTs     []model.ONT `json:"recent_onts"`
}

type FTTHMapItem struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Status    string   `json:"status"`
	ParentID  *string  `json:"parent_id,omitempty"`
}

type FTTHRepository interface {
	GetStats(ctx context.Context, tenantID string) (*FTTHStats, error)
	GetMapItems(ctx context.Context, tenantID string) ([]FTTHMapItem, error)
}

type ftthRepository struct {
	db *pgxpool.Pool
}

func NewFTTHRepository(db *pgxpool.Pool) FTTHRepository {
	return &ftthRepository{db: db}
}

func (r *ftthRepository) GetStats(ctx context.Context, tenantID string) (*FTTHStats, error) {
	var stats FTTHStats

	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM olts WHERE tenant_id = $1`, tenantID,
	).Scan(&stats.TotalOLTs)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM pon_ports pp JOIN olts o ON o.id = pp.olt_id WHERE o.tenant_id = $1`, tenantID,
	).Scan(&stats.TotalPONPorts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM splitters WHERE tenant_id = $1`, tenantID,
	).Scan(&stats.TotalSplitters)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM odps WHERE tenant_id = $1`, tenantID,
	).Scan(&stats.TotalODPs)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM odp_ports dp JOIN odps d ON d.id = dp.odp_id WHERE d.tenant_id = $1`, tenantID,
	).Scan(&stats.TotalODPPorts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM odp_ports dp JOIN odps d ON d.id = dp.odp_id WHERE d.tenant_id = $1 AND dp.status = 'used'`, tenantID,
	).Scan(&stats.UsedODPPorts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM onts WHERE tenant_id = $1`, tenantID,
	).Scan(&stats.TotalONTs)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM onts WHERE tenant_id = $1 AND status = 'active'`, tenantID,
	).Scan(&stats.ActiveONTs)
	if err != nil {
		return nil, err
	}

	// Recent ONTs
	ontRows, err := r.db.Query(ctx, `
		SELECT o.id, o.tenant_id, o.customer_id, o.odp_port_id, o.serial_number,
		       o.model, o.vendor, o.mac_address, o.ip_address, o.rx_power, o.tx_power,
		       o.status, o.notes, o.last_online_at, o.created_at, o.updated_at,
		       c.id, c.name
		FROM onts o
		LEFT JOIN customers c ON c.id = o.customer_id
		WHERE o.tenant_id = $1
		ORDER BY o.created_at DESC
		LIMIT 10
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer ontRows.Close()

	for ontRows.Next() {
		var ont model.ONT
		var custID, custName *string
		if err := ontRows.Scan(
			&ont.ID, &ont.TenantID, &ont.CustomerID, &ont.ODPPortID,
			&ont.SerialNumber, &ont.Model, &ont.Vendor,
			&ont.MACAddress, &ont.IPAddress, &ont.RxPower, &ont.TxPower,
			&ont.Status, &ont.Notes, &ont.LastOnlineAt,
			&ont.CreatedAt, &ont.UpdatedAt,
			&custID, &custName,
		); err != nil {
			return nil, err
		}
		if custID != nil {
			ont.Customer = &model.Customer{ID: *custID, Name: *custName}
		}
		stats.RecentONTs = append(stats.RecentONTs, ont)
	}
	if stats.RecentONTs == nil {
		stats.RecentONTs = []model.ONT{}
	}

	return &stats, nil
}

func (r *ftthRepository) GetMapItems(ctx context.Context, tenantID string) ([]FTTHMapItem, error) {
	var items []FTTHMapItem

	// OLTs
	rows, err := r.db.Query(ctx,
		`SELECT id, name, latitude, longitude, status FROM olts WHERE tenant_id = $1`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item FTTHMapItem
		item.Type = "olt"
		if err := rows.Scan(&item.ID, &item.Name, &item.Latitude, &item.Longitude, &item.Status); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	rows.Close()

	// Splitters
	rows, err = r.db.Query(ctx,
		`SELECT id, name, latitude, longitude, COALESCE(pon_port_id, parent_splitter_id) FROM splitters WHERE tenant_id = $1`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item FTTHMapItem
		item.Type = "splitter"
		item.Status = "active"
		if err := rows.Scan(&item.ID, &item.Name, &item.Latitude, &item.Longitude, &item.ParentID); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	rows.Close()

	// ODPs
	rows, err = r.db.Query(ctx,
		`SELECT id, name, latitude, longitude, COALESCE(splitter_id, olt_id) FROM odps WHERE tenant_id = $1`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item FTTHMapItem
		item.Type = "odp"
		item.Status = "active"
		if err := rows.Scan(&item.ID, &item.Name, &item.Latitude, &item.Longitude, &item.ParentID); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	rows.Close()

	return items, nil
}
