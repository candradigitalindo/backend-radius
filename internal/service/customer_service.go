package service

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/billing"
	"github.com/candrasyahputra/radius-server/internal/pkg/netutil"
	radiuspkg "github.com/candrasyahputra/radius-server/internal/pkg/radius"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrCustomerNotFound      = errors.New("Pelanggan tidak ditemukan")
	ErrCustomerCodeExists    = errors.New("Kode pelanggan sudah ada")
	ErrPPPoEUsernameExists   = errors.New("Username PPPoE sudah ada")
	ErrCustomerAlreadyActive = errors.New("Pelanggan sudah aktif")
	ErrCustomerNotActive     = errors.New("Pelanggan tidak aktif")
	ErrCustomerLimitReached  = errors.New("Batas jumlah pelanggan telah tercapai, upgrade paket untuk menambah lebih banyak pelanggan")
)

type customerInvoiceRepository interface {
	FindByCustomerPeriod(ctx context.Context, tenantID, customerID string, month, year int) (*model.Invoice, error)
	FindCurrentByCustomer(ctx context.Context, tenantID, customerID string) (*model.Invoice, error)
}

type CustomerService struct {
	customerRepo  repository.CustomerRepository
	sessionRepo   repository.SessionRepository
	invoiceRepo   customerInvoiceRepository
	bandwidthRepo repository.BandwidthRepository
	ontRepo       repository.ONTRepository
	genieacsSvc   *GenieACSService
	tenantRepo    repository.TenantRepository
	rewardSvc     *RewardService
	appURL        string
	cwmpURL       string // explicit ACS URL (e.g. https://app.dradius.net/cwmp); takes precedence over IP:port
	cwmpPort      string
}

func NewCustomerService(
	customerRepo repository.CustomerRepository,
	sessionRepo repository.SessionRepository,
) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		sessionRepo:  sessionRepo,
	}
}

func (s *CustomerService) WithInvoice(invoiceRepo repository.InvoiceRepository) *CustomerService {
	s.invoiceRepo = invoiceRepo
	return s
}

func (s *CustomerService) WithBandwidth(bandwidthRepo repository.BandwidthRepository) *CustomerService {
	s.bandwidthRepo = bandwidthRepo
	return s
}

func (s *CustomerService) WithGenieACS(ontRepo repository.ONTRepository, genieacsSvc *GenieACSService) *CustomerService {
	s.ontRepo = ontRepo
	s.genieacsSvc = genieacsSvc
	return s
}

func (s *CustomerService) WithTenant(tenantRepo repository.TenantRepository, appURL string) *CustomerService {
	s.tenantRepo = tenantRepo
	s.appURL = appURL
	return s
}

func (s *CustomerService) WithReward(rewardSvc *RewardService) *CustomerService {
	s.rewardSvc = rewardSvc
	return s
}

func (s *CustomerService) WithCWMPPort(port string) *CustomerService {
	s.cwmpPort = port
	return s
}

func (s *CustomerService) WithCWMPURL(url string) *CustomerService {
	s.cwmpURL = url
	return s
}

type CreateCustomerInput struct {
	TenantID         string
	Name             string
	NIK              string
	Phone            string
	Email            string
	Address          string
	Latitude         *float64
	Longitude        *float64
	ConnectionType   string
	IPAddress        string
	PackageID        *string
	RouterID         *string
	ODPPortID        *string
	JoinDate         string
	BillingType      string
	BillingProfileID *string
	CustomPrice      *int64
	Discount         int64
	AdditionalFee    int64
	FeeDescription   string
	Notes            string
	SerialNumber     string
	ONTVendor        *string
	ONTModel         *string
	ReferralCodeUsed string
	ResellerID       *string
}

type UpdateCustomerProfileInput struct {
	Name       string
	NIK        string
	Phone      string
	Email      string
	Address    string
	Latitude   *float64
	Longitude  *float64
	Notes      string
	ResellerID *string
}

type UpdateCustomerAccessInput struct {
	ConnectionType string
	PPPoEUsername  string
	PPPoEPassword  string
	IPAddress      string
	RouterID       *string
	ODPPortID      *string
}

type UpdateCustomerServiceInput struct {
	PackageID        *string
	BillingProfileID *string // nil = tidak diubah, "" = hapus profil
	JoinDate         *time.Time
	InvoiceDate      *time.Time
	BillingDueDate   *time.Time
	BillingType      string
	CustomPrice      *int64
	Discount         *int64
	AdditionalFee    *int64
	FeeDescription   string
}

func (s *CustomerService) Create(ctx context.Context, input CreateCustomerInput) (*model.Customer, error) {
	if input.ConnectionType == "" {
		input.ConnectionType = "pppoe"
	}
	if input.BillingType != "" {
		input.BillingType = normalizeBillingType(input.BillingType)
	}
	if input.BillingType == "" {
		// Use tenant's default billing type if available
		if s.tenantRepo != nil {
			if t, err := s.tenantRepo.FindByID(ctx, input.TenantID); err == nil && t != nil && t.DefaultBillingType != "" {
				input.BillingType = t.DefaultBillingType
			}
		}
		if input.BillingType == "" {
			input.BillingType = "fixed"
		}
	}

	// Enforce max_customers limit from tenant subscription plan.
	// max_customers == 0 means unlimited.
	if s.tenantRepo != nil {
		if tenant, err := s.tenantRepo.FindByID(ctx, input.TenantID); err == nil && tenant != nil && tenant.MaxCustomers > 0 {
			current, err := s.customerRepo.CountActive(ctx, input.TenantID)
			if err == nil && current >= tenant.MaxCustomers {
				return nil, ErrCustomerLimitReached
			}
		}
	}

	joinDate := parseJoinDate(input.JoinDate)
	billingDeadline := billing.NormalizeDay(joinDate.Day())
	billingDate := billingDateFromDeadline(billingDeadline)

	// For cycle billing, override billing dates with tenant-level settings.
	if input.BillingType == "cycle" && s.tenantRepo != nil {
		if t, err := s.tenantRepo.FindByID(ctx, input.TenantID); err == nil && t != nil {
			if t.DueDay > 0 {
				billingDeadline = billing.NormalizeDay(t.DueDay)
			}
			if t.BillingCycle > 0 {
				billingDate = billing.NormalizeDay(t.BillingCycle)
			} else {
				billingDate = billing.InvoiceDayFromDueDay(billingDeadline)
			}
		}
	}

	code, err := s.NextCode(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}

	pppoeUsername, err := s.generatePPPoEUsername(ctx, input.TenantID, input.Phone)
	if err != nil {
		return nil, err
	}
	pppoePassword := generatePPPoEPassword()

	customer := &model.Customer{
		TenantID:         input.TenantID,
		CustomerCode:     code,
		Name:             input.Name,
		NIK:              input.NIK,
		Phone:            input.Phone,
		Email:            input.Email,
		Address:          input.Address,
		Latitude:         input.Latitude,
		Longitude:        input.Longitude,
		ConnectionType:   input.ConnectionType,
		PPPoEUsername:    pppoeUsername,
		PPPoEPassword:    pppoePassword,
		IPAddress:        input.IPAddress,
		PackageID:        input.PackageID,
		RouterID:         input.RouterID,
		ODPPortID:        input.ODPPortID,
		ResellerID:       input.ResellerID,
		JoinDate:         joinDate,
		BillingDate:      billingDate,
		BillingType:      input.BillingType,
		BillingDeadline:  billingDeadline,
		BillingProfileID: input.BillingProfileID,
		CustomPrice:      input.CustomPrice,
		Discount:         input.Discount,
		AdditionalFee:    input.AdditionalFee,
		FeeDescription:   input.FeeDescription,
		Status:           "active",
		Notes:            input.Notes,
	}

	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			// re-generate code and username on collision retry
			customer.CustomerCode, err = s.NextCode(ctx, input.TenantID)
			if err != nil {
				return nil, err
			}
			customer.PPPoEUsername, err = s.generatePPPoEUsername(ctx, input.TenantID, input.Phone)
			if err != nil {
				return nil, err
			}
		}
		err = s.customerRepo.Create(ctx, customer)
		if err == nil {
			break
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "pppoe_username") {
				return nil, ErrPPPoEUsernameExists
			}
			// customer_code collision: retry
			continue
		}
		return nil, err
	}
	if err != nil {
		return nil, ErrCustomerCodeExists
	}

	if input.ConnectionType == "ftth" && input.SerialNumber != "" && s.ontRepo != nil {
		ont := &model.ONT{
			TenantID:     input.TenantID,
			CustomerID:   &customer.ID,
			SerialNumber: input.SerialNumber,
			Vendor:       input.ONTVendor,
			Model:        input.ONTModel,
			Status:       "active",
		}
		if err := s.ontRepo.Create(ctx, ont); err != nil {
			log.Printf("[CustomerService] gagal membuat ONT untuk pelanggan %s: %v", customer.ID, err)
		} else if s.genieacsSvc != nil {
			ontID := ont.ID
			tenantID := input.TenantID
			go func() {
				pCtx := context.Background()
				if err := s.genieacsSvc.ProvisionONT(pCtx, tenantID, ontID); err != nil {
					log.Printf("[CustomerService] GenieACS provision gagal untuk ONT %s: %v", ontID, err)
				}
			}()
		}
	}

	// Auto-create referral record if a referral code was provided
	if input.ReferralCodeUsed != "" && s.rewardSvc != nil {
		go s.processReferralRegistration(context.Background(), input.TenantID, customer.ID, input.ReferralCodeUsed)
	}

	return s.customerRepo.FindByID(ctx, customer.TenantID, customer.ID)
}

func (s *CustomerService) processReferralRegistration(ctx context.Context, tenantID, newCustomerID, referralCode string) {
	referrer, err := s.customerRepo.FindByReferralCode(ctx, tenantID, referralCode)
	if err != nil || referrer == nil {
		return
	}
	// Find the active referral reward
	reward, err := s.rewardSvc.FindActiveRewardByType(ctx, tenantID, "referral")
	if err != nil || reward == nil {
		return
	}
	ref := &model.Referral{
		TenantID:     tenantID,
		ReferrerID:   referrer.ID,
		ReferredID:   newCustomerID,
		RewardID:     reward.ID,
		ReferralCode: referralCode,
		RewardAmount: reward.Value,
		Status:       "pending",
	}
	if err := s.rewardSvc.CreateReferral(ctx, ref); err != nil {
		log.Printf("[CustomerService] gagal membuat referral untuk kode %s: %v", referralCode, err)
	}
}

func (s *CustomerService) GetByID(ctx context.Context, tenantID, customerID string) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}
	if s.ontRepo != nil {
		ont, err := s.ontRepo.FindByCustomerID(ctx, tenantID, customerID)
		if err == nil && ont != nil {
			customer.ONT = ont
			s.attachONTStatusDetail(ctx, tenantID, customerID, customer.ONT)
		}
	}
	if s.invoiceRepo != nil {
		if inv, err := s.loadCurrentInvoice(ctx, tenantID, customer, time.Now()); err == nil && inv != nil {
			customer.CurrentInvoice = inv
		}
	}
	if s.sessionRepo != nil && customer.PPPoEUsername != "" {
		if sess, err := s.sessionRepo.FindActiveByUsername(ctx, tenantID, customer.PPPoEUsername); err == nil && sess != nil {
			if s.bandwidthRepo != nil && sess.SessionID != "" {
				if sample, sampleErr := s.bandwidthRepo.GetLatestSampleBySession(ctx, tenantID, sess.SessionID); sampleErr == nil && sample != nil {
					sess.RealtimeDownloadBps = sample.DownloadBps
					sess.RealtimeUploadBps = sample.UploadBps
					sess.RealtimeSampledAt = &sample.SampledAt
				}
			}
			customer.ActiveSession = sess
		}
	}
	if s.cwmpURL != "" {
		u := s.cwmpURL
		customer.ACSURL = &u
	} else if s.cwmpPort != "" {
		publicIP := netutil.GetPublicIP()
		if publicIP != "" {
			acsURL := "http://" + publicIP + ":" + s.cwmpPort
			customer.ACSURL = &acsURL
		}
	}
	return customer, nil
}

func (s *CustomerService) loadCurrentInvoice(ctx context.Context, tenantID string, customer *model.Customer, now time.Time) (*model.Invoice, error) {
	if s.invoiceRepo == nil {
		return nil, nil
	}
	if customer == nil {
		return nil, nil
	}

	invoiceDay, deadlineDay := s.resolveBillingDays(ctx, tenantID, customer)
	targetMonth, targetYear := billing.CurrentDuePeriod(now, invoiceDay, deadlineDay)
	currentPeriodInvoice, err := s.invoiceRepo.FindByCustomerPeriod(ctx, tenantID, customer.ID, targetMonth, targetYear)
	if err != nil {
		return nil, err
	}
	if currentPeriodInvoice != nil {
		return currentPeriodInvoice, nil
	}

	return s.invoiceRepo.FindCurrentByCustomer(ctx, tenantID, customer.ID)
}

// resolveBillingDays returns (invoiceDay, deadlineDay) for a customer.
// For "cycle" type the days come from tenant-level settings; for "fixed" from the stored per-customer values.
func (s *CustomerService) resolveBillingDays(ctx context.Context, tenantID string, customer *model.Customer) (invoiceDay, deadlineDay int) {
	if billing.NormalizeBillingType(customer.BillingType) == "cycle" && s.tenantRepo != nil {
		if t, err := s.tenantRepo.FindByID(ctx, tenantID); err == nil && t != nil {
			deadlineDay = billing.NormalizeDay(t.DueDay)
			invoiceDay = billing.NormalizeDay(t.BillingCycle)
			if deadlineDay == 0 {
				deadlineDay = billing.NormalizeDay(customer.JoinDate.Day())
			}
			if invoiceDay == 0 {
				invoiceDay = billing.InvoiceDayFromDueDay(deadlineDay)
			}
			return
		}
	}
	deadlineDay = normalizedBillingDeadlineDay(customer)
	invoiceDay = customer.BillingDate
	if invoiceDay <= 0 {
		invoiceDay = billing.InvoiceDayFromDueDay(deadlineDay)
	}
	return
}

func normalizedBillingDeadlineDay(customer *model.Customer) int {
	if customer == nil {
		return 1
	}

	deadlineDay := customer.BillingDeadline
	if deadlineDay <= 0 && !customer.JoinDate.IsZero() {
		deadlineDay = customer.JoinDate.Day()
	}
	return billing.NormalizeDay(deadlineDay)
}

func (s *CustomerService) attachONTStatusDetail(ctx context.Context, tenantID, customerID string, ont *model.ONT) {
	if ont == nil || ont.SerialNumber == "" || s.genieacsSvc == nil {
		return
	}

	ont.Actions = &model.ONTAvailableActions{
		CanRestart:    true,
		CanManageWiFi: true,
	}

	statusCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	status, err := s.genieacsSvc.GetDeviceStatus(statusCtx, tenantID, ont.SerialNumber)
	if err != nil {
		if errors.Is(err, ErrONTNotFound) || errors.Is(err, ErrGenieACSDeviceNotFound) {
			return
		}
		log.Printf("[CustomerService] gagal memuat status ONT %s untuk customer %s: %v", ont.SerialNumber, customerID, err)
		return
	}

	ont.StatusDetail = mapDeviceInfoToONTStatusDetail(status)

	// Sinkronisasi last_online_at dari GenieACS _lastInform jika belum ada atau lebih lama
	if status.LastInform != "" {
		if t, errP := time.Parse(time.RFC3339Nano, status.LastInform); errP == nil {
			if ont.LastOnlineAt == nil || t.After(*ont.LastOnlineAt) {
				ont.LastOnlineAt = &t
				if s.ontRepo != nil {
					saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer saveCancel()
					_ = s.ontRepo.Update(saveCtx, ont)
				}
			}
		}
	}
}

func mapDeviceInfoToONTStatusDetail(info *DeviceInfo) *model.ONTStatusDetail {
	if info == nil {
		return nil
	}

	connectedHosts := make([]model.ONTConnectedHost, 0, len(info.ConnectedHosts))
	for _, host := range info.ConnectedHosts {
		connectedHosts = append(connectedHosts, model.ONTConnectedHost{
			IPAddress:     host.IPAddress,
			MACAddress:    host.MACAddress,
			HostName:      host.HostName,
			InterfaceType: host.InterfaceType,
			Active:        host.Active,
		})
	}

	return &model.ONTStatusDetail{
		DeviceID:               info.DeviceID,
		SerialNumber:           info.SerialNumber,
		Manufacturer:           info.Manufacturer,
		ProductClass:           info.ProductClass,
		SoftwareVersion:        info.SoftwareVersion,
		Uptime:                 info.Uptime,
		RxPower:                info.RxPower,
		TxPower:                info.TxPower,
		WiFiSSID:               info.WiFiSSID,
		WiFiPassword:           info.WiFiPassword,
		WiFiPasswordReported:   info.WiFiPasswordReported,
		WiFiPasswordSource:     info.WiFiPasswordSource,
		WiFiSecurity:           info.WiFiSecurity,
		WiFiSecurityModes:      info.WiFiSecurityModes,
		WANIP:                  info.WANIP,
		PPPoEUsername:          info.PPPoEUsername,
		LastInform:             info.LastInform,
		DataRealtime:           info.DataRealtime,
		ConnectedHostCount:     info.ConnectedHostCount,
		ConnectedHostsComplete: info.ConnectedHostsComplete,
		ConnectedHostsSource:   info.ConnectedHostsSource,
		ConnectedHosts:         connectedHosts,
	}
}

func (s *CustomerService) GetByCode(ctx context.Context, tenantID, customerCode string) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByCode(ctx, tenantID, customerCode)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

func (s *CustomerService) UpdateProfile(ctx context.Context, tenantID, customerID string, input UpdateCustomerProfileInput) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	if input.Name != "" {
		customer.Name = input.Name
	}
	if input.NIK != "" {
		customer.NIK = input.NIK
	}
	if input.Phone != "" {
		customer.Phone = input.Phone
	}
	if input.Email != "" {
		customer.Email = input.Email
	}
	if input.Address != "" {
		customer.Address = input.Address
	}
	if input.Latitude != nil {
		customer.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		customer.Longitude = input.Longitude
	}
	if input.Notes != "" {
		customer.Notes = input.Notes
	}
	// Reseller assignment: "" clears, a value sets, nil leaves unchanged.
	if input.ResellerID != nil {
		if *input.ResellerID == "" {
			customer.ResellerID = nil
		} else {
			customer.ResellerID = input.ResellerID
		}
	}

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}
	return s.customerRepo.FindByID(ctx, tenantID, customerID)
}

func (s *CustomerService) UpdateAccess(ctx context.Context, tenantID, customerID string, input UpdateCustomerAccessInput) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	oldRouterID := stringPtrValue(customer.RouterID)
	newRouterID := oldRouterID
	if input.RouterID != nil {
		newRouterID = stringPtrValue(input.RouterID)
	}
	credentialsChanged := false
	if input.PPPoEUsername != "" && input.PPPoEUsername != customer.PPPoEUsername {
		credentialsChanged = true
	}
	if input.PPPoEPassword != "" && input.PPPoEPassword != customer.PPPoEPassword {
		credentialsChanged = true
	}
	if oldRouterID != newRouterID || credentialsChanged {
		s.disconnectCustomer(ctx, customer)
	}

	if input.ConnectionType != "" {
		customer.ConnectionType = input.ConnectionType
	}
	if input.PPPoEUsername != "" {
		customer.PPPoEUsername = input.PPPoEUsername
	}
	if input.PPPoEPassword != "" {
		customer.PPPoEPassword = input.PPPoEPassword
	}
	if input.IPAddress != "" {
		customer.IPAddress = input.IPAddress
	}
	if input.RouterID != nil {
		customer.RouterID = input.RouterID
	}
	if input.ODPPortID != nil {
		customer.ODPPortID = input.ODPPortID
	}

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "pppoe_username") {
			return nil, ErrPPPoEUsernameExists
		}
		return nil, err
	}
	return s.customerRepo.FindByID(ctx, tenantID, customerID)
}

func (s *CustomerService) UpdateServicePackage(ctx context.Context, tenantID, customerID string, input UpdateCustomerServiceInput) (*model.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	oldPkgID := stringPtrValue(customer.PackageID)
	newPkgID := oldPkgID
	if input.PackageID != nil {
		newPkgID = stringPtrValue(input.PackageID)
	}
	if oldPkgID != newPkgID {
		s.disconnectCustomer(ctx, customer)
	}

	if input.PackageID != nil {
		customer.PackageID = input.PackageID
	}
	applyUpdatedCustomerBillingSchedule(customer, input)
	if input.BillingType != "" {
		customer.BillingType = normalizeBillingType(input.BillingType)
	}
	// For cycle billing, billing dates must always reflect the current tenant settings.
	if billing.NormalizeBillingType(customer.BillingType) == "cycle" && s.tenantRepo != nil {
		if t, err := s.tenantRepo.FindByID(ctx, tenantID); err == nil && t != nil {
			if t.DueDay > 0 {
				customer.BillingDeadline = billing.NormalizeDay(t.DueDay)
			}
			if t.BillingCycle > 0 {
				customer.BillingDate = billing.NormalizeDay(t.BillingCycle)
			} else {
				customer.BillingDate = billing.InvoiceDayFromDueDay(customer.BillingDeadline)
			}
		}
	}
	if input.CustomPrice != nil {
		customer.CustomPrice = input.CustomPrice
	}
	if input.Discount != nil {
		customer.Discount = *input.Discount
	}
	if input.AdditionalFee != nil {
		customer.AdditionalFee = *input.AdditionalFee
	}
	if input.FeeDescription != "" {
		customer.FeeDescription = input.FeeDescription
	}
	// BillingProfileID: diset langsung di sini HANYA jika bukan dari ChangeBillingProfile
	// (ChangeBillingProfile menangani pending logic sendiri dan tidak masuk ke sini)

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}
	return s.customerRepo.FindByID(ctx, tenantID, customerID)
}

func applyUpdatedCustomerBillingSchedule(customer *model.Customer, input UpdateCustomerServiceInput) {
	if customer == nil {
		return
	}

	if input.JoinDate != nil {
		customer.JoinDate = *input.JoinDate
	}

	if input.InvoiceDate == nil && input.BillingDueDate == nil {
		if input.JoinDate != nil {
			deadlineDay := billing.NormalizeDay(customer.JoinDate.Day())
			customer.BillingDeadline = deadlineDay
			customer.BillingDate = billingDateFromDeadline(customer.BillingDeadline)
		}
		return
	}

	deadlineDay := normalizedBillingDeadlineDay(customer)
	invoiceDay := customer.BillingDate
	if invoiceDay <= 0 {
		invoiceDay = billingDateFromDeadline(deadlineDay)
	}

	if input.InvoiceDate != nil {
		invoiceDay = billing.NormalizeDay(input.InvoiceDate.Day())
	}
	if input.BillingDueDate != nil {
		deadlineDay = billing.NormalizeDay(input.BillingDueDate.Day())
	}

	customer.BillingDate = invoiceDay
	customer.BillingDeadline = deadlineDay
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *CustomerService) Delete(ctx context.Context, tenantID, customerID string) error {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	return s.customerRepo.Delete(ctx, tenantID, customerID)
}

func (s *CustomerService) List(ctx context.Context, tenantID string, filter repository.CustomerFilter) ([]model.Customer, int, error) {
	return s.customerRepo.List(ctx, tenantID, filter)
}

func billingDateFromDeadline(deadline int) int {
	return billing.InvoiceDayFromDueDay(deadline)
}

// normalizeBillingType delegates to the shared billing package helper.
func normalizeBillingType(t string) string {
	return billing.NormalizeBillingType(t)
}

func parseJoinDate(s string) time.Time {
	if s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			return parsed
		}
	}
	return time.Now()
}

func (s *CustomerService) NextCode(ctx context.Context, tenantID string) (string, error) {
	prefix := time.Now().Format("060102")
	count, err := s.customerRepo.CountByCodePrefix(ctx, tenantID, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}

func (s *CustomerService) generatePPPoEUsername(ctx context.Context, tenantID, phone string) (string, error) {
	digits := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits += string(ch)
		}
	}
	last4 := ""
	if len(digits) >= 4 {
		last4 = digits[len(digits)-4:]
	} else if len(digits) > 0 {
		for len(digits) < 4 {
			digits = "0" + digits
		}
		last4 = digits
	} else {
		// no phone: use 4 random digits
		var buf [2]byte
		_, _ = crand.Read(buf[:])
		n := int(buf[0])<<8 | int(buf[1])
		last4 = fmt.Sprintf("%04d", n%10000)
	}
	prefix := time.Now().Format("060102") + last4
	count, err := s.customerRepo.CountByPPPoEPrefix(ctx, tenantID, prefix)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return prefix, nil
	}
	suffix := string(rune('a' + count - 1))
	if count > 26 {
		suffix = fmt.Sprintf("%d", count)
	}
	return prefix + suffix, nil
}

func generatePPPoEPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	for i, v := range b {
		b[i] = chars[int(v)%len(chars)]
	}
	return string(b)
}

func (s *CustomerService) Isolate(ctx context.Context, tenantID, customerID string) error {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	if customer.Status != "active" {
		return ErrCustomerNotActive
	}
	s.disconnectCustomer(ctx, customer)
	now := time.Now()
	return s.customerRepo.SetIsolated(ctx, tenantID, customerID, &now)
}

func (s *CustomerService) Activate(ctx context.Context, tenantID, customerID string) error {
	customer, err := s.customerRepo.FindByID(ctx, tenantID, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	if customer.Status == "active" {
		return ErrCustomerAlreadyActive
	}
	return s.customerRepo.SetIsolated(ctx, tenantID, customerID, nil)
}

func (s *CustomerService) disconnectCustomer(ctx context.Context, customer *model.Customer) {
	if customer.Router == nil || customer.PPPoEUsername == "" {
		return
	}
	session, err := s.sessionRepo.FindActiveByUsername(ctx, customer.TenantID, customer.PPPoEUsername)
	if err != nil || session == nil {
		return
	}
	// Reach the router on its VPN IP (WireGuard mode) or, for Direct/IP Publik
	// routers, on the live session's NAS IP (the WAN IP that authenticated).
	target := customer.Router.CoAAddress()
	if target == "" {
		target = session.NASIPAddress
	}
	_ = radiuspkg.DisconnectUser(target, customer.Router.CoAPort, customer.Router.RADIUSSecret, customer.PPPoEUsername, session.SessionID)
}
