---
tags: ["adr", "idea-layer", "capture", "tickets"]
---

# ADR-P-003 — Tickets as the Second Capture Layer

**Status:** Proposed
**Date:** 2026-07-28
**Revised:** 2026-08-03 (Phase 11 — v0.2 lifecycle: seven statuses + reopen + guarded dismissal, closed kind set, internal-only + permission gates; see delta section)
**Operationalizes:** [[VISION]] §6 V2 · **Blocks:** V1 ([[ADR-P-004 — The Today View]]), V4 (change requests)
**Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 4, 12–13) · **Decided by:** D-A, D-E, D-H, D-M, D-14, D-15 · **Implemented by:** [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]]

## Context

The Capture pillar today records **effort** (time entries) and **cost** (expenses) — but not **demand**. The requests that cause work ("update that report", "the customer wants the export changed", "fix this before Friday") live in chat, email, and memory. They have no owner, no status, no priority, and no link to the time they eventually consume.

Consequences of the gap:

* "What should I be working on?" cannot be answered from system data — the input side of work is invisible.
* Time entries are unattributable to the requests that caused them; contract/project history is effort-only.
* Scope changes on projects happen out-of-band and are indistinguishable from the original plan.

## Decision

Introduce **tickets** as the Capture layer for demand. A ticket is a tracked request with:

* **Origin** — `internal` in v0.2 (D-E): raised by an org member. External desk intake is a future *port* (hexagonal secondary adapter), not a v0.2 feature — tickets are **not customer-facing**.
* **Requester** and **owner** — who asked, who is responsible for resolving it
* **Kind** — a **closed set**: `question · bug · change · evolution` (D-H, TICK-01); kinds drive funding-eligibility rules in Phase 12 (stored now, rules later)
* **Status** — the v0.2 state machine (D-A/D-14): `open → triage → planned → in_progress → resolved → closed`, plus `reopen` (`resolved → in_progress`) and guarded `dismissed` (from `open` or `triage`)
* **Optional links** — to activities via the origin axis ([[ADR-P-013 — Origins]]: `customer_ticket` → `ticket_id`); the chain is **ticket → activity → entries** — tickets never reference entries directly (TICK-03)
* **Priority** — deferred; not part of the v0.2 field set (a later additive migration if the Today view needs it)

Tickets close the loop: **demand → effort → cost** becomes traceable end-to-end.

### v0.2 lifecycle (D-A, D-14)

```
open → triage → planned → in_progress → resolved → closed
  │        │
  │        └────→ dismissed    (dismissal guard: Σ logged hours == 0)
  └────────────→ dismissed    (dismissal guard: Σ logged hours == 0)
resolved → in_progress        (reopen — one demand thread; follow-up-ticket chains rejected)
```

| Rule | Enforcement |
|------|-------------|
| Status vocabulary `open, triage, planned, in_progress, resolved, closed, dismissed` | DB CHECK (migration 014) |
| Transition edges (incl. reopen, guarded dismissal) | ticket service `CanTransition`; any other edge → `ErrInvalidTransition`; one audit row per transition |
| `resolved` requires all linked activities **terminal** | service check over the linked-activity subtree |
| `resolved → in_progress` (reopen) | allowed — the demand thread continues; follow-up-ticket chains are rejected (D-A: one demand thread per real-world request) |
| `dismissed` | only from `open` or `triage`, guarded by the dismissal guard (D-M, TICK-04) |
| `closed`, `dismissed` | terminal states |

### Dismissal guard and note (D-M, TICK-04)

A ticket with logged hours **cannot be dismissed**: the `open|triage → dismissed` transition is blocked while any linked activity carries logged hours. v0.2 ships the guard on **raw Σ** (submitted + approved entries, `is_deleted` excluded); Phase 12 swaps the computation for net-of-compensations when compensation machinery lands — guard signature unchanged. The hours are facts and outlive the demand; their *cost* is decided at allocation time, never by dismissal.

A dismissed ticket keeps the note **"dismissed with N h logged"** — N stored in the `dismissed_hours` column (set at dismissal) and mirrored by the dismissal audit row.

### Kinds (D-H, TICK-01)

Kinds are a **closed set**: `question · bug · change · evolution`, enforced by DB CHECK. Unlike `activity_kinds` (a catalog), ticket kinds are fixed vocabulary — no per-org extension in v0.2.

### Permission gates (D-15, TICK-06)

Tickets are internal-only in v0.2 (D-E). "Auto-approved" means no approval-workflow step — **not** no permission:

| Action | Gate |
|--------|------|
| Create | any employee |
| Update / comment | owner / assignee / manager+ |
| Triage / dismiss | manager + finance |
| View | all internal members |

The `customer` role is **rejected** — tickets are not customer-facing per the v0.2 out-of-scope table (research Part 13). Phase 18 surfaces add UI but never loosen these backend gates.

### Hard boundary (per [[VISION]] §8)

Tickets are **demand tracking, not task execution**. Explicitly excluded:

* No Kanban board, swimlanes, or sprint planning
* No story points, estimation poker, or velocity metrics
* No comment threads (a resolution note, not a conversation)
* No nested sub-task trees

If a need looks like project management, it belongs to a PM tool. Hourglass tickets exist so that *requests stop being invisible* — and so demand can be linked to the time and money it consumed.

## Delta from the 2026-07-28 draft (Phase 11 revision)

* **Lifecycle sketch → full v0.2 state machine** (D-A, D-14): the three-status sketch (`open → in_progress → resolved / dismissed`) becomes `open → triage → planned → in_progress → resolved → closed`, plus `reopen` (`resolved → in_progress`) and `dismissed` from `open|triage`. `resolved` requires all linked activities terminal.
* **Kinds introduced** as a closed set `question · bug · change · evolution` (D-H, TICK-01).
* **Internal-only confirmed** for v0.2 (D-E) with backend permission gates per D-15 (create/update/comment/triage/dismiss/view; `customer` role rejected).
* **Dismissal guard added** (D-M, TICK-04): `open|triage → dismissed` blocked while any linked activity carries logged hours; `dismissed_hours` note column + audit row.
* **Chain pinned** as ticket → activity → entries — tickets never reference entries directly (TICK-03).
* **Hard boundary preserved verbatim** — still demand tracking, not task execution: no kanban, no sub-task trees, no comment threads as conversation.
* Schema encoding recorded in [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] (migrations 014–017, three-layer model).

## Consequences

* The Capture pillar becomes complete: demand, effort, cost.
* V1 (Today view) gets its primary data source — "what's on my plate" = my owned open tickets by priority.
* V4 (project change requests) is reframed as a ticket *type*, not a new subsystem.
* Contracts/projects gain demand history, which V5 (pricing analytics) later mines ("similar requests cost us X hours").
* ✅ The required domain design (ticket entity, state machine, links) is now the first backend artifact of v0.2: migrations 014–017 + [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]].
* ⚠️ Triage is a permission gate, not an approval step: employees can create tickets, only manager/finance triage them (D-11/D-15).

## Related

* [[VISION]] §4 (Capture), §6 V2, §8 (task-execution anchor)
* [[ADR-P-004 — The Today View]] (primary consumer) · [[ADR-P-002 — Four Pillars & Feature Purposes]]
* [[ADR-P-013 — Origins]] (customer_ticket refs — the ticket → activity link) · [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] (three-plane model; dismissal guard D-M)
* [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] (schema + lifecycle encoding)
* [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 4, 12–13 — record of truth)
