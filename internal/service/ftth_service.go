package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type FTTHService struct {
	ftthRepo repository.FTTHRepository
}

func NewFTTHService(ftthRepo repository.FTTHRepository) *FTTHService {
	return &FTTHService{ftthRepo: ftthRepo}
}

func (s *FTTHService) GetStats(ctx context.Context, tenantID string) (*repository.FTTHStats, error) {
	return s.ftthRepo.GetStats(ctx, tenantID)
}

func (s *FTTHService) GetMapItems(ctx context.Context, tenantID string) ([]repository.FTTHMapItem, error) {
	return s.ftthRepo.GetMapItems(ctx, tenantID)
}
