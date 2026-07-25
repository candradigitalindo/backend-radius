package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrPlanNotFound  = errors.New("Paket tidak ditemukan")
	ErrOrderNotFound = errors.New("Order tidak ditemukan")
	ErrPlanInactive  = errors.New("Paket tidak aktif")
	ErrPendingOrder  = errors.New("Masih ada order yang belum dibayar")
)

type SubscriptionService struct {
	subRepo      repository.SubscriptionRepository
	tenantRepo   repository.TenantRepository
	reminderRepo repository.ReminderRepository
	settingRepo  repository.SettingRepository
	waClient     *whatsapp.Client
	pgCfg        *config.PGConfig // fallback from .env
	baseURL      string
}

func NewSubscriptionService(subRepo repository.SubscriptionRepository, tenantRepo repository.TenantRepository) *SubscriptionService {
	return &SubscriptionService{subRepo: subRepo, tenantRepo: tenantRepo}
}

func (s *SubscriptionService) WithPG(pgCfg *config.PGConfig, baseURL string) *SubscriptionService {
	s.pgCfg = pgCfg
	s.baseURL = baseURL
	return s
}

func (s *SubscriptionService) WithWAClient(waClient *whatsapp.Client) *SubscriptionService {
	s.waClient = waClient
	return s
}

func (s *SubscriptionService) WithReminderRepo(repo repository.ReminderRepository) *SubscriptionService {
	s.reminderRepo = repo
	return s
}

func (s *SubscriptionService) WithSettingRepo(repo repository.SettingRepository) *SubscriptionService {
	s.settingRepo = repo
	return s
}

// loadSAPGConfig loads the superadmin PG config from the DB settings table.
// Falls back to the static pgCfg from .env if settingRepo is not set or key is missing.
func (s *SubscriptionService) loadSAPGConfig(ctx context.Context) *config.PGConfig {
	if s.settingRepo == nil {
		return s.pgCfg
	}
	settings, err := s.settingRepo.List(ctx, "tnt-superadmin")
	if err != nil || len(settings) == 0 {
		return s.pgCfg
	}
	m := make(map[string]string, len(settings))
	for _, st := range settings {
		m[st.Key] = st.Value
	}
	provider := m["sa_pg_provider"]
	apiKey := m["sa_pg_api_key"]
	secretKey := m["sa_pg_secret_key"]
	merchantID := m["sa_pg_merchant_id"]
	sandbox := m["sa_pg_sandbox"] == "true"

	if provider == "" && s.pgCfg != nil {
		return s.pgCfg
	}
	return &config.PGConfig{
		Provider:   provider,
		APIKey:     apiKey,
		SecretKey:  secretKey,
		MerchantID: merchantID,
		Sandbox:    sandbox,
	}
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	plans, err := s.subRepo.ListPlans(ctx, true)
	if err != nil {
		return nil, err
	}
	// "trial" adalah paket sistem — tidak boleh muncul di daftar pilihan tenant
	result := plans[:0]
	for _, p := range plans {
		if p.Slug != "trial" {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *SubscriptionService) ListAllPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	return s.subRepo.ListPlans(ctx, false)
}

func (s *SubscriptionService) GetPlan(ctx context.Context, planID string) (*model.SubscriptionPlan, error) {
	plan, err := s.subRepo.FindPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

func (s *SubscriptionService) CreatePlan(ctx context.Context, plan *model.SubscriptionPlan) error {
	return s.subRepo.CreatePlan(ctx, plan)
}

func (s *SubscriptionService) UpdatePlan(ctx context.Context, plan *model.SubscriptionPlan) error {
	return s.subRepo.UpdatePlan(ctx, plan)
}

func (s *SubscriptionService) DeletePlan(ctx context.Context, planID string) error {
	return s.subRepo.DeletePlan(ctx, planID)
}

type SubscribeInput struct {
	TenantID       string
	PlanID         string
	DurationMonths int
}

// Subscribe creates an order for the given plan. For free plans, it activates immediately.
func (s *SubscriptionService) Subscribe(ctx context.Context, input SubscribeInput) (*model.SubscriptionOrder, error) {
	plan, err := s.subRepo.FindPlanByID(ctx, input.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}
	if !plan.IsActive {
		return nil, ErrPlanInactive
	}

	// Check for existing pending orders
	orders, _, err := s.subRepo.ListOrders(ctx, input.TenantID, repository.OrderFilter{Status: "pending", Page: 1, PerPage: 1})
	if err != nil {
		return nil, err
	}
	if len(orders) > 0 {
		return nil, ErrPendingOrder
	}

	now := time.Now()

	// Determine duration: 1 (monthly) or 12 (yearly), default to plan's duration
	duration := input.DurationMonths
	if duration != 1 && duration != 12 {
		duration = plan.DurationMonths
	}

	// Calculate amount: yearly gets 20% discount
	amount := plan.Price * int64(duration)
	if duration == 12 {
		amount = amount * 80 / 100
	}

	order := &model.SubscriptionOrder{
		TenantID:       input.TenantID,
		PlanID:         plan.ID,
		PlanName:       plan.Name,
		Amount:         amount,
		DurationMonths: duration,
		Status:         "pending",
	}

	// Free plan → activate immediately
	if plan.Price == 0 {
		order.Status = "paid"
		order.PaidAt = &now
		order.StartsAt = &now
		order.Notes = "Paket gratis, aktif otomatis"

		if err := s.subRepo.CreateOrder(ctx, order); err != nil {
			return nil, err
		}

		// Activate the tenant plan
		if err := s.activatePlan(ctx, input.TenantID, plan, &now, nil); err != nil {
			return nil, err
		}

		return order, nil
	}

	// Paid plan → create pending order
	if err := s.subRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// ConfirmPayment marks an order as paid and activates the plan on the tenant.
func (s *SubscriptionService) ConfirmPayment(ctx context.Context, orderID string, paymentMethod, paymentRef string) (*model.SubscriptionOrder, error) {
	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.Status != "pending" {
		return nil, errors.New("Order sudah diproses")
	}

	plan, err := s.subRepo.FindPlanByID(ctx, order.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	now := time.Now()
	order.Status = "paid"
	order.PaidAt = &now
	order.PaymentMethod = paymentMethod
	order.PaymentRef = paymentRef

	// Calculate starts_at: from now or from current expiry (if extending)
	tenant, err := s.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil {
		return nil, err
	}

	startsAt := now
	if tenant != nil && tenant.PlanExpiresAt != nil && tenant.PlanExpiresAt.After(now) {
		startsAt = *tenant.PlanExpiresAt
	}
	expiresAt := startsAt.AddDate(0, order.DurationMonths, 0)
	order.StartsAt = &startsAt
	order.ExpiresAt = &expiresAt

	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if err := s.activatePlan(ctx, order.TenantID, plan, &startsAt, &expiresAt); err != nil {
		return nil, err
	}

	// Send WA payment confirmation (async)
	go s.sendPaymentConfirmationWA(order, plan, &expiresAt)

	return order, nil
}

func (s *SubscriptionService) ListOrders(ctx context.Context, tenantID string, filter repository.OrderFilter) ([]model.SubscriptionOrder, int, error) {
	return s.subRepo.ListOrders(ctx, tenantID, filter)
}

func (s *SubscriptionService) GetOrder(ctx context.Context, orderID, tenantID string) (*model.SubscriptionOrder, error) {
	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.TenantID != tenantID {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// ── Superadmin order management ──────────────────────────────────────────────

func (s *SubscriptionService) AdminListAllOrders(ctx context.Context, filter repository.AdminOrderFilter) ([]repository.AdminOrderRow, int, error) {
	return s.subRepo.ListAllOrders(ctx, filter)
}

func (s *SubscriptionService) AdminGetOrder(ctx context.Context, orderID string) (*model.SubscriptionOrder, error) {
	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

type AdminUpdateOrderInput struct {
	Status        string
	PaymentMethod string
	PaymentRef    string
	Notes         string
	PaidAt        *time.Time
	StartsAt      *time.Time
	ExpiresAt     *time.Time
}

func (s *SubscriptionService) AdminUpdateOrder(ctx context.Context, orderID string, input AdminUpdateOrderInput) (*model.SubscriptionOrder, error) {
	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	prevStatus := order.Status

	if input.Status != "" {
		order.Status = input.Status
	}
	if input.PaymentMethod != "" {
		order.PaymentMethod = input.PaymentMethod
	}
	if input.PaymentRef != "" {
		order.PaymentRef = input.PaymentRef
	}
	order.Notes = input.Notes
	if input.PaidAt != nil {
		order.PaidAt = input.PaidAt
	}
	if input.StartsAt != nil {
		order.StartsAt = input.StartsAt
	}
	if input.ExpiresAt != nil {
		order.ExpiresAt = input.ExpiresAt
	}

	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	// When status changes to paid, activate the plan on the tenant
	if order.Status == "paid" && prevStatus != "paid" {
		plan, err := s.subRepo.FindPlanByID(ctx, order.PlanID)
		if err == nil && plan != nil {
			now := time.Now()
			var startsAt, expiresAt time.Time
			if order.StartsAt != nil && order.ExpiresAt != nil {
				// Explicit dates provided — respect them.
				startsAt = *order.StartsAt
				expiresAt = *order.ExpiresAt
			} else {
				// Extend from the current expiry so marking an order paid never
				// shortens an already-active subscription (mirrors ConfirmPayment).
				startsAt = now
				if tenant, terr := s.tenantRepo.FindByID(ctx, order.TenantID); terr == nil &&
					tenant != nil && tenant.PlanExpiresAt != nil && tenant.PlanExpiresAt.After(now) {
					startsAt = *tenant.PlanExpiresAt
				}
				expiresAt = startsAt.AddDate(0, order.DurationMonths, 0)
			}
			_ = s.activatePlan(ctx, order.TenantID, plan, &startsAt, &expiresAt)
		}
	}

	return order, nil
}

func (s *SubscriptionService) AdminCreateOrder(ctx context.Context, tenantID, planID string, durationMonths int) (*model.SubscriptionOrder, error) {
	plan, err := s.subRepo.FindPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	now := time.Now()
	// Yearly (12-month) orders get the same 20% discount as the self-service path.
	amount := plan.Price * int64(durationMonths)
	if durationMonths == 12 {
		amount = amount * 80 / 100
	}
	order := &model.SubscriptionOrder{
		ID:             id.New(),
		TenantID:       tenantID,
		PlanID:         planID,
		PlanName:       plan.Name,
		Amount:         amount,
		DurationMonths: durationMonths,
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.subRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SubscriptionService) AdminDeleteOrder(ctx context.Context, orderID string) error {
	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}
	return s.subRepo.DeleteOrder(ctx, orderID)
}

func (s *SubscriptionService) activatePlan(ctx context.Context, tenantID string, plan *model.SubscriptionPlan, startsAt *time.Time, expiresAt *time.Time) error {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	tenant.Plan = plan.Slug
	tenant.MaxCustomers = plan.MaxCustomers
	tenant.PlanExpiresAt = expiresAt

	return s.tenantRepo.UpdatePlan(ctx, tenant)
}

// PaymentGatewayResult holds the gateway response for a subscription payment.
type SubPaymentResult struct {
	OrderID    string     `json:"order_id"`
	PaymentURL string     `json:"payment_url"`
	SnapToken  string     `json:"snap_token,omitempty"`
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
}

// CreatePayment generates a payment gateway transaction for a pending subscription order.
func (s *SubscriptionService) CreatePayment(ctx context.Context, orderID, tenantID, returnURL string) (*SubPaymentResult, error) {
	pgCfg := s.loadSAPGConfig(ctx)
	if pgCfg == nil || (pgCfg.APIKey == "" && pgCfg.SecretKey == "") {
		return nil, errors.New("Payment gateway belum dikonfigurasi")
	}

	order, err := s.subRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.TenantID != tenantID {
		return nil, ErrOrderNotFound
	}
	if order.Status != "pending" {
		return nil, errors.New("Order sudah diproses")
	}

	// Already has a payment URL → return it
	if order.PaymentURL != "" {
		return &SubPaymentResult{
			OrderID:    order.ID,
			PaymentURL: order.PaymentURL,
			SnapToken:  order.SnapToken,
		}, nil
	}

	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	merchantRef := "SUB-" + order.ID

	switch pgCfg.Provider {
	case "midtrans":
		return s.createSubMidtransPayment(ctx, order, tenant, pgCfg, merchantRef, returnURL)
	case "xendit":
		return s.createSubXenditPayment(ctx, order, tenant, pgCfg, merchantRef, returnURL)
	default:
		callbackURL := s.baseURL + "/api/v1/webhooks/subscription/tripay"
		return s.createSubTripayPayment(ctx, order, tenant, pgCfg, merchantRef, returnURL, callbackURL)
	}
}

func (s *SubscriptionService) createSubTripayPayment(ctx context.Context, order *model.SubscriptionOrder, tenant *model.Tenant, pgCfg *config.PGConfig, merchantRef, returnURL, callbackURL string) (*SubPaymentResult, error) {
	client := payment.NewTripayClient(pgCfg.APIKey, pgCfg.SecretKey, pgCfg.MerchantID, pgCfg.Sandbox)

	resp, err := client.CreateTransaction(ctx, payment.CreateTransactionRequest{
		Method:        "BRIVA",
		MerchantRef:   merchantRef,
		Amount:        order.Amount,
		CustomerName:  tenant.Name,
		CustomerEmail: tenant.Email,
		CustomerPhone: tenant.Phone,
		ReturnURL:     returnURL,
		CallbackURL:   callbackURL,
	})
	if err != nil {
		return nil, fmt.Errorf("tripay create transaction: %w", err)
	}

	expiredAt := time.Unix(resp.Data.ExpiredTime, 0)

	// Store payment URL and ref on the order
	order.PaymentURL = resp.Data.PaymentURL
	order.PaymentRef = resp.Data.Reference
	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	return &SubPaymentResult{
		OrderID:    order.ID,
		PaymentURL: resp.Data.PaymentURL,
		ExpiredAt:  &expiredAt,
	}, nil
}

func (s *SubscriptionService) createSubMidtransPayment(ctx context.Context, order *model.SubscriptionOrder, tenant *model.Tenant, pgCfg *config.PGConfig, merchantRef, returnURL string) (*SubPaymentResult, error) {
	client := payment.NewMidtransClient(pgCfg.SecretKey, pgCfg.Sandbox)

	notificationURL := s.baseURL + "/api/v1/webhooks/subscription/midtrans"

	resp, err := client.CreateTransaction(ctx, payment.MidtransCreateRequest{
		OrderID:         merchantRef,
		GrossAmount:     order.Amount,
		FirstName:       tenant.Name,
		Email:           tenant.Email,
		Phone:           tenant.Phone,
		FinishURL:       returnURL,
		ErrorURL:        returnURL,
		ItemName:        fmt.Sprintf("Langganan %s", order.PlanName),
		NotificationURL: notificationURL,
	})
	if err != nil {
		return nil, fmt.Errorf("midtrans create transaction: %w", err)
	}

	order.PaymentURL = resp.RedirectURL
	order.PaymentRef = merchantRef
	order.SnapToken = resp.Token
	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	return &SubPaymentResult{
		OrderID:    order.ID,
		PaymentURL: resp.RedirectURL,
		SnapToken:  resp.Token,
	}, nil
}

func (s *SubscriptionService) createSubXenditPayment(ctx context.Context, order *model.SubscriptionOrder, tenant *model.Tenant, pgCfg *config.PGConfig, merchantRef, returnURL string) (*SubPaymentResult, error) {
	client := payment.NewXenditClient(pgCfg.APIKey, pgCfg.SecretKey, pgCfg.Sandbox)

	callbackURL := s.baseURL + "/api/v1/webhooks/subscription/xendit"

	resp, err := client.CreateInvoice(ctx, payment.XenditCreateInvoiceRequest{
		ExternalID:      merchantRef,
		Amount:          order.Amount,
		Description:     fmt.Sprintf("Langganan %s", order.PlanName),
		PayerEmail:      tenant.Email,
		CustomerName:    tenant.Name,
		CustomerPhone:   tenant.Phone,
		SuccessRedirect: returnURL,
		FailureRedirect: returnURL,
		CallbackURL:     callbackURL,
	})
	if err != nil {
		return nil, fmt.Errorf("xendit create invoice: %w", err)
	}

	order.PaymentURL = resp.InvoiceURL
	order.PaymentRef = merchantRef
	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	return &SubPaymentResult{
		OrderID:    order.ID,
		PaymentURL: resp.InvoiceURL,
	}, nil
}

// ProcessSubTripayWebhook handles Tripay callback for subscription orders.
func (s *SubscriptionService) ProcessSubTripayWebhook(ctx context.Context, rawBody []byte, signature string, payload payment.TripayCallbackPayload) error {
	pgCfg := s.loadSAPGConfig(ctx)
	if pgCfg == nil || pgCfg.SecretKey == "" {
		return fmt.Errorf("payment gateway superadmin belum dikonfigurasi")
	}

	tripayClient := payment.NewTripayClient(pgCfg.APIKey, pgCfg.SecretKey, pgCfg.MerchantID, pgCfg.Sandbox)
	if !tripayClient.VerifyCallbackSignature(rawBody, signature) {
		return fmt.Errorf("signature webhook Tripay tidak valid")
	}

	if payload.Status != "PAID" {
		log.Printf("[subscription webhook] tripay status=%s (ignored)", payload.Status)
		return nil
	}

	order, err := s.subRepo.FindOrderByPaymentRef(ctx, payload.Reference)
	if err != nil {
		return retryable(err)
	}
	if order == nil {
		return fmt.Errorf("subscription order not found for ref=%s", payload.Reference)
	}
	if order.Status != "pending" {
		return nil // already processed
	}

	_, err = s.ConfirmPayment(ctx, order.ID, payload.PaymentMethod, payload.Reference)
	return retryable(err)
}

// ProcessSubMidtransWebhook handles Midtrans notification for subscription orders.
func (s *SubscriptionService) ProcessSubMidtransWebhook(ctx context.Context, n payment.MidtransNotification) error {
	pgCfg := s.loadSAPGConfig(ctx)
	if pgCfg == nil || pgCfg.SecretKey == "" {
		return fmt.Errorf("payment gateway superadmin belum dikonfigurasi")
	}

	midtransClient := payment.NewMidtransClient(pgCfg.SecretKey, pgCfg.Sandbox)
	if !midtransClient.VerifyWebhookSignature(n) {
		return fmt.Errorf("signature webhook Midtrans tidak valid")
	}

	if n.TransactionStatus != "capture" && n.TransactionStatus != "settlement" {
		log.Printf("[subscription webhook] midtrans status=%s (ignored)", n.TransactionStatus)
		return nil
	}

	order, err := s.subRepo.FindOrderByPaymentRef(ctx, n.OrderID)
	if err != nil {
		return retryable(err)
	}
	if order == nil {
		return fmt.Errorf("subscription order not found for ref=%s", n.OrderID)
	}
	if order.Status != "pending" {
		return nil
	}

	_, err = s.ConfirmPayment(ctx, order.ID, n.PaymentType, n.TransactionID)
	return retryable(err)
}

// ProcessSubXenditWebhook handles Xendit Invoice callback for subscription orders.
func (s *SubscriptionService) ProcessSubXenditWebhook(ctx context.Context, callbackToken string, payload payment.XenditCallbackPayload) error {
	pgCfg := s.loadSAPGConfig(ctx)
	if pgCfg == nil || pgCfg.SecretKey == "" {
		return fmt.Errorf("payment gateway superadmin belum dikonfigurasi")
	}

	// For Xendit: APIKey = secret key, SecretKey = webhook verification token
	xenditClient := payment.NewXenditClient(pgCfg.APIKey, pgCfg.SecretKey, pgCfg.Sandbox)
	if !xenditClient.VerifyWebhookToken(callbackToken) {
		return fmt.Errorf("token webhook Xendit tidak valid")
	}

	if !payment.IsXenditPaymentSuccess(payload) {
		log.Printf("[subscription webhook] xendit status=%s (ignored)", payload.Status)
		return nil
	}

	order, err := s.subRepo.FindOrderByPaymentRef(ctx, payload.ExternalID)
	if err != nil {
		return retryable(err)
	}
	if order == nil {
		return fmt.Errorf("subscription order not found for external_id=%s", payload.ExternalID)
	}
	if order.Status != "pending" {
		return nil // already processed
	}

	method := payload.PaymentMethod
	if payload.PaymentChannel != "" {
		method = payload.PaymentChannel
	}

	_, err = s.ConfirmPayment(ctx, order.ID, method, payload.ID)
	return retryable(err)
}

func (s *SubscriptionService) sendPaymentConfirmationWA(order *model.SubscriptionOrder, plan *model.SubscriptionPlan, expiresAt *time.Time) {
	if s.waClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := s.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil || tenant == nil || tenant.Phone == "" {
		return
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	salam := greetingByTimeWIB()
	expiryStr := "-"
	if expiresAt != nil {
		expiryStr = expiresAt.In(loc).Format("02 January 2006")
	}
	durasi := fmt.Sprintf("%d bulan", order.DurationMonths)
	jumlah := formatAmountSub(order.Amount)

	defaultTpl := "✅ *Pembayaran Langganan Berhasil!*\n\nSelamat {salam} 🎉\n\nKepada Tim *{nama_isp}*,\n\nPembayaran langganan D Radius Anda telah berhasil kami terima.\n\n📋 *Detail Pembayaran:*\n• Paket: *{nama_paket}*\n• Durasi: *{durasi}*\n• Total: *Rp{jumlah}*\n• Aktif Hingga: *{tanggal_berakhir}*\n\nTerima kasih atas kepercayaan Anda. 🙏"

	tpl := defaultTpl
	if s.reminderRepo != nil && s.tenantRepo != nil {
		if saT, err := s.tenantRepo.FindBySlug(ctx, "superadmin"); err == nil && saT != nil {
			if rem, err := s.reminderRepo.FindActiveByType(ctx, saT.ID, "sub_payment"); err == nil && rem != nil {
				tpl = rem.MessageTemplate
			}
		}
	}

	msg := strings.ReplaceAll(tpl, "{salam}", salam)
	msg = strings.ReplaceAll(msg, "{nama_isp}", tenant.Name)
	msg = strings.ReplaceAll(msg, "{nama_paket}", plan.Name)
	msg = strings.ReplaceAll(msg, "{durasi}", durasi)
	msg = strings.ReplaceAll(msg, "{jumlah}", jumlah)
	msg = strings.ReplaceAll(msg, "{tanggal_berakhir}", expiryStr)

	if _, err := s.waClient.SendMessage(ctx, "superadmin", tenant.Phone, msg); err != nil {
		log.Printf("[sub-payment-wa] send failed for tenant %s: %v", tenant.ID, err)
	}
}

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

func formatAmountSub(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if (n-i)%3 == 0 && i != 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
