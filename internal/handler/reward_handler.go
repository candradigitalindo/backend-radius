package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type RewardHandler struct {
	rewardService *service.RewardService
}

func NewRewardHandler(rewardService *service.RewardService) *RewardHandler {
	return &RewardHandler{rewardService: rewardService}
}

// Reward CRUD

func (h *RewardHandler) CreateReward(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req model.Reward
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID

	if req.Name == "" || req.Type == "" || req.ValueType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, tipe, dan value_type wajib diisi"})
	}

	if err := h.rewardService.CreateReward(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat reward"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *RewardHandler) GetReward(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	rewardID := c.Params("id")

	reward, err := h.rewardService.GetReward(c.Context(), tenantID, rewardID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data reward"})
	}
	if reward == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Reward tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"data": reward})
}

func (h *RewardHandler) UpdateReward(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	rewardID := c.Params("id")

	var req model.Reward
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.ID = rewardID
	req.TenantID = tenantID

	if err := h.rewardService.UpdateReward(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui reward"})
	}

	return c.JSON(fiber.Map{"message": "Reward diperbarui"})
}

func (h *RewardHandler) DeleteReward(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	rewardID := c.Params("id")

	if err := h.rewardService.DeleteReward(c.Context(), tenantID, rewardID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus reward"})
	}

	return c.JSON(fiber.Map{"message": "Reward dihapus"})
}

func (h *RewardHandler) ListRewards(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	activeOnly := c.Query("active_only") == "true"
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	rewards, total, err := h.rewardService.ListRewards(c.Context(), tenantID, activeOnly, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar reward"})
	}

	return c.JSON(fiber.Map{"data": rewards, "total": total, "page": page, "per_page": perPage})
}

// Referral

func (h *RewardHandler) CreateReferral(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req model.Referral
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID

	if req.ReferrerID == "" || req.ReferredID == "" || req.RewardID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID perujuk, ID yang dirujuk, dan ID reward wajib diisi"})
	}

	if err := h.rewardService.CreateReferral(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat referral"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *RewardHandler) GetReferral(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	referralID := c.Params("id")

	ref, err := h.rewardService.GetReferral(c.Context(), tenantID, referralID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data referral"})
	}
	if ref == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Referral tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"data": ref})
}

func (h *RewardHandler) ListReferrals(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	filter := repository.ReferralFilter{
		CustomerID: c.Query("customer_id"),
		Status:     c.Query("status"),
		Page:       1,
		PerPage:    20,
	}
	if p, err := strconv.Atoi(c.Query("page", "1")); err == nil {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(c.Query("per_page", "20")); err == nil {
		filter.PerPage = pp
	}

	refs, total, err := h.rewardService.ListReferrals(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar referral"})
	}

	return c.JSON(fiber.Map{"data": refs, "total": total, "page": filter.Page, "per_page": filter.PerPage})
}

func (h *RewardHandler) MarkRewarded(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	referralID := c.Params("id")

	if err := h.rewardService.MarkRewarded(c.Context(), tenantID, referralID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menandai referral sebagai diberi reward"})
	}

	return c.JSON(fiber.Map{"message": "Referral ditandai diberi reward"})
}

// Claims

func (h *RewardHandler) CreateClaim(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req model.RewardClaim
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID

	if req.CustomerID == "" || req.RewardID == "" || req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID pelanggan, ID reward, dan jumlah wajib diisi"})
	}

	if err := h.rewardService.CreateClaim(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat klaim"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *RewardHandler) ListClaims(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	filter := repository.ClaimFilter{
		CustomerID: c.Query("customer_id"),
		Status:     c.Query("status"),
		Page:       1,
		PerPage:    20,
	}
	if p, err := strconv.Atoi(c.Query("page", "1")); err == nil {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(c.Query("per_page", "20")); err == nil {
		filter.PerPage = pp
	}

	claims, total, err := h.rewardService.ListClaims(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar klaim"})
	}

	return c.JSON(fiber.Map{"data": claims, "total": total, "page": filter.Page, "per_page": filter.PerPage})
}

func (h *RewardHandler) ApplyClaim(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	claimID := c.Params("claimId")

	if err := h.rewardService.ApplyClaim(c.Context(), tenantID, claimID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Klaim diterapkan"})
}

func (h *RewardHandler) GetCustomerBalance(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	balance, err := h.rewardService.GetCustomerBalance(c.Context(), tenantID, customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil saldo"})
	}

	return c.JSON(fiber.Map{"balance": balance})
}

func (h *RewardHandler) GetRewardStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	stats, err := h.rewardService.GetRewardStats(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil statistik reward"})
	}

	return c.JSON(fiber.Map{"data": stats})
}

func (h *RewardHandler) GetRewardDashboard(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	months, _ := strconv.Atoi(c.Query("months", "12"))
	if months <= 0 || months > 24 {
		months = 12
	}

	dashboard, err := h.rewardService.GetRewardDashboard(c.Context(), tenantID, months)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil dashboard reward"})
	}

	return c.JSON(fiber.Map{"data": dashboard, "months": months})
}
