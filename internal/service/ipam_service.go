package service

import (
	"context"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type IPAMService struct {
	ipamRepo repository.IPAMRepository
}

func NewIPAMService(ipamRepo repository.IPAMRepository) *IPAMService {
	return &IPAMService{ipamRepo: ipamRepo}
}

func (s *IPAMService) CreatePool(ctx context.Context, pool *model.IPPool) error {
	return s.ipamRepo.CreatePool(ctx, pool)
}

func (s *IPAMService) GetPool(ctx context.Context, tenantID, poolID string) (*model.IPPool, error) {
	return s.ipamRepo.GetPool(ctx, tenantID, poolID)
}

func (s *IPAMService) UpdatePool(ctx context.Context, pool *model.IPPool) error {
	return s.ipamRepo.UpdatePool(ctx, pool)
}

func (s *IPAMService) DeletePool(ctx context.Context, tenantID, poolID string) error {
	return s.ipamRepo.DeletePool(ctx, tenantID, poolID)
}

func (s *IPAMService) ListPools(ctx context.Context, tenantID string, page, perPage int) ([]model.IPPool, int, error) {
	return s.ipamRepo.ListPools(ctx, tenantID, page, perPage)
}

func (s *IPAMService) CreateAddress(ctx context.Context, addr *model.IPAddress) error {
	return s.ipamRepo.CreateAddress(ctx, addr)
}

func (s *IPAMService) CreateAddressBatch(ctx context.Context, addrs []model.IPAddress) error {
	return s.ipamRepo.CreateAddressBatch(ctx, addrs)
}

func (s *IPAMService) GetAddress(ctx context.Context, tenantID, addrID string) (*model.IPAddress, error) {
	return s.ipamRepo.GetAddress(ctx, tenantID, addrID)
}

func (s *IPAMService) UpdateAddress(ctx context.Context, addr *model.IPAddress) error {
	return s.ipamRepo.UpdateAddress(ctx, addr)
}

func (s *IPAMService) DeleteAddress(ctx context.Context, tenantID, addrID string) error {
	return s.ipamRepo.DeleteAddress(ctx, tenantID, addrID)
}

func (s *IPAMService) ListAddresses(ctx context.Context, tenantID, poolID string, filter repository.IPAddressFilter) ([]model.IPAddress, int, error) {
	return s.ipamRepo.ListAddresses(ctx, tenantID, poolID, filter)
}

func (s *IPAMService) AssignAddress(ctx context.Context, tenantID, addrID, customerID string) error {
	return s.ipamRepo.AssignAddress(ctx, tenantID, addrID, customerID)
}

func (s *IPAMService) ReleaseAddress(ctx context.Context, tenantID, addrID string) error {
	return s.ipamRepo.ReleaseAddress(ctx, tenantID, addrID)
}

func (s *IPAMService) FindAvailable(ctx context.Context, tenantID, poolID string) (*model.IPAddress, error) {
	return s.ipamRepo.FindAvailable(ctx, tenantID, poolID)
}

func (s *IPAMService) ReleaseAndAssign(ctx context.Context, tenantID, poolID, customerID string) (*model.IPAddress, error) {
	return s.ipamRepo.ReleaseAndAssign(ctx, tenantID, poolID, customerID)
}

func (s *IPAMService) GetPoolStats(ctx context.Context, tenantID, poolID string) (*repository.IPPoolStats, error) {
	return s.ipamRepo.GetPoolStats(ctx, tenantID, poolID)
}
