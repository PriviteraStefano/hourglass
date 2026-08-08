// Package directionsvc implements the plan plane's business semantics
// (ADR-P-015, ADR-BE-018): the create orchestration with the mode gate +
// BE-014 routing gate + WG-scope predicate, the lifecycle orchestration
// (activate/cancel/unclaim with matrix fast-fail), the WG claim orchestration
// (membership + Σ fast-fail feeding the authoritative repo tx), the warning
// pure-function overlay, and the ListPlan/Coverage read-model assembly.
//
// The repository owns the transactions; every invariant a user can observe
// is decided here:
//
//   - D-13-05: the XOR target fast-fail (exactly one of directed_to / wg_id)
//     before any fetch.
//   - D-13-02/03: est_hours must be positive and whole-cent; a scheduled row
//     (planned_date set) must carry est_hours.
//   - D-13-17/A5/Pitfall 9: WG rows are queued-only and must stay within the
//     WG scope (activity == the WG's anchored activity, or the anchor in the
//     activity's ancestry).
//   - D-13-19/20 + A9: the mode gate — membership override → org default →
//     manager_planned fallback (orgsettings.ResolvePlanningMode); strict
//     reading: self_planned → only the employee creates own rows (no routing
//     call, D-S); manager_planned → every create passes the BE-014 routing
//     gate (the coverage gate shape verbatim, D-G parity).
//   - A10: WG-row creation routes on the WG's ANCHORED activity.
//   - D-13-07/10/12/13/16: lifecycle + claim fast-fails are pool-level UX
//     only (CR-01) — the repo re-validates the matrix and the Σ under the
//     FOR UPDATE lock; the in-tx re-check is authoritative.
//   - D-13-28/30/31: warnings are a pure overlay (away | partial |
//     over-capacity | invalid) with messages pre-rendered server-side in the
//     13-UI-SPEC pinned formats — warnings NEVER block a write.
//   - D-13-25/26/31: the coverage read-model resolves the employee set by
//     scope, drops validity-outside employees from the repo call, excludes
//     fully-absent days from uncovered surfacing, and delivers rows + period
//     totals + warnings.
//
// Reads are gated (T-13-26): the org-wide plan view (employee_id omitted) is
// manager-only; non-managers read only their own plan. Coverage unit/wg
// scopes are manager-only; the employee scope is the self-view for
// non-managers.
package directionsvc

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	orgsettingssvc "github.com/stefanoprivitera/hourglass/internal/core/services/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service orchestrates the direction plane (ADR-BE-018). Deps: the direction
// repo (transactions + read-models), the activity repo (same-org + ancestry
// for the WG scope predicate), the WG repo (membership + scope anchors), the
// unit repo (primary-unit resolution + scope resolution), the org repo
// (membership validity), the shared orgsettings service (mode gate) and the
// shared routing service (BE-014 manager reach) — the last two built once in
// cmd/server wiring and shared (D-G parity).
type Service struct {
	repo         ports.DirectionRepository
	activityRepo ports.ActivityRepository
	wgRepo       ports.WorkingGroupRepository
	unitRepo     ports.UnitRepository
	orgRepo      ports.OrganizationRepository
	orgSettings  *orgsettingssvc.Service
	routing      *routing.Service
}

func NewService(repo ports.DirectionRepository, activityRepo ports.ActivityRepository, wgRepo ports.WorkingGroupRepository, unitRepo ports.UnitRepository, orgRepo ports.OrganizationRepository, orgSettingsSvc *orgsettingssvc.Service, routingSvc *routing.Service) *Service {
	return &Service{
		repo:         repo,
		activityRepo: activityRepo,
		wgRepo:       wgRepo,
		unitRepo:     unitRepo,
		orgRepo:      orgRepo,
		orgSettings:  orgSettingsSvc,
		routing:      routingSvc,
	}
}

// CreateDirectionRequest mirrors the D-13-01 write fields (13-08 handler
// binds the JSON body onto this shape).
type CreateDirectionRequest struct {
	DirectedTo   *uuid.UUID `json:"directed_to,omitempty"`
	WgID         *uuid.UUID `json:"wg_id,omitempty"`
	ActivityID   uuid.UUID  `json:"activity_id"`
	PlannedDate  *time.Time `json:"planned_date,omitempty"`
	EstHours     *float64   `json:"est_hours,omitempty"`
	Priority     *int       `json:"priority,omitempty"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	SupersedesID *uuid.UUID `json:"supersedes_id,omitempty"`
}

// contains reports whether id is in ids (coverage service helper analog).
func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// wholeCent reports whether hours is a positive whole-cent value (D-13-03,
// coverage precedent — sub-cent values would corrupt the repo's cents
// arithmetic and violate the DECIMAL(8,2) column).
func wholeCent(hours float64) bool {
	return hours > 0 && math.Round(hours*100) == hours*100
}

// primaryUnitID resolves the employee's primary unit membership (A9/OQ7
// strict reading): the membership flagged primary wins, else the first
// membership; uuid.Nil when the employee has no unit membership — the
// routing resolution then degrades to the terminal role-gated stage
// (proposer pattern, house style).
func primaryUnitID(members []unitdomain.UnitMember) uuid.UUID {
	for _, m := range members {
		if m.IsPrimary {
			if uid, err := uuid.Parse(m.UnitID); err == nil {
				return uid
			}
		}
	}
	for _, m := range members {
		if uid, err := uuid.Parse(m.UnitID); err == nil {
			return uid
		}
	}
	return uuid.Nil
}

// primaryUnitIDFor resolves the employee's primary unit via the unit repo
// (ListMembershipsForUser — the strict A9/OQ7 reading: unitID = the target
// employee's primary unit).
func (s *Service) primaryUnitIDFor(ctx context.Context, employeeID uuid.UUID) (uuid.UUID, error) {
	members, err := s.unitRepo.ListMembershipsForUser(ctx, employeeID)
	if err != nil {
		return uuid.Nil, err
	}
	return primaryUnitID(members), nil
}

// managerReach is the BE-014 manager-stage gate, the coverage service shape
// VERBATIM (D-G parity — no re-implementation): the actor passes when in the
// ApproverIDs set, or the resolution is role-gated and the org role claim is
// exactly 'manager'. The ErrActivityNotLoggable normalization mirrors the
// coverage precedent: a commercial activity without an anchored WG has no
// legitimate manager-stage writer.
func (s *Service) managerReach(ctx context.Context, orgID, activityID, unitID, ownerID, actorID uuid.UUID, role string) error {
	res, err := s.routing.ResolveManagerStage(ctx, orgID, activityID, unitID, ownerID)
	if err != nil {
		if errors.Is(err, activitydomain.ErrActivityNotLoggable) {
			return directiondomain.ErrForbidden
		}
		return err
	}
	if !res.RoleGated && !contains(res.ApproverIDs, actorID) {
		return directiondomain.ErrForbidden
	}
	if res.RoleGated && role != string(models.RoleManager) {
		return directiondomain.ErrForbidden
	}
	return nil
}

// ---------------------------------------------------------------------------
// Create — the gate chain (D-13-02/03/05/17/20, A5/A9/A10)
// ---------------------------------------------------------------------------

// Create enforces the pinned gate chain in order and delegates the insert
// (with supersede-on-create when requested) to the repo tx. Pool-level
// failures map to their sentinels — 400/403/404, never 500:
//
//  1. XOR fast-fail (D-13-05): exactly one of directed_to / wg_id set.
//  2. est_hours validation (D-13-03): when set, positive + whole-cent; the
//     scheduled shape (D-13-02): planned_date set requires est_hours.
//  3. Same-org activity (D-02 pattern): activityRepo.Get must resolve in-org.
//  4. WG rows (D-13-17, A5, Pitfall 9): queued-only (planned_date forbidden)
//     and the WG must be same-org with the scope predicate — the activity is
//     the WG's anchored activity OR the anchor sits in its ancestry.
//  5. Mode gate (D-13-19/20, A9): self_planned → only the employee creates
//     their own rows (D-S — no routing call); manager_planned → the routing
//     gate decides (strict reading: even self-direction must pass). WG rows
//     route on the anchored activity (A10).
//  6. Supersede fast-fail (CR-01 UX): the target must be draft|active per the
//     pool read; the repo re-checks under lock.
//
// The response carries the created row + the warnings overlay (D-13-03) —
// warnings never reject the write (D-13-28).
func (s *Service) Create(ctx context.Context, orgID, actorID uuid.UUID, role string, req *CreateDirectionRequest) (*directiondomain.Direction, []directiondomain.Warning, error) {
	// 1. XOR fast-fail (D-13-05).
	if (req.DirectedTo == nil) == (req.WgID == nil) {
		return nil, nil, directiondomain.ErrInvalidTarget
	}

	// 2. est_hours + scheduled shape (D-13-02/03).
	if req.EstHours != nil && !wholeCent(*req.EstHours) {
		return nil, nil, directiondomain.ErrInvalidHours
	}
	if req.PlannedDate != nil && req.EstHours == nil {
		return nil, nil, directiondomain.ErrInvalidHours
	}

	// 3. Same-org activity (D-02 pattern — ref validation: fetch errors and
	// cross-org ids are indistinguishable at this boundary).
	a, err := s.activityRepo.Get(ctx, orgID, req.ActivityID)
	if err != nil || a == nil || a.OrgID != orgID {
		return nil, nil, directiondomain.ErrInvalidRequest
	}

	// 4. WG rows: queued-only + same-org + scope predicate (D-13-17, A5).
	var wgActivityID uuid.UUID
	if req.WgID != nil {
		if req.PlannedDate != nil {
			return nil, nil, directiondomain.ErrInvalidRequest
		}
		g, err := s.wgRepo.GetByID(ctx, *req.WgID)
		if err != nil || g == nil || g.OrgID != orgID {
			return nil, nil, directiondomain.ErrInvalidRequest
		}
		inScope := req.ActivityID == g.SubprojectID
		if !inScope {
			ancestry, err := s.activityRepo.GetAncestry(ctx, req.ActivityID)
			if err != nil {
				return nil, nil, directiondomain.ErrInvalidRequest
			}
			for _, anc := range ancestry {
				if anc.ID == g.SubprojectID {
					inScope = true
					break
				}
			}
		}
		if !inScope {
			return nil, nil, directiondomain.ErrInvalidRequest
		}
		wgActivityID = g.SubprojectID
	}

	// 5. Permission gate: mode gate for user-targeted rows, routing gate for
	// the manager path and all WG rows.
	if req.DirectedTo != nil {
		mode, err := s.orgSettings.ResolvePlanningMode(ctx, orgID, *req.DirectedTo)
		if err != nil {
			return nil, nil, err
		}
		if mode == orgsettings.ModeSelfPlanned && *req.DirectedTo != actorID {
			return nil, nil, directiondomain.ErrForbidden
		}
		if mode == orgsettings.ModeManagerPlanned {
			unitID, err := s.primaryUnitIDFor(ctx, *req.DirectedTo)
			if err != nil {
				return nil, nil, err
			}
			if err := s.managerReach(ctx, orgID, req.ActivityID, unitID, *req.DirectedTo, actorID, role); err != nil {
				return nil, nil, err
			}
		}
		// self_planned self-direction: no routing call (D-S — no approval).
	} else {
		if err := s.managerReach(ctx, orgID, wgActivityID, uuid.Nil, uuid.Nil, actorID, role); err != nil {
			return nil, nil, err
		}
	}

	// 6. Supersede fast-fail (pool-level; the repo re-checks under lock —
	// CR-01): the target must be draft|active, else ErrInvalidTransition.
	if req.SupersedesID != nil {
		target, err := s.repo.Get(ctx, orgID, *req.SupersedesID)
		if err != nil {
			return nil, nil, err
		}
		if target.Status != directiondomain.StatusDraft && target.Status != directiondomain.StatusActive {
			return nil, nil, directiondomain.ErrInvalidTransition
		}
	}

	// 7. Row + created audit → the repo tx (BE-016: audit in the same tx).
	row := &directiondomain.Direction{
		ID:           uuid.New(),
		OrgID:        orgID,
		DirectedBy:   actorID,
		DirectedTo:   req.DirectedTo,
		WgID:         req.WgID,
		ActivityID:   req.ActivityID,
		PlannedDate:  req.PlannedDate,
		EstHours:     req.EstHours,
		Priority:     req.Priority,
		DueDate:      req.DueDate,
		Status:       directiondomain.StatusDraft,
		SupersedesID: req.SupersedesID,
	}
	actor := actorID
	created, err := s.repo.Create(ctx, orgID, row, req.SupersedesID, []*audit.AuditLog{{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   row.ID,
		Action:     directiondomain.AuditActionCreated,
		ActorID:    &actor,
		CreatedAt:  time.Now().UTC(),
	}})
	if err != nil {
		return nil, nil, err
	}

	// 8. Warnings overlay — scheduled user-targeted rows warn on the planned
	// day; never a rejection (D-13-03/28).
	var warnings []directiondomain.Warning
	if req.PlannedDate != nil && req.DirectedTo != nil {
		warnings, err = s.computeWarnings(ctx, orgID, []uuid.UUID{*req.DirectedTo}, *req.PlannedDate, *req.PlannedDate)
		if err != nil {
			return nil, nil, err
		}
	}
	return created, warnings, nil
}

// ---------------------------------------------------------------------------
// Lifecycle — Activate / Cancel / Unclaim (D-13-07/10/16)
// ---------------------------------------------------------------------------

// lifecycleAllowed is the activate/cancel/unclaim permission: the row's
// creator (DirectedBy) always, else the BE-014 manager reach resolved on the
// row's activity (managers may lifecycle rows they did not create). WG rows
// (DirectedTo nil) resolve with uuid.Nil unit/owner — the routing degrades
// to the anchored-WG approver set or the role-gated terminal stage.
func (s *Service) lifecycleAllowed(ctx context.Context, orgID, actorID uuid.UUID, role string, d *directiondomain.Direction) error {
	if d.DirectedBy == actorID {
		return nil
	}
	var unitID, owner uuid.UUID
	if d.DirectedTo != nil {
		uid, err := s.primaryUnitIDFor(ctx, *d.DirectedTo)
		if err != nil {
			return err
		}
		unitID = uid
		owner = *d.DirectedTo
	}
	return s.managerReach(ctx, orgID, d.ActivityID, unitID, owner, actorID, role)
}

// Activate transitions draft → active (explicit endpoint, OQ1). The matrix
// fast-fails at the pool level BEFORE the repo call (CR-01 UX — the repo
// re-validates under the FOR UPDATE lock); the permission is creator-or-
// manager-reach. The 'activated' audit row lands in the same tx.
func (s *Service) Activate(ctx context.Context, orgID, actorID uuid.UUID, role string, id uuid.UUID) (*directiondomain.Direction, error) {
	d, err := s.repo.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if !directiondomain.CanTransition(d.Status, directiondomain.StatusActive) {
		return nil, directiondomain.ErrInvalidTransition
	}
	if err := s.lifecycleAllowed(ctx, orgID, actorID, role, d); err != nil {
		return nil, err
	}
	actor := actorID
	return s.repo.Activate(ctx, orgID, id, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   id,
		Action:     directiondomain.AuditActionActivated,
		ActorID:    &actor,
		CreatedAt:  time.Now().UTC(),
	})
}

// Cancel transitions draft|active → cancelled with a mandatory reason
// (D-13-10): the reason requirement is enforced BEFORE the pool read, then
// the matrix fast-fail, then the creator-or-manager permission. The
// 'cancelled' audit carries {reason} (ADR-BE-018 §3).
func (s *Service) Cancel(ctx context.Context, orgID, actorID uuid.UUID, role string, id uuid.UUID, reason string) (*directiondomain.Direction, error) {
	if reason == "" {
		return nil, directiondomain.ErrCancelReasonRequired
	}
	d, err := s.repo.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if !directiondomain.CanTransition(d.Status, directiondomain.StatusCancelled) {
		return nil, directiondomain.ErrInvalidTransition
	}
	if err := s.lifecycleAllowed(ctx, orgID, actorID, role, d); err != nil {
		return nil, err
	}
	actor := actorID
	return s.repo.Cancel(ctx, orgID, id, reason, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   id,
		Action:     directiondomain.AuditActionCancelled,
		ActorID:    &actor,
		Payload:    map[string]any{"reason": reason},
		CreatedAt:  time.Now().UTC(),
	})
}

// Unclaim cancels a CLAIM row (D-13-16): the claim-row guard (only rows with
// origin_direction_id are unclaimable), the reason requirement, and the
// claimant (DirectedTo) / creator (DirectedBy) / manager gate. The unclaim
// audit is the 'cancelled' action with {reason} — the repo tx re-validates
// the claim-row guard + matrix under the FOR UPDATE lock (authoritative,
// CR-01); hours return to the WG budget automatically (Σ-derived).
func (s *Service) Unclaim(ctx context.Context, orgID, actorID uuid.UUID, role string, claimRowID uuid.UUID, reason string) (*directiondomain.Direction, error) {
	claim, err := s.repo.Get(ctx, orgID, claimRowID)
	if err != nil {
		return nil, err
	}
	if claim.OriginDirectionID == nil {
		return nil, directiondomain.ErrInvalidRequest
	}
	if reason == "" {
		return nil, directiondomain.ErrCancelReasonRequired
	}
	if claim.DirectedBy != actorID && (claim.DirectedTo == nil || *claim.DirectedTo != actorID) {
		if err := s.lifecycleAllowed(ctx, orgID, actorID, role, claim); err != nil {
			return nil, err
		}
	}
	actor := actorID
	return s.repo.Unclaim(ctx, orgID, claimRowID, reason, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   claimRowID,
		Action:     directiondomain.AuditActionCancelled,
		ActorID:    &actor,
		Payload:    map[string]any{"reason": reason},
		CreatedAt:  time.Now().UTC(),
	})
}

// ---------------------------------------------------------------------------
// Claim — the WG claim orchestration (D-13-11..16)
// ---------------------------------------------------------------------------

// Claim creates the claim row for an active WG row: pool fast-fails — the WG
// row exists + is active (ErrWgRowNotActive), the claimant is a WG member
// (ErrNotWgMember, D-13-12), the claimed hours are positive whole-cent
// (ErrInvalidHours). The repo's in-tx Σ guard under the WG-row FOR UPDATE
// lock is authoritative (D-13-13, CR-01) — these checks are fast-fail UX
// only. The 'claimed' audit carries {wg_row_id, est_hours} with uuid.Nil
// entity_id — the repo pins it to the claim row it creates (ADR-BE-018 §3,
// 13-05).
func (s *Service) Claim(ctx context.Context, orgID, actorID uuid.UUID, role string, wgRowID uuid.UUID, estHours float64) (*directiondomain.Direction, error) {
	wg, err := s.repo.Get(ctx, orgID, wgRowID)
	if err != nil {
		return nil, err
	}
	if wg.Status != directiondomain.StatusActive {
		return nil, directiondomain.ErrWgRowNotActive
	}
	members, err := s.wgRepo.ListMembers(ctx, *wg.WgID)
	if err != nil {
		return nil, err
	}
	member := false
	for _, m := range members {
		if m.UserID == actorID {
			member = true
			break
		}
	}
	if !member {
		return nil, directiondomain.ErrNotWgMember
	}
	if !wholeCent(estHours) {
		return nil, directiondomain.ErrInvalidHours
	}
	actor := actorID
	return s.repo.Claim(ctx, orgID, wgRowID, actorID, estHours, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: directiondomain.AuditEntityDirection,
		EntityID:   uuid.Nil, // the repo pins entity_id to the claim row (13-05)
		Action:     directiondomain.AuditActionClaimed,
		ActorID:    &actor,
		Payload:    map[string]any{"wg_row_id": wgRowID.String(), "est_hours": estHours},
		CreatedAt:  time.Now().UTC(),
	})
}

// PlanResponse is the plan read-model response (D-13-27): the rows with the
// derived-on-read states plus the warnings overlay (D-13-28) — the Phase 19
// contract from 13-UI-SPEC.
type PlanResponse struct {
	Rows     []directiondomain.PlanRow `json:"rows"`
	Warnings []directiondomain.Warning `json:"warnings"`
}

// CoverageTotals is the per-employee period aggregate of the coverage
// read-model (D-Z, D-13-25): Σ planned, Σ capacity and Σ gap over the
// queried period (including fully-absent days — the away day contributes its
// zero capacity, so the totals stay truthful).
type CoverageTotals struct {
	EmployeeID uuid.UUID `json:"employee_id"`
	Planned    float64   `json:"planned"`
	Capacity   float64   `json:"capacity"`
	Gap        float64   `json:"gap"`
}

// CoverageResponse is the direction-coverage read-model response (DIR-06,
// 13-UI-SPEC): the uncovered rows (fully-absent days excluded, D-13-26), the
// period totals per employee, and the warnings overlay (D-13-28/31).
type CoverageResponse struct {
	Rows     []directiondomain.CoverageRow `json:"rows"`
	Totals   []CoverageTotals              `json:"totals"`
	Warnings []directiondomain.Warning     `json:"warnings"`
}

// ListPlan returns the org plan read-model with the self-or-manager gate
// (checker fix, T-13-26): the org-wide view (employee_id omitted) is
// manager-only; a non-manager may read only their own plan (employee_id ==
// actorID). The warnings overlay is computed over the rows' employees/days.
func (s *Service) ListPlan(ctx context.Context, orgID, actorID uuid.UUID, role string, employeeID *uuid.UUID, periodStart, periodEnd time.Time) (PlanResponse, error) {
	if (employeeID == nil || *employeeID != actorID) && role != string(models.RoleManager) {
		return PlanResponse{}, directiondomain.ErrForbidden
	}
	rows, err := s.repo.ListPlan(ctx, orgID, employeeID, periodStart, periodEnd)
	if err != nil {
		return PlanResponse{}, err
	}
	employeeSet := make(map[uuid.UUID]bool)
	for _, r := range rows {
		if r.DirectedTo != nil {
			employeeSet[*r.DirectedTo] = true
		}
	}
	employeeIDs := make([]uuid.UUID, 0, len(employeeSet))
	for emp := range employeeSet {
		employeeIDs = append(employeeIDs, emp)
	}
	sort.Slice(employeeIDs, func(i, j int) bool { return employeeIDs[i].String() < employeeIDs[j].String() })
	warnings, err := s.computeWarnings(ctx, orgID, employeeIDs, periodStart, periodEnd)
	if err != nil {
		return PlanResponse{}, err
	}
	return PlanResponse{Rows: rows, Warnings: warnings}, nil
}

// Coverage resolves the DIR-06 read-model: the employee set by scope
// (employee | unit | wg — D-13-25), the validity split (validity-outside
// employees get the invalid warning and never reach the repo call, D-13-31),
// the fully-absent-day exclusion from uncovered surfacing (D-13-26), and the
// per-employee period totals (D-Z). Scope gates (checker fix, T-13-26):
// unit/wg scopes are manager-only; the employee scope allows the non-manager
// self-view (scope_id == actorID).
func (s *Service) Coverage(ctx context.Context, orgID, actorID uuid.UUID, role, scope, scopeID string, periodStart, periodEnd time.Time) (CoverageResponse, error) {
	employees, err := s.resolveScopeEmployees(ctx, orgID, actorID, role, scope, scopeID)
	if err != nil {
		return CoverageResponse{}, err
	}

	// Validity split (D-13-31): validity-outside employees are dropped from
	// the repo call but still feed the warning overlay (invalid warning).
	valid := make([]uuid.UUID, 0, len(employees))
	for _, emp := range employees {
		m, err := s.orgRepo.GetMembership(ctx, emp, orgID)
		if err != nil {
			return CoverageResponse{}, err
		}
		if membershipValid(m, periodStart, periodEnd) {
			valid = append(valid, emp)
		}
	}

	rows, err := s.repo.Coverage(ctx, orgID, valid, periodStart, periodEnd)
	if err != nil {
		return CoverageResponse{}, err
	}

	warnings, err := s.computeWarnings(ctx, orgID, employees, periodStart, periodEnd)
	if err != nil {
		return CoverageResponse{}, err
	}

	// D-13-26: fully-absent days (capacity 0) are excluded from the uncovered
	// surfacing; the row data stays available for the warning overlay.
	uncovered := make([]directiondomain.CoverageRow, 0, len(rows))
	for _, r := range rows {
		if r.Capacity > 0 {
			uncovered = append(uncovered, r)
		}
	}

	// Period totals per employee over the FULL row set (the away day keeps
	// its zero capacity — D-Z aggregates stay truthful).
	totalsByEmp := make(map[uuid.UUID]*CoverageTotals)
	for _, r := range rows {
		t := totalsByEmp[r.EmployeeID]
		if t == nil {
			t = &CoverageTotals{EmployeeID: r.EmployeeID}
			totalsByEmp[r.EmployeeID] = t
		}
		t.Planned += r.Planned
		t.Capacity += r.Capacity
		t.Gap += r.Gap
	}
	totals := make([]CoverageTotals, 0, len(totalsByEmp))
	for _, t := range totalsByEmp {
		totals = append(totals, *t)
	}
	sort.Slice(totals, func(i, j int) bool { return totals[i].EmployeeID.String() < totals[j].EmployeeID.String() })

	return CoverageResponse{Rows: uncovered, Totals: totals, Warnings: warnings}, nil
}

// resolveScopeEmployees resolves the DIR-06 employee set by scope (D-13-25,
// A6): employee → the one employee; unit → the unit's members plus every
// descendant unit's members; wg → the WG's members. Scope gates (T-13-26):
// unit/wg are manager-only; employee allows the non-manager self-view.
func (s *Service) resolveScopeEmployees(ctx context.Context, orgID, actorID uuid.UUID, role, scope, scopeID string) ([]uuid.UUID, error) {
	set := make(map[uuid.UUID]bool)
	switch scope {
	case "employee":
		if role != string(models.RoleManager) && scopeID != actorID.String() {
			return nil, directiondomain.ErrForbidden
		}
		empID, err := uuid.Parse(scopeID)
		if err != nil {
			return nil, directiondomain.ErrInvalidRequest
		}
		set[empID] = true
	case "unit":
		if role != string(models.RoleManager) {
			return nil, directiondomain.ErrForbidden
		}
		members, err := s.unitRepo.ListMembers(ctx, scopeID)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			set[m.UserID] = true
		}
		descendants, err := s.unitRepo.GetDescendants(ctx, scopeID)
		if err != nil {
			return nil, err
		}
		for _, d := range descendants {
			dm, err := s.unitRepo.ListMembers(ctx, d.ID)
			if err != nil {
				return nil, err
			}
			for _, m := range dm {
				set[m.UserID] = true
			}
		}
	case "wg":
		if role != string(models.RoleManager) {
			return nil, directiondomain.ErrForbidden
		}
		wgID, err := uuid.Parse(scopeID)
		if err != nil {
			return nil, directiondomain.ErrInvalidRequest
		}
		members, err := s.wgRepo.ListMembers(ctx, wgID)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			set[m.UserID] = true
		}
	default:
		return nil, directiondomain.ErrInvalidRequest
	}
	out := make([]uuid.UUID, 0, len(set))
	for emp := range set {
		out = append(out, emp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

// ---------------------------------------------------------------------------
// Warning overlay — pure function, never blocking (D-13-28/30/31)
// ---------------------------------------------------------------------------

// dayFormat is the pinned message day layout (13-UI-SPEC Copywriting
// Contract): "2 Jan" style — "14 Aug".
const dayFormat = "2 Jan"

// enDash joins the away range days (13-UI-SPEC example "Away 10–21 Aug").
const enDash = "\u2013"

// sameDay reports whether a and b fall on the same calendar day (the
// read-model day semantics are timezone-free — UTC midnight, 13-06).
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// dayBefore reports whether day d falls strictly before t (date-only).
func dayBefore(d, t time.Time) bool {
	dy, dm, dd := d.Date()
	ty, tm, td := t.Date()
	if dy != ty {
		return dy < ty
	}
	if dm != tm {
		return dm < tm
	}
	return dd < td
}

// dayAfter reports whether day d falls strictly after t (date-only).
func dayAfter(d, t time.Time) bool {
	dy, dm, dd := d.Date()
	ty, tm, td := t.Date()
	if dy != ty {
		return dy > ty
	}
	if dm != tm {
		return dm > tm
	}
	return dd > td
}

// membershipValid reports whether the membership's employment validity
// window (D-2, migration 012) intersects the queried period (D-13-31):
// nil membership (not an org employee) or a fully non-overlapping window
// makes the employee validity-outside.
func membershipValid(m *auth.OrganizationMembership, periodStart, periodEnd time.Time) bool {
	if m == nil {
		return false
	}
	// The period is fully outside the window when it ends before valid_from
	// or starts after valid_until (boundaries inclusive — a day equal to a
	// boundary is valid).
	if m.ValidFrom != nil && dayBefore(periodEnd, *m.ValidFrom) {
		return false
	}
	if m.ValidUntil != nil && dayAfter(periodStart, *m.ValidUntil) {
		return false
	}
	return true
}

// computeWarnings is the shared warning overlay (D-13-28/30/31): the single
// channel for away | partial | over-capacity | invalid warnings, consumed by
// the create response AND the read-model responses (never duplicated). It is
// a pure function over repo data — AbsenceWindows (declared + confirmed,
// D-13-29), the coverage read-model (planned vs capacity) and the membership
// validity window. Messages are pre-rendered server-side verbatim; identical
// (type, message) pairs are deduplicated (the invalid message carries no
// day, so a fully-invalid period yields one warning per employee).
func (s *Service) computeWarnings(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, start, end time.Time) ([]directiondomain.Warning, error) {
	windows, err := s.repo.AbsenceWindows(ctx, orgID, employeeIDs, start, end)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Coverage(ctx, orgID, employeeIDs, start, end)
	if err != nil {
		return nil, err
	}
	// Index the coverage rows per (employee, day) for the over-capacity check.
	type dayKey struct {
		emp uuid.UUID
		day time.Time
	}
	coverageByDay := make(map[dayKey]directiondomain.CoverageRow, len(rows))
	for _, r := range rows {
		coverageByDay[dayKey{emp: r.EmployeeID, day: r.Date}] = r
	}

	var warnings []directiondomain.Warning
	seen := make(map[string]bool)
	appendUnique := func(w directiondomain.Warning) {
		key := w.Type + "\x00" + w.Message
		if seen[key] {
			return
		}
		seen[key] = true
		warnings = append(warnings, w)
	}

	// away: one warning per full-absence window (the range message covers the
	// window's intersection with the queried period).
	for _, w := range windows {
		if w.Hours != nil {
			continue
		}
		wStart, wEnd := w.StartsOn, w.EndsOn
		if dayBefore(wStart, start) {
			wStart = start
		}
		if dayAfter(wEnd, end) {
			wEnd = end
		}
		if dayAfter(wStart, wEnd) {
			continue // window fully outside the period
		}
		msg := "Away " + wStart.Format(dayFormat)
		if !sameDay(wStart, wEnd) {
			msg += enDash + wEnd.Format(dayFormat)
		}
		appendUnique(directiondomain.Warning{Type: directiondomain.WarningAway, Message: msg})
	}

	// per-day: invalid / partial / over-capacity.
	for _, emp := range employeeIDs {
		membership, err := s.orgRepo.GetMembership(ctx, emp, orgID)
		if err != nil {
			return nil, err
		}
		for d := start; !dayAfter(d, end); d = d.AddDate(0, 0, 1) {
			if !membershipValid(membership, d, d) {
				appendUnique(directiondomain.Warning{Type: directiondomain.WarningInvalid, Message: "Outside validity period"})
				continue
			}
			for _, w := range windows {
				if w.EmployeeID != emp || w.Hours == nil {
					continue
				}
				if !dayBefore(d, w.StartsOn) && !dayAfter(d, w.EndsOn) {
					appendUnique(directiondomain.Warning{Type: directiondomain.WarningPartial, Message: "Partial " + d.Format(dayFormat)})
					break
				}
			}
			if row, ok := coverageByDay[dayKey{emp: emp, day: d}]; ok && row.Planned > row.Capacity {
				appendUnique(directiondomain.Warning{Type: directiondomain.WarningOverCapacity, Message: "Over capacity " + d.Format(dayFormat)})
			}
		}
	}

	sort.SliceStable(warnings, func(i, j int) bool { // deterministic output
		if warnings[i].Type != warnings[j].Type {
			return warnings[i].Type < warnings[j].Type
		}
		return warnings[i].Message < warnings[j].Message
	})
	return warnings, nil
}
