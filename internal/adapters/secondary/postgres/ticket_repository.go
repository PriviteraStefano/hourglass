package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// TicketRepository implements ports.TicketRepository using a pgxpool
// (schema 014). This plan exposes only the org-scoped Get; plan 06 extends
// the port with the lifecycle surface.
type TicketRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: TicketRepository implements the port.
var _ ports.TicketRepository = (*TicketRepository)(nil)

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool}
}

// Get returns a single ticket scoped to the org. A missing ticket — or a
// ticket that exists in another org — surfaces as ticket.ErrTicketNotFound
// (cross-org id fails the same-org precondition, D-02).
func (r *TicketRepository) Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error) {
	var t ticketdomain.Ticket
	var assigneeID *uuid.UUID
	var dismissedHours *float64

	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, title, description, kind, status, requester_id, assignee_id,
		        dismissed_hours, created_at, updated_at
		 FROM tickets WHERE id = $1 AND org_id = $2`,
		ticketID, orgID).Scan(
		&t.ID, &t.OrgID, &t.Title, &t.Description, &t.Kind, &t.Status, &t.RequesterID,
		&assigneeID, &dismissedHours, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ticketdomain.ErrTicketNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	t.AssigneeID = assigneeID
	t.DismissedHours = dismissedHours
	return &t, nil
}
