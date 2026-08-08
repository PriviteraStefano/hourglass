package testdata

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockCoverageRepo implements ports.CoverageRepository for service unit
// tests (mock_ticket_repo pattern).
//
// State:
//   - Allocations keyed by entry id — ReplaceAllocations stores the full set
//     there; ListByEntry reads it back.
//   - Snapshots keyed by close id — ClosePeriod stores the primed
//     ClosePeriodResult under Snapshots[closeID]; GetSnapshot reads it back.
//   - Audits captures every audit row handed to a mutator
//     (ReplaceAllocations / ClosePeriod), so tests assert both the state
//     change and the in-tx audit payload without a DB.
//
// Behavior knobs, in precedence order per method: ErrOn* error knob first,
// then the per-method Fn override, then the deterministic default.
type MockCoverageRepo struct {
	mu          sync.Mutex
	Allocations map[uuid.UUID][]*coverage.CoverageAllocation
	Snapshots   map[uuid.UUID]*coverage.PeriodClose
	Audits      []*audit.AuditLog

	// Error knobs: when set, the method returns this error immediately.
	ErrOnReplaceAllocations error
	ErrOnListByEntry        error
	ErrOnToCoverQueue       error
	ErrOnBucketBalance      error
	ErrOnClosePeriod        error
	ErrOnGetSnapshot        error
	ErrOnListHistory        error

	// Per-method Fn overrides (take precedence over defaults).
	ReplaceAllocationsFn func(ctx context.Context, orgID, entryID uuid.UUID, allocs []*coverage.CoverageAllocation, auditLog *audit.AuditLog) ([]*coverage.CoverageAllocation, error)
	ListByEntryFn        func(ctx context.Context, orgID, entryID uuid.UUID) ([]*coverage.CoverageAllocation, error)
	ToCoverQueueFn       func(ctx context.Context, orgID uuid.UUID) ([]coverage.ToCoverQueueRow, error)
	BucketBalanceFn      func(ctx context.Context, orgID, contractID uuid.UUID) (float64, error)
	ClosePeriodFn        func(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time, closeID, closedBy uuid.UUID, auditLog *audit.AuditLog) (*coverage.PeriodClose, error)
	GetSnapshotFn        func(ctx context.Context, orgID, closeID uuid.UUID) (*coverage.PeriodClose, error)
	ListHistoryFn        func(ctx context.Context, orgID, entryID uuid.UUID) ([]audit.AuditLog, error)

	// Default-behavior result knobs.
	BucketBalanceResult    float64                    // BucketBalance default return
	ToCoverQueueRows       []coverage.ToCoverQueueRow // ToCoverQueue default return
	ClosePeriodResult      *coverage.PeriodClose      // ClosePeriod default: stored under Snapshots[closeID] and returned
	ClosePeriodOverlapping bool                       // when true, ClosePeriod default returns coverage.ErrPeriodAlreadyClosed
}

var _ ports.CoverageRepository = (*MockCoverageRepo)(nil)

func (m *MockCoverageRepo) ReplaceAllocations(ctx context.Context, orgID, entryID uuid.UUID, allocs []*coverage.CoverageAllocation, auditLog *audit.AuditLog) ([]*coverage.CoverageAllocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnReplaceAllocations != nil {
		return nil, m.ErrOnReplaceAllocations
	}
	if m.ReplaceAllocationsFn != nil {
		return m.ReplaceAllocationsFn(ctx, orgID, entryID, allocs, auditLog)
	}
	if m.Allocations == nil {
		m.Allocations = make(map[uuid.UUID][]*coverage.CoverageAllocation)
	}
	m.Allocations[entryID] = allocs
	if auditLog != nil {
		m.Audits = append(m.Audits, auditLog)
	}
	return allocs, nil
}

func (m *MockCoverageRepo) ListByEntry(ctx context.Context, orgID, entryID uuid.UUID) ([]*coverage.CoverageAllocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnListByEntry != nil {
		return nil, m.ErrOnListByEntry
	}
	if m.ListByEntryFn != nil {
		return m.ListByEntryFn(ctx, orgID, entryID)
	}
	return m.Allocations[entryID], nil
}

func (m *MockCoverageRepo) ToCoverQueue(ctx context.Context, orgID uuid.UUID) ([]coverage.ToCoverQueueRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnToCoverQueue != nil {
		return nil, m.ErrOnToCoverQueue
	}
	if m.ToCoverQueueFn != nil {
		return m.ToCoverQueueFn(ctx, orgID)
	}
	return m.ToCoverQueueRows, nil
}

func (m *MockCoverageRepo) BucketBalance(ctx context.Context, orgID, contractID uuid.UUID) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnBucketBalance != nil {
		return 0, m.ErrOnBucketBalance
	}
	if m.BucketBalanceFn != nil {
		return m.BucketBalanceFn(ctx, orgID, contractID)
	}
	return m.BucketBalanceResult, nil
}

func (m *MockCoverageRepo) ClosePeriod(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time, closeID, closedBy uuid.UUID, auditLog *audit.AuditLog) (*coverage.PeriodClose, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnClosePeriod != nil {
		return nil, m.ErrOnClosePeriod
	}
	if m.ClosePeriodFn != nil {
		return m.ClosePeriodFn(ctx, orgID, periodStart, periodEnd, closeID, closedBy, auditLog)
	}
	if m.ClosePeriodOverlapping {
		return nil, coverage.ErrPeriodAlreadyClosed
	}
	if m.Snapshots == nil {
		m.Snapshots = make(map[uuid.UUID]*coverage.PeriodClose)
	}
	var stored *coverage.PeriodClose
	if m.ClosePeriodResult != nil {
		cp := *m.ClosePeriodResult
		stored = &cp
	} else {
		stored = &coverage.PeriodClose{ID: closeID, OrgID: orgID, PeriodStart: periodStart, PeriodEnd: periodEnd, ClosedBy: closedBy, ClosedAt: time.Now().UTC()}
	}
	m.Snapshots[closeID] = stored
	if auditLog != nil {
		m.Audits = append(m.Audits, auditLog)
	}
	return stored, nil
}

func (m *MockCoverageRepo) GetSnapshot(ctx context.Context, orgID, closeID uuid.UUID) (*coverage.PeriodClose, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnGetSnapshot != nil {
		return nil, m.ErrOnGetSnapshot
	}
	if m.GetSnapshotFn != nil {
		return m.GetSnapshotFn(ctx, orgID, closeID)
	}
	s, ok := m.Snapshots[closeID]
	if !ok || s.OrgID != orgID {
		return nil, coverage.ErrNotFound
	}
	return s, nil
}

func (m *MockCoverageRepo) ListHistory(ctx context.Context, orgID, entryID uuid.UUID) ([]audit.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrOnListHistory != nil {
		return nil, m.ErrOnListHistory
	}
	if m.ListHistoryFn != nil {
		return m.ListHistoryFn(ctx, orgID, entryID)
	}
	var out []audit.AuditLog
	for _, a := range m.Audits {
		if a.EntityID == entryID && a.OrgID == orgID {
			out = append(out, *a)
		}
	}
	return out, nil
}
