package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type NotificationRepository interface {
	// Device tokens
	RegisterDevice(ctx context.Context, device *model.PushDevice) error
	UnregisterDevice(ctx context.Context, tenantID, customerID, fcmToken string) error
	GetDevicesByCustomer(ctx context.Context, tenantID, customerID string) ([]model.PushDevice, error)
	GetDevicesByTenant(ctx context.Context, tenantID string) ([]model.PushDevice, error)

	// Notifications
	Create(ctx context.Context, notif *model.Notification) error
	CreateBatch(ctx context.Context, notifs []model.Notification) error
	List(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Notification, int, error)
	MarkRead(ctx context.Context, tenantID, notifID string) error
	MarkReadByCustomer(ctx context.Context, tenantID, notifID, customerID string) error
	MarkAllRead(ctx context.Context, tenantID, customerID string) error
	UnreadCount(ctx context.Context, tenantID, customerID string) (int, error)
}

type notificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) RegisterDevice(ctx context.Context, device *model.PushDevice) error {
	device.ID = id.New()
	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	device.IsActive = true

	query := `
		INSERT INTO push_devices (id, tenant_id, customer_id, device_type, fcm_token, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, customer_id, fcm_token)
		DO UPDATE SET is_active = true, updated_at = $8
	`

	_, err := r.db.Exec(ctx, query,
		device.ID, device.TenantID, device.CustomerID, device.DeviceType,
		device.FCMToken, device.IsActive, device.CreatedAt, device.UpdatedAt,
	)
	return err
}

func (r *notificationRepository) UnregisterDevice(ctx context.Context, tenantID, customerID, fcmToken string) error {
	query := `UPDATE push_devices SET is_active = false, updated_at = NOW() WHERE tenant_id = $1 AND customer_id = $2 AND fcm_token = $3`
	_, err := r.db.Exec(ctx, query, tenantID, customerID, fcmToken)
	return err
}

func (r *notificationRepository) GetDevicesByCustomer(ctx context.Context, tenantID, customerID string) ([]model.PushDevice, error) {
	query := `SELECT id, tenant_id, customer_id, device_type, fcm_token, is_active, created_at, updated_at
		FROM push_devices WHERE tenant_id = $1 AND customer_id = $2 AND is_active = true`

	rows, err := r.db.Query(ctx, query, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.PushDevice
	for rows.Next() {
		var d model.PushDevice
		if err := rows.Scan(&d.ID, &d.TenantID, &d.CustomerID, &d.DeviceType, &d.FCMToken, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *notificationRepository) GetDevicesByTenant(ctx context.Context, tenantID string) ([]model.PushDevice, error) {
	query := `SELECT id, tenant_id, customer_id, device_type, fcm_token, is_active, created_at, updated_at
		FROM push_devices WHERE tenant_id = $1 AND is_active = true`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.PushDevice
	for rows.Next() {
		var d model.PushDevice
		if err := rows.Scan(&d.ID, &d.TenantID, &d.CustomerID, &d.DeviceType, &d.FCMToken, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (r *notificationRepository) Create(ctx context.Context, notif *model.Notification) error {
	notif.ID = id.New()
	notif.CreatedAt = time.Now()

	query := `
		INSERT INTO notifications (id, tenant_id, customer_id, title, body, data, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	data, err := notificationDataValue(notif.Data)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, query,
		notif.ID, notif.TenantID, notif.CustomerID, notif.Title, notif.Body,
		data, notif.IsRead, notif.CreatedAt,
	)
	return err
}

func (r *notificationRepository) CreateBatch(ctx context.Context, notifs []model.Notification) error {
	if len(notifs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(`INSERT INTO notifications (id, tenant_id, customer_id, title, body, data, is_read, created_at) VALUES `)
	args := make([]interface{}, 0, len(notifs)*8)

	for i, n := range notifs {
		if n.ID == "" {
			notifs[i].ID = id.New()
			n.ID = notifs[i].ID
		}
		if n.CreatedAt.IsZero() {
			now := time.Now()
			notifs[i].CreatedAt = now
			n.CreatedAt = now
		}

		if i > 0 {
			sb.WriteString(", ")
		}
		base := i*8 + 1
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6, base+7))
		data, err := notificationDataValue(n.Data)
		if err != nil {
			return err
		}
		args = append(args, n.ID, n.TenantID, n.CustomerID, n.Title, n.Body, data, n.IsRead, n.CreatedAt)
	}

	_, err := r.db.Exec(ctx, sb.String(), args...)
	return err
}

func notificationDataValue(data string) (interface{}, error) {
	if strings.TrimSpace(data) == "" {
		return nil, nil
	}
	if !json.Valid([]byte(data)) {
		return nil, fmt.Errorf("invalid notification data json")
	}
	return data, nil
}

func (r *notificationRepository) List(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Notification, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if customerID != "" {
		conditions = append(conditions, fmt.Sprintf("customer_id = $%d", argIdx))
		args = append(args, customerID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM notifications "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, customer_id, title, body, COALESCE(data::text,''), is_read, read_at, created_at
		FROM notifications %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifs []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.TenantID, &n.CustomerID, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		notifs = append(notifs, n)
	}
	return notifs, total, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, tenantID, notifID string) error {
	query := `UPDATE notifications SET is_read = true, read_at = NOW() WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, notifID, tenantID)
	return err
}

func (r *notificationRepository) MarkReadByCustomer(ctx context.Context, tenantID, notifID, customerID string) error {
	query := `UPDATE notifications SET is_read = true, read_at = NOW() WHERE id = $1 AND tenant_id = $2 AND customer_id = $3`
	_, err := r.db.Exec(ctx, query, notifID, tenantID, customerID)
	return err
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, tenantID, customerID string) error {
	query := `UPDATE notifications SET is_read = true, read_at = NOW() WHERE tenant_id = $1 AND customer_id = $2 AND is_read = false`
	_, err := r.db.Exec(ctx, query, tenantID, customerID)
	return err
}

func (r *notificationRepository) UnreadCount(ctx context.Context, tenantID, customerID string) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND customer_id = $2 AND is_read = false`
	var count int
	err := r.db.QueryRow(ctx, query, tenantID, customerID).Scan(&count)
	return count, err
}
