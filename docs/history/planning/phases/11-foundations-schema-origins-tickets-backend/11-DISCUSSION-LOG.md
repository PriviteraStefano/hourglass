# Phase 11: Foundations — Schema + Origins + Tickets Backend - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-02
**Phase:** 11-Foundations — Schema + Origins + Tickets Backend
**Areas discussed:** Origin refs storage & validation, Ticket event stream (BE-012), sold_hours semantics, Triage shape & permissions, Dismissal-guard hours, Transition enforcement, Ticket permissions, Legacy data check

---

## Origin refs storage & validation

| Option | Description | Selected |
|--------|-------------|----------|
| Discriminator + columns | origin_type CHECK + nullable UUID columns + refs-to-type CHECK | ✓ |
| JSONB refs blob | Flexible, not FK-enforceable | |
| Separate origins table | Clean separation but extra join; R4 said "directly on activities" | |

| Option | Description | Selected |
|--------|-------------|----------|
| Strict same-org validation | Refs point at existing users/tickets in same org; per-type required refs | ✓ |
| Existence-only validation | UUIDs valid + rows exist, no org check | |
| Type-CHECK only | No ref validation beyond discriminator | |

| Option | Description | Selected |
|--------|-------------|----------|
| Service-enforced immutability | Service rejects origin changes; no trigger | ✓ |
| DB trigger too | Defense in depth, no precedent | |
| Domain-level only | Guard in Update method only | |

| Option | Description | Selected |
|--------|-------------|----------|
| Extend create endpoint + role gates | POST /activities gains origin; manager/employee/ticket role gates | ✓ |
| No role gates | Any member sets any origin | |
| Dedicated sub-flows | Propose/assign/triage endpoints set origin | |

**User's choice:** Discriminator + columns; Strict same-org validation; Service-enforced immutability; Extend create endpoint + role gates
**Notes:** All recommended options accepted. EAV already ruled out by R4.

---

## Ticket event stream (BE-012)

| Option | Description | Selected |
|--------|-------------|----------|
| General audit_logs table | entity_type/entity_id/action/payload; reused by Phase 12/13 | ✓ |
| Dedicated ticket_events | Mirrors time_entry_approvals pattern | |
| General table + views | Per-domain wrappers | |

| Option | Description | Selected |
|--------|-------------|----------|
| Events = audit rows only | Comments/notes all become audit rows | |
| Split: comments table + audit | Separate comment storage | |
| Dual-write (state + audit) | State fields + audit rows | |

| Option | Description | Selected |
|--------|-------------|----------|
| Transactional audit writes | Synchronous in-transaction | |
| Fire-and-forget like today | ADR-BE-012 model | |
| Outbox/queue | BE-012 documented successor | |

**User's choice (freeform):** General audit_logs table + three-layer split — "I think we should differentiate between the state and the audit log, even comments. Comments are the description around the ticket, so they provide more information. The stage describes whatever the ticket is — not the status of the ticket. Lastly, the audit log tracks everything, every event about it, including new comments and state updates. We may need a better audit infrastructure."
**Notes:** Audit durability explicitly deferred — "it is not part of this phase, let's push it back for later".

---

## sold_hours semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Total per contract | Lifetime total for V5 mining | |
| Per-period breakdown | Monthly/period rows | |
| Total + unit field | Hours vs days | |

| Option | Description | Selected |
|--------|-------------|----------|
| contract_type + dual semantics | project → total; support → per-period | ✓ |
| Total only; buckets carry support | Support figure via Phase 12 buckets | |
| Explicit dual columns | sold_hours_total + sold_hours_period | |

| Option | Description | Selected |
|--------|-------------|----------|
| Nullable + period column | sold_hours DECIMAL + sold_period CHECK | ✓ |
| Required at creation | NOT NULL for new contracts | |
| Read-only this phase | No write path | |

**User's choice (freeform):** "it depends on what the contract is: if it is a project total hours, if it support per hours per period, we should distinguish between the two type, I don't think there are any other differences to look out for" → contract_type + dual semantics (project → total, support → per-period); sold_hours DECIMAL nullable + sold_period CHECK (month/quarter/year), required only for support.
**Notes:** Contracts table has no type field today — contract_type is a new discriminator; legacy contracts stay NULL (treated as project).

---

## Triage shape & permissions

| Option | Description | Selected |
|--------|-------------|----------|
| Atomic triage call | POST /tickets/{id}/triage creates 1..N activities in-transaction | ✓ |
| Two-step (triage then create) | Triage classifies; activities via normal endpoint | |
| Non-atomic with error note | Lazy validation | |

| Option | Description | Selected |
|--------|-------------|----------|
| Manager + finance | Triage gated to both roles | ✓ |
| Any employee | Open triage | |
| Creator or manager | Ticket creator convenience | |

| Option | Description | Selected |
|--------|-------------|----------|
| is_active=false until approved | Proposal model; approval flips is_active | ✓ |
| New activity status column | Explicit proposed→active vocabulary | |
| No gating, audit-only | Proposals active immediately | |

**User's choice:** Atomic triage call; Manager + finance; is_active=false until approved
**Notes:** All recommended options accepted.

---

## Dismissal-guard hours

| Option | Description | Selected |
|--------|-------------|----------|
| Raw Σ now, net in P12 | Guard on submitted+approved Σ; Phase 12 swaps in net-of-compensations | ✓ |
| Approved-only raw Σ | Excludes submitted/draft | |
| Defer guard to P12 | Ship net from day one | |

**User's choice:** Raw Σ now, net in P12
**Notes:** Guard signature unchanged, computation swapped in Phase 12.

---

## Transition enforcement

| Option | Description | Selected |
|--------|-------------|----------|
| DB CHECK + service state machine | DB enforces vocabulary; service enforces transition rules | ✓ |
| Add DB trigger | Defense in depth, no precedent | |
| Service-only, no DB CHECK | Plain VARCHAR | |

**User's choice:** DB CHECK + service state machine

---

## Ticket permissions

| Option | Description | Selected |
|--------|-------------|----------|
| Backend gates now | create/update/comment/view role+ownership gates in Phase 11 | ✓ |
| Defer to Phase 18 | Open until surfaces land | |
| Gate triage/dismissal only | Others open to all members | |

**User's choice:** Backend gates now
**Notes:** "Auto-approved" = no approval workflow step, not no permission.

---

## Legacy data check

| Option | Description | Selected |
|--------|-------------|----------|
| Additive only, no backfill | All new columns nullable; legacy contracts NULL | ✓ |
| Backfill type=project | Data-writing migration | |
| Force classification | Reject entries on NULL-type contracts | |

**User's choice:** Additive only, no backfill

---

## the agent's Discretion

- Exact ticket endpoint list and URL shapes (within atomic-triage + permission decisions)
- Audit-log read-path exposure (history endpoint design)
- Test layout for new domain packages (follow existing per-package suite pattern)
- Contract-type exposure in contract read-model responses

## Deferred Ideas

- **Better audit infrastructure (durability/queue)** — explicitly pushed beyond Phase 11 by the user; the `audit_logs` table shape must not preclude a later durability upgrade (transactional writes, fire-and-forget, or outbox/queue — the documented ADR-BE-012 successor).
