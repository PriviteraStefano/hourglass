package ticket

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Ticket is the first-class internal demand-tracking entity (ADR-P-003 rev,
// D-A/D-E). v0.2 keeps tickets internal-only — no customer-facing portal;
// external desk intake is a future hexagonal port. Tickets are demand
// tracking, not task execution: no kanban, no sub-task trees, no comment
// threads as conversation (P-003 hard boundary list kept verbatim).
type Ticket struct {
	ID             uuid.UUID  `json:"id"`
	OrgID          uuid.UUID  `json:"org_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Kind           string     `json:"kind"`
	Status         string     `json:"status"`
	RequesterID    uuid.UUID  `json:"requester_id"`
	AssigneeID     *uuid.UUID `json:"assignee_id,omitempty"`
	DismissedHours *float64   `json:"dismissed_hours,omitempty"` // TICK-04: hours logged before dismissal (D-13 raw Σ)
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TicketComment is an append-only comment on a ticket (CASCADE on ticket
// delete — schema 014).
type TicketComment struct {
	ID        uuid.UUID `json:"id"`
	TicketID  uuid.UUID `json:"ticket_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Closed kind vocabulary (D-A): question / bug / change / evolution.
const (
	KindQuestion  = "question"
	KindBug       = "bug"
	KindChange    = "change"
	KindEvolution = "evolution"
)

// Closed status vocabulary (D-A lifecycle): open→triage→planned→in_progress→
// resolved→closed, plus the dismissal terminal state.
const (
	StatusOpen       = "open"
	StatusTriage     = "triage"
	StatusPlanned    = "planned"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
	StatusDismissed  = "dismissed"
)

var (
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidTransition   = errors.New("invalid ticket status transition")
	ErrDismissalBlocked    = errors.New("ticket dismissal blocked: linked activities have logged hours")
	ErrActivityNotTerminal = errors.New("ticket activity is not in a terminal state")
)

// JSONNames maps sentinel errors to stable JSON-safe names (house style:
// sentinels carry plain error messages; the map serves serialization/UI keys).
var JSONNames = map[error]string{
	ErrTicketNotFound:      "ticket_not_found",
	ErrInvalidRequest:      "invalid_request",
	ErrForbidden:           "forbidden",
	ErrInvalidTransition:   "invalid_transition",
	ErrDismissalBlocked:    "dismissal_blocked",
	ErrActivityNotTerminal: "activity_not_terminal",
}

// transitionMatrix pins the locked transition matrix (A7/OQ6 + ADR-BE-016):
// open→triage; triage→planned; triage→dismissed; planned→in_progress;
// in_progress→resolved; resolved→closed; resolved→in_progress (reopen);
// open→dismissed (superset of A7, pinned in ADR-BE-016). Nothing else.
var transitionMatrix = map[string]map[string]bool{
	StatusOpen: {
		StatusTriage:    true,
		StatusDismissed: true,
	},
	StatusTriage: {
		StatusPlanned:   true,
		StatusDismissed: true,
	},
	StatusPlanned: {
		StatusInProgress: true,
	},
	StatusInProgress: {
		StatusResolved: true,
	},
	StatusResolved: {
		StatusClosed:      true,
		StatusInProgress:  true, // reopen
	},
}

// CanTransition reports whether the locked matrix allows from→to.
func CanTransition(from, to string) bool {
	return transitionMatrix[from][to]
}

// IsTerminalStatus reports whether the status is a terminal state: closed or
// dismissed.
func IsTerminalStatus(s string) bool {
	return s == StatusClosed || s == StatusDismissed
}

// IsOwner reports whether userID is the ticket's requester.
func (t *Ticket) IsOwner(userID uuid.UUID) bool {
	return t.RequesterID == userID
}

// IsAssignee reports whether userID is the ticket's assignee.
func (t *Ticket) IsAssignee(userID uuid.UUID) bool {
	return t.AssigneeID != nil && *t.AssigneeID == userID
}
