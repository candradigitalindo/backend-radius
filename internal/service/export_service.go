package service

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type ExportService struct {
	reportRepo repository.ReportRepository
}

func NewExportService(reportRepo repository.ReportRepository) *ExportService {
	return &ExportService{reportRepo: reportRepo}
}

// ---------- Excel Export ----------

func (s *ExportService) ExportRevenueExcel(data []repository.MonthlyRevenueReport, year int) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Revenue"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Bulan", "Revenue (Rp)", "Pengeluaran (Rp)", "Profit (Rp)", "Invoice Terbayar", "Total Invoice"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheet, cell, h)
	}
	s.styleHeader(f, sheet, len(headers))

	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	for i, d := range data {
		row := i + 2
		monthName := ""
		if d.Month >= 1 && d.Month <= 12 {
			monthName = months[d.Month]
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), monthName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), d.Revenue)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), d.Expenses)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), d.Profit)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), d.InvoicesPaid)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), d.InvoicesTotal)
	}

	buf := &bytes.Buffer{}
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *ExportService) ExportCustomerGrowthExcel(data []repository.MonthlyCustomerGrowth, year int) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Customer Growth"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Bulan", "Pelanggan Baru", "Total Aktif", "Total Semua", "Churn"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheet, cell, h)
	}
	s.styleHeader(f, sheet, len(headers))

	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	for i, d := range data {
		row := i + 2
		monthName := ""
		if d.Month >= 1 && d.Month <= 12 {
			monthName = months[d.Month]
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), monthName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), d.NewJoined)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), d.TotalActive)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), d.TotalAll)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), d.Churned)
	}

	buf := &bytes.Buffer{}
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *ExportService) ExportProfitLossExcel(data *repository.ProfitLossStat, month, year int) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Profit Loss"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Keterangan", "Jumlah (Rp)"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheet, cell, h)
	}
	s.styleHeader(f, sheet, len(headers))

	rows := []struct {
		label  string
		amount int64
	}{
		{"Pendapatan Invoice", data.Revenue},
		{"Penjualan Voucher", data.VoucherSales},
		{"Total Pengeluaran", data.Expenses},
		{"Profit Bersih", data.Profit},
		{"Grand Total", data.GrandTotal},
	}

	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.amount)
	}

	buf := &bytes.Buffer{}
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *ExportService) styleHeader(f *excelize.File, sheet string, cols int) {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i := 0; i < cols; i++ {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

// ---------- PDF Export ----------

func (s *ExportService) ExportRevenuePDF(data []repository.MonthlyRevenueReport, year int) (*bytes.Buffer, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, fmt.Sprintf("Laporan Revenue Tahun %d", year), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 5, fmt.Sprintf("Dicetak: %s", time.Now().Format("02 Jan 2006 15:04")), "", 1, "R", false, 0, "")
	pdf.Ln(5)

	// Table header
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	widths := []float64{40, 45, 45, 45, 40, 40}
	headers := []string{"Bulan", "Revenue", "Pengeluaran", "Profit", "Inv. Terbayar", "Total Invoice"}
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table body
	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetTextColor(0, 0, 0)
	months := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	for i, d := range data {
		fill := i%2 == 0
		monthName := ""
		if d.Month >= 1 && d.Month <= 12 {
			monthName = months[d.Month]
		}
		pdf.CellFormat(widths[0], 7, monthName, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[1], 7, formatRupiah(d.Revenue), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(widths[2], 7, formatRupiah(d.Expenses), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(widths[3], 7, formatRupiah(d.Profit), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(widths[4], 7, fmt.Sprintf("%d", d.InvoicesPaid), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[5], 7, fmt.Sprintf("%d", d.InvoicesTotal), "1", 0, "C", fill, 0, "")
		pdf.Ln(-1)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *ExportService) ExportCustomerGrowthPDF(data []repository.MonthlyCustomerGrowth, year int) (*bytes.Buffer, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, fmt.Sprintf("Laporan Pertumbuhan Pelanggan Tahun %d", year), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 5, fmt.Sprintf("Dicetak: %s", time.Now().Format("02 Jan 2006 15:04")), "", 1, "R", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	widths := []float64{50, 45, 45, 45, 45}
	headers := []string{"Bulan", "Pelanggan Baru", "Total Aktif", "Total Semua", "Churn"}
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetTextColor(0, 0, 0)
	months := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	for i, d := range data {
		fill := i%2 == 0
		monthName := ""
		if d.Month >= 1 && d.Month <= 12 {
			monthName = months[d.Month]
		}
		pdf.CellFormat(widths[0], 7, monthName, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[1], 7, fmt.Sprintf("%d", d.NewJoined), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[2], 7, fmt.Sprintf("%d", d.TotalActive), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[3], 7, fmt.Sprintf("%d", d.TotalAll), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(widths[4], 7, fmt.Sprintf("%d", d.Churned), "1", 0, "C", fill, 0, "")
		pdf.Ln(-1)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *ExportService) ExportProfitLossPDF(data *repository.ProfitLossStat, month, year int) (*bytes.Buffer, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, fmt.Sprintf("Laporan Laba Rugi - %d/%d", month, year), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 5, fmt.Sprintf("Dicetak: %s", time.Now().Format("02 Jan 2006 15:04")), "", 1, "R", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(68, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(100, 8, "Keterangan", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 8, "Jumlah (Rp)", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 10)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetTextColor(0, 0, 0)

	rows := []struct {
		label  string
		amount int64
		bold   bool
	}{
		{"Pendapatan Invoice", data.Revenue, false},
		{"Penjualan Voucher", data.VoucherSales, false},
		{"Total Pengeluaran", data.Expenses, false},
		{"Profit Bersih", data.Profit, true},
		{"Grand Total", data.GrandTotal, true},
	}

	for i, r := range rows {
		fill := i%2 == 0
		if r.bold {
			pdf.SetFont("Arial", "B", 10)
		} else {
			pdf.SetFont("Arial", "", 10)
		}
		pdf.CellFormat(100, 7, r.label, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(60, 7, formatRupiah(r.amount), "1", 0, "R", fill, 0, "")
		pdf.Ln(-1)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func formatRupiah(amount int64) string {
	neg := ""
	if amount < 0 {
		neg = "-"
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return neg + s
	}
	var result []byte
	for i, c := range s {
		if (n-i)%3 == 0 && i != 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return neg + string(result)
}
