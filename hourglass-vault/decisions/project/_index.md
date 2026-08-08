# Project Decisions — Index

---
tags: ["adr", "idea-layer", "index"]
---

# Idea-Layer ADRs (Project Decisions)

These ADRs record **what we build and why** — the vision made binding. They are the first gate any idea passes through, per the steering test in [[VISION]] §7.

**What belongs here:** pillar assignments, feature purposes, sequencing decisions, scope rulings, resolutions to idea-layer tensions.

**What doesn't:** how-to-build choices (Go patterns, frameworks, libraries) — those live in `decisions/backend/` or the global knowledge vault.

## Format

`ADR-P-NNN — Title.md` · tags: `adr`, `idea-layer`, plus the pillar(s) it touches.

## ADRs

| ADR                                                                | Decision                                                                                                                                                                        | Pillar(s)          | Status   |
|--------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------|----------|
| [[ADR-P-001 — Units vs Working Groups]]                            | Units = accountability, working groups = execution                                                                                                                              | Structure          | Accepted |
| [[ADR-P-002 — Four Pillars & Feature Purposes]]                    | Ratifies the pillar model + each feature's purpose                                                                                                                              | All                | Proposed |
| [[ADR-P-003 — Tickets as the Second Capture Layer]]                | Demand capture via internal tickets (not a task board) — v0.2 lifecycle `open→triage→planned→in_progress→resolved→closed` + reopen + guarded dismissal, closed kind set, ticket→activity→entries chain (rev. 2026-08-03) | Capture            | Proposed |
| [[ADR-P-004 — The Today View]]                                     | First Insight feature: "what should I be working on?"                                                                                                                           | Insight            | Proposed |
| [[ADR-P-005 — Insight Pillar Roadmap]]                             | V1–V6 sequenced by data dependency                                                                                                                                              | Insight            | Proposed |
| [[ADR-P-006 — Out-of-Scope Enforcement]]                           | §8 anchors binding + rejection log                                                                                                                                              | Governance         | Proposed |
| [[ADR-P-007 — Activity Ontology]]                                  | One recursive `activities` entity replaces projects+subprojects; commercial optional & inherited; entries link via one `activity_id`; big-bang into v0.1                        | Structure, Capture | Accepted |
| [[ADR-P-008 — Availability & Employment Validity]]                 | Declared availability windows + membership validity dates, surfaced at assignment time; `hr` org role as curator/consumer, never approver; leave machinery stays rejected       | Structure          | Proposed |
| [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]    | Sidebar groups are job-language, pillar-mapped; surfaces follow the stakeholder map with role-scoped visibility; Today lands ticketless pre-deploy; `/projects` → `/activities` | All                | Proposed |
| [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] | Captured effort is a fact, coverage is a decision — per-entry allocation ledger with Σ-invariant, to-cover queue, manager-confirmed, snapshot-not-lock                          | Structure, Insight | Accepted |
| [[ADR-P-013 — Origins]]                                           | Origin axis: activities carry a type + reference set set once at creation (manager-assignment / employee-proposal / customer-ticket), immutable (`ErrOriginImmutable`), same-org validated, proposal approval routed via BE-014 | Structure, Capture | Proposed |
| [[ADR-P-015 — Direction, The Plan Plane]]                         | Third plane: the plan (direction → facts → coverage). One entity, mode derived from `planned_date` (scheduled/queued), per-day rows with partial-day multiplicity, supersede-chain mutability, derived done/lapsed/claimed, WG split claims with Σ-consumption, org policy stored-not-enforced, P-008 warning overlay, direction-coverage read-model, origin fallback from the first direction record | Structure, Capture, Insight | Proposed |

## Roadmap dependency chain

```
ADR-P-002 (pillars)  ─┐
ADR-P-001 (units/WG) ─┤
ADR-P-007 (ontology) ─┤
                      ▼
            ADR-P-003 (tickets) ──► ADR-P-004 (today view) ──► ADR-P-005 (V3–V6)
                      ▲                     ▲
            ADR-P-006 (scope gate)   ADR-P-012 (coverage ledger) ──► ADR-P-014 (funding sources)
```

## Next candidates (post-v0.1)

* **ADR-P-009** — Employee knowledge profile shape (skills, load, history; availability already in via [[ADR-P-008 — Availability & Employment Validity]])
* **ADR-P-010** — Contract budget/target fields — prerequisite of V6, decided at V4 design time
* **Customer-facing read surface** — open slot flagged in [[VISION]] §3 stakeholder map and [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-5; spec when the customer role gets real users

## Rules

* Append-only. Supersede via status + `Superseded by` links.
* Every idea-layer ADR links back to the section of [[VISION]] it operationalizes.
* Rejected ideas go to the log in [[ADR-P-006 — Out-of-Scope Enforcement]], not back into discussion.
