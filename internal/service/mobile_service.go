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

// GetProfile returns the customer profile.
func (s *MobileService) GetProfile(ctx context.Context, tenantID, customerID string) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrMobileCustomerNotFound
	}
	return customer, nil
}

// GetInvoices returns paginated invoices for the customer.
func (s *MobileService) GetInvoices(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Invoice, int, error) {
	return s.invoiceRepo.ListByCustomer(ctx, tenantID, customerID, page, perPage)
}

// GetPackages returns all available packages for the tenant.
func (s *MobileService) GetPackages(ctx context.Context, tenantID string) ([]model.Package, error) {
	packages, _, err := s.packageRepo.List(ctx, tenantID, repository.PackageFilter{Page: 1, PerPage: 100})
	return packages, err
}

// GetTickets lists tickets for the customer.
func (s *MobileService) GetTickets(ctx context.Context, tenantID, customerID string, page, perPage int) ([]model.Ticket, int, error) {
	return s.ticketRepo.List(ctx, tenantID, repository.TicketFilter{
		CustomerID: customerID,
		Page:       page,
		PerPage:    perPage,
	})
}

// GetTicket returns a single ticket with its messages, verifying customer ownership.
func (s *MobileService) GetTicket(ctx context.Context, tenantID, customerID, ticketID string) (*model.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil || ticket.CustomerID != customerID {
		return nil, ErrMobileTicketNotFound
	}
	msgs, err := s.ticketRepo.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	ticket.Messages = msgs
	return ticket, nil
}

// CreateTicket creates a support ticket from the mobile app.
func (s *MobileService) CreateTicket(ctx context.Context, ticket *model.Ticket) error {
	return s.ticketRepo.Create(ctx, ticket)
}

// AddTicketMessage adds a message to a ticket, verifying customer ownership.
func (s *MobileService) AddTicketMessage(ctx context.Context, tenantID, customerID string, msg *model.TicketMessage) error {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, msg.TicketID)
	if err != nil {
		return err
	}
	if ticket == nil || ticket.CustomerID != customerID {
		return ErrMobileTicketNotFound
	}
	return s.ticketRepo.AddMessage(ctx, msg)
}

// SubmitUpgradeRequest creates a ticket for package upgrade request.
func (s *MobileService) SubmitUpgradeRequest(ctx context.Context, tenantID, customerID, packageID, notes string) error {
	pkg, err := s.packageRepo.FindByID(ctx, tenantID, packageID)
	if err != nil {
		return err
	}
	if pkg == nil {
		return errors.New("Paket tidak ditemukan")
	}

	ticket := &model.Ticket{
		TenantID:    tenantID,
		CustomerID:  customerID,
		Subject:     fmt.Sprintf("Upgrade request ke paket %s", pkg.Name),
		Description: fmt.Sprintf("Pelanggan mengajukan upgrade ke paket %s (%dMbps/%dMbps). %s", pkg.Name, pkg.BandwidthDown, pkg.BandwidthUp, notes),
		Category:    "upgrade",
		Priority:    "medium",
		Status:      "open",
	}

	return s.ticketRepo.Create(ctx, ticket)
}
