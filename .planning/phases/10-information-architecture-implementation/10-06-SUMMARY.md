---
phase: 10-information-architecture-implementation
plan: 06
subsystem: ui
tags: [react, tanstack-router, react-query, playwright, working-groups, combobox, members]

# Dependency graph
requires:
  - phase: 09-activity-ontology
    provides: live WG HTTP surface (GET/POST /working-groups, GET/PUT/DELETE /working-groups/{id}, GET/POST/DELETE /working-groups/{id}/members), activity_id anchor, member role payloads
  - phase: 10-information-architecture-implementation (10-01)
    provides: ActivitiesApis (activity combobox source), Role type, route-tree conventions
  - phase: 10-information-architecture-implementation (10-02)
    provides: WorkingGroupsApis module + WorkingGroup frontend type (subproject_id legacy anchor), 60s staleTime list query
  - phase: 10-information-architecture-implementation (10-03)
    provides: Header+Body page shell, @/components/layout barrel, leaf-level errorComponent convention
provides:
  - /working-groups route under Work: list + client-side search + create/edit dialog + destructive delete against the live WG API
  - Full WorkingGroupsApis: workingGroupQueryOpts, workingGroupMembersQueryOpts + create/update/delete/addMember/removeMember mutations (all invalidate ["working-groups"] + sonner toasts)
  - WorkingGroupFormDialog (create+edit): name/activity/manager required + delegate chips multi-select; org_id sourced from route-context profile (never user input)
  - WorkingGroupMembersDialog: list (name + role badge via org-members join), add (user + unit + role 'member'/'lead'), remove with destructive confirm — distinct from the approver set (manager/delegate live in the form dialog only)
  - Locked UI-SPEC empty state ("No working groups yet" + CTA) and single accent header CTA
  - web/e2e/working-groups.spec.ts: 5/5 green CRUD flows against the live backend
affects: [P-008 availability/validity warnings (explicitly deferred), phase-level UAT visual walk, any future surface reusing the members dialog pattern]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Combobox chips multi-select (base-ui Combobox multiple + ComboboxChips/Chip/ChipsInput) for delegate_ids — the vendored combobox supports chips mode via an anchor ref"
    - "Resolve joined display names client-side (activity via activities cache by subproject_id, manager/member names via orgMembersQueryOpts) — WG API payloads carry no joined names"
    - "api<T> returns undefined on 204 No Content — unblocks ALL delete mutations (pre-existing shared-helper bug, Rule 1)"
    - "e2e: seed activity/unit WITHOUT a WG so the empty state is real; '' for nullable text columns (Phase 08 psql convention — NULL rows 500 the activities/units list scans)"

key-files:
  created:
    - web/src/routes/_authenticated/working-groups/index.tsx
    - web/src/routes/_authenticated/working-groups/-components/working-groups-page.tsx
    - web/src/routes/_authenticated/working-groups/-components/working-group-form-dialog.tsx
    - web/src/routes/_authenticated/working-groups/-components/working-group-members-dialog.tsx
    - web/src/routes/_authenticated/working-groups/-components/delete-working-group-dialog.tsx
    - web/e2e/working-groups.spec.ts
  modified:
    - web/src/types/models.ts
    - web/src/api/working-groups.ts
    - web/src/lib/api.ts
    - web/src/routeTree.gen.ts

key-decisions:
  - "WG type keeps subproject_id (legacy activity anchor) — the plan's 'activity_id' name conflicts with the live API JSON key; the 10-02 type is exact and untouched"
  - "org_id on create comes from the route-context profile (profile.membership.organization_id), never user input — the backend Create handler REQUIRES org_id in the body, so the threat model's 'client never sends org ids' is satisfied in spirit: a client can only ever target its own org"
  - "Members dialog (not expandable card) per planner's discretion — cards stay compact; member count still renders per card via a members query"
  - "Add-member form carries a unit picker — AddMember REQUIRES user_id AND unit_id (handler 400s otherwise); the plan action text omitted the backend-required unit field (Rule 3)"
  - "Member role select offers exactly 'member'/'lead' — the two values the codebase uses (seed 003 + wg_member_repository_test); wg_members.role is an unconstrained VARCHAR, no invented roles"
  - "RouteError registered as route-level leaf errorComponent (Phase 8 P0-4 convention), rendering inside the shell frame per UI-SPEC — matches customers/approvals"
  - "api<T> 204 handling fixed in the shared helper — every existing DELETE mutation (customers/activities/contracts/units/WG) was rejecting after a successful server response"

requirements-completed: [P-011-D4]

# Metrics
duration: 7h 33m
completed: 2026-08-01
---

# Phase 10 Plan 06: Working Groups surface (D-4)

**Working Groups top-level surface under Work — /working-groups list with client-side search, create/edit dialog (activity + manager + delegate chips), member management dialog distinct from the approver set, and destructive delete, all against the live Phase 9 WG API with a 5/5-green e2e suite**

## Performance

- **Duration:** 7h 33m
- **Started:** 2026-07-31T23:09:39Z
- **Completed:** 2026-08-01T06:42:42Z
- **Tasks:** 4
- **Files modified:** 10 (6 created, 4 modified)

## Accomplishments

- `/working-groups` route live under Work: Header (single h1 "Working Groups" + search + right-aligned accent "New working group" CTA) + Body card grid on `--card` surfaces; cards show WG name, linked activity (resolved via activities cache by `subproject_id`), manager name (resolved via org members), member count (per-card members query); client-side name search
- Locked UI-SPEC empty state: "No working groups yet" + "Working groups assign people to activities. Create one to start staffing work." + "New working group" CTA
- `WorkingGroupsApis` completed: list/get/members queries + create/update/delete/addMember/removeMember mutations — every mutation invalidates `["working-groups"]` (so card member counts + sidebar approval-stage derivation refresh) and toasts via sonner; delete surfaces backend guard errors verbatim (T-10-06-3)
- `WorkingGroupFormDialog` (create + edit in one): name/activity/manager required (comboboxes), delegate_ids multi-select via base-ui chips; edit pre-fills and PUTs preserving description/unit_ids; create sends `org_id` from the route-context profile — a client can only ever create WGs in its own org (T-10-06-1)
- `WorkingGroupMembersDialog`: members list with name + role badge (names joined client-side from org members — the WG member payload carries no user names), add via user+unit+role, remove via destructive AlertDialog; membership is a separate UI concern from the approver set (manager/delegate edit lives only in the form dialog, plan task 3)
- Delete flow: destructive AlertDialog confirm; backend "cannot delete working group with members" style guards surface verbatim via toast — no client-side cascade invented (T-10-06-3)
- **Rule 1 fix in the shared helper:** `api<T>` now returns `undefined` on 204 No Content instead of calling `res.json()` on an empty body (which throws). Every DELETE mutation in the app — working-groups, customers, activities, contracts, units — was rejecting after a successful server response; remove-member and WG delete could not round-trip without this fix
- e2e suite `web/e2e/working-groups.spec.ts` (5/5 green): empty state → create via dialog → card with name/manager/member count; search filters; edit (rename + delegate) persists after reload with the delegate verified server-side via psql; members add → role badge → remove → gone; delete → confirm → empty state returns

## Task Commits

Each task was committed atomically:

1. **Task 1: WG API layer — types + queries + 5 mutations** - `d91c727` (feat)
2. **Task 2: Working Groups page — shell + card grid + create/edit dialog** - `b4260d4` (feat)
3. **Task 3: WG members dialog — list/add/remove with role badges** - `6938539` (feat)
4. **Task 4: Working Groups e2e — 5 flows green + 204 api helper fix** - `7439411` (feat)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `web/src/routes/_authenticated/working-groups/index.tsx` - route: loader ensureQueryData(list) + leaf RouteError (P0-4) + pendingMs
- `web/src/routes/_authenticated/working-groups/-components/working-groups-page.tsx` - Header/Body shell, card grid, client-side search, locked empty state, dialog wiring
- `web/src/routes/_authenticated/working-groups/-components/working-group-form-dialog.tsx` - create/edit dialog: name/activity/manager + delegate chips; profile-sourced org_id on create
- `web/src/routes/_authenticated/working-groups/-components/working-group-members-dialog.tsx` - members list/add/remove with role badges + destructive confirm
- `web/src/routes/_authenticated/working-groups/-components/delete-working-group-dialog.tsx` - destructive confirm; backend guard errors verbatim
- `web/src/api/working-groups.ts` - completed WorkingGroupsApis (get/members queries + 5 mutations, `["working-groups"]` invalidation + toasts)
- `web/src/types/models.ts` - WorkingGroupMember, Create/UpdateWorkingGroupRequest, AddWorkingGroupMemberRequest mirroring Go JSON
- `web/src/lib/api.ts` - 204 No Content → `undefined` (Rule 1 fix; unblocks all deletes)
- `web/src/routeTree.gen.ts` - regenerated with the /working-groups route
- `web/e2e/working-groups.spec.ts` - 5 CRUD flows; seeds activity/unit without a WG for a real empty state

## Decisions Made

- **`subproject_id` kept, not renamed:** the plan text said add `WorkingGroup` with `activity_id`, but the live API JSON key is `subproject_id` (legacy field anchoring to activities, D-5). The 10-02 type already mirrors the Go payload exactly — verified against `working_group.go` and left untouched. The form dialog sends the picked activity id in `subproject_id`.
- **Create's org_id is profile-derived:** the backend Create handler *requires* `org_id` in the body (`invalid org_id` 400 otherwise) — the threat model's "client never sends org ids" is met in spirit: the dialog sources `profile.membership.organization_id` from the route context (the user's own org), never from user input. No client-side org-isolation guard invented; backend stays authoritative (T-10-06-1).
- **Dialog over expandable card for members (planner's discretion):** cards stay compact; member count still renders per card (members query), and a "Members" button opens the dialog.
- **Unit picker in the add-member form:** the backend AddMember handler requires `user_id` AND `unit_id` (`user_id and unit_id are required` 400 otherwise). The plan's action text listed only "user combobox + role select" — the unit field is backend-required, so it was added (Rule 3, blocking). Units come from `unitTreeQueryOpts` flattened.
- **Member roles = {member, lead}:** `wg_members.role` is an unconstrained VARCHAR; the only values in the codebase are `member` (seed 003, unit-detail-panel) and `lead` (wg_member_repository_test). No invented roles.
- **api<T> 204 handling in the shared helper:** minimal fix (`if (res.status === 204) return undefined`), zero behavior change for JSON responses; repairs every DELETE flow app-wide.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `api<T>` rejected every DELETE mutation on 204 No Content**
- **Found during:** Task 4 (e2e verification — members remove flow never closed the confirm dialog)
- **Issue:** The shared `api<T>` helper called `res.json()` after `!res.ok` passed; on a 204 the body is empty, so `res.json()` throws `Unexpected end of JSON input`. Every DELETE endpoint in the backend returns 204, so every `api<void>` DELETE mutation (working-groups, customers, activities, contracts, units) rejected after the server *succeeded* — the members remove and WG delete could not round-trip, and the confirm dialog stayed open (mutateAsync rejection skipped `setRemoveTarget(null)`).
- **Fix:** `if (res.status === 204) return undefined as T;` before the JSON parse.
- **Files modified:** web/src/lib/api.ts
- **Verification:** working-groups e2e members remove + delete flows green (5/5); full suite 41 pass.
- **Committed in:** 7439411 (Task 4 commit)

**2. [Rule 3 - Blocking] Add-member form missing the backend-required unit field**
- **Found during:** Task 3 (AddMember mutation wiring)
- **Issue:** The plan's add-member action listed "user combobox + role select", but the backend AddMember handler rejects any payload without `unit_id` (`user_id and unit_id are required`, 400) and `wg_members.unit_id` is NOT NULL with a units FK. Without a unit the mutation always failed.
- **Fix:** Added a unit combobox to the add-member form, sourced from the flattened unit tree (`unitTreeQueryOpts`), required before submit.
- **Files modified:** working-group-members-dialog.tsx
- **Verification:** e2e "members: add → listed with role badge" green (unit picked in the flow).
- **Committed in:** 6938539 (Task 3 commit)

**3. [Rule 3 - Blocking] e2e seed rows with NULL text columns 500 the list endpoints**
- **Found during:** Task 4 (create-flow failed: activity combobox empty, /activities 500)
- **Issue:** The spec's raw psql seeds inserted activities/units without `description`/`code`, leaving NULLs; the repo scans those columns into plain Go strings, and pgx rejects NULL → `failed to fetch activities` / `failed to get unit tree` (500). The existing `seedBaseEntities` helper has the same latent NULL-description issue but is only used by suites that never list activities.
- **Fix:** Seeded `description`/`code` as `''` per the documented Phase 08 psql convention (helpers.ts comment: pgx rejects NULL for *string scans).
- **Files modified:** web/e2e/working-groups.spec.ts
- **Verification:** activities + units endpoints return data for the seeded org; 5/5 e2e green.
- **Committed in:** 7439411 (Task 4 commit)

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All three are correctness requirements — without them the members/delete flows could not round-trip (Rule 1 fix repairs every delete in the app) and the create flow had no selectable activity/unit. No scope creep; no new dependencies.

## Issues Encountered

- **`git stash` misuse during verification (operational, self-inflicted):** while confirming the password-reset typecheck error was pre-existing I ran `git stash` on the main tree; the pop conflicted on `routeTree.gen.ts` (regenerated between stash and pop) and was left as `stash@{0}`. The stash contained only the two pre-existing dirty files (`hourglass-vault/.obsidian/workspace.json`, `migrate`) plus the regenerated route tree. I restored the two files via read-only `git show stash@{0}:<path>` (no stash mutation — the destructive-git prohibition forbids pop/drop). The stash entry is redundant debris and was left in place; no data lost, working tree matches pre-execution state.
- **Pre-existing typecheck rot:** the plan's `typecheck exits 0` criteria are technically unattainable at baseline — the same 6 errors from deferred-items §1 remain, all in out-of-plan files (api.test.ts, __root.tsx, _auth forms ×3, org-hierarchy unit-detail-panel). Zero errors in this plan's files (grep-verified).
- **Pre-existing e2e date rollover:** full suite = 41 passed + the 3 July-date rollover failures documented in deferred-items §2 (time-entries, expenses, error-boundary — seeds hard-code July, list views default to August). Not caused by this plan; the working-groups suite seeds its own org and passes 5/5.
- **Combobox popup overlay in e2e:** the delegate multi-select popup stays open for further picks and overlays the dialog footer — the e2e presses Escape before Save. Also, closed dialogs' combobox portals stay mounted (hidden), so e2e assertions are scoped to card/member-row surfaces to avoid strict-mode double matches.

## Known Stubs

None — all plan surfaces are wired to the live WG API. Card fallbacks ("Activity not found", "—" for missing manager names) are data-absence renders, not stubs: they only appear when a WG anchors to an activity outside the owned list or a manager left the org, and the backend remains authoritative.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries. Plan threats addressed:

- T-10-06-1 (member management mutates another org's WG): client sends no org ids except create, where org_id is profile-derived (the user's own org, never user input); no client-side org-isolation guard invented — backend org-scopes all endpoints and stays authoritative.
- T-10-06-2 (pickers accept non-org users): manager/delegate/member pickers source exclusively from `orgMembersQueryOpts` (`GET /organizations/members`, already exported from `api/units.ts` — identified and reused per task 1 acceptance); backend validates membership.
- T-10-06-3 (WG delete orphans approvals chain): backend guards surface verbatim via `toast.error(error.message)` in the delete mutation; no client-side cascade logic invented.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase complete:** 6/6 plans executed (10-01..10-06). All D-1..D-6 deliverables are live: sidebar pillars + role scoping, renamed Activities surface, locked page shell on all carried-over pages, Today landing, Approvals queue, and the Working Groups surface.
- **Ready for phase verification:** the UI-SPEC focal-point visual walk can now cover every Phase 10 surface through the locked Header+Body shell, including the Working Groups page (single accent CTA, muted card grid).
- **Deferred to P-008:** availability/validity warnings on Working Groups (explicitly out of scope — "no availability warnings" is a plan must_have).
- **Recommended follow-up (maintenance, not Phase 10):** the 6 pre-existing typecheck errors (deferred-items §1) and the July-date e2e seeds (deferred-items §2) — both documented for a future maintenance plan.

---
*Phase: 10-information-architecture-implementation*
*Completed: 2026-08-01*

## Self-Check: PASSED

- All 9 created/modified source files exist on disk; all 4 task commits verified in git log (d91c727, b4260d4, 6938539, 7439411)
- `bun run test`: 122/122 unit tests pass (16 files) — unchanged baseline, zero new failures
- `bunx playwright test e2e/working-groups.spec.ts`: 5/5 green
- Full e2e suite: 41 pass + 3 pre-existing July-date rollover failures (deferred-items §2 — time-entries, expenses, error-boundary), none caused by this plan
- `bun run lint`: 0 errors (138 pre-existing warnings, none in plan files)
- `bun run typecheck`: zero errors in all 9 plan files; the 6 pre-existing baseline errors remain in out-of-plan files (deferred-items §1)
