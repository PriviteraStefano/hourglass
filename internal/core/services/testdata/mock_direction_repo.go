package testdata

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockDirectionRepo implements ports.DirectionRepository for service unit
// tests (13-07) and the activity-service origin-fallback tests (13-08).
//
// Default behaviors:
//   - Get serves the Directions map org-scoped — a missing pair (or a
//     cross-org id) surfaces direction.ErrDirectionNotFound, which the
//     service fast-fails (13-07) rely on.
//   - Create assigns an ID when empty, stores the row and returns it.
//   - Claim builds the A8 claim row (draft, queued, priority/due_date
//     copied from the WG row, directed_by = WG creator, directed_to =
//     claimant, origin_direction_id = wgRowID) and appends it to Claims.
//   - ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs return the
//     configured stub data (set via the Set* helpers; empty/nil by
//     default).
//
// Every mutator captures the audit row(s) it was handed in Audits so tests
// can assert both the state change and the in-tx audit payload without a
// DB. Every method has a per-method Fn override (GetFn-style) for tests
// that need a non-derived answer.
type MockDirectionRepo struct {
	mu         sync.Mutex
	Directions map[uuid.UUID]*directiondomain.Direction
	// Claims captures every claim row produced by the default Claim.
	Claims []*directiondomain.Direction

	// Audit capture: every audit row passed to a mutator lands here.
	Audits []*audit.AuditLog

	// Stub data for the read-methods (set via the Set* helpers).
	PlanRows      []directiondomain.PlanRow
	CoverageRows  []directiondomain.CoverageRow
	Windows       []directiondomain.AbsenceWindow
	DirectionRefs *directiondomain.DirectionRefs

	// Per-method overrides (GetFn-style; nil = default behavior).
	GetFn                func(ctx context.Context, orgID, id uuid.UUID) (*directiondomain.Direction, error)
	CreateFn             func(ctx context.Context, orgID uuid.UUID, d *directiondomain.Direction, supersedesID *uuid.UUID, audits []*audit.AuditLog) (*directiondomain.Direction, error)
	ActivateFn           func(ctx context.Context, orgID, id uuid.UUID, audit *audit.AuditLog) (*directiondomain.Direction, error)
	CancelFn             func(ctx context.Context, orgID, id uuid.UUID, reason string, audit *audit.AuditLog) (*directiondomain.Direction, error)
	UnclaimFn            func(ctx context.Context, orgID, claimRowID uuid.UUID, reason string, audit *audit.AuditLog) (*directiondomain.Direction, error)
	ClaimFn              func(ctx context.Context, orgID, wgRowID, claimantID uuid.UUID, estHours float64, audit *audit.AuditLog) (*directiondomain.Direction, error)
	ListPlanFn           func(ctx context.Context, orgID uuid.UUID, employeeID *uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.PlanRow, error)
	CoverageFn           func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.CoverageRow, error)
	AbsenceWindowsFn     func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, start, end time.Time) ([]directiondomain.AbsenceWindow, error)
	FirstDirectionRefsFn func(ctx context.Context, orgID, activityID uuid.UUID) (*directiondomain.DirectionRefs, error)
}

var _ ports.DirectionRepository = (*MockDirectionRepo)(nil)

// SetPlanRows configures the ListPlan stub output.
func (m *MockDirectionRepo) SetPlanRows(rows []directiondomain.PlanRow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlanRows = rows
}

// SetCoverageRows configures the Coverage stub output.
func (m *MockDirectionRepo) SetCoverageRows(rows []directiondomain.CoverageRow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CoverageRows = rows
}

// SetAbsenceWindows configures the AbsenceWindows stub output.
func (m *MockDirectionRepo) SetAbsenceWindows(windows []directiondomain.AbsenceWindow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Windows = windows
}

// SetDirectionRefs configures the FirstDirectionRefs stub output (nil =
// "no such row" per D-13-33).
func (m *MockDirectionRepo) SetDirectionRefs(refs *directiondomain.DirectionRefs) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DirectionRefs = refs
}

func (m *MockDirectionRepo) Get(ctx context.Context, orgID, id uuid.UUID) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetFn != nil {
		return m.GetFn(ctx, orgID, id)
	}
	d, ok := m.Directions[id]
	if !ok || d.OrgID != orgID {
		return nil, directiondomain.ErrDirectionNotFound
	}
	return d, nil
}

func (m *MockDirectionRepo) Create(ctx context.Context, orgID uuid.UUID, d *directiondomain.Direction, supersedesID *uuid.UUID, audits []*audit.AuditLog) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateFn != nil {
		return m.CreateFn(ctx, orgID, d, supersedesID, audits)
	}
	if m.Directions == nil {
		m.Directions = make(map[uuid.UUID]*directiondomain.Direction)
	}
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	d.OrgID = orgID
	m.Directions[d.ID] = d
	for _, a := range audits {
		if a != nil {
			m.Audits = append(m.Audits, a)
		}
	}
	return d, nil
}

func (m *MockDirectionRepo) Activate(ctx context.Context, orgID, id uuid.UUID, a *audit.AuditLog) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ActivateFn != nil {
		return m.ActivateFn(ctx, orgID, id, a)
	}
	d, ok := m.Directions[id]
	if !ok || d.OrgID != orgID {
		return nil, directiondomain.ErrDirectionNotFound
	}
	d.Status = directiondomain.StatusActive
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return d, nil
}

func (m *MockDirectionRepo) Cancel(ctx context.Context, orgID, id uuid.UUID, reason string, a *audit.AuditLog) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CancelFn != nil {
		return m.CancelFn(ctx, orgID, id, reason, a)
	}
	d, ok := m.Directions[id]
	if !ok || d.OrgID != orgID {
		return nil, directiondomain.ErrDirectionNotFound
	}
	d.Status = directiondomain.StatusCancelled
	d.Reason = &reason
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return d, nil
}

func (m *MockDirectionRepo) Claim(ctx context.Context, orgID, wgRowID, claimantID uuid.UUID, estHours float64, a *audit.AuditLog) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ClaimFn != nil {
		return m.ClaimFn(ctx, orgID, wgRowID, claimantID, estHours, a)
	}
	wg, ok := m.Directions[wgRowID]
	if !ok || wg.OrgID != orgID {
		return nil, directiondomain.ErrDirectionNotFound
	}
	// A8 claim row: draft, queued (planned_date NULL), copying
	// priority/due_date from the WG row; directed_by = WG creator
	// (manager attribution preserved, D-13-11).
	claim := &directiondomain.Direction{
		OrgID:             orgID,
		DirectedBy:        wg.DirectedBy,
		DirectedTo:        &claimantID,
		ActivityID:        wg.ActivityID,
		EstHours:          &estHours,
		Priority:          wg.Priority,
		DueDate:           wg.DueDate,
		Status:            directiondomain.StatusDraft,
		OriginDirectionID: &wgRowID,
	}
	if claim.ID == uuid.Nil {
		claim.ID = uuid.New()
	}
	if m.Directions == nil {
		m.Directions = make(map[uuid.UUID]*directiondomain.Direction)
	}
	m.Directions[claim.ID] = claim
	m.Claims = append(m.Claims, claim)
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return claim, nil
}

// Unclaim cancels a CLAIM row (D-13-16): the same cancel semantics plus the
// claim-row guard — a row without origin_direction_id is rejected with
// ErrInvalidRequest (the service fast-fails the same shape at the pool
// level; this guard is the mock's authoritative line).
func (m *MockDirectionRepo) Unclaim(ctx context.Context, orgID, claimRowID uuid.UUID, reason string, a *audit.AuditLog) (*directiondomain.Direction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UnclaimFn != nil {
		return m.UnclaimFn(ctx, orgID, claimRowID, reason, a)
	}
	d, ok := m.Directions[claimRowID]
	if !ok || d.OrgID != orgID {
		return nil, directiondomain.ErrDirectionNotFound
	}
	if d.OriginDirectionID == nil {
		return nil, directiondomain.ErrInvalidRequest
	}
	d.Status = directiondomain.StatusCancelled
	d.Reason = &reason
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return d, nil
}

func (m *MockDirectionRepo) ListPlan(ctx context.Context, orgID uuid.UUID, employeeID *uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.PlanRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ListPlanFn != nil {
		return m.ListPlanFn(ctx, orgID, employeeID, periodStart, periodEnd)
	}
	return m.PlanRows, nil
}

func (m *MockDirectionRepo) Coverage(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, periodStart, periodEnd time.Time) ([]directiondomain.CoverageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CoverageFn != nil {
		return m.CoverageFn(ctx, orgID, employeeIDs, periodStart, periodEnd)
	}
	return m.CoverageRows, nil
}

func (m *MockDirectionRepo) AbsenceWindows(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, start, end time.Time) ([]directiondomain.AbsenceWindow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.AbsenceWindowsFn != nil {
		return m.AbsenceWindowsFn(ctx, orgID, employeeIDs, start, end)
	}
	return m.Windows, nil
}

func (m *MockDirectionRepo) FirstDirectionRefs(ctx context.Context, orgID, activityID uuid.UUID) (*directiondomain.DirectionRefs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FirstDirectionRefsFn != nil {
		return m.FirstDirectionRefsFn(ctx, orgID, activityID)
	}
	return m.DirectionRefs, nil
}
