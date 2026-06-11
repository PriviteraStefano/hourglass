---
phase: 06-time-entries-expenses
plan: 05
subsystem: frontend-expenses
tags: [frontend, expenses, ui, api, types]
requires: [06-04]
provides: [ExpenseUI, ExpenseAPIQueries, ExpenseTypes]
affects: [sidebar]
tech-stack:
  added: []
  patterns:
    - "TypeScript barrel export with selective re-export to avoid type collisions"
    - "TanStack Query v5 mutationOptions with onSuccess for cache invalidation"
    - "shadcn Select onValueChange null-guard pattern for v5 compatibility"
key-files:
  created:
    - web/src/types/expense-types.ts
    - web/src/api/expenses.ts
    - web/src/routes/_authenticated/expenses/index.tsx
    - web/src/routes/_authenticated/expenses/-components/expense-calendar.tsx
    - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx
    - web/src/routes/_authenticated/expenses/-components/expense-detail.tsx
    - web/src/routes/_authenticated/expenses/-components/expense-row.tsx
    - web/src/routes/_authenticated/expenses/-components/expense-form.tsx
  modified:
    - web/src/types/index.ts
    - web/src/components/layout/sidebar.tsx
decisions: []
metrics:
  duration: 12m
  completed_date: "2026-06-11"
---

# Phase 6 Plan 5: Build expense frontend — types, API, route, components, sidebar

**One-liner:** Full expense frontend UI with calendar-based navigation, CRUD forms, receipt upload, and approval workflow using shared components — includes type definitions, API module, TanStack Router route, calendar+detail page layout, and inline create/edit form.

## Summary

Built the complete expense frontend from scratch: 8 new files and 2 modified files. The expense UI mirrors the time entries layout (MiniCalendar sidebar + day detail panel) with expense-specific fields (category selector with 9 options, amount input, conditional km_distance for mileage, receipt upload button). Types follow the D-23 contract. API module provides 3 query and 7 mutation options. Route uses Zod search validation with TanStack Router loaders. Sidebar Expenses link is now enabled.

## Tasks

| Task | Name | Type | Status |
|------|------|------|--------|
| 1 | Expense types + API module | auto | ✅ Done |
| 2 | Expense route + page layout + sidebar | auto | ✅ Done |
| 3 | Expense detail + row + form components | auto | ✅ Done |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] shadcn Select onValueChange null-guard**

- **Found during:** Task 3 (build verification)
- **Issue:** The installed version of shadcn `Select` expects `onValueChange` signature `(value: string | null, eventDetails) => void`, not `(value: string) => void`. This is a known breaking change in newer Radix versions. Directly passing `setState` functions caused TypeScript errors.
- **Fix:** Wrapped all `onValueChange` handlers with null-guard: `(v) => v !== null && setState(v)`
- **Files modified:** `expense-detail.tsx`, `expense-form.tsx`
- **Commit:** 60d837f

Also applied the same pattern to `expense-row.tsx` which already used the null-guard pattern correctly from the beginning.

### Type Collision Resolution

- `ExpenseCategory` type was already defined in `web/src/types/models.ts` (from a previous phase). Created `expense-types.ts` with the same type definition but omitted it from the barrel export in `types/index.ts` to avoid TypeScript `export *` ambiguous re-export errors. The type is importable from both `@/types/models` and `@/types/expense-types` directly.

## Verification

- `bun run build` succeeds — only pre-existing errors in unrelated files (auth tests, contracts, customers, projects tests, login/password-reset route types)
- Zero TypeScript errors in any expense frontend files

## Commit Log

| Hash | Message |
|------|---------|
| c1675da | feat(06-time-entries-expenses-05): add expense types and API module |
| 6d1cdf1 | feat(06-time-entries-expenses-05): add expense route, calendar, page layout, enable sidebar |
| 60d837f | feat(06-time-entries-expenses-05): add expense detail, row, and form components |

## Self-Check

- [x] `web/src/types/expense-types.ts` exists — Expense, CreateExpenseRequest, UpdateExpenseRequest, ExpenseApproval types with all fields per D-23
- [x] `web/src/api/expenses.ts` exists — All 10 query/mutation options (3 queries + 7 mutations including receipt upload)
- [x] `web/src/routes/_authenticated/expenses/index.tsx` exists — Route with Zod search validation, loaders for date and month expenses + projects
- [x] `web/src/routes/_authenticated/expenses/-components/expense-calendar.tsx` exists — Calendar with 6 status colors and legend
- [x] `web/src/routes/_authenticated/expenses/-components/expenses-page.tsx` exists — Layout with calendar + detail
- [x] `web/src/routes/_authenticated/expenses/-components/expense-detail.tsx` exists — Day detail with CRUD, approval buttons, approval history, empty state
- [x] `web/src/routes/_authenticated/expenses/-components/expense-row.tsx` exists — Category selector (9 options), amount, conditional km_distance, description, receipt upload, status badge
- [x] `web/src/routes/_authenticated/expenses/-components/expense-form.tsx` exists — Create/edit form with all fields
- [x] `web/src/components/layout/sidebar.tsx` modified — Expenses link enabled (disabled removed)
- [x] `bun run build` passes with zero new errors

## Known Stubs

None identified. All components are wired to real data sources (ExpensesApis queries/mutations, ProjectsApis for project selectors).

## Threat Flags

None identified. All threat items from the threat model (T-06-16 receipt upload, T-06-17 approval button visibility) are properly mitigated or accepted per plan.
