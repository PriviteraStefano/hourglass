# Hourglass Roadmap

## Milestones

- ✅ **v0.1 MVP Consolidation** — Phases 0-10 (shipped 2026-08-01)
- 🚧 **v0.2 UX Polish + Tickets + Availability** — Phases 11-22 (in progress)

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

### 🚧 v0.2 UX Polish + Tickets + Availability (In Progress)

**Milestone Goal:** Polish the product page by page through sketch-driven UX/UI work, add a ticket ontology (internal tasks + customer helpdesk), and surface availability (absences + resource views) — folding in all v0.1 UAT debt per page.

**Build order:** UX foundation (tokens) → ticket backend → ticket frontend → staffing backend → availability frontend → per-page polish. Each polish phase is sketch-driven (2–3 gsd-sketch options → user agrees → implement → verify), UI-only, and folds in that page's v0.1 UAT/verification debt.

- [ ] **Phase 11: UX Foundation** - Design tokens + shared components frozen before any page work; sketch loop contract established
- [ ] **Phase 12: Ticket Ontology Backend** - Migration 014, ticket domain/service/repo, counts endpoint, auto-approve + assignee ADR decisions
- [ ] **Phase 13: Ticket Frontend + Today Composition** - Tickets in Track pillar, event timeline, customer request counts + export, Today ticket block
- [ ] **Phase 14: Staffing Backend** - Absence declare/confirm/reject over availability_windows + capacity queries
- [ ] **Phase 15: Availability + Capacity Frontend** - Absence calendars + capacity grid in People pillar
- [ ] **Phase 16: Today Polish** - Sketch-driven polish of Today landing, folding 10-UAT + 10-VERIFICATION
- [ ] **Phase 17: Track Polish (Time Entries + Expenses)** - Sketch-driven polish, folding 06-UAT (14 scenarios)
- [ ] **Phase 18: Activities Polish** - Sketch-driven polish, folding 09-UAT + 09-VERIFICATION
- [ ] **Phase 19: Approvals + Working Groups Polish** - Sketch-driven polish, folding 10-UAT scenarios
- [ ] **Phase 20: Customers + Contracts Polish** - Sketch-driven polish, folding 08-UAT + 08-VERIFICATION
- [ ] **Phase 21: Exports + People/Org/Admin Polish** - Sketch-driven polish of tail pages
- [ ] **Phase 22: Auth Pages Polish** - Sketch-driven polish of login/register/reset

## Phase Details

### Phase 11: UX Foundation — Design Tokens + Shared Components
**Goal**: The design system is frozen before any page work: semantic status/state tokens in index.css and a shared component set that every new and polished page consumes; the sketch loop contract is established for all page phases.
**Depends on**: Nothing (first phase of v0.2)
**Requirements**: UXFD-01, UXFD-02
**Success Criteria** (what must be TRUE):
  1. Every status/state color used by ≥2 pages renders from a semantic token in index.css; no page phase introduces ad-hoc hex values (UXFD-01)
  2. The frozen shared component set (PageHeader, FilterBar, DataTable, StatusBadge variants, EmptyState, ConfirmDialog) exists and new ticket/availability pages use it from day one (UXFD-01)
  3. A user sees identical status colors and components across all pages — no two pages render the same status differently (UXFD-01)
  4. Each page polish phase follows the sketch loop: 2–3 gsd-sketch options shown, user agrees on one, UI-only plan, ≤3 sketch rounds (UXFD-02)
**Plans**: TBD
**UI hint**: yes

### Phase 12: Ticket Ontology Backend
**Goal**: The unified ticket entity exists server-side — migration 014 (tickets + ticket_events), ticket domain with per-kind transition matrix, service, repo with derived-customer CTE, and the /tickets/counts endpoint — with TICK-05 assignee rules and TICK-06 auto-approve semantics locked in an ADR.
**Depends on**: Phase 11 (sequential order; no technical dependency)
**Requirements**: TICK-01, TICK-03, TICK-04, TICK-05, TICK-06
**Success Criteria** (what must be TRUE):
  1. User can create a ticket on any activity via the API — kinds task/helpdesk, priority, assignee, helpdesk customer requester — and invalid activities or missing commercial context are rejected with clear errors (TICK-01; backend half of TICK-02)
  2. Ticket status transitions follow the per-kind matrix: invalid transitions are rejected, one shared status vocabulary with customer-facing label projection (TICK-03)
  3. Ticket history is an immutable event stream (comments, resolution notes, status changes) that cannot be edited or deleted (TICK-04)
  4. Manager can assign/unassign tickets to WG members with assignee rules enforced; tickets are auto-approved tracked work with manager-intervention semantics decided and documented in an ADR (TICK-05, TICK-06)
  5. API returns per-customer request counts (received/resolved) under the creation-month counting rule (supports TICK-07 in Phase 13)
**Plans**: TBD

### Phase 13: Ticket Frontend + Today Composition
**Goal**: Users work tickets in the app: Track pillar list with filter/sort, detail with event timeline, create/assign/transition dialogs, in-app customer helpdesk surface, per-customer request counts with export, and Today's "my open tickets" block.
**Depends on**: Phase 12
**Requirements**: TICK-02, TICK-07, TICK-08
**Success Criteria** (what must be TRUE):
  1. User can open the Tickets surface under the Track pillar (sidebar entry via navStructure), filter/sort the list, and open a detail showing the full event timeline (TICK-08)
  2. Customer (external + internal) can open a helpdesk ticket in-app and see a minimal customer-facing status projection — no public portal (TICK-02)
  3. User can view per-customer request counts (received/resolved) on the customer detail page and export them (TICK-07)
  4. User can create, assign, and transition tickets from dialogs against the Phase 12 API (UI half of TICK-01/TICK-03/TICK-05)
  5. Today page shows "my open tickets by priority" block per P-004 rules (TICK-08)
**Plans**: TBD
**UI hint**: yes

### Phase 14: Staffing Backend — Availability + Capacity
**Goal**: Absence request lifecycle works server-side over the shipped availability_windows schema (declare → confirm/reject, HR medical curation), plus derived capacity queries (weekly hours − confirmed absences) with workload from submitted+approved entries.
**Depends on**: Phase 12 (sequential order; no technical dependency)
**Requirements**: AVAIL-01, AVAIL-02
**Success Criteria** (what must be TRUE):
  1. Employee can declare an absence with a type and date range via the API; invalid or overlapping windows are rejected (AVAIL-01)
  2. Manager/HR can confirm or reject declared absences via the API — only declared windows are confirmable, rejects carry a reason, HR curates medical absences with certificate_ref (AVAIL-02)
  3. API returns capacity per activity/WG = weekly hours − confirmed absences, with workload from submitted+approved entries on the activity subtree (supports AVAIL-04 in Phase 15)
**Plans**: TBD

### Phase 15: Availability + Capacity Frontend
**Goal**: Availability and capacity are usable surfaces in the People pillar: absence request/confirm UI, personal + team/org calendars, and a custom date-fns/Tailwind capacity grid per activity/WG.
**Depends on**: Phase 14
**Requirements**: AVAIL-03, AVAIL-04, AVAIL-05
**Success Criteria** (what must be TRUE):
  1. Employee can request an absence from the UI and view personal + team/org absence calendars (AVAIL-03)
  2. Manager/HR can confirm or reject absences from the UI with distinct status badges (declared → confirmed/rejected) (UI half of AVAIL-02)
  3. Manager can view capacity per activity/WG as a capacity-vs-workload grid (custom date-fns + Tailwind, not a calendar library) (AVAIL-04)
  4. Availability and capacity entries appear in the People pillar sidebar with role-scoped visibility (AVAIL-05)
  5. Non-blocking assignment-time warnings show on WG surfaces (extends AVAIL-04)
**Plans**: TBD
**UI hint**: yes

### Phase 16: Today Polish
**Goal**: The Today landing is polished through the sketch loop and its v0.1 UAT debt is closed.
**Depends on**: Phase 13 (Today gained the ticket block), Phase 11 tokens
**Requirements**: POLS-01
**Success Criteria** (what must be TRUE):
  1. Today's sections ("Waiting on you", "Your week", "My open tickets") render consistently on frozen tokens/components per the agreed sketch (POLS-01)
  2. Today's 10-UAT scenarios pass verification and 10-VERIFICATION human review items are addressed (POLS-01)
  3. Today remains a read-only composition with locked empty states for new/caught-up users (POLS-01)
**Plans**: TBD
**UI hint**: yes

### Phase 17: Track Polish — Time Entries + Expenses
**Goal**: Time entries and expenses pages are polished through the sketch loop; the 14-scenario 06-UAT debt folds in here.
**Depends on**: Phase 15 (entry pickers/status badges reflect settled models), Phase 11 tokens
**Requirements**: POLS-02, POLS-03
**Success Criteria** (what must be TRUE):
  1. Time entries page passes its 06-UAT scenarios (list/calendar/export tabs, entry CRUD, status badges) (POLS-02)
  2. Expenses page passes its 06-UAT scenarios (receipts, approval workflow UI, status badges) (POLS-03)
  3. Both pages use frozen tokens/components per the agreed sketch; changes are UI-only — no new API endpoints (POLS-02, POLS-03)
**Plans**: TBD
**UI hint**: yes

### Phase 18: Activities Polish
**Goal**: Activities pages are polished through the sketch loop; 09-UAT + 09-VERIFICATION debt folds in here.
**Depends on**: Phase 17, Phase 11 tokens
**Requirements**: POLS-06
**Success Criteria** (what must be TRUE):
  1. Activities pages (index, detail, create/edit dialogs) pass their 09-UAT scenario and 09-VERIFICATION human review items (POLS-06)
  2. Activity tree, derived commercial context, and billability display polished per the agreed sketch (POLS-06)
**Plans**: TBD
**UI hint**: yes

### Phase 19: Approvals + Working Groups Polish
**Goal**: Approvals queue and Working Groups surfaces are polished through the sketch loop; their 10-UAT scenarios fold in here.
**Depends on**: Phase 18, Phase 11 tokens
**Requirements**: POLS-04, POLS-05
**Success Criteria** (what must be TRUE):
  1. Approvals queue passes its 10-UAT scenarios (stage tabs, approve/reject, reason-required reject) (POLS-04)
  2. Working Groups surface passes its 10-UAT scenarios (list/search/create/edit/members) (POLS-05)
  3. Remaining 10-VERIFICATION items for approvals/WG are addressed (POLS-04, POLS-05)
**Plans**: TBD
**UI hint**: yes

### Phase 20: Customers + Contracts Polish
**Goal**: Customers and Contracts pages are polished through the sketch loop; 08-UAT + 08-VERIFICATION debt folds in here.
**Depends on**: Phase 19, Phase 11 tokens (customer detail gained request counts in Phase 13)
**Requirements**: POLS-07, POLS-08
**Success Criteria** (what must be TRUE):
  1. Customers pages (index, detail) pass their 08-UAT scenarios and 08-VERIFICATION human review items (POLS-07)
  2. Contracts pages polished per the agreed sketch (detail, dialogs, export tabs) (POLS-08)
  3. Customer detail's per-customer request counts section renders consistently with the ticket surface styling (POLS-07)
**Plans**: TBD
**UI hint**: yes

### Phase 21: Exports + People/Org/Admin Polish
**Goal**: The tail pages — Exports and People/org tree + Admin — are polished through the sketch loop.
**Depends on**: Phase 20, Phase 11 tokens
**Requirements**: POLS-09, POLS-10
**Success Criteria** (what must be TRUE):
  1. Exports page + export tabs polished per the agreed sketch (format switch, date ranges, download states) (POLS-09)
  2. People/org tree (ReactFlow) and Admin surfaces polished (member management, primary designation, role display) (POLS-10)
**Plans**: TBD
**UI hint**: yes

### Phase 22: Auth Pages Polish
**Goal**: Login, register, and password-reset pages are polished through the sketch loop.
**Depends on**: Phase 21, Phase 11 tokens
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
| 11. UX Foundation | v0.2 | 0/TBD | Not started | - |
| 12. Ticket Backend | v0.2 | 0/TBD | Not started | - |
| 13. Ticket Frontend | v0.2 | 0/TBD | Not started | - |
| 14. Staffing Backend | v0.2 | 0/TBD | Not started | - |
| 15. Avail+Capacity FE | v0.2 | 0/TBD | Not started | - |
| 16. Today Polish | v0.2 | 0/TBD | Not started | - |
| 17. Track Polish | v0.2 | 0/TBD | Not started | - |
| 18. Activities Polish | v0.2 | 0/TBD | Not started | - |
| 19. Approvals+WG Polish | v0.2 | 0/TBD | Not started | - |
| 20. Customers+Contracts Polish | v0.2 | 0/TBD | Not started | - |
| 21. Exports+People Polish | v0.2 | 0/TBD | Not started | - |
| 22. Auth Polish | v0.2 | 0/TBD | Not started | - |

*Full v0.1 phase details: [milestones/v0.1-ROADMAP.md](milestones/v0.1-ROADMAP.md)*
*Requirements: [REQUIREMENTS.md](REQUIREMENTS.md)*
