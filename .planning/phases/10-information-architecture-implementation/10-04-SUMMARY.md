---
phase: 10-information-architecture-implementation
plan: 04
subsystem: ui
tags: [react, tanstack-router, react-query, vitest, msw, playwright, today-landing, read-only-composition]

# Dependency graph
requires:
  - phase: 10-information-architecture-implementation (10-02)
    provides: deriveApprovalStages (WG manager_id/delegate_ids + org-role stage derivation), WorkingGroupsApis.workingGroupsQueryOpts
  - phase: 09-activity-ontology
    provides: entry DTOs carrying activity_id + activity_name, working-groups HTTP surface
  - phase: 08-pre-deployment-hardening-p0-audit-fixes
    provides: leaf-level errorComponent + router.invalidate() recovery convention, RouteError component
provides:
  - Today landing page at `/` (ADR-P-011 D-2 / ADR-P-004): read-only composition — "Waiting on you" (approvers) + "Your week" (own draft/submitted/rejected entries in current ISO week), never blank
  - Pending-query enabled gate keyed on deriveApprovalStages (no 403 spam for plain employees/HR)
  - Locked empty states ("You're all caught up" / "Welcome to Hourglass") both reachable and unit-tested
  - Updated landing assertions across e2e (auth, customers deep-link, error-boundary home link)
  - The `-components` directory convention for the landing route (today-page.tsx keeps the route lean)
affects: [10-05 Approvals (reviewTo path now referenced from Today), any future surface consuming the landing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Today landing as a lean file-route (component + errorComponent + pendingComponent) with all composition in `-components/today-page.tsx`"
    - "Pending-query gating via spread of queryOptions + enabled override tied to derived approval stages"
    - "Client-side ISO-week filter on the month query (date-fns startOfWeek/endOfWeek, string-compare on entry_date.slice(0,10)) — no new endpoint"
    - "Links to not-yet-created routes (/approvals) typed as ToPathOption (sidebar convention) so they compile against the pre-route routeTree"

key-files:
  created:
    - web/src/routes/_authenticated/-components/today-page.tsx
    - web/src/routes/_authenticated/-components/__tests__/today-page.test.tsx
  modified:
    - web/src/routes/_authenticated/index.tsx
    - web/e2e/auth.spec.ts
    - web/e2e/customers.spec.ts
    - web/e2e/error-boundary.spec.ts

key-decisions:
  - "In-Body error state for the locked copy 'We couldn't load Today. {reason}. Try again.' with router.invalidate() recovery — Today's queries are component-level (non-suspense), so query failures render in-component rather than tripping the route errorComponent (which stays registered as the leaf boundary)"
  - "ISO-week filter compares entry_date.slice(0,10) against startOfWeek/endOfWeek bounds computed by date-fns — string compare avoids the date-only-vs-RFC3339 timezone shift and matches the list-view convention"
  - "Preview values render .font-text for both hours (time entries) and amount (expenses) — UI-SPEC reserves .font-text for numeric hours/counts in queue rows"
  - "Header CTA + section accent link to /approvals typed as ToPathOption — the route lands in 10-05; typed Link to would not compile"
  - "Pending endpoints not called for employees/HR proven via msw request capture in unit tests (enabled gate), not just by inspection"

patterns-established:
  - "Today page never-blank ladder: skeleton (loading) → locked error state (query failure) → sections or locked empty states — every state renders inside Body"

requirements-completed: [P-004, P-011-D2]

# Metrics
duration: 17min
completed: 2026-07-31
---

# Phase 10 Plan 04: Today landing page — read-only composition, never blank (P-004 + D-2)

**The `/` landing is now the Today page — a read-only composition of "Waiting on you" (approval-stage gated) + "Your week" (current-ISO-week draft/submitted/rejected entries) with locked empty states for new users and all-caught-up users, replacing the `<Navigate to="/time-entries" />` redirect**

## Performance

- **Duration:** 17 min
- **Started:** 2026-07-31T21:39:10Z
- **Completed:** 2026-07-31T21:56:12Z
- **Tasks:** 3
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- TodayPage composes `<Header>` (Display 28px "Today" + right-aligned primary CTA "Review now"/"Log time") + `<Body>` (p-6, gap-6 section stack) exactly per UI-SPEC; the file route stays lean with `errorComponent: RouteError` + a pendingComponent rendering inside Body
- "Waiting on you" renders only when `deriveApprovalStages(profile, workingGroups).length > 0`; pending TE + expense queries are `enabled: isApprover` so plain employees and HR never fire `/time-entries/pending` or `/expenses/pending` (threat T-10-04-3) — proven by msw request capture in tests
- "Your week" reuses `timeEntriesForMonthQueryOpts` and filters client-side to the current ISO week (date-fns `startOfWeek`/`endOfWeek`, string-compare on the stored date part), statuses `draft|submitted|rejected` — approved excluded; `.font-text` numerics + `StatusBadge` unchanged (no seventh status color)
- Both locked empty states reachable: "You're all caught up" (has data, nothing pending/in-week) and "Welcome to Hourglass" (no data at all, proxy = empty month query + empty pending) with the "Log time" CTA — never blank
- In-Body error state renders "We couldn't load Today. {reason}. Try again." with `router.invalidate()` recovery (Phase 8 convention); skeleton loading states for both sections
- 6 new unit tests through the real routeTree harness (approver section, employee no-section + no-call proof, week filter, both empty states, HR never) — full suite 114/114 green
- e2e: auth.spec register/login land on `/` with Today heading; new smoke test proves `/time-entries` remains directly reachable; full suite 42/42 green (including the customers deep-link and error-boundary home-link tests that my landing change forced me to update)

## Task Commits

Each task was committed atomically:

1. **Task 1: TodayPage composition + route rewire** - `733adc4` (feat)
2. **Task 2: today-page composition tests (six cases)** - `35f4ee4` (test)
3. **Task 3: e2e landing assertions + fallout fixes** - `715dba3` (test)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `web/src/routes/_authenticated/-components/today-page.tsx` - `TodayPage`: Header/Body shell, stage-gated pending section (merged TE+expense preview, max 5 rows, count, "Review now" accent link), ISO-week "Your week" section, locked empty states, in-Body error + skeleton states
- `web/src/routes/_authenticated/index.tsx` - route now `component: TodayPage`, `errorComponent: RouteError`, pendingComponent inside Body; Navigate removed
- `web/src/routes/_authenticated/-components/__tests__/today-page.test.tsx` - 6 composition tests with msw request capture for the pending gate
- `web/e2e/auth.spec.ts` - `expectTodayLanding` helper (Today heading + never-blank union), register/login land on `/`, new /time-entries direct-reach smoke test
- `web/e2e/customers.spec.ts` - deep-link test post-login assertion now expects Today at `/` (was `/time-entries`)
- `web/e2e/error-boundary.spec.ts` - home-link test "Go to Today" now asserts the Today page at `/` (was `/time-entries`)

## Decisions Made

- **Error state placement:** Today's data queries are component-level (non-suspense `useQuery`), so query failures never trip the route `errorComponent` on their own. The locked copy ("We couldn't load Today. {reason}. Try again.") is rendered in-Body with a `router.invalidate()` "Try again" button; `RouteError` stays registered as the leaf-level boundary per convention.
- **Week-filter mechanism:** bounds computed with date-fns `startOfWeek`/`endOfWeek` (weekStartsOn: 1), membership tested by comparing `entry_date.slice(0, 10)` against the bound strings — immune to the date-only-vs-RFC3339 local-timezone shift and consistent with the existing list-view convention.
- **`/approvals` links typed as `ToPathOption`:** typed `<Link to="/approvals">` fails to compile because the route lands in Plan 10-05. The sidebar's `ToPathOption` (loose-string) convention carries the links until the route exists — same as the existing nav item.
- **Expense preview values:** expenses have no hours — their numeric value renders as `amount.toFixed(2)` in `.font-text` (UI-SPEC: `.font-text` reserved for numeric queue-row values; time entries render `hours`).
- **New-user proxy:** "no data at all" is proxied by empty month query + empty pending (per plan); a user with entries but nothing in the current week lands on "You're all caught up".

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] e2e landing fallout in customers.spec and error-boundary.spec**
- **Found during:** Task 3 (full e2e suite run)
- **Issue:** Two suites asserted the old landing contract my change replaced: customers.spec's deep-link test expected `/time-entries` + "No time entries in this period." after UI login, and error-boundary.spec's "home link navigates to the landing route" expected `toHaveURL(/\/time-entries/)` after clicking "Go to Today". Both failed 2/42 in the first full run (RESEARCH §2.7 predicted the landing-route change may alter error-boundary's setup).
- **Fix:** customers.spec deep-link asserts `waitForURL(/\/$/)` + Today heading; error-boundary.spec home-link test asserts URL `/` + Today heading + no alert + never-blank body union (`Waiting on you|Your week|You're all caught up|Welcome to Hourglass`).
- **Files modified:** web/e2e/customers.spec.ts, web/e2e/error-boundary.spec.ts
- **Verification:** full e2e suite 42/42 green (second run)
- **Committed in:** 715dba3 (Task 3 commit)

**2. [Rule 3 - Blocking] `bun run build` exits non-zero at baseline (pre-existing, out of scope)**
- **Found during:** Task 3 acceptance verification
- **Issue:** `bun run build` (`tsc -b && vite build`) fails on the 6 pre-existing typecheck errors documented in deferred-items.md (all in out-of-phase surfaces: `_auth` forms incl. the `useSearchParams` missing export that hard-fails rolldown, `__root` theme provider, api.test.ts, org-hierarchy unit-detail-panel). This is the same baseline rot 10-01/10-02/10-03 documented; the criterion "build exits 0" is unattainable until those are fixed, and they are explicitly out of Phase 10 scope.
- **Fix:** none in-scope. Verified the parts that ARE verifiable: routeTree.gen.ts has `AuthenticatedIndexRoute` at `path: '/'` (Today wired at `/`); tsc reports zero errors in all 6 plan files; `bunx vite build`'s only failure is the pre-existing `useSearchParams` missing export; e2e proves Today renders at `/` in a real Chromium.
- **Files modified:** none
- **Verification:** `bun run typecheck` → 6 errors, all in deferred-items.md list, none in plan files; full e2e 42/42
- **Committed in:** n/a (documented; tracked in deferred-items.md)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking/pre-existing-documented)
**Impact on plan:** The e2e fixes were directly required by the plan's own landing change (no scope creep). The build criterion is blocked by the pre-existing rot, consistent with prior Phase 10 plans.

## Issues Encountered

- **Base UI `nativeButton` dev warning:** `<Button render={<Link>}>` emits a dev-mode warning ("expected a native <button>") in unit tests. Matches the existing RouteError/sidebar pattern — warning only, no behavior impact, not suppressed to stay consistent with the codebase.
- **Full-suite test timing:** the approver test failed intermittently under parallel vitest workers (findBy* default 1000ms timeout vs ~1.7s render under load). Fixed with explicit 5000ms timeouts on async assertions; full suite then passed twice consecutively (114/114).
- **TanStack typed-route `to` constraint:** `/approvals` isn't in the route tree until 10-05, so typed `<Link to="/approvals">` failed TS2322 — resolved via the `ToPathOption` loose-typing convention (see Decisions).

## Known Stubs

None — Today's "Review now" CTA links to `/approvals`, which does not exist until Plan 10-05; clicking it today hits a router "not found" (expected pre-10-05 state, same as the sidebar's Approvals item). The pending preview caps at 5 rows by design (plan spec).

## Threat Flags

None — no new network endpoints, auth paths, file access, or schema changes. Threat model dispositions verified:

- T-10-04-1 (dashboard creep): zero charts/KPIs; no recharts import (grep-verified); acceptance asserts section composition only; CTA is a link, not a widget.
- T-10-04-2 (blank render): skeleton → error → sections/empty-state ladder means every load outcome renders content; both empty states unit-tested.
- T-10-04-3 (waiting-leak to non-approvers): section gated on `deriveApprovalStages` AND pending queries `enabled: isApprover`; msw capture proves zero pending calls for employee/HR; backend remains authoritative (UX scoping only).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 10-05 (Approvals):** Today already links to `/approvals` (header CTA + section accent link); 10-05 creates the route and the Today links go live without further changes. `deriveApprovalStages` remains the shared stage source; the pending-query gating pattern (spread + enabled override) carries over to the Approvals queue.
- **Blockers:** the 6 pre-existing typecheck/build errors in out-of-phase files remain (deferred-items.md) — they block `bun run build` for every Phase 10 plan; a future phase owns them.

---
*Phase: 10-information-architecture-implementation*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 6 created/modified files exist on disk (6/6 FOUND); all 3 task commits verified in git log (733adc4, 35f4ee4, 715dba3)
- `bun run test`: 114/114 unit tests pass (15 files) — twice consecutively
- `bunx playwright test`: 42/42 e2e pass (full suite incl. new /time-entries smoke test)
- `bun run lint`: 0 errors (126 pre-existing warnings, none in plan files)
- `bun run typecheck`: zero errors in all 6 plan files; 6 pre-existing out-of-scope errors remain (deferred-items.md) — `bun run build` blocked by the same baseline rot, documented in Deviations
- routeTree.gen.ts verified: `AuthenticatedIndexRoute` at `path: '/'` (Today wired at `/`)
