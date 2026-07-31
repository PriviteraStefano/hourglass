# ADR-P-011 — Information Architecture & Role-Scoped Surfaces

---
tags: ["adr", "idea-layer", "structure", "insight", "ia", "navigation"]
---

# ADR-P-011 — Information Architecture: Pillar-Mapped Navigation & Role-Scoped Surfaces

**Status:** Proposed
**Date:** 2026-07-30
**Operationalizes:** [[VISION]] §3 (stakeholder map), §4 (four pillars) · **Concretizes:** [[ADR-P-004 — The Today View]] (landing rule) · **Applies naming of:** [[ADR-P-007 — Activity Ontology]] D-1 · **Gives surfaces to:** [[ADR-P-001 — Units vs Working Groups]] (execution structure) and [[ADR-P-008 — Availability & Employment Validity]] D-3/D-4 · **Revises:** [[ADR-P-004 — The Today View]] (ticket dependency softened — D-2 below)

---

## Context

The v0.1 frontend grew by mechanics: the sidebar groups "Tracking" (time, expenses, exports) and "Management" (contracts, customers, projects, org) describe *what the screens manipulate*, not *why they exist*. Consequences:

* The pillar model ([[VISION]] §4) is invisible in the product — Insight (exports) sits inside a Capture group, and Control's primary surface (approvals) is a disabled nav item with no page.
* The landing route redirects to `/time-entries` — a recording surface. Question #1 ("what should I be working on?") has no home, despite [[ADR-P-004 — The Today View]] deciding that landing becomes Today.
* Three decided entities have **no surface at all**: working groups (execution structure, [[ADR-P-001 — Units vs Working Groups]]), availability windows + HR curation ([[ADR-P-008 — Availability & Employment Validity]] D-3/D-4), and the activity-kinds catalog ([[ADR-P-007 — Activity Ontology]] D-2).
* The nav is flat — identical for every role, while the stakeholder map ([[VISION]] §3) already defines exactly which edge carries what.

If the IA does not encode the pillars, the product drifts back toward "a time & expense tracker" — the gap [[VISION]] §1 exists to close.

## Decision

### D-1 — Sidebar groups are job-language, pillar-mapped

Pillar names (Capture/Structure/Control/Insight) stay **architectural vocabulary** — they never appear as user-facing labels. Users get job words. Each group maps to exactly one pillar:

| Group | Items | Pillar | Answers | Lands |
|-------|-------|--------|---------|-------|
| *(landing)* | **Today** `/` | Insight | Q1 | v0.1 interim (D-2), full composition at v0.2 |
| **Track** | Time · Expenses · Tickets | Capture | Q1 | v0.1 · v0.1 · v0.2 ([[ADR-P-003 — Tickets as the Second Capture Layer]]) |
| **Work** | Activities · Working Groups | Structure | Q1/Q2 | v0.1 rename (D-6) + new WG surface (D-4) |
| **People** | Org · Availability | Structure | Q2 | v0.1 (Org) · P-008 surfaces follow schema |
| **Economics** | Contracts · Customers | Structure (commercial) | Q3 | v0.1 |
| **Review** | Approvals — manager + finance queues | Control | Q2 | v0.1 (D-3) |
| **Reports** | Exports: timesheet / expense / combined / payroll | Insight | Q3 | v0.1 · payroll view with [[ADR-P-008 — Availability & Employment Validity]] D-1c |
| **Admin** | Invitations · Activity kinds · Roles/Settings | Control | — | kinds catalog required by [[ADR-P-007 — Activity Ontology]] D-2 |

### D-2 — Landing is Today from v0.1; ticketless composition admitted

The Today view ships **pre-deployment** as the landing surface, composing only what exists: my pending approvals (for approvers), my draft/submitted/rejected entries this week, and the empty-state fallbacks [[ADR-P-004 — The Today View]] already mandates. Tickets join the composition when [[ADR-P-003 — Tickets as the Second Capture Layer]] ships (v0.2).

P-004's design rules bind from day one: **read-only composition** (no new state), **one answer, not a dashboard** (no charts/KPIs), **never blank**. This softens P-004's stated dependency on ADR-P-003; the revision is recorded in P-004's header.

### D-3 — Review is its own group, role-gated

Approvals leave the Track group and become the **Review** group — Control's primary surface. One page, two stage-filtered queues (manager stage, finance stage, per the BE-014 two-stage chain). Rendered only for users who hold an approval stage (WG manager/delegate, org-role manager/finance). **HR never sees Review** ([[ADR-P-008 — Availability & Employment Validity]] D-4: curator/consumer, never approver).

### D-4 — Working Groups get a top-level surface under Work

Execution structure needs a home of its own — not nested inside activity detail. Rationale: WG formation is *assignment*, and assignment is the single consumption point of availability/validity warnings ([[ADR-P-008 — Availability & Employment Validity]] D-3). Those warnings have nowhere to render without this surface. V3's knowledge-based team formation builds here later.

### D-5 — Surfaces follow the stakeholder map; visibility is role-scoped

The sidebar renders per org role. Matrix (✓ = full surface, read = read-only, — = hidden):

| Surface | Employee | Manager | Finance | HR | Customer |
|---------|----------|---------|---------|-----|----------|
| Today | ✓ | ✓ | ✓ | ✓ | — |
| Track | ✓ | ✓ | ✓ | ✓ | — |
| Work | read | ✓ form/edit | read | read | read own *(unspecced)* |
| People | declare own windows | subtree + holiday confirm | read | **curator: all windows, expiry queues** | — |
| Economics | — | read | ✓ | read (payroll link) | read own *(unspecced)* |
| Review | — | manager stage | finance stage | ✗ **never** (P-008 D-4) | — |
| Reports | own | subtree | org-wide | payroll view | — |
| Admin | — | — | — | — | org admin only |

Hidden in the sidebar is a UX scoping, not an authorization rule — backend enforcement stays authoritative (ADR-BE-005/006 stack). The **customer** column is deliberately minimal: the customer-facing surface is an open ADR candidate ([[VISION]] §3), not implied by this IA.

### D-6 — Route naming follows the ontology

* `/projects` → **`/activities`** (user-facing word per [[ADR-P-007 — Activity Ontology]] D-1).
* New routes: `/working-groups` (D-4), `/availability` (P-008), `/approvals` (D-3). `/` is reserved for Today (D-2).
* `/org-hierarchy` keeps its URL — bulk cosmetic renames are rejected pre-deploy; rename only where the ontology forces it.

### What stays rejected

* **Pillar names as user-facing labels** — architectural vocabulary only.
* **Dashboards/KPIs on Today** — P-004 boundary; that is V5/V6 dashboard infrastructure, sequenced behind it.
* **A customer-facing surface** — open ADR candidate, not smuggled in via the IA.
* **HR anywhere near Review** — reaffirmed from P-008 D-4/D-5.

## Consequences

* `sidebar.tsx` regroups per D-1; new pages: **Today** (landing), **Approvals queue**, **Working Groups**, **Availability** — v0.1 pre-deploy scope grows by the first three; Availability lands with P-008 implementation.
* E2E expectations that `/` → `/time-entries` must be updated (P-004's consequence, now immediate).
* Route rename `/projects` → `/activities` rides the ADR-P-007 big-bang migration — same pre-deploy window, no deployed URLs to break.
* Feature docs F07–F13 cite this ADR for where their surfaces live.
* The role-scoped sidebar needs the current membership's role available client-side at render time (already true via `GET /auth/me`).

## Related

* [[VISION]] §3 (stakeholder map), §4 (pillars), §7 (steering test)
* [[ADR-P-001 — Units vs Working Groups]] · [[ADR-P-003 — Tickets as the Second Capture Layer]] · [[ADR-P-004 — The Today View]] · [[ADR-P-006 — Out-of-Scope Enforcement]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-008 — Availability & Employment Validity]]
* Feature docs: [[F07-Org-Hierarchy-Units]] · [[F08-Customers]] · [[F09-Contracts]] · [[F10-Activities]] · [[F11-Time-Entries]] · [[F12-Expenses]] · [[F13-Exports]]
