# F07 — Org Hierarchy & Units

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Structure |
| **Purpose** | "Who is accountable for whom" — the stable org tree that carries reporting lines and visibility scoping. Units explicitly do **not** route approvals (execution does). |
| **Answers** | "Is the work on track?" (Q2) — via accountability and subtree visibility |
| **Vision ref** | [[VISION]] §5 |
| **Decision ref** | [[ADR-P-001 — Units vs Working Groups]] (binding), [[ADR-P-002 — Four Pillars & Feature Purposes]] |
| **Surface** | People → Org (`/org-hierarchy`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

The organization is a tree of **units** with multi-unit membership and a designated primary unit per person. Managers see their subtree's data; org-role manager/finance see the whole org. Units are the accountability layer — stable, slow-moving — distinct from working groups, which are the execution layer.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As an org admin, I can create/edit/delete units in a parent-child tree so that accountability is mapped | ✅ Implemented | |
| US-002 | As an org admin, I can add members to units and designate a primary unit so that reporting lines are unambiguous | ✅ Implemented | |
| US-003 | As a user, I can belong to multiple units so that matrix work is representable | ✅ Implemented | |
| US-004 | As an org admin, I can reparent a unit via edge drag so that re-orgs are quick | ✅ Implemented | |

## Acceptance Criteria

- [x] Unit CRUD with parent-unit hierarchy; org tree visualization (ReactFlow)
- [x] Delete protection: root unit, units with children, units with members cannot be deleted
- [x] Circular parent reference prevention
- [x] Multi-unit membership with "Make Primary"
- [ ] Backfill: full AC list from `legacy/13-Organization-Users`

## Boundaries

- Does not: route approvals through the unit tree — approval routes through execution (WG manager/delegate), per [[ADR-P-001 — Units vs Working Groups]].
- Does not: manage people (leave, documents, contracts of employment) — [[VISION]] §8 HR anchor; the only membership data beyond role is validity dates per [[ADR-P-008 — Availability & Employment Validity]] D-2.
- Does not: represent teams — that is working groups ([[F10-Activities]] execution layer).

## Related

- [[VISION]] · [[ADR-P-001 — Units vs Working Groups]] · [[ADR-P-008 — Availability & Employment Validity]] (validity dates on memberships) · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F10-Activities]] · Schema: [[S01-Database-ERD]]

## Status

- **Status:** Implemented (v0.1)
- **Last updated:** 2026-07-30
