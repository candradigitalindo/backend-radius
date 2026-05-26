package main

import (
	"context"
	"log"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

func main() {
	cfg, _ := config.Load()
	db := database.Connect(cfg)
	defer db.Close()

	invoiceRepo := repository.NewInvoiceRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	waClient := whatsapp.NewClient(cfg.WhatsApp)

	invoiceSvc := service.NewInvoiceService(invoiceRepo, paymentRepo, customerRepo)
	invoiceSvc.WithWAClient(waClient)
	customerSvc := service.NewCustomerService(customerRepo, sessionRepo)

	ctx := context.Background()
	tenantID := "01KNB5AZ41SE1TJ0JNB265FYFW"

	// Try multiple 'now' dates to catch different billing cycles in May
	checkDates := []time.Time{
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range checkDates {
		count, err := invoiceSvc.GenerateScheduled(ctx, tenantID, d)
		if err != nil {
			log.Printf("Error generating for %v: %v", d, err)
		} else if count > 0 {
			log.Printf("Generated %d invoices using date %v", count, d)
		}
	}

	// Now isolate anyone who is late
	tenant, _ := tenantRepo.FindByID(ctx, tenantID)
	cutoff := time.Now().AddDate(0, 0, -tenant.GracePeriod)
	cutoffStr := cutoff.Format("2006-01-02")
	
	invoices, _ := invoiceRepo.ListOverdueForIsolir(ctx, tenantID, cutoffStr)
	for _, inv := range invoices {
		if err := customerSvc.Isolate(ctx, tenantID, inv.CustomerID); err == nil {
			log.Printf("Isolated customer %s (Invoice %s, Due %v)", inv.CustomerID, inv.InvoiceNumber, inv.DueDate)
		}
	}
}
