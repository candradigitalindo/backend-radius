package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type Handlers struct {
	DB              *pgxpool.Pool
	InvoiceService  *service.InvoiceService
	CustomerService *service.CustomerService
	ReminderService *service.ReminderService
	TenantRepo      repository.TenantRepository
	InvoiceRepo     repository.InvoiceRepository
	ReminderRepo    repository.ReminderRepository
	RouterRepo      repository.RouterRepository
	SessionRepo     repository.SessionRepository
	IPAMRepo        repository.IPAMRepository
	RouterService   *service.RouterService
	RewardService   *service.RewardService
	WAClient        *whatsapp.Client
	GenieACSService *service.GenieACSService
	ONTRepo         repository.ONTRepository
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGenerateInvoices, h.HandleGenerateInvoices)
	mux.HandleFunc(TaskAutoIsolir, h.HandleAutoIsolir)
	mux.HandleFunc(TaskTriggerReminders, h.HandleTriggerReminders)
	mux.HandleFunc(TaskExpirePayments, h.HandleExpirePayments)
	mux.HandleFunc(TaskRouterMonitor, h.HandleRouterMonitor)
	mux.HandleFunc(TaskCleanSessions, h.HandleCleanSessions)
	mux.HandleFunc(TaskONTProvisionRetry, h.HandleONTProvisionRetry)
	mux.HandleFunc(TaskONTDiscover, h.HandleONTDiscover)
	mux.HandleFunc(TaskONTAutoMatch, h.HandleONTAutoMatch)
	mux.HandleFunc(TaskExpireRewardClaims, h.HandleExpireRewardClaims)
}

func (h *Handlers) HandleGenerateInvoices(ctx context.Context, t *asynq.Task) error {
	var p GenerateInvoicesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	count, err := h.InvoiceService.GenerateScheduled(ctx, p.TenantID, time.Now())
	if err != nil {
		return fmt.Errorf("generate invoices for tenant %s: %w", p.TenantID, err)
	}

	log.Printf("[worker] generated %d scheduled invoices for tenant %s", count, p.TenantID)
	return nil
}

func (h *Handlers) HandleAutoIsolir(ctx context.Context, t *asynq.Task) error {
	var p AutoIsolirPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	tenant, err := h.TenantRepo.FindByID(ctx, p.TenantID)
	if err != nil {
		return fmt.Errorf("find tenant %s: %w", p.TenantID, err)
	}
	if tenant == nil {
		return fmt.Errorf("tenant %s not found", p.TenantID)
	}

	cutoff := time.Now().AddDate(0, 0, -tenant.GracePeriod)
	cutoffStr := cutoff.Format("2006-01-02")

	invoices, err := h.InvoiceRepo.ListOverdueForIsolir(ctx, p.TenantID, cutoffStr)
	if err != nil {
		return fmt.Errorf("list overdue invoices: %w", err)
	}

	isolated := 0
	var isolatedCustomers []isolatedCustomerInfo
	for _, inv := range invoices {
		if err := h.CustomerService.Isolate(ctx, p.TenantID, inv.CustomerID); err != nil {
			log.Printf("[worker] isolir failed for customer %s (tenant %s): %v", inv.CustomerID, p.TenantID, err)
			continue
		}
		isolated++
		if inv.Customer != nil {
			isolatedCustomers = append(isolatedCustomers, isolatedCustomerInfo{
				Name:          inv.Customer.Name,
				Phone:         inv.Customer.Phone,
				CustomerCode:  inv.Customer.CustomerCode,
				InvoiceNumber: inv.InvoiceNumber,
				TotalAmount:   inv.TotalAmount,
				DueDate:       inv.DueDate,
				PeriodMonth:   inv.PeriodMonth,
				PeriodYear:    inv.PeriodYear,
			})
		}
	}

	// Send isolir WA notification (async)
	if h.WAClient != nil && len(isolatedCustomers) > 0 {
		go h.sendIsolirNotifications(p.TenantID, isolatedCustomers)
	}

	log.Printf("[worker] auto-isolir tenant %s: %d/%d customers isolated", p.TenantID, isolated, len(invoices))
	return nil
}

type isolatedCustomerInfo struct {
	Name          string
	Phone         string
	CustomerCode  string
	InvoiceNumber string
	TotalAmount   int64
	DueDate       time.Time
	PeriodMonth   int
	PeriodYear    int
}

func (h *Handlers) sendIsolirNotifications(tenantID string, customers []isolatedCustomerInfo) {
	ctx := context.Background()

	var messageTemplate string
	if h.ReminderRepo != nil {
		reminder, err := h.ReminderRepo.FindActiveByType(ctx, tenantID, "isolir")
		if err != nil {
			log.Printf("[isolir-wa] failed to get template: %v", err)
		}
		if reminder != nil {
			messageTemplate = reminder.MessageTemplate
		}
	}

	if messageTemplate == "" {
		messageTemplate = "Halo {nama}, layanan internet Anda telah diisolir karena tagihan {nomor_invoice} sebesar Rp{jumlah} belum dibayar (jatuh tempo {jatuh_tempo}). Silakan segera lakukan pembayaran untuk mengaktifkan kembali layanan. Terima kasih."
	}

	var items []whatsapp.ReminderItem
	monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	salam := greetingByTimeWIB()

	for _, c := range customers {
		if c.Phone == "" {
			continue
		}

		periode := fmt.Sprintf("%s %d", monthNames[c.PeriodMonth], c.PeriodYear)

		msg := strings.ReplaceAll(messageTemplate, "{salam}", salam)
		msg = strings.ReplaceAll(msg, "{nama}", c.Name)
		msg = strings.ReplaceAll(msg, "{jumlah}", formatAmountDot(c.TotalAmount))
		msg = strings.ReplaceAll(msg, "{nomor_invoice}", c.InvoiceNumber)
		msg = strings.ReplaceAll(msg, "{jatuh_tempo}", c.DueDate.Format("02/01/2006"))
		msg = strings.ReplaceAll(msg, "{periode}", periode)
		msg = strings.ReplaceAll(msg, "{kode_pelanggan}", c.CustomerCode)

		items = append(items, whatsapp.ReminderItem{
			Phone:         c.Phone,
			CustomerName:  c.Name,
			InvoiceNumber: c.InvoiceNumber,
			Amount:        c.TotalAmount,
			DueDate:       c.DueDate.Format("02/01/2006"),
			Message:       msg,
		})
	}

	if len(items) == 0 {
		return
	}

	result, err := h.WAClient.SendReminders(ctx, tenantID, items)
	if err != nil {
		log.Printf("[isolir-wa] send failed: %v", err)
		return
	}
	log.Printf("[isolir-wa] sent=%d, failed=%d for tenant %s", result.Success, result.Failed, tenantID)
}

func (h *Handlers) HandleTriggerReminders(ctx context.Context, t *asynq.Task) error {
	var p TriggerRemindersPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	result, err := h.ReminderService.TriggerReminders(ctx, p.TenantID)
	if err != nil {
		return fmt.Errorf("trigger reminders for tenant %s: %w", p.TenantID, err)
	}

	log.Printf("[worker] reminders tenant %s: %s", p.TenantID, result.Message)
	return nil
}

func (h *Handlers) HandleExpirePayments(ctx context.Context, t *asynq.Task) error {
	expired, err := h.InvoiceService.ExpireStalePayments(ctx)
	if err != nil {
		return fmt.Errorf("expire stale payments: %w", err)
	}
	log.Printf("[worker] expired %d stale gateway payments", expired)
	return nil
}

func (h *Handlers) HandleRouterMonitor(ctx context.Context, t *asynq.Task) error {
	var p RouterMonitorPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	if h.RouterRepo == nil {
		return nil
	}
	// Routers not sending a heartbeat in the last 10 minutes are marked offline.
	threshold := time.Now().Add(-10 * time.Minute)
	offlineIDs, err := h.RouterRepo.MarkStaleOffline(ctx, p.TenantID, threshold)
	if err != nil {
		return fmt.Errorf("mark stale routers offline (tenant %s): %w", p.TenantID, err)
	}
	if len(offlineIDs) > 0 {
		log.Printf("[worker] router-monitor tenant %s: %d router(s) marked offline", p.TenantID, len(offlineIDs))
		if h.RouterService != nil {
			h.RouterService.LogDisconnections(ctx, offlineIDs)
		}
	}
	return nil
}

func (h *Handlers) HandleCleanSessions(ctx context.Context, t *asynq.Task) error {
	var p CleanSessionsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	if h.SessionRepo == nil {
		return nil
	}
	cleaned, customerIDs, err := h.SessionRepo.CleanAllStaleSessionsWithCustomers(ctx, p.TenantID)
	if err != nil {
		return fmt.Errorf("clean stale sessions (tenant %s): %w", p.TenantID, err)
	}
	if cleaned > 0 {
		log.Printf("[worker] clean-sessions tenant %s: %d stale session(s) terminated", p.TenantID, cleaned)
		// Release IPAM IPs for customers whose sessions were cleaned up
		if h.IPAMRepo != nil {
			for _, cid := range customerIDs {
				if err := h.IPAMRepo.ReleaseAllByCustomerID(ctx, cid); err != nil {
					log.Printf("[worker] clean-sessions: failed to release IPAM IP for customer %s: %v", cid, err)
				}
			}
		}
	}
	return nil
}

// greetingByTimeWIB returns a time-appropriate Indonesian greeting based on WIB (UTC+7).
func greetingByTimeWIB() string {
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

// formatAmountDot formats amount with dot thousand separator (e.g. 200000 -> "200.000").
func formatAmountDot(amount int64) string {
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

// HandleONTProvisionRetry mencoba provisioning ulang semua ONT aktif yang belum terprovisi.
// Dipanggil secara periodik oleh scheduler — otomatis mendeteksi perangkat baru yang online.
func (h *Handlers) HandleONTProvisionRetry(ctx context.Context, t *asynq.Task) error {
	if h.GenieACSService == nil || h.ONTRepo == nil {
		return nil
	}

	onts, err := h.ONTRepo.FindUnprovisioned(ctx)
	if err != nil {
		return fmt.Errorf("ont provision retry: find unprovisioned: %w", err)
	}

	if len(onts) == 0 {
		return nil
	}

	success, skipped := 0, 0
	for _, ont := range onts {
		if err := h.GenieACSService.ProvisionONT(ctx, ont.TenantID, ont.ID); err != nil {
			// Device masih offline atau belum kontak GenieACS — coba lagi di interval berikutnya
			skipped++
		} else {
			success++
		}
	}

	log.Printf("[worker] ont-provision-retry: %d berhasil, %d masih offline (dari %d total)", success, skipped, len(onts))
	return nil
}

// HandleONTDiscover memindai GenieACS untuk menemukan perangkat baru yang belum terdaftar di DB.
// Dipanggil secara periodik — perangkat baru otomatis masuk ke tabel ONT dengan status "discovered".
// Dijalankan SEKALI secara global (bukan per-tenant) karena GenieACS adalah service bersama.
// Tenant pertama yang aktif digunakan sebagai konteks awal; serial number dicek lintas semua tenant
// sehingga tidak ada duplikat meskipun ada banyak tenant.
func (h *Handlers) HandleONTDiscover(ctx context.Context, t *asynq.Task) error {
	if h.GenieACSService == nil || h.ONTRepo == nil {
		return nil
	}

	// Ambil satu tenant aktif sebagai konteks default untuk ONT yang belum ter-match
	tenants, err := h.TenantRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("ont-discover: list tenants: %w", err)
	}
	if len(tenants) == 0 {
		return nil
	}

	// Gunakan tenant pertama sebagai "holding" tenant; AutoMatch akan meluruskan tenant
	// ke tenant yang benar setelah PPPoE username cocok dengan pelanggan.
	defaultTenantID := tenants[0].ID
	result, err := h.GenieACSService.DiscoverONTs(ctx, defaultTenantID)
	if err != nil {
		return fmt.Errorf("ont-discover: %w", err)
	}
	if result.NewDiscovered > 0 {
		log.Printf("[worker] ont-discover: %d perangkat baru ditemukan dari %d total di GenieACS",
			result.NewDiscovered, result.TotalInGenieACS)
	}

	return nil
}

// HandleONTAutoMatch mencocokkan ONT yang belum ter-assign ke pelanggan
// menggunakan PPPoE username dari parameter TR-069 di GenieACS.
// Dijalankan SEKALI secara global — ONT dicari lintas semua tenant, pelanggan juga dicari global.
func (h *Handlers) HandleONTAutoMatch(ctx context.Context, t *asynq.Task) error {
	if h.GenieACSService == nil || h.ONTRepo == nil {
		return nil
	}

	// Global call — parameter tenantID diabaikan di dalam AutoMatchONTs
	result, err := h.GenieACSService.AutoMatchONTs(ctx, "")
	if err != nil {
		return fmt.Errorf("ont-auto-match: %w", err)
	}
	if result.Matched > 0 {
		log.Printf("[worker] ont-auto-match: %d perangkat berhasil dihubungkan ke pelanggan (dari %d unlinked)",
			result.Matched, result.TotalUnlinked)
	}

	return nil
}

func (h *Handlers) HandleExpireRewardClaims(ctx context.Context, t *asynq.Task) error {
	if h.RewardService == nil {
		return nil
	}
	expired, err := h.RewardService.ExpireOldClaims(ctx)
	if err != nil {
		return fmt.Errorf("expire reward claims: %w", err)
	}
	if expired > 0 {
		log.Printf("[worker] expired %d stale reward claims", expired)
	}
	return nil
}
