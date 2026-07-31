# ADR-P-003 — Tickets as the Second Capture Layer

---
tags: ["adr", "idea-layer", "capture", "tickets"]

---

# ADR-P-003 — Tickets as the Second Capture Layer

**Status:** Proposed
**Date:** 2026-07-28
**Operationalizes:** [[VISION]] §6 V2 · **Blocks:** V1 ([[ADR-P-004 — The Today View]]), V4 (change requests)

## Context

The Capture pillar today records **effort** (time entries) and **cost** (expenses) — but not **demand**. The requests that cause work ("update that report", "the customer wants the export changed", "fix this before Friday") live in chat, email, and memory. They have no owner, no status, no priority, and no link to the time they eventually consume.

Consequences of the gap:

* "What should I be working on?" cannot be answered from system data — the input side of work is invisible.
* Time entries are unattributable to the requests that caused them; contract/project history is effort-only.
* Scope changes on projects happen out-of-band and are indistinguishable from the original plan.

## Decision

Introduce **tickets** as the Capture layer for demand. A ticket is a tracked request with:

* **Origin** — `internal` (raised by an org member) or `external` (raised on behalf of a customer)
* **Requester** and **owner** — who asked, who is responsible for resolving it
* **Status** — a small state machine (e.g. `open → in_progress → resolved / dismissed`)
* **Priority** — the input to the Today view's ordering
* **Optional links** — to a project/contract, and to the time entries booked against it

Tickets close the loop: **demand → effort → cost** becomes traceable end-to-end.

### Hard boundary (per [[VISION]] §8)

Tickets are **demand tracking, not task execution**. Explicitly excluded:

* No Kanban board, swimlanes, or sprint planning
* No story points, estimation poker, or velocity metrics
* No comment threads (a resolution note, not a conversation)
* No nested sub-task trees

If a need looks like project management, it belongs to a PM tool. Hourglass tickets exist so that *requests stop being invisible* — and so demand can be linked to the time and money it consumed.

## Consequences

* The Capture pillar becomes complete: demand, effort, cost.
* V1 (Today view) gets its primary data source — "what's on my plate" = my owned open tickets by priority.
* V4 (project change requests) is reframed as a ticket *type*, not a new subsystem.
* Contracts/projects gain demand history, which V5 (pricing analytics) later mines ("similar requests cost us X hours").
* ⚠️ Requires a domain design (ticket entity, state machine, links) before any SPEC — that design is the first backend artifact of v0.2.

## Related

* [[VISION]] §4 (Capture), §6 V2, §8 (task-execution anchor)
* [[ADR-P-004 — The Today View]] (primary consumer)
* [[ADR-P-002 — Four Pillars & Feature Purposes]]
