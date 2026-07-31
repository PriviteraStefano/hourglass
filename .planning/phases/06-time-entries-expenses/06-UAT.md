---
status: testing
phase: 06-time-entries-expenses
source: 06-01-SUMMARY.md, 06-02-SUMMARY.md, 06-03-SUMMARY.md, 06-04-SUMMARY.md, 06-05-SUMMARY.md
started: 2026-07-11T19:18:37Z
updated: 2026-07-11T22:14:32Z
---

## Current Test

number: 1
name: Time Entries Page Loads
expected: |
  Navigate to /time-entries. The page shows a MiniCalendar on the left and a day detail panel on the right. The date defaults to today. The calendar shows month navigation arrows and a legend with 6 status colors (draft=yellow, submitted=blue, pending_manager=green, pending_finance=purple, approved=blue, rejected=red).
awaiting: user response

## Tests

### 1. Time Entries Page Loads
expected: Navigate to /time-entries. The page shows a MiniCalendar on the left and a day detail panel on the right. The date defaults to today. The calendar shows month navigation arrows and a legend with 6 status colors (draft=yellow, submitted=blue, pending_manager=green, pending_finance=purple, approved=blue, rejected=red).
result: pass
note: "Initial 500 error fixed by applying migration 006. User confirmed page loads after fix."

### 2. Empty Day Shows Create Entry CTA
expected: Click on a date with no entries. The day detail panel shows "No entries for {date}" with descriptive text and a "Create Entry" button.
result: pass

### 3. Create a Time Entry (Draft)
expected: Click "Create Entry" on an empty day. An entry row appears with a project selector, hours input, and description field. Click "Save Draft" — entry is saved, toast shows "Entry created", and the calendar cell shows the draft (yellow) status.
result: [pending]

### 4. Edit an Existing Entry
expected: Click on a date that has a draft entry. The entry row is editable with project selector, hours, and description. Change a value and click "Save Draft". Toast shows "Entry updated".
result: [pending]

### 5. Submit Entry for Approval
expected: On a draft entry, click "Submit Entry". Toast shows "Entry submitted for approval". Status changes to submitted (blue). Calendar cell updates.
result: [pending]

### 6. Delete Entry with Confirmation
expected: On a draft entry, click delete. An AlertDialog appears saying "Delete Entry?" with body "This action cannot be undone." and "Delete" (destructive) + "Cancel" buttons. Confirm — toast shows "Entry deleted", entry removed.
result: [pending]

### 7. Approval Buttons Visibility
expected: An entry in pending_manager status shows "Approve" and "Reject" buttons for a manager user, but not for an employee. An entry in pending_finance status shows them for a finance user only.
result: [pending]

### 8. Reject Entry with Reason
expected: Click "Reject" on a pending entry. An inline textarea appears with "Reason for rejection (required)" — must have ≥10 chars. Submit — toast shows "Entry rejected", status changes to rejected (red).
result: [pending]

### 9. Approval History Timeline
expected: An entry with approval history shows the approval-history component with an immutable timeline: action icons, actor role, timestamp, and optional comment. Empty state shows "No approval history".
result: [pending]

### 10. Expenses Page Loads
expected: Navigate to /expenses via sidebar link. Page shows MiniCalendar + day detail panel layout matching time entries. Sidebar "Expenses" link is enabled and navigatable.
result: [pending]

### 11. Create an Expense (Draft)
expected: Click "Create Expense" on an empty day. Expense form appears with category selector (9 options: mileage, meal, accommodation, parking, travel_tickets, tolls, taxi, equipment, other), amount field, description field. Save draft — toast shows "Expense created".
result: [pending]

### 12. Mileage Category Shows km_distance
expected: When creating/editing an expense, selecting "Mileage" from the category dropdown reveals a km_distance input field. Selecting any other category hides it.
result: [pending]

### 13. Receipt Upload on Expense
expected: On an expense row, a receipt upload button/icon is visible. Clicking opens a file picker accepting PDF/JPEG/PNG files. After upload, receipt URL shown as a download link.
result: [pending]

### 14. Submit Expense for Approval
expected: On a draft expense, click "Submit Expense". Toast shows "Expense submitted for approval". Status changes to submitted (blue).
result: [pending]

### 15. Expense Approval Workflow
expected: An expense in pending_manager status shows Approve/Reject buttons for manager. Approve → toast "Expense approved", transitions to pending_finance. Reject → toast "Expense rejected".
result: [pending]

### 16. Calendar Status Resolution
expected: If a date has multiple entries with different statuses, the calendar cell shows the highest-priority status color: approved > rejected > submitted > draft.
result: [pending]

### 17. Org Hierarchy Page Loads (Members Query)
expected: Navigate to /org-hierarchy. The page loads showing the org tree with members listed. No Zod validation errors in console.
result: pass
note: "Fixed: added JSON tags to Member struct. Verified endpoint returns snake_case keys."

## Summary

total: 17
passed: 2
issues: 1
pending: 14
skipped: 0

## Gaps

- truth: "Time entries page loads without errors, showing MiniCalendar and day detail panel"
  status: resolved
  reason: "User reported: page shows error 'failed to fetch time entries'. Backend returns 500 on GET /time-entries"
  severity: blocker
  test: 1
  root_cause: "Migration 006 (add_approval_fields) not applied to database. The time_entry_repository.go SELECT query references current_approver_role and submitted_at columns that don't exist in the table. Server doesn't auto-migrate — relies on manual go run ./cmd/migrate -up."
  artifacts:
    - path: "migrations/006_add_approval_fields.up.sql"
      issue: "Migration not applied to database"
  missing:
    - "Run migration 006 to add current_approver_role and submitted_at columns to time_entries and expenses tables"
  debug_session: ""
  fix: "Ran go run ./cmd/migrate -up -dir migrations. Applied migrations 004, 005, 006, and 009."

- truth: "Org hierarchy page loads without Zod validation errors"
  status: resolved
  reason: "User reported: Zod validation error on /organizations/members — expected string for fields id, user_id, role, is_active but received undefined"
  severity: major
  test: 17
  root_cause: "Member struct in internal/core/domain/organization/organization.go:37-47 has no JSON tags. Go serializes as {ID, UserID, IsActive, UserName, UserEmail} but frontend Zod schema in web/src/api/units.ts:15-22 expects snake_case {id, user_id, is_active, user_name, user_email}. Zod parse fails because keys don't match."
  artifacts:
    - path: "internal/core/domain/organization/organization.go"
      issue: "Member struct missing JSON tags"
    - path: "web/src/api/units.ts"
      issue: "OrgMemberSchema expects snake_case keys"
  missing:
    - "Add json:\"id\" tags to Member.ID field"
    - "Add json:\"user_id\" tags to Member.UserID field"
    - "Add json:\"role\" tags to Member.Role field"
    - "Add json:\"is_active\" tags to Member.IsActive field"
    - "Add json:\"user_name\" tags to Member.UserName field"
    - "Add json:\"user_email\" tags to Member.UserEmail field"
  debug_session: ""
  fix: "Added JSON tags to Member struct in internal/core/domain/organization/organization.go. Verified: endpoint returns snake_case keys matching Zod schema."
