package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type AdminService struct {
	adminRepo repository.AdminRepository
}

func NewAdminService(adminRepo repository.AdminRepository) *AdminService {
	return &AdminService{adminRepo: adminRepo}
}

func (s *AdminService) GetDashboardStats(ctx context.Context) (*repository.AdminDashboardStats, error) {
	return s.adminRepo.GetDashboardStats(ctx)
}

func (s *AdminService) GetTenantStats(ctx context.Context) ([]repository.TenantStat, error) {
	return s.adminRepo.GetTenantStats(ctx)
}

func (s *AdminService) GetAllRouters(ctx context.Context, page, perPage int) ([]repository.TenantRouterStat, int, error) {
	return s.adminRepo.GetAllRouters(ctx, page, perPage)
}

func (s *AdminService) GetTenantCustomerCounts(ctx context.Context) ([]repository.TenantStat, error) {
	return s.adminRepo.GetTenantCustomerCounts(ctx)
}