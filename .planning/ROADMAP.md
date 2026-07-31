# Hourglass Roadmap

## Current Milestone: v0.1 MVP Consolidation

## Phase 0: Testing foundation

**Status:** Complete — all 6 plans executed, testcontainers infrastructure operational, auth bugs fixed, full test suite green, E2E verified

**Goal:** Test infrastructure using testcontainers-go for PostgreSQL-backed service-layer integration tests. Fix known auth bugs. Loop until all bugs cleared. Establishes foundation for all feature phases.

**Depends on:** None

### Original plans (SurrealDB era — superseded)

The 9 original plans were executed against SurrealDB test infrastructure. Service-layer tests used in-memory mocks (still valid for unit tests). Integration/handler tests targeted `GetTestDBWithNamespace()` (SurrealDB helper) and must be rewritten for PostgreSQL.

### PostgreSQL reboot plans

**Waves:**

- Wave 1: Plan 02 (infrastructure)
- Wave 2: Plan 01 (auth fixes — needs testcontainers)
- Wave 3: Plan 03 (service tests)
- Wave 4: Plan 04 (handler tests)
- Wave 5: Plan 05 (bug buffer with human review)
- Wave 6: Plan 06 (E2E verification)

Plans:

- [x] 00-01-PLAN.md — Fix 4 known auth bugs + full auth cleanup (refresh rotation, cookie fix, password reset, rate limiting) TEST-01
- [x] 00-02-PLAN.md — Set up testcontainers-go infrastructure (replaces TestPool) TEST-02
- [x] 00-03-PLAN.md — Rewrite service-layer integration tests against PostgreSQL TEST-03
- [x] 00-04-PLAN.md — Rewrite handler integration tests for PostgreSQL TEST-04
- [x] 00-05-PLAN.md — Bug buffer: batch-fix all bugs discovered during PG test rewrite TEST-05
- [x] 00-06-PLAN.md — Verify Playwright E2E still works TEST-06

---

## Phase 1: Authorization

**Status:** Complete — all 3 plans executed, backend auth fixes deployed, frontend auth integration done, E2E verified

**Goal:** Frontend auth integration — login/register pages, protected route handling, auth hydration for all features.

**Depends on:** Phase 0 (backend auth bugs fixed)

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 1)

**Day:** Tue June 9

### Frontend scope

- Login page with error handling
- Register page with validation
- Protected route redirect (already wired in `_authenticated.tsx`, verify it works)
- AppShell with user profile, org switcher, logout

### Verification

- All 6 seed users login successfully (alex.rivera, sarah.chen, mike.obrien, emma.wilson, james.park, lisa.torres / demo123)
- `GET /auth/me` returns full profile with role + org_id
- `GET /auth/memberships` returns membership list without panic
- Cookie refresh flow works (401 → POST /auth/refresh → retry)
- Login form validation works (empty fields, wrong password, invalid email)

### Waves

- Wave 1: Plan 01 (backend) + Plan 02 (frontend) — parallel
- Wave 2: Plan 03 (E2E verification)

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Backend auth fixes: Register token generation + cookies, password reset entropy AUTH-01, AUTH-02
- [x] 01-02-PLAN.md — Frontend auth integration: OrgSwitcher, / redirect, password reset API type AUTH-03, AUTH-04, AUTH-05

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-03-PLAN.md — End-to-end verification of all AUTH requirements AUTH-01-05

---

## Phase 2: Org Hierarchy

**Status:** Complete — all 3 plans executed, delete protection enforced (root/children/members), PUT member endpoint Live, batch members endpoint Live, reparent dialog uses dedicated mutation, "Make Primary" button in side panel, subtree member expandable groups

**Goal:** Organization tree visualization using ReactFlow, unit CRUD with parent-unit hierarchy, member management, edge-driven reparenting.

**Depends on:** Phase 1

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 2)

**Day:** Wed-Thu June 10-11 (parallel with Phase 3)

**Note:** Replaces old Phase 1 (org-hierarchy-edge-driven) — rewritten with PostgreSQL context, broader scope.

### Key behaviors

- Unit CRUD (create, read, update, delete) with parent-unit hierarchy
- Unit member management — add/remove members, designate primary unit
- Multi-unit membership — a user can belong to multiple units
- Org tree visualization using ReactFlow
- Edge-driven reparenting — drag edge to reassign unit parent

### Status per requirement

| Req | Description | Status |
|-----|-------------|--------|
| ORG-01 | Org tree visualization using ReactFlow | ✅ Already implemented in org-hierarchy-page.tsx |
| ORG-02 | Unit CRUD with parent-unit hierarchy | ✅ Already implemented (backend + frontend dialogs) |
| ORG-03 | Member management — add/remove/primary | ✅ Complete: PUT endpoint + frontend primary UI + subtree members |
| ORG-04 | Edge-driven reparenting | ✅ Complete: reparent dialog uses dedicated mutation |
| ORG-05 | Delete protection (root/children/members) | ✅ Complete: backend enforcement by sentinel errors |

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 02-01 | Backend: delete protection + PUT member endpoint + batch members | 1 | 3 | 9 |
| [x] 02-02 | Frontend: reparent mutation switch + pendingEdgeConnect cleanup | 1 | 2 | 3 |
| [x] 02-03 | Frontend: "Make Primary" UI + subtree member groups | 2 | 2 | 3 |

### Edge cases

- Cannot delete unit with children (reassign parent first)
- Cannot delete unit with members (remove members first)
- Root unit cannot be deleted
- Circular parent reference prevention

---

## Phase 3: Customers

**Status:** Complete — all 3 plans executed, full customer CRUD with internal customers, search, detail page, sidebar nav, and test coverage

**Goal:** Full customer CRUD with internal customer support, search/filter, delete protection.

**Depends on:** Phase 1

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 3)

**Day:** Wed-Thu June 10-11 (parallel with Phase 2)

**Note:** Replaces old Phase 2 (customers-management-page) — broader scope including internal customer handling.

### Key behaviors

- Customer list page with search/filter
- Create customer (name, contact name, email, phone, VAT, address)
- Edit customer
- Delete customer — 409 if has active contracts
- Internal customer (organization itself) visual indicator

### Backend

Already exists (CustomerRepository, Customer handler)

### Frontend files to create

- `web/src/routes/_authenticated/customers/index.tsx`
- `web/src/routes/_authenticated/customers/new.tsx`
- `web/src/routes/_authenticated/customers/$id.tsx`
- Components: customer-form.tsx, customer-table.tsx

### Edge cases

- Delete blocked with active contracts (show tooltip with contract count)
- Internal customer shown with badge, non-editable fields
- Empty state when no customers

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 03-01 | Backend: DB migration + search + internal customer | 1 | 3 | 8 |
| 03-02 | Frontend: API layer + sidebar + detail page | 2 | 2 | 5 |
| 03-03 | Frontend polish: badge, form lock, tests | 3 | 2 | 4 |

---

## Phase 4: Contracts

**Status:** Complete — all 2 plans executed, customer combobox with internal indicator, HasProjects delete guard, frontend + backend tests passing

**Goal:** Contract CRUD with customer dropdown, project display on detail page, delete protection.

**Depends on:** Phase 3

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 4)

**Day:** Fri June 12 (parallel with Phase 5)

**Note:** Replaces old Phase 3 + Phase 4 — combined with broader scope.

### Key behaviors

- Contract list page with filtering (status, org) — ✅ already built
- Create contract — includes customer dropdown (from Phase 3) — ✅ customer_id wiring (plan 01)
- Edit contract — ✅ already built (customer Select on detail page)
- Delete contract — blocked if has active projects — ✅ HasProjects check (plan 01)
- Projects list displayed on contract detail page (from Phase 5) — ✅ already built
- "Internal customer" option in customer selector — ✅ combobox UX with "(Internal)" suffix
- Zero-value contracts allowed — ✅ already working

### Scope of this phase

Backend gaps: add `customer_id` to CreateContractRequest, add `HasProjects` to delete protection, add `ErrHasActiveProjects` error. ✅ Done in plan 01.
Frontend gaps: add customer combobox to CreateContractDialog, add `customer_id` to frontend type.
Tests: backend service unit tests completed, frontend API tests completed.

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 04-01 | Contract create with customer + HasProjects delete guard | 1 | 5 | 7 |
| [x] 04-02 | Frontend customer combobox + internal customer indicator | 1 | 3 | 3 |

### Edge cases

- Contract without customer (existing data) still renders — ✅ already handled
- Contract with adopted projects shown differently — deferred to Phase 5
- Zero-value contracts allowed — ✅ already working
- Create with "No customer" stores NULL in DB — covered by nullable `*uuid.UUID`
- Delete blocked by projects returns specific 409 — distinct from time entries 409

---

## Phase 5: Projects

**Status:** All 4 plans complete — 10 tasks, 3 waves. Service + handler + handler test coverage for Update/Delete/ListSubprojects.

**Goal:** Project CRUD with subproject support, org-scope filtering, governance models.

**Depends on:** Phase 4

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 5)

**Day:** Fri June 12 (parallel with Phase 4)

**Note:** New phase — no equivalent in previous roadmap. Replaces old Phase 5 (seed data — superseded).

### Key behaviors

- Project list page with filtering (contract_id, org_id, type: billable/internal) — ✅ already built
- Create project (name, type, contract, governance model, scope) — ✅ already built
- Edit project — ⏳ dialog-based (Plan 01 backend ✅ + Plan 03 frontend)
- Delete project — blocked if has time entries or subprojects — ✅ protection checks (Plan 01 backend + Plan 02 handler + Plan 04 tests)
- Subproject list on project detail — ✅ subprojects endpoint + expandable section (Plan 02 + Plan 03)
- Distinct 409 errors for direct vs subproject active entries — ✅ (Plan 02 handler + Plan 04 tests)
- Adopted projects display creation org — ✅ already works

### Scope of this phase

Backend additions: UpdateProjectRequest domain type, Update/Delete/HasActiveTimeEntries on repository, Update/Delete on service with finance role gating + owner check + active time entry protection, Update/Delete/ListSubprojects on handler, route registration.
Frontend additions: UpdateProjectRequest type, update/delete/subprojects API hooks, EditProjectDialog component, wired Edit/Delete buttons on detail page, expandable subproject section.

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 05-01 | Backend: domain + ports + mocks + repo + service for Update/Delete/HasActiveTimeEntries | 1 | 3 | 5 |
| [x] 05-02 | Backend: HTTP handlers (Update/Delete/ListSubprojects) + route wiring | 2 | 2 | 2 |
| [x] 05-03 | Frontend: API types + mutations/queries + EditProjectDialog + detail page wiring | 1 | 3 | 5 |
| [x] 05-04 | Tests: service tests + handler tests + build verification | 3 | 3 | 2 |

### Edge cases

- Adopted projects (shared across orgs) display creation org — ✅ already works
- Shared vs org-scoped project filtering — ✅ already works
- Project without contract (internal projects) — contract is required per D-10
- Delete blocked by active time entries on project — returns distinct 409 (D-06)
- Delete blocked by active time entries on subprojects — returns distinct 409 (D-04, D-06)
- Delete cascade-cleans adoptions — transactions (D-05)
- Non-owner org cannot delete — forbidden (D-03)
- Edit/Delete requires finance role — matches contract pattern

---

## Phase 6: Time Entries + Expenses

**Status:** All 5 plans complete — full time entry + expense frontend UI with client-side calendar, approval workflow, receipt upload

**Goal:** Full CRUD + two-stage approval workflow for time entries and expenses. Flat model (one entry per project per date), client-side month computation, shared approval components.

**Depends on:** Phase 5

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 6)

**Day:** Sat-Sun June 13-14

### Key behaviors

- Time entry list with filtering (status, date range, project, user) — TIME-01
- Create time entry (date, hours, project, subproject, WG, description) — TIME-02
- Edit time entry (draft/submitted/rejected only) — TIME-03
- Submit for approval — TIME-04
- Cannot edit approved/rejected entries — TIME-05
- Cannot delete entries with approvals — TIME-06
- Employee cannot self-approve — TIME-07
- Manager cannot approve own entries — TIME-08
- Same pattern for expenses (9 categories, receipt upload) — EXPN-01-06
- Two-stage approval workflow: draft → submitted → pending_manager → pending_finance → approved/rejected — APPR-03
- Approval history is immutable — APPR-01
- Rejected entries show reason — APPR-02

### Waves

| Wave | Plans | Description |
|------|-------|-------------|
| 1 | Plan 01 | Backend foundation: domain, ports, migrations, mocks |
| 2 | Plan 02 + Plan 03 (parallel) | Service layer + handler/repo/route wiring |
| 3 | Plan 04 + Plan 05 (sequential) | Frontend time entry UI + expense UI |

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 06-01 | Backend Foundation: domain models, port interfaces, mocks, factories, migrations | 1 | 2 | 10 |
| [x] 06-02 | Backend Service Layer: two-stage approval in TimeEntryService, full ExpenseService, unit tests | 2 | 3 | 4 |
| [x] 06-03 | Backend Repos + Handlers + Route: PG repo extensions, ExpenseHandler, route wiring, tests | 2 | 3 | 7 |
| [x] 06-04 | Frontend Time Entry: flat model rewrite, client-side calendar, shared approval components | 3 | 3 | 10 |
| [x] 06-05 | Frontend Expenses: types, API, route, calendar, detail, row, sidebar link | 3 | 3 | 10 |

### Edge cases

- Cannot edit approved/rejected entries
- Cannot delete entries with approvals
- Employee cannot self-approve
- Manager cannot approve own entries
- Rejected entries show reason
- Approval history is immutable
- WG manager == creator: skip manager approval stage (D-11)
- km_distance only meaningful for mileage category (D-07)

---

## Phase 7: Exports

**Status:** Complete — all 3 plans executed

**Goal:** Downloadable CSV/XLSX exports for timesheets, expenses, and combined reports with date range filtering, format selection, role-scoped data, and auth requirement.

**Depends on:** Phase 6

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 7)

**Day:** Sun June 14

### Key behaviors

- Timesheet export (CSV/Excel) with date range filter — EXPT-01
- Expense export (CSV/Excel) with date range filter — EXPT-02
- Combined export with both time + expense data — EXPT-03
- Download as file via fetch+blob — EXPT-04
- Empty export shows friendly toast message — EXPT-05
- Auth required, user-scoped data — EXPT-06

### Waves

| Wave | Plans | Description |
|------|-------|-------------|
| 1 | Plan 01 + Plan 02 (parallel) | Backend extensions (count endpoints, XLSX, format param, filters) + Frontend core (hook, API module, ExportForm component, sidebar) |
| 2 | Plan 03 (depends on 02) | Export tabs on time-entries and expenses pages |

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 07-01 | Backend: count endpoints, XLSX generation, format param, CSV streaming, project/user filters | 1 | 3 | 10 |
| [x] 07-02 | Frontend: useDownload hook, API module, ExportForm component, combined page sidebar | 1 | 3 | 5 |
| [x] 07-03 | Frontend: Export tabs on time entries and expenses pages | 2 | 2 | 2 |

### Edge cases

- Empty export (no data in range) shows friendly toast
- Large export handled server-side with CSV streaming + count pre-check
- Auth required for exports (user-scoped data) — backend middleware.Auth() on all routes
- 1-year max date range enforced both frontend and backend
- XLSX combined export uses two sheets (Timesheets + Expenses)

## Phase 8: Pre-Deployment Hardening (P0 audit fixes)

**Status:** Complete — 4/4 plans executed (P0 gate closed)

**Goal:** Close the remaining P0 findings from the 2026-07-28 Pre-Deployment Audit after code verification (2026-07-31): P0-2 list views, P0-3 `/customers` route, P0-4 error boundaries, P0-5-lite refresh-token reuse detection — plus folded-in S3 input length caps. P0-1 (status CHECK) and P0-6 (reset-code exposure) verified already-fixed pre-audit. Phase gates first deployment of v0.1.

**Depends on:** Phase 7

**Source:** `hourglass-vault/research/2026-07-28 — Pre-Deployment Audit — Hourglass v0.1.md` (§6 P0 matrix + 2026-07-31 Corrections)

**Context:** `08-CONTEXT.md` — audit-ingest express path (discuss-phase bypassed: audit is the spec, all decisions locked)

### Key behaviors

- ~~P0-1: widen time-entry status CHECK constraint~~ — ✅ already fixed by `004_time_entries_status_check` (verified 2026-07-31)
- P0-2: real filterable list views on the time-entries and expenses List tabs (shared table shell, URL-driven filters)
- P0-3: `/customers` index route (list page exists but is unreachable)
- P0-4: route error boundaries per ADR-FE-014 (layout default + auth slim variant)
- P0-5: refresh-token **reuse detection** on top of existing rotation — `family_id` + `rotated_at`, replay → `ErrTokenReuse` + family revocation, atomic rotate tx (ADR-FE-013 mechanism unchanged)
- ~~P0-6: reset code out of response body~~ — ✅ already fixed + regression-tested per D-16 (verified 2026-07-31)
- S3: request string length caps at the handler boundary (400 on violation)

### Waves

| Wave | Plans | Description |
|------|-------|-------------|
| 1 | 08-01 + 08-02 (parallel) | Backend security hardening ∥ Frontend completion |
| 2 | 08-03 + 08-04 (parallel) | Backend regression tests ∥ Frontend E2E + audit closeout |

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 08-01 | Backend: refresh-token reuse detection (family model, atomic rotate) + input length caps | 1 | 2 | 21 |
| [x] 08-02 | Frontend: /customers route, time/expense list views + shared table, error boundaries | 1 | 5 | ~9 |
| [x] 08-03 | Backend regression tests: reuse detection (incl. race), S3 caps, E2E auth cookie-rotation | 2 | 3 | ~5 |
| [x] 08-04 | Frontend E2E: list views, customers, error recovery; audit P0 table closeout; 00-Index | 2 | 4 | ~7 |

### Edge cases

- Rotation reuse race: two concurrent refreshes with one token → exactly one succeeds, family revokes on replay
- Down-migration of 010 must not fail if rows already carry the new states (rollback only on clean DB or after state cleanup)
- Password reset: unknown email returns identical-shape 200 (no enumeration)
- Filter state must survive page reload via URL params (validateSearch)
- Error boundary retry must re-run the loader (router reset), not just re-render
- Phase explicitly does NOT pre-rename projects→activities (Phase 9, ADR-P-007)

---

## Phase 9: Activity Ontology (Big-Bang Migration + Routing Rewrite)

**Status:** Complete — 5/5 plans executed

**Goal:** Replace `projects`+`subprojects` with the recursive `activities` entity, rewrite all FKs and approval routing onto the new ontology, land the additive staffing schema (ADR-P-008), and leave the backend API fully activity-shaped. Backend + database only — frontend rename and new surfaces are Phase 10.

**Depends on:** Phase 8 (green tests = migration safety net)

**Source:** `hourglass-vault/decisions/project/ADR-P-007 — Activity Ontology.md` (Accepted) + `hourglass-vault/decisions/backend/ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution.md` (Accepted) + `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` (schema only)

**Context:** `09-CONTEXT.md` — ADR-ingest express path (discuss-phase bypassed: ADRs are the spec, all decisions locked)

### Key behaviors

- Recursive `activities` table replaces `projects`+`subprojects`; `activity_kinds` org-extensible catalog (P-007 D-1/D-2)
- Commercial context optional + inherited upward via CTE; `billable` nullable (D-3/D-7)
- Entries link through one `activity_id NOT NULL`; `wg_id`/`project_id`/`subproject_id`/`customer_id` dropped from entries (D-4, R-5)
- `working_groups.activity_id` replaces `subproject_id`; `enforce_unit_tuple` dropped (D-5, R-5)
- Approval routing: activity → anchored WG → manager/delegate; unit-manager fallback for personal activities; D-11 skip incl. delegates; `ErrActivityNotLoggable` enforcement (R-1…R-3)
- `project_repository`+`subproject_repository` → one `activity_repository`; one `/api/activities` endpoint set (R-6)
- Staffing schema (P-008): `availability_windows`, membership validity dates, `hr` role — additive, zero-coupled
- Frontend left compiling-but-stale; quarantined for Phase 10

### Waves

| Wave | Plans | Description |
|------|-------|-------------|
| 1 | 09-01 + 09-02 (parallel) | Ontology migration ∥ Staffing schema |
| 2 | 09-03 + 09-04 + 09-05 (sequential within wave) | Domain/repo collapse → service rewrite → handler/router wiring |

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [x] 09-01 | Ontology migration: activities schema + data migration (011 up/down, completed 2026-07-31) | 1 | 2 | 4 |
| [x] 09-02 | Staffing schema: availability_windows + membership validity + hr role (012 up/down, completed 2026-07-31) | 1 | 2 | 4 |
| [x] 09-03 | Domain + repository collapse: Activity entity, ports, PG adapters, CTE queries (completed 2026-07-31) | 2 | 3 | 23 |
| [x] 09-04 | Service layer: routing rewrite (R-1/R-2/R-3) + ActivityService (completed 2026-07-31) | 2 | 3 | 16 |
| [x] 09-05 | HTTP handlers + router: /api/activities, entry DTO updates, route wiring (completed 2026-07-31) | 2 | 3 | 18 |

### Edge cases

- Migration up→down→up cycle must be clean (testcontainers test)
- Seed data: zero orphaned entries post-migration (every entry has a valid `activity_id`)
- `ErrActivityNotLoggable` on commercial activity without anchored WG → 409
- D-11 skip fires for delegates, not for plain WG members
- Frontend will not compile after this phase (types reference old FKs) — expected, Phase 10 fixes it
- `activity_kinds` seed must include the MVP org; org-extensible per D-2

---

## Phase 10: Information Architecture Implementation

**Status:** Not yet planned

**Goal:** Implement ADR-P-011: sidebar regrouping, `/projects` → `/activities` rename, Today landing (ticketless), Approvals queue, Working Groups surface, role-scoped visibility. Requires Phase 9 backend live.

**Depends on:** Phase 9

**Source:** `hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md` (Proposed)

---

## Superseded Phases

The following phases from the previous roadmap are superseded by the new structure:

- **Phase 1 (org-hierarchy-edge-driven)** — Superseded by Phase 2
- **Phase 2 (customers-management-page)** — Superseded by Phase 3
- **Phase 3 (contracts-add-projects-display)** — Superseded by Phase 4
- **Phase 4 (integrate-customers-into-contracts)** — Superseded by Phase 4
- **Phase 5 (mvp-consolidation-seed)** — Delivered as `migrations/003_seed.up.sql`
- **Phase 6 (api-audit)** — Superseded by auth verification in Phase 1 installation

## Dependency Graph

```
Phase 0 (Testing)
   │
Phase 1 (Auth) ─┬─ Phase 2 (Org Hierarchy)
                ├─ Phase 3 (Customers) ── Phase 4 (Contracts) ──┐
                └─ Phase 5 (Projects) ──────────────────────────┤
                                                                 │
                                    Phase 6 (Time Entries + Expenses)
                                             │
                                        Phase 7 (Exports)
                                             │
                                    Phase 8 (P0 Hardening)
                                             │
                                    Phase 9 (Activity Ontology)
                                             │
                                    Phase 10 (IA Implementation)
```

**Parallel execution:**

- Phase 2 + Phase 3 (parallel, both depend on Phase 1)
- Phase 4 + Phase 5 (parallel, Phase 4 depends on Phase 3, Phase 5 depends on Phase 4)
- Phase 9 Wave 1: Plans 09-01 + 09-02 (parallel, different tables)
- Phase 9 Wave 2: Plans 09-03 → 09-04 → 09-05 (sequential — each layer builds on the previous)
