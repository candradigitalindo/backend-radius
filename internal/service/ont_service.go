package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrONTNotFound = errors.New("ONT tidak ditemukan")
)

type ONTService struct {
	ontRepo     repository.ONTRepository
	logRepo     repository.CustomerLogRepository
}

func NewONTService(ontRepo repository.ONTRepository) *ONTService {
	return &ONTService{ontRepo: ontRepo}
}

// WithCustomerLog enables customer transfer logging.
func (s *ONTService) WithCustomerLog(logRepo repository.CustomerLogRepository) *ONTService {
	s.logRepo = logRepo
	return s
}

type CreateONTInput struct {
	TenantID     string
	SerialNumber string
	CustomerID   *string
	Vendor       *string
	Model        *string
	Notes        *string
}

type UpdateONTInput struct {
	SerialNumber string
	CustomerID   *string
	Vendor       *string
	Model        *string
	Status       string
	Notes        *string
	PerformedBy  *string
}

func (s *ONTService) Create(ctx context.Context, input CreateONTInput) (*model.ONT, error) {
	ont := &model.ONT{
		TenantID:     input.TenantID,
		SerialNumber: input.SerialNumber,
		CustomerID:   input.CustomerID,
		Vendor:       input.Vendor,
		Model:        input.Model,
		Status:       "active",
		Notes:        input.Notes,
	}

	if err := s.ontRepo.Create(ctx, ont); err != nil {
		return nil, err
	}
	return s.ontRepo.FindByID(ctx, ont.TenantID, ont.ID)
}

func (s *ONTService) GetByID(ctx context.Context, tenantID, ontID string) (*model.ONT, error) {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return nil, err
	}
	if ont == nil {
		return nil, ErrONTNotFound
	}
	return ont, nil
}

func (s *ONTService) Update(ctx context.Context, tenantID, ontID string, input UpdateONTInput) (*model.ONT, error) {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return nil, err
	}
	if ont == nil {
		return nil, ErrONTNotFound
	}

	// Detect customer transfer: old customer_id != new customer_id.
	oldCustomerID := ont.CustomerID
	newCustomerID := input.CustomerID
	customerChanged := !ptrEqual(oldCustomerID, newCustomerID) &&
		(oldCustomerID != nil && newCustomerID != nil)

	ont.SerialNumber = input.SerialNumber
	ont.CustomerID = input.CustomerID
	ont.Vendor = input.Vendor
	ont.Model = input.Model
	ont.Status = input.Status
	ont.Notes = input.Notes

	// When transferred to a new customer the device must be re-provisioned.
	if customerChanged {
		ont.ProvisionedAt = nil
	}

	if err := s.ontRepo.Update(ctx, ont); err != nil {
		return nil, err
	}

	// Write customer logs for the transfer (best-effort, non-fatal).
	if customerChanged && s.logRepo != nil {
		serial := ont.SerialNumber
		meta, _ := json.Marshal(map[string]interface{}{
			"ont_id":          ontID,
			"serial_number":   serial,
			"from_customer_id": *oldCustomerID,
			"to_customer_id":  *newCustomerID,
		})

		_ = s.logRepo.Create(ctx, &model.CustomerLog{
			TenantID:    tenantID,
			CustomerID:  *oldCustomerID,
			Action:      "ont_transferred_out",
			Description: fmt.Sprintf("ONT %s dipindahkan ke pelanggan lain", serial),
			Metadata:    meta,
			PerformedBy: input.PerformedBy,
		})

		_ = s.logRepo.Create(ctx, &model.CustomerLog{
			TenantID:    tenantID,
			CustomerID:  *newCustomerID,
			Action:      "ont_transferred_in",
			Description: fmt.Sprintf("ONT %s diterima dari pelanggan lain", serial),
			Metadata:    meta,
			PerformedBy: input.PerformedBy,
		})
	}

	return s.ontRepo.FindByID(ctx, tenantID, ontID)
}

// ptrEqual returns true when both pointers are nil or both point to equal strings.
func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *ONTService) Delete(ctx context.Context, tenantID, ontID string) error {
	ont, err := s.ontRepo.FindByID(ctx, tenantID, ontID)
	if err != nil {
		return err
	}
	if ont == nil {
		return ErrONTNotFound
	}
	return s.ontRepo.Delete(ctx, tenantID, ontID)
}

func (s *ONTService) List(ctx context.Context, tenantID string, filter repository.ONTFilter) ([]model.ONT, int, error) {
	return s.ontRepo.List(ctx, tenantID, filter)
}

func (s *ONTService) GetByCustomerID(ctx context.Context, tenantID, customerID string) (*model.ONT, error) {
	return s.ontRepo.FindByCustomerID(ctx, tenantID, customerID)
}
