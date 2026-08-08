package postgres

import (
	"context"
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
