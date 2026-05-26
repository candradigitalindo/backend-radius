package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type AuditLogHandler struct {
	repo repository.AuditLogRepository
}

func NewAuditLogHandler(repo repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

func (h *AuditLogHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.AuditLogFilter{
		UserID:   c.Query("user_id"),
		Action:   c.Query("action"),
		Resource: c.Query("resource"),
		Search:   c.Query("search"),
		Page:     page,
		PerPage:  perPage,
	}

	logs, total, err := h.repo.List(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat audit log"})
	}

	return c.JSON(fiber.Map{
		"data":     logs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
