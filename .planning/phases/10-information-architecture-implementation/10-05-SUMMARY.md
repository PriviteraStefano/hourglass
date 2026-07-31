---
phase: 10-information-architecture-implementation
plan: 05
subsystem: ui
tags: [react, tanstack-router, react-query, vitest, msw, playwright, approvals, working-groups, go, pending-queue]

# Dependency graph
requires:
  - phase: 10-information-architecture-implementation (10-02)
    provides: deriveApprovalStages (WG manager_id/delegate_ids + org-role stage derivation), WorkingGroupsApis.workingGroupsQueryOpts
  - phase: 10-information-architecture-implementation (10-04)
    provides: pending-query gating pattern (spread + enabled override), /approvals links from the Today header + "Review now" accent
  - phase: 09-activity-ontology
    provides: entry DTOs with activity_id/activity_name, working-groups HTTP surface, repo ListPending "wg_manager" role branch
provides:
  - /approvals route (ADR-P-011 D-3): one page, stage-filtered Manager/Finance tabs, merged pending TE+expense queue with Approve/Reject round-trips
  - Backend ListPending handler gate relaxation: WG manager/delegate (org-role employee) admitted via Service.IsWGManager + role "wg_manager" (T-10-05-3)
  - Stage-gated pending queries (enabled: stages.length > 0) — no 403 spam for plain employees/HR
  - 403 ≠ "Queue is clear": locked error state with router.invalidate() recovery when tabs render (T-10-05-2)
  - e2e approvals suite (approve, reject-with-reason, employee visibility, finance chain) — 4/4 green
affects: [10-06 Working Groups (WG surface), any consumer of pending approval state]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Stage tabs from deriveApprovalStages output; single-stage users skip the tab bar and render their queue directly (UI-SPEC)"
    - "Backend approval admission: handler resolves WG membership via Service.IsWGManager (mirrors resolveManagerStage's wgRepo path) and passes role wg_manager — the repo's existing WG-scoped branch"
    - "Unified queue row shape { kind, id, date, activityName, value, status } merged client-side from pending TE + expense queries, sorted by entry_date"
    - "Approve/reject mutations invalidate pending keys + month keys so the Today preview stays fresh"

key-files:
  created:
    - web/src/routes/_authenticated/approvals/index.tsx
    - web/src/routes/_authenticated/approvals/-components/approvals-page.tsx
    - web/src/routes/_authenticated/approvals/-components/__tests__/approvals-page.test.tsx
    - web/e2e/approvals.spec.ts
  modified:
    - internal/adapters/primary/http/time_entry.go (ListPending gate)
    - internal/adapters/primary/http/expense.go (ListPending gate)
    - internal/core/services/time_entry/time_entry.go (IsWGManager)
    - internal/core/services/expense/expense.go (IsWGManager)
    - web/src/routeTree.gen.ts (regenerated with /approvals route)
    - web/e2e/helpers.ts (WG seed description fix)

key-decisions:
  - "Relaxed the ListPending handler gate (T-10-05-3): org-role manager/finance OR WG manager/delegate admitted. The service layer already admitted WG-stage approvers via the repo's wg_manager branch (WG-scoped manager_id/delegate_ids query, proven by TestTimeEntryRepository_ListPending_WGRoleFilter) — only the handler gate and the middleware-role gap blocked them. IsWGManager mirrors resolveManagerStage's wgRepo path."
  - "Backend stays authoritative for approve/reject (unchanged service gates); the client tab scoping is UX only (T-10-05-1)"
  - "Stage tabs render only from deriveApprovalStages output; single-stage users (the common case) skip the tab bar entirely (UI-SPEC)"
  - "403 while tabs render surfaces the locked error state (We couldn't load Approvals. {reason}. Try again.) with router.invalidate() — never 'Queue is clear' (T-10-05-2)"
  - "URL-shareable stage via validateSearch (ADR-FE-017): /approvals?stage=finance"

patterns-established:
  - "Approvals page = lean file route (validateSearch + errorComponent: RouteError) + composition in -components/approvals-page.tsx (10-04 convention)"
  - "Non-stage direct access renders a locked muted notice with pending queries enabled:false — proven by msw request capture in unit tests"

requirements-completed: [P-011-D3, BE-014-R1]

# Metrics
duration: 48min
completed: 2026-08-01
---

# Phase 10 Plan 05: Approvals queue — one page, stage-filtered Manager/Finance tabs

**`/approvals` is live: a single page rendering stage-filtered Manager/Finance queues of merged pending time entries + expenses, with reason-required reject and approve round-trips against the live backend — and the backend ListPending gate now admits WG manager/delegate approvers whose org role is employee (T-10-05-3), closing the employee-with-WG-stage 403 gap**

## Performance

- **Duration:** 48 min
- **Started:** 2026-07-31T22:15:00Z
- **Completed:** 2026-08-01T23:03:24Z
- **Tasks:** 4 (5 commits)
- **Files modified:** 10 (4 created, 6 modified; 1029 insertions)

## Accomplishments

- **Backend gate resolution (Task 1):** verified the repo's `ListPending` already admits WG-stage approvers via its `wg_manager` role branch (WG-scoped on `manager_id`/`delegate_ids`), but the handler hard-gated org-role only AND middleware never sets `wg_manager` — so org-role employees who are WG managers got 403 on both pending endpoints. Added `Service.IsWGManager` (mirrors `resolveManagerStage`'s wgRepo path) to the time-entry + expense services and relaxed both `ListPending` handlers to admit org-role manager/finance OR WG manager/delegate, passing role `wg_manager` so the repository WG-scopes the queue. Handler tests: employee-without-stage → 403 (kept), employee-WG-manager → 200 (new). Gate semantics recorded in code comments per the acceptance criterion.
- **Approvals page (Task 2):** `/approvals` route with `validateSearch: { stage? }` (URL-shareable, ADR-FE-017) and `errorComponent: RouteError`. `ApprovalsPage` renders the shell `Header` with the single `h1` "Approvals" + Manager/Finance `Tabs` (only for dual-stage users; single-stage users skip the tab bar), a merged pending TE+expense queue (rows `py-3`, `.font-text` numerics, `StatusBadge` unchanged, existing `ApprovalButtons` with reason-required reject), per-stage locked empty state "Queue is clear" with `{stage}` interpolated, and a locked error state on query failure — a 403 while tabs render is an error state, never empty (T-10-05-2). Approve/reject mutations invalidate pending + month keys (Today stays fresh) with sonner toasts. Non-stage direct access (employee/HR) renders a muted notice with pending queries `enabled: false` — zero 403 spam (proven by msw request capture).
- **Unit tests (Task 3):** 8 composition tests through the real routeTree harness — manager/finance queue rows with the Approve/Reject pair, single-stage no-tab-bar, dual-stage both tabs, approve fires `POST /time-entries/{id}/approve` + invalidates pending (msw refetch observed), reject reason-gated then `POST …/reject` with `{ reason }`, 403 → error state with empty state absent, employee/HR muted notice with zero pending requests, empty queue copy with stage interpolated.
- **E2E (Task 4):** `e2e/approvals.spec.ts` seeds manager+employee+finance in one org (psql membership moves; registration always creates a fresh org), the manager as WG manager so entries route to the manager stage — exercising the Task 1 admission end-to-end. Covers: employee visibility (no Review group, `/approvals` muted notice), manager approve (row leaves queue, status `pending_finance` via API), manager reject-with-reason (status `rejected`, reason persisted in `time_entry_approvals`, employee sees the Rejected badge), and the finance chain (manager approve → finance sees `pending_finance` → approve → `approved`). 4/4 green.

## Task Commits

Each task was committed atomically:

1. **Task 1: handler gate resolution (IsWGManager + ListPending admission)** - `83fd31f` (feat)
2. **Task 2: Approvals page composition** - `2f1443d` (feat)
3. **Task 3: approvals-page composition tests (eight cases)** - `857b3e9` (test)
4. **Task 4: approvals e2e suite** - `e7d1547` (test)
5. **Task 2/3 follow-up: WG cast typecheck fix in tests** - `c03ed33` (fix)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `internal/core/services/time_entry/time_entry.go` - `Service.IsWGManager`: resolves the manager-stage approver set (WG manager_id + delegate_ids) from wgRepo — the admission primitive for the ListPending gate
- `internal/core/services/expense/expense.go` - `Service.IsWGManager` (expense twin, same semantics)
- `internal/adapters/primary/http/time_entry.go` - `ListPending` gate: org-role manager/finance OR WG manager/delegate (passes role `wg_manager` → WG-scoped queue); gate semantics in comments
- `internal/adapters/primary/http/expense.go` - `ListPending` gate identical relaxation
- `web/src/routes/_authenticated/approvals/index.tsx` - file route: `validateSearch: { stage? }`, `errorComponent: RouteError`, component `ApprovalsPage`
- `web/src/routes/_authenticated/approvals/-components/approvals-page.tsx` - `ApprovalsPage`: Header + stage Tabs, merged queue, mutation wiring (invalidate + toasts), empty/error/muted states
- `web/src/routes/_authenticated/approvals/-components/__tests__/approvals-page.test.tsx` - 8 msw-backed composition tests incl. request-capture gate proofs
- `web/e2e/approvals.spec.ts` - 4 e2e tests (approve, reject-with-reason, visibility, finance chain)
- `web/src/routeTree.gen.ts` - regenerated with `AuthenticatedApprovalsIndexRoute` at `/approvals`
- `web/e2e/helpers.ts` - `seedBaseEntities` WG insert now sets `description` (was NULL → `scanWorkingGroup` failed scanning NULL into string, 500-ing both `GET /working-groups` and the submit `resolveManagerStage` path)

## Decisions Made

- **Handler gate relaxation over service expansion:** the repo's `wg_manager` role branch already returns WG-scoped queues (proven by `TestTimeEntryRepository_ListPending_WGRoleFilter`); the only gaps were the handler's org-role-only check and the middleware never producing `wg_manager`. Relaxing the handler (with `IsWGManager` admission) reuses the existing, tested repo semantics instead of expanding service scope. Approve/Reject service gates are untouched — backend stays authoritative (T-10-05-1).
- **Single-stage users skip the tab bar** (UI-SPEC lock): the Manager tab is the direct queue for org-role managers and WG managers; Finance for finance. Tabs only appear for dual-stage users.
- **Error-vs-empty contract:** a 403 (or any query failure) while tabs render shows the locked "We couldn't load Approvals. {reason}. Try again." error state with `router.invalidate()` — never "Queue is clear" (T-10-05-2).
- **Stage derived from row status:** `pending_finance` → finance stage; `submitted`/`pending_manager` → manager stage — the client-side mirror of the BE-014 two-stage chain.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] e2e WG seed produced NULL description → scanWorkingGroup 500**
- **Found during:** Task 4 (approvals e2e submit 500)
- **Issue:** `seedBaseEntities` inserted `working_groups` without `description`; `scanWorkingGroup` scans the nullable TEXT column into a plain Go string, so both `GET /working-groups` and the submit path's `resolveManagerStage` (which lists WGs) returned 500. Latent in prior suites because none submitted through the WG resolution path — surfaced by my e2e's submit step.
- **Fix:** `web/e2e/helpers.ts` now inserts `description` in the WG seed (matching the Go test helper `seedWorkingGroup`, which always sets it).
- **Files modified:** web/e2e/helpers.ts
- **Verification:** approvals e2e submit 200; `GET /working-groups` 200
- **Committed in:** e7d1547 (Task 4 commit)

**2. [Rule 1 - Bug] Approvals unit test WG partial cast broke typecheck**
- **Found during:** post-restore typecheck sweep
- **Issue:** `{ id, org_id, manager_id, delegate_ids: [] } as WorkingGroup` — `delegate_ids: []` infers `never[]` and TS2352 flags the cast.
- **Fix:** dropped `delegate_ids: []` from the partial cast (matches the 10-04 today-page test convention `{ manager_id: USER_ID } as WorkingGroup`).
- **Files modified:** web/src/routes/_authenticated/approvals/-components/__tests__/approvals-page.test.tsx
- **Verification:** `bun run typecheck` clean for all approvals files; 8/8 tests green
- **Committed in:** c03ed33

**3. [Rule 3 - Blocking] Accidental `git stash pop` pulled an old pre-existing stash (hexagonal-migration WIP) into the working tree**
- **Found during:** post-Task-4 working-tree hygiene (attempting to prove the e2e date failures were pre-existing)
- **Issue:** a stale `stash@{0}` from a prior session (WIP on the hexagonal migration) was popped by mistake, partially applying deleted SurrealDB-era files (project.go, surrealdb/, `(auth)` routes, schema/, surreal.go, legacy handlers) as untracked/conflicting entries.
- **Fix:** restored tracked conflicts via `git checkout HEAD --` + staged them back, then removed the untracked stash-pollution paths (verified none exist in modern HEAD; the stash entry was preserved intact, no data loss). Working tree re-verified: only my plan's changes + the two pre-existing dirty files (workspace.json, migrate) remain; `go build ./...` and `bun run typecheck` re-verified.
- **Files modified:** none (restoration only)
- **Verification:** `git status --short` clean; `go build ./...` OK; approvals typecheck + tests green
- **Committed in:** n/a (working-tree hygiene, no content change)

**4. [Rule 1 - Bug] Approvals e2e row locator matched a sidebar `li`**
- **Found during:** Task 4 first e2e runs
- **Issue:** `locator('li').filter({ hasText: '4' }).first()` — the sidebar org-switcher `li` contains the org name `aprm_…`, and `.first()` could match a sidebar item; also hours-based matching is ambiguous across rows.
- **Fix:** seeded each entry on a distinct date (2026-08-01/02/03) and matched rows by the unique date text.
- **Files modified:** web/e2e/approvals.spec.ts
- **Verification:** approvals e2e 4/4 green twice
- **Committed in:** e7d1547 (Task 4 commit)

---

**Total deviations:** 4 auto-fixed (3 bugs, 1 blocking/environmental)
**Impact on plan:** All fixes were required for the plan's own tests to pass (seed correctness, locator stability, test typecheck). The stash-pop incident was an execution-environment mishap fully restored with zero content change. No scope creep.

## Issues Encountered

- **Pre-existing typecheck rot (unchanged baseline):** `bun run typecheck` still reports the 6 documented deferred-items errors; plan files contribute zero. `bun run build` remains blocked by the pre-existing `useSearchParams` missing export (rolldown hard-fail) — same as 10-01..10-04.
- **Pre-existing e2e date rollover (2026-08-01):** `time-entries.spec`, `expenses.spec`, and `error-boundary.spec` seed hard-coded **July 2026** dates; the default-month list views now show "No entries in this period" since the calendar rolled to August. These passed 42/42 on 2026-07-31 (10-04) — the failures are the seed dates, not a Phase 10 regression. My approvals spec seeds current-month (August) dates and is green 4/4. Logged to deferred-items.md; out of scope for Phase 10 IA work.
- **Interim flake:** expenses-list unit test timed out once under parallel vitest workers (passed in isolation and on the full re-run — same intermittent pattern the 10-04 summary documented for the approver test under load).
- **Graphify hook** rebuilds the knowledge graph on every commit (AGENTS.md directive) — handled automatically by the pre-commit hook.

## Known Stubs

None introduced by this plan. Pre-existing note: the time-entry detail surface (`entry-detail.tsx`) hardcodes `ApprovalHistory approvals={[]}` — the rejection reason is persisted backend-side (`time_entry_approvals.comment`, proven in e2e via psql) but not yet surfaced in the detail UI; there is no approvals-list endpoint yet. Out of scope for 10-05 (the plan reuses the existing pattern unchanged).

## Threat Flags

None — no new network endpoints, auth paths, file access, or schema changes were added. Threat model dispositions:

- T-10-05-1 (approve/reject by non-stage user): backend authoritative — Approve/Reject service + handler gates unchanged; client tab gating is UX scoping only.
- T-10-05-2 (403 conflated with "queue is clear"): the page renders the locked error state on any pending-query failure while tabs render; empty state renders only on a successful empty fetch — unit-tested.
- T-10-05-3 (employee-with-WG-stage 403): **resolved** — `ListPending` handlers now admit WG manager/delegate via `Service.IsWGManager` + role `wg_manager` (WG-scoped queue); handler tests prove 403 → 200; e2e proves the end-to-end round-trip.
- T-10-05-4 (reject without reason): existing `ApprovalButtons` reason-required pattern reused unchanged; e2e + unit tests assert the confirm stays disabled until a reason ≥ 10 chars.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 10-06 (Working Groups):** the WG surface builds on the same `WorkingGroupsApis` module; the approvals page's `IsWGManager`-style derivation is the mirror image of the WG detail's manager/delegate UI. Today's "Review now" links now resolve to a real route.
- **Ready for verification:** `/approvals` is live for manager/finance/WG-approver roles; the queue round-trips against the real backend with reason-required reject; HR and plain employees are cleanly scoped out (sidebar + page).
- **Blockers:** the pre-existing 6 typecheck errors + the e2e July-seed date rollover remain (deferred-items.md) — both predate 10-05 and block `bun run build` / full-e2e-green for every Phase 10 plan.

---

*Phase: 10-information-architecture-implementation*
*Completed: 2026-08-01*

## Self-Check: PASSED

- All 5 created/modified plan files exist on disk (5/5 FOUND); all 5 task commits verified in git log (83fd31f, 2f1443d, 857b3e9, e7d1547, c03ed33)
- `go test ./internal/...`: green (full suite, incl. postgres + http integration packages)
- `bun run test`: 122/122 unit tests pass (16 files) — twice consecutively
- `bunx playwright test e2e/approvals.spec.ts`: 4/4 green (twice)
- `bun run lint`: 0 errors (126 pre-existing warnings, none in plan files)
- `bun run typecheck`: zero errors in all 10-05 plan files; 6 pre-existing out-of-scope errors remain (deferred-items.md)
- Full e2e: 36 passed / 3 failed — the 3 failures are pre-existing July-seed date rollover in unrelated suites (deferred-items.md §2), not regressions; approvals suite green within the run
- routeTree.gen.ts verified: `AuthenticatedApprovalsIndexRoute` at `path: '/approvals'`
