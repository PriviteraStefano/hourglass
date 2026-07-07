# Phase 7: Exports - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Downloadable CSV/XLSX exports for timesheets, expenses, and combined reports with date range filtering, format selection, role-scoped data, and auth requirement.

**Backend already fully built** — ExportRepository, ExportService, ExportHandler with 3 endpoints (`GET /exports/timesheets`, `/exports/expenses`, `/exports/combined`), CSV download with date range filtering (from/to query params, defaults to current month), role-based data scoping (employee sees own, manager sees own+managed, finance/admin sees all), routes wired in `cmd/server/main.go`.

**Backend work needed:** Add XLSX generation via excelize, add project/user filter query params, add count endpoints for empty-state pre-check, add `?format=csv|xlsx` param, stream CSV rows for large exports.

**Frontend work needed:** Create shared ExportForm component, add export tabs to time-entries and expenses pages, create combined export at `/exports` route, add sidebar nav item, add `useDownload` hook, add API module.

### Requirements (from REQUIREMENTS.md)
- **EXPT-01:** Timesheet export (CSV/Excel) with date range filter
- **EXPT-02:** Expense export (CSV/Excel) with date range filter
- **EXPT-03:** Combined export with both time + expense data
- **EXPT-04:** Download as file
- **EXPT-05:** Empty export shows friendly message
- **EXPT-06:** Auth required — user-scoped data only

</domain>

<decisions>
## Implementation Decisions

### Download Approach
- **D-01:** **Fetch+blob** — Use fetch to get CSV/XLSX, create blob URL, programmatically trigger download. Not a direct `<a>` link.
- **D-02:** **Shared `useDownload` hook** — Download utility goes in `web/src/lib/` as a reusable hook. Not inline in the exports API module.
- **D-03:** **Backend-driven filename** — Use Content-Disposition header from backend. No frontend filename construction.
- **D-04:** **Raw CSV response** — Keep backend returning `text/csv` directly. Not wrapping in JSON envelope. Check `Content-Type` to differentiate success vs error.
- **D-05:** **Toast promise** — Use sonner `toast.promise()` for error/loading/success feedback.
- **D-06:** **Button spinner + disabled state** — Download button shows spinner + "Downloading..." and is disabled while request is in flight.
- **D-07:** **No progress bar** — Fetch without XHR. Toast states (loading → success/error) are sufficient for CSV.
- **D-08:** **Button click trigger** — User fills form, clicks Download. Not auto-triggered on form change.
- **D-09:** **No preview** — Download triggers immediately. No preview-first flow.
- **D-10:** **api.ts auth retry** — Reuse the existing cookie refresh + retry pattern from `web/src/lib/api.ts` for 401 handling.
- **D-11:** **Client-side validation** — Validate `from < to` on frontend before sending request.
- **D-12:** **1 year max range** — Block download if date range exceeds 1 year. Prevents huge exports.
- **D-13:** **Current month defaults** — From = 1st of current month, To = end of current month pre-filled on load.
- **D-14:** **60 second timeout** — AbortController with 60s timeout on download fetch.
- **D-15:** **Shared ExportForm component** — Single export form component accepting `type` ('timesheets'|'expenses'|'combined') and optional user filter props.

### Excel Support
- **D-16:** **Server-side XLSX** — Generate .xlsx on the backend using `excelize` v2 Go library. Not client-side conversion.
- **D-17:** **`?format=csv|xlsx` query param** — Same endpoints, format selected via query param. Default is CSV.
- **D-18:** **Formatted XLSX** — Bold headers, auto-sized columns, basic formatting.
- **D-19:** **Two sheets for combined XLSX** — Sheet 1 = timesheets, Sheet 2 = expenses. CSV combined remains single merged file with EntryType column.
- **D-20:** **Same .xlsx filename pattern** — `{type}_{from}_{to}.xlsx` via Content-Disposition.

### Export Page Layout
- **D-21:** **In-page export tabs** — [List] [Calendar] [Export] tabs on time-entries and expenses pages. Export replaces the content area entirely (no slide-in panel).
- **D-22:** **Separate `/exports` route** for combined export — standalone page with full form including user filter for managers/finance.
- **D-23:** **Sidebar Exports nav item** in Tracking section with Download icon.

### Date Range UX
- **D-24:** **Calendar date picker** — Use react-day-picker (already a dependency). Not plain text inputs.
- **D-25:** **Preset periods** — "This Month", "Last Month", "This Quarter", "This Year" buttons that auto-fill the date range.
- **D-26:** **Pre-filled to current month** — Calendar opens with current month selected.

### Backend Extensions
- **D-27:** **`?project_id=&user_id=` query params** on export endpoints — matches existing API conventions.
- **D-28:** **Count endpoint per type** — `GET /exports/timesheets/count`, `/exports/expenses/count`, `/exports/combined/count`. Returns `{ data: { count: N } }`.
- **D-29:** **Stream CSV rows** — Write CSV rows one by one with csv.Writer flush to reduce memory for large exports.

### Empty State
- **D-30:** **Block download + toast** — Call count endpoint first. If count = 0, show toast "No data to export" and don't trigger download.
- **D-31:** **Generic error messages** — Simple "Export failed. Please try again." for all error types.

### Mobile
- **D-32:** **Scrollable form** — Same layout stacks vertically on mobile. Calendar picker becomes modal popover. Presets become horizontal scroll.

### Format Selector
- **D-33:** **Segmented control** — [CSV] [XLSX] button group. Not dropdown or radio buttons.

### the agent's Discretion
- Sidebar icon selection (Download icon suggested)
- Exact export form layout and field ordering within the shared component
- Export tab bar implementation details (shadcn Tabs component)
- Calendar popover implementation (react-day-picker with Popover)
- Toast promise configuration (success/error messages)
- Excelize version pinning in go.mod
- Test file locations and specific test cases within existing patterns

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project context
- `.planning/PROJECT.md` — Project overview, key decisions, constraints
- `.planning/REQUIREMENTS.md` — Requirements (EXPT-01 through EXPT-06)
- `.planning/ROADMAP.md` — Phase definitions, dependency graph, key behaviors and edge cases
- `.planning/STATE.md` — Current phase state and session info

### Backend: Exports (existing — adapt)
- `internal/adapters/primary/http/export.go` — HTTP handler (3 endpoints, CSV generation, Content-Disposition, date range parsing)
- `internal/adapters/primary/http/export_test.go` — Handler tests (need update for param changes)
- `internal/core/services/export/export.go` — Service layer (thin, delegates to repo — needs format param, XLSX generation)
- `internal/core/services/export/export_test.go` — Service tests (need update)
- `internal/core/ports/export_repository.go` — ExportRow struct, ExportRepository interface (Timesheets, Expenses)
- `internal/adapters/secondary/postgres/export_repository.go` — PG repo (role-based SQL queries, roleFilter helper)
- `internal/adapters/secondary/postgres/export_repository_test.go` — PG repo tests
- `cmd/server/main.go` — Route registration (lines 198-200: GET /exports/timesheets, /expenses, /combined)

### Backend: Shared patterns to follow
- `migrations/000_full_schema.up.sql` — Base schema (for reference)
- `internal/middleware/middleware.go` — Auth middleware, GetRole, GetUserID, GetOrganizationID helpers
- `pkg/api/response.go` — API response envelope (for count endpoint responses)

### Frontend: Reference patterns
- `web/src/routes/_authenticated/time-entries/index.tsx` — Time entries route (add export tab)
- `web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx` — Existing tab layout (add Export tab)
- `web/src/api/time-entries.ts` — API module pattern (reference for exports API)
- `web/src/api/__tests__/` — Frontend test patterns
- `web/src/components/layout/sidebar.tsx` — Sidebar (add Exports nav item)
- `web/src/components/ui/` — Existing shadcn components (combobox, calendar, button)
- `web/src/lib/api.ts` — HTTP client with cookie auth + 401 auto-refresh
- `web/src/types/api.ts` — API types pattern
- `web/src/types/models.ts` — Model types pattern

### Prior Phase Context
- `.planning/phases/06-time-entries-expenses/06-CONTEXT.md` — Phase 6 context (exports deferred note, approval workflow patterns)
- `.planning/phases/05-mvp-consolidation/05-CONTEXT.md` — Project patterns (dialog-based CRUD, mutation patterns, test approach)
- `.planning/phases/04-contracts/04-CONTEXT.md` — Delete protection patterns, combobox patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Already Built (backend — adapt)
- Full ExportRepository with Timesheets and Expenses SQL queries, role-based filtering
- Full ExportService (thin, delegates to repo)
- Full ExportHandler (Timesheets, Expenses, Combined — all return CSV with Content-Disposition)
- All 3 routes wired in `cmd/server/main.go` with auth middleware
- Admin/manager role scoping: employee sees own data, manager sees own+managed, finance/admin sees all
- Date range defaults to current month via `parseExportRange()`

### Already Built (backend tests — minimal)
- Handler tests export_test.go: basic missing-auth and with-auth tests (mostly panic-recovery tests with nil service)
- Service tests export_test.go: basic Timesheets/Expenses/Combined with mocks (no real data tests)
- No repository integration tests for export repo

### Missing (to build/change)
- **Backend:** XLSX generation via excelize, `?format=` query param, `?project_id=&user_id=` filter params, count endpoints, CSV streaming
- **Frontend:** `useDownload` hook, API module, ExportForm component, export tabs on existing pages, /exports route for combined, sidebar nav item
- **Tests:** Updated handler tests for new params, integration test for export repo, frontend tests

### Reusable Assets
- `react-day-picker` — Already installed, use for date range picker
- `sonner` toast — Already installed, use toast.promise for feedback
- `utils.ts` — cn() helper for conditional classes
- `Combobox` component — For user/project selectors if added
- `Sidebar` component — Add new nav item following existing pattern
- Existing tab bar pattern on time-entries page — Add Export as 3rd tab

### Established Patterns
- `queryOptions`/`mutationOptions` wrappers in API modules
- Dialog-based CRUD (not applicable here — download is direct action)
- `useSuspenseQuery` / `useQuery` for data fetching
- `api<T>()` helper with `credentials: 'include'`
- Kebab-case file names for components, camelCase for API modules

### Integration Points
- Time entries page tabs: add "Export" tab alongside List/Calendar
- Expenses page tabs: add "Export" tab alongside List/Calendar
- New `/exports` route: add to TanStack Router route tree
- Sidebar: add Exports nav item in Tracking section
- `cmd/server/main.go`: add new route params and count endpoints

</code_context>

<specifics>
## Specific Ideas

- **Export tab behavior**: When user selects the Export tab on time-entries page, the calendar/list content is replaced by the export form. Switching back restores the previous view.
- **Preset buttons**: Clicking "This Month" updates the calendar picker visually to show the selected range.
- **Combined export page**: Standalone page with format selector, date range, project filter, user selector (for managers/finance). No in-page tabs.
- **CSV streaming**: Current backend loads all rows into memory, builds CSV string, then writes response. Stream by writing CSV rows with csv.Writer directly to the http.ResponseWriter as the query rows come in.
- **XLSX multi-sheet**: Combined XLSX has two sheets named "Timesheets" and "Expenses" with formatted headers.

</specifics>

<deferred>
## Deferred Ideas

- **Download history / audit trail** — Tracking who exported what and when. Not needed for MVP.
- **Scheduled recurring exports** — Automating exports on a schedule. Post-MVP.
- **Email exports** — Sending export files via email. Post-MVP.
- **Excel advanced formatting** — Conditional formatting, charts, pivot tables in XLSX. Post-MVP.
- **Multiple format export at once** — Download both CSV and XLSX in one click. Post-MVP.

</deferred>

---

*Phase: 07-Exports*
*Context gathered: 2026-07-07*
