package main

import (
	"context"
	"fmt"
	"log"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

func main() {
	cfg, _ := config.Load()
	db := database.Connect(cfg)
	defer db.Close()

	invoiceRepo := repository.NewInvoiceRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	waClient := whatsapp.NewClient(cfg.WhatsApp)

	ctx := context.Background()
	tenants, _ := tenantRepo.ListActive(ctx)

	monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

	for _, t := range tenants {
		log.Printf("Processing reminders for tenant %s (%s)...", t.ID, t.Name)
		
		invoices, _, err := invoiceRepo.List(ctx, t.ID, repository.InvoiceFilter{
			Status: "unpaid",
			Page: 1,
			PerPage: 1000,
		})
		if err != nil {
			log.Printf("Failed to list invoices for %s: %v", t.ID, err)
			continue
		}

		log.Printf("Found %d unpaid invoices for tenant %s", len(invoices), t.ID)

		var items []whatsapp.ReminderItem
		for _, inv := range invoices {
			if inv.Customer == nil {
				log.Printf("Invoice %s has no customer data", inv.InvoiceNumber)
				continue
			}
			if inv.Customer.Phone == "" {
				log.Printf("Customer %s has no phone number", inv.Customer.Name)
				continue
			}

			periode := fmt.Sprintf("%s %d", monthNames[inv.PeriodMonth], inv.PeriodYear)
			
			msg := fmt.Sprintf("Halo %s, ini adalah pengingat untuk tagihan internet Anda.\n\n"+
				"No. Invoice: %s\n"+
				"Periode: %s\n"+
				"Total: Rp%d\n"+
				"Jatuh Tempo: %s\n\n"+
				"Mohon segera lakukan pembayaran. Jika sudah membayar, abaikan pesan ini. Terima kasih.",
				inv.Customer.Name, inv.InvoiceNumber, periode, inv.TotalAmount, inv.DueDate.Format("02/01/2006"))

			items = append(items, whatsapp.ReminderItem{
				Phone:         inv.Customer.Phone,
				CustomerName:  inv.Customer.Name,
				InvoiceNumber: inv.InvoiceNumber,
				Amount:        inv.TotalAmount,
				DueDate:       inv.DueDate.Format("02/01/2006"),
				Message:       msg,
			})
		}

		if len(items) > 0 {
			log.Printf("Sending %d reminders for tenant %s...", len(items), t.ID)
			result, err := waClient.SendReminders(ctx, t.ID, items)
			if err != nil {
				log.Printf("Failed to send reminders for tenant %s: %v", t.ID, err)
			} else {
				log.Printf("Sent %d reminders for tenant %s (%d failed)", result.Success, t.ID, result.Failed)
			}
		}
	}
}
