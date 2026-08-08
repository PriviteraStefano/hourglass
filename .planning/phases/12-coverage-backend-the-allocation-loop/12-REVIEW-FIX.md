---
phase: 12
fixed_at: 2026-08-08T10:35:45Z
review_path: .planning/phases/12-coverage-backend-the-allocation-loop/12-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 12: Code Review Fix Report

**Fixed at:** 2026-08-08T10:35:45Z
**Source review:** `.planning/phases/12-coverage-backend-the-allocation-loop/12-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope (critical + warning): 7
- Fixed: 7
- Skipped: 0

**Verification environment:** `go build ./...` and `go test ./...` (full
suite incl. the testcontainers integration packages) ran inside the isolated
review-fix git worktree (branch `gsd-reviewfix/12-47412`, worktree
`/tmp/sv-12-reviewfix-8FtM4Z`), then fast-forwarded onto `main`. The
numbers below are reproducible from that tree state; the same suite passed
on the merged branch.

## Fixed Issues

### CR-01: Allocation rows are inserted with `uuid.Nil` IDs — the ledger can hold at most one row in the entire database

**Files modified:** `internal/adapters/secondary/postgres/coverage_repository.go`
**Commit:** `2ce8f7e`
**Applied fix:** `ReplaceAllocations` now generates `a.ID = uuid.New()` for
every row whose ID is `uuid.Nil` before the INSERT (the boundary DTO never
carries allocation ids, D-07). The table-wide PK can no longer collide on
the second row anywhere in the ledger. The port contract stays intact —
callers still supply rows without ids; generated ids surface on the
read-back.
**Status:** `fixed: requires human verification` — the nil-ID path is not
directly exercised by an integration test (the repo test helper sets
`ID: uuid.New()` explicitly, per IN-03), so the multi-row/multi-entry
behavior deserves a manual smoke test (e.g. two PUTs with ≥2 rows against a
live DB).

### WR-01: Fractional-cent hours pass service validation, then hit the unmapped DB CHECK → 500

**Files modified:** `internal/core/services/coverage/coverage.go`, `internal/core/services/coverage/coverage_test.go`
**Commits:** `adc5f4e` (fix), `f2aab15` (regression test)
**Applied fix:** Step 5 of `ReplaceAllocations` now rejects any hours value
that is not a positive whole-cent amount
(`a.Hours <= 0 || math.Round(a.Hours*100) != a.Hours*100`) as
`coverage.ErrInvalidRequest` (400). A compensated set like `7.999 + 0.001`
against an 8h entry previously passed the cents Σ fast-fail and died on the
`DECIMAL(8,2) hours > 0` CHECK (23514, unmapped in `wrapPGError` → 500);
it is now a clean 400, and the stored Σ can no longer silently diverge from
the validated cents sum. New service test: `fractional-cent hours rejected
(WR-01)` asserts `ErrInvalidRequest` for `7.999` on an 8h entry.
**Status:** `fixed`

### WR-02: ClosePeriod accepts inverted periods (start > end), permanently poisoning the range

**Files modified:** `internal/core/services/coverage/coverage.go`, `internal/core/services/coverage/coverage_test.go`
**Commits:** `c6e984b` (fix), `f2aab15` (regression test)
**Applied fix:** `ClosePeriod` now validates `periodStart.After(periodEnd)`
→ `coverage.ErrInvalidRequest` (400) before the repo call. An inverted
close previously "succeeded" with an empty snapshot and the inclusive
overlap predicate then rejected every legitimate close of that range
forever (snapshots are append-only). New service test: `inverted period
rejected (WR-02)`.
**Status:** `fixed`

### WR-03: ClosePeriod overlap check is not concurrency-safe — duplicate overlapping closes can both commit

**Files modified:** `internal/adapters/secondary/postgres/coverage_repository.go`
**Commit:** `3f7ddd2`
**Applied fix:** `ClosePeriod` now takes
`SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))` keyed on the
org id at the very start of the transaction, before the overlap EXISTS.
Closes for an org are serialized (xact-scoped, released at commit), so the
loser's overlap check runs only after the winner's transaction committed
and sees its header → correct 409. Chosen over a re-check-after-lock
because the entry FOR UPDATE locks nothing when the period is empty — the
advisory lock is airtight for all periods. Closes are rare manager-only
operations, so per-org serialization is free.
**Status:** `fixed: requires human verification` — the concurrent-close
race is not covered by a test (the existing concurrent battery covers
`ReplaceAllocations` only); a manual two-goroutine close test against a
live DB would confirm the 409 under contention.

### WR-04: Activity Update bypasses kind-catalog and governance-model validation → FK/CHECK violations → 500

**Files modified:** `internal/core/services/activity/activity.go`
**Commit:** `9bc4cff`
**Applied fix:** `Update` now mirrors `Create`: when `Kind` is set it must
exist in the org's `activity_kinds` catalog (`KindExists`), and a non-empty
`GovernanceModel` must pass `IsValid()` — both mapping to
`activitydomain.ErrInvalidRequest` (400). A bogus kind previously wrote
through to the FK (23503 → 500); a bogus governance model to the unmapped
CHECK (23514 → 500).
**Status:** `fixed`

### WR-05: Raw unit/contract repo sentinels leak to 500 on activity Create/Update; Update never validates the contract ref

**Files modified:** `internal/core/services/activity/activity.go`, `internal/core/services/activity/activity_test.go`, `internal/core/services/activity/activity_beneficiary_unit_test.go`
**Commit:** `cf99cd5`
**Applied fix:**
- `Create`: `contractdomain.ErrContractNotFound` and
  `unitdomain.ErrUnitNotFound` are normalized to
  `activitydomain.ErrInvalidRequest` at the service boundary (other errors
  still propagate) — both previously fell through the handler's
  `activitydomain.*`-only switch to a 500.
- `Update`: same normalization on the beneficiary-unit path, and the
  missing contract validation was added — `contractRepo.Get` + org
  visibility (`CreatedByOrgID == orgID || (IsShared && IsAdopted)`, the
  same predicate the coverage service applies to allocation refs), so PUT
  can no longer repoint `contract_id` at another org's contract.
- Tests updated to assert the normalized 400 sentinel; new
  `TestService_UpdateRefValidation` battery covers kind, governance model,
  missing and cross-org contract on Update, plus the same-org contract
  happy path.
**Status:** `fixed`

### WR-06: `::date` casts depend on the DB session timezone — period closes can silently shift by one day

**Files modified:** `internal/db/db.go`, `internal/adapters/secondary/postgres/coverage_repository.go`
**Commit:** `4b491d2`
**Applied fix:**
- `db.NewPool` now pins the session timezone (`timezone=UTC` via
  `pgxpool.ParseConfig` + `NewWithConfig`) — the root cause: a non-UTC VPS
  previously applied the server zone to every `::date` cast.
- The `ClosePeriod` SQL is now session-timezone-independent on all three
  touch points: the overlap predicate and header insert cast parameters
  UTC-explicitly (`($n AT TIME ZONE 'UTC')::date`), and the freeze
  predicate compares truncated dates on both sides —
  `(entry_date AT TIME ZONE 'UTC')::date BETWEEN ($2 AT TIME ZONE
  'UTC')::date AND ($3 AT TIME ZONE 'UTC')::date`. The field-side
  truncation matters: a raw `AT TIME ZONE 'UTC'` (no `::date`) yields a
  timestamp, and a boundary entry at e.g. noon on the last day compared
  against the midnight-truncated period bound was silently excluded —
  caught by `TestCoverageRepository_ClosePeriod_Scope` and fixed before
  the commit was finalized.
**Status:** `fixed` — the integration suite (incl. the boundary-inclusive
scope test and the overlap/duplicate-close tests) exercises the exact
date-cast semantics.

## Skipped Issues

None — all 7 in-scope findings were fixed.

## Info Findings (documented, not fixed — out of scope)

- **IN-01:** `Propose` performs entry-existence checks before the read gate
  (404/403 ordering leaks entry existence). Service ordering unchanged.
- **IN-02:** Handler scope defaults to `"owned"` while the domain filter
  documents `"own"` — works via the repo's default branch. Unchanged.
- **IN-03:** Test mocks drift from the real implementations (the service
  test helper `contractAllocation` omits `ID`; `MockActivityRepo.Update`
  handles a subset of fields) — the very reason CR-01 was invisible at the
  mock layers. Unchanged; the CR-01 fix lives in the postgres repo, which
  the integration tests exercise with real PK semantics.

---

_Fixed: 2026-08-08T10:35:45Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
