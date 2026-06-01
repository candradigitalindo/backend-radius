package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrReminderNotFound = errors.New("Pengingat tidak ditemukan")
)

type ReminderService struct {
	reminderRepo repository.ReminderRepository
	invoiceRepo  repository.InvoiceRepository
	customerRepo repository.CustomerRepository
	tenantRepo   repository.TenantRepository
	settingRepo  repository.SettingRepository
	paymentRepo  repository.PaymentRepository
	waClient     *whatsapp.Client
}

func NewReminderService(
	reminderRepo repository.ReminderRepository,
	invoiceRepo repository.InvoiceRepository,
	customerRepo repository.CustomerRepository,
	tenantRepo repository.TenantRepository,
	waClient *whatsapp.Client,
) *ReminderService {
	return &ReminderService{
		reminderRepo: reminderRepo,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		tenantRepo:   tenantRepo,
		waClient:     waClient,
	}
}

func (s *ReminderService) WithSettingRepo(repo repository.SettingRepository) *ReminderService {
	s.settingRepo = repo
	return s
}

func (s *ReminderService) WithPaymentRepo(repo repository.PaymentRepository) *ReminderService {
	s.paymentRepo = repo
	return s
}

type CreateReminderInput struct {
	TenantID        string
	Name            string
	Type            string
	DaysOffset      int
	MessageTemplate string
	IsActive        bool
}

type UpdateReminderInput struct {
	Name            string
	Type            string
	DaysOffset      int
	MessageTemplate string
	IsActive        bool
}

func (s *ReminderService) Create(ctx context.Context, input CreateReminderInput) (*model.Reminder, error) {
	reminder := &model.Reminder{
		TenantID:        input.TenantID,
		Name:            input.Name,
		Type:            input.Type,
		DaysOffset:      input.DaysOffset,
		MessageTemplate: input.MessageTemplate,
		IsActive:        input.IsActive,
	}

	if err := s.reminderRepo.Create(ctx, reminder); err != nil {
		return nil, err
	}

	return s.reminderRepo.FindByID(ctx, input.TenantID, reminder.ID)
}

func (s *ReminderService) GetByID(ctx context.Context, tenantID, reminderID string) (*model.Reminder, error) {
	r, err := s.reminderRepo.FindByID(ctx, tenantID, reminderID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReminderNotFound
	}
	return r, nil
}

func (s *ReminderService) Update(ctx context.Context, tenantID, reminderID string, input UpdateReminderInput) (*model.Reminder, error) {
	r, err := s.reminderRepo.FindByID(ctx, tenantID, reminderID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReminderNotFound
	}

	r.Name = input.Name
	r.Type = input.Type
	r.DaysOffset = input.DaysOffset
	r.MessageTemplate = input.MessageTemplate
	r.IsActive = input.IsActive

	if err := s.reminderRepo.Update(ctx, r); err != nil {
		return nil, err
	}

	return s.reminderRepo.FindByID(ctx, tenantID, reminderID)
}

func (s *ReminderService) Delete(ctx context.Context, tenantID, reminderID string) error {
	r, err := s.reminderRepo.FindByID(ctx, tenantID, reminderID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrReminderNotFound
	}
	return s.reminderRepo.Delete(ctx, tenantID, reminderID)
}

func (s *ReminderService) List(ctx context.Context, tenantID string, filter repository.ReminderFilter) ([]model.Reminder, int, error) {
	return s.reminderRepo.List(ctx, tenantID, filter)
}

// TriggerReminders processes all active reminders for a tenant.
// It finds unpaid invoices that match each reminder's criteria and sends WhatsApp messages.
func (s *ReminderService) TriggerReminders(ctx context.Context, tenantID string) (*TriggerResult, error) {
	reminders, err := s.reminderRepo.ListActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(reminders) == 0 {
		return &TriggerResult{Message: "No active reminders configured"}, nil
	}

	// Check once whether this tenant has an active payment gateway configured.
	hasPG := false
	if s.paymentRepo != nil && s.tenantRepo != nil {
		tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
		if err == nil && tenant != nil && tenant.PGAPIKey != "" {
			hasPG = true
		}
	}

	result := &TriggerResult{}
	today := time.Now().Truncate(24 * time.Hour)

	for _, reminder := range reminders {
		// Calculate the target due date based on reminder type and offset
		var targetDate time.Time
		switch reminder.Type {
		case "before_due":
			// Send X days before due date -> look for invoices due in X days
			targetDate = today.AddDate(0, 0, reminder.DaysOffset)
		case "on_due":
			// Send on due date
			targetDate = today
		case "after_due":
			// Send X days after due date -> look for invoices that were due X days ago
			targetDate = today.AddDate(0, 0, -reminder.DaysOffset)
		default:
			continue
		}

		dateStr := targetDate.Format("2006-01-02")
		invoices, err := s.invoiceRepo.ListUnpaidDue(ctx, tenantID, dateStr)
		if err != nil {
			log.Printf("[reminder] failed to list unpaid invoices for %s: %v", dateStr, err)
			continue
		}

		if len(invoices) == 0 {
			continue
		}

		// Only look up payment URLs when the template actually uses the variable.
		needsPaymentURL := strings.Contains(reminder.MessageTemplate, "{payment_url}")

		// Build reminder items for WhatsApp
		monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
		salam := greetingByTime()

		var reminderItems []whatsapp.ReminderItem
		for _, inv := range invoices {
			if inv.Customer == nil || inv.Customer.Phone == "" {
				result.Skipped++
				continue
			}

			periode := fmt.Sprintf("%s %d", monthNames[inv.PeriodMonth], inv.PeriodYear)
			paket := "-"
			if inv.Customer.Package != nil {
				paket = inv.Customer.Package.Name
			}

			// Resolve payment URL only when needed and PG is active.
			paymentURL := ""
			if needsPaymentURL && hasPG {
				if url, err := s.paymentRepo.FindActivePaymentURL(ctx, tenantID, inv.ID); err == nil {
					paymentURL = url
				} else {
					log.Printf("[reminder] failed to fetch payment URL for invoice %s: %v", inv.ID, err)
				}
			}

			msg := strings.ReplaceAll(reminder.MessageTemplate, "{salam}", salam)
			msg = strings.ReplaceAll(msg, "{nama}", inv.Customer.Name)
			msg = strings.ReplaceAll(msg, "{jumlah}", formatInvoiceAmount(inv.TotalAmount))
			msg = strings.ReplaceAll(msg, "{nomor_invoice}", inv.InvoiceNumber)
			msg = strings.ReplaceAll(msg, "{jatuh_tempo}", inv.DueDate.Format("02/01/2006"))
			msg = strings.ReplaceAll(msg, "{periode}", periode)
			msg = strings.ReplaceAll(msg, "{paket}", paket)
			msg = strings.ReplaceAll(msg, "{kode_pelanggan}", inv.Customer.CustomerCode)
			msg = strings.ReplaceAll(msg, "{alamat}", inv.Customer.Address)
			msg = strings.ReplaceAll(msg, "{payment_url}", paymentURL)

			// Attach CTA button when a payment URL is available.
			var ctaButton *whatsapp.CtaButton
			if paymentURL != "" {
				ctaButton = &whatsapp.CtaButton{Text: "💳 Bayar Online", URL: paymentURL}
			}

			reminderItems = append(reminderItems, whatsapp.ReminderItem{
				Phone:         inv.Customer.Phone,
				CustomerName:  inv.Customer.Name,
				InvoiceNumber: inv.InvoiceNumber,
				Amount:        inv.TotalAmount,
				DueDate:       inv.DueDate.Format("02/01/2006"),
				Message:       msg,
				CtaButton:     ctaButton,
			})
		}

		if len(reminderItems) == 0 {
			continue
		}

		// Send via WhatsApp service (async)
		waSession := WASessionForTenant(ctx, tenantID, s.settingRepo)
		go func(items []whatsapp.ReminderItem, reminderName, session string) {
			waResult, err := s.waClient.SendReminders(context.Background(), session, items)
			if err != nil {
				log.Printf("[reminder] WA send failed for reminder %q: %v", reminderName, err)
				return
			}
			log.Printf("[reminder] %q: sent=%d, failed=%d", reminderName, waResult.Success, waResult.Failed)
		}(reminderItems, reminder.Name, waSession)

		result.TotalSent += len(reminderItems)
	}

	result.Message = fmt.Sprintf("Memicu %d pengingat, %d pesan diantrekan, %d dilewati", len(reminders), result.TotalSent, result.Skipped)
	return result, nil
}

type TriggerResult struct {
	TotalSent int    `json:"total_sent"`
	Skipped   int    `json:"skipped"`
	Message   string `json:"message"`
}
