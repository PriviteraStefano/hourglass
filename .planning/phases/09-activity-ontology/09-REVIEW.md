---
phase: 09-activity-ontology
reviewed: 2026-07-31T21:10:00Z
depth: standard
files_reviewed: 42
files_reviewed_list:
  - cmd/server/main.go
  - cmd/server/main_test.go
  - internal/adapters/primary/http/activity_handler.go
  - internal/adapters/primary/http/activity_handler_test.go
  - internal/adapters/primary/http/expense.go
  - internal/adapters/primary/http/handler_integration_test.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/time_entry.go
  - internal/adapters/primary/http/time_entry_test.go
  - internal/adapters/primary/http/validate_test.go
  - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
  - internal/adapters/secondary/postgres/activity_repository.go
  - internal/adapters/secondary/postgres/activity_repository_test.go
  - internal/adapters/secondary/postgres/contract_repository.go
  - internal/adapters/secondary/postgres/expense_repository.go
  - internal/adapters/secondary/postgres/export_repository.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  - internal/adapters/secondary/postgres/time_entry_repository.go
  - internal/adapters/secondary/postgres/working_group_repository.go
  - internal/core/domain/activity/activity.go
  - internal/core/domain/expense/expense.go
  - internal/core/domain/time_entry/time_entry.go
  - internal/core/ports/activity_repository.go
  - internal/core/ports/expense_repository.go
  - internal/core/ports/time_entry_repository.go
  - internal/core/services/activity/activity.go
  - internal/core/services/activity/activity_test.go
  - internal/core/services/expense/expense.go
  - internal/core/services/expense/expense_test.go
  - internal/core/services/testdata/factories.go
  - internal/core/services/testdata/mocks.go
  - internal/core/services/testdata/mocks_test.go
  - internal/core/services/time_entry/time_entry.go
  - internal/core/services/time_entry/time_entry_test.go
  - internal/core/services/working_group/working_group_integration_test.go
  - migrations/011_activity_ontology.down.sql
  - migrations/011_activity_ontology.up.sql
  - migrations/012_staffing_schema.down.sql
  - migrations/012_staffing_schema.up.sql
  - migrations/013_activity_kind_phase_fix.down.sql
  - migrations/013_activity_kind_phase_fix.up.sql
findings:
  critical: 3
  warning: 11
  info: 4
  total: 18
status: issues_found
---

# Phase 9: Code Review Report

**Reviewed:** 2026-07-31T21:10:00Z
**Depth:** standard
**Files Reviewed:** 42
**Status:** issues_found

## Summary

Reviewed the activity-ontology rewrite (migrations 011–013), the new hexagonal activity/expense/time-entry services, repositories, handlers, and their tests. The migration work (011 up/down cycle, 012 staffing, 013 label fix) is thorough and well-tested. The service-layer approval routing (R-1/R-2/R-3, D-11 skip, cycle prevention) is coherent and well covered by unit tests.

However, the HTTP boundary introduced several authorization gaps that did not exist in the legacy handlers:

- **CR-01** — Employee visibility scoping was dropped from the time-entry/expense List endpoints (the old handlers filtered `te.user_id` for the employee role; the new query builders ignore `Role`/`RequestUserID` entirely). Every employee can now read the whole org's timesheets and expenses.
- **CR-02** — `POST /expenses/{id}/receipt` performs an unauthenticated cross-org write: no ownership or org check before writing the file and mutating `receipt_url`.
- **CR-03** — `GET /activities/{id}` and `GET /activities/{id}/children` are not org-scoped at all (`Get` uses orgID only for the `is_adopted` flag), leaking any org's activity tree — including `budget_amount` and the derived customer id — to any authenticated user.

The three criticals share a root cause: the hex rewrite moved authorization to "the service/repo layer" but the service/repo layer never implemented it, and the handler tests exercise only invalid-input paths with nil services plus happy-path integration tests that never assert cross-user isolation.

## Critical Issues

### CR-01: Employee data-visibility scoping dropped from List endpoints (authorization regression)

**File:** `internal/adapters/primary/http/time_entry.go:45-83`, `internal/adapters/primary/http/expense.go:58-96`, `internal/adapters/secondary/postgres/time_entry_repository.go:97-155`, `internal/adapters/secondary/postgres/expense_repository.go:106-162`
**Issue:** The legacy handler (`internal/handlers/time_entry_handler.go`, removed in this phase) forced the employee role to `AND te.user_id = <self>` on List, and rejected `?user_id=` targeting another user. In the rewrite, the handlers populate `filters.Role` and `filters.RequestUserID`, but `buildTimeEntryListQuery` / `buildExpenseListQuery` never read those fields. Result: any authenticated employee can list the entire org's time entries and expenses, and `?user_id=<any-member>` retrieves any specific member's entries. The `role == "employee"` restriction in `Get` (time_entry.go:113, expense.go:127) still exists, so the List gap is a plain regression, not a deliberate change. The `customer` and `hr` roles are likewise unrestricted.
**Fix:** Apply role scoping in the query builders (mirroring the old handler):

```go
// in buildTimeEntryListQuery / buildExpenseListQuery, after org/is_deleted conditions:
if filters.Role == "employee" {
    args = append(args, filters.RequestUserID)
    conditions = append(conditions, fmt.Sprintf("te.user_id = $%d", len(args)))
}
```
and in the handlers, when `role == "employee"` and a `user_id` query param is present, reject with 403 unless it equals the caller (as the old handler did). Add an integration test asserting an employee cannot see another employee's entries via List.

### CR-02: ReceiptUpload / SetReceiptURL perform an unauthenticated cross-org write (IDOR)

**File:** `internal/adapters/primary/http/expense.go:462-531`, `internal/core/services/expense/expense.go:370-380`
**Issue:** `ReceiptUpload` takes the expense id from the path and the org from the caller's context, but never verifies that the expense belongs to that org (or that the caller owns it, or holds manager/finance role). `Service.SetReceiptURL` likewise does `GetByID` → set URL → `Update` with no ownership/org check. Any authenticated user can therefore attach (and overwrite) the receipt of any expense in any org, and the file lands on disk under the caller's own org directory while the victim expense's `receipt_url` points at it. The extension check (`filepath.Ext`) is also the only content gate — no magic-byte check — though no static file server is currently wired, limiting the XSS angle.
**Fix:** Authorize before writing:

```go
// service level
func (s *Service) SetReceiptURL(ctx context.Context, orgID, userID uuid.UUID, role string, id uuid.UUID, receiptURL string) (*expense.Expense, error) {
    e, err := s.repo.GetByID(ctx, id)
    if err != nil { return nil, err }
    if e.OrgID != orgID { return nil, expense.ErrExpenseNotFound }
    if e.UserID != userID && role != "manager" && role != "finance" {
        return nil, expense.ErrForbidden
    }
    ...
}
```
Also delete the written file if the DB update fails (currently the file is written before the expense is even looked up).

### CR-03: Activity Get / ListChildren endpoints are not org-scoped (cross-org read)

**File:** `internal/adapters/secondary/postgres/activity_repository.go:108-118` and `:164-172`, `internal/adapters/primary/http/activity_handler.go:168-209` and `:334-346`, `internal/core/services/activity/activity.go:42-44,53-55`
**Issue:** `ActivityRepository.Get` builds `baseActivityQuery() + WHERE a.id = $2` — orgID is used only for the `is_adopted` EXISTS subquery, never as a predicate. `ListChildren` has no org predicate at all. The handler detail endpoint then returns the full `ActivityDetailResponse` (activity + ancestry + commercial context including the derived `customer_id` + resolved billability) and `GET /activities/{id}/children` returns the child list with `contract_name`, `adoption_count`, `budget_amount`, etc. Any authenticated user of any org can read any org's activity tree by id. Unlike `List` (which scopes owned/adopted/shared), there is no relationship check — not even for shared/adopted activities.
**Fix:** Gate by org membership (owned, adopted, or shared — same rule as List):

```go
// repo Get: add an ownership/adoption visibility predicate
query := baseActivityQuery() + ` WHERE a.id = $2 AND (
    a.created_by_org_id = $1
    OR a.is_shared = true
    OR EXISTS(SELECT 1 FROM activity_adoptions aa WHERE aa.activity_id = a.id AND aa.organization_id = $1)
)`
```
Pass `orgID` into `ListChildren` through the service (`Service.ListChildren(ctx, orgID, parentID)`) and add the same predicate. Add an integration test with two orgs asserting org B gets 404 on org A's activity id and an empty children list.

## Warnings

### WR-01: Entry Create/Update never validate that the activity/unit belongs to the caller's org

**File:** `internal/adapters/primary/http/time_entry.go:156-175,222-237`, `internal/adapters/primary/http/expense.go:175-190,238-245`, `internal/core/services/time_entry/time_entry.go:41-72`, `internal/core/services/expense/expense.go:40-76`
**Issue:** `activity_id` (and `unit_id`) are UUID-parsed but never checked against the caller's org before insert/update. The legacy handler validated project/subproject/wg membership. A user can create time entries/expenses referencing another org's activity or unit; the entry row carries the caller's org_id but points at foreign entities (the `LEFT JOIN` in List then returns that foreign activity's name/kind). Update can move an existing entry onto a foreign activity/unit the same way.
**Fix:** In the service Create/Update, `activityRepo.Get(ctx, orgID, activityID)` (and unit lookup for `unit_id`) and return `ErrInvalidRequest`/404 when not owned/adopted/shared.

### WR-02: Update validation allows `hours <= 0` and expense `amount <= 0`

**File:** `internal/adapters/primary/http/time_entry.go:238-244`, `internal/adapters/primary/http/expense.go:249-251`, `internal/core/services/expense/expense.go:100-102`
**Issue:** Create rejects `hours <= 0 || hours > 24` and `amount <= 0`, but Update checks only `hours > 24` (0 and negative hours pass) and the expense Update path validates neither handler-side nor service-side (a negative `amount` is persisted).
**Fix:** `if *req.Hours <= 0 || *req.Hours > 24 { 400 }` in the time-entry Update handler, and `if *req.Amount <= 0 { 400 }` in the expense Update handler (or in the services, which is the better single place).

### WR-03: Nullable activity fields can never be cleared via the Update API

**File:** `internal/adapters/primary/http/activity_handler.go:48-59,240-280`, `internal/adapters/secondary/postgres/activity_repository.go:423-472`
**Issue:** `UpdateActivityRequest` uses `*string`/`*bool`/`*float64` with `omitempty`, so an explicit JSON `null` is indistinguishable from an absent field. Combined with the repo's `if req.Name != ""` / `if req.Description != ""` guards, it is impossible to: re-root an activity (clear `parent_id`), detach it from a contract (clear `contract_id`), reset `billable` to NULL (inherit), clear `budget_amount`, or clear the description. Sending `""` for contract_id fails UUID parse with 400 instead.
**Fix:** Use explicit presence detection for nullable clears — e.g. `*string` request fields with a documented sentinel (or `json.RawMessage`), and repo handling for "present but empty" meaning SQL NULL/''.

### WR-04: Activity Update skips kind-catalog and governance-model validation → 500 on invalid values

**File:** `internal/core/services/activity/activity.go:94-111`, `internal/adapters/primary/http/activity_handler.go:263-265`
**Issue:** `Service.Create` validates `KindExists` and `GovernanceModel.IsValid()`, but `Service.Update` validates neither. Updating `kind` to a value absent from the org's `activity_kinds` catalog or `governance_model` to an invalid string trips the DB FK/CHECK and surfaces as HTTP 500 ("failed to update activity") instead of a clean 400.
**Fix:** Mirror Create's validation in Update (`KindExists` when `req.Kind != ""`, `IsValid()` when `req.GovernanceModel != nil`), returning `ErrInvalidRequest`.

### WR-05: Invalid date strings map to HTTP 500 instead of 400

**File:** `internal/core/services/time_entry/time_entry.go:50-53`, `internal/core/services/expense/expense.go:53-56`, `internal/adapters/primary/http/time_entry.go:177-185`, `internal/adapters/primary/http/expense.go:192-200`
**Issue:** `time.Parse("2006-01-02", req.Date)` failure returns a plain wrapped error; both handlers only special-case `ErrPeriodLocked`, so a malformed client date yields 500 "failed to create/update entry". Same on the Update paths (services at time_entry.go:99-105, expense.go:109-115).
**Fix:** Return a domain sentinel (e.g. `ErrInvalidDate`) from the services and map it to 400 in the handlers.

### WR-06: Period lock enforced on Create only — Update/Submit/Approve bypass it

**File:** `internal/core/services/time_entry/time_entry.go:74-109,228-262`, `internal/core/services/expense/expense.go:78-119,233-267`
**Issue:** `IsPeriodLocked` is checked in Create only. A draft/submitted entry whose date falls inside a locked period can be edited (including moving the date), submitted, and approved. For a financial cutoff ("freeze") mechanism this defeats the lock's purpose.
**Fix:** Check `IsPeriodLocked` in `Update` (when the date changes) and in `Submit`/`Approve`, mirroring Create.

### WR-07: ListPending misses the R-2 unit-manager fallback stage

**File:** `internal/adapters/secondary/postgres/time_entry_repository.go:265-301`, `internal/adapters/secondary/postgres/expense_repository.go:271-307`
**Issue:** The manager branch of `ListPending` filters by `e.activity_id IN (SELECT wg.activity_id FROM working_groups wg WHERE wg.manager_id = $2 OR $2 = ANY(wg.delegate_ids))`. Entries routed to a unit manager via the R-2 fallback (personal activity, no WG) — and the role-gated terminal case — never appear in the manager's pending inbox. The direct Approve endpoint resolves those approvers correctly (service), so the workflow works only if the approver happens to know the entry id; the "pending" surface silently misses them.
**Fix:** Extend the manager branch to also include entries whose activity chain resolves to a unit-tree manager (same upward walk as `resolveUnitManager`), or have the repo return entries for which the requesting user is the resolved manager-stage approver.

### WR-08: New orgs can never create activities (no way to populate activity_kinds)

**File:** `migrations/011_activity_ontology.up.sql:36-41`, `internal/core/services/activity/activity.go:64-90`, `internal/adapters/primary/http/activity_handler.go:349-357`
**Issue:** `activity_kinds` is seeded only for the MVP org (hard-coded org id `019df8b0-0001-7000-8000-000000000001`) and the "General & Admin" fallback. There is no repository method, handler, or route to insert kinds (`GET /activity-kinds` is read-only; no `POST /activity-kinds` anywhere). Since `Service.Create` rejects any kind not in the catalog, every activity creation for a newly registered org fails with 400 "invalid activity payload" — the domain comment ("orgs extend the catalog with their own kinds", D-2) describes capability that does not exist. Integration tests only pass because `seedKind` inserts rows behind the API's back.
**Fix:** Either seed the four canonical kinds for every org at registration/bootstrap, or add an org-admin endpoint to manage `activity_kinds` (and wire it in `cmd/server/main.go`). At minimum, document the restriction explicitly.

### WR-09: SetupTestSchema swallows migration errors

**File:** `internal/adapters/secondary/postgres/exported_test_helpers.go:60-75`
**Issue:** On migration failure the helper logs (`t.Logf`) and continues. With 011–013 now doing stateful schema rewrites, a mid-chain failure leaves a partially-migrated schema and every downstream test fails with obscure "relation does not exist" errors instead of pointing at the broken migration. The newer `applyMigrations` helper (activity_ontology_migration_test.go:28-47) correctly uses `require.NoError` — the shared helper should match.
**Fix:** `require.NoError(t, err, "migration %s failed: %v", filepath.Base(f), err)` instead of `t.Logf`.

### WR-10: Handler integration fixture never wires the expense handler

**File:** `internal/adapters/primary/http/handler_test_helper.go:83-96,176-190`
**Issue:** `newHandlerFixture` builds every handler except `NewExpenseHandler` and registers no `/expenses` routes (nor `POST /contracts/{id}/adopt` / the export count routes registered in `cmd/server/main.go`). The entire expense surface — including the ReceiptUpload path of CR-02 and the List scoping of CR-01 — has zero handler-level integration coverage.
**Fix:** Wire `expenseService`/`expenseHandler` and register all `/expenses` routes in the fixture, and add a cross-org isolation test that would have caught CR-01/CR-02.

### WR-11: Reject handlers ignore the JSON decode error

**File:** `internal/adapters/primary/http/expense.go:404-405`, `internal/adapters/primary/http/time_entry.go:390-391`
**Issue:** `json.NewDecoder(r.Body).Decode(&req)` result is discarded; a malformed body silently rejects with an empty reason (and the reason is not required anywhere — service `Reject` accepts ""). Sloppy error handling that hides client mistakes.
**Fix:** Check the decode error and return 400; consider making `reason` required for rejects (the approval history is the audit record).

## Info

### IN-01: `timePtr` is dead code in both entry services

**File:** `internal/core/services/time_entry/time_entry.go:378`, `internal/core/services/expense/expense.go:393`
**Issue:** `timePtr` is defined but never called in either service. Remove it.

### IN-02: Oversized-field tests still reference removed columns

**File:** `internal/adapters/primary/http/validate_test.go:107,116,134`
**Issue:** The time-entry, working-group, and expense oversized-field test bodies still send `project_id`/`subproject_id`/`wg_id` JSON fields that no longer exist in the request DTOs. They still pass (the length cap fires before field validation and unknown fields are ignored), but they no longer reflect the real request shapes — misleading as documentation.
**Fix:** Update bodies to the current DTO field names (`activity_id`, `unit_id`, etc.).

### IN-03: ActivityDetailResponse comment contradicts actual ancestry ordering

**File:** `internal/adapters/primary/http/activity_handler.go:61-70`
**Issue:** The doc comment says the ancestry chain is "ordered parent → root", but `GetAncestry` (recursive CTE anchored at the node) returns leaf → root, and the repository test asserts that order (activity_repository_test.go:259-262). Update the comment to "ordered leaf → root" (or reorder the response).

### IN-04: AuditLogRepository is defined but never wired

**File:** `internal/adapters/secondary/postgres/time_entry_repository.go:303-343`, `internal/core/ports/time_entry_repository.go:37-39`
**Issue:** `AuditLogRepository` (and its `Create`) is not constructed or used anywhere in `cmd/server/main.go` or the fixture. It is dead code until the audit-log feature lands.
**Fix:** Wire it or remove it until needed.

---

_Reviewed: 2026-07-31T21:10:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
