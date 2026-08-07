---
gsd_state_version: 1.0
milestone: v0.2
milestone_name: Ontology Extension — Origins, Tickets & Coverage + Direction
status: executing
stopped_at: Completed 11-04-PLAN.md
last_updated: "2026-08-07T10:22:50.778Z"
last_activity: 2026-08-07 -- Phase 11 execution started
progress:
  total_phases: 16
  completed_phases: 0
  total_plans: 6
  completed_plans: 4
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-02)

**Core value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

**Current focus:** Phase 11 — foundations-schema-origins-tickets-backend

## Current Position

Phase: 11 (foundations-schema-origins-tickets-backend) — EXECUTING
Plan: 5 of 6
Status: Ready to execute
Last activity: 2026-08-07 -- Phase 11 execution started

## Accumulated Context

### Decisions

Full log in PROJECT.md Key Decisions table. Decisions from the 2026-08-02 ontology research round (vault: `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md`, Parts 12–15) — all closed (D-A … D-AA), the note is the record of truth:

- **Three planes**: direction (plan) → facts (entries) → coverage (label). "The plan never rewrites the fact; the decision never rewrites the fact."
- **Coverage** (P-012): allocation ledger per entry, Σ allocations = entry hours invariant, to-cover queue, proposals computed-on-read (D-I), one-step manager confirm (D-L), editable indefinitely — cutoff is a reporting snapshot, not a lock (D-F), snapshot mechanics backend-only (F). ADR-P-012 draft already exists in vault (Proposed).
- **Tickets** (P-003 rev): first-class, internal-only; lifecycle open→triage→planned→in_progress→resolved→closed + reopen (D-A); kinds question/bug/change/evolution (closed set); chain is ticket→activity→entries; dismissal guard: blocked while any linked activity has logged hours (D-M).
- **Origins** (P-013): type + reference set on activities (manager-assignment → assigned_by/assigned_to · employee-proposal → proposed_by/reviewed_by · customer-ticket → ticket_id), set once at creation (D-D); proposal approval mirrors activity approval routing (D-G); refs stored directly on activities, derivation from first direction record is an additive read-path fallback (R4 resolution, Part 15).
- **Funding sources** (P-014): contract budget (default for billable) · support bucket (hours-only, carry-over, no expiry — D-P) · service request = zero-value contract (D-J) · internal absorption (mandatory reason: WarrantyBug/UnderEstimate/Goodwill) · cross-project transfer (explicit justification). Beneficiary unit collapsed with absorbing unit (D-B), nullable on activity, inherited downward like contract_id.
- **Direction** (P-015): one entity, mode derived (planned_date set → scheduled; null → queued) (D-R); managers + self-direction both first-class, self-direction no approval (D-S); WG-direction queued-only + claim model via origin_direction_id (D-T); lifecycle draft→active→superseded/cancelled, done/lapsed/claimed derived (D-V); per-day storage always, partial days first-class, no intra-day ordering (D-W/D-AA); org policy: deadline/horizon/mode configurable (D-X); scheduler warns on P-008 absences, never blocks (D-Y); direction-coverage read-model = planned vs capacity per employee/period (D-Z); plan-adherence aggregate-only per-period, never per-day-per-person (D-U).
- **Sold hours**: contracts carry `sold_hours` in v0.2 (D-N); V5 mines sold vs Σ actual. Budget machinery (rates/money/estimates) stays with P-010 at V4.
- **ADRs deferred to phase landing** (Stefano's call): formal P-003 rev / P-013 / P-014 / P-015 written as phases land; P-012 drafted now (exists in vault). Each backend phase drafts its ADR + BE encoding ADR.
- **Requirement mapping rule (v0.1 house style, kept)**: backend-deliverable requirements map to backend phases; user-visible requirements (SURF-*, TICK-06, AVAIL-03/04/05) map to frontend/surface phases.
- [Phase 11-foundations-schema-origins-tickets-backend]: Cycle tests self-seed their pre-state inline (helpers + direct SQL) — 003_seed.up.sql is retired; scripts/seed_demo.sql is demo data, not a test fixture

011 test pre-state org MUST reuse the fixed MVP seed UUID 019df8b0-0001-7000-8000-000000000001 because migration 011 seeds activity_kinds only for that org
014 numbered before 015 so activities.ticket_id FK resolves at apply time (A8 ordering)
CHECK constraints follow `origin_type IS NULL OR (...)` / `contract_type IS NULL OR (...)` so legacy NULL-discriminator rows pass (D-01/Pitfall 1)
reviewed_by deliberately unconstrained on employee_proposal origins (D-02, research OQ1) — Seed fixtures retired with the MVP seed; tests must not load demo data (plan 11-01 Task 1)
Migration 011's kind catalog seed targets the exact seed org; the activities (org_id, kind) FK depends on it
PostgreSQL resolves FKs at apply time; 015 references tickets
Three-valued logic: NULL passes CHECK; guard keeps legacy rows valid (D-16)
Only proposed_by is required for employee proposals (research OQ1)

- [Phase 11-foundations-schema-origins-tickets-backend]: managerResolution fields exported (ApproverIDs/RoleGated/SkipToFinance) so cross-package callers can consume the resolution; the struct type stays unexported — Go visibility rule blocks unexported field reads across packages — Plan required unexported fields consumed cross-package, which cannot compile; exporting the three fields preserves the plan's intent (type = implementation detail) with identical call-site semantics
- [Phase 11-foundations-schema-origins-tickets-backend]: routing.Service constructed once in cmd/server wiring and shared: time_entry now, proposal approval (plan 05) later — single instance, single repo set (D-G parity) — Extraction Pattern 5: shared package prevents entry/proposal routing drift; cmd/server builds the service once next to the other services
- [Phase 11-foundations-schema-origins-tickets-backend]: reviewed_by stays NULL at creation for employee_proposal origins (OQ1): CHECK requires only proposed_by; the approver is recorded in the proposal_approved audit row; ErrInvalidRequest on non-nil reviewed_by at create
- [Phase 11-foundations-schema-origins-tickets-backend]: Ticket audit rows are written synchronously inside the same transaction as the state change (OQ4/A3, Pitfall 2); BE-012 fire-and-forget stays for entry approvals only; outbox documented as the reversible successor of the user-deferred durability choice
- [Phase 11-foundations-schema-origins-tickets-backend]: Hard boundary list kept verbatim in the P-003 revision — tickets are demand tracking, not task execution (no kanban / sub-task trees / comment threads as conversation)
- [Phase 11-foundations-schema-origins-tickets-backend]: Dismissal guard signature pinned as LoggedHours(ctx, ticketID) (float64, error) on raw Σ (submitted+approved, not deleted) — Phase 12 swaps computation to net-of-compensations without signature change (D-13)
- [Phase 11-foundations-schema-origins-tickets-backend]: Terminal activity defined as: no non-terminal time entries (draft/submitted/pending_manager/pending_finance, is_deleted=false) on the linked-activity subtree via recursive CTE (OQ2)
- [Phase 11-foundations-schema-origins-tickets-backend]: Transition matrix pinned (A7/OQ6): open→triage, triage→planned, triage→dismissed, planned→in_progress, in_progress→resolved, resolved→closed, resolved→in_progress (reopen), open→dismissed; closed/dismissed terminal; else ErrInvalidTransition
- [Phase ?]: Sold-period clear uses the empty-string sentinel (sold_period: empty string) mirroring the existing customer_id nullable-clear pattern; absent field never emits NULL
- [Phase ?]: Update validates only fields present in the request; support-without-period surfaces ErrInvalidSoldConfig before the DB CHECK fires (house style sentinel-first)
- [Phase ?]: sold_hours has no update-clear branch — only sold_period can be cleared (plan-scoped nullable-clear)

### Pending Decisions (resolve during plan phase)

- **Phase 11:** BE schema shape for origins refs (nullable FK columns vs EAV) and origin-refs validation rules; ticket tables shape (status vocabulary + kind-transition matrix); `sold_hours` column semantics (per what period? total contract?). ADR-P-003 rev + P-013 drafted here.
- **Phase 12:** Funding-source polymorphic shape (tagged union vs one table per variant); proposal computation query shape; snapshot mechanics choice (frozen snapshot vs as-of-close reconstruction from BE-012 audit log — F is backend-only, both acceptable).
- **Phase 13:** Direction entity schema (per-day rows), claim-model tables, org-policy storage shape; supersede-chaining read path.
- **Phase 17/18:** D-O UI leans to validate in prototypes: allocation screen + to-cover queue → Review group; per-unit non-billed report → Reports; bucket setup + balance → Economics → Contracts; employee own-coverage → read-only yes. No P-011 revision until prototypes land.

### Blockers/Concerns

- [Phase 18 planning] Tickets are internal-only in v0.2 (D-E); external desk intake is a future hexagonal port — do NOT build a customer-facing ticket portal (explicitly out of scope)
- [Phase 12] Expense coverage is schema-ready only (D-K: polymorphic entry, `time` only allowed in v0.2) — the extra validation branch must be costed in the BE ADR and revisited if dead weight
- [Polish phases] Playwright e2e specs are the contract for polish phases — change test + behavior in the same plan
- UAT debt files (06/08/09/10-UAT, 08/09/10-VERIFICATION) are no longer on disk (archived at v0.1 close); STATE.md Deferred Items table below is the authoritative record — plan-phase should re-derive scenario details from phase plan archives under .planning/phases/

### Pending Todos

None new. See `.planning/todos/` for captured ideas.

## Deferred Items

Adopted from v0.1 close (2026-08-01). Each polish phase folds in its own debt — never deferred to a trailing verification phase:

| Category | Item | Status | Folded Into |
|----------|------|--------|-------------|
| uat_gap | 06-UAT.md | 14 pending scenarios | Phase 21 (Track polish) |
| uat_gap | 08-UAT.md | 4 pending scenarios | Phase 24 (Customers + Contracts polish) |
| uat_gap | 09-UAT.md | 1 pending scenario | Phase 22 (Activities polish) |
| uat_gap | 10-UAT.md | 6 pending scenarios | Phases 20 + 23 (Today, Approvals+WG polish) |
| verification_gap | 08-VERIFICATION.md | human_needed | Phase 24 (Customers + Contracts polish) |
| verification_gap | 09-VERIFICATION.md | human_needed | Phase 22 (Activities polish) |
| verification_gap | 10-VERIFICATION.md | human_needed | Phases 20 + 23 (Today, Approvals+WG polish) |
| quick_task | 260801-got-investigate-sidebar-collapsed-mode-hover | unknown | Triage during Phase 15/18 (UX Foundation, Today surfaces) |
| quick_task | 260801-o06-failed-to-initialize-postgresql-pool-pas | unknown | Triage during Phase 11/14 (backend startup) |

Remediation: `/gsd-verify-work` (UAT + human verification) per polish phase.

## Performance Metrics

**Velocity:** 62 plans completed in v0.1 (avg ~26 min across measured plans; P08-04 longest at 176 min, P10-06 at 453 min).

**Trend:** Stable — execution consistently lands 2-4 task plans per session day; verification debt accumulated at close (25 UAT + 3 human reviews) is the v0.2 folding target.

## Session Continuity

Last session: 2026-08-07T10:22:23.402Z
Stopped at: Completed 11-04-PLAN.md
Resume file: None
Next step: `/gsd-discuss-phase 11` (Foundations — schema + origins + tickets backend)

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 11-foundations-schema-origins-tickets-backend P01 | 6 | 3 tasks | 11 files |
| Phase 11-foundations-schema-origins-tickets-backend P03 | 6min | 2 tasks | 8 files |
| Phase 11-foundations-schema-origins-tickets-backend P02 | 12min | 3 tasks | 5 files |
| Phase 11-foundations-schema-origins-tickets-backend P04 | 7min | 2 tasks | 6 files |
