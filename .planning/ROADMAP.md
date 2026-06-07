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

**Status:** Not started

**Goal:** Port all 16+ SurrealDB repositories to `internal/adapters/secondary/postgres/`. Hand-written SQL with `pgx`, proper JOINs replacing nested subqueries. Keep hexagonal boundary intact — same domain models, same service layer.

**Depends on:** 0, Pg-1

### Repositories to port
- auth/user, organization, organization_membership
- unit, unit_member
- project, subproject, contract, customer
- working_group, wg_member
- time_entry, expense, audit_log
- invitation, password_reset, refresh_token, export

### Key technical choices
- `github.com/jackc/pgx/v5/pgxpool` for connection pool
- `uuid.UUID` everywhere with `pgtype.UUID` scanning
- Hand-written SQL with `CollectRows`, `GetFieldFromRow` (no ORM)
- Proper SQL JOINs instead of SurrealDB nested subqueries

---

## Pg-3: Wiring, cleanup & verification

**Status:** Not started

**Goal:** Wire all postgres repos in server init, remove SurrealDB entirely, verify everything works end-to-end.

**Depends on:** Pg-2

### Key deliverables
- Wire postgres repos in `cmd/server/main.go`
- Delete `internal/adapters/secondary/surrealdb/`
- Delete `internal/db/surreal.go`
- Delete `cmd/schema/` and `schema/` directory
- Remove SurrealDB from docker-compose
- Update Makefile (remove schema targets, add `make setup` for migrate+seed)
- Update AGENTS.md, env vars, docs
- Full manual verification: fresh bootstrap → seed → login → CRUD every entity

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