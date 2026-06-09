---
phase: 01-authorization
plan: 02
subsystem: frontend-auth
tags: [react, tanstack-query, tanstack-router, org-switcher, auth]

# Dependency graph
requires:
  - phase: 01-01
    provides: Backend auth fixes (register cookies, password reset entropy)
provides:
  - Self-fetching OrgSwitcher with real membership data
  - Org switch mutation wired to full cache refresh
  - Landing page redirect from / to /time-entries
  - Corrected password reset API response type
affects:
  - 01-03 (remaining auth frontend tasks)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Self-contained component pattern (ProfileMenu analog): component fetches own data via useSuspenseQuery + useMutation
    - Full cache refresh on org switch: queryClient.clear() followed by queryClient.invalidateQueries
    - Minimal route redirect via Navigate component

key-files:
  created: []
  modified:
    - web/src/components/app/org-switcher.tsx
    - web/src/components/layout/sidebar.tsx
    - web/src/routes/_authenticated/index.tsx
    - web/src/api/auth.ts

key-decisions:
  - "OrgSwitcher follows ProfileMenu pattern: self-contained component fetching own data via useSuspenseQuery"
  - "queryClient.clear() then invalidateQueries(['auth', 'me']) on org switch for full data refresh per D-05"
  - "Simple Navigate redirect at / instead of Dashboard placeholder per D-07"

requirements-completed: [AUTH-03, AUTH-04, AUTH-05]

# Metrics
duration: 2 min
completed: 2026-06-09
---

# Phase 01: Authorization — Plan 02 Summary

**Real org memberships in OrgSwitcher with switch mutation, `/` redirect to `/time-entries`, password reset API type cleanup**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-09T23:34:15Z
- **Completed:** 2026-06-09T23:35:36Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- OrgSwitcher now self-fetches real memberships via `useSuspenseQuery(AuthApis.membershipsQueryOpts)` instead of receiving empty `organizations={[]}` prop (D-03)
- Org switch mutation wired to dropdown `onClick` with `queryClient.clear()` for full data refresh (D-04, D-05)
- Sidebar renders `<OrgSwitcher/>` without the empty prop
- `/` route now redirects to `/time-entries` via `Navigate` component (D-06, D-07)
- Password reset mutation response type corrected from `{message, code?}` to `{message}` (D-11)

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire OrgSwitcher with real memberships and org switch mutation** - `a1c5969` (feat)
2. **Task 2: Add `/` redirect to `/time-entries` and fix password reset API type** - `081e13c` (feat)

**Plan metadata:** (committed below)

## Files Created/Modified

- `web/src/components/app/org-switcher.tsx` — Removed `organizations` prop, added self-fetched memberships via `useSuspenseQuery`, wired org switch mutation with `queryClient.clear()`
- `web/src/components/layout/sidebar.tsx` — Changed `<OrgSwitcher organizations={[]}/>` to `<OrgSwitcher/>`
- `web/src/routes/_authenticated/index.tsx` — Replaced Dashboard placeholder with `<Navigate to="/time-entries" replace />`
- `web/src/api/auth.ts` — Removed `code?: string` from `requestPasswordResetMutationOpts` response type

## Decisions Made

- **Self-contained OrgSwitcher:** Follows the ProfileMenu pattern — the component fetches its own data via `useSuspenseQuery` and `useMutation`, rather than receiving props from the parent. This is more maintainable for a sidebar component.
- **Full cache clear on org switch:** After switching orgs, `queryClient.clear()` cleans all cached queries, then `invalidateQueries(['auth', 'me'])` refetches the profile for the new org context. This ensures no stale data from the previous org remains.
- **Simple Navigate redirect:** The landing page uses TanStack Router's `Navigate` component with `replace` to avoid adding a redirect to the browser history.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- **Pre-existing TypeScript build errors:** The project has ~60 pre-existing TypeScript errors in test files and other routes (test files calling `mutationFn` directly without proper `TanStack Query` typing, route type mismatches). These are unrelated to this plan's changes — no new errors were introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Frontend auth integration progressing — OrgSwitcher functional, landing redirect ready
- Ready for Plan 01-03: Remaining auth frontend tasks

## Self-Check: PASSED

- All 4 modified files exist and are correct: ✓
- OrgSwitcher has no `organizations` prop: ✓
- Sidebar has no hardcoded org array: ✓
- Password reset type has no `code?` field: ✓
- Index route renders Navigate to /time-entries: ✓
- No new TypeScript errors from modified files: ✓

---

*Phase: 01-authorization*
*Completed: 2026-06-09*
