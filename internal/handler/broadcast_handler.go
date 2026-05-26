package handler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type BroadcastHandler struct {
	broadcastService *service.BroadcastService
	baseURL          string
}

func NewBroadcastHandler(broadcastService *service.BroadcastService) *BroadcastHandler {
	return &BroadcastHandler{broadcastService: broadcastService}
}

func (h *BroadcastHandler) WithBaseURL(url string) *BroadcastHandler {
	h.baseURL = url
	return h
}

type sendBroadcastRequest struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	ImageURL string `json:"image_url"`
	Target   string `json:"target"`
}

func (h *BroadcastHandler) Send(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)

	var reqType, title, message, imageURL, target string
	var tempFilePath string

	if ct := c.Get("Content-Type"); len(ct) >= 19 && ct[:19] == "multipart/form-data" {
		reqType = c.FormValue("type")
		title = c.FormValue("title")
		message = c.FormValue("message")
		target = c.FormValue("target")

		if file, err := c.FormFile("file"); err == nil {
			tempDir := "./storage/temp"
			_ = os.MkdirAll(tempDir, 0755)
			ext := filepath.Ext(file.Filename)
			fileName := uuid.New().String() + ext
			tempFilePath = filepath.Join(tempDir, fileName)
			if err := c.SaveFile(file, tempFilePath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file sementara"})
			}
			imageURL = fmt.Sprintf("%s/temp/%s", h.baseURL, fileName)
		}
	} else {
		var req sendBroadcastRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
		}
		reqType = req.Type
		title = req.Title
		message = req.Message
		imageURL = req.ImageURL
		target = req.Target
	}

	if reqType == "" || title == "" || message == "" {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipe, judul, dan pesan wajib diisi"})
	}

	var cleanup func()
	if tempFilePath != "" {
		tp := tempFilePath
		cleanup = func() { _ = os.Remove(tp) }
	}

	broadcast, err := h.broadcastService.Send(c.Context(), service.SendBroadcastInput{
		TenantID:    tenantID,
		Type:        reqType,
		Title:       title,
		Message:     message,
		ImageURL:    imageURL,
		Target:      target,
		SentBy:      userID,
		CleanupFunc: cleanup,
	})
	if err != nil {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim broadcast"})
	}

	return c.Status(fiber.StatusCreated).JSON(broadcast)
}

func (h *BroadcastHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	broadcastID := c.Params("id")

	broadcast, err := h.broadcastService.GetByID(c.Context(), tenantID, broadcastID)
	if err != nil {
		if errors.Is(err, service.ErrBroadcastNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(broadcast)
}

func (h *BroadcastHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	broadcastID := c.Params("id")

	if err := h.broadcastService.Delete(c.Context(), tenantID, broadcastID); err != nil {
		if errors.Is(err, service.ErrBroadcastNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{"message": "Broadcast dihapus"})
}

func (h *BroadcastHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.BroadcastFilter{
		Type:    c.Query("type"),
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	broadcasts, total, err := h.broadcastService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	return c.JSON(fiber.Map{
		"data":     broadcasts,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
