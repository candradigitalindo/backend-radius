package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/billing"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrInvoiceNotFound           = errors.New("Faktur tidak ditemukan")
	ErrInvoiceAlreadyPaid        = errors.New("Faktur sudah dibayar")
	ErrInvoiceExists             = errors.New("Faktur sudah ada untuk periode ini")
	ErrPaymentNotFound           = errors.New("Pembayaran tidak ditemukan")
	ErrInvoiceNotOwnedByCustomer = errors.New("Faktur bukan milik pelanggan ini")
)

type InvoiceService struct {
	invoiceRepo  repository.InvoiceRepository
	paymentRepo  repository.PaymentRepository
	customerRepo repository.CustomerRepository
	reminderRepo repository.ReminderRepository
	tenantRepo   repository.TenantRepository
	rewardSvc    *RewardService
	waClient     *whatsapp.Client
	baseURL      string // Base URL for constructing callback URLs (e.g. https://api.example.com)
}

func NewInvoiceService(
	invoiceRepo repository.InvoiceRepository,
	paymentRepo repository.PaymentRepository,
	customerRepo repository.CustomerRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  invoiceRepo,
		paymentRepo:  paymentRepo,
		customerRepo: customerRepo,
	}
}

// WithWAClient sets the WhatsApp client for sending invoice notifications.
func (s *InvoiceService) WithWAClient(waClient *whatsapp.Client) *InvoiceService {
	s.waClient = waClient
	return s
}

// WithReminderRepo sets the reminder repo for fetching notification templates.
func (s *InvoiceService) WithReminderRepo(reminderRepo repository.ReminderRepository) *InvoiceService {
	s.reminderRepo = reminderRepo
	return s
}

// WithTenantRepo returns a new InvoiceService copy with the tenant repo set.
// Used when payment gateway features need tenant PG credentials.
func (s *InvoiceService) WithTenantRepo(tenantRepo repository.TenantRepository) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  s.invoiceRepo,
		paymentRepo:  s.paymentRepo,
		customerRepo: s.customerRepo,
		reminderRepo: s.reminderRepo,
		tenantRepo:   tenantRepo,
		rewardSvc:    s.rewardSvc,
		waClient:     s.waClient,
		baseURL:      s.baseURL,
	}
}

// WithBaseURL returns a new InvoiceService copy with the base URL set.
func (s *InvoiceService) WithBaseURL(baseURL string) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  s.invoiceRepo,
		paymentRepo:  s.paymentRepo,
		customerRepo: s.customerRepo,
		reminderRepo: s.reminderRepo,
		tenantRepo:   s.tenantRepo,
		rewardSvc:    s.rewardSvc,
		waClient:     s.waClient,
		baseURL:      baseURL,
	}
}

// WithReward sets the RewardService for post-payment reward automation.
func (s *InvoiceService) WithReward(rewardSvc *RewardService) *InvoiceService {
	s.rewardSvc = rewardSvc
	return s
}

// WebhookURLs returns the full webhook callback URLs for the tenant to register
// in their payment gateway dashboard.
func (s *InvoiceService) WebhookURLs() map[string]string {
	return map[string]string{
		"tripay":   s.baseURL + "/api/v1/webhooks/tripay",
		"midtrans": s.baseURL + "/api/v1/webhooks/midtrans",
	}
}

type CreateInvoiceInput struct {
	TenantID       string
	CustomerID     string
	PeriodMonth    int
	PeriodYear     int
	PackagePrice   int64
	Discount       int64
	AdditionalFee  int64
	FeeDescription string
	DueDate        time.Time
	Notes          string
}

type UpdateInvoiceInput struct {
	PackagePrice   int64
	Discount       int64
	AdditionalFee  int64
	FeeDescription string
	DueDate        time.Time
	Notes          string
}

type RecordPaymentInput struct {
	TenantID      string
	InvoiceID     string
	Amount        int64
	PaymentMethod string
	CollectedBy   string
	Notes         string
}

func (s *InvoiceService) Create(ctx context.Context, input CreateInvoiceInput) (*model.Invoice, error) {
	// Validate that the customer belongs to this tenant
	customer, err := s.customerRepo.FindByID(ctx, input.TenantID, input.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	existing, err := s.invoiceRepo.FindByCustomerPeriod(ctx, input.TenantID, input.CustomerID, input.PeriodMonth, input.PeriodYear)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrInvoiceExists
	}

	totalAmount := input.PackagePrice - input.Discount + input.AdditionalFee
	// Add 4-char random suffix to avoid collision when two requests hit at the same second
	invoiceNumber := fmt.Sprintf("%04d%02d-%s%04x", input.PeriodYear, input.PeriodMonth, time.Now().Format("150405"), rand.Intn(0x10000))

	invoice := &model.Invoice{
		TenantID:       input.TenantID,
		CustomerID:     input.CustomerID,
		InvoiceNumber:  invoiceNumber,
		PeriodMonth:    input.PeriodMonth,
		PeriodYear:     input.PeriodYear,
		PackagePrice:   input.PackagePrice,
		Discount:       input.Discount,
		AdditionalFee:  input.AdditionalFee,
		FeeDescription: input.FeeDescription,
		TotalAmount:    totalAmount,
		Status:         "unpaid",
		DueDate:        input.DueDate,
		Notes:          input.Notes,
		AutoGenerated:  false,
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	return s.invoiceRepo.FindByID(ctx, input.TenantID, invoice.ID)
}

func (s *InvoiceService) GetByID(ctx context.Context, tenantID, invoiceID string) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	return invoice, nil
}

func (s *InvoiceService) Update(ctx context.Context, tenantID, invoiceID string, input UpdateInvoiceInput) (*model.Invoice, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	if invoice.Status == "paid" {
		return nil, ErrInvoiceAlreadyPaid
	}

	invoice.PackagePrice = input.PackagePrice
	invoice.Discount = input.Discount
	invoice.AdditionalFee = input.AdditionalFee
	invoice.FeeDescription = input.FeeDescription
	invoice.TotalAmount = input.PackagePrice - input.Discount + input.AdditionalFee
	invoice.DueDate = input.DueDate
	invoice.Notes = input.Notes

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
}

func (s *InvoiceService) Delete(ctx context.Context, tenantID, invoiceID string) error {
	invoice, err := s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if invoice == nil {
		return ErrInvoiceNotFound
	}
	if invoice.Status == "paid" {
		return ErrInvoiceAlreadyPaid
	}
	return s.invoiceRepo.Delete(ctx, tenantID, invoiceID)
}

func (s *InvoiceService) List(ctx context.Context, tenantID string, filter repository.InvoiceFilter) ([]model.Invoice, int, error) {
	return s.invoiceRepo.List(ctx, tenantID, filter)
}

func (s *InvoiceService) ListByCustomer(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Invoice, int, error) {
	return s.invoiceRepo.ListByCustomer(ctx, tenantID, customerID, page, perPage)
}

// GenerateMonthly generates invoices for a specific due period.
// Rules:
//   - billing_date = invoice date day-of-month
//   - billing_deadline = payment due date day-of-month
//   - For "fixed" billing type: schedule uses the stored customer billing_date and billing_deadline
//   - For other billing types: if customer paid late, next due date shifts to the paid date's day and invoice date follows that shift
//   - If customer has unpaid invoice, skip (no new invoice)
func (s *InvoiceService) GenerateMonthly(ctx context.Context, tenantID string, month, year, _ int) (int, error) {
	now := time.Now()

	customers, _, err := s.customerRepo.List(ctx, tenantID, repository.CustomerFilter{
		Status:  "active",
		Page:    1,
		PerPage: 100000,
	})
	if err != nil {
		return 0, err
	}

	var tenant *model.Tenant
	if s.tenantRepo != nil {
		if t, err := s.tenantRepo.FindByID(ctx, tenantID); err == nil {
			tenant = t
		}
	}

	var invoiceItems []invoiceWithCustomer
	for _, c := range customers {
		invoiceDay, deadlineDay, err := s.resolveCustomerBillingSchedule(ctx, tenantID, c, tenant)
		if err != nil {
			return 0, err
		}

		// 1. Skip if customer already has an invoice for this period (paid or unpaid)
		existingInv, err := s.invoiceRepo.FindByCustomerPeriod(ctx, tenantID, c.ID, month, year)
		if err != nil {
			return 0, err
		}
		if existingInv != nil {
			continue
		}

		// 2. Build invoice date and due date for this period using the explicit billing schedule.
		invoiceDate, dueDate := billing.CycleDates(invoiceDay, deadlineDay, month, year)

		// 3. Only generate if we've reached the invoice date
		if now.Before(invoiceDate) {
			continue
		}

		// 4. Build invoice
		price := int64(0)
		if c.Package != nil {
			price = c.Package.Price
		}
		if c.CustomPrice != nil {
			price = *c.CustomPrice
		}

		totalAmount := price - c.Discount + c.AdditionalFee

		// 5. Skip free packages (total = 0) — no invoice needed
		if totalAmount <= 0 {
			continue
		}

		invoiceItems = append(invoiceItems, invoiceWithCustomer{
			invoice: model.Invoice{
				TenantID:       tenantID,
				CustomerID:     c.ID,
				InvoiceNumber:  fmt.Sprintf("%04d%02d-%s", year, month, c.CustomerCode),
				PeriodMonth:    month,
				PeriodYear:     year,
				PackagePrice:   price,
				Discount:       c.Discount,
				AdditionalFee:  c.AdditionalFee,
				FeeDescription: c.FeeDescription,
				TotalAmount:    totalAmount,
				Status:         "unpaid",
				DueDate:        dueDate,
				AutoGenerated:  true,
			},
			customer: c,
		})
	}

	// Extract invoices for batch insert
	var invoices []model.Invoice
	for _, item := range invoiceItems {
		invoices = append(invoices, item.invoice)
	}

	if len(invoices) == 0 {
		return 0, nil
	}

	if err := s.invoiceRepo.CreateBatch(ctx, invoices); err != nil {
		return 0, err
	}

	// Send WA notification for newly created invoices (async)
	if s.waClient != nil && len(invoiceItems) > 0 {
		go s.sendInvoiceCreatedNotifications(tenantID, invoiceItems)
	}

	return len(invoices), nil
}

// GenerateScheduled generates invoices for the billing cycle that is active today for each customer.
// This is used by the daily worker so explicit billing_date values such as 24 -> due 1 are respected.
func (s *InvoiceService) GenerateScheduled(ctx context.Context, tenantID string, now time.Time) (int, error) {
	customers, _, err := s.customerRepo.List(ctx, tenantID, repository.CustomerFilter{
		Status:  "active",
		Page:    1,
		PerPage: 100000,
	})
	if err != nil {
		return 0, err
	}

	var tenant *model.Tenant
	if s.tenantRepo != nil {
		if t, err := s.tenantRepo.FindByID(ctx, tenantID); err == nil {
			tenant = t
		}
	}

	var invoiceItems []invoiceWithCustomer
	for _, c := range customers {
		invoiceDay, deadlineDay, err := s.resolveCustomerBillingSchedule(ctx, tenantID, c, tenant)
		if err != nil {
			return 0, err
		}

		month, year := billing.CurrentDuePeriod(now, invoiceDay, deadlineDay)
		existingInv, err := s.invoiceRepo.FindByCustomerPeriod(ctx, tenantID, c.ID, month, year)
		if err != nil {
			return 0, err
		}
		if existingInv != nil {
			continue
		}

		invoiceDate, dueDate := billing.CycleDates(invoiceDay, deadlineDay, month, year)
		if now.Before(invoiceDate) {
			continue
		}

		price := int64(0)
		if c.Package != nil {
			price = c.Package.Price
		}
		if c.CustomPrice != nil {
			price = *c.CustomPrice
		}

		totalAmount := price - c.Discount + c.AdditionalFee
		if totalAmount <= 0 {
			continue
		}

		invoiceItems = append(invoiceItems, invoiceWithCustomer{
			invoice: model.Invoice{
				TenantID:       tenantID,
				CustomerID:     c.ID,
				InvoiceNumber:  fmt.Sprintf("%04d%02d-%s", year, month, c.CustomerCode),
				PeriodMonth:    month,
				PeriodYear:     year,
				PackagePrice:   price,
				Discount:       c.Discount,
				AdditionalFee:  c.AdditionalFee,
				FeeDescription: c.FeeDescription,
				TotalAmount:    totalAmount,
				Status:         "unpaid",
				DueDate:        dueDate,
				AutoGenerated:  true,
			},
			customer: c,
		})
	}

	var invoices []model.Invoice
	for _, item := range invoiceItems {
		invoices = append(invoices, item.invoice)
	}

	if len(invoices) == 0 {
		return 0, nil
	}

	if err := s.invoiceRepo.CreateBatch(ctx, invoices); err != nil {
		return 0, err
	}

	if s.waClient != nil && len(invoiceItems) > 0 {
		go s.sendInvoiceCreatedNotifications(tenantID, invoiceItems)
	}

	return len(invoices), nil
}

// resolveCustomerBillingSchedule returns (invoiceDay, deadlineDay) for a customer.
// For "cycle" billing type the dates come from the tenant-level billing_cycle / due_day
// so every cycle customer on the same tenant shares the same billing schedule.
// For "fixed" billing type the per-customer billing_date / billing_deadline are used.
func (s *InvoiceService) resolveCustomerBillingSchedule(ctx context.Context, _ string, customer model.Customer, tenant *model.Tenant) (int, int, error) {
	if billing.NormalizeBillingType(customer.BillingType) == "cycle" && tenant != nil {
		deadlineDay := billing.NormalizeDay(tenant.DueDay)
		if deadlineDay == 0 {
			deadlineDay = billing.NormalizeDay(customer.JoinDate.Day())
		}
		invoiceDay := billing.NormalizeDay(tenant.BillingCycle)
		if invoiceDay == 0 {
			invoiceDay = billing.InvoiceDayFromDueDay(deadlineDay)
		}
		return invoiceDay, deadlineDay, nil
	}

	// Fixed billing (and any unrecognised type): use per-customer stored dates.
	deadlineDay := customer.BillingDeadline
	if deadlineDay == 0 {
		deadlineDay = customer.JoinDate.Day()
	}
	deadlineDay = billing.NormalizeDay(deadlineDay)

	invoiceDay := customer.BillingDate
	if invoiceDay == 0 {
		invoiceDay = billing.InvoiceDayFromDueDay(deadlineDay)
	}

	return invoiceDay, deadlineDay, nil
}

// sendInvoiceCreatedNotifications sends WhatsApp messages for newly created invoices.

type invoiceWithCustomer struct {
	invoice  model.Invoice
	customer model.Customer
}

func (s *InvoiceService) sendInvoiceCreatedNotifications(tenantID string, items []invoiceWithCustomer) {
	ctx := context.Background()

	// Get the "invoice_created" reminder template
	var messageTemplate string
	if s.reminderRepo != nil {
		reminder, err := s.reminderRepo.FindActiveByType(ctx, tenantID, "invoice_created")
		if err != nil {
			log.Printf("[invoice-wa] failed to get template: %v", err)
		}
		if reminder != nil {
			messageTemplate = reminder.MessageTemplate
		}
	}

	if messageTemplate == "" {
		messageTemplate = `{salam} {nama},

Berikut informasi tagihan internet Anda:

📋 *No. Invoice:* {nomor_invoice}
📅 *Periode:* {periode}
📦 *Paket:* {paket}
💰 *Total Tagihan:* Rp{jumlah}
⏰ *Jatuh Tempo:* {jatuh_tempo}

Mohon segera lakukan pembayaran sebelum jatuh tempo untuk menghindari pemutusan layanan.

Terima kasih. 🙏`
	}

	monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	salam := greetingByTime()

	var reminderItems []whatsapp.ReminderItem
	for _, item := range items {
		if item.customer.Phone == "" {
			continue
		}

		periode := fmt.Sprintf("%s %d", monthNames[item.invoice.PeriodMonth], item.invoice.PeriodYear)
		paket := "-"
		if item.customer.Package != nil {
			paket = item.customer.Package.Name
		}

		msg := strings.ReplaceAll(messageTemplate, "{salam}", salam)
		msg = strings.ReplaceAll(msg, "{nama}", item.customer.Name)
		msg = strings.ReplaceAll(msg, "{jumlah}", formatInvoiceAmount(item.invoice.TotalAmount))
		msg = strings.ReplaceAll(msg, "{nomor_invoice}", item.invoice.InvoiceNumber)
		msg = strings.ReplaceAll(msg, "{jatuh_tempo}", item.invoice.DueDate.Format("02/01/2006"))
		msg = strings.ReplaceAll(msg, "{periode}", periode)
		msg = strings.ReplaceAll(msg, "{paket}", paket)
		msg = strings.ReplaceAll(msg, "{kode_pelanggan}", item.customer.CustomerCode)
		msg = strings.ReplaceAll(msg, "{alamat}", item.customer.Address)

		reminderItems = append(reminderItems, whatsapp.ReminderItem{
			Phone:         item.customer.Phone,
			CustomerName:  item.customer.Name,
			InvoiceNumber: item.invoice.InvoiceNumber,
			Amount:        item.invoice.TotalAmount,
			DueDate:       item.invoice.DueDate.Format("02/01/2006"),
			Message:       msg,
		})
	}

	if len(reminderItems) == 0 {
		return
	}

	result, err := s.waClient.SendReminders(ctx, tenantID, reminderItems)
	if err != nil {
		log.Printf("[invoice-wa] send failed: %v", err)
		return
	}
	log.Printf("[invoice-wa] sent=%d, failed=%d for tenant %s", result.Success, result.Failed, tenantID)
}

// formatInvoiceAmount formats amount with dot thousand separator (e.g. 200000 -> "200.000").
func formatInvoiceAmount(amount int64) string {
	neg := ""
	if amount < 0 {
		neg = "-"
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return neg + s
	}
	var result []byte
	for i, c := range s {
		if (n-i)%3 == 0 && i != 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return neg + string(result)
}

// greetingByTime returns a time-appropriate Indonesian greeting based on WIB (UTC+7).
func greetingByTime() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	hour := time.Now().In(loc).Hour()
	switch {
	case hour >= 5 && hour < 11:
		return "Selamat Pagi"
	case hour >= 11 && hour < 15:
		return "Selamat Siang"
	case hour >= 15 && hour < 18:
		return "Selamat Sore"
	default:
		return "Selamat Malam"
	}
}

// sendPaymentNotification sends a WhatsApp payment confirmation to the customer.
func (s *InvoiceService) sendPaymentNotification(ctx context.Context, tenantID string, invoice *model.Invoice, paymentMethod string) {
	if s.waClient == nil {
		return
	}

	// Load full customer info (phone, package, address)
	customer, err := s.customerRepo.FindByID(ctx, tenantID, invoice.CustomerID)
	if err != nil || customer == nil || customer.Phone == "" {
		return
	}

	monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	salam := greetingByTime()

	periode := fmt.Sprintf("%s %d", monthNames[invoice.PeriodMonth], invoice.PeriodYear)
	paket := "-"
	if customer.Package != nil {
		paket = customer.Package.Name
	}

	paidAt := time.Now().Format("02/01/2006 15:04")
	if invoice.PaidAt != nil {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		paidAt = invoice.PaidAt.In(loc).Format("02/01/2006 15:04")
	}

	amount := invoice.TotalAmount
	if invoice.PaidAmount != nil {
		amount = *invoice.PaidAmount
	}

	method := paymentMethod
	if method == "" {
		method = "Tunai"
	}

	msg := fmt.Sprintf(`%s %s,

Pembayaran Anda telah *berhasil* dikonfirmasi! ✅

📋 *No. Invoice:* %s
📅 *Periode:* %s
📦 *Paket:* %s
💰 *Jumlah Bayar:* Rp%s
💳 *Metode Bayar:* %s
🕐 *Waktu Bayar:* %s WIB
👤 *Kode Pelanggan:* %s

Terima kasih atas pembayaran Anda. Layanan internet Anda aktif dan dapat digunakan. 🙏

_Simpan pesan ini sebagai bukti pembayaran._`,
		salam, customer.Name,
		invoice.InvoiceNumber,
		periode,
		paket,
		formatInvoiceAmount(amount),
		method,
		paidAt,
		customer.CustomerCode,
	)

	result, err := s.waClient.SendMessage(ctx, tenantID, customer.Phone, msg)
	if err != nil {
		log.Printf("[payment-wa] send failed for %s: %v", customer.CustomerCode, err)
		return
	}
	log.Printf("[payment-wa] sent to %s (%s) status=%s", customer.Name, customer.Phone, result.Status)
}

// RecordPayment records a manual/cash payment for an invoice.
func (s *InvoiceService) RecordPayment(ctx context.Context, input RecordPaymentInput) (*model.Payment, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, input.TenantID, input.InvoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	if invoice.Status == "paid" {
		return nil, ErrInvoiceAlreadyPaid
	}

	now := time.Now()
	payment := &model.Payment{
		TenantID:      input.TenantID,
		InvoiceID:     input.InvoiceID,
		Amount:        input.Amount,
		PaymentMethod: input.PaymentMethod,
		Status:        "paid",
		PaidAt:        &now,
		CollectedBy:   &input.CollectedBy,
		Notes:         input.Notes,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	invoice.PaidAmount = &input.Amount
	invoice.PaymentMethod = &input.PaymentMethod
	if err := s.invoiceRepo.MarkPaid(ctx, invoice); err != nil {
		// Rollback: delete the orphan payment record to maintain consistency
		_ = s.paymentRepo.Delete(context.Background(), input.TenantID, payment.ID)
		return nil, err
	}

	// Auto-activate customer if currently isolated
	s.autoActivateCustomer(ctx, input.TenantID, invoice.CustomerID)

	// Process reward automation (referral qualification + loyalty check)
	if s.rewardSvc != nil {
		go s.rewardSvc.ProcessRewardOnPayment(context.Background(), input.TenantID, invoice.CustomerID)
	}

	// Send payment confirmation via WhatsApp
	go s.sendPaymentNotification(context.Background(), input.TenantID, invoice, input.PaymentMethod)

	return s.paymentRepo.FindByID(ctx, input.TenantID, payment.ID)
}

func (s *InvoiceService) GetPayments(ctx context.Context, tenantID, invoiceID string) ([]model.Payment, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	return s.paymentRepo.ListByInvoice(ctx, tenantID, invoiceID)
}

func (s *InvoiceService) ListPayments(ctx context.Context, tenantID string, filter repository.PaymentFilter) ([]model.Payment, int, error) {
	return s.paymentRepo.List(ctx, tenantID, filter)
}

// --- Payment Gateway Methods ---

type PaymentGatewayInput struct {
	TenantID      string
	InvoiceID     string
	PaymentMethod string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	ReturnURL     string
}

type PaymentGatewayResult struct {
	PaymentID    string     `json:"payment_id"`
	GatewayTrxID string     `json:"gateway_trx_id"`
	PaymentURL   string     `json:"payment_url"`
	ExpiredAt    *time.Time `json:"expired_at"`
}

// CreateGatewayPayment creates a Tripay transaction and records a pending payment.
func (s *InvoiceService) CreateGatewayPayment(ctx context.Context, input PaymentGatewayInput) (*PaymentGatewayResult, error) {
	if s.tenantRepo == nil {
		return nil, fmt.Errorf("tenant repo tidak dikonfigurasi pada invoice service")
	}

	invoice, err := s.invoiceRepo.FindByID(ctx, input.TenantID, input.InvoiceID)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}
	if invoice.Status == "paid" {
		return nil, ErrInvoiceAlreadyPaid
	}

	tenant, err := s.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant: %w", err)
	}
	if tenant == nil || tenant.PGAPIKey == "" {
		return nil, fmt.Errorf("payment gateway belum dikonfigurasi untuk tenant ini")
	}

	switch tenant.PGProvider {
	case "midtrans":
		return s.createMidtransPayment(ctx, tenant, invoice, input)
	default: // "tripay" or empty defaults to Tripay
		return s.createTripayPayment(ctx, tenant, invoice, input)
	}
}

func (s *InvoiceService) createTripayPayment(ctx context.Context, tenant *model.Tenant, invoice *model.Invoice, input PaymentGatewayInput) (*PaymentGatewayResult, error) {
	client := payment.NewTripayClient(
		tenant.PGAPIKey,
		tenant.PGSecretKey,
		tenant.PGMerchantID,
		false,
	)

	resp, err := client.CreateTransaction(ctx, payment.CreateTransactionRequest{
		Method:        input.PaymentMethod,
		MerchantRef:   invoice.InvoiceNumber,
		Amount:        invoice.TotalAmount,
		CustomerName:  input.CustomerName,
		CustomerEmail: input.CustomerEmail,
		CustomerPhone: input.CustomerPhone,
		ReturnURL:     input.ReturnURL,
		CallbackURL:   s.baseURL + "/api/v1/webhooks/tripay",
	})
	if err != nil {
		return nil, fmt.Errorf("create tripay transaction: %w", err)
	}

	expiredAt := time.Unix(resp.Data.ExpiredTime, 0)
	rawResp, _ := json.Marshal(resp.Data)

	paymentRecord := &model.Payment{
		TenantID:        input.TenantID,
		InvoiceID:       input.InvoiceID,
		Amount:          invoice.TotalAmount,
		PaymentMethod:   input.PaymentMethod,
		Gateway:         "tripay",
		GatewayTrxID:    resp.Data.Reference,
		GatewayStatus:   resp.Data.Status,
		GatewayResponse: rawResp,
		Status:          "pending",
		ExpiredAt:       &expiredAt,
	}

	if err := s.paymentRepo.Create(ctx, paymentRecord); err != nil {
		return nil, fmt.Errorf("save tripay payment record: %w", err)
	}

	return &PaymentGatewayResult{
		PaymentID:    paymentRecord.ID,
		GatewayTrxID: resp.Data.Reference,
		PaymentURL:   resp.Data.PaymentURL,
		ExpiredAt:    &expiredAt,
	}, nil
}

func (s *InvoiceService) createMidtransPayment(ctx context.Context, tenant *model.Tenant, invoice *model.Invoice, input PaymentGatewayInput) (*PaymentGatewayResult, error) {
	client := payment.NewMidtransClient(tenant.PGSecretKey, false)

	itemName := fmt.Sprintf("Invoice %s", invoice.InvoiceNumber)
	resp, err := client.CreateTransaction(ctx, payment.MidtransCreateRequest{
		OrderID:         invoice.InvoiceNumber,
		GrossAmount:     invoice.TotalAmount,
		FirstName:       input.CustomerName,
		Email:           input.CustomerEmail,
		Phone:           input.CustomerPhone,
		FinishURL:       input.ReturnURL,
		ErrorURL:        input.ReturnURL,
		ItemName:        itemName,
		NotificationURL: s.baseURL + "/api/v1/webhooks/midtrans",
	})
	if err != nil {
		return nil, fmt.Errorf("create midtrans transaction: %w", err)
	}

	expiredAt := time.Now().Add(24 * time.Hour)
	rawResp, _ := json.Marshal(resp)

	paymentRecord := &model.Payment{
		TenantID:        input.TenantID,
		InvoiceID:       input.InvoiceID,
		Amount:          invoice.TotalAmount,
		PaymentMethod:   input.PaymentMethod,
		Gateway:         "midtrans",
		GatewayTrxID:    invoice.InvoiceNumber, // order_id used as lookup key
		GatewayStatus:   "pending",
		GatewayResponse: rawResp,
		Status:          "pending",
		ExpiredAt:       &expiredAt,
	}

	if err := s.paymentRepo.Create(ctx, paymentRecord); err != nil {
		return nil, fmt.Errorf("save midtrans payment record: %w", err)
	}

	return &PaymentGatewayResult{
		PaymentID:    paymentRecord.ID,
		GatewayTrxID: invoice.InvoiceNumber,
		PaymentURL:   resp.RedirectURL,
		ExpiredAt:    &expiredAt,
	}, nil
}

// ProcessTripayWebhook handles a Tripay callback and marks the invoice paid on success.
func (s *InvoiceService) ProcessTripayWebhook(ctx context.Context, payload payment.TripayCallbackPayload) error {
	paymentRecord, err := s.paymentRepo.FindByGatewayTrxID(ctx, payload.Reference)
	if err != nil {
		return fmt.Errorf("find payment by trx_id: %w", err)
	}
	if paymentRecord == nil {
		return fmt.Errorf("pembayaran tidak ditemukan untuk referensi %s", payload.Reference)
	}

	// Verify HMAC signature using tenant's private key
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repo tidak dikonfigurasi, tidak dapat memverifikasi signature")
	}
	tenant, err := s.tenantRepo.FindByID(ctx, paymentRecord.TenantID)
	if err != nil {
		return fmt.Errorf("muat tenant untuk verifikasi signature: %w", err)
	}
	if tenant == nil || tenant.PGSecretKey == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi untuk tenant")
	}
	tripayClient := payment.NewTripayClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGMerchantID, false)
	if !tripayClient.VerifyWebhookSignature(payload) {
		return fmt.Errorf("signature webhook Tripay tidak valid")
	}

	switch payload.Status {
	case "PAID":
		if err := s.paymentRepo.UpdateStatus(ctx, paymentRecord.ID, "paid"); err != nil {
			return fmt.Errorf("update payment status: %w", err)
		}

		invoice, err := s.invoiceRepo.FindByID(ctx, paymentRecord.TenantID, paymentRecord.InvoiceID)
		if err != nil {
			return fmt.Errorf("load invoice: %w", err)
		}
		if invoice != nil && invoice.Status != "paid" {
			method := payload.PaymentMethod
			invoice.PaidAmount = &paymentRecord.Amount
			invoice.PaymentMethod = &method
			if err := s.invoiceRepo.MarkPaid(ctx, invoice); err != nil {
				return fmt.Errorf("mark invoice paid: %w", err)
			}

			// Auto-activate customer if currently isolated
			s.autoActivateCustomer(ctx, paymentRecord.TenantID, invoice.CustomerID)

			// Process reward automation
			if s.rewardSvc != nil {
				go s.rewardSvc.ProcessRewardOnPayment(context.Background(), paymentRecord.TenantID, invoice.CustomerID)
			}

			// Send payment confirmation via WhatsApp
			go s.sendPaymentNotification(context.Background(), paymentRecord.TenantID, invoice, method)
		}

	case "EXPIRED", "FAILED":
		if err := s.paymentRepo.UpdateStatus(ctx, paymentRecord.ID, "failed"); err != nil {
			return fmt.Errorf("update payment status to failed: %w", err)
		}
	}

	return nil
}

// ProcessMidtransWebhook handles a Midtrans HTTP notification and marks the invoice paid on success.
// Signature verification is performed using the tenant's server key.
func (s *InvoiceService) ProcessMidtransWebhook(ctx context.Context, n payment.MidtransNotification) error {
	// Look up payment by order_id (stored as GatewayTrxID for Midtrans)
	paymentRecord, err := s.paymentRepo.FindByGatewayTrxID(ctx, n.OrderID)
	if err != nil {
		return fmt.Errorf("find payment by order_id: %w", err)
	}
	if paymentRecord == nil {
		return fmt.Errorf("pembayaran tidak ditemukan untuk order_id %s", n.OrderID)
	}

	// Verify signature using tenant's server key (mandatory)
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repo tidak dikonfigurasi, tidak dapat memverifikasi signature")
	}
	tenant, err := s.tenantRepo.FindByID(ctx, paymentRecord.TenantID)
	if err != nil {
		return fmt.Errorf("muat tenant untuk verifikasi signature: %w", err)
	}
	if tenant == nil || tenant.PGSecretKey == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi untuk tenant")
	}
	midtransClient := payment.NewMidtransClient(tenant.PGSecretKey, false)
	if !midtransClient.VerifyWebhookSignature(n) {
		return fmt.Errorf("signature webhook Midtrans tidak valid")
	}

	if payment.IsPaymentSuccess(n) {
		if err := s.paymentRepo.UpdateStatus(ctx, paymentRecord.ID, "paid"); err != nil {
			return fmt.Errorf("update payment status: %w", err)
		}

		invoice, err := s.invoiceRepo.FindByID(ctx, paymentRecord.TenantID, paymentRecord.InvoiceID)
		if err != nil {
			return fmt.Errorf("load invoice: %w", err)
		}
		if invoice != nil && invoice.Status != "paid" {
			method := n.PaymentType
			amount, err := payment.GrossAmountInt64(n)
			if err != nil {
				return fmt.Errorf("parse gross amount: %w", err)
			}
			invoice.PaidAmount = &amount
			invoice.PaymentMethod = &method
			if err := s.invoiceRepo.MarkPaid(ctx, invoice); err != nil {
				return fmt.Errorf("mark invoice paid: %w", err)
			}

			// Auto-activate customer if currently isolated
			s.autoActivateCustomer(ctx, paymentRecord.TenantID, invoice.CustomerID)

			// Process reward automation
			if s.rewardSvc != nil {
				go s.rewardSvc.ProcessRewardOnPayment(context.Background(), paymentRecord.TenantID, invoice.CustomerID)
			}

			// Send payment confirmation via WhatsApp
			go s.sendPaymentNotification(context.Background(), paymentRecord.TenantID, invoice, method)
		}
	} else if payment.IsPaymentFailed(n) {
		if err := s.paymentRepo.UpdateStatus(ctx, paymentRecord.ID, "failed"); err != nil {
			return fmt.Errorf("update payment status to failed: %w", err)
		}
	}

	return nil
}

// autoActivateCustomer re-activates a customer if they are currently isolated.
// Errors are logged but not propagated — payment confirmation should not fail if activation fails.
func (s *InvoiceService) autoActivateCustomer(ctx context.Context, tenantID, customerID string) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil || customer == nil {
		return
	}
	if customer.Status == "isolated" {
		_ = s.customerRepo.SetIsolated(ctx, tenantID, customerID, nil)
	}
}

// --- Public payment page methods ---

// VerifyInvoiceOwnership checks that the given invoice belongs to the customer identified by customerCode.
func (s *InvoiceService) VerifyInvoiceOwnership(ctx context.Context, tenantID, customerCode, invoiceID string) error {
	customer, err := s.customerRepo.FindByCode(ctx, tenantID, customerCode)
	if err != nil {
		return fmt.Errorf("find customer: %w", err)
	}
	if customer == nil {
		return ErrCustomerNotFound
	}

	invoice, err := s.invoiceRepo.FindByID(ctx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("find invoice: %w", err)
	}
	if invoice == nil {
		return ErrInvoiceNotFound
	}

	if invoice.CustomerID != customer.ID {
		return ErrInvoiceNotOwnedByCustomer
	}
	return nil
}

type PublicCustomerInfo struct {
	CustomerCode string `json:"customer_code"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	PackageName  string `json:"package_name"`
	Status       string `json:"status"`
}

type PublicInvoiceInfo struct {
	InvoiceID     string `json:"invoice_id"`
	InvoiceNumber string `json:"invoice_number"`
	PeriodMonth   int    `json:"period_month"`
	PeriodYear    int    `json:"period_year"`
	TotalAmount   int64  `json:"total_amount"`
	Status        string `json:"status"`
	DueDate       string `json:"due_date"`
}

// GetPublicCustomerInfo returns limited customer info for the public payment page.
func (s *InvoiceService) GetPublicCustomerInfo(ctx context.Context, tenantID, customerCode string) (*PublicCustomerInfo, []PublicInvoiceInfo, error) {
	customer, err := s.customerRepo.FindByCode(ctx, tenantID, customerCode)
	if err != nil {
		return nil, nil, fmt.Errorf("find customer: %w", err)
	}
	if customer == nil {
		return nil, nil, ErrCustomerNotFound
	}

	info := &PublicCustomerInfo{
		CustomerCode: customer.CustomerCode,
		Name:         customer.Name,
		Phone:        customer.Phone,
		Status:       customer.Status,
	}
	if customer.Package != nil {
		info.PackageName = customer.Package.Name
	}

	invoices, _, err := s.invoiceRepo.ListByCustomer(ctx, tenantID, customer.ID, 1, 12)
	if err != nil {
		return nil, nil, fmt.Errorf("list invoices: %w", err)
	}

	var publicInvoices []PublicInvoiceInfo
	for _, inv := range invoices {
		if inv.Status == "paid" {
			continue
		}
		publicInvoices = append(publicInvoices, PublicInvoiceInfo{
			InvoiceID:     inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			PeriodMonth:   inv.PeriodMonth,
			PeriodYear:    inv.PeriodYear,
			TotalAmount:   inv.TotalAmount,
			Status:        inv.Status,
			DueDate:       inv.DueDate.Format("2006-01-02"),
		})
	}

	return info, publicInvoices, nil
}

// CheckPaymentStatus returns the current status of a gateway payment by trx ID.
func (s *InvoiceService) CheckPaymentStatus(ctx context.Context, gatewayTrxID string) (*model.Payment, error) {
	p, err := s.paymentRepo.FindByGatewayTrxID(ctx, gatewayTrxID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPaymentNotFound
	}
	return p, nil
}

// ExpireStalePayments marks pending gateway payments whose expired_at has passed as "expired".
func (s *InvoiceService) ExpireStalePayments(ctx context.Context) (int, error) {
	payments, err := s.paymentRepo.ListExpiredPending(ctx)
	if err != nil {
		return 0, fmt.Errorf("list expired pending: %w", err)
	}

	expired := 0
	for _, p := range payments {
		if err := s.paymentRepo.UpdateStatus(ctx, p.ID, "expired"); err != nil {
			continue
		}
		expired++
	}
	return expired, nil
}
