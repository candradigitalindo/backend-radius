package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type WhatsAppHandler struct {
	waClient *whatsapp.Client
	repo     repository.WhatsAppRepository
	baseURL  string
}

func NewWhatsAppHandler(waClient *whatsapp.Client, repo repository.WhatsAppRepository) *WhatsAppHandler {
	return &WhatsAppHandler{waClient: waClient, repo: repo}
}

func (h *WhatsAppHandler) WithBaseURL(url string) *WhatsAppHandler {
	h.baseURL = url
	return h
}

// StartSession starts a WhatsApp session for the current tenant.
func (h *WhatsAppHandler) StartSession(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.waClient.StartSession(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "error", "message": "WhatsApp service tidak tersedia: " + err.Error()})
	}

	return c.JSON(result)
}

// GetStatus returns the WhatsApp session status for the current tenant.
func (h *WhatsAppHandler) GetStatus(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.waClient.GetSessionStatus(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.Map{"status": "disconnected", "message": "WhatsApp service tidak tersedia"})
	}

	return c.JSON(result)
}

// GetQR returns the QR code for WhatsApp pairing.
func (h *WhatsAppHandler) GetQR(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.waClient.GetQR(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.Map{"status": "disconnected", "message": "WhatsApp service tidak tersedia"})
	}

	return c.JSON(result)
}

// StopSession disconnects and removes the WhatsApp session for the current tenant.
func (h *WhatsAppHandler) StopSession(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	result, err := h.waClient.StopSession(c.Context(), tenantID)
	if err != nil {
		return c.JSON(fiber.Map{"status": "disconnected", "message": "Sesi sudah tidak aktif"})
	}

	return c.JSON(result)
}

// SendMessage handles both JSON and Multipart/Form-Data requests.
func (h *WhatsAppHandler) SendMessage(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var phone, message, fileURL string
	var tempFilePath string

	// Check if it's a multipart request (file upload)
	if contentType := c.Get("Content-Type"); len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		phone = c.FormValue("phone")
		message = c.FormValue("message")

		file, err := c.FormFile("file")
		if err == nil {
			// Create temp storage directory if it doesn't exist
			tempDir := "./storage/temp"
			_ = os.MkdirAll(tempDir, 0755)

			// Generate unique filename to avoid collisions
			ext := filepath.Ext(file.Filename)
			fileName := uuid.New().String() + ext
			tempFilePath = filepath.Join(tempDir, fileName)

			if err := c.SaveFile(file, tempFilePath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file sementara"})
			}

			// Generate temporary public URL for WA service to fetch
			fileURL = fmt.Sprintf("%s/temp/%s", h.baseURL, fileName)
		}
	} else {
		// JSON Request
		type request struct {
			Phone    string `json:"phone"`
			Message  string `json:"message"`
			ImageURL string `json:"image_url"`
			FileURL  string `json:"file_url"`
		}
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
		}
		phone = req.Phone
		message = req.Message
		fileURL = req.FileURL
		if fileURL == "" {
			fileURL = req.ImageURL
		}
	}

	if phone == "" || message == "" {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nomor telepon dan pesan wajib diisi"})
	}

	var result interface{}
	var err error

	if fileURL != "" {
		result, err = h.waClient.SendMessageWithFile(c.Context(), tenantID, phone, message, fileURL)
	} else {
		result, err = h.waClient.SendMessage(c.Context(), tenantID, phone, message)
	}

	// Auto-cleanup: remove temp file after sending to WA service
	if tempFilePath != "" {
		_ = os.Remove(tempFilePath)
	}

	// Log the send attempt
	logStatus := "sent"
	logErr := ""
	if err != nil {
		logStatus = "failed"
		logErr = err.Error()
	}
	_ = h.repo.CreateLog(c.Context(), &repository.WhatsAppLog{
		TenantID: tenantID,
		Phone:    phone,
		Message:  message,
		Status:   logStatus,
		ErrorMsg: logErr,
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// GetLogs returns the WhatsApp message logs for the current tenant.
func (h *WhatsAppHandler) GetLogs(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}

	logs, total, err := h.repo.ListLogs(c.Context(), tenantID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if logs == nil {
		logs = []repository.WhatsAppLog{}
	}

	return c.JSON(fiber.Map{"data": logs, "total": total})
}
