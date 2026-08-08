package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
)

// DirectionRepository is the direction plane's persistence surface
// (ADR-P-015, ADR-BE-018): the per-day plan rows, the guarded lifecycle
// mutators, the claim model, and the plan/coverage read-models.
//
// This interface is the COMPILE-TIME CONTRACT between the postgres repo
// (13-05/13-06), the direction service (13-07), the activity service's
// origin-fallback dependency (13-08) and the testdata mocks — signatures
// are pinned here and must not change later.
//
// Write shape: every mutator (Create/Activate/Cancel/Claim) takes its audit
// row(s) and writes them IN THE SAME TRANSACTION as the state write
// (BE-016 Pitfall 2, ADR-BE-018 §3) — never fire-and-forget. The repo
// re-validates the matrix/Σ against the FOR UPDATE locked row (CR-01
// closure); pool-level service checks are fast-fail UX only.
//
//   - Create performs supersede-on-create in one tx (D-13-08): the new row
//     carries supersedes_id → the target flips to superseded in the same
//     transaction (two audit rows: created + superseded).
//   - Claim re-checks Σ claims ≤ WG-row est_hours in-tx under the WG-row
//     lock in cents (D-13-13, ADR-BE-018 §5) → direction.ErrClaimOverBudget
//     (409); uncapped when the WG row's est_hours is NULL (D-13-14).
type DirectionRepository interface {
	// Get returns a single direction row org-scoped (pgx.ErrNoRows →
	// direction.ErrDirectionNotFound). Pool-level single-row read used by
	// the service for Activate/Cancel/Claim fast-fails (13-07) and the
	// supersede fast-fail.
	Get(ctx context.Context, orgID, id uuid.UUID) (*direction.Direction, error)

	// Create inserts a new direction row; supersede-on-create in one tx
	// when supersedesID is non-nil (D-13-08): the target row is locked
	// FOR UPDATE, re-checked draft|active (else ErrInvalidTransition),
	// flipped to superseded, and both audit rows (created + superseded)
	// are written in the same transaction. When the target is a claim row
	// (origin_direction_id set), the new row inherits origin_direction_id
	// and MUST remain user-targeted (ADR-BE-018 §5 — WG-shaped superseding
	// rows → ErrInvalidTarget).
	Create(ctx context.Context, orgID uuid.UUID, d *direction.Direction, supersedesID *uuid.UUID, audits []*audit.AuditLog) (*direction.Direction, error)

	// Activate transitions draft → active (explicit endpoint, OQ1
	// resolution — create-with-planned_date does NOT auto-activate). One
	// audit row in the same tx; matrix re-validated under lock.
	Activate(ctx context.Context, orgID, id uuid.UUID, audit *audit.AuditLog) (*direction.Direction, error)

	// Cancel transitions draft|active → cancelled with a mandatory reason
	// (D-13-10; schema CHECK direction_cancel_reason_check). One audit row
	// in the same tx. Also serves unclaim = cancel of a claim row
	// (D-13-16) — hours return to the WG budget automatically since
	// consumption is Σ-derived.
	Cancel(ctx context.Context, orgID, id uuid.UUID, reason string, audit *audit.AuditLog) (*direction.Direction, error)

	// Unclaim cancels a CLAIM row (D-13-16, the 13-07 service path): the
	// same reason requirement and matrix re-validation as Cancel, plus the
	// claim-row guard — a row without origin_direction_id is rejected with
	// ErrInvalidRequest. Hours return to the WG budget automatically since
	// consumption is Σ-derived. One 'unclaimed' audit row in the same tx
	// (ADR-BE-018 §3).
	Unclaim(ctx context.Context, orgID, claimRowID uuid.UUID, reason string, audit *audit.AuditLog) (*direction.Direction, error)

	// Claim creates the claim row (D-13-11..13): user-targeted row with
	// directed_by = the WG row's creator (manager attribution preserved),
	// origin_direction_id = wgRowID, est_hours = claimed amount. The Σ
	// guard (Σ claims ≤ WG est_hours, in cents) is re-checked IN-TX under
	// the WG-row FOR UPDATE lock (CR-01, ADR-BE-018 §5) over the predicate
	// origin_direction_id = wgRowID AND status IN ('draft','active') —
	// superseded/cancelled claim rows never consume budget; uncapped when
	// the WG budget is NULL. One claimed audit row in the same tx.
	Claim(ctx context.Context, orgID, wgRowID, claimantID uuid.UUID, estHours float64, audit *audit.AuditLog) (*direction.Direction, error)

	// ListPlan returns the plan read-model (D-13-27): direction rows
	// (status draft|active) with derived states computed on read — done
	// (terminal-activity CTE semantic inversion), lapsed (past date, no
	// non-deleted entries anywhere on the subtree — A3), claim spectrum
	// for WG rows (D-13-15). employeeID nil = all employees (org-scoped).
	// Scheduled rows by planned_date; queued rows (planned_date NULL) by
	// priority ASC NULLS LAST, due_date ASC NULLS LAST, created_at.
	ListPlan(ctx context.Context, orgID uuid.UUID, employeeID *uuid.UUID, periodStart, periodEnd time.Time) ([]direction.PlanRow, error)

	// Coverage returns the direction-coverage read-model (DIR-06, D-13-25/
	// 26): (employee, date, capacity, planned, gap) per day where capacity
	// = planning_daily_hours (default 8.0 — ADR-BE-018 §8.3, applied when
	// the key is absent) minus confirmed+declared absence hours that day
	// (full absence → 0). employeeIDs is the resolved employee set — scope
	// resolution (employee/unit/WG) lives in the service, never here
	// (D-13-25 flagged assumption).
	Coverage(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, periodStart, periodEnd time.Time) ([]direction.CoverageRow, error)

	// AbsenceWindows returns availability_windows rows with status IN
	// ('declared','confirmed') — BOTH statuses per D-13-29 (Phase 14
	// tightens to confirmed-only), overlapping [start, end], org-scoped.
	// Feeds the service-side warning computation (13-07).
	AbsenceWindows(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, start, end time.Time) ([]direction.AbsenceWindow, error)

	// FirstDirectionRefs returns the manager-assignment-shaped refs of the
	// EARLIEST created_at non-cancelled direction row for the activity
	// (D-13-32/33): assigned_by = directed_by, assigned_to = directed_to —
	// never the other origin shapes. nil when no such row (refs stay empty,
	// D-13-34). Read-only; the origin-fallback port the activity service
	// consumes (13-08).
	FirstDirectionRefs(ctx context.Context, orgID, activityID uuid.UUID) (*direction.DirectionRefs, error)
}
