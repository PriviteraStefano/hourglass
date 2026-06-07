# Hourglass Roadmap

## Current Milestone: MVP Consolidation

## Phase 0: Testing foundation

**Status:** Ready to execute

**Goal:** Backend & frontend testing foundation — catch bugs before building new features. Establish test infrastructure, patterns, and baseline coverage.

**Depends on:** None (prerequisite for all future phases)

### Plans (9 plans, 4 waves)

- [x] 00-01-PLAN.md — Quick-scan probe, testify dep, testdata package, BUGS.md (Wave 1)
- [x] 00-02-PLAN.md — Vitest + MSW + RTL infra setup (Wave 1)
- [x] 00-03-PLAN.md — Service tests: auth, org, time-entry (state machines), model validation (Wave 2)
- [x] 00-04-PLAN.md — Service tests: contract, customer, project, unit, working-group, invitation, password-reset, export (Wave 2)
- [x] 00-05-PLAN.md — Handler integration tests: all 10 domains (Wave 3)
- [x] 00-06-PLAN.md — Repository tests: all untested repos (Wave 3)
- [x] 00-07-PLAN.md — Frontend Vitest: API client, hooks, validation (Wave 2)
- [x] 00-08-PLAN.md — Playwright E2E: all CRUD flows (Wave 1)
- [x] 00-09-PLAN.md — Batch-fix all bugs from BUGS.md (Wave 4)

---

## Pg-1: Foundation — PostgreSQL schema, pool, docker-compose

**Status:** Planned — 3 plans in 2 waves

**Goal:** Create PostgreSQL schema covering all entities, `pgxpool` connection management, migrate CLI update, demo seed as SQL migration, docker-compose with Postgres as default. SurrealDB moved to optional profile.

**Depends on:** 0

### Plans (3 plans in 2 waves)

| Plan | Wave | Objective | Tasks |
|------|------|-----------|-------|
| pg-1-01 | 1 | Schema migrations — DDL (002_full_schema) + seed (003_seed) | 2 |
| pg-1-02 | 1 | pgxpool singleton + port migrate CLI to pgx + -all flag | 2 |
| pg-1-03 | 2 | Docker compose (Postgres default) + server wiring + docs | 3 |

---

## Pg-2: PostgreSQL adapters

**Status:** Complete — 6/6 plans executed

**Goal:** Port all 16+ SurrealDB repositories to `internal/adapters/secondary/postgres/`. Hand-written SQL with `pgx`, proper JOINs replacing nested subqueries. Keep hexagonal boundary intact — same domain models, same service layer.

**Depends on:** 0, Pg-1

### Deliverables
- 18 PostgreSQL repository files in `internal/adapters/secondary/postgres/`
- 2 new port interfaces: `ExpenseRepository`, `SubprojectRepository`
- Shared sentinel errors in `internal/core/ports/errors.go` (ErrNotFound, ErrConflict, ErrForeignKey)
- `users.name` column removed from migration schema
- 18 repository test files exercising every method against live PostgreSQL

### Repositories to port (includes 2 new ones)
- user_repository, user_finder
- organization_repo, organization_membership_repo, organization_management_repo
- unit_repository, unit_member_repository
- working_group_repository, wg_member_repository
- project_repository, subproject_repository (NEW)
- contract_repository, customer_repository
- time_entry_repository, audit_log_repository, expense_repository (NEW), export_repository
- invitation_repository, password_reset_repository, refresh_token_repo

### Key technical choices
- `github.com/jackc/pgx/v5/pgxpool` for connection pool
- Hand-written SQL with pgx (no ORM, no query builder)
- `pgxpool.QueryRow` + `Scan` for single-row, `pool.Query` + `rows.Next()` for multi-row
- `pgx.Batch` / `pool.Begin` for transactional writes
- SurrealDB nested subqueries → proper SQL JOINs (D-06)
- Aggregate fields → SQL aggregate functions (D-07)
- `wrapPGError` for translating pgx/pgconn errors to domain sentinel errors
- String↔UUID conversion at adapter boundary for unit/unit_member IDs

### Plans (6 plans in 2 waves)

| Plan | Wave | Objective | Tasks |
|------|------|-----------|-------|
| pg-2-01 | 1 | Foundation — schema change (remove users.name), sentinel errors, new port interfaces, postgres.go helpers, exported_test_helpers.go, UserFinder + tests | 2 |
| pg-2-02 | 2 | UserRepository + OrganizationRepository + OrganizationMembershipRepository + OrganizationManagementRepository + tests | 2 |
| pg-2-03 | 2 | UnitRepository + UnitMemberRepository + WorkingGroupRepository + WGMemberRepository + InvitationRepository + tests | 3 |
| pg-2-04 | 2 | ProjectRepository + SubprojectRepository + ContractRepository + CustomerRepository + tests | 3 |
| pg-2-05 | 2 | RefreshTokenRepository + PasswordResetRepository + tests | 2 |
| pg-2-06 | 2 | TimeEntryRepository (dynamic WHERE) + AuditLogRepository + ExpenseRepository + ExportRepository (4-level JOINs) + tests | 3 |

---

## Pg-3: Wiring, cleanup & verification

**Status:** Planned — 3 plans in 2 waves

**Goal:** Wire all postgres repos in server init, remove SurrealDB entirely, verify everything works end-to-end.

**Depends on:** Pg-2

### Plans (3 plans in 2 waves)

| Plan | Wave | Objective | Tasks |
|------|------|-----------|-------|
| pg-3-01 | 1 | Core wiring — adapters (TokenService, PasswordHasher), CORS middleware, main.go rewrite, auth_test.go rewrite | 3 |
| pg-3-02 | 2 | Cleanup — delete SurrealDB files, update docker-compose/Makefile/AGENTS.md/go.mod | 2 |
| pg-3-03 | 2 | Verification — smoke test (main_test.go) + manual D-15 flow checkpoint | 2 (1 checkpoint) |

---

## Phase 1: Org hierarchy edge-driven

**Status:** Not started

**Goal:** Implement edge-driven reparenting via ReactFlow's onConnect API. Replace node-drag→DOM-detection.

**Depends on:** 0, Pg-3

### Plans

- [ ] 01-CONTEXT.md
- [ ] 01-DISCUSSION-LOG.md

---

## Phase 2: Customers management page

**Goal:** Create a new customers page with full CRUD operations. Customers can be internal (organization itself) or external.

**Depends on:** 0, Pg-3

### Success Criteria

- [x] Customers list page at /customers
- [x] Create new customer (name, contact, email, phone, VAT, address)
- [x] Edit existing customer
- [x] Delete customer (handle contracts assignment)
- [x] Edge case: cannot delete customer with active contracts (409 from API)
- [ ] Edge case: "internal customer" represented specially

---

## Phase 3: Contracts - add projects display

**Goal:** Add projects list display to contract detail page. Show related projects per contract.

**Depends on:** 0, Pg-3

### Success Criteria

- [x] Projects list displayed on contract detail page
- [x] Projects filterable by contract (via contract_id filter)
- [x] Edge case: contract with no projects handled
- [ ] Edge case: contract with adopted projects handled

---

## Phase 4: Integrate customers into contracts

**Goal:** Update contract creation dialog to include customer selection. Add "internal customer" option.

**Depends on:** 0, Pg-3

### Success Criteria

- [ ] Customer dropdown in create contract dialog
- [ ] "Internal customer" option available
- [ ] Customer required for new contracts
- [ ] Existing contracts without customer still work

---

## Phase 5: MVP Consolidation

**Status:** Planned — 3 plans in 3 waves

**Goal:** Consolidate the project structure and create initial MVP demo seeding. Seed data as SQL migration (`003_seed.up.sql`) populating all core entities (org, users, units, memberships, projects, contracts, customer, time entries, expenses). Now targets PostgreSQL instead of SurrealDB.

**Depends on:** Pg-1 (seed is part of the migration)

### Canonical refs:
- `migrations/002_full_schema.up.sql` — PostgreSQL schema
- `migrations/003_seed.up.sql` — MVP demo seed
- `internal/db/pgpool.go` — Connection pool

### Success Criteria

- [ ] `003_seed.up.sql` created with 6 users (2 managers + 1 finance + 3 employees), 8 units, 3 contracts, 6 projects, 1 customer
- [ ] Seed data is idempotent — safe to re-run without errors
- [ ] All-UUID consistent ID format throughout
- [ ] Demo credentials documented (pre-hashed passwords)
- [ ] Sample time entries (3-5 per employee) and expenses (1-2 per employee) seeded
- [ ] Manual verification pass: run migrate, start server, log in as demo manager, all pages render with data
- [ ] Bootstrap flow kept separate from seeding

### Plans (3 plans in 3 waves)

| Plan | Wave | Objective | Tasks |
|------|------|-----------|-------|
| 05-01 | 1 | Foundation seed — org, 6 users, 8 units, customer, memberships, 3 contracts, 6 projects, subprojects, WGs, WG members | 3 |
| 05-02 | 2 | Demo entries — 9-15 time entries + 3-6 expenses | 1 |
| 05-03 | 3 | Manual verification — migrate, server, login as all roles, page checks | 1 (checkpoint) |

---

## Phase 6: API Audit

**Status:** Planned — 3 plans in 3 waves

**Goal:** System-wide API audit — exercise every REST endpoint against a live server pre-loaded with demo seed data, verify HTTP correctness, response shapes, auth flows, and CORS. Targets PostgreSQL server (instead of SurrealDB).

**Depends on:** 0, Pg-3

### Canonical refs:
- `migrations/003_seed.up.sql` — MVP demo seed data
- `internal/adapters/primary/http/*.go` — All 11 handler files (endpoint surface to audit)
- `pkg/api/response.go` — Response envelope format
- `cmd/server/main.go` — Route registrations, middleware wiring

### Success Criteria

- [ ] Wave 1: Core domain audit (auth, units, contracts, projects, customers, time-entries, expenses)
- [ ] Wave 2: Auxiliary domain audit (working-groups, invitations, password-reset, exports, org management)
- [ ] Wave 3: Batch-fix all discovered bugs from 06-BUGS.md, full re-audit
- [ ] `internal/audit/` package created with `//go:build audit` tag
- [ ] `make audit` target manages full lifecycle
- [ ] All 6 seed users login-verified
- [ ] Full cookie flow tested (login → refresh → logout)
- [ ] CORS preflight + headers verified
- [ ] Error envelope `{ error: ... }` verified for all 4xx/5xx responses