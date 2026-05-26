package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrTenantNotFound  = errors.New("Tenant tidak ditemukan")
	ErrSlugAlreadyUsed = errors.New("Slug sudah digunakan")
)

type TenantService struct {
	tenantRepo repository.TenantRepository
}

func NewTenantService(tenantRepo repository.TenantRepository) *TenantService {
	return &TenantService{tenantRepo: tenantRepo}
}

type CreateTenantInput struct {
	Name                string
	Slug                string
	Email               string
	Phone               string
	Address             string
	Timezone            string
	Currency            string
	BillingCycle        int
	DueDay              int
	IsolirDay           int
	GracePeriod         int
	DefaultBillingType  string
	Plan                string
	MaxCustomers        int
}

type UpdateTenantInput struct {
	Name                string
	Email               string
	Phone               string
	Address             string
	LogoURL             string
	Timezone            string
	Currency            string
	BillingCycle        int
	DueDay              int
	IsolirDay           int
	GracePeriod         int
	DefaultBillingType  string
	IsActive            bool
}

type UpdateSettingsInput struct {
	WAAPIKey     string
	WASender     string
	PGProvider   string
	PGAPIKey     string
	PGSecretKey  string
	PGMerchantID string
	PGSandbox    bool
}

func (s *TenantService) Create(ctx context.Context, input CreateTenantInput) (*model.Tenant, error) {
	existing, err := s.tenantRepo.FindBySlug(ctx, input.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugAlreadyUsed
	}

	if input.Timezone == "" {
		input.Timezone = "Asia/Jakarta"
	}
	if input.Currency == "" {
		input.Currency = "IDR"
	}
	if input.BillingCycle == 0 {
		input.BillingCycle = 1
	}
	if input.DueDay == 0 {
		input.DueDay = 20
	}
	if input.IsolirDay == 0 {
		input.IsolirDay = 21
	}
	if input.GracePeriod == 0 {
		input.GracePeriod = 3
	}
	if input.Plan == "" {
		input.Plan = "trial"
	}
	if input.MaxCustomers == 0 {
		input.MaxCustomers = 100
	}
	if input.DefaultBillingType == "" {
		input.DefaultBillingType = "fixed"
	}

	tenant := &model.Tenant{
		Name:         input.Name,
		Slug:         input.Slug,
		Email:        input.Email,
		Phone:        input.Phone,
		Address:      input.Address,
		Timezone:     input.Timezone,
		Currency:     input.Currency,
		BillingCycle:        input.BillingCycle,
		DueDay:              input.DueDay,
		IsolirDay:           input.IsolirDay,
		GracePeriod:         input.GracePeriod,
		DefaultBillingType:  input.DefaultBillingType,
		Plan:                input.Plan,
		MaxCustomers: input.MaxCustomers,
		IsActive:     true,
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return s.tenantRepo.FindByID(ctx, tenant.ID)
}

func (s *TenantService) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

func (s *TenantService) Update(ctx context.Context, tenantID string, input UpdateTenantInput) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	tenant.Name = input.Name
	tenant.Email = input.Email
	tenant.Phone = input.Phone
	tenant.Address = input.Address
	tenant.LogoURL = input.LogoURL
	tenant.Timezone = input.Timezone
	tenant.Currency = input.Currency
	tenant.BillingCycle = input.BillingCycle
	tenant.DueDay = input.DueDay
	tenant.IsolirDay = input.IsolirDay
	tenant.GracePeriod = input.GracePeriod
	if input.DefaultBillingType != "" {
		tenant.DefaultBillingType = input.DefaultBillingType
	}
	tenant.IsActive = input.IsActive

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return s.tenantRepo.FindByID(ctx, tenantID)
}

func (s *TenantService) UpdateSettings(ctx context.Context, tenantID string, input UpdateSettingsInput) error {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	tenant.WAAPIKey = input.WAAPIKey
	tenant.WASender = input.WASender
	tenant.PGProvider = input.PGProvider
	tenant.PGAPIKey = input.PGAPIKey
	tenant.PGSecretKey = input.PGSecretKey
	tenant.PGMerchantID = input.PGMerchantID
	tenant.PGSandbox = input.PGSandbox

	return s.tenantRepo.UpdateSettings(ctx, tenant)
}

func (s *TenantService) List(ctx context.Context, filter repository.TenantFilter) ([]model.Tenant, int, error) {
	return s.tenantRepo.List(ctx, filter)
}

// TestPGConnection verifies the payment gateway credentials configured for the tenant.
// It instantiates the appropriate client and calls a lightweight authenticated endpoint.
// Returns nil if the connection is successful, or an error with a descriptive message.
func (s *TenantService) TestPGConnection(ctx context.Context, tenantID string) error {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	if tenant.PGProvider == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi")
	}
	if tenant.PGAPIKey == "" {
		return fmt.Errorf("API Key payment gateway belum diisi")
	}

	switch tenant.PGProvider {
	case "tripay":
		if tenant.PGSecretKey == "" || tenant.PGMerchantID == "" {
			return fmt.Errorf("Tripay: Private Key dan Merchant Code wajib diisi")
		}
		client := payment.NewTripayClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGMerchantID, tenant.PGSandbox)
		return client.TestConnection(ctx)

	case "midtrans":
		client := payment.NewMidtransClient(tenant.PGAPIKey, tenant.PGSandbox)
		return client.TestConnection(ctx)

	case "xendit":
		client := payment.NewXenditClient(tenant.PGAPIKey, tenant.PGSecretKey, tenant.PGSandbox)
		return client.TestConnection(ctx)

	default:
		return fmt.Errorf("provider payment gateway '%s' tidak didukung", tenant.PGProvider)
	}
}
