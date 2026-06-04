package service

import (
	"context"
	"errors"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/billing"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrBillingProfileNotFound  = errors.New("profil billing tidak ditemukan")
	ErrBillingProfileInUse     = errors.New("profil billing masih digunakan oleh pelanggan")
	ErrBillingProfileNameEmpty = errors.New("nama profil billing wajib diisi")
	ErrBillingModelInvalid     = errors.New("model billing tidak valid (cycle atau fixed)")
	ErrPaymentTimingInvalid    = errors.New("payment timing tidak valid (due_date atau advance)")
)

type BillingProfileService struct {
	repo repository.BillingProfileRepository
}

func NewBillingProfileService(repo repository.BillingProfileRepository) *BillingProfileService {
	return &BillingProfileService{repo: repo}
}

type BillingProfileInput struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	BillingModel  string `json:"billing_model"`
	InvoiceDay    int    `json:"invoice_day"`
	DueDay        int    `json:"due_day"`
	IsolirDay     int    `json:"isolir_day"`
	GracePeriod   int    `json:"grace_period"`
	PaymentTiming string `json:"payment_timing"`
	IsActive      bool   `json:"is_active"`
}

func (s *BillingProfileService) validate(input BillingProfileInput) error {
	if input.Name == "" {
		return ErrBillingProfileNameEmpty
	}
	if input.BillingModel != "cycle" && input.BillingModel != "fixed" {
		return ErrBillingModelInvalid
	}
	if input.PaymentTiming != "due_date" && input.PaymentTiming != "advance" {
		return ErrPaymentTimingInvalid
	}
	return nil
}

func (s *BillingProfileService) List(ctx context.Context, tenantID string) ([]model.BillingProfile, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *BillingProfileService) GetByID(ctx context.Context, tenantID, profileID string) (*model.BillingProfile, error) {
	p, err := s.repo.FindByID(ctx, tenantID, profileID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrBillingProfileNotFound
	}
	return p, nil
}

func (s *BillingProfileService) Create(ctx context.Context, tenantID string, input BillingProfileInput) (*model.BillingProfile, error) {
	if err := s.validate(input); err != nil {
		return nil, err
	}

	invoiceDay, dueDay := s.normalizeDays(input)

	p := &model.BillingProfile{
		ID:            id.New(),
		TenantID:      tenantID,
		Name:          input.Name,
		Description:   input.Description,
		BillingModel:  input.BillingModel,
		InvoiceDay:    invoiceDay,
		DueDay:        dueDay,
		IsolirDay:     clampMin(input.IsolirDay, 0),
		GracePeriod:   clampMin(input.GracePeriod, 0),
		PaymentTiming: input.PaymentTiming,
		IsActive:      input.IsActive,
	}

	// First profile created → set as default automatically
	existing, _ := s.repo.List(ctx, tenantID)
	if len(existing) == 0 {
		p.IsDefault = true
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *BillingProfileService) Update(ctx context.Context, tenantID, profileID string, input BillingProfileInput) (*model.BillingProfile, error) {
	p, err := s.repo.FindByID(ctx, tenantID, profileID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrBillingProfileNotFound
	}

	if err := s.validate(input); err != nil {
		return nil, err
	}

	invoiceDay, dueDay := s.normalizeDays(input)

	p.Name = input.Name
	p.Description = input.Description
	p.BillingModel = input.BillingModel
	p.InvoiceDay = invoiceDay
	p.DueDay = dueDay
	p.IsolirDay = clampMin(input.IsolirDay, 0)
	p.GracePeriod = clampMin(input.GracePeriod, 0)
	p.PaymentTiming = input.PaymentTiming
	p.IsActive = input.IsActive

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *BillingProfileService) Delete(ctx context.Context, tenantID, profileID string) error {
	p, err := s.repo.FindByID(ctx, tenantID, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrBillingProfileNotFound
	}
	return s.repo.Delete(ctx, tenantID, profileID)
}

func (s *BillingProfileService) SetDefault(ctx context.Context, tenantID, profileID string) error {
	p, err := s.repo.FindByID(ctx, tenantID, profileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrBillingProfileNotFound
	}
	return s.repo.SetDefault(ctx, tenantID, profileID)
}

// normalizeDays validates and normalizes invoice_day and due_day based on billing model.
// cycle: due_day 1-28, invoice_day 0 (auto) or 1-28
// fixed: invoice_day = days before due (0 = auto 7), due_day not used
func (s *BillingProfileService) normalizeDays(input BillingProfileInput) (invoiceDay, dueDay int) {
	if input.BillingModel == "cycle" {
		dueDay = billing.NormalizeDay(input.DueDay)
		invoiceDay = input.InvoiceDay
		if invoiceDay < 0 {
			invoiceDay = 0
		}
		if invoiceDay > 28 {
			invoiceDay = 28
		}
		return invoiceDay, dueDay
	}
	// fixed: due_day not used, invoice_day = days before JT
	invoiceDay = input.InvoiceDay
	if invoiceDay < 0 {
		invoiceDay = 0
	}
	return invoiceDay, 0
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}
