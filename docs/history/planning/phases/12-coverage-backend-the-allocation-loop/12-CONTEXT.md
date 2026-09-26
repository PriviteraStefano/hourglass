# Phase 12: Coverage Backend — The Allocation Loop - Context

**Gathered:** 2026-08-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend-only phase (no UI). The coverage plane works server-side:

- **Coverage allocations** — per-entry allocation ledger with the Σ invariant (Σ allocations = entry hours), only `time` entries coverable in v0.2 (D-K polymorphic `entry_type` + `entry_id`)
- **Funding sources** — all five work: contract budget, support bucket (hours-only, carry-over, no expiry, overlapping), service request (zero-value contract), internal absorption (mandatory reason WarrantyBug/UnderEstimate/Goodwill), cross-project transfer (explicit justification)
- **To-cover queue** — uncovered approved entries are a visible state, never an implicit gap
- **Proposals computed on read** — no proposal table; only confirmed allocations persist (D-I)
- **One-step manager confirmation** — no finance chain; every allocation change audit-logged via the general `audit_logs` table (D-L, BE-016)
- **Period-close snapshots** — a reported period never changes retroactively; allocations stay editable indefinitely (D-F)
- **Beneficiary unit** — nullable on activities, inherited downward like `contract_id`; absorption sources default from it (COV-05)
- **ADRs** — ADR-P-012 accepted (vault draft is Proposed); BE encoding ADR written this phase (incl. D-K polymorphic validation cost)

Deliverables are API endpoints + migrations + domain/ports/services/adapters + integration tests. All migrations append-only per ADR-BE-004 with up/down pairs + cycle tests.

</domain>

<decisions>
## Implementation Decisions

### Funding-source storage shape
- **D-01:** **Tagged union on the allocation row** — `coverage_allocations` carries `source_type` + nullable ref columns (`contract_id`, `unit_id`) with a CHECK matching refs to type, mirroring the Phase 11 origin pattern (D-01 of Phase 11: discriminator + nullable columns + three-valued-logic CHECK guard). No `funding_sources` hub table. Service requests are zero-value contracts drawn as `source_type='contract'`. — **Reversibility:** costly — a hub table could be introduced later but the ledger shape, ports, and write endpoints would all need to change; the origin precedent makes the tagged union the durable choice.
- **D-02:** **Bucket balance derived, never stored** — support bucket balance = Σ sold_hours on the support contract − Σ allocations drawn from it, computed on read. Consistent with D-I (never store what can be computed); carry-over/no-expiry (D-P) falls out naturally because the ledger is cumulative. No counter column, no write-path lock on the bucket row.
- **D-03:** **Overdraw allowed, visible in report** — allocations may push a bucket balance negative; the invariant Σ = entry hours is never a bucket gate. D-C ("no anti-abuse control on support buckets"); the report is the control.

### Proposal eligibility rules
- **D-04:** **Single chain-driven default-source rule** — one decision function over the entry's activity chain (the D-3 upward walk resolving `contract_id`, same CTE used for billability): contract found & type `project` → its contract budget; type `support` → its bucket; zero-value contract → contract draw (service request); no contract → internal absorption with the beneficiary unit (inherited downward like `contract_id`). No separate billability-first branch.
- **D-05:** **Ticket-kind → funding eligibility is an extension point only** — D-H says kinds drive eligibility, but no kind→source matrix was ever defined. Build the proposal function with one extension seam; implement only the chain rules (contract vs bucket vs absorption) now. Proposals are computed on read, so later kind rules are a code change, not a migration.
- **D-06:** **To-cover queue includes no-source entries, flagged** — every uncovered approved `time` entry appears (Σ alloc < hours). Entries with no derivable source (activity with neither contract nor beneficiary unit) appear in the same queue flagged "no eligible source — manual" with the reason. One read-model, self-explaining rows; the "never an implicit gap" rule holds for the no-source case.

### Write semantics & permission
- **D-07:** **Replace-set per entry, atomic** — one save call (e.g. PUT `/time-entries/{id}/allocations`) takes the full allocation set (1..N rows), the service validates Σ = entry hours inside the transaction, and replaces everything atomically. No incremental CRUD endpoints. Matches the week-1 flow: system proposes → manager confirms whole split or edits it → one save. Invariant checked by construction, no partial states.
- **D-08:** **Writer = the entry's own manager** — permission resolved via BE-014 routing (entry's activity chain → WG → manager), the same resolution that approved the entry. Not any org manager; finance is read-only (D-L: overview through reports + audit trail, no second confirm step).
- **D-09:** **No correction handling** — coverage only ever sees approved `time` entries with `hours > 0` (schema CHECK already forbids negatives). Compensating entries do not exist in the codebase (`created_from_entry_id` is schema-only); the D-13 "net-of-compensations" dismissal-guard swap has nothing to swap to — leave the guard on raw Σ. If a correction mechanism ever lands, coverage interaction must be designed then (see Deferred Ideas). — **Reversibility:** reversible — additive if corrections ever arrive.

### Snapshot mechanics
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Ontology research (record of truth)
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` — All decisions D-A…D-P closed here. Specifically: Part 5 (funding sources), Part 6 (monthly rhythm), Part 7 (coverage decision flow + worked example), Part 8 (beneficiary unit), Part 12 (D-F editable, D-H bucket mechanics + beneficiary inheritance, D-I proposals on read, D-J zero-value contracts, D-K polymorphism), Part 13 (D-L one-step manager confirm, D-P bucket carry-over/no-expiry, F snapshot mechanics backend-only), Q10 amendment (snapshot-not-lock)

### ADRs (vault decisions)
- `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md` — Accepted this phase. Two planes + Σ invariant, to-cover queue, D-1..D-6, snapshot-not-lock; the record the BE encoding ADR operationalizes.
- `hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md` — Phase 11 encoding: `audit_logs` table (017) reused for every coverage change (in-tx synchronous writes, payload JSONB); CHECK-guard house rule; referenced for shape conventions.
- `hourglass-vault/decisions/backend/ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution.md` — Resolution used for D-08 (writer = entry's own manager via activity → WG → manager).
- `hourglass-vault/decisions/backend/ADR-BE-004` — Append-only migrations rule (referenced; file under `hourglass-vault/decisions/backend/`)
- New ADR drafted this phase: **ADR-BE-0xx — Coverage encoding** (schema, proposal computation, snapshot mechanics, D-K polymorphic validation cost — the "one extra validation branch" must be costed honestly per ROADMAP)
- Related (context): `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md`, `hourglass-vault/decisions/project/ADR-P-007 — Activity Ontology.md` (D-3 contract inheritance, D-7 billability), `hourglass-vault/decisions/project/ADR-P-013 — Origins.md`

### Milestone docs
- `.planning/ROADMAP.md` — Phase 12 entry: goal, requirements (COV-01..05), 8 success criteria
- `.planning/REQUIREMENTS.md` — COV-01..05 requirement text

### Codebase (read-only context)
- `migrations/016_contract_sold_hours.up.sql` — `contract_type`/`sold_hours`/`sold_period` on contracts: support buckets ARE support-type contracts; `contracts_sold_check` CHECK-guard precedent
- `migrations/017_audit_logs.up.sql` — general audit table for coverage change events (in-tx writes, payload JSONB)
- `migrations/011_activity_ontology.up.sql` — `activities.contract_id` nullable, inherited upward (D-3) — the chain-walk CTE the default-source rule reuses; `beneficiary_unit_id` (nullable, inherited downward) lands here
- `migrations/000_full_schema.up.sql` — `time_entries` (hours DECIMAL(8,2) CHECK > 0, `created_from_entry_id`), `financial_cutoff_periods` (NOT the close trigger per D-12), `contracts`
- `internal/core/domain/activity/activity.go` — Activity entity; beneficiary unit field + validation extends this package
- `internal/adapters/secondary/postgres/time_entry_repository.go` — entry read paths + `created_from_entry_id` handling (corrections excluded from coverage per D-09)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `migrations/017_audit_logs.up.sql` + its repo — every allocation write (create/replace/close) audit-logged in-tx with payload, mirroring the Phase 11 ticket pattern (BE-016)
- Phase 11 origin encoding (`migrations/015_activity_origins.up.sql`) — the discriminator + nullable refs + three-valued-logic CHECK pattern D-01 copies for `coverage_allocations`
- Approval-routing resolution (BE-014, activity → WG → manager) — reused for the D-08 write-permission gate
- Activity chain-walk CTE (contract_id upward inheritance, D-3) — reused for the D-04 default-source derivation; beneficiary-unit downward inheritance mirrors `contract_id` (COV-05)
- `pkg/api/response.go` envelope, sentinel errors in `internal/core/domain/*/errors.go`, `wrapPGError` in postgres adapters

### Established Patterns
- Hexagonal: domain → ports → services → HTTP handlers → postgres repos; services own invariants, DB owns shapes (CHECK constraints)
- Hand-written SQL with pgx, no ORM; migrations append-only with up/down pairs + cycle tests
- Status/source vocabulary enforced via DB CHECK constraints (house style)
- API response envelope `{ data | error }` via `pkg/api/response.go`
- Integration tests via testcontainers-go; per-package test suites
- Concurrent-write safety: re-validate invariants inside the mutator tx under FOR UPDATE locks (Phase 11 CR-01 lesson) — the Σ-invariant check is a replace-all tx, single statement set, but the write path must still hold the entry row lock

### Integration Points
- New: coverage service + routes registered in `cmd/server/main.go` (Go 1.22+ pattern)
- `time_entries` approved-state read + `activities` chain — inputs to proposals and the to-cover queue
- `contracts` (016: support vs project) + `activities.beneficiary_unit_id` — funding targets
- `audit_logs` (017) — every coverage change event
- Phase 17 surfaces read the read-models built here (allocation screen, to-cover queue, own-coverage, bucket balance, per-unit report); no P-011 revision until prototypes land

</code_context>

<specifics>
## Specific Ideas

- **D-10 rationale (user's own words on the payment concern):** "since the payment already came we shouldn't be able to modify allocations, we can in the mean time. This is the sort of logical bugs that come out when there isn't a clear system." — resolved: frozen snapshot now; when billing/invoicing lands later, invoices MUST read the frozen snapshot and the real allocation lock attaches at the billing layer, not the coverage layer (see Deferred Ideas)
- **No-source queue entry (user-confirmed example):** Entry B (6h study work, activity has no contract and no beneficiary unit) appears in the same to-cover queue as Entry A (8h → contract proposal), flagged "no eligible source — needs a unit or contract", hours shown uncovered
- **User confirmed skip of correction handling after the negative-hours example was explained** — the hours CHECK > 0 constraint makes negative compensation entries impossible; D-13's net-of-compensations guard swap is vacuous until a correction mechanism exists

</specifics>

<deferred>
## Deferred Ideas

- **Correction/compensation entries** — `created_from_entry_id` is schema-only; no mechanism exists. When/if one lands, design the coverage interaction then (and re-evaluate the D-13 dismissal-guard computation). Not Phase 12.
- **Billing/invoicing lock (v2+)** — when an invoicing module lands, invoices read the frozen snapshot and THAT is where an allocation lock attaches for paid periods. The snapshot schema (D-11) must not preclude it.

</deferred>

---

*Phase: 12-Coverage Backend — The Allocation Loop*
*Context gathered: 2026-08-07*
