package billing

import (
	"testing"
	"time"
)

func TestCurrentDuePeriodUsesExplicitInvoiceDateBoundary(t *testing.T) {
	month, year := CurrentDuePeriod(time.Date(2026, time.May, 24, 8, 0, 0, 0, time.UTC), 24, 1)
	if month != int(time.June) || year != 2026 {
		t.Fatalf("CurrentDuePeriod returned (%d,%d), want (%d,%d)", month, year, int(time.June), 2026)
	}
}

func TestCurrentCycleDatesCrossMonthWindow(t *testing.T) {
	invoiceDate, dueDate := CurrentCycleDates(time.Date(2026, time.May, 24, 8, 0, 0, 0, time.UTC), 24, 1)

	wantInvoiceDate := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)
	wantDueDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	if !invoiceDate.Equal(wantInvoiceDate) {
		t.Fatalf("invoiceDate = %v, want %v", invoiceDate, wantInvoiceDate)
	}
	if !dueDate.Equal(wantDueDate) {
		t.Fatalf("dueDate = %v, want %v", dueDate, wantDueDate)
	}
}
