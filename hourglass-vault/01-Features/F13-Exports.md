# F13 — Exports

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Insight |
| **Purpose** | Today's primitive Insight — proof the captured data is queryable and meaningful, and the bridge that hands *trusted data* to payroll and billing systems without becoming them. |
| **Answers** | "What does the work cost and earn?" (Q3) |
| **Vision ref** | [[VISION]] §4 (Insight), §5, §8 (payroll anchor) |
| **Decision ref** | [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-008 — Availability & Employment Validity]] D-1c (payroll view) |
| **Surface** | Reports → Exports (`/exports`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

Downloadable CSV/XLSX exports for timesheets, expenses, and combined reports — date-ranged, format-selectable, role-scoped, and respectful of cutoff locks. With [[ADR-P-008 — Availability & Employment Validity]] D-1c, a **payroll view** joins the set: approved, cutoff-locked entries plus confirmed absence windows, per person per period — the monthly input a payroll provider or consulente consumes.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As a user, I can export my timesheet as CSV/XLSX for a date range so that my data is portable | ✅ Implemented | |
| US-002 | As a finance user, I can export combined time + expense reports so that cost review is one file | ✅ Implemented | |
| US-003 | As a user, I get a friendly message when a range has no data so that empty exports aren't confusing | ✅ Implemented | |
| US-004 | As an HR/finance user, I can export the payroll view (entries + confirmed absence windows) so that wages have trusted input | ⬜ (P-008 D-1c) | |

## Acceptance Criteria

- [x] Timesheet / expense / combined exports, CSV and XLSX
- [x] 1-year max range enforced frontend + backend; large exports streamed with count pre-check
- [x] Auth required; data scoped to role (own / subtree / org)
- [ ] Payroll export view: kinds, dates, hours, `certificate_ref` for medical — flat `(person, date range, kind, hours)` (P-008 D-1b/D-1c)

## Boundaries

- Does not: compute payroll, balances, or invoices — Hourglass produces *trusted data*; payslips are produced elsewhere ([[VISION]] §8).
- Does not: include unconfirmed absence windows in the payroll feed — holiday windows need both confirmations (P-008 D-1a).
- Does not: grow charts or dashboards — that is V5/V6 Insight infrastructure ([[VISION]] §6).

## Related

- [[VISION]] · [[ADR-P-008 — Availability & Employment Validity]] · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F11-Time-Entries]] · [[F12-Expenses]] · Schema: [[S04-API-Contracts]]

## Status

- **Status:** Implemented (v0.1); payroll view pending P-008
- **Last updated:** 2026-07-30
