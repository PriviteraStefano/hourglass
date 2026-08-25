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

## Milestone: v0.2 — Ontology Extension — Origins, Tickets & Coverage + Direction

**Shipped:** 2026-08-25
**Phases:** 6 (11–16) | **Plans:** 41 | **Tasks:** 108 | **Timeline:** 2026-08-01 → 2026-08-25
**Git:** 254 commits after `v0.1`; 509 files changed, +54989 / −43674

### What Was Built

- Three-plane ontology backends: origins + tickets (Phase 11), coverage allocation ledger (Phase 12), direction plan plane (Phase 13), availability absences/capacity (Phase 14)
- UX foundation: semantic status tokens + frozen shared components + sketch-loop contract (Phase 15)
- Integrity repair close: own-coverage read, expense `unit_id`, receipt auth, WR-05 capacity org-isolation, two rate-limiter defects; Phase 12 leftover smokes recorded in `16-01-SMOKE.md` (Phase 16)
- Route-oriented Phases 17–26 were **cancelled unbuilt** — not part of the shipped record

### What Worked

- Backend-first staging: planes landed before any new UI, so later presentation work can run against a complete API
- ADR + BE encoding drafted as each backend phase landed (P-003 rev / P-013 / P-012+BE-017 / P-015+BE-018 / P-008 rev+BE-019)
- Repair-only Phase 16 as a close gate: security/authz leaks fixed without opening surface scope
- Explicit cancellation of unbuilt route phases instead of leaving 17–26 as fake “in progress” work

### What Was Inefficient

- Original v0.2 roadmap mixed ontology backends with 10 route/page surface+polish phases; that structure had to be deleted rather than executed
- REQUIREMENTS.md still mapped SURF/POLS/AVAIL-FE to Phases 16–26 at close — 23 unchecked boxes that were not actual v0.2 blockers
- `SKETCH-LOOP-CONTRACT.md` was amended in the Phase 15 summary (2026-08-23) but the file on disk still says `applies-to: phases 16-26` and a global UI-only ban
- Phase 12 UAT/verification files were left stale even after equivalent smokes landed in Phase 16
- `gsd-tools milestone.complete` dumped every plan one-liner into MILESTONES.md and reset STATE frontmatter `milestone:` to v0.1 — needed a manual rewrite

### Patterns Established

- Repair-only close phase is a valid last phase when integrity leaks would otherwise leak into presentation
- Cancelled-unbuilt is a first-class roadmap outcome; do not keep empty phase numbers live
- Presentation work is contract-first job clusters, not current routes
- Smoke evidence can close a leftover scenario without rewriting the original UAT file — but the leftover must be acknowledged, not silently ticked

### Key Lessons

1. Do not plan UI as a list of existing pages. Role contracts + composition map first, then sketch, then implement by job.
2. A sketch-loop contract that names future phase numbers will rot the moment those phases are deleted. Keep process contracts phase-number-free.
3. Unchecked presentation requirements at backend close are inputs, not blockers. Archive them with outcomes; do not copy them into the next REQUIREMENTS.md.
4. Acknowledge audit leftovers explicitly in STATE.md Deferred Items. Equivalent evidence ≠ silent complete.

### Cost Observations

- Model mix: adaptive profile (planner/executor split)
- Sessions: ~3.5 weeks (2026-08-01 → 2026-08-25)
- Notable: Phase 16 was 1 plan / 8 tasks / 9 commits — cheaper than opening a surface phase on a leaking backend

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Key Change |
|-----------|--------|------------|
| v0.1 | 16 | ADR-express discuss bypass; wave-parallel gap closure; milestone audit added as recommended pre-close step |
| v0.2 | 6 | Backend-first three-plane ontology; repair-only close; cancelled unbuilt route phases; next milestone is contract-first |

### Cumulative Quality

| Milestone | Go packages green | Vitest | Playwright | Notes |
|-----------|-------------------|--------|------------|-------|
| v0.1 | ~30 | 51+ | ~25+ specs | e2e suites per domain (auth, approvals, activities, WGs, customers) |
| v0.2 | 26+ (Phase 16 suite green) | 51+ | carried | Ontology backends + integrity repair; no new Playwright surface suites |

### Top Lessons (Verified Across Milestones)

1. Lock decisions in ADRs before execution — downstream phases become mechanical.
2. Reconciliation debt (requirements status, UAT backlog, superseded dirs) compounds if not paid per-phase.
3. Do not leave unbuilt phases in the live roadmap — cancel them or they become fake scope.
4. Presentation is jobs × roles, not routes × pages.
