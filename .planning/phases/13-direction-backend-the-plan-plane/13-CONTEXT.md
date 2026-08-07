# Phase 13: Direction Backend — The Plan Plane - Context

**Gathered:** 2026-08-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend-only phase (no UI). The third plane of the three-plane ontology (direction → facts → coverage) lands server-side:

- **Direction entity** — one entity, per-day rows, mode derived from `planned_date` (set → scheduled, null → queued); est_hours semantics per mode; user-XOR-WG target (DIR-01)
- **Lifecycle** — draft → active → superseded/cancelled with audit-first via BE-012; done/lapsed/claimed derived, never stored; supersedes_id chains replanning (DIR-02)
- **WG-direction + claim model** — WG rows queued-only; hours-based split claims create user-targeted rows via `origin_direction_id`; claimed = derived spectrum (DIR-03)
- **Org planning policy** — org_settings key/value storage; deadline, horizon, per-employee mode; store + permission-gating only, no deadline enforcement (DIR-04)
- **Scheduler warning read path** — overlay on read-models: P-008 absence windows + employment validity → away/partial/over-capacity warnings, never blocks (DIR-05)
- **Direction-coverage read-model** — planned vs capacity per employee/unit/WG, uncovered-day surfacing, one endpoint with scope params (DIR-06)
- **Origin fallback** — activities with empty origin refs resolve from the first direction record (FND-04, read-only additive)
- **ADRs** — ADR-P-015 + BE encoding ADR drafted and recorded in the vault decisions folder

Deliverables are API endpoints + migrations + domain/ports/services/adapters + integration tests. All migrations append-only per ADR-BE-004 with up/down pairs + cycle tests.

</domain>

<decisions>
## Implementation Decisions

### Schema & est_hours semantics
- **D-13-01:** **Single direction table** — mode derived from `planned_date` (set → scheduled, null → queued), one entity per D-R. Rows carry `directed_by`, `directed_to` XOR `wg_id`, `activity_id`, `planned_date` (nullable), `est_hours` (nullable), `priority`, `due_date`, `status`, `supersedes_id`, `origin_direction_id`. CHECK-constrained vocabulary per house style.
- **D-13-02:** **est_hours required on scheduled rows, optional on queued rows.** Queued est_hours = the row's optional total budget (D-AA): the total estimate for the activity; the scheduler pre-fills per-day rows from remaining day capacity at drop.
- **D-13-03:** **Hard per-row validation, soft per-day warnings.** Service rejects `est_hours <= 0` (and absurd values) at write; Σ est_hours over day capacity is a soft warning returned in read-model/create responses, never a rejection (D-AA realism).
- **D-13-04:** **Immutable rows + supersede-only writes.** No in-place edit of planned_date/est_hours: replanning always creates a new row with `supersedes_id` pointing at the old row, which flips to `superseded` in the same transaction. The plan is mutable as a chain of rows, never by rewriting the fact of a prior plan. Audit-first via BE-012.
- **D-13-05:** **User-XOR-WG enforced via DB CHECK** — `(directed_to IS NULL AND wg_id IS NOT NULL) OR (directed_to IS NOT NULL AND wg_id IS NULL)`, mirroring the origin-refs CHECK pattern (D-01). Service-level validation as fast-fail on top.
- **D-13-06:** **Queued ordering: `priority INT` (lower = higher) + `due_date DATE`, both nullable.** Read ordering: priority then due_date.

### Lifecycle & supersede chaining
- **D-13-07:** **Ticket-style transition matrix** — `draft → active → superseded/cancelled`; draft is a real created state (created as draft, activated explicitly or via first plan action). Every transition writes an audit row to `audit_logs`. Enforced in the service; repo-layer re-validation inside the mutator tx under FOR UPDATE per the CR-01 closure pattern (house style for state machines).
- **D-13-08:** **Supersede is implicit via create** — a new row carrying `supersedes_id` flips the target row to `superseded` in the same transaction; the superseded row keeps its audit trail as history. No separate supersede endpoint.
- **D-13-09:** **Derived states computed on read, never stored, no nightly jobs** (D-V): `done` = linked activity terminal (reuse Phase 11's terminal-activity CTE semantics); `lapsed` = planned_date/due_date in the past without logged hours; `claimed` = existence of a claim row (see claim spectrum). Computed in the service/read-model.
- **D-13-10:** **Cancellation requires a reason** (mirroring the reject-with-reason pattern); draft and active rows are cancellable; `cancelled` is terminal with audit trail.

### WG claim model
- **D-13-11:** **Claim = derived row, by = WG row creator.** A member's claim creates a user-targeted row with `directed_by` = the WG row's creator (manager attribution preserved), `directed_to` = claimant, `origin_direction_id` = WG row ID, same activity. Not self-direction.
- **D-13-12:** **WG members only may claim** — membership checked at claim time.
- **D-13-13:** **Hours-based split claims (user override of single-claim lean):** a WG row's `est_hours` is its allocated hours; a claim may take all or part of them. Each claim carries its own `est_hours` (claimed amount). **Σ claimed ≤ WG est_hours enforced under tx lock** (first-wins/over-subscription race closed like CR-01). A single claim = claims all; multiple claims = split.
- **D-13-14:** **No cap when the WG budget is absent** (budget optional per D-AA): claims still require est_hours > 0 but are uncapped; the `consumed` state never derives.
- **D-13-15:** **Claimed state = derived spectrum:** `not_claimed → partially_claimed → fully_claimed` (fully only when a budget is set and Σ == budget). Never stored.
- **D-13-16:** **Unclaim = cancel the claim row** (reason required, like cancellation); hours return to the WG row automatically since consumption is Σ-derived, never stored.
- **D-13-17:** **WG rows are queued-only** (CHECK: `planned_date IS NULL` when `wg_id` set, per D-T "scheduling stays personal") and the activity must be within the WG's scope (same-org, reachable via WG subtree).

### Org policy storage & mode
- **D-13-18:** **Settings table, key/value JSONB (user override of typed-column lean):** `org_settings(org_id, key, value JSONB, updated_at, PK(org_id, key))` — "configurations are getting bigger and bigger". Validation lives in the domain/service per known key (CHECK on JSONB isn't feasible; house-style vocabulary enforced in code).
- **D-13-19:** **Org default + per-employee override** for planning mode: nullable override on `organization_memberships` falls back to the org default (`manager_planned` / `self_planned`).
- **D-13-20:** **Store + permission-gating only — no deadline enforcement.** Mode gates who may create scheduled rows for whom: manager-planned → managers create for that employee; self-planned → the employee creates own rows. Block-vs-nag soft policy deferred to UI prototyping (D-X); backend never blocks on deadline/horizon.
- **D-13-21:** **Horizon stored, not enforced** — the dynamic period is UI cadence + policy, not a schema dimension (D-W note).
- **D-13-22:** **Every settings change audit-logged** via BE-012 `audit_logs` (`entity_type=org_settings`) with before/after payload — who changed the planning deadline matters.
- **D-13-23:** **Generic settings endpoints, JWT-resolved org (house style):** `GET/PUT /organizations/settings` with `{key: value}` payloads, manager+ gated, additive keys. No org path param (auth context resolves the org).

### Coverage read-model & capacity
- **D-13-24:** **Daily working hours = org setting** (e.g., `planning_daily_hours`, default 8). Capacity per day = daily hours − confirmed P-008 absence hours that day (partial-day permits reduce by their `hours`, full absences zero the day).
- **D-13-25:** **One endpoint, scope params:** `GET /direction/coverage?scope=employee|unit|wg&scope_id=&period=` — aggregation differs only in which-employees resolution; unit/WG scopes aggregate employees underneath.
- **D-13-26:** **Uncovered day = absence-aware gap list:** a day whose Σ est_hours < capacity (including 0 — no direction), surfaced as (employee, date, capacity, planned, gap) plus period totals; fully-absent days excluded from uncovered surfacing.
- **D-13-27:** **Combined in one direction service** — the read-model endpoint delivers both the D-Z view and derived states (done/lapsed/claimed).

### Absence warnings & origin fallback
- **D-13-28:** **Warnings overlay on read paths** — coverage/plan read-model overlays absence windows + employment validity per (employee, day) and returns advisory warnings; never blocks writes (D-Y).
- **D-13-29:** **Read `availability_windows` with both `declared` and `confirmed` statuses for now** — Phase 14 adds confirm/reject; until then declared is provisional and counts.
- **D-13-30:** **Three warning types:** `away` (full absence), `partial` (partial-day permit), `over-capacity` (Σ est_hours > capacity). Soft, never blocking; surfaced in read-model and at create-response time.
- **D-13-31:** **Validity-aware surfacing:** an employee outside employment validity (`valid_from`/`valid_until` on membership) gets a validity warning and is NOT flagged uncovered — can't plan what can't work.
- **D-13-32:** **Origin fallback lives in the activity read path** (service layer): when origin refs are empty, look up the first direction record for that activity and derive manager-assignment-shaped refs (`directed_by` → `assigned_by`, `directed_to` → `assigned_to`). Additive, no migration (R4).
- **D-13-33:** **"First" = earliest `created_at` among non-cancelled rows** for the activity; if none, refs stay empty. The fallback only fills the manager-assignment shape, never employee_proposal/customer_ticket.
- **D-13-34:** **Fallback is read-only derivation, never written back** — stored refs stay authoritative for pre-direction activities (R4).

### the agent's Discretion
- Exact direction CRUD/transition endpoint list and URL shapes (within the decisions above)
- est_hours hour granularity (mirror time entries' DECIMAL, exact scale)
- Draft → active activation mechanics (explicit endpoint vs first-action activation)
- Read-model response envelope shape (fields, grouping, pagination)
- Test layout for the new direction/org_settings packages (follow per-package suite pattern)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Ontology research (record of truth)
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` — All direction decisions D-Q…D-AA closed here, Part 14 (§"Part 14 — Direction: the third plane", D-R…D-AA, lines ~376-494) and Part 15 (build order, §"R4 resolution: origin refs storage", lines ~500-538). Terminology convention (§"The naming collision — resolved by convention"): **direction** = plan, **coverage allocation** = money label.

### ADRs (vault decisions)
- `hourglass-vault/decisions/backend/ADR-BE-012 — Audit Log Writes.md` — Accepted. `audit_logs` table (migration 017) is the audit-first channel for direction lifecycle/supersede/settings events.
- `hourglass-vault/decisions/backend/ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution.md` — Manager reach for direction creation (managers direct within subtree/WG reach); shared routing service in `internal/core/services/routing/`.
- `hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md` — Schema-encoding house style this milestone references (CHECK constraints, additive migrations).
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` — Absence windows (holiday/permit/medical/unavailable) + employment validity; consumed by the DIR-05 warning path.
- `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md` — Three-plane context and the terminology convention (direction vs coverage allocation).
- New ADRs drafted this phase: **ADR-P-015** (direction plane) + **BE encoding ADR** (per milestone convention "Each backend phase drafts its ADR + BE encoding ADR").

### Milestone docs
- `.planning/ROADMAP.md` — Phase 13 entry: goal, requirements (DIR-01..06), 8 success criteria
- `.planning/REQUIREMENTS.md` — DIR-01..06 requirement text

### Codebase (read-only context)
- `migrations/012_staffing_schema.up.sql` — `availability_windows` (kinds, status declared/confirmed, partial-day `hours`) + `organization_memberships` validity columns (`valid_from`/`valid_until`/`work_permit_expires_at`) the warning path reads
- `migrations/017_audit_logs.up.sql` — `audit_logs` table shape direction/settings events reuse
- `migrations/014_ticket_schema.up.sql` + `migrations/015_activity_origins.up.sql` — ticket state-machine and origin-refs CHECK patterns to mirror (XOR CHECK, matrix, dismiss-guard-style in-tx re-validation)
- `internal/core/domain/ticket/ticket.go` + `internal/core/services/ticket/ticket.go` — state-machine pattern: repo-authoritative with in-tx FOR UPDATE re-validation (CR-01 closure), service-level fast-fail
- `internal/core/services/routing/routing.go` — BE-014 manager reach resolution for direction creation permission gating
- `internal/adapters/secondary/postgres/time_entry_repository.go` — audit-repo write pattern (BE-012)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `audit_logs` table + audit repository pattern (Phase 11) — direction lifecycle/supersede/settings events append here
- `internal/core/services/routing/routing.go` — manager reach resolution (subtree/WG) for direction-creation gating
- Ticket service state machine (matrix + in-tx re-validation under FOR UPDATE, CR-01 closure) — the pattern for direction lifecycle and claim over-subscription racing
- Terminal-activity recursive CTE (Phase 11, "no non-terminal entries on the subtree") — reuse for derived `done` state
- `organization_memberships` validity columns + `availability_windows` (migration 012) — consumed read-only by warnings/coverage

### Established Patterns
- Hexagonal: domain → ports → services → HTTP handlers → postgres repos; services own invariants, DB owns shapes (CHECK constraints)
- Hand-written SQL with pgx, no ORM; migrations append-only with up/down pairs + cycle tests (ADR-BE-004)
- CHECK-enforced vocabularies (house style); XOR CHECK precedent from origin refs (D-01)
- API response envelope `{ data | error }` via `pkg/api/response.go`; sentinel errors in domain, `wrapPGError` in postgres adapters
- Integration tests via testcontainers-go; per-package suites
- Audit-first via BE-012 for governed changes

### Integration Points
- New: `/direction` routes registered in `cmd/server/main.go` (Go 1.22+ pattern)
- New: `org_settings` table + repository — settings endpoints under `/organizations/settings`
- `GET /activities` read path (`internal/adapters/primary/http/activity_handler.go`) — origin fallback derivation hooks here (service layer)
- Shared routing service (`internal/core/services/routing/`) — permission gating for direction creation
- `availability_windows` + `organization_memberships` — read-only inputs to the coverage read-model and warning overlay

</code_context>

<specifics>
## Specific Ideas

- User on settings storage (verbatim intent): "I think that we should keep a settings table since configurations are getting bigger and bigger"
- User on claim lifecycle (verbatim intent): "a task should be consumed by its allocated hours, so that we are open to: claiming alone (just use all the allocated hours) or multiple claims (different users claim an amount of hours from it)" — this overrode the single-claim lean (D-T) in favor of hours-based split claims
- User on settings org-scoping: "settings are org wide, so we should at least define an org for them" — resolved as org_id column + JWT-resolved org (house style)

</specifics>

<deferred>
## Deferred Ideas

- **Block-vs-nag soft policy** (D-X parked): does an uncovered month block anything (timesheet submit / payroll export) or nag like the to-cover queue? Decided in UI prototyping (Phase 19), not backend.
- **Absence confirm/reject tightening** (Phase 14): warning path reads declared+confirmed for now; Phase 14 restricts to confirmed-only.
- **Plan-adherence analytics** (D-U): aggregate-only, per-period; V5 territory — the read-model must not become per-day-per-person surveillance.

</deferred>

---

*Phase: 13-Direction Backend — The Plan Plane*
*Context gathered: 2026-08-07*
