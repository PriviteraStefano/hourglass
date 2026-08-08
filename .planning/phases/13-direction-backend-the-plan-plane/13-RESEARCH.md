# Phase 13: Direction Backend — The Plan Plane - Research

**Researched:** 2026-08-08
**Domain:** Go/PostgreSQL backend — direction entity (per-day rows, derived modes), ticket-style lifecycle with supersede chaining, WG claim model with Σ-consumption, org_settings key/value policy storage, P-008-warning overlay, direction-coverage read-model, origin-ref fallback, hexagonal domain/services/adapters
**Confidence:** HIGH (stack + architecture patterns — verified in codebase this session), MEDIUM (open semantic points — flagged in Assumptions Log / Open Questions)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Schema & est_hours semantics
- **D-13-01:** **Single direction table** — mode derived from `planned_date` (set → scheduled, null → queued), one entity per D-R. Rows carry `directed_by`, `directed_to` XOR `wg_id`, `activity_id`, `planned_date` (nullable), `est_hours` (nullable), `priority`, `due_date`, `status`, `supersedes_id`, `origin_direction_id`. CHECK-constrained vocabulary per house style.
- **D-13-02:** **est_hours required on scheduled rows, optional on queued rows.** Queued est_hours = the row's optional total budget (D-AA): the total estimate for the activity; the scheduler pre-fills per-day rows from remaining day capacity at drop.
- **D-13-03:** **Hard per-row validation, soft per-day warnings.** Service rejects `est_hours <= 0` (and absurd values) at write; Σ est_hours over day capacity is a soft warning returned in read-model/create responses, never a rejection (D-AA realism).
- **D-13-04:** **Immutable rows + supersede-only writes.** No in-place edit of planned_date/est_hours: replanning always creates a new row with `supersedes_id` pointing at the old row, which flips to `superseded` in the same transaction. The plan is mutable as a chain of rows, never by rewriting the fact of a prior plan. Audit-first via BE-012.
- **D-13-05:** **User-XOR-WG enforced via DB CHECK** — `(directed_to IS NULL AND wg_id IS NOT NULL) OR (directed_to IS NOT NULL AND wg_id IS NULL)`, mirroring the origin-refs CHECK pattern (D-01). Service-level validation as fast-fail on top.
- **D-13-06:** **Queued ordering: `priority INT` (lower = higher) + `due_date DATE`, both nullable.** Read ordering: priority then due_date.

#### Lifecycle & supersede chaining
- **D-13-07:** **Ticket-style transition matrix** — `draft → active → superseded/cancelled`; draft is a real created state (created as draft, activated explicitly or via first plan action). Every transition writes an audit row to `audit_logs`. Enforced in the service; repo-layer re-validation inside the mutator tx under FOR UPDATE per the CR-01 closure pattern (house style for state machines).
- **D-13-08:** **Supersede is implicit via create** — a new row carrying `supersedes_id` flips the target row to `superseded` in the same transaction; the superseded row keeps its audit trail as history. No separate supersede endpoint.
- **D-13-09:** **Derived states computed on read, never stored, no nightly jobs** (D-V): `done` = linked activity terminal (reuse Phase 11's terminal-activity CTE semantics); `lapsed` = planned_date/due_date in the past without logged hours; `claimed` = existence of a claim row (see claim spectrum). Computed in the service/read-model.
- **D-13-10:** **Cancellation requires a reason** (mirroring the reject-with-reason pattern); draft and active rows are cancellable; `cancelled` is terminal with audit trail.

#### WG claim model
- **D-13-11:** **Claim = derived row, by = WG row creator.** A member's claim creates a user-targeted row with `directed_by` = the WG row's creator (manager attribution preserved), `directed_to` = claimant, `origin_direction_id` = WG row ID, same activity. Not self-direction.
- **D-13-12:** **WG members only may claim** — membership checked at claim time.
- **D-13-13:** **Hours-based split claims (user override of single-claim lean):** a WG row's `est_hours` is its allocated hours; a claim may take all or part of them. Each claim carries its own `est_hours` (claimed amount). **Σ claimed ≤ WG est_hours enforced under tx lock** (first-wins/over-subscription race closed like CR-01). A single claim = claims all; multiple claims = split.
- **D-13-14:** **No cap when the WG budget is absent** (budget optional per D-AA): claims still require est_hours > 0 but are uncapped; the `consumed` state never derives.
- **D-13-15:** **Claimed state = derived spectrum:** `not_claimed → partially_claimed → fully_claimed` (fully only when a budget is set and Σ == budget). Never stored.
- **D-13-16:** **Unclaim = cancel the claim row** (reason required, like cancellation); hours return to the WG row automatically since consumption is Σ-derived, never stored.
- **D-13-17:** **WG rows are queued-only** (CHECK: `planned_date IS NULL` when `wg_id` set, per D-T "scheduling stays personal") and the activity must be within the WG's scope (same-org, reachable via WG subtree).

#### Org policy storage & mode
- **D-13-18:** **Settings table, key/value JSONB (user override of typed-column lean):** `org_settings(org_id, key, value JSONB, updated_at, PK(org_id, key))` — "configurations are getting bigger and bigger". Validation lives in the domain/service per known key (CHECK on JSONB isn't feasible; house-style vocabulary enforced in code).
- **D-13-19:** **Org default + per-employee override** for planning mode: nullable override on `organization_memberships` falls back to the org default (`manager_planned` / `self_planned`).
- **D-13-20:** **Store + permission-gating only — no deadline enforcement.** Mode gates who may create scheduled rows for whom: manager-planned → managers create for that employee; self-planned → the employee creates own rows. Block-vs-nag soft policy deferred to UI prototyping (D-X); backend never blocks on deadline/horizon.
- **D-13-21:** **Horizon stored, not enforced** — the dynamic period is UI cadence + policy, not a schema dimension (D-W note).
- **D-13-22:** **Every settings change audit-logged** via BE-012 `audit_logs` (`entity_type=org_settings`) with before/after payload — who changed the planning deadline matters.
- **D-13-23:** **Generic settings endpoints, JWT-resolved org (house style):** `GET/PUT /organizations/settings` with `{key: value}` payloads, manager+ gated, additive keys. No org path param (auth context resolves the org).

#### Coverage read-model & capacity
- **D-13-24:** **Daily working hours = org setting** (e.g., `planning_daily_hours`, default 8). Capacity per day = daily hours − confirmed P-008 absence hours that day (partial-day permits reduce by their `hours`, full absences zero the day).
- **D-13-25:** **One endpoint, scope params:** `GET /direction/coverage?scope=employee|unit|wg&scope_id=&period=` — aggregation differs only in which-employees resolution; unit/WG scopes aggregate employees underneath.
- **D-13-26:** **Uncovered day = absence-aware gap list:** a day whose Σ est_hours < capacity (including 0 — no direction), surfaced as (employee, date, capacity, planned, gap) plus period totals; fully-absent days excluded from uncovered surfacing.
- **D-13-27:** **Combined in one direction service** — the read-model endpoint delivers both the D-Z view and derived states (done/lapsed/claimed).

#### Absence warnings & origin fallback
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

### Deferred Ideas (OUT OF SCOPE)
- **Block-vs-nag soft policy** (D-X parked): does an uncovered month block anything (timesheet submit / payroll export) or nag like the to-cover queue? Decided in UI prototyping (Phase 19), not backend.
- **Absence confirm/reject tightening** (Phase 14): warning path reads declared+confirmed for now; Phase 14 restricts to confirmed-only.
- **Plan-adherence analytics** (D-U): aggregate-only, per-period; V5 territory — the read-model must not become per-day-per-person surveillance.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIR-01 | Direction rows: per-day storage always, est_hours per row, partial days first-class, no intra-day ordering; mode derived (planned_date set → scheduled, null → queued with priority + due_date); self-direction (`directed_by == directed_to`) needs no approval; managers direct within subtree/WG reach (D-R, D-S, D-W, D-AA) | D-13-01..06 locked; est_hours DECIMAL(8,2) mirrors `time_entries.hours` `[VERIFIED: migrations/000_full_schema.up.sql:278]`; manager gate reuses `routing.ResolveManagerStage` `[VERIFIED: internal/core/services/routing/routing.go:57-104]`; coverage service shows the exact gate shape `[VERIFIED: internal/core/services/coverage/coverage.go:353-369]` |
| DIR-02 | Lifecycle draft → active → superseded/cancelled; done/lapsed/claimed derived never stored; supersedes_id chains replanning; audit-first via BE-012 (D-V) | Ticket state-machine pattern is the template: matrix in domain `[VERIFIED: internal/core/domain/ticket/ticket.go:89-113]`, repo-authoritative in-tx re-validation under FOR UPDATE `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:360-378]`, in-tx audit insert `[VERIFIED: internal/adapters/secondary/postgres/coverage_repository.go:45]`; terminal-activity CTE for derived `done` `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:816-835]` (anchor must switch to `activities.id`); audit_logs shape `[VERIFIED: migrations/017_audit_logs.up.sql:16-26]` |
| DIR-03 | WG-direction queued-only; member claim creates user-targeted row via origin_direction_id; claimed derived (D-T) | D-13-11..17 locked; Σ-claims ≤ WG est_hours under tx lock = CR-01 closure (lock the WG row FOR UPDATE, re-check in-tx — precedent `[VERIFIED: ticket_repository.go:370]` and coverage ReplaceAllocations `[VERIFIED: coverage_repository.go:105-163]`); WG membership via `ports.WorkingGroupRepository.ListMembers` `[VERIFIED: internal/core/ports/working_group_repository.go:9-20]`; WG anchor = `SubprojectID` (activity) `[VERIFIED: internal/core/domain/working_group/working_group.go:15-32]` |
| DIR-04 | Org policy org-configurable: deadline, horizon (day/week/month), per-employee mode (manager-planned vs self-planned); soft policy deferred to UI (D-X) | D-13-18..23 locked; `org_settings` key/value JSONB, validation in code per known key; `GET/PUT /organizations/settings` literal routes coexist with existing `GET/PUT /organizations/{id}/settings` (typed `organization_settings`) — ServeMux literal-beats-wildcard `[CITED: pkg.go.dev/net/http@go1.25.3]` + in-repo proof `[VERIFIED: cmd/server/main.go:207-214]`; JWT-resolved org via `middleware.GetOrganizationID` `[VERIFIED: internal/middleware/middleware.go:73-74]`; membership override column on `organization_memberships` (012 already added validity columns `[VERIFIED: migrations/012_staffing_schema.up.sql:39-42]`) |
| DIR-05 | Scheduler reads P-008 absence windows + employment validity and warns at plan time — never blocks (D-Y) | `availability_windows` verified: `kind IN ('holiday','permit','medical','unavailable')`, `status IN ('declared','confirmed')`, `hours DECIMAL(4,2)` partial-day, `starts_on/ends_on` `[VERIFIED: migrations/012_staffing_schema.up.sql:15-30]`; validity columns `[VERIFIED: 012:39-42]`; D-13-29 reads both statuses (Phase 14 tightens); warning types closed set `away|partial|over-capacity|invalid` pinned in 13-UI-SPEC |
| DIR-06 | Direction-coverage read-model: planned vs capacity per employee/period, uncovered-day surfacing, per employee/unit/WG (D-Z) | D-13-24..27 locked; capacity = `planning_daily_hours` setting (default 8) − absence hours; uncovered = Σ est_hours < capacity, fully-absent days + validity-outside employees excluded (D-13-26/31); scope resolution via unit tree (`GetDescendants`/`ListMembers` `[VERIFIED: internal/core/ports/unit_repository.go:9-30]`) + WG members (`ListMembers`); derived states computed on read (D-13-27) |

## Summary

Phase 13 is a **backend-only, purely additive phase** on the settled hexagonal codebase (Go 1.26.1 + pgx v5.10.0 + stdlib net/http). It lands the third plane: a single `direction` table with per-day rows and derived mode (`planned_date` set → scheduled, null → queued), a ticket-style lifecycle (draft → active → superseded/cancelled) with implicit supersede-on-create chaining and audit-first via BE-012, an hours-based WG claim model with Σ-consumption enforced under a transaction lock, `org_settings` key/value JSONB policy storage (deadline, horizon, daily hours, per-employee planning mode) with generic JWT-resolved endpoints, a P-008 absence/validity warning overlay that never blocks, a direction-coverage read-model (planned vs capacity per employee/unit/WG with uncovered-day surfacing), and the origin-ref read-path fallback that derives manager-assignment refs from the first direction record (FND-04/R4).

**No new external packages are needed.** Every library (pgx, uuid, testify, testcontainers-go) is already in `go.mod` — verified this session. Every mechanism has a verified codebase precedent: the ticket state machine (matrix + in-tx FOR UPDATE re-validation, CR-01 closure), the in-tx audit-insert helper (BE-016/017), the coverage Σ-in-cents arithmetic, the shared BE-014 routing gate, the chain-walk CTE family, and the 3VL CHECK-guarded vocabularies.

**Primary recommendation:** plan two additive migrations (021 `direction` rows, 022 `org_settings` + `organization_memberships.planning_mode` override), a new `direction` domain + service + postgres repo + HTTP handler pair, a small `orgsettings` domain/service/repo/handler pair (keeps the typed `organization_settings` surface untouched), extension of the activity service read path for the origin fallback (via a new small direction-ref port — a constructor/wiring change in `cmd/server/main.go`), and the coverage read-model + warning overlay in the direction service. The BE encoding ADR must pin: direction status/derived-state vocabularies, the claim spectrum, audit entity_type/action vocabulary, settings key vocabulary, the `planning_daily_hours` default, and the claim over-subscription lock. ADR-P-015 is drafted from the Part 14/15 record of truth.

**⚠️ Key planning constraints:** (1) `GET/PUT /organizations/settings` must be registered as literal routes — they coexist with the existing `GET/PUT /organizations/{id}/settings` (ServeMux literal-beats-wildcard, proven in-repo by `POST /organizations/invite` beside `GET /organizations/{id}`); (2) the activity service gains a new dependency for the origin fallback — its constructor call sites and the testdata mocks change; (3) every claim/Σ check must re-run inside the mutator tx under the WG-row FOR UPDATE lock (CR-01) — pool-level checks are fast-fail UX only; (4) all new tables join the shared teardown list; (5) migration numbering continues at 021 (ADR-BE-004).

## Project Constraints (from AGENTS.md)

- Hexagonal architecture: domain → ports → services → adapters; business logic in `internal/core/services/`, handlers in `internal/adapters/primary/http/` stay thin (AGENTS.md "Hexagonal Architecture" section)
- Hand-written SQL with pgx, **no ORM**; migrations append-only with up/down pairs + cycle tests (ADR-BE-004)
- API envelope `{ data | error }` via `pkg/api/response.go`; sentinel errors in domain + `wrapPGError` in postgres adapters
- Routes use Go 1.22+ `mux.HandleFunc("METHOD /path", handler)` with kebab-case paths; `middleware.Auth` on protected routes
- Models use `uuid.UUID` IDs and timezone-aware `time.Time`; CHECK-constrained vocabularies (house style)
- After modifying code files this session, run the graphify rebuild command to keep the graph current; openwiki docs exist under `/openwiki` (consult for architecture notes)
- Integration tests via testcontainers-go; per-package suites; cycle tests self-seed pre-state inline (Phase 11 rule — no demo data in tests)

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Direction row storage + XOR/vocabulary CHECKs (DIR-01, D-13-01/05/17) | Database/Storage | API/Backend (service) | DB owns shapes (house style); service fast-fails same rules |
| Derived mode/state computation (DIR-01/02, D-13-09) | API/Backend (service) | Database/Storage (read queries) | done/lapsed/claimed computed on read; never stored, no nightly jobs |
| Lifecycle + supersede chain (DIR-02, D-13-04/07/08/10) | Database/Storage (transaction) | API/Backend (service) | Matrix + in-tx FOR UPDATE re-validation (CR-01); supersede flips target in the same tx |
| Audit trail (DIR-02/04, BE-012) | Database/Storage (append-only) | API/Backend | In-tx synchronous `audit_logs` rows (ticket/coverage precedent) |
| WG claim model + Σ consumption (DIR-03, D-13-11..16) | Database/Storage (transaction) | API/Backend (service) | Σ claimed ≤ WG est_hours under FOR UPDATE WG-row lock; membership checked at claim time |
| Org policy storage (DIR-04, D-13-18/19/22/23) | Database/Storage (key/value JSONB) | API/Backend (service) | org_settings PK(org_id,key); vocabulary validated in code; manager+ gate; audit with before/after |
| Mode permission gating (DIR-01/04, D-13-20) | API/Backend (service) | — | routing.ResolveManagerStage (BE-014) for manager writes; self-direction = directed_by == directed_to, no approval |
| Warning overlay (DIR-05, D-13-28..31) | API/Backend (read-model) | Database/Storage (read-only) | availability_windows + membership validity overlaid on read paths; never blocks |
| Coverage read-model (DIR-06, D-13-24..27) | API/Backend (read-model) | Database/Storage | Planned Σ vs capacity per employee/period; scope params resolve employee sets |
| Origin fallback (D-13-32..34, FND-04) | API/Backend (service) | — | First non-cancelled direction row derives manager-assignment refs; read-only, never written back |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go (stdlib `net/http`) | 1.26.1 | Server, routing (`mux.HandleFunc("POST /direction", ...)`), JSON | `[VERIFIED: go.mod + cmd/server/main.go]` — Go 1.22+ method patterns everywhere; literal routes proven alongside wildcards |
| jackc/pgx v5 | v5.10.0 | PostgreSQL access, `pgxpool`, transactions, FOR UPDATE locks | `[VERIFIED: go.mod]` — `BeginTx` + in-tx audit precedent in ticket/coverage repos `[CITED: pgx autodocs via Context7]` |
| github.com/google/uuid | v1.6.0 | UUIDs for entities | `[VERIFIED: go.mod]` — house style (`uuid.UUID` PKs) |
| stretchr/testify | v1.11.1 | assert/require in all tests | `[VERIFIED: go.mod]` — used by every suite |
| testcontainers-go (postgres module) | v0.42.0 | Integration-test DB (`postgres:16-alpine`) | `[VERIFIED: go.mod]` + `test_setup.go` (`SetupPackageContainer`) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cmd/migrate` | — | Apply `migrations/*.up.sql` / `*.down.sql` | All schema changes (ADR-BE-004); cycle tests call `applyMigrations` |
| `pkg/api` | — | `{ data \| error }` envelope | Every handler response/error |
| `internal/middleware` | — | `Auth`, `GetRole`, `GetOrganizationID`, `GetUserID` | Claims from context — JWT-resolved org for `/organizations/settings` `[VERIFIED: middleware.go:73-74]` |
| `internal/core/services/routing` | — | BE-014 manager-stage resolution (`ResolveManagerStage`) | Manager create gate — reuse, do not re-implement `[VERIFIED: routing.go:57-104]` |
| `internal/core/ports` repos | — | activity, wg, unit, org, coverage, audit ports | Read-only inputs (activity/WG/unit/validity) + in-tx audit insert helper |
| `scripts/seed_demo.sql` | — | MVP demo data | Dev/demo only; **not** loaded by test helpers (Phase 11 Pitfall 5) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Single direction table with derived mode (D-13-01) | Separate scheduled/queued tables | Locked — D-R/D-13-01; mode is one nullable column, one CHECK |
| Supersede-on-create (D-13-08) | Dedicated supersede endpoint | Locked — no separate endpoint; chain written in one tx |
| `org_settings` key/value JSONB (D-13-18) | Extend typed `organization_settings` columns | Locked — user: "configurations are getting bigger and bigger"; the typed table stays untouched |
| Literal `/organizations/settings` (D-13-23) | Path-param `/organizations/settings/{key}` | Locked — JWT-resolved org, no path param; ServeMux literal/wildcard coexistence proven in-repo |
| Claim = derived spectrum (D-13-15) | Stored `claimed` status | Locked — D-V philosophy; Σ-derived on read |

**Installation:**
```bash
# No new packages. Zero dependency changes this phase — verify with `go mod verify` (OK).
```

**Version verification:** all libraries present in the repo's committed `go.mod` (Go 1.26.1, pgx v5.10.0, uuid v1.6.0, testify v1.11.1, testcontainers-go v0.42.0 — read this session). No registry lookup needed — this phase adds zero dependencies.

## Package Legitimacy Audit

> No new external packages are installed by this phase. The phase runs entirely on the existing dependency set. The package-legitimacy seam supports npm/pypi/crates only (Go unsupported); equivalent verification performed via `go.mod` + `go.sum` + `go mod verify`.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| jackc/pgx/v5 v5.10.0 | Go modules | long-standing | high | github.com/jackc/pgx | OK | Approved (already in go.mod) |
| google/uuid v1.6.0 | Go modules | long-standing | high | github.com/google/uuid | OK | Approved (already in go.mod) |
| stretchr/testify v1.11.1 | Go modules | long-standing | high | github.com/stretchr/testify | OK | Approved (already in go.mod) |
| testcontainers-go v0.42.0 | Go modules | long-standing | high | github.com/testcontainers/testcontainers-go | OK | Approved (already in go.mod) |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
**No `[ASSUMED]` package names:** all names verified in the committed `go.mod`/`go.sum` — no external discovery involved.

## Architecture Patterns

### System Architecture Diagram

```mermaid
flowchart LR
    subgraph API["HTTP layer (thin handlers, pkg/api envelope)"]
        CR["POST /direction<br/>(create: draft · scheduled · queued · WG)<br/>returns row + warnings (D-13-03)"]
        AC["POST /direction/{id}/activate<br/>(draft → active)"]
        CA["POST /direction/{id}/cancel<br/>(reason required)"]
        CL["POST /direction/claims<br/>(claim from WG row, Σ lock)"]
        UC["POST /direction/claims/{id}/cancel<br/>(unclaim, reason required)"]
        PL["GET /direction<br/>(plan/queue read-model + derived states<br/>+ warnings overlay)"]
        CV["GET /direction/coverage<br/>(scope=employee|unit|wg + period)"]
        SG["GET/PUT /organizations/settings<br/>(key/value JSONB, manager+)"]
        AG["GET /activities, GET /activities/{id}<br/>(+ origin fallback)"]
    end
    subgraph SVC["Services (invariants, gates)"]
        DS["direction.Service<br/>mode · lifecycle matrix · supersede<br/>claim Σ · warnings · coverage read-model"]
        OS["orgsettings.Service<br/>known-key validation · manager+ gate"]
        AS["activity.Service<br/>(+ origin fallback derivation)"]
        RS["routing.Service (shared BE-014)<br/>ResolveManagerStage"]
    end
    subgraph DB["PostgreSQL (pgx repos)"]
        DIR["direction rows (021)<br/>XOR user/wg · status vocab<br/>queued-only WG · supersedes_id"]
        OSG["org_settings (022)<br/>key/value JSONB · PK(org,key)"]
        MEM["organization_memberships<br/>+ planning_mode override (022)<br/>valid_from/valid_until (012)"]
        AVW["availability_windows (012)<br/>declared+confirmed (D-13-29)"]
        ACT["activities (origin refs)"]
        TE["time_entries (derived done/lapsed)"]
        AUL["audit_logs (in-tx rows)"]
        WGU["working_groups + members"]
    end
    CR --> DS --> RS
    CR -->|"tx: FOR UPDATE target row<br/>supersede flip · audit"| DIR
    CL -->|"tx: lock WG row FOR UPDATE<br/>Σ re-check → insert claim → audit"| DIR
    PL --> DS -->|"warnings + derived states"| AVW
    PL --> DS --> TE
    CV --> DS -->|"capacity − absences"| AVW
    CV --> DS -->|"scope employee set"| MEM
    CV --> DS -->|"unit/WG scope"| WGU
    SG --> OS --> OSG
    SG --> OS -->|"in-tx audit row"| AUL
    AG --> AS -->|"empty origin refs → first direction row"| DIR
    DS -->|"in-tx audit rows (BE-012)"| AUL
```

**Read path (DIR-06):** coverage endpoint → scope resolution (employee / unit+descendants members / WG members) → per (employee, day): capacity = `planning_daily_hours` (org setting, default 8) − Σ P-008 absence hours that day → Σ planned (non-cancelled, non-superseded scheduled rows) → uncovered days surfaced `(employee, date, capacity, planned, gap)`; fully-absent days and validity-outside employees excluded; warnings (`away`/`partial`/`over-capacity`/`invalid`) overlaid. Never blocks (D-13-28).

**Origin fallback (D-13-32..34):** activity read path (service layer) — if origin refs empty, look up earliest `created_at` non-cancelled direction row for the activity, derive `assigned_by = directed_by`, `assigned_to = directed_to`. Read-only, never written back.

### Recommended Project Structure
```
internal/core/domain/direction/        # NEW: Direction entity, status vocab, claim spectrum, errors.go
internal/core/domain/orgsettings/      # NEW: settings key vocabulary, known-key validation, errors.go (small)
internal/core/ports/                   # NEW: direction_repository.go, org_settings_repository.go; EXTEND: activity_repository.go (fallback port) 
internal/core/services/direction/      # NEW: create/activate/cancel/claim/unclaim, plan read-model, coverage, warnings
internal/core/services/orgsettings/    # NEW: Get/Put with per-key validation + audit
internal/core/services/activity/       # EXTEND: origin fallback in GetByID/List (new direction-ref port dependency)
internal/adapters/primary/http/        # EXTEND: direction_handler.go (NEW), org_settings_handler.go (NEW), activity_handler.go (fallback enrichment)
internal/adapters/secondary/postgres/  # EXTEND: direction_repository.go (NEW), org_settings_repository.go (NEW), activity_repository.go (fallback lookup)
migrations/                            # NEW: 021_direction_rows, 022_org_settings (up/down pairs)
hourglass-vault/decisions/             # ADR-P-015 (NEW, draft) + ADR-BE-0xx direction encoding (NEW) + _index.md
internal/adapters/secondary/postgres/exported_test_helpers.go  # EXTEND: teardown table list (+ direction, org_settings)
internal/core/services/testdata/       # EXTEND: MockDirectionRepo, MockOrgSettingsRepo (+ MockActivityRepo constructor change)
cmd/server/main.go                     # EXTEND: service wiring + routes
```

**Suggested migration split (numbering continues from 020, per ADR-BE-004):**
- `021_direction_rows.{up,down}.sql` — the `direction` table (shape below). **Down:** `DROP TABLE IF EXISTS direction CASCADE;`
- `022_org_settings.{up,down}.sql` — `org_settings` table + `ALTER TABLE organization_memberships ADD COLUMN planning_mode VARCHAR(20)` (nullable override; vocabulary CHECK optional — house style favors a CHECK, but the D-13-18 code-enforced vocabulary leans toward none; planner call). **Down:** drop column then table.
Each with a cycle test (up → down → up) in the postgres package, mirroring the `TestMigration014..020` self-seed pattern `[VERIFIED: ontology_extension_migrations_test.go / coverage_ontology_migrations_test.go]`.

### Pattern 1: Single direction table with derived mode + XOR CHECKs (D-13-01/05/17)
**What:** one entity, mode derived from `planned_date`. Per-day rows always; multiple rows may share a day (no unique constraint on (employee, activity, day)); no intra-day ordering.

```sql
-- 021 up (recommended shape — planner confirms vocabulary):
CREATE TABLE direction (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID NOT NULL REFERENCES organizations(id),
    directed_by        UUID NOT NULL REFERENCES users(id),   -- creator/manager attribution
    directed_to        UUID REFERENCES users(id),            -- XOR with wg_id (D-13-05)
    wg_id              UUID REFERENCES working_groups(id),   -- WG target (D-T)
    activity_id        UUID NOT NULL REFERENCES activities(id),
    planned_date       DATE,                                 -- NULL = queued (D-R)
    est_hours          DECIMAL(8,2),                         -- required on scheduled; optional queued budget (D-13-02)
    priority           INT,                                  -- queued ordering, lower = higher (D-13-06)
    due_date           DATE,                                 -- queued ordering (D-13-06)
    status             VARCHAR(20) NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft','active','superseded','cancelled')),  -- D-13-07
    supersedes_id      UUID REFERENCES direction(id),        -- replanning chain (D-13-04/08)
    origin_direction_id UUID REFERENCES direction(id),       -- claim chain: WG row → claim row (D-13-11)
    reason             TEXT,                                 -- mandatory for cancel/unclaim (D-13-10/16)
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- User-XOR-WG (D-13-05 — unconditional, no 3VL guard needed: new table, no legacy rows):
ALTER TABLE direction ADD CONSTRAINT direction_target_check
    CHECK ((directed_to IS NOT NULL AND wg_id IS NULL)
        OR (directed_to IS NULL AND wg_id IS NOT NULL));
-- WG rows are queued-only (D-13-17):
ALTER TABLE direction ADD CONSTRAINT direction_wg_queued_check
    CHECK (wg_id IS NULL OR planned_date IS NULL);
-- est_hours: > 0 when present; required on scheduled rows (D-13-02):
ALTER TABLE direction ADD CONSTRAINT direction_est_hours_check
    CHECK (est_hours IS NULL OR est_hours > 0);
ALTER TABLE direction ADD CONSTRAINT direction_scheduled_hours_check
    CHECK (planned_date IS NULL OR est_hours IS NOT NULL);
-- Cancellation requires a reason (D-13-10):
ALTER TABLE direction ADD CONSTRAINT direction_cancel_reason_check
    CHECK (status <> 'cancelled' OR reason IS NOT NULL);
-- Indexes: (org_id, directed_to, planned_date) · (org_id, wg_id) ·
-- (activity_id, created_at) for the origin fallback (D-13-33) · (supersedes_id)
```
**Verified constraint:** `est_hours DECIMAL(8,2)` mirrors `time_entries.hours` exactly — `[VERIFIED: migrations/000_full_schema.up.sql:278]` — quote: `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)`. Hour granularity (discretion) = 2 decimal places, same as entries.

**Planner notes:** `supersedes_id`/`origin_direction_id` self-FKs are fine (audit trail preserved — superseded/claim rows are never deleted). There is **no `is_deleted` soft-delete** — `cancelled` is the terminal state (D-13-10). The claim row (`origin_direction_id` set, `directed_to` set) is itself a user-targeted row with its own `est_hours` (claimed amount, D-13-13) — mode is queued (`planned_date` NULL) by default; the claimant then schedules via the normal replanning chain (planner confirms). Whether the claim row copies `priority`/`due_date` from the WG row is planner discretion (recommend yes — queue ordering consistency).

### Pattern 2: Ticket-style lifecycle with in-tx re-validation (D-13-07/10, CR-01 closure)
**What:** the direction service mirrors the ticket service exactly: a domain `transitionMatrix` + `CanTransition`, service-level fast-fail, and repo-authoritative re-validation inside the mutator tx under `SELECT ... FOR UPDATE`.

**Pinned matrix** (D-13-07): `draft → active`; `draft|active → cancelled` (D-13-10, reason required); `draft|active → superseded` (only via implicit supersede-on-create, D-13-08 — NOT a transition endpoint). `superseded`/`cancelled` are terminal. Every transition writes an `audit_logs` row in the same tx.

```go
// Domain matrix (mirrors ticket domain — [VERIFIED: internal/core/domain/ticket/ticket.go:89-113]):
var transitionMatrix = map[string]map[string]bool{
    StatusDraft:   {StatusActive: true, StatusCancelled: true},
    StatusActive:  {StatusCancelled: true},
    // superseded reachable ONLY via Create with supersedes_id (D-13-08)
}
```
**Verified lock precedent:** `SELECT status FROM tickets WHERE id = $1 AND org_id = $2 FOR UPDATE` — `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:370]`; the in-tx re-check pattern (matrix + status-precondition UPDATE backstop) is the CR-01 closure recorded in STATE.md. The direction mutator must lock the target row (for activate/cancel/supersede) and re-check status in-tx — pool-level checks are fast-fail UX only.

**Audit vocabulary (pin in BE ADR, like ADR-BE-017):** `entity_type='direction'`, actions `created` / `activated` / `cancelled` / `superseded` / `claimed` / `unclaimed`; payload carries the row state (planned_date, est_hours, supersedes_id, reason). `entity_id` = the direction row id. `audit_logs` shape verified: `[VERIFIED: migrations/017_audit_logs.up.sql:16-26]` (free-form `entity_type` VARCHAR(50) — the vocabulary is a convention, pinned in the ADR).

### Pattern 3: Supersede-on-create in one transaction (D-13-04/08)
**What:** replanning = `Create` with `supersedes_id` set. One repo tx: lock the target row FOR UPDATE → re-check it is `draft`|`active` (CR-01) → INSERT the new row → UPDATE the target to `superseded` → write the `superseded` + `created` audit rows → COMMIT. No separate supersede endpoint (D-13-08).

**Create flow (recommended):**
1. Service validates: mode shape (scheduled needs planned_date + est_hours > 0; queued optional budget; WG rows queued-only), XOR target, activity same-org, WG-scope for WG rows (D-13-17), permission (D-13-20 gate).
2. Permission gate: self-direction (`directed_to == actor`) always allowed (D-S, no approval). Manager-directed: `routing.ResolveManagerStage(ctx, orgID, activityID, unitID?, ownerID?)` — the BE-014 gate, same shape as coverage `[VERIFIED: internal/core/services/coverage/coverage.go:353-369]`:
```go
// quote from coverage service (read this session) — the D-08 gate shape:
if !res.RoleGated && !contains(res.ApproverIDs, actor) { return nil, coverage.ErrForbidden }
if res.RoleGated && role != string(models.RoleManager) { return nil, coverage.ErrForbidden }
```
3. Mode gate (D-13-20): resolve the target employee's planning mode (`planning_mode` override on membership → org default setting); manager-planned → manager creates (routing gate); self-planned → only the employee creates own rows. (Exact exclusivity is planner/Open Question 7 — recommend the strict reading.)
4. If `supersedes_id` set → supersede tx (above). Else plain insert + `created` audit.
5. Compute soft warnings for scheduled rows (Pattern 6) and return them with the created row (D-13-03).

### Pattern 4: Derived states on read — never stored (D-13-09/15)
**What:** `done`, `lapsed`, `claimed` (and the claim spectrum) are computed by the service/read-model, never persisted, no nightly jobs.

- **`done`** = the linked activity is terminal. Reuse Phase 11's terminal-activity CTE semantics but re-anchor: the ticket CTE anchors `activities.ticket_id = $1` `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:819-829]` — direction must anchor `activities.id = <activity_id>` and keep the same subtree walk (activity + descendants) and the same non-terminal entry predicate (quote: `te.status IN ('draft','submitted','pending_manager','pending_finance')` and `te.is_deleted = false`). Note the semantic inversion: ticket `HasNonTerminalActivities` returns "has non-terminal"; direction `done` = NOT has-non-terminal.
- **`lapsed`** = `planned_date`/`due_date` in the past AND no logged hours on the activity subtree (exact "logged hours" predicate — see Open Question 2; recommend: no non-deleted time entries at all on the subtree).
- **`claimed`** (WG rows) = derived spectrum `not_claimed → partially_claimed → fully_claimed` (D-13-15): Σ est_hours of claim rows (`origin_direction_id = <wg row id>`, status != cancelled) vs the WG row's `est_hours`; `fully_claimed` only when the budget is set and Σ == budget; uncapped rows never derive `fully_claimed` (D-13-14).
- Only `draft`|`active` non-superseded rows participate in derived-state computation and the coverage read-model (superseded/cancelled rows are history — D-13-08).

### Pattern 5: WG claim model with Σ-consumption under tx lock (D-13-11..16, CR-01)
**What:** claims are inserts of user-targeted rows with `origin_direction_id`; consumption is Σ-derived, never stored.

**Claim tx (recommended):**
1. Lock the WG row: `SELECT status, est_hours FROM direction WHERE id = $1 AND org_id = $2 AND wg_id IS NOT NULL FOR UPDATE` (CR-01 — serializes concurrent claims; the in-tx re-check is authoritative).
2. Re-check in-tx: WG row is `active` (or `draft`? — planner; recommend active), claimant is a WG member (`wgRepo.ListMembers` — `[VERIFIED: internal/core/ports/working_group_repository.go:16]`), activity still within WG scope, claimed `est_hours > 0`.
3. Σ guard (when budget set): `SELECT COALESCE(SUM(est_hours),0) FROM direction WHERE origin_direction_id = $1 AND status IN ('draft','active')` — superseded/cancelled claim rows never consume budget (checker-fix predicate pinned in ADR-BE-018; superseded rows are immutable history and a supersede of a claim row carries `origin_direction_id` onto the new row) + claimed amount ≤ WG `est_hours` → else `ErrClaimOverBudget` (409). Uncapped when WG `est_hours` is NULL (D-13-14).
4. INSERT claim row: `directed_by` = WG row's creator (D-13-11 attribution), `directed_to` = claimant, `origin_direction_id` = WG row id, same activity, `est_hours` = claimed amount; write `claimed` audit row in the same tx.
**Unclaim** = cancel the claim row (reason required, D-13-16) — hours return automatically because consumption is Σ-derived.

**⚠️ The over-subscription race** is the CR-01 failure mode: pool-level Σ checks leave a check-then-act window between two concurrent claims. The in-tx `SELECT ... FOR UPDATE` on the WG row + in-tx Σ re-check is the correctness guarantee (precedent: `ticket_repository.go:370`; the concurrent-write test template is `ticket_repository_test.go:418-506` — mirror it for claims).

### Pattern 6: Warning overlay — read-only, never blocks (D-13-28..31)
**What:** a service-side warning function overlaid on plan/coverage read paths and create responses. Inputs (read-only): `availability_windows` `[VERIFIED: migrations/012_staffing_schema.up.sql:15-30]` — quote of the two vocabularies: `kind IN ('holiday', 'permit', 'medical', 'unavailable')`, `status IN ('declared', 'confirmed')`, `hours DECIMAL(4,2)` (partial-day), `starts_on`/`ends_on`; and `organization_memberships.valid_from`/`valid_until` `[VERIFIED: 012:39-42]` (quote: `ADD COLUMN valid_from DATE, ADD COLUMN valid_until DATE, ADD COLUMN work_permit_expires_at DATE;`).

Per (employee, day):
- `away` — a full absence window covers the day (window with `hours IS NULL` or window whose absence zeroes capacity)
- `partial` — a partial-day permit (`hours` set) reduces capacity by `hours`
- `over-capacity` — Σ est_hours (scheduled rows) > capacity
- `invalid` — day outside `valid_from`/`valid_until` (employee NOT flagged uncovered, D-13-31)

**Contract (UI-SPEC pinned):** warnings are `{ type, message }` objects, types closed set `away | partial | over-capacity | invalid`; message pre-rendered server-side as `{Type} {date-range-or-day}` (e.g. `"Away 10–21 Aug"`, `"Partial 14 Aug"`, `"Over capacity 16 Aug"`, `"Outside validity period"`) — the UI renders `message` verbatim (Phase 19). D-13-29: read both `declared` and `confirmed` statuses now.

### Pattern 7: org_settings key/value storage + code-enforced vocabulary (D-13-18..23)
**What:** `org_settings(org_id, key, value JSONB, updated_at, PK(org_id, key))` — generic; validation per known key lives in the domain/service (CHECK on JSONB isn't feasible). `GET/PUT /organizations/settings` with `{key: value}` payloads, manager+-gated, JWT-resolved org (`middleware.GetOrganizationID` — `[VERIFIED: internal/middleware/middleware.go:73-74]`), additive keys, unknown key → 400.

**Recommended key vocabulary (pin in BE ADR):** `planning_daily_hours` (number, default 8 — capacity denominator, D-13-24), `planning_deadline` (date — D-X axis ①), `planning_horizon` (`day|week|month` — D-X axis ②, stored not enforced D-13-21), `planning_mode` (org default `manager_planned|self_planned` — D-13-19). Per-employee override: nullable `planning_mode` column on `organization_memberships` (022). Code-level defaults apply when a key is absent (no seed rows needed).

**Settings change audit (D-13-22):** `entity_type='org_settings'`, `entity_id` = the org id (audit_logs.entity_id is UUID NOT NULL — `[VERIFIED: migrations/017_audit_logs.up.sql:18]`), `action='settings-updated'`, payload = `{key, before, after}` — written in the same tx as the PUT.

**⚠️ Route collision awareness (NOT a blocker):** `GET/PUT /organizations/settings` are **literal** routes that coexist with the existing `GET/PUT /organizations/{id}/settings` (typed `organization_settings` — migration 000, lines 459-486; handler `[VERIFIED: internal/adapters/primary/http/organization.go:100-145]`). ServeMux precedence: "the most specific pattern takes precedence" — `[CITED: pkg.go.dev/net/http@go1.25.3]`; proven in-repo: `POST /organizations/invite` + `POST /organizations/invite-customer` literal routes already sit beside `GET /organizations/{id}` `[VERIFIED: cmd/server/main.go:207-214]`. Register the new literal routes in `cmd/server/main.go` — the wildcard keeps catching other ids. The two tables are deliberately separate (typed legacy vs generic key/value).

### Pattern 8: Origin fallback in the activity read path (D-13-32..34, FND-04/R4)
**What:** when an activity's stored origin refs are empty, the read path derives manager-assignment-shaped refs from the first direction record: `assigned_by = directed_by`, `assigned_to = directed_to` of the earliest `created_at` non-cancelled direction row for the activity (D-13-33). Never fills `proposed_by`/`reviewed_by`/`ticket_id`; never written back (D-13-34); no migration.

**Wiring (planning-sensitive):** the fallback lives in the **activity service** (`internal/core/services/activity/activity.go` — `GetByID` at line 70, `List` at line 54; returns `ActivityResponse` `[VERIFIED: internal/core/domain/activity/activity.go:90-97]` which embeds the origin fields at `[VERIFIED: activity.go:81-86]`). The activity service currently has no direction dependency — it must gain a small port (e.g. `FirstDirectionRefs(ctx, orgID, activityID) (*DirectionRefs, error)` on the direction repo) or a `direction.Service` dependency. **Consequence:** `NewService` signature + `cmd/server/main.go` wiring + `testdata` mocks (MockActivityRepo constructor) all change. Apply the fallback when the stored refs are empty — recommended predicate: `OriginType == nil` (no origin set at all; exact edge — OriginType set but refs empty — is Open Question 3).

## Anti-Patterns to Avoid
- **Storing derived states:** `done`/`lapsed`/`claimed`/claim-spectrum columns or nightly jobs — D-13-09/D-V: computed on read, never stored (ticket `DismissedNote` is the in-repo precedent of derived-on-read).
- **In-place edits of planned_date/est_hours:** D-13-04 — no `PUT /direction/{id}` mutating plan facts; replanning = new row + `supersedes_id` flip in one tx.
- **Pool-level-only claim Σ checks:** CR-01 TOCTOU — the Σ-claims ≤ WG est_hours check must re-run inside the mutator tx under the WG-row FOR UPDATE lock; pool-level checks are fast-fail UX only.
- **Fire-and-forget audit writes:** BE-012/BE-016 — direction/settings events follow the ticket in-tx pattern, never the detached-goroutine pattern.
- **A dedicated supersede endpoint:** D-13-08 — supersede is implicit via create.
- **WG rows with planned_date:** D-13-17 — the `wg_id IS NULL OR planned_date IS NULL` CHECK is the guard; "scheduling stays personal".
- **A `consumed`/`fully_claimed` stored state:** D-13-14/15 — the spectrum derives only when a budget is set; uncapped rows never derive `fully_claimed`.
- **Reading only `confirmed` absence windows:** D-13-29 — read `declared` + `confirmed` now; Phase 14 tightens.
- **Blocking writes on warnings:** D-13-03/28/30 — Σ over capacity, absences, validity are soft advisories; the backend never blocks on deadline/horizon (D-13-20/21).
- **Flagging validity-outside employees as uncovered:** D-13-31 — they get the `invalid` warning and are excluded from uncovered surfacing.
- **DB triggers for invariants:** no trigger precedent; invariants live in services + CHECKs.
- **Writing the fallback back to activities:** D-13-34 — stored refs stay authoritative; derivation is read-only.
- **Typed settings for new keys:** D-13-18 — extend `org_settings` key/value, do not add columns to `organization_settings`.
- **Teardown list not updated:** the shared `TeardownTestSchema` (`[VERIFIED: exported_test_helpers.go:77-122]`) must drop `direction`, `org_settings` in dependency order — cross-package pollution otherwise.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Transaction management (supersede, claim, cancel) | Hand-rolled commit/rollback scaffolding | pgx `pool.BeginTx` + `defer tx.Rollback` (precedents: `ticket_repository.go`, `coverage_repository.go:105-163`) | pgx handles isolation, deferred rollback safety, connection release `[CITED: pgx autodocs via Context7]` |
| Manager-stage permission (D-13-20) | Re-implement BE-014 precedence | Shared `routing.ResolveManagerStage` `[VERIFIED: routing.go:57-104]` | One routing rule for approvals, coverage, direction; re-implementation lets rules drift |
| Terminal-activity semantics (derived `done`) | New ad-hoc recursion | Phase 11 recursive CTE re-anchored at `activities.id` `[VERIFIED: ticket_repository.go:816-835]` | Same subtree walk + non-terminal entry predicate |
| State-machine re-validation | Pool-level-only checks | In-tx FOR UPDATE + re-check (CR-01 closure) | TOCTOU — check-and-act in one tx |
| Audit trail | Dual-write or fire-and-forget | In-tx `audit_logs` insert helper (insertTicketAudit / insertCoverageAudit pattern `[VERIFIED: coverage_repository.go:45]`) | The event stream must not be absent |
| Test database | Ad-hoc SQL fixtures per package | testcontainers `SetupPackageContainer` + seed helpers | Exists; one container per package via `sync.Once` |
| Migration runner | Custom DDL tooling | `cmd/migrate` + `readMigration`/`applyMigrations` cycle tests | ADR-BE-004; self-seed cycle pattern verified |
| JSON response envelope | Ad-hoc marshaling | `pkg/api.RespondWithJSON/RespondWithError` | House standard, verified everywhere |
| Authentication/claims | Token parsing in handlers | `middleware.Auth` + `GetRole`/`GetUserID`/`GetOrganizationID` | Exists; JWT-resolved org for settings (D-13-23) |
| Sentinel error mapping | Raw pg errors in handlers | Domain `errors.go` sentinels + `wrapPGError` + handler switch (ADR-BE-001) | Established across all handlers (coverage `writeError` is the newest template) |

**Key insight:** this phase's complexity is almost entirely in *existing* patterns — the ticket state machine (matrix + in-tx FOR UPDATE), the coverage Σ-in-cents arithmetic, the in-tx audit helper, the BE-014 routing gate, the 3VL CHECK house rule, the recursive terminal CTE. The risk is in **semantic definition** (claim-row mode, lapsed predicate, mode-gate exclusivity, origin-fallback predicate, settings key vocabulary) and in **wiring changes** (activity service gains a direction port; two new literal routes beside the wildcard), not in technology. No new library is needed.

## Runtime State Inventory

Not applicable — this is not a rename/refactor phase. All changes are additive (new `direction` + `org_settings` tables, one new nullable column on `organization_memberships`, new endpoints, new read-model). No stored data carries a renamed identifier; no secrets, env vars, or OS-registered state change. The only pre-existing data touched: `organization_memberships` gains a nullable `planning_mode` column (no backfill — NULL = fall back to org default, D-13-19).

## Common Pitfalls

### Pitfall 1: Claim over-subscription raced (TOCTOU on Σ)
**What goes wrong:** two members claim the last hours of a WG row simultaneously; both pool-level Σ checks pass; Σ claimed ends up > WG est_hours.
**Why it happens:** CR-01 root cause — check-then-act across the tx boundary.
**How to avoid:** the claim tx locks the WG direction row `FOR UPDATE` (`[VERIFIED: ticket_repository.go:370]` pattern), re-reads `est_hours` + Σ claims in-tx, and only then inserts. Concurrent-write integration test mirroring `ticket_repository_test.go:418-506`.
**Warning signs:** two claims for the same WG row both succeed in a concurrent test; Σ guard enforced only in the service, not the repo.

### Pitfall 2: CHECK constraints silently passing NULLs (XOR / queued-only / reason)
**What goes wrong:** a naive `CHECK (directed_to IS NOT NULL AND wg_id IS NULL)` is satisfied when `directed_to IS NULL AND wg_id IS NULL` (3VL) — a targetless row slips through; a cancelled row without a reason slips through.
**Why it happens:** PostgreSQL CHECK passes on TRUE or NULL — `[CITED: postgresql.org/docs/16/ddl-constraints.html]`; house rule from ADR-BE-016 (mirrors 015 `origin_type IS NULL OR (...)`, 019 source refs).
**How to avoid:** pin both sides of the XOR with explicit `IS [NOT] NULL` (see Pattern 1); mandatory-reason CHECK `(status <> 'cancelled' OR reason IS NOT NULL)` — this fails on NULL reason because `FALSE OR ...` is never NULL. New tables need no 3VL guard (no legacy rows); the guard only matters for ALTER-on-legacy (e.g. if a future status column is added to an existing table).
**Warning signs:** cycle test inserting a targetless row succeeding; cancelled row with NULL reason in the table.

### Pitfall 3: est_hours semantics drift (scheduled vs queued vs claim)
**What goes wrong:** scheduled rows created without est_hours; queued rows with `est_hours <= 0`; claim rows with est_hours > the WG budget; float64 equality on DECIMAL sums.
**Why it happens:** D-13-02 distinguishes required (scheduled) vs optional (queued); claims carry their own amount (D-13-13).
**How to avoid:** CHECKs `direction_scheduled_hours_check` (`planned_date IS NULL OR est_hours IS NOT NULL`) + `direction_est_hours_check` (`est_hours IS NULL OR est_hours > 0`) (Pattern 1); service rejects `est_hours <= 0` and absurd values before the repo call (D-13-03); claim Σ compares in cents (`math.Round(h * 100)` — the coverage precedent `[VERIFIED: internal/core/services/coverage/coverage.go:340-346]`).
**Warning signs:** a scheduled row with NULL est_hours in an integration test; claim splitting leaving Σ > budget due to float drift.

### Pitfall 4: Lifecycle bypass (direct UPDATE to superseded/cancelled, or activation of terminal rows)
**What goes wrong:** a transition endpoint allows `draft → cancelled` without reason, or `cancelled → active`, or supersede of an already-superseded row (chain rewrite).
**Why it happens:** the matrix exists in the domain but the repo UPDATE has no status precondition.
**How to avoid:** mirror the ticket pattern — domain matrix + service fast-fail + repo in-tx re-validation under FOR UPDATE with a status-precondition UPDATE backstop (CR-01 closure, `[VERIFIED: ticket_repository.go:360-378]`); the backstop SQL rejects the write when the locked status isn't the expected one. Supersede re-checks the target is `draft`|`active` in-tx.
**Warning signs:** concurrent activate+cancel both succeed; superseded row re-superseded; cancelled row re-activated.

### Pitfall 5: Activity service wiring breaks the origin fallback
**What goes wrong:** the fallback (D-13-32) requires the activity service to read direction rows; if the new port is wired only into the handler or forgotten, refs stay empty and FND-04's promise silently fails — or the constructor change breaks call sites and mocks.
**Why it happens:** `NewService` signature change ripples to `cmd/server/main.go` and `testdata` mocks.
**How to avoid:** plan the port + constructor change + mock update + wiring as one task; add a contract test (activity with empty origin refs + one direction row → response carries derived assigned_by/assigned_to; no direction rows → refs stay empty).
**Warning signs:** compile errors at mock call sites; `GET /activities` response refs empty in an integration test with a seeded direction row.

### Pitfall 6: Route collision between `/organizations/settings` and `/organizations/{id}/settings`
**What goes wrong:** a developer "fixes" the duplicate path by removing the wildcard route, or the literal route 404s/500s because it accidentally matches the wildcard handler.
**Why it happens:** both patterns match `/organizations/settings`; ServeMux resolves literal-beats-wildcard, so both can be registered — but a naive reader sees a conflict and "deduplicates".
**How to avoid:** keep both registrations (proven in-repo: `POST /organizations/invite` beside `GET /organizations/{id}` `[VERIFIED: cmd/server/main.go:207-214]`); handler integration test asserting `GET /organizations/settings` (no id) hits the new handler while `GET /organizations/<uuid>/settings` still returns the typed settings.
**Warning signs:** the wildcard handler returns 404 for `/organizations/settings` (registration order shadowed — impossible in Go 1.22+, but symptom-check anyway); or a panic on duplicate registration.

### Pitfall 7: Coverage read-model counts the wrong rows (capacity, statuses, scope)
**What goes wrong:** capacity counts cancelled/superseded rows as planned; uncovered surfacing includes fully-absent days or validity-outside employees; unit scope misses descendants; Σ est_hours includes superseded history rows.
**Why it happens:** the eligibility predicate (status IN draft|active, plus per-day planned rows only) and scope resolution (unit + descendants; WG members) must be exact.
**How to avoid:** planned Σ = `status IN ('draft','active')` AND `planned_date = day` AND `directed_to = employee` (superseded/cancelled rows excluded — D-13-08 history); capacity = `planning_daily_hours` (default 8) − absence hours (partial `hours` reduce, full windows zero — D-13-24); exclude fully-absent days (D-13-26) and validity-outside employees (D-13-31); unit scope via `GetDescendants` + `ListMembers` `[VERIFIED: internal/core/ports/unit_repository.go:16-30]` (descendants' members included — the "employees underneath" wording of D-13-25).
**Warning signs:** a superseded row's hours counted in coverage; an away-day surfaced as uncovered with gap = capacity; unit scope missing sub-unit employees.

### Pitfall 8: Teardown list and migration numbering not updated
**What goes wrong:** `TeardownTestSchema` lacks `direction`/`org_settings` — cross-package pollution; migrations numbered 021+ incorrectly (ADR-BE-004 sequential; 007 gap frozen).
**Why it happens:** adding tables without touching the shared teardown.
**How to avoid:** add `direction`, `org_settings` to the teardown list before `working_groups`/`activities`/`organization_memberships` (dependency order); number new migrations 021/022; cycle tests self-seed pre-state inline (verified pattern).
**Warning signs:** repository tests in later packages failing on leftover direction rows.

### Pitfall 9: WG-scope check too loose or too strict (D-13-17)
**What goes wrong:** a WG row directed at an activity outside the WG's anchored subtree (cross-WG or cross-org smuggling), or the check rejects legitimate descendant activities.
**Why it happens:** "activity must be within the WG's scope (same-org, reachable via WG subtree)" needs a concrete predicate; the WG anchors to an activity via `SubprojectID` `[VERIFIED: internal/core/domain/working_group/working_group.go:23]`.
**How to avoid:** recommend the predicate: the WG's anchored activity is `wg.SubprojectID`; the directed activity is in scope iff same-org AND (activity id == anchor OR the anchor appears in `activityRepo.GetAncestry(activityID)` — i.e. the activity is the anchor or a descendant). Service-level check (WG row creation + claims); pool-level is fine here (no Σ invariant) but same-org must be re-checked against the locked row where relevant.
**Warning signs:** a WG direction row referencing an activity in another WG's subtree accepted; a descendant activity rejected.

### Pitfall 10: Absence-window double-counting in capacity
**What goes wrong:** a window spanning multiple days subtracts its `hours` once instead of per-day; overlapping windows both subtract; partial `hours` semantics vs full-day zeroing conflated.
**Why it happens:** `availability_windows` is a (starts_on, ends_on) range with optional `hours` `[VERIFIED: migrations/012_staffing_schema.up.sql:15-30]` — capacity math must expand per-day.
**How to avoid:** per-day computation: for each day in the period, find windows overlapping the day; full window (hours IS NULL) → capacity 0 (`away`); partial window → capacity −= hours (`partial`); the day's capacity can floor at 0. Warning types returned per (employee, day).
**Warning signs:** a 10-day holiday subtracting its (NULL) hours as if partial; capacity going negative and being surfaced as `over-capacity` instead of `away`.

## Code Examples

Verified patterns from official sources + codebase precedents (read this session):

### Common Operation 1: Supersede-on-create tx (repo shape)
```go
// Source shape: ticket_repository.go Dismiss/UpdateState (BeginTx + FOR UPDATE
// + in-tx audit) + coverage_repository.go ReplaceAllocations — read this session.
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil { return nil, fmt.Errorf("begin direction create: %w", err) }
defer func() { _ = tx.Rollback(ctx) }() // safe even after Commit

// 1. Lock the superseded target (CR-01) and re-check it is supersedable:
var targetStatus string
err = tx.QueryRow(ctx,
    `SELECT status FROM direction WHERE id = $1 AND org_id = $2 FOR UPDATE`,
    req.SupersedesID, orgID).Scan(&targetStatus)
if errors.Is(err, pgx.ErrNoRows) { return nil, direction.ErrDirectionNotFound }
if targetStatus != "draft" && targetStatus != "active" {
    return nil, direction.ErrInvalidTransition // in-tx authoritative check
}
// 2. INSERT the new row (mode/est_hours/priority/due_date per D-13-01..06)
// 3. UPDATE direction SET status = 'superseded' WHERE id = $1 (status precondition backstop)
// 4. insertDirectionAudit(ctx, tx, ...) for 'created' + 'superseded' (BE-012, in-tx)
if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit direction create: %w", err) }
```

### Common Operation 2: Claim tx with the Σ guard (D-13-13, CR-01)
```go
// Source shape: coverage ReplaceAllocations (FOR UPDATE + in-tx checks) —
// [VERIFIED: internal/adapters/secondary/postgres/coverage_repository.go:105-163].
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil { return nil, fmt.Errorf("begin claim: %w", err) }
defer func() { _ = tx.Rollback(ctx) }()

// Lock the WG row; the locked values are the commit-order truth:
var wgEstHours *float64
var wgStatus string
err = tx.QueryRow(ctx,
    `SELECT est_hours, status FROM direction
      WHERE id = $1 AND org_id = $2 AND wg_id IS NOT NULL FOR UPDATE`,
    wgRowID, orgID).Scan(&wgEstHours, &wgStatus)
if errors.Is(err, pgx.ErrNoRows) { return nil, direction.ErrDirectionNotFound }
// in-tx re-checks: status supersedable; claimant is a WG member;
if wgEstHours != nil {
    // Σ in cents (float artifacts — coverage precedent):
    var claimed int64
    tx.QueryRow(ctx,
        `SELECT COALESCE(SUM(est_hours), 0) FROM direction
          WHERE origin_direction_id = $1 AND status IN ('draft','active')`,
        wgRowID).Scan(&claimed)
    claimCents := int64(math.Round(claimEstHours * 100))
    if claimed+claimCents > int64(math.Round(*wgEstHours*100)) {
        return nil, direction.ErrClaimOverBudget // 409
    }
}
// INSERT claim row (directed_by = WG creator, directed_to = claimant,
//   origin_direction_id = wgRowID, est_hours = claimEstHours, status = 'draft')
// + insertDirectionAudit(ctx, tx, ...) 'claimed' — COMMIT.
```

### Common Operation 3: Derived `done` via the re-anchored terminal CTE
```sql
-- Source shape: ticket_repository.go:816-835 — [VERIFIED] quote of the
-- non-terminal predicate: status IN ('draft','submitted','pending_manager',
-- 'pending_finance') AND is_deleted = false. Re-anchor at activities.id:
WITH RECURSIVE subtree AS (
    SELECT id FROM activities WHERE id = $1          -- the direction's activity
    UNION ALL
    SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
)
SELECT NOT EXISTS (
    SELECT 1 FROM time_entries te
    WHERE te.is_deleted = false
      AND te.status IN ('draft','submitted','pending_manager','pending_finance')
      AND te.activity_id IN (SELECT id FROM subtree)
) AS terminal;
-- terminal = true → derived state 'done' (D-13-09)
```

### Common Operation 4: Warning overlay row (D-13-30/31, UI-SPEC contract)
```go
// Contract pinned in 13-UI-SPEC: warnings are {type, message}; message is
// pre-rendered server-side, rendered verbatim by Phase 19.
type Warning struct {
    Type    string `json:"type"`    // away | partial | over-capacity | invalid
    Message string `json:"message"` // "Away 10–21 Aug", "Partial 14 Aug", ...
}
// away:     full absence window covering the day        → "Away {date-range-or-day}"
// partial:  partial-day permit                          → "Partial {day}"
// over-cap: Σ est_hours > capacity                      → "Over capacity {day}"
// invalid:  outside valid_from/valid_until              → "Outside validity period"
```

### Common Operation 5: Settings PUT with before/after audit (D-13-22/23)
```go
// GET /organizations/settings → {"planning_daily_hours": 8, ...} (org from claims)
// PUT /organizations/settings {"planning_daily_hours": 7.5} — manager+ only.
// Known-key validation in the service: unknown key → 400 (D-13-18).
// The repo's upsert tx writes the audit row with before/after:
//   entity_type='org_settings', entity_id=<orgID>, action='settings-updated',
//   payload={"key":"planning_daily_hours","before":8,"after":7.5}
// (audit_logs.entity_id is UUID NOT NULL — [VERIFIED: 017:18] — org id is the address)
```

### Common Operation 6: Migration cycle-test skeleton (021/022)
```go
// Source shape: ontology_extension_migrations_test.go TestMigration014..020 —
// self-seed pattern (read this session's Phase 12 analog).
func TestMigration021_DirectionRows_UpDownUpCycle(t *testing.T) {
    pool := TestPool(t)
    t.Cleanup(func() { TeardownTestSchema(t, pool) })
    up021 := readMigration(t, "021_direction_rows.up.sql")
    down021 := readMigration(t, "021_direction_rows.down.sql")
    applyMigrations(t, pool, true, "022_org_settings.up.sql")
    orgID := seedOrg(t, pool, time.Now())
    userID := seedUser(t, pool, time.Now())
    // ... seed activity + working group
    // --- UP --- insert row with planned_date + est_hours; assert scheduled OK,
    //           targetless row rejected (XOR CHECK), WG row with planned_date
    //           rejected (queued-only CHECK), est_hours <= 0 rejected
    // --- DOWN --- drop table; --- UP --- re-apply green
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Proto-direction via `activities.assigned_to` (v0.1) | First-class `direction` plane with per-day rows + derived modes (D-R/D-W/D-AA) | This phase | The plan becomes mutable-as-a-chain, never rewriting the fact (D-13-04) |
| Origin refs stored only (P-013, Phase 11) | Stored refs + read-path fallback to the first direction record (R4 resolution) | This phase | Pre-direction activities keep honest stored refs; direction becomes the derivation source (D-13-32..34) |
| Two-plane ontology (direction deferred) | Three-plane ontology: direction (plan) → facts → coverage (label) (P-015) | This phase | P-004 Today revision, scheduler surfaces (Phase 19) build against the complete model |
| Manual replanning (rewrite the row) | Supersede chain — new row + flip in one tx (D-13-08) | This phase | Plan history is audit-trail-preserved; "the plan never rewrites the fact" |
| Single-claim lean (D-T draft) | Hours-based split claims with Σ-consumption (D-13-13) | This phase (user override) | A WG row supports partial claims; unclaim returns hours automatically (Σ-derived) |
| Typed org settings columns | Generic key/value JSONB `org_settings` (D-13-18) | This phase (user choice) | New policy keys are data, not migrations; vocabulary validated in code |

**Deprecated/outdated:**
- **`assigned_to` as the only direction signal:** superseded by the direction plane — the column stays as the stored origin ref, and the fallback derives from direction when empty (FND-04).
- **"Consumed" as a stored claim state:** D-13-14 — the spectrum derives on read; a stored consumed flag would drift from Σ.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Audit vocabulary: `entity_type='direction'` with actions `created`/`activated`/`cancelled`/`superseded`/`claimed`/`unclaimed`; `entity_type='org_settings'` with `settings-updated` + before/after payload; `entity_id` = org id for settings (UUID NOT NULL constraint) | Pattern 2/7 | `audit_logs` columns are free-form (verified 017) — any convention works, but Phase 19 history surfaces read by entity; pin in the BE ADR or history reads must guess |
| A2 | `est_hours` = `DECIMAL(8,2)` mirroring `time_entries.hours` (verified quote 000:278) | Pattern 1 | Discretion area — a different scale is a pure DDL tweak, but plan-time consistency with entries matters |
| A3 | `lapsed` predicate = past `planned_date`/`due_date` AND no non-deleted time entries (any status) on the activity subtree | Pattern 4 / OQ2 | If "logged hours" means approved-only or non-terminal-only, the SQL changes (one query); must be pinned for consistent read-model output |
| A4 | Origin-fallback trigger = `origin_type IS NULL` (no origin set at all); applies to activity `GetByID` + `List` via service enrichment; a new small direction-ref port is added to the activity service (constructor + wiring + mocks change) | Pattern 8 / OQ3 | If refs-empty-but-type-set rows must also derive, the predicate widens; if the fallback lives elsewhere (handler/direction service), the activity service stays untouched but the response assembly changes |
| A5 | WG-scope predicate = directed activity same-org AND (activity == WG's anchored `SubprojectID` OR the anchor is in the activity's ancestry) | Pitfall 9 / OQ8 | If WG scope means "any activity the WG manager manages" (routing-set based), the predicate differs — planner call |
| A6 | Unit scope in the coverage read-model = the unit + its descendants' members (D-13-25 "aggregate employees underneath") | Pattern 7 / OQ6 | If only direct members count, the query drops `GetDescendants` — a read-model difference Phase 19 surfaces |
| A7 | Settings key vocabulary = `planning_daily_hours` (number, default 8), `planning_deadline` (date), `planning_horizon` (`day|week|month`), `planning_mode` (`manager_planned|self_planned`); membership override column named `planning_mode` | Pattern 7 | Keys are code-validated (D-13-18) — renaming is a code change; the UI-SPEC pins only `planning_daily_hours` + the mode vocabulary, so the rest is this phase's pin |
| A8 | Claim rows are created in `draft` status, queued mode (`planned_date` NULL), copying `priority`/`due_date` from the WG row; claimant then schedules via the normal supersede chain | Pattern 1/5 / OQ4 | If claim rows should be `active`-on-create or scheduled directly, create flow and matrix edges change |
| A9 | Mode gate (D-13-20) strict reading: manager-planned → only managers create rows for that employee; self-planned → only the employee creates their own rows; queued self-rows follow the same mode | Pattern 3 / OQ7 | If self-direction is always allowed regardless of mode (D-S first-class), the gate loosens — additive code change |
| A10 | WG-direction creation gate = manager within BE-014 reach of the anchored activity (same routing resolution as entries) | Pattern 3 | If any org manager may create WG rows, the routing call is dropped — permission difference |

## Open Questions (RESOLVED)

All nine questions were resolved during planning; every recommendation is adopted verbatim, pinned in ADR-BE-018 (13-02) and compiled into the implementing plans listed per question below.

1. **Draft → active activation mechanics (discretion area, D-13-07)** (RESOLVED — adopted by 13-05 T2, 13-07 T2, 13-08 T3)
   - What we know: "created as draft, activated explicitly or via first plan action".
   - What's unclear: explicit `POST /direction/{id}/activate` endpoint vs auto-activation on first create-with-planned_date.
   - Recommendation: **explicit activate endpoint** — one audit row per transition, symmetric with cancel, and the matrix stays simple (`draft → active → cancelled`; create-with-planned_date does NOT auto-activate). Planner confirms; either is within the locked decision.

2. **`lapsed` "without logged hours" predicate (D-13-09)** (RESOLVED — adopted by 13-02 T2 [ADR-BE-018 A3 pin], 13-06 T1)
   - What we know: lapsed = past planned_date/due_date without logged hours; the terminal CTE defines non-terminal entries as `status IN ('draft','submitted','pending_manager','pending_finance')` + `is_deleted = false` (verified).
   - What's unclear: does "logged hours" mean *any* non-deleted entry (even draft), or only submitted+approved?
   - Recommendation: **no non-deleted entries at all** on the activity subtree (any status) — a plan "lapsed" when nothing was ever logged; a draft entry indicates work started. Planner pins the SQL; see A3.

3. **Origin fallback trigger and shape (D-13-32..34)** (RESOLVED — adopted by 13-02 T2 [ADR-BE-018 A4 pin], 13-08 T1)
   - What we know: fallback fills `assigned_by`/`assigned_to` from the first non-cancelled direction row when stored refs are empty; never other origin shapes.
   - What's unclear: (a) trigger predicate — `origin_type IS NULL` only, or any empty ref set; (b) whether the response's `origin_type` should report `manager_assignment` when refs are derived (it stays NULL in storage).
   - Recommendation: trigger on `OriginType == nil`; **report the derived refs without flipping origin_type** (the stored value stays authoritative — D-13-34 spirit); document in the BE ADR so Phase 19 surfaces don't re-derive.

4. **Claim row mode/status semantics (D-13-11..16)** (RESOLVED — adopted by 13-02 T2 [ADR-BE-018 A8 pin], 13-05 T3)
   - What we know: claim = user-targeted row via `origin_direction_id`; Σ ≤ WG budget under lock; unclaim = cancel with reason.
   - What's unclear: is the claim row `draft` or `active`? Queued or scheduled? Does it copy priority/due_date?
   - Recommendation: **draft + queued, copying priority/due_date** — the claim lands in the claimant's queue (scheduling stays personal, D-T), and the claimant schedules it through the normal supersede chain. See A8.

5. **Coverage read-model period + envelope (discretion area, D-13-25)** (RESOLVED — adopted by 13-06 T2, 13-08 T3)
   - What we know: `GET /direction/coverage?scope=employee|unit|wg&scope_id=&period=`; rows `(employee, date, capacity, planned, gap)` + period totals; derived states included (D-13-27).
   - What's unclear: `period` as single string ("2026-08") vs start/end pair; response grouping (per-employee map vs flat rows); pagination.
   - Recommendation: mirror the coverage close's two-date shape (`period_start`+`period_end`, "2006-01-02" strings parsed at the boundary); return grouped per-employee rows + period totals + warnings; no pagination in v0.2 (org-scoped, period-bounded).

6. **Unit scope employee resolution (D-13-25)** (RESOLVED — adopted by 13-07 T3 [A6 pin])
   - What we know: "aggregation differs only in which-employees resolution; unit/WG scopes aggregate employees underneath".
   - What's unclear: does unit scope include members of descendant units? Do WG members outside the unit count?
   - Recommendation: unit scope = members of the unit + all descendant units (`GetDescendants` + `ListMembers`, verified ports); WG scope = `wgRepo.ListMembers` of that WG. See A6.

7. **Mode gate exclusivity (D-13-20)** (RESOLVED — adopted by 13-02 T2 [ADR-BE-018 A9 pin], 13-07 T1)
   - What we know: mode gates who may create scheduled rows for whom; self-direction is first-class with no approval (D-S).
   - What's unclear: in manager-planned mode, may the employee still self-direct? In self-planned mode, may a manager still create a scheduled row?
   - Recommendation: strict reading — manager-planned → manager creates via BE-014 reach; self-planned → only the employee creates own rows (see A9). If a hybrid is wanted (self-direction always allowed), it's an additive loosening.

8. **WG-direction row creation permission (D-13-17, D-13-20)** (RESOLVED — adopted by 13-07 T1 [A10 pin])
   - What we know: activity must be within the WG's scope; managers direct within subtree/WG reach (BE-014 machinery).
   - What's unclear: who may create a WG row — the WG manager/delegates only, or any manager within BE-014 reach of the anchored activity?
   - Recommendation: reuse `routing.ResolveManagerStage` on the anchored activity (WG manager/delegates or role-gated manager) — same resolution as entry approval, no new permission machinery. See A5/A10.

9. **Settings endpoint shape: one service or fold into direction (D-13-18..23)** (RESOLVED — adopted by 13-04 orgsettings vertical, 13-03 port/domain)
   - What we know: generic `GET/PUT /organizations/settings`, key/value, manager+ gated, audit-logged; "configurations are getting bigger and bigger".
   - What's unclear: separate `orgsettings` package vs extending the existing organization service (which owns the typed `/organizations/{id}/settings`).
   - Recommendation: **separate small `orgsettings` package** (domain + port + service + repo + handler) — keeps the typed legacy surface untouched and gives future settings keys a home independent of direction; wiring in `cmd/server/main.go` beside the existing org routes.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend build/tests | ✓ | 1.26.1 (matches go.mod) | — |
| Docker | testcontainers integration tests (postgres:16-alpine) | ✓ | 29.4.0 | Tests skip when Docker absent (`t.Skip` in `SetupPackageContainer`) |
| PostgreSQL | Dev runtime (docker-compose service); tests use containers | ✓ (docker-compose) | postgres:15 via compose / 16-alpine in tests | Testcontainers covers tests; dev DB via `docker-compose up` |
| Node | Not needed this phase (backend-only) | ✓ | v22.23.1 | — |
| `DATABASE_URL` | `go run ./cmd/server`, `cmd/migrate` | ✓ | default `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` | Set env var explicitly |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — all tooling present.

## Validation Architecture

> `workflow.nyquist_validation` absent from `.planning/config.json` (verified — only `workflow._auto_chain_active` present) → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go 1.26.1 `go test` + testify v1.11.1 + testcontainers-go v0.42.0 (postgres:16-alpine) |
| Config file | none — scaffolding built in-plan; patterns from `.planning/codebase/TESTING.md` |
| Quick run command | `go test ./internal/core/services/direction/ ./internal/core/services/orgsettings/ ./internal/core/services/activity/ -count=1` |
| Full suite command | `make test` (go test -v ./...) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DIR-01 | Create scheduled + queued rows; mode derived from planned_date; multiple rows share a day; est_hours required on scheduled / optional queued; ≤ 0 rejected | unit + integration (CHECK) | `go test ./internal/core/services/direction/ -run TestService_Create -count=1` | ❌ Wave 0 |
| DIR-01 | XOR target CHECK + queued-only WG CHECK + est_hours CHECK enforced by DB (cycle test) | integration (cycle) | `go test ./internal/adapters/secondary/postgres/ -run TestMigration021 -count=1` | ❌ Wave 0 |
| DIR-01 | Self-direction allowed with no approval; manager create gated via routing (approver set / role-gated); non-manager rejected | unit + handler integration | `go test ./internal/adapters/primary/http/ -run TestDirectionHandler -count=1` | ❌ Wave 0 |
| DIR-02 | Matrix: draft→active→cancelled with reason; superseded only via create-with-supersedes_id; terminal states immutable | unit (matrix) + integration (in-tx) | `go test ./internal/core/services/direction/ ./internal/adapters/secondary/postgres/ -count=1` | ❌ Wave 0 |
| DIR-02 | Supersede chain: new row flips target to superseded in one tx; audit rows written in-tx; chain readable as history | integration | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionRepository_Supersede -count=1` | ❌ Wave 0 |
| DIR-02 | Derived done (terminal CTE), lapsed (past + no entries), claimed spectrum computed on read, never stored | integration (repo/read-model) | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionRepository_DerivedStates -count=1` | ❌ Wave 0 |
| DIR-03 | Claim creates user-targeted row with origin_direction_id + attribution; non-member rejected; Σ ≤ WG est_hours; uncapped when no budget | unit + integration | `go test ./internal/core/services/direction/ -run TestService_Claim -count=1` | ❌ Wave 0 |
| DIR-03 | Concurrent claims: FOR UPDATE lock serializes; over-subscription never commits (CR-01-style) | integration (concurrent, mirrors ticket_repository_test.go:418-506) | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionClaim_Concurrent -count=1` | ❌ Wave 0 |
| DIR-03 | Unclaim = cancel claim row with reason; hours return (Σ-derived) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionRepository_Unclaim -count=1` | ❌ Wave 0 |
| DIR-04 | org_settings GET/PUT key/value; unknown key 400; manager+ gate; membership override falls back to org default; settings change audit-logged with before/after | unit + handler integration | `go test ./internal/core/services/orgsettings/ ./internal/adapters/primary/http/ -count=1` | ❌ Wave 0 |
| DIR-05 | Warnings: away / partial / over-capacity / invalid; declared+confirmed windows read; never blocks writes | integration (read-model) | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionRepository_Warnings -count=1` | ❌ Wave 0 |
| DIR-06 | Coverage: planned vs capacity per employee/period; uncovered days surfaced; unit + WG scopes aggregate; fully-absent + invalid employees excluded | integration (read-model) | `go test ./internal/adapters/secondary/postgres/ -run TestDirectionRepository_Coverage -count=1` | ❌ Wave 0 |
| FND-04 | Origin fallback: empty refs + first direction row → derived assigned_by/assigned_to; no rows → empty; never written back | integration (activity read path) | `go test ./internal/core/services/activity/ ./internal/adapters/primary/http/ -count=1` | ❌ Wave 0 |
| Migrations | 021/022 up → down → up cycles + CHECK assertions (XOR, queued-only WG, est_hours, cancel reason) | integration (cycle) | `go test ./internal/adapters/secondary/postgres/ -run TestMigration021 -count=1` (+022) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/services/direction/ ./internal/core/services/orgsettings/ ./internal/adapters/secondary/postgres/ -count=1`
- **Per wave merge:** `make test` (full suite)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/core/services/direction/direction_test.go` — service unit tests (create/matrix/claim/derived states/permission) with `testdata.MockDirectionRepo`
- [ ] `internal/core/services/orgsettings/orgsettings_test.go` — settings unit tests (known-key validation, defaults, gate)
- [ ] `internal/core/services/testdata/` — MockDirectionRepo + MockOrgSettingsRepo + MockActivityRepo constructor update (fallback port)
- [ ] `internal/adapters/secondary/postgres/direction_repository_test.go` — repo integration (supersede tx, claim lock, derived states, warnings, coverage)
- [ ] `internal/adapters/secondary/postgres/org_settings_repository_test.go` — settings upsert + audit integration
- [ ] `internal/adapters/secondary/postgres/direction_ontology_migrations_test.go` — cycle tests TestMigration021..022 (self-seed)
- [ ] `internal/adapters/primary/http/direction_handler_test.go` — handler integration (permission matrix, envelope, sentinel mapping, warnings in create response)
- [ ] `internal/adapters/primary/http/org_settings_handler_test.go` — literal-route coexistence with `/organizations/{id}/settings` (Pitfall 6)
- [ ] `internal/adapters/secondary/postgres/exported_test_helpers.go` — teardown list += direction, org_settings (Pitfall 8)

## Security Domain

> `security_enforcement` absent from `.planning/config.json` → treated as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (unchanged) | Existing `middleware.Auth` + JWT cookie flow — no new auth surface |
| V3 Session Management | no (unchanged) | Existing refresh rotation — untouched |
| V4 Access Control | yes | D-13-20 mode gate: manager writes via shared `routing.ResolveManagerStage` (ApproverIDs / RoleGated — `[VERIFIED: routing.go:41-45]`); self-direction = actor is the target; settings PUT manager+-gated (D-13-23); WG claims membership-checked (D-13-12); claims only by members |
| V5 Input Validation | yes | est_hours > 0 + absurd-value rejection (D-13-03); XOR/queued-only/cancel-reason CHECKs (Pattern 1); same-org refs (activity/WG); known-key settings validation → 400 on unknown key (D-13-18); claim Σ invariant re-validated in-tx (CR-01) |
| V6 Cryptography | no | No new crypto — none introduced by a plan/queue plane |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Claim over-subscription race (concurrent claims on a WG row) | Tampering | In-tx re-validation under `SELECT ... FOR UPDATE` on the WG row (CR-01 closure; precedent `[VERIFIED: ticket_repository.go:370]`) |
| Cross-org reference injection (activity_id / wg_id / directed_to from another org) | Spoofing / Tampering | Service-level same-org existence checks on every ref (Phase 11 D-02 pattern); org_id column on the row; WG-scope predicate (A5/OQ8) |
| Mode-gate bypass (employee creating rows for others; non-manager creating manager-planned rows) | Elevation of privilege | D-13-20 gate: self only self-rows; manager rows via routing resolution; settings manager+; claim membership check (D-13-12) |
| Lifecycle tampering (direct status writes, superseding terminal rows, activation of cancelled) | Tampering | Domain matrix + repo in-tx FOR UPDATE re-validation + status-precondition UPDATE backstop (ticket pattern, CR-01) |
| Audit trail loss (transitions/settings changes unlogged) | Repudiation | Synchronous in-tx audit rows (BE-012/016) — never fire-and-forget; rollback on audit failure; before/after payload for settings (D-13-22) |
| Unknown settings keys / malformed JSONB values | Tampering | Code-enforced per-key validation → 400 (D-13-18); no CHECK on JSONB (impossible) — validation at the service boundary is the only gate |

## Sources

### Primary (HIGH confidence)
- Codebase (read this session): `migrations/000_full_schema.up.sql` (time_entries hours quote :278, organization_settings :459-486), `012_staffing_schema.up.sql` (availability_windows + validity + roles), `015_activity_origins.up.sql` (XOR CHECK pattern), `017_audit_logs.up.sql` (audit shape); `internal/core/domain/ticket/ticket.go` (matrix), `internal/core/domain/activity/activity.go` (origin fields, ActivityResponse), `internal/core/domain/working_group/working_group.go` (WG anchor), `internal/core/services/ticket/ticket.go`, `internal/core/services/coverage/coverage.go` (Σ cents, gates, audit), `internal/core/services/activity/activity.go` (GetByID/List), `internal/core/services/routing/routing.go` (ResolveManagerStage), `internal/adapters/secondary/postgres/ticket_repository.go` (FOR UPDATE, terminal CTE), `coverage_repository.go` (in-tx audit helper, claim-lock analog), `exported_test_helpers.go` (teardown + seeds), `internal/core/ports/working_group_repository.go`, `unit_repository.go`, `activity_repository.go`, `internal/middleware/middleware.go` (claims), `internal/adapters/primary/http/coverage_handler.go` (handler template), `organization.go` (typed settings), `cmd/server/main.go` (routes + wiring), `go.mod`, `internal/models/models.go` (roles), `.planning/config.json`
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` (Part 14 D-R..D-AA, Part 15 R4 resolution) — record of truth
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` (D-1/D-2/D-3 window + validity semantics)
- `.planning/phases/13.../13-CONTEXT.md` (D-13-01..34 locked) and `13-UI-SPEC.md` (warning contract, status vocabularies, settings keys pinned for Phase 19)
- `[VERIFIED: pkg.go.dev/net/http@go1.25.3 via Context7]` — ServeMux "most specific pattern takes precedence" — cross-checked against `cmd/server/main.go:207-214` literal/wildcard coexistence
- `[VERIFIED: pgx autodocs via Context7]` — BeginTx / pool Tx release semantics — cross-checked against `coverage_repository.go` and `ticket_repository.go`

### Secondary (MEDIUM confidence)
- `[CITED: postgresql.org/docs/16/ddl-constraints.html]` — CHECK passes on TRUE or NULL (tier MEDIUM per classify-confidence seam; elevated by 015/019 codebase cross-check)
- `[CITED: .planning/REQUIREMENTS.md DIR-01..06]` — requirement text verbatim
- `[CITED: .planning/ROADMAP.md Phase 13 entry]` — 8 success criteria

### Tertiary (LOW confidence)
- None — no training-memory-only claims are load-bearing; every schema/pattern claim was verified against files read this session or official docs. Assumed items (A1..A10) are planner decisions, not facts.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified in `go.mod`; zero new dependencies
- Architecture: HIGH — every pattern verified against a codebase precedent read this session (ticket matrix + FOR UPDATE, in-tx audit, Σ cents, routing gate, terminal CTE, XOR CHECK)
- Pitfalls: HIGH — derived from Phase 11/12 execution history (CR-01, teardown, 3VL) + schema shapes verified this session; open semantic points flagged in Assumptions Log / Open Questions at MEDIUM

**Research date:** 2026-08-08
**Valid until:** 2026-09-07 (stable stack; re-check go.mod only if new deps are proposed)
