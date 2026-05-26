package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket *model.Ticket) error
	FindByID(ctx context.Context, tenantID, ticketID string) (*model.Ticket, error)
	Update(ctx context.Context, ticket *model.Ticket) error
	UpdateStatus(ctx context.Context, tenantID, ticketID, status string) error
	Assign(ctx context.Context, tenantID, ticketID, userID string) error
	Delete(ctx context.Context, tenantID, ticketID string) error
	List(ctx context.Context, tenantID string, filter TicketFilter) ([]model.Ticket, int, error)
	AddMessage(ctx context.Context, msg *model.TicketMessage) error
	ListMessages(ctx context.Context, ticketID string) ([]model.TicketMessage, error)
}

type TicketFilter struct {
	Search     string
	CustomerID string
	Status     string
	Category   string
	Priority   string
	AssignedTo string
	Page       int
	PerPage    int
}

type ticketRepository struct {
	db *pgxpool.Pool
}

func NewTicketRepository(db *pgxpool.Pool) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ctx context.Context, ticket *model.Ticket) error {
	ticket.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO tickets (
			id, tenant_id, customer_id, ticket_number, subject, description,
			category, priority, status, assigned_to
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`,
		ticket.ID, ticket.TenantID, ticket.CustomerID, ticket.TicketNumber,
		ticket.Subject, ticket.Description, ticket.Category, ticket.Priority,
		ticket.Status, ticket.AssignedTo,
	)
	return err
}

func (r *ticketRepository) FindByID(ctx context.Context, tenantID, ticketID string) (*model.Ticket, error) {
	var t model.Ticket
	var custName, custCode string
	err := r.db.QueryRow(ctx, `
		SELECT t.id, t.tenant_id, t.customer_id, t.ticket_number, t.subject, COALESCE(t.description,''),
		       t.category, t.priority, t.status, t.assigned_to, t.resolved_at, t.closed_at,
		       t.created_at, t.updated_at,
		       c.name, c.customer_code
		FROM tickets t
		JOIN customers c ON c.id = t.customer_id
		WHERE t.id = $1 AND t.tenant_id = $2
	`, ticketID, tenantID).Scan(
		&t.ID, &t.TenantID, &t.CustomerID, &t.TicketNumber, &t.Subject, &t.Description,
		&t.Category, &t.Priority, &t.Status, &t.AssignedTo, &t.ResolvedAt, &t.ClosedAt,
		&t.CreatedAt, &t.UpdatedAt,
		&custName, &custCode,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.Customer = &model.Customer{ID: t.CustomerID, Name: custName, CustomerCode: custCode}
	return &t, nil
}

func (r *ticketRepository) Update(ctx context.Context, ticket *model.Ticket) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tickets SET
			subject = $1, description = $2, category = $3, priority = $4, updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6
	`,
		ticket.Subject, ticket.Description, ticket.Category, ticket.Priority,
		ticket.ID, ticket.TenantID,
	)
	return err
}

func (r *ticketRepository) UpdateStatus(ctx context.Context, tenantID, ticketID, status string) error {
	var extra string
	switch status {
	case "resolved":
		extra = ", resolved_at = NOW()"
	case "closed":
		extra = ", closed_at = NOW()"
	}
	query := fmt.Sprintf("UPDATE tickets SET status = $1%s, updated_at = NOW() WHERE id = $2 AND tenant_id = $3", extra)
	_, err := r.db.Exec(ctx, query, status, ticketID, tenantID)
	return err
}

func (r *ticketRepository) Assign(ctx context.Context, tenantID, ticketID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tickets SET assigned_to = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, userID, ticketID, tenantID)
	return err
}

func (r *ticketRepository) Delete(ctx context.Context, tenantID, ticketID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tickets WHERE id = $1 AND tenant_id = $2`, ticketID, tenantID)
	return err
}

func (r *ticketRepository) List(ctx context.Context, tenantID string, filter TicketFilter) ([]model.Ticket, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("t.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(t.ticket_number ILIKE $%d OR t.subject ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.CustomerID != "" {
		conditions = append(conditions, fmt.Sprintf("t.customer_id = $%d", argIdx))
		args = append(args, filter.CustomerID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("t.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("t.category = $%d", argIdx))
		args = append(args, filter.Category)
		argIdx++
	}

	if filter.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("t.priority = $%d", argIdx))
		args = append(args, filter.Priority)
		argIdx++
	}

	if filter.AssignedTo != "" {
		conditions = append(conditions, fmt.Sprintf("t.assigned_to = $%d", argIdx))
		args = append(args, filter.AssignedTo)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM tickets t JOIN customers c ON c.id = t.customer_id " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	dataQuery := fmt.Sprintf(`
		SELECT t.id, t.tenant_id, t.customer_id, t.ticket_number, t.subject,
		       t.category, t.priority, t.status, t.assigned_to, t.resolved_at, t.closed_at,
		       t.created_at, t.updated_at,
		       c.name, c.customer_code
		FROM tickets t
		JOIN customers c ON c.id = t.customer_id
		%s
		ORDER BY t.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []model.Ticket
	for rows.Next() {
		var t model.Ticket
		var custName, custCode string
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.CustomerID, &t.TicketNumber, &t.Subject,
			&t.Category, &t.Priority, &t.Status, &t.AssignedTo, &t.ResolvedAt, &t.ClosedAt,
			&t.CreatedAt, &t.UpdatedAt,
			&custName, &custCode,
		); err != nil {
			return nil, 0, err
		}
		t.Customer = &model.Customer{ID: t.CustomerID, Name: custName, CustomerCode: custCode}
		tickets = append(tickets, t)
	}

	return tickets, total, nil
}

func (r *ticketRepository) AddMessage(ctx context.Context, msg *model.TicketMessage) error {
	msg.ID = id.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO ticket_messages (id, ticket_id, sender_type, sender_id, message, attachment_url)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, msg.ID, msg.TicketID, msg.SenderType, msg.SenderID, msg.Message, msg.AttachmentURL)
	return err
}

func (r *ticketRepository) ListMessages(ctx context.Context, ticketID string) ([]model.TicketMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ticket_id, sender_type, sender_id, message, COALESCE(attachment_url,''), created_at
		FROM ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []model.TicketMessage
	for rows.Next() {
		var m model.TicketMessage
		if err := rows.Scan(
			&m.ID, &m.TicketID, &m.SenderType, &m.SenderID, &m.Message, &m.AttachmentURL, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	return messages, nil
}
