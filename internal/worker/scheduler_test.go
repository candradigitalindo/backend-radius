package worker

import (
	"testing"
	"time"
)

func TestInvoiceTargetPeriod(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantMonth int
		wantYear  int
	}{
		{
			name:      "crosses into next month",
			now:       time.Date(2026, time.April, 30, 8, 0, 0, 0, time.UTC),
			wantMonth: int(time.May),
			wantYear:  2026,
		},
		{
			name:      "stays in same month",
			now:       time.Date(2026, time.May, 24, 8, 0, 0, 0, time.UTC),
			wantMonth: int(time.May),
			wantYear:  2026,
		},
		{
			name:      "crosses into next year",
			now:       time.Date(2026, time.December, 28, 8, 0, 0, 0, time.UTC),
			wantMonth: int(time.January),
			wantYear:  2027,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMonth, gotYear := invoiceTargetPeriod(tt.now)
			if gotMonth != tt.wantMonth || gotYear != tt.wantYear {
				t.Fatalf("invoiceTargetPeriod(%v) = (%d, %d), want (%d, %d)", tt.now, gotMonth, gotYear, tt.wantMonth, tt.wantYear)
			}
		})
	}
}
