---
tags: ["adr", "idea-layer", "capture", "structure", "origins"]
---

# ADR-P-013 — Origins

**Status:** Proposed
**Date:** 2026-08-03
**Operationalizes:** [[VISION]] §4 (Capture) · **Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 12–15) · **Decided by:** D-D, D-G, D-12, D-01..D-04, OQ1 (Phase 11 locked) · **Implemented by:** [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] (migrations 014–017)

---

## Context

Every activity exists for a reason — someone directed it, someone proposed it, or some demand it answers. That reason is currently invisible: activities carry no origin, and the request that caused the work cannot be reconstructed from system data. The ontology round closed the shape (D-D):

> **Origin = type + reference set.** A single fact written at creation; refs are set once. Lifecycle events (a proposal later *becoming* assigned work) belong to the activity's state/audit, NOT to origin.

This ADR pins the origin axis: the three origin types, how refs are stored and validated, the immutability rule, creation gates, and the proposal-approval model — so Phase 12/13 consumers and future readers have the record of truth outside the research note.

## Decision

### The three origin types (D-D, FND-01)

| `origin_type` | Reference set | Meaning |
|---------------|---------------|---------|
| `manager_assignment` | `assigned_by`, `assigned_to` | manager-directed work |
| `employee_proposal` | `proposed_by`, `reviewed_by` | employee-proposed work, approval-routed (D-G) |
| `customer_ticket` | `ticket_id` | demand from an internal ticket (revised [[ADR-P-003 — Tickets as the Second Capture Layer]]) |

Origin is a closed vocabulary (`manager_assignment`, `employee_proposal`, `customer_ticket`), DB CHECK-enforced. An activity with no origin (`origin_type IS NULL`) is the v0.1 legacy shape and remains valid.

### Storage: discriminator + nullable columns (D-01)

Origin refs are stored **directly on activities** as a discriminator plus five nullable columns (`assigned_by`, `assigned_to`, `proposed_by`, `reviewed_by`, `ticket_id`), with a table CHECK matching refs to type. **EAV and JSONB are rejected** (R4 resolution): they give no FK enforcement, no CHECKs, harder joins, and weaker immutability. DB FKs are used where possible (`assigned_by`/`assigned_to`/`proposed_by`/`reviewed_by` → `users(id)`, `ticket_id` → `tickets(id)`); everything else is validated at the service level.

### Same-org validation (D-02)

Refs must point at existing users/tickets **within the same org**: user refs validated via membership at the service level, `ticket_id` via `tickets.org_id = activities.org_id`. Cross-org refs are rejected with the same-org validation error. Per-type minimums: `employee_proposal` requires `proposed_by`; `manager_assignment` requires `assigned_by` + `assigned_to`; `customer_ticket` requires `ticket_id`.

### Immutability (D-03)

Origin refs are **set once at creation**. The service rejects any update that would change `origin_type` or any origin ref column with the sentinel **`ErrOriginImmutable`**. No DB trigger — no trigger precedent in the codebase; the service enforces the rule. This also means `reviewed_by` cannot be filled in post-creation — see the reviewed_by resolution below.

### Creation gates (D-04, FND-01/FND-02)

Origins surface via the existing `POST /activities` payload (optional `origin_type` + refs):

| `origin_type` | Who may create | Notes |
|---------------|----------------|-------|
| `manager_assignment` | manager / finance role | org-admin included per role model |
| `employee_proposal` | any employee, `proposed_by` = self | proposal starts `is_active=false` (D-12) |
| `customer_ticket` | manager+ (manager or finance) | pre-triage fast path allowed while the ticket is `open`/`triage` (OQ5) |

### Proposal approval (D-G, D-12, FND-02)

An employee proposal is an activity created with `origin_type='employee_proposal'` + `is_active=false` — **no new activity status column**. Approval routes through the same manager-stage machinery as entry approvals ([[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]]): internal/personal → proposer's unit manager; contract-linked → anchored WG's manager. **One routing rule for entries and proposals** (D-G) — the routing resolution is extracted to a shared package so the rules cannot drift. Approval flips `is_active=true`; the approval record and all lifecycle events land in `audit_logs` ([[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]]), **never in origin refs**.

### reviewed_by resolution (OQ1)

`reviewed_by` is deliberately **unconstrained** in v0.2: the CHECK requires only `proposed_by` (D-02). Phase 11 leaves `reviewed_by` **NULL at creation** — the actual approver is captured in the `proposal_approved` audit row at approval time. A create carrying a non-nil `reviewed_by` → `ErrInvalidRequest`. A future phase may pin `reviewed_by` at creation time; until then the audit log is the record of the approver.

### FND-04: read-path fallback (R4 resolution)

Origin refs are stored directly on activities (Phase 1 — honest data, not placeholders). The derivation rule from D-T — *when an activity's origin refs are empty, fall back to the first direction record* — is an **additive Phase-13 read-path enhancement**: no migration, the stored data stays authoritative for pre-direction activities. The read model must not be painted into a corner: refs are the primary source, derivation layers on later.

## Consequences

* Activities become attributable to the demand that caused them (ticket → activity → entries, revised P-003).
* Manager-assignment refs may *derive* from direction records in Phase 13 (D-T lean) — an additive read-path fallback, not a D-D revision.
* The CHECK + service gates prevent origin spoofing (mass assignment) and cross-org refs; `proposed_by = self` is enforced.
* Proposals and entries share one approval-routing rule (D-G) — no drift between the two paths.
* Schema shape (migration 015) and the semantic resolutions are recorded in [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]].

## Related

* [[ADR-P-003 — Tickets as the Second Capture Layer]] — `customer_ticket` origin refs require the tickets entity (revised P-003; chain ticket → activity → entries)
* [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] — the three-plane model: origin is the plan-plane attribution of the fact
* [[ADR-P-014 — Funding Sources & Beneficiary Unit]] · [[ADR-P-015 — Direction: The Plan Plane]] (Phase 13 derivation source)
* [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] · [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]]
* [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 12–15 — record of truth)
