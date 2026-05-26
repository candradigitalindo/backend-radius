package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type ReportService struct {
	reportRepo repository.ReportRepository
}

func NewReportService(reportRepo repository.ReportRepository) *ReportService {
	return &ReportService{reportRepo: reportRepo}
}

func (s *ReportService) GetRevenueReport(ctx context.Context, tenantID string, year int) ([]repository.MonthlyRevenueReport, error) {
	return s.reportRepo.GetRevenueReport(ctx, tenantID, year)
}

func (s *ReportService) GetCustomerGrowth(ctx context.Context, tenantID string, year int) ([]repository.MonthlyCustomerGrowth, error) {
	return s.reportRepo.GetCustomerGrowth(ctx, tenantID, year)
}

func (s *ReportService) GetPaymentMethodBreakdown(ctx context.Context, tenantID string, month, year int) ([]repository.PaymentMethodStat, error) {
	return s.reportRepo.GetPaymentMethodBreakdown(ctx, tenantID, month, year)
}

func (s *ReportService) GetCollectionRate(ctx context.Context, tenantID string, month, year int) (*repository.CollectionRateStat, error) {
	return s.reportRepo.GetCollectionRate(ctx, tenantID, month, year)
}

func (s *ReportService) GetProfitLoss(ctx context.Context, tenantID string, month, year int) (*repository.ProfitLossStat, error) {
	return s.reportRepo.GetProfitLoss(ctx, tenantID, month, year)
}

func (s *ReportService) GetVoucherSalesReport(ctx context.Context, tenantID string, month, year int) (*repository.VoucherSalesStat, error) {
	return s.reportRepo.GetVoucherSalesReport(ctx, tenantID, month, year)
}
