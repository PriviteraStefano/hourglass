# Requirements: Hourglass v0.2

**Defined:** 2026-08-02
**Core Value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

## v0.2 Requirements

Requirements for milestone v0.2 — Ontology Extension (Origins, Tickets & Coverage + Direction). Each maps to roadmap phases. Grounded in the ontology research note (D-A…D-AA, `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md`, Parts 12–15).

### Foundations / Origins

- [x] **FND-01**: Activity carries an origin (type + reference set) set once at creation: manager-assignment → `assigned_by` + `assigned_to`, employee-proposal → `proposed_by` + `reviewed_by`, customer-ticket → `ticket_id` (D-D)
- [ ] **FND-02**: Employee can propose an activity; proposal approval routes via the activity's approval routing (internal/personal → proposer's unit manager, contract-linked → anchored WG's manager); lifecycle events live in activity state/audit, not origin (D-G, D-D)
- [x] **FND-03**: Contract carries `sold_hours`; V5 mines sold vs Σ actual hours (D-N)
- [x] **FND-04**: Origin refs are stored directly on activities; when empty, the read path falls back to the first direction record — additive enhancement, no migration (R4 resolution, Part 15)

### Tickets

- [x] **TICK-01**: User can create an internal ticket with a closed-set kind (question · bug · change · evolution); kinds drive funding-eligibility rules (Q2, D-H)
- [x] **TICK-02**: Ticket lifecycle: open → triage → planned → in_progress → resolved → closed, with reopen (`resolved → in_progress`); `resolved` requires all linked activities terminal; one demand thread per real-world request (D-A)
- [ ] **TICK-03**: Triage classifies a ticket (kind + eligible funding sources) and converts it into 1..N activities; chain is ticket → activity → entries — tickets never reference entries directly (revised P-003)
- [x] **TICK-04**: Dismissal guard: `triage → dismissed` is blocked while any linked activity carries logged hours (net of compensating corrections); a dismissed ticket keeps a "dismissed with N h logged" note (D-M)
- [x] **TICK-05**: Ticket history is an immutable event stream (comments, resolution notes, status changes) via the BE-012 audit trail — never editable or deletable
- [ ] **TICK-06**: User can view tickets and their status (Track pillar + Today "my open tickets" per P-004); tickets are tracked work, auto-approved with permission control

### Coverage

- [ ] **COV-01**: Approved time entries are coverable by 1..N coverage allocations; Σ allocations = entry hours (invariant); uncovered hours land in the to-cover queue — a visible state, never an implicit gap (P-012)
- [ ] **COV-02**: Funding sources exist and work: contract budget (default for billable, D-7 rule), support bucket (hours-only, carry-over, no expiry, overlapping buckets allowed — D-P), service request (zero-value contract, D-J), internal absorption (mandatory reason: WarrantyBug · UnderEstimate · Goodwill), cross-project transfer (explicit justification) (P-014)
- [ ] **COV-03**: Manager confirms allocations in one step (no finance chain, D-L); every change is audit-logged (BE-012); proposals are computed on read, never stored (D-I)
- [ ] **COV-04**: Allocations stay editable indefinitely; period close produces a reporting snapshot (billing, bucket levels, per-unit report), never a lock (D-F); snapshot mechanics backend-only (F)
- [ ] **COV-05**: Activity carries a nullable beneficiary unit (inherited downward like `contract_id`); absorption funding sources default from it; coverage entries are polymorphic (`entry_type` + `entry_id`), `time` only allowed in v0.2 (D-B, D-K)

### Direction

- [ ] **DIR-01**: User (manager or self) can create direction rows: per-day storage always, `est_hours` per row, partial days first-class, no intra-day ordering; mode derived from `planned_date` (set → scheduled, null → queued with priority + `due_date`); self-direction (`directed_by == directed_to`) needs no approval (D-R, D-S, D-W, D-AA)
- [ ] **DIR-02**: Direction lifecycle: draft → active → superseded / cancelled; `done`, `lapsed`, `claimed` are derived, never stored; `supersedes_id` chains replanning; audit-first via BE-012 (D-V)
- [ ] **DIR-03**: WG-direction exists: queued-only rows target a WG; a member's claim creates a user-targeted row via `origin_direction_id`; "claimed" is derived (D-T)
- [ ] **DIR-04**: Org planning policy is org-configurable: deadline date, horizon (day/week/month), per-employee mode (manager-planned vs self-planned); soft policy (block vs nag) decided in UI prototyping (D-X)
- [ ] **DIR-05**: Scheduler reads P-008 absence windows + employment validity and warns at plan time ("away 10–21 Aug") — never blocks (D-Y)
- [ ] **DIR-06**: Direction-coverage read-model: planned hours per employee per period vs capacity, plus uncovered-day surfacing — per employee / unit / WG (D-Z)

### Availability

- [ ] **AVAIL-01**: Employee can declare an absence with a type and date range (existing `availability_windows` schema); invalid or overlapping windows rejected
- [ ] **AVAIL-02**: Manager/HR can confirm or reject absences (declared → confirmed/rejected); rejects carry a reason; HR curates medical absences with certificate_ref
- [ ] **AVAIL-03**: User can view an absence calendar (personal + team/org)
- [ ] **AVAIL-04**: Manager can view capacity per activity/WG (weekly hours − confirmed absences, with workload from submitted+approved entries)
- [ ] **AVAIL-05**: Availability surfaces in the People pillar with role-scoped visibility

### UX Foundation

- [ ] **UXFD-01**: Design token foundation extended (status palette, state colors) and shared component set frozen before any surface/polish work
- [ ] **UXFD-02**: Sketch-driven loop established: each surface/polish phase shows 2–3 gsd-sketch options, user agrees, UI-only plans, ≤3 sketch rounds

### Surfaces (prototype-driven, D-O leans)

- [ ] **SURF-01**: Manager sees the week-1 allocation screen with proposals computed on read; confirms or adjusts with mandatory reasons; allocation work sits in the Review group (4a, D-I, D-O)
- [ ] **SURF-02**: To-cover queue is visible — uncovered work is an explicit state; a soft mid-month target nudges the queue, never blocks (4a, D-F)
- [ ] **SURF-03**: Employee can view own coverage (billed vs absorbed) on own entries — read-only (4a, D-O)
- [ ] **SURF-04**: Bucket setup + balance visible under Economics → Contracts (4b, D-O)
- [ ] **SURF-05**: Per-unit non-billed cost report (the resoconto) in Reports, including warranty/goodwill costs per customer (4b, Part 9)
- [ ] **SURF-06**: Today view prototyped in both shapes — v0.2-launch (tickets + assigned activities) and with-direction (plan + queue); P-011 IA reserves the direction slot without shipping it (4c)
- [ ] **SURF-07**: Direction scheduler is a calendar view (drag & drop) reading P-008 absence windows with warnings; serves manager-planned and self-planned modes (4d, D-Y, D-S)
- [ ] **SURF-08**: Direction queue + direction-coverage read-model surfaced (uncovered capacity visible per employee/unit/WG) (4d, D-Z)

### Per-Page Polish (sketch-driven, each folds in its v0.1 UAT debt)

- [ ] **POLS-01**: Today landing polished
- [ ] **POLS-02**: Time entries (Track) polished
- [ ] **POLS-03**: Expenses polished
- [ ] **POLS-04**: Approvals queue polished
- [ ] **POLS-05**: Working Groups polished
- [ ] **POLS-06**: Activities polished
- [ ] **POLS-07**: Customers polished
- [ ] **POLS-08**: Contracts polished
- [ ] **POLS-09**: Exports polished
- [ ] **POLS-10**: People/org tree + Admin surfaces polished
- [ ] **POLS-11**: Auth pages (login/register/reset) polished

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Insight / Analytics (V5)

- **INSG-01**: Estimate accuracy per activity kind/customer (actual vs estimated) — mines `sold_hours` vs Σ actual hours from v0.2
- **INSG-02**: Plan-adherence analytics (aggregate-only, per-period — never per-day-per-person, D-U)
- **INSG-03**: Per-customer request counts (received/resolved) — billing story superseded by coverage allocations; reconsider when invoicing needs data

### Tickets (extended)

- **TICK-07**: External ticket intake port (hexagonal secondary adapter, D-E)
- **TICK-08**: SLA/deadline rules with breach indicator
- **TICK-09**: Email ingestion (ticket by email)
- **TICK-10**: Escalation chains

### Coverage (extended)

- **COV-06**: Expense coverage allocations (polymorphic entry is schema-ready; blocked until an expense-splitting need is demonstrated, D-K)
- **COV-07**: Smart/auto allocation proposals (P-005 data-maturity gate)
- **COV-08**: Full budget machinery — rates, money, per-activity estimates (ADR-P-010, V4)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Customer-facing ticket portal | Tickets are internal-only in v0.2 (D-E); external intake is a future port |
| Kanban / sub-task trees / comment threads on tickets | Tickets are demand tracking, not task execution (P-003 hard boundary) |
| Warranty certification flow | Warranty is declared at allocation time; the warranty-cost report is the control (D-H/D-C) |
| Anti-abuse control on support buckets | Marking bucket work as absorbed is a management choice, visible in the report (D-C) |
| Intra-day ordering / start times in direction | Calendar-app territory = meta-work (VISION §2); day + hours only (D-AA) |
| Hard cutoff locks on allocations | Cutoff is a reporting snapshot, never a lock (D-F) |
| Real-time push notifications | Polling pattern suffices; SSE waits until a notification requirement exists |
| Full invoicing module | Billing stays contractual; coverage allocations feed the conversation |
| Resource scheduling/booking engine | Capacity views are derived visibility, not booking |
| Fullcalendar / schedule-x calendar libs | Timeline/Resource views are paywalled; custom date-fns + Tailwind grid instead |
| SLA engine / escalation chains / customer portal / KB / email ingestion / auto-routing | Anti-features for v0.2 — full ITSM creep |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FND-01 | Phase 11 | Complete |
| FND-02 | Phase 11 | Pending |
| FND-03 | Phase 11 | Complete |
| FND-04 | Phase 11 | Complete |
| TICK-01 | Phase 11 | Complete |
| TICK-02 | Phase 11 | Complete |
| TICK-03 | Phase 11 | Pending |
| TICK-04 | Phase 11 | Complete |
| TICK-05 | Phase 11 | Complete |
| COV-01 | Phase 12 | Pending |
| COV-02 | Phase 12 | Pending |
| COV-03 | Phase 12 | Pending |
| COV-04 | Phase 12 | Pending |
| COV-05 | Phase 12 | Pending |
| DIR-01 | Phase 13 | Pending |
| DIR-02 | Phase 13 | Pending |
| DIR-03 | Phase 13 | Pending |
| DIR-04 | Phase 13 | Pending |
| DIR-05 | Phase 13 | Pending |
| DIR-06 | Phase 13 | Pending |
| AVAIL-01 | Phase 14 | Pending |
| AVAIL-02 | Phase 14 | Pending |
| UXFD-01 | Phase 15 | Pending |
| UXFD-02 | Phase 15 | Pending |
| AVAIL-03 | Phase 16 | Pending |
| AVAIL-04 | Phase 16 | Pending |
| AVAIL-05 | Phase 16 | Pending |
| SURF-01 | Phase 17 | Pending |
| SURF-02 | Phase 17 | Pending |
| SURF-03 | Phase 17 | Pending |
| SURF-04 | Phase 17 | Pending |
| SURF-05 | Phase 17 | Pending |
| SURF-06 | Phase 18 | Pending |
| TICK-06 | Phase 18 | Pending |
| SURF-07 | Phase 19 | Pending |
| SURF-08 | Phase 19 | Pending |
| POLS-01 | Phase 20 | Pending |
| POLS-02 | Phase 21 | Pending |
| POLS-03 | Phase 21 | Pending |
| POLS-06 | Phase 22 | Pending |
| POLS-04 | Phase 23 | Pending |
| POLS-05 | Phase 23 | Pending |
| POLS-07 | Phase 24 | Pending |
| POLS-08 | Phase 24 | Pending |
| POLS-09 | Phase 25 | Pending |
| POLS-10 | Phase 25 | Pending |
| POLS-11 | Phase 26 | Pending |

**Coverage:**

- v0.2 requirements: 47 total
- Mapped to phases: 47
- Unmapped: 0 ✓ (all requirements mapped during roadmap creation)

---
*Requirements defined: 2026-08-02 (redefined after ontology research round D-A…D-AA; supersedes the 2026-08-01 v0.2 requirement set)*
*Last updated: 2026-08-02 after roadmap creation*
