# F12 — Expenses

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Capture |
| **Purpose** | The atomic unit of "what the work cost beyond hours" — captured with the same fidelity and routed through the same approval chain as time. |
| **Answers** | "What does the work cost and earn?" (Q3) |
| **Vision ref** | [[VISION]] §4 (Capture), §5 |
| **Decision ref** | [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-001 — Units vs Working Groups]] Q1 (expenses route like time) · [[ADR-P-007 — Activity Ontology]] D-4 |
| **Surface** | Track → Expenses (`/expenses`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

Expenses capture non-labor cost against an activity, with nine categories and receipt upload. Routing is symmetric with time entries: every expense routes through its activity → anchored WG → manager/delegate, then finance. There is no project-manager expense queue and no fallback path.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As an employee, I can claim an expense with category, amount, and receipt so that costs are captured at the source | ✅ Implemented | |
| US-002 | As an employee, I can log mileage with km distance so that travel is reimbursed correctly | ✅ Implemented | |
| US-003 | As an approver, I review expenses through the same two-stage chain as time so that cost data is equally trusted | ✅ Implemented | |

## Acceptance Criteria

- [x] Expense CRUD with 9 categories; `km_distance` meaningful only for mileage
- [x] Receipt upload
- [x] Same status machine and immutability rules as time entries
- [ ] `activity_id` required and `customer_id` dropped after the P-007 migration (D-4) — symmetric with time

## Boundaries

- Does not: reimburse — payment happens outside ([[VISION]] §8 payroll anchor).
- Does not: route via project managers — that approval source is removed ([[ADR-P-001 — Units vs Working Groups]] Q1).
- Does not: allow orphan expenses without an activity — the state is unrepresentable post-migration (P-007 D-4).

## Related

- [[VISION]] · [[ADR-P-001 — Units vs Working Groups]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F11-Time-Entries]] · [[F13-Exports]] · Schema: [[S05-State-Machines]]

## Status

- **Status:** Implemented (v0.1); P-007 FK migration pending
- **Last updated:** 2026-07-30
