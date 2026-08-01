package worker

import "testing"

func TestParseDailyCron(t *testing.T) {
	cases := []struct {
		expr       string
		wantHour   int
		wantMinute int
		wantOK     bool
	}{
		{"0 6 * * *", 6, 0, true},
		{"30 9 * * *", 9, 30, true},
		{"0 8 * * *", 8, 0, true},
		{"*/5 * * * *", 0, 0, false},  // interval cron, bukan harian sederhana
		{"0 6 * * 1", 0, 0, false},    // dibatasi hari kerja tertentu
		{"0 6 1 * *", 0, 0, false},    // dibatasi tanggal tertentu
		{"invalid", 0, 0, false},
		{"a b * * *", 0, 0, false},
	}
	for _, c := range cases {
		hour, minute, ok := parseDailyCron(c.expr)
		if ok != c.wantOK {
			t.Fatalf("parseDailyCron(%q): ok=%v, want %v", c.expr, ok, c.wantOK)
		}
		if ok && (hour != c.wantHour || minute != c.wantMinute) {
			t.Fatalf("parseDailyCron(%q) = %d:%d, want %d:%d", c.expr, hour, minute, c.wantHour, c.wantMinute)
		}
	}
}
