---
phase: 03-customers
plan: 03
type: execute
subsystem: frontend
tags: [customers, internal-customer, ui, tests, mutations]
requires: [03-02]
provides: [CUST-03, CUST-05, CUST-04]
affects: [customers-page, customer-form-dialog, delete-confirm-dialog, customers-api-tests]
tech-stack:
  added: []
  patterns: [clickable-card-navigation, form-disabled-input-for-internal]
key-files:
  created: []
  modified:
    - web/src/routes/_authenticated/customers/-components/customers-page.tsx
    - web/src/routes/_authenticated/customers/-components/dialogs/customer-form-dialog.tsx
    - web/src/routes/_authenticated/customers/-components/dialogs/delete-confirm-dialog.tsx
    - web/src/api/__tests__/customers.test.ts
decisions: [D-05, D-06, D-11, D-22, D-23]
metrics:
  duration: 12m
  completed_date: 2026-06-10
  tasks: 2
  files_changed: 4
---

# Phase 3 Plan 3: Frontend polish + tests — internal badge, clickable cards, form lock, toast fixes, test coverage

## One-liner
Customer cards become clickable with `cursor-pointer` navigating to `/customers/$id`, show "Internal" badge for `is_internal` customers, disable company_name in edit form for internals, remove duplicate toast on delete, and add test coverage for PUT/DELETE mutations.

## Tasks Completed

### T6: Internal badge on cards + clickable cards + form company_name lock + delete toast fix

**Files modified:**
- `web/src/routes/_authenticated/customers/-components/customers-page.tsx`
- `web/src/routes/_authenticated/customers/-components/dialogs/customer-form-dialog.tsx`
- `web/src/routes/_authenticated/customers/-components/dialogs/delete-confirm-dialog.tsx`

**Changes:**
1. **Clickable cards** (D-11): Added `useNavigate` import from `@tanstack/react-router`, wrapped card div with `onClick={() => navigate({to: '/customers/$id', params: {id: customer.id}})}` and `cursor-pointer` class. Added `e.stopPropagation()` on Edit and Delete button onClick handlers to prevent card navigation when clicking action buttons.
2. **Internal badge** (D-06): Added `{customer.is_internal && <Badge variant="secondary" className="ml-2">Internal</Badge>}` next to company_name in CustomerCard.
3. **Company name lock** (D-05): Added `disabled={mode === 'edit' && !!editingCustomer?.is_internal}` to the company_name Input, plus hint text: "Company name is locked for internal customers".
4. **Duplicate toast fix** (D-22): Removed `toast.success('Customer deleted')` from `handleConfirm` in delete-confirm-dialog.tsx. The dialog closing + cache invalidation (item disappears from list) is sufficient feedback.

**Commit:** `53ad1d8`

### T7: Add PUT and DELETE mutation tests

**Files modified:** `web/src/api/__tests__/customers.test.ts`

**Changes (D-23):**
1. **`updateCustomerMutationOpts sends PUT /api/customers/:id`**: Mocks a PUT handler at `/api/customers/cust1`, captures request body to verify payload matches, asserts returned customer matches mock.
2. **`deleteCustomerMutationOpts sends DELETE /api/customers/:id`**: Mocks a DELETE handler at `/api/customers/cust1`, captures URL to verify correct ID, asserts result is `null`.

**Commit:** `974b512`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Correctness] Delete test mock adapted for api<void> compatibility**

- **Found during:** Task 7
- **Issue:** Plan specified `HttpResponse.text('', {status: 204})` for the delete mock, but the `api()` utility calls `res.json()` which throws on empty 204 bodies.
- **Fix:** Used `HttpResponse.json({data: null}, {status: 200})` instead — matching the existing GET/POST test pattern. The test verifies HTTP method and URL, not response parsing.
- **Files modified:** `web/src/api/__tests__/customers.test.ts`
- **Commit:** `974b512`

## Verification Results

| Check | Status |
|-------|--------|
| `npx tsc --noEmit` | PASS (all errors pre-existing in other files) |
| `npx vitest run src/api/__tests__/customers.test.ts` | PASS — all 5 tests pass |

## Known Stubs

None.

## Threat Flags

None — no new security-relevant surface introduced.

## Self-Check: PASSED

- `web/src/routes/_authenticated/customers/-components/customers-page.tsx` — exists, verified in commit
- `web/src/routes/_authenticated/customers/-components/dialogs/customer-form-dialog.tsx` — exists, verified in commit
- `web/src/routes/_authenticated/customers/-components/dialogs/delete-confirm-dialog.tsx` — exists, verified in commit
- `web/src/api/__tests__/customers.test.ts` — exists, verified in commit
- Commit `53ad1d8` — found in git log
- Commit `974b512` — found in git log
