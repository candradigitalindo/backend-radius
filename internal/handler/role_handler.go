package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type RoleHandler struct {
	roleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

func (h *RoleHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	roles, err := h.roleService.List(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat role"})
	}

	return c.JSON(fiber.Map{"data": roles})
}

func (h *RoleHandler) ListPermissions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": model.AllPermissionGroups()})
}

type createRoleRequest struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *RoleHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	actorRole, _ := c.Locals("role").(string)
	actorPerms, _ := c.Locals("permissions").([]string)

	var req createRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" || req.Slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama dan slug wajib diisi"})
	}

	role, err := h.roleService.Create(c.Context(), service.CreateRoleInput{
		TenantID:         tenantID,
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      req.Description,
		Permissions:      req.Permissions,
		ActorRole:        actorRole,
		ActorPermissions: actorPerms,
	})
	if err != nil {
		if errors.Is(err, service.ErrRoleSlugExists) || errors.Is(err, service.ErrInvalidSlug) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrCannotGrantUnheld) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat role"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": role})
}

type updateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *RoleHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	actorRole, _ := c.Locals("role").(string)
	actorPerms, _ := c.Locals("permissions").([]string)
	roleID := c.Params("id")

	var req updateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	role, err := h.roleService.Update(c.Context(), service.UpdateRoleInput{
		TenantID:         tenantID,
		RoleID:           roleID,
		Name:             req.Name,
		Description:      req.Description,
		Permissions:      req.Permissions,
		ActorRole:        actorRole,
		ActorPermissions: actorPerms,
	})
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrCannotEditOwnerRole) || errors.Is(err, service.ErrCannotGrantUnheld) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui role"})
	}

	return c.JSON(fiber.Map{"data": role})
}

func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	roleID := c.Params("id")

	err := h.roleService.Delete(c.Context(), tenantID, roleID)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrCannotDeleteSystemRole) || errors.Is(err, service.ErrRoleInUse) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus role"})
	}

	return c.JSON(fiber.Map{"message": "Role dihapus"})
}
