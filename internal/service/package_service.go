package service

import (
	"context"
	"errors"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrPackageNotFound = errors.New("Paket tidak ditemukan")
)

type PackageService struct {
	packageRepo repository.PackageRepository
}

func NewPackageService(packageRepo repository.PackageRepository) *PackageService {
	return &PackageService{packageRepo: packageRepo}
}

type CreatePackageInput struct {
	TenantID      string
	Name          string
	Description   string
	BandwidthUp   int
	BandwidthDown int
	Price         int64
	BurstLimit    string
	AddressList   string
}

type UpdatePackageInput struct {
	Name          string
	Description   string
	BandwidthUp   int
	BandwidthDown int
	Price         int64
	BurstLimit    string
	AddressList   string
	IsActive      bool
}

func (s *PackageService) Create(ctx context.Context, input CreatePackageInput) (*model.Package, error) {
	pkg := &model.Package{
		TenantID:      input.TenantID,
		Name:          input.Name,
		Description:   input.Description,
		BandwidthUp:   input.BandwidthUp,
		BandwidthDown: input.BandwidthDown,
		Price:         input.Price,
		BurstLimit:    input.BurstLimit,
		AddressList:   input.AddressList,
		IsActive:      true,
	}

	if err := s.packageRepo.Create(ctx, pkg); err != nil {
		return nil, err
	}

	return s.packageRepo.FindByID(ctx, pkg.TenantID, pkg.ID)
}

func (s *PackageService) GetByID(ctx context.Context, tenantID, packageID string) (*model.Package, error) {
	pkg, err := s.packageRepo.FindByID(ctx, tenantID, packageID)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, ErrPackageNotFound
	}
	return pkg, nil
}

func (s *PackageService) Update(ctx context.Context, tenantID, packageID string, input UpdatePackageInput) (*model.Package, error) {
	pkg, err := s.packageRepo.FindByID(ctx, tenantID, packageID)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, ErrPackageNotFound
	}

	pkg.Name = input.Name
	pkg.Description = input.Description
	pkg.BandwidthUp = input.BandwidthUp
	pkg.BandwidthDown = input.BandwidthDown
	pkg.Price = input.Price
	pkg.BurstLimit = input.BurstLimit
	pkg.AddressList = input.AddressList
	pkg.IsActive = input.IsActive

	if err := s.packageRepo.Update(ctx, pkg); err != nil {
		return nil, err
	}

	return s.packageRepo.FindByID(ctx, tenantID, packageID)
}

func (s *PackageService) Delete(ctx context.Context, tenantID, packageID string) error {
	pkg, err := s.packageRepo.FindByID(ctx, tenantID, packageID)
	if err != nil {
		return err
	}
	if pkg == nil {
		return ErrPackageNotFound
	}
	return s.packageRepo.Delete(ctx, tenantID, packageID)
}

func (s *PackageService) List(ctx context.Context, tenantID string, filter repository.PackageFilter) ([]model.Package, int, error) {
	return s.packageRepo.List(ctx, tenantID, filter)
}
