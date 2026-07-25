package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
	"github.com/candrasyahputra/radius-server/internal/pkg/token"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("Email atau password salah")
	ErrEmailAlreadyExists = errors.New("Email sudah terdaftar")
	ErrAccountInactive    = errors.New("Akun tidak aktif")
	ErrInvalidToken       = errors.New("Token tidak valid atau kedaluwarsa")
	ErrUserNotFound       = errors.New("Pengguna tidak ditemukan")
	ErrMultipleTenants    = errors.New("Ditemukan beberapa tenant, silakan tentukan tenant_id")
)

type AuthService struct {
	userRepo      repository.UserRepository
	customerRepo  repository.CustomerRepository
	tenantRepo    repository.TenantRepository
	roleRepo      repository.RoleRepository
	subRepo       repository.SubscriptionRepository
	reminderRepo  repository.ReminderRepository
	tokenManager  *token.Manager
	tenantService *TenantService
	redis         *redis.Client
}

func NewAuthService(userRepo repository.UserRepository, customerRepo repository.CustomerRepository, tenantRepo repository.TenantRepository, roleRepo repository.RoleRepository, tokenManager *token.Manager, rdb *redis.Client) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		customerRepo: customerRepo,
		tenantRepo:   tenantRepo,
		roleRepo:     roleRepo,
		tokenManager: tokenManager,
		redis:        rdb,
	}
}

func (s *AuthService) WithReminderRepo(reminderRepo repository.ReminderRepository) *AuthService {
	s.reminderRepo = reminderRepo
	return s
}

func (s *AuthService) WithTenantService(tenantService *TenantService) *AuthService {
	s.tenantService = tenantService
	return s
}

func (s *AuthService) WithSubscriptionRepo(subRepo repository.SubscriptionRepository) *AuthService {
	s.subRepo = subRepo
	return s
}

func (s *AuthService) ListActivePlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	if s.subRepo == nil {
		return nil, fmt.Errorf("subscription repository not available")
	}
	return s.subRepo.ListPlans(ctx, true)
}

func (s *AuthService) SelectInitialPlan(ctx context.Context, tenantID, planSlug string) error {
	if s.subRepo == nil {
		return fmt.Errorf("subscription repository not available")
	}

	plan, err := s.subRepo.FindPlanBySlug(ctx, planSlug)
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("paket tidak ditemukan")
	}

	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}

	// Update tenant plan and expiry
	tenant.Plan = plan.Slug
	tenant.MaxCustomers = plan.MaxCustomers

	now := time.Now()
	if plan.DurationMonths > 0 {
		expiry := now.AddDate(0, int(plan.DurationMonths), 0)
		tenant.PlanExpiresAt = &expiry
	} else if plan.Slug == "trial" {
		expiry := now.AddDate(0, 0, 30)
		tenant.PlanExpiresAt = &expiry
	} else {
		tenant.PlanExpiresAt = nil // Unlimited/Free
	}

	return s.tenantRepo.UpdatePlan(ctx, tenant)
}

func (s *AuthService) SendRegistrationOTP(ctx context.Context, email, phone string) error {
	if s.tenantService == nil || s.tenantService.waClient == nil {
		return fmt.Errorf("whatsapp service not available")
	}

	// Check if email already exists
	existing, _ := s.userRepo.FindByEmailOnly(ctx, email)
	if len(existing) > 0 {
		return ErrEmailAlreadyExists
	}

	// Generate 6-digit OTP
	otp := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		otp += fmt.Sprintf("%d", n.Int64())
	}

	// Save to Redis (5 mins TTL)
	redisKey := fmt.Sprintf("reg_otp:%s", phone)
	if err := s.redis.Set(ctx, redisKey, otp, 5*time.Minute).Err(); err != nil {
		return err
	}

	msg := ""
	if s.reminderRepo != nil && s.tenantRepo != nil {
		saT, err := s.tenantRepo.FindBySlug(ctx, "superadmin")
		if err == nil && saT != nil {
			rem, err := s.reminderRepo.FindActiveByType(ctx, saT.ID, "otp_registration")
			if err == nil && rem != nil {
				msg = rem.MessageTemplate
				msg = strings.ReplaceAll(msg, "{kode_otp}", otp)
			}
		}
	}

	if msg == "" {
		msg = fmt.Sprintf("🔐 *Kode Verifikasi D Radius*\n\n"+
			"Halo,\n\n"+
			"Kode verifikasi (OTP) untuk pendaftaran akun Anda adalah:\n\n"+
			"*%s*\n\n"+
			"Kode ini berlaku selama 5 menit. Jangan bagikan kode ini kepada siapapun.\n\n"+
			"Terima kasih,\n"+
			"_Tim Support D Radius_", otp)
	}

	_, err := s.tenantService.waClient.SendMessage(ctx, "superadmin", phone, msg)
	return err
}

type RegisterInput struct {
	TenantID    string
	Name        string
	Email       string
	Password    string
	Phone       string
	Fingerprint string
	IP          string
	OTP         string
}

type LoginInput struct {
	TenantID string
	Email    string
	Password string
}

type AuthResponse struct {
	User      UserResponse     `json:"user"`
	TokenPair *token.TokenPair `json:"token"`
	Status    string           `json:"status,omitempty"`
}

type UserResponse struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	Phone         string     `json:"phone"`
	Plan          string     `json:"plan,omitempty"`
	PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	Permissions   []string   `json:"permissions"`
}

// slugify converts a business name into a URL-safe slug (lowercase, hyphen-
// separated, ascii alnum only) matching the tenant slug rules.
func slugify(name string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if b.Len() > 0 && !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = strings.Trim(out[:50], "-")
	}
	return out
}

// uniqueTenantSlug derives a readable slug from name and ensures it is unused,
// appending -2, -3, ... on collision. Falls back to a ULID if exhausted.
func (s *AuthService) uniqueTenantSlug(ctx context.Context, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "tenant"
	}
	candidate := base
	for i := 2; i <= 1000; i++ {
		existing, err := s.tenantRepo.FindBySlug(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return id.New(), nil
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	// Verify OTP
	if input.OTP == "" {
		return nil, errors.New("kode OTP wajib diisi")
	}

	redisKey := fmt.Sprintf("reg_otp:%s", input.Phone)
	storedOTP, err := s.redis.Get(ctx, redisKey).Result()
	if err != nil || storedOTP != input.OTP {
		return nil, errors.New("kode OTP tidak valid atau sudah kadaluwarsa")
	}

	// Success verification, delete OTP
	s.redis.Del(ctx, redisKey)

	// Public registration only creates new tenants — cannot join existing tenant
	if input.TenantID != "" {
		return nil, errors.New("registrasi hanya untuk membuat tenant baru")
	}

	existingList, err := s.userRepo.FindByEmailOnly(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if len(existingList) > 0 {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		TenantID:     input.TenantID,
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         "admin",
		Phone:        input.Phone,
		IsActive:     true,
	}

	status := "active"
	if user.TenantID == "" {
		if s.tenantService != nil {
			slug, slugErr := s.uniqueTenantSlug(ctx, input.Name)
			if slugErr != nil {
				return nil, slugErr
			}
			tenant, err := s.tenantService.Create(ctx, CreateTenantInput{
				Name:             input.Name,
				Slug:             slug,
				Email:            input.Email,
				Phone:            input.Phone,
				Plan:             "", // Empty plan initially
				Fingerprint:      input.Fingerprint,
				RegistrationIP:   input.IP,
				SelfRegistration: true,
			})
			if err != nil {
				return nil, err
			}
			user.TenantID = tenant.ID
			user.Role = "owner"
			status = tenant.Status
		} else {
			slug, slugErr := s.uniqueTenantSlug(ctx, input.Name)
			if slugErr != nil {
				return nil, slugErr
			}
			tenant := &model.Tenant{
				Name:         input.Name,
				Slug:         slug,
				Email:        input.Email,
				Timezone:     "Asia/Jakarta",
				Currency:     "IDR",
				BillingCycle: 1,
				DueDay:       20,
				IsolirDay:    7,
				GracePeriod:  3,
				Plan:         "free",
				MaxCustomers: 50,
				IsActive:     true,
			}
			if err := s.tenantRepo.Create(ctx, tenant); err != nil {
				return nil, err
			}
			user.TenantID = tenant.ID
			user.Role = "owner"

			// Seed default roles for new tenant
			_ = s.seedDefaultRoles(ctx, tenant.ID)
		}
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch {
			case strings.Contains(pgErr.ConstraintName, "email"):
				return nil, ErrEmailAlreadyExists
			case strings.Contains(pgErr.ConstraintName, "phone"):
				return nil, errors.New("nomor telepon sudah terdaftar, gunakan nomor lain")
			default:
				return nil, errors.New("data yang Anda masukkan sudah terdaftar di sistem")
			}
		}
		return nil, err
	}

	pair, err := s.tokenManager.GeneratePair(token.Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Role:     user.Role,
	})
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:      s.toUserResponseWithPerms(ctx, user),
		TokenPair: pair,
		Status:    status,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	var user *model.User

	if input.TenantID != "" {
		u, err := s.userRepo.FindByEmail(ctx, input.TenantID, input.Email)
		if err != nil {
			return nil, err
		}
		user = u
	} else {
		users, err := s.userRepo.FindByEmailOnly(ctx, input.Email)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return nil, ErrInvalidCredentials
		}
		if len(users) > 1 {
			return nil, ErrMultipleTenants
		}
		user = users[0]
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Check tenant status. Distinguish a real DB error (propagate → 500 + log)
	// from a genuinely missing tenant (treat as invalid credentials). Mapping DB
	// errors to ErrInvalidCredentials hides the real cause behind a misleading
	// "Email atau password salah" message.
	tenant, err := s.tenantRepo.FindByID(ctx, user.TenantID)
	if err != nil {
		log.Printf("Login: gagal memuat tenant %q untuk user %q: %v", user.TenantID, user.Email, err)
		return nil, fmt.Errorf("gagal memuat data tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidCredentials
	}
	if tenant.Status == "pending" {
		return nil, errors.New("Akun Anda sedang dalam peninjauan keamanan. Kami akan mengaktifkannya segera.")
	}
	if tenant.Status == "suspended" || !tenant.IsActive {
		return nil, errors.New("Akun Anda ditangguhkan atau tidak aktif")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	pair, err := s.tokenManager.GeneratePair(token.Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Role:     user.Role,
	})
	if err != nil {
		return nil, err
	}

	// Reuse the tenant already loaded above for plan info — no second query.
	resp := s.toUserResponseWithPerms(ctx, user)
	resp.Plan = tenant.Plan
	resp.PlanExpiresAt = tenant.PlanExpiresAt

	return &AuthResponse{
		User:      resp,
		TokenPair: pair,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*token.TokenPair, error) {
	claims, err := s.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Portal customer tokens use the customers table
	if claims.Role == "customer" {
		customer, err := s.customerRepo.FindByID(ctx, claims.TenantID, claims.UserID)
		if err != nil {
			return nil, err
		}
		if customer == nil || customer.Status != "active" {
			return nil, ErrInvalidToken
		}
		return s.tokenManager.GeneratePair(token.Claims{
			UserID:   customer.ID,
			TenantID: customer.TenantID,
			Role:     "customer",
		})
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, ErrInvalidToken
	}

	return s.tokenManager.GeneratePair(token.Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Role:     user.Role,
	})
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID, role, tenantID string) (*UserResponse, error) {
	// Portal customer: look up in customers table
	if role == "customer" {
		customer, err := s.customerRepo.FindByID(ctx, tenantID, userID)
		if err != nil {
			return nil, err
		}
		if customer == nil {
			return nil, ErrUserNotFound
		}
		resp := UserResponse{
			ID:       customer.ID,
			TenantID: customer.TenantID,
			Name:     customer.Name,
			Email:    customer.Email,
			Role:     "customer",
			Phone:    customer.Phone,
		}
		return &resp, nil
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	resp := s.toUserResponseWithPerms(ctx, user)

	// Attach tenant plan info for staff roles
	if user.TenantID != "" {
		tenant, err := s.tenantRepo.FindByID(ctx, user.TenantID)
		if err == nil && tenant != nil {
			resp.Plan = tenant.Plan
			resp.PlanExpiresAt = tenant.PlanExpiresAt
		}
	}

	return &resp, nil
}

func toUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:       u.ID,
		TenantID: u.TenantID,
		Name:     u.Name,
		Email:    u.Email,
		Role:     u.Role,
		Phone:    u.Phone,
	}
}

func (s *AuthService) toUserResponseWithPerms(ctx context.Context, u *model.User) UserResponse {
	resp := toUserResponse(u)
	resp.Permissions = s.getPermissionsForRole(ctx, u.TenantID, u.Role)
	return resp
}

func (s *AuthService) getPermissionsForRole(ctx context.Context, tenantID, roleSlug string) []string {
	if roleSlug == "superadmin" {
		return model.AllPermissionKeys()
	}
	if roleSlug == "owner" {
		return model.AllPermissionKeys()
	}

	_ = s.ensureDefaultRoles(ctx, tenantID)

	role, err := s.roleRepo.FindBySlug(ctx, tenantID, roleSlug)
	if err != nil || role == nil {
		return []string{}
	}
	return role.Permissions
}

func (s *AuthService) ensureDefaultRoles(ctx context.Context, tenantID string) error {
	count, err := s.roleRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.seedDefaultRoles(ctx, tenantID)
}

func (s *AuthService) seedDefaultRoles(ctx context.Context, tenantID string) error {
	defaults := []model.Role{
		{TenantID: tenantID, Name: "Owner", Slug: "owner", Description: "Pemilik tenant dengan akses penuh", IsSystem: true, Permissions: model.AllPermissionKeys()},
		{TenantID: tenantID, Name: "Admin", Slug: "admin", Description: "Administrator dengan akses luas", IsSystem: true, Permissions: model.DefaultAdminPermissions()},
		{TenantID: tenantID, Name: "Teknisi", Slug: "technician", Description: "Teknisi lapangan", IsSystem: true, Permissions: model.DefaultTechnicianPermissions()},
	}
	for i := range defaults {
		// Uses ON CONFLICT DO NOTHING — safe for concurrent calls
		if err := s.roleRepo.Create(ctx, &defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

// ChangePassword verifies the current password and updates to the new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("Password saat ini salah")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.userRepo.Update(ctx, user)
}

// UpdateProfileInput berisi field yang boleh diubah oleh user sendiri.
type UpdateProfileInput struct {
	Name  string
	Phone string
	Email string
}

// UpdateProfile memperbarui profil user yang sedang login (name, phone, email).
// Email divalidasi unik dalam tenant yang sama.
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Validasi email unik jika berubah
	if input.Email != "" && input.Email != user.Email {
		existing, err := s.userRepo.FindByEmail(ctx, user.TenantID, input.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != userID {
			return nil, ErrEmailAlreadyExists
		}
		user.Email = input.Email
	}

	if input.Name != "" {
		user.Name = input.Name
	}
	user.Phone = input.Phone

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	resp := s.toUserResponseWithPerms(ctx, user)
	return &resp, nil
}

