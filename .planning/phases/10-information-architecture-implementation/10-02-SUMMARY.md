---
phase: 10-information-architecture-implementation
plan: 02
subsystem: ui
tags: [react, tanstack-router, react-query, vitest, msw, sidebar, role-based-visibility]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: working-groups HTTP surface (manager_id/delegate_ids payloads), Role union with hr on the frontend
  - phase: 10-information-architecture-implementation (10-01)
    provides: Role type incl. hr, ActivitiesApis/activities nav item, typecheck-clean api test conventions
provides:
  - D-1 pillar-mapped sidebar (Today/Track/Work/People/Economics/Review/Reports/Admin) with exact locked labels and render order
  - ADR-P-011 D-5 role-scoped visibility driven by pure helpers (deriveApprovalStages/isReviewVisible/isEconomicsVisible/isAdminVisible), unit-tested across all five roles
  - WorkingGroupsApis.workingGroupsQueryOpts (GET /working-groups, 60s staleTime) — stable home for the WG mutation layer in Plan 10-06
  - Disabled placeholders (Tickets/Availability) with UI-SPEC locked tooltip copy
  - Review group gated on WG-derived approval stage (manager_id/delegate_ids + auth/me user id — no new endpoint)
affects: [10-04 Today (stage derivation), 10-05 Approvals (stage tabs), 10-06 Working Groups (query module + mutations), any future surface consuming approval stage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sidebar nav as a single declarative navStructure array filtered by pure role-matrix predicates (no visibility logic inline in JSX)"
    - "Approval stage derived client-side from route-context profile + org-scoped WG list; UX scoping only, backend stays authoritative"
    - "Route-context profile consumption via useRouteContext({ from: '/_authenticated' }) — the beforeLoad hydration source, matching ProfileMenu's data"

key-files:
  created:
    - web/src/api/working-groups.ts
    - web/src/lib/role-visibility.ts
    - web/src/lib/__tests__/role-visibility.test.ts
    - web/src/components/layout/__tests__/sidebar-groups.test.tsx
  modified:
    - web/src/components/layout/sidebar.tsx
    - web/src/types/models.ts

key-decisions:
  - "WorkingGroup frontend type added to models.ts mirroring the Go payload (manager_id, delegate_ids) — the plan's helper signature requires the type; models.ts is the established domain-type home"
  - "workingGroupsQueryOpts exported both as a named const and inside WorkingGroupsApis so the acceptance contract and Plan 10-06 imports are both satisfied"
  - "Profile read via useRouteContext({ from: '/_authenticated' }) (beforeLoad hydration) instead of a second useSuspenseQuery — no extra fetch, no new suspense boundary; ProfileMenu/OrgSwitcher self-fetch patterns untouched"
  - "Tooltip-copy tests hover the collapsed sidebar (base-ui opens on mouseenter after the 600ms rest delay) — the only DOM-visible path for tooltip content"
  - "Admin group renders nothing for every v0.1 role (isAdminVisible false) — the disabled Settings item disappears per the D-5 matrix, contrary to today's always-visible Settings"

patterns-established:
  - "Role-matrix sidebar rendering: declarative navStructure + filter by pure predicates; group labels inherit SidebarGroupLabel primitive styling (locked 500-weight exception)"

requirements-completed: [P-011-D1, P-011-D5]

# Metrics
duration: 9min
completed: 2026-07-31
---

# Phase 10 Plan 02: Sidebar regroup (D-1 pillars) + role-scoped visibility (D-5)

**ADR-P-011 D-1 pillar groups (Today/Track/Work/People/Economics/Review/Reports/Admin) replace the mechanic-named Tracking/Management sidebar, with D-5 role-scoped visibility computed by pure, unit-tested helpers deriving approval stages from the route-context profile + WG manager_id/delegate_ids payloads**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T21:00:02Z
- **Completed:** 2026-07-31T21:08:40Z
- **Tasks:** 3
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments

- Sidebar rebuilt onto a single declarative `navStructure` (8 entries, locked labels + render order), preserving the AppSidebar shell, OrgSwitcher/ProfileMenu/ThemeToggle blocks and collapse toggle verbatim; `SidebarSeparator` between groups as today
- Role matrix implemented as pure helpers: `deriveApprovalStages` (org-role finance/manager + WG `manager_id`/`delegate_ids` union; hr stripped per ADR-P-008 D-4), `isReviewVisible` (stages > 0 && role ≠ hr), `isEconomicsVisible` (not employee/customer), `isAdminVisible` (false for all v0.1 roles)
- Disabled placeholders carried with locked UI-SPEC copy: Tickets → "Tickets arrive in v0.2", Availability → "Availability lands with the staffing schema"
- Today active-state uses `useMatchRoute({ to: item.href, fuzzy: item.href !== "/" })` so `/` matches only the landing route (threat T-10-02-3)
- `WorkingGroupsApis.workingGroupsQueryOpts` (GET /working-groups, `["working-groups"]` key, 60s staleTime) gives the WG stage derivation a stable, mutation-ready home (Plan 10-06 adds CRUD mutations)
- 23 new tests: 15 pure-helper matrix tests + 8 sidebar render tests through the real routeTree harness

## Task Commits

Each task was committed atomically:

1. **Task 1: WG query + role-visibility helpers** - `8d4b695` (feat)
2. **Task 1 (criterion gap): named export of workingGroupsQueryOpts** - `8c24297` (feat)
3. **Task 2: sidebar D-1 regroup + role-scoped visibility** - `aaaf6b7` (feat)
4. **Task 3: role-visibility + sidebar render tests** - `5ccc0ba` (test)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `web/src/lib/role-visibility.ts` - `ApprovalStage` type + `deriveApprovalStages`/`isReviewVisible`/`isEconomicsVisible`/`isAdminVisible` pure predicates (UX scoping only — comments flag backend authority, threat T-10-02-1)
- `web/src/api/working-groups.ts` - `workingGroupsQueryOpts` (named + via `WorkingGroupsApis`); mutations deferred to 10-06
- `web/src/types/models.ts` - `WorkingGroup` interface mirroring the Go payload (`manager_id`, `delegate_ids`, `subproject_id` legacy anchor)
- `web/src/components/layout/sidebar.tsx` - navStructure regroup, role filtering, locked tooltips, fuzzy-exact Today match
- `web/src/lib/__tests__/role-visibility.test.ts` - 15 helper tests (all roles, delegate, undefined-WG)
- `web/src/components/layout/__tests__/sidebar-groups.test.tsx` - 8 render tests (order, Today href, role-conditional groups, tooltip copy, no legacy labels)

## Decisions Made

- **WorkingGroup type location:** `web/src/types/models.ts` (domain-type home, re-exported via `@/types`) — the plan's `deriveApprovalStages(profile, workingGroups: WorkingGroup[])` signature required a type the plan didn't place; Rule 2 addition.
- **Named + namespaced export:** `workingGroupsQueryOpts` exported as a named const AND inside `WorkingGroupsApis` — satisfies the acceptance criterion verbatim and keeps the 10-06 object-extension convention.
- **Profile source:** `useRouteContext({ from: "/_authenticated" })` — reads the beforeLoad-hydrated route context (already the data source for everything in this tree) instead of a second self-fetch; no suspense boundary added to the shell.
- **Admin group behavior:** hidden entirely for v0.1 (isAdminVisible false), so the disabled Settings item disappears — a deliberate change from today's always-visible Settings, matching the D-5 matrix (no org-admin role exists yet).
- **Tooltip testing:** collapse sidebar + `fireEvent.mouseEnter` (base-ui opens on mouseenter after 600ms rest delay) — the only DOM-observable path for TooltipContent, which renders only while open and only when collapsed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] WorkingGroup type added to models.ts**
- **Found during:** Task 1 (implementing `deriveApprovalStages`)
- **Issue:** The plan's helper signature requires a `WorkingGroup` type, but no frontend WG type exists yet (Plan 10-06 was expected to introduce the surface) and the plan didn't specify where the type lives.
- **Fix:** Added `WorkingGroup` to `web/src/types/models.ts` mirroring the Go payload exactly (id, org_id, subproject_id legacy anchor, name, description, unit_ids, enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at).
- **Files modified:** web/src/types/models.ts
- **Verification:** typecheck clean in my files; all 23 new tests green
- **Committed in:** 8d4b695 (Task 1 commit)

**2. [Rule 2 - Missing Critical] Named export of `workingGroupsQueryOpts`**
- **Found during:** Task 1 acceptance verification
- **Issue:** Acceptance criterion reads "exporting `workingGroupsQueryOpts`" — the initial module only exported it inside the `WorkingGroupsApis` object.
- **Fix:** `export const workingGroupsQueryOpts` alongside the object export (both consumers satisfied: the criterion and the 10-06 extension convention).
- **Files modified:** web/src/api/working-groups.ts
- **Verification:** grep confirms named export; typecheck unchanged at 10 baseline errors
- **Committed in:** 8c24297

---

**Total deviations:** 2 auto-fixed (2 missing critical)
**Impact on plan:** Both additions are minimal and necessary for the plan's own signatures and acceptance criteria. No scope creep; no new dependencies.

## Issues Encountered

- **Pre-existing typecheck blocker (baseline, unchanged by 10-02):** `cd web && bun run typecheck` reports 10 errors, all in files outside this plan's surface (api.test.ts, __root.tsx, _auth/*, contracts/*, org-hierarchy unit-detail-panel). This is the same blocker documented in deferred-items.md from 10-01 — the plan's `typecheck exits 0` criterion is technically unattainable at baseline; my files contribute zero errors (grep-filtered verification). 10-03 owns the contracts/* errors.
- **Tooltip hover in jsdom:** `fireEvent.pointerEnter` dispatches `pointerType: ""` and base-ui's non-move hover interaction listens to `mouseenter`; switching to `fireEvent.mouseEnter` opens the popup after the 600ms rest delay (verified with the collapsed-sidebar test). No production-code impact.
- **"Today first" assertion:** the first `sidebar-menu-item` is the collapse toggle, not Today — fixed by asserting document-order (Today precedes the first `sidebar-group-label` "Track") instead of positional index.

## Known Stubs

None — all sidebar nav items are wired to real routes (or intentionally disabled placeholders per UI-SPEC with locked copy); the WG query hits the live `GET /working-groups` endpoint.

## Threat Flags

None — no new network endpoints, auth paths, file access, or schema changes. The plan's threats were addressed:

- T-10-02-1 (role scoping mistaken for authorization): explicit comments in `role-visibility.ts` and `sidebar.tsx` ("hiding is UX scoping only; every role-restricted surface stays backend-gated"); zero client-side route guards invented.
- T-10-02-2 (stale WG list): 60s staleTime on `workingGroupsQueryOpts`, invalidated on org switch via the existing `queryClient.clear()` convention.
- T-10-02-3 (`/` prefix-matching active state): `fuzzy: item.href !== "/"` for Today + sidebar test asserting `href="/"` and DOM position.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 10-04 (Today):** `deriveApprovalStages` is the shared stage source the "Waiting on you" section needs to decide whether to render the approver block.
- **Ready for 10-05 (Approvals):** Review group visibility + stage derivation already live in the sidebar; the Approvals page can reuse `deriveApprovalStages` for its Manager/Finance tab scoping.
- **Ready for 10-06 (Working Groups):** `WorkingGroupsApis` module + `WorkingGroup` type are the stable home for the CRUD/member mutations and the WG surface.
- **Blockers:** none beyond the documented pre-existing typecheck rot (deferred-items.md).

---
*Phase: 10-information-architecture-implementation*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 4 created files exist on disk; all 4 task commits verified in git log
- `bun run test`: 108/108 unit tests pass (85 prior + 15 role-visibility + 8 sidebar-groups)
- `bun run lint`: 0 errors (126 pre-existing warnings, none in plan files)
- `bun run typecheck`: zero errors in all 6 plan files; 10 pre-existing baseline errors in out-of-plan files remain (deferred-items.md)
- `go test ./...`: backend untouched (frontend-only plan), no regression surface
