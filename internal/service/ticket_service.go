package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrTicketNotFound      = errors.New("Tiket tidak ditemukan")
	ErrTicketAlreadyClosed = errors.New("Tiket sudah ditutup")
)

type TicketService struct {
	ticketRepo repository.TicketRepository
}

func NewTicketService(ticketRepo repository.TicketRepository) *TicketService {
	return &TicketService{ticketRepo: ticketRepo}
}

type CreateTicketInput struct {
	TenantID    string
	CustomerID  string
	Subject     string
	Description string
	Category    string
	Priority    string
}

type UpdateTicketInput struct {
	Subject     string
	Description string
	Category    string
	Priority    string
}

type AddMessageInput struct {
	TenantID      string
	TicketID      string
	SenderType    string
	SenderID      string
	Message       string
	AttachmentURL string
}

func (s *TicketService) Create(ctx context.Context, input CreateTicketInput) (*model.Ticket, error) {
	if input.Category == "" {
		input.Category = "general"
	}
	if input.Priority == "" {
		input.Priority = "medium"
	}

	ticketNumber := fmt.Sprintf("TK-%s", time.Now().Format("20060102150405"))

	ticket := &model.Ticket{
		TenantID:     input.TenantID,
		CustomerID:   input.CustomerID,
		TicketNumber: ticketNumber,
		Subject:      input.Subject,
		Description:  input.Description,
		Category:     input.Category,
		Priority:     input.Priority,
		Status:       "open",
	}

	if err := s.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(ctx, input.TenantID, ticket.ID)
}

func (s *TicketService) GetByID(ctx context.Context, tenantID, ticketID string) (*model.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}

	messages, err := s.ticketRepo.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	ticket.Messages = messages

	return ticket, nil
}

func (s *TicketService) Update(ctx context.Context, tenantID, ticketID string, input UpdateTicketInput) (*model.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	if ticket.Status == "closed" {
		return nil, ErrTicketAlreadyClosed
	}

	ticket.Subject = input.Subject
	ticket.Description = input.Description
	ticket.Category = input.Category
	ticket.Priority = input.Priority

	if err := s.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(ctx, tenantID, ticketID)
}

func (s *TicketService) UpdateStatus(ctx context.Context, tenantID, ticketID, status string) error {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return ErrTicketNotFound
	}
	if ticket.Status == "closed" {
		return ErrTicketAlreadyClosed
	}
	return s.ticketRepo.UpdateStatus(ctx, tenantID, ticketID, status)
}

func (s *TicketService) Assign(ctx context.Context, tenantID, ticketID, userID string) error {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return ErrTicketNotFound
	}
	return s.ticketRepo.Assign(ctx, tenantID, ticketID, userID)
}

func (s *TicketService) Delete(ctx context.Context, tenantID, ticketID string) error {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return ErrTicketNotFound
	}
	return s.ticketRepo.Delete(ctx, tenantID, ticketID)
}

func (s *TicketService) List(ctx context.Context, tenantID string, filter repository.TicketFilter) ([]model.Ticket, int, error) {
	return s.ticketRepo.List(ctx, tenantID, filter)
}

func (s *TicketService) AddMessage(ctx context.Context, input AddMessageInput) (*model.TicketMessage, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, input.TenantID, input.TicketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	if ticket.Status == "closed" {
		return nil, ErrTicketAlreadyClosed
	}

	msg := &model.TicketMessage{
		TicketID:      input.TicketID,
		SenderType:    input.SenderType,
		SenderID:      input.SenderID,
		Message:       input.Message,
		AttachmentURL: input.AttachmentURL,
	}

	if err := s.ticketRepo.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Reopen ticket if customer replies to a resolved ticket
	if input.SenderType == "customer" && ticket.Status == "resolved" {
		_ = s.ticketRepo.UpdateStatus(ctx, input.TenantID, input.TicketID, "open")
	}

	return msg, nil
}

func (s *TicketService) GetMessages(ctx context.Context, tenantID, ticketID string) ([]model.TicketMessage, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	return s.ticketRepo.ListMessages(ctx, ticketID)
}
