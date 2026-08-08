# Hourglass Roadmap

## Milestones

- ✅ **v0.1 MVP Consolidation** — Phases 0-10 (shipped 2026-08-01)
- 🚧 **v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction** — Phases 11-26 (in progress)

## Phases

<details>
<summary>✅ v0.1 MVP Consolidation (Phases 0-10) — SHIPPED 2026-08-01</summary>

- [x] Phase 0: Testing foundation (6/6 plans) — testcontainers-go, auth bug fixes, PG-backed test rewrite, E2E verification
- [x] Phase 1: Authorization (3/3 plans) — backend auth fixes, frontend auth integration, E2E verified
- [x] Phase 2: Org Hierarchy (3/3 plans) — ReactFlow tree, unit CRUD, member management, edge-driven reparenting, delete protection
- [x] Phase 3: Customers (3/3 plans) — full CRUD, internal customers, search, delete protection
- [x] Phase 4: Contracts (2/2 plans) — customer combobox, HasProjects delete guard
- [x] Phase 5: Projects (4/4 plans) — Update/Delete/ListSubprojects, EditProjectDialog — superseded by activities (Phase 9)
- [x] Phase 6: Time Entries + Expenses (5/5 plans) — flat model, calendar UI, two-stage approval workflow
- [x] Phase 7: Exports (3/3 plans) — CSV/XLSX, count endpoints, export tabs
- [x] Phase 8: Pre-Deployment Hardening (4/4 plans) — P0 audit gate closed, refresh-token reuse detection, list views, error boundaries
- [x] Phase 9: Activity Ontology (8/8 plans) — projects/subprojects → recursive activities, routing rewrite, staffing schema
- [x] Phase 10: Information Architecture (6/6 plans) — activity surface, pillar sidebar, page shell, Today landing, Approvals queue, Working Groups

</details>

### 🚧 v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction (In Progress)

**Milestone Goal:** Extend the activity ontology into the three-plane model (direction → facts → coverage): tickets as the second capture layer with triage, origins on activities, coverage allocations with funding sources, and the direction plan plane — then surface them prototype-driven, and finish with per-page polish folding v0.1 UAT debt.

**Build order (research Part 15, agreed):** Foundations (schema + origins + tickets) → Coverage backend → Direction backend → Availability (kept from original scope) → UX Foundation → UI-last prototype-driven surfaces → trailing per-page polish. Backend planes land before any new UI so prototype sessions run against the complete backend with real data. ADRs are drafted as phases land (P-003 rev + P-013 in Phase 11, P-012 + BE in Phase 12, P-015 + BE in Phase 13); ADR-P-012 draft already exists in the vault.

- [x] **Phase 11: Foundations** - Schema + origins + tickets backend: activity origin refs, sold_hours, ticket lifecycle with triage + dismissal guard; ADR-P-003 rev + P-013 (completed 2026-08-07)
- [ ] **Phase 12: Coverage Backend** - Allocation ledger, funding sources, to-cover queue, proposals-on-read, one-step confirm, snapshot mechanics; ADR-P-012 + BE encoding
- [ ] **Phase 13: Direction Backend** - Plan plane: direction entity, lifecycle, claim model, org policy, coverage read-model; ADR-P-015 + BE encoding
- [ ] **Phase 14: Availability Backend** - Absence declare/confirm/reject + capacity queries over availability_windows
- [ ] **Phase 15: UX Foundation** - Design tokens + shared components frozen; sketch loop contract established
- [ ] **Phase 16: Availability Frontend** - Absence calendars + capacity grid in People pillar
- [ ] **Phase 17: Coverage Surfaces** - Week-1 allocation screen, to-cover queue, own-coverage, buckets, per-unit report (4a+4b)
- [ ] **Phase 18: Today + Tickets Surfaces** - Today both shapes + tickets surface in Track/Today (4c)
- [ ] **Phase 19: Direction Surfaces** - Scheduler calendar + direction queue + coverage read-model (4d)
- [ ] **Phase 20: Today Polish** - Sketch-driven polish of Today landing, folding 10-UAT + 10-VERIFICATION
- [ ] **Phase 21: Track Polish (Time Entries + Expenses)** - Sketch-driven polish, folding 06-UAT (14 scenarios)
- [ ] **Phase 22: Activities Polish** - Sketch-driven polish, folding 09-UAT + 09-VERIFICATION
- [ ] **Phase 23: Approvals + Working Groups Polish** - Sketch-driven polish, folding 10-UAT scenarios
- [ ] **Phase 24: Customers + Contracts Polish** - Sketch-driven polish, folding 08-UAT + 08-VERIFICATION
- [ ] **Phase 25: Exports + People/Org/Admin Polish** - Sketch-driven polish of tail pages
- [ ] **Phase 26: Auth Pages Polish** - Sketch-driven polish of login/register/reset

## Phase Details

### Phase 11: Foundations — Schema + Origins + Tickets Backend

**Goal**: The three-plane ontology takes its first shape server-side: activities carry origin (type + reference set, FND-01/02/04), contracts carry sold_hours (FND-03), and the ticket entity exists with lifecycle + triage + dismissal guard + immutable event stream (TICK-01..05). ADR-P-003 revision and ADR-P-013 drafted and recorded.
**Depends on**: Nothing (first phase of v0.2)
**Requirements**: FND-01, FND-02, FND-03, FND-04, TICK-01, TICK-02, TICK-03, TICK-04, TICK-05
**Success Criteria** (what must be TRUE):

  1. An activity created via the API carries an origin (type + reference set per D-D: assigned_by/assigned_to, proposed_by/reviewed_by, ticket_id); refs are set once at creation and immutable (FND-01)
  2. An employee can propose an activity; the proposal routes through the activity's approval routing (unit manager for internal/personal, anchored WG manager for contract-linked) and lifecycle events land in activity state/audit — never in origin (FND-02)
  3. Contracts expose `sold_hours` read/write; the field is recorded for V5 mining (FND-03)
  4. Ticket CRUD + lifecycle work: open → triage → planned → in_progress → resolved → closed + reopen (resolved → in_progress, requires linked activities terminal); kinds question/bug/change/evolution are a closed set; resolved blocks on non-terminal activities (TICK-01, TICK-02)
  5. Triage converts a ticket into 1..N activities — the only linkage is ticket → activity; entries reference activities, never tickets (TICK-03)
  6. `triage → dismissed` is rejected while any linked activity has logged hours (net of compensations); dismissed tickets carry the "dismissed with N h logged" note (TICK-04)
  7. Ticket events (status changes, comments, resolution notes) are append-only via the BE-012 audit trail — no update/delete endpoints exist (TICK-05)
  8. Origin refs stored on activities; empty refs fall back to the first direction record on read (additive, Phase 13 landing) (FND-04)
  9. ADR-P-003 revision + ADR-P-013 recorded in the vault decisions folder; all migrations are append-only per ADR-BE-004 with up/down pairs + cycle tests

**Plans**: 8/8 plans executed
Plans:
**Wave 1**

- [x] 11-01-PLAN.md — Migration foundation: fix red cycle tests 011/012 + teardown list; migrations 014-017 (tickets, origins, sold_hours, audit_logs) + cycle tests
- [x] 11-03-PLAN.md — BE-014 routing extraction to shared package (proposal approval parity, D-G)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 11-02-PLAN.md — ADRs: ADR-P-003 revision, ADR-P-013 (origins), ADR-BE-016 (schema encoding) + index updates
- [x] 11-04-PLAN.md — Contract sold_hours read/write (FND-03): domain + validation + repo + handler

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 11-05-PLAN.md — Activity origins (FND-01/04): role gates, same-org validation, immutability; proposal approval (FND-02); ticket/audit domain + repos foundation

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 11-06-PLAN.md — Ticket lifecycle backend (TICK-01..05): state machine, atomic triage, dismissal guard, comments, append-only history

**Wave 5 (gap closure)** *(blocked on Wave 4 completion)*

- [x] 11-07-PLAN.md — CR-01 TOCTOU fix (TICK-02/TICK-04): matrix + hours Σ re-validated inside the mutator tx under FOR UPDATE locks + race test battery
- [x] 11-08-PLAN.md — Dismissal note rendered on read (TICK-04/IN-02) + title validation ride-alongs (WR-04/IN-01)

### Phase 12: Coverage Backend — The Allocation Loop

**Goal**: The coverage plane works server-side: funding sources, per-entry coverage allocations with the Σ invariant, to-cover queue, proposals computed on read, one-step manager confirmation, and period-close snapshots (COV-01..05). ADR-P-012 accepted; BE encoding ADR written (incl. D-K polymorphic validation cost).
**Depends on**: Phase 11 (activities + entries settled, origin refs live)
**Requirements**: COV-01, COV-02, COV-03, COV-04, COV-05
**Success Criteria** (what must be TRUE):

  1. Approved time entries can receive 1..N coverage allocations; the API rejects any state where Σ allocations ≠ entry hours (COV-01)
  2. All five funding sources work: contract budget (default for billable), support bucket (hours, carry-over, no expiry, overlapping buckets), service request (zero-value contract), internal absorption (mandatory reason WarrantyBug/UnderEstimate/Goodwill + beneficiary unit), cross-project transfer (explicit justification) (COV-02)
  3. Allocation proposals are computed on read from entry + activity chain + funding config — no proposal table exists; only confirmed allocations are persisted (COV-03)
  4. A single manager confirmation suffices (no finance chain); every allocation change is audit-logged via BE-012 (COV-03)
  5. Uncovered entries are queryable through the to-cover queue read-model; allocations remain editable indefinitely (COV-01, COV-04)
  6. Period close generates a reporting snapshot (billing, bucket levels, per-unit report) from either a frozen snapshot or as-of-close audit replay (F) — a reported period never changes retroactively; no lock on allocations (COV-04)
  7. Coverage references a polymorphic entry (`entry_type` + `entry_id`); validation rejects non-`time` types in v0.2 (COV-05)
  8. Beneficiary unit is nullable on activities, inherited downward like contract_id; absorption sources default from it (COV-05)

**Plans**: 4/7 plans executed
Plans:
**Wave 1**

- [x] 12-01-PLAN.md — Migrations 018-020 (beneficiary unit, allocation ledger, snapshots) + teardown + cycle tests (3VL + mandatory-field CHECK assertions)
- [x] 12-02-PLAN.md — ADRs: ADR-P-012 accepted + ADR-BE-017 coverage encoding (D-K cost, pinned OQ resolutions) + index updates
- [x] 12-04-PLAN.md — Coverage domain + ports.CoverageRepository + MockCoverageRepo (contracts all plans compile against)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 12-03-PLAN.md — Beneficiary unit vertical (COV-05, phase tracer): activity field + GetAncestry/scan/Update + ResolveBeneficiaryUnit + ResolveFundingContext CTEs + service/handler + tests
- [ ] 12-06-PLAN.md — Coverage repo: replace-set tx (FOR UPDATE + in-tx Σ + audit), queue, bucket balance, close snapshot, history + CR-01 concurrency battery

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 12-05-PLAN.md — Coverage service: D-04 proposal decision fn, replace-set (Σ + D-08 gate + D-K), queue enrichment, close orchestration + unit tests

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 12-07-PLAN.md — Coverage HTTP surface: 8 routes + sentinel map + permission-matrix tests + cmd/server wiring

### Phase 13: Direction Backend — The Plan Plane

**Goal**: The third plane lands: direction entity with per-day storage and derived modes, lifecycle with supersede chaining, WG claim model, org-configurable planning policy, and the direction-coverage read-model (DIR-01..06). ADR-P-015 + BE encoding drafted.
**Depends on**: Phase 12 (coverage terminology settled — "direction" vs "coverage allocation" convention)
**Requirements**: DIR-01, DIR-02, DIR-03, DIR-04, DIR-05, DIR-06
**Success Criteria** (what must be TRUE):

  1. Direction rows exist per (employee, activity, day, est_hours) — per-day storage always; multiple rows may share a day; no intra-day ordering; mode derived (planned_date set → scheduled, null → queued with priority + due_date) (DIR-01)
  2. Self-direction is first-class (`directed_by == directed_to`, no approval); managers direct within subtree/WG reach via BE-014 machinery (DIR-01)
  3. Lifecycle draft → active → superseded/cancelled works; done/lapsed/claimed are derived, never stored; supersedes_id chains replanning with audit-first via BE-012 (DIR-02)
  4. WG-direction rows are queued-only; a member's claim creates a user-targeted row via origin_direction_id; claimed is derived (DIR-03)
  5. Org policy is configurable: deadline date, horizon (day/week/month), per-employee mode (manager-planned vs self-planned); soft-policy (block vs nag) configurable for UI (DIR-04)
  6. Scheduler read path consumes P-008 absence windows + employment validity and returns plan-time warnings; never blocks (DIR-05)
  7. Direction-coverage read-model returns planned hours vs capacity per employee/period with uncovered days surfaced, per employee / unit / WG (DIR-06)
  8. Origin fallback active: activities with empty origin refs resolve refs from the first direction record (FND-04 read path, additive)

**Plans**: TBD

### Phase 14: Availability Backend — Absences + Capacity

**Goal**: Absence lifecycle works server-side over the shipped availability_windows schema (declare → confirm/reject, HR medical curation), plus derived capacity queries (weekly hours − confirmed absences) with workload from submitted+approved entries (AVAIL-01/02, supports AVAIL-04 and Phase 13's DIR-05).
**Depends on**: Phase 11 (sequential order; no technical dependency)
**Requirements**: AVAIL-01, AVAIL-02
**Success Criteria** (what must be TRUE):

  1. Employee can declare an absence with a type and date range via the API; invalid or overlapping windows are rejected (AVAIL-01)
  2. Manager/HR can confirm or reject declared absences via the API — only declared windows are confirmable, rejects carry a reason, HR curates medical absences with certificate_ref (AVAIL-02)
  3. API returns capacity per activity/WG = weekly hours − confirmed absences, with workload from submitted+approved entries on the activity subtree (supports AVAIL-04 in Phase 16)
  4. Absence windows are consumable by the direction scheduler read path (DIR-05 dependency)

**Plans**: TBD

### Phase 15: UX Foundation — Design Tokens + Shared Components

**Goal**: The design system is frozen before any surface work: semantic status/state tokens in index.css and a shared component set that every new and polished page consumes; the sketch loop contract is established for all surface/polish phases (UXFD-01, UXFD-02).
**Depends on**: Nothing technical (can run parallel to Phase 14; UI-only)
**Requirements**: UXFD-01, UXFD-02
**Success Criteria** (what must be TRUE):

  1. Every status/state color used by ≥2 pages renders from a semantic token in index.css; no surface/polish phase introduces ad-hoc hex values (UXFD-01)
  2. The frozen shared component set (PageHeader, FilterBar, DataTable, StatusBadge variants, EmptyState, ConfirmDialog) exists and new surfaces (allocation screen, tickets, scheduler) use it from day one (UXFD-01)
  3. Each surface/polish phase follows the sketch loop: 2–3 gsd-sketch options shown, user agrees on one, UI-only plan, ≤3 sketch rounds (UXFD-02)
  4. Sidebar collapsed-mode quick task triaged here (quick_task 260801-got-investigate-sidebar)

**Plans**: TBD
**UI hint**: yes

### Phase 16: Availability Frontend

**Goal**: Availability and capacity are usable surfaces in the People pillar: absence request/confirm UI, personal + team/org calendars, and a custom date-fns/Tailwind capacity grid per activity/WG (AVAIL-03/04/05).
**Depends on**: Phase 14 (backend), Phase 15 (tokens)
**Requirements**: AVAIL-03, AVAIL-04, AVAIL-05
**Success Criteria** (what must be TRUE):

  1. Employee can request an absence from the UI and view personal + team/org absence calendars (AVAIL-03)
  2. Manager/HR can confirm or reject absences from the UI with distinct status badges (declared → confirmed/rejected) (UI half of AVAIL-02)
  3. Manager can view capacity per activity/WG as a capacity-vs-workload grid (custom date-fns + Tailwind, not a calendar library) (AVAIL-04)
  4. Availability and capacity entries appear in the People pillar sidebar with role-scoped visibility (AVAIL-05)

**Plans**: TBD
**UI hint**: yes

### Phase 17: Coverage Surfaces — Allocation Screen + Buckets + Reports (4a+4b)

**Goal**: The coverage plane is usable: week-1 allocation screen with read-computed proposals (Review group), to-cover queue, employee own-coverage read-only view, bucket setup + balance (Economics → Contracts), and the per-unit non-billed report (Reports) (SURF-01..05). D-O IA leans validated here — P-011 revision accumulates.
**Depends on**: Phase 12 (backend), Phase 15 (tokens)
**Requirements**: SURF-01, SURF-02, SURF-03, SURF-04, SURF-05
**Success Criteria** (what must be TRUE):

  1. Manager opens the week-1 screen and sees allocation proposals computed on read; one click confirms the default; exceptions (split/warranty/transfer) carry mandatory reasons (SURF-01)
  2. To-cover queue renders uncovered entries as an explicit state; soft mid-month target nudges, never blocks (SURF-02)
  3. Employee sees own coverage (billed vs absorbed) read-only on own entries (SURF-03)
  4. Bucket setup + balance visible under Economics → Contracts; overlapping buckets allowed; balance carries over periods (SURF-04)
  5. Per-unit non-billed report (resoconto) renders in Reports incl. warranty/goodwill cost per customer (SURF-05)
  6. Allocation work lives in the Review group per the D-O lean — validated or revised during prototyping

**Plans**: TBD
**UI hint**: yes

### Phase 18: Today + Tickets Surfaces (4c)

**Goal**: Today view is prototyped in both shapes — v0.2-launch (tickets + assigned activities) and with-direction (plan + queue) — and the tickets surface lands in Track + Today ("my open tickets" per P-004) (SURF-06, TICK-06). P-011 IA reserves the direction slot without shipping it.
**Depends on**: Phase 13 (direction backend for the with-direction shape), Phase 15 (tokens)
**Requirements**: SURF-06, TICK-06
**Success Criteria** (what must be TRUE):

  1. Today renders the v0.2-launch shape: tickets + assigned activities composition per P-004 rules; locked empty states preserved (SURF-06)
  2. Today's with-direction prototype shows plan + queue per the D-O lean; direction slot reserved in IA, not shipped (SURF-06)
  3. Tickets surface in the Track pillar: list with filter/sort by kind/status, detail with immutable event timeline, create/triage/transition dialogs (TICK-06)
  4. Today shows "my open tickets by priority" block (TICK-06)
  5. Tickets are auto-approved tracked work; permission control enforced, no approval chain (TICK-06)

**Plans**: TBD
**UI hint**: yes

### Phase 19: Direction Surfaces (4d)

**Goal**: The plan plane is usable: scheduler calendar (drag & drop, P-008 absence warnings), direction queue, and the direction-coverage read-model surfacing uncovered capacity (SURF-07, SURF-08).
**Depends on**: Phase 13 (backend), Phase 15 (tokens), Phase 16 (P-008 absences available for warnings)
**Requirements**: SURF-07, SURF-08
**Success Criteria** (what must be TRUE):

  1. Manager spreads activities across employee-days via a calendar surface; the same surface serves self-planned mode (SURF-07)
  2. Scheduler shows absence warnings ("away 10–21 Aug") at plan time from P-008 windows; warnings never block (SURF-07)
  3. Direction queue renders queued rows with priority + due_date, incl. WG rows and claim actions (SURF-08)
  4. Direction-coverage view shows planned vs capacity per employee/period with uncovered days visible (per employee / unit / WG) (SURF-08)
  5. Org policy (deadline/horizon/mode) configurable from the UI per D-X; soft-policy block-vs-nag decision made here during prototyping

**Plans**: TBD
**UI hint**: yes

### Phase 20: Today Polish

**Goal**: The Today landing is polished through the sketch loop and its v0.1 UAT debt is closed.
**Depends on**: Phase 18 (Today gained ticket + direction shapes), Phase 15 tokens
**Requirements**: POLS-01
**Success Criteria** (what must be TRUE):

  1. Today's sections render consistently on frozen tokens/components per the agreed sketch (POLS-01)
  2. Today's 10-UAT scenarios pass verification and 10-VERIFICATION human review items are addressed (POLS-01)
  3. Today remains a read-only composition with locked empty states for new/caught-up users (POLS-01)

**Plans**: TBD
**UI hint**: yes

### Phase 21: Track Polish — Time Entries + Expenses

**Goal**: Time entries and expenses pages are polished through the sketch loop; the 14-scenario 06-UAT debt folds in here.
**Depends on**: Phase 20, Phase 15 tokens (entry pickers/status badges reflect settled models)
**Requirements**: POLS-02, POLS-03
**Success Criteria** (what must be TRUE):

  1. Time entries page passes its 06-UAT scenarios (list/calendar/export tabs, entry CRUD, status badges) (POLS-02)
  2. Expenses page passes its 06-UAT scenarios (receipts, approval workflow UI, status badges) (POLS-03)
  3. Both pages use frozen tokens/components per the agreed sketch; changes are UI-only — no new API endpoints (POLS-02, POLS-03)

**Plans**: TBD
**UI hint**: yes

### Phase 22: Activities Polish

**Goal**: Activities pages are polished through the sketch loop; 09-UAT + 09-VERIFICATION debt folds in here.
**Depends on**: Phase 21, Phase 15 tokens
**Requirements**: POLS-06
**Success Criteria** (what must be TRUE):

  1. Activities pages (index, detail, create/edit dialogs) pass their 09-UAT scenario and 09-VERIFICATION human review items (POLS-06)
  2. Activity tree, derived commercial context, origin display, and billability display polished per the agreed sketch (POLS-06)

**Plans**: TBD
**UI hint**: yes

### Phase 23: Approvals + Working Groups Polish

**Goal**: Approvals queue and Working Groups surfaces are polished through the sketch loop; their 10-UAT scenarios fold in here.
**Depends on**: Phase 22, Phase 15 tokens
**Requirements**: POLS-04, POLS-05
**Success Criteria** (what must be TRUE):

  1. Approvals queue passes its 10-UAT scenarios (stage tabs, approve/reject, reason-required reject) (POLS-04)
  2. Working Groups surface passes its 10-UAT scenarios (list/search/create/edit/members) (POLS-05)
  3. Remaining 10-VERIFICATION items for approvals/WG are addressed (POLS-04, POLS-05)

**Plans**: TBD
**UI hint**: yes

### Phase 24: Customers + Contracts Polish

**Goal**: Customers and Contracts pages are polished through the sketch loop; 08-UAT + 08-VERIFICATION debt folds in here.
**Depends on**: Phase 23, Phase 15 tokens
**Requirements**: POLS-07, POLS-08
**Success Criteria** (what must be TRUE):

  1. Customers pages (index, detail) pass their 08-UAT scenarios and 08-VERIFICATION human review items (POLS-07)
  2. Contracts pages polished per the agreed sketch (detail, dialogs, sold_hours display, export tabs) (POLS-08)

**Plans**: TBD
**UI hint**: yes

### Phase 25: Exports + People/Org/Admin Polish

**Goal**: The tail pages — Exports and People/org tree + Admin — are polished through the sketch loop.
**Depends on**: Phase 24, Phase 15 tokens
**Requirements**: POLS-09, POLS-10
**Success Criteria** (what must be TRUE):

  1. Exports page + export tabs polished per the agreed sketch (format switch, date ranges, download states) (POLS-09)
  2. People/org tree (ReactFlow) and Admin surfaces polished (member management, primary designation, role display) (POLS-10)

**Plans**: TBD
**UI hint**: yes

### Phase 26: Auth Pages Polish

**Goal**: Login, register, and password-reset pages are polished through the sketch loop.
**Depends on**: Phase 25, Phase 15 tokens
**Requirements**: POLS-11
**Success Criteria** (what must be TRUE):

  1. Login/register/reset pages render on frozen tokens with consistent validation and error states (POLS-11)
  2. Password reset flow (request → code → new password) verifiable end-to-end from the polished UI (POLS-11)

**Plans**: TBD
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 0. Testing | v0.1 | 6/6 | Complete | 2026-06-09 |
| 1. Authorization | v0.1 | 3/3 | Complete | 2026-06-09 |
| 2. Org Hierarchy | v0.1 | 3/3 | Complete | 2026-06-11 |
| 3. Customers | v0.1 | 3/3 | Complete | 2026-06-11 |
| 4. Contracts | v0.1 | 2/2 | Complete | 2026-06-12 |
| 5. Projects | v0.1 | 4/4 | Complete | 2026-06-12 |
| 6. Time+Expenses | v0.1 | 5/5 | Complete | 2026-06-14 |
| 7. Exports | v0.1 | 3/3 | Complete | 2026-06-14 |
| 8. Hardening | v0.1 | 4/4 | Complete | 2026-07-31 |
| 9. Activity Ont. | v0.1 | 8/8 | Complete | 2026-07-31 |
| 10. IA Impl. | v0.1 | 6/6 | Complete | 2026-08-01 |
| 11. Foundations | v0.2 | 8/8 | Complete    | 2026-08-07 |
| 12. Coverage Backend | v0.2 | 4/7 | In Progress|  |
| 13. Direction Backend | v0.2 | 0/TBD | Not started | - |
| 14. Availability Backend | v0.2 | 0/TBD | Not started | - |
| 15. UX Foundation | v0.2 | 0/TBD | Not started | - |
| 16. Availability FE | v0.2 | 0/TBD | Not started | - |
| 17. Coverage Surfaces | v0.2 | 0/TBD | Not started | - |
| 18. Today+Tickets Surfaces | v0.2 | 0/TBD | Not started | - |
| 19. Direction Surfaces | v0.2 | 0/TBD | Not started | - |
| 20. Today Polish | v0.2 | 0/TBD | Not started | - |
| 21. Track Polish | v0.2 | 0/TBD | Not started | - |
| 22. Activities Polish | v0.2 | 0/TBD | Not started | - |
| 23. Approvals+WG Polish | v0.2 | 0/TBD | Not started | - |
| 24. Customers+Contracts Polish | v0.2 | 0/TBD | Not started | - |
| 25. Exports+People Polish | v0.2 | 0/TBD | Not started | - |
| 26. Auth Polish | v0.2 | 0/TBD | Not started | - |

*Full v0.1 phase details: [milestones/v0.1-ROADMAP.md](milestones/v0.1-ROADMAP.md)*
*Requirements: [REQUIREMENTS.md](REQUIREMENTS.md)*
