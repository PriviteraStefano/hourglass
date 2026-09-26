# Phase 14: Availability Backend — Absences + Capacity - Research

**Researched:** 2026-08-08
**Domain:** Go backend + PostgreSQL — absence lifecycle over the shipped `availability_windows` schema, work-schedule model, derived capacity read-model
**Confidence:** HIGH (all findings verified in-repo this session; no new external packages)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Confirmation routing & approvers
- **D-14-01:** **Unit manager only confirms** — holiday/permit/unavailable are confirmable; the confirm/reject authority is the employee's unit manager resolved via the unit-tree upward walk (`routing.ResolveUnitManager`, BE-014). **ADR-P-008 D-1a's second line (one WG manager) is dropped** — no WG-manager confirmation. This is an ADR revision. — **Reversibility:** costly — D-1a routing is referenced in ADR-P-008 and the payroll/export story; reintroducing a WG confirm line later needs a schema/status-model change.
- **D-14-02:** **Medical is record-only** — no approval step; medical windows are confirmed immediately at declare (notification, not a request, per ADR D-1a).
- **D-14-03:** **HR curates, never confirms** — HR may create windows for anyone, correct medical windows, and set/attach certificate data; confirmation/rejection is exclusively the unit manager's call (ADR-P-008 D-4 "never an approver"). AVAIL-02's "Manager/HR can confirm" reads as manager-confirms + HR-curates.
- **D-14-04:** **Self-confirmation allowed** — if the employee IS the resolved unit manager, they confirm their own window; no upward walk for self-absences (deliberate deviation from entry-approval's self-approval prevention; the manager declaring their own absence knows their own calendar).
- **D-14-05:** **certificate_ref is required at declare for medical** — the employee provides the INPS protocol number; HR may correct it later as curator.

#### Medical certificate documents (ADR D-5 boundary change)
- **D-14-06:** **The certificate document (image/pdf) IS stored in Hourglass** — required at declare alongside the ref. The employee uploads the document with the medical window. — **Reversibility:** one-way — ADR-P-008 D-5 explicitly rejected medical document storage citing GDPR special-category data; this phase revises that boundary. Undoing means deleting the attachment table + storage path.
- **D-14-07:** **DB-backed attachment table** — documents stored in PostgreSQL (new table: org_id, entity_type='availability_window', entity_id, content_type, size, storage/bytes), not file storage or object store (nothing in the stack; no S3/minio in compose). Served back only to `hr` + the employee's unit manager (ADR D-1a visibility scope).

#### Absence lifecycle & schema gap
- **D-14-08:** **Status vocabulary extended** — migration adds `rejected` + `withdrawn` to the `status` CHECK (schema today: `declared`, `confirmed`) and a `rejection_reason` column. Full vocabulary: `declared → confirmed | rejected | withdrawn`; `rejected` and `withdrawn` are terminal; `medical` auto-confirms at declare.
- **D-14-09:** **Rejected = terminal** — no editing, no re-submission on the same row; the employee creates a NEW window if circumstances changed. Reject requires a reason (mirrors entry-reject-with-reason pattern).
- **D-14-10:** **Withdraw = declared-only, status not delete** — an employee changes an absence window by withdrawing the first one and creating a new one that goes through the normal declare → confirm flow. Withdrawn windows are terminal rows (`status='withdrawn'`, audit-logged), NOT hard-deleted — history stays intact per BE-012.
- **D-14-11:** **Nobody edits windows** — no in-place editing of dates/kind/hours/note by employees or managers (change = withdraw + redeclare). The single carve-out: **HR may edit medical windows directly** (dates + certificate_ref) since medical is record-only and withdrawal never applies to it — HR edits are the medical correction path.
- **D-14-12:** **Full in-tx audit trail** — every window event (declare, withdraw, confirm, reject, HR medical edit, certificate attach) writes an `audit_logs` row (`entity_type='availability_window'`, payload before/after), synchronous inside the mutator transaction (BE-016 house style, mirroring tickets/coverage/direction).

#### Overlap rejection semantics (AVAIL-01)
- **D-14-13:** **Active-only, kind-blind overlap** — the overlap guard counts only `declared` + `confirmed` windows of the same user; `withdrawn`/`rejected` are excluded. Kind-blind: a declared holiday overlapping a confirmed medical still rejects.
- **D-14-14:** **Date-range only** — overlap = date-range intersection; the schema stores `hours` but no time-of-day, so same-day partial windows count as overlap (no hours-aware comparison possible).
- **D-14-15:** **Service in-tx check** — the overlap guard runs inside the declare transaction (CR-01 pattern: SELECT the user's active overlapping windows, reject if any, under row lock); no DB EXCLUDE constraint, no btree_gist extension (first-extension-free house style preserved).

#### Work schedules & capacity basis
- **D-14-16:** **Work-schedule model: contract_types + membership override** — new `contract_types` table (org_id, name, cadence `week`|`month`, hours_per_period, default day-hours matrix) as the reusable template; `organization_memberships` gains `contract_type_id` + per-employee day-hours override rows (weekday → hours, e.g. Mon 6h instead of the type's 8h). The type answers "reuse a contract type across employees"; the override answers "different days or different hours on the same day for this user" — the user's explicitly named most-complex case.
- **D-14-17:** **Monthly cadence with dynamic days = derived per-day** — monthly hours ÷ working days in the month (working days = the fixed weekday list when fixed, else calendar workdays). No per-day hours stored for dynamic patterns.
- **D-14-18:** **Fallback chain: override → contract_type → org default → 8h × Mon–Fri** — capacity never breaks when nothing is configured; the resolution level is documented in the response. The org default schedule is a `contract_type` flagged as default (planner discretion: flag column vs org_settings reference).
- **D-14-19:** **Workload = Σ submitted+approved entries on the activity subtree** — recursive CTE (Phase 11 terminal-activity semantics), grouped per employee. Only `time` entries with those statuses count.

#### Capacity read-model
- **D-14-20:** **One endpoint, scope params** — `GET /availability/capacity?scope=activity|wg|unit|org&scope_id=&period=` mirroring Phase 13 D-13-25's shape; aggregation differs only in employee-universe resolution (WG members / unit subtree members / employees with entries on the activity subtree / org).
- **D-14-21:** **Confirmed-only subtraction, declared advisory** — capacity subtracts only `confirmed` windows; `declared` windows surface as an advisory field in the response, never subtracted. **This closes Phase 13's D-13-29 deferred item: the direction warning read path (DIR-05) switches to confirmed-only in this phase.**
- **D-14-22:** **Employment validity filters capacity** — employees outside their `valid_from`/`valid_until` window are EXCLUDED from capacity responses entirely (parity with Phase 13 D-13-31: "can't plan what can't work").
- **D-14-23:** **Partial-day windows reduce capacity by their `hours`**; full-day windows zero the day (Phase 13 D-13-24 semantics carried over).

#### Read visibility & permission gates
- **D-14-24:** **Absence windows are org-wide visible** — "absence concerns everyone in the org" — any org member sees any employee's windows: kind + dates + status, including declared. The privacy carve-out is the medical record: **`certificate_ref` + attached documents visible only to `hr` + the employee's unit manager** (ADR D-1a); the medical *kind label* stays public. Server-side field filtering, not client-side.
- **D-14-25:** **REST endpoints under `/availability`** — POST /availability/windows (declare), POST /availability/windows/{id}/withdraw, POST /availability/windows/{id}/confirm, POST /availability/windows/{id}/reject (reason required), PUT /availability/windows/{id} (HR medical edit), POST /availability/windows/{id}/certificate (attach doc, medical only), GET /availability/windows (org-wide read), GET /availability/capacity, contract-types CRUD. Exact URL details are planner discretion.
- **D-14-26:** **Role gates** — declare → employee (self) or hr (anyone); withdraw → window owner only (declared-only by lifecycle); confirm/reject → resolved unit manager; HR medical edit + certificate doc write → `hr` role only (first consumer of the `hr` role in backend code); capacity read + org-wide windows read → any authenticated org member (with the D-14-24 field filtering).

#### Contract-type management
- **D-14-27:** **HR-owned full CRUD this phase** — contract_types create/edit/delete is `hr`-gated; managers read-only. Full CRUD lands now because capacity's fallback chain depends on it.
- **D-14-28:** **Hard delete if unused** — contract_types with no referencing memberships can be deleted (FK blocks otherwise); no soft-deactivate lifecycle.
- **D-14-29:** **Override attached via membership endpoint extension** — `contract_type_id` + day-hours override set through the existing membership/org endpoints (Phase 10 territory), extended this phase.

### the agent's Discretion
- Exact endpoint URL shapes and route registration within the `/availability` REST surface (D-14-25)
- `rejection_reason` column shape (TEXT vs VARCHAR + CHECK) and whether `confirmed_by`/`rejected_by`/timestamps land on the window row or live in audit only
- Certificate document storage details (BYTEA vs chunking; size limits; MIME allowlist) within the DB-backed decision (D-14-07)
- Org default schedule representation (flagged contract_type vs org_settings key) within D-14-18
- Day-hours override storage shape (rows table vs JSONB on membership) within D-14-16
- Windows list read-model filters/pagination; capacity period format (ISO week vs date range)
- Contract-types CRUD endpoints' exact shapes; whether schedule CRUD for the org default uses the same routes
- Test layout for the new availability domain package (follow per-package suite pattern)

### Deferred Ideas (OUT OF SCOPE)
- **Payroll export** (ADR-P-008 D-1c): confirmed absence windows feed the payroll view in Exports — that's Phase 25 (Exports + People polish); this phase only guarantees confirmed windows are queryable
- **work_permit_expires_at** (migration 012): not consumed by Phase 14 capacity; it stays for Phase 13's warning path and future validity surfacing — do NOT build permit-expiry logic here
- **Block-vs-nag soft policy** (D-X, carried from Phase 13): still UI-decided in Phase 19 — backend never blocks
- **Absence balances/accruals/carry-over** (ADR-P-008 D-5): still rejected — work schedules are a capacity basis, not entitlement counters; do not build balances
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AVAIL-01 | Employee can declare an absence with a type and date range (existing `availability_windows` schema); invalid or overlapping windows rejected | Declare endpoint over `availability_windows` (migration 012 shape verified); overlap guard = active-only date-range intersection in-tx under lock (D-14-13..15); validation = kind vocabulary + date range + `hours` ceiling (DECIMAL(4,2)) + medical certificate_ref requirement (D-14-05) |
| AVAIL-02 | Manager/HR can confirm or reject absences (declared → confirmed/rejected); rejects carry a reason; HR curates medical absences with certificate_ref | Unit-manager confirm/reject via `routing.ResolveUnitManager` (D-14-01/04); HR-curates-never-confirms (D-14-03); reject-with-reason CHECK; medical auto-confirm at declare (D-14-02); HR medical edit + certificate attach (D-14-11, D-14-06/07) |

Supporting (non-requirement) deliverables the planner must still plan: work-schedule model (D-14-16..18), capacity read-model (D-14-20..23), D-13-29 closure of the direction read path (D-14-21), `hr` role first consumer (D-14-26), contract-types CRUD (D-14-27..29), ADR-P-008 revision + BE encoding ADR.
</phase_requirements>

## Summary

Phase 14 is a pure Go/PostgreSQL backend phase that builds the absence lifecycle and the derived capacity read-model **over the schema already shipped in migration 012** (`availability_windows`, membership validity columns, `hr` role CHECK). The phase is a mirror of the Phase 11→13 house patterns rather than new ground: the lifecycle is a pinned state matrix (ticket/direction precedent), every mutator re-validates under `FOR UPDATE` inside its transaction with audit rows written in the same tx (CR-01 + BE-012/BE-016), migrations are append-only numbered **023+** with up/down pairs + cycle tests, and the read-models are hand-written SQL with pgx (no ORM). All patterns below were verified by reading the shipped code this session; **zero new external packages** are needed (go.mod already carries pgx v5, google/uuid, testify, testcontainers-go).

The one cross-cutting, easy-to-miss piece is the **D-13-29 closure**: Phase 13's `AbsenceWindows` repo method and the two absence subqueries inside the `Coverage` read-model currently read `status IN ('declared','confirmed')` (`direction_repository.go:767,775,816`). D-14-21 requires confirmed-only — so this phase *changes Phase 13 code and its tests* (seeds that rely on declared windows producing warnings must flip to `confirmed`). The other gap the phase must fill is the **`hr` role in Go**: `internal/models/models.go:11-25` has `RoleEmployee/Manager/Finance/Customer` but no `RoleHR`, and `IsValid()` misses it — JWT claims pass the membership role verbatim (`auth.go:503`), so adding the constant + switch case (and updating `models_test.go`) is the only auth change needed.

**Primary recommendation:** plan the phase as the Phase 12/13 template — Wave 1 migrations (023 availability status extension + rejection_reason, 024 contract_types + membership override, 025 certificate attachments; teardown list + cycle tests), Wave 2 ADRs + domain/ports + `models.RoleHR`, Wave 3 repo mutators (declare-with-overlap-guard tx, confirm/reject/withdraw txs, HR edit, attachment insert), Wave 4 read-models (windows list with field filtering, capacity query) + D-13-29 closure, Wave 5 service + handler + wiring + integration battery. The capacity query reuses the Phase 11 terminal-activity recursive CTE shape (verified at `direction_repository.go:547-566`) and the Coverage generate_series day-expansion shape (`direction_repository.go:744-803`), with the work-schedule fallback chain mirroring `orgsettings.ResolvePlanningMode` (`orgsettings.go:62-95`).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Absence lifecycle (declare/confirm/reject/withdraw) | API / Backend | Database / Storage | Service owns the state matrix + role gates; DB owns shapes via CHECK constraints (house style: services own invariants, DB owns shapes) |
| Overlap rejection | API / Backend | Database / Storage | In-tx SELECT under FOR UPDATE (D-14-15, CR-01) — never a DB EXCLUDE constraint |
| Work-schedule resolution (fallback chain) | API / Backend | Database / Storage | Fallback chain (override → type → org default → 8×5) is a service-side resolution mirroring `ResolvePlanningMode`; contract_types/override are storage |
| Capacity read-model | API / Backend | Database / Storage | Repo computes per-employee math (subtree CTE + generate_series); service owns scope resolution + validity filtering + declared-advisory field |
| Certificate document storage | Database / Storage | API / Backend | DB-backed attachment table (D-14-07) — bytes live in PostgreSQL; handler enforces MIME allowlist + size cap |
| Medical privacy field filtering | API / Backend | — | Server-side field filtering on the windows read (D-14-24): `certificate_ref` + docs only for `hr` + unit manager |
| Confirmation authority resolution | API / Backend | — | `routing.ResolveUnitManager` upward walk (BE-014), shared service — no re-implementation |
| D-13-29 direction warning path | API / Backend | Database / Storage | Phase 13 repo read changes to confirmed-only (D-14-21) |

## Standard Stack

**Phase 14 adds ZERO new external packages.** Everything required is already in `go.mod` and in use by Phases 11–13. Verified versions via `go.mod` + running toolchain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` (1.22+ ServeMux) | 1.26.1 (toolchain) | HTTP routes: `mux.HandleFunc("POST /availability/windows", ...)` method+path patterns | House pattern since Phase 11 — see `cmd/server/main.go:284-290` for the direction route block this phase mirrors |
| `github.com/jackc/pgx/v5` | v5 (in go.mod) | Hand-written SQL, transactions (`pgx.TxOptions{}` + `FOR UPDATE`), `pgx.ErrNoRows` | House standard — no ORM; all repos in `internal/adapters/secondary/postgres/` |
| `github.com/google/uuid` | v1 (in go.mod) | UUID ids, `uuid.Parse` at handler boundary, `uuid.Nil` conventions | House standard |
| `github.com/stretchr/testify` | v1 (in go.mod) | `require`/`assert` in all tests | House standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testcontainers-go` | v0.x (in go.mod) | Per-package PG integration suites via `SetupPackageContainer`/`TestPool` | Every postgres + http package test — the `exported_test_helpers.go` seeding pattern |
| `internal/middleware` | — | `middleware.Auth`, `GetRole`/`GetUserID`/`GetOrganizationID` claims | All `/availability` routes (role gates read the JWT role claim; no auth changes needed) |
| `internal/pkg/api` | — | `{ data \| error }` envelope via `api.RespondWithJSON`/`RespondWithError` | Every handler response |

**Version verification:** `go version` → `go1.26.1 darwin/arm64` (verified this session); Docker 29.4.0 (testcontainers host). pgx v5 + uuid + testify versions confirmed present in go.mod by the Phase 11–13 code compiling against them — no new installs this phase.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none — no new packages installed this phase)* | — | — | — | — | — | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

The phase consumes only the existing toolchain (Go stdlib, pgx v5, uuid, testify, testcontainers-go) already pinned in go.mod by Phases 11–13 — no install step exists, so no checkpoint:human-verify gates are required.

## Architecture Patterns

### System Architecture Diagram

```
┌─ HTTP ───────────────────────────────────────────────────────────────────┐
│  POST /availability/windows            (declare; medical auto-confirms)  │
│  POST /availability/windows/{id}/withdraw   (owner, declared-only)       │
│  POST /availability/windows/{id}/confirm    (unit manager)               │
│  POST /availability/windows/{id}/reject     (unit manager, reason req.)  │
│  PUT  /availability/windows/{id}            (HR medical edit only)       │
│  POST /availability/windows/{id}/certificate (HR, medical only)          │
│  GET  /availability/windows            (org-wide read, field-filtered)   │
│  GET  /availability/capacity?scope=&scope_id=&period=                    │
│  contract-types CRUD                     (HR write, manager read-only)   │
│         │  middleware.Auth claims: user_id / org_id / role ("hr" ✓)      │
│         ▼                                                                 │
│  availability_handler.go (thin: parse → service → pkg/api envelope)      │
│         │  sentinel → 404/400/403/409/500 map (direction_handler.go:304) │
│         ▼                                                                 │
│  availability service (internal/core/services/availability/)             │
│   ├─ state matrix fast-fail + role gates + validity checks (service)     │
│   ├─ confirm authority: routing.ResolveUnitManager (shared, BE-014)      │
│   ├─ capacity: scope→employee-universe + validity split + declared       │
│   │   advisory + schedule fallback chain (ResolveSchedule)               │
│   └─ audit rows built here, written by repo IN-TX (BE-012/BE-016)        │
│         │                                                                 │
│         ▼                                                                 │
│  ports.AvailabilityRepository + ports.AvailabilityAttachmentRepository   │
│         │                                                                 │
│         ▼                                                                 │
│  postgres availability_repository.go (pgx, hand-written SQL)             │
│   ├─ Declare tx: FOR UPDATE user's active windows → overlap? → INSERT    │
│   │     → audit 'declared' (+'confirmed' when medical) — ONE tx          │
│   ├─ Confirm/Reject/Withdraw txs: FOR UPDATE row → matrix re-check →     │
│   │     UPDATE with status-precondition backstop → audit row (CR-01)     │
│   ├─ HR edit tx (medical only) + certificate attach (BYTEA insert)       │
│   ├─ Windows read: org-wide SELECT + service-side field filtering        │
│   ├─ Capacity: schedule resolution + confirmed-only absence subtraction  │
│   │     + workload subtree CTE (status IN ('submitted','approved'))      │
│   └─ audit_logs rows (entity_type='availability_window')                 │
│                                                                           │
│  availability_windows (012) ◄─ extended 023: status +rejected/+withdrawn │
│    + rejection_reason column; contract_types + membership override (024);│
│    certificate attachments table (025)                                   │
└───────────────────────────────────────────────────────────────────────────┘
  Phase 13 closure (D-14-21): direction_repository.go AbsenceWindows +
  Coverage absence subqueries flip to status = 'confirmed' — Phase 13 seeds
  in direction tests must flip declared → confirmed.
```

### Recommended Project Structure

```
migrations/
├── 023_availability_status_ext.up.sql / .down.sql   # status CHECK +rejected,+withdrawn; rejection_reason
├── 024_work_schedules.up.sql / .down.sql            # contract_types + membership contract_type_id (+override shape)
├── 025_certificate_attachments.up.sql / .down.sql   # DB-backed attachment table
└── (ADR draft files are vault-only, not migrations)
internal/core/domain/
├── availability/               # NEW: Window entity, status/kind vocab, matrix, sentinels, JSONNames,
│   └── availability.go         #   AuditEntityWindow + audit action constants, schedule domain
internal/core/ports/
└── availability_repository.go  # NEW: port (compile-time contract for repo + testdata mocks)
internal/core/services/
└── availability/               # NEW: Service (lifecycle orchestration, overlap fast-fail, capacity
    └── availability.go         #   assembly, schedule resolution, field filtering)
internal/adapters/secondary/postgres/
├── availability_repository.go  # NEW: mutators + read-models (mirrors direction_repository.go)
└── exported_test_helpers.go    # EDIT: teardown list + seedAvailabilityWindowWithCert + seedContractType
internal/adapters/primary/http/
├── availability_handler.go     # NEW: thin handler + writeError sentinel map
└── handler_test_helper.go      # EDIT: fixture wiring for the /availability routes
internal/models/models.go       # EDIT: add RoleHR + IsValid() case (+ models_test.go validCases)
cmd/server/main.go              # EDIT: availability wiring + route registration block
internal/adapters/secondary/postgres/direction_repository.go  # EDIT: D-13-29 confirmed-only closure
```

### Pattern 1: State Machine Matrix (declared → confirmed | rejected | withdrawn)

**What:** The locked transition matrix lives in the domain package as a `map[string]map[string]bool` with `CanTransition(from, to)` + `IsTerminalStatus(s)` — the ticket (`ticket.go:89-119`) and direction (`direction.go:51-70`) precedent. `declared → confirmed|rejected|withdrawn`; rejected/withdrawn terminal; medical skips the matrix (auto-confirmed at declare, D-14-02). The DB `status` CHECK mirrors the vocabulary (house style: services own invariants, DB owns shapes).

**When to use:** Every status-bearing mutator (confirm/reject/withdraw) and the declare path (medical).

**Example (direction precedent, verbatim shape to mirror):**
```go
// internal/core/domain/direction/direction.go:51-70
var transitionMatrix = map[string]map[string]bool{
	StatusDraft: {
		StatusActive:    true,
		StatusCancelled: true,
	},
	StatusActive: {
		StatusCancelled: true,
	},
}
func CanTransition(from, to string) bool { return transitionMatrix[from][to] }
func IsTerminalStatus(s string) bool     { return s == StatusSuperseded || s == StatusCancelled }
```

### Pattern 2: In-Tx Re-Validation Under FOR UPDATE (CR-01) + Audit In-Tx

**What:** Every mutator transaction: (1) `SELECT ... FOR UPDATE` the affected row(s) in-org, (2) re-check the matrix/Σ/overlap against the **locked** values — the authoritative check; pool-level service checks are fast-fail UX only, (3) `UPDATE` with the locked status as a `WHERE` precondition backstop, (4) write the `audit_logs` row(s) **in the same tx** — a failed audit insert rolls back the state write (BE-012/BE-016), (5) re-read in-tx and commit. Verified verbatim in `direction_repository.go:164-256` (Create), `265-313` (Activate), `339-400` (cancelWithGuard).

**When to use:** Declare (overlap guard under the user's windows lock), Confirm/Reject/Withdraw (matrix re-check under the window row lock), HR edit, certificate attach.

**Example (overlap guard inside the declare tx — the D-14-15 shape):**
```go
// Authoritative overlap check inside the declare transaction (CR-01):
// lock the user's ACTIVE windows overlapping the new range; any row → reject.
var overlapping uuid.UUID
err := tx.QueryRow(ctx,
	`SELECT aw.id FROM availability_windows aw
	  WHERE aw.org_id = $1 AND aw.user_id = $2
	    AND aw.status IN ('declared','confirmed')          -- D-14-13: active-only
	    AND aw.starts_on <= $4::date AND aw.ends_on >= $3::date  -- D-14-14: range intersection
	  ORDER BY aw.id LIMIT 1 FOR UPDATE`,                  -- serialize concurrent declares
	orgID, userID, startsOn, endsOn).Scan(&overlapping)
if err == nil {
	return nil, availabilitydomain.ErrOverlap                    // → 409
}
if !errors.Is(err, pgx.ErrNoRows) { ... }
```
The overlap predicate is the exact inverse shape of the shipped `AbsenceWindows` read (`direction_repository.go:817`: `AND starts_on <= $4::date AND ends_on >= $3::date`).

### Pattern 3: Audit Vocabulary Pinned in the Domain

**What:** Audit constants live in the domain package (exported) so repo/service can never drift — the direction block (`direction.go:160-169`: `AuditEntityDirection` + `AuditActionCreated/Activated/Cancelled/Superseded/Claimed/Unclaimed`) and coverage block analogs. For availability: `AuditEntityWindow = "availability_window"` with actions **`declared`, `confirmed`, `rejected`, `withdrawn`, `edited` (HR medical edit), `certificate_attached`**. Reject/withdraw audit payloads carry `{reason}` (mirror `direction.go:439`); HR edits carry `{before, after}` per D-14-12.

**When to use:** Every mutator builds its `*audit.AuditLog` (`audit.go:15-25` shape: OrgID, EntityType, EntityID, Action, ActorID, Payload, CreatedAt) and hands it to the repo for the in-tx write — never fire-and-forget.

**Anti-pattern:** do NOT reuse direction's `cancelled` action or `superseded` status for windows — windows withdraw, they don't cancel (vocabulary drift pitfall below).

### Pattern 4: Fallback Chain Resolution (override → type → org default → 8h × Mon–Fri)

**What:** The effective schedule for an employee resolves service-side, mirroring `orgsettings.ResolvePlanningMode` (`orgsettings.go:62-95`: membership override → org key → hardcoded default). For availability: membership `contract_type_id` + per-employee day-hours override → the org's default `contract_type` → `8h × Mon–Fri`. The resolution level is returned in the capacity response (D-14-18: "the resolution level is documented in the response").

**When to use:** Capacity computation per employee; contract-type CRUD validation (cadence vocabulary `week`|`month`, `hours_per_period` > 0, day-hours matrix validation — code-level, mirroring the org_settings JSONB validation stance: "CHECK on JSONB isn't feasible; validation lives in the domain/service per known key").

### Pattern 5: Capacity Read-Model (per-employee math in repo, scope in service)

**What:** Mirrors the direction `Coverage` read-model split (`direction_repository.go:744-803`): the **repo** does the math (day expansion via `generate_series`, confirmed-only absence subtraction via the partial/full split — `hours IS NOT NULL` subtracts, `hours IS NULL` zeroes — floors at 0 with `GREATEST`), the **service** owns employee-universe resolution per scope (activity subtree / WG members / unit subtree / org), the validity split (D-14-22 — reuse `membershipValid`, `direction.go:777-791`), and the declared-advisory field (D-14-21). Workload = the Phase 11 terminal-activity recursive CTE re-anchored for the scope's activity with `status IN ('submitted','approved')` grouped per employee (see Code Examples).

**When to use:** `GET /availability/capacity` — the only capacity endpoint (D-14-20). Direction's `Coverage` read-model is NOT replaced — it keeps its own day-level capacity; Phase 14's capacity is the weekly per-employee aggregate view over the same absence facts (confirmed-only).

### Pattern 6: Migration Conventions (append-only, up/down pairs, cycle tests, CHECK drop+recreate)

**What:** Migrations continue from the max (023+; the 012 header comment documents "per ADR-BE-004, new migration files continue from the max" — the Phase 11 A8 lesson). The status CHECK extension requires PostgreSQL's drop-and-recreate — the exact pattern shipped in `012_staffing_schema.up.sql:48-50` for the role CHECK. Every new CHECK is written with explicit `IS [NOT] NULL` on both sides (the 021 convention, `direction_ontology_migrations_test.go` Pitfall-2 regression guards); the reject-requires-reason constraint uses the never-NULL-satisfiable form `CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL)` (021:60-61 precedent). Each migration gets an up/down/up cycle test asserting shape + functional CHECK rejection by constraint name (23514) — the `TestMigration021/022` template (`direction_ontology_migrations_test.go:47-240`).

**When to use:** All three migrations + cycle tests; teardown list update in `exported_test_helpers.go:81-128` (add `contract_types` after `organization_memberships` — dependency order — and the attachments table before `availability_windows`; drop order must respect the new FKs).

### Pattern 7: Role Gate + Org-Scoped Reads (first `hr` consumer)

**What:** The `hr` role exists in the DB (012 role CHECK: `('employee','manager','finance','customer','hr')`) and flows through JWT claims verbatim (`auth.go:503` generates the token from `membership.Role`; `middleware.GetRole(ctx)` returns it) — **but** the Go `models.Role` constants at `models.go:11-25` lack `RoleHR` and `IsValid()` lacks the case. Phase 14 adds:
```go
// internal/models/models.go:11-16 (current, verbatim):
const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleFinance  Role = "finance"
	RoleCustomer Role = "customer"
)
// → add: RoleHR Role = "hr"  +  case RoleHR: return true in IsValid()
//    + update models_test.go:14 validCases
```
Role gates then compare `role != string(models.RoleHR)` etc. (the `orgsettings.go:128` manager-gate shape). Every read is org-scoped (`WHERE org_id = $1`) — cross-org ids resolve to 404 (no existence oracle, the direction `Get` pattern, `direction_repository.go:80-91`).

**When to use:** Declare (employee-self or hr), withdraw (owner), confirm/reject (resolved unit manager), HR medical edit + certificate write (`hr`), windows/capacity reads (any org member) + D-14-24 field filtering (`certificate_ref` + docs only for `hr` + the employee's resolved unit manager — service-side filtering on the response, never client-side).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Overlap rejection between date ranges | Custom per-day iteration or DB `EXCLUDE USING gist` (requires `btree_gist` extension) | Single in-tx `SELECT ... FOR UPDATE` with `starts_on <= new_end AND ends_on >= new_start` (D-14-15, CR-01) | First-extension-free house style preserved; the predicate is the shipped `AbsenceWindows` shape inverted; per-day iteration is O(days) and race-prone without the lock |
| State machine transitions | Ad-hoc `if` chains per endpoint | Domain `transitionMatrix` + `CanTransition` (ticket/direction precedent) | One locked matrix, tested once, enforced twice (service fast-fail + repo in-tx re-check) |
| Audit trail for window events | A bespoke window-history table | The shipped `audit_logs` table (017) with pinned `entity_type='availability_window'` actions | BE-012 house channel; Phase 25 payroll and Phase 19 surfaces read it; no new table |
| Day expansion for capacity | Iterating dates in Go per employee | `generate_series` + `CROSS JOIN LATERAL` (shipped `Coverage` shape, `direction_repository.go:762-777`) | Set-based, timezone-free via `normalizeDay`; already proven in the Phase 13 read-model |
| Workload subtree aggregation | Walking the activity tree in Go | `WITH RECURSIVE` subtree CTE (Phase 11 terminal-activity semantics, `direction_repository.go:547-566`) | One query, index-friendly, already proven twice (ticket guard + direction derived states) |
| Binary certificate storage | Local disk + URL column | DB attachment table with BYTEA (D-14-07) — note the expense receipt precedent stores to `uploads/` on disk (`expense.go:513-537`) and is DELIBERATELY NOT followed here (locked D-14-07) | No object store in the stack; DB row travels with the window (FK/scoping); GDPR-boundary revised by ADR revision |
| DECIMAL vs float64 arithmetic | Raw float comparisons | Cents math (`math.Round(h*100)`) — `roundCents`/wholeCent helpers, `direction.go:122-124`, `coverage_repository.go` precedent | DECIMAL(4,2) window hours: sub-cent or >99.99 values must be rejected as 400, never hit PG 22003 |
| `hr` role handling | Reimplementing role resolution | Extend `models.Role` + use `middleware.GetRole` claims | JWT already carries it; one constant + one switch case + test update |

**Key insight:** Every hard problem in this phase (state machines, races, audits, recursive aggregation, day expansion, field-level privacy) has a shipped in-repo precedent from Phases 11–13. The planner should forbid novel patterns — the costliest failure mode is inventing a second way to do something the house already does (e.g., a window-history table instead of `audit_logs`, or a Go-side tree walk instead of the CTE).

## Common Pitfalls

### Pitfall 1: NULL three-valued logic in CHECK constraints
**What goes wrong:** A CHECK like `CHECK (status = 'rejected' AND rejection_reason IS NOT NULL)` evaluates to NULL for non-rejected rows (`status = 'rejected'` is false → the AND is false... actually `FALSE AND x` = FALSE; the danger is `CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL)` forms with *other* NULL-bearing shapes) — the canonical 021 comment explains it: *"Every CHECK is written with explicit IS [NOT] NULL on both sides... this form is never NULL-satisfiable: a cancelled row with NULL reason is FALSE OR FALSE and rejected"* (`direction_rows.up.sql:57-61`). A NULL-returning CHECK silently ACCEPTS the row.
**Why it happens:** SQL CHECKs accept rows when the expression is NULL or TRUE; only FALSE rejects.
**How to avoid:** Write the reject-reason constraint as `CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL)` (the never-NULL-satisfiable form, 021 precedent) and mirror the 021 "explicit IS [NOT] NULL on both sides" comment in the 023 migration header. Assert it in the cycle test with a 23514 + constraint-name check (the `direction_ontology_migrations_test.go:146-158` shape).
**Warning signs:** A migration CHECK test that passes with a NULL reason column.

### Pitfall 2: Status/audit vocabulary drift
**What goes wrong:** Phase 13 uses `draft/active/superseded/cancelled` (direction) and `declared/confirmed` (windows); entry approvals use `submit/approve/reject/...`. If the new window statuses or audit actions borrow `cancelled`/`superseded`, history reads and Phase 19/25 consumers can't distinguish a direction cancellation from a window withdrawal.
**Why it happens:** Three planes, three vocabularies, one `audit_logs` table — the phase must extend the window vocabulary (`rejected`, `withdrawn`) while keeping the audit *action* vocabulary distinct (`withdrawn`, not `cancelled`).
**How to avoid:** Pin `availability` domain constants (statuses + audit entity/actions) in one block mirroring `direction.go:160-169`; DB CHECK and domain constants must match exactly — the migration cycle test asserts the CHECK vocabulary via insert probes; the domain constants are the single source for the service.
**Warning signs:** Any literal `'cancelled'` or `'superseded'` appearing in the availability package.

### Pitfall 3: Capacity query performance — subtree CTE per employee + day explosion
**What goes wrong:** The org-scoped capacity query cross-joins `unnest(employee_ids)` × `generate_series(days)` (the Coverage shape) — for a large org over a long period this is employees×days rows; adding a per-row subtree CTE lookup (or per-employee activity ancestry walk in Go) makes it O(employees × days × subtree depth) and times out.
**Why it happens:** The workload aggregation and the absence subtraction are naturally per-(employee, day); naive composition multiplies.
**How to avoid:** Compute the workload Σ with ONE `WITH RECURSIVE` subtree per scope activity, grouped per employee, then join the aggregates — never a per-row CTE invocation. Keep the period bounded (required `period_start`/`period_end` — the `parsePeriod` boundary, `direction_handler.go:283-298`); reuse the shipped `idx_availability_windows_org_user_dates` index (012:33-34) by filtering `org_id + user_id = ANY($2)` before the date predicate. Day expansion only where genuinely needed (the absence subtraction), not for the workload.
**Warning signs:** The capacity plan's SQL runs a correlated recursive CTE inside `SELECT` per row; response times grow with employees × days.

### Pitfall 4: Overlapping-window rejection — predicate and race mistakes
**What goes wrong:** (a) The overlap predicate misses edge-adjacent ranges (`new.starts_on == existing.ends_on` must NOT overlap — dates are inclusive, so use `existing.starts_on <= new.ends_on AND existing.ends_on >= new.starts_on` — a window ending the day before starts the day after does not overlap); (b) the guard counts `withdrawn`/`rejected` rows (D-14-13: active-only); (c) two concurrent declares both pass the pool-level check and both commit (TOCTOU).
**Why it happens:** The shipped `AbsenceWindows` read is the correct inclusive-overlap shape (`direction_repository.go:817`); copying it inverted without care inverts the boundary; the CR-01 closure (lock + re-check in-tx) is the only fix for (c).
**How to avoid:** In-tx guard with `FOR UPDATE` on the user's active overlapping rows (D-14-15); service-level SELECT is fast-fail UX only; a concurrency battery test (two goroutines declaring overlapping ranges → exactly one 409/conflict) mirrors the 12-06/13-05 race batteries.
**Warning signs:** The declare tx has no `FOR UPDATE`; the overlap predicate uses `<`/`>` instead of `<=`/`>=`.

### Pitfall 5: D-13-29 closure breaks Phase 13 tests and read behavior
**What goes wrong:** Flipping `AbsenceWindows` and the `Coverage` absence subqueries from `status IN ('declared','confirmed')` to `status = 'confirmed'` silently changes Phase 13's warnings (declared windows no longer warn "away"/"partial") — and Phase 13's own tests (direction repo/service/HTTP batteries that seed **declared** windows and assert away warnings) go red.
**Why it happens:** The closure is a behavior change to shipped, tested Phase 13 code, not an additive endpoint.
**How to avoid:** Plan the closure as an explicit task owning: `direction_repository.go:812-839` (AbsenceWindows), `:767,775` (Coverage partial_abs/full_abs subqueries), port doc comments (`ports/direction_repository.go:89-102`), the `direction.AbsenceWindow` doc comment (`direction.go:142-153`), and the Phase 13 test seeds (flip declared → confirmed where warnings are asserted; add a declared-window subtest asserting it now produces NO warning — the behavioral proof of D-14-21).
**Warning signs:** The plan lists no test updates for `direction_repository_test.go`/`direction_test.go`/`direction_handler_test.go`.

### Pitfall 6: DECIMAL(4,2) ceiling and float64 cents
**What goes wrong:** `availability_windows.hours` is `DECIMAL(4,2)` (012:23) — max 99.99 — while direction's `est_hours` is `DECIMAL(8,2)` with `maxEstHours = 999999.99` (`direction.go:116`). Copying the direction validator verbatim lets `hours = 100.00` reach PG and 500.
**Why it happens:** Two different column ceilings in the same codebase; the wholeCent helper is parameter-free.
**How to avoid:** Validate window hours against the 99.99 ceiling (a `windowHoursValid` helper or a ceiling parameter); keep cents math (`math.Round(h*100)`) for the capacity subtraction so DECIMAL exactness isn't lost in float rendering — the `roundCents` read-render precedent (`direction_repository.go:793-796`).
**Warning signs:** The availability service reuses `wholeCent` unmodified.

### Pitfall 7: Migration/teardown order mistakes
**What goes wrong:** (a) Numbering a new migration below 023 or colliding with a taken number (the Phase 11 A8 lesson — 011 was taken, so 012 was named 012); (b) forgetting the new tables in `exported_test_helpers.go` teardown — testcontainers per-package suites then cascade-drop across packages and flake; (c) dropping tables in the teardown in the wrong FK order (`contract_types` must drop AFTER `organization_memberships`; attachments BEFORE `availability_windows` if FK'd).
**Why it happens:** The teardown list (lines 81-128) is manually ordered; new tables are easy to skip.
**How to avoid:** Migrations start at **023** (glob-verified: latest is `022_org_settings.up.sql`); the migration plan lists the teardown edits + cycle tests as first-wave tasks (12-01/13-01 precedent).
**Warning signs:** The plan's file list omits `exported_test_helpers.go` and `models_test.go`.

### Pitfall 8: The `hr` role gap in Go
**What goes wrong:** The service compares `role != string(models.RoleHR)` and fails to compile — or worse, the planner skips the models change and gates with a string literal `"hr"` that drifts from the DB vocabulary.
**Why it happens:** The DB CHECK got `hr` in migration 012 but the Go constants were never extended (no backend consumer existed until this phase).
**How to avoid:** Add `RoleHR` + `IsValid()` case + `models_test.go` validCases in the same plan as the first role gate; JWT claims need no change (verified `auth.go:503`).
**Warning signs:** `"hr"` appears as a raw string literal in service code.

## Code Examples

Verified in-repo patterns (all read this session) the planner's task snippets should anchor on:

### Common Operation 1: Status CHECK extension + rejection_reason (migration 023 up)
```sql
-- Source: pattern from migrations/012_staffing_schema.up.sql:48-50 (role CHECK drop+recreate)
-- + migrations/021_direction_rows.up.sql:57-61 (never-NULL-satisfiable reason CHECK)
-- PostgreSQL cannot alter CHECK constraints in place — drop and recreate.
ALTER TABLE availability_windows DROP CONSTRAINT IF EXISTS availability_windows_status_check;
ALTER TABLE availability_windows ADD CONSTRAINT availability_windows_status_check
    CHECK (status IN ('declared','confirmed','rejected','withdrawn'));
-- D-14-09: reject requires a reason — the 2VL form is never NULL-satisfiable:
-- a rejected row with NULL reason is FALSE OR FALSE and rejected.
ALTER TABLE availability_windows ADD COLUMN rejection_reason TEXT;
ALTER TABLE availability_windows ADD CONSTRAINT availability_windows_reject_reason_check
    CHECK (status <> 'rejected' OR rejection_reason IS NOT NULL);
```
(Down: drop the new column, drop the new constraint, restore the original CHECK.)

### Common Operation 2: In-tx state transition with audit (Confirm — the CR-01 shape)
```go
// Source: internal/adapters/secondary/postgres/direction_repository.go:265-313 (Activate)
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
defer func() { _ = tx.Rollback(ctx) }()
var currentStatus string
err = tx.QueryRow(ctx,
	`SELECT status FROM availability_windows WHERE id = $1 AND org_id = $2 FOR UPDATE`,
	id, orgID).Scan(&currentStatus)
// (ErrNoRows → ErrWindowNotFound; re-check matrix against the LOCKED status:)
if !availabilitydomain.CanTransition(currentStatus, availabilitydomain.StatusConfirmed) {
	return nil, availabilitydomain.ErrInvalidTransition
}
ct, err := tx.Exec(ctx,
	`UPDATE availability_windows SET status = 'confirmed' WHERE id = $1 AND org_id = $2 AND status = $3`,
	id, orgID, currentStatus) // status-precondition backstop
// audit row IN THE SAME TX (BE-012/BE-016): insertAudit(ctx, tx, log) — the
// insertDirectionAudit shape (direction_repository.go:119-144), entity_type from
// the caller-controlled vocabulary.
```

### Common Operation 3: Workload subtree CTE (submitted+approved entries)
```sql
-- Source: internal/adapters/secondary/postgres/direction_repository.go:547-566
-- (terminalActivitySubtree — Phase 11 terminal-activity semantics, re-anchored
--  for the workload aggregation: same subtree walk, status predicate replaced
--  per D-14-19).
WITH RECURSIVE subtree AS (
    SELECT id FROM activities WHERE id = $1          -- scope activity (or root of scope)
    UNION ALL
    SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
)
SELECT te.user_id AS employee_id, SUM(te.hours) AS workload
FROM time_entries te
WHERE te.is_deleted = false
  AND te.status IN ('submitted','approved')          -- D-14-19: only these count
  AND te.activity_id IN (SELECT id FROM subtree)
GROUP BY te.user_id;
```

### Common Operation 4: Confirmed-only absence subtraction (per-day capacity)
```sql
-- Source: internal/adapters/secondary/postgres/direction_repository.go:762-777
-- (partial_abs / full_abs subqueries — the ONLY change is the status predicate
--  becoming status = 'confirmed' for the D-13-29 closure; the rest is verbatim.)
LEFT JOIN (
    SELECT aw.user_id AS employee_id, gs.day, SUM(aw.hours) AS hours
    FROM availability_windows aw
    CROSS JOIN LATERAL generate_series(GREATEST(aw.starts_on, $3::date), LEAST(aw.ends_on, $4::date), '1 day') AS gs(day)
    WHERE aw.org_id = $1 AND aw.user_id = ANY($2)
      AND aw.status = 'confirmed' AND aw.hours IS NOT NULL       -- was IN ('declared','confirmed')
    GROUP BY aw.user_id, gs.day
) partial_abs ON partial_abs.employee_id = e.employee_id AND partial_abs.day = d.day
-- full_abs twin with hours IS NULL; capacity = GREATEST(daily - COALESCE(partial,0), 0) when not full-absent
```

### Common Operation 5: Sentinel → HTTP map (availability handler)
```go
// Source: internal/adapters/primary/http/direction_handler.go:304-323 (writeError shape)
func (h *AvailabilityHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, availabilitydomain.ErrWindowNotFound):
		api.RespondWithError(w, http.StatusNotFound, "window not found")
	case errors.Is(err, availabilitydomain.ErrInvalidRequest),
		errors.Is(err, availabilitydomain.ErrRejectReasonRequired):
		api.RespondWithError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, availabilitydomain.ErrForbidden):
		api.RespondWithError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, availabilitydomain.ErrInvalidTransition),
		errors.Is(err, availabilitydomain.ErrOverlap):
		api.RespondWithError(w, http.StatusConflict, "conflict")
	default:
		api.RespondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
```

### Common Operation 6: Membership validity filter (D-14-22)
```go
// Source: internal/core/services/direction/direction.go:777-791 (membershipValid)
// nil membership, or period fully outside valid_from/valid_until → excluded
// from capacity responses entirely (boundaries inclusive).
func membershipValid(m *auth.OrganizationMembership, periodStart, periodEnd time.Time) bool { ... }
// Reuse as-is (export or mirror in the availability package — D-G parity: do not fork).
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `http.ServeMux` path-only routing | Go 1.22+ method+path patterns (`"POST /availability/windows/{id}/confirm"`) | Go 1.22 (2024) | The whole v0.2 backend already uses it (`main.go`) — Phase 14 routes follow verbatim |
| Status CHECK with `('declared','confirmed')` | Extended vocabulary `('declared','confirmed','rejected','withdrawn')` + `rejection_reason` | This phase (023) | Windows become a real lifecycle; payroll (Phase 25) reads confirmed-only |
| Direction read path reads declared+confirmed (D-13-29 provisional) | Confirmed-only (D-14-21 closure) | This phase | Warnings/capacity reflect approved absence only — declared windows become advisory |
| `hr` role only in the DB CHECK | `hr` enforced in Go (`models.RoleHR`, role gates) | This phase | First backend consumer of the role; org/HR admin surfaces (Phase 25) reuse the gates |

**Deprecated/outdated:**
- ADR-P-008 D-1a's two-line confirmation (unit manager + WG manager): **revised this phase** — D-14-01 drops the WG line; the ADR revision + BE encoding ADR are phase deliverables.
- ADR-P-008 D-5's no-medical-document boundary: **revised this phase** — D-14-06/07 allow DB-backed certificate attachments (GDPR special-category flag noted in the ADR revision).
- The direction read path's `status IN ('declared','confirmed')` (D-13-29 provisional reading): **closed this phase** — confirmed-only, with Phase 13 test seeds updated.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `contract_types` table name does not collide with the existing `contracts.contract_type` column (migration 016). They are different concepts — the column is a contract funding-shape label ('support'), the table is a work-schedule template. | Architecture Patterns | Low — naming proximity only; the planner should add a comment in the 024 migration header distinguishing them (the 012 header-comment convention) |
| A2 | Workload statuses read literally as `('submitted','approved')` per D-14-19/AVAIL-04 text. Entries in `pending_manager`/`pending_finance` (in-flight submitted work) do NOT count under the literal pin. | Common Operation 3 | Medium — if the user meant "all submitted work in the pipeline", the query undercounts workload during approval; flagged in Open Questions for the discuss phase |

**Everything else in this research is [VERIFIED]** against in-repo files read this session (sources below) — no user confirmation needed for the stack, patterns, schema facts, or D-13-29 closure mechanics.

## Open Questions (RESOLVED)

All 7 questions were adopted by the phase plans — each is resolved below with the adopting plan and the locked/decided outcome. No open question remains for execution.

1. **Workload status set: literal `submitted+approved` or the full post-submit pipeline?**
   - What we know: D-14-19 and AVAIL-04 pin "submitted+approved"; entry statuses are `draft → submitted → pending_manager → pending_finance → approved/rejected` (models.go:165-172) — `submitted` is transient.
   - What's unclear: whether `pending_manager`/`pending_finance` entries (submitted work in the approval pipeline) should count as workload.
   - Recommendation: implement the literal pin (`('submitted','approved')`) per the locked decision; raise the pipeline reading in the discuss phase before locking tests.
   - **RESOLVED (adopted):** the literal `('submitted','approved')` set is implemented in the workload subtree CTE (14-07 Task 1, Common Operation 3) and asserted by the repo battery (draft/pending excluded); the pipeline reading was raised in discuss (D-14-19) and not adopted — pinned again in the 14-07 flagged assumption A2.

2. **`rejection_reason` column shape and action-actor columns**
   - What we know: D-14-08/09 require the column + a reason; discretion explicitly covers TEXT vs VARCHAR+CHECK and whether `confirmed_by`/`rejected_by`/timestamps live on the row or in audit only.
   - Recommendation: `rejection_reason TEXT` (the `direction.reason TEXT` precedent) + the 2VL CHECK; keep actors/timestamps in `audit_logs` only (the house pattern — direction rows carry no confirmed_by; history reads via audit), unless Phase 25 payroll needs row-level actors.
   - **RESOLVED (adopted):** `rejection_reason TEXT` + the named 2VL CHECK `availability_windows_reject_reason_check` land in migration 023 (14-01 Task 1); actors/timestamps stay in `audit_logs` only (14-02 Task 2 domain, 14-05 audit DTOs).

3. **Certificate storage details within the DB-backed decision**
   - What we know: D-14-07 locks PostgreSQL storage; the expense receipt handler precedent (`expense.go:507-509`) allows only PDF/JPEG/PNG and caps size via `MaxBytesReader`.
   - Recommendation: single BYTEA column with a 5 MB cap + the PDF/JPEG/PNG allowlist; skip chunking (single-window documents, PG toast handles large values). Planner discretion to adjust.
   - **RESOLVED (adopted):** single `storage BYTEA` column in migration 025 (14-01 Task 3); 5 MB MaxBytesReader cap + PDF/JPEG/PNG allowlist at the handler (14-08 Task 1, expense.go gate shape, never disk); attach repo tx in 14-05 Task 2.

4. **Org default schedule representation (D-14-18)**
   - What we know: either a `contract_types.is_default` flag (needs a partial unique index per org) or an `org_settings` key referencing the type id (the `planning_daily_hours` precedent).
   - Recommendation: `org_settings` key (`default_contract_type_id`), consistent with D-13-18's settings store and validated in the orgsettings service; avoids uniqueness machinery.
   - **RESOLVED (adopted):** `org_settings` key `default_contract_type_id`; NO `is_default` flag on contract_types (024 in 14-01 Task 2); read via the existing PUT /organizations/settings endpoint (14-06 Task 3) and consumed by ResolveSchedule (14-06 Task 2).

5. **Day-hours override storage shape (D-14-16)**
   - What we know: rows table (weekday → hours, UNIQUE per membership) vs JSONB on membership; the org_settings JSONB precedent ("CHECK on JSONB isn't feasible; validation in domain") exists.
   - Recommendation: JSONB day-hours matrix on BOTH `contract_types` (default matrix) and the membership override, validated code-side — uniform, matches the settings precedent, no extra table. The rows-table alternative is fine if the planner prefers DB-enforced weekday keys.
   - **RESOLVED (adopted):** JSONB `day_hours` on contract_types + JSONB `day_hours_override` on organization_memberships (024 in 14-01 Task 2), validated code-side by the domain DayHours helpers (14-02 Task 2) and the service (14-06 Task 2); override merges over the type matrix, override wins per weekday.

6. **Capacity period format**
   - What we know: direction uses `period_start`/`period_end` (`2006-01-02`) at the boundary (`parsePeriod`); D-14-20 says "mirroring D-13-25's shape"; weekly hours is the basis.
   - Recommendation: `period_start`/`period_end` required (reuse `parsePeriod` verbatim); the service derives per-day schedule hours and totals weekly within the period. ISO-week strings only if Phase 16 UI needs them — the handler can normalize.
   - **RESOLVED (adopted):** required `period_start`/`period_end` parsed with `parsePeriod` verbatim (14-08 Task 2); per-day schedule expansion + weekly totals in the service (14-07 Task 2); malformed/missing period → 400.

7. **Windows list read-model filters/pagination (discretion)**
   - Recommendation: `GET /availability/windows?user_id=&kind=&status=&period_start=&period_end=` with `LIMIT/OFFSET` or cursor — keep it minimal; Phase 16 UI is the first consumer.
   - **RESOLVED (adopted):** `GET /availability/windows` with optional `user_id`/`kind`/`status`/period filters + `LIMIT/OFFSET` pagination (14-07 Task 1 repo predicates + 14-08 Task 2 handler parsing).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All backend code + tests | ✓ | go1.26.1 darwin/arm64 (verified) | — |
| Docker (testcontainers host) | Per-package PG integration suites (`TestPool` → `SetupPackageContainer`) | ✓ | 29.4.0 (verified) | No fallback — suites require Docker |
| PostgreSQL | Migrations + repos (local dev default `postgres://hourglass:hourglass@localhost:5432/hourglass`) | ✓ (via docker-compose; testcontainers for tests) | — | testcontainers spins its own container for tests |
| Node.js | Not required (backend-only phase) | ✓ | v22.23.1 | — |
| `make` | `make test` (full suite) | ✓ | — | `go test ./...` equivalent |

**Missing dependencies with no fallback:** none — Docker is the only hard requirement and it is present.
**Missing dependencies with fallback:** none.

Step 2.6 environment audit performed this session — all phase dependencies probed and available.

## Validation Architecture

> Per-phase validation contract for feedback sampling during execution (Nyquist VALIDATION.md). `.planning/config.json` has no `workflow.nyquist_validation` key → treated as enabled. Framework: Go `testing` with testify + testcontainers per-package suites (house standard since Phase 0 — no new framework install).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `go test` + testify (require/assert) + testcontainers-go per-package PG suites |
| Config file | none — Go stdlib testing; suite setup via `internal/adapters/secondary/postgres/test_setup.go` (`SetupPackageContainer`) |
| Quick run command | `go test ./internal/core/domain/availability/ ./internal/core/services/availability/ -count=1` (unit fast-fail loop; service tests run against the testcontainers pool via `TestPool`) |
| Full suite command | `make test` (runs `go test -v ./...`; 24 packages green at Phase 13 close) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AVAIL-01 | Declare valid window (holiday/permit/unavailable) via POST /availability/windows → 200 `declared` | integration | `go test ./internal/adapters/primary/http/ -run TestAvailabilityHandler -count=1` | ❌ Wave 0 |
| AVAIL-01 | Declare invalid: bad kind, `ends_on < starts_on`, `hours > 99.99`, non-integer hours → 400 (never 500) | integration | same | ❌ Wave 0 |
| AVAIL-01 | Declare overlapping window (any active kind) → 409; declared+confirmed count, withdrawn/rejected do not (D-14-13) | integration | same | ❌ Wave 0 |
| AVAIL-01 | Concurrent overlapping declares → exactly one succeeds (CR-01 race battery) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository -count=1` | ❌ Wave 0 |
| AVAIL-01 | Medical declare requires `certificate_ref` + auto-confirms immediately (D-14-02/05) | integration | same | ❌ Wave 0 |
| AVAIL-02 | Confirm by resolved unit manager → `confirmed` + audit row; non-manager 403; self-confirm when employee IS unit manager (D-14-04); only `declared` confirmable → 409 otherwise | integration | `go test ./internal/adapters/primary/http/ -run TestAvailabilityHandler -count=1` | ❌ Wave 0 |
| AVAIL-02 | Reject requires reason → 400 without, `rejected` + audit `{reason}` with; rejected terminal (no re-confirm, no edit) (D-14-08/09) | integration | same | ❌ Wave 0 |
| AVAIL-02 | Withdraw declared-only (owner), terminal `withdrawn` + audit; non-owner 403 (D-14-10) | integration | same | ❌ Wave 0 |
| AVAIL-02 | HR medical edit (PUT) + certificate attach (POST .../certificate) `hr`-gated; non-hr 403; edit on non-medical 400 (D-14-03/11) | integration | same | ❌ Wave 0 |
| (D-14-24) | Windows read org-wide (any member) with `certificate_ref` + docs filtered out for non-hr/non-unit-manager | integration | same | ❌ Wave 0 |
| (D-14-20..23) | Capacity: weekly hours per schedule (fallback chain levels), confirmed-only subtraction, declared advisory field, validity-excluded employees absent, workload Σ submitted+approved on activity subtree, per scope activity/wg/unit/org | integration | same | ❌ Wave 0 |
| (D-14-21) | D-13-29 closure: direction warnings no longer fire for declared-only windows; confirmed windows still warn | integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration|Direction' -count=1` + `go test ./internal/core/services/direction/ ./internal/adapters/primary/http/ -run TestDirection -count=1` | ❌ Wave 0 (existing files, behavior changed) |
| (D-14-12) | Every window event writes `audit_logs` (`entity_type='availability_window'`) in-tx; failed audit rolls back the state write | integration | `go test ./internal/adapters/secondary/postgres/ -run TestAvailabilityRepository -count=1` | ❌ Wave 0 |
| (migrations) | 023/024/025 up/down/up cycles; CHECK vocabularies + reject-reason 2VL asserted with 23514 + constraint name | integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration02[3-5]' -count=1` | ❌ Wave 0 |
| (hr role) | `models.Role.IsValid()` accepts `hr`; role gates compile + JWT claims carry it | unit | `go test ./internal/models/ -count=1` | ✅ existing file, ❌ new cases |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/domain/availability/ ./internal/core/services/availability/ -count=1` (or the package(s) touched by the task)
- **Per wave merge:** `make test` (full suite — catches cross-package breaks, e.g. the D-13-29 closure's effect on direction packages)
- **Phase gate:** Full suite green before `/gsd-verify-work` (Phase 13 precedent: `make test` exit 0, 24 packages)

### Wave 0 Gaps
- [ ] `internal/core/domain/availability/` — domain package (Window, vocab, matrix, sentinels, audit constants) + unit tests
- [ ] `internal/core/ports/availability_repository.go` — port (compile-time contract) + testdata mocks
- [ ] `internal/adapters/secondary/postgres/availability_repository_test.go` — migration cycle tests 023/024/025 + mutator/read-model batteries
- [ ] `internal/adapters/primary/http/availability_handler_test.go` — integration battery (permission matrix, sentinels, race test)
- [ ] `exported_test_helpers.go` — teardown list additions + `seedAvailabilityWindowWithCert`/`seedContractType` helpers (named WithCert — the Phase 13 `seedAvailabilityWindow` already exists in direction_repository_test.go)
- [ ] `internal/models/models.go` + `models_test.go` — `RoleHR` + validCases
- [ ] D-13-29 closure test updates in `direction_repository_test.go` / `direction_test.go` / `direction_handler_test.go` (declared-window seeds → confirmed; new no-warning-on-declared subtest)

## Security Domain

`security_enforcement` not explicitly false in `.planning/config.json` → section required. Phase 14 is backend-only; the security surface is authorization (role + org scoping), input validation, and privacy field filtering.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Existing JWT cookie auth — no auth changes (role claims verified to carry `hr` verbatim, `auth.go:503`) |
| V3 Session Management | no | Existing session machinery untouched |
| V4 Access Control | yes | Role gates (D-14-26) + org-scoped reads (`WHERE org_id = $1` — cross-org → 404 no-existence-oracle) + D-14-24 server-side field filtering (`certificate_ref` + documents only `hr` + unit manager); unit-manager authority via `routing.ResolveUnitManager` |
| V5 Input Validation | yes | Domain validation before any write (kind vocabulary, date-range, `hours` DECIMAL(4,2) ceiling, `certificate_ref` required for medical, reject reason required, overlap guard) + boundary parse (uuid/period → 400, never 500) |
| V6 Cryptography | no | No new crypto; attachments are internal HR data (GDPR special-category flag carried in the ADR-P-008 revision — visibility scope is the control, not encryption-at-rest) |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Overlap TOCTOU (two declares race past the check) | Tampering | In-tx `FOR UPDATE` re-check (CR-01) + concurrency battery test |
| Privacy leak of medical data (certificate_ref/doc visible org-wide) | Information Disclosure | Server-side field filtering at the service layer (D-14-24) — never client-side; unit-manager scope resolved server-side |
| Cross-org read/write (org A actor mutating org B window) | Elevation of Privilege | Org-scoped predicates on every query + no-existence-oracle 404 pattern |
| Role drift (`hr` gate as string literal) | Spoofing | `models.RoleHR` constant + `IsValid()` — compile-time vocabulary |
| Upload abuse (oversized / wrong-MIME certificate) | Denial of Service | `MaxBytesReader` cap + PDF/JPEG/PNG allowlist (expense receipt precedent) |
| Reject-without-reason bypass | Tampering | 2VL CHECK `(status <> 'rejected' OR rejection_reason IS NOT NULL)` + service fast-fail + cycle test |

## Sources

### Primary (HIGH confidence — in-repo files read this session)
- `migrations/012_staffing_schema.up.sql` — availability_windows shape (columns, kind/status CHECKs, hours DECIMAL(4,2), certificate_ref), membership validity columns, role CHECK extension pattern
- `migrations/017_audit_logs.up.sql` — audit table shape + entity index
- `migrations/021_direction_rows.up.sql` — CHECK conventions (explicit IS [NOT] NULL, never-NULL-satisfiable reason form)
- `migrations/022_org_settings.up.sql` (via cycle test) — org_settings PK/key/value shape
- `internal/adapters/secondary/postgres/direction_repository.go` — CR-01 in-tx mutators, audit-in-tx, AbsenceWindows + Coverage read-models (the D-13-29 closure sites: lines 767, 775, 816), terminal-activity CTE (547-566), cents rounding
- `internal/core/services/direction/direction.go` — warning overlay, `membershipValid` (777-791), scope resolution, wholeCent/maxEstHours
- `internal/core/domain/direction/direction.go` — transition matrix, audit vocabulary block (160-169), AbsenceWindow doc
- `internal/core/domain/ticket/ticket.go` — matrix + CanTransition precedent (89-119)
- `internal/core/services/routing/routing.go` — `ResolveUnitManager` (109-134)
- `internal/core/services/orgsettings/orgsettings.go` — fallback-chain resolution precedent (62-95), manager gate
- `internal/adapters/secondary/postgres/org_settings_repository.go` — in-tx audit upsert pattern
- `internal/adapters/primary/http/direction_handler.go` — writeError sentinel map (304-323), parsePeriod (283-298)
- `cmd/server/main.go` — route registration block (284-290), wiring order
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — teardown list (81-128), seed helpers
- `internal/adapters/secondary/postgres/direction_ontology_migrations_test.go` — migration cycle test template (47-240)
- `internal/models/models.go` — Role constants (11-25) — the `hr` gap
- `internal/core/services/auth/auth.go:503` — JWT role claim verbatim from membership
- `internal/adapters/primary/http/expense.go:479-544` — upload MIME/size precedent (deliberately NOT followed for storage, D-14-07)
- `internal/core/domain/audit/audit.go` — AuditLog shape
- `.planning/phases/14-availability-backend-absences-capacity/14-CONTEXT.md` — all D-14 decisions (locked)
- `.planning/phases/13-direction-backend-the-plan-plane/13-CONTEXT.md` — D-13-24/25/29/31 references
- `.planning/REQUIREMENTS.md` — AVAIL-01/02 text; `.planning/ROADMAP.md` — phase goal + 4 success criteria
- `.planning/config.json` — nyquist_validation absent (enabled), security_enforcement absent (enabled)

### Secondary (MEDIUM confidence)
- None — no external sources needed; every claim verified against shipped code this session. External search providers are disabled in the init config (exa/brave/firecrawl false), and no new-library documentation is required because Phase 14 installs zero new packages.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new packages; toolchain versions verified live this session
- Architecture: HIGH — every pattern (matrix, CR-01, audit-in-tx, CTE, generate_series, migration cycles) verified against shipped Phase 11–13 code with line anchors
- Pitfalls: HIGH — each pitfall traced to a concrete shipped site (012/021/017 migrations, direction repo/service/handler)
- Open decisions: all remaining choices are explicitly delegated to the planner by CONTEXT.md's Discretion list, with recommendations above

**Research date:** 2026-08-08
**Valid until:** 2026-08-15 (repo-internal patterns — the phase's own code supersedes this document once execution starts; no fast-moving external dependencies)




