# F11 — Time Entries

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Capture |
| **Purpose** | The atomic unit of "what work actually happened" — everything else (approvals, exports, every future Insight feature) derives from this. |
| **Answers** | "What should I be working on?" (Q1, via Today composition) and feeds Q2/Q3 |
| **Vision ref** | [[VISION]] §4 (Capture), §5 |
| **Decision ref** | [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-007 — Activity Ontology]] D-4 (single required `activity_id`) · [[ADR-P-001 — Units vs Working Groups]] Q2 (subtree visibility) |
| **Surface** | Track → Time (`/time-entries`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

One entry per activity per date (flat model), captured once at the source. Entries move through the two-stage approval chain (manager → finance) and become *trusted* time — the property that makes Hourglass data usable for billing and payroll exports later.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As an employee, I can log hours against an activity for a date so that work is captured at the source | ✅ Implemented | |
| US-002 | As an employee, I can edit draft/submitted/rejected entries so that mistakes are correctable before approval | ✅ Implemented | |
| US-003 | As an employee, I can submit entries for approval so that my time becomes trusted | ✅ Implemented | |
| US-004 | As a manager, I see only my subtree's entries so that visibility is scoped correctly | ✅ Implemented | |

## Acceptance Criteria

- [x] Entry CRUD on the flat model (activity, date, hours, description)
- [x] Status machine: draft → submitted → pending_manager → pending_finance → approved/rejected
- [x] Approved/rejected entries are immutable; rejected show reason
- [x] No self-approval (employee cannot approve own; manager cannot approve own)
- [ ] Entry links to exactly one `activity_id` after the P-007 migration (D-4); `wg_id` column dropped

## Boundaries

- Does not: track demand (the request that caused the work) — that is tickets ([[ADR-P-003 — Tickets as the Second Capture Layer]], v0.2).
- Does not: block submission during declared absence — reality is messy; availability is advisory at assignment time only ([[ADR-P-008 — Availability & Employment Validity]] D-3).
- Does not: capture *outcomes* (what was made/learnt) — deferred V7 substrate ([[VISION]] §6).

## Related

- [[VISION]] · [[ADR-P-001 — Units vs Working Groups]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F10-Activities]] · [[F12-Expenses]] · [[F13-Exports]] · Schema: [[S05-State-Machines]]

## Status

- **Status:** Implemented (v0.1); P-007 FK migration pending
- **Last updated:** 2026-07-30
