package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/payment"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrTenantNotFound    = errors.New("Tenant tidak ditemukan")
	ErrSlugAlreadyUsed   = errors.New("Slug sudah digunakan")
	ErrPhoneAlreadyUsed  = errors.New("Nomor telepon sudah terdaftar, gunakan nomor lain atau login")
	ErrEmailAlreadyUsed  = errors.New("Email sudah terdaftar, silakan login atau gunakan email lain")
)

type TenantService struct {
	tenantRepo   repository.TenantRepository
	userRepo     repository.UserRepository
	roleRepo     repository.RoleRepository
	reminderRepo repository.ReminderRepository
	waClient     *whatsapp.Client
	baseURL      string
}

func NewTenantService(tenantRepo repository.TenantRepository, userRepo repository.UserRepository, roleRepo repository.RoleRepository) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		roleRepo:   roleRepo,
	}
}

func (s *TenantService) WithReminderRepo(reminderRepo repository.ReminderRepository) *TenantService {
	s.reminderRepo = reminderRepo
	return s
}

func (s *TenantService) WithWAClient(waClient *whatsapp.Client) *TenantService {
	s.waClient = waClient
	return s
}

func (s *TenantService) WithBaseURL(url string) *TenantService {
	s.baseURL = url
	return s
}

type CreateTenantInput struct {
	Name                string
	Slug                string
	Email               string
	Phone               string
	Address             string
	Timezone            string
	Currency            string
	BillingCycle        int
	DueDay              int
	IsolirDay           int
	GracePeriod         int
	DefaultBillingType  string
	Plan                string
	MaxCustomers        int
	Fingerprint         string
	RegistrationIP      string
}

type UpdateTenantInput struct {
	Name                  string
	Email                 string
	Phone                 string
	Address               string
	LogoURL               string
	Timezone              string
	Currency              string
	BillingCycle          int
	DueDay                int
	IsolirDay             int
	GracePeriod           int
	DefaultBillingType    string
	DefaultPaymentTiming  string
	IsActive              bool
}

type AdminUpdateTenantInput struct {
	Name          string
	Email         string
	Phone         string
	Address       string
	IsActive      bool
	Status        string
	Plan          string
	PlanExpiresAt *time.Time
	MaxCustomers  int
}

type UpdateSettingsInput struct {
	WAAPIKey     string
	WASender     string
	PGProvider   string
	PGAPIKey     string
	PGSecretKey  string
	PGMerchantID string
	PGSandbox    bool
}

func (s *TenantService) generateRandomPassword() (string, error) {
	pass := ""
	for i := 0; i < 8; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		pass += fmt.Sprintf("%d", n.Int64())
	}
	return pass, nil
}

func (s *TenantService) sendWelcomeMessage(ctx context.Context, tenant *model.Tenant, password string) {
	if s.waClient == nil || tenant.Phone == "" {
		return
	}

	url := s.baseURL
	if url == "" {
		url = "http://10.10.1.2" // Fallback
	}

	msg := ""
	if s.reminderRepo != nil && s.tenantRepo != nil {
		saT, err := s.tenantRepo.FindBySlug(ctx, "superadmin")
		if err == nil && saT != nil {
			rem, err := s.reminderRepo.FindActiveByType(ctx, saT.ID, "welcome_tenant")
			if err == nil && rem != nil {
				msg = rem.MessageTemplate
				msg = strings.ReplaceAll(msg, "{nama_isp}", tenant.Name)
				msg = strings.ReplaceAll(msg, "{panel_url}", url)
				msg = strings.ReplaceAll(msg, "{email}", tenant.Email)
				msg = strings.ReplaceAll(msg, "{password}", password)
			}
		}
	}

	if msg == "" {
		msg = fmt.Sprintf("🚀 *Selamat Datang di D Radius!*\n\n"+
			"Halo *%s*,\n\n"+
			"Pendaftaran ISP Anda telah berhasil! Berikut adalah detail akun panel manajemen Anda:\n\n"+
			"🌐 *Panel URL:* %s\n"+
			"📧 *Email:* %s\n"+
			"🔑 *Password:* `%s` (8 digit angka)\n\n"+
			"⚠️ *Penting:*\n"+
			"Segera ganti password Anda setelah login pertama kali demi keamanan akun.\n\n"+
			"Terima kasih telah bergabung dengan D Radius!\n"+
			"_Tim Support D Radius_",
			tenant.Name, url, tenant.Email, password)
	}

	_, _ = s.waClient.SendMessage(ctx, "superadmin", tenant.Phone, msg)
}

func (s *TenantService) ResetPassword(ctx context.Context, tenantID string) (string, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if tenant == nil {
		return "", ErrTenantNotFound
	}

	// Generate new random password
	password, err := s.generateRandomPassword()
	if err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Update password for all users with role 'owner' in this tenant
	users, err := s.userRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return "", err
	}

	found := false
	for _, u := range users {
		if u.Role == "owner" {
			u.PasswordHash = string(hash)
			if err := s.userRepo.Update(ctx, u); err != nil {
				return "", err
			}
			found = true
		}
	}

	if !found {
		// If no owner found, create one
		user := &model.User{
			TenantID:     tenant.ID,
			Name:         tenant.Name,
			Email:        tenant.Email,
			PasswordHash: string(hash),
			Role:         "owner",
			Phone:        tenant.Phone,
			IsActive:     true,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", err
		}
	}

	// Send notification via WhatsApp
	msg := ""
	if s.reminderRepo != nil && s.tenantRepo != nil {
		saT, err := s.tenantRepo.FindBySlug(ctx, "superadmin")
		if err == nil && saT != nil {
			rem, err := s.reminderRepo.FindActiveByType(ctx, saT.ID, "reset_password_new")
			if err == nil && rem != nil {
				msg = rem.MessageTemplate
				msg = strings.ReplaceAll(msg, "{nama_isp}", tenant.Name)
				msg = strings.ReplaceAll(msg, "{email}", tenant.Email)
				msg = strings.ReplaceAll(msg, "{password}", password)
			}
		}
	}

	if msg == "" {
		msg = fmt.Sprintf("🔐 *Password Baru D Radius*\n\n"+
			"Halo *%s*,\n\n"+
			"Password akun panel manajemen Anda telah direset oleh SuperAdmin.\n\n"+
			"📧 *Email:* %s\n"+
			"🔑 *Password Baru:* `%s` (8 digit angka)\n\n"+
			"Silakan gunakan password baru ini untuk login. Segera ganti password Anda demi keamanan.\n\n"+
			"Terima kasih,\n"+
			"_Tim Support D Radius_",
			tenant.Name, tenant.Email, password)
	}

	_, _ = s.waClient.SendMessage(ctx, "superadmin", tenant.Phone, msg)

	return password, nil
}

func (s *TenantService) Approve(ctx context.Context, tenantID string) error {
	return s.tenantRepo.Approve(ctx, tenantID)
}

func (s *TenantService) Create(ctx context.Context, input CreateTenantInput) (*model.Tenant, error) {
	existing, err := s.tenantRepo.FindBySlug(ctx, input.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSlugAlreadyUsed
	}

	if input.Timezone == "" {
		input.Timezone = "Asia/Jakarta"
	}
	if input.Currency == "" {
		input.Currency = "IDR"
	}
	if input.BillingCycle == 0 {
		input.BillingCycle = 1
	}
	if input.DueDay == 0 {
		input.DueDay = 20
	}
	if input.IsolirDay == 0 {
		input.IsolirDay = 21
	}
	if input.GracePeriod == 0 {
		input.GracePeriod = 3
	}
	if input.MaxCustomers == 0 {
		input.MaxCustomers = 100
	}
	if input.DefaultBillingType == "" {
		input.DefaultBillingType = "fixed"
	}

	var expiry *time.Time
	if input.Plan == "trial" {
		t := time.Now().AddDate(0, 0, 30)
		expiry = &t
	} else if input.Plan != "" {
		if strings.Contains(strings.ToLower(input.Plan), "bulan") || strings.Contains(strings.ToLower(input.Plan), "monthly") {
			t := time.Now().AddDate(0, 1, 0)
			expiry = &t
		} else if strings.Contains(strings.ToLower(input.Plan), "tahun") || strings.Contains(strings.ToLower(input.Plan), "yearly") {
			t := time.Now().AddDate(1, 0, 0)
			expiry = &t
		}
	}

	status := "active"

	// ─── Anti-Cheating Logic ───
	riskScore := 0
	var riskReasons []string

	// Check duplicate fingerprint
	if input.Fingerprint != "" {
		count, _ := s.tenantRepo.CountByFingerprint(ctx, input.Fingerprint)
		if count > 0 {
			riskScore += 50
			riskReasons = append(riskReasons, fmt.Sprintf("Duplicate device fingerprint (%d existing)", count))
		}
	}

	// Check duplicate IP
	if input.RegistrationIP != "" {
		count, _ := s.tenantRepo.CountByIP(ctx, input.RegistrationIP)
		if count > 0 {
			riskScore += 30
			riskReasons = append(riskReasons, fmt.Sprintf("Duplicate registration IP (%d existing)", count))
		}
	}

	isActive := true
	if riskScore >= 40 {
		status = "pending"
		isActive = false
	}

	tenant := &model.Tenant{
		Name:               input.Name,
		Slug:               input.Slug,
		Email:              input.Email,
		Phone:              input.Phone,
		Address:            input.Address,
		Timezone:           input.Timezone,
		Currency:           input.Currency,
		BillingCycle:       input.BillingCycle,
		DueDay:             input.DueDay,
		IsolirDay:          input.IsolirDay,
		GracePeriod:        input.GracePeriod,
		DefaultBillingType: input.DefaultBillingType,
		Plan:               input.Plan,
		PlanExpiresAt:      expiry,
		MaxCustomers:       input.MaxCustomers,
		IsActive:           isActive,
		Status:             status,
		Fingerprint:        input.Fingerprint,
		RegistrationIP:     input.RegistrationIP,
		RiskScore:          riskScore,
		RiskReasons:        strings.Join(riskReasons, ", "),
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch {
			case strings.Contains(pgErr.ConstraintName, "phone"):
				return nil, ErrPhoneAlreadyUsed
			case strings.Contains(pgErr.ConstraintName, "email"):
				return nil, ErrEmailAlreadyUsed
			case strings.Contains(pgErr.ConstraintName, "slug"):
				return nil, ErrSlugAlreadyUsed
			default:
				return nil, errors.New("data yang Anda masukkan sudah terdaftar di sistem")
			}
		}
		return nil, err
	}

	// Create initial owner user for the tenant
	password, _ := s.generateRandomPassword()
	if password == "" {
		password = "password123" // Fallback
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err == nil {
		user := &model.User{
			TenantID:     tenant.ID,
			Name:         tenant.Name,
			Email:        tenant.Email,
			PasswordHash: string(hash),
			Role:         "owner",
			Phone:        tenant.Phone,
			IsActive:     true,
		}
		if err := s.userRepo.Create(ctx, user); err == nil {
			// Send welcome message via WhatsApp
			go s.sendWelcomeMessage(context.Background(), tenant, password)
		}
	}

	// Seed default roles
	_ = s.seedDefaultRoles(ctx, tenant.ID)

	return s.tenantRepo.FindByID(ctx, tenant.ID)
}

func (s *TenantService) seedDefaultRoles(ctx context.Context, tenantID string) error {
	defaults := []model.Role{
		{TenantID: tenantID, Name: "Owner", Slug: "owner", Description: "Pemilik tenant dengan akses penuh", IsSystem: true, Permissions: model.AllPermissionKeys()},
		{TenantID: tenantID, Name: "Admin", Slug: "admin", Description: "Administrator dengan akses luas", IsSystem: true, Permissions: model.DefaultAdminPermissions()},
		{TenantID: tenantID, Name: "Teknisi", Slug: "technician", Description: "Teknisi lapangan", IsSystem: true, Permissions: model.DefaultTechnicianPermissions()},
	}
	for i := range defaults {
		_ = s.roleRepo.Create(ctx, &defaults[i])
	}
	return nil
}

func (s *TenantService) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

func (s *TenantService) Update(ctx context.Context, tenantID string, input UpdateTenantInput) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	tenant.Name = input.Name
	tenant.Email = input.Email
	tenant.Phone = input.Phone
	tenant.Address = input.Address
	tenant.LogoURL = input.LogoURL
	tenant.Timezone = input.Timezone
	tenant.Currency = input.Currency
	tenant.BillingCycle = input.BillingCycle
	tenant.DueDay = input.DueDay
	tenant.IsolirDay = input.IsolirDay
	tenant.GracePeriod = input.GracePeriod
	if input.DefaultBillingType != "" {
		tenant.DefaultBillingType = input.DefaultBillingType
	}
	if input.DefaultPaymentTiming != "" {
		tenant.DefaultPaymentTiming = input.DefaultPaymentTiming
	}
	tenant.IsActive = input.IsActive

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return s.tenantRepo.FindByID(ctx, tenantID)
}

func (s *TenantService) AdminUpdate(ctx context.Context, tenantID string, input AdminUpdateTenantInput) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	oldPlan := tenant.Plan
	tenant.Name = input.Name
	tenant.Email = input.Email
	tenant.Phone = input.Phone
	tenant.Address = input.Address
	tenant.IsActive = input.IsActive
	if input.Status != "" {
		tenant.Status = input.Status
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	// Ensure at least one owner user exists for the tenant
	users, _ := s.userRepo.ListByTenant(ctx, tenantID)
	hasOwner := false
	for _, u := range users {
		if u.Role == "owner" {
			hasOwner = true
			break
		}
	}

	if !hasOwner {
		password, _ := s.generateRandomPassword()
		if password == "" {
			password = "password123" // Fallback
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err == nil {
			user := &model.User{
				TenantID:     tenant.ID,
				Name:         tenant.Name,
				Email:        tenant.Email,
				PasswordHash: string(hash),
				Role:         "owner",
				Phone:        tenant.Phone,
				IsActive:     true,
			}
			if err := s.userRepo.Create(ctx, user); err == nil {
				// Send welcome message via WhatsApp
				go s.sendWelcomeMessage(context.Background(), tenant, password)
			}
		}
		_ = s.seedDefaultRoles(ctx, tenant.ID)
	}

	if input.Plan != "" || input.MaxCustomers > 0 || input.PlanExpiresAt != nil {
		planChanged := input.Plan != "" && input.Plan != oldPlan
		if input.Plan != "" {
			tenant.Plan = input.Plan
		}

		// Only change expiry if plan is explicitly changed or it was nil
		if planChanged || tenant.PlanExpiresAt == nil {
			if input.PlanExpiresAt != nil {
				tenant.PlanExpiresAt = input.PlanExpiresAt
			} else {
				// Auto calculate based on plan name if not provided
				pName := strings.ToLower(tenant.Plan)
				if strings.Contains(pName, "bulan") || strings.Contains(pName, "monthly") {
					t := time.Now().AddDate(0, 1, 0)
					tenant.PlanExpiresAt = &t
				} else if strings.Contains(pName, "tahun") || strings.Contains(pName, "yearly") {
					t := time.Now().AddDate(1, 0, 0)
					tenant.PlanExpiresAt = &t
				} else if tenant.Plan == "trial" {
					t := time.Now().AddDate(0, 0, 30)
					tenant.PlanExpiresAt = &t
				}
			}
		} else if input.PlanExpiresAt != nil {
			// If plan didn't change but expiry is explicitly provided, use it
			tenant.PlanExpiresAt = input.PlanExpiresAt
		}

		if input.MaxCustomers > 0 {
			tenant.MaxCustomers = input.MaxCustomers
		}
		tenant.UpdatedAt = time.Now()
		if err := s.tenantRepo.UpdatePlan(ctx, tenant); err != nil {
			return nil, err
		}
	}

	return s.tenantRepo.FindByID(ctx, tenantID)
}

func (s *TenantService) UpdateSettings(ctx context.Context, tenantID string, input UpdateSettingsInput) error {
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	tenant.WAAPIKey = input.WAAPIKey
	tenant.WASender = input.WASender
	tenant.PGProvider = input.PGProvider
	tenant.PGAPIKey = input.PGAPIKey
	tenant.PGSecretKey = input.PGSecretKey
	tenant.PGMerchantID = input.PGMerchantID
	tenant.PGSandbox = input.PGSandbox

	return s.tenantRepo.UpdateSettings(ctx, tenant)
}

func (s *TenantService) List(ctx context.Context, filter repository.TenantFilter) ([]model.Tenant, int, error) {
	return s.tenantRepo.List(ctx, filter)
}

// PGCredentials holds payment gateway credentials for an ad-hoc connection test.
type PGCredentials struct {
	Provider   string
	APIKey     string
	SecretKey  string
	MerchantID string
	Sandbox    bool
}

// TestPGConnection verifies payment gateway credentials.
// If override is non-nil, those credentials are tested directly (without reading from DB).
// Otherwise, the credentials stored in the database are used.
func (s *TenantService) TestPGConnection(ctx context.Context, tenantID string, override *PGCredentials) error {
	var provider, apiKey, secretKey, merchantID string
	var sandbox bool

	if override != nil {
		provider = override.Provider
		apiKey = override.APIKey
		secretKey = override.SecretKey
		merchantID = override.MerchantID
		sandbox = override.Sandbox
	} else {
		tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			return err
		}
		if tenant == nil {
			return ErrTenantNotFound
		}
		provider = tenant.PGProvider
		apiKey = tenant.PGAPIKey
		secretKey = tenant.PGSecretKey
		merchantID = tenant.PGMerchantID
		sandbox = tenant.PGSandbox
	}

	if provider == "" {
		return fmt.Errorf("payment gateway belum dikonfigurasi")
	}

	switch provider {
	case "tripay":
		if apiKey == "" {
			return fmt.Errorf("Tripay: API Key wajib diisi")
		}
		if secretKey == "" {
			return fmt.Errorf("Tripay: Private Key wajib diisi")
		}
		if merchantID == "" {
			return fmt.Errorf("Tripay: Merchant Code wajib diisi")
		}
		client := payment.NewTripayClient(apiKey, secretKey, merchantID, sandbox)
		return client.TestConnection(ctx)

	case "midtrans":
		// Midtrans test only needs the Server Key (pg_secret_key)
		if secretKey == "" {
			return fmt.Errorf("Midtrans: Server Key wajib diisi")
		}
		client := payment.NewMidtransClient(secretKey, sandbox)
		return client.TestConnection(ctx)

	case "xendit":
		// Xendit test uses the Secret Key (pg_api_key)
		if apiKey == "" {
			return fmt.Errorf("Xendit: Secret Key wajib diisi")
		}
		client := payment.NewXenditClient(apiKey, secretKey, sandbox)
		return client.TestConnection(ctx)

	default:
		return fmt.Errorf("provider payment gateway '%s' tidak didukung", provider)
	}
}
