package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
)

// DirectionRepository implements the mutator surface of
// ports.DirectionRepository (ADR-P-015, ADR-BE-018): create (plain +
// supersede-on-create in one tx), activate, cancel, claim (Σ-consumption
// guard under the WG-row FOR UPDATE lock), unclaim — every mutator writes
// its audit rows IN THE SAME TRANSACTION as the state write (BE-012), never
// fire-and-forget, and re-validates the lifecycle matrix / Σ against the
// FOR UPDATE locked row (CR-01 closure — pool-level service checks are
// fast-fail UX only, ticket_repository.go precedent).
//
// The read-model methods (ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs)
// and the full-interface compile-time assertion against ports.DirectionRepository
// land with plan 13-06, which extends this file (the port declares those
// methods, so the var _ assertion cannot compile on the mutator half alone).
type DirectionRepository struct {
	pool *pgxpool.Pool
}

func NewDirectionRepository(pool *pgxpool.Pool) *DirectionRepository {
	return &DirectionRepository{pool: pool}
}

// directionColumns is the canonical SELECT column list for direction rows
// (migration 021).
const directionColumns = `id, org_id, directed_by, directed_to, wg_id, activity_id,
	planned_date, est_hours, priority, due_date, status, supersedes_id,
	origin_direction_id, reason, created_at, updated_at`

// scanDirectionRow scans a pgx.Row into a Direction, normalizing the nullable
// columns (directed_to/wg_id/planned_date/est_hours/priority/due_date/
// supersedes_id/origin_direction_id/reason) into locals before assigning —
// the ticket scan shape (scanTicketRow, ticket_repository.go).
func scanDirectionRow(row pgx.Row) (*directiondomain.Direction, error) {
	var d directiondomain.Direction
	var directedTo, wgID, supersedesID, originDirectionID *uuid.UUID
	var reason *string
	var estHours *float64
	var priority *int
	var plannedDate, dueDate *time.Time
	err := row.Scan(&d.ID, &d.OrgID, &d.DirectedBy, &directedTo, &wgID, &d.ActivityID,
		&plannedDate, &estHours, &priority, &dueDate, &d.Status, &supersedesID,
		&originDirectionID, &reason, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.DirectedTo = directedTo
	d.WgID = wgID
	d.PlannedDate = plannedDate
	d.EstHours = estHours
	d.Priority = priority
	d.DueDate = dueDate
	d.SupersedesID = supersedesID
	d.OriginDirectionID = originDirectionID
	d.Reason = reason
	return &d, nil
}

// Get returns a single direction row org-scoped (pgx.ErrNoRows →
// direction.ErrDirectionNotFound — a cross-org id fails the same-org
// precondition, no existence oracle, T-13-18). The pool-level fast-fail read
// the service (13-07) uses for Activate/Cancel/Claim and the supersede
// fast-fail.
func (r *DirectionRepository) Get(ctx context.Context, orgID, id uuid.UUID) (*directiondomain.Direction, error) {
	d, err := scanDirectionRow(r.pool.QueryRow(ctx,
		`SELECT `+directionColumns+` FROM direction WHERE id = $1 AND org_id = $2`,
		id, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get direction: %w", err)
	}
	return d, nil
}

// getByIDTx re-reads a direction row inside the caller's transaction — the
// post-mutator re-read the mutators return (the ticket loggedHoursTx /
// hasNonTerminalActivitiesTx Tx-variant pattern). The row is visible in-tx
// (the mutator's own writes are the last writer under FOR UPDATE), so the
// returned state is the state about to commit.
func (r *DirectionRepository) getByIDTx(ctx context.Context, tx pgx.Tx, orgID, id uuid.UUID) (*directiondomain.Direction, error) {
	d, err := scanDirectionRow(tx.QueryRow(ctx,
		`SELECT `+directionColumns+` FROM direction WHERE id = $1 AND org_id = $2`,
		id, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get direction in tx: %w", err)
	}
	return d, nil
}

// insertDirectionAudit writes one audit_logs row inside the given transaction
// (BE-012 — mirrors insertCoverageAudit, coverage_repository.go: entity_type
// is written from log.EntityType so the caller controls the vocabulary — the
// direction plane pins direction.AuditEntityDirection); payload JSONB
// marshaled from the audit's Payload map; nil actor/empty comment written as
// SQL NULL. The row id is generated here — the AuditLog's ID field is not
// persisted. Never fire-and-forget: mutators write the audit row in-tx and
// roll back the whole operation if it fails (T-13-17).
func insertDirectionAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()

	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil {
			return fmt.Errorf("marshal direction audit payload: %w", err)
		}
		payload = payloadJSON
	}

	var comment any
	if log.Comment != "" {
		comment = log.Comment
	}

	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, log.OrgID, log.EntityType, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil {
		return wrapPGError(err, "insert direction audit log")
	}
	return nil
}

// Create inserts a new direction row; supersede-on-create in ONE transaction
// when supersedesID is non-nil (D-13-08): the target row is locked FOR
// UPDATE in-org, re-checked draft|active in-tx (CR-01 — the status read after
// the lock is the commit-order truth), the new row is inserted carrying
// supersedes_id → the target, the target flips to 'superseded' with a
// status-precondition UPDATE backstop, and BOTH audit rows (created +
// superseded) are written in the same tx (BE-012). There is no separate
// supersede/transition endpoint — Create-with-supersedes_id is the ONLY
// channel to superseded (D-13-08); a second supersede of the same target
// returns ErrInvalidTransition (no chain rewrite, Pitfall 4).
//
// Claim-row supersede (ADR-BE-018 §5): when the locked target is a claim row
// (origin_direction_id IS NOT NULL), the new row INHERITS origin_direction_id
// — claim hours move along the supersede chain instead of stranding on an
// immutable superseded row — and MUST stay user-targeted: a WG-shaped
// superseding row (WgID set) → ErrInvalidTarget (a WG-shaped row cannot
// carry a claim's origin). The Σ predicate (draft|active claim rows only)
// keeps the WG budget double-count-free across the chain.
func (r *DirectionRepository) Create(ctx context.Context, orgID uuid.UUID, d *directiondomain.Direction, supersedesID *uuid.UUID, audits []*audit.AuditLog) (*directiondomain.Direction, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin direction create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()

	// 1. Supersede-on-create: lock the target in-org and re-check draft|active
	//    against the LOCKED row (CR-01 — the authoritative check; the service's
	//    pool-level supersede fast-fail is UX only).
	var targetStatus string
	if supersedesID != nil {
		var targetOrigin *uuid.UUID
		err = tx.QueryRow(ctx,
			`SELECT status, origin_direction_id FROM direction
			 WHERE id = $1 AND org_id = $2 FOR UPDATE`,
			*supersedesID, orgID).Scan(&targetStatus, &targetOrigin)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, directiondomain.ErrDirectionNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("lock supersede target: %w", err)
		}
		if targetStatus != directiondomain.StatusDraft && targetStatus != directiondomain.StatusActive {
			// Terminal (superseded/cancelled) rows are never supersedable
			// again (Pitfall 4) — the chain is append-only.
			return nil, directiondomain.ErrInvalidTransition
		}
		// Claim-row supersede: inherit the origin onto the superseding row and
		// forbid the WG shape (ADR-BE-018 §5).
		if targetOrigin != nil {
			d.OriginDirectionID = targetOrigin
			if d.WgID != nil {
				return nil, directiondomain.ErrInvalidTarget
			}
		}
	}

	// 2. INSERT the new row (all D-13-01 columns; migration 021 shape).
	//    Status defaults to 'draft' — the caller may pass d.Status.
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	status := d.Status
	if status == "" {
		status = directiondomain.StatusDraft
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id,
			planned_date, est_hours, priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)`,
		d.ID, orgID, d.DirectedBy, d.DirectedTo, d.WgID, d.ActivityID,
		d.PlannedDate, d.EstHours, d.Priority, d.DueDate, status, supersedesID, d.OriginDirectionID, d.Reason, now)
	if err != nil {
		return nil, wrapPGError(err, "create direction")
	}

	// 3. Supersede flip with the status-precondition backstop (CR-01): the
	//    UPDATE only lands on the row the in-tx re-check approved — under the
	//    FOR UPDATE lock no concurrent mutator can change the target between
	//    the re-check and this UPDATE; the precondition is the SQL backstop.
	if supersedesID != nil {
		ct, err := tx.Exec(ctx,
			`UPDATE direction SET status = 'superseded', updated_at = $1
			 WHERE id = $2 AND org_id = $3 AND status = $4`,
			now, *supersedesID, orgID, targetStatus)
		if err != nil {
			return nil, wrapPGError(err, "supersede direction target")
		}
		if ct.RowsAffected() == 0 {
			return nil, directiondomain.ErrInvalidTransition
		}
	}

	// 4. EVERY audit row (created + superseded) in the SAME tx (BE-012,
	//    T-13-17): a failed audit insert rolls back the whole create.
	for _, a := range audits {
		if err := insertDirectionAudit(ctx, tx, a); err != nil {
			return nil, err
		}
	}

	created, err := r.getByIDTx(ctx, tx, orgID, d.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit direction create: %w", err)
	}
	return created, nil
}

// Activate transitions draft → active (explicit endpoint, OQ1 resolution —
// create-with-planned_date does NOT auto-activate). The matrix is re-validated
// against the FOR UPDATE locked row (CR-01): a concurrent transition that
// commits before this tx acquires the lock is read as the locked status, so
// terminal rows can no longer be flipped (Pitfall 4). The UPDATE carries the
// locked status as a precondition backstop; the 'activated' audit row is
// written in the same tx (BE-012).
func (r *DirectionRepository) Activate(ctx context.Context, orgID, id uuid.UUID, auditLog *audit.AuditLog) (*directiondomain.Direction, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin direction activation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT status FROM direction WHERE id = $1 AND org_id = $2 FOR UPDATE`,
		id, orgID).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock direction for activation: %w", err)
	}

	// Authoritative in-tx matrix re-check (D-13-07, CR-01).
	if !directiondomain.CanTransition(currentStatus, directiondomain.StatusActive) {
		return nil, directiondomain.ErrInvalidTransition
	}

	ct, err := tx.Exec(ctx,
		`UPDATE direction SET status = 'active', updated_at = $1
		 WHERE id = $2 AND org_id = $3 AND status = $4`,
		time.Now().UTC(), id, orgID, currentStatus)
	if err != nil {
		return nil, wrapPGError(err, "activate direction")
	}
	if ct.RowsAffected() == 0 {
		return nil, directiondomain.ErrDirectionNotFound
	}

	if auditLog != nil {
		if err := insertDirectionAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	activated, err := r.getByIDTx(ctx, tx, orgID, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit direction activation: %w", err)
	}
	return activated, nil
}

// Cancel transitions draft|active → cancelled with a MANDATORY reason
// (D-13-10; the schema CHECK direction_cancel_reason_check is the second
// line) and writes the 'cancelled' audit row (reason payload) in the same tx.
// The matrix is re-validated against the FOR UPDATE locked row (CR-01) with
// the status-precondition UPDATE backstop; terminal rows reject further
// transitions. Also serves unclaim = cancel of a claim row (D-13-16) — hours
// return to the WG budget automatically since consumption is Σ-derived.
func (r *DirectionRepository) Cancel(ctx context.Context, orgID, id uuid.UUID, reason string, auditLog *audit.AuditLog) (*directiondomain.Direction, error) {
	return r.cancelWithGuard(ctx, orgID, id, reason, auditLog, false)
}

// Unclaim is cancel of a CLAIM row (D-13-16): the same reason requirement and
// matrix re-validation, plus the claim-row guard — a row without
// origin_direction_id is not unclaimable through this path
// (ErrInvalidRequest). The service (13-07) fast-fails "unclaim only on claim
// rows" at the pool level; this in-tx guard is authoritative.
func (r *DirectionRepository) Unclaim(ctx context.Context, orgID, claimRowID uuid.UUID, reason string, auditLog *audit.AuditLog) (*directiondomain.Direction, error) {
	return r.cancelWithGuard(ctx, orgID, claimRowID, reason, auditLog, true)
}

// cancelWithGuard is the shared cancel tx internals (Cancel + Unclaim): lock
// the row FOR UPDATE in-org, enforce the claim-row guard when required,
// re-check the matrix against the locked status (CR-01), UPDATE with a
// status-precondition backstop, write the audit row in-tx, commit.
func (r *DirectionRepository) cancelWithGuard(ctx context.Context, orgID, id uuid.UUID, reason string, auditLog *audit.AuditLog, requireClaimRow bool) (*directiondomain.Direction, error) {
	if reason == "" {
		// Fast-fail at the repo boundary (D-13-10) — the DB CHECK is the
		// second line, this is the first.
		return nil, directiondomain.ErrCancelReasonRequired
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin direction cancellation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	var originDirectionID *uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT status, origin_direction_id FROM direction WHERE id = $1 AND org_id = $2 FOR UPDATE`,
		id, orgID).Scan(&currentStatus, &originDirectionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock direction for cancellation: %w", err)
	}

	// Unclaim path: only claim rows (origin_direction_id set) are unclaimable.
	if requireClaimRow && originDirectionID == nil {
		return nil, directiondomain.ErrInvalidRequest
	}

	// Authoritative in-tx matrix re-check (D-13-07, CR-01): draft|active →
	// cancelled; terminal rows reject.
	if !directiondomain.CanTransition(currentStatus, directiondomain.StatusCancelled) {
		return nil, directiondomain.ErrInvalidTransition
	}

	ct, err := tx.Exec(ctx,
		`UPDATE direction SET status = 'cancelled', reason = $1, updated_at = $2
		 WHERE id = $3 AND org_id = $4 AND status = $5`,
		reason, time.Now().UTC(), id, orgID, currentStatus)
	if err != nil {
		return nil, wrapPGError(err, "cancel direction")
	}
	if ct.RowsAffected() == 0 {
		return nil, directiondomain.ErrDirectionNotFound
	}

	if auditLog != nil {
		if err := insertDirectionAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	cancelled, err := r.getByIDTx(ctx, tx, orgID, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit direction cancellation: %w", err)
	}
	return cancelled, nil
}

// Claim creates the claim row (D-13-11..13): a user-targeted direction row
// with directed_by = the WG row's creator (manager attribution preserved,
// D-13-11), directed_to = the claimant, origin_direction_id = wgRowID, same
// activity, est_hours = the claimed amount, queued (planned_date NULL) and
// draft, copying priority/due_date from the WG row (A8 — the claimant
// schedules via the normal supersede chain).
//
// The Σ over-subscription guard (D-13-13, ADR-BE-018 §5) is AUTHORITATIVE
// inside this tx (CR-01 closure, Pitfall 1): the WG row is locked FOR UPDATE,
// then status/membership/Σ are re-checked in-tx — pool-level service checks
// are fast-fail UX only. Σ is computed in CENTS (math.Round(h*100), the
// coverage precedent) over the predicate origin_direction_id = wgRowID AND
// status IN ('draft','active') — superseded/cancelled claim rows never
// consume budget; over budget → ErrClaimOverBudget (409). Uncapped when the
// WG row's est_hours is NULL (D-13-14). The 'claimed' audit row is written
// in the same tx (BE-012).
func (r *DirectionRepository) Claim(ctx context.Context, orgID, wgRowID, claimantID uuid.UUID, estHours float64, auditLog *audit.AuditLog) (*directiondomain.Direction, error) {
	if estHours <= 0 {
		// Fast-fail at the repo boundary (D-13-03); the DB CHECK
		// direction_est_hours_check is the second line.
		return nil, directiondomain.ErrInvalidHours
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin direction claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Lock the WG row in-org (CR-01): the lock serializes concurrent
	//    claims — the locked values are the commit-order truth. wg_id IS NOT
	//    NULL pins the WG shape (a user row is not claimable).
	var wgEstHours *float64
	var wgStatus string
	var directedBy, activityID, wgID uuid.UUID
	var priority *int
	var dueDate *time.Time
	err = tx.QueryRow(ctx,
		`SELECT est_hours, status, directed_by, activity_id, priority, due_date, wg_id
		 FROM direction WHERE id = $1 AND org_id = $2 AND wg_id IS NOT NULL FOR UPDATE`,
		wgRowID, orgID).Scan(&wgEstHours, &wgStatus, &directedBy, &activityID, &priority, &dueDate, &wgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock working-group row for claim: %w", err)
	}

	// 2. In-tx re-checks (D-13-12, D-13-16): the WG row must be active — a
	//    draft WG row is not claimable (ADR-BE-018 pinned reading) and
	//    superseded/cancelled rows are closed.
	if wgStatus != directiondomain.StatusActive {
		return nil, directiondomain.ErrWgRowNotActive
	}

	// 3. Membership re-check in-tx (D-13-12 — authoritative, never
	//    pool-only): the claimant must be a member of the WG owning the row
	//    (wg_members, migration 000).
	var member bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM wg_members WHERE wg_id = $1 AND user_id = $2)`,
		wgID, claimantID).Scan(&member)
	if err != nil {
		return nil, fmt.Errorf("check working-group membership: %w", err)
	}
	if !member {
		return nil, directiondomain.ErrNotWgMember
	}

	// 4. Σ guard in cents when the budget is set (D-13-13, ADR-BE-018 §5):
	//    superseded/cancelled claim rows never consume budget — the predicate
	//    is draft|active only, so hours freed by cancel/unclaim return
	//    automatically (D-13-16) and a supersede of a claim row keeps the Σ
	//    unchanged (the superseding row carries origin_direction_id).
	if wgEstHours != nil {
		var claimed float64
		err = tx.QueryRow(ctx,
			`SELECT COALESCE(SUM(est_hours), 0) FROM direction
			 WHERE origin_direction_id = $1 AND status IN ('draft','active')`,
			wgRowID).Scan(&claimed)
		if err != nil {
			return nil, fmt.Errorf("compute claimed hours: %w", err)
		}
		claimedCents := int64(math.Round(claimed * 100))
		claimCents := int64(math.Round(estHours * 100))
		budgetCents := int64(math.Round(*wgEstHours * 100))
		if claimedCents+claimCents > budgetCents {
			return nil, directiondomain.ErrClaimOverBudget
		}
	}

	// 5. INSERT the claim row (D-13-11, A8): user-targeted (wg_id NULL,
	//    directed_to = claimant), queued (planned_date NULL), draft status,
	//    attribution preserved (directed_by = the WG row's creator),
	//    priority/due_date copied from the locked WG row.
	now := time.Now().UTC()
	claimID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO direction (id, org_id, directed_by, directed_to, wg_id, activity_id,
			planned_date, est_hours, priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NULL, $5, NULL, $6, $7, $8, 'draft', NULL, $9, NULL, $10, $10)`,
		claimID, orgID, directedBy, claimantID, activityID, estHours, priority, dueDate, wgRowID, now)
	if err != nil {
		return nil, wrapPGError(err, "insert claim row")
	}

	// 6. 'claimed' audit row in the SAME tx (BE-012, T-13-17). The row id is
	//    generated here (the port signature takes no id), so the audit's
	//    entity_id — which the caller cannot know in advance — is pinned to
	//    the claim row the tx creates (ADR-BE-018 §3: entity_id = the
	//    direction row id).
	if auditLog != nil {
		if auditLog.EntityID == uuid.Nil {
			auditLog.EntityID = claimID
		}
		if err := insertDirectionAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	created, err := r.getByIDTx(ctx, tx, orgID, claimID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit direction claim: %w", err)
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// Read-model (plan 13-06) — ListPlan, Coverage, AbsenceWindows,
// FirstDirectionRefs. Derived states are computed ON READ, never stored, no
// nightly jobs (D-V, D-13-09).
// ---------------------------------------------------------------------------

// terminalActivitySubtree reports whether the direction row's activity
// subtree is terminal (D-13-09, ADR-BE-018 §2): the Phase 11 ticket
// dismissal-guard recursive CTE (ticket_repository.go
// HasNonTerminalActivities) RE-ANCHORED at activities.id with the SEMANTIC
// INVERSION — the read-model asks NOT EXISTS (no non-terminal entry) where
// the ticket guard asks EXISTS. Same subtree walk (activity + descendants),
// same non-terminal predicate verbatim: status IN
// ('draft','submitted','pending_manager','pending_finance') AND
// is_deleted = false.
func terminalActivitySubtree(ctx context.Context, pool *pgxpool.Pool, activityID uuid.UUID) (bool, error) {
	var terminal bool
	err := pool.QueryRow(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id FROM activities WHERE id = $1
			UNION ALL
			SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
		 )
		 SELECT NOT EXISTS (
			SELECT 1 FROM time_entries te
			WHERE te.is_deleted = false
			  AND te.status IN ('draft','submitted','pending_manager','pending_finance')
			  AND te.activity_id IN (SELECT id FROM subtree)
		 )`,
		activityID).Scan(&terminal)
	if err != nil {
		return false, wrapPGError(err, "check terminal activity subtree")
	}
	return terminal, nil
}

// hasAnyEntries reports whether the activity subtree carries ANY non-deleted
// time entry — ANY status, draft included (OQ2/A3): the lapsed predicate
// input. A single draft entry on the subtree kills lapsed ("a draft entry
// indicates work started", ADR-BE-018 §2).
func hasAnyEntries(ctx context.Context, pool *pgxpool.Pool, activityID uuid.UUID) (bool, error) {
	var any bool
	err := pool.QueryRow(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id FROM activities WHERE id = $1
			UNION ALL
			SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
		 )
		 SELECT EXISTS (
			SELECT 1 FROM time_entries te
			WHERE te.is_deleted = false
			  AND te.activity_id IN (SELECT id FROM subtree)
		 )`,
		activityID).Scan(&any)
	if err != nil {
		return false, wrapPGError(err, "check activity subtree entries")
	}
	return any, nil
}

// claimSpectrum derives the D-13-15 claim spectrum for a WG row from the Σ of
// its draft|active claim rows vs the row's budget (est_hours) — compared in
// CENTS (Pitfall 3/6, the coverage precedent). Cancelled/superseded claim
// rows never consume: a superseded claim drops out of the Σ while its
// superseding row (origin_direction_id carried, ADR-BE-018 §5) counts the
// same hours, so cancel/unclaim releases hours automatically (D-13-16).
// Budget NULL = uncapped (D-13-14): partially_claimed once any claim exists,
// fully_claimed never derives.
func claimSpectrum(ctx context.Context, pool *pgxpool.Pool, wgRowID uuid.UUID, budget *float64) (string, error) {
	var claimed float64
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(est_hours), 0) FROM direction
		 WHERE origin_direction_id = $1 AND status IN ('draft','active')`,
		wgRowID).Scan(&claimed)
	if err != nil {
		return "", wrapPGError(err, "compute claim spectrum")
	}
	claimedCents := int64(math.Round(claimed * 100))
	if claimedCents == 0 {
		return directiondomain.ClaimStateNotClaimed, nil
	}
	if budget == nil {
		return directiondomain.ClaimStatePartiallyClaimed, nil
	}
	budgetCents := int64(math.Round(*budget * 100))
	if claimedCents >= budgetCents {
		return directiondomain.ClaimStateFullyClaimed, nil
	}
	return directiondomain.ClaimStatePartiallyClaimed, nil
}

// ListPlan returns the plan read-model (D-13-27): draft|active direction rows
// for the period with the derived states computed ON READ, never stored
// (D-V): done (terminal-activity CTE semantic inversion), lapsed (the row's
// date — planned_date, or due_date for queued — in the past AND no non-deleted
// entries of ANY status on the activity subtree, A3), and the claim spectrum
// for WG rows (D-13-15). Superseded/cancelled rows are history and never
// appear (D-13-08).
//
// Selection: scheduled rows (planned_date set) with planned_date within
// [periodStart, periodEnd]; queued rows (planned_date NULL) with due_date
// within the period OR no due_date at all — the queue is part of the plan
// view. employeeID nil = all employees (org-scoped). Ordering (D-13-06,
// probe DIR-01): planned_date ASC NULLS LAST, priority ASC NULLS LAST
// (lower = higher), due_date ASC NULLS LAST, created_at ASC — stable for
// equal keys.
func (r *DirectionRepository) ListPlan(ctx context.Context, orgID uuid.UUID, employeeID *uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.PlanRow, error) {
	query := `SELECT ` + directionColumns + ` FROM direction
	 WHERE org_id = $1 AND status IN ('draft','active')`
	args := []any{orgID}
	n := 1
	if employeeID != nil {
		n++
		query += fmt.Sprintf(" AND directed_to = $%d", n)
		args = append(args, *employeeID)
	}
	n++
	query += fmt.Sprintf(` AND (
		(planned_date IS NOT NULL AND planned_date >= $%d::date AND planned_date <= $%d::date)
		OR (planned_date IS NULL AND (due_date IS NULL OR (due_date >= $%d::date AND due_date <= $%d::date)))
	)
	 ORDER BY planned_date ASC NULLS LAST, priority ASC NULLS LAST, due_date ASC NULLS LAST, created_at ASC`,
		n, n+1, n, n+1)
	args = append(args, periodStart, periodEnd)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapPGError(err, "list plan rows")
	}
	defer rows.Close()

	var directions []directiondomain.Direction
	for rows.Next() {
		d, err := scanDirectionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan plan row: %w", err)
		}
		directions = append(directions, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan rows: %w", err)
	}

	// Derived states, one pass per DISTINCT activity of the returned rows.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	type activityDerived struct{ done, anyEntries bool }
	derived := make(map[uuid.UUID]activityDerived)
	for _, d := range directions {
		if _, ok := derived[d.ActivityID]; ok {
			continue
		}
		done, err := terminalActivitySubtree(ctx, r.pool, d.ActivityID)
		if err != nil {
			return nil, err
		}
		anyEntries, err := hasAnyEntries(ctx, r.pool, d.ActivityID)
		if err != nil {
			return nil, err
		}
		derived[d.ActivityID] = activityDerived{done: done, anyEntries: anyEntries}
	}

	planRows := make([]directiondomain.PlanRow, 0, len(directions))
	for _, d := range directions {
		pr := directiondomain.PlanRow{Direction: d}
		ad := derived[d.ActivityID]
		pr.Done = ad.done
		effectiveDate := d.PlannedDate
		if effectiveDate == nil {
			effectiveDate = d.DueDate
		}
		pr.Lapsed = effectiveDate != nil && effectiveDate.Before(today) && !ad.anyEntries
		// Claim spectrum for WG rows only (D-13-15).
		if d.WgID != nil {
			cs, err := claimSpectrum(ctx, r.pool, d.ID, d.EstHours)
			if err != nil {
				return nil, err
			}
			pr.ClaimState = cs
		}
		planRows = append(planRows, pr)
	}
	return planRows, nil
}

// normalizeDay returns the date-only instant (UTC midnight) of a scanned DATE
// column. PostgreSQL DATE columns scan back in the SESSION timezone (e.g.
// Local — a `2026-08-08` value reads `2026-08-08T02:00:00+02:00` on a
// +02:00 session), which makes day-key comparisons (maps, dedup) and JSON
// serialization nondeterministic. The read-model's day semantics are
// timezone-free — the read-model normalizes, the mutator scans stay as-is.
func normalizeDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Coverage returns the direction-coverage read-model (DIR-06, D-13-25/26):
// one row per (employee, day) for every day in the period — zero rows are
// still surfaced (an uncovered day is planned 0, gap == capacity; D-13-26 —
// the SERVICE owns the surfacing policy, e.g. excluding fully-absent days
// and validity-outside employees, this repo owns the math).
//
// Capacity per day = planning_daily_hours (org_settings key, COALESCE'd to
// the 8.0 default — ADR-BE-018 consequence: the default is code-level, never
// a seed row) − that day's absence hours: a full window (hours NULL,
// declared|confirmed) covering the day zeroes it (away); a partial window
// subtracts its hours; overlapping windows both subtract; the result floors
// at 0 (Pitfall 10 — a zeroed day is an away-day, never negative capacity
// that would fake over-capacity). planned = Σ est_hours of draft|active rows
// for that employee on that exact day (superseded/cancelled history rows can
// never inflate the Σ — T-13-19); gap = capacity − planned (negative =
// over-capacity). employeeIDs is the resolved employee set — scope
// resolution lives in the service (D-13-25 flagged assumption).
func (r *DirectionRepository) Coverage(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.CoverageRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT e.employee_id, d.day,
		        CASE WHEN full_abs.day IS NOT NULL THEN 0.0
		             ELSE GREATEST(daily.daily_hours - COALESCE(partial_abs.hours, 0.0), 0.0)
		        END AS capacity,
		        COALESCE(planned.hours, 0.0) AS planned
		 FROM unnest($2::uuid[]) AS e(employee_id)
		 CROSS JOIN generate_series($3::date, $4::date, '1 day') AS d(day)
		 CROSS JOIN (SELECT COALESCE((SELECT value::text::numeric FROM org_settings
		                               WHERE org_id = $1 AND key = 'planning_daily_hours'), 8.0) AS daily_hours) daily
		 LEFT JOIN (
			SELECT directed_to AS employee_id, planned_date AS day, SUM(est_hours) AS hours
			FROM direction
			WHERE org_id = $1 AND status IN ('draft','active')
			  AND planned_date >= $3::date AND planned_date <= $4::date
			GROUP BY directed_to, planned_date
		 ) planned ON planned.employee_id = e.employee_id AND planned.day = d.day
		 LEFT JOIN (
			SELECT aw.user_id AS employee_id, gs.day, SUM(aw.hours) AS hours
			FROM availability_windows aw
			CROSS JOIN LATERAL generate_series(GREATEST(aw.starts_on, $3::date), LEAST(aw.ends_on, $4::date), '1 day') AS gs(day)
			WHERE aw.org_id = $1 AND aw.user_id = ANY($2)
			  AND aw.status IN ('declared','confirmed') AND aw.hours IS NOT NULL
			GROUP BY aw.user_id, gs.day
		 ) partial_abs ON partial_abs.employee_id = e.employee_id AND partial_abs.day = d.day
		 LEFT JOIN (
			SELECT aw.user_id AS employee_id, gs.day
			FROM availability_windows aw
			CROSS JOIN LATERAL generate_series(GREATEST(aw.starts_on, $3::date), LEAST(aw.ends_on, $4::date), '1 day') AS gs(day)
			WHERE aw.org_id = $1 AND aw.user_id = ANY($2)
			  AND aw.status IN ('declared','confirmed') AND aw.hours IS NULL
			GROUP BY aw.user_id, gs.day
		 ) full_abs ON full_abs.employee_id = e.employee_id AND full_abs.day = d.day
		 ORDER BY e.employee_id, d.day`,
		orgID, employeeIDs, periodStart, periodEnd)
	if err != nil {
		return nil, wrapPGError(err, "compute coverage read-model")
	}
	defer rows.Close()

	var coverage []directiondomain.CoverageRow
	for rows.Next() {
		var c directiondomain.CoverageRow
		if err := rows.Scan(&c.EmployeeID, &c.Date, &c.Capacity, &c.Planned); err != nil {
			return nil, fmt.Errorf("scan coverage row: %w", err)
		}
		c.Date = normalizeDay(c.Date)
		// Cents-rounded render (Pitfall 6: DECIMAL arithmetic in SQL is
		// exact; the float64 split only renders the read model).
		c.Capacity = roundCents(c.Capacity)
		c.Planned = roundCents(c.Planned)
		c.Gap = roundCents(c.Capacity - c.Planned)
		coverage = append(coverage, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coverage rows: %w", err)
	}
	return coverage, nil
}

// AbsenceWindows returns availability_windows rows with status IN
// ('declared','confirmed') — BOTH statuses per D-13-29 (Phase 14 tightens to
// confirmed-only) — overlapping [start, end], org-scoped, for the given
// employees. Kind + hours + starts_on + ends_on feed the service-side warning
// formatting (13-07): hours NULL = full absence, set = partial-day permit.
// The schema column is user_id (migration 012) — mapped to
// direction.AbsenceWindow.EmployeeID.
func (r *DirectionRepository) AbsenceWindows(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, start, end time.Time) ([]directiondomain.AbsenceWindow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, kind, starts_on, ends_on, hours FROM availability_windows
		 WHERE org_id = $1 AND user_id = ANY($2)
		   AND status IN ('declared','confirmed')
		   AND starts_on <= $4::date AND ends_on >= $3::date
		 ORDER BY user_id, starts_on`,
		orgID, employeeIDs, start, end)
	if err != nil {
		return nil, wrapPGError(err, "list absence windows")
	}
	defer rows.Close()

	var windows []directiondomain.AbsenceWindow
	for rows.Next() {
		var w directiondomain.AbsenceWindow
		if err := rows.Scan(&w.EmployeeID, &w.Kind, &w.StartsOn, &w.EndsOn, &w.Hours); err != nil {
			return nil, fmt.Errorf("scan absence window: %w", err)
		}
		w.StartsOn = normalizeDay(w.StartsOn)
		w.EndsOn = normalizeDay(w.EndsOn)
		windows = append(windows, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate absence windows: %w", err)
	}
	return windows, nil
}
