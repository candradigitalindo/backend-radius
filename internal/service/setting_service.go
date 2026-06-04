package service

import (
	"context"
	"errors"
	"strings"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrSettingNotFound   = errors.New("Pengaturan tidak ditemukan")
	ErrSettingKeyInvalid = errors.New("Key pengaturan tidak diizinkan")
	ErrSettingValueLimit = errors.New("Nilai pengaturan terlalu panjang")
)

// defaultSettingValues holds the default value returned when a key exists in the
// allowlist but has not been saved to the database yet.
var defaultSettingValues = map[string]string{
	"wa_notification_sender": "own",
}

// allowedSettingKeys whitelist of valid setting keys.
var allowedSettingKeys = map[string]bool{
	"billing_auto_generate_invoice": true,
	"billing_auto_isolate":          true,
	"isolir_page_title":             true,
	"isolir_page_message":           true,
	"isolir_page_bg_color":          true,
	"isolir_page_text_color":        true,
	"isolir_page_logo":              true,
	"isolir_page_contact_phone":     true,
	"isolir_page_contact_wa":        true,
	"isolir_page_payment_url":       true,
	"isolir_page_dark_mode":         true,
	"isolir_page_custom_css":        true,
	"app_theme":                     true,
	"app_language":                  true,
	"sa_wa_notify_subscribe":        true,
	"sa_wa_notify_payment":          true,
	"sa_wa_notify_due_date":         true,
	"sa_wa_notify_otp":              true,
	"sa_wa_notify_broadcast":        true,
	"wa_notification_sender":        true, // "own" | "superadmin"
	// Superadmin payment gateway
	"sa_pg_provider":    true,
	"sa_pg_api_key":     true,
	"sa_pg_secret_key":  true,
	"sa_pg_merchant_id": true,
	"sa_pg_sandbox":     true,
	// Superadmin general
	"sa_timezone":  true,
	"sa_app_theme": true,
}

const maxSettingValueLen = 10000

type SettingService struct {
	settingRepo repository.SettingRepository
}

func NewSettingService(settingRepo repository.SettingRepository) *SettingService {
	return &SettingService{settingRepo: settingRepo}
}

func (s *SettingService) Set(ctx context.Context, tenantID, key, value string) error {
	key = strings.TrimSpace(key)
	if !allowedSettingKeys[key] {
		return ErrSettingKeyInvalid
	}
	if len(value) > maxSettingValueLen {
		return ErrSettingValueLimit
	}
	setting := &model.Setting{
		TenantID: tenantID,
		Key:      key,
		Value:    value,
	}
	return s.settingRepo.Upsert(ctx, setting)
}

func (s *SettingService) Get(ctx context.Context, tenantID, key string) (*model.Setting, error) {
	setting, err := s.settingRepo.FindByKey(ctx, tenantID, key)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		if defaultVal, ok := defaultSettingValues[key]; ok {
			return &model.Setting{TenantID: tenantID, Key: key, Value: defaultVal}, nil
		}
		return nil, ErrSettingNotFound
	}
	return setting, nil
}

func (s *SettingService) Delete(ctx context.Context, tenantID, key string) error {
	setting, err := s.settingRepo.FindByKey(ctx, tenantID, key)
	if err != nil {
		return err
	}
	if setting == nil {
		return ErrSettingNotFound
	}
	return s.settingRepo.Delete(ctx, tenantID, key)
}

func (s *SettingService) List(ctx context.Context, tenantID string) ([]model.Setting, error) {
	return s.settingRepo.List(ctx, tenantID)
}

func (s *SettingService) BulkSet(ctx context.Context, tenantID string, settings map[string]string) error {
	for k, v := range settings {
		if !allowedSettingKeys[strings.TrimSpace(k)] {
			return ErrSettingKeyInvalid
		}
		if len(v) > maxSettingValueLen {
			return ErrSettingValueLimit
		}
	}
	return s.settingRepo.BulkUpsert(ctx, tenantID, settings)
}

func (s *SettingService) GetAsMap(ctx context.Context, tenantID string) (map[string]string, error) {
	settings, err := s.settingRepo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// WASessionForTenant returns the WA session ID to use when sending notifications to customers.
// Returns "superadmin" if the tenant chose to use the platform WA, otherwise returns the tenantID.
func WASessionForTenant(ctx context.Context, tenantID string, settingRepo repository.SettingRepository) string {
	if settingRepo == nil {
		return tenantID
	}
	setting, err := settingRepo.FindByKey(ctx, tenantID, "wa_notification_sender")
	if err == nil && setting != nil && setting.Value == "superadmin" {
		return "superadmin"
	}
	return tenantID
}
