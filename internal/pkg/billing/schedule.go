package billing

import "time"

// NormalizeBillingType maps frontend billing type names to canonical backend values.
// Canonical values: "fixed" | "cycle"
// Frontend aliases for cycle: "date_range", "monthly"
func NormalizeBillingType(t string) string {
	switch t {
	case "cycle", "date_range", "monthly":
		return "cycle"
	default:
		return "fixed"
	}
}

func NormalizeDay(day int) int {
	if day <= 0 {
		return 1
	}
	if day > 28 {
		return 28
	}
	return day
}

func InvoiceDayFromDueDay(dueDay int) int {
	dueDay = NormalizeDay(dueDay)
	invoiceDay := dueDay - 7
	if invoiceDay <= 0 {
		invoiceDay += 30
	}
	return NormalizeDay(invoiceDay)
}

func CurrentDuePeriod(now time.Time, invoiceDay, dueDay int) (int, int) {
	invoiceDay = normalizedInvoiceDay(invoiceDay, dueDay)
	dueDay = NormalizeDay(dueDay)

	if invoiceDay <= dueDay {
		if now.Day() >= invoiceDay {
			return int(now.Month()), now.Year()
		}
		previousMonth := now.AddDate(0, -1, 0)
		return int(previousMonth.Month()), previousMonth.Year()
	}

	if now.Day() >= invoiceDay {
		nextMonth := now.AddDate(0, 1, 0)
		return int(nextMonth.Month()), nextMonth.Year()
	}

	return int(now.Month()), now.Year()
}

func CycleDates(invoiceDay, dueDay, dueMonth, dueYear int) (time.Time, time.Time) {
	invoiceDay = normalizedInvoiceDay(invoiceDay, dueDay)
	dueDay = NormalizeDay(dueDay)
	dueDate := time.Date(dueYear, time.Month(dueMonth), dueDay, 0, 0, 0, 0, time.UTC)

	invoiceBase := dueDate
	if invoiceDay > dueDay {
		invoiceBase = dueDate.AddDate(0, -1, 0)
	}

	invoiceDate := time.Date(invoiceBase.Year(), invoiceBase.Month(), invoiceDay, 0, 0, 0, 0, time.UTC)
	return invoiceDate, dueDate
}

func CurrentCycleDates(now time.Time, invoiceDay, dueDay int) (time.Time, time.Time) {
	month, year := CurrentDuePeriod(now, invoiceDay, dueDay)
	return CycleDates(invoiceDay, dueDay, month, year)
}

func normalizedInvoiceDay(invoiceDay, dueDay int) int {
	if invoiceDay <= 0 {
		return InvoiceDayFromDueDay(dueDay)
	}
	return NormalizeDay(invoiceDay)
}