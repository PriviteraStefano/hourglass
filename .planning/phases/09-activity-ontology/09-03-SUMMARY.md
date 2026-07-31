---
phase: 09-activity-ontology
plan: 03
subsystem: api
tags: [go, postgres, hexagonal, activities, adr-p-007, adr-be-014]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: 011_activity_ontology migration (activities/activity_kinds/activity_adoptions/activity_managers schema)
provides:
  - `Activity` domain type + `ActivityKind` string catalog type (D-2, not enum), ActivityRepository port, single postgres adapter with GetAncestry / ResolveCommercialContext / ResolveBillability recursive CTEs
  - Entry repos (time_entry + expense) on `activity_id` + `unit_id` only; ListPending manager stage via activity → WG → manager/delegate (R-1)
  - working_groups + financial_cutoff_periods re-anchored to activities; contract/export commercial-chain queries rewritten as activity-tree CTEs
affects: [09-04 service layer, 09-05 http handlers, contract service delete-guards, export service, all project/subproject-dependent Go code]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Recursive CTE upward walk (leaf → root) for commercial chain (D-3) and billability inheritance (D-7) — same pattern as the units subtree CTE"
    - "Derived-never-stored: ResolveCommercialContext/ResolveBillability walk parent_id; entries carry no commercial fields"
    - "Single-repository collapse: one ActivityRepository port replaces project + subproject ports (R-6)"

key-files:
  created:
    - internal/core/domain/activity/activity.go
    - internal/core/ports/activity_repository.go
    - internal/adapters/secondary/postgres/activity_repository.go
    - internal/adapters/secondary/postgres/activity_repository_test.go
  modified:
    - internal/core/domain/time_entry/time_entry.go
    - internal/core/domain/expense/expense.go
    - internal/core/ports/time_entry_repository.go
    - internal/core/ports/expense_repository.go
    - internal/adapters/secondary/postgres/time_entry_repository.go
    - internal/adapters/secondary/postgres/expense_repository.go
    - internal/adapters/secondary/postgres/working_group_repository.go
    - internal/adapters/secondary/postgres/contract_repository.go
    - internal/adapters/secondary/postgres/export_repository.go
  deleted:
    - internal/core/domain/project/
    - internal/core/ports/project_repository.go
    - internal/core/ports/subproject_repository.go
    - internal/adapters/secondary/postgres/project_repository.go
    - internal/adapters/secondary/postgres/subproject_repository.go

key-decisions:
  - "Domain layout follows the repo's existing per-entity subdirectory convention: domain/activity/activity.go (plan's flat 'activity.go' path mapped to it)"
  - "BudgetAmount modeled as *float64, not *decimal — no decimal library exists in the module and the codebase models all money as float64 (Amount, KmRate, DefaultKmRate); DECIMAL(12,2) scans into float64 natively via pgx"
  - "working_group domain keeps its legacy SubprojectID field name (mapped to activities.activity_id) — the plan's files_modified does not list the domain file and services/working_group + http/working_group.go are not in plans 04/05 file lists, so renaming would leave the module uncompilable with no plan to fix it; cosmetic rename deferred"
  - "contract_repository + export_repository commercial-chain queries rewritten against the activity tree (Rule 3) — their old SQL referenced the dropped projects table and the plan verification requires the whole postgres package green"
  - "financial_cutoff_period_repository.go does not exist as its own file — its queries (IsPeriodLocked) live in the entry repos and were updated in place to org + activity + date range (R-5)"

patterns-established:
  - "Repository collapse discipline: delete the old domain/port/adapter files in the same commit as the new single Activity repository"
  - "Compile-time interface assertion `var _ ports.ActivityRepository = (*ActivityRepository)(nil)` on the new adapter"

requirements-completed: [P-007-D1, P-007-D3, P-007-D4, P-007-D5, BE-014-R1, BE-014-R5, BE-014-R6]

# Metrics
duration: 20min
completed: 2026-07-31
---

# Phase 09 Plan 03: Domain + Repository Collapse Summary

**Projects and subprojects collapse into one recursive `Activity` entity end-to-end: `Activity` domain type with `ActivityKind` string catalog (D-2), one `ActivityRepository` port + PG adapter (R-6) whose `GetAncestry`/`ResolveCommercialContext`/`ResolveBillability` walk `parent_id` upward as recursive CTEs (D-3/D-7 derived-never-stored), and both entry repositories rewritten onto `activity_id` + `unit_id` with `ListPending` manager stage resolving through the activity → WG → manager/delegate chain (R-1) — proven by testcontainers integration tests.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-31T16:06:57Z
- **Completed:** 2026-07-31T16:27:43Z
- **Tasks:** 3
- **Files modified:** 23 (18 created/modified + 5 deleted)

## Accomplishments

- `Activity` domain type with the full P-007 field set (ID, OrgID, ParentID, Name, Description, Kind, ContractID, GovernanceModel, CreatedByOrgID, IsShared, Billable *bool, BudgetAmount, IsActive, timestamps), `ActivityKind` as a free-catalog string type (no enum, no IsValid gate), plus CreateActivityRequest/UpdateActivityRequest/ActivityFilter DTOs and CommercialContext
- `activity_repository.go` (domain + port + adapter) replaces the project + subproject pair: CRUD, ListChildren, ListByContract, adoption, managers (activity_managers), subtree-aware delete guards (HasActiveTimeEntries/HasActiveExpenses/HasChildren)
- `GetAncestry` (recursive CTE leaf → root), `ResolveCommercialContext` (nearest contract-bearing ancestor → contract + customer, nil for internal trees), `ResolveBillability` (nearest explicit billable wins; contract-linked default; explicit FALSE distinguishable from NULL) — all tested against a 3-level engagement→phase→task tree
- `time_entries`/`expenses` repos: `activity_id` + `unit_id` only; `IsPeriodLocked` key is org + activity + date range (R-5); `ListPending` manager stage joins working_groups by `activity_id` → manager/delegate for both entry types (R-1 — expenses now match time entries per ADR-P-001 Q1)
- `working_groups` re-anchored to `activity_id` with the legacy unit-tuple toggle gone from all SQL; contract + export repos' commercial-chain queries rewritten as recursive CTEs against the activity tree (their old SQL hit the dropped `projects` table)
- Full postgres integration suite green against testcontainers: 3-level ancestry, grandparent commercial context, internal-nil, billability nearest-wins + contract default + explicit-FALSE override, entry CRUD/filters/IsPeriodLocked/ListPending-via-WG, export commercial chain, contract subtree counts

## Task Commits

Each task was committed atomically:

1. **Task 1: Domain + ports collapse** - `53dfc94` (refactor)
2. **Task 2: Activity repository + WG/cutoff/commercial-chain rewrites** - `e7a6d88` (feat)
3. **Task 3: Entry repositories onto activity FKs** - `c101494` (feat)
4. **Scope fix: revert working_group domain rename** (keeps wg service/handler compiling — plan's file list omits the domain, and services/working_group + http/working_group.go are not in plans 04/05 file lists) - `05cbe4c` (refactor)

**Plan metadata:** `62c7809` (docs: complete plan)

## Self-Check: PASSED

- All created files exist on disk (activity.go ×3, activity_repository_test.go, 09-03-SUMMARY.md)
- All 3 task commits present in git log: `53dfc94`, `e7a6d88`, `c101494`
- `go build ./internal/core/domain/... ./internal/core/ports/... ./internal/adapters/secondary/postgres/...` — PASS
- `go vet` on the same packages — PASS
- `go test ./internal/adapters/secondary/postgres/ -count=1` — PASS (19s)
- Re-verified post-write: all 5 SUMMARY-listed files on disk, all 3 commits in git log — PASSED

## Files Created/Modified

- `internal/core/domain/activity/activity.go` - Activity struct (all P-007 fields), ActivityKind string type, Create/Update/Filter DTOs, ActivityAdoption, ActivityManager, CommercialContext, sentinel errors
- `internal/core/ports/activity_repository.go` - single port (R-6): CRUD, ListChildren, ListByContract, GetAncestry, ResolveCommercialContext, ResolveBillability, adoption, managers, delete guards
- `internal/adapters/secondary/postgres/activity_repository.go` - full adapter; recursive CTEs for ancestry/commercial/billability; subtree-aware entry guards
- `internal/adapters/secondary/postgres/activity_repository_test.go` - 13 integration tests covering the Task 2 acceptance criteria
- `internal/core/domain/time_entry/time_entry.go` - ProjectID/SubprojectID/WGID → ActivityID (+UnitID kept)
- `internal/core/domain/expense/expense.go` - ProjectID → ActivityID (+UnitID kept)
- `internal/core/ports/time_entry_repository.go` + `expense_repository.go` - IsPeriodLocked takes activityID; ListFilters/ExpenseListFilters ActivityID replaces ProjectID/WGID
- `internal/adapters/secondary/postgres/time_entry_repository.go` + `expense_repository.go` - activity_id SQL everywhere; ListPending via activity→WG chain; IsPeriodLocked org+activity+date
- `internal/adapters/secondary/postgres/working_group_repository.go` - activity_id anchor, unit-tuple toggle dropped from all SQL
- `internal/adapters/secondary/postgres/contract_repository.go` - time_entries_count / HasTimeEntries / RecalculateMileage / HasProjects via activity subtree CTEs
- `internal/adapters/secondary/postgres/export_repository.go` - timesheets/expenses resolve project/contract/customer through the commercial chain CTE; manager role filter via WG anchor
- Test helpers (`exported_test_helpers.go`): seedProject/seedSubproject → seedActivity/seedActivityKind; entry + wg + export tests updated to activity seeds

## Decisions Made

- **Domain layout follows the repo convention** — the plan's flat `internal/core/domain/activity.go` maps to `internal/core/domain/activity/activity.go` (package `activity`), matching every other entity's subdirectory layout (project/, time_entry/, expense/…).
- **BudgetAmount is `*float64`, not `*decimal`** — the module has no decimal library; every money field in the codebase is float64 (Expense.Amount, Contract.KmRate, OrganizationSettings.DefaultKmRate). DECIMAL(12,2) scans into float64 natively via pgx. Adding shopspring/decimal for a single legacy-only field was rejected as scope creep.
- **working_group domain keeps its legacy `SubprojectID` field name** — the plan's `files_modified` does not list the domain file, and `services/working_group` + `http/working_group.go` are absent from plans 04/05 file lists. Renaming the field would break those packages with no follow-up plan to repair them. The repository maps the `activity_id` column into the field; the cosmetic rename is deferred (noted in the field comment).
- **contract/export repo rewrites are Rule 3 auto-fixes, not scope creep** — both repositories' SQL referenced the `projects` table dropped in 011. The plan's verification (`go test ./internal/adapters/secondary/postgres/...` green) is unachievable without them, and the plan's own objective ("commercial-chain queries rewritten as recursive CTEs against the activity tree") names exactly this work.
- **`financial_cutoff_period_repository.go` doesn't exist as a file** — the plan lists it in files_modified, but the financial_cutoff_periods queries live inside the entry repos' `IsPeriodLocked` methods. Those were updated in place to the R-5 key (org + activity + date range).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] contract_repository.go commercial-chain queries hit the dropped `projects` table**
- **Found during:** Task 2 (postgres package verification)
- **Issue:** `baseContractQuery` (time_entries_count subquery), `RecalculateMileage`, `HasProjects`, `HasTimeEntries` all reference `projects`/`project_id` — the tables/columns were dropped by migration 011. The plan verification requires the whole postgres package green; these queries are commercial-chain queries the plan intro explicitly scopes ("rewritten as recursive CTEs against the activity tree").
- **Fix:** Rewrote all four to walk the contract's activity subtree (recursive CTE from `activities.contract_id`); `HasProjects` counts contract-linked activities.
- **Files modified:** internal/adapters/secondary/postgres/contract_repository.go
- **Verification:** full postgres suite passes (contract tests, incl. HasTimeEntries)
- **Committed in:** e7a6d88 (Task 2 commit)

**2. [Rule 3 - Blocking] export_repository.go joins hit the dropped `projects` table**
- **Found during:** Task 2 (postgres package verification)
- **Issue:** `Timesheets`/`Expenses` LEFT JOIN `projects` on `te.project_id`/`e.project_id`; `roleFilter` manager branch joined `project_managers`. All gone after 011 → export tests fail → plan verification fails.
- **Fix:** Rewrote both queries with a `WITH RECURSIVE commercial` chain CTE resolving each entry's activity name + nearest contract + customer (D-3); manager role filter now resolves via `working_groups.activity_id` (R-1).
- **Files modified:** internal/adapters/secondary/postgres/export_repository.go
- **Verification:** export integration tests pass (timesheets, expenses, empty, employee filter)
- **Committed in:** e7a6d88 (Task 2 commit)

**3. [Rule 1 - Bug] `RETURNING` clause used the `te.`/`e.` table alias**
- **Found during:** Task 3 (first full-suite run)
- **Issue:** PostgreSQL rejects table aliases in INSERT/UPDATE `RETURNING` clauses (`missing FROM-clause entry for table "te"`, SQLSTATE 42P01) — the new entry repos aliased `time_entries te`/`expenses e` and reused the aliased column list in `RETURNING`.
- **Fix:** Split each repo's column constant into an aliased SELECT variant and an unaliased RETURNING variant.
- **Files modified:** internal/adapters/secondary/postgres/time_entry_repository.go, expense_repository.go
- **Verification:** entry repo integration tests pass
- **Committed in:** c101494 (Task 3 commit)

**4. [Rule 1 - Bug] Activity Create test violated the kind-catalog FK**
- **Found during:** Task 2 (first test run)
- **Issue:** `TestActivityRepository_Create_Get` created kind `"phase"` but only the `"engagement"` kind row existed in the org's catalog → FK violation (`activities(org_id, kind) → activity_kinds`).
- **Fix:** Seeded `"phase"` via `seedActivityKind` in the test; also fixed a raw `INSERT ... expenses` column-count mismatch in `TestActivityRepository_HasActiveExpenses`.
- **Files modified:** internal/adapters/secondary/postgres/activity_repository_test.go
- **Verification:** activity repo tests pass
- **Committed in:** e7a6d88 (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (2 blocking, 2 bug)
**Impact on plan:** The two Rule 3 fixes were required for the plan's own verification gate (postgres package green) and match the plan's stated commercial-chain objective. The two Rule 1 fixes were required for the tests to run. No scope creep — no service or handler code was touched.

## Issues Encountered

- **Expected mid-phase breakage (not a deviation):** five packages fail to compile until plans 04/05 rewrite them, exactly as the plan's Task 1 acceptance states ("services/handlers will be broken until Plans 04/05 — that's expected"): `cmd/server` (wires `NewProjectRepository`/`NewSubprojectRepository`), `internal/adapters/primary/http` (`project.go` handler + helper), `internal/core/services/{project,time_entry,expense}` (import deleted project domain / use renamed request fields). `services/working_group` and its handler still compile (legacy field name kept); only `working_group_integration_test.go`'s project-seeding test remains runtime-red like the rest of the services suite.
- `go test ./...` remains red by design (services layer) — same accepted state documented in 09-01/09-02 summaries.

## Verification Results

- `go build ./internal/core/domain/... ./internal/core/ports/... ./internal/adapters/secondary/postgres/...` — **PASS**
- `go vet` on the same packages — **PASS**
- `go test ./internal/adapters/secondary/postgres/ -count=1` — **PASS** (19s; includes 13 new activity-repo tests + rewritten entry/wg/export/contract tests)
- `go build ./...` — **EXPECTED FAIL** on the 5 service/handler/wiring packages listed above (until 04/05)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Repository layer fully on the activity ontology: one Activity domain/port/adapter, entries pinned to activity_id + unit_id, commercial chain + billability derived via CTE, ListPending routed through activity → WG
- **Ready for 09-04 (Service Layer)** — consumes `ports.ActivityRepository` (GetAncestry/ResolveCommercialContext/ResolveBillability/ListChildren) and the activity-shaped entry ports; must repair the 5 broken packages (services/project delete, time_entry/expense routing rewrite, mocks/factories)
- **Heads-up for the 09-04 executor:** `services/testdata/mocks.go` + `factories.go` import the deleted `domain/project` package — they need the activity-shaped mocks; the working_group service keeps the legacy `SubprojectID` field name (compile-safe) until its cosmetic rename is scheduled
- **Blockers/concerns:** none from this plan. `go build ./...` red on services/handlers is the accepted mid-phase state (same as 09-01/09-02's red tests by design). The pre-existing 000 down-migration gap remains tracked in deferred-items.md.

---
*Phase: 09-activity-ontology*
*Completed: 2026-07-31*
