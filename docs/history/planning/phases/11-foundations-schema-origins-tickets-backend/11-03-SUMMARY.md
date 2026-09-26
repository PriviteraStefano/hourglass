---
phase: 11-foundations-schema-origins-tickets-backend
plan: 03
subsystem: api
tags: [go, hexagonal, routing, approval, adr-be-014, working-groups, units]

# Dependency graph
requires:
  - phase: 10-backend-foundations
    provides: time_entry service with private resolveManagerStage/resolveUnitManager (ADR-BE-014 R-1/R-2), WG/activity/unit repos + testdata mocks
provides:
  - shared internal/core/services/routing package: NewService, ResolveManagerStage, ResolveUnitManager (BE-014 semantics verbatim, D-G parity)
  - routing unit tests pinning R-1 anchored-WG approver set, R-2 commercial rejection + unit-manager fallback, upward walk, terminal role-gated, D-11 skip
  - time_entry service delegating to routing at Submit/Approve call sites (identical behavior, suite green)
affects: [11-foundations-schema-origins-tickets-backend plan 05 (proposal approval consumes routing.Service), 12-funding-sources]

# Tech tracking
tech-stack:
  added: [none — pure Go refactor on the existing dependency set]
  patterns:
    - "Shared approval-routing service (Pattern 5 extraction): entry approval and proposal approval consume the same manager-stage resolution so routing rules cannot drift (D-G)"
    - "Unexported resolution struct with exported fields — callers outside the package consume the resolution through the returned pointer; type remains an implementation detail"

key-files:
  created:
    - internal/core/services/routing/routing.go
    - internal/core/services/routing/routing_test.go
  modified:
    - internal/core/services/time_entry/time_entry.go
    - internal/core/services/time_entry/time_entry_test.go
    - cmd/server/main.go
    - cmd/server/main_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - internal/adapters/primary/http/time_entry_test.go

key-decisions:
  - "managerResolution fields exported (ApproverIDs/RoleGated/SkipToFinance) so cross-package callers can consume the resolution; the struct type stays unexported — plan's literal 'unexported fields' reading cannot compile across packages (Go visibility rule)"
  - "routing.Service constructed once in cmd/server wiring and shared: time_entry now, proposal approval (plan 05) later — single instance, single repo set"

patterns-established:
  - "New service packages follow the svc-suffix import alias convention in cmd/server wiring (routing imported bare per plan's call-shape contract)"

requirements-completed: [FND-02]

# Metrics
duration: 6min
completed: 2026-08-07
---

# Phase 11 Plan 3: Manager-Stage Routing Extraction Summary

**BE-014 manager-stage resolution extracted verbatim into a new shared `internal/core/services/routing` package — the time_entry service delegates at Submit/Approve, its suite stays green, and proposal approval (plan 05) will consume the same routing so entry and proposal approval cannot drift (D-G parity)**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-07T09:57:26Z
- **Completed:** 2026-08-07T10:03:54Z
- **Tasks:** 2
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- **New `routing` package** (`internal/core/services/routing/routing.go`): `managerResolution` (ApproverIDs, RoleGated, SkipToFinance with the D-11 comment), `Service` holding wgRepo/activityRepo/unitRepo, `NewService`, `ResolveManagerStage`, `ResolveUnitManager`. Extracted verbatim (Pattern 5) — anchored-WG approver set via `ListByOrg(&activityID)`, R-2 enforcement returning `activity.ErrActivityNotLoggable` for commercial-without-WG, unit-manager upward walk with `ErrUnitNotFound` short-circuit, terminal role-gated case, D-11 skip including delegates. ADR-BE-014 R-1/R-2 doc comments preserved.
- **time_entry service refactored to delegate**: private `resolveManagerStage`/`resolveUnitManager`/`managerResolution` removed; `routing *routing.Service` field + appended `NewService` param; both call sites (Submit L144, Approve L183) call `s.routing.ResolveManagerStage(...)`. `unit` import pruned (was the moved code's only user); `errors` kept (Approve's ErrActivityNotLoggable mapping). `IsWGManager`, `contains`, `strPtr`, `timePtr` untouched.
- **Wiring**: `cmd/server/main.go` + `main_test.go` construct `routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)` once and pass it to the time_entry service — the same instance is reused later by the activity service (plan 05).
- **Six routing unit tests** pin the BE-014 semantics: R-1 approver set + D-11 skip (manager, delegate, non-member), R-2 commercial rejection, R-2 unit fallback incl. owner-is-manager skip, upward walk, terminal role-gated, ResolveUnitManager not-found.
- **Full repo suite green**: `go test ./... -count=1` passes in every package (wave gate — valid because 11-01's migration work already landed in this wave).

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract routing to a shared package** - `d6825e0` (refactor)
2. **Task 2: Routing unit tests + time_entry suite green** - `0e916a1` (test)
3. **Rule 3 fix: http handler unit-test fixture wiring** - `8e261d7` (fix)

**Plan metadata:** `(docs commit below)`

## Files Created/Modified

- `internal/core/services/routing/routing.go` - NEW: managerResolution + Service + NewService + ResolveManagerStage + ResolveUnitManager, verbatim BE-014 semantics
- `internal/core/services/routing/routing_test.go` - NEW: six scenario tests over testdata mocks (testify)
- `internal/core/services/time_entry/time_entry.go` - removed private routing code; Service gains routing field + NewService param; Submit/Approve delegate
- `internal/core/services/time_entry/time_entry_test.go` - fixture passes routing.NewService over the same mock repos
- `cmd/server/main.go` - constructs routingSvc once, passes to tesvc.NewService
- `cmd/server/main_test.go` - same wiring in the smoke-test server
- `internal/adapters/primary/http/handler_test_helper.go` - newHandlerFixture wires routingSvc (third construction site, plan-missed)
- `internal/adapters/primary/http/time_entry_test.go` - newTEService fixture wires routingSvc (fourth construction site, plan-missed)

## Decisions Made

- **managerResolution fields exported** (`ApproverIDs`/`RoleGated`/`SkipToFinance`) while the struct type stays unexported. The plan required "unexported struct with consumption at call sites stays identical" — but Go visibility forbids another package from reading unexported fields, so the literal instruction could not compile. Exporting the fields is the minimal change preserving the plan's intent (type = implementation detail; values flow to callers identically).
- **Single routing instance in wiring**: constructed next to the other services and shared by time_entry now and proposal approval (plan 05) later — no duplicate repo sets.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] managerResolution fields exported for cross-package consumption**
- **Found during:** Task 1 (first compile)
- **Issue:** Plan required the struct to stay unexported with unexported fields (`approverIDs`, `roleGated`, `skipToFinance`) consumed from the time_entry package — Go does not allow another package to read unexported fields, so `go build` failed on the three call-site references.
- **Fix:** Exported the three fields (type name stays `managerResolution`, unexported). Call-site consumption (`res.SkipToFinance`, `res.RoleGated`, `res.ApproverIDs`) is unchanged in shape; BE-014 semantics byte-identical.
- **Files modified:** internal/core/services/routing/routing.go, internal/core/services/time_entry/time_entry.go
- **Verification:** `go build ./...` + `go vet` pass; full suite green
- **Committed in:** d6825e0 (Task 1 commit)

**2. [Rule 3 - Blocking] Third tesvc.NewService construction site missed by the plan**
- **Found during:** Task 1 (go build)
- **Issue:** `internal/adapters/primary/http/handler_test_helper.go` (newHandlerFixture) also constructs the time-entry service with the pre-extraction signature; the plan's files list only covered cmd/server/main.go + main_test.go.
- **Fix:** Wired `routingSvc := routing.NewService(wgRepo, activityRepo, unitRepo)` in the fixture, matching main.go's wiring.
- **Files modified:** internal/adapters/primary/http/handler_test_helper.go
- **Verification:** `go build ./...` passes
- **Committed in:** d6825e0 (Task 1 commit)

**3. [Rule 3 - Blocking] Fourth construction site in the http handler unit-test fixture**
- **Found during:** Task 2 (wave-gate full-suite run)
- **Issue:** `internal/adapters/primary/http/time_entry_test.go` newTEService used the old signature; `go test ./...` reported a build failure in the http package that the two-package task gate could not see.
- **Fix:** Passed `routing.NewService(f.wgRepo, f.activityRepo, f.unitRepo)` into tesvc.NewService.
- **Files modified:** internal/adapters/primary/http/time_entry_test.go
- **Verification:** `go test ./... -count=1` green in all 21 packages
- **Committed in:** 8e261d7 (separate fix commit)

---

**Total deviations:** 3 auto-fixed (all Rule 3 blocking — compile/visibility requirements)
**Impact on plan:** All three fixes were necessary for compilation and preserve the plan's semantics exactly. No scope creep — no logic changed, no extra packages, only the minimal visibility/wiring adjustments the extraction itself forced.

## Issues Encountered

- The plan's `artifacts` exports list (`NewService`, `ResolveManagerStage`, `ResolveUnitManager`) matched the implementation exactly; the must-have truth "BE-014 semantics preserved verbatim" holds — the two-package suite and the full suite confirm identical behavior through the delegation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Plan 05 (proposal approval) can consume `routing.Service` directly** — it is constructed in cmd/server wiring and exported; `ResolveUnitManager` is a public primitive of the package per the plan's own artifacts contract.
- time_entry behavior is provably unchanged: the routing unit tests pin the semantics the time_entry suite previously exercised through Submit/Approve, and the existing time_entry routing tests (R-1/R-2/R-3, upward walk, D-11 variants) stay green untouched.
- Full repo suite green — no drift.

## Self-Check: PASSED

- Created files verified on disk: internal/core/services/routing/routing.go, internal/core/services/routing/routing_test.go, 11-03-SUMMARY.md
- Commits verified in git log: d6825e0 (refactor), 0e916a1 (test), 8e261d7 (fix)
- Verification commands: `go build ./...` OK; `go vet ./internal/core/services/routing/ ./internal/core/services/time_entry/` OK; `go test ./internal/core/services/routing/ ./internal/core/services/time_entry/ -count=1` green; `go test ./... -count=1` green in all 21 packages

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
