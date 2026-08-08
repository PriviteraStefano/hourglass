---
phase: 13-direction-backend-the-plan-plane
plan: 06
subsystem: database
tags: [postgres, pgx, direction, read-model, recursive-cte, coverage, derived-states]

requires:
  - phase: 13-01
    provides: migrations 021/022 — direction rows (status vocab, XOR/queued/scheduled CHECKs, idx_direction_activity_created) + org_settings key/value
  - phase: 13-03
    provides: direction domain read-model shapes (PlanRow/CoverageRow/AbsenceWindow/DirectionRefs + derived-state vocabularies) + ports.DirectionRepository pin
  - phase: 13-05
    provides: direction_repository.go mutator half (Create/Activate/Cancel/Claim/Unclaim + scanDirectionRow/insertDirectionAudit/getByIDTx) — extended here with the read side
provides:
  - direction_repository.go read-model half: ListPlan (derived done/lapsed/claim-spectrum on read), Coverage (per employee/day capacity/planned/gap), AbsenceWindows (declared+confirmed), FirstDirectionRefs (origin fallback) + CTE helpers terminalActivitySubtree/hasAnyEntries + claimSpectrum + normalizeDay + the full-interface `var _ ports.DirectionRepository` assertion
  - direction_repository_test.go read-model integration suite: 8 new test functions (period/scope, ordering, derived states, claim spectrum, capacity math, custom daily hours, planned+gap, absence windows, fallback refs)
affects: [13-07 (service consumes ListPlan/Coverage/AbsenceWindows read-models + warning overlay), 13-08 (activity-service origin fallback via FirstDirectionRefs), Phase 19 surfaces]

actuals:
  tokens: 12247    # 48,987 diff chars / 4 over the two declared files (estimate: 34000)
  tasks: 3
  commits: 7        # 6 production (3 test + 3 feat) + 1 docs

tech-stack:
  added: []
  patterns:
    - "Derived-on-read (D-13-09): the Phase 11 terminal-activity recursive CTE re-anchored at activities.id with the NOT EXISTS semantic inversion (terminalActivitySubtree) + the A3 lapsed input (hasAnyEntries — any non-deleted entry, any status)"
    - "Read-model day series: unnest(uuid[]) x generate_series(dates) with LEFT JOINs for planned Σ, partial-absence reduction and full-absence zeroing — one query, COALESCE 8.0 daily-hours default via org_settings scalar subquery (ADR-BE-018 consequence)"
    - "normalizeDay: scanned DATE columns carry the session timezone — read-model days normalized to UTC midnight so day keys/serialization are deterministic"
    - "TDD per task: test(13-06) RED commits (compile-fail on missing methods) then feat(13-06) GREEN commits"

key-files:
  created: []
  modified:
    - internal/adapters/secondary/postgres/direction_repository.go
    - internal/adapters/secondary/postgres/direction_repository_test.go

key-decisions:
  - "normalizeDay for read-model dates: PostgreSQL DATE values scan back in the SESSION timezone (a 2026-08-08 value reads 2026-08-08T02:00:00+02:00 on a +02:00 session) — day-key map lookups and JSON serialization were nondeterministic (proven by a failing test); Coverage.Date / AbsenceWindow.StartsOn/EndsOn are normalized to UTC midnight, the mutator scans stay as-is"
  - "AbsenceWindows maps availability_windows.user_id (migration 012 column — the plan text said 'verify: availability_windows.employee_id') to direction.AbsenceWindow.EmployeeID"
  - "Full-interface assertion var _ ports.DirectionRepository = (*DirectionRepository)(nil) landed with 13-06 as deferred from 13-05 — it compiles only once the last read-model method exists"

patterns-established:
  - "Coverage capacity = COALESCE(planning_daily_hours, 8.0) − absence hours in ONE query: full window (hours NULL) zeroes the day via a separate LEFT JOIN, partial windows sum and subtract, GREATEST(…,0) floors (Pitfall 10); planned Σ joins on (directed_to, planned_date) with status IN draft|active only (T-13-19)"

requirements-completed: [DIR-02, DIR-05, DIR-06]

coverage:
  - id: D1
    description: "ListPlan read-model (D-13-27): draft|active rows for the period — scheduled by planned_date in-range, queued by due_date in-range or NULL (the queue is part of the plan view), employee-scoped or org-wide; ordering planned_date ASC NULLS LAST -> priority ASC NULLS LAST (lower = higher, D-13-06) -> due_date ASC -> created_at tiebreak; superseded/cancelled history rows never appear (D-13-08)"
    requirement: DIR-02
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_ListPlan_PeriodAndScope"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_ListPlan_Ordering"
        status: pass
    human_judgment: false
  - id: D2
    description: "Derived states computed on read, never stored (D-V, D-13-09, ADR-BE-018 §2): done = terminal-activity CTE re-anchored at activities.id with the exact non-terminal predicate and the NOT EXISTS semantic inversion; lapsed = past planned_date (or due_date for queued) AND no non-deleted entries of ANY status on the subtree (A3 — a draft entry kills lapsed); claim spectrum for WG rows (D-13-15): budget NULL -> not_claimed/partially_claimed (never fully, D-13-14), budget set -> not/partially/fully with Σ in cents; cancelled claims release hours; supersede-of-claim keeps the chain consistent (fully_claimed holds through the supersede)"
    requirement: DIR-02
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_DerivedStates_DoneLapsed"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_ListPlan_ClaimSpectrum"
        status: pass
    human_judgment: false
  - id: D3
    description: "Coverage read-model (DIR-06, D-13-24..26): one row per (employee, day) for every day in the period; capacity = planning_daily_hours (org_settings COALESCE 8.0) − absence hours (full window hours NULL zeroes the day, partial subtracts, overlapping windows both subtract, floor at 0 — Pitfall 10); planned = Σ est_hours of draft|active rows for that employee on that exact day (history rows excluded, T-13-19, multiplicity sums); gap = capacity − planned (negative = over-capacity); zero-planned days surfaced (uncovered day, gap == capacity); fully-absent days still return rows (the service owns surfacing, D-13-26)"
    requirement: DIR-06
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Coverage_CapacityMath"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Coverage_CustomDailyHours"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Coverage_PlannedAndGap"
        status: pass
    human_judgment: false
  - id: D4
    description: "AbsenceWindows (D-13-29, DIR-05 repo slice): availability_windows rows with status IN ('declared','confirmed') — BOTH statuses — overlapping [start, end], org-scoped, user_id mapped to EmployeeID, kind + hours + starts_on + ends_on carried for the 13-07 warning formatting"
    requirement: DIR-05
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_AbsenceWindows"
        status: pass
    human_judgment: false
  - id: D5
    description: "FirstDirectionRefs origin fallback (D-13-32..34): earliest created_at non-cancelled direction row for the activity, org-scoped (T-13-21), served by idx_direction_activity_created; assigned_by = directed_by, assigned_to = directed_to — manager-assignment shape only; pgx.ErrNoRows -> (nil, nil) (refs stay empty); read-only by construction (no INSERT/UPDATE path, D-13-34)"
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_FirstDirectionRefs"
        status: pass
    human_judgment: false

duration: 13 min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 6: Direction Repository Read-Models — ListPlan Derived States, Coverage, AbsenceWindows, Origin Fallback Summary

The read side of the direction repository: `ListPlan` with derived states (done/lapsed/claim spectrum) computed on read via the re-anchored terminal-activity CTE, the Coverage read-model (per employee/day capacity − absence hours vs planned Σ with uncovered-day surfacing), AbsenceWindows (declared+confirmed) feeding the 13-07 warning overlay, and FirstDirectionRefs for the 13-08 origin fallback — completing `ports.DirectionRepository` so the full-interface compile assertion (deferred from 13-05) finally lands.

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-08T13:46:37Z
- **Completed:** 2026-08-08T13:59:15Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- **ListPlan with derived states (Task 1)**: draft|active rows for the period (scheduled by planned_date; queued by due_date in-range or NULL — the queue is part of the plan view), employee-scoped or org-wide, ordered `planned_date ASC NULLS LAST → priority ASC NULLS LAST → due_date ASC NULLS LAST → created_at ASC` (D-13-06, stable for equal keys). Derived on read, never stored (D-V): `done` from `terminalActivitySubtree` — the Phase 11 ticket dismissal-guard CTE re-anchored at `activities.id` with the exact non-terminal predicate and the **NOT EXISTS semantic inversion**; `lapsed` from `hasAnyEntries` — past date AND no non-deleted entries of ANY status on the subtree (A3: a draft entry kills lapsed); the D-13-15 claim spectrum for WG rows (`claimSpectrum` in cents: budget NULL → not/partially — never fully; budget set → not → partially → fully). Superseded/cancelled rows are history and never appear (D-13-08).
- **Coverage read-model (Task 2)**: one row per (employee, day) for EVERY day in the period — one query: `unnest(uuid[]) × generate_series(dates)` cross-joined with the daily-hours scalar (`COALESCE(org_settings.planning_daily_hours, 8.0)` — ADR-BE-018 consequence: the default is code-level, never a seed row), LEFT JOIN planned Σ (draft|active, exact day, T-13-19), LEFT JOIN partial absences (sum of hours, both declared+confirmed), LEFT JOIN full absences (hours NULL → day zeroes); `GREATEST(…, 0)` floors capacity (Pitfall 10 — a zeroed day is away, never fake over-capacity); gap = capacity − planned (negative = over-capacity); zero-planned days surfaced (gap == capacity — the D-13-26 uncovered shape; the repo returns all rows, the service owns surfacing).
- **AbsenceWindows (Task 2)**: `status IN ('declared','confirmed')` — BOTH per D-13-29 (Phase 14 tightens) — overlapping [start, end], org-scoped, carrying kind/hours/starts_on/ends_on for the 13-07 warning message formatting. Schema column is `user_id` (migration 012) — mapped to `AbsenceWindow.EmployeeID`.
- **FirstDirectionRefs (Task 3)**: earliest created_at non-cancelled row for the activity, org-scoped, served by `idx_direction_activity_created`; `pgx.ErrNoRows → (nil, nil)` (refs stay empty, D-13-33); manager-assignment pair only; read-only by construction (D-13-34) — the seam the 13-08 activity-service origin fallback consumes.
- **Full-interface assertion**: `var _ ports.DirectionRepository = (*DirectionRepository)(nil)` — deferred from 13-05 (13-05-SUMMARY deviation note (a)) — compiles now that the last read-model method exists. No port signature changed (13-03 contract untouched).
- **TDD gate compliance**: per-task RED→GREEN — three `test(13-06)` commits (each failing on the missing method) preceded their `feat(13-06)` implementations.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1: ListPlan + derived states** - `fb9e986` (test: ListPlan + derived-states integration tests), `2e6bb57` (feat: ListPlan read-model with derived states on read)
2. **Task 2: Coverage read-model + AbsenceWindows** - `13eba10` (test: Coverage + AbsenceWindows integration tests), `ab573b0` (feat: Coverage read-model + AbsenceWindows)
3. **Task 3: FirstDirectionRefs + tests** - `d5872a5` (test: FirstDirectionRefs integration tests), `e34029a` (feat: FirstDirectionRefs + full-interface assertion)

**Plan metadata:** (docs commit, this SUMMARY)

## Files Created/Modified

- `internal/adapters/secondary/postgres/direction_repository.go` — read-model half: `ListPlan` + `terminalActivitySubtree`/`hasAnyEntries`/`claimSpectrum` helpers, `Coverage`, `AbsenceWindows`, `FirstDirectionRefs`, `normalizeDay`, `var _ ports.DirectionRepository` assertion (+ `ports` import)
- `internal/adapters/secondary/postgres/direction_repository_test.go` — 8 new integration tests + `seedAvailabilityWindow`/`seedOrgSettings` helpers + `planRowIDs`/`planRowByID`/`coverageByCell` assertion helpers

## Decisions Made

- **normalizeDay (Rule 1 bug fix, see Deviations)**: read-model DATE columns normalized to UTC midnight — the session-timezone scan made day-key comparisons and serialization nondeterministic.
- **user_id mapping**: `availability_windows.user_id` is the employee column (012); `AbsenceWindows` maps it to `EmployeeID` — the plan's "verify" note resolved.
- **One-query coverage assembly** over a Go-side day loop: the `unnest × generate_series` cross join keeps the per-day math in SQL where DECIMAL arithmetic is exact; only the final cents-rounded render happens in Go.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Read-model days carried the session timezone — map keys and serialization nondeterministic**
- **Found during:** Task 2 (TestDirectionRepository_Coverage_CapacityMath failed: expected 2.5, actual 0)
- **Issue:** PostgreSQL `DATE` columns scan back in the SESSION timezone: a `2026-08-08` value reads `2026-08-08T02:00:00+02:00` (Local) on a +02:00 session. `time.Time` `==` (the map-key comparison) fails between the scanned instant and the UTC-midnight test key even though `Equal()` is true — all Coverage/AbsenceWindows assertions missed their cells and read zero values. This is a repo-side correctness bug (the read-model's day semantics must be timezone-free), not a test bug.
- **Fix:** Added `normalizeDay` (UTC midnight re-derivation) applied to `CoverageRow.Date` and `AbsenceWindow.StartsOn/EndsOn` at scan time; mutator scans (scanDirectionRow) stay as-is. Proven with a temporary probe query (deleted after): the SQL itself produced correct values, isolating the scan-time normalization as the fix.
- **Files modified:** internal/adapters/secondary/postgres/direction_repository.go
- **Verification:** All 8 read-model tests pass with UTC-normalized days; full suite green.
- **Committed in:** ab573b0 (Task 2 feat commit)

**2. [Rule 3 - Blocking issue] Plan's `availability_windows.employee_id` column does not exist**
- **Found during:** Task 2 (read_first: migrations/012_staffing_schema.up.sql)
- **Issue:** The plan's AbsenceWindows SQL named `availability_windows.employee_id` — migration 012 defines the employee column as `user_id` (with an inline "verify" instruction in the plan text, which the schema check resolved).
- **Fix:** `AbsenceWindows` selects `user_id` and maps it to `direction.AbsenceWindow.EmployeeID` (documented in the method comment).
- **Files modified:** internal/adapters/secondary/postgres/direction_repository.go
- **Verification:** TestDirectionRepository_AbsenceWindows asserts the mapped EmployeeID, declared+confirmed, org-scoped.
- **Committed in:** ab573b0 (Task 2 feat commit)

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking). **Impact:** none on the plan's invariants — both keep the plan's intent (deterministic read-model days; the plan itself flagged the column name for verification). No shared contract changed.

## Issues Encountered

- One test-authoring slip during Task 2 RED (asserted window `Kind == "declared"` where `declared` is the window's *status*, kind is `holiday`) — fixed in the same RED commit before GREEN; no plan impact.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **13-07 (direction service)**: consumes `ListPlan`/`Coverage`/`AbsenceWindows` with the pinned read-model shapes; `Coverage` returns all days (the service excludes fully-absent days and validity-outside employees from uncovered surfacing — D-13-26/31), `AbsenceWindows` returns declared+confirmed (service derives the warning types). Warning math inputs (capacity/planned/gap) are cents-rounded.
- **13-08 (origin fallback)**: `FirstDirectionRefs` nil-when-none contract ready for the activity-service seam (A4 predicate: stored refs empty → derive).
- Requirements DIR-02/05/06 stay Pending under the shared-ID gate until 13-07/13-08 finish (they declare the same IDs).
- Port interface unchanged — the full-interface assertion now compiles, so no later plan can drift the signature silently.

## Self-Check: PASSED

- [x] `internal/adapters/secondary/postgres/direction_repository.go` exists and compiles (go build ./...)
- [x] `internal/adapters/secondary/postgres/direction_repository_test.go` exists
- [x] Commit fb9e986 exists (test RED: ListPlan)
- [x] Commit 2e6bb57 exists (feat GREEN: ListPlan)
- [x] Commit 13eba10 exists (test RED: Coverage/AbsenceWindows)
- [x] Commit ab573b0 exists (feat GREEN: Coverage/AbsenceWindows)
- [x] Commit d5872a5 exists (test RED: FirstDirectionRefs)
- [x] Commit e34029a exists (feat GREEN: FirstDirectionRefs + assertion)
- [x] Task 1 verify `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionRepository_ListPlan|TestDirectionRepository_DerivedStates' -count=1` exits 0
- [x] Task 2 verify `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionRepository_Coverage|TestDirectionRepository_Warnings|TestDirectionRepository_AbsenceWindows' -count=1` exits 0
- [x] Task 3 verify `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionRepository_FirstDirectionRefs' -count=1` exits 0
- [x] Plan verification 1 `go test ./internal/adapters/secondary/postgres/ -run 'TestDirection' -count=1` exits 0
- [x] Plan verification 2 `go build ./...` compiles
- [x] Plan verification 3 `make test` full suite green (23 packages ok, 0 FAIL)
- [x] `go vet ./...` clean
- [x] Grep guard: done CTE anchored at activities.id with NOT EXISTS inversion + verbatim non-terminal predicate; hasAnyEntries without status filter (A3); FirstDirectionRefs is a single org-scoped SELECT (no INSERT/UPDATE path); `var _ ports.DirectionRepository` present

## Threat Flags

None — the read-model surface matches the plan's `<threat_model>` exactly: T-13-19 (planned Σ pinned to draft|active + exact day, asserted), T-13-20 (full-absence zeroing, partial reduction, floor at 0, asserted), T-13-21 (FirstDirectionRefs org-scoped + nil-on-no-rows), T-13-22 (derived states from the verbatim non-terminal predicate). No new endpoints, auth paths, file access, or schema changes.

## Known Stubs

None — no placeholder values, no TODO/FIXME, no empty data sources. `var _ ports.DirectionRepository` assertion compiled; every port method implemented.
