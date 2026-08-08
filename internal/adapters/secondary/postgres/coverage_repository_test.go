package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	"github.com/stretchr/testify/require"
)

// contractAllocation builds a contract-draw allocation row for the tests.
func contractAllocation(hours float64, contractID uuid.UUID) *coverage.CoverageAllocation {
	return &coverage.CoverageAllocation{
		ID:         uuid.New(),
		SourceType: coverage.SourceTypeContract,
		ContractID: &contractID,
		Hours:      hours,
	}
}

// coverageSetAudit builds the 'allocations-set' audit row the service passes
// to the repo (entity_type pinned to coverage.AuditEntityCoverageAllocation).
func coverageSetAudit(orgID, entryID, actorID uuid.UUID, payload map[string]any, now time.Time) *audit.AuditLog {
	return &audit.AuditLog{
		OrgID:      orgID,
		EntityType: coverage.AuditEntityCoverageAllocation,
		EntityID:   entryID,
		Action:     coverage.AuditActionAllocationsSet,
		ActorID:    &actorID,
		Payload:    payload,
		CreatedAt:  now,
	}
}

func sumHours(t *testing.T, allocs []*coverage.CoverageAllocation) float64 {
	t.Helper()
	var sum float64
	for _, a := range allocs {
		sum += a.Hours
	}
	return sum
}

// ---------------------------------------------------------------------------
// ReplaceAllocations — the atomic replace-set with the FOR UPDATE in-tx
// re-validation (CR-01 closure, T-12-14).
// ---------------------------------------------------------------------------

func TestCoverageRepository_ReplaceAllocations_CommitsAndReadsBack(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, "approved", now)

	allocs := []*coverage.CoverageAllocation{
		contractAllocation(5, contractID),
		contractAllocation(3, contractID),
	}
	returned, err := repo.ReplaceAllocations(ctx, orgID, entryID, allocs,
		coverageSetAudit(orgID, entryID, userID, map[string]any{"set": "A"}, now))
	require.NoError(t, err)
	require.Len(t, returned, 2)
	require.Equal(t, 8.0, sumHours(t, returned))

	// Reads back exactly the stored set, org-scoped.
	stored, err := repo.ListByEntry(ctx, orgID, entryID)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	require.Equal(t, 8.0, sumHours(t, stored))
	for _, a := range stored {
		require.Equal(t, coverage.EntryTypeTime, a.EntryType)
		require.Equal(t, orgID, a.OrgID)
		require.Equal(t, contractID, *a.ContractID)
		require.Equal(t, coverage.SourceTypeContract, a.SourceType)
	}

	// The audit row was written in the same tx.
	var auditCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'coverage_allocation' AND entity_id = $1 AND action = 'allocations-set'`,
		entryID).Scan(&auditCount)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}

func TestCoverageRepository_ReplaceAllocations_SumMismatchLeavesNoRows(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, "approved", now)

	// A valid set first, so the mismatch test proves the rollback leaves the
	// PREVIOUS state (not just zero rows on a first write).
	_, err := repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(8, contractID)},
		coverageSetAudit(orgID, entryID, userID, nil, now))
	require.NoError(t, err)

	// Σ = 7 ≠ 8 → rejected in-tx; nothing changed.
	_, err = repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(5, contractID), contractAllocation(2, contractID)},
		coverageSetAudit(orgID, entryID, userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrAllocationSumMismatch)

	stored, err := repo.ListByEntry(ctx, orgID, entryID)
	require.NoError(t, err)
	require.Len(t, stored, 1, "the failed replace must leave the previous committed set intact")
	require.Equal(t, 8.0, sumHours(t, stored))

	// No audit row was written for the failed replace (partial states are
	// impossible — state write and audit row share the tx).
	var auditCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'coverage_allocation' AND entity_id = $1`,
		entryID).Scan(&auditCount)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount, "only the successful replace's audit row exists")
}

func TestCoverageRepository_ReplaceAllocations_RejectsNotCoverable(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	allocs := []*coverage.CoverageAllocation{contractAllocation(8, contractID)}

	cases := []struct {
		name   string
		status string
	}{
		{"draft entry", "draft"},
		{"pending_manager entry", "pending_manager"},
		{"rejected entry", "rejected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, tc.status, now)
			_, err := repo.ReplaceAllocations(ctx, orgID, entryID, allocs,
				coverageSetAudit(orgID, entryID, userID, nil, now))
			require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
		})
	}

	// Deleted entry → not coverable.
	deletedID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, "approved", now)
	_, err := pool.Exec(ctx, `UPDATE time_entries SET is_deleted = true WHERE id = $1`, deletedID)
	require.NoError(t, err)
	_, err = repo.ReplaceAllocations(ctx, orgID, deletedID, allocs,
		coverageSetAudit(orgID, deletedID, userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)

	// Cross-org entry → not coverable (the org_id filter fails the lock
	// SELECT the same way a missing row would).
	otherOrgID := seedOrg(t, pool, now)
	otherUserID := seedUser(t, pool, now)
	otherEntryID := seedTimeEntry(t, pool, otherOrgID, otherUserID, activityID, unitID, 8, now, "approved", now)
	_, err = repo.ReplaceAllocations(ctx, orgID, otherEntryID, allocs,
		coverageSetAudit(orgID, otherEntryID, userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
}

// ---------------------------------------------------------------------------
// ListByEntry
// ---------------------------------------------------------------------------

func TestCoverageRepository_ListByEntry_ScopedToEntry(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryA := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, "approved", now)
	entryB := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 4, now.Add(time.Hour), "approved", now)

	_, err := repo.ReplaceAllocations(ctx, orgID, entryA,
		[]*coverage.CoverageAllocation{contractAllocation(8, contractID)},
		coverageSetAudit(orgID, entryA, userID, nil, now))
	require.NoError(t, err)

	// Only entry A's rows are visible.
	storedA, err := repo.ListByEntry(ctx, orgID, entryA)
	require.NoError(t, err)
	require.Len(t, storedA, 1)

	storedB, err := repo.ListByEntry(ctx, orgID, entryB)
	require.NoError(t, err)
	require.Empty(t, storedB, "an entry without allocations reads back as an empty set")

	// Cross-org scoping: another org cannot see the rows.
	otherOrgID := seedOrg(t, pool, now)
	storedOther, err := repo.ListByEntry(ctx, otherOrgID, entryA)
	require.NoError(t, err)
	require.Empty(t, storedOther)
}

// ---------------------------------------------------------------------------
// ToCoverQueue — every approved, non-deleted, org-scoped entry with
// uncovered hours > 0 (D-06, COV-01).
// ---------------------------------------------------------------------------

func TestCoverageRepository_ToCoverQueue(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	// Fully covered → excluded (Σ == hours exactly).
	covered := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now.Add(-48*time.Hour), "approved", now)
	_, err := repo.ReplaceAllocations(ctx, orgID, covered,
		[]*coverage.CoverageAllocation{contractAllocation(8, contractID)},
		coverageSetAudit(orgID, covered, userID, nil, now))
	require.NoError(t, err)

	// Partially covered → included with the raw split. A partial state cannot
	// be CREATED by the replace-set (Σ == hours enforced in-tx, CR-01) — it
	// arises when an approved entry's hours are later edited upward. Simulate
	// that: allocate the full 3h against a 3h entry, then bump the entry to
	// 5h (the entry-hours edit path is outside the coverage plane).
	partial := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 3, now.Add(-24*time.Hour), "approved", now)
	_, err = repo.ReplaceAllocations(ctx, orgID, partial,
		[]*coverage.CoverageAllocation{contractAllocation(3, contractID)},
		coverageSetAudit(orgID, partial, userID, nil, now))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE time_entries SET hours = 5.0 WHERE id = $1`, partial)
	require.NoError(t, err)

	// No allocations at all (no-source entry) → included, fully uncovered (D-06).
	noSource := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 4, now, "approved", now)

	// Non-approved / deleted entries → excluded.
	draft := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 6, now.Add(24*time.Hour), "draft", now)
	pending := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 3, now.Add(48*time.Hour), "pending_manager", now)
	deleted := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 2, now.Add(72*time.Hour), "approved", now)
	_, err = pool.Exec(ctx, `UPDATE time_entries SET is_deleted = true WHERE id = $1`, deleted)
	require.NoError(t, err)

	queue, err := repo.ToCoverQueue(ctx, orgID)
	require.NoError(t, err)

	// entry_date ordering: partial (-24h) < noSource (now) < draft (+24h — but
	// excluded). Only partial + noSource are eligible.
	require.Len(t, queue, 2, "only the partially covered and the no-source entries are uncovered")
	require.NotEqual(t, draft, queue[0].EntryID, "draft entries must be excluded")
	require.NotEqual(t, pending, queue[0].EntryID, "pending entries must be excluded")
	require.NotEqual(t, deleted, queue[0].EntryID, "deleted entries must be excluded")
	require.Equal(t, partial, queue[0].EntryID)
	require.Equal(t, noSource, queue[1].EntryID)

	// Raw split on the partial row: 5h entry, 3h covered, 2h uncovered.
	row := queue[0]
	require.Equal(t, userID, row.EmployeeID)
	require.Equal(t, "Test User", row.EmployeeName)
	require.Equal(t, activityID, row.ActivityID)
	require.Equal(t, "Test Activity", row.ActivityName)
	require.Equal(t, 5.0, row.Hours)
	require.Equal(t, 3.0, row.CoveredHours)
	require.Equal(t, 2.0, row.UncoveredHours)

	// No-source row: zero covered, fully uncovered, still present (D-06).
	require.Equal(t, 4.0, queue[1].Hours)
	require.Equal(t, 0.0, queue[1].CoveredHours)
	require.Equal(t, 4.0, queue[1].UncoveredHours)

	// Deterministic tiebreak: same entry_date orders by id (noSource shares
	// the now() date, so compare the pair's relative order only).
	same1 := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 1, now, "approved", now)
	same2 := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 1, now, "approved", now)
	queue, err = repo.ToCoverQueue(ctx, orgID)
	require.NoError(t, err)
	pos1, pos2 := -1, -1
	for i, row := range queue {
		if row.EntryID == same1 {
			pos1 = i
		}
		if row.EntryID == same2 {
			pos2 = i
		}
	}
	require.NotEqual(t, -1, pos1, "same1 must appear in the queue")
	require.NotEqual(t, -1, pos2, "same2 must appear in the queue")
	if same1.String() < same2.String() {
		require.Less(t, pos1, pos2, "same-date rows order by id")
	} else {
		require.Greater(t, pos1, pos2, "same-date rows order by id")
	}
}

// ---------------------------------------------------------------------------
// BucketBalance — sold_hours − Σ drawn, adoption-aware visibility (D-02,
// D-03, COV-02, Pitfall 9).
// ---------------------------------------------------------------------------

func TestCoverageRepository_BucketBalance(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)

	// Support bucket: sold 10h.
	bucket := seedCoverageContract(t, pool, orgID, "Support Bucket", "support", ptr(10.0), false, now)
	// Unrelated project contract — allocations on it must not count.
	other := seedCoverageContract(t, pool, orgID, "Other Contract", "project", ptr(50.0), false, now)

	// Draw 4h from the bucket.
	entry := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 4, now, "approved", now)
	_, err := repo.ReplaceAllocations(ctx, orgID, entry,
		[]*coverage.CoverageAllocation{contractAllocation(4, bucket)},
		coverageSetAudit(orgID, entry, userID, nil, now))
	require.NoError(t, err)

	// Draw 3h from the other contract.
	entry2 := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 3, now.Add(time.Hour), "approved", now)
	_, err = repo.ReplaceAllocations(ctx, orgID, entry2,
		[]*coverage.CoverageAllocation{contractAllocation(3, other)},
		coverageSetAudit(orgID, entry2, userID, nil, now))
	require.NoError(t, err)

	// Balance = 10 − 4 = 6; the other contract's draw does not count (the
	// join scopes by contract_id, any source_type — Pitfall 9).
	balance, err := repo.BucketBalance(ctx, orgID, bucket)
	require.NoError(t, err)
	require.Equal(t, 6.0, balance)

	balance, err = repo.BucketBalance(ctx, orgID, other)
	require.NoError(t, err)
	require.Equal(t, 47.0, balance)

	// Overdraw allowed — the invariant is never a bucket gate (D-03): an
	// entry drawing 12h from the 10h bucket pushes the balance negative.
	entry3 := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 12, now.Add(2*time.Hour), "approved", now)
	_, err = repo.ReplaceAllocations(ctx, orgID, entry3,
		[]*coverage.CoverageAllocation{contractAllocation(12, bucket)},
		coverageSetAudit(orgID, entry3, userID, nil, now))
	require.NoError(t, err)

	balance, err = repo.BucketBalance(ctx, orgID, bucket)
	require.NoError(t, err)
	require.Equal(t, -6.0, balance, "negative balances are returned as-is, never gated")
}

func TestCoverageRepository_BucketBalance_AdoptionAwareVisibility(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgA := seedOrg(t, pool, now)
	orgB := seedOrg(t, pool, now)
	orgC := seedOrg(t, pool, now)

	// A shared support bucket created by org A, adopted only by org B.
	shared := seedCoverageContract(t, pool, orgA, "Shared Bucket", "support", ptr(10.0), true, now)
	seedContractAdoption(t, pool, shared, orgB, now)

	// Org B draws 4h from the adopted shared bucket.
	userB := seedUser(t, pool, now)
	activityB := seedActivity(t, pool, orgB, "engagement", nil, now)
	unitB := seedUnit(t, pool, orgB, now)
	entryB := seedTimeEntry(t, pool, orgB, userB, activityB, unitB, 4, now, "approved", now)
	_, err := repo.ReplaceAllocations(ctx, orgB, entryB,
		[]*coverage.CoverageAllocation{contractAllocation(4, shared)},
		coverageSetAudit(orgB, entryB, userB, nil, now))
	require.NoError(t, err)

	// Adopted org resolves the balance: 10 − 4 = 6.
	balance, err := repo.BucketBalance(ctx, orgB, shared)
	require.NoError(t, err)
	require.Equal(t, 6.0, balance)

	// The creator org also sees it (created_by_org_id = orgA).
	balance, err = repo.BucketBalance(ctx, orgA, shared)
	require.NoError(t, err)
	require.Equal(t, 6.0, balance)

	// A non-adopted org reads the shared contract as not found — the same
	// adoption-aware predicate the 12-05 contract-ref validation applies.
	_, err = repo.BucketBalance(ctx, orgC, shared)
	require.ErrorIs(t, err, coverage.ErrNotFound)
}

// ptr returns a pointer to v (test helper for nullable seed columns).
func ptr[T any](v T) *T { return &v }

// coverageCloseAudit builds the 'coverage-closed' audit row the service passes
// to ClosePeriod — addressed to the CLOSE (entity_id = closeID), the
// close-level event the org's audit stream records (A7: per-entry history
// reads entity_id = entry id, so the close event is read by close, not entry).
func coverageCloseAudit(orgID, closeID, actorID uuid.UUID, payload map[string]any, now time.Time) *audit.AuditLog {
	return &audit.AuditLog{
		OrgID:      orgID,
		EntityType: coverage.AuditEntityCoverageAllocation,
		EntityID:   closeID,
		Action:     coverage.AuditActionCoverageClosed,
		ActorID:    &actorID,
		Payload:    payload,
		CreatedAt:  now,
	}
}

// day returns a UTC time at noon on the given date — date-part comparisons
// (entry_date::date BETWEEN ...) are insensitive to the time component.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// ClosePeriod — the frozen snapshot tx (D-10/D-11/D-12, T-12-15).
// ---------------------------------------------------------------------------

func TestCoverageRepository_ClosePeriod_FreezesSnapshot(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, day(2026, 7, 10), "approved", now)

	// Original set: 5h + 3h (the state "as reported").
	_, err := repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(5, contractID), contractAllocation(3, contractID)},
		coverageSetAudit(orgID, entryID, userID, nil, now))
	require.NoError(t, err)

	closeID := uuid.New()
	closePayload := map[string]any{"period_start": "2026-07-01", "period_end": "2026-07-31", "rows": 2}
	pc, err := repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 31), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, closePayload, now))
	require.NoError(t, err)
	require.Equal(t, closeID, pc.ID)
	require.Equal(t, orgID, pc.OrgID)
	require.Len(t, pc.Rows, 2)
	require.Equal(t, 8.0, pc.Rows[0].Hours+pc.Rows[1].Hours)

	// Later replace changes the LIVE set to 2h + 6h — the snapshot must be
	// untouched (COV-04, Pitfall 7: reports read the copy).
	_, err = repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(2, contractID), contractAllocation(6, contractID)},
		coverageSetAudit(orgID, entryID, userID, nil, now))
	require.NoError(t, err)

	snap, err := repo.GetSnapshot(ctx, orgID, closeID)
	require.NoError(t, err)
	require.Len(t, snap.Rows, 2, "the frozen rows survive the later replace")
	hours := map[float64]bool{}
	for _, row := range snap.Rows {
		require.Equal(t, entryID, row.EntryID)
		require.Equal(t, userID, row.EmployeeID)
		require.Equal(t, activityID, row.ActivityID)
		require.Equal(t, contractID, *row.ContractID)
		require.Equal(t, coverage.SourceTypeContract, row.SourceType)
		hours[row.Hours] = true
	}
	require.True(t, hours[5], "the ORIGINAL 5h allocation must be frozen")
	require.True(t, hours[3], "the ORIGINAL 3h allocation must be frozen")

	// The live set is the new one.
	live, err := repo.ListByEntry(ctx, orgID, entryID)
	require.NoError(t, err)
	require.Equal(t, 8.0, sumHours(t, live))
	liveHours := map[float64]bool{}
	for _, a := range live {
		liveHours[a.Hours] = true
	}
	require.True(t, liveHours[2])
	require.True(t, liveHours[6])
}

func TestCoverageRepository_ClosePeriod_Scope(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	inside := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 4, day(2026, 7, 15), "approved", now)
	boundary := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 2, day(2026, 7, 31), "approved", now)
	outside := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 6, day(2026, 6, 15), "approved", now)

	for _, e := range []uuid.UUID{inside, boundary, outside} {
		hours := 4.0
		switch e {
		case boundary:
			hours = 2.0
		case outside:
			hours = 6.0
		}
		_, err := repo.ReplaceAllocations(ctx, orgID, e,
			[]*coverage.CoverageAllocation{contractAllocation(hours/2, contractID), contractAllocation(hours/2, contractID)},
			coverageSetAudit(orgID, e, userID, nil, now))
		require.NoError(t, err)
	}

	closeID := uuid.New()
	pc, err := repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 31), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, nil, now))
	require.NoError(t, err)

	// Inclusive bounds on both ends, date-cast: inside AND boundary frozen;
	// outside not.
	require.Len(t, pc.Rows, 4, "inside + boundary entries each contribute 2 rows")
	frozenEntries := map[uuid.UUID]int{}
	for _, row := range pc.Rows {
		frozenEntries[row.EntryID]++
	}
	require.Equal(t, 2, frozenEntries[inside])
	require.Equal(t, 2, frozenEntries[boundary])
	_, hasOutside := frozenEntries[outside]
	require.False(t, hasOutside, "entries outside the period are not frozen")
}

func TestCoverageRepository_ClosePeriod_DuplicateRejected(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	// Existing close [2026-07-01, 2026-07-05].
	closeID := uuid.New()
	_, err := repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 5), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, nil, now))
	require.NoError(t, err)

	// Identical bounds → rejected.
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 5), uuid.New(), userID,
		coverageCloseAudit(orgID, uuid.New(), userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrPeriodAlreadyClosed)

	// Partial overlap [07-04, 07-10] vs existing [07-01, 07-05] → shares the
	// [07-04, 07-05] overlap → rejected (A6: inclusive-overlap, not just
	// identical bounds).
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 7, 4), day(2026, 7, 10), uuid.New(), userID,
		coverageCloseAudit(orgID, uuid.New(), userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrPeriodAlreadyClosed)

	// A close fully contained in the existing period → rejected.
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 7, 2), day(2026, 7, 3), uuid.New(), userID,
		coverageCloseAudit(orgID, uuid.New(), userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrPeriodAlreadyClosed)

	// A close WIDER than the existing period → rejected (the predicate catches
	// [06-01, 08-31] which contains [07-01, 07-05]).
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 6, 1), day(2026, 8, 31), uuid.New(), userID,
		coverageCloseAudit(orgID, uuid.New(), userID, nil, now))
	require.ErrorIs(t, err, coverage.ErrPeriodAlreadyClosed)

	// Later non-overlapping period → succeeds.
	close2 := uuid.New()
	pc, err := repo.ClosePeriod(ctx, orgID, day(2026, 7, 11), day(2026, 7, 15), close2, userID,
		coverageCloseAudit(orgID, close2, userID, nil, now))
	require.NoError(t, err)
	require.Equal(t, close2, pc.ID)
	require.Empty(t, pc.Rows)
}

func TestCoverageRepository_ClosePeriod_Audit(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 4, day(2026, 7, 10), "approved", now)
	_, err := repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(4, contractID)},
		coverageSetAudit(orgID, entryID, userID, nil, now))
	require.NoError(t, err)

	closeID := uuid.New()
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 31), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, map[string]any{"period_start": "2026-07-01", "period_end": "2026-07-31", "rows": 1}, now))
	require.NoError(t, err)

	// Exactly one coverage-closed audit row, payload present (in-tx write,
	// T-12-16).
	var action, payload string
	err = pool.QueryRow(ctx,
		`SELECT action, COALESCE(payload::text, '') FROM audit_logs
		 WHERE entity_type = 'coverage_allocation' AND entity_id = $1`,
		closeID).Scan(&action, &payload)
	require.NoError(t, err)
	require.Equal(t, coverage.AuditActionCoverageClosed, action)
	require.Contains(t, payload, "period_start")
	require.Contains(t, payload, "rows")

	var closeAudits int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'coverage_allocation' AND entity_id = $1 AND action = 'coverage-closed'`,
		closeID).Scan(&closeAudits)
	require.NoError(t, err)
	require.Equal(t, 1, closeAudits)
}

// ---------------------------------------------------------------------------
// GetSnapshot
// ---------------------------------------------------------------------------

func TestCoverageRepository_GetSnapshot_NotFoundAndScope(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	closeID := uuid.New()
	_, err := repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 31), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, nil, now))
	require.NoError(t, err)

	// Missing close → ErrNotFound.
	_, err = repo.GetSnapshot(ctx, orgID, uuid.New())
	require.ErrorIs(t, err, coverage.ErrNotFound)

	// Cross-org close → ErrNotFound (the org_id filter fails the same way).
	otherOrg := seedOrg(t, pool, now)
	_, err = repo.GetSnapshot(ctx, otherOrg, closeID)
	require.ErrorIs(t, err, coverage.ErrNotFound)
}

// ---------------------------------------------------------------------------
// ListHistory — the entry-scoped audit stream (A7).
// ---------------------------------------------------------------------------

func TestCoverageRepository_ListHistory(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)

	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, day(2026, 7, 10), "approved", now)
	_, err := repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(8, contractID)},
		coverageSetAudit(orgID, entryID, userID, map[string]any{"set": "first"}, now))
	require.NoError(t, err)
	_, err = repo.ReplaceAllocations(ctx, orgID, entryID,
		[]*coverage.CoverageAllocation{contractAllocation(5, contractID), contractAllocation(3, contractID)},
		coverageSetAudit(orgID, entryID, userID, map[string]any{"set": "second"}, now.Add(time.Second)))
	require.NoError(t, err)

	// A close of a period containing the entry (its event is addressed to the
	// CLOSE, not the entry — A7 entry history covers allocation changes only).
	closeID := uuid.New()
	_, err = repo.ClosePeriod(ctx, orgID, day(2026, 7, 1), day(2026, 7, 31), closeID, userID,
		coverageCloseAudit(orgID, closeID, userID, nil, now))
	require.NoError(t, err)

	history, err := repo.ListHistory(ctx, orgID, entryID)
	require.NoError(t, err)
	require.Len(t, history, 2, "two allocations-set events for the entry, in created order")
	require.Equal(t, coverage.AuditActionAllocationsSet, history[0].Action)
	require.Equal(t, coverage.AuditActionAllocationsSet, history[1].Action)
	require.Equal(t, entryID, history[0].EntityID)
	require.Equal(t, coverage.AuditEntityCoverageAllocation, history[0].EntityType)
	require.Equal(t, "first", history[0].Payload["set"])
	require.Equal(t, "second", history[1].Payload["set"])

	// The close event is NOT in the entry history (entity_id = closeID) — it
	// lives in the org stream addressed to the close; read via the close id.
	closeHistory, err := repo.ListHistory(ctx, orgID, closeID)
	require.NoError(t, err)
	require.Len(t, closeHistory, 1)
	require.Equal(t, coverage.AuditActionCoverageClosed, closeHistory[0].Action)

	// Cross-org reads return nothing.
	otherOrg := seedOrg(t, pool, now)
	other, err := repo.ListHistory(ctx, otherOrg, entryID)
	require.NoError(t, err)
	require.Empty(t, other)
}

// ---------------------------------------------------------------------------
// CR-01 gap-closure battery — concurrent replace-sets on one entry (T-12-14).
// Mirrors the ticket battery shape (start channel + buffered results channel):
// no wall-clock timing decides anything, only the FOR UPDATE entry-row lock.
// ---------------------------------------------------------------------------

func TestCoverageReplace_Concurrent(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewCoverageRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	unitID := seedUnit(t, pool, orgID, now)
	contractID := seedCoverageContract(t, pool, orgID, "Budget", "project", ptr(100.0), false, now)
	entryID := seedTimeEntry(t, pool, orgID, userID, activityID, unitID, 8, now, "approved", now)

	t.Run("two valid sets — invariant holds after both commit", func(t *testing.T) {
		setA := []*coverage.CoverageAllocation{
			contractAllocation(5, contractID),
			contractAllocation(3, contractID),
		}
		setB := []*coverage.CoverageAllocation{contractAllocation(8, contractID)}

		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			_, err := repo.ReplaceAllocations(ctx, orgID, entryID, setA, coverageSetAudit(orgID, entryID, userID, nil, now))
			results <- err
		}()
		go func() {
			<-start
			_, err := repo.ReplaceAllocations(ctx, orgID, entryID, setB, coverageSetAudit(orgID, entryID, userID, nil, now))
			results <- err
		}()
		close(start)

		for i := 0; i < 2; i++ {
			require.NoError(t, <-results)
		}

		// The committed state is exactly one of the two sets — Σ == hours
		// either way; a violating state must never be observable.
		stored, err := repo.ListByEntry(ctx, orgID, entryID)
		require.NoError(t, err)
		require.Equal(t, 8.0, sumHours(t, stored), "no violating state may ever be observable")
	})

	t.Run("mismatched set racing a valid set — the invalid never commits", func(t *testing.T) {
		bad := []*coverage.CoverageAllocation{contractAllocation(7, contractID)} // Σ=7 ≠ 8
		good := []*coverage.CoverageAllocation{contractAllocation(8, contractID)}

		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			_, err := repo.ReplaceAllocations(ctx, orgID, entryID, bad, coverageSetAudit(orgID, entryID, userID, nil, now))
			results <- err
		}()
		go func() {
			<-start
			_, err := repo.ReplaceAllocations(ctx, orgID, entryID, good, coverageSetAudit(orgID, entryID, userID, nil, now))
			results <- err
		}()
		close(start)

		successes, mismatches := 0, 0
		for i := 0; i < 2; i++ {
			err := <-results
			switch {
			case err == nil:
				successes++
			case errors.Is(err, coverage.ErrAllocationSumMismatch):
				mismatches++
			default:
				t.Fatalf("unexpected race outcome: %v", err)
			}
		}
		require.Equal(t, 1, successes, "exactly one replace must win the race")
		require.Equal(t, 1, mismatches, "the invalid set must fail the in-tx Σ re-check")

		stored, err := repo.ListByEntry(ctx, orgID, entryID)
		require.NoError(t, err)
		require.Equal(t, 8.0, sumHours(t, stored), "no violating state may ever be observable")
	})
}
