---
phase: 02-org-hierarchy
plan: 02
subsystem: ui
tags: [react, zustand, reactflow, org-hierarchy, reparent]
requires:
  - phase: 02-org-hierarchy
    provides: Zustand store with OrgHierarchyState/OrgHierarchyActions
provides:
  - Clean reparent flow using dedicated reparentUnitMutationOpts
  - Removal of dead pendingEdgeConnect state and all consumer references
affects: []

tech-stack:
  added: []
  patterns: []
key-files:
  created: []
  modified:
    - web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx
    - web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx
key-decisions:
  - "ReparentConfirmDialog now uses reparentUnitMutationOpts instead of updateUnitMutationOpts — sends only {parent_unit_id} instead of full unit body"
patterns-established: []
requirements-completed: [ORG-04]

duration: ~1 min
completed: 2026-06-10
---

# Phase 2 Plan 2: Clean up reparent flow — switch to dedicated mutation, remove dead pendingEdgeConnect state

**Dedicated reparent mutation with clean contract, dead state removal across all consumers**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-06-10T21:49:29Z
- **Completed:** 2026-06-10T21:49:40Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Switched `ReparentConfirmDialog` from `updateUnitMutationOpts` (sends full unit body) to `reparentUnitMutationOpts` (sends only `{parent_unit_id}`)
- Removed `pendingEdgeConnect` dead state: no longer read from store in dialog, no longer set in `onConnect` callback
- Cleaner API contract for reparent operation — only the parent change is sent, not the full unit payload

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove pendingEdgeConnect from Zustand store** — already committed in prior wave (`e5d344e`)
2. **Task 2: Switch ReparentConfirmDialog to reparentUnitMutationOpts + clean up page consumer** — `358b4dd` (refactor)

**Plan metadata:** `358b4dd` (refactor commit includes Task 2 + completes the plan)

_Note: Task 1 (store cleanup) was completed in a prior wave. Task 2 executed in this wave._

## Files Created/Modified
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx` — Switched to `reparentUnitMutationOpts`, removed `pendingEdgeConnect`/`setPendingEdgeConnect` reads and `setPendingEdgeConnect(null)` close handler; mutation now sends `{id, parent_unit_id}` only
- `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` — Removed `setPendingEdgeConnect` store destructure, removed `setPendingEdgeConnect` call in `onConnect`, cleaned dependency array

## Decisions Made
- Used the existing `reparentUnitMutationOpts` (line 79 of `units.ts`) instead of the heavyweight `updateUnitMutationOpts` — the dedicated mutation already existed, just needed wiring

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- The Zustand store (`org-hierarchy-context.tsx`) had already been cleaned of `pendingEdgeConnect` in a prior commit (`e5d344e`). Task 1 was therefore a no-op for this execution wave.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Reparent flow cleansed of dead state. Ready for Plan 03 (frontend features).

## Self-Check: PASSED

- [x] SUMMARY.md exists on disk
- [x] Task commit `358b4dd` exists in git log
- [x] No `pendingEdgeConnect` references remain anywhere in codebase
- [x] ReparentConfirmDialog imports `reparentUnitMutationOpts` (not `updateUnitMutationOpts`)
- [x] ReparentConfirmDialog sends `{id, parent_unit_id}` only (verified by reading dialog code)
- [x] OrgHierarchyPage no longer references `setPendingEdgeConnect`
- [x] Plan metadata commit `c598ae7` exists

---
*Phase: 02-org-hierarchy*
*Completed: 2026-06-10*
