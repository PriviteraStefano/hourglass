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

**Status:** Planned — 1 plan, 8 tasks, 3 waves

**Goal:** Contract CRUD with customer dropdown, project display on detail page, delete protection.

**Depends on:** Phase 3

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 4)

**Day:** Fri June 12 (parallel with Phase 5)

**Note:** Replaces old Phase 3 + Phase 4 — combined with broader scope.

### Key behaviors

- Contract list page with filtering (status, org) — ✅ already built
- Create contract — includes customer dropdown (from Phase 3) — ⏳ customer_id wiring (plan 01)
- Edit contract — ✅ already built (customer Select on detail page)
- Delete contract — blocked if has active projects — ⏳ HasProjects check (plan 01)
- Projects list displayed on contract detail page (from Phase 5) — ✅ already built
- "Internal customer" option in customer selector — ⏳ combobox UX (plan 01)
- Zero-value contracts allowed — ✅ already working

### Scope of this phase

Backend gaps: add `customer_id` to CreateContractRequest, add `HasProjects` to delete protection, add `ErrHasActiveProjects` error.
Frontend gaps: add customer combobox to CreateContractDialog, add `customer_id` to frontend type.
Tests: backend service unit tests + frontend API tests.

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [ ] 04-01 | Contract create with customer + HasProjects delete guard | 1 | 8 | 13 |

### Edge cases

- Contract without customer (existing data) still renders — ✅ already handled
- Contract with adopted projects shown differently — deferred to Phase 5
- Zero-value contracts allowed — ✅ already working
- Create with "No customer" stores NULL in DB — covered by nullable `*uuid.UUID`
- Delete blocked by projects returns specific 409 — distinct from time entries 409

---

## Phase 5: Projects

**Status:** Planned — 4 plans, 10 tasks, 3 waves

**Goal:** Project CRUD with subproject support, org-scope filtering, governance models.

**Depends on:** Phase 4

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 5)

**Day:** Fri June 12 (parallel with Phase 4)

**Note:** New phase — no equivalent in previous roadmap. Replaces old Phase 5 (seed data — superseded).

### Key behaviors

- Project list page with filtering (contract_id, org_id, type: billable/internal) — ✅ already built
- Create project (name, type, contract, governance model, scope) — ✅ already built
- Edit project — ⏳ dialog-based (Plan 01 backend + Plan 03 frontend)
- Delete project — blocked if has time entries or subprojects — ⏳ protection checks (Plan 01 + Plan 02)
- Subproject list on project detail — ⏳ subprojects endpoint + expandable section (Plan 02 + Plan 03)
- Distinct 409 errors for direct vs subproject active entries — ⏳ (Plan 02 handler)
- Adopted projects display creation org — ✅ already works

### Scope of this phase

Backend additions: UpdateProjectRequest domain type, Update/Delete/HasActiveTimeEntries on repository, Update/Delete on service with finance role gating + owner check + active time entry protection, Update/Delete/ListSubprojects on handler, route registration.
Frontend additions: UpdateProjectRequest type, update/delete/subprojects API hooks, EditProjectDialog component, wired Edit/Delete buttons on detail page, expandable subproject section.

### Plans

| Plan | Objective | Wave | Tasks | Files |
|------|-----------|------|-------|-------|
| [ ] 05-01 | Backend: domain + ports + mocks + repo + service for Update/Delete/HasActiveTimeEntries | 1 | 3 | 5 |
| [ ] 05-02 | Backend: HTTP handlers (Update/Delete/ListSubprojects) + route wiring | 2 | 2 | 2 |
| [ ] 05-03 | Frontend: API types + mutations/queries + EditProjectDialog + detail page wiring | 1 | 3 | 4 |
| [ ] 05-04 | Tests: service tests + handler tests + build verification | 3 | 3 | 2 |

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

**Status:** Not started

**Goal:** Full CRUD + approval workflow for time entries and expenses.

**Depends on:** Phase 5

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 6)

**Day:** Sat-Sun June 13-14

### Key behaviors

- Time entry list with filtering (status, date range, project, user)
- Create time entry (date, hours, project, subproject, WG, description)
- Edit time entry (draft/submitted only)
- Submit for approval
- Approval workflow: employee creates → submits → manager approves → finance approves/rejects
- Same pattern for expenses (mileage, meal, accommodation, other)

### Backend

Already exists (TimeEntryRepository, ExpenseRepository + handlers)

### Frontend files to create

- `web/src/routes/_authenticated/time-entries/index.tsx`
- `web/src/routes/_authenticated/time-entries/new.tsx`
- `web/src/routes/_authenticated/time-entries/$id.tsx`
- Same structure for expenses
- Components: time-entry-form.tsx, expense-form.tsx, approval-buttons.tsx, status-badge.tsx, approval-history.tsx

### Edge cases

- Cannot edit approved/rejected entries
- Cannot delete entries with approvals
- Employee cannot self-approve
- Manager cannot approve own entries
- Rejected entries show reason
- Approval history is immutable

---

## Phase 7: Exports

**Status:** Not started

**Goal:** Downloadable CSV/Excel exports for timesheets, expenses, and combined reports.

**Depends on:** Phase 6

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 7)

**Day:** Sun June 14

### Key behaviors

- Timesheet export (CSV/Excel) with date range filter
- Expense export (CSV/Excel) with date range filter
- Combined export with both time + expense data
- Download as file

### Backend

Already exists (ExportRepository, Export handler)

### Frontend files to create

- `web/src/routes/_authenticated/exports/index.tsx`
- Components: export-form.tsx (date range, type selector, download button)

### Edge cases

- Empty export (no data in range) shows friendly message
- Large export handled server-side
- Auth required for exports (user-scoped data)

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
```

**Parallel execution:**

- Phase 2 + Phase 3 (parallel, both depend on Phase 1)
- Phase 4 + Phase 5 (parallel, Phase 4 depends on Phase 3, Phase 5 depends on Phase 4)
