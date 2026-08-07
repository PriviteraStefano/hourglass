---
tags: ["adr", "backend", "schema", "tickets", "origins", "audit"]
---

# ADR-BE-016 — Origins, Tickets & Audit: Schema Encoding

**Status:** Proposed
**Date:** 2026-08-03
**Code:** `migrations/014…017`, `internal/core/services/ticket/`, `internal/core/services/activity/`, `internal/core/services/contract/`, `internal/core/ports/`, `internal/adapters/secondary/postgres/`
**Resolves:** Phase 11 semantic resolutions (research OQ1…OQ6), locked decisions D-05/D-06/D-10/D-13/D-14 · **Basis:** [[ADR-P-003 — Tickets as the Second Capture Layer]] (rev.), [[ADR-P-013 — Origins]], [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] · **Supersedes:** [[ADR-BE-012 — Audit Log Writes]] scope note (extended below)

---

## Context

Phase 11 encodes three ontology extensions in the schema: the **origin axis** on activities (FND-01), the **ticket domain** with lifecycle, triage, and dismissal guard (TICK-01..05), and **sold hours** on contracts (FND-03) — plus the **general audit log** that Phase 12 (coverage) and Phase 13 (direction) reuse (D-05). Four additive migrations (014–017) shape the tables. This ADR pins the encoding decisions and the semantic resolutions the code plans (11-04..11-06) rely on, so the schema's intent survives as the record of truth for later phases and future readers.

## Decision

### Migration shapes (014–017)

**014 — tickets + ticket_comments.** `tickets` (id, org_id, title, description, kind, status, requester_id, assignee_id, `dismissed_hours DECIMAL(8,2)`, created_at, updated_at) with the kind CHECK `('question','bug','change','evolution')` and the status CHECK `('open','triage','planned','in_progress','resolved','closed','dismissed')` — closed vocabulary, DB-enforced (house style). `ticket_comments` is append-only (ticket_id CASCADE, author_id, body, created_at), FKs indexed for per-org listing and status-filtered queues.

**015 — activity origins.** `origin_type VARCHAR(50)` discriminator + five nullable ref columns (`assigned_by`, `assigned_to`, `proposed_by`, `reviewed_by`, `ticket_id`, all FK to `users(id)`/`tickets(id)`), `activities_origin_type_check` (closed origin vocabulary) and `activities_origin_refs_check` (refs-to-type matching, see house rule below), plus `idx_activities_ticket_id`. `reviewed_by` deliberately unconstrained (D-02, OQ1). Numbered after 014 so the `tickets(id)` FK resolves at apply time (A8).

**016 — contract sold hours.** `contract_type VARCHAR(50)` (`'project'`,`'support'`), `sold_hours DECIMAL(8,2)` (scale matches `time_entries.hours`), `sold_period VARCHAR(10)` (`'month'`,`'quarter'`,`'year'`), with `contracts_sold_check` enforcing type consistency (see semantics below).

**017 — general audit_logs.** Append-only table (id, org_id, entity_type, entity_id, action, actor_id nullable, comment, `payload JSONB`, created_at) with the entity-scoped index `(entity_type, entity_id, created_at)` — the dominant history-read path. The schema exposes **no UPDATE/DELETE paths**.

All four are **purely additive, no backfill** (D-16): every new column is nullable; legacy rows pass every new CHECK.

### House rule: the three-valued-logic CHECK guard (Pitfall 1)

Every multi-column CHECK carries the **`discriminator IS NULL OR (<per-type rules with explicit IS [NOT] NULL>)`** guard. PostgreSQL CHECKs pass on TRUE **or NULL**, so a naive per-type CHECK would either reject legacy rows (NULL discriminator) or let mixed refs through (e.g. `customer_ticket` with `assigned_by` set). The guard (a) keeps every legacy row valid and (b) pins the *entire* ref set per type. Applied to `activities_origin_refs_check` (D-01) and `contracts_sold_check` (D-08/D-09); any future discriminator CHECK must follow the same shape.

### Three-layer ticket model (D-06)

Tickets are split across three layers — **state / comments / audit** — per the user's explicit design:

1. **State** — current ticket status/fields in `tickets`
2. **Comments** — first-class, separate storage in `ticket_comments` (the "description around the ticket", not audit rows)
3. **Audit** — `audit_logs` tracks **everything**: every state change, every comment, every transition, the dismissal — one stream

Comments are created via their own endpoint and each comment **also** writes an audit row. **No UPDATE/DELETE paths exist on any of the three** (TICK-05): no endpoints, no repo methods — the ticket history is immutable by construction.

### Audit-write durability: synchronous in-transaction (OQ4, Pitfall 2)

Ticket audit rows are written **synchronously inside the same transaction** as the state change — create, triage, transitions, dismissal, comments all write their audit rows in the same tx (required for atomic triage per D-10). The [[ADR-BE-012 — Audit Log Writes]] fire-and-forget pattern **stays for entry approvals** (approvals are regenerable) but is **NOT used for tickets** (Pitfall 2): the ticket event stream *is* the ticket's history — it must not be absent. The deferred durability upgrade (outbox/queue) remains possible because the `audit_logs` table shape is consumer-neutral — a future outbox consumes the same rows; no destructive/update paths are added now.

> **Transparency note (context_compliance I-3):** the durability write-mechanism itself was a **user-deferred choice** ("it is not part of this phase, let's push it back for later" — CONTEXT.md Deferred Ideas). This plan picks synchronous in-transaction writes per D-10 (audit rows in the triage tx) + the research OQ4 recommendation — consistent with the locked decision — and records the outbox as the documented successor, so the deferred choice stays reversible.

### Triage validates activity plans inside the transaction (Pitfall 7)

Triage creates 1..N activities inside the same tx as the ticket state change — and runs the **same validations `POST /activities` enforces** (kind in org catalog, parent same-org, contract exists, origin CHECK) **inside that tx**. No validation is split out of the tx; the triage method reuses the activity repo queries within the transaction (11-06 Task 2).

## Semantic resolutions

### Terminal activity (OQ2)

An activity is **terminal** when it has **no non-terminal time entries** on its linked-activity subtree — entries with `status IN ('draft','submitted','pending_manager','pending_finance')` and `is_deleted = false` — computed via a **recursive CTE** over the subtree (`HasNonTerminalActivities`). `is_active=false` alone is NOT a work-progress signal (it is the proposal flag, D-12). The `resolved` transition requires all linked activities terminal.

### Dismissal guard and signature (D-13, TICK-04)

The dismissal guard ships on **raw Σ logged hours** (submitted + approved, `is_deleted` excluded) across linked activities, behind the **stable port signature `LoggedHours(ctx, ticketID) (float64, error)`**. Phase 12 upgrades the computation to net-of-compensations when compensation machinery lands — **signature unchanged, computation swapped**. The dismissed ticket carries `dismissed_hours` (set at dismissal, migration 014) and the dismissal audit row records the same value.

### Transition matrix (A7/OQ6)

Pinned edges, enforced by the service (`CanTransition`); any other edge → `ErrInvalidTransition`, audit row only on success:

```
open → triage        triage → planned      triage → dismissed
open → dismissed     planned → in_progress in_progress → resolved
resolved → closed    resolved → in_progress (reopen)
```

`closed` and `dismissed` are terminal; reopen exists only from `resolved`; dismissed is reachable from `open` and `triage` (guarded).

### Pre-triage fast path (OQ5)

`POST /activities` with origin `customer_ticket` is **allowed while the ticket is `open`/`triage`** (urgent work before triage), manager+ gated (D-04). The dismissal guard and the resolved check both read linked activities via `ticket_id`, so the fast path composes with the lifecycle.

### sold_hours semantics (D-07..D-09, D-N)

| `contract_type` | Semantics | `contracts_sold_check` |
|-----------------|-----------|------------------------|
| `project` | `sold_hours` = lifetime total of the contract | `sold_period` must stay NULL |
| `support` | `sold_hours` = hours per period | `sold_hours` + `sold_period` both required |
| NULL (legacy) | treated as project; no funding commitment yet | passes via the 3VL guard |

Read/write through the existing contract endpoints (FND-03); V5 mines sold vs Σ actual hours (D-N).

### BE-012 scope note (extended)

[[ADR-BE-012 — Audit Log Writes]] was written for approval-table writes. Its scope now covers the **general `audit_logs` table** with the split: entry approvals keep the BE-012 fire-and-forget pattern (detached context, logged failures); ticket events use synchronous in-transaction writes (this ADR). The BE-012 successor (durable queue) applies to the general table as the documented outbox upgrade.

## Consequences

* The schema is the Phase 12/13 substrate: coverage and direction consume `audit_logs` and the origin refs without further migration.
* The 3VL guard rule becomes a house convention — regression-guarded by the 014–017 migration cycle tests (Pitfall-1 functional assertions).
* TICK-05's immutability guarantee is real: no update/delete paths exist at the schema, repo, or endpoint level; ticket events cannot be silently lost (synchronous in-tx).
* The deferred outbox choice stays reversible: consumer-neutral table shape, successor documented.
* ⚠️ In-transaction audit writes mean a ticket operation fails as a whole if the audit row cannot be written — accepted: the event stream is the guarantee (OQ4).

## Related

* [[ADR-P-003 — Tickets as the Second Capture Layer]] (rev.) — lifecycle, kinds, permission gates, dismissal guard
* [[ADR-P-013 — Origins]] — origin types, immutability, creation gates, reviewed_by resolution
* [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] — three-plane model; Phase 12 coverage consumes this schema
* [[ADR-BE-012 — Audit Log Writes]] (scope extended) · [[ADR-BE-004 — Database Migrations]] (append-only rule) · [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]]
* `migrations/014…017` (up/down pairs) · `internal/adapters/secondary/postgres/ontology_extension_migrations_test.go` (cycle tests)
* [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (record of truth)
