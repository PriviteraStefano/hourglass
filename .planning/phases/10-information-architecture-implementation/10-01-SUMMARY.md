---
phase: 10-information-architecture-implementation
plan: 01
subsystem: ui, api
tags: [react, tanstack-router, react-query, typescript, playwright]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: /activities + /activity-kinds HTTP surface, Activity/ActivityDetail DTOs, activity_id on entry payloads
provides:
  - Frontend Activity/ActivityDetail/CreateActivityRequest/UpdateActivityRequest types mirroring Phase 9 DTOs
  - ActivitiesApis API layer (queries + create/update/delete mutations) replacing the dead projects API
  - /projects → /activities route rename with activity vocabulary (list, detail, create/edit dialogs)
  - Entry pickers rewired to activity_id + joined activity_name/activity_kind display
  - Role union extended with 'hr'
  - activities e2e suite with activity-based seeding (kind catalog + recursive activities + WG)
affects: [10-02 sidebar regroup (nav item), 10-03 shell wrap (activities page), 10-04 Today, 10-05 Approvals, 10-06 Working Groups]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "base-ui Combobox primitive for kind/parent/contract pickers (create-contract-dialog customer pattern)"
    - "Entry list views consume joined activity_name/activity_kind display fields instead of client-side name maps"
    - "Expense currency resolved activity_id → contract_id → contract.currency (Phase 8 decision preserved on the activity model)"

key-files:
  created:
    - web/src/api/activities.ts
    - web/src/api/__tests__/activities.test.ts
    - web/src/routes/_authenticated/activities/index.tsx
    - web/src/routes/_authenticated/activities/$id.tsx
    - web/src/routes/_authenticated/activities/-components/activity-list.tsx
    - web/src/routes/_authenticated/activities/-components/activity-detail.tsx
    - web/src/routes/_authenticated/activities/-components/create-activity-dialog.tsx
    - web/src/routes/_authenticated/activities/-components/edit-activity-dialog.tsx
    - web/e2e/activities.spec.ts
    - .planning/phases/10-information-architecture-implementation/deferred-items.md
  modified:
    - web/src/types/models.ts
    - web/src/types/api.ts
    - web/src/types/expense-types.ts
    - web/src/api/__tests__/{auth,contracts,customers,time-entries}.test.ts
    - web/src/components/layout/sidebar.tsx
    - web/src/components/layout/__tests__/route-error.test.tsx
    - web/src/routeTree.gen.ts
    - web/src/routes/_authenticated/time-entries/-components/{entry-detail,entry-row,time-entries-list}.tsx
    - web/src/routes/_authenticated/time-entries/index.tsx
    - web/src/routes/_authenticated/expenses/-components/{expenses-list,expense-form,expense-detail}.tsx
    - web/src/routes/_authenticated/expenses/index.tsx
    - web/src/routes/_authenticated/{time-entries,expenses}/-components/__tests__/*.test.tsx
    - web/e2e/helpers.ts

key-decisions:
  - "Billable checkbox in create/edit dialogs is a plain checkbox: checked = true, unchecked = undefined (inherit) — tri-state omitted per plan"
  - "owned|adopted|all tabs retained (GET /activities supports scope); Adopt button/dialog dropped (no endpoint)"
  - "Activity detail shows derived commercial context (contract linked / internal) and resolved billability from ActivityDetail payload; adoption count shows '—' (detail endpoint carries no adoption_count)"
  - "Entry creation cascade simplified to activity → children → working-group → unit (backend create needs only activity_id + unit_id)"
  - "kind picker uses the base-ui Combobox primitive fed by activityKindsQueryOpts (org catalog, D-2)"

patterns-established:
  - "API layer keyed on ['activities', ...] with children nested under ['activities', id, 'children']"
  - "grep-sweep gate for dead vocabulary: zero /projects references in web/src after rename"

requirements-completed: [P-007-D1, P-011-D6]

# Metrics
duration: 37min
completed: 2026-07-31
---

# Phase 10 Plan 01: Foundation — Role type + activities API layer + /projects → /activities rename

**Frontend rebuilt onto the Phase 9 activity ontology: Activity types + ActivitiesApis API layer replace the dead projects surface, /projects routes renamed to /activities with activity vocabulary, entry pickers rewired to activity_id, and the projects e2e suite replaced by activities e2e with recursive-activity seeding**

## Performance

- **Duration:** 37 min
- **Started:** 2026-07-31T20:15:00Z
- **Completed:** 2026-07-31T20:52:35Z
- **Tasks:** 3
- **Files modified:** 27 (9 created, 18 modified)

## Accomplishments

- `Role` union extended with `hr`; `Project`/`Subproject` interfaces deleted; `Activity`, `ActivityResponse`, `ActivityDetail`, `CreateActivityRequest`, `UpdateActivityRequest` mirror the Phase 9 Go DTOs
- `ActivitiesApis` (activities/activity/children/kinds queries + create/update/delete mutations keyed on `["activities"]`) replaces `api/projects.ts`; adopt mutation dropped with its endpoint
- `/projects` route directory renamed to `/activities`: list with owned|adopted|all tabs and kind badges, detail with ancestry breadcrumb + children list + derived commercial/billability display, create/edit dialogs with kind combobox (org catalog), parent + contract pickers, and billable checkbox
- Entry surfaces rewired: time-entry create cascade (activity → children → working-group → unit), entry-row/expense-form/expense-detail pickers on `activity_id`, list views use joined `activity_name`/`activity_kind`, expense currency via activity→contract chain
- Sidebar nav item Projects → Activities (`/activities`); routeTree regenerated with activities routes only
- e2e: `projects.spec.ts` → `activities.spec.ts`; helpers seed activity_kinds + recursive activities + WG on the child activity; full suite 41/41 green against the live backend (raised RATE_LIMIT/ANONYMOUS_RATE_LIMIT per Phase 8 convention)

## Task Commits

Each task was committed atomically:

1. **Task 1: Role type + activities API layer** - `e5e5994` (feat)
2. **Task 1 (deviation): queryFn/mutationFn typecheck rot in API tests** - `8a864d9` (fix)
3. **Task 2: route/component/vocabulary rename** - `e5f862f` (feat)
4. **Task 3: activities e2e + seeding** - `c9e3b13` (test)
5. **Task 3 (cleanup): drop unused wgId lookup** - `ee624cf` (refactor)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `web/src/types/models.ts` - Role + hr; Activity/ActivityResponse/ActivityDetail/Create+UpdateActivityRequest; TimeEntry on activity_id + joined display fields
- `web/src/types/api.ts` - Create/UpdateTimeEntryRequest on activity_id/unit_id; project request types removed
- `web/src/types/expense-types.ts` - Expense/Create+UpdateExpenseRequest on activity_id + joined display fields
- `web/src/api/activities.ts` - ActivitiesApis with 4 query + 3 mutation options
- `web/src/api/__tests__/activities.test.ts` - 8 MSW unit tests (renamed from projects.test.ts)
- `web/src/routes/_authenticated/activities/` - index/$id routes + ActivityList/ActivityDetail/CreateActivityDialog/EditActivityDialog
- `web/src/components/layout/sidebar.tsx` - nav item Projects → Activities (/activities)
- `web/src/routeTree.gen.ts` - regenerated (activities routes, no projects)
- `web/src/routes/_authenticated/time-entries/-components/` - entry-detail (activity cascade), entry-row (activity picker), time-entries-list (activity_name)
- `web/src/routes/_authenticated/expenses/-components/` - expenses-list (activity_name + currency chain), expense-form/expense-detail (activity pickers)
- `web/e2e/activities.spec.ts`, `web/e2e/helpers.ts` - activities CRUD e2e + activity-based seeding

## Decisions Made

- **Billable checkbox semantics:** checked → `billable: true`, unchecked → `undefined` (inherit). Tri-state intentionally omitted per plan instruction.
- **Detail-page commercial display:** uses `commercial_context` + resolved `billable` from the detail payload. Adoption count renders "—" because the detail endpoint (`GET /activities/{id}` → ActivityDetail) carries no adoption_count — the old `?? 0` fallback had no data source anymore.
- **Entry creation cascade:** activity → (children) → working-group → unit, matching migration 011 semantics where entries attach to child activities; backend create requires only `activity_id` + `unit_id` (`wg_id` dropped from the frontend request type).
- **Kind picker:** base-ui Combobox (the create-contract-dialog customer-combobox pattern) fed by `activityKindsQueryOpts` (staleTime 5 min).
- **Tabs:** kept `owned|adopted|all` (the list endpoint supports scope); removed the Adopt button/dialog entirely.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing TanStack v5.101 typecheck rot in the API test suite**
- **Found during:** Task 1 (verification)
- **Issue:** Baseline `bun run typecheck` already failed with **87 errors**. TanStack Query v5.101 types make `queryFn`/`mutationFn` possibly-undefined and require `MutationFunctionContext` as the second `mutationFn` argument — every direct-call site in the API test files (auth/contracts/customers/time-entries, and my new activities.test.ts) failed. Also `time-entries.test.ts` carried a stale create-fixture shape (`items` array from the old flat model).
- **Fix:** Non-null assertion + `undefined as any` context stub at direct-call sites (runtime unchanged — MSW intercepts); create-fixture updated to the current `CreateTimeEntryRequest` shape. My own activities.test.ts uses the same pattern.
- **Files modified:** web/src/api/__tests__/{auth,contracts,customers,time-entries,activities}.test.ts
- **Verification:** typecheck errors in test files → 0; unit suite 85/85 green
- **Committed in:** 8a864d9

**2. [Rule 3 - Blocking] Local docker DB was on the pre-Phase-9 schema (migration 011+ never applied)**
- **Found during:** Task 3 (e2e seeding — `relation "activity_kinds" does not exist`)
- **Issue:** The local `hourglass-postgres` DB still had the projects/subprojects schema; e2e seeding against `activities`/`activity_kinds` was impossible, and migration 011 failed on e2e-residue orgs because the kind catalog seed only covers the MVP org.
- **Fix:** Recreated the disposable e2e DB (`DROP/CREATE DATABASE hourglass`) and ran `go run ./cmd/migrate -up -dir migrations` — all migrations 000→013 applied cleanly on the fresh DB; backend restarted with `RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500`. Environment-only fix, no code change.
- **Verification:** full e2e suite 41/41 green
- **Committed in:** n/a (environment)

**3. [Rule 1 - Bug] Carried-over navigate type errors in the renamed components**
- **Found during:** Task 2 verification
- **Issue:** `activity-list.tsx` tab-change `navigate({ search: { tab } })` and `create-activity-dialog.tsx` post-create `navigate({ to: "/activities/$id" })` inherited baseline type errors from the project files (TS2353/TS2345 — the `$id` route requires its `from` search param).
- **Fix:** `navigate({ to: "/activities", search: { tab: next } })` with a typed union; `search: { from: "owned" }` on the post-create navigation.
- **Files modified:** activity-list.tsx, create-activity-dialog.tsx
- **Verification:** typecheck clean for all 10-01 files
- **Committed in:** c9e3b13

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug)
**Impact on plan:** All auto-fixes were necessary to meet the plan's own verification gates (typecheck, unit suite, full e2e against the live backend). No scope creep. The one criterion not fully met — `typecheck exits 0` — is blocked by 10 pre-existing errors in files owned by later Phase 10 plans (10-03 contract wrap) and out-of-phase surfaces; see deferred-items.md.

## Issues Encountered

- **Local DB schema drift:** the dev/e2e database predated Phase 9's schema rewrite. Migrations were applied by recreating the disposable DB (all non-seed rows were e2e residue). This is a one-time environment fix; the DB now matches migrations 000–013.
- **Migration 011 kind-catalog assumption:** migration 011 seeds `activity_kinds` only for the MVP org (`019df8b0-…`), so any other org with projects breaks the data migration. Under the MVP-scope assumption (ADR-P-007 D-6) this is by design; the fresh DB contains only the MVP org, so it applies cleanly.
- **e2e finance gate:** update/delete on activities are finance-gated — the activities spec promotes the seeded user via the existing `promoteToFinance` helper (matching customers-suite convention).

## Known Stubs

- `activity-detail.tsx` — "Adoption Count" renders "—" (the ActivityDetail payload has no `adoption_count`). Deliberate fallback for a field with no data source; will resolve when the detail endpoint carries sharing stats (future plan).
- No other stubs: all list/detail/form surfaces are wired to live API data.

## Threat Flags

None — no new security-relevant surface. The plan's threats were addressed:
- T-10-01-1 (dead /projects links): grep-sweep confirms zero live references in `web/src` (sidebar, routes, API, tests).
- T-10-01-2 (API drift): all frontend calls map 1:1 to `/activities` + `/activity-kinds`; e2e suite green against the live backend.
- T-10-01-3 (adopt affordance): Adopt button/dialog removed; `api/activities.ts` carries no adopt mutation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 10-02 (sidebar regroup + role-scoped visibility):** sidebar nav item already points at `/activities`; `Role` now includes `hr` so the role-matrix rendering can type-narrow; all Phase 10 plans can import `ActivitiesApis`/`Activity` types.
- **Ready for 10-03 (shell wrap):** activities index/$id pages are cleanly rename-complete, awaiting the Header+Body wrap.
- **Ready for 10-06 (Working Groups):** the create-activity dialog's parent picker pattern (activity combobox fed by `activitiesQueryOpts("owned")`) is reusable for the WG activity picker.
- **Blockers:** none. The 10 pre-existing typecheck errors (deferred-items.md) live outside this plan's files; 10-03 should fold contract-page fixes into its wrap migration.

---
*Phase: 10-information-architecture-implementation*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 9 created files exist on disk; all 5 task/deviation commits verified in git log
- `bun run test`: 85/85 unit tests pass
- `bunx playwright test`: 41/41 e2e tests pass (activities rename + all carried surfaces)
- `bun run lint`: exit 0
- `go test ./...`: green (backend untouched, non-interference confirmed)
- `bun run typecheck`: zero errors in 10-01 files; 10 pre-existing errors in unrelated files remain (deferred-items.md)
