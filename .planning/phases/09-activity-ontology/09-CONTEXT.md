# Phase 9: Activity Ontology (Big-Bang Migration + Routing Rewrite) - Context

**Gathered:** 2026-07-31
**Status:** Ready for planning
**Source:** [[ADR-P-007 — Activity Ontology]] (Accepted) + [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] (Accepted) + [[ADR-P-008 — Availability & Employment Validity]] (schema only, rides the batch) + [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] (rename consequences)

<domain>
## Phase Boundary

Replace the rigid two-table work decomposition (`projects` + `subprojects`) with the single recursive `activities` entity, rewrite every FK and approval-routing query onto the new ontology, and land the additive staffing schema (ADR-P-008) in the same migration batch. This phase is **backend + database only** — the frontend rename (`/projects` → `/activities`) and all new surfaces (Today, Approvals, Working Groups, Availability) are Phase 10.

The phase ends when: (a) the migration applies cleanly up and down, (b) the backend compiles against the collapsed repositories, (c) all existing Go tests pass against the new schema, (d) the staffing schema is live, and (e) the frontend is left in a compiling-but-stale state, quarantined for Phase 10.

This phase is deliberately sequenced AFTER Phase 8 (P0 fixes): the migration rewrites the same handlers/repos the P0 list touches — fixing first means testing the final shape once, not twice (08-CONTEXT.md).
</domain>

<decisions>
## Implementation Decisions

All decisions are locked by the ADRs — they are the spec for this phase. No discuss-phase; this is an executory phase.

### ADR-P-007 — The ontology (Accepted 2026-07-29)

- **D-1** The table, domain type, and user-facing word is **activity** (`activities`). The legacy word "project" survives only in `activity_managers` (governance role name) and historical references.
- **D-2** `activities.parent_id` is nullable, self-referencing — nesting is data, not schema. `kind` is a free label from an org-level catalog (`activity_kinds`), seeded with `engagement`, `phase`, `task`, `internal`. No level ladder, no kind↔depth constraint.
- **D-3** `activities.contract_id` is nullable. Activities without a contract are internal work, first-class. Commercial context is **derived, not stored** — resolved by walking `parent_id` upward to the nearest ancestor with a `contract_id` (recursive CTE).
- **D-4** Both `time_entries` and `expenses` reference exactly one `activity_id`, NOT NULL on both. `time_entries`: drop `project_id`, `subproject_id`, `wg_id`; keep `unit_id NOT NULL` (accountability pin). `expenses`: drop `project_id`, `customer_id`; keep `unit_id NOT NULL`.
- **D-5** `working_groups.subproject_id` → `working_groups.activity_id`. A WG anchors at any depth. WGs are not mandatory on solo activities (D-8).
- **D-6** Big-bang migration, pre-deploy. One migration replaces the tables, rewrites all FKs, drops `enforce_unit_tuple`. Data migration is trivial (MVP seed only): existing projects → activities with `engagement` kind; subprojects → children.
- **D-7/D-8** `billable` is nullable (NULL = inherit from contract link / nearest ancestor). Personal/internal activities (learning, briefings) capture with the same fidelity as project work; approvals fall back to the unit manager.

### ADR-BE-014 — Routing & resolution (Accepted 2026-07-29)

- **R-1** Every entry resolves routing from its `activity_id` — never from pinned FKs. Chain: activity → anchored WG → WG manager/delegate (manager stage) → org finance role (finance stage). Commercial chain derived separately via upward CTE.
- **R-2** Manager-stage precedence: (1) WG anchored to the activity → WG manager or delegate; (2) no anchored WG (personal activity, D-8) → submitter's unit manager (nearest manager walking the unit tree upward). Commercial activities **must** anchor a WG before accepting entries — service-layer validation, sentinel `ErrActivityNotLoggable`.
- **R-3** D-11 skip, delegates included: if the entry's WG manager **or any delegate** is the entry owner, `submitted` → `pending_finance` directly. Never true self-approval — the skip fires only when the approver *role* coincides with the owner.
- **R-4** Visibility: unit-subtree gating via `unit_memberships.role = 'manager'` + recursive CTE. Org-role manager/finance see the whole org. Routing (R-1/R-2) and visibility (R-4) are separate axes.
- **R-5** Schema consequences: `time_entries` drop `project_id`/`subproject_id`/`wg_id`, add `activity_id NOT NULL`; `expenses` drop `project_id`/`customer_id`, add `activity_id NOT NULL`; `working_groups.subproject_id` → `activity_id`, drop `enforce_unit_tuple`; `project_managers` → `activity_managers`; `financial_cutoff_periods.project_id` → `activity_id`, `IsPeriodLocked` key becomes org + activity + date range.
- **R-6** Repository collapse: `project_repository` + `subproject_repository` → one `activity_repository`. One endpoint set replaces two. Approval `ListPending` for both entry types queries through the activity → WG join.

### ADR-P-008 — Staffing schema (schema only, rides the batch)

- One new table `availability_windows` (id, org_id, user_id, kind CHECK `holiday|permit|medical|unavailable`, starts_on, ends_on, hours, certificate_ref, note, status CHECK `declared|confirmed`, created_by, created_at).
- Three new columns on `organization_memberships`: `valid_from`, `valid_until`, `work_permit_expires_at` (all DATE, nullable).
- Extend `organization_memberships.role` CHECK: add `'hr'`.
- **Zero FK coupling** to the activity rewrite — same migration batch, separate file.
- **No surfaces in this phase.** Window declaration UI, assignment-time warnings, HR curator surfaces, and the payroll export view are all Phase 10 or later.

### ADR-P-011 — Rename consequences (D-6)

- `/projects` → `/activities` is the user-facing route change. **Deferred to Phase 10** — this phase only ensures the backend API surface is activity-shaped so the frontend rename is a pure rename.
- `/org-hierarchy` keeps its URL (cosmetic renames rejected pre-deploy).

### Claude's Discretion

- Migration file splitting strategy (one big-bang file vs. two: ontology + staffing) — ADR-BE-004's append-only rule governs; both are acceptable as long as the batch is atomic.
- Internal helper/CTE extraction strategy for the upward-walk (commercial chain) and unit-subtree queries — the recursive-CTE pattern already exists in the units repository; reuse it.
- Test fixture strategy for the new schema — testcontainers infrastructure is stable from Phase 0.

### Out of scope

- Frontend rename, new pages (Today, Approvals, WG, Availability), sidebar regrouping — **Phase 10** (ADR-P-011).
- Payroll export view, availability window UI, assignment-time warnings, expiry queues — Phase 10 or v0.2 per ADR-P-008 sequencing note.
- P1 audit items (skeletons, OrgID validation, JSON decode, cookie names, audit-log context) — deferred (candidate Phase 12).
- P2 items — post-v0.1 backlog.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The spec (ADR layer)
- `hourglass-vault/decisions/project/ADR-P-007 — Activity Ontology.md` — the ontology decision (D-1…D-8, table sketch)
- `hourglass-vault/decisions/backend/ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution.md` — routing, visibility, schema consequences, repo collapse (R-1…R-6)
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` — staffing schema sketch (D-1…D-5, SQL)
- `hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md` — rename + surface consequences (D-1…D-6)

### Code anchors (verify before planning)
- `migrations/` — sequential numbered; 010 is the next free number (004/005 already widened status CHECKs; 009 added `is_internal` to customers)
- `internal/core/domain/` — `project.go`, `subproject.go` → collapse to `activity.go`; `time_entry.go`, `expense.go` FK fields change
- `internal/core/ports/` — `project_repository.go`, `subproject_repository.go` → `activity_repository.go`; `time_entry_repository.go`, `expense_repository.go` query signatures change
- `internal/adapters/secondary/postgres/` — `project_repository.go`, `subproject_repository.go` → `activity_repository.go`; `time_entry_repository.go`, `expense_repository.go`, `working_group_repository.go`, `financial_cutoff_period_repository.go` all rewrite queries
- `internal/core/services/` — `time_entry/`, `expense/` approval routing logic (ListPending, Submit, Approve, Reject) rewrite per R-1/R-2/R-3
- `internal/adapters/primary/http/` — `project_handler.go`, `subproject_handler.go` → `activity_handler.go`; `time_entry_handler.go`, `expense_handler.go` request/response DTOs change
- `web/src/` — **do not touch**; quarantine for Phase 10
- `.planning/phases/08-pre-deployment-hardening-p0-audit-fixes/` — must complete before this phase starts (green tests = migration safety net)
</canonical_refs>

<specifics>
## Specific Ideas

- The migration batch should be **two files**: `010_activity_ontology.up.sql` (the P-007 rewrite) + `011_staffing_schema.up.sql` (the P-008 additive schema). Separate files keep the blast radius reviewable; same deploy batch keeps it atomic. Both get matching `.down.sql`.
- Data migration: MVP seed only. Existing `projects` rows → `activities` with `kind = 'engagement'`, `contract_id` preserved, `parent_id = NULL`. Existing `subprojects` rows → `activities` with `kind = 'task'` (or `phase` — review seed data to pick), `parent_id` = the parent's new activity id, `contract_id` inherited from parent. One `UPDATE` pass after the table swap.
- The recursive CTE for commercial-chain resolution already has a working pattern in `unit_repository.go` (subtree walk). Extract a shared helper or duplicate the pattern — either is fine, but it must be tested against a 3-level activity tree.
- `activity_kinds` seed: `engagement`, `phase`, `task`, `internal` with `is_seed = TRUE`, `org_id` = the seed org. Org-extensible per D-2.
- The `ErrActivityNotLoggable` sentinel (R-2) needs a corresponding HTTP 409 mapping per ADR-BE-001.
- Frontend quarantine: after the backend rewrite, the frontend will fail to compile (types reference `project_id`, `subproject_id`, etc.). That's expected — Phase 10 fixes it. Mark the frontend build as `// TODO(phase-10): ontology rename` in `routeTree.gen.ts` or a top-level file if needed to keep CI green.
</specifics>

<deferred>
## Deferred Ideas

- Frontend rename + new surfaces (Today, Approvals, WG, Availability) → Phase 10 (ADR-P-011)
- Payroll export view → Phase 10 or v0.2 (ADR-P-008 D-1c)
- Assignment-time availability warnings → v0.2 (needs WG surface from Phase 10)
- P1 audit batch → candidate Phase 12
- P2 batch → post-v0.1 backlog
</deferred>

---

*Phase: 09-activity-ontology*
*Context gathered: 2026-07-31 via ADR-ingest express path (manual, bypassing discuss-phase per GSD executory-phase rule)*
