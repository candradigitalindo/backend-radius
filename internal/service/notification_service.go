package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var ErrInvalidNotificationData = errors.New("data notifikasi harus berupa JSON valid")

type NotificationService struct {
	notifRepo    repository.NotificationRepository
	customerRepo repository.CustomerRepository
	waClient     *whatsapp.Client
	fcmKey       string
	fcmEnabled   bool
	httpClient   *http.Client
}

func NewNotificationService(notifRepo repository.NotificationRepository, customerRepo repository.CustomerRepository, fcmKey string, fcmEnabled bool) *NotificationService {
	return &NotificationService{
		notifRepo:    notifRepo,
		customerRepo: customerRepo,
		fcmKey:       fcmKey,
		fcmEnabled:   fcmEnabled,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *NotificationService) WithWAClient(waClient *whatsapp.Client) *NotificationService {
	s.waClient = waClient
	return s
}

// RegisterDevice registers a push notification device token.
func (s *NotificationService) RegisterDevice(ctx context.Context, device *model.PushDevice) error {
	return s.notifRepo.RegisterDevice(ctx, device)
}

// UnregisterDevice deactivates a device token.
func (s *NotificationService) UnregisterDevice(ctx context.Context, tenantID, customerID, fcmToken string) error {
	return s.notifRepo.UnregisterDevice(ctx, tenantID, customerID, fcmToken)
}

// GetNotifications returns paginated notifications for a customer.
func (s *NotificationService) GetNotifications(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Notification, int, error) {
	return s.notifRepo.List(ctx, tenantID, customerID, page, perPage)
}

// MarkRead marks a notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, tenantID, notifID string) error {
	return s.notifRepo.MarkRead(ctx, tenantID, notifID)
}

// MarkReadByCustomer marks a notification as read with customer ownership validation.
func (s *NotificationService) MarkReadByCustomer(ctx context.Context, tenantID, notifID, customerID string) error {
	return s.notifRepo.MarkReadByCustomer(ctx, tenantID, notifID, customerID)
}

// MarkAllRead marks all notifications for a customer as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, tenantID, customerID string) error {
	return s.notifRepo.MarkAllRead(ctx, tenantID, customerID)
}

// UnreadCount returns the count of unread notifications.
func (s *NotificationService) UnreadCount(ctx context.Context, tenantID, customerID string) (int, error) {
	return s.notifRepo.UnreadCount(ctx, tenantID, customerID)
}

// SendToCustomer sends a push notification to a specific customer and stores it.
func (s *NotificationService) SendToCustomer(ctx context.Context, tenantID, customerID, title, body, data, fileURL string) error {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	if data != "" && !json.Valid([]byte(data)) {
		return ErrInvalidNotificationData
	}

	// Store notification
	notif := &model.Notification{
		TenantID:   tenantID,
		CustomerID: customerID,
		Title:      title,
		Body:       body,
		Data:       data,
	}
	if err := s.notifRepo.Create(ctx, notif); err != nil {
		return err
	}

	// Send push via FCM (opsional)
	if s.fcmEnabled && s.fcmKey != "" {
		devices, err := s.notifRepo.GetDevicesByCustomer(ctx, tenantID, customerID)
		if err != nil {
			log.Printf("[notification] gagal ambil devices FCM untuk customer %s: %v", customerID, err)
		} else {
			for _, d := range devices {
				_ = s.sendFCM(d.FCMToken, title, body, data)
			}
		}
	}

	// Send WhatsApp message (independen dari FCM)
	if s.waClient != nil && customer.Phone != "" {
		msg := fmt.Sprintf("*%s*\n\n%s", title, body)
		if _, err := s.waClient.SendMessageWithFile(ctx, tenantID, customer.Phone, msg, fileURL); err != nil {
			log.Printf("[notification] WA send failed for customer %s: %v", customerID, err)
		}
	}

	return nil
}

// BroadcastToTenant sends a push notification to all customers in a tenant.
func (s *NotificationService) BroadcastToTenant(ctx context.Context, tenantID, title, body, data, fileURL, target string, cleanup func()) (int, error) {
	if data != "" && !json.Valid([]byte(data)) {
		return 0, ErrInvalidNotificationData
	}

	devices, err := s.notifRepo.GetDevicesByTenant(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, d := range devices {
		if err := s.sendFCM(d.FCMToken, title, body, data); err == nil {
			sent++
		}
	}

	// Send WhatsApp broadcast filtered by target
	if s.waClient != nil {
		var statuses []string
		switch target {
		case "active":
			statuses = []string{"active"}
		case "inactive":
			statuses = []string{"inactive"}
		case "isolated":
			statuses = []string{"isolated"}
		default: // "all" or empty
			statuses = []string{"active", "inactive", "isolated"}
		}

		var phones []string
		for _, status := range statuses {
			page := 1
			const perPage = 500
			for {
				customers, _, err := s.customerRepo.List(ctx, tenantID, repository.CustomerFilter{Status: status, Page: page, PerPage: perPage})
				if err != nil {
					log.Printf("[notification-broadcast] gagal ambil customers status=%s: %v", status, err)
					break
				}
				for _, c := range customers {
					if c.Phone != "" {
						phones = append(phones, c.Phone)
					}
				}
				if len(customers) < perPage {
					break
				}
				page++
			}
		}
		if len(phones) > 0 {
			msg := fmt.Sprintf("*%s*\n\n%s", title, body)
			go func(tID string, p []string, m, fURL string, cleanFn func()) {
				if _, err := s.waClient.SendBroadcastWithFile(context.Background(), tID, p, m, fURL); err != nil {
					log.Printf("[notification-broadcast] WA broadcast failed: %v", err)
				}
				if cleanFn != nil {
					cleanFn()
				}
			}(tenantID, phones, msg, fileURL, cleanup)
		} else {
			if cleanup != nil {
				cleanup()
			}
		}
	} else {
		if cleanup != nil {
			cleanup()
		}
	}

	return sent, nil
}

type fcmMessage struct {
	To           string            `json:"to"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *NotificationService) sendFCM(token, title, body, data string) error {
	msg := fcmMessage{
		To: token,
		Notification: fcmNotification{
			Title: title,
			Body:  body,
		},
	}

	if data != "" {
		msg.Data = map[string]string{"payload": data}
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("key=%s", s.fcmKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FCM mengembalikan status %d", resp.StatusCode)
	}
	return nil
}
