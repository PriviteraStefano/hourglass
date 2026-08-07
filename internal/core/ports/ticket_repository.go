package ports

import (
	"context"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
)

// TicketRepository is the full ticket persistence surface (ADR-P-003 rev,
// ADR-BE-016). Get existed from plan 05 (origin validation); plans 06 adds
// the lifecycle surface:
//
//   - Create/ListByOrg/UpdateDetails/UpdateState — state layer
//   - Dismiss/Triage — guarded lifecycle actions (TICK-03/TICK-04)
//   - AddComment/ListComments — first-class comment layer (D-06)
//   - ListHistory — the append-only audit stream (TICK-05)
//   - LoggedHours/HasNonTerminalActivities — read-only guards computed
//     server-side from time_entries (T-11-07, OQ2)
//
// Every mutator (Create/UpdateDetails/UpdateState/Dismiss/Triage/AddComment)
// writes its audit_logs row IN THE SAME TRANSACTION as the state write
// (Pitfall 2, ADR-BE-016): the caller passes the audit row(s) to write.
// The interface is append-only by construction — no update/delete paths on
// comments or audit rows exist (TICK-05).
type TicketRepository interface {
	Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticket.Ticket, error)
	Create(ctx context.Context, orgID uuid.UUID, t *ticket.Ticket, audit *audit.AuditLog) (*ticket.Ticket, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, status, kind string) ([]ticket.Ticket, error)
	UpdateDetails(ctx context.Context, orgID, ticketID uuid.UUID, title, description *string, assigneeID *uuid.UUID, audit *audit.AuditLog) (*ticket.Ticket, error)
	UpdateState(ctx context.Context, orgID, ticketID uuid.UUID, to string, note *string, audit *audit.AuditLog) (*ticket.Ticket, error)
	Dismiss(ctx context.Context, orgID, ticketID uuid.UUID, hours float64, audit *audit.AuditLog) (*ticket.Ticket, error)
	Triage(ctx context.Context, orgID, ticketID uuid.UUID, kind *string, plans []*activitydomain.CreateActivityRequest, audits []*audit.AuditLog) (*ticket.Ticket, []*activitydomain.ActivityResponse, error)
	AddComment(ctx context.Context, orgID, ticketID uuid.UUID, c *ticket.TicketComment, audit *audit.AuditLog) (*ticket.TicketComment, error)
	ListComments(ctx context.Context, orgID, ticketID uuid.UUID) ([]ticket.TicketComment, error)
	ListHistory(ctx context.Context, orgID, ticketID uuid.UUID) ([]audit.AuditLog, error)
	LoggedHours(ctx context.Context, ticketID uuid.UUID) (float64, error)
	HasNonTerminalActivities(ctx context.Context, ticketID uuid.UUID) (bool, error)
}
