# Phase 9: Activity Ontology — Specification

**Created:** 2026-07-31
**Ambiguity score:** 0.08 (gate: ≤ 0.20)
**Requirements:** 8 locked

## Goal

Replace the `projects` + `subprojects` two-table work decomposition with the single recursive `activities` entity, rewrite every FK and approval-routing query onto the new ontology, land the additive staffing schema (ADR-P-008), and leave the backend API fully activity-shaped — backend + database only, with the frontend quarantined compiling-but-stale for Phase 10.

## Background

The v0.1 MVP models work decomposition as two rigid tables at exactly two depths (`projects`, `subprojects`), forcing a four-FK capture chain on `time_entries` (`project_id` + `subproject_id` + `wg_id` + `unit_id` all NOT NULL) while `expenses` link asymmetrically (nullable `project_id`/`customer_id`, no `wg_id`). This blocks expense routing by working group (ADR-P-001 Q1) and makes internal/personal work require commercial scaffolding. ADR-P-007 (accepted 2026-07-29) settles the ontology: work is one recursive `Activity`; ADR-BE-014 encodes routing/visibility consequences; ADR-P-008 rides the same migration batch with an additive staffing schema. Phase 8 (P0 hardening) is complete, so the migration test net (testcontainers + green `go test ./...`) is in place.

Current code state verified 2026-07-31: `internal/core/domain/{project,subproject}/`, `internal/core/ports/{project,subproject}_repository.go`, and postgres adapters all exist with no `activity` counterparts. `time_entries` carry `project_id`/`subproject_id`/`wg_id`; `working_groups.enforce_unit_tuple` exists in schema and domain; `migrations/` max number is `010_refresh_token_reuse_detection` (Phase 8) — the next free numbers are **011** and **012**.

## Requirements

1. **Activities schema**: Recursive `activities` table replaces `projects` + `subprojects`, with org-extensible `activity_kinds` catalog.
   - Current: `projects` and `subprojects` tables exist as separate two-depth entities; no `activities` table; no `activity_kinds` catalog
   - Target: `activities` (nullable self-referencing `parent_id` ON DELETE RESTRICT, nullable `contract_id` ON DELETE RESTRICT, `billable` nullable boolean, `kind` FK to `activity_kinds`), `activity_kinds` seeded with `engagement`/`phase`/`task`/`internal` for the MVP org, `activity_adoptions` mirroring `project_adoptions`
   - Acceptance: `011_activity_ontology.up.sql` creates the three tables matching the ADR-P-007 sketch; `SELECT * FROM activity_kinds` for the seed org returns exactly `engagement`, `phase`, `task`, `internal`; `projects`/`subprojects` no longer exist

2. **FK rewrite on entry tables**: Entries link through exactly one required `activity_id`.
   - Current: `time_entries` have NOT NULL `project_id`/`subproject_id`/`wg_id`/`unit_id`; `expenses` have nullable `project_id`/`customer_id` and no WG link
   - Target: `time_entries` keep `unit_id NOT NULL`, drop `project_id`/`subproject_id`/`wg_id`, add `activity_id NOT NULL`; `expenses` keep `unit_id NOT NULL`, drop `project_id`/`customer_id`, add `activity_id NOT NULL`
   - Acceptance: Post-migration `SELECT COUNT(*) FROM time_entries WHERE activity_id IS NULL` = 0 (same for `expenses`); `information_schema.columns` confirms the dropped columns are gone and `activity_id` is NOT NULL on both tables

3. **Working-group and governance re-anchor**: `working_groups.activity_id` replaces `subproject_id`; `enforce_unit_tuple` dropped; `project_managers` → `activity_managers`; `financial_cutoff_periods.activity_id` replaces `project_id`.
   - Current: `working_groups.subproject_id` with `enforce_unit_tuple BOOLEAN DEFAULT TRUE`; `project_managers` table; `financial_cutoff_periods.project_id`
   - Target: WG anchors at any activity depth via `activity_id`; `enforce_unit_tuple` column gone; governance role table renamed to `activity_managers` with `activity_id`; cutoff periods keyed by `activity_id`; `IsPeriodLocked` resolves org + activity + date range
   - Acceptance: `enforce_unit_tuple` absent from schema and Go domain type; `project_managers` absent, `activity_managers` present with migrated rows; `working_groups.activity_id` references `activities(id)`

4. **Data migration**: MVP seed data migrates with zero orphans.
   - Current: Seed has 2 projects + 6 subprojects (all phase-shaped, e.g. "Platform Engineering — Phase 1", "Cloud Migration — Assessment & Plan") with existing time entries/expenses referencing them
   - Target: Projects → activities with `kind = 'engagement'`, `parent_id NULL`, `contract_id` preserved; subprojects → activities with `kind = 'phase'`, `parent_id` = parent's new id, `contract_id` inherited from parent; all entry FKs re-pointed in the same transaction
   - Acceptance: Post-migration every pre-existing time entry and expense has a valid `activity_id` (zero orphans); subproject-derived activities carry `kind = 'phase'`; contract chain is intact (descendant activities reach a contract via upward walk)

5. **Down migration**: `011_activity_ontology.down.sql` reverses the schema and data, best-effort per ADR-BE-004.
   - Current: No down migration exists for the new ontology
   - Target: Recreates `projects`/`subprojects` with original shapes; restores entry columns (`project_id`/`subproject_id`/`wg_id` on time_entries, `project_id`/`customer_id` on expenses); restores `working_groups.subproject_id` + `enforce_unit_tuple`; restores `project_managers`/`project_adoptions`/`financial_cutoff_periods.project_id`; documented lossy for `billable`/`budget_amount`/`kind` metadata and activities beyond the two-level model
   - Acceptance: Up → down → up cycle completes without error in a testcontainers test; down migration has a comment header documenting lossiness

6. **Approval routing rewrite**: Entries resolve routing from `activity_id` only (ADR-BE-014 R-1/R-2/R-3).
   - Current: Time entries route via WG manager + finance; expenses route via project manager (no WG link); no `ErrActivityNotLoggable` sentinel
   - Target: Chain = activity → anchored WG → WG manager/delegate → org finance; no anchored WG → submitter's unit manager (nearest manager walking unit tree upward); D-11 skip fires when WG manager *or delegate* equals the owner (`submitted` → `pending_finance` directly); commercial activity (contract set) without anchored WG rejected at submission with `ErrActivityNotLoggable`
   - Acceptance: Routing unit tests cover (a) WG path, (b) unit-manager fallback incl. upward walk, (c) D-11 skip incl. delegates and no self-approval, (d) `ErrActivityNotLoggable` on commercial-without-WG; `ErrActivityNotLoggable` maps to HTTP 409

7. **Visibility preserved**: Unit-subtree gating for entry lists (ADR-BE-014 R-4).
   - Current: Unit manager visibility via subtree recursive CTE already in units repository; entries carry `unit_id`
   - Target: Entry list queries gate on `unit_id` subtree for unit managers; org-role manager/finance see whole org; routing and visibility remain separate axes
   - Acceptance: List tests confirm a unit manager sees only their subtree entries; org-role manager/finance see all; no routing query changed for visibility purposes

8. **Repository/API collapse + staffing schema**: One `activity_repository` and one `/api/activities` endpoint set; staffing schema lands additive.
   - Current: `project_repository` + `subproject_repository` (two port files, two postgres adapters, two endpoint sets on project handler); no `availability_windows`; `organization_memberships.role` CHECK lacks `'hr'`; no validity date columns
   - Target: Single `activity_repository` (port + postgres adapter) replaces both; one `/api/activities` endpoint set; `ListPending` for both entry types queries via activity → WG join; `012_staffing_schema` creates `availability_windows` (kind CHECK `holiday|permit|medical|unavailable`, status CHECK `declared|confirmed`, `hours`, `certificate_ref`), adds `valid_from`/`valid_until`/`work_permit_expires_at` to `organization_memberships`, extends role CHECK with `'hr'` — no surfaces, zero FK coupling to the activity rewrite
   - Acceptance: `go build ./...` compiles with no `project_repository`/`subproject_repository` references; `/api/projects` and `/api/subprojects` endpoints removed, `/api/activities` live; `012_staffing_schema.up.sql` applies and `organization_memberships` role CHECK accepts `'hr'`; no `availability_windows` UI/endpoints in this phase

## Boundaries

**In scope:**
- Migration `011_activity_ontology.up.sql` / `.down.sql` — schema rewrite + seed data migration
- Migration `012_staffing_schema.up.sql` / `.down.sql` — availability_windows + membership validity columns + `hr` role (schema only)
- `activities`/`activity_kinds`/`activity_adoptions`/`activity_managers` tables and Go domain types
- Repository collapse (`activity_repository` replaces project+subproject) across domain, ports, postgres adapters
- Approval routing rewrite in `internal/core/services/{time_entry,expense}/` per R-1/R-2/R-3 incl. `ErrActivityNotLoggable`
- HTTP DTO updates: `/api/activities` endpoint set, entry request/response shapes activity-shaped, route rewiring in `cmd/server/main.go`
- Unit tests + testcontainers integration tests migrated to the new schema
- Index `activities(parent_id)` and `working_groups(activity_id)`
- Cycle prevention on `activities.parent_id` (path check on insert/update)

**Out of scope:**
- Frontend rename (`/projects` → `/activities`) and all new surfaces (Today, Approvals, WG, Availability) — Phase 10 (ADR-P-011); frontend deliberately left compiling-but-stale
- Availability window surfacing at assignment, payroll export view, expiry queues — Phase 10 or v0.2 (ADR-P-008 sequencing)
- Staffing UI/endpoints — schema only this phase
- P1 audit items (skeletons, OrgID validation, JSON decode, cookie names) — candidate Phase 12
- P2 backlog items — post-v0.1
- Renaming the `activity_managers` governance-role word — separate cosmetic decision (ADR-P-007 Consequences)

## Constraints

- Migrations land as **new numbered files** (`011`, `012` — next free after Phase 8's `010`) per ADR-BE-004; applied history is immutable, no edits to `000_full_schema.up.sql`
- Both migration files apply in the same deploy batch (atomic); up → down → up cycle must be clean in a testcontainers test
- Index `activities(parent_id)` and `working_groups(activity_id)` — the recursive-CTE hot paths
- `ErrActivityNotLoggable` maps to HTTP 409 per ADR-BE-001
- Commercial-chain and unit-subtree resolution reuse the recursive-CTE pattern already in `unit_repository.go`; depth expected < 6, no denormalization of contract onto entries
- No new endpoints beyond the collapsed `/api/activities` set; staffing schema is zero-coupled (no FK to `activities`)
- Frontend build is excluded from this phase's pass criteria (known-broken until Phase 10)
- No additional constraints beyond standard project conventions

## Acceptance Criteria

- [ ] `011_activity_ontology` up/down exist; up applies on fresh PostgreSQL, up→down→up cycle is clean (testcontainers)
- [ ] `projects`, `subprojects`, `project_adoptions`, `project_managers` tables absent after up migration
- [ ] `activities`, `activity_kinds` (seeded 4 kinds for MVP org), `activity_adoptions`, `activity_managers` present after up migration
- [ ] `enforce_unit_tuple` absent from both schema and Go domain type
- [ ] `time_entries` and `expenses` have `activity_id NOT NULL` and zero orphaned rows post-migration
- [ ] Subproject-derived activities carry `kind = 'phase'`; project-derived activities carry `kind = 'engagement'`
- [ ] `go build ./...` succeeds; no `project_repository`/`subproject_repository` symbols remain
- [ ] `/api/activities` live; `/api/projects`/`/api/subprojects` removed from route wiring
- [ ] Routing tests pass for: WG path, unit-manager fallback (incl. upward walk), D-11 skip (incl. delegates, no self-approval), `ErrActivityNotLoggable` → 409
- [ ] Unit-manager subtree visibility tests pass for entry lists
- [ ] `012_staffing_schema` applies; `availability_windows` exists with kind/status CHECKs; role CHECK accepts `'hr'`; validity columns exist on `organization_memberships`
- [ ] `go test ./...` green (migrations + service + handler + repo suites)
- [ ] No frontend source files under `web/src/` modified by this phase

## Ambiguity Report

| Dimension          | Score | Min  | Status | Notes                              |
|--------------------|-------|------|--------|------------------------------------|
| Goal Clarity       | 0.95  | 0.75 | ✓      | ADR-P-007 D-1..D-8 locked          |
| Boundary Clarity   | 0.94  | 0.70 | ✓      | Backend-only; frontend excluded from pass criteria |
| Constraint Clarity | 0.92  | 0.65 | ✓      | Migration numbers locked to 011/012 (010 taken by Phase 8) |
| Acceptance Criteria| 0.82  | 0.70 | ✓      | Backend-only gate; kind mapping locked to 'phase' |
| **Ambiguity**      | 0.08  | ≤0.20| ✓      |                                  |

Status: ✓ = met minimum, ⚠ = below minimum (planner treats as assumption)

## Interview Log

| Round | Perspective     | Question summary              | Decision locked                         |
|-------|-----------------|------------------------------|-----------------------------------------|
| 1     | Researcher      | Migration numbering collides (010 taken by Phase 8); what numbers? | Locked `011_activity_ontology` + `012_staffing_schema` |
| 1     | Researcher      | How does verification treat the knowingly-broken frontend build? | Frontend excluded from pass criteria; backend-only gate |
| 1     | Researcher      | Seed subprojects are phase-shaped — which kind? | Subprojects migrate with `kind = 'phase'` |
| —     | Seed Closer      | All dimensions above minimum, ambiguity 0.08 | User approved writing SPEC.md |

---

*Phase: 09-activity-ontology*
*Spec created: 2026-07-31*
*Next step: /gsd-discuss-phase 9 — implementation decisions (CTE helper extraction, test fixture strategy, migration split review)*
