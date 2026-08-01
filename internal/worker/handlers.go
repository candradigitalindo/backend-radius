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
	"github.com/redis/go-redis/v9"

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
	SettingRepo     repository.SettingRepository
	RouterService   *service.RouterService
	SNMPService     *service.SNMPService
	RewardService   *service.RewardService
	WAClient        *whatsapp.Client
	GenieACSService *service.GenieACSService
	// GenieACSEnabled is true when the env-driven GenieACS config (GENIEACS_URL)
	// is set. The per-tenant "genieacs_url" setting is only a legacy fallback.
	GenieACSEnabled     bool
	ONTRepo             repository.ONTRepository
	SubscriptionRepo    repository.SubscriptionRepository
	NotificationService *service.NotificationService
	RDB                 *redis.Client
}

// genieACSReady reports whether the ONT workers should run: GenieACS is either
// configured globally via env, or at least one tenant has the legacy setting.
func (h *Handlers) genieACSReady(ctx context.Context) bool {
	if h.GenieACSEnabled {
		return true
	}
	ok, _ := h.SettingRepo.ExistsWithValue(ctx, "genieacs_url")
	return ok
}

// saTenantID returns the superadmin's tenant ID by looking up slug="superadmin".
func (h *Handlers) saTenantID(ctx context.Context) string {
	if h.TenantRepo == nil {
		return ""
	}
	t, err := h.TenantRepo.FindBySlug(ctx, "superadmin")
	if err != nil || t == nil {
		return ""
	}
	return t.ID
}

// subReminderTemplate fetches a subscription reminder template from the superadmin tenant.
// Falls back to the provided default if not found.
func (h *Handlers) subReminderTemplate(ctx context.Context, reminderType, fallback string) string {
	if h.ReminderRepo == nil {
		return fallback
	}
	tid := h.saTenantID(ctx)
	if tid == "" {
		return fallback
	}
	rem, err := h.ReminderRepo.FindActiveByType(ctx, tid, reminderType)
	if err != nil || rem == nil {
		return fallback
	}
	return rem.MessageTemplate
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGenerateInvoices, h.HandleGenerateInvoices)
	mux.HandleFunc(TaskAutoIsolir, h.HandleAutoIsolir)
	mux.HandleFunc(TaskGracePeriodCheck, h.HandleGracePeriodCheck)
	mux.HandleFunc(TaskTriggerReminders, h.HandleTriggerReminders)
	mux.HandleFunc(TaskExpirePayments, h.HandleExpirePayments)
	mux.HandleFunc(TaskRouterMonitor, h.HandleRouterMonitor)
	mux.HandleFunc(TaskCleanSessions, h.HandleCleanSessions)
	mux.HandleFunc(TaskONTProvisionRetry, h.HandleONTProvisionRetry)
	mux.HandleFunc(TaskONTDiscover, h.HandleONTDiscover)
	mux.HandleFunc(TaskONTAutoMatch, h.HandleONTAutoMatch)
	mux.HandleFunc(TaskExpireRewardClaims, h.HandleExpireRewardClaims)
	mux.HandleFunc(TaskSubExpiryCheck, h.HandleSubscriptionExpiryCheck)
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

	if count > 0 {
		log.Printf("[worker] generated %d scheduled invoices for tenant %s", count, p.TenantID)
	}
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

	// defaultIsolirDay = tenant.IsolirDay (days after due date before isolation),
	// used as fallback for customers without a billing profile. Customers with a
	// billing profile use billing_profiles.isolir_day from the DB query directly.
	defaultIsolirDay := tenant.IsolirDay

	invoices, err := h.InvoiceRepo.ListOverdueForIsolir(ctx, p.TenantID, defaultIsolirDay)
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
				CustomerID:    inv.CustomerID,
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

	// Web-push + in-app isolir notification (async, independent of WhatsApp)
	if h.NotificationService != nil && len(isolatedCustomers) > 0 {
		go h.pushIsolirNotifications(p.TenantID, isolatedCustomers)
	}

	// Send isolir WA notification (async)
	if h.WAClient != nil && len(isolatedCustomers) > 0 {
		go h.sendIsolirNotifications(p.TenantID, isolatedCustomers)
	}

	if isolated > 0 {
		log.Printf("[worker] auto-isolir tenant %s: %d/%d customers isolated", p.TenantID, isolated, len(invoices))
	}
	return nil
}

// HandleGracePeriodCheck sends a final WA warning to customers whose grace period
// has elapsed — they have been isolated and still haven't paid.
func (h *Handlers) HandleGracePeriodCheck(ctx context.Context, t *asynq.Task) error {
	var p GracePeriodCheckPayload
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

	// defaultGracePeriod from tenant (fallback for customers without billing profile)
	customerIDs, err := h.InvoiceRepo.ListIsolatedGraceExpired(ctx, p.TenantID, tenant.GracePeriod)
	if err != nil {
		return fmt.Errorf("list grace-expired customers: %w", err)
	}

	if len(customerIDs) == 0 {
		return nil
	}

	if h.WAClient == nil {
		log.Printf("[worker] grace-period tenant %s: %d customers, WA client unavailable", p.TenantID, len(customerIDs))
		return nil
	}

	// Load WA reminder template if configured
	defaultTpl := "⛔ {salam} *{nama}*,\n\nLayanan internet Anda (kode: *{kode_pelanggan}*) telah melewati masa toleransi.\n\nSilakan segera lunasi tagihan untuk mengaktifkan kembali layanan Anda.\n\nTerima kasih."
	tpl := defaultTpl
	if h.ReminderRepo != nil {
		if r, err := h.ReminderRepo.FindActiveByType(ctx, p.TenantID, "grace_expired"); err == nil && r != nil {
			tpl = r.MessageTemplate
		}
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	salam := greetingByTimeWorker(loc)
	waSession := service.WASessionForTenant(ctx, p.TenantID, h.SettingRepo)

	sent := 0
	for _, custID := range customerIDs {
		cust, err := h.CustomerService.GetByID(ctx, p.TenantID, custID)
		if err != nil || cust == nil || cust.Phone == "" {
			continue
		}
		msg := strings.ReplaceAll(tpl, "{salam}", salam)
		msg = strings.ReplaceAll(msg, "{nama}", cust.Name)
		msg = strings.ReplaceAll(msg, "{kode_pelanggan}", cust.CustomerCode)

		waCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = h.WAClient.SendMessage(waCtx, waSession, cust.Phone, msg)
		cancel()
		sent++
	}

	if sent > 0 {
		log.Printf("[worker] grace-period tenant %s: sent %d notifications", p.TenantID, sent)
	}
	return nil
}

func greetingByTimeWorker(loc *time.Location) string {
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

type isolatedCustomerInfo struct {
	CustomerID    string
	Name          string
	Phone         string
	CustomerCode  string
	InvoiceNumber string
	TotalAmount   int64
	DueDate       time.Time
	PeriodMonth   int
	PeriodYear    int
}

// pushIsolirNotifications sends web-push + in-app notifications for isolated
// customers. Independent of WhatsApp so it fires even when WA isn't configured.
func (h *Handlers) pushIsolirNotifications(tenantID string, customers []isolatedCustomerInfo) {
	ctx := context.Background()
	for _, c := range customers {
		if c.CustomerID == "" {
			continue
		}
		body := fmt.Sprintf("Layanan internet Anda diisolir karena tagihan %s sebesar Rp%s belum dibayar. Silakan lakukan pembayaran untuk mengaktifkan kembali.",
			c.InvoiceNumber, formatAmountDot(c.TotalAmount))
		data := fmt.Sprintf(`{"type":"isolir","invoice":"%s"}`, c.InvoiceNumber)
		_ = h.NotificationService.PushAndStore(ctx, tenantID, c.CustomerID, "Layanan Diisolir", body, data)
	}
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

	waSession := service.WASessionForTenant(ctx, tenantID, h.SettingRepo)
	result, err := h.WAClient.SendReminders(ctx, waSession, items)
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

	if result.TotalSent > 0 {
		log.Printf("[worker] reminders tenant %s: %s", p.TenantID, result.Message)
	}
	return nil
}

func (h *Handlers) HandleExpirePayments(ctx context.Context, t *asynq.Task) error {
	expired, err := h.InvoiceService.ExpireStalePayments(ctx)
	if err != nil {
		return fmt.Errorf("expire stale payments: %w", err)
	}
	if expired > 0 {
		log.Printf("[worker] expired %d stale gateway payments", expired)
	}
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
	// Routers not sending a heartbeat in the last 10 minutes are candidates for SNMP poll.
	threshold := time.Now().Add(-10 * time.Minute)

	// SNMP fallback: poll stale routers that have a VPN IP before marking them offline.
	// Each poll uses a 2s timeout with 0 retries — minimal load on router and server.
	if h.SNMPService != nil {
		stale, err := h.RouterRepo.ListStaleWithVPN(ctx, p.TenantID, threshold)
		if err == nil {
			for _, rt := range stale {
				data, err := h.SNMPService.PollRouterHeartbeat(ctx, rt.VPNIP, rt.SNMPCommunity)
				if err != nil {
					continue // unreachable via SNMP — will be marked offline below
				}
				wasOffline, _ := h.RouterRepo.UpdateHeartbeat(ctx, rt.ID, repository.HeartbeatInfo{
					Identity: data.Identity,
					Uptime:   data.Uptime,
					CPULoad:  data.CPULoad,
				})
				if wasOffline {
					log.Printf("[worker] router-snmp tenant=%s router=%s back online via SNMP", p.TenantID, rt.VPNIP)
				}
			}
		}
	}

	// Mark routers that are still stale (heartbeat AND SNMP both failed) as offline.
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
	if h.GenieACSService == nil || h.ONTRepo == nil || h.SettingRepo == nil {
		return nil
	}
	if !h.genieACSReady(ctx) {
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
	if h.GenieACSService == nil || h.ONTRepo == nil || h.SettingRepo == nil {
		return nil
	}

	// Skip jika GenieACS tidak dikonfigurasi — hindari error berulang.
	if !h.genieACSReady(ctx) {
		return nil
	}

	tenants, err := h.TenantRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("ont-discover: list tenants: %w", err)
	}
	if len(tenants) == 0 {
		return nil
	}

	defaultTenantID := tenants[0].ID
	result, err := h.GenieACSService.DiscoverONTs(ctx, defaultTenantID)
	if err != nil {
		return fmt.Errorf("ont-discover: %w", err)
	}
	if result.NewDiscovered > 0 {
		log.Printf("[worker] ont-discover: %d perangkat baru ditemukan dari %d total di GenieACS",
			result.NewDiscovered, result.TotalInGenieACS)

		// Fast path: langsung auto-match + provision tanpa menunggu worker berikutnya.
		// Ini memotong waktu dari ~12 menit menjadi ~1 menit.
		matchResult, merr := h.GenieACSService.AutoMatchONTs(ctx, "")
		if merr == nil && matchResult.Matched > 0 {
			log.Printf("[worker] ont-discover fast-match: %d device langsung terhubung ke pelanggan", matchResult.Matched)
		}
		// Provision semua yang belum terprovisi (termasuk yang baru saja ter-match)
		onts, perr := h.ONTRepo.FindUnprovisioned(ctx)
		if perr == nil {
			provOK := 0
			for _, ont := range onts {
				if err := h.GenieACSService.ProvisionONT(ctx, ont.TenantID, ont.ID); err == nil {
					provOK++
				}
			}
			if provOK > 0 {
				log.Printf("[worker] ont-discover fast-provision: %d device berhasil diprovisioning", provOK)
			}
		}
	}

	return nil
}

// HandleONTAutoMatch mencocokkan ONT yang belum ter-assign ke pelanggan
// menggunakan PPPoE username dari parameter TR-069 di GenieACS.
// Dijalankan SEKALI secara global — ONT dicari lintas semua tenant, pelanggan juga dicari global.
func (h *Handlers) HandleONTAutoMatch(ctx context.Context, t *asynq.Task) error {
	if h.GenieACSService == nil || h.ONTRepo == nil || h.SettingRepo == nil {
		return nil
	}
	if !h.genieACSReady(ctx) {
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

func (h *Handlers) HandleSubscriptionExpiryCheck(ctx context.Context, t *asynq.Task) error {
	tenants, err := h.TenantRepo.ListExpiringSoon(ctx, 8)
	if err != nil {
		return fmt.Errorf("sub expiry check: list expiring: %w", err)
	}
	if len(tenants) == 0 {
		MarkDailyRun(ctx, h.RDB, TaskSubExpiryCheck)
		return nil
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc).Truncate(24 * time.Hour)
	salam := greetingByTimeWIB()

	// Load templates once
	tplH7 := h.subReminderTemplate(ctx, "sub_expiry_h7",
		"Selamat {salam} 🙏\n\nKepada Tim *{nama_isp}*,\n\nLangganan paket *{nama_paket}* akan berakhir dalam *7 hari* pada *{tanggal_berakhir}*.\n\nSegera perpanjang agar layanan tidak terganggu.\n\nTerima kasih.")
	tplH1 := h.subReminderTemplate(ctx, "sub_expiry_h1",
		"⚠️ Selamat {salam},\n\nLangganan paket *{nama_paket}* ISP *{nama_isp}* berakhir *BESOK, {tanggal_berakhir}*.\n\nSegera lakukan perpanjangan. Terima kasih.")
	tplH0 := h.subReminderTemplate(ctx, "sub_expiry_h0",
		"🔴 Selamat {salam},\n\nLangganan paket *{nama_paket}* ISP *{nama_isp}* telah *BERAKHIR* pada *{tanggal_berakhir}*.\n\nAkses panel dibatasi. Silakan perpanjang sekarang.")

	sent := 0
	for _, tenant := range tenants {
		if tenant.PlanExpiresAt == nil || h.WAClient == nil || tenant.Phone == "" {
			continue
		}

		expiresLocal := tenant.PlanExpiresAt.In(loc).Truncate(24 * time.Hour)
		daysLeft := int(expiresLocal.Sub(today).Hours() / 24)
		expiryStr := tenant.PlanExpiresAt.In(loc).Format("02 January 2006")

		var tpl string
		switch daysLeft {
		case 7:
			tpl = tplH7
		case 1:
			tpl = tplH1
		case 0:
			tpl = tplH0
		default:
			continue
		}

		msg := strings.ReplaceAll(tpl, "{salam}", salam)
		msg = strings.ReplaceAll(msg, "{nama_isp}", tenant.Name)
		msg = strings.ReplaceAll(msg, "{nama_paket}", tenant.Plan)
		msg = strings.ReplaceAll(msg, "{tanggal_berakhir}", expiryStr)

		waCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = h.WAClient.SendMessage(waCtx, "superadmin", tenant.Phone, msg)
		cancel()
		sent++
	}

	if sent > 0 {
		log.Printf("[worker] sub-expiry-check: sent %d notifications", sent)
	}
	MarkDailyRun(ctx, h.RDB, TaskSubExpiryCheck)
	return nil
}
