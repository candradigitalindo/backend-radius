package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
}

type updateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
	IsActive *bool  `json:"is_active"`
	Password string `json:"password"`
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	users, err := h.userService.List(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data pengguna"})
	}

	return c.JSON(fiber.Map{"data": users})
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, email, password, dan role wajib diisi"})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password minimal 8 karakter"})
	}

	user, err := h.userService.Create(c.Context(), service.CreateUserInput{
		TenantID: tenantID,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Phone:    req.Phone,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserEmailExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvalidRole) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pengguna"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": user})
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	currentUserID, _ := c.Locals("user_id").(string)
	userID := c.Params("id")

	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Email == "" || req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, email, dan role wajib diisi"})
	}

	if req.Password != "" && len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password minimal 8 karakter"})
	}

	user, err := h.userService.Update(c.Context(), service.UpdateUserInput{
		TenantID:    tenantID,
		UserID:      userID,
		CurrentUser: currentUserID,
		Name:        req.Name,
		Email:       req.Email,
		Role:        req.Role,
		Phone:       req.Phone,
		IsActive:    req.IsActive,
		Password:    req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrUserEmailExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrCannotEditSelf) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvalidRole) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah pengguna"})
	}

	return c.JSON(fiber.Map{"data": user})
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	currentUserID, _ := c.Locals("user_id").(string)
	userID := c.Params("id")

	err := h.userService.Delete(c.Context(), tenantID, userID, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrCannotDeleteSelf) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus pengguna"})
	}

	return c.JSON(fiber.Map{"message": "Pengguna berhasil dihapus"})
}

func (h *UserHandler) ToggleActive(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	currentUserID, _ := c.Locals("user_id").(string)
	userID := c.Params("id")

	user, err := h.userService.ToggleActive(c.Context(), tenantID, userID, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrCannotEditSelf) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah status pengguna"})
	}

	return c.JSON(fiber.Map{"data": user})
}
