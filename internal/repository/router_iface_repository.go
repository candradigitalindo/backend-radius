package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IfaceSample is one throughput sample for a router interface.
type IfaceSample struct {
	Iface     string    `json:"iface"`
	RxBps     int64     `json:"rx_bps"`
	TxBps     int64     `json:"tx_bps"`
	SampledAt time.Time `json:"sampled_at"`
}

type RouterIfaceRepository interface {
	// RecordSample stores raw interface counters and derives bit/s from the delta
	// against the previous sample (router pushes counters; no SNMP needed).
	RecordSample(ctx context.Context, routerID, tenantID, iface string, rxBytes, txBytes int64) error
	// ListNames returns interface names that reported data within sinceMinutes.
	ListNames(ctx context.Context, routerID string, sinceMinutes int) ([]string, error)
	// ListSamples returns recent throughput samples (for charting).
	ListSamples(ctx context.Context, routerID string, sinceMinutes int) ([]IfaceSample, error)
	// Prune deletes samples older than keepMinutes (housekeeping).
	Prune(ctx context.Context, keepMinutes int) error
}

type routerIfaceRepository struct {
	db *pgxpool.Pool
}

func NewRouterIfaceRepository(db *pgxpool.Pool) RouterIfaceRepository {
	return &routerIfaceRepository{db: db}
}

func (r *routerIfaceRepository) RecordSample(ctx context.Context, routerID, tenantID, iface string, rxBytes, txBytes int64) error {
	now := time.Now()

	// Look up the previous raw counters to derive throughput.
	var prevRx, prevTx int64
	var prevAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT rx_bytes, tx_bytes, sampled_at FROM router_iface_samples
		WHERE router_id = $1 AND iface = $2 ORDER BY sampled_at DESC LIMIT 1
	`, routerID, iface).Scan(&prevRx, &prevTx, &prevAt)

	var rxBps, txBps int64
	if err == nil {
		secs := now.Sub(prevAt).Seconds()
		// Guard against counter reset (router reboot) and zero/negative intervals.
		if secs > 0 && rxBytes >= prevRx && txBytes >= prevTx {
			rxBps = int64(float64(rxBytes-prevRx) * 8 / secs)
			txBps = int64(float64(txBytes-prevTx) * 8 / secs)
		}
	} else if err != pgx.ErrNoRows {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO router_iface_samples (router_id, tenant_id, iface, rx_bytes, tx_bytes, rx_bps, tx_bps, sampled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, routerID, tenantID, iface, rxBytes, txBytes, rxBps, txBps, now)
	return err
}

func (r *routerIfaceRepository) ListNames(ctx context.Context, routerID string, sinceMinutes int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT iface FROM router_iface_samples
		WHERE router_id = $1 AND sampled_at > NOW() - ($2 * interval '1 minute')
		ORDER BY iface
	`, routerID, sinceMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, nil
}

func (r *routerIfaceRepository) ListSamples(ctx context.Context, routerID string, sinceMinutes int) ([]IfaceSample, error) {
	rows, err := r.db.Query(ctx, `
		SELECT iface, rx_bps, tx_bps, sampled_at FROM router_iface_samples
		WHERE router_id = $1 AND sampled_at > NOW() - ($2 * interval '1 minute')
		ORDER BY sampled_at ASC
	`, routerID, sinceMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IfaceSample
	for rows.Next() {
		var s IfaceSample
		if err := rows.Scan(&s.Iface, &s.RxBps, &s.TxBps, &s.SampledAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *routerIfaceRepository) Prune(ctx context.Context, keepMinutes int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM router_iface_samples WHERE sampled_at < NOW() - ($1 * interval '1 minute')`,
		keepMinutes,
	)
	return err
}
