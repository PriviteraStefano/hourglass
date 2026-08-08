---
phase: 12-coverage-backend-the-allocation-loop
plan: 04
subsystem: api
tags: [go, coverage, domain, ports, hexagonal, allocation-ledger, mock]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop
    provides: schema 019 CHECK vocabularies (12-01), ADR-BE-017 pinned decisions (12-02)
provides:
  - "coverage domain package: CoverageAllocation/Proposal/ToCoverQueueRow/PeriodClose/SnapshotRow + closed vocabularies + 6 sentinels"
  - "ports.CoverageRepository — 7-method replace-set-only contract (D-07) with in-tx audit contract (BE-016)"
  - "testdata.MockCoverageRepo — hermetic service-test driver (mutex state + ErrOn*/Fn knobs + Audits capture)"
affects: [12-05, 12-06, 12-07, phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (14000) on the same scale.
actuals:
  tokens: 5177    # chars/4 over the realized diff (20708 chars / 4)
  tasks: 2        # tasks completed
  commits: 3      # commits made (2 task commits + 1 docs metadata)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tagged-union allocation entity: discriminator + *uuid.UUID nullable refs with omitempty json tags (015/019 analog)"
    - "Closed vocabulary as exported constants mirroring schema CHECKs — single source of truth, no drift"
    - "Port doc comments state the in-tx audit contract and the replace-set-only write shape (D-07/BE-016)"
    - "Mock with precedence ErrOn* error knob → Fn override → deterministic default (mock_ticket_repo extension)"

key-files:
  created:
    - internal/core/domain/coverage/coverage.go
    - internal/core/domain/coverage/errors.go
    - internal/core/ports/coverage_repository.go
    - internal/core/services/testdata/mock_coverage_repo.go
  modified:
    - .gitignore

key-decisions:
  - "Port signatures pinned exactly as written — 12-05 (service) and 12-06 (repo) compile against them: ClosePeriod takes the caller-supplied close id (uuid.New()), ToCoverQueue returns raw rows (proposal enrichment is service-side, D-06)"
  - "MockCoverageRepo default behaviors kept minimal and deterministic for service tests: ReplaceAllocations stores + records audit; ClosePeriod stores the primed ClosePeriodResult under Snapshots[closeID] (or ErrPeriodAlreadyClosed when ClosePeriodOverlapping primed); BucketBalance returns the settable BucketBalanceResult"
  - ".gitignore 'coverage/' anchored to '/coverage/': the unanchored pattern shadowed internal/core/domain/coverage/ and blocked tracking of the new package"

patterns-established:
  - "Mock state keyed by aggregate id (Allocations by entry id, Snapshots by close id) + Audits capture for in-tx audit assertions"
  - "Entity structs carry json tags and *uuid.UUID for nullable refs — the Phase 17 read-models consume the same shapes"

requirements-completed: [COV-01, COV-02, COV-03, COV-04]

coverage:
  - id: D1
    description: "coverage domain package — CoverageAllocation entity + CoverageProposal/ToCoverQueueRow/PeriodClose/SnapshotRow read models, closed vocabulary constants (entry_type/source_type/absorption reasons/audit actions) mirroring the 019 schema CHECKs, and 6 sentinels + JSONNames"
    requirement: COV-01
    verification:
      - kind: other
        ref: "go build ./internal/core/domain/coverage/ && go vet ./internal/core/domain/coverage/ (exit 0) + vocabulary/field-set/JSONNames assertion test (PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "ports.CoverageRepository — exactly 7 methods (ReplaceAllocations, ListByEntry, ToCoverQueue, BucketBalance, ClosePeriod, GetSnapshot, ListHistory); replace-set-only write shape (no incremental CRUD, D-07); doc comments pin the in-tx audit contract and FOR UPDATE re-validation"
    requirement: COV-03
    verification:
      - kind: other
        ref: "go build ./... (exit 0) + go vet ./internal/core/ports/ (exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "testdata.MockCoverageRepo — compile-time var _ ports.CoverageRepository assertion, mutex-guarded Allocations/Snapshots/Audits state, per-method ErrOn* error knobs + Fn overrides, deterministic defaults incl. ClosePeriodOverlapping -> ErrPeriodAlreadyClosed; hermetic driver for 12-05 service tests"
    requirement: COV-04
    verification:
      - kind: other
        ref: "go test ./internal/core/services/testdata/ -count=1 (ok) — mock satisfies the port via the compile-time assertion"
        status: pass
    human_judgment: false

# Metrics
duration: 3min
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 04: Coverage Domain, Port & Mock Summary

**The coverage plane's shared contracts: domain package (entity + read models + closed vocabularies + sentinels), the 7-method replace-set-only CoverageRepository port, and the hermetic MockCoverageRepo — the compile-time anchor plans 12-05 (service) and 12-06 (repo) both build against**

## Performance

- **Duration:** 3 min
- **Started:** 2026-08-08T09:06:13Z
- **Completed:** 2026-08-08T09:08:43Z
- **Tasks:** 2
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments

- `internal/core/domain/coverage/` — `CoverageAllocation` tagged-union entity (D-01) with `*uuid.UUID` nullable refs and json tags (ticket.go shape); read models `CoverageProposal` (Flagged/FlagReason carry the D-06 no-source state), `ToCoverQueueRow` (raw covered/uncovered split + proposal slot), `PeriodClose` (DATE-semantics doc comment — compare via Format, entry_date is TIMESTAMPTZ) and `SnapshotRow` (entry-level, D-11)
- Closed vocabulary exported as constants mirroring the 019 schema CHECKs exactly: `EntryTypeTime`, `SourceTypeContract/Absorption/Transfer` (A1 — 3 row-level draws, five sources derived from the referenced contract), `AbsorptionReasonWarrantyBug/UnderEstimate/Goodwill` (COV-02, A2 — the Part-5 "plain internal" superseded), `AuditActionAllocationsSet/CoverageClosed` + `AuditEntityCoverageAllocation` (A7 — repo and service can never drift)
- `errors.go` — six sentinels (`ErrEntryNotCoverable`, `ErrAllocationSumMismatch`, `ErrPeriodAlreadyClosed`, `ErrForbidden`, `ErrInvalidRequest`, `ErrNotFound`) + `JSONNames` map with an entry per sentinel (ticket analog)
- `ports.CoverageRepository` — exactly the seven pinned methods; replace-set is the ONLY write shape (D-07 prohibition honored: no incremental CRUD, no stored balances/proposals); doc comments state the in-tx audit contract (BE-016) and the FOR UPDATE re-validation contract (CR-01)
- `testdata.MockCoverageRepo` — mutex-guarded `Allocations` (keyed by entry id) + `Snapshots` (keyed by close id) + `Audits` capture; per-method `ErrOn*` error knobs and `Fn` overrides; deterministic defaults (ClosePeriod stores the primed `ClosePeriodResult`, `ClosePeriodOverlapping` → `ErrPeriodAlreadyClosed`); compile-time assertion `var _ ports.CoverageRepository = (*MockCoverageRepo)(nil)`

## Task Commits

Each task was committed atomically:

1. **Task 1: coverage domain package — entity, read models, constants, sentinels** - `98edb5f` (feat)
2. **Task 2: ports.CoverageRepository + testdata.MockCoverageRepo** - `ff902e5` (feat)

**Plan metadata:** (pending final metadata commit)

## Files Created/Modified

- `internal/core/domain/coverage/coverage.go` - CoverageAllocation + 4 read models + 10 vocabulary constants (schema-CHECK mirror)
- `internal/core/domain/coverage/errors.go` - 6 sentinels + JSONNames (ticket analog)
- `internal/core/ports/coverage_repository.go` - 7-method replace-set-only interface with in-tx audit contract docs
- `internal/core/services/testdata/mock_coverage_repo.go` - MockCoverageRepo (state + knobs + Audits capture)
- `.gitignore` - `coverage/` anchored to `/coverage/` (Rule 3 deviation, see below)

## Decisions Made

- **Port signatures pinned verbatim** — `ClosePeriod` receives the caller-supplied close id (`uuid.New()` per the plan) and returns the full `PeriodClose` incl. rows (OQ4); `ToCoverQueue` returns raw uncovered rows with the proposal slot left for service-side enrichment (D-06); `BucketBalance` allows negatives (D-03) — all contract points 12-05/12-06 compile against
- **Mock default behaviors minimal and deterministic** — ReplaceAllocations stores the set + records the audit; ClosePeriod stores the primed snapshot under `Snapshots[closeID]`; `ClosePeriodOverlapping` primes the 409 path; `BucketBalanceResult` is the settable balance — service tests drive every method without a DB

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] .gitignore 'coverage/' shadowed the new domain package**
- **Found during:** Task 1 commit (stage failed: "The following paths are ignored by one of your .gitignore files: internal/core/domain/coverage")
- **Issue:** The GSD baseline `.gitignore` entry `coverage/` (line 73) is unanchored — intended for root-level Go coverage output, it matched ANY directory named `coverage` at any depth, silently ignoring the new `internal/core/domain/coverage/` package.
- **Fix:** Anchored the pattern to the repo root (`/coverage/`), preserving the Go-coverage-output intent while letting the domain package track.
- **Files modified:** `.gitignore`
- **Verification:** `git add internal/core/domain/coverage/` succeeds; `git status` shows the files tracked; commit landed.
- **Committed in:** `98edb5f` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for the deliverable to exist in git at all — without the anchor, the domain package (the entire Task 1 artifact and the contract 12-05/12-06 compile against) could not be committed. No scope creep.

## Issues Encountered

None beyond the auto-fixed .gitignore shadowing above. Both task verification commands (`go build ./internal/core/domain/coverage/ && go vet`, `go build ./... && go test ./internal/core/services/testdata/`) and the plan-level verification (`go build ./...`, `go vet ./internal/core/domain/coverage/ ./internal/core/ports/`) passed on first run; the pre-commit graphify hook rebuilt the knowledge graph cleanly on both commits.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The compile-time anchor for plans 12-05 (service) and 12-06 (repo) is committed: every type, constant, sentinel, and method signature they reference now exists and builds
- 12-05 service tests can drive every repo method deterministically via MockCoverageRepo (state assertions via `Allocations`/`Snapshots`, audit assertions via `Audits`, 409 path via `ClosePeriodOverlapping`)
- 12-06 repo must implement exactly the seven signatures — a signature drift fails at compile time on both sides

---

*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*

## Self-Check: PASSED
- All 4 key files exist on disk (coverage.go, errors.go, coverage_repository.go, mock_coverage_repo.go) + SUMMARY
- All 3 commits present in git history (98edb5f, ff902e5, 090f75a)
- `go build ./...` green; `go vet ./internal/core/domain/coverage/ ./internal/core/ports/` green
- `go test ./internal/core/services/testdata/ -count=1` ok (mock satisfies the port)
