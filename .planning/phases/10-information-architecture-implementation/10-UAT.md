---
status: testing
phase: 10-information-architecture-implementation
source: [10-VERIFICATION.md]
started: "2026-08-01T08:00:00Z"
updated: "2026-08-01T08:00:00Z"
---

## Current Test

number: 1
name: Visual walk of the sidebar regroup (D-1) across all five org roles
expected: |
  Group labels and order match the ADR-P-011 D-1 table verbatim; HR sees no Review group;
  employee sees no Economics; the Today item sits ungrouped at top with href '/' and is active
  only on '/'
awaiting: user response

## Tests

### 1. Visual walk of the sidebar regroup (D-1) across all five org roles
expected: Log in as employee, manager, finance, hr, customer and check the pillar groups render in locked order (Today/Track/Work/People/Economics/Review/Reports) with exact labels and no legacy 'Tracking'/'Management' wording; check disabled Tickets/Availability items show the locked tooltips on hover when collapsed.
result: [pending]

### 2. Visual walk of the Today landing at '/' per UI-SPEC focal-point table
expected: No charts/KPIs; sections stack top-down ('Waiting on you' then 'Your week'); empty states use the locked copy verbatim; the page is never blank in any state (spacing lg between sections, Display 28px 'Today', right-aligned CTA 'Review now' for approvers / 'Log time' otherwise).
result: [pending]

### 3. Visual walk of the Approvals page at /approvals
expected: Single h1 'Approvals' in the Header; Manager/Finance tabs (only for dual-stage users); rows at py-3 density with Approve/Reject pair as focal accent; reject dialog requiring a reason (disabled until ≥ 10 chars); 'Queue is clear' empty state per stage; error state (not empty) when pending queries fail.
result: [pending]

### 4. Visual walk of the Working Groups page at /working-groups
expected: Header with title + search + single accent 'New working group' CTA; card grid on muted surfaces; locked empty state; create/edit/members/delete dialogs work; cards show WG name, linked activity, manager name, member count; delete surfaces backend guard errors via toast.
result: [pending]

### 5. Visual walk of the page shell on all carried-over pages
expected: Exactly one h1 in the 48px Header band on time-entries, expenses, exports, contracts, customers, activities; content scrolls inside Body (window does not scroll); no double padding or re-layout artifacts from the wrap.
result: [pending]

### 6. Product decision on WR-01 (WG-stage employee action path)
expected: A recorded decision: either relax the Approve/Reject handler gates to admit WG manager/delegate via IsWGManager (mirroring ListPending), or accept and document the asymmetry as a follow-up. Core manager/finance round-trips are e2e-proven either way.
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
