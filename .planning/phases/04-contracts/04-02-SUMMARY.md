---
phase: 04-contracts
plan: 02
subsystem: frontend
tags: [contracts, customers, combobox, tests]
requires:
  - phase: 04-01
    provides: backend customer_id on CreateContractRequest + HasProjects
provides:
  - Customer combobox in create contract dialog
  - Internal customer "(Internal)" suffix in combobox
  - "No customer" option in combobox
  - Frontend tests for customer_id create + undefined-customer case
affects:
  - Phase 05-projects (project create dialog also uses customer selection)
---

# Phase 4 — Plan 02: Frontend Customer Combobox & Tests

**One-liner:** Add customer combobox to CreateContractDialog with search, "No customer" option, and "(Internal)" suffix for internal customers, plus updated test coverage.

## Changes Made

### Files Modified

| File | Change | Lines |
|------|--------|-------|
| `web/src/types/api.ts` | Already had `customer_id?: string` on `CreateContractRequest` | 0 (pre-existing) |
| `web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx` | Added customer combobox with search, "(Internal)" suffix, "No customer" option | +44 |
| `web/src/api/__tests__/contracts.test.ts` | Updated create test with customer_id, added undefined-customer test | +32 |

### Type (`web/src/types/api.ts`)
- Already had **`customer_id?: string`** on `CreateContractRequest` from prior plan (04-01 backend work).
- *No changes needed — verified in place.*

### Create Contract Dialog
- Added **`useSuspenseQuery`** (from `@tanstack/react-query`) and **`CustomersApis`** imports for fetching customers.
- Added **`Combobox`**, **`ComboboxContent`**, **`ComboboxInput`**, **`ComboboxList`**, **`ComboboxItem`**, and **`useComboboxAnchor`** from `@/components/ui/combobox`.
- Added **`customerId`** state (default `""`) and **`comboboxAnchor`** ref.
- Fetches all org customers via **`useSuspenseQuery(CustomersApis.customersQueryOpts())`**.
- **Customer combobox field** placed after the KM Rate/Currency row, before Governance Model.
  - First combobox item: **"No customer"** (value `""`) — clears customer selection.
  - List items rendered from `customers` data, showing `company_name`.
  - Internal customers display **"(Internal)"** suffix in `text-xs text-muted-foreground`.
  - Searchable via `ComboboxInput` with placeholder "Search customers...".
- Updated **`handleSubmit`** to pass `customer_id: customerId || undefined`.
- Updated **`resetForm`** to reset `setCustomerId('')`.

### Frontend Tests
- **Updated** existing `createContractMutationOpts` test to include `customer_id: 'cust-1'` in both request and response.
- **Added** new test: `createContractMutationOpts without customer_id omits the field` — verifies `customer_id` is `undefined` when not provided.

## Tasks Executed

| Task | Name | Type | Commit | Files |
|------|------|------|--------|-------|
| 1 | Update frontend CreateContractRequest type | auto | Already committed (pre-existing) | `web/src/types/api.ts` |
| 2 | Add customer combobox to CreateContractDialog | auto | `a4efeec` | `create-contract-dialog.tsx` |
| 3 | Update frontend tests for customer_id in create flow | auto | `c8b042f` | `contracts.test.ts` |

## Verification

| Check | Result |
|-------|--------|
| `bunx vitest run src/api/__tests__/contracts.test.ts` | ✅ **5/5 passed** (28ms) |
| `bun run build` (TypeScript) | ⚠️ Pre-existing errors in unrelated files (auth test, projects, bootstrap, etc.) — none in our changed lines |

## Success Criteria

- ✅ Frontend tests pass (5/5) including new customer_id tests
- ✅ Customer combobox renders in create dialog after currency row, before governance model
- ✅ "No customer" is the first combobox item
- ✅ Internal customers show "(Internal)" suffix in muted text
- ✅ Creating contract with customer selected sends `customer_id` in POST body
- ✅ Creating contract with "No customer" omits `customer_id` from POST body

## Deviations from Plan

None — plan executed exactly as written.

## Pre-existing Build Issues (Out of Scope)

The `bun run build` command fails with TypeScript errors in multiple files across the project. These are pre-existing issues unrelated to this plan's changes:

- **`src/api/__tests__/*.test.ts`** — `queryFn` and `mutationFn` are optional in TanStack Query v5 types ("possibly undefined" / "Expected 1-2 arguments"). Affects `auth.test.ts`, `contracts.test.ts`, `customers.test.ts`, `projects.test.ts`, `time-entries.test.ts`. Pre-existing pattern.
- **`create-contract-dialog.tsx`** line 137 — `Select` `onValueChange` type mismatch with `Dispatch<SetStateAction<string>>`. Same issue in `create-project-dialog.tsx` and `edit-project-dialog.tsx`.
- **Other files** — Route type mismatches, missing environment types, etc. in `bootstrap-form.tsx`, `invitation-accept-form.tsx`, `login-form.tsx`, `password-reset-request-form.tsx`, `project-list.tsx`, `__root.tsx`, `api.ts`.

Tracked in: `.planning/phases/04-contracts/deferred-items.md`

## Requirements Satisfied
- **CTRT-02:** Customer combobox in create contract dialog with search, "No customer" option
- **CTRT-06:** Internal customers show "(Internal)" suffix in combobox items

## Commit History

| Hash | Message |
|------|---------|
| `a4efeec` | `feat(04-contracts-02): add customer combobox to CreateContractDialog` |
| `c8b042f` | `test(04-contracts-02): update create tests with customer_id cases` |

## Self-Check: PASSED

- ✅ `create-contract-dialog.tsx` — file exists with 224 lines (≥200 min_lines)
- ✅ `contracts.test.ts` — file exists with 125 lines (≥100 min_lines)
- ✅ `a4efeec` commit exists: `git log --oneline | grep a4efeec` → found
- ✅ `c8b042f` commit exists: `git log --oneline | grep c8b042f` → found
- ✅ Frontend tests pass: 5/5
- ✅ Combobox shows "No customer" as first option in JSX
- ✅ Combobox shows "(Internal)" suffix for internal customers in JSX
- ✅ `customer_id: customerId \|\| undefined` passed in handleSubmit
- ✅ `setCustomerId('')` in resetForm
