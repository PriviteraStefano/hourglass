---
phase: 02-org-hierarchy
plan: 03
subsystem: ui-api
tags: react, tanstack-query, zod, lucide-react

# Dependency graph
requires:
  - phase: 02-org-hierarchy
    provides: ORG-01 component structure (unit-detail-panel, MemberRow, types/api infra)
provides:
  - UpdateUnitMemberRequest type and schema
  - updateUnitMemberMutationOpts for PUT /units/:id/members/:membershipId
  - "Make Primary" ghost button on non-primary member rows
  - SubtreeMembersSection and SubtreeGroup recursive expandable components
affects: [verification, testing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mutation opts follow addUnitMember/removeUnitMember pattern with dual cache invalidation"
    - "Recursive component pattern for tree traversal (SubtreeGroup → SubtreeGroup)"
    - "Lazy-loading sub-unit members on expand via enabled: expanded"

key-files:
  created: []
  modified:
    - web/src/types/unit.ts
    - web/src/api/units.ts
    - web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx

key-decisions:
  - "SubtreeGroup lazily fetches members via `unitMembersQueryOpts` with `enabled: expanded` to avoid waterfall"
  - "Make Primary button uses opacity-0 group-hover:opacity-100 pattern matching existing remove button"
  - "Primary badge and Make Primary button are mutually exclusive in a flex container"

patterns-established:
  - "Cache invalidation: member mutations always invalidate both ['units', 'members', unitId] and ['units', 'tree']"
  - "SubtreeGroup follows recursive depth-indent convention with marginLeft: depth * 12px"

requirements-completed: [ORG-01, ORG-03]

# Metrics
duration: 12 min
completed: 2026-06-10
---

# Phase 02: Org Hierarchy Plan 03 Summary

**Frontend type, API mutation, and UI for "Make Primary" unit member designation and expandable sub-unit member browsing in the side panel**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-10T21:51:42Z
- **Completed:** 2026-06-10T22:03:36Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added `UpdateUnitMemberRequestSchema` and `UpdateUnitMemberRequest` type to `types/unit.ts`
- Added `updateUnitMemberMutationOpts` to `api/units.ts` targeting `PUT /units/{id}/members/{membershipId}`
- Added `onMakePrimary` prop to `MemberRow` rendering "Primary" badge for primary members and a ghost "Make Primary" button on hover for non-primary
- Wired `handleMakePrimary` mutation with toast notifications in `UnitDetailPanel`
- Created `SubtreeMembersSection` component — renders expandable sub-unit groups when a unit has children
- Created `SubtreeGroup` recursive component — lazy-loads members on expand, exposes recursive depth-nested groups
- All mutations invalidate both `['units', 'members', unitId]` and `['units', 'tree']` query caches

## Task Commits

Each task was committed atomically:

1. **Task 1: API layer — UpdateUnitMemberRequest type + mutation opts** - `bde1465` (feat)
2. **Task 2: "Make Primary" button + SubtreeMembersSection** - `8be63c1` (feat)

## Files Created/Modified
- `web/src/types/unit.ts` — Added `UpdateUnitMemberRequestSchema` and `UpdateUnitMemberRequest` type (6 new lines)
- `web/src/api/units.ts` — Added `updateUnitMemberMutationOpts` following existing mutation pattern (14 new lines)
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` — MemberRow gains onMakePrimary prop, "Make Primary" ghost button, SubtreeMembersSection and SubtreeGroup components, wired mutation and handler (130 new lines, 16 modified)

## Decisions Made
- SubtreeGroup uses lazy member loading (`enabled: expanded`) to avoid waterfall fetches at render time
- Make Primary button follows the existing hover-visibility pattern (`opacity-0 group-hover:opacity-100`) matching the remove button
- Primary badge and Make Primary button placed in a shared flex container for consistent layout
- Server-side recursive rendering in SubtreeGroup uses depth-based indentation (`marginLeft: ${depth * 12}px`)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing TypeScript error at `unit-detail-panel.tsx:154` (`m.user_id` type narrowing from `string | null` to `string` in `handleAdd`) — not caused by this plan's changes, logged as deferred item.

## Next Phase Readiness
- D-01 (Make Primary button) and D-15/D-16 (Subtree Members) UI complete
- Ready for verification and testing of the org hierarchy frontend features
- Next plan should focus on testing or verification

## Self-Check: PASSED

- ✅ All 3 source files exist on disk
- ✅ All 3 commits present in git history
- ✅ Zero new errors introduced by this plan (only pre-existing error at unit-detail-panel.tsx:154)
- ✅ SUMMARY.md created and committed

---

*Phase: 02-org-hierarchy*
*Completed: 2026-06-10*
