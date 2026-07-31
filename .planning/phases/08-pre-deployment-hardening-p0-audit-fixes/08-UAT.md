---
status: testing
phase: 08-pre-deployment-hardening-p0-audit-fixes
source: [08-VERIFICATION.md]
started: 2026-07-31T15:05:16Z
updated: 2026-07-31T15:05:16Z
---

## Current Test

number: 1
name: P0-4 error-boundary recovery reliability (deployment-gate decision)
expected: |
  4/4 passing on every run — specifically "Try again re-runs the loader and recovers to data" must clear the error panel within the 15s assertion.
awaiting: user response

## Tests

### 1. P0-4 error-boundary recovery reliability (deployment-gate decision)
expected: Run `npx playwright test error-boundary` on an idle machine, repeated 10+ times. 4/4 passing every run; "Try again" clears the error panel within its 15s assertion.
result: [pending]

### 2. List-view manual smoke (P0-2)
expected: On /time-entries and /expenses, apply status/date/category filters, reload the URL, confirm filter state persists; click a row and confirm the detail surface opens; try a single-bound date range (one click in the calendar).
result: [pending]

### 3. Customers route manual smoke (P0-3)
expected: Sidebar → Customers; search; open a detail; deep-link a fresh session to /customers. List renders with data, search narrows, detail loads, no 404/blank; fresh session flows login → app → /customers.
result: [pending]

### 4. Real-outage recovery loop (P0-4)
expected: Stop the backend while on an authenticated page; confirm the error panel; restart the backend; click "Try again". Error panel with message + Try again + Go to Today (never a blank screen); recovery to real data after restart.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
