# Phase 11: Foundations — Schema + Origins + Tickets Backend - Context

**Gathered:** 2026-08-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend-only phase (no UI). The three-plane ontology takes its first shape server-side:

- **Origins** — activities carry an origin (type + reference set, FND-01), employees can propose activities with approval routed through the activity's approval routing (FND-02), origin refs stored directly on activities with an empty-refs fallback to the first direction record on read — additive, lands in Phase 13 (FND-04)
- **sold_hours** — contracts carry `sold_hours` for V5 mining (FND-03)
- **Tickets** — the ticket entity exists server-side: lifecycle + triage + dismissal guard + immutable event stream + first-class comments (TICK-01..05)
- **ADRs** — ADR-P-003 revision + ADR-P-013 drafted and recorded in the vault decisions folder

Deliverables are API endpoints + migrations + domain/ports/services/adapters + integration tests. All migrations append-only per ADR-BE-004 with up/down pairs + cycle tests.

</domain>

<decisions>
## Implementation Decisions

### Origin refs storage & validation
- **D-01:** Origin refs stored on activities as a **discriminator + nullable columns**: `origin_type VARCHAR CHECK ('manager_assignment','employee_proposal','customer_ticket')` + nullable UUID columns `assigned_by`, `assigned_to`, `proposed_by`, `reviewed_by`, `ticket_id` + a CHECK constraint matching refs to the type (e.g., customer_ticket → ticket_id set, others null). EAV ruled out by R4. No JSONB blob.
- **D-02:** **Strict same-org validation**: refs must point at existing users/tickets within the same org; employee-proposal requires `proposed_by`; manager-assignment requires `assigned_by` + `assigned_to`; customer-ticket requires `ticket_id` (an internal ticket of the org). DB FKs where possible, rejected at service level otherwise.
- **D-03:** **Service-enforced immutability** — service rejects any update changing `origin_type` or refs after creation (D-D "set once"). No DB trigger (no trigger precedent in codebase).
- **D-04:** Origin creation surfaces via **extending the existing `POST /activities` payload** (optional `origin_type` + refs) with role gates: manager_assignment → manager/org-admin; employee_proposal → any employee (proposed_by = self); customer_ticket → manager+.

### Ticket event stream (BE-012) & comments
- **D-05:** **General `audit_logs` table** (id, org_id, entity_type, entity_id, action, actor_id, comment, payload JSONB, created_at) — tickets use it in Phase 11; Phase 12 coverage changes and Phase 13 direction events reuse it. BE-012 ADR gets extended from approval-table-only to a real table. Today the "audit trail" is `time_entry_approvals` doubling as history — that precedent moves to the new table for tickets.
- **D-06:** **Three-layer split** (user's explicit design): (1) **state** — current ticket status/fields in the `tickets` table; (2) **comments** — separate first-class storage (comments are "the description around the ticket", not audit rows); (3) **audit log** — tracks EVERYTHING, every event, including new comments and state updates. Not a dual-write of state; comments are not audit-only.

### sold_hours semantics
- **D-07:** Semantics **depend on contract type** (user: "it depends on what the contract is: if it is a project total hours, if it support per hours per period").
- **D-08:** Add `contract_type VARCHAR CHECK ('project','support')` to contracts — NULL stays ambiguous (existing v0.1 contracts treated as project). `project` → sold_hours = lifetime total; `support` → sold_hours = hours per period.
- **D-09:** Shape: `sold_hours DECIMAL` nullable + `sold_period VARCHAR CHECK ('month','quarter','year')` nullable, required only when `contract_type = 'support'`. Read/write via existing contract endpoints (FND-03 read/write requirement). V5 mines sold vs Σ actual.

### Triage shape & permissions
- **D-10:** **Atomic triage call** — `POST /tickets/{id}/triage` accepts kind (re)classification + 1..N activity definitions (name, parent, contract, governance); activities created in the same transaction; ticket → planned; audit rows for triaged + activities-created. No partial states.
- **D-11:** **Manager + finance may triage** (finance needed when funding eligibility is set in Phase 12); employees can create tickets but not triage.
- **D-12:** **Proposal approval** (FND-02) = activity created with `origin_type='employee_proposal'` + `is_active=false`; approval (routed via activity approval routing) flips `is_active=true`; approval record + lifecycle events land in audit_logs, never in origin refs (D-G). No new activity status column.

### Ticket lifecycle & permissions
- **D-13:** **Dismissal guard** (TICK-04) ships on **raw Σ logged hours** (submitted + approved) across linked activities; the "dismissed with N h logged" note carries that number. Phase 12 upgrades the computation to net-of-compensations when compensation machinery lands — guard signature unchanged, computation swapped.
- **D-14:** **DB CHECK + service state machine**: status vocabulary enforced by DB CHECK (house style — governance_model, entry statuses all CHECK-enforced); transition rules (allowed edges, resolved-blocks-on-non-terminal-activities, reopen resolved→in_progress) enforced in the ticket service with audit rows per transition. No DB triggers.
- **D-15:** **Backend permission gates land now** (TICK-06 "auto-approved with permission control"): create → any employee; update/comment → owner/assignee/manager+; triage → manager+finance; view → all org members. "Auto-approved" = no approval workflow step, not no permission. Phase 18 surfaces add UI but don't loosen backend gates.

### Legacy data
- **D-16:** **Purely additive migration, no backfill**: v0.1 has zero tickets, zero origin refs, zero sold_hours. All new columns nullable; legacy contracts get `contract_type NULL` (treated as project). Append-only per ADR-BE-004.

### the agent's Discretion
- Exact endpoint list for tickets CRUD/comments/history routes and their URL shapes (within the atomic-triage and permission decisions above)
- Audit-log read-path exposure shape (history endpoint design)
- Test layout for the new domain packages (follow existing per-package suite pattern)
- Contract-type exposure in existing contract read-models (response fields)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Ontology research (record of truth)
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` — All decisions D-A…D-AA closed here. Specifically: D-D (origin = type + ref set, Part 12), D-G (proposal approval routing), D-H (ticket kinds, funding rules), D-M (dismissal guard), D-N (sold_hours "exact shape TBD by the backend ADR"), Q2/Q3 (kinds, buckets), R4 resolution (origin refs storage — §"R4 resolution: origin refs storage", ~line 522), Part 15 (build order). Table sketches inside are illustrative, NOT schema proposals.

### ADRs (vault decisions)
- `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` — Proposed; revised this phase. Tickets = demand tracking, NOT task execution (no kanban, no sub-task trees, no comment threads as conversation; resolution note, not conversation). Hard boundary list MUST be respected.
- `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md` — Drafted (Proposed). Three-plane context; what V5 mines (sold vs actual).
- `hourglass-vault/decisions/backend/ADR-BE-012 — Audit Log Writes.md` — Accepted. Current audit = fire-and-forget writes into approval tables; extended this phase to the general `audit_logs` table. Durability upgrade deferred (see Deferred Ideas).
- `hourglass-vault/decisions/backend/ADR-BE-004` — Append-only migrations rule (referenced; file under `hourglass-vault/decisions/backend/`)
- New ADRs drafted this phase: ADR-P-003 revision + ADR-P-013 (origin axis), plus the BE encoding ADR covering schema (per milestone convention "Each backend phase drafts its ADR + BE encoding ADR")

### Milestone docs
- `.planning/ROADMAP.md` — Phase 11 entry: goal, requirements (FND-01..04, TICK-01..05), 9 success criteria
- `.planning/REQUIREMENTS.md` — FND-01..04 and TICK-01..06 requirement text (TICK-06 is the frontend twin — backend gates per D-15)

### Codebase (read-only context)
- `migrations/011_activity_ontology.up.sql` — `activities` table: the base schema origin columns extend (kind is a catalog FK, NOT an enum — origin_type is a real enum, this is the deliberate difference)
- `internal/core/domain/activity/activity.go` — Activity entity: has `IsActive bool`, no status column (proposal model per D-12)
- `internal/adapters/secondary/postgres/time_entry_repository.go` — `AuditLogRepository` writes into `time_entry_approvals` (the pattern the general audit_logs table replaces for tickets, line ~325)
- `migrations/000_full_schema.up.sql` — `contracts` table (no type field today; line ~125) and `time_entry_approvals` (line ~301)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/adapters/secondary/postgres/time_entry_repository.go` — `AuditLogRepository` precedent (fire-and-forget writes, ADR-BE-012); the general `audit_logs` repo can follow this shape
- `internal/core/domain/activity/activity.go` — Activity entity + factory patterns; origin fields/validation extend this domain package
- `internal/adapters/secondary/postgres/contract_repository.go` — contract read/write paths for `sold_hours`/`contract_type`/`sold_period` columns
- Approval-routing machinery (activity → WG → manager/delegate) referenced by FND-02 proposal approval — resolve via existing services in `internal/core/services/`

### Established Patterns
- Hexagonal: domain → ports → services → HTTP handlers → postgres repos; services own invariants, DB owns shapes (CHECK constraints)
- Hand-written SQL with pgx, no ORM; migrations append-only with up/down pairs
- Roles/status vocabulary enforced via DB CHECK constraints (house style, D-14)
- API response envelope `{ data | error }` via `pkg/api/response.go`
- Integration tests via testcontainers-go; per-package test suites
- Sentinel errors in domain (`internal/core/domain/*/errors.go`), `wrapPGError` in postgres adapters

### Integration Points
- `POST /activities` in `internal/adapters/primary/http/activity_handler.go` — extended with origin payload (D-04)
- Contract endpoints (`internal/adapters/primary/http/contract.go`) — extended with sold_hours/contract_type/sold_period (D-09)
- New: `/tickets` routes registered in `cmd/server/main.go` (Go 1.22+ pattern)
- New: `audit_logs` table + repository — shared by Phase 12 (coverage changes) and Phase 13 (direction events)
- Approval routing for proposals reuses existing routing resolution (FND-02)

</code_context>

<specifics>
## Specific Ideas

- User's three-layer model for tickets (verbatim intent): "differentiate between the state and the audit log, even comments. Comments are the description around the ticket, so they provide more information. The stage describes whatever the ticket is — not the status of the ticket. Lastly, the audit log tracks everything, every event about it, including new comments and state updates. We may need a better audit infrastructure."
- User on sold_hours (verbatim intent): "it depends on what the contract is: if it is a project total hours, if it support per hours per period, we should distinguish between the two type"

</specifics>

<deferred>
## Deferred Ideas

- **Better audit infrastructure (durability/queue)** — user: "it is not part of this phase, let's push it back for later". Options discussed: transactional audit writes, fire-and-forget like today, outbox/queue (the documented ADR-BE-012 successor). Decide in a later phase — the `audit_logs` table shape (D-05) must not preclude a durability upgrade.

</deferred>

---

*Phase: 11-Foundations — Schema + Origins + Tickets Backend*
*Context gathered: 2026-08-02*
