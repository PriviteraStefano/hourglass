# Hourglass

## What This Is

Hourglass is a time entry and expense tracking system with approval workflows for organizations. Employees log time and expenses, managers review and approve, finance teams manage financial cutoffs and reporting. Built with Go (hexagonal architecture) + PostgreSQL + React 19.

## Core Value

Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities. Approval routing resolves through the activity → working-group → manager/delegate chain, enforceable at approval time.

## Current State

**Shipped:** v0.1 MVP Consolidation (2026-08-01) — 16 phases, 62 plans, 138 tasks + **v0.2 Ontology Extension** (2026-08-25) — 6 phases, 41 plans, 108 tasks: origins + tickets, coverage ledger, direction plane, availability backend, UX foundation, integrity repair. Route-oriented Phases 17–26 cancelled unbuilt.

- Full PostgreSQL stack: append-only migrations through 025, testcontainers-go integration tests, Go package suites green
- Auth: JWT HttpOnly cookies, refresh-token rotation with reuse detection + family revocation, rate limiting, password reset hardening; Phase 16 closed two limiter defects (tier-limit inflation, proxy-aware anonymous buckets)
- Org hierarchy: ReactFlow tree, unit CRUD, member management, edge-driven reparenting, delete protection
- Customers (incl. internal customers), Contracts (incl. `sold_hours`), Exports (CSV/XLSX)
- **Activity ontology** (Phase 9): recursive `activities` table; commercial context + billability via CTE; approval routing via activity → WG → manager/delegate
- **Three-plane ontology** (v0.2): direction (plan) → facts (entries) → coverage (money label). Origins on activities; tickets first-class internal-only; coverage allocation ledger with Σ invariant; direction per-day rows + claim model
- **Availability backend** (Phase 14): absence declare/confirm/reject, HR medical curation, capacity queries, work-schedule model
- **Information architecture** (Phase 10): pillar sidebar (Today/Track/Work/People/Economics/Review/Reports/Admin), role-scoped visibility, Today landing, `/approvals` queue, Working Groups surface
- **UX foundation** (Phase 15): semantic status tokens, frozen shared components (PageHeader, FilterBar, DataTable v9, StatusBadge, EmptyState, ConfirmDialog), sketch-loop contract (stale applies-to 16–26 — amend in v0.2.1 if still ambiguous)
- Frontend: React 19, TanStack Router + Query v5, shadcn/ui, Tailwind; Playwright e2e + Vitest suites
- Integrity repair (Phase 16): employee own-coverage read, expense `unit_id` persistence, receipt-upload authorization, WR-05 capacity org-isolation

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
- ✓ Origins on activities — type + reference set (manager-assignment / employee-proposal / customer-ticket), proposal approval via activity routing (FND-01..04) — v0.2 Phase 11
- ✓ Tickets — first-class, internal-only: lifecycle + triage + reopen, kinds question/bug/change/evolution, ticket→activity→entries chain, concurrency-safe dismissal guard + server-rendered hours note (TICK-01..05) — v0.2 Phase 11
- ✓ Coverage — allocation ledger per entry, five funding sources, to-cover queue, proposals-on-read, one-step manager confirm, snapshot-not-lock (COV-01..05) — v0.2 Phase 12
- ✓ Direction — the plan plane: scheduled/queued modes, claim model, lifecycle, org policy, direction-coverage read-model (DIR-01..06) — v0.2 Phase 13
- ✓ Availability backend — absence declare/confirm/reject lifecycle, HR medical curation, capacity queries over availability_windows (AVAIL-01..02) — v0.2 Phase 14
- ✓ UX foundation tokens + frozen shared components (UXFD-01) — v0.2 Phase 15
- ✓ Integrity repair — own-coverage read, expense `unit_id`, receipt auth, WR-05, rate-limiter defects — v0.2 Phase 16

### Active

<!-- Next milestone: v0.2.1 contract-first presentation. Fresh REQUIREMENTS.md is created by /gsd-new-milestone. Do not copy v0.2 SURF/POLS/route requirements blindly. -->

- [ ] Design-language contract
- [ ] Chrome contract
- [ ] Five role contracts: Employee, Manager, Finance, HR, Customer (Customer may conclude “no app surface”; customer portal out of scope)
- [ ] One cross-role composition map (org tree belongs here as manager/HR composition, not Admin/Settings)
- [ ] Reconcile/amend sketch-loop contract only if ambiguity remains after the contracts + map
- [ ] Implement by job cluster, not current route/page structure — do not recreate Phases 17–26

Historical v0.2 presentation leftovers (TICK-06, AVAIL-03..05, UXFD-02, SURF-01..08, POLS-01..11) are archived in `.planning/milestones/v0.2-REQUIREMENTS.md` as job-shaped **inputs**, not live route work.

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
- Customer-facing ticket portal — tickets are internal-only (D-E); v0.2.1 Customer contract may conclude “no app surface”
- Admin/Settings work — out of v0.2.1 scope
- Recreating deleted route-oriented Phases 17–26 — presentation is job-cluster, contract-first
- SLA engine / escalation chains / email ingestion / KB — anti-features for v0.2, ITSM creep
- Plan-adherence per-day-per-person metrics — aggregate-only per-period (D-U), never a surveillance number
- Per-customer request counts for billing — billing story superseded by coverage allocations; request counts deferred
- Kanban board — tickets are demand tracking, not task execution (no kanban, no sub-task trees, no comment threads)

## Context

Hourglass was originally built on SurrealDB; v0.1 fully ported to PostgreSQL (Phases Pg-1/Pg-2/Pg-3, ~7,300 lines deleted) and rebuilt the test infrastructure on testcontainers-go. The big-bang activity ontology migration (Phase 9) replaced the projects/subprojects model with a recursive activities tree; all approval routing was rewritten onto the activity chain. v0.1 shipped the Information Architecture phase (Phase 10) with the demo deployment topology documented (Compose + Caddy + cloudflared, ADR-BE-015).

v0.2 closed 2026-08-25 with 4 acknowledged audit leftovers (see STATE.md Deferred Items). Route-oriented Phases 17–26 were cancelled unbuilt. Presentation continues in v0.2.1 as contract-first job clusters.

**v0.2 ontology research round (2026-08-01/02):** a domain walkthrough surfaced two gaps in the accepted activity ontology (ADR-P-007): *origins* (where demand came from) and *coverage* (who pays). The research extended the model to three orthogonal planes — **direction** (the plan, mutable, manager/self-owned) → **facts** (time entries, immutable after approval) → **coverage** (the money label, mutable, snapshot-protected) — with the cardinal principle "the plan/decision never rewrites the fact". All decisions D-A…D-AA are closed; the vault note (`hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md`) is the record of truth. Formal ADRs (P-003 rev, P-013, P-014, P-015) are deferred and written as phases land; ADR-P-012 (coverage ledger) is drafted (Proposed) in the vault. No legacy-data migration is needed — Hourglass has never been deployed (P-007 D-6 big-bang landed pre-deploy).

## Constraints

- **[Tech stack]**: Go 1.26.1, PostgreSQL (pgx/v5), React 19, TanStack Router + Query, shadcn/ui + Tailwind — no SurrealDB
- **[Database]**: PostgreSQL via pgxpool, all repos exist, hand-written SQL, no ORM; migrations append-only per ADR-BE-004
- **[Auth]**: JWT in HttpOnly cookies (auth_token/refresh_token), bcrypt password hashing, strict reuse model with family revocation
- **[Architecture]**: Hexagonal — services in `internal/core/services/*`, HTTP adapters in `internal/adapters/primary/http/*`, PostgreSQL adapters in `internal/adapters/secondary/postgres/*`
- **[Domain]**: Captured effort is a fact, coverage is a decision, direction is a plan — the decision/plan never rewrites the fact (Σ allocations = entry hours; deviations are data, not violations)
- **[UI]**: v0.2.1 is contract-first — design-language, chrome, five role contracts, then one composition map, then sketch, then implement by job cluster. Sketching does not precede contracts. Do not implement before the contract/map sequence is settled. D-O IA leans are inputs, not a P-011 revision until prototypes land.

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
| Contracts carry `sold_hours` (D-N) | "Sold 4h, took 8h → next time sell 16" needs explicit sold figure; richer budget machinery stays at V4 (P-010) | ✓ Good |
| ADRs deferred to phase landing | Pre-deploy, the only ADR consumer is the build process itself (Stefano's call) | ✓ Good |
| Coverage allocations editable indefinitely (D-F) | Cutoff is a reporting snapshot, not a lock; realism over enforcement | ✓ Good |
| v0.2.1 is contract-first job clusters, not route phases | Route-oriented Phases 17–26 would have implemented pages; presentation must be designed as jobs per role, then composed | — Pending |

## Current Milestone: v0.2.1 Contract-first presentation — job clusters

**Goal:** Define how Hourglass is presented — visual language, chrome, five role contracts, and one composition map — then sketch and implement by job cluster. Not by current routes.

**Target features:**
- Design-language contract (Phase 15 tokens/components are inputs)
- Chrome contract (shell, nav, page anatomy; no Admin/Settings)
- Five role contracts: Employee, Manager, Finance, HR, Customer (Customer may conclude “no app surface”)
- One cross-role composition map (org tree is manager/HR composition)
- Sketch-loop contract amended only if still ambiguous after the map
- Job-cluster implementation **after** contracts + map + sketch — phases inserted then, not pre-drawn as routes

**Hard rules:** Do not recreate cancelled v0.2 Phases 17–26. Do not implement before the contract/map sequence is settled. Do not copy archived SURF/POLS/TICK-06/AVAIL-03..05 as page requirements.

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

*Last updated: 2026-08-25 after v0.2.1 start*
