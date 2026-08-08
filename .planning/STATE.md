---
gsd_state_version: 1.0
milestone: v0.2
milestone_name: Ontology Extension — Origins, Tickets & Coverage + Direction
current_phase: 12
current_phase_name: Coverage Backend — The Allocation Loop
status: executing
stopped_at: Completed 12-03-PLAN.md
last_updated: "2026-08-08T09:25:51.177Z"
last_activity: 2026-08-08
last_activity_desc: Phase 12 execution started
progress:
  total_phases: 16
  completed_phases: 1
  total_plans: 15
  completed_plans: 12
  percent: 6
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-02)

**Core value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

**Current focus:** Phase 12 — Coverage Backend — The Allocation Loop

## Current Position

Phase: 12 (Coverage Backend — The Allocation Loop) — EXECUTING
Plan: 5 of 7
Status: Ready to execute
Last activity: 2026-08-08 — Phase 12 execution started

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
- [Phase ?]: General audit types named GeneralAuditLogRepository (port + postgres): ports.AuditLogRepository and postgres.AuditLogRepository already exist for the entry-scoped BE-012 audit (time_entry_approvals); renaming the new D-05 types is the minimal collision-free path preserving both behaviors
- [Phase ?]: Dead entry-scoped MockAuditLogRepo in testdata renamed MockTimeEntryAuditLogRepo; the MockAuditLogRepo name now serves the general audit.AuditLog port (zero usages before)
- [Phase ?]: ApproveProposal flips is_active via the repo Update directly (bypassing the service Update finance gate) — the routing approver check IS the gate; self-approval checked before routing so no-self-approval is deterministic even on D-11 skipToFinance paths
- [Phase ?]: Proposer primary-unit lookup degrades to uuid.Nil on malformed unit IDs (no panic); routing then falls to the terminal role-gated resolution
- [Phase 11-foundations-schema-origins-tickets-backend]: Transition rejects to==dismissed: the pinned matrix's dismissal edges (open/triage -> dismissed) are consumed ONLY by Dismiss — the guarded path with the D-11 role gate + D-13 hours guard + dismissed_hours snapshot. Allowing them in Transition would let an owner/assignee bypass the guard (T-11-07) — Security fix (Rule 2): without the block, an owner/assignee could dismiss a ticket with logged hours via the transition endpoint, bypassing the D-13 guard that TICK-04 mandates
- [Phase 11-foundations-schema-origins-tickets-backend]: Triage implements the service-level fast-fail (KindExists/parent/contract same-org) as optional UX exactly as planned — the repo's in-tx SELECT EXISTS checks remain the correctness guarantee with DB FK/CHECK constraints as the third line (Pitfall 7, T-11-06) — Plan said the service MAY fast-fail; unit tests exercise the fast-fail, the contract test proves the in-tx gate
- [Phase 11-foundations-schema-origins-tickets-backend]: Dismissal-guard contract test links the activity via the OQ5 customer_ticket path on an open ticket (the exact shape the guard must catch) rather than via triage — triage moves the ticket to planned, from which dismissal is illegal per the matrix — Test correctness: the 409 scenario must hold while the ticket is open|triage
- [Phase 11-foundations-schema-origins-tickets-backend]: Repo layer is authoritative for ticket state-machine and dismissal-guard decisions: every matrix/guard check re-validated inside the mutator tx under the FOR UPDATE row lock (Pitfall 7, ADR-BE-016); service pool-level checks are fast-fail UX only (CR-01 closure) — CR-01 TOCTOU root cause: pool-level checks before the mutator tx left a check-then-act window. In-tx lock + re-check + status-precondition UPDATE backstop closes it; pool signatures stay Phase-12-stable (loggedHoursTx/hasNonTerminalActivitiesTx are private tx-executed helpers)
- [Phase 11]: DismissedNote is a derived read-model field, not a column: computed in scanTicketRow only when Status == 'dismissed' && DismissedHours != nil — OQ3/A4, D-13, no migration; note number formatted with FormatFloat precision -1 so the raw Σ reads naturally (5.00 -> '5', 7.50 -> '7.5') — IN-02 closure: the TICK-04 note claim is observable at the API boundary, rendered server-side on every ticket read
- [Phase 11]: Title validation mirrors migration 014's VARCHAR(255) column exactly: Create/UpdateDetails reject >255-char titles (WR-04) and empty-title updates (IN-01) with ErrInvalidRequest -> 400 — validation precedes side effects (payload map / repo call), no 500 path remains for title input — T-11-16/T-11-17 mitigated at the service boundary; the oversized-input 500 path from the column is eliminated
- [Phase 12]: source_type stays nullable: the 3VL guard CHECK (source_type IS NULL OR ...) is the enforcement, not a NOT NULL clause — legacy all-NULL rows pass (mirrors 015 origin_type / 016 contract_type) — source_type stays nullable: the 3VL guard CHECK (source_type IS NULL OR ...) is the enforcement, not a NOT NULL clause — legacy all-NULL rows pass (mirrors 015 origin_type / 016 contract_type)
- [Phase 12]: coverage_allocations.entry_id has no FK (polymorphic D-K); entry_type CHECK ('time') + 12-04 service branch are the costed belt-and-braces pair — coverage_allocations.entry_id has no FK (polymorphic D-K); entry_type CHECK ('time') + 12-04 service branch are the costed belt-and-braces pair
- [Phase 12]: 020 has no UNIQUE(org_id, period_start, period_end): duplicate-close rejection is a repo-level in-tx check returning 409 (A6) — 020 has no UNIQUE(org_id, period_start, period_end): duplicate-close rejection is a repo-level in-tx check returning 409 (A6)
- [Phase 12-coverage-backend-the-allocation-loop]: ADR-P-012 accepted 2026-08-07; D-1..D-6 operationalized via ADR-BE-017; snapshot-not-lock implemented as the frozen period-close snapshot (D-10/D-11/D-12) — ADR-P-012 accepted 2026-08-07; D-1..D-6 operationalized via ADR-BE-017; snapshot-not-lock implemented as the frozen period-close snapshot (D-10/D-11/D-12)
- [Phase 12-coverage-backend-the-allocation-loop]: ADR-BE-017 pins: zero-value predicate contract_type=project AND sold_hours IS NOT DISTINCT FROM 0 (A3), raw bucket balance without period scaling (A8), duplicate close rejected with 409 (A6), audit vocabulary entity_type=coverage_allocation + actions allocations-set/coverage-closed (A7) — ADR-BE-017 pins: zero-value predicate contract_type=project AND sold_hours IS NOT DISTINCT FROM 0 (A3), raw bucket balance without period scaling (A8), duplicate close rejected with 409 (A6), audit vocabulary entity_type=coverage_allocation + actions allocations-set/coverage-closed (A7)
- [Phase 12-coverage-backend-the-allocation-loop]: D-K polymorphic validation cost stated honestly in ADR-BE-017: one service branch rejecting entry_type != time + the entry_type CHECK; COV-06 (expense) needs an additive ALTER + service rule change, not a redesign — D-K polymorphic validation cost stated honestly in ADR-BE-017: one service branch rejecting entry_type != time + the entry_type CHECK; COV-06 (expense) needs an additive ALTER + service rule change, not a redesign
- [Phase ?]: FundingContext chain-data type defined in domain/activity (ContractID/ContractType/SoldHours, all pointer) — coverage service 12-05 consumes it via the port, never stored
- [Phase ?]: Two separate CTE resolvers with independent NULL-walk guards: ResolveBeneficiaryUnit walks beneficiary_unit_id (absorption default), ResolveFundingContext walks contract_id + contracts JOIN for contract_type/sold_hours (D-04 input)
- [Phase ?]: beneficiary_unit_id is EDITABLE on Update (unlike origin refs): Update SET branch mirrors ContractID, service re-validates same-org on every write, hasOriginFields untouched (T-12-06)
- [Phase ?]: Same-org validation via unitRepo.GetByID + u.OrgID == orgID (expense-service pattern); ErrInvalidRequest on mismatch -> 400, fetch error surfaces as-is

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
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 11-foundations-schema-origins-tickets-backend P07 | 15min | 3 tasks | 3 files |
| Phase 11 P08 | 3h 50m | 2 tasks | 6 files |
| Phase 12 P01 | 6min | 2 tasks | 11 files |
| Phase 12-coverage-backend-the-allocation-loop P02 | 4min | 2 tasks | 4 files |
| Phase 12-coverage-backend-the-allocation-loop P04 | 3min | 2 tasks | 5 files |
| Phase 12-coverage-backend-the-allocation-loop P03 | 12min | 2 tasks | 8 files |

## Session Continuity

Last session: 2026-08-08T09:25:50.668Z
Stopped at: Completed 12-03-PLAN.md
Resume file: None
Next step: `/gsd-discuss-phase 11` (Foundations — schema + origins + tickets backend)

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 11-foundations-schema-origins-tickets-backend P01 | 6 | 3 tasks | 11 files |
| Phase 11-foundations-schema-origins-tickets-backend P03 | 6min | 2 tasks | 8 files |
| Phase 11-foundations-schema-origins-tickets-backend P02 | 12min | 3 tasks | 5 files |
| Phase 11-foundations-schema-origins-tickets-backend P04 | 7min | 2 tasks | 6 files |
| Phase 11-foundations-schema-origins-tickets-backend P05 | 10min | 3 tasks | 21 files |
| Phase 11-foundations-schema-origins-tickets-backend P06 | 44min | 3 tasks | 11 files |
