---
phase: 12-coverage-backend-the-allocation-loop
plan: 06
subsystem: database
tags: [go, postgres, coverage, allocation-ledger, for-update, cr-01, snapshot, audit, pgx]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop
    provides: schema 019/020 tables + CHECK vocabularies (12-01), domain entities + 6 sentinels (12-04), port contract (12-04), pool-level service fast-fails (12-05)
provides:
  - "CoverageRepository (7-method port impl): atomic replace-set tx with FOR UPDATE in-tx Σ re-check, to-cover queue read-model, derived bucket balance, frozen period-close snapshot tx, snapshot/history reads"
  - "CR-01 closure for the coverage plane: concurrent replace-sets serialize on the entry-row lock; violating states never commit (proven by TestCoverageReplace_Concurrent)"
  - "In-tx audit helper insertCoverageAudit (BE-016) — every coverage mutator writes its event in the same tx"
affects: [12-07, phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (32000) on the same scale.
actuals:
  tokens: 17268    # chars/4 over the realized diff (69070 bytes / 4)
  tasks: 2         # tasks completed
  commits: 3       # commits made (2 task commits + 1 docs metadata)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "In-tx authoritative validation under FOR UPDATE (CR-01): pool-level checks are fast-fail UX only; the tx re-checks status/is_deleted and Σ in cents before DELETE+INSERT"
    - "In-tx audit writes via a private tx-aware helper (insertCoverageAudit mirrors insertTicketAudit exactly — no public CreateTx on the port, BE-016)"
    - "Inclusive-overlap rejection predicate for period closes (period_start <= new.end AND period_end >= new.start — catches contained/partial/wider closes, not just identical bounds, A6)"
    - "pgx constraint: a rows cursor cannot be interleaved with Exec on the same tx connection — collect rows into memory first, then write"

key-files:
  created:
    - internal/adapters/secondary/postgres/coverage_repository.go
    - internal/adapters/secondary/postgres/coverage_repository_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go

key-decisions:
  - "Close audit row addresses the CLOSE (entity_id = closeID), not the entry: A7 per-entry history covers allocation changes only; the close event is read via ListHistory(closeID) or direct query"
  - "Partial coverage states cannot be created via the replace-set (Σ == hours enforced in-tx) — they arise from later entry-hours edits; the queue test simulates that path"
  - "BucketBalance COALESCE(c.sold_hours, 0): a NULL sold_hours (legacy/project contract) reads as a zero commitment instead of a NULL-scan error — support buckets are unaffected (016 CHECK pins sold_hours NOT NULL for 'support')"

patterns-established:
  - "Repo methods for the coverage plane follow the ticket repo shape: BeginTx + defer Rollback + FOR UPDATE lock + in-tx re-validation + audit helper + Commit"
  - "Queue/balance reads org-scope every WHERE clause (T-12-17); snapshot rows are append-only by construction (no UPDATE/DELETE surface, D-10)"
  - "Concurrency tests use the start-channel + buffered-results shape — deterministic outcome sets, no wall-clock timing"

requirements-completed: [COV-01, COV-02, COV-04]

coverage:
  - id: D1
    description: "ReplaceAllocations — atomic replace-set tx: FOR UPDATE entry lock, in-tx status/is_deleted re-check, Σ re-validation in cents, DELETE+INSERT, in-tx audit (COV-01, D-07, CR-01)"
    requirement: COV-01
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ReplaceAllocations_CommitsAndReadsBack"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ReplaceAllocations_SumMismatchLeavesNoRows"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ReplaceAllocations_RejectsNotCoverable"
        status: pass
    human_judgment: false
  - id: D2
    description: "Concurrent replace-sets serialize on the entry-row lock; a violating Σ state never commits — CR-01 battery (two valid sets + mismatched-vs-valid race)"
    requirement: COV-01
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageReplace_Concurrent"
        status: pass
    human_judgment: false
  - id: D3
    description: "ToCoverQueue — every approved, non-deleted, org-scoped time entry with uncovered hours > 0, incl. no-source entries, ordered by (entry_date, id) (D-06, Pitfall 6)"
    requirement: COV-01
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ToCoverQueue"
        status: pass
    human_judgment: false
  - id: D4
    description: "BucketBalance — sold_hours − Σ drawn scoped by contract_id (any source_type), negative allowed, adoption-aware visibility pre-check (D-02/D-03, Pitfall 9)"
    requirement: COV-02
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_BucketBalance"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_BucketBalance_AdoptionAwareVisibility"
        status: pass
    human_judgment: false
  - id: D5
    description: "ClosePeriod — frozen snapshot in one tx: in-tx inclusive-overlap rejection, FOR UPDATE on the period's entries, header + per-allocation rows, in-tx audit (COV-04, D-10/D-11/D-12, A6)"
    requirement: COV-04
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ClosePeriod_FreezesSnapshot"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ClosePeriod_Scope"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ClosePeriod_DuplicateRejected"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ClosePeriod_Audit"
        status: pass
    human_judgment: false
  - id: D6
    description: "GetSnapshot + ListHistory — read-only org-scoped snapshot read and the entry-scoped audit stream (A7, T-12-16)"
    requirement: COV-04
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_GetSnapshot_NotFoundAndScope"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/coverage_repository_test.go#TestCoverageRepository_ListHistory"
        status: pass
    human_judgment: false

# Metrics
duration: 8m
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 6: Coverage Repository — The Allocation Loop Summary

**PostgreSQL coverage repository: atomic replace-set tx with FOR UPDATE in-tx Σ re-check, to-cover queue read-model, derived bucket balance, frozen period-close snapshot tx, and the CR-01 concurrency battery proving no violating state commits**

## Performance

- **Duration:** 8 min
- **Started:** 2026-08-08T09:29:09Z
- **Completed:** 2026-08-08T09:37:27Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `CoverageRepository` (all 7 port methods): `ReplaceAllocations` is ONE transaction — FOR UPDATE entry-row lock → in-tx status/is_deleted re-check → Σ re-validation in cents → DELETE-all + INSERT-set → `insertCoverageAudit` → Commit. The pool-level service fast-fails (12-05) are UX only; this tx is the CR-01 correctness closure — partial states are impossible by construction.
- `ToCoverQueue` single query: every approved, non-deleted, org-scoped `time` entry with uncovered hours > 0 (LEFT JOIN + `HAVING Σ < hours`), including no-source entries (D-06 — never an implicit gap), with the `CONCAT(COALESCE(u.firstname,''), ' ', COALESCE(u.lastname,''))` employee projection (users has no `name` column), ordered by `(entry_date, id)` for deterministic pagination.
- `BucketBalance` derived on read: `sold_hours − Σ allocations` scoped by `contract_id` (any `source_type` — transfers draw the target bucket, Pitfall 9), negative returned as-is (D-03), gated by the adoption-aware visibility predicate shared with the 12-05 contract-ref rule (created_by_org_id OR shared+adopted).
- `ClosePeriod` is ONE transaction: in-tx inclusive-overlap rejection (`period_start <= new.end AND period_end >= new.start` — catches identical, contained, partial, and wider closes → `ErrPeriodAlreadyClosed`, 409), FOR UPDATE lock on the period's entries (serializes with in-flight replaces), header + one `coverage_snapshot_rows` row per current allocation with frozen resolved refs, in-tx audit → Commit. Later allocation edits never alter the snapshot (COV-04, Pitfall 7).
- `GetSnapshot` (org-scoped header + rows) and `ListHistory` (entry-scoped audit stream, `entity_type='coverage_allocation'`, payload JSONB round-trip) — both read-only.
- CR-01 battery (`TestCoverageReplace_Concurrent`): two valid sets race on one entry → committed state always Σ == hours; a mismatched set (Σ=7) racing a valid one → exactly one success + one `ErrAllocationSumMismatch`, final state Σ == 8. No violating state ever observable.

## Task Commits

Each task was committed atomically:

1. **Task 1: ReplaceAllocations tx + ListByEntry + ToCoverQueue + BucketBalance** - `15603d0` (feat)
2. **Task 2: ClosePeriod tx + GetSnapshot + ListHistory + concurrency battery** - `a5f7651` (feat)

**Plan metadata:** `docs(12-06)` (final commit, after SUMMARY/STATE/ROADMAP)

## Files Created/Modified

- `internal/adapters/secondary/postgres/coverage_repository.go` - CoverageRepository (7 methods), insertCoverageAudit in-tx helper, scanCoverageAllocationRow, roundCents
- `internal/adapters/secondary/postgres/coverage_repository_test.go` - 14 integration tests incl. TestCoverageReplace_Concurrent battery
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - seedCoverageContract (016 columns + sold_period), seedContractAdoption, seedTimeEntry, nullableStr

## Decisions Made

- **Close audit row addresses the close, not the entry** (entity_id = closeID): A7 pins per-entry history to `entity_id = entry`; the close event is read by close id. The plan's test wording ("allocations-set + coverage-closed rows for the entry") conflicts with the port contract — the port wins, and the test asserts both behaviors explicitly.
- **Partial coverage simulated via entry-hours edit**: the replace-set can never create a partial state (Σ enforced in-tx); the queue test bumps `hours` after allocating to model the realistic partial path.
- **`COALESCE(c.sold_hours, 0)` in BucketBalance**: prevents a NULL-scan error for legacy/project contracts without sold hours; support buckets unaffected (016 CHECK).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] pgx "conn busy" on interleaved query+exec in ClosePeriod**
- **Found during:** Task 2 (ClosePeriod freeze loop)
- **Issue:** iterating `tx.Query` allocation rows while calling `tx.Exec` for the snapshot-row insert on the same transaction errors with `insert snapshot row: conn busy` — pgx forbids interleaving a live rows cursor with Exec on one connection.
- **Fix:** collect each entry's allocations into a slice, close the rows, then INSERT the snapshot rows.
- **Files modified:** internal/adapters/secondary/postgres/coverage_repository.go
- **Verification:** all close tests pass (FreezesSnapshot, Scope, DuplicateRejected, Audit)
- **Committed in:** a5f7651 (Task 2 commit)

**2. [Rule 1 - Bug] Contract seed violated contracts_sold_check**
- **Found during:** Task 1 (BucketBalance tests)
- **Issue:** seeding `contract_type='support'` without `sold_period` fails the 016 three-valued-logic CHECK (`support` requires sold_hours AND sold_period).
- **Fix:** seedCoverageContract sets `sold_period='monthly'` for support contracts.
- **Files modified:** internal/adapters/secondary/postgres/exported_test_helpers.go
- **Verification:** BucketBalance + adoption tests pass
- **Committed in:** 15603d0 (Task 1 commit)

**3. [Rule 1 - Bug] ToCoverQueue test attempted partial state via the replace-set**
- **Found during:** Task 1 (queue matrix test)
- **Issue:** ReplaceAllocations enforces Σ == hours in-tx, so a 3h allocation against a 5h entry was rejected — the test conflated "partial state" with the invariant the repo exists to enforce.
- **Fix:** allocate the full 3h against a 3h entry, then bump the entry's hours to 5 (the entry-hours edit path is outside the coverage plane) — the realistic origin of partial coverage.
- **Files modified:** internal/adapters/secondary/postgres/coverage_repository_test.go
- **Verification:** queue matrix test passes
- **Committed in:** 15603d0 (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 — all test/implementation correctness fixes within task scope)
**Impact on plan:** All fixes necessary for correctness. No scope creep; no architectural changes.

## Issues Encountered

- Pre-existing `seedContract` (time_entry_repository_test.go) lacks the 016 columns and collides with the plan's helper name — renamed the new helper to `seedCoverageContract` (documented in its doc comment). No change to the pre-existing helper.
- Pre-existing gofmt drift in `customer_repository_test.go` / `refresh_token_rotate_test.go` / `working_group_repository_test.go` observed but left untouched (out of scope per scope boundary).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 12-07 (HTTP handlers + route wiring) can consume the full port: all 7 methods implemented, compile-time assertion green, integration-tested including the concurrency battery.
- Phase 17 report surfaces read the snapshot rows (`GetSnapshot`) and derived balances (`BucketBalance`) exactly as the port documents.
- The `make test` full suite is green (including the postgres package).

---
*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*

## Self-Check: PASSED

- [x] SUMMARY.md exists on disk
- [x] Task 1 commit `15603d0` present in git log
- [x] Task 2 commit `a5f7651` present in git log
- [x] Metadata commit `3d59a95` (SUMMARY/STATE/ROADMAP)
- [x] `go build ./...` green; `make test` full suite green
- [x] Both plan-level `<verification>` commands exit 0
