package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type MobileHandler struct {
	mobileService *service.MobileService
}

func NewMobileHandler(mobileService *service.MobileService) *MobileHandler {
	return &MobileHandler{mobileService: mobileService}
}

type mobileLoginRequest struct {
	CustomerCode string `json:"customer_code"`
	Password     string `json:"password"`
}

// Login authenticates a customer by customer_code and password.
func (h *MobileHandler) Login(c *fiber.Ctx) error {
	var req mobileLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerCode == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor pelanggan dan password wajib diisi"})
	}

	tokens, customer, err := h.mobileService.Login(c.Context(), req.CustomerCode, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrMobileInvalidPassword) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Nomor pelanggan atau password salah"})
		}
		if errors.Is(err, service.ErrMobileAccountNotActive) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akun belum diaktivasi, silakan hubungi admin atau gunakan lupa password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Login gagal"})
	}

	return c.JSON(fiber.Map{
		"tokens":   tokens,
		"customer": customer,
	})
}
