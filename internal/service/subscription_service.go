package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrPlanNotFound  = errors.New("Paket tidak ditemukan")
	ErrOrderNotFound = errors.New("Order tidak ditemukan")
	ErrPlanInactive  = errors.New("Paket tidak aktif")
	ErrPendingOrder  = errors.New("Masih ada order yang belum dibayar")
)

type SubscriptionService struct {
	subRepo    repository.SubscriptionRepository
	tenantRepo repository.TenantRepository
	pgCfg      *config.PGConfig
	baseURL    string
}

func NewSubscriptionService(subRepo repository.SubscriptionRepository, tenantRepo repository.TenantRepository) *SubscriptionService {
	return &SubscriptionService{subRepo: subRepo, tenantRepo: tenantRepo}
}

func (s *SubscriptionService) WithPG(pgCfg *config.PGConfig, baseURL string) *SubscriptionService {
	s.pgCfg = pgCfg
	s.baseURL = baseURL
	return s
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	return s.subRepo.ListPlans(ctx, true)
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

	// Calculate amount: yearly gets 10% discount
	amount := plan.Price * int64(duration)
	if duration == 12 {
		amount = amount * 90 / 100
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
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
}

// CreatePayment generates a payment gateway transaction for a pending subscription order.
func (s *SubscriptionService) CreatePayment(ctx context.Context, orderID, tenantID, returnURL string) (*SubPaymentResult, error) {
	if s.pgCfg == nil || s.pgCfg.APIKey == "" {
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
		}, nil
	}

	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	merchantRef := "SUB-" + order.ID
	callbackURL := s.baseURL + "/api/v1/webhooks/subscription/tripay"

	switch s.pgCfg.Provider {
	case "midtrans":
		return s.createSubMidtransPayment(ctx, order, tenant, merchantRef, returnURL)
	default:
		return s.createSubTripayPayment(ctx, order, tenant, merchantRef, returnURL, callbackURL)
	}
}

func (s *SubscriptionService) createSubTripayPayment(ctx context.Context, order *model.SubscriptionOrder, tenant *model.Tenant, merchantRef, returnURL, callbackURL string) (*SubPaymentResult, error) {
	client := payment.NewTripayClient(s.pgCfg.APIKey, s.pgCfg.SecretKey, s.pgCfg.MerchantID, s.pgCfg.Sandbox)

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

func (s *SubscriptionService) createSubMidtransPayment(ctx context.Context, order *model.SubscriptionOrder, tenant *model.Tenant, merchantRef, returnURL string) (*SubPaymentResult, error) {
	client := payment.NewMidtransClient(s.pgCfg.SecretKey, s.pgCfg.Sandbox)

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
	if err := s.subRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	return &SubPaymentResult{
		OrderID:    order.ID,
		PaymentURL: resp.RedirectURL,
	}, nil
}

// ProcessSubTripayWebhook handles Tripay callback for subscription orders.
func (s *SubscriptionService) ProcessSubTripayWebhook(ctx context.Context, payload payment.TripayCallbackPayload) error {
	if payload.Status != "PAID" {
		log.Printf("[subscription webhook] tripay status=%s (ignored)", payload.Status)
		return nil
	}

	// Find order by payment ref
	order, err := s.subRepo.FindOrderByPaymentRef(ctx, payload.Reference)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("subscription order not found for ref=%s", payload.Reference)
	}
	if order.Status != "pending" {
		return nil // already processed
	}

	_, err = s.ConfirmPayment(ctx, order.ID, payload.PaymentMethod, payload.Reference)
	return err
}

// ProcessSubMidtransWebhook handles Midtrans notification for subscription orders.
func (s *SubscriptionService) ProcessSubMidtransWebhook(ctx context.Context, n payment.MidtransNotification) error {
	if n.TransactionStatus != "capture" && n.TransactionStatus != "settlement" {
		log.Printf("[subscription webhook] midtrans status=%s (ignored)", n.TransactionStatus)
		return nil
	}

	order, err := s.subRepo.FindOrderByPaymentRef(ctx, n.OrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("subscription order not found for ref=%s", n.OrderID)
	}
	if order.Status != "pending" {
		return nil
	}

	_, err = s.ConfirmPayment(ctx, order.ID, n.PaymentType, n.TransactionID)
	return err
}

// ProcessSubXenditWebhook handles Xendit Invoice callback for subscription orders.
func (s *SubscriptionService) ProcessSubXenditWebhook(ctx context.Context, callbackToken string, payload payment.XenditCallbackPayload) error {
	if !payment.IsXenditPaymentSuccess(payload) {
		log.Printf("[subscription webhook] xendit status=%s (ignored)", payload.Status)
		return nil
	}

	order, err := s.subRepo.FindOrderByPaymentRef(ctx, payload.ExternalID)
	if err != nil {
		return err
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
	return err
}
