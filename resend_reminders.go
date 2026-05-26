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
	customerRepo := repository.NewCustomerRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	reminderRepo := repository.NewReminderRepository(db)
	waClient := whatsapp.NewClient(cfg.WhatsApp)

	reminderSvc := service.NewReminderService(reminderRepo, invoiceRepo, customerRepo, tenantRepo, waClient)
	
	ctx := context.Background()
	tenants, _ := tenantRepo.ListActive(ctx)

	for _, t := range tenants {
		log.Printf("Processing reminders for tenant %s...", t.ID)
		
		// 1. Manually trigger standard reminders (H-1, Today, etc)
		result, err := reminderSvc.TriggerReminders(ctx, t.ID)
		if err != nil {
			log.Printf("Failed to trigger reminders for %s: %v", t.ID, err)
		} else {
			log.Printf("Trigger result for %s: %s", t.ID, result.Message)
		}

		// 2. Custom check for ANY unpaid past-due invoices to send a "Late Payment" alert
		// This is for the "cek reminder yang terlambat" part.
		invoices, _ := invoiceRepo.List(ctx, t.ID, repository.InvoiceFilter{
			Status: "unpaid",
			Page: 1,
			PerPage: 1000,
		})

		var lateInvoices []repository.InvoiceRepository // Wait, I need model.Invoice
	}
}
