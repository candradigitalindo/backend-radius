package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) GetRevenueReport(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetRevenueReport(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get revenue report"})
	}

	return c.JSON(fiber.Map{"data": data, "year": year})
}

func (h *ReportHandler) GetCustomerGrowth(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetCustomerGrowth(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get customer growth report"})
	}

	return c.JSON(fiber.Map{"data": data, "year": year})
}

func (h *ReportHandler) GetPaymentBreakdown(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetPaymentMethodBreakdown(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get payment breakdown"})
	}

	return c.JSON(fiber.Map{"data": data, "month": month, "year": year})
}

func (h *ReportHandler) GetCollectionRate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetCollectionRate(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get collection rate"})
	}

	return c.JSON(data)
}

func (h *ReportHandler) GetProfitLoss(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetProfitLoss(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get profit/loss report"})
	}

	return c.JSON(data)
}

func (h *ReportHandler) GetVoucherSales(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetVoucherSalesReport(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get voucher sales report"})
	}

	return c.JSON(data)
}
