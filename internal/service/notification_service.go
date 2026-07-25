package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var ErrInvalidNotificationData = errors.New("data notifikasi harus berupa JSON valid")

type NotificationService struct {
	notifRepo    repository.NotificationRepository
	customerRepo repository.CustomerRepository
	waClient     *whatsapp.Client
	fcmClient    *messaging.Client
	fcmEnabled   bool
}

// NewNotificationService wires the notification service. When fcmEnabled is true
// and a valid service-account credentials file is provided, it initializes the
// Firebase Cloud Messaging HTTP v1 client. The legacy FCM server-key API was shut
// down by Google in June 2024, so v1 (OAuth2 service account) is the only path.
// If initialization fails, FCM is disabled (logged) and the service still works
// for in-app storage + WhatsApp delivery.
func NewNotificationService(notifRepo repository.NotificationRepository, customerRepo repository.CustomerRepository, fcmProjectID, fcmCredentialsFile string, fcmEnabled bool) *NotificationService {
	s := &NotificationService{
		notifRepo:    notifRepo,
		customerRepo: customerRepo,
	}

	if fcmEnabled {
		client, err := initFCMClient(fcmProjectID, fcmCredentialsFile)
		if err != nil {
			log.Printf("[notification] FCM dinonaktifkan: gagal inisialisasi client: %v", err)
		} else {
			s.fcmClient = client
			s.fcmEnabled = true
			log.Printf("[notification] FCM HTTP v1 aktif (project: %s)", fcmProjectID)
		}
	}

	return s
}

// initFCMClient builds a Firebase Cloud Messaging client from a service-account JSON file.
func initFCMClient(projectID, credentialsFile string) (*messaging.Client, error) {
	if credentialsFile == "" {
		return nil, fmt.Errorf("FCM_CREDENTIALS_FILE kosong")
	}
	cfg := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(context.Background(), cfg, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("init messaging client: %w", err)
	}
	return client, nil
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

// PushAndStore stores an in-app notification and sends a web-push (FCM) to the
// customer's registered devices. It deliberately does NOT send WhatsApp — use it
// for events that already deliver their own WA message (invoice created, payment,
// isolir) so the customer doesn't get a duplicate WA. The in-app row is stored
// even when FCM is disabled, so it still builds notification history.
func (s *NotificationService) PushAndStore(ctx context.Context, tenantID, customerID, title, body, data string) error {
	if customerID == "" {
		return nil
	}
	if data != "" && !json.Valid([]byte(data)) {
		data = ""
	}

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

	if s.fcmEnabled {
		devices, err := s.notifRepo.GetDevicesByCustomer(ctx, tenantID, customerID)
		if err != nil {
			log.Printf("[notification] gagal ambil devices FCM untuk customer %s: %v", customerID, err)
		} else {
			for _, d := range devices {
				s.sendFCM(ctx, d.TenantID, d.CustomerID, d.FCMToken, title, body, data)
			}
		}
	}
	return nil
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
	if s.fcmEnabled {
		devices, err := s.notifRepo.GetDevicesByCustomer(ctx, tenantID, customerID)
		if err != nil {
			log.Printf("[notification] gagal ambil devices FCM untuk customer %s: %v", customerID, err)
		} else {
			for _, d := range devices {
				s.sendFCM(ctx, d.TenantID, d.CustomerID, d.FCMToken, title, body, data)
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
		if s.sendFCM(ctx, d.TenantID, d.CustomerID, d.FCMToken, title, body, data) {
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

// sendFCM delivers a single push via FCM HTTP v1. Returns true on success.
// On an unregistered/invalid token it deactivates the device so stale tokens
// stop being retried on every send.
func (s *NotificationService) sendFCM(ctx context.Context, tenantID, customerID, token, title, body, data string) bool {
	if s.fcmClient == nil {
		return false
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: title,
				Body:  body,
				Icon:  "/favicon.svg",
			},
		},
	}
	if data != "" {
		msg.Data = map[string]string{"payload": data}
	}

	if _, err := s.fcmClient.Send(ctx, msg); err != nil {
		if messaging.IsUnregistered(err) {
			// Token no longer valid — deactivate so we stop retrying it.
			if uerr := s.notifRepo.UnregisterDevice(ctx, tenantID, customerID, token); uerr != nil {
				log.Printf("[notification] gagal menonaktifkan token mati: %v", uerr)
			}
		} else {
			log.Printf("[notification] FCM send gagal (customer %s): %v", customerID, err)
		}
		return false
	}
	return true
}
