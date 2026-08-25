---
gsd_state_version: 1.0
milestone: v0.2
milestone_name: Ontology Extension — Origins, Tickets & Coverage + Direction
current_phase: 16
status: completed
stopped_at: Completed 16-01-PLAN.md
last_updated: "2026-08-24T21:55:54.512Z"
last_activity: 2026-08-25
last_activity_desc: Completed quick task 260825-no-error-leak — no error leak (#6); next #7 export streaming
progress:
  total_phases: 16
  completed_phases: 6
  total_plans: 41
  completed_plans: 41
  percent: 38
current_phase_name: ux-foundation-design-tokens-shared-components
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-02)

**Core value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

**Current focus:** Phase 15 — ux-foundation-design-tokens-shared-components

## Current Position

Phase: 16 — COMPLETE
Plan: 4 of 4
Status: Phase 16 complete
Last activity: 2026-08-24 — Phase 16 marked complete

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
- [Phase 12-coverage-backend-the-allocation-loop]: Close audit row addresses the close (entity_id = closeID), not the entry: A7 per-entry history covers allocation changes only; the close event is read by close id — The plan's test wording implied close rows appear in entry history; the port contract pins entity_id = entry — port wins
- [Phase 12-coverage-backend-the-allocation-loop]: Partial coverage states cannot be created via the replace-set (Σ == hours enforced in-tx); they arise from later entry-hours edits — Test simulates the realistic partial path (allocate full, then bump entry hours)
- [Phase 12-coverage-backend-the-allocation-loop]: FundingContext gained BeneficiaryUnitID (additive): the pinned DefaultSource signature needs the absorption branch input; the service attaches the resolved unit when the chain has no contract — Rule 3 fix — the 12-03 chain struct lacked the absorption default; keeps DefaultSource pure and the six-case matrix table-driven
- [Phase 12-coverage-backend-the-allocation-loop]: GetAllocations reuses the pinned Propose read path — the 12-05 service surface has no ListByEntry method, so the thin handler consumes Propose (proposal, allocs) instead of extending the service surface — Plan referenced a service read missing from the pinned 12-05 surface; reusing Propose honors the pinned surface. time_entry.ErrTimeEntryNotFound normalizes to coverage.ErrEntryNotCoverable in the service so the 404 contract holds without leaking cross-domain sentinels into the handler.
- [Phase 13]: 021 status vocabulary lives in the single named constraint direction_status_check; the inline column CHECK was dropped (PostgreSQL auto-names inline CHECKs, colliding 42710 with the explicit ALTER — Rule 1 auto-fix) — 021 status vocabulary lives in the single named constraint direction_status_check; the inline column CHECK was dropped (PostgreSQL auto-names inline CHECKs, colliding 42710 with the explicit ALTER — Rule 1 auto-fix)
- [Phase 13]: assertPrimaryKey helper added locally in direction_ontology_migrations_test.go (pg_constraint contype p) — no shared PK-assert helper existed in the postgres test package — assertPrimaryKey helper added locally in direction_ontology_migrations_test.go (pg_constraint contype p) — no shared PK-assert helper existed in the postgres test package
- [Phase 13]: 021 header comment avoids the literal word "unique" — grep-based acceptance checks for an absent UNIQUE constraint would trip on the comment; the DDL carries none — 021 header comment avoids the literal word "unique" — grep-based acceptance checks for an absent UNIQUE constraint would trip on the comment; the DDL carries none
- [Phase 13-direction-backend-the-plan-plane]: [Phase 13-02]: ADR-P-015 + ADR-BE-018 drafted into the vault: direction plane (derived mode, per-day rows, supersede chain, derived states, WG claim model, org policy stored-not-enforced, P-008 warning overlay, coverage read-model, origin fallback) + BE encoding (status/derived/claim-spectrum/audit/settings vocabularies, claim lock FOR UPDATE + in-tx Sigma in cents over draft|active claim rows -> 409 ErrClaimOverBudget, supersede-of-claim-row inheritance, est_hours DECIMAL(8,2), settings CRUD literal routes, 8 assumption pins) — [Phase 13-02]: vocab/mechanism pins recorded BEFORE code lands — domain constants (13-03) and audit inserts (13-05) compile against them; three assumption-delta decisions recorded (identity no-change / fallback add-alongside / policy promoted)
- [Phase ?]: MockDirectionRepo absence-window stub field is named Windows (setter SetAbsenceWindows): the plan's literal field name AbsenceWindows collides with the port method AbsenceWindows — Go forbids a field and method with the same name on one type; the pinned setter surface is unchanged (13-07/13-08 tests seed via the setters)
- [Phase 13]: ResolvePlanningMode precedence: membership override -> org default -> manager_planned fallback; invalid mode in either position is ErrInvalidValue (D-13-19) — Seam for 13-07 mode gate; JSONB store unvalidated so corruption surfaces, never silently defaults
- [Phase 13]: Claim audit entity_id pinning: the repo generates the claim row id (the port signature takes no id), so Claim pins the audit row entity_id to the claim row it creates when the caller passed uuid.Nil — entity_id = the direction row id per ADR-BE-018 §3. — Claim audit entity_id pinning: the repo generates the claim row id (the port signature takes no id), so Claim pins the audit row entity_id to the claim row it creates when the caller passed uuid.Nil — entity_id = the direction row id per ADR-BE-018 §3.
- [Phase 13]: Full-interface assertion deferred to 13-06: the port declares read-model methods that 13-06 owns, so the var _ ports.DirectionRepository assertion cannot compile on the mutator-only half; Get ships in 13-05, the assertion lands with 13-06. — Full-interface assertion deferred to 13-06: the port declares read-model methods that 13-06 owns, so the var _ ports.DirectionRepository assertion cannot compile on the mutator-only half; Get ships in 13-05, the assertion lands with 13-06.
- [Phase 13]: Coverage/AbsenceWindows normalize scanned DATE columns to UTC midnight (normalizeDay): PostgreSQL DATE values scan back in the SESSION timezone (e.g. +02:00 Local), making day-key comparisons and JSON serialization nondeterministic; the read-model day semantics are timezone-free, the mutator scans stay as-is — Coverage/AbsenceWindows normalize scanned DATE columns to UTC midnight (normalizeDay): PostgreSQL DATE values scan back in the SESSION timezone (e.g. +02:00 Local), making day-key comparisons and JSON serialization nondeterministic; the read-model day semantics are timezone-free, the mutator scans stay as-is
- [Phase 13]: ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs all landed on direction_repository.go and the full-interface assertion var _ ports.DirectionRepository compiles (deferred from 13-05); AbsenceWindows maps availability_windows.user_id (migration 012 column name) to AbsenceWindow.EmployeeID — ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs all landed on direction_repository.go and the full-interface assertion var _ ports.DirectionRepository compiles (deferred from 13-05); AbsenceWindows maps availability_windows.user_id (migration 012 column name) to AbsenceWindow.EmployeeID
- [Phase 13-direction-backend-the-plan-plane]: Unclaim added to ports.DirectionRepository + MockDirectionRepo (additive, plan-sanctioned "repo.Unclaim"): the postgres repo's in-tx claim-row guard is now reachable via the port
- [Phase 13-direction-backend-the-plan-plane]: Unclaim audit = 'cancelled' action with {reason} (plan text + 13-05 repo tests); AuditActionUnclaimed stays a pinned ADR vocabulary constant, never written by the unclaim path
- [Phase 13-direction-backend-the-plan-plane]: Claim reuses the create-side whole-cent validation (D-13-03): sub-cent claims would corrupt the repo's cents-based Sigma (rounded Sigma != stored DECIMAL(8,2))
- [Phase 13-direction-backend-the-plan-plane]: Coverage period totals are computed over the FULL row set (capacity-0 away days keep their zero capacity); only the uncovered rows list excludes them (D-13-26)
- [Phase 13-direction-backend-the-plan-plane]: created/activated audits carry nil payloads (the 13-05 repo test contract); cancelled carries {reason}, claimed carries {wg_row_id, est_hours} with uuid.Nil entity (repo pins it to the claim row)
- [Phase 13]: Origin fallback lives in the activity read path at the service layer (OriginType == nil predicate), derived from FirstDirectionRefs — read-only, never written back
- [Phase 13]: The create response shape is {row, warnings} with warnings normalized to an always-array at the handler boundary (D-13-03/13-UI-SPEC); read-models carry rows/coverage rows + totals + warnings
- [Phase 13]: The seven direction routes are wired with middleware.Auth; the direction service reuses the SHARED orgsettings + routing services (no second instances, D-G parity)
- [Phase 13-09]: CR-01 handler regression activates the user-targeted row first: the nil-guard sits after the status fast-fail, so the 404 contract is only reachable for an ACTIVE user row — Plan's own unit behavior pinned status active; its abbreviated handler-test text omitted the activate step
- [Phase 13-09]: wrapPGError untouched: with the service-side maxEstHours ceiling the PG 22003 path is unreachable for client input; a global 22003 mapping would alter unrelated repos (time entries) — Per plan Task 3 action; scope boundary honored
- [Phase ?]: WR-03 reversal (13-10): Unclaim writes AuditActionUnclaimed ('unclaimed') — ADR-BE-018 §3 always pinned the vocabulary; the code drifted to 'cancelled'. Aligning code to the ADR makes unclaim events distinguishable from cancels for Phase 19 history filters and makes the exported constant live. Reverses the 13-05 note. Reversible: one line + docs.
- [Phase 14-01]: 024 carries NO default-flag column on contract_types: the org default schedule is an org_settings key (D-14-18, research OQ4); header comment avoids the literal token so grep acceptance checks don't trip (Phase 13 'unique' lesson)
- [Phase 14-01]: 023 down downgrades rejected/withdrawn rows to 'declared' before restoring the two-value CHECK (23514-safe restore); teardown FK-safe order: certificate_attachments before availability_windows, contract_types after organization_memberships
- [Phase ?]: The D-14-21 behavioral proof lives at the repo boundary, not the service mock: MockDirectionRepo returns windows verbatim and the domain AbsenceWindow carries no status — the RED test asserts the SQL predicate directly — Service-level declared/confirmed distinction is mechanically inexpressible
- [Phase ?]: Handler-level no-warning proof uses the coverage self-view, not the plan read: ListPlan derives its employee set from plan rows, so employees without rows get no warnings regardless of status — Coverage resolves employees by scope and runs the identical warning overlay
- [Phase 14-availability-backend-absences-capacity]: Full-interface assertion lands in 14-03 via not-implemented stubs: the plan pins BOTH the assertion and the tracer fixture wiring in Task 1 while the port declares 14 methods owned by 14-05/06/07 — the Phase 13 deferral precedent (13-05) only worked because nothing wired the partial repo; the stubs fail every later plan's RED tests cleanly and no route can reach them — Full-interface assertion lands in 14-03 via not-implemented stubs: the plan pins BOTH the assertion and the tracer fixture wiring in Task 1 while the port declares 14 methods owned by 14-05/06/07 — the Phase 13 deferral precedent (13-05) only worked because nothing wired the partial repo; the stubs fail every later plan's RED tests cleanly and no route can reach them
- [Phase 14-availability-backend-absences-capacity]: windowHoursValid exported as WindowHoursValid: the plan's service fast-fail names the domain helper, which 14-02 pinned unexported — exporting keeps single-source-of-truth (no re-implementation drift) — windowHoursValid exported as WindowHoursValid: the plan's service fast-fail names the domain helper, which 14-02 pinned unexported — exporting keeps single-source-of-truth (no re-implementation drift)
- [Phase 14-availability-backend-absences-capacity]: Service sets the observable status (declared/confirmed) before the repo call; the repo re-derives from kind authoritatively — the mock-based unit test caught the empty-status gap — Service sets the observable status (declared/confirmed) before the repo call; the repo re-derives from kind authoritatively — the mock-based unit test caught the empty-status gap
- [Phase 14-availability-backend-absences-capacity]: The confirmed audit row for medical is repo-internal: the port's Declare takes ONE audit row, so the 'two audit rows with actor id' assertion lives at the repo boundary battery (Task 2), not the service mock (mechanically inexpressible there — same class as the 14-04 RED-placement deviation) — The confirmed audit row for medical is repo-internal: the port's Declare takes ONE audit row, so the 'two audit rows with actor id' assertion lives at the repo boundary battery (Task 2), not the service mock (mechanically inexpressible there — same class as the 14-04 RED-placement deviation)
- [Phase ?]: The orgsvc constructor's new dependency is the SHARED availability service: it validates contract_type_id same-org via ListContractTypes (no second repo) and pins the schedule audit vocabulary (D-14-29, D-G parity)
- [Phase ?]: The {before, after} audit payloads are built REPO-side from the FOR UPDATE locked rows (contract-type update + membership schedule — the UpdateMedical shape): the service cannot know the before state
- [Phase ?]: ScheduleRequest fields are optional but at least one must be present (no-op writes -> 400); a cross-org/missing contract_type_id -> 400 invalid request, not 404
- [Phase ?]: default_contract_type_id is write-time-UUID-string + read-time-existence/org validated: the orgsettings validator only checks addressability, ResolveSchedule surfaces missing/wrong-org/unparsable as ErrInvalidValue (T-14g-19)
- [Phase ?]: An override WITHOUT a contract type merges over the 8x5 fallback base (flagged-assumption discretion pinned with a test)
- [Phase 14-availability-backend-absences-capacity]: The Service constructor gained the orgMgmtRepo dependency (ports.OrganizationManagementRepository): the org capacity scope needs ListMembers, which the pinned orgRepo (GetMembership only) cannot serve — all 4 wiring sites updated (Rule 3) — Plan mandates org -> orgRepo.ListMembers; the member list lives on the org-mgmt port
- [Phase 14-availability-backend-absences-capacity]: The workload subtree CTE anchors at the org's root activities: the pinned Capacity signature carries no activity parameter, so the workload column is the employee's org-subtree Sum for every scope; the activity-scope universe comes from ActivityWorkloadEmployees (D-14-19/20) — Pinned port cannot change; recursion depth >= 2 repo battery proves the walk
- [Phase 14-availability-backend-absences-capacity]: Month-without-matrix types resolve DayHours = nil through the membership path (never the 8x5 substitute): the nil matrix is the D-14-17 derivation signal; a week type with a nil matrix surfaces ErrInvalidValue — Task 3 RED caught the 8x5 substitute (8/day instead of 100/5=20/day); 14-06-pinned paths unchanged
- [Phase 14-availability-backend-absences-capacity]: The writeError sentinel map needed no extension: the 14-03 map already covered every error the 14-08 handlers surface (ErrNotMedical/ErrRejectReasonRequired/ErrCertificateRequired -> 400, ErrOverlap/ErrInvalidTransition -> 409) — Plan text said writeError gains remaining sentinel cases; the map was already complete - no change needed
- [Phase 14-availability-backend-absences-capacity]: Empty-string certificate_ref rejected at the service boundary (Declare parity, availability.go:106) AND nil/empty refused in-tx at the repo before UPDATE/audit — the D-14-05 invariant holds on every path (belt-and-braces per plan) — gap closure plan (CR-01/WR-01) — belt-and-braces placement of the D-14-05 invariant at service + repo boundaries
- [Phase 14-availability-backend-absences-capacity]: WindowHoursValid epsilon is < 1e-9 — the review-verified shape — keeping 4.005/99.995 invalid while accepting binary-inexact cent values (0.29/1.15/2.30) — gap closure plan (CR-01/WR-01) — belt-and-braces placement of the D-14-05 invariant at service + repo boundaries
- [Phase 14]: wrapPGError extended with a 22003 -> ports.ErrInvalidRequest case despite the Phase 13 13-09 scope note (wrapPGError untouched) — WR-03 mandates the mapping; effect on unrelated repos is strictly 500->400 on numeric overflow, the house rule's direction; no existing test asserts a 500 on 22003 — Client-input numeric ceilings validated in the service (fast-fail before any repo call); adapter still maps the overflow SQLSTATE as belt-and-braces so the surface can never 500
- [Phase 14-availability-backend-absences-capacity]: Workload CTE period predicate mirrors the sibling columns verbatim (entry_date >= $3::date AND entry_date < $4::date + INTERVAL '1 day') — same args declared/partial_abs/full_abs consume, CTE order unchanged (WR-02) — Task 2's HTTP regression is committed as the proven end-to-end guard: its RED cannot fail after Task 1's GREEN lands the predicate; failure mode proven at repo level (22.0 vs 10.0) — same RED-placement class as 14-04
- [Phase 15-ux-foundation-design-tokens-shared-components]: StatusBadgeProps export shape kept verbatim so all 7 consumer sites + time-entries re-export compile with zero edits (Pitfall 3); generic StatusBadge<S> + STATUS_ROLE_MAP covers all 5 vocabularies + D-15-04 warning keys with unknown→neutral fallback (Phase 15-01) — StatusBadgeProps export shape kept verbatim so all 7 consumer sites + time-entries re-export compile with zero edits (Pitfall 3); generic StatusBadge<S> + STATUS_ROLE_MAP covers all 5 vocabularies + D-15-04 warning keys with unknown→neutral fallback (Phase 15-01)
- [Phase 15-ux-foundation-design-tokens-shared-components]: Role variant recipes are static per-role class literals (parenthesized custom-property alpha, bg-(--status-{role})/10) — compiled cleanly under the project Tailwind v4 setup, no bg-status-{role}/10 fallback needed (Phase 15-01) — Role variant recipes are static per-role class literals (parenthesized custom-property alpha, bg-(--status-{role})/10) — compiled cleanly under the project Tailwind v4 setup, no bg-status-{role}/10 fallback needed (Phase 15-01)
- [Phase 15-ux-foundation-design-tokens-shared-components]: EmptyTitle 500→600 remap intentionally changes today-page/approvals-page empty-state appearance per the 2-weight typography contract (Pitfall 5); empty-state test negative assertion split-regex'd (/font-(medium)/) to keep the prohibition grep green (Phase 15-01) — EmptyTitle 500→600 remap intentionally changes today-page/approvals-page empty-state appearance per the 2-weight typography contract (Pitfall 5); empty-state test negative assertion split-regex'd (/font-(medium)/) to keep the prohibition grep green (Phase 15-01)
- [Phase 15-ux-foundation-design-tokens-shared-components]: Frozen ConfirmDialog (D-15-07): controlled presentational destructive confirmation with required-reason gate mirroring the server 400 invariant (D-13-10/D-13-16); error semantics error!==undefined with empty fallback to the default copy; invalidateQueries marked void (TanStack v5 Promise); confirm Button uses variant="destructive" — Frozen ConfirmDialog (D-15-07): controlled presentational destructive confirmation with required-reason gate mirroring the server 400 invariant (D-13-10/D-13-16); error semantics error!==undefined with empty fallback to the default copy; invalidateQueries marked void (TanStack v5 Promise); confirm Button uses variant="destructive"
- [Phase 15-ux-foundation-design-tokens-shared-components]: User-override: @tanstack/react-table pinned at ^9.1.2 (v9, published 2026-08-09) instead of the plan's ^8.21.3 v8 pin — approved at the Task 1 package gate; the DataTable is implemented entirely against the installed v9 API (useTable + tableFeatures slots + table.FlexRender); the plan's no-v9-leak prohibition is inverted to require the v9 surface while the data-lifecycle prohibition still holds; exports DataTableFeatures so consumers type ColumnDef<DataTableFeatures, T>[]

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

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260825-migration-ledger | Add schema_migrations ledger + per-file transactions to migration runner | 2026-08-25 | 8dc304d | [260825-migration-ledger](./quick/260825-migration-ledger/) |
| 260825-compose-root | Extract composition root (buildGraph) + explicit time-entry approval port (CONCERNS #2 #3) | 2026-08-25 | 2ee8ae9 | [260825-compose-root](./quick/260825-compose-root/) |
| 260825-route-tree-gitignore | Untrack generated TanStack route tree (CONCERNS #5) | 2026-08-25 | 08b202b | [260825-route-tree-gitignore](./quick/260825-route-tree-gitignore/) |
| 260825-no-error-leak | Stop leaking internal error strings to clients in auth handlers (CONCERNS #6) | 2026-08-25 | f4ee220 | [260825-no-error-leak](./quick/260825-no-error-leak/) |
| 260825-export-streaming | Bound export memory — range cap, CSV/XLSX streaming, drop in-memory sort (CONCERNS #7) | 2026-08-25 | (see commit) | [260825-export-streaming](./quick/260825-export-streaming/) |

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
| Phase 12-coverage-backend-the-allocation-loop P06 | 8m | 2 tasks | 3 files |
| Phase 12-coverage-backend-the-allocation-loop P05 | 7 min | 2 tasks | 3 files |
| Phase 12-coverage-backend-the-allocation-loop P07 | 6min | 2 tasks | 5 files |
| Phase 13 P13-01 | 18 min | 2 tasks | 6 files |
| Phase 13-direction-backend-the-plan-plane P13-02 | 4min | 2 tasks | 4 files |
| Phase 13 P13-03 | 42 min | 3 tasks | 8 files |
| Phase 13 P04 | 40min | 3 tasks | 10 files |
| Phase 13 P05 | 47 min | 3 tasks | 2 files |
| Phase 13 P13-06 | 38 min | 3 tasks | 2 files |
| Phase 13-direction-backend-the-plan-plane P07 | 11min | 3 tasks | 4 files |
| Phase 13 P13-08 | 2h 23m | 3 tasks | 9 files |
| Phase 13-direction-backend-the-plan-plane P09 | 6min | 3 tasks | 5 files |
| Phase 13-direction-backend-the-plan-plane P09 | 6min | 3 tasks | 5 files |
| Phase 13-direction-backend-the-plan-plane P10 | 18 min | 2 tasks | 5 files |
| Phase 14-availability-backend-absences-capacity P01 | 90min | 3 tasks | 9 files |
| Phase 14-availability-backend-absences-capacity P04 | 9min | 2 tasks | 6 files |
| Phase 14-availability-backend-absences-capacity P03 | 17min | 3 tasks | 10 files |
| Phase 14-availability-backend-absences-capacity P05 | 30min | 3 tasks | 5 files |
| Phase 14-availability-backend-absences-capacity P06 | 31min | 3 tasks | 23 files |
| Phase 14-availability-backend-absences-capacity P07 | 25min | 3 tasks | 14 files |
| Phase 14-availability-backend-absences-capacity P08 | 11min | 3 tasks | 4 files |
| Phase 14-availability-backend-absences-capacity P09 | 15min | 3 tasks | 6 files |
| Phase 14 P11 | 32 | 2 tasks | 7 files |
| Phase 14-availability-backend-absences-capacity P14-10 | 11min | 2 tasks | 3 files |
| Phase 15-ux-foundation-design-tokens-shared-components P15-01 | 23min | 3 tasks | 10 files |
| Phase 15-ux-foundation-design-tokens-shared-components P15-03 | 25min | 3 tasks | 8 files |
| Phase 15-ux-foundation-design-tokens-shared-components PP15-02 | 31min | 3 tasks | 6 files |
| Phase 16 P01 | 25m | 8 tasks | 9 files |

## Session Continuity

Last session: 2026-08-24T21:54:07.104Z
Stopped at: Completed 16-01-PLAN.md
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
