---
phase: 07-exports
plan: 01
subsystem: api
tags: [go, excelize, xlsx, csv, export, count]
requires:
  - phase: 06-time-entries-expenses
    provides: TimeEntry + Expense domain models, service interfaces, PG repositories, HTTP handlers
provides:
  - CountTimesheets/CountExpenses/CountCombined service methods
  - CountTimesheets/CountExpenses SQL COUNT queries with roleFilter
  - XLSX generation via excelize/v2 for timesheets, expenses, combined (multi-sheet)
  - Format param (?format=csv|xlsx) dispatch on all 3 export endpoints
  - 3 new count endpoints (GET /exports/{type}/count) with auth middleware
  - Filter query param helpers (?project_id=, ?user_id=)
  - Rewritten handler tests with proper mock-service assertions (7 subtests)
affects:
  - 07-02 (Frontend export UI — count endpoints for empty-state pre-check, XLSX download)

tech-stack:
  added:
    - github.com/xuri/excelize/v2 v2.11.0
  patterns:
    - XLSX generation with bold headers, auto-sized columns, multi-sheet workbooks
    - Format-based dispatch (CSV vs XLSX) via ?format= query param
    - Count endpoint returning {data:{count:N}} JSON envelope
    - Filter param extraction via queryProjectID/queryUserID helpers

key-files:
  created: []
  modified:
    - internal/core/ports/export_repository.go
    - internal/adapters/secondary/postgres/export_repository.go
    - internal/core/services/export/export.go
    - internal/core/services/export/export_test.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/primary/http/export.go
    - internal/adapters/primary/http/export_test.go
    - cmd/server/main.go
    - go.mod
    - go.sum

key-decisions:
  - "excelize/v2 v2.11.0 for server-side XLSX generation per D-16/D-18"
  - "xlsxSheet struct type for typed multi-sheet workbook construction"
  - "Separate writeTimesheetsXLSX/writeExpensesXLSX/writeCombinedXLSX helpers for clean format dispatch"
  - "Combined XLSX produces two sheets (Timesheets + Expenses) per D-19"
  - "queryProjectID/queryUserID helpers for filter param extraction (actual SQL filtering deferred to future optimization per D-27)"
  - "CSV remains default format; xlsx selected via ?format=xlsx per D-17"
  - "Handler tests rewritten from panic-recovery pattern to proper mock-service assertions per plan requirement"

requirements-completed:
  - EXPT-01
  - EXPT-02
  - EXPT-03
  - EXPT-05
  - EXPT-06

duration: 3 min
completed: 2026-07-08
---

# Phase 7: Exports — Plan 01 Summary

**Backend export extensions: count endpoints, XLSX generation via excelize, format param switching, filter param helpers, and auth-gated count route wiring**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-08T21:39:50+02:00
- **Completed:** 2026-07-08T21:42:56+02:00
- **Tasks:** 3
- **Files modified:** 10 (8 source files, 2 dependency files)

## Accomplishments

- Added `CountTimesheets`/`CountExpenses` to `ExportRepository` port interface (2 new methods)
- Added `CountTimesheets`/`CountExpenses` SQL COUNT(*) queries to PG repository with `roleFilter` support
- Added `CountTimesheets`/`CountExpenses`/`CountCombined` service methods (thin delegation + sum)
- Added `CountTimesheets`/`CountExpenses` mock methods to `MockExportRepo`
- Added 3 service-level count tests (`TestService_CountTimesheets`, `CountExpenses`, `CountCombined`)
- Added `writeXLSX` core method using excelize/v2 with bold headers, auto-sized columns, multi-sheet
- Added `writeTimesheetsXLSX`/`writeExpensesXLSX`/`writeCombinedXLSX` (two sheets for combined)
- Added format param dispatch (`?format=csv|xlsx`) to each export endpoint
- Added `queryProjectID`/`queryUserID` helper functions for filter params
- Added 3 count handler methods (`CountTimesheets`/`CountExpenses`/`CountCombined`)
- Wired 3 new auth-gated routes in `cmd/server/main.go`
- Rewrote handler tests: 7 proper assertions (3 CSV success, 3 count success, 1 missing auth)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add CountTimesheets/CountExpenses to port, repo, service, mock, tests** - `50e2390` (feat)
2. **Task 2: Add XLSX generation, format param switching, and filter params** - `9838d4e` (feat)
3. **Task 3: Add count handler methods, wire count routes, rewrite handler tests** - `b690dea` (feat)

## Files Created/Modified

- `internal/core/ports/export_repository.go` — Added CountTimesheets, CountExpenses to interface
- `internal/adapters/secondary/postgres/export_repository.go` — Added CountTimesheets, CountExpenses SQL queries
- `internal/core/services/export/export.go` — Added CountTimesheets, CountExpenses, CountCombined service methods
- `internal/core/services/export/export_test.go` — Added 3 count test functions
- `internal/core/services/testdata/mocks.go` — Added CountTimesheets, CountExpenses mock methods
- `internal/adapters/primary/http/export.go` — Added writeXLSX, write*XLSX helpers, format dispatch, count handlers, filter param helpers, CSV streaming documentation
- `internal/adapters/primary/http/export_test.go` — Rewritten: 7 subtests with mock service
- `cmd/server/main.go` — Added 3 count routes with auth middleware
- `go.mod` / `go.sum` — Added excelize/v2 v2.11.0

## Decisions Made

- **excelize/v2 v2.11.0** for server-side XLSX generation — industry standard Go XLSX library (D-16)
- **Separate XLSX helpers per endpoint type** — cleaner than one generic helper for format dispatch
- **Combined XLSX = two sheets** — Timesheets sheet + Expenses sheet per D-19
- **Filter param extraction via helpers** — `project_id`/`user_id` extracted from query params; actual SQL filtering deferred as future optimization
- **CSV default** — `?format=xlsx` selects XLSX; CSV is default when unspecified (D-17)
- **Handler tests use mock service** — replaces panic-recovery pattern with proper assertions

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Backend export extensions complete: count endpoints, XLSX generation, format param, filter param helpers
- Ready for Plan 07-02: Frontend export UI (export form, download hook, export tabs, sidebar nav)

## Self-Check: PASSED

- All 8 modified files verified on disk: ✓
- All 5 git commits exist: ✓
- SUMMARY.md exists and is valid: ✓
- `go build ./...` passes: ✓
- `go test -count=1 -run 'Export' ./internal/...` — all export tests pass: ✓
- `go mod verify` passes: ✓
