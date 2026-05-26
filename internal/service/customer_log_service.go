package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrCustomerLogNotFound = errors.New("Log pelanggan tidak ditemukan")
)

type CustomerLogService struct {
	logRepo repository.CustomerLogRepository
}

func NewCustomerLogService(logRepo repository.CustomerLogRepository) *CustomerLogService {
	return &CustomerLogService{logRepo: logRepo}
}

type CreateCustomerLogInput struct {
	TenantID    string
	CustomerID  string
	Action      string
	Description string
	Metadata    map[string]interface{}
	PerformedBy *string
}

func (s *CustomerLogService) Create(ctx context.Context, input CreateCustomerLogInput) (*model.CustomerLog, error) {
	var metadataJSON json.RawMessage
	if input.Metadata != nil {
		data, err := json.Marshal(input.Metadata)
		if err != nil {
			return nil, err
		}
		metadataJSON = data
	}

	log := &model.CustomerLog{
		TenantID:    input.TenantID,
		CustomerID:  input.CustomerID,
		Action:      input.Action,
		Description: input.Description,
		Metadata:    metadataJSON,
		PerformedBy: input.PerformedBy,
	}

	if err := s.logRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	return s.logRepo.FindByID(ctx, input.TenantID, log.ID)
}

func (s *CustomerLogService) GetByID(ctx context.Context, tenantID, logID string) (*model.CustomerLog, error) {
	log, err := s.logRepo.FindByID(ctx, tenantID, logID)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, ErrCustomerLogNotFound
	}
	return log, nil
}

func (s *CustomerLogService) List(ctx context.Context, tenantID string, filter repository.CustomerLogFilter) ([]model.CustomerLog, int, error) {
	return s.logRepo.List(ctx, tenantID, filter)
}

func (s *CustomerLogService) ListByCustomer(ctx context.Context, tenantID, customerID string, filter repository.CustomerLogFilter) ([]model.CustomerLog, int, error) {
	return s.logRepo.ListByCustomer(ctx, tenantID, customerID, filter)
}
