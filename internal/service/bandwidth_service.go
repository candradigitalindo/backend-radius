package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type BandwidthService struct {
	bandwidthRepo repository.BandwidthRepository
}

func NewBandwidthService(bandwidthRepo repository.BandwidthRepository) *BandwidthService {
	return &BandwidthService{bandwidthRepo: bandwidthRepo}
}

func (s *BandwidthService) GetCustomerUsage(ctx context.Context, tenantID, customerID string, month, year int) (*repository.BandwidthUsage, error) {
	return s.bandwidthRepo.GetCustomerUsage(ctx, tenantID, customerID, month, year)
}

func (s *BandwidthService) GetCustomerUsageHistory(ctx context.Context, tenantID, customerID string, year int) ([]repository.MonthlyBandwidth, error) {
	return s.bandwidthRepo.GetCustomerUsageHistory(ctx, tenantID, customerID, year)
}

func (s *BandwidthService) GetTopUsers(ctx context.Context, tenantID string, month, year, limit int) ([]repository.TopBandwidthUser, error) {
	return s.bandwidthRepo.GetTopUsers(ctx, tenantID, month, year, limit)
}

func (s *BandwidthService) GetUsageSummary(ctx context.Context, tenantID string, month, year int) (*repository.BandwidthSummary, error) {
	return s.bandwidthRepo.GetUsageSummary(ctx, tenantID, month, year)
}

func (s *BandwidthService) GetSaturationReport(ctx context.Context, tenantID string, hours, thresholdPct, limit int) ([]repository.SaturationEntry, error) {
	return s.bandwidthRepo.GetSaturationReport(ctx, tenantID, hours, thresholdPct, limit)
}

func (s *BandwidthService) GetSessionTransferTotals(ctx context.Context, tenantID string, sessionIDs []string) (map[string]repository.SessionTransferTotal, error) {
	return s.bandwidthRepo.GetSessionTransferTotals(ctx, tenantID, sessionIDs)
}
