package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type ExportHandler struct {
	reportService *service.ReportService
	exportService *service.ExportService
}

func NewExportHandler(reportService *service.ReportService, exportService *service.ExportService) *ExportHandler {
	return &ExportHandler{reportService: reportService, exportService: exportService}
}

func (h *ExportHandler) ExportRevenueExcel(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetRevenueReport(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get revenue report"})
	}

	buf, err := h.exportService.ExportRevenueExcel(data, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate Excel"})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=revenue_%d.xlsx", year))
	return c.Send(buf.Bytes())
}

func (h *ExportHandler) ExportRevenuePDF(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetRevenueReport(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get revenue report"})
	}

	buf, err := h.exportService.ExportRevenuePDF(data, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate PDF"})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=revenue_%d.pdf", year))
	return c.Send(buf.Bytes())
}

func (h *ExportHandler) ExportCustomerGrowthExcel(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetCustomerGrowth(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get customer growth"})
	}

	buf, err := h.exportService.ExportCustomerGrowthExcel(data, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate Excel"})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=customer_growth_%d.xlsx", year))
	return c.Send(buf.Bytes())
}

func (h *ExportHandler) ExportCustomerGrowthPDF(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetCustomerGrowth(c.Context(), tenantID, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get customer growth"})
	}

	buf, err := h.exportService.ExportCustomerGrowthPDF(data, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate PDF"})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=customer_growth_%d.pdf", year))
	return c.Send(buf.Bytes())
}

func (h *ExportHandler) ExportProfitLossExcel(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetProfitLoss(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get profit/loss"})
	}

	buf, err := h.exportService.ExportProfitLossExcel(data, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate Excel"})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=profit_loss_%d_%d.xlsx", month, year))
	return c.Send(buf.Bytes())
}

func (h *ExportHandler) ExportProfitLossPDF(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	data, err := h.reportService.GetProfitLoss(c.Context(), tenantID, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get profit/loss"})
	}

	buf, err := h.exportService.ExportProfitLossPDF(data, month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate PDF"})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=profit_loss_%d_%d.pdf", month, year))
	return c.Send(buf.Bytes())
}
