package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
)

// CoverageRepository is the coverage plane's persistence surface
// (ADR-P-012, ADR-BE-017): the allocation ledger, derived bucket balances,
// the to-cover queue read-model, and the period-close snapshot pair.
//
// Write shape (D-07): ReplaceAllocations is the ONLY write method — the
// atomic replace-set. There is deliberately no incremental CRUD
// (CreateAllocation/UpdateAllocation/DeleteAllocation): the Σ = entry hours
// invariant holds by construction, with no partial states. The replace-set
// re-validates the invariant inside its transaction under the FOR UPDATE
// entry-row lock (CR-01 lesson) — pool-level service checks are fast-fail UX
// only; the in-tx check is the correctness guarantee.
//
// Every mutator (ReplaceAllocations, ClosePeriod) writes its audit_logs row
// IN THE SAME TRANSACTION as the state write (BE-016, Pitfall 2): the caller
// passes the audit row(s) to write. The interface is append-only by
// construction — balances are derived on read (D-02), proposals compute on
// read (D-I), only confirmed allocations and snapshots persist.
type CoverageRepository interface {
	// ReplaceAllocations atomically replaces the full allocation set for an
	// entry (D-07): delete-all + insert-set + audit in one transaction, with
	// the Σ re-validation under the FOR UPDATE entry lock as the repo's
	// contract. Returns the stored set.
	ReplaceAllocations(ctx context.Context, orgID, entryID uuid.UUID, allocs []*coverage.CoverageAllocation, auditLog *audit.AuditLog) ([]*coverage.CoverageAllocation, error)

	// ListByEntry returns the current allocation set for an entry (empty
	// slice when none exist).
	ListByEntry(ctx context.Context, orgID, entryID uuid.UUID) ([]*coverage.CoverageAllocation, error)

	// ToCoverQueue returns the raw uncovered rows for the org (D-06, COV-01):
	// every approved, non-deleted 'time' entry with Σ allocations < entry
	// hours, including no-source entries. Proposal enrichment is service-side
	// (D-06) — the repo supplies the raw covered/uncovered split only.
	ToCoverQueue(ctx context.Context, orgID uuid.UUID) ([]coverage.ToCoverQueueRow, error)

	// BucketBalance returns the derived support-bucket balance for a contract
	// (D-02): sold_hours − Σ allocations drawn from it, computed on read —
	// never stored. Negative balances are allowed (D-03); the report is the
	// control, not a gate.
	BucketBalance(ctx context.Context, orgID, contractID uuid.UUID) (float64, error)

	// ClosePeriod writes the frozen period-close snapshot (D-10/D-11/D-12,
	// COV-04): entry-level rows for entries whose entry_date falls in
	// [periodStart, periodEnd], with allocations as they stand at close. The
	// caller (service) supplies the close id (uuid.New()) and the repo
	// inserts the header with it in-tx; a duplicate/overlapping period
	// surfaces coverage.ErrPeriodAlreadyClosed (A6, 409). Returns the full
	// snapshot incl. rows in one call (OQ4).
	ClosePeriod(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time, closeID, closedBy uuid.UUID, auditLog *audit.AuditLog) (*coverage.PeriodClose, error)

	// GetSnapshot returns a frozen period-close snapshot (header + rows) by
	// close id.
	GetSnapshot(ctx context.Context, orgID, closeID uuid.UUID) (*coverage.PeriodClose, error)

	// ListHistory returns the audit stream for an entry, filtered to
	// entity_type='coverage_allocation' (A7) — the append-only trail behind
	// every allocation change.
	ListHistory(ctx context.Context, orgID, entryID uuid.UUID) ([]audit.AuditLog, error)
}
