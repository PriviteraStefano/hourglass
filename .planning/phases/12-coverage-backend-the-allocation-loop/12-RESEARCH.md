# Phase 12: Coverage Backend — The Allocation Loop - Research

**Researched:** 2026-08-07
**Domain:** Go/PostgreSQL backend — coverage allocation ledger (Σ invariant), derived bucket balances, proposals-on-read, atomic replace-set writes, period-close snapshots, hexagonal domain/services/adapters
**Confidence:** HIGH (stack + architecture patterns — verified in codebase), MEDIUM (open vocabulary/semantic points — flagged in Assumptions Log / Open Questions)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Funding-source storage shape
- **D-01:** **Tagged union on the allocation row** — `coverage_allocations` carries `source_type` + nullable ref columns (`contract_id`, `unit_id`) with a CHECK matching refs to type, mirroring the Phase 11 origin pattern (D-01 of Phase 11: discriminator + nullable columns + three-valued-logic CHECK guard). No `funding_sources` hub table. Service requests are zero-value contracts drawn as `source_type='contract'`. — **Reversibility:** costly — a hub table could be introduced later but the ledger shape, ports, and write endpoints would all need to change; the origin precedent makes the tagged union the durable choice.
- **D-02:** **Bucket balance derived, never stored** — support bucket balance = Σ sold_hours on the support contract − Σ allocations drawn from it, computed on read. Consistent with D-I (never store what can be computed); carry-over/no-expiry (D-P) falls out naturally because the ledger is cumulative. No counter column, no write-path lock on the bucket row.
- **D-03:** **Overdraw allowed, visible in report** — allocations may push a bucket balance negative; the invariant Σ = entry hours is never a bucket gate. D-C ("no anti-abuse control on support buckets"); the report is the control.

#### Proposal eligibility rules
- **D-04:** **Single chain-driven default-source rule** — one decision function over the entry's activity chain (the D-3 upward walk resolving `contract_id`, same CTE used for billability): contract found & type `project` → its contract budget; type `support` → its bucket; zero-value contract → contract draw (service request); no contract → internal absorption with the beneficiary unit (inherited downward like `contract_id`). No separate billability-first branch.
- **D-05:** **Ticket-kind → funding eligibility is an extension point only** — D-H says kinds drive eligibility, but no kind→source matrix was ever defined. Build the proposal function with one extension seam; implement only the chain rules (contract vs bucket vs absorption) now. Proposals are computed on read, so later kind rules are a code change, not a migration.
- **D-06:** **To-cover queue includes no-source entries, flagged** — every uncovered approved `time` entry appears (Σ alloc < hours). Entries with no derivable source (activity with neither contract nor beneficiary unit) appear in the same queue flagged "no eligible source — manual" with the reason. One read-model, self-explaining rows; the "never an implicit gap" rule holds for the no-source case.

#### Write semantics & permission
- **D-07:** **Replace-set per entry, atomic** — one save call (e.g. PUT `/time-entries/{id}/allocations`) takes the full allocation set (1..N rows), the service validates Σ = entry hours inside the transaction, and replaces everything atomically. No incremental CRUD endpoints. Matches the week-1 flow: system proposes → manager confirms whole split or edits it → one save. Invariant checked by construction, no partial states.
- **D-08:** **Writer = the entry's own manager** — permission resolved via BE-014 routing (entry's activity chain → WG → manager), the same resolution that approved the entry. Not any org manager; finance is read-only (D-L: overview through reports + audit trail, no second confirm step).
- **D-09:** **No correction handling** — coverage only ever sees approved `time` entries with `hours > 0` (schema CHECK already forbids negatives). Compensating entries do not exist in the codebase (`created_from_entry_id` is schema-only); the D-13 "net-of-compensations" dismissal-guard swap has nothing to swap to — leave the guard on raw Σ. If a correction mechanism ever lands, coverage interaction must be designed then (see Deferred Ideas). — **Reversibility:** reversible — additive if corrections ever arrive.

#### Snapshot mechanics
- **D-10:** **Frozen snapshot at period close** — a dedicated close operation writes an immutable snapshot of the period's allocation state; reports (billing, bucket levels, per-unit) and any future invoicing read the snapshot, never live rows. "A reported period never changes retroactively" holds by construction. Live allocations stay editable indefinitely (D-F) with full audit trail (actor, timestamp, payload) as the visibility control. — **Reversibility:** costly — abandoning the snapshot for audit replay later is possible but the report read-models and Phase 17 surfaces would need rework; the audit-log replay path remains viable as a fallback because every allocation write is audit-logged with payload.
- **D-11:** **Snapshot granularity: entry-level rows** — snapshot rows carry the allocation state per entry/source for the closed period (entry_id, source refs, hours, activity chain snapshot, employee, entry date). Bucket levels, billing totals, and per-unit report aggregates are computed from these rows on read when the Phase 17 surfaces land.
- **D-12:** **Close triggered by a new endpoint, not `financial_cutoff_periods`** — a coverage-specific close endpoint (e.g. `POST /coverage/close` scoped to org + period range) writes the snapshot for entries whose `entry_date` falls in the period, with allocations as they stand at close. Do not reuse `financial_cutoff_periods` as the trigger (user's choice); how the two mechanisms relate, if at all, is planner discretion.

### the agent's Discretion
- Exact endpoint list and URL shapes for coverage routes (allocations write, to-cover queue read, proposals read, close, snapshot read) within D-07/D-12
- Proposal read-path exposure shape (per-entry proposal endpoint vs queue-with-proposals combined read-model)
- The chain-walk CTE reuse for default-source derivation (upward walk exists for contract_id/billability)
- Snapshot table shape and naming (entry-level rows per D-11; aggregate absence per D-11)
- Whether the close endpoint also returns the snapshot data in one call (D-12 "new close endpoint" implies close + report in one call — confirm during planning)
- Audit-log write mechanics: coverage changes follow the Phase 11 in-tx synchronous pattern (BE-016), not fire-and-forget — mirror tickets
- How `source_type` vocabulary is CHECK-enforced (house style) and whether cross-project transfer carries an extra `justification` column vs internal absorption's `reason` column (both mandatory, distinct fields)
- Test layout for the new coverage domain package (follow existing per-package suite pattern)

### Deferred Ideas (OUT OF SCOPE)
- **Correction/compensation entries** — `created_from_entry_id` is schema-only; no mechanism exists. When/if one lands, design the coverage interaction then (and re-evaluate the D-13 dismissal-guard computation). Not Phase 12.
- **Billing/invoicing lock (v2+)** — when an invoicing module lands, invoices read the frozen snapshot and THAT is where an allocation lock attaches for paid periods. The snapshot schema (D-11) must not preclude it.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| COV-01 | Approved time entries are coverable by 1..N coverage allocations; Σ allocations = entry hours (invariant); uncovered hours land in the to-cover queue — a visible state, never an implicit gap (P-012) | D-07 replace-set write with Σ validation in-tx under a FOR UPDATE entry lock (CR-01 lesson); `time_entries.status` vocabulary verified (`'approved'` is a valid status, migration 004); to-cover queue read-model = approved entries LEFT JOIN Σ allocations, `hours > 0` CHECK already in schema (migration 000 line 278) |
| COV-02 | Funding sources exist and work: contract budget (default for billable, D-7 rule), support bucket (hours-only, carry-over, no expiry, overlapping buckets allowed — D-P), service request (zero-value contract, D-J), internal absorption (mandatory reason: WarrantyBug · UnderEstimate · Goodwill), cross-project transfer (explicit justification) (P-014) | D-01 tagged union (`source_type` + `contract_id`/`unit_id` + CHECK); D-04 chain-driven default rule over `ResolveCommercialContext` (verified CTE, activity_repository.go:236-264) + `contract_type`/`sold_hours` (migration 016); zero-value contract detection needs a pinned definition (see Open Question 3); D-02 derived bucket balance = Σ sold_hours − Σ drawn; reason/justification columns both mandatory, distinct fields (discretion) |
| COV-03 | Manager confirms allocations in one step (no finance chain, D-L); every change is audit-logged (BE-012); proposals are computed on read, never stored (D-I) | D-08 permission via shared `routing.ResolveManagerStage` (verified: ApproverIDs/RoleGated/SkipToFinance fields, routing.go:41-45; Approve gate at time_entry.go:191-193); general `audit_logs` table (migration 017) with in-tx write pattern (insertTicketAudit precedent, ticket_repository.go:83-110); no proposal table — D-04 decision function computed on read |
| COV-04 | Allocations stay editable indefinitely; period close produces a reporting snapshot (billing, bucket levels, per-unit report), never a lock (D-F); snapshot mechanics backend-only (F) | D-10/D-11 frozen snapshot written by a new close endpoint (D-12); entry-level snapshot rows (no aggregates); append-only (no UPDATE/DELETE paths — ticket precedent); `financial_cutoff_periods` exists (migration 000:410-419) but is NOT the trigger |
| COV-05 | Activity carries a nullable beneficiary unit (inherited downward like `contract_id`); absorption funding sources default from it; coverage entries are polymorphic (`entry_type` + `entry_id`), `time` only allowed in v0.2 (D-B, D-K) | New migration adds `activities.beneficiary_unit_id`; upward-walk resolution mirrors `ResolveCommercialContext` CTE shape; `GetAncestry` SELECT list is explicit (activity_repository.go:196-211) and MUST be extended when the column lands; `coverage_allocations.entry_type`+`entry_id` polymorphic pair with service-level rejection of non-`time` (the "one extra validation branch" the BE ADR must cost honestly) |

## Summary

Phase 12 is a **backend-only, purely additive phase** on the settled hexagonal codebase (Go 1.26.1 + pgx v5 + stdlib net/http). It lands the coverage plane: `coverage_allocations` (tagged-union ledger with the Σ = entry hours invariant), a derived-on-read support-bucket balance, proposals computed on read from a chain-driven decision function (D-04), an atomic replace-set write endpoint gated to the entry's own manager (D-08 via the shared BE-014 routing package), a to-cover queue read-model that includes no-source entries flagged, and a frozen period-close snapshot with entry-level rows (D-10/D-11/D-12). Coverage references a polymorphic entry (`entry_type` + `entry_id`, `time` only in v0.2 — D-K), and activities gain a nullable `beneficiary_unit_id` inherited upward through the same chain-walk CTE family as `contract_id`. Every allocation change is audit-logged synchronously in-transaction via the general `audit_logs` table (017), mirroring the Phase 11 ticket pattern exactly. A new backend ADR (ADR-BE-0xx — Coverage encoding) is drafted this phase, and ADR-P-012 moves from Proposed to Accepted in the vault.

**No new external packages are needed.** Every library required (pgx, uuid, testify, testcontainers-go) is already in `go.mod`; the phase reuses established patterns verified in this session: thin handlers → service invariants → pgx repositories with sentinel errors, DB CHECK vocabularies with the three-valued-logic guard (house style, verified in migrations 015/016), `pool.BeginTx` atomic multi-write with in-tx audit rows (precedent: ticket_repository.go `Create`/`UpdateState`/`Triage`), `SELECT ... FOR UPDATE` re-validation inside mutator transactions (Phase 11 CR-01 closure — ticket_repository.go:250), and per-package test suites with testcontainers + self-seeding cycle tests.

**Primary recommendation:** plan three additive migrations (018 beneficiary unit, 019 `coverage_allocations`, 020 snapshot tables), a new `coverage` domain + service + postgres repo + HTTP handler pair, extension of the activity domain/repo/CTE (beneficiary unit), a replace-set allocations write (`PUT /time-entries/{id}/allocations`) with in-tx Σ validation under a FOR UPDATE entry-row lock, read endpoints for proposal / to-cover queue / bucket balance / snapshot, and the coverage close endpoint (`POST /coverage/close`). The D-04 default-source decision function reuses and extends the existing chain-walk resolvers (`ResolveCommercialContext` gains the contract-type data needed; a new `ResolveBeneficiaryUnit` mirrors its CTE shape). The BE encoding ADR must cost the D-K polymorphic validation branch honestly (one service-level check rejecting `entry_type != 'time'`).

**⚠️ Key planning constraint:** every claim about schema shape below is grounded in files read this session (migrations 000/004/011/012/015/016/017, routing.go, time_entry.go, activity_repository.go, ticket_repository.go, audit_log_repository.go, exported_test_helpers.go, cmd/server/main.go). The open items (source_type vocabulary, zero-value-contract definition, snapshot table naming, close idempotency, close-returns-report) are flagged in Open Questions and must be resolved in the plan — none is a blocker, all are planner decisions with recommendations below.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Σ-invariant ledger write (COV-01, D-07) | Database/Storage (transaction) | API/Backend (service) | Replace-set committed atomically; Σ re-validated in-tx under the FOR UPDATE entry-row lock (CR-01 lesson); service drives validation |
| Funding-source shape (COV-02, D-01) | Database/Storage | API/Backend | Tagged union + refs-to-type CHECK in DB (house style); same-org ref validation at service |
| Bucket balance (COV-02, D-02/D-03) | Database/Storage (read query) | API/Backend | Derived Σ query on read; never stored; overdraw allowed (negative visible in report) |
| Default-source proposals (COV-02/03, D-04) | API/Backend (service) | Database/Storage | Decision function over the activity chain; reuses chain-walk CTEs (ResolveCommercialContext + new ResolveBeneficiaryUnit) |
| To-cover queue (COV-01/04, D-06) | API/Backend (read-model) | Database/Storage | Approved entries LEFT JOIN Σ allocations; no-source flagged; one self-explaining read-model |
| Write permission (COV-03, D-08) | API/Backend (service) | — | routing.ResolveManagerStage gate — the same BE-014 resolution that approved the entry |
| Audit trail (COV-03, BE-016) | Database/Storage (append-only) | API/Backend | In-tx synchronous audit_logs rows; no UPDATE/DELETE paths |
| Period-close snapshot (COV-04, D-10..12) | Database/Storage (immutable copy) | API/Backend | Frozen entry-level rows written by close tx; reports read the copy; allocations stay editable |
| Beneficiary unit (COV-05) | Database/Storage (column) | API/Backend | Nullable column + upward-walk inheritance; absorption default derives from it |
| ADRs (P-012 accept, BE-0xx draft) | Docs (vault) | — | Decision records in `hourglass-vault/decisions/` |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go (stdlib `net/http`) | 1.26.1 | Server, routing (`mux.HandleFunc("PUT /time-entries/{id}/allocations", ...)`), JSON | Verified in `go.mod` + `cmd/server/main.go`; Go 1.22+ method patterns used everywhere |
| jackc/pgx v5 | v5.10.0 | PostgreSQL access, `pgxpool`, transactions, FOR UPDATE locks | Verified in `go.mod` + all repos; `BeginTx` + in-tx audit precedent in `ticket_repository.go` |
| github.com/google/uuid | v1.6.0 | UUIDs for entities | Verified in `go.mod` + all domains; house style (`uuid.UUID` PKs) |
| stretchr/testify | v1.11.1 | assert/require in all tests | Verified in `go.mod`; used by every test suite |
| testcontainers-go (postgres module) | v0.42.0 | Integration-test DB (`postgres:16-alpine`) | Verified in `go.mod` + `test_setup.go` (`SetupPackageContainer`) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cmd/migrate` | — | Apply `migrations/*.up.sql` / `*.down.sql` | All schema changes (ADR-BE-004); cycle tests call `applyMigrations` |
| `pkg/api` | — | `{ data \| error }` envelope | Every handler response/error |
| `internal/middleware` | — | `Auth`, `GetRole`, `GetOrganizationID`, `GetUserID` | Role/ownership gates (D-08) read claims from context |
| `internal/core/services/routing` | — | BE-014 manager-stage resolution (`ResolveManagerStage`) | D-08 write-permission gate — reuse, do not re-implement |
| `internal/core/ports` GeneralAuditLogRepository | — | `audit_logs` appends (017) | Every coverage change event; in-tx variant needed (see Pattern 4) |
| `scripts/seed_demo.sql` | — | MVP demo data | Dev/demo environments only; **not** loaded by test helpers (Phase 11 Pitfall 5) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Tagged union on allocation row (D-01) | `funding_sources` hub table | Locked — hub table would change the ledger shape, ports, write endpoints; origin precedent makes the union durable |
| Stored bucket counter (D-02) | Derived balance on read | Locked — counter needs write-path coupling; derivation is consistent with D-I |
| Incremental allocation CRUD (D-07) | Replace-set PUT | Locked — incremental writes create partial states; replace-set makes the invariant hold by construction |
| Frozen snapshot at close (D-10) | As-of-close audit replay | Locked by D-10 — replay remains the documented fallback; snapshot is the read-model Phase 17 surfaces consume |
| `financial_cutoff_periods` as close trigger (D-12) | New `POST /coverage/close` | Locked — cutoff stays facts-only (Q10 amendment); relation between the two is planner discretion |

**Installation:**
```bash
# No new packages. Zero dependency changes this phase — verify with `go mod verify` (OK).
```

**Version verification:** All libraries above are present and current in the repo's `go.mod` (Go 1.26.1, pgx v5.10.0, uuid v1.6.0, testify v1.11.1, testcontainers-go v0.42.0 — verified via `go.mod` this session). No registry lookup needed — this phase adds no dependencies.

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
**No `[ASSUMED]` package names:** all names verified in the repo's committed `go.mod`/`go.sum` — no external discovery involved.

## Architecture Patterns

### System Architecture Diagram

```mermaid
flowchart LR
    subgraph API["HTTP layer (thin handlers, pkg/api envelope)"]
        PW["PUT /time-entries/{id}/allocations<br/>(replace-set, D-07)"]
        PR["GET /coverage/proposals/{entry_id}<br/>or queue-with-proposals (discretion)"]
        TQ["GET /coverage/to-cover<br/>(queue read-model, D-06)"]
        BB["GET /coverage/buckets/{contract_id}/balance<br/>(derived, D-02)"]
        CL["POST /coverage/close<br/>(org + period, D-12)"]
        SR["GET /coverage/snapshots/{id}<br/>(frozen read)"]
        AC["PUT /activities/{id}<br/>(+ beneficiary_unit_id, COV-05)"]
    end
    subgraph SVC["Services (invariants, gates)"]
        CS["coverage.Service<br/>Σ invariant · source validation<br/>proposal decision fn (D-04)<br/>manager gate (D-08)"]
        RS["routing.Service (shared BE-014)<br/>ResolveManagerStage"]
    end
    subgraph DB["PostgreSQL (pgx repos)"]
        CAA["coverage_allocations<br/>entry_type+entry_id · source_type<br/>contract_id/unit_id · CHECK<br/>reason / justification"]
        ACT["activities<br/>+ beneficiary_unit_id (018)"]
        CTR["contracts (type/sold_hours)"]
        TE["time_entries (read: approved)"]
        AUL["audit_logs (in-tx rows)"]
        SNA["coverage snapshot tables (020)<br/>close header + entry-level rows"]
    end
    PW --> CS --> RS
    PW -->|"tx: FOR UPDATE entry row<br/>Σ check → replace → audit"| CAA
    PR --> CS -->|"chain walk"| ACT
    PR --> CS -->|"contract_type/sold_hours"| CTR
    TQ --> CS -->|"approved entries LEFT JOIN Σ"| TE
    TQ --> CS --> CAA
    BB --> CS -->|"Σ sold − Σ drawn (read)"| CAA
    CL --> CS -->|"tx: freeze rows + audit"| SNA
    SR --> CS --> SNA
    AC --> CS --> ACT
    CS -->|"in-tx audit rows"| AUL
```

**Read path (D-04 proposal):** entry → activity → upward chain walk for `contract_id` (existing `ResolveCommercialContext` CTE) + `beneficiary_unit_id` (new mirror CTE) → contract `contract_type`/`sold_hours` (016 columns) → proposal row (or "no eligible source" flag). Nothing is persisted — proposals exist only in the response.

### Recommended Project Structure
```
internal/core/domain/coverage/          # NEW: allocation entity, source types, reason vocabulary, errors.go
internal/core/domain/activity/          # EXTEND: BeneficiaryUnitID *uuid.UUID field + request fields
internal/core/ports/                    # EXTEND: coverage_repository.go (NEW), activity resolver additions
internal/core/services/coverage/        # NEW: proposals, queue, replace-set write, close, permission gate
internal/adapters/primary/http/         # EXTEND: coverage_handler.go (NEW), activity_handler.go (beneficiary unit)
internal/adapters/secondary/postgres/   # EXTEND: coverage_repository.go (NEW), activity_repository.go (CTE + scan)
migrations/                             # NEW: 018..020 up/down pairs (see below)
hourglass-vault/decisions/              # ADR-P-012 status → Accepted; ADR-BE-0xx coverage encoding (NEW); _index.md
internal/adapters/secondary/postgres/exported_test_helpers.go  # EXTEND: teardown table list
internal/core/services/testdata/        # EXTEND: MockCoverageRepo (+ MockActivityRepo beneficiary field)
```

**Suggested migration split (numbering continues from 017, per ADR-BE-004):**
- `018_activity_beneficiary_unit.{up,down}.sql` — `ALTER TABLE activities ADD COLUMN beneficiary_unit_id UUID REFERENCES units(id)` (nullable, no backfill) + `idx_activities_beneficiary_unit_id`. No 3VL CHECK needed (single nullable column, like `contract_id`). **Down:** drop column.
- `019_coverage_allocations.{up,down}.sql` — `coverage_allocations` table (shape below). **Down:** `DROP TABLE IF EXISTS coverage_allocations CASCADE;`
- `020_coverage_snapshots.{up,down}.sql` — close header + entry-level snapshot rows (shape below). **Down:** `DROP TABLE IF EXISTS coverage_snapshot_rows CASCADE; DROP TABLE IF EXISTS coverage_period_closes CASCADE;`
Each with a cycle test (up → down → up) in the postgres package, mirroring `TestMigration014..017` self-seed pattern (verified: ontology_extension_migrations_test.go).

### Pattern 1: Tagged-union allocation ledger with a refs-to-type CHECK (D-01)
**What:** `coverage_allocations` mirrors the Phase 11 origin encoding: a `source_type` discriminator + nullable `contract_id`/`unit_id` ref columns + a table CHECK pinning the ref set per type (house rule from ADR-BE-016: "any future discriminator CHECK must follow the same shape"). The **five funding sources are derived semantics of three row-level draws**:
- `source_type='contract'` — covers contract budget (project contract), support bucket (support contract), and service request (zero-value contract). Which of the three it *is* is derived from the referenced contract's `contract_type` + `sold_hours` (D-04). This keeps the ledger self-contained per D-01 ("Service requests are zero-value contracts drawn as `source_type='contract'`").
- `source_type='absorption'` — internal absorption; requires `unit_id` (beneficiary unit) + mandatory `reason`.
- `source_type='transfer'` — cross-project transfer; requires `contract_id` (the other project's contract) + mandatory `justification`.

**When to use:** any "one of several reference shapes" fact (origin precedent, 015).

**Recommended shape for 019 (planner confirms vocabulary):**
```sql
CREATE TABLE coverage_allocations (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id),
    entry_type     VARCHAR(50) NOT NULL,                -- D-K: 'time' only in v0.2
    entry_id       UUID NOT NULL,                       -- polymorphic; service validates
    source_type    VARCHAR(50) NOT NULL,
    contract_id    UUID REFERENCES contracts(id),
    unit_id        UUID REFERENCES units(id),
    hours          DECIMAL(8,2) NOT NULL CHECK (hours > 0),
    reason         VARCHAR(50),                         -- mandatory for absorption (COV-02)
    justification  TEXT,                                -- mandatory for transfer (COV-02)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Refs-to-type CHECK (3VL guard kept for house-style consistency):
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_source_check
    CHECK (source_type IS NULL OR
           (source_type = 'contract'   AND contract_id IS NOT NULL AND unit_id IS NULL) OR
           (source_type = 'absorption' AND unit_id IS NOT NULL AND contract_id IS NULL) OR
           (source_type = 'transfer'   AND contract_id IS NOT NULL AND unit_id IS NULL));
-- Mandatory-field CHECKs:
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_reason_check
    CHECK (source_type <> 'absorption' OR reason IS NOT NULL);
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_justification_check
    CHECK (source_type <> 'transfer' OR justification IS NOT NULL);
-- Closed vocabularies (house style, BE-016):
--   source_type IN ('contract','absorption','transfer')
--   reason IN ('WarrantyBug','UnderEstimate','Goodwill')  [CITED: REQUIREMENTS.md COV-02]
-- Indexes: (org_id), (entry_type, entry_id), (contract_id), (unit_id)
```
**Verified constraint:** `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)` matches `time_entries.hours` exactly — `[VERIFIED: migrations/000_full_schema.up.sql:278]` — quote: `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)`.

**Planner notes:** `entry_id` has no FK (polymorphic — an expense FK would be wrong; service validates entry exists, same org, `status='approved'`, `is_deleted=false`). `entry_type` vocabulary: either a `CHECK (entry_type IN ('time'))` (cheap ALTER when expenses land) or no CHECK with the D-K branch in the service — recommendation: **schema CHECK + service branch**, belt and braces, both documented as the "one extra validation branch" costed in the BE ADR. `reason` CHECK with 3 values (`WarrantyBug`,`UnderEstimate`,`Goodwill`) per COV-02 — the research-note Part 5 fourth value "plain internal" is superseded by COV-02.

### Pattern 2: Chain-driven default-source decision function (D-04) with one extension seam (D-05)
**What:** one function maps (entry → activity chain → funding config) to a proposal. Reuses the verified upward-walk CTE family; no billability-first branch (D-04 locked).

```go
// Decision order (D-04):
// 1. ResolveCommercialContext(activityID) → nearest ancestor contract_id (verified CTE)
// 2. contract found:
//      contract_type = 'support'       → proposal {source_type: contract, contract_id}   (bucket draw)
//      contract_type = 'project':
//        sold_hours = 0 (zero-value)   → proposal {source_type: contract, contract_id}   (service request draw)
//        else                          → proposal {source_type: contract, contract_id}   (budget draw)
// 3. no contract: ResolveBeneficiaryUnit(activityID) (new mirror CTE)
//      found → proposal {source_type: absorption, unit_id: beneficiary}
//      none  → no proposal → queue flag "no eligible source — needs a unit or contract" (D-06)
// Extension seam (D-05): one switch point on ticket kind (activity origin → tickets.kind)
// reserved — chain rules only implemented now.
```
**Verified CTE:** `ResolveCommercialContext` returns `(contract_id, customer_id)` — `[VERIFIED: internal/adapters/secondary/postgres/activity_repository.go:236-264]`. It does **not** return `contract_type`/`sold_hours` — the coverage repo needs an extended resolver (e.g. `ResolveFundingContext` returning contract_id + contract_type + sold_hours, or a JOIN on contracts in the proposal query — planner discretion per CONTEXT "chain-walk CTE reuse"). `ResolveBillability` exists (activity_repository.go:271-314) but is deliberately NOT branched on (D-04).

**Beneficiary unit resolution (COV-05):** new resolver `ResolveBeneficiaryUnit(ctx, activityID) (*uuid.UUID, error)` mirroring the `ResolveCommercialContext` CTE exactly (walk upward while `beneficiary_unit_id IS NULL`, return nearest non-null). ⚠️ **Integration point:** `GetAncestry` lists columns explicitly — `[VERIFIED: internal/adapters/secondary/postgres/activity_repository.go:196-211]` — the SELECT list in the recursion must gain `beneficiary_unit_id`, and the `Activity` struct + `scanActivity` must gain the field (the Phase 11 pattern for origin columns).

### Pattern 3: Atomic replace-set write with in-tx Σ validation under FOR UPDATE (D-07, CR-01 lesson)
**What:** one repo method: begin tx → lock the entry row → re-validate (entry approved, Σ = entry hours) → DELETE all allocation rows for the entry → INSERT the new set → write the audit row in the same tx → commit. Mirrors `refresh_token_repo.go` `Rotate` (`BeginTx` + `defer Rollback` + `Commit`) and the ticket in-tx audit precedent (`insertTicketAudit`, ticket_repository.go:83-110).

**Verified lock precedent:** `SELECT status FROM tickets WHERE id = $1 AND org_id = $2 FOR UPDATE` — `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:250]`; CR-01 closure recorded in STATE.md: "in-tx lock + re-check + status-precondition UPDATE backstop closes it". The coverage write must hold `SELECT ... FROM time_entries WHERE id = $1 AND org_id = $2 FOR UPDATE` and re-check `status = 'approved'` + `is_deleted = false` inside the tx (pool-level pre-checks are fast-fail UX only — the in-tx re-check is the correctness guarantee).

**Σ check:** compare the sum of the request's allocation hours against the entry's `hours`. `hours` is `DECIMAL(8,2)` — the service should compare in cents (multiply by 100 and round) to avoid float64 artifacts, or compute the sum in SQL. At least 1 allocation and at most N rows (1..N per COV-01); each `hours > 0` (schema CHECK).

**Audit row:** `entity_type='coverage_allocation'`, `entity_id=<entry id>`, `action='allocations-set'`, `payload=<full allocation set JSON>`, actor = writer (D-08 manager), comment nullable. Written **in the same tx** — never fire-and-forget (BE-016, mirror tickets; CONTEXT discretion confirms this).

### Pattern 4: In-transaction audit writes via a tx-aware audit helper (BE-016)
**What:** the general `audit_logs` table (017) is written from inside mutator transactions. The Phase 11 precedent is a package-private helper: `insertTicketAudit(ctx, tx pgx.Tx, log *audit.AuditLog) error` — `[VERIFIED: internal/adapters/secondary/postgres/ticket_repository.go:83-110]`, inserting `(id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)` — `[VERIFIED: migrations/017_audit_logs.up.sql:24-34]` with `payload JSONB` from a marshaled `map[string]any`.

**When to use:** every coverage write (replace-set, close). The pool-level `GeneralAuditLogRepository.Create` (verified: audit_log_repository.go) stays for non-transactional needs; the coverage repo adds its own `insertCoverageAudit(ctx, tx, log)` helper — do NOT add a public `CreateTx` to the port unless the planner prefers it (ticket precedent is a private helper; follow it).

### Pattern 5: Frozen snapshot with entry-level rows (D-10/D-11/D-12)
**What:** a close operation writes immutable rows; reports read the copy. Two-table shape recommended (planner discretion on naming):
```sql
-- 020 up (recommended shape):
CREATE TABLE coverage_period_closes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id),
    period_start DATE NOT NULL,
    period_end   DATE NOT NULL,
    closed_by    UUID NOT NULL REFERENCES users(id),
    closed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE coverage_snapshot_rows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    close_id      UUID NOT NULL REFERENCES coverage_period_closes(id) ON DELETE CASCADE,
    entry_id      UUID NOT NULL,            -- polymorphic entry (time in v0.2)
    employee_id   UUID NOT NULL,            -- entry owner at close (D-11 "employee")
    entry_date    DATE NOT NULL,            -- entry_date at close (D-11)
    activity_id   UUID NOT NULL,            -- activity chain snapshot (D-11)
    source_type   VARCHAR(50) NOT NULL,
    contract_id   UUID,                     -- resolved contract at close (frozen)
    unit_id       UUID,                     -- beneficiary unit at close (frozen)
    hours         DECIMAL(8,2) NOT NULL CHECK (hours > 0),
    reason        VARCHAR(50),
    justification TEXT
);
CREATE INDEX idx_coverage_snapshot_rows_close ON coverage_snapshot_rows(close_id);
CREATE INDEX idx_coverage_snapshot_rows_entry ON coverage_snapshot_rows(entry_id);
```
**Semantics:** only entries with `entry_date` in `[period_start, period_end]` are frozen; snapshot rows copy allocation state + the resolved chain (contract/unit refs) as of close — "activity chain snapshot" per D-11. Aggregates (billing, bucket levels, per-unit) are computed from these rows on read (Phase 17). No UPDATE/DELETE paths (ticket precedent). **Overlapping/duplicate close for the same period:** recommend rejecting with a sentinel (409) or upserting by period — planner call (Open Question 6). The close tx also writes one audit row (`entity_type='coverage_period_close'` or `'coverage_allocation'`, `action='coverage-closed'`, payload = period + row count).

### Pattern 6: Manager gate via the shared routing package (D-08)
**What:** the coverage write service mirrors the entry `Approve` gate exactly — `[VERIFIED: internal/core/services/time_entry/time_entry.go:191-193]` — quote: `if !res.RoleGated && !contains(res.ApproverIDs, userID) { return nil, time_entry.ErrForbidden }`, with the resolution from `routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)` (`[VERIFIED: internal/core/services/routing/routing.go:57-104]`, fields `ApproverIDs []uuid.UUID`, `RoleGated bool`, `SkipToFinance bool` — `[VERIFIED: routing.go:41-45]`).
- **Allowed:** actor ∈ `ApproverIDs` (WG manager/delegate or unit manager) OR (`RoleGated` && org role is `manager`).
- **Rejected:** `ErrActivityNotLoggable` from the resolver (commercial activity without anchored WG — no legitimate manager) and the structural self-barrier (`e.UserID == userID` → forbidden — the employee can never allocate their own coverage, mirroring Approve).
- **Finance:** read-only — reads (queue, proposals, snapshots, balance) open to manager+finance; writes manager-only. Role vocabulary verified: `('employee', 'manager', 'finance', 'customer', 'hr')` — `[VERIFIED: migrations/012_staffing_schema.up.sql:41-44]`.

### Recommended endpoint surface (discretion — planner confirms)
```
PUT  /time-entries/{id}/allocations        # D-07 replace-set (1..N rows) — manager gate
GET  /coverage/proposals/{entry_id}        # D-04 computed-on-read proposal (+ current allocations)
GET  /coverage/to-cover                    # D-06 queue read-model (org-scoped; manager+finance)
GET  /coverage/buckets/{contract_id}/balance  # D-02 derived balance (support contracts)
GET  /time-entries/{id}/allocations        # current allocations read-back (optional; queue may embed)
POST /coverage/close                       # D-12 org + period; writes snapshot (+ returns it — confirm)
GET  /coverage/snapshots/{close_id}        # frozen snapshot read (Phase 17 consumes)
GET  /coverage/allocations/{entry_id}/history  # audit stream for the entry (reuse ListHistory analog)
```
Queue read-model rows (D-06): `{entry_id, employee, entry_date, activity, hours, covered_hours, uncovered_hours, proposal?, flag?: "no eligible source — needs a unit or contract", reason?}` — every uncovered approved `time` entry appears; no-source entries are flagged, not omitted.

### Anti-Patterns to Avoid
- **Storing bucket balances or proposal rows:** D-02/D-03/D-I — balance is derived on read; proposals are computed on read; only confirmed allocations persist. A counter column or proposal table is the exact anti-pattern D-02/D-03/D-I forbid.
- **Pool-level-only Σ validation:** CR-01 TOCTOU — the Σ check and status check must re-run inside the mutator tx under the FOR UPDATE entry lock; pool-level checks are fast-fail UX only.
- **Fire-and-forget audit writes:** BE-016 — coverage changes follow the ticket in-tx pattern, not the BE-012 detached-goroutine pattern (CONTEXT discretion).
- **Billability-first branching in the proposal rule:** D-04 locked a single chain-driven function — do not branch on `billable` first.
- **Incremental allocation CRUD:** D-07 — no partial-state endpoints; one save call, replace everything.
- **Locking allocations at close:** D-F/D-10 — the snapshot is immutable, allocations stay editable; no `is_locked` on allocation rows.
- **Reusing `financial_cutoff_periods` as the trigger:** D-12 — new close endpoint; the existing table (migration 000:410-419, org + activity + period + is_locked) stays facts-only per the Q10 amendment.
- **DB triggers for invariants:** no trigger precedent (Phase 11 research); invariants live in services + CHECKs.
- **entry_type without the validation branch:** D-K's cost is one service check rejecting non-`time` — skipping it makes the polymorphism a lie; skipping the schema CHECK makes the vocabulary unchecked. Cost it honestly in the BE ADR.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Transaction management (replace-set, close) | Hand-rolled commit/rollback scaffolding | pgx `pool.BeginTx` + `defer tx.Rollback` (precedent: `ticket_repository.go`) | pgx handles isolation, deferred rollback safety, connection release (verified: pgx autodocs + codebase) |
| Manager-stage routing (D-08) | Re-implement BE-014 precedence | Shared `routing.ResolveManagerStage` | Exists; one routing rule for approvals and coverage (D-08); re-implementation would let rules drift |
| Contract/beneficiary chain walk | New ad-hoc recursion | `ResolveCommercialContext` CTE shape + new `ResolveBeneficiaryUnit` mirror | Verified CTE; same upward-walk semantics (D-3, COV-05) |
| Audit trail | Dual-write or fire-and-forget | In-tx `audit_logs` insert helper (ticket precedent) | TICK-05/COV-03 guarantee: the event stream must not be absent |
| Test database | Ad-hoc SQL fixtures per package | testcontainers `SetupPackageContainer` + `applyMigrations`/`seedOrg`/`seedUser`/`seedActivity` helpers | Exists; one container per package via `sync.Once` |
| Migration runner | Custom DDL tooling | `cmd/migrate` + `readMigration`/`applyMigrations` cycle tests | ADR-BE-004; self-seed cycle pattern verified in ontology_extension_migrations_test.go |
| JSON response envelope | Ad-hoc marshaling | `pkg/api.RespondWithJSON/RespondWithError` | House standard, verified everywhere |
| Authentication | Token parsing in handlers | `middleware.Auth` + claims context (`GetRole`, `GetUserID`, `GetOrganizationID`) | Exists; all route gates read from context |
| Sentinel error mapping | Raw pg errors in handlers | Domain `errors.go` sentinels + `wrapPGError` + handler switch (ADR-BE-001) | Established across all handlers |

**Key insight:** this phase's complexity is almost entirely in *existing* patterns — tagged-union CHECKs (015), in-tx audit (017/ticket repo), FOR UPDATE re-validation (CR-01), chain-walk CTEs (activity repo). The risk is in semantic definition (source_type vocabulary, zero-value contract detection, snapshot table shape, close idempotency), not in technology. No new library is needed; every mechanism has a verified precedent in this codebase.

## Runtime State Inventory

Not applicable — this is not a rename/refactor phase. All changes are additive (new tables, new nullable column, new endpoints). No existing string keys, service names, secrets, or build artifacts change. No stored data carries a renamed identifier; no OS-registered state or env vars are touched.

## Common Pitfalls

### Pitfall 1: Σ-invariant checked outside the transaction (TOCTOU)
**What goes wrong:** the service sums the request allocations and compares against entry hours *before* the mutator tx; two concurrent saves (or an entry-hours change between check and write) pass the check and commit a violating state.
**Why it happens:** Phase 11 CR-01 root cause — pool-level checks before the mutator tx leave a check-then-act window.
**How to avoid:** re-validate inside the tx: `SELECT ... FROM time_entries WHERE id = $1 AND org_id = $2 FOR UPDATE` (verified precedent: ticket_repository.go:250), re-check `status='approved'` + `is_deleted=false` + Σ equality, then DELETE+INSERT the set and audit row, all before COMMIT. Pool-level checks are fast-fail UX only.
**Warning signs:** replace-set unit tests pass while a concurrent-write integration test (CR-01-style, mirroring ticket_repository_test.go:418-506) produces a violating state; Σ compared with raw float64 equality.

### Pitfall 2: CHECK constraints silently passing NULLs (source refs / mandatory reason)
**What goes wrong:** a naive `CHECK (source_type = 'absorption' AND unit_id IS NOT NULL)` is satisfied when `source_type IS NULL` (3VL) — mixed refs (absorption with contract_id set) or missing mandatory reason/justification slip through.
**Why it happens:** PostgreSQL CHECK passes on TRUE **or NULL** — `[CITED: postgresql.org/docs/16/ddl-constraints.html]` ("A check constraint is satisfied if the expression evaluates to true or if the value is null"), cross-verified against migrations 015/016 which guard with `discriminator IS NULL OR (...)`.
**How to avoid:** follow the house rule — pin the *entire* ref set per type with explicit `IS [NOT] NULL` and guard with `source_type IS NULL OR (...)`. Mandatory-field CHECKs: `(source_type <> 'absorption' OR reason IS NOT NULL)` — this pattern fails on `reason = NULL` even when the expression is otherwise NULL-safe, because `FALSE OR ...` is never NULL. Reason vocabulary CHECK: `reason IN ('WarrantyBug','UnderEstimate','Goodwill')` (COV-02).
**Warning signs:** cycle test inserting absorption-without-unit succeeding; transfer-without-justification row in the ledger.

### Pitfall 3: Proposal rule branching on billability first (D-04 violation)
**What goes wrong:** a developer "helpfully" reuses `ResolveBillability` (verified: activity_repository.go:271-314) to branch `billable → contract, non-billable → absorption`, then the chain rules — producing wrong proposals for non-billable work under a contract (support buckets, service requests, warranties) and for billable work without a contract.
**Why it happens:** ADR-P-012 D-1 says "Billability (P-007 D-7) becomes the default-source rule" — but D-04 (locked, CONTEXT) supersedes the wording with the single chain-driven function and explicitly says "No separate billability-first branch".
**How to avoid:** implement exactly the D-04 order (contract → type/value → draw; no contract → beneficiary → absorption; neither → flagged no-source). The billability signal still feeds Phase 17 report surfaces, not the proposal function.
**Warning signs:** proposal for an internal-activity-under-a-support-contract drawn from absorption; proposal for billable work without contract erroring instead of proposing absorption.

### Pitfall 4: Zero-value contract undefined — service requests misproposed
**What goes wrong:** the D-04 rule needs to detect "zero-value contract → contract draw (service request)" but the `contracts` table has **no value column** (verified: migration 000:125-137 — only `km_rate` (DEFAULT 0!), `currency`, `customer_id`, `governance_model`, `is_shared`, `is_active` + 016's `contract_type`/`sold_hours`/`sold_period`). `km_rate = 0` cannot discriminate (it is the default for every contract). If the rule is not pinned, project contracts without sold_hours get drawn as budget and service requests are indistinguishable.
**Why it happens:** "zero-value contract" (F09 US-004) was a v0.1 concept with no schema discriminator; D-J maps it to a contract draw but the detection predicate was never defined.
**How to avoid:** pin the predicate in the plan (Open Question 3). Recommendation: **service request ⇔ `contract_type = 'project' AND sold_hours IS NOT DISTINCT FROM 0`** (project contracts default sold_hours NULL = no commitment; explicit 0 = zero-value service request). A fresh service-request contract is created with `sold_hours = 0`.
**Warning signs:** proposal computed for a service-request contract with `sold_hours = 0` drawn as budget; no test asserting the zero-value predicate.

### Pitfall 5: GetAncestry / scanActivity missing the new beneficiary column
**What goes wrong:** migration 018 adds `activities.beneficiary_unit_id`, but `GetAncestry` (explicit column list — verified activity_repository.go:196-211), the `Activity` struct, `scanActivity`, and `baseActivityQuery` do not carry it — the resolver `ResolveBeneficiaryUnit` returns nil forever, absorption proposals lose their default unit, and the queue flags valid entries "no eligible source".
**Why it happens:** the ancestry SELECT is a hardcoded column list; adding a column to the table silently doesn't propagate to CTE-based reads.
**How to avoid:** extend the struct + SELECT list + scan in the same plan task that lands migration 018 (Phase 11 origin columns were the same pattern). Add a unit test asserting `GetAncestry` returns the beneficiary unit and `ResolveBeneficiaryUnit` inherits from the nearest ancestor.
**Warning signs:** `GetAncestry`-based reads missing the field after 018 applies; resolver returns nil in an integration test despite a seeded beneficiary unit.

### Pitfall 6: To-cover queue counting the wrong entries
**What goes wrong:** the queue includes draft/submitted/rejected entries, deleted entries, or expense entries; or Σ allocs includes allocations on entries outside the org.
**Why it happens:** the eligibility predicate (approved + `is_deleted=false` + `entry_type='time'` + org-scoped) must be exact.
**How to avoid:** `WHERE te.status = 'approved' AND te.is_deleted = false AND te.org_id = $1` LEFT JOIN `COALESCE(SUM(ca.hours),0)`, filter `sum < te.hours`. Status vocabulary verified: `('draft', 'submitted', 'pending_manager', 'pending_finance', 'approved', 'rejected')` — `[VERIFIED: migrations/004_time_entries_status_check.up.sql:3-6]`. Queue must include no-source entries flagged (D-06) — never filter them out.
**Warning signs:** queue shows entries in `pending_finance`; deleted entries appear; a fully-allocated entry (Σ = hours) still listed due to float drift.

### Pitfall 7: Close snapshot reads live rows or locks allocations
**What goes wrong:** Phase 17 reports read live `coverage_allocations` (so a late edit retroactively changes "what we reported for March"), or the close endpoint sets `is_locked` on allocation rows (violating D-F "allocations stay editable indefinitely").
**Why it happens:** D-10's guarantee ("a reported period never changes retroactively") is only real if reports read the frozen copy; the user's payment concern ("since the payment already came we shouldn't be able to modify allocations") tempts a lock, which D-10 explicitly resolves as snapshot-not-lock.
**How to avoid:** close writes the frozen rows; all report/snapshot reads hit `coverage_snapshot_rows`; no lock column exists on allocations. The real lock attaches at the future billing layer (Deferred Ideas).
**Warning signs:** snapshot read endpoint joining live allocations; any `is_locked`/`closed` flag on `coverage_allocations`.

### Pitfall 8: Teardown list and migration numbering not updated
**What goes wrong:** `TeardownTestSchema` (verified: exported_test_helpers.go:81-116) lacks the new tables — cross-package test pollution; new migrations numbered wrong (next is 018, 019, 020 — not a renumber).
**Why it happens:** adding tables without touching the shared teardown; forgetting ADR-BE-004 sequential numbering (007 gap is frozen).
**How to avoid:** add `coverage_snapshot_rows`, `coverage_period_closes`, `coverage_allocations` to the teardown list **before** `time_entries`/`activities` (dependency order); number new migrations 018+. Cycle tests self-seed pre-state inline (verified pattern, ontology_extension_migrations_test.go).
**Warning signs:** repository tests in a later package failing on leftover allocation rows; migration file numbered 019 before 018 exists.

### Pitfall 9: Bucket balance formula drift (D-02)
**What goes wrong:** balance computed as `sold_hours − Σ allocations` at the service level while the report computes it differently, or allocations drawn from a *different* contract's bucket counted against this one.
**Why it happens:** D-02's formula is one line but the JOIN must be exact: `Σ allocations WHERE contract_id = <bucket contract>` (any `source_type` — including transfers — draws the target contract's balance; the query must scope by `contract_id`, not `source_type`).
**How to avoid:** one repo method `BucketBalance(ctx, contractID)` used by both the balance endpoint and Phase 17 read-models: `SELECT c.sold_hours - COALESCE(SUM(ca.hours), 0) FROM contracts c LEFT JOIN coverage_allocations ca ON ca.contract_id = c.id WHERE c.id = $1 GROUP BY c.sold_hours`. Negative allowed (D-03). Note: the per-period multiplication question (`sold_hours × periods elapsed` vs raw `sold_hours`) is flagged in Open Question 5 — D-02's wording is raw `Σ sold_hours`.
**Warning signs:** balance endpoint and report disagree; overdraw rejected anywhere (D-03 forbids a gate).

## Code Examples

Verified patterns from official sources + codebase precedents (read this session):

### Common Operation 1: Replace-set write with in-tx Σ validation + audit (repo shape)
```go
// Source shape: refresh_token_repo.go Rotate (BeginTx) + ticket_repository.go
// insertTicketAudit (in-tx audit) — both read this session.
tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
if err != nil {
    return nil, fmt.Errorf("begin replace allocations: %w", err)
}
defer func() { _ = tx.Rollback(ctx) }() // safe even after Commit

// 1. Lock the entry row and re-validate (CR-01: in-tx check, never pool-only)
var hours float64
err = tx.QueryRow(ctx,
    `SELECT hours FROM time_entries
      WHERE id = $1 AND org_id = $2 AND status = 'approved' AND is_deleted = false
      FOR UPDATE`, entryID, orgID).Scan(&hours)
if errors.Is(err, pgx.ErrNoRows) {
    return nil, coverage.ErrEntryNotCoverable
}
// 2. Σ validation (cents arithmetic avoids float64 artifacts)
// 3. DELETE FROM coverage_allocations WHERE entry_id = $1 AND entry_type = 'time'
// 4. INSERT each allocation row (source_type + refs + hours + reason/justification)
// 5. insertCoverageAudit(ctx, tx, &audit.AuditLog{
//        OrgID: orgID, EntityType: "coverage_allocation", EntityID: entryID,
//        Action: "allocations-set", ActorID: &userID, Payload: map[string]any{...}, ...})
if err := tx.Commit(ctx); err != nil {
    return nil, fmt.Errorf("commit replace allocations: %w", err)
}
```

### Common Operation 2: Chain-walk CTE for the default-source rule (extended resolver)
```sql
-- Source shape: ResolveCommercialContext (activity_repository.go:237-249) —
-- extend the SELECT with contract_type/sold_hours via JOIN; new resolver for
-- beneficiary_unit_id mirrors the same CTE with beneficiary_unit_id.
WITH RECURSIVE chain AS (
    SELECT id, parent_id, contract_id, beneficiary_unit_id FROM activities WHERE id = $1
    UNION ALL
    SELECT a.id, a.parent_id, a.contract_id, a.beneficiary_unit_id
    FROM activities a
    INNER JOIN chain c ON a.id = c.parent_id
    WHERE c.contract_id IS NULL OR c.beneficiary_unit_id IS NULL
)
SELECT c.contract_id, ct.contract_type, ct.sold_hours, c.beneficiary_unit_id
FROM chain c
LEFT JOIN contracts ct ON ct.id = c.contract_id
WHERE c.contract_id IS NOT NULL OR c.beneficiary_unit_id IS NOT NULL
LIMIT 1
```

### Common Operation 3: Manager gate for the allocations write (D-08)
```go
// Source: time_entry.go Approve gate (L191-193) + routing.go (L57-104) — verbatim pattern.
if e.UserID == userID {
    return nil, coverage.ErrForbidden // structural self-barrier, mirrors Approve
}
res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
if err != nil {
    if errors.Is(err, activity.ErrActivityNotLoggable) {
        return nil, coverage.ErrForbidden
    }
    return nil, err
}
if !res.RoleGated && !contains(res.ApproverIDs, userID) {
    return nil, coverage.ErrForbidden
}
// RoleGated && role == "manager" passes; finance/employee rejected for writes.
```

### Common Operation 4: Migration cycle-test skeleton (self-seed, 018)
```go
// Source: ontology_extension_migrations_test.go:26-84 (TestMigration014 shape) — read this session.
func TestMigration018_ActivityBeneficiaryUnit_UpDownUpCycle(t *testing.T) {
    pool := TestPool(t)
    t.Cleanup(func() { TeardownTestSchema(t, pool) })
    ctx := context.Background()
    now := time.Now()
    up018 := readMigration(t, "018_activity_beneficiary_unit.up.sql")
    down018 := readMigration(t, "018_activity_beneficiary_unit.down.sql")
    applyMigrations(t, pool, true, "019_coverage_allocations.up.sql", "020_coverage_snapshots.up.sql")
    orgID := seedOrg(t, pool, now)
    seedActivityKind(t, pool, orgID, "engagement")
    unitID := seedUnit(t, pool, orgID, now)
    activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
    // --- UP ---
    _, err := pool.Exec(ctx, up018)
    require.NoError(t, err)
    // set + read beneficiary_unit_id; assert null default passes (legacy rows valid)
    // --- DOWN --- drop column; --- UP again --- re-apply green
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Coverage = editing/duplicating time entries (Excel practice) | Allocation ledger: facts vs decisions (ADR-P-012) | This phase | The corrupt practice becomes impossible to perform silently (P-012 consequences) |
| No coverage read-model | To-cover queue — uncovered work is a visible state | This phase | "Never an implicit gap" (COV-01/D-06) |
| `time_entry_approvals` as the only audit trail | General `audit_logs` (017) consumed by coverage in-tx | Phase 11 → consumed this phase | One event stream; COV-03 audit-first |
| `financial_cutoff_periods` locks entries (facts) | Coverage close = frozen snapshot, not a lock (Q10 amendment, D-10) | This phase | A reported period never changes retroactively; allocations stay editable |
| Stored counters / proposal tables (anti-pattern) | Derived balances (D-02) + computed-on-read proposals (D-I) | This phase | No staleness window, no write-path coupling |
| `billable` flag branching (P-007 D-7 wording) | Single chain-driven default-source rule (D-04) | This phase (locked) | Contract type/value + beneficiary unit drive the proposal; billability stays a report signal |

**Deprecated/outdated:**
- **`financial_cutoff_periods` as an allocation lock:** superseded by D-12 — the table stays facts-only; the coverage close endpoint is separate (how they relate is planner discretion).
- **"Plain internal" as an absorption reason:** COV-02 lists exactly `WarrantyBug · UnderEstimate · Goodwill` — the research-note Part 5 fourth value is superseded.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `source_type` vocabulary = 3 values (`contract`/`absorption`/`transfer`) with the five funding sources derived from the referenced contract (`contract_type` + `sold_hours`); refs-to-type CHECK pins exactly one ref per type | Pattern 1 | If the planner prefers 5 explicit values, the CHECK + proposal function + queue flags change shape (additive vocabulary, still one migration) — confirm at plan time |
| A2 | Absorption `reason` vocabulary = exactly `WarrantyBug`/`UnderEstimate`/`Goodwill` (COV-02 wording) with a DB CHECK; transfer `justification` is free-text mandatory | Pattern 1 | If "plain internal" (research Part 5) is wanted, CHECK gains a 4th value — additive ALTER |
| A3 | Zero-value contract ⇔ `contract_type='project' AND sold_hours IS NOT DISTINCT FROM 0` (no value column exists on contracts) | Pitfall 4 / Open Q3 | If detection differs (e.g. km_rate), the D-04 branch changes — a pure code predicate, cheap to swap, but must be pinned or service requests mispropose |
| A4 | `entry_id` on `coverage_allocations` has no FK (polymorphic); service validates entry exists/same-org/approved/not-deleted; `entry_type` has a schema CHECK `('time')` + service branch (D-K) | Pattern 1 | If an FK to `time_entries` is preferred, the migration gains one FK and expenses (COV-06) need a later ALTER — reversible either way |
| A5 | Snapshot = two tables (close header + entry-level rows) with frozen resolved refs (contract/unit) as the "activity chain snapshot"; aggregates computed on read | Pattern 5 / Open Q6 | Naming/shape is discretion; D-11 requires entry-level rows and no aggregates — both hold under this shape |
| A6 | Duplicate/overlapping close for the same period is rejected (sentinel → 409) | Pattern 5 / Open Q6 | If overwrite-by-period is preferred, the close table gains a unique (org, period) key — planner call |
| A7 | Coverage audit rows use `entity_type='coverage_allocation'` (per entry) with actions `allocations-set` / `coverage-closed`; close rows addressed to the close id | Pattern 3/5 | Entity vocabulary is free-form text in `audit_logs` (VARCHAR(50), no CHECK) — any convention works; pin it for Phase 17 history reads |
| A8 | Bucket balance = raw `sold_hours − Σ allocations` per D-02 (no per-period multiplication); overdraw → negative, never rejected | Pitfall 9 / Open Q5 | If period-scaling (`sold_hours × periods elapsed`) is intended, the balance endpoint and Phase 17 bucket surface change — flag for user confirmation |
| A9 | The manager gate rejects the entry owner (structural self-barrier mirroring `Approve`) and re-resolves routing at write time | Pattern 6 | If the owner may allocate own coverage (self-direction territory, Phase 13), the gate loosens later — additive |
| A10 | Proposal = single allocation of the full entry hours to the default source (the "one click" default); splits are manager edits at confirm time | Pattern 2 | If proposals must pre-split (e.g. contract + absorption for over-budget work), the proposal shape gains complexity — D-04's wording supports the single-default reading |

## Open Questions

1. **`source_type` vocabulary: 3 values (derived five sources) vs 5 explicit values (A1)**
   - What we know: D-01 requires `source_type` + `contract_id`/`unit_id` refs; D-04 maps the five funding sources to contract-type-driven draws; the discretion area says "How `source_type` vocabulary is CHECK-enforced (house style)".
   - What's unclear: whether the row carries the *draw kind* (`contract`/`absorption`/`transfer` — five sources derived from the referenced contract) or five explicit values.
   - Recommendation: **3 values, derived semantics** — matches D-04's decision function exactly ("zero-value contract drawn as `source_type='contract'`" — D-01 wording confirms 'contract' is the row-level value for all three contract-based sources), keeps the CHECK small, and the derivation (JOIN contracts) is where reports already go.

2. **Proposal read-path exposure: per-entry endpoint vs queue-with-proposals (discretion)**
   - What we know: D-06 requires one queue read-model including flagged no-source rows; SURF-01 (Phase 17) needs per-entry proposals.
   - What's unclear: one combined read (queue rows embed the computed proposal) vs separate `GET /coverage/proposals/{entry_id}`.
   - Recommendation: **both, sharing the service decision function** — a per-entry proposal endpoint (cheap, used by the future allocation screen) and the queue read-model that embeds the proposal when derivable (D-06 self-explaining rows). The planner may fold them into one combined read if the per-entry endpoint is considered redundant with the queue.

3. **Zero-value contract detection predicate (A3)**
   - What we know: contracts have no value column (verified 000:125-137); 016 added `contract_type`/`sold_hours`/`sold_period`; D-J = service requests are zero-value contracts.
   - What's unclear: what makes a contract "zero-value" in the schema.
   - Recommendation: `contract_type='project' AND sold_hours IS NOT DISTINCT FROM 0` — needs user confirmation in discuss/plan (A3).

4. **Does the close endpoint return the snapshot data in one call? (D-12 discretion)**
   - What we know: "D-12 'new close endpoint' implies close + report in one call — confirm during planning" (CONTEXT).
   - What's unclear: response = just the close id, or the full frozen snapshot rows.
   - Recommendation: **return the snapshot rows in the close response** (one round trip; Phase 17 prototypes can drop it later). Idempotency semantics per A6.

5. **Bucket balance: raw sold_hours vs period-scaled (A8)**
   - What we know: D-02 wording is "Σ sold_hours on the support contract − Σ allocations drawn from it"; `sold_period` (month/quarter/year) exists on support contracts; D-P carry-over/no-expiry.
   - What's unclear: whether a 12-month support contract with sold_hours=8/month has balance 8h (raw) or 96h (scaled).
   - Recommendation: **raw per D-02 for v0.2**, documented in the BE ADR; period-scaling, if wanted, lands with the Phase 17 bucket surface (a read-model change, not a migration).

6. **Close idempotency and snapshot shape (A5/A6)**
   - What we know: D-11 entry-level rows, no aggregates; a new close endpoint scoped to org + period.
   - What's unclear: second close of the same period — reject, overwrite, or append?
   - Recommendation: **reject overlapping close with 409** (a reported period is immutable — D-10's guarantee argues against overwrite); snapshot read returns the close header + rows.

7. **How `coverage/close` relates to `financial_cutoff_periods` (D-12 discretion)**
   - What we know: the existing table (000:410-419) is facts-only per the Q10 amendment; user chose a separate close endpoint.
   - What's unclear: whether the coverage close should *also* write a `financial_cutoff_periods` row (join the mechanisms) or stay fully separate.
   - Recommendation: **stay separate this phase** — the coverage close is org + period-scoped; `financial_cutoff_periods` is per-activity and facts-oriented. Revisit when entry cutoff semantics land.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend build/tests | ✓ | 1.26.1 (matches go.mod) | — |
| Docker | testcontainers integration tests (postgres:16-alpine) | ✓ | 29.4.0 | Tests skip when Docker absent (t.Skip in `SetupPackageContainer`) |
| PostgreSQL | Dev runtime (docker-compose service); tests use containers | ✓ (docker-compose) | postgres:15 via compose / 16-alpine in tests | Testcontainers covers tests; dev DB via `docker-compose up` |
| Node/Bun | Not needed this phase (backend-only) | ✓ | node v22.23.1 / bun 1.3.13 | — |
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
| Quick run command | `go test ./internal/core/services/coverage/ ./internal/core/services/activity/ -count=1` |
| Full suite command | `make test` (go test -v ./...) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| COV-01 | Replace-set write: Σ = entry hours enforced; ≠ rejected; 1..N rows; partial state impossible | unit + integration (tx) | `go test ./internal/core/services/coverage/ -run TestService_ReplaceAllocations -count=1` | ❌ Wave 0 |
| COV-01 | Concurrent replace-set: FOR UPDATE lock serializes; violating state never commits | integration (CR-01-style, mirrors ticket_repository_test.go:418-506) | `go test ./internal/adapters/secondary/postgres/ -run TestCoverageReplace_Concurrent -count=1` | ❌ Wave 0 |
| COV-02 | Five funding sources produce correct allocations; reason/justification mandatory; bucket overdraw allowed | unit (decision fn) + integration (CHECK) | `go test ./internal/core/services/coverage/ -run TestService_Propose -count=1` | ❌ Wave 0 |
| COV-02 | Derived bucket balance = sold_hours − Σ drawn; negative allowed | integration (repo) | `go test ./internal/adapters/secondary/postgres/ -run TestCoverageRepository_BucketBalance -count=1` | ❌ Wave 0 |
| COV-03 | Manager gate: WG manager/delegate or unit manager writes; employee/finance/customer rejected; finance reads | unit + handler integration | `go test ./internal/adapters/primary/http/ -run TestCoverageHandler -count=1` | ❌ Wave 0 |
| COV-03 | Every allocation change audit-logged in-tx (count + payload assertions) | integration | `go test ./internal/adapters/secondary/postgres/ -run TestCoverageRepository_Audit -count=1` | ❌ Wave 0 |
| COV-04 | Close freezes snapshot rows; later allocation edits do not change the snapshot | integration | `go test ./internal/adapters/secondary/postgres/ -run TestCoverageRepository_Close -count=1` | ❌ Wave 0 |
| COV-05 | `entry_type='expense'` rejected (D-K branch); beneficiary unit inherited upward; absorption defaults from it | unit + integration | `go test ./internal/core/services/coverage/ ./internal/adapters/secondary/postgres/ -count=1` | ❌ Wave 0 |
| Migrations | 018/019/020 up → down → up cycles + 3VL/mandatory-field CHECK assertions | integration (cycle) | `go test ./internal/adapters/secondary/postgres/ -run TestMigration018 -count=1` (+019/020) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/services/coverage/ ./internal/adapters/secondary/postgres/ -count=1`
- **Per wave merge:** `make test` (full suite)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/core/services/coverage/coverage_test.go` — service unit tests (proposals, Σ, permission) with `testdata.MockCoverageRepo`
- [ ] `internal/core/services/testdata/` — MockCoverageRepo added (mirror MockTicketRepo pattern)
- [ ] `internal/adapters/secondary/postgres/coverage_repository_test.go` — repo integration tests (replace-set, balance, close, history)
- [ ] `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` — cycle tests TestMigration018..020 (self-seed)
- [ ] `internal/adapters/primary/http/coverage_handler_test.go` — handler integration (permission matrix, envelope, sentinel mapping)
- [ ] `internal/adapters/secondary/postgres/exported_test_helpers.go` — teardown list += coverage tables (Pitfall 8)

## Security Domain

> `security_enforcement` absent from `.planning/config.json` → treated as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (unchanged) | Existing `middleware.Auth` + JWT cookie flow — no new auth surface |
| V3 Session Management | no (unchanged) | Existing refresh rotation — untouched |
| V4 Access Control | yes | D-08 manager gate via shared `routing.ResolveManagerStage` (ApproverIDs / RoleGated); finance read-only; customer rejected; role claims from `middleware.GetRole` (verified pattern time_entry.go:191-193) |
| V5 Input Validation | yes | Σ invariant re-validated in-tx (CR-01); same-org validation of `contract_id`/`unit_id` refs; `entry_type='time'` branch (D-K); `hours > 0` CHECK; reason/justification mandatory CHECKs; DB CHECK vocabularies (house style) |
| V6 Cryptography | no | No new crypto — none introduced by an allocation ledger |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| TOCTOU on Σ-invariant (concurrent replace-set) | Tampering | In-tx re-validation under `SELECT ... FOR UPDATE` on the entry row (CR-01 closure; verified precedent ticket_repository.go:250) |
| Cross-org reference injection (contract_id/unit_id from another org) | Spoofing / Tampering | Service-level same-org existence checks on every ref (Phase 11 D-02 pattern); org_id column on the ledger |
| Unauthorized allocation write (non-manager, owner, customer) | Elevation of privilege | D-08 gate: actor ∈ resolved approver set OR role-gated manager; structural self-barrier (owner never writes own coverage); finance read-only |
| Snapshot tampering / retroactive report change | Tampering / Repudiation | Append-only snapshot tables (no UPDATE/DELETE endpoints or repo methods — ticket precedent); in-tx audit rows with actor + payload |
| Audit trail loss | Repudiation | Synchronous in-tx audit writes (BE-016) — never fire-and-forget; rollback on audit failure |
| Non-`time` entry_type smuggling (D-K) | Tampering | Service branch rejecting `entry_type != 'time'` + schema CHECK — the costed validation branch |

## Sources

### Primary (HIGH confidence)
- Codebase (read this session): `migrations/000_full_schema.up.sql` (time_entries/contracts/units/financial_cutoff_periods), `004` (status vocabulary), `011` (activities), `012` (roles), `015` (origin tagged-union CHECK), `016` (contract_type/sold_hours), `017` (audit_logs); `internal/core/services/routing/routing.go`, `internal/core/services/time_entry/time_entry.go` (Approve gate), `internal/adapters/secondary/postgres/activity_repository.go` (GetAncestry/ResolveCommercialContext/ResolveBillability), `ticket_repository.go` (BeginTx + in-tx audit + FOR UPDATE), `audit_log_repository.go`, `exported_test_helpers.go` (teardown/seed helpers), `ontology_extension_migrations_test.go` (cycle-test self-seed), `cmd/server/main.go` (route wiring), `internal/core/ports/audit_log_repository.go`, `internal/core/domain/audit/audit.go`
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` (Parts 5-8, 12, 13 A-F, Q10 amendment) — record of truth
- `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions.md` (D-1..D-6, snapshot-not-lock)
- `hourglass-vault/decisions/backend/ADR-BE-016` (3VL house rule, in-tx audit) and `ADR-BE-014` (routing)
- `.planning/phases/12.../12-CONTEXT.md` (D-01..D-12 locked decisions) and `12-DISCUSSION-LOG.md`
- [VERIFIED: postgresql.org/docs/16/ddl-constraints.html + migrations/015] — CHECK 3VL semantics cross-checked against code
- [VERIFIED: pgx autodocs (github.com/jackc/pgx) + ticket_repository.go] — BeginTx/Commit/Rollback + FOR UPDATE cross-checked against code

### Secondary (MEDIUM confidence)
- [CITED: postgresql.org/docs/16/ddl-constraints.html] — CHECK passes on TRUE or NULL (official docs; tier MEDIUM per classify-confidence seam, elevated by codebase cross-check)
- [CITED: pgx transaction autodocs] — BeginTx/Commit/Rollback patterns (tier MEDIUM per seam, elevated by codebase cross-check)
- [CITED: .planning/REQUIREMENTS.md:29 (COV-02)] — absorption reason vocabulary `WarrantyBug · UnderEstimate · Goodwill`
- [CITED: hourglass-vault/01-Features/F09-Contracts.md US-004] — zero-value contracts allowed (no schema discriminator exists — see Open Question 3)

### Tertiary (LOW confidence)
- None — no training-memory-only claims are load-bearing; every schema/pattern claim was verified against files read this session or official docs.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified in `go.mod`; zero new dependencies
- Architecture: HIGH — every pattern verified against a codebase precedent read this session (tagged-union CHECK 015, in-tx audit 017, FOR UPDATE CR-01, chain-walk CTEs, routing gate)
- Pitfalls: HIGH — derived from Phase 11 execution history (CR-01, Pitfall 1/2/8) + schema shapes verified this session; open semantic points flagged in Assumptions Log / Open Questions at MEDIUM

**Research date:** 2026-08-07
**Valid until:** 2026-09-06 (stable stack; re-check go.mod only if new deps are proposed)

