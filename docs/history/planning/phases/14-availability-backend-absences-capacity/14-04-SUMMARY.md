---
phase: 14-availability-backend-absences-capacity
plan: 04
subsystem: backend
tags: [go, postgres, direction, availability, warnings, confirmed-only]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane
    provides: the direction read path (AbsenceWindows + Coverage absence subqueries), seedAvailabilityWindow helper, service warning overlay computeWarnings
  - phase: 14-availability-backend-absences-capacity (plan 01)
    provides: migration 023 status vocabulary the confirmed-only predicate reads
  - phase: 14-availability-backend-absences-capacity (plan 02)
    provides: domain/port contracts the direction packages compile against
provides:
  - D-13-29/D-14-21 closure: AbsenceWindows + Coverage (partial_abs/full_abs) read ONLY status = 'confirmed' windows — declared absences never reach the scheduler warning overlay
  - Phase 13 test seeds flipped declared → confirmed; declared-window no-warning behavioral proofs at the repo boundary and the HTTP boundary
  - Doc pins (port Coverage/AbsenceWindows, domain AbsenceWindow, service computeWarnings) making confirmed-only the reader's contract
affects: [14-05 confirm lifecycle (declared windows become advisory-only), 14-07 capacity reads (confirmed-only fact set), Phase 19 history filters]

actuals:
  tokens: 4443    # chars/4 over realized diff (17774 chars, 6 files, 3 commits)
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Confirmed-only predicate + behavioral proof pair: the repo test asserts the excluded status produces no row AND the included status still produces one (D-14-21 proof style)"
    - "Doc pins travel with behavior changes: port + domain + service doc comments updated in the same commit as the SQL flip (T-14g-11)"

key-files:
  created: []
  modified:
    - internal/adapters/secondary/postgres/direction_repository.go
    - internal/adapters/secondary/postgres/direction_repository_test.go
    - internal/core/ports/direction_repository.go
    - internal/core/domain/direction/direction.go
    - internal/core/services/direction/direction.go
    - internal/adapters/primary/http/direction_handler_test.go

key-decisions:
  - "The D-14-21 behavioral proof lives at the repo boundary, not the service mock: MockDirectionRepo.AbsenceWindows returns windows verbatim and the domain AbsenceWindow carries no status — a service-level declared/confirmed distinction is mechanically inexpressible, so the RED test asserts the SQL predicate directly"
  - "Handler-level no-warning proof uses the coverage self-view, not the plan read: ListPlan derives its employee set from plan rows, so an employee without rows gets no warnings regardless of status — the coverage read resolves employees by scope and exercises the same warning overlay"

patterns-established:
  - "Behavioral closure proofs seed BOTH statuses and assert the excluded one is absent while the included one still fires"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "AbsenceWindows returns confirmed-only windows — the in-range DECLARED holiday is excluded while the CONFIRMED partial permit returns (D-14-21/D-13-29 closure at the read path feeding the warning overlay)"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_AbsenceWindows"
        status: pass
    human_judgment: false
  - id: D2
    description: "Coverage capacity subtracts confirmed-only absence hours — declared full/partial windows subtract nothing; confirmed full zeroes the day and confirmed partial reduces by its hours"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Coverage_DeclaredExcluded"
        status: pass
    human_judgment: false
  - id: D3
    description: "HTTP boundary proof — a declared window on the coverage self-view yields zero warnings; confirming the same window yields exactly one 'Away 14 Aug' warning"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/declared_absence_produces_no_warning;_confirmed_does_(D-14-21)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Port + domain + service docs pin confirmed-only (Coverage doc, AbsenceWindows doc, AbsenceWindow doc, computeWarnings doc) — no reader can reintroduce the both-statuses reading (T-14g-11)"
    verification:
      - kind: other
        ref: "go build ./... + doc comment rewrites verified at commit b81b1c8"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 04: Confirmed-Only Direction Read Closure (D-14-21) Summary

**The D-13-29 deferred item is closed: AbsenceWindows and both Coverage absence subqueries now read ONLY `status = 'confirmed'` windows, the Phase 13 test seeds that relied on declared-window warnings flipped to confirmed, and declared-window no-warning proofs land at both the repo boundary (AbsenceWindows + Coverage) and the HTTP boundary (coverage self-view) — the scheduler warning overlay can no longer warn on provisional absence requests that may be rejected.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-08-11T13:01:08Z
- **Completed:** 2026-08-11T13:10:26Z
- **Tasks:** 2 (Task 1 full TDD cycle: RED + GREEN; Task 2 seed flips + handler proof)
- **Files modified:** 6

## Accomplishments

- **Confirmed-only predicates (Task 1 GREEN):** all three query sites flipped — Coverage `partial_abs` (line 767) and `full_abs` (line 775): `status IN ('declared','confirmed')` → `status = 'confirmed'`; AbsenceWindows (line 816): same flip. Declared absences are provisional requests, not facts — the warning overlay ("away 10–21 Aug") reflects only confirmed absence per D-14-21.
- **D-14-21 behavioral proof (Task 1 RED→GREEN):** `TestDirectionRepository_AbsenceWindows` rewritten to assert the in-range DECLARED holiday is excluded while the CONFIRMED partial permit returns; new `TestDirectionRepository_Coverage_DeclaredExcluded` proves declared full/partial windows subtract no capacity while confirmed windows still subtract. Both RED against the shipped both-statuses code (verified: `expected 8, actual 0` / `expected 1, actual 2`), green after the flip.
- **Phase 13 seed flips (Task 2):** `TestDirectionRepository_Coverage_CapacityMath` d2..d5 seeds (multi-day full, spanning partial, oversized partial) flipped declared → confirmed with comments updated — the coverage read-model consumes confirmed-only now.
- **HTTP boundary proof (Task 2):** new `TestDirectionHandler` subtest — a declared window on the coverage self-view yields an empty warnings array; after `UPDATE ... SET status='confirmed'` the same read yields exactly one `{"type":"away","message":"Away 14 Aug"}` warning. The closure is observable at the API contract.
- **Doc pins (Task 1 GREEN):** port `Coverage` doc (capacity subtracts confirmed-only absence hours), port `AbsenceWindows` doc, domain `AbsenceWindow` doc, repo method doc, and the service `computeWarnings` doc all rewritten to confirmed-only — T-14g-11 doc drift mitigated in the same commit as the behavior change.

## Task Commits

Each task was committed atomically (Task 1 TDD: test commit then feat commit):

1. **Task 1 RED: confirmed-only absence tests** - `6aaab25` (test)
2. **Task 1 GREEN: predicate flips + docs** - `b81b1c8` (feat)
3. **Task 2: seed flips + handler no-warning proof** - `d3e196e` (test)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/secondary/postgres/direction_repository.go` - three status predicates flipped to `'confirmed'`; AbsenceWindows method doc updated (modified)
- `internal/adapters/secondary/postgres/direction_repository_test.go` - AbsenceWindows test rewritten (declared excluded), Coverage DeclaredExcluded test added, CapacityMath seeds flipped (modified)
- `internal/core/ports/direction_repository.go` - Coverage + AbsenceWindows doc comments rewritten to confirmed-only (modified)
- `internal/core/domain/direction/direction.go` - AbsenceWindow doc comment updated (modified)
- `internal/core/services/direction/direction.go` - computeWarnings doc updated (modified)
- `internal/adapters/primary/http/direction_handler_test.go` - declared-no-warning vs confirmed-warns HTTP subtest added (modified)

## Decisions Made

- **Behavioral proof at the repo boundary, not the service mock** — the plan asked for the RED subtest in `services/direction/direction_test.go`, but `MockDirectionRepo.AbsenceWindows` returns windows verbatim and the domain `AbsenceWindow` struct carries no status field: a service-level test cannot express declared-vs-confirmed and could never be RED against the shipped predicate. The proof lives where the SQL predicate lives (postgres repo tests); the handler-level proof (Task 2) satisfies the "service and/or handler level" acceptance criterion.
- **Handler proof on the coverage self-view, not the plan read** — `ListPlan` derives its employee set from plan rows, so a fresh employee without direction rows receives no warnings regardless of window status (vacuous proof). The coverage read resolves employees by scope and exercises the identical warning overlay.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RED subtest placement moved from the service mock to the repo boundary**
- **Found during:** Task 1 (RED authoring)
- **Issue:** The plan's literal instruction (add the declared/confirmed subtests to `services/direction/direction_test.go`) is mechanically infeasible: the testdata mock returns `SetAbsenceWindows` windows verbatim and the domain `AbsenceWindow` struct has no status field — the service has no declared/confirmed concept, so a service-level test could not fail pre-closure (not a real RED) and would test nothing.
- **Fix:** The behavioral proof was written at the repo boundary where the SQL predicate lives: `TestDirectionRepository_AbsenceWindows` rewritten (declared excluded, confirmed returned) + new `TestDirectionRepository_Coverage_DeclaredExcluded` (declared subtracts nothing). RED verified against shipped code before the flip. The handler-level proof in Task 2 covers the "service and/or handler level" acceptance criterion.
- **Files modified:** internal/adapters/secondary/postgres/direction_repository_test.go
- **Verification:** RED run failed exactly as intended (declared window still returned/subtracted); GREEN run passed.
- **Committed in:** 6aaab25, b81b1c8

**2. [Rule 3 - Blocking] Handler proof read path: plan read gives fresh employees no warnings at all**
- **Found during:** Task 2 (handler subtest, first run failed: `"[]" should have 1 item(s), but has 0` even after confirming the window)
- **Issue:** `ListPlan` builds its employee set from the plan rows' `directed_to` values — an employee with zero direction rows is never passed to `computeWarnings`, so the confirmed window produced no warning and the proof was vacuous.
- **Fix:** Switched the subtest to the coverage self-view (`scope=employee&scope_id=<emp>`), which resolves employees by scope and runs the identical warning overlay; declared → empty warnings, confirmed → exactly one away warning.
- **Files modified:** internal/adapters/primary/http/direction_handler_test.go
- **Verification:** `go test ./internal/adapters/primary/http/ -run TestDirectionHandler -count=1` green.
- **Committed in:** d3e196e

**3. [Rule 2 - Doc drift] Service computeWarnings doc still read "declared + confirmed, D-13-29"**
- **Found during:** Task 1 (GREEN doc sweep)
- **Issue:** The plan's doc list covered the port and domain files, but `internal/core/services/direction/direction.go` (computeWarnings, line 796) still documented both statuses — exactly the T-14g-11 drift (severity medium → mitigate) the closure must prevent.
- **Fix:** Rewrote the doc comment to confirmed-only per the D-14-21/D-13-29 closure in the GREEN commit.
- **Files modified:** internal/core/services/direction/direction.go
- **Verification:** comment updated; build green.
- **Committed in:** b81b1c8

---

**Total deviations:** 3 auto-fixed (2 Rule 3 blocking mechanics, 1 Rule 2 doc drift)
**Impact on plan:** All three fixes were required to land a real, non-vacuous behavioral proof and to prevent doc drift from reintroducing the both-statuses reading. No scope creep — the closure, flips, and proofs all land exactly as the plan's must-haves specified.

## Issues Encountered

- **Plan `-run` regexes matched no tests:** the plan's verify commands use `-run 'TestWarnings|Absence'` and `-run TestDirection` against the service package, whose tests are named `TestService_Warnings`/`TestService_Create`/etc. — Go's `-run` matched nothing ("no tests to run"). Executed with corrected patterns (`Warnings|Absence`, and the full package for services); all green. No code impact.
- **Interim expected-red state (plan-sanctioned):** between Task 1 GREEN and Task 2, `TestDirectionRepository_Coverage_CapacityMath` was red because its d2..d5 seeds were still 'declared' — the plan explicitly forbade chasing those failures at Task 1 ("expected red until Task 2 flips them"); Task 2 flipped the seeds and the suite went fully green.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-05 (confirm lifecycle):** declared windows are now advisory-only on the direction read path — the confirm mutator's status transition becomes the single source of "facts" the scheduler sees; the D-14-21 closure proof is the behavioral anchor for that contract.
- **14-07 (capacity reads):** the availability capacity read-models consume the same confirmed-only fact set — the direction closure and the availability semantics are aligned per the plan's key link.
- **Phase 19 history filters:** the confirmed-only read path is stable; no both-statuses reading remains in direction packages.
- The full suite is green (`make test` exit 0) — availability plans (14-03 pending) are additive and unaffected.

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*

## Self-Check: PASSED
