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

**Status:** In progress — Plan 01-01 complete, Plans 01-02/01-03 pending

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

- [ ] 01-03-PLAN.md — End-to-end verification of all AUTH requirements AUTH-01-05

---

## Phase 2: Org Hierarchy

**Status:** Not started

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

### Backend

Already exists (UnitRepository, UnitMemberRepository, Unit handler)

### Frontend files to create

- `web/src/routes/_authenticated/org-hierarchy/index.tsx`
- `web/src/routes/_authenticated/org-hierarchy/units/$id.tsx`
- `web/src/routes/_authenticated/org-hierarchy/members/$id.tsx`
- Components: org-tree.tsx (ReactFlow), unit-form.tsx, member-list.tsx

### Edge cases

- Cannot delete unit with children (reassign parent first)
- Cannot delete unit with members (remove members first)
- Root unit cannot be deleted
- Circular parent reference prevention

---

## Phase 3: Customers

**Status:** Not started

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

---

## Phase 4: Contracts

**Status:** Not started

**Goal:** Contract CRUD with customer dropdown, project display on detail page, delete protection.

**Depends on:** Phase 3

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 4)

**Day:** Fri June 12 (parallel with Phase 5)

**Note:** Replaces old Phase 3 + Phase 4 — combined with broader scope.

### Key behaviors

- Contract list page with filtering (status, org)
- Create contract — includes customer dropdown (from Phase 3)
- Edit contract
- Delete contract (blocked if has active projects)
- Projects list displayed on contract detail page (from Phase 5)
- "Internal customer" option in customer selector

### Backend

Already exists (ContractRepository, Contract handler)

### Frontend files to create

- `web/src/routes/_authenticated/contracts/index.tsx`
- `web/src/routes/_authenticated/contracts/new.tsx`
- `web/src/routes/_authenticated/contracts/$id.tsx`
- Components: contract-form.tsx, contract-table.tsx, projects-on-contract.tsx

### Edge cases

- Contract without customer (existing data) still renders
- Contract with adopted projects shown differently
- Zero-value contracts allowed

---

## Phase 5: Projects

**Status:** Not started

**Goal:** Project CRUD with subproject support, org-scope filtering, governance models.

**Depends on:** Phase 4

**Source:** `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` (Feature 5)

**Day:** Fri June 12 (parallel with Phase 4)

**Note:** New phase — no equivalent in previous roadmap.

### Key behaviors

- Project list page with filtering (contract_id, org_id, type: billable/internal)
- Create project (name, type, contract, governance model, scope)
- Edit project
- Delete project (blocked if has time entries or subprojects)
- Subproject list on project detail

### Backend

Already exists (ProjectRepository, SubprojectRepository, Project handler)

### Frontend files to create

- `web/src/routes/_authenticated/projects/index.tsx`
- `web/src/routes/_authenticated/projects/new.tsx`
- `web/src/routes/_authenticated/projects/$id.tsx`
- Components: project-form.tsx, project-table.tsx, subproject-list.tsx

### Edge cases

- Adopted projects (shared across orgs) display creation org
- Shared vs org-scoped project filtering
- Project without contract (internal projects)

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
