package service

import (
	"testing"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/billing"
)

// Skenario insiden NET.id 15-16 Agu 2026: pelanggan diimpor 14 Agu dengan
// billing cycle invoiceDay=28/dueDay=2. Worker 15 Agu membuat invoice periode
// Agustus dengan due 2 Agu (sudah lewat 13 hari) dan auto-isolir memutus 77
// pelanggan keesokan paginya. Guard joinedAfterCycleStart harus men-skip
// pelanggan yang bergabung setelah tanggal invoice siklus berjalan.
func TestJoinedAfterCycleStart_NETidIncident(t *testing.T) {
	now := time.Date(2026, 8, 15, 1, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		invoiceDay int
		dueDay     int
		join       time.Time
		wantSkip   bool
	}{
		{"impor tengah siklus (join 1 Agu, cycle 28/2)", 28, 2, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), true},
		{"impor tengah siklus (join 14 Agu, cycle 28/2)", 28, 2, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), true},
		{"fixed join 1 Agu, billing_date 24 due 1", 24, 1, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), true},
		{"fixed join 13 Agu, billing_date 6 due 13", 6, 13, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), true},
		{"pelanggan lama (join 20 Jul, cycle 28/2)", 28, 2, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), false},
		{"join tepat di tanggal invoice", 28, 2, time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			month, year := billing.CurrentDuePeriod(now, tc.invoiceDay, tc.dueDay)
			invoiceDate, dueDate := billing.CycleDates(tc.invoiceDay, tc.dueDay, month, year)
			if now.Before(invoiceDate) {
				t.Fatalf("skenario tidak valid: worker belum mencapai tanggal invoice %v", invoiceDate)
			}

			got := joinedAfterCycleStart(model.Customer{JoinDate: tc.join}, invoiceDate)
			if got != tc.wantSkip {
				t.Errorf("joinedAfterCycleStart(join=%s, invoiceDate=%s) = %v, want %v (due %s)",
					tc.join.Format("2006-01-02"), invoiceDate.Format("2006-01-02"), got, tc.wantSkip,
					dueDate.Format("2006-01-02"))
			}
		})
	}

	// JoinDate kosong → fallback ke CreatedAt.
	created := time.Date(2026, 8, 14, 6, 33, 0, 0, time.UTC)
	invoiceDate := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	if !joinedAfterCycleStart(model.Customer{CreatedAt: created}, invoiceDate) {
		t.Error("JoinDate kosong harus fallback ke CreatedAt")
	}

	// Siklus berikutnya (worker 28 Agu): pelanggan impor Agustus sudah ikut ditagih, due 2 Sep.
	nextRun := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	month, year := billing.CurrentDuePeriod(nextRun, 28, 2)
	nextInvoiceDate, nextDue := billing.CycleDates(28, 2, month, year)
	if joinedAfterCycleStart(model.Customer{JoinDate: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)}, nextInvoiceDate) {
		t.Errorf("pelanggan join 14 Agu harus ditagih pada siklus 28 Agu (invoiceDate %s)", nextInvoiceDate.Format("2006-01-02"))
	}
	if want := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC); !nextDue.Equal(want) {
		t.Errorf("due siklus berikutnya = %s, want 2026-09-02", nextDue.Format("2006-01-02"))
	}
}
