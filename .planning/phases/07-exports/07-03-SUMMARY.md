---
phase: 07-exports
plan: 03
subsystem: ui
tags: [react, tabs, @base-ui, typescript, tanstack-router]

# Dependency graph
requires:
  - phase: 07-02
    provides: ExportForm shared component, useDownload hook, exports API module
provides:
  - Export tabs on time-entries and expenses pages with [List] [Calendar] [Export] tabs
  - Export tab renders ExportForm with type-specific configuration (timesheets / expenses)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Tabbed page layout using shadcn Tabs wrapping @base-ui/react/tabs primitives

key-files:
  created: []
  modified:
    - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
    - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx

key-decisions:
  - "Followed PATTERNS.md tab layout exactly: Tabs defaultValue='list', List/Calendar/Export triggers"
  - "List tab is a placeholder (no list view component exists yet on either page)"
  - "Calendar content preserves the same flex layout as before (no visual regression)"

requirements-completed: [EXPT-01, EXPT-02]

# Metrics
duration: ~1 min
completed: 2026-07-08
---

# Phase 7 Plan 3: Export Tabs on Time Entries & Expenses Pages

**Export tabs added to time entries and expenses pages — [List] [Calendar] [Export] using shadcn Tabs with @base-ui/react primitives**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-07-08T21:52:04Z
- **Completed:** 2026-07-08T21:52:34Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Time entries page now has [List] [Calendar] [Export] tabs with ExportForm rendering `type="timesheets"`
- Expenses page now has [List] [Calendar] [Export] tabs with ExportForm rendering `type="expenses"`
- Existing MiniCalendar/EntryDetail and ExpenseCalendar/ExpenseDetail components preserved under Calendar tab
- Default tab is "list" on both pages per UI-SPEC
- No regressions in existing calendar functionality

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Export tab to time entries page** - `8f926e4` (feat)
2. **Task 2: Add Export tab to expenses page** - `1cc8e1f` (feat)

**Plan metadata:** `30e2864` (docs: complete plan)

## Files Created/Modified
- `web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx` — Tabbed layout with Tabs component, ExportForm with `type="timesheets"`, MiniCalendar+EntryDetail under Calendar tab
- `web/src/routes/_authenticated/expenses/-components/expenses-page.tsx` — Tabbed layout with Tabs component, ExportForm with `type="expenses"`, ExpenseCalendar+ExpenseDetail under Calendar tab

## Decisions Made
- Followed PATTERNS.md exactly for tab layout: `Tabs defaultValue="list"` with `TabsList` containing three triggers
- List tab left as empty placeholder — no list view component exists yet on either page (consistent with plan instruction)
- Calendar tab preserves original `flex` layout to avoid visual regressions
- Both pages use identical pattern for consistency

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing TypeScript build errors in unrelated files (contracts, org-hierarchy, projects) — none affecting this plan's changes

## Next Phase Readiness
- Export tabs are ready for use — both pages now expose Export as a tab alongside List and Calendar
- ExportForm component (built in 07-02) renders correctly in both contexts
- Phase complete — no remaining plans in Phase 7

## Self-Check: PASSED

- `time-entries-page.tsx` exists on disk with 28 lines: ✓
- `expenses-page.tsx` exists on disk with 28 lines: ✓
- `07-03-SUMMARY.md` exists on disk: ✓
- `8f926e4` — Task 1 commit exists: ✓
- `1cc8e1f` — Task 2 commit exists: ✓
- No TypeScript errors from our files: ✓

---

*Phase: 07-exports*
*Completed: 2026-07-08*
