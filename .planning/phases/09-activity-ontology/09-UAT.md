---
status: testing
phase: 09-activity-ontology
source: [09-VERIFICATION.md]
started: 2026-07-31T20:50:00Z
updated: 2026-07-31T20:50:00Z
---

## Current Test

number: 1
name: Live end-to-end activity flow smoke test (incl. cycle rejection)
expected: |
  Run the server against a testcontainers/live PostgreSQL and exercise the full flow: create parent + child activity, list with filters, fetch detail (ancestry + commercial_context + billable), attempt a cycle-creating PUT /api/activities/{id} reparent (parent = own descendant or self), log a time entry with activity_id, submit, approve as WG manager.
  Expected: all endpoints respond correctly; the reparent attempt returns HTTP 400 "activity parent would create a cycle" (not 500); submit on a commercial activity without an anchored WG returns 409 with the ErrActivityNotLoggable message; D-11 skip routes owner-is-approver entries straight to pending_finance; WG manager approve transitions to pending_finance.
awaiting: user response

## Tests

### 1. Live end-to-end activity flow smoke test (incl. cycle rejection)
expected: Create/list/children/detail endpoints return ancestry + commercial_context + billable; a reparent into the activity's own subtree returns 400 (ErrActivityCycle), not 500; entry submit on a commercial activity without an anchored WG returns 409 with the ErrActivityNotLoggable message; WG manager approve transitions to pending_finance.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
