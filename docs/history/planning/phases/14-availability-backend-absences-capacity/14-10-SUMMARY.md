---
phase: 14-availability-backend-absences-capacity
plan: 10
subsystem: api
tags: [availability, capacity, workload, postgres, cte, wr-02]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity
    provides: 14-09's test-file additions (file-level dependency only) and the 14-07 Capacity read-model with its unbounded workload CTE (the WR-02 defect)
provides:
  - Period-scoped capacity workload (WR-02 closure): the workload CTE is bounded by $3/$4 exactly like declared/partial_abs/full_abs
  - Repo battery proving out-of-period entries contribute 0 to workload_hours
  - HTTP boundary regression proving the same contract end-to-end
affects: [16-grid (DIR-05 warnings), 20-today, availability capacity consumers, gsd-verify-work phase 14]

actuals:
  tokens: 3700
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Read-model sibling-column parity: every summed term in the capacity query obeys the same $3/$4 period window (the D-14-19 workload contract)"
    - "TDD RED battery seeds both inside- and outside-period rows so a lifetime-sum regression fails loudly"

key-files:
  created: []
  modified:
    - internal/adapters/secondary/postgres/availability_read_models.go
    - internal/adapters/secondary/postgres/availability_repository_test.go
    - internal/adapters/primary/http/availability_handler_test.go

key-decisions:
  - "Workload CTE period predicate mirrors the sibling columns verbatim: te.entry_date >= $3::date AND te.entry_date < $4::date + INTERVAL '1 day' — the same args declared/partial_abs/full_abs consume; CTE order (subtree → workload → declared → sched) unchanged"
  - "Task 2 RED cannot fail on its own: Task 1 GREEN lands the predicate first, so the HTTP regression is committed as the proven end-to-end guard (same RED-placement class as the 14-04 deviation); its failure mode is proven by Task 1's repo-level RED (22.0 vs 10.0)"

patterns-established:
  - "Period contract on derived capacity: available_hours = period_capacity − period_workload is only meaningful when EVERY term is period-bounded (D-14-19/D-14-20)"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Capacity workload period-scoping (WR-02): workload CTE bounded by $3/$4 like its sibling columns; out-of-period entries contribute 0"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Capacity_WorkloadPeriodBound"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Capacity"
        status: pass
      - kind: e2e
        ref: "internal/adapters/primary/http/availability_handler_test.go#GET /availability/capacity subtest (WR-02 boundary regression)"
        status: pass
    human_judgment: false

# Metrics
duration: 11min
completed: 2026-08-12
status: complete
---

# Phase 14: Plan 10 — WR-02 Gap Closure: Period-Scoped Capacity Workload Summary

**The capacity workload CTE is now period-bounded by the same $3/$4 window as its sibling columns — `available_hours = period_capacity − period_workload` is semantically correct, proven by a repo battery that seeds in- and out-of-period entries and an HTTP boundary regression.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-08-12T09:29:49Z
- **Completed:** 2026-08-12T09:40:52Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- **WR-02 closed at the SQL level:** the `workload` CTE in `Capacity` gained `AND te.entry_date >= $3::date AND te.entry_date < $4::date + INTERVAL '1 day'` — identical in shape to the `declared`/`partial_abs`/`full_abs` CTEs — so historical entries no longer drag capacity down forever (D-14-19).
- **Repo battery `TestAvailabilityRepository_Capacity_WorkloadPeriodBound`:** seeds in-period (submitted 6h + approved 4h) and out-of-period (5h before start, 7h after end) entries on the same nested activity; asserts workload 10.0 for userA and 0.0 for userB on EVERY one of the 7 day rows. RED run before the fix: 22.0 (the lifetime sum) vs expected 10.0.
- **HTTP boundary regression:** the `GET /availability/capacity` scope=activity subtest seeds a second submitted entry dated 2026-10-03 (outside the 2026-11-02..11-08 period) on the same activity for the same employee; `workload_hours` stays exactly 4.0 — the period contract holds through the real repo/service/handler stack.
- **Existing batteries untouched and green:** `TestAvailabilityRepository_Capacity` (in-period seeds, 10.0 at keys[0], capacity vectors) and the full `TestAvailabilityHandler` suite (lifecycle matrix, privacy e2e, upload/download gates) pass with unchanged expected values.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): period-bound workload battery** - `cdf6931` (test)
2. **Task 1 (GREEN): workload CTE period predicate + doc comment** - `e736c0a` (feat)
3. **Task 2: HTTP capacity boundary regression** - `091c9fa` (test)

## Files Created/Modified

- `internal/adapters/secondary/postgres/availability_read_models.go` - workload CTE gains the `$3`/`$4` period predicate (135-142 region); Capacity doc comment (116-119 region) states the period binding
- `internal/adapters/secondary/postgres/availability_repository_test.go` - +`TestAvailabilityRepository_Capacity_WorkloadPeriodBound` (in- + out-of-period seeds, full 7-day iteration per-employee assertions)
- `internal/adapters/primary/http/availability_handler_test.go` - capacity scope=activity subtest gains the out-of-period submitted-entry seed (2026-10-03, 6h) on the same activity/user

## Decisions Made

- The workload predicate copies the sibling-column shape verbatim (`entry_date >= $3::date AND entry_date < $4::date + INTERVAL '1 day'`) — same args, same inclusive-start/exclusive-end semantics, no new parameter positions; CTE order (subtree → workload → declared → sched) unchanged.
- Task 2 committed as a test-only commit: its RED cannot fail after Task 1's GREEN lands the predicate (the assertion passes from the first run). The regression's failure mode is already proven by Task 1's repo-level RED (22.0 vs 10.0); the HTTP test is the end-to-end guard that will catch a future predicate regression at the boundary. Same RED-placement class as the 14-04 deviation recorded in STATE.md.

## Deviations from Plan

None - plan executed exactly as written (the Task 2 RED sequencing note above is a plan-inherent artifact of two TDD tasks sharing one fix, documented in Decisions Made, not a deviation).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- WR-02 closed (3 of 3 verification gaps — CR-01 in 14-09, WR-03 in 14-11, WR-02 here): all failed must-have truths from 14-VERIFICATION.md now have fix commits with regression batteries.
- Phase 14 is complete: all 11 plans have summaries. Ready for phase re-verification (`/gsd-verify-work`) and next-step routing (grid consumers in Phase 16 now read a period-correct capacity metric; DIR-05 warnings in Phase 13 direction reads were already confirmed-only via 14-11's WR-03 closure).

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-12*

## Self-Check: PASSED

- Files: availability_read_models.go, availability_repository_test.go, availability_handler_test.go all present
- Commits: cdf6931 (test), e736c0a (feat), 091c9fa (test) all in git log
- Verification: `go build ./...` clean; postgres Capacity battery ok (3.6s); HTTP TestAvailabilityHandler ok (13.7s)

