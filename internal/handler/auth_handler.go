package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type AuthHandler struct {
	authService  *service.AuthService
	userRepo     repository.UserRepository
	tenantRepo   repository.TenantRepository
	reminderRepo repository.ReminderRepository
	redis        *redis.Client
	waClient     *whatsapp.Client
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) WithResetDeps(userRepo repository.UserRepository, tenantRepo repository.TenantRepository, rdb *redis.Client, wa *whatsapp.Client) *AuthHandler {
	h.userRepo = userRepo
	h.tenantRepo = tenantRepo
	h.redis = rdb
	h.waClient = wa
	return h
}

func (h *AuthHandler) WithReminderRepo(repo repository.ReminderRepository) *AuthHandler {
	h.reminderRepo = repo
	return h
}

// otpTemplate returns the OTP WA message template from the superadmin tenant reminders.
func (h *AuthHandler) otpTemplate(ctx context.Context) string {
	defaultTpl := "🔐 *Kode OTP Reset Password - D Radius*\n\n" +
		"Halo *{nama}*,\n\n" +
		"Kami menerima permintaan reset password untuk akun *{nama_isp}* Anda.\n\n" +
		"🔑 *Kode OTP Anda:*\n" +
		"┌─────────────┐\n" +
		"│  *{kode_otp}*  │\n" +
		"└─────────────┘\n\n" +
		"⏱ Berlaku selama *{durasi}*\n" +
		"⚠️ Jangan bagikan kode ini kepada siapapun\n\n" +
		"Jika Anda tidak merasa meminta reset password, abaikan pesan ini.\n\n" +
		"Terima kasih,\n" +
		"_Tim D Radius_"

	if h.reminderRepo == nil || h.tenantRepo == nil {
		return defaultTpl
	}
	saT, err := h.tenantRepo.FindBySlug(ctx, "superadmin")
	if err != nil || saT == nil {
		return defaultTpl
	}
	rem, err := h.reminderRepo.FindActiveByType(ctx, saT.ID, "otp_reset_password")
	if err != nil || rem == nil {
		return defaultTpl
	}
	return rem.MessageTemplate
}

type registerRequest struct {
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Phone       string `json:"phone"`
	Fingerprint string `json:"fingerprint"`
	OTP         string `json:"otp"`
}

type loginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.authService.ListActivePlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": plans})
}

func (h *AuthHandler) SelectPlan(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		PlanSlug string `json:"plan_slug"`
	}
	if err := c.BodyParser(&req); err != nil || req.PlanSlug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pilih paket terlebih dahulu"})
	}

	if err := h.authService.SelectInitialPlan(c.Context(), tenantID, req.PlanSlug); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Paket berhasil dipilih"})
}

func (h *AuthHandler) SendRegistrationOTP(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email dan nomor WhatsApp wajib diisi"})
	}

	if err := h.authService.SendRegistrationOTP(c.Context(), req.Email, req.Phone); err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "OTP telah dikirim via WhatsApp"})
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" || req.Phone == "" || req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Semua kolom termasuk OTP wajib diisi"})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password minimal 8 karakter"})
	}

	resp, err := h.authService.Register(c.Context(), service.RegisterInput{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Email:       req.Email,
		Password:    req.Password,
		Phone:       req.Phone,
		Fingerprint: req.Fingerprint,
		IP:          c.IP(),
		OTP:         req.OTP,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email dan password wajib diisi"})
	}

	resp, err := h.authService.Login(c.Context(), service.LoginInput{
		TenantID: req.TenantID,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountInactive) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrMultipleTenants) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error(), "code": "MULTIPLE_TENANTS"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Login gagal"})
	}

	return c.JSON(resp)
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Refresh token wajib diisi"})
	}

	pair, err := h.authService.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui token"})
	}

	return c.JSON(pair)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi"})
	}
	role, _ := c.Locals("role").(string)
	tenantID, _ := c.Locals("tenant_id").(string)

	user, err := h.authService.GetCurrentUser(c.Context(), userID, role, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data pengguna"})
	}

	return c.JSON(user)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Blacklist the current token until its expiry
	if h.redis != nil {
		tokenStr, _ := c.Locals("token_raw").(string)
		tokenExp, _ := c.Locals("token_exp").(time.Time)
		if tokenStr != "" && !tokenExp.IsZero() {
			ttl := time.Until(tokenExp)
			if ttl > 0 {
				h.redis.Set(c.Context(), "token_blacklist:"+tokenStr, "1", ttl)
			}
		}
	}
	return c.JSON(fiber.Map{"message": "Berhasil logout"})
}

// RequestResetPIN sends a PIN via WhatsApp for tenant user password reset.
// POST /api/v1/auth/reset-pin
func (h *AuthHandler) RequestResetPIN(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email dan nomor WhatsApp wajib diisi"})
	}

	// Normalize phone
	phone := strings.TrimSpace(req.Phone)

	// Find user by email (may be in multiple tenants)
	users, err := h.userRepo.FindByEmailOnly(c.Context(), req.Email)
	if err != nil || len(users) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email tidak ditemukan"})
	}

	// Verify phone matches and tenant is active
	var matchedUser *struct {
		userID   string
		tenantID string
		name     string
		phone    string
	}
	for _, u := range users {
		if u.Phone == phone && u.IsActive {
			// Check tenant status
			tenant, _ := h.tenantRepo.FindByID(c.Context(), u.TenantID)
			if tenant != nil && tenant.Status == "active" && tenant.IsActive {
				matchedUser = &struct {
					userID   string
					tenantID string
					name     string
					phone    string
				}{u.ID, u.TenantID, u.Name, u.Phone}
				break
			}
		}
	}
	if matchedUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Akun tidak ditemukan atau belum aktif"})
	}

	// Rate limit
	redisKey := fmt.Sprintf("auth_reset:%s", matchedUser.userID)
	existing, _ := h.redis.Get(c.Context(), redisKey).Result()
	if existing != "" {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "PIN sudah dikirim, tunggu beberapa menit sebelum meminta ulang",
		})
	}

	ctx := context.Background()

	pin, err := generateResetPIN()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat PIN"})
	}

	// Store PIN in Redis with 5-min TTL
	if err := h.redis.Set(c.Context(), redisKey, pin, 5*time.Minute).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan PIN"})
	}

	// Resolve tenant name
	tenantName := "D Radius"
	if t, err := h.tenantRepo.FindByID(ctx, matchedUser.tenantID); err == nil && t != nil {
		tenantName = t.Name
	}

	// Build message from template (superadmin WA sender)
	tpl := h.otpTemplate(ctx)
	msg := strings.ReplaceAll(tpl, "{nama}", matchedUser.name)
	msg = strings.ReplaceAll(msg, "{kode_otp}", pin)
	msg = strings.ReplaceAll(msg, "{nama_isp}", tenantName)
	msg = strings.ReplaceAll(msg, "{durasi}", "5 menit")

	_, waErr := h.waClient.SendMessage(ctx, "superadmin", matchedUser.phone, msg)
	if waErr != nil {
		h.redis.Del(c.Context(), redisKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim OTP melalui WhatsApp"})
	}

	// Mask phone
	masked := phone
	if len(phone) > 8 {
		masked = phone[:4] + strings.Repeat("*", len(phone)-8) + phone[len(phone)-4:]
	}

	return c.JSON(fiber.Map{
		"message": "PIN telah dikirim melalui WhatsApp",
		"phone":   masked,
	})
}

// ResetPasswordWithPIN verifies PIN and resets the tenant user's password.
// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPasswordWithPIN(c *fiber.Ctx) error {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		PIN         string `json:"pin"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Email == "" || req.Phone == "" || req.PIN == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Semua field wajib diisi"})
	}
	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru minimal 8 karakter"})
	}

	phone := strings.TrimSpace(req.Phone)

	// Find user by email + phone
	users, err := h.userRepo.FindByEmailOnly(c.Context(), req.Email)
	if err != nil || len(users) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email tidak ditemukan"})
	}

	var matchedUser *model.User
	for _, u := range users {
		if u.Phone == phone && u.IsActive {
			matchedUser = u
			break
		}
	}
	if matchedUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor WhatsApp tidak sesuai"})
	}

	// Verify PIN
	redisKey := fmt.Sprintf("auth_reset:%s", matchedUser.ID)
	attemptsKey := redisKey + ":attempts"

	attempts, _ := h.redis.Get(c.Context(), attemptsKey).Int()
	if attempts >= 5 {
		h.redis.Del(c.Context(), redisKey)
		h.redis.Del(c.Context(), attemptsKey)
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Terlalu banyak percobaan. PIN telah dibatalkan. Silakan minta PIN baru.",
		})
	}

	storedPIN, err := h.redis.Get(c.Context(), redisKey).Result()
	if err != nil || storedPIN == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "PIN tidak valid atau sudah kedaluwarsa"})
	}

	if storedPIN != req.PIN {
		h.redis.Incr(c.Context(), attemptsKey)
		h.redis.Expire(c.Context(), attemptsKey, 5*time.Minute)
		remaining := 4 - attempts
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("PIN salah. Sisa percobaan: %d", remaining),
		})
	}

	// PIN matched
	h.redis.Del(c.Context(), attemptsKey)

	hash, hashErr := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mereset password"})
	}

	matchedUser.PasswordHash = string(hash)
	if err := h.userRepo.Update(c.Context(), matchedUser); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mereset password"})
	}

	h.redis.Del(c.Context(), redisKey)

	return c.JSON(fiber.Map{"message": "Password berhasil direset. Silakan login dengan password baru."})
}

func generateResetPIN() (string, error) {
	pin := ""
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		pin += fmt.Sprintf("%d", n.Int64())
	}
	return pin, nil
}

// UpdateProfile memperbarui nama, nomor telepon, dan/atau email user yang sedang login.
// PUT /auth/me
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi"})
	}

	var req struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	user, err := h.authService.UpdateProfile(c.Context(), userID, service.UpdateProfileInput{
		Name:  req.Name,
		Phone: req.Phone,
		Email: req.Email,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui profil"})
	}

	return c.JSON(user)
}

// ChangePassword mengganti password user yang sedang login.
// PUT /auth/change-password
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tidak terautentikasi"})
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password lama dan password baru wajib diisi"})
	}
	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru minimal 8 karakter"})
	}

	if err := h.authService.ChangePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		// "Password saat ini salah" bukan ErrXxx sentinel, tangkap pesan langsung
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Password berhasil diperbarui"})
}
