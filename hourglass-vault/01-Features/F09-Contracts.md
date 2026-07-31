# F09 — Contracts

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Structure (commercial) |
| **Purpose** | The economic boundary of work — links activities to customers and price, and is the source of the billability default inherited downward. Economics only, never work decomposition. |
| **Answers** | "What does the work cost and earn?" (Q3) |
| **Vision ref** | [[VISION]] §5 |
| **Decision ref** | [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-007 — Activity Ontology]] D-3/D-7 |
| **Surface** | Economics → Contracts (`/contracts`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

Contracts bind a customer to a scope of work and its economics. Activities link to a contract optionally — absence of a contract means first-class internal work, and the commercial chain is derived by walking the activity tree upward, never denormalized.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As a finance user, I can create/edit contracts with a customer so that work has an economic boundary | ✅ Implemented | |
| US-002 | As a user, I can filter contracts by status so that live vs. closed economics are distinguishable | ✅ Implemented | |
| US-003 | As a finance user, I cannot delete a contract with linked projects/activities so that captured work keeps its boundary | ✅ Implemented | |
| US-004 | As a finance user, I can create zero-value contracts so that non-priced arrangements are representable | ✅ Implemented | |

## Acceptance Criteria

- [x] Contract CRUD with customer combobox including "(Internal)" indicator
- [x] Delete guard: 409 when linked work exists (`ErrHasActiveProjects`)
- [x] Zero-value contracts allowed
- [ ] Backfill: full AC list from `legacy/12-Contracts-Projects`

## Boundaries

- Does not: decompose work — that is activities ([[F10-Activities]]); the contract is economics only ([[ADR-P-002 — Four Pillars & Feature Purposes]]).
- Does not: issue invoices or payslips — trusted data *for* those systems ([[VISION]] §8).
- Does not: carry budget/target fields yet — prerequisite of V6, reserved for ADR-P-010 at V4 design time ([[VISION]] §6).

## Related

- [[VISION]] · [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F08-Customers]] · [[F10-Activities]] · Schema: [[S01-Database-ERD]] · [[S04-API-Contracts]]

## Status

- **Status:** Implemented (v0.1)
- **Last updated:** 2026-07-30
