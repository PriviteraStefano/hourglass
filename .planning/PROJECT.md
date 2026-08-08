# Hourglass

## What This Is

Hourglass is a time entry and expense tracking system with approval workflows for organizations. Employees log time and expenses, managers review and approve, finance teams manage financial cutoffs and reporting. Built with Go (hexagonal architecture) + PostgreSQL + React 19.

## Core Value

Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities. Approval routing resolves through the activity → working-group → manager/delegate chain, enforceable at approval time.

## Current State

**Shipped:** v0.1 MVP Consolidation (2026-08-01) — 16 phases, 62 plans, 138 tasks + **Phase 11 v0.2 foundations** (2026-08-07) — 8 plans: origins + sold_hours + tickets backend + **Phase 12 coverage backend** (2026-08-08) — 7 plans: allocation ledger + **Phase 13 direction backend** (2026-08-08) — 10 plans: the plan plane

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
- ✓ Origins on activities — type + reference set (manager-assignment / employee-proposal / customer-ticket), proposal approval via activity routing (FND-01..04) — Validated in Phase 11: Foundations — Schema + Origins + Tickets backend
- ✓ Tickets — first-class, internal-only: lifecycle + triage + reopen, kinds question/bug/change/evolution, ticket→activity→entries chain, concurrency-safe dismissal guard + server-rendered hours note (TICK-01..05) — Validated in Phase 11: Foundations — Schema + Origins + Tickets backend

### Active

<!-- Current scope: v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction. Detailed REQ-IDs in REQUIREMENTS.md. -->

- [ ] Coverage — allocation ledger per entry, funding sources (contract budget / support bucket / service request / internal absorption / cross-project transfer), to-cover queue, monthly rhythm, snapshot-not-lock (COV-01..05)
- [x] Direction — the plan plane: scheduled/queued modes, claim model, lifecycle, org policy, direction-coverage read-model (DIR-01..06) — Validated in Phase 13: Direction Backend — The Plan Plane
- [ ] Availability — employee absences UI + capacity views per activity/WG (AVAIL-01..05)
- [ ] Surfaces — prototype-driven: allocation screen + to-cover queue, buckets + per-unit report, Today both shapes, direction scheduler (SURF-01..08)
- [ ] Per-page UX polish — one phase per page, gsd-sketch-driven, folding v0.1 UAT debt (POLS-01..11)
- [ ] UX foundation — design tokens + shared components + sketch loop contract (UXFD-01..02)

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

- SurrealDB — fully replaced by PostgreSQL, no longer supported
- Real-time features — high complexity, not core to v0.1
- Mobile apps — web-first, not planned for v0.1
- OAuth/Social login — email/password sufficient for v0.1
- CI/CD pipeline — ad-hoc for now, will add later
- External ticket intake (helpdesk port) — future hexagonal secondary adapter, not v0.2 (D-E)
- Expense coverage allocations — schema-ready only (polymorphic entry D-K), `time` only in v0.2; revisit in a later milestone if expense-splitting need is demonstrated
- Smart/auto allocation proposals — blocked by P-005 data-maturity gate
- Estimate-accuracy analytics (V5) — v0.2 stores the raw material (sold_hours + actuals); mining is V5
- Warranty certification flow — warranty is declared at allocation time; the warranty-cost report is the control (D-H/D-C)
- Full budget machinery — rates, money, per-activity estimates (ADR-P-010, V4); only `sold_hours` on contracts lands in v0.2 (D-N)
- Customer-facing ticket portal — tickets are internal-only (D-E)
- SLA engine / escalation chains / email ingestion / KB — anti-features for v0.2, ITSM creep
- Plan-adherence per-day-per-person metrics — aggregate-only per-period (D-U), never a surveillance number
- Per-customer request counts for billing — billing story superseded by coverage allocations; request counts deferred
- Kanban board — tickets are demand tracking, not task execution (no kanban, no sub-task trees, no comment threads)

## Context

Hourglass was originally built on SurrealDB; v0.1 fully ported to PostgreSQL (Phases Pg-1/Pg-2/Pg-3, ~7,300 lines deleted) and rebuilt the test infrastructure on testcontainers-go. The big-bang activity ontology migration (Phase 9) replaced the projects/subprojects model with a recursive activities tree; all approval routing was rewritten onto the activity chain. v0.1 shipped the Information Architecture phase (Phase 10) with the demo deployment topology documented (Compose + Caddy + cloudflared, ADR-BE-015).

Known debt at close: 25 pending UAT scenarios, 3 human verification reviews, 2 quick tasks with unknown status — deferred to next milestone (see STATE.md).

**v0.2 ontology research round (2026-08-01/02):** a domain walkthrough surfaced two gaps in the accepted activity ontology (ADR-P-007): *origins* (where demand came from) and *coverage* (who pays). The research extended the model to three orthogonal planes — **direction** (the plan, mutable, manager/self-owned) → **facts** (time entries, immutable after approval) → **coverage** (the money label, mutable, snapshot-protected) — with the cardinal principle "the plan/decision never rewrites the fact". All decisions D-A…D-AA are closed; the vault note (`hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md`) is the record of truth. Formal ADRs (P-003 rev, P-013, P-014, P-015) are deferred and written as phases land; ADR-P-012 (coverage ledger) is drafted (Proposed) in the vault. No legacy-data migration is needed — Hourglass has never been deployed (P-007 D-6 big-bang landed pre-deploy).

## Constraints

- **[Tech stack]**: Go 1.26.1, PostgreSQL (pgx/v5), React 19, TanStack Router + Query, shadcn/ui + Tailwind — no SurrealDB
- **[Database]**: PostgreSQL via pgxpool, all repos exist, hand-written SQL, no ORM; migrations append-only per ADR-BE-004
- **[Auth]**: JWT in HttpOnly cookies (auth_token/refresh_token), bcrypt password hashing, strict reuse model with family revocation
- **[Architecture]**: Hexagonal — services in `internal/core/services/*`, HTTP adapters in `internal/adapters/primary/http/*`, PostgreSQL adapters in `internal/adapters/secondary/postgres/*`
- **[Domain]**: Captured effort is a fact, coverage is a decision, direction is a plan — the decision/plan never rewrites the fact (Σ allocations = entry hours; deviations are data, not violations)
- **[UI]**: UI-last in v0.2 — all new surfaces are prototype-driven (gsd-sketch) against the complete backend; D-O IA leans validated in prototypes before any P-011 revision

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
| Three-plane ontology (direction → facts → coverage) | Origins + coverage gaps surfaced in domain walkthrough; "the decision never rewrites the fact" keeps V5 analytics honest; direction completes the timeline (D-Q) | ✓ Good |
| Coverage allocation ledger (P-012) | 4+4 split applied by editing entries would corrupt actual-effort truth; allocations are money-labeling with Σ invariant + to-cover queue | ✓ Good |
| Tickets first-class, internal-only (P-003 rev) | Demand tracking (not task execution); ticket→activity→entries chain preserves single-FK capture (P-007 D-4); external intake is a future port | ✓ Good |
| Direction in v0.2 ontology, build staged (P-015) | Retrofit after v0.2 would destroy drafted ADRs; additive pre-deploy big-bang, same logic as P-007 D-6 | ✓ Good |
| Contracts carry `sold_hours` (D-N) | "Sold 4h, took 8h → next time sell 16" needs explicit sold figure; richer budget machinery stays at V4 (P-010) | — Pending |
| ADRs deferred to phase landing | Pre-deploy, the only ADR consumer is the build process itself (Stefano's call) | ✓ Good |
| Coverage allocations editable indefinitely (D-F) | Cutoff is a reporting snapshot, not a lock; realism over enforcement | ✓ Good |

## Current Milestone: v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction

**Goal:** Extend the activity ontology into the three-plane model (direction → facts → coverage): tickets as the second capture layer with triage, origins on activities, coverage allocations with funding sources, and the direction plan plane — then surface them prototype-driven, and finish with per-page polish folding v0.1 UAT debt.

**Target features:**
- Tickets: first-class, internal-only, lifecycle + triage + reopen, kinds question/bug/change/evolution, ticket→activity→entries chain, dismissal guard
- Origins: activity carries origin type + reference set (assigned_by/assigned_to · proposed_by/reviewed_by · ticket_id), proposal approval via activity routing
- Coverage: per-entry allocation ledger, funding sources (contract budget, support bucket, zero-value service request, internal absorption, cross-project transfer), to-cover queue, monthly rhythm, one-step manager confirm, snapshot-not-lock
- Direction: scheduled/queued modes, claim model, lifecycle draft→active→superseded/cancelled, org policy, P-008 absence warnings, direction-coverage read-model
- Availability: absences + capacity views (kept from original v0.2 scope)
- Surfaces: prototype-driven — allocation screen + to-cover queue + own-coverage, buckets + per-unit non-billed report, Today both shapes, direction scheduler
- Trailing: 7 per-page UX polish phases folding v0.1 UAT/verification debt

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

*Last updated: 2026-08-08 after Phase 13 completion (Direction backend — the plan plane)*
