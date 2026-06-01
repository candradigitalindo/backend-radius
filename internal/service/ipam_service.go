package service

import (
	"context"
	"encoding/binary"
	"net"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type IPAMService struct {
	ipamRepo repository.IPAMRepository
}

func NewIPAMService(ipamRepo repository.IPAMRepository) *IPAMService {
	return &IPAMService{ipamRepo: ipamRepo}
}

// CreatePool creates the pool and automatically populates all usable IP addresses
// within the network range (excludes network address and broadcast).
// Gateway is excluded from the available pool but added as a reserved entry.
// Supports up to /16 (65534 hosts); larger networks are created without auto-populate.
func (s *IPAMService) CreatePool(ctx context.Context, pool *model.IPPool) error {
	if err := s.ipamRepo.CreatePool(ctx, pool); err != nil {
		return err
	}

	_, ipNet, err := net.ParseCIDR(pool.Network)
	if err != nil {
		return nil // pool created, skip auto-populate if network is invalid
	}

	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones
	if hostBits > 16 {
		return nil // too large (>65534 IPs), skip auto-populate
	}

	totalHosts := (1 << hostBits) - 2 // exclude network + broadcast
	if totalHosts <= 0 {
		return nil
	}

	netInt := binary.BigEndian.Uint32(ipNet.IP.To4())
	gateway := pool.Gateway

	addrs := make([]model.IPAddress, 0, totalHosts)
	for i := 1; i <= totalHosts; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, netInt+uint32(i))
		ipStr := ip.String()

		status := "available"
		notes := ""
		if ipStr == gateway {
			status = "reserved"
			notes = "gateway"
		}
		addrs = append(addrs, model.IPAddress{
			TenantID: pool.TenantID,
			PoolID:   pool.ID,
			Address:  ipStr,
			Status:   status,
			Notes:    notes,
		})
	}

	_ = s.ipamRepo.CreateAddressBatch(ctx, addrs)
	return nil
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

func (s *IPAMService) FindPoolByRouterID(ctx context.Context, routerID string) (*model.IPPool, error) {
	return s.ipamRepo.FindPoolByRouterID(ctx, routerID)
}
