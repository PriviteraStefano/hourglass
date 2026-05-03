# Contracts Page Update Design

**Date**: 2026-05-04
**Goal**: Update the contracts page to match new backend features: edit, delete, mileage recalculation, customer association, and active/inactive status management.

## Approach

Extend existing components in place (approach A). Modify `contract-detail.tsx`, `contract-list.tsx`, and related files directly. This follows existing patterns and enables faster implementation.

## Section 1: Contract Detail Page - Edit Form

The detail page "Edit" button (currently disabled) becomes functional:

- The details card content is replaced with an inline form when Edit is clicked
- **Form fields**:
  - `name` — text input
  - `km_rate` — number input
  - `currency` — select (USD, EUR, GBP)
  - `governance_model` — select (creator_controlled, unanimous, majority)
  - `is_shared` — checkbox
  - `customer_id` — shadcn combobox populated from `GET /customers`
  - `is_active` — toggle switch
- **Save/Cancel buttons** at bottom of form
- On save: calls `PUT /contracts/:id` with all fields, shows success toast, reverts to detail view
- On cancel: reverts to detail view without saving
- Add `updateContractMutationOpts` to `ContractsApis`

## Section 2: Contract Detail Page - Actions Section

Below the details card, add an "Actions" section:

- **Status toggle**: Shows "Active" (green) / "Inactive" (gray) badge + toggle switch. On toggle: calls `PUT /contracts/:id` with `is_active` field only. Shows confirmation toast.
- **Recalculate Mileage**: Date picker input + "Recalculate" button. Sends `POST /contracts/:id/recalculate-mileage` with `from_date`. Shows toast with `recalculated_count`.
- **Delete**: Red "Delete" button.
  - Client-side check: if contract has time entries → disable with tooltip explaining deletion is blocked
  - If allowed: confirmation dialog, then `DELETE /contracts/:id`
  - On success: shows toast, redirects to contracts list

## Section 3: Contract List Page Updates

- Add `is_active` badge to each contract row (green "Active" / gray "Inactive")
- Add filter dropdown above the list: "All", "Active", "Inactive"
- Filter is client-side: filters existing `contracts` data by `is_active` field
- Update `Contract` type in `types/models.ts` to ensure `is_active` is included
- Active/inactive filter persists in URL search params alongside existing `tab` and `searchQuery`

## Section 4: Backend Changes Needed

- Add `DELETE /contracts/:id` endpoint in `internal/adapters/primary/http/contract.go`
  - Check if contract has time entries → return 409 Conflict if yes
  - Only allow finance role (reuse existing `middleware.GetRole` auth pattern)
  - Call `service.Delete()` in core service layer
- Add `Delete()` method to `internal/core/services/contract/contract.go`
- Add `Delete()` and `HasTimeEntries()` methods to `internal/core/ports/contract_repository.go` interface
- Repository implementation checks for linked time entries before deleting
- Update `ContractResponse` to include `time_entries_count` field for client-side delete guard (optional, since we check server-side too)

## Section 5: Customer API Support

- Verify `GET /customers` endpoint exists in backend (from recent customer management feature, issue #5)
- Frontend: add `CustomersApis.customersQueryOpts()` to new `web/src/api/customers.ts`
- Update `Contract` type to include `customer_id` and `customer_name` fields
- Combobox shows customer names, returns selected `customer_id` on selection

## Section 6: Component Files to Modify

**Frontend:**
- `web/src/types/models.ts` — add `customer_id`, `customer_name` to Contract interface
- `web/src/api/contracts.ts` — add `updateContractMutationOpts`, `deleteContractMutationOpts`
- `web/src/api/customers.ts` — new file for customer API calls (query options)
- `web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx` — add edit form, actions section, status toggle
- `web/src/routes/_authenticated/contracts/-components/contract-list.tsx` — add active/inactive badges and filter dropdown

**Backend:**
- `internal/adapters/primary/http/contract.go` — add Delete handler, register route
- `internal/core/services/contract/contract.go` — add Delete method
- `internal/core/ports/contract_repository.go` — add Delete and HasTimeEntries methods
- `internal/adapters/secondary/surrealdb/contract_repository.go` — implement new repository methods

## Data Flow

1. User navigates to contract detail → `GET /contracts/:id` returns contract with `customer_name`
2. User clicks Edit → details card replaced with form → Save → `PUT /contracts/:id` → invalidate query → show detail view
3. User toggles status → `PUT /contracts/:id` with `is_active` → invalidate query
4. User clicks Recalculate Mileage → date picker + button → `POST /contracts/:id/recalculate-mileage` → toast
5. User clicks Delete → check time entries (client-side) → confirm → `DELETE /contracts/:id` → redirect to list
6. List page filter → client-side filter on `is_active` field → URL search param persists

## Error Handling

- Edit/Update: Show toast on error, keep form open with current values
- Delete: If 409 Conflict (has time entries), show error toast and keep on detail page
- Mileage recalc: Show toast with error message from backend
- Customer combobox: Show loading/error state if `GET /customers` fails
