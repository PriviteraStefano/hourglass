---
phase: 10-information-architecture-implementation
reviewed: 2026-08-01T00:00:00Z
depth: standard
files_reviewed: 53
files_reviewed_list:
  - .planning/phases/10-information-architecture-implementation/deferred-items.md
  - internal/adapters/primary/http/expense.go
  - internal/adapters/primary/http/time_entry.go
  - internal/core/services/expense/expense.go
  - internal/core/services/time_entry/time_entry.go
  - web/e2e/activities.spec.ts
  - web/e2e/approvals.spec.ts
  - web/e2e/auth.spec.ts
  - web/e2e/customers.spec.ts
  - web/e2e/error-boundary.spec.ts
  - web/e2e/helpers.ts
  - web/e2e/working-groups.spec.ts
  - web/src/api/__tests__/activities.test.ts
  - web/src/api/activities.ts
  - web/src/api/working-groups.ts
  - web/src/components/layout/__tests__/route-error.test.tsx
  - web/src/components/layout/__tests__/sidebar-groups.test.tsx
  - web/src/components/layout/index.ts
  - web/src/components/layout/sidebar.tsx
  - web/src/lib/__tests__/role-visibility.test.ts
  - web/src/lib/api.ts
  - web/src/lib/role-visibility.ts
  - web/src/routes/_authenticated/-components/__tests__/today-page.test.tsx
  - web/src/routes/_authenticated/-components/today-page.tsx
  - web/src/routes/_authenticated/activities/-components/activity-detail.tsx
  - web/src/routes/_authenticated/activities/-components/activity-list.tsx
  - web/src/routes/_authenticated/activities/-components/create-activity-dialog.tsx
  - web/src/routes/_authenticated/activities/-components/edit-activity-dialog.tsx
  - web/src/routes/_authenticated/activities/$id.tsx
  - web/src/routes/_authenticated/activities/index.tsx
  - web/src/routes/_authenticated/approvals/-components/__tests__/approvals-page.test.tsx
  - web/src/routes/_authenticated/approvals/-components/approvals-page.tsx
  - web/src/routes/_authenticated/approvals/index.tsx
  - web/src/routes/_authenticated/contracts/-components/contract-list.tsx
  - web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
  - web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx
  - web/src/routes/_authenticated/customers/-components/customer-detail.tsx
  - web/src/routes/_authenticated/customers/-components/customers-page.tsx
  - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx
  - web/src/routes/_authenticated/expenses/index.tsx
  - web/src/routes/_authenticated/exports/-components/exports-page.tsx
  - web/src/routes/_authenticated/index.tsx
  - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
  - web/src/routes/_authenticated/time-entries/index.tsx
  - web/src/routes/_authenticated/working-groups/-components/delete-working-group-dialog.tsx
  - web/src/routes/_authenticated/working-groups/-components/working-group-form-dialog.tsx
  - web/src/routes/_authenticated/working-groups/-components/working-group-members-dialog.tsx
  - web/src/routes/_authenticated/working-groups/-components/working-groups-page.tsx
  - web/src/routes/_authenticated/working-groups/index.tsx
  - web/src/routeTree.gen.ts
  - web/src/types/api.ts
  - web/src/types/expense-types.ts
  - web/src/types/models.ts
findings:
  critical: 2
  warning: 8
  info: 9
  total: 19
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-08-01T00:00:00Z
**Depth:** standard
**Files Reviewed:** 53
**Status:** issues_found

## Summary

Reviewed the Phase 10 surface (activities rename + approvals + working groups + Today landing) at standard depth: 5 backend Go files (2 HTTP handlers, 2 services, 1 planning doc), 7 e2e specs/helpers, 6 unit-test files, and 33 frontend source files. The frontend work is generally clean — sensible TanStack Query/Router patterns, well-scoped role-visibility logic, and good test coverage of the enabled-gating contracts.

Two critical backend issues surfaced, both in the in-scope handler/service files (the repositories were read to confirm the behavior):

1. **`GET /time-entries` and `GET /expenses` leak the entire org to any authenticated member.** The handlers populate `filters.Role` / `filters.RequestUserID`, but the repository `List` query builders never use those fields — no role or ownership scoping is applied. This contradicts the `Get` handlers, which do enforce `role == "employee" && e.UserID != userID → 403`.
2. **`POST /expenses/{id}/receipt` is an unauthenticated-mutation IDOR.** `ReceiptUpload` (and `Service.SetReceiptURL`) never check expense org membership, ownership, or status, so any authenticated user can attach a receipt URL to any expense by ID — including expenses of other orgs. The uploaded files are additionally never served (no static handler for `uploads/`), so stored receipt links 404.

Several business-rule bypasses via `PUT` (hours ≤ 0, amount ≤ 0, locked-period escape) and an approval-role inconsistency (WG managers can view the pending queue but the `Approve`/`Reject` handlers 403 them) round out the warnings.

## Critical Issues

### CR-01: List endpoints return every org member's time entries and expenses to any authenticated user

**File:** `internal/adapters/primary/http/time_entry.go:51-55` (and `internal/adapters/primary/http/expense.go:64-68`)
**Issue:** Both handlers construct `ports.ListFilters` with `Role` and `RequestUserID` set from the JWT context, signalling the intent to scope results by role/ownership. However, the repository query builders (`buildTimeEntryListQuery` in `internal/adapters/secondary/postgres/time_entry_repository.go`, and the expense equivalent) never reference `filters.Role` or `filters.RequestUserID` — confirmed by grep across both repo files. The queries filter only by org + optional client-supplied filters (`user_id`, `status`, `month`…). Since `GET /time-entries` and `GET /expenses` are registered with only `middleware.Auth` (cmd/server/main.go:214,225), a plain employee can list every org member's entries and expenses (hours, amounts, activity names, statuses, receipt URLs). This is a broken-access-control / data-exposure defect, and it is inconsistent with the `Get` handlers, which do enforce `if role == "employee" && e.UserID != userID → 403` (time_entry.go:113-116, expense.go:127-130).
**Fix:** Enforce ownership scoping in the repository when the role is `employee` (and decide the policy for `customer`/`hr`), e.g. in `buildTimeEntryListQuery`:
```go
// after the org condition:
if filters.Role == "employee" {
    args = append(args, filters.RequestUserID)
    conditions = append(conditions, fmt.Sprintf("te.user_id = $%d", len(args)))
}
```
Apply the same to `buildExpenseListQuery`. Alternatively scope in the service layer; do not rely on the client-supplied `user_id` filter for authorization (it is attacker-controlled).

### CR-02: Receipt upload mutates any expense without org/owner/status authorization (IDOR)

**File:** `internal/adapters/primary/http/expense.go:480-548`
**Issue:** `ReceiptUpload` parses the target expense ID from the path and calls `h.service.SetReceiptURL(ctx, expenseID, receiptURL)` with no checks that the expense (a) belongs to the caller's org, (b) is owned by the caller, or (c) is in an editable status. `Service.SetReceiptURL` (internal/core/services/expense/expense.go:403-414) also performs no authorization — it fetches by ID, sets `ReceiptURL`, and updates. Consequences:
- Any authenticated user can attach a receipt URL to **any** expense ID, including expenses belonging to other orgs (cross-org data tampering; the file itself lands under the caller's org dir but the DB row mutated is arbitrary).
- A rejected/approved/foreign expense's receipt can be overwritten at will.
Additionally, the uploaded file is written to `uploads/receipts/{org}/{expense}/` but the server has **no static handler** for `/uploads` anywhere in `cmd/` or `internal/` (verified by grep) — the stored `receipt_url` is a relative path that the frontend renders as `href={expense.receipt_url}` (expense-row.tsx:160), so the receipt link always 404s.
**Fix:** In `ReceiptUpload`, before writing the file, resolve the expense and enforce org membership + ownership + draft/editable status (mirror the `Update` handler's checks); pass `userID`/`orgID` into `SetReceiptURL` and enforce them in the service. For the 404: either register a `GET /uploads/{file...}` handler that serves files after an org/ownership check, or store an absolute/cdn URL.
```go
e, err := h.service.Get(ctx, expenseID)
if err != nil { /* 404 */ }
if e.OrgID != orgID { /* 404 */ }
if e.UserID != userID { /* 403 */ }
// ... then upload + SetReceiptURL
```

## Warnings

### WR-01: WG managers can view the pending queue but can never approve/reject from it

**File:** `internal/adapters/primary/http/time_entry.go:341-350` (and `expense.go:354-363`)
**Issue:** `ListPending` deliberately admits working-group managers/delegates with org-role `employee` (passing `role = "wg_manager"`), and `Service.Approve` explicitly authorizes manager-stage approval for members of `res.approverIDs` (WG manager + delegates). But the `Approve`/`Reject` HTTP handlers 403 every caller whose role is not `manager` or `finance` *before* reaching the service. A WG manager with org-role employee therefore sees the queue (T-10-05-3) but every action 403s — the "approver set" can look but never act. Either the handler should admit WG managers (resolve via `IsWGManager` and pass the resolved role, as `ListPending` does), or the queue gate should be narrowed; the current state is inconsistent with the documented design intent in the `ListPending` comments ("mirroring Service.Approve's resolveManagerStage path").
**Fix:** In `Approve`/`Reject` handlers, mirror the `ListPending` gate: if role is not manager/finance, check `IsWGManager` and, if true, delegate to the service with the WG role so `Service.Approve`'s `approverIDs` check can authorize.

### WR-02: PUT endpoints bypass create-time value validation (hours ≤ 0, amount ≤ 0)

**File:** `internal/adapters/primary/http/time_entry.go:238-244`, `internal/adapters/primary/http/expense.go:249-251`
**Issue:** `Create` enforces `req.Hours > 0` and `req.Amount > 0`, but `Update` only bounds hours at 24 (`*req.Hours > 24`) and has no amount check at all; neither service `Update` re-validates (time_entry service:93-95, expense service:100-102). A client can PUT `{"hours": -3}` or `{"amount": 0}` and persist invalid values that the list UI will display and the workflow will carry.
**Fix:** Mirror the create bounds in the update handlers (and/or service): `if req.Hours != nil { if *req.Hours <= 0 || *req.Hours > 24 { 400 } }` and `if req.Amount != nil { if *req.Amount <= 0 { 400 } }`.

### WR-03: Update can move an entry/expense into a locked financial period

**File:** `internal/core/services/time_entry/time_entry.go:74-109`, `internal/core/services/expense/expense.go:78-119`
**Issue:** `Create` checks `IsPeriodLocked` (org + activity + date), but neither `Update` re-checks it when `Date` and/or `ActivityID` change. A locked period can be trivially bypassed: create in an open period, then PUT the date into the locked one (or move to a locked activity). `IsPeriodLocked` is date+activity-scoped (time_entry_repository.go:248), so a lock is only as good as the update path's enforcement.
**Fix:** In both `Update` services, when `Date` or `ActivityID` is being changed, call `repo.IsPeriodLocked(ctx, e.OrgID, effectiveActivityID, effectiveDate)` and return `ErrPeriodLocked` (map to 400 in the handlers, which already handle it).

### WR-04: Expense approval routing — unit-manager (R-2) fallback is unreachable because expenses are never assigned a unit

**File:** `internal/adapters/primary/http/expense.go:181-190`, `internal/core/services/expense/expense.go:40-76`
**Issue:** `CreateExpenseRequest` has no `unit_id` (unlike time entries), and neither the handler nor the service sets `e.UnitID`. The repo inserts the zero UUID (expense_repository.go Create inserts `e.UnitID` verbatim). `resolveManagerStage` then calls `resolveUnitManager(ctx, uuid.Nil)` for personal activities, which resolves no manager → every WG-less personal expense falls to the terminal `roleGated` case (any org-role manager) instead of the submitter's unit manager. The documented R-2 fallback ("submitter's unit manager, walking the unit tree upward") is dead code for expenses, so approval routing diverges from time entries for the same activity. Either derive the unit from the activity at create time (as time entries do), or drop the fallback and document role-gating for expenses.
**Fix:** Resolve and persist `UnitID` at expense creation (e.g., from the activity's anchor unit, or require `unit_id` in the create payload like time entries) so `resolveManagerStage`'s fallback path is reachable.

### WR-05: 409 handling in contract delete is dead code — the API client never attaches `.status`

**File:** `web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx:134-141`
**Issue:** `handleDelete` checks `(e as { status?: number }).status === 409` to show "Cannot delete contract with time entries", but `web/src/lib/api.ts:82-87` throws `new Error(error.message || ...)` without attaching a status. `err.status` is always `undefined`, so every failure (including 409) shows the generic "Failed to delete contract" and the specific branch is unreachable.
**Fix:** Attach the status in the client (`const err = new Error(...) as Error & { status?: number }; err.status = res.status; throw err;`) or parse the message. Then the 409 branch works.

### WR-06: Toggling contract active state drops `customer_id` from the PUT payload

**File:** `web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx:96-118`
**Issue:** `handleToggleActive` spreads `...formData` (empty when not editing) and sends `name`, `km_rate`, `currency`, `governance_model`, `is_shared`, `is_active` — but **not** `customer_id`. If the contract update is full-replace (PUT with the shared `UpdateContractRequest`), toggling active on a contract linked to a customer silently detaches the customer. It also silently preserves a stale `formData` if a user edits, cancels… (cancel resets to `{}`, so the main risk is the omitted `customer_id`).
**Fix:** Include `customer_id: c.customer_id` in the toggle payload (and prefer a dedicated PATCH or explicit full-state object rather than spreading `formData`).

### WR-07: E2E seed helpers hard-code July 2026 dates — suites break on month rollover

**File:** `web/e2e/helpers.ts:144-149, 169-173`
**Issue:** `seedTimeEntries` and `seedExpenses` hard-code `2026-07-15..20` / `2026-07-10..14`. The list views default to the current month, so after the calendar rolls to August these suites fail with "No time entries in this period." This is documented in deferred-items.md §2, but the defect lives in the reviewed file and will recur every month. `approvals.spec.ts` already seeds current-month dates — the helpers should do the same.
**Fix:** Compute dates relative to the current month, e.g. `const d = (day: number) => \`2026-08-${String(day).padStart(2, "0")}\`` — or better, derive from `new Date()` so the suites stay green indefinitely.

### WR-08: Uploaded receipt files are never served — stored receipt URLs 404

**File:** `internal/adapters/primary/http/expense.go:513-535`
**Issue:** `ReceiptUpload` persists files under `uploads/receipts/{org}/{expense}/` and stores the relative path in `receipt_url`, but no route serves `/uploads` (verified: no `http.FileServer`/`ServeFile`/`StripPrefix` anywhere in `cmd/` or `internal/`). The frontend renders `href={expense.receipt_url}` (expense-row.tsx:160), which resolves against the app origin and 404s. Receipts are effectively unusable end-to-end.
**Fix:** Register an authenticated static route for uploaded receipts (with org/ownership checks per file, since the path embeds org id), or store a fully-qualified URL to a real object store.

## Info

### IN-01: Reject handlers ignore the JSON decode error

**File:** `internal/adapters/primary/http/expense.go:404-405`, `internal/adapters/primary/http/time_entry.go:390-391`
**Issue:** `json.NewDecoder(r.Body).Decode(&req)` errors are discarded; a malformed body silently yields `reason = ""` and the rejection proceeds with an empty reason. The frontend requires a reason ≥ 10 chars, but the API accepts none.
**Fix:** Handle the decode error with a 400, or explicitly document that reason is optional server-side.

### IN-02: Dead code — `timePtr` unused in both services

**File:** `internal/core/services/time_entry/time_entry.go:412`, `internal/core/services/expense/expense.go:427`
**Issue:** `timePtr` is defined but never referenced anywhere (grep confirms). Dead code in both services.
**Fix:** Delete both helpers.

### IN-03: `ExpenseCategory` union duplicated across two type files

**File:** `web/src/types/models.ts:212-221`, `web/src/types/expense-types.ts:3-12`
**Issue:** The identical `ExpenseCategory` union is defined and exported twice; the two can drift independently.
**Fix:** Define once (e.g., in `expense-types.ts`) and re-export from `models.ts`.

### IN-04: Customer detail Edit/Delete buttons are stubs that just navigate to the list

**File:** `web/src/routes/_authenticated/customers/-components/customer-detail.tsx:48-64`
**Issue:** For finance users, the "Edit" and "Delete" buttons both `navigate({ to: "/customers" })` — a destructive-looking Delete that does nothing is misleading. (The customers.spec even documents this: "the detail page's own Edit button is a list-redirect stub".)
**Fix:** Wire them to the real form/delete dialogs, or remove the buttons until implemented.

### IN-05: Unhandled promise rejection on failed working-group delete

**File:** `web/src/routes/_authenticated/working-groups/-components/delete-working-group-dialog.tsx:39-42`
**Issue:** `handleConfirm` awaits `deleteWg.mutateAsync(wg.id)` with no try/catch. On failure the mutation's `onError` toast fires but the rejection is unhandled (console noise); the dialog stays open, which is fine, but the rejection should be caught.
**Fix:** Wrap in try/catch (or use `mutate` + `onSettled`).

### IN-06: Recalculate-mileage reads the date input via `document.getElementById`

**File:** `web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx:371-386`
**Issue:** `document.getElementById("recalc-date")` can be null (cast hides it), and the input is uncontrolled (`defaultValue`), so the value is not reset/re-synced across renders.
**Fix:** Use a controlled state or a ref; guard the null case.

### IN-07: "Welcome to Hourglass" empty state shows for users who only have expenses

**File:** `web/src/routes/_authenticated/-components/today-page.tsx:116-117, 189-208`
**Issue:** `hasAnyData` is derived only from the time-entries month query (`monthEntries.length > 0`). A user with expenses but zero time entries is not an approver → sees the "Welcome to Hourglass — start by logging time" state even though they have data.
**Fix:** Also check the expenses month query (or drop the copy to be time-centric explicitly).

### IN-08: "Adoption Count —" placeholder in activity detail

**File:** `web/src/routes/_authenticated/activities/-components/activity-detail.tsx:192-197`
**Issue:** Shared activities render a hardcoded `—` for adoption count instead of the real value (contracts detail shows `c.adoption_count ?? 0`).
**Fix:** Use the real count from the payload or remove the row.

### IN-09: Edit WG dialog lets the user change the activity, but the change is never submitted

**File:** `web/src/routes/_authenticated/working-groups/-components/working-group-form-dialog.tsx:96-108`
**Issue:** In edit mode the Activity combobox is editable, but the update payload (`UpdateWorkingGroupRequest`) contains no `subproject_id`, so a selected new activity is silently ignored on save — the UI accepts the change and the server never sees it.
**Fix:** Either disable the activity picker in edit mode or include `subproject_id` in the update payload.

---

_Reviewed: 2026-08-01T00:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
