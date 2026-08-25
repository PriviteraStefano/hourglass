---
phase: 14-availability-backend-absences-capacity
plan: 07
subsystem: backend
tags: [go, postgres, read-models, windows-read, capacity, workload-cte, privacy-filtering, validity-split, schedule-expansion, D-14-17..24]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity (plan 06)
    provides: ResolveSchedule fallback chain (override → type → org default → 8×5) with exported level constants + ContractTypeID on the resolution
  - phase: 14-availability-backend-absences-capacity (plan 05)
    provides: the unitManagerAuthorized authority resolution (routing ResolveUnitManager walk) the D-14-24 privacy carve-out reuses
  - phase: 14-availability-backend-absences-capacity (plan 02)
    provides: the pinned ports.AvailabilityRepository (Capacity/Windows/Attachment/AttachmentsByWindowIDs/ActivityWorkloadEmployees signatures), the domain types (CapacityRow, WindowsFilter, EmployeeSchedule, ScheduleResolution)
  - phase: 13-direction-backend-the-plan-plane
    provides: the Coverage read-model shape (generate_series day expansion, partial/full absence joins, normalizeDay, roundCents), membershipValid (D-14-22 mirror), scope-resolution walk (unit descendants)
provides:
  - The org-wide Windows read (D-14-24): deterministic ordering (user_id, starts_on, created_at + id tiebreaker), user/kind/status/period filters, opt-in limit/offset pagination, normalizeDay on scanned DATEs
  - The Capacity read-model (D-14-20/21/23): unnest × generate_series per-day expansion over the SERVICE-expanded day-keyed schedule basis, confirmed-only partial/full absence subtraction with GREATEST floor at 0, the declared advisory column (never subtracted), one WITH RECURSIVE subtree CTE for the workload (D-14-19 literal status pin)
  - The privileged certificate reads: AttachmentsByWindowIDs (metadata-only — BYTEA never crosses the list path), Attachment (single-doc bytes), ActivityWorkloadEmployees (org-guarded subtree universe)
  - Service read-model assembly: ListWindows with the D-14-24 server-side medical carve-out (hr + resolved unit manager see certificate_ref + documents; everyone else sees the kind label only), Capacity (scope→universe, D-14-22 validity split, D-14-17/18 schedule expansion with the month-dynamic derivation, per-employee aggregation with the resolution level documented, declared advisory list), Attachment (privileged single-doc gate)
affects: [14-08 HTTP surface (GET /availability/windows + /availability/capacity + certificate GET consume these shapes), Phase 16 UI (capacity grid), Phase 19 history filters]

actuals:
  tokens: 20955    # chars/4 over the realized diff (83819 chars, 14 files, 5 commits) — estimate was 22000
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "WITH RECURSIVE multi-CTE ordering: the recursive CTE must precede the CTEs that reference it (42P01 on the first draft) — subtree first, then workload/declared/sched"
    - "jsonb_to_recordset for the service-expanded schedule basis: the repo consumes the day-keyed map (to_char(d, 'YYYY-MM-DD') lookup), never re-derives the schedule"
    - "The metadata-only batch read: certificate list paths select WITHOUT storage — BYTEA crosses only the single-document read"
    - "Server-side privacy filtering with a response wrapper: WindowWithDocuments embeds the domain Window (JSON-flattened) + Documents — the pinned domain type never changes"
    - "Month-dynamic nil-matrix as the derivation signal: ResolveSchedule returns DayHours = nil for month-without-matrix types (never the 8×5 substitute); the capacity expansion derives hours_per_period ÷ 5"

key-files:
  created:
    - internal/adapters/secondary/postgres/availability_read_models.go
    - internal/core/services/availability/read_models.go
    - internal/core/services/availability/read_models_test.go
  modified:
    - internal/adapters/secondary/postgres/availability_repository.go
    - internal/adapters/secondary/postgres/availability_repository_test.go
    - internal/core/domain/availability/availability.go
    - internal/core/services/availability/availability.go
    - internal/core/services/availability/availability_test.go
    - internal/core/services/availability/contract_types.go
    - internal/core/services/testdata/mocks.go
    - cmd/server/main.go
    - cmd/server/main_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - internal/core/services/organization/organization_integration_test.go

key-decisions:
  - "The Service constructor gained the orgMgmtRepo dependency (ports.OrganizationManagementRepository): the org capacity scope needs ListMembers, which the pinned orgRepo (GetMembership only) cannot serve — all 4 wiring sites updated (Rule 3)"
  - "The workload subtree CTE anchors at the org's root activities (org_id + parent_id IS NULL, recursive walk): the pinned Capacity signature carries no activity parameter, so the workload column is the employee's org-subtree Σ for every scope; the activity-scope universe comes from ActivityWorkloadEmployees (D-14-19/20)"
  - "WindowsFilter gained Kind/Limit/Offset (additive pointers): the plan's must-have truth mandates kind + limit/offset filters the pinned 14-02 type could not express; nil Limit = the full org-wide read (the 14-05 lifecycle authority resolution reads unbounded)"
  - "AttachmentsByWindowIDs is metadata-only (no storage column): the list path never loads BYTEA; Attachment() is the only bytes-bearing read"
  - "Month-without-matrix types resolve DayHours = nil through the membership path (never the 8×5 substitute): the nil matrix is the D-14-17 derivation signal; a week type with a nil matrix surfaces ErrInvalidValue"
  - "The declared advisory list is built service-side from the Windows read (status=declared, period-bounded, universe-filtered) — per-window kind/dates/hours; the repo's per-employee Declared column is the SQL-level proof the repo battery asserts"

patterns-established:
  - "Read-model split (direction Coverage precedent): the repo owns the set-based math, the service owns scope resolution + validity + schedule expansion + privacy filtering"
  - "The D-14-22 validity mirror: capacityMembershipValid copies direction.membershipValid verbatim with an attribution comment (D-G parity — the semantics never fork)"
  - "Day-keyed schedule handoff: EmployeeSchedule.DayHours carries YYYY-MM-DD keys for the repo's jsonb lookup — the expansion (weekly matrix repetition, month derivation) happens exactly once, service-side"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Repo Windows read (D-14-24): org-wide, deterministic ordering (user_id, starts_on, created_at + id tiebreaker), user/kind/status/period filters, opt-in limit/offset pagination, normalizeDay on scanned DATEs"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Windows"
        status: pass
    human_judgment: false
  - id: D2
    description: "Repo Capacity math (D-14-20/21/23): per-day expansion over the service-expanded schedule basis, confirmed-only partial/full subtraction, GREATEST floor at 0, declared advisory column never subtracted, workload subtree CTE with the D-14-19 literal status pin (depth ≥ 2), no-absence employees keep full schedule hours"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_Capacity"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_ReadModelReads"
        status: pass
    human_judgment: false
  - id: D3
    description: "Service ListWindows with the D-14-24 server-side medical privacy carve-out (T-14g-05): hr and the owner's resolved unit manager see certificate_ref + document metadata; other members see the medical kind label but never the ref or documents"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/core/services/availability/read_models_test.go#TestReadModels_ListWindows_PrivacyMatrix"
        status: pass
    human_judgment: false
  - id: D4
    description: "Service Capacity assembly (D-14-17..23): scope → employee universe (wg/unit+descendants/org/activity; unknown → ErrInvalidRequest), D-14-22 validity split (mock call-count, boundaries inclusive), schedule expansion (week matrix / month matrix / month-derived / override merge / 8×5 fallback), per-employee aggregation with the resolution level documented, declared advisory never subtracted"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: "internal/core/services/availability/read_models_test.go#TestReadModels_Capacity_ScopeUniverses"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/read_models_test.go#TestReadModels_Capacity_ValiditySplit"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/read_models_test.go#TestReadModels_Capacity_ScheduleExpansion"
        status: pass
      - kind: unit
        ref: "internal/core/services/availability/read_models_test.go#TestReadModels_Capacity_DeclaredAdvisory"
        status: pass
    human_judgment: false
  - id: D5
    description: "Service Attachment privileged single-doc read (hr or resolved unit manager of the owner, ErrForbidden otherwise); repo Attachment/AttachmentsByWindowIDs bytes vs metadata split"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/availability_repository_test.go#TestAvailabilityRepository_ReadModelReads"
        status: pass
    human_judgment: true
    rationale: "The service-side authority gate of Attachment() is exercised by the 14-08 route battery (routes absent in this plan); only the repo byte/metadata reads are proven here"

# Metrics
duration: 25min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 07: Windows Read + Capacity Read-Model (AVAIL-01/02) Summary

**The read-model plane of the availability domain is fully live: the org-wide windows read with the D-14-24 server-side medical privacy carve-out (certificate_ref + documents only for hr and the owner's resolved unit manager), and the capacity read-model — per-employee per-day hours from the resolved schedule basis, minus confirmed-only absences (partial subtracts, full zeroes, floored at 0), with the declared advisory that is never subtracted, the workload Σ from submitted+approved entries on the activity subtree, per scope (activity/wg/unit/org) — plus the privileged certificate reads and the activity-scope employee universe.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-11T15:13Z (first task commit)
- **Completed:** 2026-08-11T15:29Z
- **Tasks:** 3 (Task 1 + Task 3 full RED→GREEN TDD cycles; Task 2 compile-verified per plan)
- **Files modified:** 14 (3 created, 11 modified)

## Accomplishments

- **Repo Windows read (Task 1):** org-wide read over `availability_windows` with optional user/kind/status/date-range predicates, deterministic ordering (`user_id, starts_on, created_at` + the `id` tiebreaker that makes equal keys stable — the must-have ordering edge), opt-in limit/offset pagination (nil Limit = the full read the 14-05 authority resolution depends on), and `normalizeDay` on scanned DATE columns (the Phase 13 lesson). Battery proves ordering, every filter, pagination, and the UTC-midnight normalization.
- **Repo Capacity math (Task 1):** the direction Coverage shape re-based: `unnest(employee_ids) × generate_series(days)` day expansion with the per-day schedule basis handed in as the service-expanded day-keyed JSONB map (`jsonb_to_recordset` + `to_char(d, 'YYYY-MM-DD')` lookup — the repo never re-derives the schedule); confirmed-only partial/full absence joins (partial subtracts its hours, full zeroes the day, `GREATEST` floors at 0 — D-14-23); the declared advisory column (period-bounded Σ of declared hours, NEVER subtracted — D-14-21); ONE `WITH RECURSIVE` subtree CTE anchored at the org's root activities for the workload (submitted+approved only, the D-14-19 literal pin, depth ≥ 2 recursion proven by a seeded nested activity — Pitfall 3, no per-row recursion). Employees with no windows/workload still appear with their full schedule hours. Battery asserts all four behavior tests.
- **Privileged certificate reads (Task 1):** `AttachmentsByWindowIDs` is a metadata-only batch read (content_type/size/timestamps — BYTEA never crosses the list path), `Attachment` returns the single document WITH the bytes (latest wins, nil when the window has none), `ActivityWorkloadEmployees` resolves the activity-scope universe (org-guarded subtree, submitted+approved distinct users).
- **Service ListWindows (Task 2):** the D-14-24 server-side medical carve-out — hr (batch attachment read for the whole list) or the owner's resolved unit manager (the 14-05 authority resolution, D-G parity) keep `certificate_ref` + document metadata; every other org member sees kind/dates/status including the medical KIND label but the ref is dropped server-side (T-14g-05 — never client-side). The `WindowWithDocuments` wrapper embeds the pinned domain Window (JSON-flattened) with an additive `documents` field.
- **Service Capacity (Task 2):** scope → employee universe (activity → `ActivityWorkloadEmployees`; wg → WG members; unit → members + descendants via `ListMembersByUnitIDs`; org → org members with a user — unknown scope vocabulary → `ErrInvalidRequest`, T-14g-24), the D-14-22 validity split (the `membershipValid` mirror, boundaries inclusive, nil membership excluded — the repo call-count battery proves excluded employees never reach the repo), the per-employee `ResolveSchedule` → per-day expansion (week matrix repeats per week; month with a matrix applies it; month WITHOUT a matrix derives `hours_per_period ÷ 5` working days — D-14-17; override merge wins per weekday; 8×5 fallback), the repo math, and the per-employee aggregation: `schedule_source` level (D-14-18 documented in the response), `period_capacity_hours`, `confirmed_absence_hours` (Σ schedule base − capacity), `workload_hours`, `available_hours` + totals + the declared-windows advisory list (per-window kind/dates/hours, universe-filtered, never subtracted).
- **Service Attachment (Task 2):** the privileged single-document gate for the 14-08 GET route — hr always, otherwise the resolved unit manager of the window's owner, `ErrForbidden` otherwise.
- **Wiring:** the availability Service gained the `orgMgmtRepo` dependency (the org scope needs `OrganizationManagementRepository.ListMembers` — the pinned `orgRepo` serves GetMembership only); all four production/test wiring sites updated (main.go, main_test.go, handler fixture, org integration test) + the unit fixture. `MockUnitRepo.ListMembersByUnitIDs` now derives from `UnitMembers` (was hard-coded nil — the unit scope could never resolve in tests).

## Task Commits

Each task committed atomically (TDD: test commit then feat commit):

1. **Task 1 RED: repo windows + capacity batteries** - `31d35e2` (test)
2. **Task 1 GREEN: repo windows + capacity read-models** - `b99a7b8` (feat)
3. **Task 2: service read-model assembly** - `682fec1` (feat)
4. **Task 3 RED: service read-model batteries** - `2bb269b` (test)
5. **Task 3 GREEN: month-dynamic nil-matrix resolution** - `c8ce80e` (feat)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/secondary/postgres/availability_read_models.go` - Windows (filters + ordering + normalizeDay + pagination), Capacity (generate_series expansion + confirmed-only joins + GREATEST floor + workload subtree CTE + declared advisory), AttachmentsByWindowIDs (metadata-only), Attachment (bytes), ActivityWorkloadEmployees (created)
- `internal/core/services/availability/read_models.go` - ListWindows (D-14-24 filtering), Capacity (scope→universe + validity split + schedule expansion + aggregation), Attachment (privileged gate), response types (created)
- `internal/core/services/availability/read_models_test.go` - privacy matrix, scope universes, validity split, schedule expansion, declared advisory batteries (created)
- `internal/adapters/secondary/postgres/availability_repository.go` - read-model stubs removed, notImplemented retired (modified)
- `internal/adapters/secondary/postgres/availability_repository_test.go` - Windows/Capacity/ReadModelReads batteries + fptr/sptr/dayKey helpers (modified)
- `internal/core/domain/availability/availability.go` - WindowsFilter gained Kind/Limit/Offset (additive) (modified)
- `internal/core/services/availability/availability.go` - Service struct + constructor gained orgMgmtRepo (modified)
- `internal/core/services/availability/contract_types.go` - ResolveSchedule month-dynamic nil-matrix branch (modified)
- `internal/core/services/testdata/mocks.go` - MockUnitRepo.ListMembersByUnitIDs derives from UnitMembers (modified)
- `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/adapters/primary/http/handler_test_helper.go`, `internal/core/services/organization/organization_integration_test.go` - constructor call sites (modified)
- `internal/core/services/availability/availability_test.go` - fixture gains MockOrgMgmtRepo (modified)

## Decisions Made

- **The Service constructor gained `orgMgmtRepo`** — the org capacity scope's member list lives on `OrganizationManagementRepository.ListMembers`; the pinned `orgRepo` (GetMembership) cannot serve it. Four wiring sites updated; the orgsvc still takes the availability SERVICE (no cycle — the availability service takes the org-mgmt REPO).
- **Workload CTE anchored at the org's root activities** — the pinned `Capacity` signature carries no activity parameter, so the workload column is the employee's Σ over the org subtree for every scope (the "activity subtree" of the org). The activity-scope precision lives in the employee universe (`ActivityWorkloadEmployees`); the recursion depth ≥ 2 repo battery proves the walk. Documented edge: adopted/shared activities with cross-org parents are outside the org-root walk.
- **`WindowsFilter` gained Kind/Limit/Offset (additive)** — the plan's must-have truth mandates kind + limit/offset filters; the 14-02-pinned type had neither. Nil Limit = the full org-wide read (the 14-05 lifecycle resolution reads unbounded — a default limit would silently break authority resolution past row 100).
- **Month-without-matrix types resolve `DayHours = nil`, never the 8×5 substitute** — the nil matrix is the D-14-17 derivation signal; the Task 3 battery caught the substitute (8/day instead of 100÷5=20/day). A week type with a nil matrix (impossible at write — DayHoursValid) surfaces `ErrInvalidValue`.
- **The declared advisory list is built service-side from the Windows read** (status=declared, period-bounded, universe-filtered) — the response needs per-window kind/dates/hours, which the repo's per-employee Σ column cannot express; the repo column remains the SQL-level proof (asserted by the repo battery).
- **Batch attachment read is metadata-only** — the list path never loads BYTEA; `Attachment()` is the single bytes-bearing read (the 14-08 GET route).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The org capacity scope needs a repo the Service does not hold**
- **Found during:** Task 2 (read_models.go design)
- **Issue:** The plan's scope resolution says "org → orgRepo.ListMembers", but the pinned Service's `orgRepo` is `ports.OrganizationRepository` (GetMembership only) — the org member list lives on `ports.OrganizationManagementRepository`. Without it the org scope could not resolve.
- **Fix:** Added the `orgMgmtRepo` field + constructor parameter; updated all four wiring sites (main.go, main_test.go, handler fixture, org integration test) + the unit fixture. The plan's own read_first listed organization_management_repository.go, confirming the intent.
- **Files modified:** availability.go, read_models.go, main.go, main_test.go, handler_test_helper.go, organization_integration_test.go, availability_test.go
- **Verification:** `go build ./...` + full suite green.
- **Committed in:** 682fec1

**2. [Rule 1 - Bug] MockUnitRepo.ListMembersByUnitIDs hard-coded nil**
- **Found during:** Task 2 (mock analysis for Task 3)
- **Issue:** The unit scope resolves descendants via `ListMembersByUnitIDs`; the mock returned `nil, nil` unconditionally — the unit-scope battery could never pass.
- **Fix:** The mock now derives members by scanning `UnitMembers` across the requested unit ids (the ListMembershipsForUser derivation precedent). Additive — no existing test depended on the nil return.
- **Files modified:** internal/core/services/testdata/mocks.go
- **Verification:** the unit-scope universe battery passes.
- **Committed in:** 682fec1

**3. [Rule 1 - Bug] ResolveSchedule substituted the 8×5 fallback for a month-dynamic type's nil matrix**
- **Found during:** Task 3 RED (month-without-matrix battery: expected 20, got 8)
- **Issue:** The membership-type branch of ResolveSchedule (14-06) initializes `base := fallbackDayHours` and only overrides it when `ct.DayHours != nil` — a month-cadence type without a matrix resolved to the 8×5 matrix with level "type", making the D-14-17 derivation unreachable through the membership path. The 14-06 tests never pinned a nil-matrix type (only week cadences).
- **Fix:** When the resolved type is month cadence with a nil matrix, the resolution now carries `DayHours = nil` — the derivation signal the capacity expansion consumes (`hours_per_period ÷ 5`); a week type with a nil matrix (unreachable at write — DayHoursValid) surfaces `ErrInvalidValue`. The 14-06-pinned paths (override-without-type over the 8×5 base, week type matrix, org-default link) are unchanged.
- **Files modified:** internal/core/services/availability/contract_types.go
- **Verification:** month-dynamic battery green (20/day Mon–Fri, 0 weekends); the 14-06 ResolveSchedule suite still green.
- **Committed in:** c8ce80e

**4. [Rule 1 - Bug] WITH RECURSIVE multi-CTE ordering (42P01)**
- **Found during:** Task 1 GREEN (first Capacity query run)
- **Issue:** `WITH sched, subtree, workload, declared` failed with "relation subtree does not exist" — a CTE referencing a later-declared recursive CTE is unresolvable.
- **Fix:** Reordered to `WITH RECURSIVE subtree, workload, declared, sched` (recursive CTE first, references only look backward).
- **Files modified:** availability_read_models.go
- **Verification:** Capacity battery green.
- **Committed in:** b99a7b8

**5. [Rule 1 - Bug] Task 1 Windows battery asserted one shared date for every row**
- **Found during:** Task 1 GREEN (first run)
- **Issue:** The normalizeDay assertion compared every row against `dayMid(2026-8-10)` — the seeded windows span different dates.
- **Fix:** Per-row expected ranges map (id → {starts, ends}).
- **Files modified:** availability_repository_test.go
- **Verification:** Windows battery green.
- **Committed in:** 2bb269b (with the Task 3 RED commit)

**6. [Rule 1 - Bug] Fallback expansion battery seeded no membership — the validity split excluded the employee**
- **Found during:** Task 3 RED (panic: index out of range — the captured schedule list was empty)
- **Issue:** With no membership row, `GetMembership` returns nil → D-14-22 excludes the employee before schedule expansion — the fallback case needs a bare membership to be an org employee.
- **Fix:** `seedMembership(f, orgID, empID, nil, nil)` in the fallback subtest.
- **Files modified:** read_models_test.go
- **Verification:** fallback battery green.
- **Committed in:** 2bb269b

---

**Total deviations:** 6 auto-fixed (5 Rule 1, 1 Rule 3)
**Impact on plan:** All fixes were required for the plan's own MUST-HAVE truths to be expressible (the org scope could not resolve, the month derivation was unreachable, the unit scope was untestable). No scope creep — the read-model shapes follow the plan's skeleton verbatim; one additive domain extension (WindowsFilter fields) mandated by the plan's own truth text.

## Issues Encountered

- **RED placement nuance for Task 3:** the plan defers the RED→GREEN loop to Task 3 by design ("this task's output is the code those batteries target") — the batteries' failures surface the Task-2 implementation's genuine gaps (month derivation, fallback validity) rather than absent methods. The RED commit is valid: both failing batteries ran red against the committed pre-fix tree.
- **Testcontainers cost:** each postgres battery run spins the container; the full postgres suite takes ~42s. No flakiness observed.
- **`WorkingGroupMember.ID` is a uuid.UUID** (not a string like `UnitMember.ID`) — three compile-fix rounds in the new battery file before the first run.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-08 (HTTP surface):** the service read-model methods are ready with the plan-pinned signatures — `ListWindows(orgID, actorID, role, filter)`, `Capacity(orgID, role, scope, scopeID, periodStart, periodEnd)`, `Attachment(orgID, actorID, role, windowID)`; the response types (WindowWithDocuments, CapacityResponse, DeclaredWindowAdvisory) are the handler envelope; parsePeriod + the writeError sentinel map (ErrInvalidRequest → 400, ErrForbidden → 403) are the 14-08 work; the fixture mirrors main.go with the orgMgmtRepo dependency.
- **Phase 16 UI (capacity grid):** the capacity response carries per-employee rows with the schedule_source level, totals, and the declared advisory — the AVAIL-04 grid shape.
- **Phase 19 history filters:** unchanged audit vocabulary; this plan added no audit writes.
- Full suite green (`make test` exit 0, 26 packages).

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*

## Self-Check: PASSED

- All 3 planned new files exist on disk (availability_read_models.go, service read_models.go/.test.go) + the SUMMARY.
- All 6 plan commits verified in git history (31d35e2, b99a7b8, 682fec1, 2bb269b, c8ce80e, 5770477).
- Plan-level verification re-run: `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository` ok; `go test ./internal/core/services/availability/ -run TestReadModels` ok; `go build ./...` ok; `make test` full-suite exit 0 (26 packages).
- TDD gate compliance: Task 1 RED (31d35e2) → GREEN (b99a7b8); Task 3 RED (2bb269b) → GREEN (c8ce80e) — each RED immediately preceding its GREEN. Task 2 is plan-sanctioned non-TDD (compile-only verify).
