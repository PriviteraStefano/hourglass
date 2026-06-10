---
phase: 03-customers
plan: 02
type: execute
wave: 2
depends_on: [03-01]
requirements: [CUST-01, CUST-05, CUST-06]
tags: [frontend, api, sidebar, detail-page]
subsystem: customers
---

# Phase 3 Plan 2: Frontend Core — API Layer, Sidebar, Customer Detail Page

**Customer API module updated with `is_internal` and backend search, sidebar link added, client-side filter removed, customer detail page at `/customers/:id` with linked contracts.**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-06-10T20:39:56Z
- **Completed:** 2026-06-10T20:43:00Z
- **Tasks:** 2
- **Files modified:** 5 (3 modified, 2 created)

## Task Commits

1. **T4: API layer + sidebar + remove client-side filter** — `cd44fad`
   - Added `is_internal: boolean` to `Customer` interface
   - `customersQueryOpts` now accepts optional `search` param, sends `?search=` to backend
   - Removed all `toast.success()` calls from mutation `onSuccess` handlers
   - Added `['customers', id]` invalidation in `updateCustomerMutationOpts` and `deleteCustomerMutationOpts`
   - Removed `toast` import (no longer used)
   - Added `Building2` to sidebar imports and Customers nav link between Contracts and Org Hierarchy
   - Removed client-side filter block from customers-page, passes `searchQuery` to backend query

2. **T5: Customer detail page route + component** — `bcd417f`
   - Created `$id.tsx` route with loader pattern matching contract detail
   - Created `customer-detail.tsx` component with:
     - Back button navigating to `/customers`
     - Customer company name heading with internal badge
     - Info card: contact_name, email, phone, vat_number, address, status
     - Edit/Delete buttons (visible for finance role, navigating to `/customers`)
     - Linked contracts card with name, status, governance model
     - Empty state: "No linked contracts"

## Files Created/Modified

| File | Action |
|------|--------|
| `web/src/api/customers.ts` | Modified — `is_internal`, search param, no toasts, cache invalidation fix |
| `web/src/components/layout/sidebar.tsx` | Modified — Building2 import, Customers nav link |
| `web/src/routes/_authenticated/customers/-components/customers-page.tsx` | Modified — removed client-side filter, passed searchQuery |
| `web/src/routes/_authenticated/customers/$id.tsx` | Created — customer detail route |
| `web/src/routes/_authenticated/customers/-components/customer-detail.tsx` | Created — customer detail page component |

## Files Deleted

None.

## Decisions Made

- **Detail page is view-only:** Edit and Delete buttons navigate to `/customers` where the dialog-based CRUD lives. This avoids re-architecting the Zustand store for dialog state on the detail page.
- **Role-gated actions:** Edit/Delete buttons only visible for `finance` role, matching existing contract detail pattern.
- **Backend search replaces client-side filter:** The existing `?search=` ILIKE backend endpoint serves as the single search mechanism, triggered on every keystroke via React Query refetch.

## Known Stubs

None.

## Threat Flags

None.

## Deviations from Plan

None — plan executed exactly as written.

## Verification

- `npx tsc --noEmit` — no errors in modified/created files
- Pre-existing test file errors (`src/api/__tests__/*`) unrelated to this plan

## Self-Check: PASSED

- [x] `web/src/api/customers.ts` — `is_internal` added, search param works, no toasts, cache invalidation includes `['customers', id]`
- [x] `web/src/components/layout/sidebar.tsx` — Building2 import, Customers link between Contracts and Org Hierarchy
- [x] `web/src/routes/_authenticated/customers/-components/customers-page.tsx` — client-side filter removed, searchQuery passed to query
- [x] `web/src/routes/_authenticated/customers/$id.tsx` — route file exists with loader
- [x] `web/src/routes/_authenticated/customers/-components/customer-detail.tsx` — component exists with all required sections
- [x] Task 4 commit exists: `cd44fad`
- [x] Task 5 commit exists: `bcd417f`
