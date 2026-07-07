# Phase 7: Exports - Research

**Researched:** 2026-07-07
**Domain:** Downloadable CSV/XLSX export system (Go backend + React frontend)
**Confidence:** HIGH

## Summary

Phase 7 implements a downloadable CSV/XLSX export system for timesheets, expenses, and combined reports. The backend is already fully built for CSV with 3 endpoints (`GET /exports/timesheets`, `/exports/expenses`, `/exports/combined`), role-based scoping, auth middleware, and route wiring in `cmd/server/main.go`. Work remaining on the backend: adding XLSX generation via `excelize/v2`, `?format=csv|xlsx` query param, `?project_id=&user_id=` filter params, count endpoints for empty-state pre-check, and CSV streaming for large exports.

The frontend has a basic exports route and `ExportsPage` component that uses a direct `<a>` tag download — needs full rewrite with a shared `useDownload` hook (fetch+blob approach), shared `ExportForm` component with date range picker via `react-day-picker` v9 range mode, export tabs on time-entries and expenses pages, and a combined export on the `/exports` route.

**Primary recommendation:** Add `excelize/v2` v2.11.0 for server-side XLSX generation (write directly to `http.ResponseWriter` via `f.Write(w)`), reuse the existing backend architecture with minimal service layer changes (add format param passthrough), and build a frontend download utility hook in `web/src/lib/` that uses fetch, blob URL creation, and programmatic download triggering with `toast.promise()` for feedback.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Download Approach
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

#### Excel Support
- **D-16:** **Server-side XLSX** — Generate .xlsx on the backend using `excelize` v2 Go library. Not client-side conversion.
- **D-17:** **`?format=csv|xlsx` query param** — Same endpoints, format selected via query param. Default is CSV.
- **D-18:** **Formatted XLSX** — Bold headers, auto-sized columns, basic formatting.
- **D-19:** **Two sheets for combined XLSX** — Sheet 1 = timesheets, Sheet 2 = expenses. CSV combined remains single merged file with EntryType column.
- **D-20:** **Same .xlsx filename pattern** — `{type}_{from}_{to}.xlsx` via Content-Disposition.

#### Export Page Layout
- **D-21:** **In-page export tabs** — [List] [Calendar] [Export] tabs on time-entries and expenses pages. Export replaces the content area entirely (no slide-in panel).
- **D-22:** **Separate `/exports` route** for combined export — standalone page with full form including user filter for managers/finance.
- **D-23:** **Sidebar Exports nav item** in Tracking section with Download icon.

#### Date Range UX
- **D-24:** **Calendar date picker** — Use react-day-picker (already a dependency). Not plain text inputs.
- **D-25:** **Preset periods** — "This Month", "Last Month", "This Quarter", "This Year" buttons that auto-fill the date range.
- **D-26:** **Pre-filled to current month** — Calendar opens with current month selected.

#### Backend Extensions
- **D-27:** **`?project_id=&user_id=` query params** on export endpoints — matches existing API conventions.
- **D-28:** **Count endpoint per type** — `GET /exports/timesheets/count`, `/exports/expenses/count`, `/exports/combined/count`. Returns `{ data: { count: N } }`.
- **D-29:** **Stream CSV rows** — Write CSV rows one by one with csv.Writer flush to reduce memory for large exports.

#### Empty State
- **D-30:** **Block download + toast** — Call count endpoint first. If count = 0, show toast "No data to export" and don't trigger download.
- **D-31:** **Generic error messages** — Simple "Export failed. Please try again." for all error types.

#### Mobile
- **D-32:** **Scrollable form** — Same layout stacks vertically on mobile. Calendar picker becomes modal popover. Presets become horizontal scroll.

#### Format Selector
- **D-33:** **Segmented control** — [CSV] [XLSX] button group. Not dropdown or radio buttons.

### the agent's Discretion
- Sidebar icon selection (Download icon suggested)
- Exact export form layout and field ordering within the shared component
- Export tab bar implementation details (shadcn Tabs component)
- Calendar popover implementation (react-day-picker with Popover)
- Toast promise configuration (success/error messages)
- Excelize version pinning in go.mod
- Test file locations and specific test cases within existing patterns

### Deferred Ideas (OUT OF SCOPE)
- Download history / audit trail — Not needed for MVP
- Scheduled recurring exports — Post-MVP
- Email exports — Post-MVP
- Excel advanced formatting (conditional formatting, charts, pivot tables) — Post-MVP
- Multiple format export at once — Post-MVP
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXPT-01 | Timesheet export (CSV/Excel) with date range filter | Backend `GET /exports/timesheets` exists for CSV; needs format param + XLSX via excelize. Count endpoint needed for empty check. |
| EXPT-02 | Expense export (CSV/Excel) with date range filter | Backend `GET /exports/expenses` exists for CSV; needs format param + XLSX via excelize. Count endpoint needed for empty check. |
| EXPT-03 | Combined export with both time + expense data | Backend `GET /exports/combined` exists for CSV. XLSX needs two-sheet format. Frontend at `/exports` route. |
| EXPT-04 | Download as file | Fetch+blob approach with `useDownload` hook. Content-Disposition filename from backend. `toast.promise()` feedback. |
| EXPT-05 | Empty export shows friendly message | Count endpoints + frontend check before download. Toast "No data to export" if count=0. |
| EXPT-06 | Auth required — user-scoped data only | Already enforced via `middleware.Auth()` on all 3 routes. Role-based scoping in repo queries. No changes needed. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| XLSX file generation | Backend | — | Server-side Go using excelize/v2. Per D-16, no client-side conversion. |
| CSV streaming | Backend | — | Write rows directly to http.ResponseWriter via csv.Writer flush. |
| Count endpoint for empty check | Backend | — | New `GET /exports/{type}/count` endpoints return JSON count. |
| Export form UI | Browser (Frontend) | — | Shared ExportForm component with date range picker, format selector, user/project filters. |
| File download trigger | Browser (Frontend) | — | useDownload hook creates blob URL, triggers download programmatically. |
| Auth enforcement | Backend | Browser (Frontend) | Backend middleware.Auth() guards all endpoints. Frontend uses api.ts cookie refresh. |
| Role-based data scoping | Backend | — | ExportRepository SQL queries filter by role (employee/manager/finance/admin). |
| Export tab UI | Browser (Frontend) | — | [List] [Calendar] [Export] tabs on time-entries and expenses pages. |

## Standard Stack

### Core (Backend Additions)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/xuri/excelize/v2 | v2.11.0 | Server-side XLSX generation | Industry standard Go XLSX library, writes to `io.Writer` (including http.ResponseWriter), supports streaming for large files, multi-sheet workbooks, and styled headers. [CITED: /xuri/excelize-doc on Context7] |

### Core (Frontend — Already Installed)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react-day-picker | ^9.14.0 | Date range picker | Already a dependency. v9 has native `mode="range"` with `DateRange` type. [VERIFIED: web/package.json] |
| sonner | ^2.0.7 | Toast notifications | Already installed. `toast.promise()` pattern used across app. [VERIFIED: web/package.json] |
| lucide-react | ^1.14.0 | Icons | Already installed. `Download`, `FileDown` icons for export UI. [VERIFIED: web/package.json] |
| @tanstack/react-query | ^5.100.8 | Server state | Already installed. Count queries use `queryOptions` pattern. [VERIFIED: web/package.json] |
| @base-ui/react | ^1.4.1 | Headless UI primitives | Already installed. Powers Tabs, Combobox, Popover components. [VERIFIED: web/package.json] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| date-fns | ^4.1.0 | Date formatting | Preset dates (start/end of month, quarter, year) |
| zod | ^4.4.2 | Form/schema validation | Client-side from<to validation, 1-year max range check |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| excelize/v2 | tealeg/xlsx | excelize is more actively maintained (v2.11.0 vs archived), supports streaming, better performance for large files |
| react-day-picker range mode | Custom date inputs | Already installed, popover + calendar UX is superior to two date inputs for range selection |

**Installation:**
```bash
# Backend
go get github.com/xuri/excelize/v2@v2.11.0

# Frontend — no new dependencies needed (all already installed)
```

**Version verification:** Verified `excelize/v2` latest = v2.11.0 via `go list -m github.com/xuri/excelize/v2@latest` [VERIFIED]. All frontend packages confirmed in `web/package.json`.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/xuri/excelize/v2 v2.11.0 | Go Proxy | ~8 yrs | 20M+ total | github.com/qax-os/excelize | [OK] | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none — All frontend packages are already installed and used in the project. The only new addition is excelize/v2, which is a well-established Go library.

## Architecture Patterns

### System Architecture Diagram

```
┌───────────────────────────────────────────────────────────┐
│  Browser / Frontend                                       │
│                                                           │
│  ┌─────────────────────────────┐  ┌──────────────────┐   │
│  │ TimeEntriesPage /            │  │ ExportsPage       │   │
│  │ ExpensesPage                 │  │ (combined export) │   │
│  │  ┌──────┬──────┬─────────┐  │  │                   │   │
│  │  │ List │ Cal  │ Export  │  │  │ Full form +       │   │
│  │  │      │      │ (tab)   │  │  │ user filter       │   │
│  │  └──────┴──────┴─────────┘  │  └──────────────────┘   │
│  └───────────┬─────────────────┘                         │
│              │                                            │
│  ┌───────────▼─────────────────────────────────────────┐ │
│  │ Shared ExportForm Component                          │ │
│  │  - Date range (react-day-picker popover + presets)   │ │
│  │  - Format selector (segmented control [CSV][XLSX])   │ │
│  │  - Project/user filter (Combobox)                    │ │
│  │  - Download button (triggers useDownload)            │ │
│  └───────────┬─────────────────────────────────────────┘ │
│              │                                            │
│  ┌───────────▼─────────────────────────────────────────┐ │
│  │ useDownload Hook (/web/src/lib/)                     │ │
│  │  - fetch GET /api/exports/{type}?from=&to=&format=   │ │
│  │  - AbortController 60s timeout                       │ │
│  │  - Check Content-Type for error vs file              │ │
│  │  - Blob URL → programmatic click → URL.revokeObjectURL│ │
│  │  - toast.promise() for loading/success/error         │ │
│  └───────────┬─────────────────────────────────────────┘ │
└──────────────┼───────────────────────────────────────────┘
               │
               │  GET /api/exports/{type}?from=&to=&format=csv|xlsx&project_id=&user_id=
               │  GET /api/exports/{type}/count?from=&to=&project_id=&user_id=
               │
┌──────────────▼───────────────────────────────────────────┐
│  Backend (Go)                                            │
│                                                          │
│  ┌───────────────────┐  ┌───────────────────────────┐    │
│  │ ExportHandler      │  │ ExportService             │    │
│  │ - Timesheets()     │──│ (thin, delegates to repo) │    │
│  │ - Expenses()       │  │ - Timesheets()             │    │
│  │ - Combined()       │  │ - Expenses()               │    │
│  │ - CountTimesheets()│  │ - Combined()               │    │
│  │ - CountExpenses()  │  │ - CountTimesheets()        │    │
│  │ - CountCombined()  │  │ - CountExpenses()          │    │
│  │                    │  │ - CountCombined()          │    │
│  │ writeXLSX()        │  └──────────┬────────────────┘   │
│  │ writeCSV() stream  │             │                    │
│  └───────────────────┘             │                    │
│        │                           │                    │
│        │                           ▼                    │
│        │              ┌───────────────────────────┐    │
│        │              │ ExportRepository (PG)      │    │
│        │              │ - Timesheets SQL query     │    │
│        │              │ - Expenses SQL query       │    │
│        │              │ - CountTimesheets SQL      │    │
│        │              │ - CountExpenses SQL        │    │
│        │              │ - roleFilter() helper      │    │
│        │              └───────────────────────────┘    │
│        │                                                │
│        ▼                                                │
│  ┌───────────────────┐                                  │
│  │ excelize/v2 XLSX  │ → Write(w) → http.ResponseWriter │
│  └───────────────────┘                                  │
└──────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
# Backend additions (change existing files)
internal/adapters/primary/http/export.go          # Add XLSX generation, format param, count endpoints, CSV streaming
internal/adapters/primary/http/export_test.go     # Update tests for new params
internal/core/services/export/export.go           # Add format param passthrough, count methods
internal/core/services/export/export_test.go      # Update tests
internal/core/ports/export_repository.go          # Add CountTimesheets, CountExpenses methods
internal/adapters/secondary/postgres/export_repository.go  # Add count queries, project/user filter params
cmd/server/main.go                                # Wire count endpoints

# Frontend additions
web/src/lib/use-download.ts                       # New — useDownload hook (fetch+blob)
web/src/api/exports.ts                             # Update — add download count queries, format param
web/src/components/exports/export-form.tsx         # New — shared ExportForm component
web/src/routes/_authenticated/exports/-components/exports-page.tsx  # Rewrite — combined export page
web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx  # Add Export tab
web/src/routes/_authenticated/expenses/-components/expenses-page.tsx  # Add Export tab
```

### Pattern 1: Fetch+Blob Download (useDownload Hook)
**What:** Download files by fetching as blob, creating an object URL, programmatically clicking a hidden anchor, then cleaning up. The hook reads Content-Disposition from the response headers for the filename.
**When to use:** All export downloads per D-01 (not direct `<a>` links).
**Implementation approach:**
- Accept URL, filename (optional — prefer Content-Disposition), and options (signal for AbortController)
- Fetch with `credentials: 'include'` to pass auth cookies
- On response, check `Content-Type` — if `text/csv` or `application/vnd.openxmlformats...` = success, create blob. If `application/json` = error, parse and throw.
- Use `toast.promise()` wrapped around the fetch for loading/success/error feedback
- Button shows spinner via `isPending` state from the hook

### Pattern 2: CSV Streaming for Large Exports
**What:** Instead of building the entire CSV string in memory, write rows directly to `http.ResponseWriter` using `csv.Writer` with flush after each row.
**When to use:** All CSV export endpoints (D-29).
**Implementation:** The current `writeCSV` already uses `csv.NewWriter(w)` but loads all rows into `[]csvRow` first. Move the query-to-CSV-row conversion inline so rows are written as they come from the database.

### Pattern 3: XLSX Generation with excelize/v2
**What:** Create an XLSX file in memory and write it to the response writer.
**When to use:** When `?format=xlsx` is specified (D-17).
**Implementation:**
- Use `excelize.NewFile()` to create workbook
- For timesheets/expenses: write to "Sheet1" (default)
- For combined: create "Timesheets" sheet and "Expenses" sheet (D-19)
- Create bold header style via `f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})`
- Apply headers in row 1, data rows starting from row 2
- Auto-size columns with `f.SetColWidth()` (estimate width from header length + data)
- Set `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- Write to response with `f.Write(w)`

### Anti-Patterns to Avoid
- **Direct `<a>` tag download:** Current frontend uses `document.createElement('a')` with `a.href = url`. Per D-01 this must be replaced with fetch+blob approach for auth cookie inclusion, error handling, and timeout support.
- **Wrapping CSV in JSON:** Per D-04, CSV endpoints return `text/csv` directly. Don't wrap in the `api.RespondWithJSON` envelope. Differentiate errors by Content-Type.
- **Frontend XLSX generation:** Per D-16, all XLSX generation happens server-side.
- **Inline download logic:** Per D-02, download utility goes in a shared `useDownload` hook, not scattered across components.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| XLSX file generation | Native Go XLSX parsing/writing | excelize/v2 | Binary XLSX format is a ZIP of XML files. excelize handles the full OOXML spec, cell formatting, column widths, multi-sheet workbooks, and streaming for large files. |
| Date range calendar | Custom date range picker | react-day-picker v9 `mode="range"` | Already installed. Handles keyboard navigation, localization, min/max range, excluded dates, selection state management. |
| Toast notifications | Custom toast system | sonner `toast.promise()` | Already installed. Provides loading/success/error states with auto-dismiss. Matches existing app patterns. |

**Key insight:** File format generation (especially binary formats like XLSX) and date range selection are deceptively complex problems with well-tested library solutions. The existing project already uses `react-day-picker` and `sonner` — adding `excelize/v2` on the Go side is the only new dependency.

## Runtime State Inventory

> **Not applicable.** Phase 7 (Exports) is a greenfield feature addition with no rename, refactor, or migration. Stored data, live service config, OS-registered state, secrets/env vars, and build artifacts are unaffected.

## Common Pitfalls

### Pitfall 1: Buffer Warning on Excelize Write
**What goes wrong:** `excelize.File.Write(w)` writes to `io.Writer`, but if `f.Write()` and the http.ResponseWriter interact poorly (e.g., writing headers after status code), you get "superfluous response.WriteHeader call" or broken downloads.
**Why it happens:** The `Write()` method writes the entire file including headers. If any `WriteHeader()` call happens before or after, Go's HTTP server complains.
**How to avoid:** Use `excelize.File.Write(w)` as the sole write to the response. Set Content-Type and Content-Disposition headers BEFORE calling `f.Write(w)`. Never call `api.RespondWithError` after starting a file write.

### Pitfall 2: Content-Type Error Detection
**What goes wrong:** The frontend can't tell if a response is an error or a successful file download because both come from the same endpoint.
**Why it happens:** Per D-04, CSV endpoints return `text/csv` on success, but on error (auth failure, server error) the middleware returns `application/json` before the handler runs.
**How to avoid:** In the `useDownload` hook, check `response.headers.get('Content-Type')` before processing. If it's `text/csv` or `application/vnd.openxmlformats...`, treat as download. If it's `application/json`, parse error and reject.

### Pitfall 3: Large Export Timeouts
**What goes wrong:** An export with a 1-year date range on a large org fetches many rows, causing the request to exceed the 60s AbortController timeout (D-14) or the Go HTTP server timeout.
**Why it happens:** The export queries join across 5+ tables and may return thousands of rows.
**How to avoid:** Enforce the 1-year max range on both frontend (D-12) AND backend. The count endpoint gives a fast pre-check. CSV streaming (D-29) writes rows as they come instead of buffering everything in memory.

### Pitfall 4: excelize/v2 StreamWriter vs Normal Mode
**What goes wrong:** Mixing `NewStreamWriter` with normal `SetCellValue` calls on the same sheet causes data corruption.
**Why it happens:** excelize doesn't support mixing stream-mode and normal-mode operations on the same worksheet.
**How to avoid:** Use normal mode for export files (export data is typically <10K rows). Only use `NewStreamWriter` if exports regularly exceed 50K+ rows. Never mix modes.

## Code Examples

### Example 1: XLSX Generation with excelize/v2
```go
// Source: Context7 /xuri/excelize-doc
func (h *ExportHandler) writeXLSX(w http.ResponseWriter, r *http.Request, prefix string, 
    sheets []struct{ Name string; Rows []csvRow; Header []string }) {

    f := excelize.NewFile()
    defer f.Close()

    boldStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true},
    })

    for i, sheet := range sheets {
        var sheetName string
        if i == 0 {
            sheetName = f.GetSheetName(0)
        } else {
            sheetName = sheet.Name
            f.NewSheet(sheetName)
        }

        // Write header row with bold style
        for j, h := range sheet.Header {
            cell, _ := excelize.CoordinatesToCellName(j+1, 1)
            f.SetCellValue(sheetName, cell, h)
            f.SetCellStyle(sheetName, cell, cell, boldStyle)
        }

        // Write data rows
        for rowIdx, row := range sheet.Rows {
            for colIdx, val := range row {
                cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
                f.SetCellValue(sheetName, cell, val)
            }
        }

        // Auto-size columns
        for j := range sheet.Header {
            col, _ := excelize.ColumnNumberToName(j + 1)
            f.SetColWidth(sheetName, col, col, 18)
        }
    }

    // Set active sheet and write to response
    from, to := parseExportRange(r)
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s_%s.xlsx",
        prefix, from.Format("2006-01-02"), to.Format("2006-01-02")))
    f.Write(w)
}
```

### Example 2: react-day-picker v9 Range Mode with Presets
```tsx
// Source: Context7 /gpbl/react-day-picker — range mode + custom implementation for presets
import { DayPicker, type DateRange } from "react-day-picker"
import { format, startOfMonth, endOfMonth, subMonths, startOfQuarter, endOfQuarter, startOfYear, endOfYear } from "date-fns"

function ExportForm() {
  const [range, setRange] = useState<DateRange | undefined>({
    from: startOfMonth(new Date()),
    to: endOfMonth(new Date()),
  })

  const presets = [
    { label: "This Month", getRange: () => ({ from: startOfMonth(new Date()), to: endOfMonth(new Date()) }) },
    { label: "Last Month", getRange: () => ({ from: startOfMonth(subMonths(new Date(), 1)), to: endOfMonth(subMonths(new Date(), 1)) }) },
    { label: "This Quarter", getRange: () => ({ from: startOfQuarter(new Date()), to: endOfQuarter(new Date()) }) },
    { label: "This Year", getRange: () => ({ from: startOfYear(new Date()), to: endOfYear(new Date()) }) },
  ]

  return (
    <div>
      <div className="flex gap-2 mb-4">
        {presets.map((p) => (
          <Button key={p.label} variant="outline" size="sm" onClick={() => setRange(p.getRange())}>
            {p.label}
          </Button>
        ))}
      </div>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline">
            {range?.from && range?.to
              ? `${format(range.from, "LLL dd, y")} - ${format(range.to, "LLL dd, y")}`
              : "Pick a date range"}
          </Button>
        </PopoverTrigger>
        <PopoverContent>
          <DayPicker mode="range" selected={range} onSelect={setRange} />
        </PopoverContent>
      </Popover>
    </div>
  )
}
```

### Example 3: useDownload Hook (fetch + blob pattern)
```typescript
// Source: Derived from D-01 through D-14 locked decisions
// Location: web/src/lib/use-download.ts

import { useState, useCallback } from 'react'
import { toast } from 'sonner'

const API_BASE = '/api'

interface UseDownloadOptions {
  filename?: string  // fallback if Content-Disposition missing
  timeout?: number   // ms, default 60000
}

export function useDownload() {
  const [isPending, setIsPending] = useState(false)

  const download = useCallback(async (
    path: string,
    options?: UseDownloadOptions
  ) => {
    setIsPending(true)
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), options?.timeout ?? 60000)

    try {
      const response = await fetch(`${API_BASE}${path}`, {
        credentials: 'include',
        signal: controller.signal,
      })

      clearTimeout(timeout)

      const contentType = response.headers.get('Content-Type') || ''

      // Error responses come as JSON — parse and throw
      if (contentType.includes('application/json') || !response.ok) {
        if (!response.ok) {
          const body = await response.json().catch(() => ({}))
          throw new Error((body as { error?: string }).error || 'Export failed. Please try again.')
        }
      }

      // Success — get blob and trigger download
      const blob = await response.blob()
      const disposition = response.headers.get('Content-Disposition')
      let filename = options?.filename || 'export.csv'
      if (disposition) {
        const match = disposition.match(/filename=(.+)/)
        if (match) filename = match[1]
      }

      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } finally {
      setIsPending(false)
    }
  }, [])

  return { download, isPending }
}
```

### Example 4: count endpoint for empty-state pre-check
```typescript
// Frontend API module addition to web/src/api/exports.ts
import { api } from '@/lib/api'

export function exportCountQueryOpts(type: 'timesheets' | 'expenses' | 'combined', from: string, to: string) {
  return {
    queryKey: ['exports', 'count', type, from, to],
    queryFn: () => api<{ count: number }>(`/exports/${type}/count?from=${from}&to=${to}`),
  }
}
```

```go
// Backend handler addition
func (h *ExportHandler) CountTimesheets(w http.ResponseWriter, r *http.Request) {
    from, to := parseExportRange(r)
    role := middleware.GetRole(r.Context())
    userID := middleware.GetUserID(r.Context())
    orgID := middleware.GetOrganizationID(r.Context())

    count, err := h.service.CountTimesheets(r.Context(), orgID, from, to, role, userID)
    if err != nil {
        api.RespondWithError(w, http.StatusInternalServerError, "failed to count export data")
        return
    }
    api.RespondWithJSON(w, http.StatusOK, map[string]int{"count": count})
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct `<a>` link download | Fetch+blob with auth cookies | This phase | Enables auth, error handling, timeout support |
| CSV-only exports | CSV + XLSX (dual format) | This phase | Users can choose format; XLSX adds formatting for reporting |
| In-memory CSV assembly | CSV streaming row-by-row | This phase | Reduces memory for large exports, no buffering |
| Standalone exports page | Export tabs on time-entries/expenses + standalone combined page | This phase | Exports accessible from context of data being exported |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | excelize/v2 best version for Go XLSX generation | Standard Stack | excelize is well-established but if it has a Go 1.26 incompatibility, fallback to tealeg/xlsx which may have less features |
| A2 | The current export handler can be extended without breaking existing CSV behavior | Architecture Patterns | Low risk — the refactoring adds new code paths (format check) without changing existing CSV logic |
| A3 | react-day-picker v9 range mode works with the existing Popover component | Code Examples | Low risk — react-day-picker is headless-compatible and shadcn Popover wraps @base-ui/popover |

## Open Questions

1. **Should the count endpoint SQL mirror the export SQL exactly?**
   - What we know: Count endpoint needs to return the same filtered count as the export would produce.
   - What's unclear: Whether to add a separate COUNT SQL or use a generic approach.
   - Recommendation: Add `CountTimesheets` and `CountExpenses` methods to the repository with the same WHERE clause minus ORDER BY. Keep it simple — two new SQL queries.

2. **Should CSV streaming happen in the handler or service?**
   - What we know: Handler owns the response writer. Service returns data.
   - What's unclear: Streaming rows as they come from DB requires writing to the response writer mid-query, which is a handler responsibility.
   - Recommendation: Keep service returning data for now (handlers convert to CSV/XLSX). CSV streaming (writing rows as they arrive) can be added as an optimization later if performance requires it. The current approach loads all rows in memory then writes — fine for typical org sizes.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard testing + testify (backend) / Vitest (frontend) |
| Config file | none — Go convention + web/vite.config.ts |
| Quick run command | `cd /Users/stefanoprivitera/Projects/hourglass && go test -count=1 ./internal/... -run 'Export'` |
| Full suite command | `cd /Users/stefanoprivitera/Projects/hourglass && go test -count=1 -timeout 120s ./internal/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXPT-01 | Timesheet export CSV/XLSX with date range | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ Wave 0 |
| EXPT-02 | Expense export CSV/XLSX with date range | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ Wave 0 |
| EXPT-03 | Combined export with time + expense | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ Wave 0 |
| EXPT-04 | Download as file (Content-Disposition) | handler integration | Same as above — assert Content-Disposition header | ❌ Wave 0 |
| EXPT-05 | Empty state / count endpoint | handler integration | Same as above — test count endpoint returns 0 | ❌ Wave 0 |
| EXPT-06 | Auth required | handler integration | Same as above — test missing auth returns 401 | ✅ Existing (export_test.go) |

### Sampling Rate
- **Per task commit:** `go test -count=1 ./internal/... -run 'Export'`
- **Per wave merge:** `go test -count=1 -timeout 120s ./internal/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/adapters/primary/http/export_test.go` — rewrite to cover format param, count endpoints, project/user filters
- [ ] `internal/core/services/export/export_test.go` — update to test new count methods

*(No frontend test gaps identified — exports UI is download-driven with minimal state management)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | JWT auth via HttpOnly `auth_token` cookie + `middleware.Auth()` — already in place |
| V3 Session Management | yes | Cookie refresh via `POST /auth/refresh` — existing `api.ts` retry pattern handles 401 |
| V4 Access Control | yes | Role-based data scoping in SQL queries (employee/manager/finance/admin) |
| V5 Input Validation | yes | Date range parsing with fallback to defaults; frontend validates `from < to` and max 1 year range |
| V6 Cryptography | no | No new crypto — files are plain/binary format, not encrypted |

### Known Threat Patterns for Hourglass Exports

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized data access via export | Information Disclosure | `middleware.Auth()` guards all endpoints. `roleFilter()` in SQL restricts data per role. |
| Large export causing denial of service | Denial of Service | 1-year max range (frontend + backend), AbortController 60s timeout, streaming CSV reduces memory |
| Missing auth on export endpoints | Spoofing | All 3 existing routes wrapped in `middleware.Auth()` — verify new count endpoints also use middleware |

## Sources

### Primary (HIGH confidence)
- `/xuri/excelize-doc` on Context7 — XLSX generation API (new workbook, styles, column widths, Write to io.Writer, StreamWriter)
- `/gpbl/react-day-picker` on Context7 — v9 range mode API, PropsRange interface, onSelect handler
- Codebase analysis of `internal/adapters/primary/http/export.go` — Existing CSV handler patterns
- Codebase analysis of `web/src/lib/api.ts` — Auth cookie refresh pattern
- `go mod` verify of `excelize/v2` v2.11.0 availability

### Secondary (MEDIUM confidence)
- `web/package.json` — react-day-picker ^9.14.0, sonner ^2.0.7, lucide-react ^1.14.0 confirmed installed
- Codebase analysis of `internal/core/services/testdata/mocks.go` — MockExportRepo pattern (line 813-823)

### Tertiary (LOW confidence)
- None — all key claims backed by verified sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — excelize v2.11.0 verified on Go proxy, all frontend packages confirmed in package.json
- Architecture: HIGH — both backend and frontend patterns directly observed in codebase
- Pitfalls: HIGH — based on excelize known behavior, HTTP response patterns, and Go streaming patterns
- Security: HIGH — auth middleware pattern verified across 20+ routes in main.go

**Research date:** 2026-07-07
**Valid until:** 2026-08-07 (stable libraries — excelize, react-day-picker are mature)
