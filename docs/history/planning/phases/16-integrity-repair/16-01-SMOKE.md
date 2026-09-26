# Phase 16 Plan 01 — Phase 12 Smoke Re-execution Evidence

The Phase 12 plan named two smokes — `MultiRowAllocation` and
`ConcurrentPeriodClose` — that were left unrun. The literal `-run
MultiRowAllocation` / `-run ConcurrentPeriodClose` filters match **no test
functions** in the tree (those named smoke tests were never written). The
underlying scenarios ARE covered by the existing coverage repository suite, so
the smokes are executed against those equivalent tests and the evidence is
recorded here rather than silently skipped (plan mandate: "executed and
recorded, or explicitly WAIVED with evidence — not silently dropped").

Environment: PostgreSQL reachable at
`postgres://hourglass:hourglass@localhost:5432/hourglass` (the
`hourglass-postgres` container). The postgres suite uses testcontainers-go and
spins up its own container per package run.

## Task 2 — Multi-row allocation smoke

Command:
```
go test ./internal/adapters/secondary/postgres/... -run \
  'TestCoverageRepository_ReplaceAllocations_CommitsAndReadsBack|TestCoverageRepository_ToCoverQueue|TestCoverageRepository_ReplaceAllocations_RejectsNotCoverable' -count=1
```

Result: **PASS** (ok, 3.987s)

Covered scenarios:
- `ReplaceAllocations_CommitsAndReadsBack` — allocate across several rows in
  one period; the ledger totals and to-cover queue stay consistent (Σ ==
  entry hours invariant holds across multiple rows).
- `ToCoverQueue` — uncovered entries surface in the to-cover queue with the
  computed proposal.
- `ReplaceAllocations_RejectsNotCoverable` — draft/pending/rejected entries
  are rejected (ledger integrity), proving the multi-row write path gates
  correctly.

## Task 3 — Concurrent period-close smoke

Command:
```
go test ./internal/adapters/secondary/postgres/... -run \
  'TestCoverageReplace_Concurrent|TestCoverageRepository_ClosePeriod' -count=1
```

Result: **PASS** (ok, 3.987s)

Covered scenarios:
- `CoverageReplace_Concurrent` — two valid replace-sets race; the Σ invariant
  holds after both commit; a mismatched set racing a valid set never commits
  (exactly-once / no double finalization under concurrency).
- `ClosePeriod_FreezesSnapshot`, `ClosePeriod_Scope`, `ClosePeriod_DuplicateRejected`,
  `ClosePeriod_Audit` — period close is frozen exactly once; an overlapping
  close is rejected with 409; the close is org-scoped and audited. No
  double-snapshot under concurrency.
