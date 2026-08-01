# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v0.1 — MVP Consolidation

**Shipped:** 2026-08-01
**Phases:** 16 | **Plans:** 62 | **Timeline:** 2026-03-26 → 2026-08-01 (incl. SurrealDB-era foundation)

### What Was Built

- PostgreSQL reboot: ~7,300 lines of SurrealDB deleted, 24-table schema, 18 repo adapters, testcontainers-go integration tests across ~30 Go packages
- Auth hardening: cookie unification, refresh-token rotation + reuse detection with family revocation, rate limiting, unbiased password reset codes
- Full feature surface: org hierarchy (ReactFlow), customers (incl. internal), contracts, exports (CSV/XLSX)
- Activity ontology (Phase 9): recursive `activities` entity replacing projects/subprojects, CTE-derived commercial context, approval routing through activity → WG → manager/delegate
- Information architecture (Phase 10): pillar sidebar with role-scoped visibility, Header+Body shell, Today landing, `/approvals` queue, Working Groups surface
- P0 audit gate closed (Phase 8): filterable list views, error boundaries, reuse detection, input length caps

### What Worked

- ADR-first phase execution (Phases 8-10): locked decisions upstream meant discuss-phase bypass was safe and execution was fast
- Testcontainers migration cycle tests (up → down → up) caught migration defects deterministically
- Big-bang migration with zero-orphan seed guarantee + frontend rename deferred to a dedicated phase (10) kept backend rewrite and UI work separable
- Wave-parallel execution (Phase 9 Wave 3 gap closures) efficiently burned down verification debt
- Sentinel-error discipline across services made handler mapping and test assertions consistent

### What Was Inefficient

- REQUIREMENTS.md traceability went stale after Phase 7 (45/60 unchecked at close) — required a full reconciliation pass before archiving; update it per-phase, not per-milestone
- Milestone closed without a `/gsd-audit-milestone` run — 25 UAT scenarios + 3 human verification reviews remain open as debt
- Legacy/superseded phase directories (01-org-hierarchy-edge-driven, pg-1) still exist on disk with plans but no summaries — they pollute progress scans and plan counts (62 vs 61 canonical)
- STATE.md "Last Activity Description" field format mismatch at archive time — minor tooling friction

### Patterns Established

- Express-path discuss-phase bypass for ADR/audit-driven phases (CONTEXT.md = the spec)
- Migration numbering from max+1 with append-only rule (ADR-BE-004) + cycle tests
- Same-id migration strategy for renames (activities keep old project UUIDs) so down migrations restore 1:1
- Detail composition (GetAncestry/ResolveCommercialContext/ResolveBillability) in the handler via repo port, derived-never-stored
- UI-SPEC + Header/Body shell + leaf errorComponents for page wrappers

### Key Lessons

1. Keep REQUIREMENTS.md traceability updated within each phase's summary — retroactive reconciliation at close is expensive and error-prone.
2. Defer UAT/verification debt is real: run `/gsd-audit-milestone` + `/gsd-verify-work` before declaring close, or budget the debt explicitly in the next milestone.
3. Archive or clean superseded phase directories at supersede time — they distort route-0 resume scans and milestone stats.
4. Atomic big-bang migrations with up/down/up cycle tests de-risk schema rewrites better than incremental "safe" migrations.

### Cost Observations

- Model mix: opus (planner) + sonnet (executor), balanced profile
- Sessions: ~40+ across the milestone
- Notable: Phase 10 P06 ran 453 min (7.5h) — largest single plan; P08-04 176 min. Big surface plans (UI + API) dominate duration.

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Key Change |
|-----------|--------|------------|
| v0.1 | 16 | ADR-express discuss bypass; wave-parallel gap closure; milestone audit added as recommended pre-close step |

### Cumulative Quality

| Milestone | Go packages green | Vitest | Playwright | Notes |
|-----------|-------------------|--------|------------|-------|
| v0.1 | ~30 | 51+ | ~25+ specs | e2e suites per domain (auth, approvals, activities, WGs, customers) |

### Top Lessons (Verified Across Milestones)

1. Lock decisions in ADRs before execution — downstream phases become mechanical.
2. Reconciliation debt (requirements status, UAT backlog, superseded dirs) compounds if not paid per-phase.
