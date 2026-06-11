# Requirements: Hourglass

**Defined:** 2026-06-08
**Core Value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/project management, and export capabilities.

## v0.1 Requirements

Requirements for the remaining v0.1 work. Previous phases (Pg-1/Pg-2/Pg-3 — PostgreSQL migration) are already validated.

### Testing Foundation

- [x] **TEST-01**: Known auth bugs fixed (`/auth/memberships` nil pointer, `/auth/me` empty role/org_id, `/units/{id}/members` 500, `/organizations/members` 500)
- [x] **TEST-02**: testcontainers-go wired for isolated PostgreSQL per test run
- [x] **TEST-03**: Service-layer tests rewritten to run against real PostgreSQL via testcontainers
- [x] **TEST-04**: Handler integration tests rewritten for PostgreSQL
- [ ] **TEST-05**: All bugs discovered during test rewrite fixed
- [x] **TEST-06**: Playwright E2E verified against PostgreSQL backend

### Auth Frontend

- [ ] **AUTH-01**: Login page with error handling
- [ ] **AUTH-02**: Register page with form validation
- [x] **AUTH-03**: Protected route redirect works end-to-end
- [x] **AUTH-04**: AppShell with user profile, org switcher, logout
- [x] **AUTH-05**: Cookie refresh flow works (401 → POST /auth/refresh → retry)

### Org Hierarchy

- [ ] **ORG-01**: Org tree visualization using ReactFlow
- [ ] **ORG-02**: Unit CRUD (create, read, update, delete) with parent-unit hierarchy
- [ ] **ORG-03**: Unit member management — add/remove members, designate primary unit
- [ ] **ORG-04**: Edge-driven reparenting — drag edge to reassign unit parent
- [ ] **ORG-05**: Delete protection — cannot delete unit with children or members, root unit cannot be deleted

### Customers

- [ ] **CUST-01**: Customer list page with search/filter — ✓ Backend search via ILIKE (03-01)
- [ ] **CUST-02**: Create customer (name, contact name, email, phone, VAT, address) — ✓ Infrastructure (is_internal column) (03-01)
- [ ] **CUST-03**: Edit customer — ✓ Company_name lock for internal (03-01)
- [ ] **CUST-04**: Delete customer — 409 if has active contracts (show tooltip with count) — ✓ Backend 409 already existed
- [ ] **CUST-05**: Internal customer (organization itself) visual indicator with badge — ✓ Backend: is_internal column, CreateInternalCustomer, auto-creation, company_name lock (03-01)
- [ ] **CUST-06**: Empty state when no customers

### Contracts

- [ ] **CTRT-01**: Contract list page with filtering (status, org)
- [x] **CTRT-02**: Create contract with customer dropdown
- [ ] **CTRT-03**: Edit contract
- [x] **CTRT-04**: Delete contract — blocked if has active projects
- [ ] **CTRT-05**: Projects list displayed on contract detail page
- [ ] **CTRT-06**: "Internal customer" option in customer selector
- [ ] **CTRT-07**: Zero-value contracts allowed

### Projects

- [ ] **PROJ-01**: Project list page with filtering (contract_id, org_id, type: billable/internal)
- [ ] **PROJ-02**: Create project (name, type, contract, governance model, scope)
- [ ] **PROJ-03**: Edit project
- [ ] **PROJ-04**: Delete project — blocked if has time entries or subprojects
- [ ] **PROJ-05**: Subproject list on project detail
- [ ] **PROJ-06**: Adopted projects display creation org

### Time Entries

- [ ] **TIME-01**: Time entry list with filtering (status, date range, project, user)
- [ ] **TIME-02**: Create time entry (date, hours, project, subproject, WG, description)
- [ ] **TIME-03**: Edit time entry (draft/submitted only)
- [ ] **TIME-04**: Submit for approval
- [ ] **TIME-05**: Cannot edit approved/rejected entries
- [ ] **TIME-06**: Cannot delete entries with approvals
- [ ] **TIME-07**: Employee cannot self-approve
- [ ] **TIME-08**: Manager cannot approve own entries

### Expenses

- [ ] **EXPN-01**: Expense list with filtering (status, date range, project, category)
- [ ] **EXPN-02**: Create expense (date, amount, category, project, description)
- [ ] **EXPN-03**: Edit expense (draft/submitted only)
- [ ] **EXPN-04**: Submit for approval
- [ ] **EXPN-05**: Expense categories: mileage, meal, accommodation, other
- [ ] **EXPN-06**: Same approval workflow constraints as time entries

### Approval Workflow

- [ ] **APPR-01**: Approval history is immutable
- [ ] **APPR-02**: Rejected entries show reason
- [ ] **APPR-03**: Approval workflow: employee creates → submits → manager approves → finance approves/rejects
- [ ] **APPR-04**: Status badge component showing current state
- [ ] **APPR-05**: Approval buttons component (approve/reject) for managers & finance

### Exports

- [ ] **EXPT-01**: Timesheet export (CSV/Excel) with date range filter
- [ ] **EXPT-02**: Expense export (CSV/Excel) with date range filter
- [ ] **EXPT-03**: Combined export with both time + expense data
- [ ] **EXPT-04**: Download as file
- [ ] **EXPT-05**: Empty export shows friendly message
- [ ] **EXPT-06**: Auth required — user-scoped data only

## Out of Scope

| Feature | Reason |
|---------|--------|
| SurrealDB | Fully replaced by PostgreSQL, no longer supported |
| Real-time notifications | High complexity, not core to v0.1 |
| Mobile apps | Web-first, not planned for v0.1 |
| OAuth/Social login | Email/password sufficient for v0.1 |
| CI/CD pipeline | Ad-hoc for now, will add later |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| TEST-01 | Phase 0 | Complete |
| TEST-02 | Phase 0 | Complete |
| TEST-03 | Phase 0 | Complete |
| TEST-04 | Phase 0 | Complete |
| TEST-05 | Phase 0 | Pending |
| TEST-06 | Phase 0 | Complete |
| AUTH-01 | Phase 1 | Pending |
| AUTH-02 | Phase 1 | Pending |
| AUTH-03 | Phase 1 | Complete |
| AUTH-04 | Phase 1 | Complete |
| AUTH-05 | Phase 1 | Complete |
| ORG-01 | Phase 2 | Pending |
| ORG-02 | Phase 2 | Pending |
| ORG-03 | Phase 2 | Pending |
| ORG-04 | Phase 2 | Pending |
| ORG-05 | Phase 2 | Pending |
| CUST-01 | Phase 3 | In progress (backend search done) |
| CUST-02 | Phase 3 | In progress (infrastructure done) |
| CUST-03 | Phase 3 | In progress (company_name lock done) |
| CUST-04 | Phase 3 | In progress (409 already implemented) |
| CUST-05 | Phase 3 | In progress (backend is_internal done) |
| CUST-06 | Phase 3 | Pending |
| CTRT-01 | Phase 4 | In progress (list page already built) |
| CTRT-02 | Phase 4 | Done (plan 01: customer_id on create) |
| CTRT-03 | Phase 4 | In progress (edit already built, detail page exists) |
| CTRT-04 | Phase 4 | Done (plan 01: HasProjects check) |
| CTRT-05 | Phase 4 | In progress (projects list on detail already built) |
| CTRT-06 | Phase 4 | In progress (plan 01: combobox with Internal suffix) |
| CTRT-07 | Phase 4 | In progress (already working — zero-value validation absence) |
| PROJ-01 | Phase 5 | Pending |
| PROJ-02 | Phase 5 | Pending |
| PROJ-03 | Phase 5 | In progress (backend: UpdateProjectRequest + Update service/repo/mock) |
| PROJ-04 | Phase 5 | In progress (backend: Delete service/repo/mock + active entries check) |
| PROJ-05 | Phase 5 | Pending |
| PROJ-06 | Phase 5 | Pending |
| TIME-01 | Phase 6 | Pending |
| TIME-02 | Phase 6 | Pending |
| TIME-03 | Phase 6 | Pending |
| TIME-04 | Phase 6 | Pending |
| TIME-05 | Phase 6 | Pending |
| TIME-06 | Phase 6 | Pending |
| TIME-07 | Phase 6 | Pending |
| TIME-08 | Phase 6 | Pending |
| EXPN-01 | Phase 6 | Pending |
| EXPN-02 | Phase 6 | Pending |
| EXPN-03 | Phase 6 | Pending |
| EXPN-04 | Phase 6 | Pending |
| EXPN-05 | Phase 6 | Pending |
| EXPN-06 | Phase 6 | Pending |
| APPR-01 | Phase 6 | Pending |
| APPR-02 | Phase 6 | Pending |
| APPR-03 | Phase 6 | Pending |
| APPR-04 | Phase 6 | Pending |
| APPR-05 | Phase 6 | Pending |
| EXPT-01 | Phase 7 | Pending |
| EXPT-02 | Phase 7 | Pending |
| EXPT-03 | Phase 7 | Pending |
| EXPT-04 | Phase 7 | Pending |
| EXPT-05 | Phase 7 | Pending |
| EXPT-06 | Phase 7 | Pending |

**Coverage:**

- v0.1 requirements: 55 total
- Mapped to phases: 55
- Unmapped: 0 ✓

---

*Requirements defined: 2026-06-08*
*Last updated: 2026-06-08 after v0.1 milestone continuation*
