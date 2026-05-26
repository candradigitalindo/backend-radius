package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type WATemplateHandler struct {
	repo repository.WABroadcastTemplateRepository
}

func NewWATemplateHandler(repo repository.WABroadcastTemplateRepository) *WATemplateHandler {
	return &WATemplateHandler{repo: repo}
}

func (h *WATemplateHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	category := c.Query("category")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))

	templates, total, err := h.repo.List(c.Context(), tenantID, category, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if templates == nil {
		templates = []repository.WABroadcastTemplate{}
	}
	return c.JSON(fiber.Map{"data": templates, "total": total})
}

func (h *WATemplateHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	t, err := h.repo.FindByID(c.Context(), tenantID, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if t == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Template tidak ditemukan"})
	}
	return c.JSON(fiber.Map{"data": t})
}

type waTemplateRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Message  string `json:"message"`
	ImageURL string `json:"image_url"`
}

func (h *WATemplateHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req waTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Name == "" || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan pesan wajib diisi"})
	}
	if req.Category == "" {
		req.Category = "pengumuman"
	}

	t := &repository.WABroadcastTemplate{
		TenantID: tenantID,
		Name:     req.Name,
		Category: req.Category,
		Message:  req.Message,
		ImageURL: req.ImageURL,
	}
	if err := h.repo.Create(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": t})
}

func (h *WATemplateHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	templateID := c.Params("id")

	existing, err := h.repo.FindByID(c.Context(), tenantID, templateID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if existing == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Template tidak ditemukan"})
	}

	var req waTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	existing.Name = req.Name
	existing.Category = req.Category
	existing.Message = req.Message
	existing.ImageURL = req.ImageURL

	if err := h.repo.Update(c.Context(), existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": existing})
}

func (h *WATemplateHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if err := h.repo.Delete(c.Context(), tenantID, c.Params("id")); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Template dihapus"})
}
