package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type ResellerHandler struct {
	resellerService *service.ResellerService
}

func NewResellerHandler(resellerService *service.ResellerService) *ResellerHandler {
	return &ResellerHandler{resellerService: resellerService}
}

func (h *ResellerHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req model.Reseller
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID

	if req.Name == "" || req.Email == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama, email, dan telepon wajib diisi"})
	}

	if err := h.resellerService.Create(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat reseller"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *ResellerHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	reseller, err := h.resellerService.GetByID(c.Context(), tenantID, resellerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data reseller"})
	}
	if reseller == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Reseller tidak ditemukan"})
	}

	summary, _ := h.resellerService.GetCommissionSummary(c.Context(), tenantID, resellerID)

	return c.JSON(fiber.Map{"data": reseller, "commission_summary": summary})
}

func (h *ResellerHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	var req model.Reseller
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.ID = resellerID
	req.TenantID = tenantID

	if err := h.resellerService.Update(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui reseller"})
	}

	return c.JSON(fiber.Map{"message": "Reseller diperbarui"})
}

func (h *ResellerHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	if err := h.resellerService.Delete(c.Context(), tenantID, resellerID); err != nil {
		if errors.Is(err, service.ErrResellerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus reseller"})
	}

	return c.JSON(fiber.Map{"message": "Reseller dihapus"})
}

func (h *ResellerHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	filter := repository.ResellerFilter{
		Status:  c.Query("status"),
		Search:  c.Query("search"),
		Page:    1,
		PerPage: 20,
	}
	if p, err := strconv.Atoi(c.Query("page", "1")); err == nil {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(c.Query("per_page", "20")); err == nil {
		filter.PerPage = pp
	}

	resellers, total, err := h.resellerService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar reseller"})
	}

	return c.JSON(fiber.Map{"data": resellers, "total": total, "page": filter.Page, "per_page": filter.PerPage})
}

func (h *ResellerHandler) AddCommission(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	var req model.ResellerCommission
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID
	req.ResellerID = resellerID

	if req.InvoiceID == "" || req.CustomerID == "" || req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID faktur, ID pelanggan, dan jumlah wajib diisi"})
	}

	if err := h.resellerService.AddCommission(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah komisi"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *ResellerHandler) ListCommissions(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	comms, total, err := h.resellerService.ListCommissions(c.Context(), tenantID, resellerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar komisi"})
	}

	return c.JSON(fiber.Map{"data": comms, "total": total, "page": page, "per_page": perPage})
}

func (h *ResellerHandler) PayCommission(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	commissionID := c.Params("commissionId")

	if err := h.resellerService.PayCommission(c.Context(), tenantID, commissionID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Komisi dibayar"})
}

func (h *ResellerHandler) PayAllPending(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	count, err := h.resellerService.PayAllPending(c.Context(), tenantID, resellerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membayar komisi"})
	}

	return c.JSON(fiber.Map{"message": "Semua komisi tertunda dibayar", "count": count})
}

func (h *ResellerHandler) GetCommissionSummary(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	summary, err := h.resellerService.GetCommissionSummary(c.Context(), tenantID, resellerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil ringkasan komisi"})
	}

	return c.JSON(fiber.Map{"data": summary})
}

func (h *ResellerHandler) ListCustomers(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resellerID := c.Params("id")

	customers, err := h.resellerService.ListCustomers(c.Context(), tenantID, resellerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pelanggan reseller"})
	}

	return c.JSON(fiber.Map{"data": customers})
}
