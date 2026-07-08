---
phase: 07-exports
plan: 02
subsystem: frontend
tags: [react, tanstack-query, download, blob, date-range, export-form, sidebar]

# Dependency graph
requires:
  - phase: 07-01
    provides: Backend export endpoints (count, format, filters)
  - phase: 06-04
    provides: TimeEntriesApis pattern, queryOptions, mutationOptions convention
provides:
  - useDownload() hook for fetch+blob file downloads with AbortController timeout
  - ExportApis namespace with count query options and URL construction helpers
  - Shared ExportForm component with date range picker, preset periods, format selector
  - Rewritten combined exports page using ExportForm
  - Exports nav item in Tracking section of sidebar
affects:
  - 07-03 (in-page export tabs on time-entries and expenses pages)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Event-driven one-shot count query via queryClient.fetchQuery()
    - Reusable ExportForm component with type prop for timesheets/expenses/combined
    - useDownload hook isolates file-download mechanics from component logic

key-files:
  created:
    - web/src/lib/use-download.ts
    - web/src/components/exports/export-form.tsx
  modified:
    - web/src/api/exports.ts
    - web/src/routes/_authenticated/exports/-components/exports-page.tsx
    - web/src/components/layout/sidebar.tsx

key-decisions:
  - "useDownload hook handles fetch+blob flow with AbortController 60s timeout, Content-Type error detection, and Content-Disposition filename parsing"
  - "ExportForm uses queryClient.fetchQuery() for one-shot count pre-check before download"
  - "ExportForm wraps count check + download in toast.promise() for consolidated feedback"
  - "Sidebar Exports nav item moved from Settings group to Tracking group"

requirements-completed:
  - EXPT-01
  - EXPT-02
  - EXPT-03
  - EXPT-04
  - EXPT-05

# Metrics
duration: 3 min
completed: 2026-07-08
---

# Phase 7 Plan 2: Frontend Export Infrastructure Summary

**Shared download hook, export API module, reusable ExportForm component, combined export page rewrite, and sidebar navigation — all 3 tasks committed atomically**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-08T21:47:29Z
- **Completed:** 2026-07-08T21:49:48Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Created `useDownload()` hook in `web/src/lib/` — fetch+blob with `AbortController` 60s timeout, `Content-Type` error detection, `Content-Disposition` filename parsing, `isPending` state
- Rewrote `web/src/api/exports.ts` — `exportCountQueryOpts()` with TanStack Query `queryOptions` pattern, `getExportUrl()` with format param and optional project/user filter params, `ExportApis` namespace
- Created shared `ExportForm` component — props for type/showUserFilter/showProjectFilter, preset period buttons (This Month, Last Month, This Quarter, This Year), calendar popover date range picker via `react-day-picker` range mode, segmented [CSV] [XLSX] format selector, client-side validation (from < to, max 1 year), count pre-check via `queryClient.fetchQuery()`, `toast.promise()` feedback
- Rewrote combined export page — clean heading with `FileDown` icon, subtitle, renders `<ExportForm type="combined" showUserFilter showProjectFilter />`
- Moved Exports nav item from Settings group to Tracking section (after Approvals) in sidebar

## Task Commits

Each task was committed atomically:

1. **Task 1: Create useDownload hook and rewrite api/exports.ts** - `952ddda` (feat)
2. **Task 2: Create shared ExportForm component** - `f290825` (feat)
3. **Task 3: Rewrite exports page and move sidebar nav item** - `7a044a6` (feat)

**Plan metadata:** (pending final commit)

## Files Created/Modified

- `web/src/lib/use-download.ts` - `useDownload()` hook with fetch+blob, AbortController, Content-Disposition parsing
- `web/src/api/exports.ts` - `exportCountQueryOpts()`, `getExportUrl()`, `ExportApis` namespace
- `web/src/components/exports/export-form.tsx` - Shared `ExportForm` with date range, presets, format selector, download flow
- `web/src/routes/_authenticated/exports/-components/exports-page.tsx` - Rewritten to use `ExportForm type="combined"`
- `web/src/components/layout/sidebar.tsx` - Exports moved from settingsItems to navItems (Tracking section)

## Decisions Made

- **useDownload hook handles error toasts internally** — The hook catches errors and shows `toast.error()` inline, so components don't need try/catch around download calls
- **One-shot count query via queryClient.fetchQuery()** — The count pre-check is event-driven (button click), not mount-driven, so `queryClient.fetchQuery()` is cleaner than `useQuery` with `enabled: false`
- **ExportForm wraps entire flow in toast.promise()** — Consolidated toast feedback for the count check → download pipeline, showing loading/success/error states

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Changed `getExportUrl()` signature to include required `format` parameter, which temporarily broke the old exports-page.tsx call. Fixed by updating the old call to include `'csv'` format until Task 3 rewrote the entire file.

## Next Phase Readiness

- Frontend export infrastructure complete — `useDownload`, `ExportApis`, `ExportForm` ready for use
- Ready for Plan 03: In-page export tabs on time-entries and expenses pages

## Self-Check: PASSED

- All 5 files exist on disk: ✓
- All 3 git commits for plan 07-02 exist: ✓
- No TypeScript errors from our new/modified component files: ✓
- Build errors are pre-existing (contracts, org-hierarchy, projects, import.meta.env type)

---

*Phase: 07-exports*
*Completed: 2026-07-08*
