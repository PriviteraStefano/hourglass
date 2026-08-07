package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
)

// TicketRepository is the minimal ticket read surface for THIS plan:
// an org-scoped Get used by origin validation (customer_ticket same-org
// check, D-02). Plan 06 extends this interface with the lifecycle methods
// (state mutators, triage, comments, history, LoggedHours,
// HasNonTerminalActivities) — no stubs are added here.
type TicketRepository interface {
	Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticket.Ticket, error)
}
