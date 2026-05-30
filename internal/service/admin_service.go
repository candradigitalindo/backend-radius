package service

import (
	"context"
	"fmt"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type AdminService struct {
	adminRepo     repository.AdminRepository
	tenantService *TenantService
}

func NewAdminService(adminRepo repository.AdminRepository) *AdminService {
	return &AdminService{adminRepo: adminRepo}
}

func (s *AdminService) WithTenantService(svc *TenantService) *AdminService {
	s.tenantService = svc
	return s
}

func (s *AdminService) ResetTenantPassword(ctx context.Context, tenantID string) (string, error) {
	if s.tenantService == nil {
		return "", fmt.Errorf("tenant service not available")
	}
	return s.tenantService.ResetPassword(ctx, tenantID)
}

func (s *AdminService) GetDashboardStats(ctx context.Context, excludeTenantID string) (*repository.AdminDashboardStats, error) {
	return s.adminRepo.GetDashboardStats(ctx, excludeTenantID)
}

func (s *AdminService) GetTenantStats(ctx context.Context, excludeTenantID string) ([]repository.TenantStat, error) {
	return s.adminRepo.GetTenantStats(ctx, excludeTenantID)
}

func (s *AdminService) GetAllRouters(ctx context.Context, excludeTenantID string, page, perPage int) ([]repository.TenantRouterStat, int, error) {
	return s.adminRepo.GetAllRouters(ctx, excludeTenantID, page, perPage)
}

func (s *AdminService) GetTenantCustomerCounts(ctx context.Context, excludeTenantID string) ([]repository.TenantStat, error) {
	return s.adminRepo.GetTenantCustomerCounts(ctx, excludeTenantID)
}

func (s *AdminService) GetRollingRevenue(ctx context.Context) ([]repository.RollingMonthlyRevenue, error) {
	return s.adminRepo.GetRollingRevenue(ctx)
}

func (s *AdminService) GetSubscriptionRevenue(ctx context.Context) ([]repository.RollingMonthlyRevenue, error) {
	return s.adminRepo.GetSubscriptionRevenue(ctx)
}
