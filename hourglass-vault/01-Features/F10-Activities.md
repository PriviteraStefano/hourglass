# F10 — Activities

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Structure |
| **Purpose** | The recursive container work belongs to, at any granularity — makes time attributable. Replaces the rigid project/subproject two-table chain; internal (non-commercial) work is first-class. |
| **Answers** | "What should I be working on?" (Q1) and "Is the work on track?" (Q2) |
| **Vision ref** | [[VISION]] §4 (Structure), §5 |
| **Decision ref** | [[ADR-P-007 — Activity Ontology]] (binding) · revises rows of [[ADR-P-002 — Four Pillars & Feature Purposes]] |
| **Surface** | Work → Activities (`/activities`, renamed from `/projects` per P-011 D-6) + Work → Working Groups (`/working-groups`, new per P-011 D-4), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] |

## Overview

An **activity** is one recursive work entity: nesting is available when work is genuinely nested, never required; `kind` is a free label from an org-level catalog with no level semantics. Commercial context (contract, billability) is optional and inherited downward. **Working groups** anchor to an activity at any depth and are the execution structure — the team doing the work and the approval-routing source.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As a manager, I can create activities at any depth with a kind so that work is modeled as it actually is | ⬜ (ships with P-007 migration) | |
| US-002 | As a manager, I can create internal activities with no contract so that non-commercial work is first-class | ⬜ | |
| US-003 | As a manager, I can form a working group on an activity so that execution and approval routing exist | ⬜ (new WG surface, P-011 D-4) | |
| US-004 | As a user, I can see an activity's derived commercial chain so that billability is transparent | ⬜ | |

## Acceptance Criteria

- [ ] `/projects` routes and UI renamed to `/activities` (P-011 D-6)
- [ ] Activity CRUD: parent (nullable), kind from org catalog, contract (nullable), governance model, billable (nullable = inherit)
- [ ] Activity kinds catalog manageable under Admin (org-extensible, P-007 D-2)
- [ ] Working group list/detail surface under Work (P-011 D-4)
- [ ] Delete protection: active entries or children block deletion (carried from project guards)

## Boundaries

- Does not: impose a level ladder or kind↔depth constraints (P-007 D-2 — rejected alternative).
- Does not: require a commercial chain to log work — internal activities always exist (P-007 D-3).
- Does not: store customer/commercial facts on the activity — derived by walking upward, never denormalized (P-007 D-3).
- Does not: make WGs mandatory on solo activities (P-007 D-5).

## Related

- [[VISION]] · [[ADR-P-001 — Units vs Working Groups]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-008 — Availability & Employment Validity]] (assignment-time warnings on the WG surface) · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F09-Contracts]] · [[F11-Time-Entries]] · Schema: [[S01-Database-ERD]]

## Status

- **Status:** In progress (ontology ships big-bang pre-deploy, P-007 D-6)
- **Last updated:** 2026-07-30
