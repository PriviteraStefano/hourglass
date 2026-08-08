---
tags: ["adr", "backend", "schema", "direction", "org-settings", "audit"]
---

# ADR-BE-018 — Direction & Org Settings Encoding

**Status:** Proposed
**Date:** 2026-08-08
**Code:** `migrations/021…022`, `internal/core/domain/direction/`, `internal/core/domain/orgsettings/`, `internal/core/services/`, `internal/core/ports/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`
**Operationalizes:** [[ADR-P-015 — Direction, The Plan Plane]] (D-1..D-11, three-plane ontology) · **Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 14–15, D-R…D-AA) · **Extends:** [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] (3VL CHECK house rule, in-tx synchronous audit writes) · **Gates on:** [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] (manager reach for direction creation + WG-row gate) · **Resolves:** Phase 13 semantic resolutions (research OQ1…OQ8), locked decisions D-13-01…D-13-34, assumptions A1…A10

---

## Context

Phase 13 encodes the plan plane: the `direction` entity with per-day rows and derived modes, a ticket-style lifecycle with supersede-on-create chaining, a WG claim model with Σ-consumption enforced under a transaction lock, org-scoped planning policy stored as key/value settings, a P-008 absence/validity warning overlay that never blocks, a direction-coverage read-model, and the origin-ref read-path fallback. ADR-P-015 is drafted this phase; this ADR is the backend encoding record — the schema shapes, the status/derived/audit/settings vocabularies, the write and gate semantics, and the resolution of every open question the research raised (OQ1…OQ8 / A1–A10), so later executor agents and the Phase 19 surfaces consult one record of truth instead of re-deriving semantics from code.

The encoding is **purely additive** (ADR-BE-004): two migrations (021–022), no backfill, no renumbering; every new column/constraint is nullable or legacy-safe via the three-valued-logic guard.

## Decisions

### 1. Status vocabulary — draft / active / superseded / cancelled

`direction.status` is DB CHECK-enforced (`direction_status_check`, migration 021) with the pinned matrix:

| From | To | Reached via |
|------|-----|-------------|
| `draft` | `active` | explicit `POST /direction/{id}/activate` (OQ1 resolution — one audit row per transition, symmetric with cancel; create-with-planned_date does **not** auto-activate) |
| `draft` / `active` | `cancelled` | `POST /direction/{id}/cancel` with mandatory `reason` (D-13-10; schema CHECK `status <> 'cancelled' OR reason IS NOT NULL`) |
| `draft` / `active` | `superseded` | **ONLY via create-with-`supersedes_id`** (D-13-08) — there is **no transition endpoint** for supersede |

`superseded` and `cancelled` are terminal. The domain matrix (`CanTransition`) mirrors the ticket domain; the repo re-validates inside the mutator tx under `SELECT ... FOR UPDATE` (CR-01 closure, house style for state machines). Every transition writes an `audit_logs` row in the same tx (§3).

### 2. Derived-state vocabulary — done / lapsed / claimed spectrum

Derived states are **computed on read, never stored, no nightly jobs** (D-V, D-13-09):

- **`done`** — the linked activity is terminal: the Phase 11 terminal-activity recursive CTE re-anchored at `activities.id` — the **semantic inversion** of the ticket dismissal-guard EXISTS (no non-terminal time entries on the linked-activity subtree, `is_deleted = false`). Re-anchored so the direction row's `activity_id` (not the ticket's) is the subtree root.
- **`lapsed`** — past `planned_date`/`due_date` **AND** no non-deleted time entries (any status — draft included, OQ2/A3) on the activity subtree. "A plan lapsed when nothing was ever logged; a draft entry indicates work started."
- **Claimed spectrum** (WG rows only, D-13-15): `not_claimed` → `partially_claimed` → `fully_claimed`. `fully_claimed` only when the WG row's budget (`est_hours`) is set **and** Σ claims == budget. Never stored.

### 3. Audit vocabulary (A1) — in-tx synchronous writes (BE-016)

- `entity_type = 'direction'`, `entity_id` = the direction row id, actions **`created` / `activated` / `cancelled` / `superseded` / `claimed` / `unclaimed`** — pinned verbatim so Phase 19 history reads filter deterministically (T-13-06). Payload carries the row state (planned_date, est_hours, supersedes_id, reason).
- `entity_type = 'org_settings'`, `entity_id` = **the org id** (`audit_logs.entity_id` is UUID NOT NULL — migration 017:18), action **`settings-updated`**, payload = `{key, before, after}` (D-13-22).
- Rows are written **synchronously inside the same transaction** as the mutation (private insert helpers mirroring `insertTicketAudit` / `insertCoverageAudit`) — never fire-and-forget.

### 4. Settings key vocabulary (A7)

`org_settings(org_id, key VARCHAR(50), value JSONB, updated_at, PK(org_id, key))` — generic key/value; vocabulary validated in code per known key (CHECK on JSONB isn't feasible, D-13-18):

| Key | Type | Default | Purpose |
|-----|------|---------|---------|
| `planning_daily_hours` | number | **8.0** | daily capacity denominator (D-13-24) |
| `planning_deadline` | date (`2006-01-02` string) | — | D-X axis ① |
| `planning_horizon` | `day` \| `week` \| `month` | — | D-X axis ②, **stored not enforced** (D-13-21) |
| `planning_mode` | `manager_planned` \| `self_planned` | — | org default (D-13-19) |

Per-employee override: nullable `planning_mode` column on `organization_memberships` (migration 022) — NULL falls back to the org default. Code-level defaults apply when a key is absent (no seed rows).

### 5. Claim over-subscription closure (D-13-13/14) + supersede-of-claim-row pin

**Claim write** (`Claim` repo method): `BEGIN` → `SELECT est_hours FROM direction WHERE id = $1 AND org_id = $2 FOR UPDATE` on the **WG row** — the lock is the CR-01 closure; pool-level checks are fast-fail UX only. **Σ re-checked in-tx in cents** (`math.Round(h*100)` — the coverage precedent) over the predicate `origin_direction_id = $1 AND status IN ('draft','active')` — **superseded/cancelled claim rows never consume budget**. If Σ + claimed > budget → `direction.ErrClaimOverBudget` → **409**. **Uncapped when the WG row's `est_hours` is NULL** (budget optional per D-AA, D-13-14) — claims still require `est_hours > 0`, but no Σ gate and the `consumed`/`fully_claimed` state never derives.

**Supersede-of-claim-row semantics** (checker fix, cross-plan contract): when create-with-`supersedes_id` targets a **claim row** (`target.origin_direction_id IS NOT NULL`), the **new row inherits `origin_direction_id`** — the hours move along the chain: the superseded row drops out of the Σ, the new draft row counts in its place. The superseding row **MUST remain user-targeted** (`directed_to` set, `wg_id` NULL — a WG-shaped superseding row → `ErrInvalidTarget`). Cancelling the superseding row therefore **releases the hours** back to the WG budget (D-13-16 holds through the chain).

### 6. est_hours scale (A2)

`est_hours DECIMAL(8,2)` mirroring `time_entries.hours` exactly (migration 000) — hour granularity 2 decimal places, same as entries. Service rejects `est_hours <= 0` (and absurd values) at write (D-13-03 hard per-row validation); Σ over day capacity is a **soft warning** returned in read-model/create responses, never a rejection.

### 7. Settings CRUD shape (D-13-23)

`GET/PUT /organizations/settings` — **literal routes** (ServeMux literal-beats-wildcard precedence, coexisting with the typed `GET/PUT /organizations/{id}/settings` on `organization_settings`, which stays untouched). JWT-resolved org via `middleware.GetOrganizationID` — **no org path param**. Manager+ gate on PUT (`role != manager` → `orgsettings.ErrForbidden` → 403). Additive keys; unknown key → `ErrUnknownKey` → **400**. `{key: value}` payloads; every PUT writes the settings-updated audit row in the same tx (§3).

### 8. Assumption pins (A1/A2/A4/A5/A6/A8/A9/A10) — recorded with alternatives

| # | Pin | Alternative documented |
|---|-----|------------------------|
| A1 | Audit vocabulary (§3) | free-form `audit_logs` columns would leave Phase 19 history reads guessing |
| A2 | `est_hours DECIMAL(8,2)` (§6) | a different scale is a pure DDL tweak, but plan-time consistency with entries matters |
| A4 | Origin-fallback trigger predicate = `OriginType == nil` (no origin set at all); applies to activity `GetByID` + `List` via service enrichment through a small direction-ref port | refs-empty-but-type-set rows stay stored-authoritative (P-013 immutability); a wider predicate would derive over explicit origins |
| A5 | WG-scope predicate = directed activity same-org **AND** (activity == `wg.SubprojectID` OR the anchor is in `GetAncestry(activityID)`) | a routing-set-based scope ("any activity the WG manager manages") would differ; pinned to the subtree shape |
| A6 | Unit scope in the coverage read-model = the unit **+ its descendants' members** (`GetDescendants` + `ListMembers`) | direct-members-only would drop the descendants join — a read-model difference Phase 19 would surface |
| A8 | Claim rows created in `draft` status, queued mode (`planned_date` NULL), copying `priority`/`due_date` from the WG row; claimant then schedules via the normal supersede chain | active-on-create or scheduled claim rows would change the create flow and matrix edges |
| A9 | Mode gate strict reading (D-13-20): manager-planned → only managers create rows for that employee; self-planned → only the employee creates their own rows; queued self-rows follow the same mode | a hybrid (self-direction always allowed regardless of mode) is an additive loosening, not a schema change |
| A10 | WG-row creation gate = `routing.ResolveManagerStage` on the anchored activity (WG manager/delegates or role-gated org manager) — same resolution as entry approval | any-org-manager creation would drop the routing call — permission difference |

### 9. Assumption-delta decisions (recorded in ADR-P-015, operationalized here)

1. **Identity no-change:** the identity noun is the direction row id — "multiple rows may share a day" is a grouping convention; **no UNIQUE constraint** on (employee, activity, day). The per-day multiplicity of D-W/D-AA is the model (migration 021 cycle-tested: two rows sharing employee/activity/day both insert).
2. **Origin fallback = add-alongside read-path derivation:** stored refs stay authoritative (P-013 immutability); the accepted debt is transient drift between stored refs and derived refs, mitigated by never writing back and by `origin_type` remaining the discriminator.
3. **Planning policy = first-class data:** `org_settings` key/value + membership `planning_mode` override — config became a modeling decision (D-13-18/19); new policy keys are data rows, never migrations.

## Schema encoding

**021 — `direction` rows.** `id UUID PK DEFAULT gen_random_uuid()`, `org_id UUID NOT NULL REFERENCES organizations(id)`, `directed_by UUID NOT NULL REFERENCES users(id)`, `directed_to UUID REFERENCES users(id)`, `wg_id UUID REFERENCES working_groups(id)`, `activity_id UUID NOT NULL REFERENCES activities(id)`, `planned_date DATE` (nullable — mode discriminator), `est_hours DECIMAL(8,2)` (nullable), `priority INT` (nullable, lower = higher), `due_date DATE` (nullable), `status VARCHAR(20) NOT NULL DEFAULT 'draft'`, `supersedes_id UUID REFERENCES direction(id)` (nullable self-FK), `origin_direction_id UUID REFERENCES direction(id)` (nullable self-FK — claim rows), `reason TEXT` (nullable, cancellation/claim-reason), `created_at/updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`. Named constraints: `direction_status_check` (status vocabulary, §1), `direction_target_check` (user-XOR-WG, D-13-05), `direction_wg_queued_check` (`wg_id IS NULL OR planned_date IS NULL` — WG rows queued-only, D-13-17), `direction_scheduled_hours_check` (`planned_date IS NULL OR est_hours IS NOT NULL` — scheduled requires est_hours, D-13-02), `direction_est_hours_check` (`est_hours IS NULL OR est_hours > 0`), `direction_cancel_reason_check` (`status <> 'cancelled' OR reason IS NOT NULL`, D-13-10). Indexes: `(org_id, directed_to, planned_date)` · `(org_id, wg_id)` · `(activity_id, created_at)` (origin fallback, D-13-33) · `(supersedes_id)`. No `is_deleted` soft-delete — `cancelled` is the terminal state; superseded/claim rows are never deleted (audit trail preserved).

**022 — `org_settings` + membership override.** `org_settings(org_id UUID NOT NULL REFERENCES organizations(id), key VARCHAR(50) NOT NULL, value JSONB NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (org_id, key))` — upsert value-replacement semantics (ON CONFLICT). `ALTER TABLE organization_memberships ADD COLUMN planning_mode VARCHAR(20)` — nullable per-employee override (D-13-19), NULL = fall back to org default, no backfill.

## Security

- **Permission gates:** mode gate (A9) + WG-member-only claims (D-13-12) + manager+ gate on settings PUT (D-13-23) + BE-014 manager reach for direction creation (A10). Pool-level checks are fast-fail UX; the mutator tx re-validates under `FOR UPDATE` (CR-01 closure).
- **Over-subscription race:** closed by the WG-row lock + in-tx Σ re-check in cents (§5) — first-wins, `ErrClaimOverBudget` 409.
- **Cross-org refs:** `directed_to`/`wg_id`/`activity_id` refs validated same-org at the service (house style) before the repo call.
- **Unknown settings keys** → 400 (`ErrUnknownKey`); invalid values → 400 (`ErrInvalidValue`) — no 500 path from settings input.
- **Audit trail:** every lifecycle/supersede/settings change lands in `audit_logs` in-tx (BE-016 scope) — repudiation of a direction/settings change is impossible (T-13-06).

## Consequences

- The vault is the record of truth for the plan plane: Phase 19 surfaces (Today plan/queue, direction scheduler, coverage read-model, settings UI) and any future executor consult this ADR + ADR-P-015, never re-derive semantics from code.
- The exported Go constants in plan 13-03 (status/derived/claim-spectrum/audit/settings vocabularies) compile against these pins — the ADR is the spec, the constants are the compile-time enforcement (T-13-05): repo/service/handler can never drift.
- The three-tier pattern holds: the schema owns shapes (CHECK vocabularies, XOR/queued/scheduled guards), the service owns invariants (Σ, gates, derived states), and the audit trail owns history (in-tx, append-only).
- Supersede-of-claim-row chaining means hours travel along the chain without a stored counter — consumption stays Σ-derived, unclaim/cancel releases hours automatically (D-13-16).
- ⚠️ `planning_daily_hours` default (8.0) is a code-level default, not a seed row; the coverage read-model (13-06) must apply it when the key is absent.

## Related

- [[ADR-P-015 — Direction, The Plan Plane]] — the idea-layer record this ADR operationalizes (D-1..D-11)
- [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] — 3VL CHECK house rule, in-tx synchronous audit precedent (017)
- [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] — `routing.ResolveManagerStage`, the direction-creation/WG-row gate
- [[ADR-BE-004 — Database Migrations]] (append-only rule) · [[ADR-BE-012 — Audit Log Writes]] (scope note)
- [[ADR-P-008 — Availability & Employment Validity]] (warning overlay inputs, declared+confirmed until Phase 14) · [[ADR-P-013 — Origins]] (FND-04 fallback) · [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] (three-plane doctrine)
- `migrations/021…022` (up/down pairs) · `internal/adapters/secondary/postgres/direction_ontology_migrations_test.go` (cycle tests)
- [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 14–15 — record of truth)
