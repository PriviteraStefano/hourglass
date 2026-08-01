# Hourglass

## What This Is

Hourglass is a time entry and expense tracking system with approval workflows for organizations. Employees log time and expenses, managers review and approve, finance teams manage financial cutoffs and reporting. Built with Go (hexagonal architecture) + PostgreSQL + React 19.

## Core Value

Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities. Approval routing resolves through the activity → working-group → manager/delegate chain, enforceable at approval time.

## Current State

**Shipped:** v0.1 MVP Consolidation (2026-08-01) — 16 phases, 62 plans, 138 tasks

- Full PostgreSQL stack: 24-table schema, 18 repos, testcontainers-go integration tests, ~30 Go package suites green
- Auth: JWT HttpOnly cookies, refresh-token rotation with reuse detection + family revocation, rate limiting, password reset hardening
- Org hierarchy: ReactFlow tree, unit CRUD, member management, edge-driven reparenting, delete protection
- Customers (incl. internal customers), Contracts, Exports (CSV/XLSX)
- **Activity ontology** (Phase 9): recursive `activities` table replaces projects/subprojects; commercial context + billability resolved via CTE; approval routing via activity → WG → manager/delegate
- **Information architecture** (Phase 10): pillar sidebar (Today/Track/Work/People/Economics/Review/Reports/Admin), role-scoped visibility, Today landing, `/approvals` queue, Working Groups surface
- Frontend: React 19, TanStack Router + Query v5, shadcn/ui, Tailwind; Playwright e2e + Vitest suites green

## Requirements

### Validated

<!-- Shipped and confirmed valuable. Full 55-item traceability: .planning/milestones/v0.1-REQUIREMENTS.md -->

- ✓ 60/60 v0.1 requirements shipped (TEST-01..06, AUTH-01..05, ORG-01..05, CUST-01..06, CTRT-01..07, PROJ-01..06, TIME-01..08, EXPN-01..06, APPR-01..05, EXPT-01..06)
- ✓ Projects/subprojects → recursive Activity model (PROJ-* superseded; intent delivered via `/activities` surface) — v0.1
- ✓ Approval workflow: two-stage (manager → finance), immutable history, reason-required reject, self-approval prevention — v0.1
- ✓ Refresh-token reuse detection with family revocation (P0-5) — v0.1
- ✓ Filterable list views, `/customers` index, route error boundaries (P0-2/3/4) — v0.1
- ✓ Working Groups surface (list/search/create/edit/members) — v0.1
- ✓ Today landing + Approvals queue (stage-filtered Manager/Finance) — v0.1

### Active

<!-- Current scope: v0.2 UX Polish + Tickets + Availability. Detailed REQ-IDs in REQUIREMENTS.md. -->

- [ ] UX/UI pass across all pages — one phase per page, gsd-sketch-driven (options → agree → implement → verify)
- [ ] Ticket ontology — unified ticket entity: internal work tasks + external helpdesk (incl. internal customers), request tracking per customer for billing
- [ ] Availability — employee absences UI + resource/capacity views per activity/WG
- [ ] Fold in v0.1 verification debt (25 UAT scenarios + 3 human verification reviews) per-page

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- SurrealDB — fully replaced by PostgreSQL, no longer supported
- Real-time features — high complexity, not core to v0.1
- Mobile apps — web-first, not planned for v0.1
- OAuth/Social login — email/password sufficient for v0.1
- CI/CD pipeline — ad-hoc for now, will add later

## Context

Hourglass was originally built on SurrealDB; v0.1 fully ported to PostgreSQL (Phases Pg-1/Pg-2/Pg-3, ~7,300 lines deleted) and rebuilt the test infrastructure on testcontainers-go. The big-bang activity ontology migration (Phase 9) replaced the projects/subprojects model with a recursive activities tree; all approval routing was rewritten onto the activity chain. v0.1 shipped the Information Architecture phase (Phase 10) with the demo deployment topology documented (Compose + Caddy + cloudflared, ADR-BE-015).

Known debt at close: 25 pending UAT scenarios, 3 human verification reviews, 2 quick tasks with unknown status — deferred to next milestone (see STATE.md).

## Constraints

- **[Tech stack]**: Go 1.26.1, PostgreSQL (pgx/v5), React 19, TanStack Router + Query, shadcn/ui + Tailwind — no SurrealDB
- **[Database]**: PostgreSQL via pgxpool, all repos exist, hand-written SQL, no ORM; migrations append-only per ADR-BE-004
- **[Auth]**: JWT in HttpOnly cookies (auth_token/refresh_token), bcrypt password hashing, strict reuse model with family revocation
- **[Architecture]**: Hexagonal — services in `internal/core/services/*`, HTTP adapters in `internal/adapters/primary/http/*`, PostgreSQL adapters in `internal/adapters/secondary/postgres/*`

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| PostgreSQL over SurrealDB | Better ecosystem, proper SQL JOINs, production-grade pgx | ✓ Good |
| Hexagonal architecture preserved | Only adapter layer changed; services/domain untouched | ✓ Good |
| Hand-written SQL with pgx (no ORM) | Full query control, pgx is Go's gold standard PG driver | ✓ Good |
| Testcontainers-go for integration tests | Isolated PostgreSQL per test run, no external DB dependency | ✓ Good |
| UUID PKs with `gen_random_uuid()` | Consistent ID format, no sequential PK exposure | ✓ Good |
| Activity ontology (projects/subprojects → recursive activities) | Single recursive entity; commercial context/billability derived via CTE; enables WG-anchored approval routing | ✓ Good |
| Approval routing via activity → WG → manager/delegate | Enforceable at approval time; unit-manager fallback for personal activities; D-11 skip incl. delegates | ✓ Good |
| Refresh-token strict reuse model | Concurrent-refresh loser is indistinguishable from attacker replay → family revokes | ✓ Good |
| Pillar-based IA (ADR-P-011) | Role-scoped visibility from pure, testable predicates; single declarative navStructure | ✓ Good |
| Ticket ontology (v0.2) | Unified ticket entity (kinds: task/helpdesk) scoped under the activity tree; request counting per customer enables billing | — Pending |

## Current Milestone: v0.2 UX Polish + Tickets + Availability

**Goal:** Polish the product page by page through sketch-driven UX/UI work, add a ticket ontology (internal tasks + customer helpdesk), and surface availability (absences + resource views).

**Target features:**
- UX/UI pass: one phase per page — gsd-sketch options → agree → implement → verify; all pillars step by step
- Ticket ontology: internal work tasks + external helpdesk (incl. internal customers); request tracking per customer for billing
- Availability: employee absences + resource/capacity views per activity/WG

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---

*Last updated: 2026-08-01 after v0.2 milestone started*
