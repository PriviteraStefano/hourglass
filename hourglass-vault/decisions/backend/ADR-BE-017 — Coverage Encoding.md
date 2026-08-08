---
tags: ["adr", "backend", "schema", "coverage", "allocations", "snapshots"]
---

# ADR-BE-017 — Coverage Encoding

**Status:** Accepted
**Date:** 2026-08-07
**Code:** `migrations/018…020`, `internal/core/services/coverage/`, `internal/core/ports/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`
**Operationalizes:** [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] (D-1..D-6, snapshot-not-lock) · **Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 5–8, 12, 13 A–F, Q10 amendment) · **Extends:** [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] (3VL CHECK house rule, in-tx synchronous audit writes) · **Gates on:** [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] (D-08 writer gate) · **Resolves:** Phase 12 semantic resolutions (research OQ1…OQ7), locked decisions D-01…D-12

---

## Context

Phase 12 encodes the coverage plane: the per-entry allocation ledger with the Σ invariant (Σ allocations = entry hours), derived funding-source semantics, proposals computed on read, an atomic replace-set write gated to the entry's own manager, a to-cover queue, and frozen period-close snapshots. ADR-P-012 is accepted this phase; this ADR is the backend encoding record — the schema shapes, the decision function, the write and gate semantics, and the resolution of every open question the research raised (OQ1…OQ7 / A1–A10), so later executor agents and the Phase 17 surfaces consult one record of truth instead of re-deriving semantics from code.

The encoding is **purely additive** (ADR-BE-004): three migrations (018–020), no backfill, no renumbering; every new column/constraint is nullable or legacy-safe via the three-valued-logic guard.

## Decision

### 1. Migration shapes (018–020)

**018 — activity beneficiary unit (COV-05).** `ALTER TABLE activities ADD COLUMN beneficiary_unit_id UUID REFERENCES units(id)` — nullable, no backfill, no 3VL CHECK (single nullable column; mirrors `contract_id` from 011 exactly, so legacy NULL rows pass untouched) — plus `CREATE INDEX idx_activities_beneficiary_unit_id ON activities(beneficiary_unit_id)`. The absorption funding source defaults from this column via a read-path resolver (`ResolveBeneficiaryUnit`, mirroring the `ResolveCommercialContext` chain-walk CTE shape); down drops the index then the column.

**019 — coverage_allocations (D-01, D-K).** The tagged-union allocation ledger: `id UUID PK DEFAULT gen_random_uuid()`, `org_id UUID NOT NULL REFERENCES organizations(id)`, `entry_type VARCHAR(50) NOT NULL`, `entry_id UUID NOT NULL` (**no FK** — polymorphic entry reference per D-K; the service validates the referenced entry exists, same org, `status='approved'`, `is_deleted=false`), `source_type VARCHAR(50)` (nullable — see §2), `contract_id UUID REFERENCES contracts(id)`, `unit_id UUID REFERENCES units(id)`, `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)` (identical to `time_entries.hours`, migration 000), `reason VARCHAR(50)`, `justification TEXT`, `created_at/updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`. Six constraints by exact name:

- `coverage_allocations_source_check` — the 3VL refs-to-type guard (see §2)
- `coverage_allocations_source_type_check` — `CHECK (source_type IN ('contract','absorption','transfer'))`
- `coverage_allocations_reason_check` — `CHECK (source_type <> 'absorption' OR reason IS NOT NULL)`
- `coverage_allocations_justification_check` — `CHECK (source_type <> 'transfer' OR justification IS NOT NULL)`
- `coverage_allocations_reason_vocab_check` — `CHECK (reason IS NULL OR reason IN ('WarrantyBug','UnderEstimate','Goodwill'))` (exactly these three per COV-02; the research-note Part 5 "plain internal" fourth value is superseded)
- `coverage_allocations_entry_type_check` — `CHECK (entry_type IN ('time'))` (D-K schema side; the service branch is §9)

Indexes: `idx_coverage_allocations_org (org_id)`, `idx_coverage_allocations_entry (entry_type, entry_id)`, `idx_coverage_allocations_contract (contract_id)`, `idx_coverage_allocations_unit (unit_id)`. Append-only ledger: no UPDATE/DELETE paths; the replace-set write is DELETE+INSERT inside one transaction. No `is_locked`/`closed` flag — period close is a frozen snapshot (020), never a lock (D-F/D-10).

**020 — period-close snapshots (D-10/D-11/D-12).** Two tables, entry-level rows only, no aggregate columns:

- `coverage_period_closes` — close header: `id UUID PK`, `org_id UUID NOT NULL REFERENCES organizations(id)`, `period_start DATE NOT NULL`, `period_end DATE NOT NULL`, `closed_by UUID NOT NULL REFERENCES users(id)`, `closed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `coverage_snapshot_rows` — frozen per-entry allocation state: `id UUID PK`, `close_id UUID NOT NULL REFERENCES coverage_period_closes(id) ON DELETE CASCADE`, `entry_id UUID NOT NULL`, `employee_id UUID NOT NULL` (entry owner at close), `entry_date DATE NOT NULL`, `activity_id UUID NOT NULL` (activity chain snapshot), `source_type VARCHAR(50) NOT NULL`, `contract_id UUID`, `unit_id UUID` (refs resolved/frozen at close), `hours DECIMAL(8,2) NOT NULL CHECK (hours > 0)`, `reason VARCHAR(50)`, `justification TEXT`

Indexes `idx_coverage_snapshot_rows_close (close_id)` and `idx_coverage_snapshot_rows_entry (entry_id)`. **Deliberately no `UNIQUE(org_id, period_start, period_end)`** — duplicate/overlapping close rejection is a repo-level in-tx check returning 409 (A6, §7), not a DB unique error. Append-only by construction; the only delete path is the CASCADE from the close header.

### 2. Tagged-union `source_type` — three row-level values, five derived sources (D-01, A1)

`coverage_allocations` carries a `source_type` discriminator + nullable `contract_id`/`unit_id` refs, mirroring the 015 origins / 016 sold-hours pattern (ADR-BE-016 house rule). The **three row-level values** are `contract` / `absorption` / `transfer`; the **five funding sources are derived semantics** of the referenced contract:

| Funding source | Row encoding | Derived from |
|----------------|--------------|--------------|
| Contract budget | `source_type='contract'` + `contract_id` | referenced contract `contract_type='project'` with `sold_hours` not zero-value (§4) |
| Support bucket | `source_type='contract'` + `contract_id` | referenced contract `contract_type='support'` (support buckets ARE support-type contracts, migration 016) |
| Service request | `source_type='contract'` + `contract_id` | zero-value predicate §4 (A3) |
| Internal absorption | `source_type='absorption'` + `unit_id` + mandatory `reason` | `unit_id` = beneficiary unit (COV-05), `reason ∈ {WarrantyBug, UnderEstimate, Goodwill}` |
| Cross-project transfer | `source_type='transfer'` + `contract_id` + mandatory `justification` | the other project's contract |

`source_type` is **nullable** in the schema: the enforcement is the 3VL guard, not a NOT NULL clause — legacy/all-NULL rows pass every CHECK (house rule, ADR-BE-016 Pitfall 1). The service always sets `source_type` on new rows, so there is no downstream impact:

```sql
ALTER TABLE coverage_allocations ADD CONSTRAINT coverage_allocations_source_check
    CHECK (source_type IS NULL OR
           (source_type = 'contract'   AND contract_id IS NOT NULL AND unit_id IS NULL) OR
           (source_type = 'absorption' AND unit_id IS NOT NULL AND contract_id IS NULL) OR
           (source_type = 'transfer'   AND contract_id IS NOT NULL AND unit_id IS NULL));
```

### 3. Derived bucket balance (D-02/D-03, A8)

The support-bucket balance is **derived, never stored** (D-I): `balance = sold_hours − Σ allocations drawn from the contract`, computed on read. The query scopes by `contract_id` only — **any** `source_type` (including transfers) draws the target contract's balance; it never scopes by `source_type` (Pitfall 9):

```sql
SELECT c.sold_hours - COALESCE(SUM(ca.hours), 0)
FROM contracts c
LEFT JOIN coverage_allocations ca ON ca.contract_id = c.id
WHERE c.id = $1
GROUP BY c.sold_hours;
```

- **Raw formula per A8 (OQ5 resolution):** `sold_hours − Σ drawn` with **no per-period scaling** — D-02's wording is raw Σ sold_hours; period-scaling (`sold_hours × periods elapsed`) is deferred to the Phase 17 bucket surface if wanted.
- **Overdraw allowed (D-03):** a negative balance is a visible report value, never a write-path gate — the Σ = entry hours invariant is not a bucket gate.
- **Carry-over / no-expiry (D-P)** falls out naturally: the ledger is cumulative, no counter column, no write-path lock on the bucket row.

### 4. Chain-driven proposal decision function (D-04) with the ticket-kind extension seam (D-05)

One pure decision function maps (entry → activity chain → funding config) to a proposal. Proposals are **computed on read, never stored** (D-3/D-I) — no proposal table, no staleness window. The decision order (no billability-first branch — Pitfall 3):

1. Resolve the activity chain upward for `contract_id` (`ResolveCommercialContext` CTE family) and `beneficiary_unit_id` (`ResolveBeneficiaryUnit`, new mirror CTE).
2. **Contract found:**
   - `contract_type = 'support'` → bucket draw (`source_type='contract'` + contract_id)
   - `contract_type = 'project'` AND `sold_hours IS NOT DISTINCT FROM 0` → **service-request draw** (zero-value predicate, A3/OQ3) — `source_type='contract'` + contract_id
   - `contract_type = 'project'` (positive sold_hours) → budget draw (`source_type='contract'` + contract_id)
3. **No contract:** `ResolveBeneficiaryUnit` found → internal absorption (`source_type='absorption'` + unit_id) — the beneficiary unit is inherited downward like `contract_id` (COV-05)
4. **Neither** → no proposal; the entry lands in the to-cover queue **flagged** "no eligible source — needs a unit or contract" (D-06) — never filtered out, never an implicit gap

**Zero-value predicate pinned (A3):** `contract_type = 'project' AND sold_hours IS NOT DISTINCT FROM 0` — contracts have no value column (migration 000); `km_rate` cannot discriminate (DEFAULT 0 for every contract). `sold_hours IS NULL` (project default = no commitment) and explicit `0` both draw as service request. The predicate lives in code — no schema discriminator — and is proven by table tests (sold=0 and sold=nil both draw contract).

**Extension seam (D-05):** one reserved switch point on ticket kind (activity origin → `tickets.kind`) where future kind→funding eligibility rules would plug in. **No kind→source matrix is implemented now** — chain rules only. Because proposals are computed on read, later kind rules are a code change, never a migration.

**Proposal exposure (OQ2 resolution):** both shapes share the same decision function — the to-cover queue read-model embeds the computed proposal when derivable (self-explaining rows, D-06), and the service's `Propose` method serves the per-entry proposal for the future allocation screen.

### 5. Replace-set write with in-tx Σ validation under FOR UPDATE (D-07, CR-01 lesson)

`PUT /time-entries/{id}/allocations` takes the **full allocation set (1..N rows)** and replaces everything atomically — no incremental CRUD endpoints, no partial states (D-07). The write path (one repo method):

1. `BEGIN`; `SELECT hours FROM time_entries WHERE id = $1 AND org_id = $2 AND status = 'approved' AND is_deleted = false FOR UPDATE` — the entry-row lock is the CR-01 closure: pool-level pre-checks are fast-fail UX only; the in-tx re-check is the correctness guarantee (TOCTOU mitigation).
2. **Σ validation inside the tx:** Σ request allocation hours must equal the entry's `hours`. **Cents arithmetic** (multiply by 100, round) avoids float64 artifacts on `DECIMAL(8,2)` sums. 1..N rows, each `hours > 0` (schema CHECK).
3. `DELETE FROM coverage_allocations WHERE entry_type = 'time' AND entry_id = $1`, then INSERT the new set.
4. Write the audit row in the same tx (§8) and COMMIT.

**Ref validation at the service:** `contract_id` refs must resolve via `contractRepo.Get` **and** be org-visible — `res.CreatedByOrgID == orgID || (res.IsShared && res.IsAdopted)`; `unit_id` refs via `GetByID` + `OrgID` compare (`Get` is not org-scoped) — cross-org reference injection is rejected (same-org rule, Phase 11 D-02 pattern).

### 6. Manager gate (D-08) — the entry's own manager, via BE-014

The writer is resolved through the **shared `routing.ResolveManagerStage(ctx, OrgID, ActivityID, UnitID, UserID)`** — the same BE-014 resolution that approved the entry; never re-implemented (single routing rule for approvals and coverage, prevents drift):

- **Allowed:** actor ∈ `ApproverIDs` (WG manager/delegate or unit manager) **OR** (`res.RoleGated` && org role == `'manager'`). The RoleGated branch (org root without unit manager) **requires the org manager role claim** — never an optional pass.
- **Rejected:** `ErrActivityNotLoggable` from the resolver (commercial activity without anchored WG — no legitimate manager) → forbidden; the **structural self-barrier** (`entry owner == writer` → forbidden — the employee can never allocate their own coverage, mirroring the entry `Approve` gate).
- **Finance is read-only (D-2/D-L):** reads (to-cover queue, proposals, snapshots, balance) open to manager + finance; writes manager-only. `customer`/`hr`/`employee` rejected for writes.
- **No correction handling (D-09):** coverage only ever sees approved `time` entries with `hours > 0` (schema CHECK forbids negatives); compensating entries do not exist (`created_from_entry_id` is schema-only) — the D-13 "net-of-compensations" guard swap has nothing to swap to; the dismissal guard stays on raw Σ. If a correction mechanism ever lands, coverage interaction is designed then (Deferred Ideas, ADR-P-012 acceptance).

### 7. Period-close snapshots (D-10/D-11/D-12, OQ4/OQ6/OQ7)

`POST /coverage/close` — org from claims, body `period_start`/`period_end` DATE — freezes the period's allocation state:

- Entries whose `entry_date::date` falls in `[period_start, period_end]` (**inclusive**) are frozen into **entry-level rows** (§1) carrying the allocation state as of close: entry_id, employee, entry_date, activity chain snapshot, source refs, hours, reason/justification. **No aggregates** (D-11): bucket levels, billing totals, per-unit aggregates are computed from these rows on read when the Phase 17 surfaces land.
- **Duplicate/overlapping close → 409** (A6/OQ6): the repo's in-tx overlap check returns `coverage.ErrPeriodAlreadyClosed`; deliberately not a DB unique constraint.
- **`financial_cutoff_periods` stays facts-only (D-12/OQ7):** the coverage close is a fully separate mechanism; the existing table (migration 000) is not the trigger and receives no rows from the coverage close. Revisit when entry cutoff semantics land.
- **The close response returns the snapshot rows in one call (OQ4):** `ClosePeriod` returns the full `PeriodClose` incl. rows — close + report in one call.
- The close tx also writes one audit row (§8). Snapshot reads (`GET /coverage/snapshots/{close_id}`) hit the frozen copy, never live rows — "a reported period never changes retroactively" holds by construction (Pitfall 7); the real allocation lock, if ever needed, attaches at the future billing layer (Deferred Ideas), which reads the frozen snapshot.

### 8. Audit vocabulary (A7) — in-tx synchronous writes (BE-016)

Every coverage change is audit-logged via the general `audit_logs` table (017) with the **pinned vocabulary**:

- `entity_type = 'coverage_allocation'` (per entry)
- `action = 'allocations-set'` for the replace-set write — payload = **full allocation set JSON**
- `action = 'coverage-closed'` for period close — payload = close summary (period + row count)
- `entity_id` = the entry id (or the close id for the close event)

Rows are written **synchronously inside the same transaction** as the mutation (private `insertCoverageAudit(ctx, tx, log)` helper mirroring the Phase 11 `insertTicketAudit` precedent) — never fire-and-forget (BE-016 scope note): the event stream is the guarantee, and a coverage operation fails as a whole if its audit row cannot be written (accepted, as for tickets).

### 9. D-K polymorphic validation cost — stated honestly

The polymorphism (`entry_type` + `entry_id`, `time` only in v0.2) costs **one extra service branch** rejecting `entry_type != 'time'` plus the schema CHECK `coverage_allocations_entry_type_check ('time')`:

- **Service branch:** a single comparison in the coverage write path — `if entryType != "time" → coverage.ErrInvalidEntryType` (one branch, no dispatch table, no per-type logic). This is the "one extra validation branch" ADR-P-012 D-5 requires costing honestly.
- **Schema CHECK:** one `IN ('time')` list entry; already landed in migration 019.

**COV-06 (expense) will need:** an `ALTER` of the `coverage_allocations_entry_type_check` (add `'expense'`) plus a service rule change (expense-splitting rules) — **an additive ALTER and a code change, not a redesign**; the tagged-union ledger shape, ports, and endpoints carry over unchanged. If the polymorphism proves dead weight, the cost to remove it is the mirror image (drop the branch + the CHECK, keep the columns nullable).

### 10. Open-question resolutions (OQ1…OQ7, A1–A10)

| OQ | Question | Resolution (pinned) |
|----|----------|---------------------|
| OQ1 | `source_type` vocabulary: 3 vs 5 values | **3 row-level values, five sources derived** from the referenced contract (§2, A1) — matches D-04's decision function exactly; CHECK enforces the 3 values |
| OQ2 | Proposal read-path exposure | **Both** — per-entry proposal endpoint and queue-with-proposals, sharing one decision function (§4) |
| OQ3 | Zero-value contract predicate | **`contract_type='project' AND sold_hours IS NOT DISTINCT FROM 0`** (§4, A3) |
| OQ4 | Does close return the snapshot? | **Yes** — close response returns the full snapshot rows (§7) |
| OQ5 | Bucket balance raw vs period-scaled | **Raw `sold_hours − Σ drawn`** (§3, A8); period-scaling deferred to Phase 17 |
| OQ6 | Duplicate close | **Rejected with 409** (`coverage.ErrPeriodAlreadyClosed`), repo in-tx check, not a DB unique key (§7, A6) |
| OQ7 | Relation to `financial_cutoff_periods` | **Fully separate** — cutoff stays facts-only (§7) |

Assumptions A1–A4/A6/A8/A10 are pinned in the sections above; A5 (two-table snapshot shape) and A9 (owner self-barrier) are pinned in §1/§6; A7 (audit vocabulary) in §8.

## Costs

- **D-K polymorphism:** one service branch + one CHECK list entry (§9) — the honest cost of ADR-P-012 D-5; removal or extension is an additive ALTER.
- **In-tx audit writes:** every coverage mutation fails as a whole if the audit row cannot be written — accepted: the event stream is the guarantee (BE-016 precedent).
- **Replace-set writes:** a full-set PUT is heavier per save than incremental CRUD, but makes the Σ invariant hold by construction (D-07); the FOR UPDATE entry-row lock serializes concurrent saves on the same entry.
- **Snapshot maintenance:** close duplicates allocation state into `coverage_snapshot_rows`; the write is per-entry at close time and the tables are append-only — no ongoing maintenance cost.

## Verification / Implementation

- Migrations 018–020 with up/down pairs per ADR-BE-004; cycle tests `TestMigration018/019/020` (up → down → up) assert the 3VL guard, mandatory-field CHECKs, vocabulary CHECKs, and CASCADE behavior functionally (failing inserts with pgErr `23514`, legacy all-NULL rows passing).
- Replace-set correctness proven by unit + integration tests: Σ mismatch rejected, concurrent replace-set serialized (CR-01-style, `FOR UPDATE`), audit rows written in-tx with payload assertions.
- Manager gate proven by a permission-matrix test (WG manager/delegate, unit manager, role-gated org manager, employee, finance, customer).
- Close semantics proven by integration tests: snapshot freezes state (later edits do not change it), duplicate close → 409, response returns the snapshot rows.
- Commands: `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration018|TestMigration019|TestMigration020' -count=1`, `go test ./internal/core/services/coverage/ ./internal/adapters/secondary/postgres/ ./internal/adapters/primary/http/ -count=1`, `make test` at the wave merge.

## Consequences

- The vault is the record of truth for the coverage plane: Phase 17 surfaces (allocation screen, to-cover queue, bucket balance, per-unit report) and any future executor consult this ADR + ADR-P-012, never re-derive semantics from code.
- Every planner-resolved open question is pinned with an explicit resolution — no ambiguity left for Phase 17 to re-litigate (T-12-04 closed).
- The three-tier pattern of the phase holds: the schema owns shapes (CHECK vocabularies, refs-to-type guard), the service owns invariants (Σ, eligibility, gate), and the audit trail owns history (in-tx, append-only).
- `financial_cutoff_periods` stays facts-only; the deferred billing-layer lock (Deferred Ideas of ADR-P-012) reads the frozen snapshot when invoicing lands.
- ⚠️ Expense coverage (COV-06) is schema-ready only: the D-K branch + CHECK change is an additive ALTER + service rule change, not a redesign (§9).

## Related

- [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] — the accepted idea-layer record this ADR operationalizes (D-1..D-6)
- [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] — 3VL CHECK house rule, in-tx synchronous audit precedent (017)
- [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] — `routing.ResolveManagerStage`, the D-08 writer gate
- [[ADR-BE-004 — Database Migrations]] (append-only rule) · [[ADR-BE-012 — Audit Log Writes]] (scope note)
- [[ADR-P-014 — Funding Sources & Beneficiary Unit]] · [[ADR-P-007 — Activity Ontology]] (D-3 contract inheritance, D-7 billability) · [[ADR-P-013 — Origins]]
- `migrations/018…020` (up/down pairs) · `internal/adapters/secondary/postgres/coverage_ontology_migrations_test.go` (cycle tests)
- [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 5–8, 12, 13 A–F, Q10 amendment — record of truth)
