package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type IPAMHandler struct {
	ipamService *service.IPAMService
}

func NewIPAMHandler(ipamService *service.IPAMService) *IPAMHandler {
	return &IPAMHandler{ipamService: ipamService}
}

// Pool handlers

func (h *IPAMHandler) CreatePool(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req model.IPPool
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID

	if req.Name == "" || req.Network == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan network wajib diisi"})
	}

	if err := h.ipamService.CreatePool(c.Context(), &req); err != nil {
		if errors.Is(err, repository.ErrDuplicateIPPool) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat pool"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *IPAMHandler) GetPool(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("id")

	pool, err := h.ipamService.GetPool(c.Context(), tenantID, poolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data pool"})
	}
	if pool == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pool tidak ditemukan"})
	}

	stats, _ := h.ipamService.GetPoolStats(c.Context(), tenantID, poolID)

	return c.JSON(fiber.Map{"data": pool, "stats": stats})
}

func (h *IPAMHandler) UpdatePool(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("id")

	var req model.IPPool
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.ID = poolID
	req.TenantID = tenantID

	if err := h.ipamService.UpdatePool(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui pool"})
	}

	return c.JSON(fiber.Map{"message": "Pool diperbarui"})
}

func (h *IPAMHandler) DeletePool(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("id")

	if err := h.ipamService.DeletePool(c.Context(), tenantID, poolID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus pool"})
	}

	return c.JSON(fiber.Map{"message": "Pool dihapus"})
}

func (h *IPAMHandler) ListPools(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	pools, total, err := h.ipamService.ListPools(c.Context(), tenantID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar pool"})
	}

	return c.JSON(fiber.Map{"data": pools, "total": total, "page": page, "per_page": perPage})
}

// IP Address handlers

func (h *IPAMHandler) CreateAddress(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")

	var req model.IPAddress
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	req.TenantID = tenantID
	req.PoolID = poolID

	if req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Alamat wajib diisi"})
	}

	if err := h.ipamService.CreateAddress(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat alamat"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": req})
}

func (h *IPAMHandler) CreateAddressBatch(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")

	var req struct {
		Addresses []string `json:"addresses"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if len(req.Addresses) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Alamat wajib diisi"})
	}

	addrs := make([]model.IPAddress, len(req.Addresses))
	for i, addr := range req.Addresses {
		addrs[i] = model.IPAddress{
			TenantID: tenantID,
			PoolID:   poolID,
			Address:  addr,
			Status:   "available",
		}
	}

	if err := h.ipamService.CreateAddressBatch(c.Context(), addrs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat alamat"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Alamat berhasil dibuat", "count": len(addrs)})
}

func (h *IPAMHandler) ListAddresses(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")

	filter := repository.IPAddressFilter{
		Status:  c.Query("status", "assigned"),
		Search:  c.Query("search"),
		Page:    1,
		PerPage: 50,
	}
	if p, err := strconv.Atoi(c.Query("page", "1")); err == nil {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(c.Query("per_page", "50")); err == nil {
		filter.PerPage = pp
	}

	addrs, total, err := h.ipamService.ListAddresses(c.Context(), tenantID, poolID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar alamat"})
	}

	return c.JSON(fiber.Map{"data": addrs, "total": total, "page": filter.Page, "per_page": filter.PerPage})
}

func (h *IPAMHandler) AssignAddress(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")
	addrID := c.Params("addrId")

	var req struct {
		CustomerID string `json:"customer_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.CustomerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID Pelanggan wajib diisi"})
	}

	err := h.ipamService.AssignAddress(c.Context(), tenantID, addrID, req.CustomerID)
	if err == nil {
		return c.JSON(fiber.Map{"message": "Alamat ditetapkan"})
	}

	// IP yang dipilih tidak tersedia — cari otomatis IP yang tersedia dari pool yang sama
	addr, autoErr := h.ipamService.ReleaseAndAssign(c.Context(), tenantID, poolID, req.CustomerID)
	if autoErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menetapkan alamat"})
	}
	if addr == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "IP yang dipilih tidak tersedia dan tidak ada IP lain yang tersedia di pool ini"})
	}

	return c.JSON(fiber.Map{"message": "IP yang dipilih tidak tersedia, otomatis ditetapkan " + addr.Address, "data": addr})
}

func (h *IPAMHandler) ReleaseAddress(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	addrID := c.Params("addrId")

	if err := h.ipamService.ReleaseAddress(c.Context(), tenantID, addrID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melepas alamat"})
	}

	return c.JSON(fiber.Map{"message": "Alamat dilepas"})
}

func (h *IPAMHandler) FindAvailable(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")

	addr, err := h.ipamService.FindAvailable(c.Context(), tenantID, poolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mencari alamat tersedia"})
	}
	if addr == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tidak ada alamat tersedia"})
	}

	return c.JSON(fiber.Map{"data": addr})
}

func (h *IPAMHandler) GetPoolStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	poolID := c.Params("poolId")

	stats, err := h.ipamService.GetPoolStats(c.Context(), tenantID, poolID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil statistik pool"})
	}

	return c.JSON(fiber.Map{"data": stats})
}
