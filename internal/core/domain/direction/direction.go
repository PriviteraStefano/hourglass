// Package direction implements the plan-plane domain (ADR-P-015,
// ADR-BE-018): the Direction entity with per-day rows, the pinned
// ticket-style lifecycle matrix, the derived-state vocabularies, and the
// read-model row shapes the Phase 19 surfaces consume.
package direction

import (
	"time"

	"github.com/google/uuid"
)

// Direction is the plan-plane entity (D-13-01, ADR-BE-018 §1): one row per
// day of planned work. Mode is DERIVED, never stored — PlannedDate set →
// scheduled, nil → queued (D-R). Rows are immutable after creation: the plan
// is mutable as a chain of rows via supersedes_id (D-13-04), never by
// rewriting the fact of a prior row. DirectedTo and WgID are XOR (D-13-05):
// exactly one is set (DB CHECK direction_target_check).
type Direction struct {
	ID                uuid.UUID  `json:"id"`
	OrgID             uuid.UUID  `json:"org_id"`
	DirectedBy        uuid.UUID  `json:"directed_by"`
	DirectedTo        *uuid.UUID `json:"directed_to,omitempty"`
	WgID              *uuid.UUID `json:"wg_id,omitempty"`
	ActivityID        uuid.UUID  `json:"activity_id"`
	PlannedDate       *time.Time `json:"planned_date,omitempty"`
	EstHours          *float64   `json:"est_hours,omitempty"`
	Priority          *int       `json:"priority,omitempty"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	Status            string     `json:"status"`
	SupersedesID      *uuid.UUID `json:"supersedes_id,omitempty"`
	OriginDirectionID *uuid.UUID `json:"origin_direction_id,omitempty"`
	Reason            *string    `json:"reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Closed status vocabulary (D-13-07, ADR-BE-018 §1): the DB CHECK
// direction_status_check mirrors these values.
const (
	StatusDraft      = "draft"
	StatusActive     = "active"
	StatusSuperseded = "superseded"
	StatusCancelled  = "cancelled"
)

// transitionMatrix pins the direction matrix (D-13-07, ADR-BE-018 §1):
// draft→active, draft→cancelled, active→cancelled. superseded is NOT in the
// matrix — it is reachable only via Create-with-supersedes_id (D-13-08); a
// supersede transition endpoint does not exist.
var transitionMatrix = map[string]map[string]bool{
	StatusDraft: {
		StatusActive:    true,
		StatusCancelled: true,
	},
	StatusActive: {
		StatusCancelled: true,
	},
}

// CanTransition reports whether the pinned matrix allows from→to.
func CanTransition(from, to string) bool {
	return transitionMatrix[from][to]
}

// IsTerminalStatus reports whether the status is terminal: superseded or
// cancelled (ADR-BE-018 §1). Neither has outgoing edges.
func IsTerminalStatus(s string) bool {
	return s == StatusSuperseded || s == StatusCancelled
}

// Derived-state vocabulary (D-V, D-13-09, ADR-BE-018 §2): computed on read,
// never stored, no nightly jobs. The claim-spectrum constants below refine
// StatusDerivedClaimed for WG rows.
const (
	StatusDerivedDone    = "done"
	StatusDerivedLapsed  = "lapsed"
	StatusDerivedClaimed = "claimed"
)

// Closed claim-spectrum vocabulary (D-13-15, ADR-BE-018 §2): derived for WG
// rows only. fully_claimed derives only when the WG row's budget (est_hours)
// is set AND Σ claims == budget; an uncapped row (budget NULL) never derives
// fully_claimed (D-13-14).
const (
	ClaimStateNotClaimed       = "not_claimed"
	ClaimStatePartiallyClaimed = "partially_claimed"
	ClaimStateFullyClaimed     = "fully_claimed"
)

// Warning is one advisory overlay on a read-model row (D-13-28/30/31,
// 13-UI-SPEC API Data Contracts): soft, never blocking. Message is
// pre-rendered server-side in the pinned "{Type} {date-range-or-day}"
// format — Phase 19 renders it verbatim.
type Warning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Closed warning-type vocabulary (D-13-30/31, 13-UI-SPEC): the four possible
// Warning.Type values.
const (
	WarningAway         = "away"
	WarningPartial      = "partial"
	WarningOverCapacity = "over-capacity"
	WarningInvalid      = "invalid"
)

// CoverageRow is one (employee, date) cell of the direction-coverage
// read-model (D-13-25/26, 13-UI-SPEC): planned vs absence-aware capacity.
// Gap = capacity − planned (negative when over-capacity); fully-absent days
// are excluded from uncovered surfacing by the service (D-13-26/31).
type CoverageRow struct {
	EmployeeID uuid.UUID `json:"employee_id"`
	Date       time.Time `json:"date"`
	Capacity   float64   `json:"capacity"`
	Planned    float64   `json:"planned"`
	Gap        float64   `json:"gap"`
}

// PlanRow is one row of the plan read-model (D-13-27): the Direction row
// plus the derived-on-read states (D-13-09/15) — done/lapsed for every row,
// ClaimState for WG rows. Derived states are never stored (D-13-09, D-V).
type PlanRow struct {
	Direction
	Done       bool   `json:"done"`
	Lapsed     bool   `json:"lapsed"`
	ClaimState string `json:"claim_state,omitempty"`
}

// DirectionRefs is the origin-fallback shape (D-13-32..34): the
// manager-assignment pair derived from the earliest non-cancelled direction
// row for an activity (directed_by → assigned_by, directed_to → assigned_to).
// It carries ONLY the manager-assignment shape — never
// proposed_by/reviewed_by/ticket_id (D-13-33). Derivation is read-only and
// never written back (D-13-34).
type DirectionRefs struct {
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
}

// AbsenceWindow is one availability window feeding the warning overlay
// (D-13-28/29, ADR-P-008): declared AND confirmed statuses are read until
// Phase 14 tightens to confirmed-only. Hours is set for partial-day permits
// (permit reduces capacity by its hours) and NULL for full absences (the day
// zeroes out — away).
type AbsenceWindow struct {
	EmployeeID uuid.UUID `json:"employee_id"`
	Kind       string    `json:"kind"`
	StartsOn   time.Time `json:"starts_on"`
	EndsOn     time.Time `json:"ends_on"`
	Hours      *float64  `json:"hours,omitempty"`
}

// Audit vocabulary (A1, ADR-BE-018 §3): direction mutators hand audit rows
// to the repo with entity_type='direction' and one of these actions —
// pinned verbatim so Phase 19 history reads filter deterministically
// (T-13-06). Exported so the repo and service can never drift from the
// vocabulary (coverage constants block analog).
const (
	AuditEntityDirection = "direction"

	AuditActionCreated    = "created"
	AuditActionActivated  = "activated"
	AuditActionCancelled  = "cancelled"
	AuditActionSuperseded = "superseded"
	AuditActionClaimed    = "claimed"
	AuditActionUnclaimed  = "unclaimed"
)
