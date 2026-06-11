---
phase: 06-time-entries-expenses
plan: 04
phase_name: Time Entries + Expenses
plan_name: Frontend Time Entry - Flat Model Rewrite
subsystem: frontend
tags: [react, tanstack-router, tanstack-query, typescript, shadcn-ui, approval-workflow]
requires: [06-02, 06-03]
provides:
  - Flattened TimeEntry type (no items, flat request types)
  - Client-side calendar status computation
  - Flat entry-detail/entry-row components
  - Shared approval-buttons and approval-history components
affects:
  - Phase 6 Plan 05 (expense frontend can reuse approval components)
tech-stack:
  added: []
  patterns:
    - "Flat model: one TimeEntry per project per date, no items/sub-items"
    - "Client-side calendar: query month entries, compute status by priority"
    - "Role-aware approval visibility: manager/finance checks via UI component"
    - "AlertDialog for delete confirmation (replacing confirm())"
key-files:
  created:
    - web/src/components/approval/approval-buttons.tsx
    - web/src/components/approval/approval-history.tsx
  modified:
    - web/src/types/models.ts
    - web/src/types/api.ts
    - web/src/api/time-entries.ts
    - web/src/routes/_authenticated/time-entries/index.tsx
    - web/src/routes/_authenticated/time-entries/-components/mini-calendar.tsx
    - web/src/routes/_authenticated/time-entries/-components/entry-detail.tsx
    - web/src/routes/_authenticated/time-entries/-components/entry-row.tsx
    - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
    - web/src/routeTree.gen.ts
decisions:
  - "Approval mutations (approve/reject) use direct api() calls with queryClient invalidation in component, not mutationOpts from API module (plan directive R-07)"
  - "userRole in entry-detail hardcoded to 'employee' for now — will come from auth context in a future integration"
  - "ApprovalHistory receives empty array until approval data fetching is wired in a later plan"
metrics:
  duration: ~15 min
  tasks: 3
  files_created: 2
  files_modified: 8
  commits: 4
  completed_date: "2026-06-11"
---

# Phase 6 Plan 4: Frontend Time Entry — Flat Model Rewrite

**Flattened TimeEntry types, client-side calendar statuses, flat entry list components, and shared approval UI components.**

---

## Performance

- **Duration:** ~15 min
- **Started:** 2026-06-11
- **Completed:** 2026-06-11
- **Tasks:** 3
- **Files created:** 2
- **Files modified:** 8
- **Commits:** 4

---

## Accomplishments

1. **Task 1 — Types + API module (3 files):**
   - Flattened `TimeEntry` in `models.ts`: removed `items: TimeEntryItem[]`, added `project_id`, `subproject_id`, `wg_id`, `unit_id`, `hours`, `description`, `entry_date`, `current_approver_role`, `submitted_at`
   - Changed `organization_id` → `org_id`, `date` → `entry_date`
   - Removed `TimeEntryItem`, `TimeEntryDaySummary`, `TimeEntryMonthlySummary` interfaces
   - Added `ApprovalRecord` interface and `ExpenseCategory` type
   - Replaced `CreateTimeEntryRequest`/`UpdateTimeEntryRequest` with flat version (no `items` array)
   - Rewrote `time-entries.ts`: removed `timeEntriesMonthlySummaryQueryOpts` and `submitMonthMutationOpts`
   - Added `timeEntriesForMonthQueryOpts`, `approveTimeEntryMutationOpts`, `rejectTimeEntryMutationOpts`

2. **Task 2 — MiniCalendar + Route (2 files):**
   - Rewrote `mini-calendar.tsx` to use client-side status computation from `timeEntriesForMonthQueryOpts`
   - Priority-based status resolution: `approved > rejected > pending_finance > pending_manager > submitted > draft`
   - Added all 6 status colors/legend items (`pending_manager` green, `pending_finance` purple)
   - Updated route loader in `index.tsx` to use `timeEntriesForMonthQueryOpts` and `timeEntryQueryOpts`

3. **Task 3 — Entry components + Approval components (5 files):**
   - Rewrote `entry-detail.tsx` for flat model: renders `TimeEntry[]` directly (one row per entry)
   - Empty state with "Create Entry" button and descriptive text per UI-SPEC copywriting contract
   - Delete confirmation uses `AlertDialog` (replaced `confirm()`)
   - Approve/reject via direct `api()` calls with query invalidation
   - Rewrote `entry-row.tsx`: flat entry prop, project selector, hours input, description, `StatusBadge`, Submit/Delete actions
   - Created `approval-buttons.tsx`: role-aware visibility matrix (manager/finance), inline reject reason textarea (≥10 chars)
   - Created `approval-history.tsx`: immutable timeline with action icons, actor role, timestamp, and optional comment

---

## Task Commits

1. **Task 1: Types + API module** — `b41a9da` (feat)
   - `web/src/types/models.ts`, `web/src/types/api.ts`, `web/src/api/time-entries.ts`
2. **Task 2: MiniCalendar + Route** — `0a4a814` (feat)
   - `web/src/routes/_authenticated/time-entries/-components/mini-calendar.tsx`, `web/src/routes/_authenticated/time-entries/index.tsx`
3. **Task 3: Entry components + Approval components** — `f5542c9` (feat)
   - `web/src/routes/_authenticated/time-entries/-components/entry-detail.tsx`, `entry-row.tsx`
   - `web/src/components/approval/approval-buttons.tsx`, `approval-history.tsx`
4. **Route tree update** — `a676082` (chore)
   - `web/src/routeTree.gen.ts` (auto-generated)

---

## Deviations from Plan

None — plan executed exactly as written.

---

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| `userRole="employee"` hardcoded | `entry-detail.tsx` | ~136 | Needs auth context integration — approval buttons always hidden until wired to real user role |
| `approvals={[]}` empty array | `entry-detail.tsx` | ~147 | Approval data fetching not yet implemented — will be wired when approval endpoints expose records |
| `isPending={false}` | `entry-detail.tsx` | ~140 | Approve/reject use direct `api()` calls (not mutations) — pending state not wired |

---

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| `threat_flag: hardcoded_role` | `entry-detail.tsx` | `userRole="employee"` is a hardcoded placeholder. If this ships, approval buttons never render. Backend enforces role checks independently (T-06-13 mitigated at backend level). Must be wired to auth context before production. |

---

## Future Integration Points

- Approval buttons need `userRole` from auth context (Phase 6 Plan 05 or auth enhancement)
- `ApprovalHistory` needs actual `ApprovalRecord[]` data from API (requires approval history endpoint)
- Approve/reject pending states should use `useMutation` from shared `mutationOpts` rather than raw `api()` calls for consistency
- MiniCalendar could be extracted as a shared component for both time entries and expenses

---

## Self-Check: PASSED

- `bun run build` — no errors in modified files ✓
- `TimeEntry` has no `items` field ✓
- `TimeEntry` has `project_id`, `hours`, `description`, `current_approver_role`, `submitted_at` ✓
- `CreateTimeEntryRequest` is flat (no items array) ✓
- `time-entries.ts` exports `timeEntriesForMonthQueryOpts`, `approveTimeEntryMutationOpts`, `rejectTimeEntryMutationOpts` ✓
- `time-entries.ts` no longer exports `timeEntriesMonthlySummaryQueryOpts` or `submitMonthMutationOpts` ✓
- MiniCalendar renders all 6 statuses with correct colors ✓
- EntryDetail renders flat entry list (not items-based) ✓
- Delete uses AlertDialog ✓
- ApprovalButtons and ApprovalHistory components created ✓

---

*Phase: 06-time-entries-expenses*
*Plan: 04*
*Completed: 2026-06-11*
