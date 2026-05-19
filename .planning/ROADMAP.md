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

## Phase 1: Org hierarchy edge-driven

**Status:** Not started

**Goal:** Define organization hierarchy with edge-driven relationships

**Depends on:** 0

### Plans

- [ ] 01-CONTEXT.md
- [ ] 01-DISCUSSION-LOG.md

---

## Phase 2: Customers management page

**Goal:** Create a new customers page with full CRUD operations. Customers can be internal (organization itself) or external.

**Depends on:** 0, 1

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

**Depends on:** 0, 1

### Success Criteria

- [x] Projects list displayed on contract detail page
- [x] Projects filterable by contract (via contract_id filter)
- [x] Edge case: contract with no projects handled
- [ ] Edge case: contract with adopted projects handled

---

## Phase 4: Integrate customers into contracts

**Goal:** Update contract creation dialog to include customer selection. Add "internal customer" option.

**Depends on:** 0, 2

### Success Criteria

- [ ] Customer dropdown in create contract dialog
- [ ] "Internal customer" option available
- [ ] Customer required for new contracts
- [ ] Existing contracts without customer still work

---

## Phase 5: MVP Consolidation

**Status:** Context gathered

**Goal:** Consolidate the project structure and create initial MVP demo seeding. Replace existing seed data with a clean, idempotent `003_seed_demo.surql` that populates all core entities (org, users, units, memberships, projects, contracts, customer, time entries, expenses) so the app is immediately demonstrable.

**Depends on:** None (can run in parallel with Phase 0)

### Canonical refs:
- `schema/001_schema.surql` — Database schema for all entities
- `schema/002_seed_tcg.surql` — Old seed (will be deprecated)
- `cmd/schema/main.go` — Schema/seed loader

### Success Criteria

- [ ] `003_seed_demo.surql` created with 6 users (2 managers + 1 finance + 3 employees), 8 units, 3 contracts, 6 projects, 1 customer
- [ ] Old `002_seed_tcg.surql` deprecated (renamed to `002_seed_tcg.deprecated.surql`)
- [ ] Seed data is idempotent — safe to re-run without errors
- [ ] All-UUID consistent ID format throughout
- [ ] Demo credentials documented (pre-hashed passwords)
- [ ] Sample time entries (3-5 per employee) and expenses (1-2 per employee) seeded
- [ ] Manual verification pass: run schema, start server, log in as demo manager, all pages render with data
- [ ] Bootstrap flow kept separate from seeding