---
phase: 12-coverage-backend-the-allocation-loop
plan: 07
subsystem: api
tags: [go, coverage, http, handler, sentinel-map, permission-matrix, wiring]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-05 coverage service (7 methods, sentinels, D-04/D-08 gates)
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-06 postgres CoverageRepository (7-method port impl)
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-04 coverage domain package (entities + sentinels + JSONNames)
provides:
  - "CoverageHandler: the full 8-route HTTP surface (replace-set PUT, read-back GET, proposals, to-cover queue, bucket balance, close, snapshots, history) behind middleware.Auth with boundary DTOs and the sentinel→status map"
  - "The coverage stack live in cmd/server/main.go: coverageRepo → NewService (six deps incl. the SHARED routing service) → NewCoverageHandler → eight Go 1.22 route registrations"
  - "Handler integration tests proving the D-08 permission matrix end-to-end (claims → gate → status): approver manager 200, owner/employee/finance/customer 403 on PUT, finance 200 on reads, employee/customer 403 on reads"
affects: [phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (25000) on the same scale.
actuals:
  tokens: 14500    # chars/4 over the realized diff (58000 chars / 4)
  tasks: 2         # tasks completed
  commits: 3       # 2 task commits + 1 docs metadata

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Thin boundary handlers: claims via middleware.Get*; r.PathValue for path params; parseOptionalUUID at the boundary; writeError switch maps the six coverage sentinels (404/400/403/409/500)"
    - "Read-back via the pinned Propose read path: the 12-05 service has no ListByEntry method — GetAllocations reuses Propose's (proposal, allocs) return instead of extending the service surface"

key-files:
  created:
    - internal/adapters/primary/http/coverage_handler.go
    - internal/adapters/primary/http/coverage_handler_test.go
  modified:
    - cmd/server/main.go
    - internal/adapters/primary/http/handler_test_helper.go
    - internal/core/services/coverage/coverage.go

key-decisions:
  - "GetAllocations (GET /time-entries/{id}/allocations) delegates to service.Propose and returns the allocations element: the 12-05-pinned service surface has no ListByEntry method (Propose returns the current set alongside the proposal), so the thin handler reuses the pinned read path instead of adding an unplanned service method"
  - "time_entry.ErrTimeEntryNotFound normalizes to coverage.ErrEntryNotCoverable inside the service (Propose + ReplaceAllocations): the postgres entry repo returns its own sentinel for missing rows, which would surface as a 500 at the boundary — the service is the normalization point, the handler stays thin"
  - "Close returns the frozen PeriodClose incl. rows at 201 (OQ4); overlap → 409; date parse errors → 400 with field-specific messages; org strictly from claims (T-12-19)"

patterns-established:
  - "Test fixture (handler_test_helper.go) wires the coverage stack exactly as main.go — the handler integration tests hit the real mux through httptest, matching the ticket handler test conventions"
  - "Permission-matrix integration tests: seed unit + unit membership (manager) so routing.ResolveManagerStage resolves ApproverIDs through the real R-2 fallback — no mocks at the boundary"

requirements-completed: [COV-01, COV-02, COV-03, COV-04, COV-05]

coverage:
  - id: D1
    description: "Replace-set write endpoint (PUT /time-entries/{id}/allocations) with boundary DTO parsing (string ids → UUIDs), the D-08 permission matrix enforced end-to-end (approver manager 200 via unit-member resolution; owner 403 structural barrier; employee/finance/customer 403), and the sentinel map (Σ mismatch 400, non-coverable 404) — all behind middleware.Auth"
    requirement: COV-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/coverage_handler_test.go#TestCoverageHandler (steps 1-6, 13 — PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Read surface: to-cover queue (finance 200 with flagged no-source proposal rows; employee/customer 403), proposal read (200 with proposal+allocations, 404 for nonexistent entry), allocations read-back, bucket balance (100 sold − 8 drawn = 92), snapshot read, and the append-only history stream (allocations-set)"
    requirement: COV-02
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/coverage_handler_test.go#TestCoverageHandler (steps 7-10, 12 — PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Period close (POST /coverage/close): org from claims only, 2006-01-02 date parsing with 400 on malformed dates, manager-only gate (service), 201 with the frozen PeriodClose incl. rows (OQ4), inclusive-overlap → 409, snapshot read-back by close id"
    requirement: COV-04
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/coverage_handler_test.go#TestCoverageHandler (step 11 — PASS)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Server wiring: coverageRepo → coveragesvc.NewService (six deps, single shared routing.Service — no second instance) → NewCoverageHandler, eight routes registered with middleware.Auth using the Go 1.22 method pattern; full suite green"
    requirement: COV-05
    verification:
      - kind: integration
        ref: "go build ./... && make test (full suite, all packages ok — PASS)"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 07: Coverage HTTP Surface — The Allocation Loop Summary

**The coverage plane's HTTP surface: a thin eight-route handler (replace-set PUT, read-back, proposals, to-cover queue, bucket balance, close, snapshots, history) with boundary DTOs, the sentinel→status map (404/400/403/409/500), and the cmd/server wiring that brings service, repo, and routing together — proven by integration tests running the full D-08 permission matrix (approver manager 200, owner/employee/finance/customer 403) against the real stack**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-08T09:54:57Z
- **Completed:** 2026-08-08T10:00:44Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- **`CoverageHandler` — eight routes, thin per ADR-BE-002**: `PutAllocations` (D-07 replace-set; the D-08 gate runs in the service, the boundary only parses), `GetAllocations` (read-back via the pinned Propose path), `GetProposal` (computed proposal + current set), `GetToCoverQueue` (D-06, no-source rows flagged), `GetBucketBalance` (D-02 derived balance, negatives as-is), `PostClose` (D-12 — org from claims, `2006-01-02` dates, 201 with the frozen `PeriodClose` incl. rows per OQ4), `GetSnapshot`, `GetHistory` (A7 audit stream). All claims come from `middleware.GetOrganizationID/GetUserID/GetRole`; path params via `r.PathValue`; string IDs parsed with the shared `parseOptionalUUID`.
- **Sentinel map at the boundary**: `ErrEntryNotCoverable`/`ErrNotFound` → 404, `ErrAllocationSumMismatch`/`ErrInvalidRequest` → 400, `ErrForbidden` → 403, `ErrPeriodAlreadyClosed` → 409, default → 500 — the ticket_handler `writeError` shape on the `pkg/api` envelope. Close body parse failures map to field-specific 400s (`invalid period_start`/`invalid period_end`).
- **Permission matrix proven end-to-end** (`TestCoverageHandler`): a unit with a manager membership drives `routing.ResolveManagerStage` through the real R-2 fallback, so the tests exercise claims → gate → status for real — approver manager PUT 200, entry owner 403 (structural self-barrier), employee/finance/customer 403 on PUT, finance 200 on the to-cover queue, employee/customer 403 on reads. Plus the sentinel battery: Σ mismatch 400, overlapping close 409 (contained period), ghost-entry proposal 404, bad dates 400, close returns its rows, and the D-07 prohibition asserted — `DELETE /time-entries/{id}/allocations` is 405 (no route).
- **Server wiring** (`cmd/server/main.go`): `coverageRepo := postgres.NewCoverageRepository(pool)` → `coveragesvc.NewService(coverageRepo, activityRepo, contractRepo, unitRepo, timeEntryRepo, routingSvc)` → `NewCoverageHandler` — reusing the single shared `routing.Service` (BE-014 parity; no second instance) — with exactly the eight routes under `middleware.Auth`, matching the Go 1.22 method pattern. The test fixture wires the same stack so handler integration tests mirror production routing.

## Task Commits

Each task was committed atomically:

1. **Task 1: coverage_handler.go — eight routes, DTOs, sentinel map + permission matrix tests** - `45ab7f1` (feat)
2. **Task 2: cmd/server/main.go wiring — service, handler, eight routes** - `b8a1709` (feat)

**Plan metadata:** pending (this docs commit)

## Files Created/Modified

- `internal/adapters/primary/http/coverage_handler.go` - CoverageHandler + NewCoverageHandler; DTOs ReplaceAllocationsRequest/AllocationRequest (string ids) + ClosePeriodRequest; eight methods; writeError sentinel map; derefStr helper
- `internal/adapters/primary/http/coverage_handler_test.go` - coverageAPIHelper + TestCoverageHandler: full permission matrix + sentinel battery + envelope + no-DELETE assertion (13 steps)
- `cmd/server/main.go` - coverageRepo/coverageService/coverageHandler construction (shared routingSvc) + eight route registrations under middleware.Auth
- `internal/adapters/primary/http/handler_test_helper.go` - fixture gains the coverage service + handler + eight routes (mirrors main.go)
- `internal/core/services/coverage/coverage.go` - Propose + ReplaceAllocations normalize `time_entry.ErrTimeEntryNotFound` → `coverage.ErrEntryNotCoverable`

## Decisions Made

- **GetAllocations reuses the pinned Propose read path** — the 12-05 service surface has no `ListByEntry` method (Propose returns the current allocations alongside the proposal); adding a dedicated read method would extend the pinned surface beyond this plan's files, so the thin handler consumes `(_, allocs, err)` and the read gate comes along for free.
- **Sentinel normalization lives in the service, not the handler** — the postgres entry repo returns its own `time_entry.ErrTimeEntryNotFound` for missing rows; mapping it in `Propose`/`ReplaceAllocations` keeps the coverage handler free of cross-domain sentinels and makes the 404 contract hold at every entry lookup.
- **Close semantics per plan**: 201 + full `PeriodClose` (OQ4), inclusive overlap → 409, malformed `period_start`/`period_end` → 400 with field-specific messages, org strictly from claims (T-12-19) — the body carries only the period.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Ghost-entry proposal returned 500 instead of the pinned 404**
- **Found during:** Task 1 verification (`TestCoverageHandler` step 9)
- **Issue:** `Propose` propagates the entry repo's `time_entry.ErrTimeEntryNotFound` verbatim; the handler's `writeError` has no mapping for it → default 500. The plan's sentinel map pins nonexistent-entry reads to 404.
- **Fix:** `Propose` and `ReplaceAllocations` now normalize `time_entrydomain.ErrTimeEntryNotFound` → `coverage.ErrEntryNotCoverable` (the 404 sentinel) at the entry-fetch step — the service is the normalization point, the boundary stays thin.
- **Files modified:** `internal/core/services/coverage/coverage.go`
- **Verification:** `TestCoverageHandler` step 9 (ghost proposal → 404) passes; service package suite still green.
- **Committed in:** `45ab7f1` (Task 1 commit)

**2. [Rule 3 - Blocking] Plan referenced a service `ListByEntry` read that does not exist**
- **Found during:** Task 1 (GetAllocations implementation)
- **Issue:** The plan's action says `GetAllocations → service read (ListByEntry)`, but the 12-05-pinned service surface exposes no `ListByEntry` — its read path `Propose` returns the current allocation set alongside the computed proposal. Adding a new service method would extend the pinned surface outside this plan's file scope.
- **Fix:** `GetAllocations` calls `service.Propose` and returns the allocations element; the manager|finance read gate and 404 semantics apply unchanged.
- **Files modified:** `internal/adapters/primary/http/coverage_handler.go`
- **Verification:** read-back step (10) asserts one stored row returned; ghost read-back path covered by the proposal 404 test (same code path).
- **Committed in:** `45ab7f1` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 plan-reference mismatch)
**Impact on plan:** Both fixes required to honor the plan's own sentinel contract and the 12-05-pinned service surface. No scope creep; the eight must-have truths hold and all threat mitigations (T-12-18/19/20/21) are exercised by the tests.

## Issues Encountered

- **Test session ordering:** the manager's `registerUserInOrg` (which logs in + switches org) took the shared cookie jar, so the owner-403 assertion initially ran as the manager and returned 200. Fixed by restoring the owner session before the assertion — the D-08 gate itself was correct (the structural self-barrier fired once the right session was active).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The coverage plane is fully observable at the API boundary: all eight routes live in the server binary behind `middleware.Auth`, the sentinel map is consistent with the envelope, and the D-08 permission semantics are proven end-to-end by integration tests.
- `make test` is green across the full suite (all packages incl. testcontainers-based postgres + http).
- Phase 17 surfaces consume the read-models (queue, proposals, balance, snapshots, history) exactly as registered here; the flagged `no eligible source` queue rows and the frozen close rows are ready for UI wiring.
- The 12-07 flagged assumption (reads are manager+finance; employee own-coverage is Phase 17 SURF-03) is honored — no employee-facing read endpoint was added.

## Self-Check: PASSED

- [x] `internal/adapters/primary/http/coverage_handler.go` exists — 8 methods + writeError + DTOs
- [x] `internal/adapters/primary/http/coverage_handler_test.go` exists — TestCoverageHandler 13-step matrix
- [x] `cmd/server/main.go` — coverageService with shared routingSvc + 8 route registrations (grep-verified)
- [x] Commit `45ab7f1` (Task 1) and `b8a1709` (Task 2) present in git log
- [x] `go test ./internal/adapters/primary/http/ -run 'TestCoverageHandler' -count=1` — ok
- [x] `go build ./...` — exit 0
- [x] `make test` — full suite green (all packages ok, no FAIL)
- [x] No second routing.Service instance (single `routing.NewService` in main.go)

---
*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*
