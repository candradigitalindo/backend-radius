package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/token"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrMobileCustomerNotFound = errors.New("Pelanggan tidak ditemukan")
	ErrMobileInvalidPassword  = errors.New("Nomor pelanggan atau password salah")
	ErrMobileAccountNotActive = errors.New("Akun belum diaktivasi, silakan hubungi admin atau gunakan lupa password")
	ErrMobileTicketNotFound   = errors.New("Tiket tidak ditemukan")
)

type MobileService struct {
	customerRepo repository.CustomerRepository
	invoiceRepo  repository.InvoiceRepository
	ticketRepo   repository.TicketRepository
	packageRepo  repository.PackageRepository
	tokenMgr     *token.Manager
}

func NewMobileService(
	customerRepo repository.CustomerRepository,
	invoiceRepo repository.InvoiceRepository,
	ticketRepo repository.TicketRepository,
	packageRepo repository.PackageRepository,
	tokenMgr *token.Manager,
) *MobileService {
	return &MobileService{
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
		ticketRepo:   ticketRepo,
		packageRepo:  packageRepo,
		tokenMgr:     tokenMgr,
	}
}

// Login authenticates a customer by customer_code and password.
func (s *MobileService) Login(ctx context.Context, customerCode, password string) (*token.TokenPair, *model.Customer, error) {
	customers, err := s.customerRepo.FindByCodeGlobal(ctx, customerCode)
	if err != nil {
		return nil, nil, err
	}
	if len(customers) == 0 {
		return nil, nil, ErrMobileInvalidPassword
	}

	// Find all customers where password matches
	var matched []*model.Customer
	for i := range customers {
		c := &customers[i]
		if c.PasswordHash == "" {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password)); err == nil {
			matched = append(matched, c)
		}
	}

	if len(matched) == 0 {
		// Check if all matches have empty password_hash → not activated
		allEmpty := true
		for _, c := range customers {
			if c.PasswordHash != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			return nil, nil, ErrMobileAccountNotActive
		}
		return nil, nil, ErrMobileInvalidPassword
	}

	// If multiple tenants matched the same code+password, reject (ambiguous)
	if len(matched) > 1 {
		return nil, nil, ErrMobileInvalidPassword
	}

	c := matched[0]

	// Password matched — generate JWT
	tokens, err := s.tokenMgr.GeneratePair(token.Claims{
		UserID:   c.ID,
		TenantID: c.TenantID,
		Role:     "customer",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Clear sensitive fields before returning
	c.PasswordHash = ""
	return tokens, c, nil
}
