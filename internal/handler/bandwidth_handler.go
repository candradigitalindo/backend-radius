package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type BandwidthHandler struct {
	bandwidthService *service.BandwidthService
	sessionRepo      repository.SessionRepository
}

func NewBandwidthHandler(bandwidthService *service.BandwidthService, sessionRepo repository.SessionRepository) *BandwidthHandler {
	return &BandwidthHandler{bandwidthService: bandwidthService, sessionRepo: sessionRepo}
}

func (h *BandwidthHandler) GetCustomerUsage(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.bandwidthService.GetCustomerUsage(c.Context(), tenantID, customerID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get bandwidth usage"})
	}

	return c.JSON(fiber.Map{"data": data, "month": month, "year": year})
}

func (h *BandwidthHandler) GetCustomerUsageHistory(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.bandwidthService.GetCustomerUsageHistory(c.Context(), tenantID, customerID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get usage history"})
	}

	return c.JSON(fiber.Map{"data": data, "year": year})
}

func (h *BandwidthHandler) GetTopUsers(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	data, err := h.bandwidthService.GetTopUsers(c.Context(), tenantID, month, year, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get top users"})
	}

	return c.JSON(fiber.Map{"data": data, "month": month, "year": year})
}

func (h *BandwidthHandler) GetUsageSummary(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.bandwidthService.GetUsageSummary(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get usage summary"})
	}

	return c.JSON(fiber.Map{"data": data, "month": month, "year": year})
}

func (h *BandwidthHandler) GetSaturationReport(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	hours, _ := strconv.Atoi(c.Query("hours", "24"))
	threshold, _ := strconv.Atoi(c.Query("threshold", "80"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	data, err := h.bandwidthService.GetSaturationReport(c.Context(), tenantID, hours, threshold, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get saturation report"})
	}

	return c.JSON(fiber.Map{"data": data, "hours": hours, "threshold_pct": threshold})
}

func (h *BandwidthHandler) GetCustomerConnections(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	sessions, total, err := h.sessionRepo.ListByCustomer(c.Context(), tenantID, customerID, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get connection history"})
	}

	type connectionItem struct {
		ID             string  `json:"id"`
		SessionID      string  `json:"session_id"`
		IP             string  `json:"ip"`
		StartedAt      string  `json:"started_at"`
		EndedAt        *string `json:"ended_at"`
		Download       int64   `json:"download"`
		Upload         int64   `json:"upload"`
		Uptime         int     `json:"uptime"`
		Status         string  `json:"status"`
		TerminateCause string  `json:"terminate_cause,omitempty"`
	}

	fallbackSessionIDs := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if s.SessionID == "" {
			continue
		}
		if s.InputOctets == 0 || s.OutputOctets == 0 {
			fallbackSessionIDs = append(fallbackSessionIDs, s.SessionID)
		}
	}

	transferTotals := map[string]repository.SessionTransferTotal{}
	if len(fallbackSessionIDs) > 0 {
		var err error
		transferTotals, err = h.bandwidthService.GetSessionTransferTotals(c.Context(), tenantID, fallbackSessionIDs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get connection history"})
		}
	}

	items := make([]connectionItem, 0, len(sessions))
	for _, s := range sessions {
		download := s.OutputOctets
		upload := s.InputOctets
		if fallback, ok := transferTotals[s.SessionID]; ok {
			if download == 0 {
				download = fallback.TotalDownload
			}
			if upload == 0 {
				upload = fallback.TotalUpload
			}
		}

		item := connectionItem{
			ID:             s.ID,
			SessionID:      s.SessionID,
			IP:             s.FramedIP,
			StartedAt:      s.StartedAt.Format(time.RFC3339),
			Download:       download,
			Upload:         upload,
			Uptime:         s.SessionTime,
			Status:         s.Status,
			TerminateCause: s.TerminateCause,
		}
		if s.EndedAt != nil {
			t := s.EndedAt.Format(time.RFC3339)
			item.EndedAt = &t
		}
		items = append(items, item)
	}

	return c.JSON(fiber.Map{
		"data":     items,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
