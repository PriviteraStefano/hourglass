# Requirements: Hourglass v0.2

**Defined:** 2026-08-01
**Core Value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

## v0.2 Requirements

Requirements for milestone v0.2 — UX Polish + Tickets + Availability. Each maps to roadmap phases.

### UX Foundation

- [ ] **UXFD-01**: Design token foundation extended (status palette, state colors) and shared component set frozen before any page work
- [ ] **UXFD-02**: Sketch-driven loop established: each page phase shows 2–3 gsd-sketch options, user agrees, UI-only plans, ≤3 sketch rounds

### Tickets

- [ ] **TICK-01**: User can create a ticket on any activity (kinds: task, helpdesk)
- [ ] **TICK-02**: Customer can open a helpdesk ticket (external + internal customers)
- [ ] **TICK-03**: Ticket has per-kind status workflows (shared status vocabulary, kind-specific transitions)
- [ ] **TICK-04**: Ticket history is an immutable event stream with comments/resolution notes
- [ ] **TICK-05**: Manager can assign tickets to WG members (assignee rules)
- [ ] **TICK-06**: Tickets are auto-approved tracked work; manager-intervention semantics decided in ontology phase ADR
- [ ] **TICK-07**: User can view per-customer request counts (received/resolved) and export them
- [ ] **TICK-08**: Tickets surface in the Track pillar

### Availability

- [ ] **AVAIL-01**: Employee can declare an absence with a type (existing `availability_windows` schema)
- [ ] **AVAIL-02**: Manager/HR can confirm or reject absences (declared → confirmed)
- [ ] **AVAIL-03**: User can view an absence calendar (team/org)
- [ ] **AVAIL-04**: Manager can view capacity per activity/WG (hours − confirmed absences)
- [ ] **AVAIL-05**: Availability surfaces in the People pillar

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

### Tickets (extended)

- **TICK-09**: SLA/deadline rules (Linear-style: Urgent→24h, High→1wk) with breach indicator
- **TICK-10**: Customer-facing ticket portal
- **TICK-11**: Email ingestion (ticket by email)
- **TICK-12**: Escalation chains (reuse approval path or nothing)
- **TICK-13**: `time_entries.ticket_id` nullable FK linking entries to tickets
- **TICK-14**: Ticket hours distribution + budget burn-down mechanics (hours vs `budget_caps`, manager redistribution) — **needs in-depth ontology discussion; deliberately not scoped in v0.2**

### Availability (extended)

- **AVAIL-06**: Quota/accrual engine (paid leave balances)
- **AVAIL-07**: Hard booking / resource scheduling engine

### UX (extended)

- **UXFD-03**: Calendar library upgrade to `@daypicker/react` (v10) — explicit standalone task, touches every calendar
- **UXFD-04**: Real-time notifications (SSE/websockets) — polling via `refetchInterval` if watch/assignee notifications appear

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Full invoicing module | Request counts feed the customer-billing conversation; billing stays contractual (count → contract → CSV export) |
| Resource scheduling/booking engine | Capacity views are derived visibility, not booking |
| Kanban board | Only if a sketch lands on one — conditional `@dnd-kit` decision, not committed scope |
| Fullcalendar / schedule-x calendar libs | Timeline/Resource views are paywalled ($480+); custom date-fns + Tailwind grid instead |
| SLA engine / escalation chains / customer portal / KB / email ingestion / auto-routing | Anti-features for v0.2 — full ITSM creep |
| Real-time push notifications | Polling pattern suffices; SSE waits until a notification requirement exists |
| Ticket approval workflow (two-stage) | Tickets are tracked work, not entries — auto-approved with permission control |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| UXFD-01 | Phase 11 | Pending |
| UXFD-02 | Phase 11 | Pending |
| TICK-01 | Phase 12 | Pending |
| TICK-02 | Phase 13 | Pending |
| TICK-03 | Phase 12 | Pending |
| TICK-04 | Phase 12 | Pending |
| TICK-05 | Phase 12 | Pending |
| TICK-06 | Phase 12 | Pending |
| TICK-07 | Phase 13 | Pending |
| TICK-08 | Phase 13 | Pending |
| AVAIL-01 | Phase 14 | Pending |
| AVAIL-02 | Phase 14 | Pending |
| AVAIL-03 | Phase 15 | Pending |
| AVAIL-04 | Phase 15 | Pending |
| AVAIL-05 | Phase 15 | Pending |
| POLS-01 | Phase 16 | Pending |
| POLS-02 | Phase 17 | Pending |
| POLS-03 | Phase 17 | Pending |
| POLS-04 | Phase 19 | Pending |
| POLS-05 | Phase 19 | Pending |
| POLS-06 | Phase 18 | Pending |
| POLS-07 | Phase 20 | Pending |
| POLS-08 | Phase 20 | Pending |
| POLS-09 | Phase 21 | Pending |
| POLS-10 | Phase 21 | Pending |
| POLS-11 | Phase 22 | Pending |

**Coverage:**
- v0.2 requirements: 26 total
- Mapped to phases: 26
- Unmapped: 0 ✓ (all requirements mapped during roadmap creation)

---
*Requirements defined: 2026-08-01*
*Last updated: 2026-08-01 after roadmap creation*
