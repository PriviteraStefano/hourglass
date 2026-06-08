# Hourglass

## What This Is

Hourglass is a time entry and expense tracking system with approval workflows for organizations. Employees log time and expenses, managers review and approve, finance teams manage financial cutoffs and reporting. Built with Go (hexagonal architecture) + PostgreSQL + React 19.

## Core Value

Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/project management, and export capabilities.

## Current Milestone: v0.1 MVP Consolidation

**Goal:** Ship a working MVP on PostgreSQL — test infrastructure reboot + all core feature CRUD pages.

**Target features:**
- Testing foundation reboot — testcontainers-go for service-layer integration tests, known auth bugs fixed, full test coverage against real PostgreSQL
- Authorization — fix broken auth endpoints
- Org hierarchy — org tree with ReactFlow, unit CRUD, member management
- Customers — full customer CRUD with search/filter
- Contracts — contract CRUD with customer dropdown
- Projects — project CRUD with subprojects
- Time entries + Expenses — full CRUD + approval workflow
- Exports — CSV/Excel downloads

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ Full PostgreSQL schema with 24 tables, FK constraints, CHECK constraints, JSONB, UUID PKs — Pg-1/Pg-2/Pg-3
- ✓ 18 PostgreSQL repository implementations with integration tests — Pg-2
- ✓ Shared sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrForeignKey`) with `wrapPGError` — Pg-2
- ✓ Recursive CTE for hierarchical unit queries — Pg-2
- ✓ CORS middleware extracted to `internal/middleware/` — Pg-3
- ✓ Automated smoke test — Pg-3
- ✓ Server compiles and runs with zero SurrealDB imports — Pg-3
- ✓ ~7,300 lines SurrealDB code deleted — Pg-3
- ✓ MVP demo seed data (6 users, 3 contracts, 6 projects, 12 time entries, 6 expenses) — Phase 5
- ✓ Testify, shared testdata factories, service tests (auth/org/time-entry/contract/project/customer/unit/WG/invitation/password-reset/export), handler integration tests, repository tests, frontend Vitest — Phase 0 (SurrealDB era, needs PG reboot)
- ✓ Playwright E2E tests for all CRUD flows — Phase 0

### Active

<!-- Current scope. Building toward these. -->

- [ ] Testcontainers-go for PostgreSQL-backed service-layer integration tests
- [ ] Fix known auth bugs (`/auth/memberships` panic, `/auth/me` empty role/org_id, `/units/{id}/members` 500, `/organizations/members` 500)
- [ ] Branch bug-fix from BUGS.md (Wave 4)
- [ ] Auth endpoint fixes verified with seed users
- [ ] Org hierarchy ReactFlow frontend
- [ ] Customer CRUD frontend
- [ ] Contract CRUD frontend with customer dropdown
- [ ] Project CRUD frontend with subprojects
- [ ] Time entry + expense frontend with approval workflow
- [ ] CSV/Excel export frontend

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- SurrealDB — fully replaced by PostgreSQL, no longer supported
- Real-time features — not part of v0.1 scope
- Mobile apps — web-first, not planned for v0.1

## Context

Hourglass was originally built on SurrealDB. All database code was ported to PostgreSQL in v0.1 (Phases Pg-1/Pg-2/Pg-3). The previous Phase 0 testing effort targeted SurrealDB infrastructure — those tests need to be rewritten to target PostgreSQL using testcontainers-go. All backend endpoints already exist; the primary gap is frontend CRUD pages plus fixing known auth bugs.

## Constraints

- **[Tech stack]**: Go 1.26.1, PostgreSQL (pgx/v5), React 19, TanStack Router + Query, shadcn/ui + Tailwind — no SurrealDB
- **[Database]**: PostgreSQL via pgxpool, all repos exist, hand-written SQL, no ORM
- **[Auth]**: JWT in HttpOnly cookies (auth_token/refresh_token), bcrypt password hashing
- **[Architecture]**: Hexagonal — services in `internal/core/services/*`, HTTP adapters in `internal/adapters/primary/http/*`, PostgreSQL adapters in `internal/adapters/secondary/postgres/*`

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| PostgreSQL over SurrealDB | Better ecosystem, proper SQL JOINs, production-grade pgx | ✓ Good |
| Hexagonal architecture preserved | Only adapter layer changed; services/domain untouched | ✓ Good |
| Hand-written SQL with pgx (no ORM) | Full query control, pgx is Go's gold standard PG driver | ✓ Good |
| Testcontainers-go for integration tests | Isolated PostgreSQL per test run, no external DB dependency | — Pending |
| UUID PKs with `gen_random_uuid()` | Consistent ID format, no sequential PK exposure | ✓ Good |

---

*Last updated: 2026-06-08 after v0.1 milestone continuation*
