package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrVoucherProductNotFound = errors.New("Produk voucher tidak ditemukan")
	ErrVoucherNotFound        = errors.New("Voucher tidak ditemukan")
	ErrVoucherNotAvailable    = errors.New("Voucher tidak tersedia")
)

type VoucherService struct {
	productRepo repository.VoucherProductRepository
	voucherRepo repository.VoucherRepository
	paymentRepo repository.VoucherPaymentRepository
	tenantRepo  repository.TenantRepository
	waClient    *whatsapp.Client
	baseURL     string
}

func NewVoucherService(
	productRepo repository.VoucherProductRepository,
	voucherRepo repository.VoucherRepository,
	paymentRepo repository.VoucherPaymentRepository,
) *VoucherService {
	return &VoucherService{
		productRepo: productRepo,
		voucherRepo: voucherRepo,
		paymentRepo: paymentRepo,
	}
}

func (s *VoucherService) WithTenantRepo(tenantRepo repository.TenantRepository) *VoucherService {
	s.tenantRepo = tenantRepo
	return s
}

func (s *VoucherService) WithWAClient(waClient *whatsapp.Client) *VoucherService {
	s.waClient = waClient
	return s
}

func (s *VoucherService) WithBaseURL(baseURL string) *VoucherService {
	s.baseURL = baseURL
	return s
}

// -- Voucher Product --

type CreateVoucherProductInput struct {
	TenantID      string
	Name          string
	Duration      int
	BandwidthUp   int
	BandwidthDown int
	Price         int64
	ProfileName   string
	RouterID      *string
}

type UpdateVoucherProductInput struct {
	Name          string
	Duration      int
	BandwidthUp   int
	BandwidthDown int
	Price         int64
	ProfileName   string
	RouterID      *string
	IsActive      bool
}

func (s *VoucherService) CreateProduct(ctx context.Context, input CreateVoucherProductInput) (*model.VoucherProduct, error) {
	product := &model.VoucherProduct{
		TenantID:      input.TenantID,
		Name:          input.Name,
		Duration:      input.Duration,
		BandwidthUp:   input.BandwidthUp,
		BandwidthDown: input.BandwidthDown,
		Price:         input.Price,
		ProfileName:   input.ProfileName,
		RouterID:      input.RouterID,
		IsActive:      true,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(ctx, input.TenantID, product.ID)
}

func (s *VoucherService) GetProduct(ctx context.Context, tenantID, productID string) (*model.VoucherProduct, error) {
	product, err := s.productRepo.FindByID(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrVoucherProductNotFound
	}
	return product, nil
}

func (s *VoucherService) UpdateProduct(ctx context.Context, tenantID, productID string, input UpdateVoucherProductInput) (*model.VoucherProduct, error) {
	product, err := s.productRepo.FindByID(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrVoucherProductNotFound
	}

	product.Name = input.Name
	product.Duration = input.Duration
	product.BandwidthUp = input.BandwidthUp
	product.BandwidthDown = input.BandwidthDown
	product.Price = input.Price
	product.ProfileName = input.ProfileName
	product.RouterID = input.RouterID
	product.IsActive = input.IsActive

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return s.productRepo.FindByID(ctx, tenantID, productID)
}

func (s *VoucherService) DeleteProduct(ctx context.Context, tenantID, productID string) error {
	product, err := s.productRepo.FindByID(ctx, tenantID, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrVoucherProductNotFound
	}
	return s.productRepo.Delete(ctx, tenantID, productID)
}

func (s *VoucherService) ListProducts(ctx context.Context, tenantID string, filter repository.VoucherProductFilter) ([]model.VoucherProduct, int, error) {
	return s.productRepo.List(ctx, tenantID, filter)
}

// -- Voucher --

type GenerateVouchersInput struct {
	TenantID  string
	ProductID string
	Quantity  int
	Prefix    string
}

func (s *VoucherService) Generate(ctx context.Context, input GenerateVouchersInput) (int, error) {
	product, err := s.productRepo.FindByID(ctx, input.TenantID, input.ProductID)
	if err != nil {
		return 0, err
	}
	if product == nil {
		return 0, ErrVoucherProductNotFound
	}

	vouchers := make([]model.Voucher, input.Quantity)
	for i := 0; i < input.Quantity; i++ {
		username := generateCode(input.Prefix, 8)
		password := generateCode("", 6)

		vouchers[i] = model.Voucher{
			TenantID:  input.TenantID,
			ProductID: input.ProductID,
			Username:  username,
			Password:  password,
			Status:    "available",
		}
	}

	if err := s.voucherRepo.CreateBatch(ctx, vouchers); err != nil {
		return 0, err
	}

	return input.Quantity, nil
}

func (s *VoucherService) GetVoucher(ctx context.Context, tenantID, voucherID string) (*model.Voucher, error) {
	voucher, err := s.voucherRepo.FindByID(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, ErrVoucherNotFound
	}
	return voucher, nil
}

func (s *VoucherService) DeleteVoucher(ctx context.Context, tenantID, voucherID string) error {
	voucher, err := s.voucherRepo.FindByID(ctx, tenantID, voucherID)
	if err != nil {
		return err
	}
	if voucher == nil {
		return ErrVoucherNotFound
	}
	return s.voucherRepo.Delete(ctx, tenantID, voucherID)
}

func (s *VoucherService) ListVouchers(ctx context.Context, tenantID string, filter repository.VoucherFilter) ([]model.Voucher, int, error) {
	return s.voucherRepo.List(ctx, tenantID, filter)
}

func (s *VoucherService) SellVoucher(ctx context.Context, tenantID, voucherID, buyerPhone string) (*model.Voucher, error) {
	voucher, err := s.voucherRepo.FindByID(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, ErrVoucherNotFound
	}
	if voucher.Status != "available" {
		return nil, ErrVoucherNotAvailable
	}

	voucher.BuyerPhone = &buyerPhone
	if err := s.voucherRepo.MarkSold(ctx, voucher); err != nil {
		return nil, err
	}

	return s.voucherRepo.FindByID(ctx, tenantID, voucherID)
}

func (s *VoucherService) CountAvailable(ctx context.Context, tenantID, productID string) (int, error) {
	return s.voucherRepo.CountByProduct(ctx, tenantID, productID, "available")
}

// -- Voucher Payments --

func (s *VoucherService) ListPayments(ctx context.Context, tenantID string, filter repository.VoucherPaymentFilter) ([]model.VoucherPayment, int, error) {
	return s.paymentRepo.List(ctx, tenantID, filter)
}

// generateCode creates a random alphanumeric code with an optional prefix.
func generateCode(prefix string, length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, length)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, string(code))
	}
	return string(code)
}

// ActivateVoucher marks a voucher as active and sets expiration based on product duration.
func (s *VoucherService) ActivateVoucher(ctx context.Context, tenantID, voucherID string) (*model.Voucher, error) {
	voucher, err := s.voucherRepo.FindByID(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, ErrVoucherNotFound
	}
	if voucher.Status != "sold" && voucher.Status != "available" {
		return nil, ErrVoucherNotAvailable
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(voucher.Product.Duration) * 24 * time.Hour)

	if err := s.voucherRepo.Activate(ctx, voucherID, now, expiresAt); err != nil {
		return nil, err
	}

	return s.voucherRepo.FindByID(ctx, tenantID, voucherID)
}

// --- Public Voucher Store ---

// PublicProductInfo is the limited product info for the public store.
type PublicProductInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Duration      int    `json:"duration"`
	BandwidthUp   int    `json:"bandwidth_up"`
	BandwidthDown int    `json:"bandwidth_down"`
	Price         int64  `json:"price"`
	Available     int    `json:"available"`
}

// GetPublicStoreProducts returns active voucher products for the public store page.
func (s *VoucherService) GetPublicStoreProducts(ctx context.Context, tenantID string) ([]PublicProductInfo, error) {
	active := true
	products, _, err := s.productRepo.List(ctx, tenantID, repository.VoucherProductFilter{
		Active:  &active,
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	var result []PublicProductInfo
	for _, p := range products {
		count, _ := s.voucherRepo.CountByProduct(ctx, tenantID, p.ID, "available")
		result = append(result, PublicProductInfo{
			ID:            p.ID,
			Name:          p.Name,
			Duration:      p.Duration,
			BandwidthUp:   p.BandwidthUp,
			BandwidthDown: p.BandwidthDown,
			Price:         p.Price,
			Available:     count,
		})
	}

	return result, nil
}

// PurchaseVoucherInput is the input for purchasing a voucher via payment gateway.
type PurchaseVoucherInput struct {
	TenantID      string
	ProductID     string
	BuyerName     string
	BuyerPhone    string
	PaymentMethod string
	ReturnURL     string
}

// PurchaseVoucherResult is the result of a voucher purchase.
type PurchaseVoucherResult struct {
	PaymentID    string     `json:"payment_id"`
	GatewayTrxID string     `json:"gateway_trx_id"`
	PaymentURL   string     `json:"payment_url"`
	ExpiredAt    *time.Time `json:"expired_at"`
}

// PurchaseVoucher picks an available voucher, creates a gateway transaction, and records a pending payment.
func (s *VoucherService) PurchaseVoucher(ctx context.Context, input PurchaseVoucherInput) (*PurchaseVoucherResult, error) {
	if s.tenantRepo == nil {
		return nil, fmt.Errorf("tenant repo tidak dikonfigurasi")
	}

	// Find an available voucher for this product
	vouchers, _, err := s.voucherRepo.List(ctx, input.TenantID, repository.VoucherFilter{
		ProductID: input.ProductID,
		Status:    "available",
		Page:      1,
		PerPage:   1,
	})
	if err != nil {
		return nil, fmt.Errorf("find available voucher: %w", err)
	}
	if len(vouchers) == 0 {
		return nil, ErrVoucherNotAvailable
	}
	voucher := vouchers[0]

	product, err := s.productRepo.FindByID(ctx, input.TenantID, voucher.ProductID)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}
	if product == nil {
		return nil, ErrVoucherProductNotFound
	}

	tenant, err := s.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant: %w", err)
	}
	if tenant == nil || tenant.PGAPIKey == "" {
		return nil, fmt.Errorf("payment gateway belum dikonfigurasi untuk tenant ini")
	}

	// Reserve voucher by marking as sold
	voucher.BuyerPhone = &input.BuyerPhone
	if err := s.voucherRepo.MarkSold(ctx, &voucher); err != nil {
		return nil, fmt.Errorf("reserve voucher: %w", err)
	}

	// Create gateway transaction
	orderID := fmt.Sprintf("VCH-%s-%s", voucher.ID[:8], time.Now().Format("150405"))

	var gatewayTrxID, paymentURL, gateway string
	var expiredAt time.Time

	switch tenant.PGProvider {
	case "xendit":
		client := payment.NewXenditClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGSandbox)
		resp, err := client.CreateInvoice(ctx, payment.XenditCreateInvoiceRequest{
			ExternalID:      orderID,
			Amount:          product.Price,
			Description:     fmt.Sprintf("Voucher %s", product.Name),
			CustomerName:    input.BuyerName,
			CustomerPhone:   input.BuyerPhone,
			SuccessRedirect: input.ReturnURL,
			FailureRedirect: input.ReturnURL,
			CallbackURL:     s.baseURL + "/api/v1/webhooks/xendit/" + tenant.Slug + "/voucher",
		})
		if err != nil {
			return nil, fmt.Errorf("create xendit invoice: %w", err)
		}
		gatewayTrxID = orderID
		paymentURL = resp.InvoiceURL
		gateway = "xendit"
		expiredAt = time.Now().Add(24 * time.Hour)
		if resp.ExpiryDate != "" {
			if t, err := time.Parse(time.RFC3339, resp.ExpiryDate); err == nil {
				expiredAt = t
			}
		}

	case "midtrans":
		client := payment.NewMidtransClient(tenant.PGSecretKey, tenant.PGSandbox)
		resp, err := client.CreateTransaction(ctx, payment.MidtransCreateRequest{
			OrderID:         orderID,
			GrossAmount:     product.Price,
			FirstName:       input.BuyerName,
			Email:           "",
			Phone:           input.BuyerPhone,
			FinishURL:       input.ReturnURL,
			ErrorURL:        input.ReturnURL,
			ItemName:        fmt.Sprintf("Voucher %s", product.Name),
			NotificationURL: s.baseURL + "/api/v1/webhooks/midtrans/" + tenant.Slug + "/voucher",
		})
		if err != nil {
			return nil, fmt.Errorf("create midtrans transaction: %w", err)
		}
		gatewayTrxID = orderID
		paymentURL = resp.RedirectURL
		gateway = "midtrans"
		expiredAt = time.Now().Add(24 * time.Hour)

	default: // tripay
		client := payment.NewTripayClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGMerchantID, tenant.PGSandbox)
		resp, err := client.CreateTransaction(ctx, payment.CreateTransactionRequest{
			Method:        input.PaymentMethod,
			MerchantRef:   orderID,
			Amount:        product.Price,
			CustomerName:  input.BuyerName,
			CustomerPhone: input.BuyerPhone,
			ReturnURL:     input.ReturnURL,
			CallbackURL:   s.baseURL + "/api/v1/webhooks/tripay/" + tenant.Slug + "/voucher",
		})
		if err != nil {
			return nil, fmt.Errorf("create tripay transaction: %w", err)
		}
		gatewayTrxID = resp.Data.Reference
		paymentURL = resp.Data.PaymentURL
		gateway = "tripay"
		expiredAt = time.Unix(resp.Data.ExpiredTime, 0)
	}

	// Record voucher payment
	voucherPayment := &model.VoucherPayment{
		TenantID:     input.TenantID,
		VoucherID:    voucher.ID,
		BuyerName:    input.BuyerName,
		BuyerPhone:   input.BuyerPhone,
		Amount:       product.Price,
		Gateway:      gateway,
		GatewayTrxID: gatewayTrxID,
		Status:       "pending",
	}
	if err := s.paymentRepo.Create(ctx, voucherPayment); err != nil {
		return nil, fmt.Errorf("save voucher payment: %w", err)
	}

	return &PurchaseVoucherResult{
		PaymentID:    voucherPayment.ID,
		GatewayTrxID: gatewayTrxID,
		PaymentURL:   paymentURL,
		ExpiredAt:    &expiredAt,
	}, nil
}

// ProcessVoucherTripayWebhook handles a Tripay callback for a voucher payment.
func (s *VoucherService) ProcessVoucherTripayWebhook(ctx context.Context, tenantSlug string, payload payment.TripayCallbackPayload) error {
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repo tidak dikonfigurasi")
	}
	tenant, err := s.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant tidak ditemukan: %s", tenantSlug)
	}
	if tenant.PGSecretKey == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi")
	}
	client := payment.NewTripayClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGMerchantID, tenant.PGSandbox)
	if !client.VerifyWebhookSignature(payload) {
		return fmt.Errorf("signature Tripay tidak valid")
	}

	vp, err := s.paymentRepo.FindByGatewayTrxID(ctx, payload.Reference)
	if err != nil {
		return fmt.Errorf("find voucher payment: %w", err)
	}
	if vp == nil {
		return fmt.Errorf("pembayaran voucher tidak ditemukan untuk referensi %s", payload.Reference)
	}
	if vp.TenantID != tenant.ID {
		return fmt.Errorf("pembayaran tidak milik tenant ini")
	}

	switch payload.Status {
	case "PAID":
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "paid"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		s.sendVoucherTobuyer(ctx, vp)

	case "EXPIRED", "FAILED":
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "failed"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		// Release voucher back to available
		_ = s.voucherRepo.UpdateStatus(ctx, vp.VoucherID, "available")
	}

	return nil
}

// ProcessVoucherMidtransWebhook handles a Midtrans notification for a voucher payment.
func (s *VoucherService) ProcessVoucherMidtransWebhook(ctx context.Context, tenantSlug string, n payment.MidtransNotification) error {
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repo tidak dikonfigurasi")
	}
	tenant, err := s.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant tidak ditemukan: %s", tenantSlug)
	}
	if tenant.PGSecretKey == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi")
	}
	client := payment.NewMidtransClient(tenant.PGSecretKey, tenant.PGSandbox)
	if !client.VerifyWebhookSignature(n) {
		return fmt.Errorf("signature Midtrans tidak valid")
	}

	vp, err := s.paymentRepo.FindByGatewayTrxID(ctx, n.OrderID)
	if err != nil {
		return fmt.Errorf("find voucher payment: %w", err)
	}
	if vp == nil {
		return fmt.Errorf("pembayaran voucher tidak ditemukan untuk order_id %s", n.OrderID)
	}
	if vp.TenantID != tenant.ID {
		return fmt.Errorf("pembayaran tidak milik tenant ini")
	}

	if payment.IsPaymentSuccess(n) {
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "paid"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		s.sendVoucherTobuyer(ctx, vp)

	} else if payment.IsPaymentFailed(n) {
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "failed"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		// Release voucher back to available
		_ = s.voucherRepo.UpdateStatus(ctx, vp.VoucherID, "available")
	}

	return nil
}

// ProcessVoucherXenditWebhook handles a Xendit Invoice callback for a voucher payment.
func (s *VoucherService) ProcessVoucherXenditWebhook(ctx context.Context, tenantSlug, callbackToken string, payload payment.XenditCallbackPayload) error {
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repo tidak dikonfigurasi")
	}
	tenant, err := s.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant tidak ditemukan: %s", tenantSlug)
	}
	if tenant.PGSecretKey == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi")
	}
	client := payment.NewXenditClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGSandbox)
	if !client.VerifyWebhookToken(callbackToken) {
		return fmt.Errorf("token Xendit tidak valid")
	}

	vp, err := s.paymentRepo.FindByGatewayTrxID(ctx, payload.ExternalID)
	if err != nil {
		return fmt.Errorf("find voucher payment: %w", err)
	}
	if vp == nil {
		return fmt.Errorf("pembayaran voucher tidak ditemukan untuk external_id %s", payload.ExternalID)
	}
	if vp.TenantID != tenant.ID {
		return fmt.Errorf("pembayaran tidak milik tenant ini")
	}

	if payment.IsXenditPaymentSuccess(payload) {
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "paid"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		s.sendVoucherTobuyer(ctx, vp)

	} else if payment.IsXenditPaymentFailed(payload) {
		if err := s.paymentRepo.UpdateStatus(ctx, vp.ID, "failed"); err != nil {
			return fmt.Errorf("update voucher payment: %w", err)
		}
		_ = s.voucherRepo.UpdateStatus(ctx, vp.VoucherID, "available")
	}

	return nil
}

// sendVoucherTobuyer sends the voucher credentials to the buyer via WhatsApp.
func (s *VoucherService) sendVoucherTobuyer(ctx context.Context, vp *model.VoucherPayment) {
	voucher, err := s.voucherRepo.FindByID(ctx, vp.TenantID, vp.VoucherID)
	if err != nil || voucher == nil {
		log.Printf("[voucher] failed to load voucher %s for WA delivery: %v", vp.VoucherID, err)
		return
	}

	productName := "Voucher"
	if voucher.Product != nil {
		productName = voucher.Product.Name
	}

	message := fmt.Sprintf(
		"Terima kasih sudah membeli voucher %s!\n\n"+
			"Username: %s\nPassword: %s\n\n"+
			"Silakan login ke hotspot dengan kredensial di atas.",
		productName, voucher.Username, voucher.Password,
	)

	if s.waClient != nil {
		if _, err := s.waClient.SendMessage(ctx, vp.TenantID, vp.BuyerPhone, message); err != nil {
			log.Printf("[voucher] WA send to %s failed: %v", vp.BuyerPhone, err)
		}
	}
}
