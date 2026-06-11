---
phase: 06-time-entries-expenses
plan: 02
type: execute
completed_date: 2026-06-11
duration: ~45min
wave: 2
subsystem: backend-service-layer
tags: [time-entries, expenses, approval-workflow, two-stage-approval]
requires: [06-01]
provides: [time-entry-service, expense-service]
affects: [06-03, 06-04]
tech-stack:
  added: []
  patterns: [two-stage-approval, synchronous-approval-history, self-approval-guard]
key-files:
  created:
    - internal/core/services/expense/expense.go
    - internal/core/services/expense/expense_test.go
  modified:
    - internal/core/services/time_entry/time_entry.go
    - internal/core/services/time_entry/time_entry_test.go
    - internal/core/services/testdata/mocks.go
    - internal/core/domain/expense/expense.go
decisions: []
metrics:
  tasks: 3
  files_created: 2
  files_modified: 4
  commits: 3
  tests_added: ~35
---

# Phase 6 Plan 02: Time Entries + Expenses (Backend Service Layer)

Extended TimeEntryService with two-stage approval workflow (manager→pending_finance→approved/rejected), created ExpenseService from scratch with matching CRUD + two-stage approval, and wrote unit tests for both.

## Objective

Build the backend service layer for time entry and expense approval. Service layer enforces status transitions, role gating, self-approval prevention, and synchronous approval history generation.

## What Was Built

### Task 1: Extended TimeEntryService with Two-Stage Approval

- **Service struct**: Replaced `auditRepo` with `approvalRepo ports.TimeEntryApprovalRepository`; removed `CreateAuditLog` method entirely
- **`Submit`**: Now sets `CurrentApproverRole = "manager"` and `SubmittedAt = time.Now()`; works from `StatusDraft` and `StatusRejected`
- **`Approve`**: Self-approval guard (creator != approver → `ErrForbidden`); role-differentiated routing:
  - `role=="manager"` + `StatusSubmitted` → `StatusPendingFinance`, `current_approver_role="finance"`
  - `role=="finance"` + `StatusPendingFinance` → `StatusApproved`, `current_approver_role=nil`
  - Synchronous approval record created via `approvalRepo.CreateApproval`
- **`Reject`**: Role check (manager or finance only); status check (submitted or pending_finance); transitions to `StatusRejected`; stores `reason` in approval `Comment` field
- **`Delete`**: Now only allows `StatusDraft` entries (direct status check instead of `CanEdit()`)
- **`strPtr`/`timePtr`**: Helper functions added

### Task 2: Created ExpenseService from Scratch

Full CRUD + two-stage approval matching TimeEntryService patterns exactly:

- **`Create`**: Category validation via `IsValidCategory`, period lock check, date parsing
- **`Update`**: Owner check, edit gating, category validation on update
- **`Delete`**: Draft-only restriction
- **`Submit`**: Same as TimeEntryService — status=submitted, current_approver_role=manager
- **`Approve`/`Reject`**: Same two-stage role-differentiated routing with synchronous approval history
- **`List`/`Get`/`ListPending`**: Delegate to repository

### Task 3: Unit Tests for Both Services

- **Time entry tests (645 lines)**: Full approval matrix (manager→pending_finance, finance→approved, self-approve forbidden, wrong-role denied), reject with reason, submit draft/rejected, update draft/submitted/rejected, delete draft-only
- **Expense tests (571 lines)**: CRUD + all approval paths mirroring time entry patterns, invalid category, period lock, non-owner guards

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing testability] MockExpenseRepo.CreateApproval didn't track approvals**
- **Found during:** Task 2 (ExpenseService create)
- **Issue:** `MockExpenseRepo.CreateApproval` returned `nil` without storing the approval record, making it impossible for expense tests to verify approval history creation
- **Fix:** Added `Approvals []*domainexpense.Approval` field to `MockExpenseRepo` and store the approval in `CreateApproval`
- **Files modified:** `internal/core/services/testdata/mocks.go`
- **Commit:** `74b5cbd`

**2. [Rule 1 - Bug] Delete method used CanEdit() which now allows submitted entries**
- **Found during:** Test run (`TestService_Delete/cannot_delete_submitted_entry` failed)
- **Issue:** `CanEdit()` was widened to include submitted/rejected (for Update), but Delete should remain draft-only
- **Fix:** Changed Delete to use direct `e.Status != StatusDraft` check instead of `!e.CanEdit()`
- **Files modified:** `internal/core/services/time_entry/time_entry.go`, `internal/core/services/expense/expense.go`
- **Commit:** No separate commit — fixed inline before commit

## Verification

```
go build ./internal/core/services/time_entry/... ./internal/core/services/expense/...  → PASS
go vet ./internal/core/services/...  → PASS
go test -count=1 -timeout 60s -run 'TestService_|TestExpenseService_' ./internal/core/services/...

12 packages all ok:
  auth, contract, customer, expense, export, invitation,
  organization, password_reset, project, time_entry, unit, working_group
```

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 74b5cbd | feat | Extend TimeEntryService with two-stage approval workflow |
| fab2aeb | feat | Create ExpenseService with CRUD + two-stage approval |
| cb742ca | test | Add unit tests for time entry and expense services |

## Threat Scan

All threat mitigations from the plan were implemented:
- **T-06-04 (EoP - Self-approval)**: Self-approval guard in both `Approve` methods (`e.UserID == userID → ErrForbidden`)
- **T-06-05 (EoP - Reject)**: Role check in `Reject` restricts to manager/finance only; status check prevents rejects at wrong stage
- **T-06-06 (EoP - Expense Approve)**: Same self-approval guard and role routing as time entry
- **T-06-07 (Tampering - History)**: Approval records created synchronously in the same request as status updates

## Self-Check: PASSED

- [x] `go build ./internal/core/services/time_entry/... ./internal/core/services/expense/...` passes
- [x] `go vet ./internal/core/services/...` passes
- [x] `go test -count=1 -timeout 60s` all 12 packages pass
- [x] TimeEntryService.Approve transitions submitted→pending_finance (manager) and pending_finance→approved (finance)
- [x] TimeEntryService.Approve returns ErrForbidden on self-approval
- [x] TimeEntryService.Reject transitions submitted/pending_finance→rejected with reason
- [x] TimeEntryService.Submit sets current_approver_role=manager and submitted_at
- [x] TimeEntryService.Update works for draft, submitted, rejected
- [x] TimeEntryService.Delete only allows draft entries
- [x] ExpenseService.Create validates categories and period lock
- [x] ExpenseService.Approve/Reject matches two-stage pattern
- [x] All approval records created synchronously (no goroutines)
- [x] 3 atomic commits made
