# Phase 7: Exports - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 07-Exports
**Areas discussed:** Download approach, Excel support, Export page layout, Date range UX, Empty state, Format selector UI, Export tab UX, Export scope, Mobile layout, Large export behavior, Shared export component, Download timeout

---

## Download Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Direct link | Simple `<a href>` — browser handles download natively | |
| Fetch + blob | fetch() + blob URL + programmatic click | ✓ |
| Both | Direct link as primary with fetch+blob fallback | |

**User's choice:** Fetch + blob
**Notes:** Loading state control and error handling are the key drivers.

| Option | Description | Selected |
|--------|-------------|----------|
| Backend-driven | Let Content-Disposition header determine filename | ✓ |
| Frontend-driven | Construct filename in JS | |

**User's choice:** Backend-driven

| Option | Description | Selected |
|--------|-------------|----------|
| Toast + stay on page | sonner toast with error | |
| Inline error in form | Error message in the form area | |
| Both | Toast + inline | |

**User's choice:** Toast promise pattern (sonner `toast.promise()` with loading/success/error states)

| Option | Description | Selected |
|--------|-------------|----------|
| In exports API module | `web/src/api/exports.ts` | |
| Shared lib hook | `web/src/lib/useDownload.ts` | ✓ |

**User's choice:** Shared lib hook

| Option | Description | Selected |
|--------|-------------|----------|
| Button click | User fills form, clicks Download | ✓ |
| Auto on change | Pre-fetch on form changes | |

**User's choice:** Button click

| Option | Description | Selected |
|--------|-------------|----------|
| Button spinner | Button shows spinner + 'Downloading...' | ✓ |
| Full-page overlay | Blocking overlay with spinner | |
| Toast promise | Sonner toast.promise for feedback | ✓ |

**User's choice:** Button spinner + disabled state + toast promise for feedback

| Option | Description | Selected |
|--------|-------------|----------|
| Raw CSV response | Keep backend as-is (text/csv) | ✓ |
| Wrap in JSON envelope | Change backend to return JSON | |

**User's choice:** Raw CSV response (recommended by agent as standard pattern for file downloads)

| Option | Description | Selected |
|--------|-------------|----------|
| Fetch without progress | fetch + toast.promise, no progress bar | ✓ |
| XHR with progress | XMLHttpRequest for real progress events | |

**User's choice:** Fetch without progress

| Option | Description | Selected |
|--------|-------------|----------|
| .csv only | Simple CSV | |
| .csv + .xlsx | Both formats | ✓ |

**User's choice:** CSV + XLSX (deferred format decision to Excel area)

| Option | Description | Selected |
|--------|-------------|----------|
| Single file | One file with time+expense rows sorted by date | ✓ |
| Two files | Separate timesheets.csv and expenses.csv | |

**User's choice:** Single file

| Option | Description | Selected |
|--------|-------------|----------|
| Date range only | From/to date. Backend scopes by role | |
| + Project filter | Add project selector | |
| + Project + User | Full filters for finance/managers | ✓ |

**User's choice:** + Project + User filters

| Option | Description | Selected |
|--------|-------------|----------|
| No preview | Click Download → file saves | ✓ |
| Preview rows first | Show first 20 rows before download | |

**User's choice:** No preview

| Option | Description | Selected |
|--------|-------------|----------|
| Validate client-side | Check from < to before fetch | ✓ |
| Trust backend | Backend handles it | |

**User's choice:** Validate client-side

| Option | Description | Selected |
|--------|-------------|----------|
| No limit | Allow any date range | |
| 1 year max | Limit to 1 year per export | ✓ |
| 6 months max | Stricter limit | |

**User's choice:** 1 year max

| Option | Description | Selected |
|--------|-------------|----------|
| Current month | From = 1st, To = end of current month | ✓ |
| Current week | From = Monday, To = Sunday | |
| Last 30 days | From = 30 days ago | |

**User's choice:** Current month

---

## Excel Support

| Option | Description | Selected |
|--------|-------------|----------|
| Server-side (Go) | Add excelize library | ✓ |
| Client-side (JS) | Convert CSV to XLSX in browser | |

**User's choice:** Server-side (Go) with excelize

| Option | Description | Selected |
|--------|-------------|----------|
| Same endpoint + format param | `?format=csv\|xlsx` | ✓ |
| Separate endpoint | New route for XLSX | |

**User's choice:** Same endpoint + format param

| Option | Description | Selected |
|--------|-------------|----------|
| Two sheets | Sheet 1 = Timesheets, Sheet 2 = Expenses | ✓ |
| Single sheet | Merged with EntryType column | |

**User's choice:** Two sheets (XLSX only; CSV remains single merged)

| Option | Description | Selected |
|--------|-------------|----------|
| Bare data | No formatting | |
| Formatted | Bold headers, auto-width, number formatting | ✓ |

**User's choice:** Formatted

| Option | Description | Selected |
|--------|-------------|----------|
| Same pattern, .xlsx | `{type}_{from}_{to}.xlsx` | ✓ |
| Append -formatted | Add "_formatted" to filename | |

**User's choice:** Same pattern, .xlsx

---

## Export Page Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated /exports page | Full route with form | |
| Export tab on existing pages | In-page tabs on time-entries & expenses | ✓ |
| Dropdown from nav | Compact dropdown in sidebar/header | |

**User's choice:** Export tab on existing pages

| Option | Description | Selected |
|--------|-------------|----------|
| Separate /exports route | Standalone page for combined | ✓ |
| Dropdown link | Combined as link from existing page | |

**User's choice:** Separate /exports route for combined

| Option | Description | Selected |
|--------|-------------|----------|
| In-page tabs | State managed in component, no new route | ✓ |
| Route-level tabs | Changes URL path | |

**User's choice:** In-page tabs

| Option | Description | Selected |
|--------|-------------|----------|
| Yes | Add Exports link in sidebar | ✓ |
| No | Keep sidebar minimal | |

**User's choice:** Yes

---

## Date Range UX

| Option | Description | Selected |
|--------|-------------|----------|
| Calendar picker | react-day-picker (already installed) | ✓ |
| Text inputs | Simple from/to YYYY-MM-DD | |
| Both | Calendar + manual input | |

**User's choice:** Calendar picker

| Option | Description | Selected |
|--------|-------------|----------|
| With presets | This Month, Last Month, etc. | ✓ |
| No presets | Just calendar picker | |

**User's choice:** With presets

| Option | Description | Selected |
|--------|-------------|----------|
| This Month, Last Month | Simple month-level | |
| This Month, Last Month, This Quarter, This Year | Broader range | ✓ |

**User's choice:** This Month, Last Month, This Quarter, This Year

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, pre-filled | Calendar opens with current month | ✓ |
| No, empty | Both dates start empty | |

**User's choice:** Yes, pre-filled

---

## Empty State

| Option | Description | Selected |
|--------|-------------|----------|
| Let them download | User gets CSV with headers only | |
| Block with message | Show friendly message, no download | ✓ |

**User's choice:** Block with message

| Option | Description | Selected |
|--------|-------------|----------|
| Inspect response | One request, check blob size | |
| Pre-check endpoint | Separate count endpoint | ✓ |

**User's choice:** Pre-check endpoint (count endpoint per type)

| Option | Description | Selected |
|--------|-------------|----------|
| Inline in form | Message appears in export form area | |
| Toast notification | Sonner toast | ✓ |
| Both | Inline + toast | |

**User's choice:** Toast notification

| Option | Description | Selected |
|--------|-------------|----------|
| Separate per type | /exports/timesheets/count, etc. | ✓ |
| Single with param | /exports/count?type=timesheets | |

**User's choice:** Separate per type

---

## Format Selector UI

| Option | Description | Selected |
|--------|-------------|----------|
| Segmented control | [CSV] [XLSX] button group | ✓ |
| Dropdown | CSV (.csv) / Excel (.xlsx) | |
| Radio buttons | Standard radio group | |

**User's choice:** Segmented control

---

## Export Tab UX

| Option | Description | Selected |
|--------|-------------|----------|
| Replace content | Export tab replaces calendar/list | ✓ |
| Slide-in panel | Export form slides in from side | |

**User's choice:** Replace content

| Option | Description | Selected |
|--------|-------------|----------|
| Add Export as 3rd tab | [List] [Calendar] [Export] | ✓ |
| Export as sub-tab | Subtle action below tabs | |

**User's choice:** Add Export as 3rd tab

---

## Export Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Yes | User selector for managers/finance | ✓ |
| No | No per-user filter | |

**User's choice:** Yes — user/employee filter on combined export for managers/finance

---

## Mobile Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Scrollable form | Same layout stacks vertically, calendar as popover | ✓ |
| Compact form | Simplified: text inputs, dropdown presets | |

**User's choice:** Scrollable form

---

## Large Export Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Stream CSV rows | Write rows one by one with csv.Writer flush | ✓ |
| Add row limit | Cap at e.g. 10,000 rows | |
| No special handling | Keep as-is | |

**User's choice:** Stream CSV rows

---

## Shared Export Component

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, shared component | Single ExportForm with props | ✓ |
| Separate per page | Each page has own form | |

**User's choice:** Yes, shared component

---

## Download Timeout

| Option | Description | Selected |
|--------|-------------|----------|
| 30 second timeout | AbortController with 30s | |
| 60 second timeout | More generous for XLSX | ✓ |
| No timeout | Let it run as long as needed | |

**User's choice:** 60 second timeout

---

## Agent's Discretion

- Sidebar icon selection (Download icon suggested)
- Exact export form layout and field ordering within the shared component
- Export tab bar implementation details (shadcn Tabs component)
- Calendar popover implementation (react-day-picker with Popover)
- Toast promise configuration (success/error messages)
- Excelize version pinning in go.mod
- Test file locations and specific test cases within existing patterns

## Deferred Ideas

- Download history / audit trail — Not needed for MVP
- Scheduled recurring exports — Post-MVP
- Email exports — Post-MVP
- Excel advanced formatting — Post-MVP
- Multiple format export at once — Post-MVP
