package handler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type NotificationHandler struct {
	notifService *service.NotificationService
	baseURL      string
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

func (h *NotificationHandler) WithBaseURL(url string) *NotificationHandler {
	h.baseURL = url
	return h
}

type registerDeviceRequest struct {
	DeviceType string `json:"device_type"`
	FCMToken   string `json:"fcm_token"`
}

func (h *NotificationHandler) RegisterDevice(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	var req registerDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.FCMToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "fcm_token wajib diisi"})
	}
	if req.DeviceType == "" {
		req.DeviceType = "android"
	}

	device := &model.PushDevice{
		TenantID:   tenantID,
		CustomerID: customerID,
		DeviceType: req.DeviceType,
		FCMToken:   req.FCMToken,
	}

	if err := h.notifService.RegisterDevice(c.Context(), device); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mendaftarkan perangkat"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Perangkat terdaftar"})
}

type unregisterDeviceRequest struct {
	FCMToken string `json:"fcm_token"`
}

func (h *NotificationHandler) UnregisterDevice(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	var req unregisterDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if err := h.notifService.UnregisterDevice(c.Context(), tenantID, customerID, req.FCMToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus perangkat"})
	}

	return c.JSON(fiber.Map{"message": "Perangkat dihapus"})
}

func (h *NotificationHandler) AdminList(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	notifs, total, err := h.notifService.GetNotifications(c.Context(), tenantID, "", page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil notifikasi"})
	}

	return c.JSON(fiber.Map{
		"data":     notifs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	notifs, total, err := h.notifService.GetNotifications(c.Context(), tenantID, customerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil notifikasi"})
	}

	return c.JSON(fiber.Map{
		"notifications": notifs,
		"total":         total,
		"page":          page,
		"per_page":      perPage,
	})
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	notifID := c.Params("id")

	if err := h.notifService.MarkRead(c.Context(), tenantID, notifID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menandai notifikasi sebagai dibaca"})
	}

	return c.JSON(fiber.Map{"message": "Notifikasi ditandai dibaca"})
}

func (h *NotificationHandler) CustomerMarkRead(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)
	notifID := c.Params("id")

	if err := h.notifService.MarkReadByCustomer(c.Context(), tenantID, notifID, customerID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menandai notifikasi sebagai dibaca"})
	}

	return c.JSON(fiber.Map{"message": "Notifikasi ditandai dibaca"})
}

func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	if err := h.notifService.MarkAllRead(c.Context(), tenantID, customerID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menandai semua sebagai dibaca"})
	}

	return c.JSON(fiber.Map{"message": "Semua notifikasi ditandai dibaca"})
}

func (h *NotificationHandler) UnreadCount(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID, _ := c.Locals("user_id").(string)

	count, err := h.notifService.UnreadCount(c.Context(), tenantID, customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil jumlah belum dibaca"})
	}

	return c.JSON(fiber.Map{"unread_count": count})
}

type sendNotificationRequest struct {
	CustomerID string `json:"customer_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Data       string `json:"data"`
}

// SendNotification sends a push to a specific customer (admin endpoint).
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var customerID, title, body, data, fileURL string
	var tempFilePath string

	if ct := c.Get("Content-Type"); len(ct) >= 19 && ct[:19] == "multipart/form-data" {
		customerID = c.FormValue("customer_id")
		title = c.FormValue("title")
		body = c.FormValue("body")
		data = c.FormValue("data")

		if file, err := c.FormFile("file"); err == nil {
			tempDir := "./storage/temp"
			_ = os.MkdirAll(tempDir, 0755)
			ext := filepath.Ext(file.Filename)
			fileName := uuid.New().String() + ext
			tempFilePath = filepath.Join(tempDir, fileName)
			if err := c.SaveFile(file, tempFilePath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file sementara"})
			}
			fileURL = fmt.Sprintf("%s/temp/%s", h.baseURL, fileName)
		}
	} else {
		var req sendNotificationRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
		}
		customerID = req.CustomerID
		title = req.Title
		body = req.Body
		data = req.Data
	}

	if customerID == "" || title == "" || body == "" {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id, judul, dan isi wajib diisi"})
	}

	if err := h.notifService.SendToCustomer(c.Context(), tenantID, customerID, title, body, data, fileURL); err != nil {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		if errors.Is(err, service.ErrCustomerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pelanggan tidak ditemukan"})
		}
		if errors.Is(err, service.ErrInvalidNotificationData) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data harus berupa JSON valid"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim notifikasi"})
	}

	if tempFilePath != "" {
		_ = os.Remove(tempFilePath)
	}

	return c.JSON(fiber.Map{"message": "Notifikasi terkirim"})
}

type broadcastNotificationRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Data   string `json:"data"`
	Target string `json:"target"`
}

// BroadcastNotification sends a push to all customers in tenant (admin endpoint).
func (h *NotificationHandler) BroadcastNotification(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var title, body, data, fileURL, target string
	var tempFilePath string

	if ct := c.Get("Content-Type"); len(ct) >= 19 && ct[:19] == "multipart/form-data" {
		title = c.FormValue("title")
		body = c.FormValue("body")
		data = c.FormValue("data")
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
			fileURL = fmt.Sprintf("%s/temp/%s", h.baseURL, fileName)
		}
	} else {
		var req broadcastNotificationRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
		}
		title = req.Title
		body = req.Body
		data = req.Data
		target = req.Target
	}

	if title == "" || body == "" {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Judul dan isi wajib diisi"})
	}

	var cleanup func()
	if tempFilePath != "" {
		tp := tempFilePath
		cleanup = func() { _ = os.Remove(tp) }
	}

	sent, err := h.notifService.BroadcastToTenant(c.Context(), tenantID, title, body, data, fileURL, target, cleanup)
	if err != nil {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		if errors.Is(err, service.ErrInvalidNotificationData) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data harus berupa JSON valid"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim notifikasi broadcast"})
	}

	return c.JSON(fiber.Map{"message": "Broadcast terkirim", "sent": sent})
}
