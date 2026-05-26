package main

import (
	"context"
	"fmt"
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
	reminderRepo := repository.NewReminderRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	waClient := whatsapp.NewClient(cfg.WhatsApp)

	invoiceSvc := service.NewInvoiceService(invoiceRepo, paymentRepo, customerRepo)
	invoiceSvc.WithWAClient(waClient).WithReminderRepo(reminderRepo)
	
	customerSvc := service.NewCustomerService(customerRepo, sessionRepo)

	ctx := context.Background()
	
	// Target customers
	customerIDs := []string{
		"01KNM0AH7Q6PJZZB3B65YPDJV3", // M. Arasyid
		"01KNP2NHDJQ0WVAEQEW4VNEM1Y", // Mushollah Adz Dzakirin
	}
	tenantID := "01KNB5AZ41SE1TJ0JNB265FYFW"

	for _, cid := range customerIDs {
		// 1. Generate for May 2026 (Due early May)
		now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
		count, err := invoiceSvc.GenerateScheduled(ctx, tenantID, now)
		if err != nil {
			log.Printf("Failed to generate for %s: %v", cid, err)
		} else {
			log.Printf("Generated %d invoices for tenant", count)
		}
	}

	// 2. Trigger Isolir again to catch these new late invoices
	tenant, _ := tenantRepo.FindByID(ctx, tenantID)
	cutoff := time.Now().AddDate(0, 0, -tenant.GracePeriod)
	cutoffStr := cutoff.Format("2006-01-02")
	
	invoices, _ := invoiceRepo.ListOverdueForIsolir(ctx, tenantID, cutoffStr)
	for _, inv := range invoices {
		if err := customerSvc.Isolate(ctx, tenantID, inv.CustomerID); err == nil {
			log.Printf("Isolated customer %s", inv.CustomerID)
		}
	}
}
