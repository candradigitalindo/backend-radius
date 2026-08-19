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

// Kasus pengaduan NET.id 19 Agu 2026: pelanggan join 17 Agu (cycle 28/2),
// halaman pelanggan menampilkan invoice 28 Jul / jatuh tempo 2 Agu (sudah
// lewat). Tanpa invoice berjalan, tampilan harus maju ke siklus mendatang:
// invoice 28 Agu, jatuh tempo 2 Sep.
func TestUpcomingCycleDatesSkipsPastDue(t *testing.T) {
	now := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC)
	invoiceDate, dueDate := UpcomingCycleDates(now, 28, 2)
	if want := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC); !invoiceDate.Equal(want) {
		t.Fatalf("invoiceDate = %v, want %v", invoiceDate, want)
	}
	if want := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC); !dueDate.Equal(want) {
		t.Fatalf("dueDate = %v, want %v", dueDate, want)
	}

	// Jatuh tempo hari ini: jangan maju — masih siklus berjalan.
	onDueDay := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	_, dueDate = UpcomingCycleDates(onDueDay, 28, 2)
	if want := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC); !dueDate.Equal(want) {
		t.Fatalf("dueDate pada hari-H = %v, want %v", dueDate, want)
	}

	// Jatuh tempo masih di depan: identik dengan CurrentCycleDates.
	before := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, dueDate = UpcomingCycleDates(before, 28, 2)
	if want := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC); !dueDate.Equal(want) {
		t.Fatalf("dueDate sebelum jatuh tempo = %v, want %v", dueDate, want)
	}

	// Mode fixed per join date: join 13 Agu (invoice 6, due 13) dilihat 19 Agu
	// harus menampilkan siklus September.
	_, dueDate = UpcomingCycleDates(now, 6, 13)
	if want := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC); !dueDate.Equal(want) {
		t.Fatalf("dueDate fixed = %v, want %v", dueDate, want)
	}
}
