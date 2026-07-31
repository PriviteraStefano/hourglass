---
phase: 08-pre-deployment-hardening-p0-audit-fixes
plan: 08-02
subsystem: ui
tags: [react, tanstack-router, react-query, zod, shadcn, playwright, vitest, msw, customers, time-entries, expenses, error-boundaries]

requires:
  - phase: 03-customers
    provides: customers list page component, customers API layer
  - phase: 04-contracts
    provides: contracts API layer, contract currency field
  - phase: 06-time-entry-expense-tracking
    provides: time-entries/expenses API hooks + query keys, EntryDetail/ExpenseDetail surfaces
provides:
  - /customers index route reachable from sidebar (P0-3 closed)
  - Filterable list views for time entries and expenses with URL-shareable filters (P0-2 closed)
  - Route error boundaries on all authenticated + auth pages (P0-4 closed)
  - Shared entries-table / status-badge / filter-control components
affects: [09-activity-ontology, phase-verification, verifier]

tech-stack:
  added: [date-fns DayPicker range mode (existing), base-ui DropdownMenuGroup pattern]
  patterns:
    - Generic type-safe table shell (EntriesTable<T>) with client-side pagination (ADR-FE-018, page 25)
    - URL-driven list filters via validateSearch with single-or-repeated param parsing (ADR-FE-017)
    - errorComponent + router.invalidate() retry pattern (TanStack Router v1.170)
    - Route-level component tests: real router + MSW handler per suite

key-files:
  created:
    - web/src/routes/_authenticated/customers/index.tsx
    - web/src/routes/_authenticated/customers/-context/customers-context.tsx
    - web/src/components/shared/entries-table.tsx
    - web/src/components/shared/entries-filters.tsx
    - web/src/components/shared/status-badge.tsx
    - web/src/components/layout/route-error.tsx
    - web/src/lib/list-filters.ts
    - web/src/routes/_authenticated/time-entries/-components/time-entries-list.tsx
    - web/src/routes/_authenticated/expenses/-components/expenses-list.tsx
  modified:
    - web/src/routes/_authenticated/time-entries/index.tsx
    - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
    - web/src/routes/_authenticated/expenses/index.tsx
    - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx
    - web/src/routes/_authenticated.tsx
    - web/src/routes/_auth.tsx
    - web/src/api/customers.ts
    - web/src/routeTree.gen.ts
    - web/e2e/customers.spec.ts
    - web/src/lib/__tests__/setup.ts
    - .gitignore

key-decisions:
  - "Expense amounts render currency resolved from the existing project→contract relationship; the expense payload carries no currency field (no new endpoint invented)"
  - "approved status badge recolored emerald so all six workflow states are visually distinct (was blue, colliding with submitted)"
  - "List filter state is URL-shareable via validateSearch (ADR-FE-017); arrays accept single, repeated, and JSON-serialized forms"
  - "Row click / New-entry affordance switch to the calendar tab and set the date search param, reusing the existing EntryDetail/ExpenseDetail surfaces"
  - "API entry_date is RFC3339, not yyyy-MM-dd — parsers accept both formats"
  - "Customers e2e suite logs in once via API and injects cookies to stay under the backend 5/min anonymous login rate limit"

patterns-established:
  - "List views: EntriesTable<T> + StatusFilterSelect/DateRangeFilter + route validateSearch wiring; pages own the search schema, shared controls are URL-agnostic"
  - "Error recovery: errorComponent uses router.invalidate() (not reset) so loaders re-run — per TanStack Router v1 docs"
  - "Route-level tests mount the real routeTree with createRouter + MSW, asserting URL state after interactions"

requirements-completed: [P0-2, P0-3, P0-4]

duration: 93min
completed: 2026-07-31
---

# Phase 8 Plan 2: Frontend Completion — List Views, Customers Route, Error Boundaries Summary

**/customers index route, filterable list views for time entries and expenses, and route-level error boundaries — closing P0-2, P0-3, and P0-4 with shared table/filter/badge components and 24 new tests (Vitest + Playwright)**

## Performance

- **Duration:** 93 min
- **Started:** 2026-07-31T10:57:00Z
- **Completed:** 2026-07-31T12:29:51Z
- **Tasks:** 5
- **Files modified:** 24 (created 10, modified 14)

## Accomplishments

- **P0-3 closed:** `/customers` index route added (mirrors the contracts list-route + `ensureQueryData` loader pattern); sidebar link verified; route tree regenerated. The route was previously unreachable — the list page component existed but no index route mapped to it, and the `customers-context` store it imports was never committed (empty directory). Both are now in place, and the full customers CRUD suite passes end-to-end.
- **P0-2 closed:** both `TabsContent value="list"` placeholder comments replaced with real, filterable tables on a shared `EntriesTable<T>` shell. Time entries: date, project, description (truncated), hours, status badge, view action. Expenses: date, project, category, description (mileage rows always surface km distance), amount + currency (resolved via the existing project→contract relationship), status badge, receipt indicator, view action. Filters (status multi-select, date range, category) are URL-shareable through `validateSearch` (ADR-FE-017) and survive reloads; client-side pagination at page size 25 (ADR-FE-018); row click opens the existing EntryDetail/ExpenseDetail surface on the calendar tab.
- **P0-4 closed:** shared `RouteError` panel (message, "Try again" via `router.invalidate()`, "Go to Today" link) registered as `errorComponent` on `_authenticated.tsx`; slim `AuthRouteError` variant on `_auth.tsx`. A failed loader now renders a recoverable panel instead of a blank screen, with dev-only console logging.
- **Shared component layer:** `components/shared/entries-table.tsx` (generic, type-safe, no `any`), `status-badge.tsx` (single source for the six states; `approved` recolored emerald so all six are distinct), `entries-filters.tsx` (URL-agnostic status multi-select + date-range picker).
- **Test coverage added:** 15 Vitest tests (shared table/badge, time-entries list via real router + MSW, expenses list via real router + MSW, error boundary forced-500 + retry) and a rewritten customers Playwright suite with sidebar→list→detail navigation coverage. Total suite: 75 Vitest + 17 Playwright tests green (projects/auth retain 2 pre-existing failures — see deferred-items).

## Task Commits

Each task was committed atomically:

1. **Task 1: `/customers` index route (P0-3)** - `4103ca1` (feat)
2. **Task 4: shared entries-table + status badge + filters** - `b27a847` (feat)
3. **Task 2: time-entries list view (P0-2 part 1)** - `78252a2` (feat)
4. **Task 3: expenses list view (P0-2 part 2)** - `79a4b0b` (feat)
5. **Task 5: route error boundaries (P0-4)** - `2eef3e2` (feat)

## Files Created/Modified

- `web/src/routes/_authenticated/customers/index.tsx` — index route with `ensureQueryData` loader, lazy-loaded component (autoCodeSplitting)
- `web/src/routes/_authenticated/customers/-context/customers-context.tsx` — zustand store (search/form/dialog/delete state) the page components referenced but was never committed
- `web/src/components/shared/entries-table.tsx` — generic `EntriesTable<T>` over shadcn Table primitives, client-side pagination
- `web/src/components/shared/status-badge.tsx` — shared six-state badge (approved = emerald); legacy route-local file re-exports it
- `web/src/components/shared/entries-filters.tsx` — `StatusFilterSelect` (base-ui DropdownMenu checkboxes) + `DateRangeFilter` (DayPicker range)
- `web/src/components/layout/route-error.tsx` — full `RouteError` + slim `AuthRouteError` with `router.invalidate()` retry
- `web/src/lib/list-filters.ts` — shared `entryStatusSchema` / `listStatusesSchema` (single/repeated/JSON-array forms)
- `web/src/routes/_authenticated/time-entries/-components/time-entries-list.tsx` + `expenses/-components/expenses-list.tsx` — the two list views
- `web/src/routes/_authenticated/time-entries/index.tsx`, `expenses/index.tsx` — validateSearch extended with list filters
- `web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx`, `expenses/-components/expenses-page.tsx` — controlled tabs; row click / New-entry → calendar tab + date param
- `web/src/routes/_authenticated.tsx`, `_auth.tsx` — `errorComponent` registration
- `web/src/api/customers.ts` — mutation invalidations fixed (4th-arg client destructure)
- `web/src/routeTree.gen.ts` — regenerated with `/customers/` route
- `web/e2e/customers.spec.ts` — sidebar→list→detail coverage; API-session login; finance promotion
- `web/src/lib/__tests__/setup.ts` — matchMedia polyfill for layout components
- `.gitignore` — playwright test-results/playwright-report output
- `web/src/components/shared/__tests__/entries-table.test.tsx`, `web/src/routes/_authenticated/time-entries/-components/__tests__/time-entries-list.test.tsx`, `web/src/routes/_authenticated/expenses/-components/__tests__/expenses-list.test.tsx`, `web/src/components/layout/__tests__/route-error.test.tsx` — 15 new tests

## Decisions Made

- **Expense currency:** the expense payload has no currency field; amounts render with the currency of the linked contract (project → contract map from existing endpoints), falling back to a 2-decimal number. No new endpoint invented (per plan constraint).
- **Status badge palette:** `approved` changed from blue to emerald so all six states are distinguishable (the plan requires distinct styling for all six).
- **List filters in the URL:** `validateSearch` carries `listStatuses`/`listCategory`/`listFrom`/`listTo`; the schema accepts a single param, repeated params, or TanStack's JSON-array serialization so shared links round-trip.
- **Detail navigation:** row click and the empty-state "New entry" affordance set the date search param and switch to the calendar tab, reusing the existing detail surfaces rather than duplicating them.
- **RFC3339 date handling:** the API returns `entry_date` as RFC3339; all list parsing accepts both RFC3339 and plain `yyyy-MM-dd` (a browser smoke with real data caught a crash from the naive `T00:00:00` append).
- **E2E session strategy:** the customers suite logs in once via API and injects cookies per test — the backend caps anonymous logins at 5/min per IP and the suite has 5 tests.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing customers-context store module**
- **Found during:** Task 1 (`/customers` route)
- **Issue:** `customers-page.tsx` and its dialogs import `../-context/customers-context` — the directory existed but the file was never committed (P3 gap). The route could not compile or render without it.
- **Fix:** Created the zustand store + provider following the org-hierarchy context pattern (search, form mode/state, delete dialogs).
- **Files modified:** `web/src/routes/_authenticated/customers/-context/customers-context.tsx`
- **Verification:** customers component type errors cleared (15 errors → 0 in that tree); e2e CRUD suite green.
- **Committed in:** `4103ca1` (Task 1)

**2. [Rule 1 - Bug] Customer mutations never invalidated the list**
- **Found during:** Task 1 (e2e create-customer flow)
- **Issue:** TanStack Query v5.101 passes the query client as the **4th** `onSuccess` argument; `customers.ts` destructured the 3rd (mutation context, usually `undefined`) → `client.invalidateQueries` threw inside the callback, was swallowed, and the list never refetched after create/edit/delete (POST 201, UI still empty).
- **Fix:** Switched to the 4-arg `(_, __, ___, { client })` destructure in `customers.ts`.
- **Files modified:** `web/src/api/customers.ts`
- **Verification:** POST /customers 201 now followed by a refetch; e2e create test passes.
- **Committed in:** `4103ca1` (Task 1)

**3. [Rule 1 - Bug] Status filter crashed at runtime (MenuGroupContext)**
- **Found during:** Task 2 (time-entries list verification)
- **Issue:** base-ui `Menu.CheckboxItem`/`GroupLabel` require a `Menu.Group` parent; `DropdownMenuLabel`/checkbox items as direct children of `DropdownMenuContent` threw "MenuGroupContext is missing". The task-4 commit shipped the unfixed file.
- **Fix:** Wrapped label, separator, and checkbox items in `DropdownMenuGroup`; landed with the Task 5 commit.
- **Files modified:** `web/src/components/shared/entries-filters.tsx`
- **Verification:** filter interaction tests pass; browser smoke narrows rows.
- **Committed in:** `2eef3e2` (Task 5)

**4. [Rule 1 - Bug] RFC3339 entry_date crash ("Invalid time value")**
- **Found during:** Task 2 (browser smoke with real API data)
- **Issue:** The API returns `entry_date` as RFC3339 (`2026-07-15T00:00:00Z`); the list appended `T00:00:00`, producing Invalid Date and blank-screen errors. Test fixtures used plain dates, hiding the bug.
- **Fix:** Date helpers accept both formats; filters compare `entry_date.slice(0, 10)`; row-click navigation uses `new Date(entry_date)` directly.
- **Files modified:** both list components + both pages; one fixture switched to RFC3339 to pin the path
- **Verification:** browser smoke with real data renders rows; 75/75 tests green.
- **Committed in:** `78252a2`, `79a4b0b` (Tasks 2–3)

**5. [Rule 3 - Blocking] E2E suite rate-limited by backend login cap**
- **Found during:** Task 1 (customers Playwright suite)
- **Issue:** The backend limits anonymous `POST /auth/login` to 5/min per IP; the 5-test customers suite (login per test) plus manual smoke runs tripped 429s, and the full suite cannot pass in one run.
- **Fix:** The customers suite logs in once via API in `beforeAll`, captures session cookies, and injects them per test; the spec also promotes its user to finance via the DB (customer CRUD is finance-only, registration assigns manager).
- **Files modified:** `web/e2e/customers.spec.ts`
- **Verification:** customers suite 5/5 green repeatedly.
- **Committed in:** `4103ca1` (Task 1)

**6. [Rule 2 - Missing Critical] Backend migration not applied during verification**
- **Found during:** Task 1 (smoke test setup)
- **Issue:** The local Postgres lacked Plan 08-01's `010_refresh_token_reuse_detection` migration; registration/login failed with a `family_id` column error.
- **Fix:** Ran `go run ./cmd/migrate -up -dir migrations` against the local DB (environment setup, not code).
- **Verification:** login 200, `/customers` 200.
- **Committed in:** n/a (environment)

---

**Total deviations:** 6 auto-fixed (3 bugs, 1 missing critical, 2 blocking)
**Impact on plan:** All fixes were necessary for the plan's acceptance criteria to hold (route reachability, list refetch, filter interaction, real-data rendering, e2e passability). No scope creep beyond the plan's files except the customers-context creation (required by the route the plan asked to add).

## Issues Encountered

- **Pre-existing backend bug discovered (out of scope):** `POST /expenses` always 500s — `expenses.unit_id` is `NOT NULL REFERENCES units(id)` but the create request chain carries no `unit_id`, so the service inserts the zero UUID (FK violation). Expense creation is broken for every org; the expenses list itself works. Logged to `deferred-items.md` with a suggested fix; the browser verification of the expenses list used seeded data instead.
- **Pre-existing mutation-invalidation bug in contracts/projects/units API modules** (same 3rd-arg destructure fixed in customers): logged to `deferred-items.md`.
- **Pre-existing e2e flakiness:** `waitForURL('/')` races the `/` → `/time-entries` redirect (Phase 01-02 behavior); projects.spec's `input[name="name"]` selector doesn't match the labeled field. Logged to `deferred-items.md`.
- **Frontend build debt:** ~85 pre-existing TypeScript errors in unrelated routes/tests mean `bun run build` (tsc) cannot pass project-wide; this plan's files are type-clean and the route tree regenerates via Vite. Logged to `deferred-items.md`.

## Authentication Gates

None — no external service credentials were required.

## Known Stubs

- Expenses amount falls back to a plain 2-decimal number when the project has no resolvable contract currency — the expense payload genuinely lacks currency; a stub only when contract linkage is absent.

## Next Phase Readiness

- P0-2, P0-3, P0-4 are closed with test coverage; the audit P0 table can read "Fixed" for these rows once the phase verification confirms.
- `/customers` unblocks the manual smoke for Plans 08-03/08-04 (org-hierarchy + approvals UI) in the same wave.
- The shared entries-table/status-badge/filter components are available to any future list view (exports, approvals).
- **Blockers for full-suite green:** the pre-existing backend expense-create bug and the e2e infra items in `deferred-items.md` should be scheduled (backend expense unit_id fix is the highest-value candidate — it blocks the expenses create workflow entirely).

## Self-Check: PASSED

All 10 key files exist on disk; all 5 task commits verified in git log; 75/75 Vitest tests pass; customers (5/5), time-entries (4/4), contracts (4/4), org-hierarchy (3/3) Playwright suites green.

---

*Phase: 08-pre-deployment-hardening-p0-audit-fixes*
*Completed: 2026-07-31*
