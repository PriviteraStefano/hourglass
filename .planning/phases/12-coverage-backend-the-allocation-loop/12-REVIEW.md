---
phase: 12-coverage-backend-the-allocation-loop
reviewed: 2026-08-08T00:00:00Z
depth: standard
files_reviewed: 34
files_reviewed_list:
  - .gitignore
  - cmd/server/main.go
  - hourglass-vault/decisions/backend/_index.md
  - hourglass-vault/decisions/project/_index.md
  - internal/adapters/primary/http/activity_handler.go
  - internal/adapters/primary/http/coverage_handler_test.go
  - internal/adapters/primary/http/coverage_handler.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go
  - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
  - internal/adapters/secondary/postgres/activity_repository.go
  - internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go
  - internal/adapters/secondary/postgres/coverage_repository_test.go
  - internal/adapters/secondary/postgres/coverage_repository.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/ontology_extension_migrations_test.go
  - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  - internal/core/domain/activity/activity.go
  - internal/core/domain/coverage/coverage.go
  - internal/core/domain/coverage/errors.go
  - internal/core/ports/activity_repository.go
  - internal/core/ports/coverage_repository.go
  - internal/core/services/activity/activity_beneficiary_unit_test.go
  - internal/core/services/activity/activity.go
  - internal/core/services/coverage/coverage_test.go
  - internal/core/services/coverage/coverage.go
  - internal/core/services/testdata/mock_coverage_repo.go
  - internal/core/services/testdata/mocks.go
  - migrations/018_activity_beneficiary_unit.down.sql
  - migrations/018_activity_beneficiary_unit.up.sql
  - migrations/019_coverage_allocations.down.sql
  - migrations/019_coverage_allocations.up.sql
  - migrations/020_coverage_snapshots.down.sql
  - migrations/020_coverage_snapshots.up.sql
findings:
  critical: 1
  warning: 6
  info: 3
  total: 10
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-08-08
**Depth:** standard
**Files Reviewed:** 34
**Status:** issues_found

## Summary

The coverage allocation loop (service + repository + handlers + migrations 018-020 + beneficiary-unit extension) is thoughtfully designed: the replace-set write with FOR UPDATE in-tx re-validation, the same-tx audit rows, and the frozen snapshot close are all sound in isolation, and the test batteries (concurrent replace race, up/down/up migration cycles, sentinel maps) are strong.

However, one **blocker** defeats the core write path: the HTTP handler never sets an allocation `ID`, so every row is inserted with the zero UUID — the table's primary key collides on the second row *anywhere in the database* (multi-row sets always fail; a second entry's first allocation always fails). The handler-level test passes only because it performs exactly one successful allocation insert.

Beyond that: `ClosePeriod` accepts inverted periods (which permanently poison the range against future closes), its overlap check races under concurrency, fractional-cent hours pass service validation and surface as unmapped 500s, the activity Update path bypasses kind/governance validation (FK/CHECK → 500), repo sentinels leak to 500 on activity Create/Update, and the `::date` casts in the close/freeze queries are session-timezone-dependent.

## Critical Issues

### CR-01: Allocation rows are inserted with `uuid.Nil` IDs — the ledger can hold at most one row in the entire database

**File:** `internal/adapters/primary/http/coverage_handler.go:111-120` (with `internal/adapters/secondary/postgres/coverage_repository.go:150-155`)
**Issue:** `PutAllocations` builds `coveragedomain.CoverageAllocation` without setting `ID`, so every row carries `uuid.Nil`. The repo's INSERT (line 154) passes `a.ID` explicitly as `$1`, bypassing the column's `DEFAULT gen_random_uuid()`. The PK spans the whole table:

- A PUT with ≥ 2 rows (the D-07 design is explicitly "1..N rows") → second INSERT hits the same PK → 23505 → `ports.ErrConflict` → 500.
- Even single-row PUTs fail once *any* allocation row exists anywhere: the first successful insert occupies `00000000-0000-0000-0000-000000000000`, so the next entry's first allocation collides → 500.

Only replace-sets for the *same* entry work (DELETE clears the zero-UUID row first). The handler test (coverage_handler_test.go) performs exactly one successful PUT, so the defect is invisible; the repo tests only pass because `contractAllocation` in `coverage_repository_test.go:16-23` sets `ID: uuid.New()` explicitly. Every real multi-row or multi-entry usage of the feature is broken.

**Fix:**
```go
// coverage_repository.go — inside ReplaceAllocations, before INSERT:
for _, a := range allocs {
    if a.ID == uuid.Nil {
        a.ID = uuid.New() // the boundary DTO never carries ids; generate here
    }
    if _, err := tx.Exec(ctx, `INSERT INTO coverage_allocations ...`, a.ID, ...); err != nil { ... }
}
```
(Generating in the repo keeps the port contract that the caller supplies rows without ids; alternatively drop `id` from the INSERT entirely and let the DEFAULT apply.)

## Warnings

### WR-01: Fractional-cent hours pass service validation, then hit the unmapped DB CHECK → 500

**File:** `internal/core/services/coverage/coverage.go:374-376` (and repo re-check `coverage_repository.go:130-136`)
**Issue:** Step 5 only rejects `a.Hours <= 0`. A row with `0 < hours < 0.005` contributes 0 cents to the Σ fast-fail (line 342), so a compensated set (e.g. `7.999 + 0.001` vs an 8h entry) passes service validation, and the INSERT then violates the `hours > 0` CHECK (019 line 40). `wrapPGError` maps only 23505/23503 — 23514 is unmapped → raw error → 500 at the handler. Hours with >2 decimals are also silently rounded by `DECIMAL(8,2)` storage, so the stored Σ can diverge from the validated cents sum. The service comment itself notes 23514 is unmapped ("would otherwise surface as 500"), so this is a live landmine in the normal path.

**Fix:**
```go
if a.Hours <= 0 || math.Round(a.Hours*100) != a.Hours*100 {
    return nil, coverage.ErrInvalidRequest // reject sub-cent and >2-decimal values
}
```
And/or map `23514` to a domain sentinel in `wrapPGError` so CHECK violations can never surface as 500.

### WR-02: ClosePeriod accepts inverted periods (start > end), permanently poisoning the range

**File:** `internal/core/services/coverage/coverage.go:474-492` (`internal/adapters/primary/http/coverage_handler.go:239-248`)
**Issue:** Neither the handler nor the service validates `periodStart <= periodEnd`. A close of e.g. `[2026-08-31, 2026-08-01]` succeeds (the BETWEEN finds no entries → 201 with an empty snapshot), and because the overlap predicate (`period_start <= $3::date AND period_end >= $2::date`, coverage_repository.go:323) is symmetric and inclusive, every subsequent legitimate close of August 2026 → 409 forever. The org's reporting period is bricked with no remediation path (snapshots are append-only by design).

**Fix:** In the service before the repo call:
```go
if periodStart.After(periodEnd) {
    return nil, coverage.ErrInvalidRequest
}
```
And/or add `CHECK (period_end >= period_start)` to migration 020.

### WR-03: ClosePeriod overlap check is not concurrency-safe — duplicate overlapping closes can both commit

**File:** `internal/adapters/secondary/postgres/coverage_repository.go:317-346`
**Issue:** The overlap `SELECT EXISTS` runs at the start of the tx (READ COMMITTED) before any lock is taken. Two concurrent closes of the same period both observe "no overlap" (no header row exists yet), then serialize only on the entry `FOR UPDATE` (step 2) — which happens *after* the check. When the loser's lock wait resolves, it proceeds with its stale check result and inserts its own header: both closes commit, producing two frozen snapshots for the same period with no unique constraint to stop them (020 deliberately has none). The comment "this in-tx check is authoritative" (line 308) is only true for sequential execution; the concurrent battery in the tests covers `ReplaceAllocations` but not `ClosePeriod`.

**Fix:** Re-run the overlap check after acquiring the entry locks (the winning close's committed header is then visible), or serialize closes per-org with `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))` before the check, or add a partial unique index / exclude constraint on overlapping periods.

### WR-04: Activity Update bypasses kind-catalog and governance-model validation → FK/CHECK violations → 500

**File:** `internal/core/services/activity/activity.go:222-255` (with `activity_repository.go:516-524` and migration `011_activity_ontology.up.sql:63`)
**Issue:** `Create` validates `KindExists` (D-2) and the handler checks `GovernanceModel.IsValid()`, but `Update` validates neither:
- `PUT /activities/{id}` with `kind: "bogus"` → repo UPDATE writes it → FK `(org_id, kind) REFERENCES activity_kinds(org_id, name)` → 23503 → `ports.ErrForeignKey` → handler default → 500.
- `PUT` with `governance_model: "bogus"` → CHECK 23514 → unmapped → 500.

Both should be clean 400s (the service's own docstring promises "every invariant maps to a sentinel, never 500").

**Fix:** In `Update`, mirror `Create`: `if req.Kind != "" { exists, err := s.activityRepo.KindExists(...); if err != nil {...}; if !exists { return nil, activitydomain.ErrInvalidRequest } }` and `if req.GovernanceModel != "" && !req.GovernanceModel.IsValid() { return nil, activitydomain.ErrInvalidRequest }`.

### WR-05: Raw unit/contract repo sentinels leak to 500 on activity Create/Update; Update never validates the contract ref

**File:** `internal/core/services/activity/activity.go:105-122, 245-253` (with `activity_handler.go:236-238, 411-413`)
**Issue:** On Create, a missing/cross-org unit returns `unitdomain.ErrUnitNotFound` (line 117) and a missing contract returns `contractdomain.ErrContractNotFound` (line 106) — both propagate raw and fall through the handler's `activitydomain.*`-only switch to 500 ("failed to create activity"). Worse, the Update path has *no* contract validation at all: `PUT` can repoint `contract_id` at another org's contract (the FK checks existence only, not org ownership), a cross-org ref that `Create` would never have allowed. The service's own convention (T-12-06 fetch-and-compare for units) is applied to units but not contracts.

**Fix:** Normalize repo sentinels to `activitydomain.ErrInvalidRequest` at the service boundary, and add the same fetch-and-compare contract validation on Update that Create performs (`contractRepo.Get` + `CreatedByOrgID`/adoption visibility).

### WR-06: `::date` casts depend on the DB session timezone — period closes can silently shift by one day

**File:** `internal/adapters/secondary/postgres/coverage_repository.go:320-346` (with `internal/db/db.go:41-61`)
**Issue:** `ClosePeriod` compares `entry_date::date BETWEEN $2::date AND $3::date` and the overlap predicate uses `$3::date`/`$2::date`. `entry_date` is TIMESTAMPTZ and the handler parses dates as UTC midnights (`time.Parse("2006-01-02", ...)`), but a `::date` cast on a TIMESTAMPTZ uses the session `TimeZone` — and `NewPool` sets no `timezone` runtime parameter, so the server's zone applies. On a non-UTC server (e.g. a VPS per ADR-BE-015 with `timezone=America/New_York`), `2026-07-01T00:00:00Z` casts to `2026-06-30`, silently freezing the wrong set of entries and shifting the overlap rejection window. The domain doc for `PeriodClose` (coverage.go:86) explicitly warns about date-part comparisons but the repo's SQL reintroduces the hazard. Tests pass only because the testcontainer runs UTC.

**Fix:** Pin the session timezone on the pool (`pgxpool.New` with `connConfig.RuntimeParams["timezone"] = "UTC"` or `?timezone=UTC` in the DSN) and/or cast explicitly: `entry_date AT TIME ZONE 'UTC' BETWEEN $2::date AND $3::date` with parameters already date-typed.

## Info

### IN-01: Propose performs entry-existence checks before the read gate — 404/403 difference leaks entry existence

**File:** `internal/core/services/coverage/coverage.go:153-169`
**Issue:** `entryRepo.GetByID` + org-scope check (→ 404) run before `readAllowed` (→ 403). An employee/customer can probe entry IDs and distinguish "exists" (403) from "missing" (404). Low risk (UUIDs, no data leaked), but the gate ordering is inverted vs. `ToCoverQueue`/`BucketBalance`, which gate first.
**Fix:** Move `readAllowed(role)` to the top of `Propose`.

### IN-02: Scope-name mismatch between handler and domain filter vocabulary

**File:** `internal/adapters/primary/http/activity_handler.go:109-112` (with `internal/core/domain/activity/activity.go:188`)
**Issue:** The handler defaults scope to `"owned"`, while the domain `ActivityFilter.Scope` documents `"adopted" | "all" | "own" (default)`. It works only because the repo's `List` default branch treats anything that isn't `"adopted"`/`"all"` as created-by-org. A future repo refactor that switches on `"own"` would silently change handler behavior.
**Fix:** Use `"own"` in the handler (or make the repo accept both explicitly).

### IN-03: Test mocks drift from the real implementations, masking the CR-01 class of bugs

**File:** `internal/core/services/testdata/mocks.go:552-573` and `internal/core/services/coverage/coverage_test.go:367-375`
**Issue:** `MockActivityRepo.Update` handles only Name/Kind/IsActive/BeneficiaryUnitID (no ContractID/ParentID/GovernanceModel/Description/Billable/BudgetAmount/IsShared), and the service tests' `contractAllocation` helper omits `ID` — the mock repo never enforces PK semantics, which is exactly why the missing-ID defect (CR-01) is invisible at both mock layers. The service tests assert "no audit written for rejected sets" against a mock that also skips Σ re-validation, so the repo remains the only real check.
**Fix:** At minimum, have the service test helper set `ID: uuid.New()` and add a note in the mocks that PK/repo-level invariants are covered only by the postgres tests.

---

_Reviewed: 2026-08-08_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
