package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type DashboardService struct {
	dashboardRepo repository.DashboardRepository
}

func NewDashboardService(dashboardRepo repository.DashboardRepository) *DashboardService {
	return &DashboardService{dashboardRepo: dashboardRepo}
}

func (s *DashboardService) GetStats(ctx context.Context, tenantID string, month, year int) (*repository.DashboardStats, error) {
	return s.dashboardRepo.GetStats(ctx, tenantID, month, year)
}

func (s *DashboardService) GetMonthlyRevenue(ctx context.Context, tenantID string, year int) ([]repository.MonthlyRevenue, error) {
	return s.dashboardRepo.GetMonthlyRevenue(ctx, tenantID, year)
}
