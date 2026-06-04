package service

import (
	"context"
	"testing"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
)

type customerInvoiceRepoStub struct {
	currentPeriodInvoice *model.Invoice
	fallbackInvoice      *model.Invoice
	currentPeriodErr     error
	fallbackErr          error
	fallbackCalls        int
	lastPeriodMonth      int
	lastPeriodYear       int
}

func (s *customerInvoiceRepoStub) FindByCustomerPeriod(ctx context.Context, tenantID, customerID string, month, year int) (*model.Invoice, error) {
	s.lastPeriodMonth = month
	s.lastPeriodYear = year
	return s.currentPeriodInvoice, s.currentPeriodErr
}

func (s *customerInvoiceRepoStub) FindCurrentByCustomer(ctx context.Context, tenantID, customerID string) (*model.Invoice, error) {
	s.fallbackCalls++
	return s.fallbackInvoice, s.fallbackErr
}

func TestLoadCurrentInvoiceUsesUpcomingPeriodWhenInsideHMinus7Window(t *testing.T) {
	repo := &customerInvoiceRepoStub{
		currentPeriodInvoice: &model.Invoice{ID: "inv-current", InvoiceNumber: "202606-001"},
		fallbackInvoice:      &model.Invoice{ID: "inv-fallback", InvoiceNumber: "202605-001"},
	}
	service := &CustomerService{invoiceRepo: repo}
	customer := &model.Customer{
		ID:              "customer-1",
		BillingDeadline: 3,
		JoinDate:        time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	inv, err := service.loadCurrentInvoice(context.Background(), "tenant-1", customer, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("loadCurrentInvoice returned error: %v", err)
	}
	if inv == nil || inv.ID != "inv-current" {
		t.Fatalf("unexpected invoice: %#v", inv)
	}
	if repo.lastPeriodMonth != 6 || repo.lastPeriodYear != 2026 {
		t.Fatalf("expected upcoming period lookup 6/2026, got %d/%d", repo.lastPeriodMonth, repo.lastPeriodYear)
	}
	if repo.fallbackCalls != 0 {
		t.Fatalf("fallback should not be called when H-7 period invoice exists")
	}
}

func TestLoadCurrentInvoiceUsesPreviousPeriodBeforeHMinus7Window(t *testing.T) {
	repo := &customerInvoiceRepoStub{
		currentPeriodInvoice: &model.Invoice{ID: "inv-previous", InvoiceNumber: "202604-001"},
	}
	service := &CustomerService{invoiceRepo: repo}
	customer := &model.Customer{
		ID:              "customer-1",
		BillingDeadline: 20,
		JoinDate:        time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
	}

	inv, err := service.loadCurrentInvoice(context.Background(), "tenant-1", customer, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("loadCurrentInvoice returned error: %v", err)
	}
	if inv == nil || inv.ID != "inv-previous" {
		t.Fatalf("unexpected invoice: %#v", inv)
	}
	if repo.lastPeriodMonth != 4 || repo.lastPeriodYear != 2026 {
		t.Fatalf("expected previous period lookup 4/2026, got %d/%d", repo.lastPeriodMonth, repo.lastPeriodYear)
	}
	if repo.fallbackCalls != 0 {
		t.Fatalf("fallback should not be called when previous billing window invoice exists")
	}
}

func TestLoadCurrentInvoiceFallsBackWhenDerivedPeriodInvoiceMissing(t *testing.T) {
	repo := &customerInvoiceRepoStub{
		fallbackInvoice: &model.Invoice{ID: "inv-fallback", InvoiceNumber: "202604-001", Status: "paid"},
	}
	service := &CustomerService{invoiceRepo: repo}
	customer := &model.Customer{
		ID:              "customer-1",
		BillingDeadline: 14,
		JoinDate:        time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
	}

	inv, err := service.loadCurrentInvoice(context.Background(), "tenant-1", customer, time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("loadCurrentInvoice returned error: %v", err)
	}
	if inv == nil || inv.ID != "inv-fallback" {
		t.Fatalf("unexpected fallback invoice: %#v", inv)
	}
	if repo.lastPeriodMonth != 5 || repo.lastPeriodYear != 2026 {
		t.Fatalf("expected derived period lookup 5/2026, got %d/%d", repo.lastPeriodMonth, repo.lastPeriodYear)
	}
	if repo.fallbackCalls != 1 {
		t.Fatalf("expected one fallback call, got %d", repo.fallbackCalls)
	}
}

func TestLoadCurrentInvoiceUsesStoredBillingDateBoundary(t *testing.T) {
	repo := &customerInvoiceRepoStub{
		currentPeriodInvoice: &model.Invoice{ID: "inv-june", InvoiceNumber: "202606-001"},
	}
	service := &CustomerService{invoiceRepo: repo}
	customer := &model.Customer{
		ID:              "customer-1",
		BillingDate:     24,
		BillingDeadline: 1,
		JoinDate:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	inv, err := service.loadCurrentInvoice(context.Background(), "tenant-1", customer, time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("loadCurrentInvoice returned error: %v", err)
	}
	if inv == nil || inv.ID != "inv-june" {
		t.Fatalf("unexpected invoice: %#v", inv)
	}
	if repo.lastPeriodMonth != 6 || repo.lastPeriodYear != 2026 {
		t.Fatalf("expected due period lookup 6/2026, got %d/%d", repo.lastPeriodMonth, repo.lastPeriodYear)
	}
}

func TestApplyUpdatedCustomerBillingScheduleUsesExplicitInvoiceAndDueDates(t *testing.T) {
	joinDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	invoiceDate := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	customer := &model.Customer{
		JoinDate:        joinDate,
		BillingDate:     10,
		BillingDeadline: 17,
	}

	applyUpdatedCustomerBillingSchedule(customer, UpdateCustomerServiceInput{
		JoinDate:       &joinDate,
		InvoiceDate:    &invoiceDate,
		BillingDueDate: &dueDate,
	})

	if customer.BillingDate != 24 {
		t.Fatalf("customer.BillingDate = %d, want 24", customer.BillingDate)
	}
	if customer.BillingDeadline != 1 {
		t.Fatalf("customer.BillingDeadline = %d, want 1", customer.BillingDeadline)
	}
}

func TestApplyUpdatedCustomerBillingScheduleFallsBackToJoinDateWhenExplicitDatesMissing(t *testing.T) {
	joinDate := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	customer := &model.Customer{}

	applyUpdatedCustomerBillingSchedule(customer, UpdateCustomerServiceInput{
		JoinDate: &joinDate,
	})

	if !customer.JoinDate.Equal(joinDate) {
		t.Fatalf("customer.JoinDate = %v, want %v", customer.JoinDate, joinDate)
	}
	if customer.BillingDeadline != 3 {
		t.Fatalf("customer.BillingDeadline = %d, want 3", customer.BillingDeadline)
	}
	if customer.BillingDate != 26 {
		t.Fatalf("customer.BillingDate = %d, want 26", customer.BillingDate)
	}
}
