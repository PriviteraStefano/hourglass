# Phase 11: Foundations — Schema + Origins + Tickets Backend - Research

**Researched:** 2026-08-03
**Domain:** Go/PostgreSQL backend — additive schema evolution, hexagonal domain/services/adapters, ticket lifecycle state machine, immutable audit event stream
**Confidence:** HIGH (stack + architecture patterns — verified in codebase), MEDIUM (semantic design points — flagged in Assumptions Log / Open Questions)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Origin refs storage & validation
- **D-01:** Origin refs stored on activities as a **discriminator + nullable columns**: `origin_type VARCHAR CHECK ('manager_assignment','employee_proposal','customer_ticket')` + nullable UUID columns `assigned_by`, `assigned_to`, `proposed_by`, `reviewed_by`, `ticket_id` + a CHECK constraint matching refs to the type (e.g., customer_ticket → ticket_id set, others null). EAV ruled out by R4. No JSONB blob.
- **D-02:** **Strict same-org validation**: refs must point at existing users/tickets within the same org; employee-proposal requires `proposed_by`; manager-assignment requires `assigned_by` + `assigned_to`; customer-ticket requires `ticket_id` (an internal ticket of the org). DB FKs where possible, rejected at service level otherwise.
- **D-03:** **Service-enforced immutability** — service rejects any update changing `origin_type` or refs after creation (D-D "set once"). No DB trigger (no trigger precedent in codebase).
- **D-04:** Origin creation surfaces via **extending the existing `POST /activities` payload** (optional `origin_type` + refs) with role gates: manager_assignment → manager/org-admin; employee_proposal → any employee (proposed_by = self); customer_ticket → manager+.

#### Ticket event stream (BE-012) & comments
- **D-05:** **General `audit_logs` table** (id, org_id, entity_type, entity_id, action, actor_id, comment, payload JSONB, created_at) — tickets use it in Phase 11; Phase 12 coverage changes and Phase 13 direction events reuse it. BE-012 ADR gets extended from approval-table-only to a real table. Today the "audit trail" is `time_entry_approvals` doubling as history — that precedent moves to the new table for tickets.
- **D-06:** **Three-layer split** (user's explicit design): (1) **state** — current ticket status/fields in the `tickets` table; (2) **comments** — separate first-class storage (comments are "the description around the ticket", not audit rows); (3) **audit log** — tracks EVERYTHING, every event, including new comments and state updates. Not a dual-write of state; comments are not audit-only.

#### sold_hours semantics
- **D-07:** Semantics **depend on contract type** (user: "it depends on what the contract is: if it is a project total hours, if it support per hours per period").
- **D-08:** Add `contract_type VARCHAR CHECK ('project','support')` to contracts — NULL stays ambiguous (existing v0.1 contracts treated as project). `project` → sold_hours = lifetime total; `support` → sold_hours = hours per period.
- **D-09:** Shape: `sold_hours DECIMAL` nullable + `sold_period VARCHAR CHECK ('month','quarter','year')` nullable, required only when `contract_type = 'support'`. Read/write via existing contract endpoints (FND-03 read/write requirement). V5 mines sold vs Σ actual.

#### Triage shape & permissions
- **D-10:** **Atomic triage call** — `POST /tickets/{id}/triage` accepts kind (re)classification + 1..N activity definitions (name, parent, contract, governance); activities created in the same transaction; ticket → planned; audit rows for triaged + activities-created. No partial states.
- **D-11:** **Manager + finance may triage** (finance needed when funding eligibility is set in Phase 12); employees can create tickets but not triage.
- **D-12:** **Proposal approval** (FND-02) = activity created with `origin_type='employee_proposal'` + `is_active=false`; approval (routed via activity approval routing) flips `is_active=true`; approval record + lifecycle events land in audit_logs, never in origin refs (D-G). No new activity status column.

#### Ticket lifecycle & permissions
- **D-13:** **Dismissal guard** (TICK-04) ships on **raw Σ logged hours** (submitted + approved) across linked activities; the "dismissed with N h logged" note carries that number. Phase 12 upgrades the computation to net-of-compensations when compensation machinery lands — guard signature unchanged, computation swapped.
- **D-14:** **DB CHECK + service state machine**: status vocabulary enforced by DB CHECK (house style — governance_model, entry statuses all CHECK-enforced); transition rules (allowed edges, resolved-blocks-on-non-terminal-activities, reopen resolved→in_progress) enforced in the ticket service with audit rows per transition. No DB triggers.
- **D-15:** **Backend permission gates land now** (TICK-06 "auto-approved with permission control"): create → any employee; update/comment → owner/assignee/manager+; triage → manager+finance; view → all org members. "Auto-approved" = no approval workflow step, not no permission. Phase 18 surfaces add UI but don't loosen backend gates.

#### Legacy data
- **D-16:** **Purely additive migration, no backfill**: v0.1 has zero tickets, zero origin refs, zero sold_hours. All new columns nullable; legacy contracts get `contract_type NULL` (treated as project). Append-only per ADR-BE-004.

### the agent's Discretion
- Exact endpoint list for tickets CRUD/comments/history routes and their URL shapes (within the atomic-triage and permission decisions above)
- Audit-log read-path exposure shape (history endpoint design)
- Test layout for the new domain packages (follow existing per-package suite pattern)
- Contract-type exposure in existing contract read-models (response fields)

### Deferred Ideas (OUT OF SCOPE)
- **Better audit infrastructure (durability/queue)** — user: "it is not part of this phase, let's push it back for later". Options discussed: transactional audit writes, fire-and-forget like today, outbox/queue (the documented ADR-BE-012 successor). Decide in a later phase — the `audit_logs` table shape (D-05) must not preclude a durability upgrade.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FND-01 | Activity carries an origin (type + reference set) set once at creation: manager-assignment → assigned_by + assigned_to, employee-proposal → proposed_by + reviewed_by, customer-ticket → ticket_id (D-D) | D-01..D-04 locked; discriminated nullable columns + table CHECK (verified PostgreSQL semantics); service-enforced immutability on update path; `activities` insert path in `activity_repository.go` extended; FKs to `users(id)` / `tickets(id)` |
| FND-02 | Employee can propose an activity; proposal approval routes via the activity's approval routing (internal/personal → proposer's unit manager, contract-linked → anchored WG's manager); lifecycle events live in activity state/audit, not origin (D-G, D-D) | D-12 locked (is_active=false until approved, no new status column); the exact routing machinery exists in `internal/core/services/time_entry/time_entry.go` (`resolveManagerStage`/`resolveUnitManager`, BE-014) — extract to a shared package for reuse; audit rows land in the new `audit_logs` table |
| FND-03 | Contract carries sold_hours; V5 mines sold vs Σ actual hours (D-N) | D-07..D-09 locked; additive columns on `contracts` + CHECK; read/write through existing contract repo/handler (`contract_repository.go` dynamic SET + `baseContractQuery` + `scanContractResponse`); align `sold_hours DECIMAL` with `time_entries.hours DECIMAL(8,2)` |
| FND-04 | Origin refs stored directly on activities; when empty, the read path falls back to the first direction record — additive enhancement, no migration (R4 resolution, Part 15) | Phase 11 scope = storage only (refs written at creation). The read-path fallback is additive and lands in Phase 13; no migration needed — document in ADR-P-013 so the read model is not painted into a corner |
| TICK-01 | User can create an internal ticket with a closed-set kind (question · bug · change · evolution); kinds drive funding-eligibility rules (Q2, D-H) | New `tickets` table with `kind VARCHAR CHECK IN ('question','bug','change','evolution')` (closed set = CHECK, unlike `activity_kinds` catalog); create endpoint open to any employee (D-15); funding eligibility is Phase 12 (kinds are stored now, rules later) |
| TICK-02 | Ticket lifecycle: open → triage → planned → in_progress → resolved → closed, with reopen (resolved → in_progress); resolved requires all linked activities terminal; one demand thread per real-world request (D-A) | DB CHECK on status vocabulary + service-enforced transition matrix (D-14); "linked activities" = activities with `origin_type='customer_ticket'` and matching `ticket_id`; **"terminal activity" is undefined in the codebase** (activities have no status column — see Open Question 2) |
| TICK-03 | Triage classifies a ticket (kind + eligible funding sources) and converts it into 1..N activities; chain is ticket → activity → entries — tickets never reference entries directly (revised P-003) | D-10 atomic triage in one transaction (pgx `BeginTx` precedent: `refresh_token_repo.go` `Rotate`); triage-created activities carry `origin_type='customer_ticket'` + `ticket_id`; entries keep single `activity_id` (P-007 D-4, BE-014 R-1) — no ticket FK on entries |
| TICK-04 | Dismissal guard: triage → dismissed is blocked while any linked activity carries logged hours (net of compensating corrections); a dismissed ticket keeps a "dismissed with N h logged" note (D-M) | D-13 raw Σ (submitted + approved) this phase; exclude `is_deleted` entries; Σ query = `SELECT COALESCE(SUM(hours),0) FROM time_entries WHERE activity_id IN (activities WHERE ticket_id = $1) AND status IN ('submitted','approved') AND is_deleted = false`; note storage (column vs audit-only) — see Open Question 3 |
| TICK-05 | Ticket history is an immutable event stream (comments, resolution notes, status changes) via the BE-012 audit trail — never editable or deletable | D-05/D-06 three-layer split; general `audit_logs` table; no UPDATE/DELETE endpoints nor repo methods for audit rows or comments; **write durability is a design decision** (fire-and-forget per ADR-BE-012 risks losing the event stream — see Open Question 4) |

## Summary

Phase 11 is a **backend-only, purely additive** phase on a settled hexagonal codebase (Go 1.26.1 + pgx v5 + stdlib net/http). The three-plane ontology takes its first shape: activities gain an origin (discriminator + five nullable ref columns + a table CHECK matching refs to type), contracts gain `contract_type`/`sold_hours`/`sold_period`, and a new ticket domain (state + comments + general `audit_logs` stream) lands with lifecycle, atomic triage, dismissal guard, and permission gates. All schema changes are append-only migrations (014+ per ADR-BE-004) with up/down pairs and cycle tests — legacy rows stay valid because every new column is nullable and every CHECK is written to pass NULL discriminators (verified PostgreSQL three-valued-logic semantics).

**No new external packages are needed.** Every library required (pgx, uuid, testify, testcontainers-go) is already in `go.mod` (verified: `go mod verify` OK, 97 modules, Go 1.26.1 matches). The phase reuses established patterns: thin HTTP handlers → service invariants → pgx repositories with sentinel errors, DB CHECK vocabularies (house style), `pool.BeginTx` atomic multi-write (precedent in `refresh_token_repo.Rotate`), and per-package test suites with testcontainers.

**Primary recommendation:** plan four additive migrations (014 origin refs, 015 sold_hours, 016 tickets + comments, 017 audit_logs), a new `ticket` domain + service + postgres repo + HTTP handler pair, a general `audit_logs` repository, extension of the activity/contract domain structs and repos, and **extraction of the BE-014 manager-stage resolution** (`resolveManagerStage`/`resolveUnitManager` from `time_entry` service) into a shared package so proposal approval (FND-02) routes identically to entry approval. Write ticket audit rows **synchronously inside the same transaction** as the state change (cheap, small volume, makes TICK-05 real and keeps the triage tx atomic) while leaving the entry-side fire-and-forget behavior untouched — the deferred outbox upgrade stays possible.

**⚠️ Pre-existing red tests:** `TestMigration011_ActivityOntology_UpDownUpCycle` and `TestMigration012_StaffingSchema_UpDownUpCycle` **fail today** (verified by running them) because the MVP seed data they assert against no longer ships in `migrations/` (the historical `003_seed.up.sql` was consolidated into `scripts/seed_demo.sql`, which the test helper `applyMigrations` never loads). Success criterion 9 requires cycle tests for the new migrations, so the planner must fix the cycle-test seed wiring (or self-seed pre-state) as part of this phase.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Origin refs storage & validation (FND-01) | API/Backend (service) | Database/Storage | Service validates same-org refs + immutability (D-02/D-03); DB owns shape (CHECK + FKs, D-01) |
| Proposal approval routing (FND-02) | API/Backend (service) | Database/Storage | Routing resolution (BE-014) is service logic; approval outcome flips `is_active` + writes audit |
| sold_hours read/write (FND-03) | API/Backend (service) | Database/Storage | Field lives on `contracts`; read/write via existing contract endpoints (D-09); CHECKs in DB |
| Ticket lifecycle state machine (TICK-01/02) | API/Backend (service) | Database/Storage | Transition edges + resolved-block enforced in ticket service (D-14); status vocabulary CHECK in DB |
| Atomic triage → 1..N activities (TICK-03) | Database/Storage (transaction) | API/Backend | Single tx commits ticket status + activities + audit rows (D-10); service drives it |
| Dismissal guard (TICK-04) | API/Backend (service) | Database/Storage | Service computes Σ logged hours via repo query before allowing dismissal (D-13) |
| Immutable event stream (TICK-05) | Database/Storage (append-only) | API/Backend | No update/delete endpoints or repo methods; audit rows written in-transaction |
| Comments (D-06) | API/Backend | Database/Storage | First-class `ticket_comments` storage; creation also audit-logged |
| ADRs (P-003 rev, P-013, BE encoding) | Docs (vault) | — | Decision records in `hourglass-vault/decisions/` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go (stdlib `net/http`) | 1.26.1 | Server, routing (`mux.HandleFunc("POST /tickets/{id}/triage", ...)`), JSON | Verified in `go.mod` + `cmd/server/main.go`; Go 1.22+ method patterns already used everywhere |
| jackc/pgx v5 | v5.10.0 | PostgreSQL access, `pgxpool`, transactions | Verified in `go.mod` + all repos; `BeginTx` precedent in `refresh_token_repo.go` |
| github.com/google/uuid | v1.6.0 | UUIDs for entities | Verified in `go.mod` + all domains; house style (`uuid.UUID` PKs) |
| stretchr/testify | v1.11.1 | assert/require in all tests | Verified in `go.mod`; used by every test suite |
| testcontainers-go (postgres module) | v0.42.0 | Integration-test DB (`postgres:16-alpine`) | Verified in `go.mod` + `test_setup.go` (`SetupPackageContainer`) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cmd/migrate` | — | Apply `migrations/*.up.sql` / `*.down.sql` | All schema changes (ADR-BE-004); cycle tests call `applyMigrations` |
| `pkg/api` | — | `{ data \| error }` envelope | Every handler response/error |
| `internal/middleware` | — | `Auth`, `GetRole`, `GetOrganizationID`, `GetUserID` | Role/ownership gates (D-04, D-11, D-15) read claims from context |
| `scripts/seed_demo.sql` | — | MVP demo data | Dev/demo environments (`cmd/migrate -all`, Makefile); **not** loaded by test helpers — see Pitfall 5 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Discriminator + columns for origin refs (D-01) | EAV table / JSONB blob | Locked by R4 + D-01 — rejected; no FK enforcement, joins, or CHECKs |
| New activity status column for proposals (D-12) | `is_active=false` + audit | Locked — no schema vocabulary churn; audit carries lifecycle |
| Dedicated ticket_events table (D-05) | General `audit_logs` table | Locked — general table is reused by Phase 12 (coverage) and 13 (direction) |
| Outbox/queue for audit writes | Synchronous in-transaction writes | Deferred by user; table shape must not preclude the upgrade |

**Installation:**
```bash
# No new packages. Zero dependency changes this phase — verified via `go mod verify` (OK).
```

**Version verification:** All libraries above verified present and current in the repo's `go.mod` (Go 1.26.1, pgx v5.10.0, uuid v1.6.0, testify v1.11.1, testcontainers-go v0.42.0). No registry lookup needed — this phase adds no dependencies.

## Package Legitimacy Audit

> No new external packages are installed by this phase. The phase runs entirely on the existing dependency set (`go mod verify` — "all modules verified", 97 modules). The package-legitimacy seam supports npm/pypi/crates only (Go unsupported); equivalent verification performed via `go.mod` + `go.sum` + `go mod verify`.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| jackc/pgx/v5 v5.10.0 | Go modules | long-standing | high | github.com/jackc/pgx | OK | Approved (already in go.mod) |
| google/uuid v1.6.0 | Go modules | long-standing | high | github.com/google/uuid | OK | Approved (already in go.mod) |
| stretchr/testify v1.11.1 | Go modules | long-standing | high | github.com/stretchr/testify | OK | Approved (already in go.mod) |
| testcontainers-go v0.42.0 | Go modules | long-standing | high | github.com/testcontainers/testcontainers-go | OK | Approved (already in go.mod) |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
**No `[ASSUMED]` package names:** all names verified in the repo's committed `go.mod`/`go.sum` — no external discovery involved.

## Architecture Patterns

### System Architecture Diagram

```mermaid
flowchart LR
    subgraph API["HTTP layer (thin handlers, pkg/api envelope)"]
        CA["POST /activities<br/>(+ origin payload)"]
        TR["POST /tickets<br/>POST /tickets/{id}/triage<br/>POST /tickets/{id}/comments<br/>POST /tickets/{id}/transition<br/>GET /tickets/{id}/history"]
        CU["POST/PUT /contracts<br/>(+ sold_hours)"]
        PA["POST /activities/{id}/approve-proposal"]
    end
    subgraph SVC["Services (invariants, gates)"]
        AS["activity.Service<br/>origin validation + immutability"]
        TS["ticket.Service<br/>state machine + triage + dismissal guard"]
        CS["contract.Service<br/>sold_hours validation"]
        RS["routing (extracted BE-014)<br/>manager-stage resolution"]
    end
    subgraph DB["PostgreSQL (pgx repos)"]
        AR["activities<br/>+ origin_type + 5 ref columns + CHECK + FKs"]
        KR["contracts<br/>+ contract_type/sold_hours/sold_period + CHECK"]
        TRB["tickets + ticket_comments"]
        AUL["audit_logs<br/>(entity_type/entity_id/action/payload JSONB)"]
        TER["time_entries (read-only: Σ hours)"]
    end
    CA --> AS --> RS
    PA --> AS --> RS
    AS -->|"tx on triage only"| AR
    TR --> TS
    TS -->|"atomic triage tx"| TRB
    TS -->|"1..N activity inserts (origin customer_ticket)"| AR
    TS -->|"audit rows in same tx"| AUL
    TS -->|"dismissal guard Σ"| TER
    CU --> CS --> KR
    AS -->|"audit rows (proposal lifecycle)"| AUL
```

**Read path (FND-04):** activity reads expose stored origin refs directly; the first-direction-record fallback is additive and lands in Phase 13 — do not build it now.

### Recommended Project Structure
```
internal/core/domain/ticket/            # NEW: ticket entity, kinds, statuses, transition matrix, errors.go
internal/core/domain/audit/             # NEW: general audit-log entity (entity_type/entity_id/action/payload)
internal/core/domain/activity/          # EXTEND: origin fields on Activity + CreateActivityRequest/UpdateActivityRequest
internal/core/domain/contract/          # EXTEND: contract_type/sold_hours/sold_period fields
internal/core/ports/                    # EXTEND: ticket_repository.go, audit_log_repository.go, activity origin refs
internal/core/services/ticket/          # NEW: lifecycle, triage, comments, permissions, dismissal guard
internal/core/services/routing/         # NEW: EXTRACT resolveManagerStage/resolveUnitManager from time_entry svc (BE-014)
internal/adapters/primary/http/         # EXTEND: ticket_handler.go (NEW), activity_handler.go, contract.go
internal/adapters/secondary/postgres/   # EXTEND: ticket_repository.go, audit_log_repository.go (NEW), activity/contract repos
migrations/                             # NEW: 014..017 up/down pairs (see below)
hourglass-vault/decisions/              # ADR-P-003 revision, ADR-P-013, BE encoding ADR (+ _index.md updates)
internal/adapters/secondary/postgres/exported_test_helpers.go  # EXTEND: teardown table list (tickets, ticket_comments, audit_logs)
```

**Suggested migration split (final numbering per A8 — tickets as 014 so the origins FK resolves):**
- `014_ticket_schema.{up,down}.sql` — `tickets` + `ticket_comments` (kind/status CHECKs, FKs, indexes)
- `015_activity_origins.{up,down}.sql` — origin columns on `activities`, refs-to-type CHECK, FKs to `users(id)`/`tickets(id)`, index on `ticket_id`
- `016_contract_sold_hours.{up,down}.sql` — `contract_type`/`sold_hours`/`sold_period` + type-consistency CHECK
- `017_audit_logs.{up,down}.sql` — general `audit_logs` table + `(entity_type, entity_id, created_at)` index
Each with a cycle test (up → down → up) in the postgres package, mirroring `TestMigration012_StaffingSchema_UpDownUpCycle`.

### Pattern 1: Discriminator + nullable ref columns with a table CHECK (origin refs)
**What:** one discriminator column + N nullable UUID columns; a table-level CHECK maps each discriminator value to exactly the refs it may carry and forces the others to NULL. Because CHECK is satisfied when the expression evaluates to TRUE **or NULL** (three-valued logic), the constraint must be written as `origin_type IS NULL OR (<per-type rules>)` so legacy rows (NULL discriminator) pass.
**When to use:** any "one of several reference shapes" fact fixed at creation.
**Example (shape for 015):**
```sql
ALTER TABLE activities ADD COLUMN origin_type VARCHAR(50);
ALTER TABLE activities ADD COLUMN assigned_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN assigned_to  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN proposed_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN reviewed_by  UUID REFERENCES users(id);
ALTER TABLE activities ADD COLUMN ticket_id    UUID REFERENCES tickets(id);
ALTER TABLE activities ADD CONSTRAINT activities_origin_type_check
  CHECK (origin_type IN ('manager_assignment','employee_proposal','customer_ticket'));
ALTER TABLE activities ADD CONSTRAINT activities_origin_refs_check
  CHECK (
    origin_type IS NULL
    OR (origin_type = 'manager_assignment' AND assigned_by IS NOT NULL AND assigned_to IS NOT NULL
        AND proposed_by IS NULL AND reviewed_by IS NULL AND ticket_id IS NULL)
    OR (origin_type = 'employee_proposal' AND proposed_by IS NOT NULL
        AND assigned_by IS NULL AND assigned_to IS NULL AND ticket_id IS NULL)
    OR (origin_type = 'customer_ticket' AND ticket_id IS NOT NULL
        AND assigned_by IS NULL AND assigned_to IS NULL
        AND proposed_by IS NULL AND reviewed_by IS NULL)
  );
CREATE INDEX idx_activities_ticket_id ON activities(ticket_id);  -- dismissal guard + resolved check
```
Note: `reviewed_by` deliberately NOT required by the CHECK (D-02 requires only `proposed_by`) — see Open Question 1 for its semantics. FKs to `users(id)` are possible because `users` is global; **same-org validation of refs happens at the service level** (membership check), per D-02. FK to `tickets(id)` requires migration 014 to exist before 015 — resolved by numbering tickets as 014 and origins as 015 (A8). **Planner note:** ordering 014 (tickets) before 015 (origins) avoids a chicken-and-egg FK problem.

### Pattern 2: DB CHECK vocabulary + service state machine (ticket lifecycle)
**What:** DB CHECK pins the status/kind vocabulary (house style — same as `governance_model`, entry statuses); the service enforces transition edges, the resolved-blocks-on-non-terminal-activities rule, and the reopen edge, writing one audit row per transition (D-14).
**When to use:** state machines with a closed vocabulary and edge rules.
**Example (recommended matrix from Part 4 + D-A; planner confirms exact edges):**
```
open → triage → planned → in_progress → resolved → closed
            ↘ dismissed (guard: Σ logged hours == 0)
resolved → in_progress (reopen, allowed when linked activities terminal again? or unconditional)
```
Status CHECK: `CHECK (status IN ('open','triage','planned','in_progress','resolved','closed','dismissed'))`; kind CHECK: `CHECK (kind IN ('question','bug','change','evolution'))`.

### Pattern 3: Atomic multi-write via `pool.BeginTx` (triage)
**What:** one repository method opens a transaction, performs all writes, commits; rollback on any error. Mirrors `RefreshTokenRepository.Rotate` (`refresh_token_repo.go` line 88) — the codebase's only existing transaction.
**When to use:** any operation that must commit together or not at all (D-10 triage: ticket status + 1..N activities + audit rows).
**Example:** see Code Examples section (verified pgx pattern + codebase precedent).

### Pattern 4: Three-layer ticket model (state / comments / audit)
**What:** `tickets` holds current state only; `ticket_comments` holds first-class comments (the "description around the ticket"); `audit_logs` records every event — status changes, transitions, comments, resolution notes (D-06). Comments are created via their own endpoint, and each comment ALSO writes an audit row. No endpoint updates or deletes any of the three (TICK-05).
**When to use:** domains needing a tamper-evident history plus user-facing discussion.

### Pattern 5: Shared approval-routing extraction (proposal approval mirrors entry routing)
**What:** `resolveManagerStage`/`resolveUnitManager` (time_entry service, BE-014 R-1/R-2) move to a shared package (e.g., `internal/core/services/routing`); the activity service reuses it for proposal approval (D-G/FND-02). For proposals, the unit context = the proposer's primary unit (`unit_memberships.is_primary`) — the proposal has no `unit_id` like entries do.
**When to use:** one routing rule for entries and proposals (D-G) — duplication would let the rules drift.
**Note:** `resolveUnitManager` walks `unit_memberships.role = 'manager'` upward; terminal case (org root without manager) is role-gated (`roleGated`). Extraction must preserve D-11 skip semantics and `ErrActivityNotLoggable` (commercial activity without anchored WG).

### Anti-Patterns to Avoid
- **EAV or JSONB blob for origin refs:** locked out by R4 + D-01 — no FKs, no CHECKs, joins and immutability get harder.
- **DB triggers for invariants:** no trigger precedent in the codebase; D-03/D-14 explicitly say service-enforced. (The one existing trigger — `auto_create_org_settings` — is a convenience, not an invariant.)
- **Mutating origin refs post-creation:** D-03 — the update path must reject changes to `origin_type` + refs with a sentinel error; do not "helpfully" allow filling `reviewed_by` later.
- **Dual-write state + audit as the only record:** D-06 — comments are first-class; audit is not a second state copy.
- **Outbox/queue in this phase:** deferred — but keep the `audit_logs` shape compatible (a future outbox consumes the same rows; do not add destructive/update paths now).
- **New activity status column:** D-12 — proposals are `is_active=false`, lifecycle lives in audit.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Transaction management (atomic triage) | Hand-rolled commit/rollback scaffolding | pgx `pool.BeginTx` + `defer tx.Rollback` (precedent: `refresh_token_repo.go`) | pgx handles isolation, deferred rollback safety, context propagation |
| Test database | ad-hoc SQL fixtures per package | testcontainers `SetupPackageContainer` (postgres:16-alpine) + `SetupTestSchema` | Exists; one container per package via `sync.Once` |
| Migration runner | Custom DDL tooling | `cmd/migrate` + `applyMigrations` test helper | ADR-BE-004; up/down/up cycle tests established |
| JSON response envelope | Ad-hoc marshaling | `pkg/api.RespondWithJSON/RespondWithError` | House standard, verified everywhere |
| Authentication | Token parsing in handlers | `middleware.Auth` + claims context (`GetRole`, `GetUserID`, `GetOrganizationID`) | Exists; all route gates read from context |
| Manager-stage routing for proposals | Re-implement BE-014 precedence | Extract `resolveManagerStage`/`resolveUnitManager` to a shared package | One routing rule (D-G); prevents drift between entries and proposals |
| Sentinel error mapping | Raw pg errors in handlers | Domain `errors.go` sentinels + `wrapPGError` + handler switch (ADR-BE-001) | Established across all handlers |

**Key insight:** this phase's complexity is almost entirely in *existing* patterns — schema CHECKs, service state machines, one transaction. Nothing here needs a new library or a novel mechanism; the risk is in semantic definition (terminal activity, reviewed_by, dismissal note) and in the broken cycle-test seed wiring, not in technology.

## Runtime State Inventory

Not applicable — this is not a rename/refactor phase. All changes are additive (new tables, new nullable columns, new endpoints). No existing string keys, service names, secrets, or build artifacts change. (The only pre-existing runtime concern — demo Postgres 28P01 quick task `260801-o06-failed-to-initialize-postgresql-pool-pas` — is triaged during Phase 11/14 per STATE.md, unrelated to schema work.)

## Common Pitfalls

### Pitfall 1: CHECK constraints silently passing NULLs (breaks legacy rows or lets mixed refs through)
**What goes wrong:** a naive `CHECK (origin_type = 'customer_ticket' AND ticket_id IS NOT NULL)` is satisfied when `origin_type IS NULL` (expression → NULL → CHECK passes). Two failure modes: (a) per-column CHECKs like `CHECK (ticket_id IS NOT NULL)` would reject every legacy activity row; (b) a type CHECK that doesn't pin *other* refs to NULL allows `origin_type='customer_ticket'` with `assigned_by` also set.
**Why it happens:** PostgreSQL three-valued logic — CHECK passes on TRUE or NULL.
**How to avoid:** shape every CHECK as `discriminator IS NULL OR (<per-type rules using explicit IS NULL / IS NOT NULL>)`, and pin the *entire* ref set per type (see Pattern 1).
**Warning signs:** migration cycle test inserting a legacy-shaped row (all NULL) failing; insert with mixed refs succeeding.

### Pitfall 2: Fire-and-forget audit writes silently losing ticket events (TICK-05 broken)
**What goes wrong:** ADR-BE-012's detached-context goroutine pattern can drop rows on DB error — acceptable for entry approvals (regenerable), but the ticket event stream IS the ticket's history (TICK-05: "never editable or deletable" — and it must not be *absent* either).
**Why it happens:** copying the existing audit pattern without noticing the different guarantee.
**How to avoid:** write ticket audit rows synchronously in the same transaction as the state change (also required by D-10's atomic triage audit rows). Entry-side fire-and-forget stays untouched. The deferred durability upgrade (outbox) remains possible later.
**Warning signs:** `go s.auditRepo.Create(ctx...)` goroutine with `context.Background()` inside ticket transitions; audit row count ≠ transition count in tests.

### Pitfall 3: Pre-existing red migration cycle tests (verified) — new cycle tests inherit broken seed wiring
**What goes wrong:** `TestMigration011_ActivityOntology_UpDownUpCycle` and `TestMigration012_StaffingSchema_UpDownUpCycle` FAIL today (verified: run both — they expect 6 memberships / 6 projects / 6 subprojects / 12 time entries that no longer exist). Root cause: historical `003_seed.up.sql` was consolidated into `scripts/seed_demo.sql`, which `applyMigrations` never loads.
**Why it happens:** seed data moved out of `migrations/`; cycle tests weren't updated.
**How to avoid:** fix the cycle-test seeding strategy FIRST (restore a seed-loading helper that runs `scripts/seed_demo.sql`, or build minimal pre-state with existing `seedOrg`/`seedUser`/`seedActivity`/`seedActivityKind` helpers), then write the 014–017 cycle tests against the working pattern. Without this, the new cycle tests will fail the same way.
**Warning signs:** `assertCount(..., 6)` on memberships returning 0; new cycle tests red for the same reason.

### Pitfall 4: reviewed_by semantics vs immutability (employee proposals)
**What goes wrong:** D-D lists `reviewed_by` (the approving manager) as a creation-time ref, but the approver is resolved by routing at approval time — writing it post-creation violates D-03 immutability.
**Why it happens:** two decisions (refs set once; approver = routing output) interact.
**How to avoid:** pin the semantics in the plan — recommended: resolve the approver via the shared routing package **at proposal creation** and store it as `reviewed_by` (it is a creation-time expectation; the *actual* approver is recorded in the audit row at approval). Simpler alternative: leave `reviewed_by` NULL and let the audit row carry the approver. Either way, the CHECK requires only `proposed_by` (D-02).
**Warning signs:** an update path that sets `reviewed_by` post-creation; a CHECK that requires `reviewed_by` non-NULL.

### Pitfall 5: "Terminal activity" is undefined — the resolved transition can't be checked
**What goes wrong:** TICK-02/D-14 require `resolved` to block while any linked activity is non-terminal, but activities have **no status column** (only `is_active`, which flips for proposals — it is not a work-progress status).
**Why it happens:** the activity ontology has no completion state; the ticket lifecycle needs one.
**How to avoid:** define "terminal" in the plan (needs user confirmation — Open Question 2). Recommended: an activity is terminal when it has **no time entries in draft/submitted/pending_manager/pending_finance** on its subtree (`activity_id IN (<activity subtree>)`), excluding deleted; `is_active=false` alone is NOT a work-progress signal.
**Warning signs:** resolved transition check comparing `is_active`; no query exists for entry status per activity.

### Pitfall 6: Dismissal guard counting the wrong hours
**What goes wrong:** Σ includes draft entries, deleted entries, or entries on the wrong activities (only direct ticket-linked vs subtree).
**Why it happens:** D-13 says "submitted + approved" across linked activities — the statuses and the linkage (`origin_type='customer_ticket'` + `ticket_id`) must be exact, and `is_deleted` must be excluded.
**How to avoid:** `SELECT COALESCE(SUM(hours),0) FROM time_entries WHERE is_deleted = false AND status IN ('submitted','approved') AND activity_id IN (SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket')`. Phase 12 swaps the sum expression for net-of-compensations — keep the guard signature (a `LoggedHours(ctx, ticketID) (float64, error)` port method) stable.
**Warning signs:** guard trips on draft entries; dismissal note shows hours from deleted entries.

### Pitfall 7: Triage transaction skipping activity validation
**What goes wrong:** activities created inside the triage tx bypass the same checks `POST /activities` enforces (kind in org catalog, parent same-org, contract exists, origin CHECK).
**Why it happens:** the ticket repo inserts activities directly; validation lives in the activity service.
**How to avoid:** the triage method must run the same validations inside the tx (reuse activity repo queries within the tx, or have the ticket repo accept a validated plan from the service). Do not split validation out of the tx.
**Warning signs:** triage accepting an unknown kind or cross-org parent.

### Pitfall 8: Teardown list and migration number not updated
**What goes wrong:** `TeardownTestSchema` (exported_test_helpers.go) lacks `tickets`, `ticket_comments`, `audit_logs` — cross-package test pollution; new migrations numbered wrong (next is 014, not a renumber).
**Why it happens:** adding tables without touching the shared teardown; forgetting ADR-BE-004 sequential numbering (007 gap is frozen).
**How to avoid:** extend the teardown table list (drop audit_logs/ticket_comments/tickets before activities); number new migrations 014+.
**Warning signs:** repository tests in a later package failing on leftover ticket rows.

## Code Examples

### Common Operation 1: Atomic multi-write transaction (triage) — pgx verified pattern
```go
// Source: official jackc/pgx autodocs (BeginTx) + codebase precedent
// (internal/adapters/secondary/postgres/refresh_token_repo.go, Rotate)
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil {
    return nil, fmt.Errorf("begin ticket triage: %w", err)
}
defer func() { _ = tx.Rollback(ctx) }() // safe even after Commit

// 1. update ticket → planned
// 2. insert 1..N activities with origin_type='customer_ticket', ticket_id=$ticket
// 3. insert audit rows ('triaged', 'activities-created') into audit_logs
if err := tx.Commit(ctx); err != nil {
    return nil, fmt.Errorf("commit ticket triage: %w", err)
}
```

### Common Operation 2: Multi-column CHECK (origin refs / sold_hours)
```sql
-- Source: PostgreSQL docs (ddl-constraints): table CHECKs may reference multiple
-- columns; satisfied when expression is TRUE or NULL → guard with `discriminator IS NULL OR ...`.
ALTER TABLE contracts ADD COLUMN contract_type VARCHAR(50);
ALTER TABLE contracts ADD COLUMN sold_hours DECIMAL(10,2);
ALTER TABLE contracts ADD COLUMN sold_period VARCHAR(10);
ALTER TABLE contracts ADD CONSTRAINT contracts_sold_check
  CHECK (
    contract_type IS NULL
    OR (contract_type = 'support' AND sold_hours IS NOT NULL AND sold_period IS NOT NULL)
    OR (contract_type = 'project'  AND sold_period IS NULL)
  );
```

### Common Operation 3: Manager-stage routing to reuse for proposal approval
```go
// Source: internal/core/services/time_entry/time_entry.go (resolveManagerStage) —
// extract verbatim to a shared package for FND-02 proposal approval.
// R-1: anchored WG → WG manager + delegates; R-2: commercial + no WG →
// ErrActivityNotLoggable; R-2 fallback: personal → unit manager (upward walk);
// terminal: roleGated. For proposals, unitID = proposer's primary unit.
res, err := routing.ResolveManagerStage(ctx, orgID, activityID, unitID, ownerID)
if res.skipToFinance { /* D-11: approver == owner */ }
```

### Common Operation 4: Migration up/down pair skeleton
```sql
-- 014_ticket_schema.up.sql — pattern source: 012_staffing_schema
CREATE TABLE tickets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    kind          VARCHAR(50) NOT NULL CHECK (kind IN ('question','bug','change','evolution')),
    status        VARCHAR(50) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','triage','planned','in_progress','resolved','closed','dismissed')),
    requester_id  UUID NOT NULL REFERENCES users(id),
    assignee_id   UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tickets_org_id ON tickets(org_id);
CREATE INDEX idx_tickets_status ON tickets(status);
```
```sql
-- 014_ticket_schema.down.sql
DROP TABLE IF EXISTS tickets CASCADE;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Projects + subprojects (two levels) | Recursive `activities` (P-007 D-1) | v0.1 close (migration 011) | Phase 11 extends this table; origin refs ride on it |
| Audit trail = `time_entry_approvals` doubling as history | General `audit_logs` table (D-05) | This phase | Tickets, Phase 12 coverage, Phase 13 direction events share one stream |
| Fire-and-forget audit writes (ADR-BE-012) | Synchronous in-transaction writes for tickets (recommended) | This phase | TICK-05 guarantee real; outbox remains the documented successor |
| EAV/JSONB for polymorphic refs | Discriminator + nullable columns + CHECK | This phase (R4 resolution) | FK-enforceable, legacy-safe, no migration of existing rows |
| Ticket = `open → in_progress → resolved/dismissed` sketch (P-003) | Full lifecycle with triage, reopen, dismissal guard (D-A, D-M) | This phase (P-003 revision) | Demand tracking + coverage hook |

**Deprecated/outdated:**
- `time_entry_approvals` as the only audit precedent: still used for entries, but no longer the model for new domains.
- `003_seed.up.sql`: historical seed migration — consolidated into `scripts/seed_demo.sql`; migration cycle tests were not updated (Pitfall 3).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `reviewed_by` (employee proposal) is resolved at creation via shared routing; CHECK requires only `proposed_by` | Pattern 1 / Pitfall 4 | Contradicts D-03 if later filled post-creation; or CHECK too strict if user wants it required |
| A2 | "Terminal activity" = no non-terminal time entries (draft/submitted/pending) on the activity subtree, `is_deleted` excluded | Pitfall 5 / Open Q2 | Resolved transition allows/block wrongly — core TICK-02 behavior |
| A3 | Ticket audit rows are written synchronously in-transaction (not fire-and-forget) | Pitfall 2 / Open Q4 | User may prefer ADR-BE-012 parity; failure modes differ (in-tx failure blocks the op) |
| A4 | Dismissal note stored as a nullable column on `tickets` (e.g., `dismissal_note`/`dismissed_with_hours`) + audit row | Open Q3 | If audit-only, read path needs a join — "dismissed tickets carry the note" implies direct read |
| A5 | Ticket minimal field set: org_id, title, description, kind, status, requester_id, assignee_id (no priority in v0.2 decisions) | Pattern 4 | If priority is needed for P-004 Today ordering (Phase 18), it is a later additive migration — not fatal |
| A6 | Pre-triage fast path allowed: `POST /activities` with `origin_type='customer_ticket'` while the ticket is open (manager+ gated) | Summary / Open Q5 | If disallowed, urgent-work-before-triage workflow breaks; if allowed, dismissal guard must count these activities (it does, via ticket_id) |
| A7 | Transition matrix exactly: open→triage→planned→in_progress→resolved→closed; triage→dismissed (guarded); resolved→in_progress (reopen); no other edges | Pattern 2 / Open Q6 | Missing edge (e.g., open→in_progress for fast path) would be added later; extra edge weakens the guarantee |
| A8 | Migration split 014–017 with 014 (tickets) before 015 (origins FK to tickets) | Pattern 1 / project structure | Cosmetic; consolidation is planner's call, but the tickets-before-origins ordering avoids FK issues |
| A9 | `resolveManagerStage` extraction is required for FND-02 (proposal approval routes identically) | Pattern 5 | If not extracted, routing logic is duplicated — drift risk, and D-G parity is weaker |
| A10 | "manager+" = org roles `manager` OR `finance` (models.RoleManager / RoleFinance) | Patterns (gates) | If 'hr' (added in 012 to membership CHECK) should also pass gates — JWT role claims govern; confirm at plan time |
| A11 | Audit-log history endpoint (GET /tickets/{id}/history) returns rows for `entity_type='ticket'`, ordered by created_at | Discretion area | Shape is the agent's discretion; nothing locks it, but Phase 12/13 consumers will reuse the same read path |
| A12 | Triage-created activities inherit `is_active=true` (only proposals start inactive per D-12) | Pattern 3 | If triage activities should start inactive pending coverage (Phase 12 semantics), a follow-up migration changes the default |

## Open Questions (RESOLVED)

> All questions below are resolved by the plan set (11-01..11-06), ADR-P-013, and ADR-BE-016. Each entry records the resolution and where it is pinned.

1. **`reviewed_by` semantics for employee proposals (A1) — RESOLVED**
   - What we know: D-D lists it as an origin ref; D-02 requires only `proposed_by`; D-03 forbids post-creation ref mutation; D-G routes approval via BE-014 machinery.
   - What's unclear: is the reviewer pinned at creation (routing resolution) or left NULL with the approver living only in audit?
   - Resolution: Phase 11 leaves `reviewed_by` NULL at creation — the CHECK requires only `proposed_by` (D-02) — and the actual approver is captured in the `proposal_approved` audit row. Any create carrying a non-nil `reviewed_by` → ErrInvalidRequest. Pinned in ADR-P-013; enforced in 11-05 Task 2; a future phase may pin it at creation.

2. **Definition of "terminal activity" for the resolved transition (TICK-02) — RESOLVED**
   - What we know: activities have `is_active` only (proposal flag, not progress); entries have statuses (draft…approved/rejected).
   - What's unclear: what makes a linked activity "terminal" — no non-terminal entries? `is_active=false`? Both?
   - Resolution: no non-terminal time entries (status IN draft/submitted/pending_manager/pending_finance, `is_deleted=false`) on the linked-activity subtree, computed via recursive CTE (`HasNonTerminalActivities`). Recorded in ADR-BE-016; implemented in 11-06 Task 2.

3. **Where the "dismissed with N h logged" note lives (TICK-04) — RESOLVED**
   - What we know: the note must be readable from a dismissed ticket; D-13 defines N as raw Σ (submitted+approved) this phase.
   - What's unclear: stored column vs audit-row-only.
   - Resolution: nullable `dismissed_hours` column set at dismissal (migration 014, 11-01 Task 2) + the audit row records the same value; guard method `LoggedHours(ctx, ticketID)` stable for the Phase 12 computation swap. Recorded in ADR-BE-016; implemented in 11-06 Task 2.

4. **Audit write durability for tickets (TICK-05 vs ADR-BE-012) — RESOLVED**
   - What we know: user deferred the durability upgrade; D-10 requires audit rows in the triage tx; TICK-05 requires the stream to exist.
   - What's unclear: fire-and-forget (ADR-BE-012 parity) vs synchronous in-transaction for ticket events.
   - Resolution: synchronous in-transaction writes for ticket events (Pitfall 2; fire-and-forget stays only for entry approvals). The user-deferred choice is honored by keeping the table shape consumer-neutral so the outbox successor remains possible — both recorded in ADR-BE-016 (11-02 Task 3); implemented in 11-05/11-06.

5. **Pre-triage fast path (urgent work before triage) — RESOLVED**
   - What we know: research Part 4 describes urgent work starting before triage; D-01 requires ticket_id at creation for customer_ticket origin; refs are immutable.
   - What's unclear: is `POST /activities` with `origin_type='customer_ticket'` + `ticket_id` allowed while the ticket is open (pre-triage)? The dismissal guard and resolved check both read linked activities via ticket_id, so it composes.
   - Resolution: allowed — `POST /activities` with origin customer_ticket while the ticket is open/triage, manager+|finance gated (D-04), with the ticket's state open/triage. Recorded in ADR-BE-016; implemented in 11-05 Task 2.

6. **Exact transition matrix confirmation (D-14) — RESOLVED**
   - What we know: open→triage→planned→in_progress→resolved→closed + reopen + guarded dismissal per D-A/Part 4.
   - What's unclear: whether reopen (resolved→in_progress) re-requires terminal activities (no — activities already terminal) and whether closed is a true terminal state.
   - Resolution: pinned edges open→triage; triage→planned; triage→dismissed; planned→in_progress; in_progress→resolved; resolved→closed; resolved→in_progress (reopen); open→dismissed (superset of A7). closed and dismissed are terminal; reopen only from resolved; the service rejects any other edge with ErrInvalidTransition (audit row only on success). Enforced via domain `CanTransition` (11-05 Task 1); recorded in ADR-BE-016; implemented in 11-06 Tasks 1-2.

7. **Pre-existing red migration cycle tests (Pitfall 3) — RESOLVED (superseded)**
   - What we know: both TestMigration011 and TestMigration012 fail today (verified by running); seed data moved to `scripts/seed_demo.sql`.
   - What's unclear: whether to fix by loading `scripts/seed_demo.sql` in tests or by self-seeding minimal pre-state.
   - Resolution: supersedes the original recommendation — SELF-SEED minimal pre-state inline in each test (demo data is not a migration fixture; `003_seed.up.sql` stays retired). Executed in 11-01 Task 1; assertions keep their original expected values.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend build/tests | ✓ | 1.26.1 (matches go.mod) | — |
| Docker | testcontainers integration tests (postgres:16-alpine) | ✓ | 29.4.0 (orbstack) | Tests skip when Docker absent (t.Skip in `SetupPackageContainer`) |
| PostgreSQL | Dev runtime (docker-compose service); tests use containers | ✓ (docker-compose) | postgres:15 via compose / 16-alpine in tests | Testcontainers covers tests; dev DB via `docker-compose up` |
| Node/Bun | Not needed this phase (backend-only) | ✓ | node v22.23.1 | — |
| `DATABASE_URL` | `go run ./cmd/server`, `cmd/migrate` | ✓ | default `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` | Set env var explicitly |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — all tooling present.

## Validation Architecture

> `workflow.nyquist_validation` absent from `.planning/config.json` → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go 1.26.1 `go test` + testify v1.11.1 + testcontainers-go v0.42.0 (postgres:16-alpine) |
| Config file | none — scaffolding is built in-plan (see 11-VALIDATION.md); patterns from `.planning/codebase/TESTING.md` |
| Quick run command | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -count=1` |
| Full suite command | `make test` (`go test -v ./...`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FND-01 | Origin refs set at creation, immutability on update, CHECK rejects mixed refs | unit + integration | `go test ./internal/core/services/activity/ ./internal/adapters/secondary/postgres/ -run TestActivityOrigin -count=1` | ✅ in-plan (11-05-02) |
| FND-02 | Proposal creation (is_active=false, proposed_by=self) + approval routing flips is_active + audit row | unit + integration | `go test ./internal/core/services/activity/ -run 'TestProposal\|TestActivityOrigin' -count=1` | ✅ in-plan (11-05-03) |
| FND-03 | sold_hours/contract_type/sold_period read+write; support requires period; project forbids period | unit + integration | `go test ./internal/adapters/secondary/postgres/ -run TestContract -count=1 && go test ./internal/core/services/contract/ -count=1` | ✅ in-plan (11-04-02) |
| FND-04 | Origin refs stored; read returns stored refs (fallback is Phase 13 — no test here) | integration | covered by FND-01 read assertions | ✅ in-plan (11-05-02) |
| TICK-01 | Ticket create (any employee), kind closed set, kind CHECK | unit + integration | `go test ./internal/core/services/ticket/ -run 'TestTicketCreate\|TestTicketLifecycle' -count=1` | ✅ in-plan (11-06-01) |
| TICK-02 | Full lifecycle incl. reopen; resolved blocked on non-terminal linked activities | unit | `go test ./internal/core/services/ticket/ -run 'TestTicketCreate\|TestTicketLifecycle' -count=1` | ✅ in-plan (11-06-01) |
| TICK-03 | Atomic triage: ticket→planned, activities created with origin customer_ticket, all-or-nothing (in-tx validation, Pitfall 7) | integration | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -run 'TestDismissalGuard\|TestTicketTriage\|TestTicketAudit' -count=1` | ✅ in-plan (11-06-02) |
| TICK-04 | Dismissal guard: blocked with logged hours (submitted+approved, not deleted), note carries N | unit + integration | `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -run 'TestDismissalGuard\|TestTicketTriage\|TestTicketAudit' -count=1` | ✅ in-plan (11-06-02) |
| TICK-05 | History append-only: comments/transitions logged; no update/delete endpoints exist | integration + API contract | `go test ./internal/adapters/primary/http/ -run TestTicket -count=1` (history stream via 11-06-02 command) | ✅ in-plan (11-06-03) |
| SC-9 | Migration up/down pairs + cycle tests for 014–017 | migration cycle | `go test ./internal/adapters/secondary/postgres/ -run TestMigration014 -count=1` (…015/016/017) | ✅ in-plan (11-01-03) |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/services/<affected>/ ./internal/adapters/secondary/postgres/ -count=1`
- **Per wave merge:** `make test` (full suite — wave gate only, after the wave completes; mid-wave full-suite false-negatives on the pre-existing red 011/012 tests are fixed by 11-01 Task 1 before the gate)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Test Scaffolding (in-plan — no Wave 0)
This plan set has NO separate Wave 0 — every artifact is created by the plan task that runs its own verify (see 11-VALIDATION.md scaffold map):
- `internal/core/domain/ticket/` + `internal/core/domain/audit/` → 11-05 Task 1
- `internal/core/services/ticket/ticket_test.go` + `ticket_integration_test.go` → 11-06 Tasks 1-2
- `internal/adapters/secondary/postgres/ticket_repository_test.go` + `audit_log_repository.go` (no separate audit test file — covered by ticket repo tests) → 11-05 Task 1 / 11-06 Tasks 1-2
- Migration cycle tests `TestMigration014..017` → 11-01 Task 3
- **Fix pre-existing red tests** `TestMigration011_ActivityOntology_UpDownUpCycle` + `TestMigration012_StaffingSchema_UpDownUpCycle` (self-seeded pre-state — Pitfall 3) → 11-01 Task 1
- `exported_test_helpers.go` teardown list extended with `tickets`, `ticket_comments`, `audit_logs` → 11-01 Task 1

## Security Domain

> `security_enforcement` not explicitly false in config → enabled.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (unchanged) | Existing JWT middleware (`middleware.Auth`), HttpOnly cookies — untouched this phase |
| V3 Session Management | no (unchanged) | Existing refresh-token rotation — untouched |
| V4 Access Control | yes | Service-level role gates (D-04 origin types, D-11 triage, D-15 ticket CRUD), ownership checks (owner/assignee), resolved-approver sets for proposal approval; all read from JWT claims via `middleware.GetRole/GetUserID` |
| V5 Input Validation | yes | Handler-boundary UUID/length parsing (house pattern), DB CHECK vocabularies (origin_type, ticket kind/status, contract_type, sold_period), service sentinel errors |
| V6 Cryptography | no (unchanged) | No new crypto — hours/notes are not secrets |

### Known Threat Patterns for {stack}
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Origin-ref spoofing / mass assignment (any employee setting manager-assignment refs) | Spoofing, Elevation of Privilege | Role gates per origin type (D-04); refs-to-type CHECK; `proposed_by` must equal the actor |
| Cross-org refs (ticket/user from another org) | Elevation of Privilege | Strict same-org validation in service (D-02): users via membership, tickets via `tickets.org_id = activities.org_id` |
| Unauthorized triage / transition / dismissal | Elevation of Privilege | D-11 (manager+finance triage) + D-14 state machine; only allowed edges; actor captured in audit row |
| Audit repudiation / tampering | Repudiation, Tampering | Append-only: no update/delete endpoints or repo methods (TICK-05); actor_id recorded on every row; synchronous in-transaction writes (recommended) |
| SQL injection | Tampering | Parameterized pgx queries everywhere (house rule, ADR-BE-003); no string-built SQL except the established dynamic-SET pattern in contract repo (values still parameterized) |
| Hours spoofing in dismissal guard | Tampering | Σ computed server-side from `time_entries` (statuses + is_deleted filter); client cannot supply the number |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase] `internal/core/services/time_entry/time_entry.go` — BE-014 manager-stage resolution to extract; approval flow; audit-write calls
- [VERIFIED: codebase] `internal/adapters/secondary/postgres/refresh_token_repo.go` — the codebase's `BeginTx`/`Rollback`/`Commit` transaction precedent
- [VERIFIED: codebase] `internal/adapters/secondary/postgres/{activity,contract,time_entry}_repository.go` — insert/scan/dynamic-SET patterns to extend; `wrapPGError` (postgres.go)
- [VERIFIED: codebase] `internal/adapters/secondary/postgres/{test_setup,exported_test_helpers,activity_ontology_migration_test,staffing_schema_migration_test}.go` — testcontainers, seed helpers, up/down/up cycle test pattern; teardown list
- [VERIFIED: codebase] `migrations/011_activity_ontology.up.sql`, `012_staffing_schema.{up,down}.sql`, `013_activity_kind_phase_fix.{up,down}.sql`, `000_full_schema.up.sql` — schema precedents (CHECK vocabularies, contracts table, time_entries.hours DECIMAL(8,2), time_entry_approvals)
- [VERIFIED: codebase] `internal/core/domain/{activity,contract,time_entry}/`, `internal/core/ports/*.go`, `internal/adapters/primary/http/{activity_handler,contract}.go`, `cmd/server/main.go` — DTOs, ports, routes
- [VERIFIED: vault] `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` — decisions D-A…D-AA, R4 resolution, Part 15 build order
- [VERIFIED: vault] `ADR-P-003`, `ADR-BE-012`, `ADR-BE-004`, `ADR-BE-014` — ADR shapes and the conventions to extend
- [CITED: github.com/jackc/pgx autodocs (via Context7)] — `BeginTx`, `TxOptions`, `BeginFunc`, `defer tx.Rollback` safety
- [CITED: postgresql.org/docs/current — ddl-constraints] — table-level CHECK multi-column; CHECK satisfied on TRUE or NULL; IS [NOT] NULL predicates
- [VERIFIED: execution] `go test` runs — TestMigration011/012 currently FAIL (seed wiring); rest of postgres package passes; `go mod verify` OK

### Secondary (MEDIUM confidence)
- [VERIFIED: repo] `.agents/skills/postgresql-table-design/SKILL.md` — DECIMAL for exact arithmetic, JSONB GIN, FK index guidance, CHECK semantics
- [VERIFIED: repo] `.planning/codebase/{CONVENTIONS,TESTING,ARCHITECTURE}.md` — naming, test layout, stack
- [CITED: postgresql.org/docs/current — functions-comparison] — `IS NULL`/`IS NOT NULL` predicates (via Context7)

### Tertiary (LOW confidence)
- [ASSUMED] General append-only audit-log design principles (index on entity_type/entity_id/created_at; immutability via absent update/delete paths) — external search providers unavailable (exa MCP + Brave API key absent); baseline from codebase precedent + ADR-BE-012 + D-05/D-06

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; every version verified in `go.mod`/`go.sum`
- Architecture: HIGH — all patterns verified in codebase (transactions, CHECKs, routing, migrations, tests); MEDIUM for the semantic design points (reviewed_by, terminal activity, audit durability) — flagged in Assumptions Log + Open Questions
- Pitfalls: HIGH for verified items (CHECK NULL semantics, red cycle tests, teardown list); MEDIUM for design-dependent items

**Research date:** 2026-08-03
**Valid until:** 2026-09-02 (stable codebase, no fast-moving deps; re-verify if go.mod changes)
