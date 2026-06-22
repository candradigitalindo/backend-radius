package service

import (
	"context"
	"errors"
	"log"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var ErrResellerNotFound = errors.New("Reseller tidak ditemukan")

type ResellerService struct {
	resellerRepo repository.ResellerRepository
}

func NewResellerService(resellerRepo repository.ResellerRepository) *ResellerService {
	return &ResellerService{resellerRepo: resellerRepo}
}

func (s *ResellerService) Create(ctx context.Context, reseller *model.Reseller) error {
	return s.resellerRepo.Create(ctx, reseller)
}

func (s *ResellerService) GetByID(ctx context.Context, tenantID, resellerID string) (*model.Reseller, error) {
	return s.resellerRepo.GetByID(ctx, tenantID, resellerID)
}

func (s *ResellerService) Update(ctx context.Context, reseller *model.Reseller) error {
	return s.resellerRepo.Update(ctx, reseller)
}

func (s *ResellerService) Delete(ctx context.Context, tenantID, resellerID string) error {
	r, err := s.resellerRepo.GetByID(ctx, tenantID, resellerID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrResellerNotFound
	}
	return s.resellerRepo.Delete(ctx, tenantID, resellerID)
}

func (s *ResellerService) List(ctx context.Context, tenantID string, filter repository.ResellerFilter) ([]model.Reseller, int, error) {
	return s.resellerRepo.List(ctx, tenantID, filter)
}

func (s *ResellerService) AddCommission(ctx context.Context, comm *model.ResellerCommission) error {
	return s.resellerRepo.AddCommission(ctx, comm)
}

func (s *ResellerService) ListCommissions(ctx context.Context, tenantID, resellerID string, page, perPage int) ([]model.ResellerCommission, int, error) {
	return s.resellerRepo.ListCommissions(ctx, tenantID, resellerID, page, perPage)
}

func (s *ResellerService) PayCommission(ctx context.Context, tenantID, commissionID string) error {
	return s.resellerRepo.PayCommission(ctx, tenantID, commissionID)
}

func (s *ResellerService) PayAllPending(ctx context.Context, tenantID, resellerID string) (int, error) {
	return s.resellerRepo.PayAllPending(ctx, tenantID, resellerID)
}

func (s *ResellerService) GetCommissionSummary(ctx context.Context, tenantID, resellerID string) (*repository.CommissionSummary, error) {
	return s.resellerRepo.GetCommissionSummary(ctx, tenantID, resellerID)
}

func (s *ResellerService) ListCustomers(ctx context.Context, tenantID, resellerID string) ([]model.Customer, error) {
	return s.resellerRepo.ListCustomers(ctx, tenantID, resellerID)
}

// ProcessCommissionOnPayment auto-creates a pending commission for the customer's
// reseller (if any) when an invoice is paid. Commission = invoiceAmount × rate%.
// Idempotent per invoice, so repeated webhooks don't double-credit.
func (s *ResellerService) ProcessCommissionOnPayment(ctx context.Context, tenantID, customerID, invoiceID string, invoiceAmount int64) {
	reseller, err := s.resellerRepo.FindResellerByCustomer(ctx, tenantID, customerID)
	if err != nil {
		log.Printf("[reseller] find reseller for customer %s: %v", customerID, err)
		return
	}
	if reseller == nil || reseller.Status != "active" || reseller.CommissionRate <= 0 {
		return
	}

	exists, err := s.resellerRepo.CommissionExistsForInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		log.Printf("[reseller] check commission for invoice %s: %v", invoiceID, err)
		return
	}
	if exists {
		return
	}

	amount := int64(float64(invoiceAmount) * reseller.CommissionRate / 100.0)
	if amount <= 0 {
		return
	}

	comm := &model.ResellerCommission{
		TenantID:   tenantID,
		ResellerID: reseller.ID,
		InvoiceID:  invoiceID,
		CustomerID: customerID,
		Amount:     amount,
		Status:     "pending",
	}
	if err := s.resellerRepo.AddCommission(ctx, comm); err != nil {
		log.Printf("[reseller] auto-create commission (reseller %s, invoice %s): %v", reseller.ID, invoiceID, err)
	}
}
