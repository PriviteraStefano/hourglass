# Feature Research

**Domain:** Time/expense tracking SaaS with approval workflows — adding ticket ontology (internal tasks + external helpdesk), employee availability/capacity, and UX polish
**Researched:** 2026-08-01
**Confidence:** MEDIUM (vendor docs verified directly; training-level claims flagged)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Unified ticket entity with kind discriminator (task vs helpdesk) | Every tracker (Jira work types, Freshdesk `type`, Linear issue types) uses ONE entity + a kind column — not two tables | MEDIUM | `tickets` table with `kind` CHECK ('task','helpdesk'); per-kind status workflow. Verified: Freshdesk API type field (Question/Incident/Problem/Feature Request/Refund); Jira work types (Task/Story/Bug/Epic/Subtask) |
| Status workflow per kind | Lifecycle states are the first thing users look at; task and support flows genuinely differ | MEDIUM | task: `todo → in_progress → done` (+ `blocked`, `cancelled`); helpdesk: `new → open → pending → resolved → closed`. Zendesk model (New/Open/Pending/On-Hold/Solved/Closed) and Freshdesk enum (Open=2, Pending=3, Resolved=4, Closed=5) verified; keep statuses **fixed per kind** (configurable workflows are a big surface — see anti-features) |
| Assignment: one assignee + watchers | "Precisely one person assigned formal responsibility" is the definitional property of a ticket (Wikipedia ITS); watchers get visibility without responsibility | LOW | assignee FK to users; watchers = join table. Verified: Freshdesk watcher API, Jira automatic-assignee |
| Priority (small, coarse scale) | Linear deliberately ships only No/Low/Medium/High/Urgent — "too many options leads to diminishing returns" (verified, linear.app/docs/priority) | LOW | Freshdesk: Low=1, Medium=2, High=3, Urgent=4 (verified via API docs). Use 4-5 levels, no custom priorities |
| Comments: public replies + internal notes, plus immutable activity history | Universal two-layer model: comments visible to requester, notes visible only to agents, audit trail of every change | MEDIUM | `ticket_comments` (with `internal` flag) + `ticket_events` append-only. Verified: Freshdesk `private:true` notes; Jira "types of activity on an issue"; Wikipedia "each issue maintains a history of each change". Reuses Hourglass's immutable approvals-history pattern |
| Tickets scoped under the activity tree | Matches Hourglass ontology: tickets ⊂ projects ⊂ activities; Jira project→issue, Linear project→issue→sub-issue all nest tickets under a container | LOW | `activity_id` FK on tickets (activities table exists from Phase 9). One optional `parent_ticket_id` level for sub-tickets — deeper nesting adds complexity without proportional value |
| Helpdesk requester = customer (incl. internal customers) | External tickets must identify the requesting customer org for routing + billing; internal customers already exist in Hourglass | MEDIUM | `requester_customer_id` FK to customers; requester user optional (external customers may not have logins). Verified: Linear Customer Requests tie issues to customer entities; Freshdesk requester→contact/company |
| Customer-visible ticket status | The requesting customer must see where their request is (minimal portal: create request, see status, read replies) | MEDIUM | Reuse existing `customer` role surface; no public internet portal — customer-org-scoped view inside the app. Verified: Zendesk/Freshdesk portals expose status + reply threads to requesters |
| Absence request + approval UI on `availability_windows` | Schema exists (migration 012: kind holiday/permit/medical/unavailable, status declared/confirmed, hours, certificate_ref) but no UI/service; users expect request → manager approve → calendar shows it | MEDIUM | Workflow: employee requests → manager approves → `declared → confirmed`. Clockify model verified: policies, approval selector, manager approves team's time, notifications, holidays |
| Absence calendar views (personal + team/WG) | Every absence product (Clockify, Float, BambooHR) centers on a calendar; people ask "when is my team available?" | MEDIUM | Personal month view + WG calendar; date-range index `idx_availability_windows_org_user_dates` already exists |
| Capacity/resource view per activity/WG: who can work when | Float's whole product is "visibility into who can work when"; Jira now ships individual capacity plans; expected in any resource-aware tool | HIGH | Read-only planning views: People timeline (per-person allocations vs availability) + Work view (per activity: who's allocated, remaining capacity). Verified: Float schedule (hours/day or % of capacity, 8h/day Mon–Fri default, overbooking detection); Jira capacity (hours/days/percentages, People/Work views, weekly grid) |

### Differentiators (Competitive Advantage)

Features that set the product apart. Aligned with Hourglass's core value: role-based approval workflows + activity ontology.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Approval-routed helpdesk/task lifecycle | Unlike Jira/Linear (self-serve status changes) and Zendesk (agent-controlled), Hourglass already has manager→finance approval machinery; routing ticket state transitions through it extends the core value to tickets | HIGH | Reuse existing activity → WG → manager/delegate routing and `*_approvals` immutable tables; add `ticket_approvals`. Careful: don't force every task status change through approval — only defined transitions (e.g. resolve/submit) |
| Per-customer request counting feeding contracts/billing | The milestone goal: "track how many requests come in per customer so the customer can be billed". Linear tracks customers (revenue/tier/size) and counts requests per customer (verified); Zendesk/Freshdesk report volume but billing happens externally. Hourglass can close the loop natively: count → contract → export | MEDIUM | Monthly count query grouped by requester_customer_id + status/kind + `created_at`; surfaced in customer view + Reports pillar; CSV export reuses existing exports feature. Billing itself stays contractual (count is the input, not an invoice generator) |
| Time entries linked to tickets | No other product combines time tracking + ticketing in one approval-governed flow. Employees log time against a ticket; ticket workload feeds capacity; approval chain covers both | MEDIUM | `ticket_id` nullable FK on time_entries (v0.2 or v0.2.x); enables per-ticket hours reports and capacity workload input |
| Capacity computed from real availability + activity/WG structure | Float/Jira require manual allocation entry; Hourglass has real absence data (`availability_windows`) + org hierarchy (WGs, activities). Capacity = member working hours − confirmed absences − allocated ticket estimates, per activity/WG | HIGH | v0.2: read-only view from existing tables (no scheduling/booking). Distinguish "availability" (can work) from "allocation" (assigned work) — don't invent a bookings table yet |
| Internal customers as first-class helpdesk requesters | Existing customers model includes internal customers; helpdesk tickets from internal customers count toward their internal org (chargeback), same flow as external | LOW | Reuses requester_customer_id — no extra schema |
| HR-role absence confirmation with certificate references | Schema already has hr role + `certificate_ref` (medical, INPS protocol no.) + two-line `declared/confirmed` status — a governance flavor most absence tools lack | LOW | Surface what the schema already supports: hr sees all, confirms medical with certificate ref; status transitions declared→confirmed |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full SLA engine (response/resolution targets, business-hours calendars, breach automation, escalation) | "Enterprise" helpdesk expectation (Zendesk SLA policies, ServiceNow) | Heavy: calendar math, policy rule engine, notifications, breach workflows — a product of its own; most orgs here run a 20-person service desk | SLA = priority-derived target date on the ticket (`due_at`) + breach indicator badge. Linear's whole SLA feature is rules that set deadlines (Urgent→24h, High→1wk, verified) — copy that simplicity |
| Configurable per-team custom workflows (drag-drop status editors) | Jira's flagship power feature | Status explosion, cross-org consistency loss, huge admin surface; every status change needs routing logic | Fixed status set per kind (task vs helpdesk), documented; revisit only if users demand it post-v0.2 |
| Customer self-service portal with knowledge base + CSAT | Zendesk/Freshdesk standard | Portal product = auth, articles, search, surveys — doubles the milestone | Minimal customer-visible ticket list/status/reply inside the existing customer-role app surface |
| Email-to-ticket ingestion (inbound mail parsing) | Classic helpdesk channel | Mail parsing, threading, identity mapping, spam — large and fiddly | Manual ticket creation in-app for v0.2; email bridge later if requested |
| Auto-routing / round-robin / rule-based assignment | "Smart" triage | Rule engines are where helpdesk configs get buried; misfires erode trust | Manual triage: unassigned queue + group filter; assignee picked by a person (Linear Triage pattern: accept/decline/duplicate/snooze) |
| Time-off quota/accrual engine (balances, carryover, negative balances) | BambooHR/Personio standard | Payroll-adjacent correctness requirements; orgs differ wildly in leave law | `declared/confirmed` windows only (schema as-is); no balances in v0.2 |
| Resource auto-scheduling / optimizer (Float's booking + conflict resolution) | Looks powerful | Scheduling algorithms + conflict UX are a standalone product; wrong guesses erode trust | Read-only capacity views + manual allocation tracking; users decide |
| Deep ticket hierarchies (multi-level sub-tickets, epics/initiatives) | Jira's epic→story→task→subtask ladder | Each level needs its own aggregation, filters, and workflow semantics | One `parent_ticket_id` level; activity tree is the existing "epic/initiative" layer |
| Real-time collaboration (presence, live cursors, chat) | Modern tracker feel | Already explicitly out of scope in PROJECT.md; infrastructure cost | Async comments + activity feed |

## Feature Dependencies

```
[Ticket entity + kind]
    └──requires──> [activities table]  (exists, Phase 9 — activity_id scoping)
    └──requires──> [customers incl. internal]  (exists — requester_customer_id)
    └──requires──> [users/memberships + roles]  (exists — assignee, customer-role visibility)

[Helpdesk status workflow] ──requires──> [Ticket entity] (kind discriminator drives per-kind status sets)

[Comments + activity history] ──requires──> [Ticket entity]
    └──enhances──> [Approval history pattern] (reuse immutable `*_approvals` convention)

[Per-customer request counting] ──requires──> [Ticket entity] + [customers]
    └──enhances──> [Contracts] (billing context per customer)
    └──enhances──> [Exports] (CSV/XLSX already shipped)

[Customer-visible ticket status] ──requires──> [Ticket entity] + [customer role surface] (exists)

[Time entries linked to tickets] ──enhances──> [Ticket entity] + [time_entries] (nullable FK add)

[Absence request/approval UI] ──requires──> [availability_windows] (exists, migration 012)
    └──requires──> [WG/manager routing] (exists — approval chain via activity→WG→manager)
    └──requires──> [hr role] (exists in role CHECK)

[Absence calendar views] ──requires──> [Absence request/approval UI] (data quality — confirmed absences)

[Capacity view per activity/WG] ──requires──> [Absence data] + [activities] + [WG memberships]
    └──enhances──> [Ticket estimates] (workload input, v0.2.x — P2)
    └──enhances──> [Time entries] (actual workload vs capacity, v0.2.x — P2)

[UX polish pass] ──per-page──> [Every existing surface] (no new data deps; runs in parallel phases)

[SLA due_at on tickets] ──requires──> [Ticket entity] + [priority] + [kind]
    └──conflicts──> [Full SLA rule engine] (defer the engine; ship the deadline)
```

### Dependency Notes

- **[Ticket entity] requires [activities/customers/memberships]:** All three already exist in the v0.1 schema — tickets are pure additive schema, no new foundational tables. This is the single biggest cost-saver for the milestone.
- **[Absence UI] requires [availability_windows + routing]:** The table, kinds, status, hr role, and WG approval routing all exist; the work is service + surface, not schema. Same story as tickets: additive, zero-coupled.
- **[Capacity view] requires [confirmed absence data]:** A capacity view over `declared`-only windows is misleading; the absence approval flow must land before or with the capacity view, or capacity shows phantom availability.
- **[Time entries → tickets] enhances [capacity]:** Linking logged time to tickets makes capacity views reflect reality; without it, capacity is estimates-only. Both are P2 — capacity ships read-only first (P1) and gains workload later.
- **[UX polish] runs per-page, no data deps:** It can proceed in parallel with tickets/availability phases; do not serialize it behind them.
- **[SLA due_at] conflicts with [full SLA engine]:** The deadline-on-ticket is cheap and satisfies the "SLA notions" expectation; the engine is an anti-feature for v0.2 (see above).

## MVP Definition

### Launch With (v0.2)

Minimum viable feature set for the milestone:

- [ ] Unified `tickets` table (kind task/helpdesk, status per kind, priority, assignee, watchers, requester_customer_id, activity_id, optional parent_ticket_id, due_at) — the ontology core
- [ ] Task workflow (`todo → in_progress → done`, plus blocked/cancelled) with comments + immutable event history
- [ ] Helpdesk workflow (`new → open → pending → resolved → closed`) with comments (public vs internal notes) + event history
- [ ] Ticket surfaces: list/filter/sort per pillar (Work for tasks, customer-facing for helpdesk), detail view with activity feed
- [ ] Per-customer request counting (monthly, by kind/status) surfaced on customer view + Reports, CSV export — the billing input
- [ ] Customer-visible ticket status + reply thread (customer role sees own org's tickets)
- [ ] Absence request/approve flow on `availability_windows` (declared → confirmed; hr confirmation for medical with certificate_ref)
- [ ] Absence calendar views: personal + per-WG
- [ ] Capacity view per activity/WG: People timeline + Work summary (availability minus confirmed absences; read-only)
- [ ] UX polish pass: one phase per page against the 10-heuristic checklist (see Sources — NN/g, Jan 2024 review)

### Add After Validation (v0.2.x)

- [ ] SLA deadline derivation (priority + kind → `due_at`, breach indicator badge) — trigger: helpdesk users ask "when will this be done?" (they will, immediately)
- [ ] Link time entries to tickets (`ticket_id` on time_entries) — trigger: capacity view users want real workload vs estimates
- [ ] Ticket estimates (story-point-style or hours) feeding capacity workload — trigger: capacity view adoption
- [ ] Sub-ticket creation UX polish — trigger: users start splitting tickets manually in titles (parent_ticket_id ships in schema, UI later)

### Future Consideration (v0.3+)

- [ ] Email-to-ticket ingestion — channel demand, not core
- [ ] Customer self-service portal + knowledge base + CSAT — portal product, out of scope for v0.2
- [ ] Quota/accrual leave engine — legal/payroll variance, revisit with a concrete customer
- [ ] Configurable workflows / drag-drop status editors — only after fixed status sets prove limiting
- [ ] Auto-routing/triage rules — manual triage first
- [ ] Kanban board views for tickets — list views first; boards are a view-layer add later
- [ ] Resource auto-scheduling — read-only capacity first, scheduling only if demanded

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Ticket entity + kind + status workflows | HIGH | MEDIUM | P1 |
| Comments (public/internal) + immutable history | HIGH | MEDIUM | P1 |
| Assignment + watchers + priority | HIGH | LOW | P1 |
| Tickets scoped under activities | HIGH | LOW | P1 |
| Per-customer request counting (billing input) | HIGH | MEDIUM | P1 |
| Customer-visible ticket status | HIGH | MEDIUM | P1 |
| Absence request/approval UI | HIGH | MEDIUM | P1 |
| Absence calendar views (personal + WG) | MEDIUM | MEDIUM | P1 |
| Capacity view per activity/WG (read-only) | MEDIUM | HIGH | P1 |
| UX polish pass per page | HIGH | MEDIUM (per page) | P1 |
| SLA due_at + breach indicator | MEDIUM | LOW | P2 |
| Time entries ↔ tickets | MEDIUM | MEDIUM | P2 |
| Ticket estimates → capacity workload | MEDIUM | MEDIUM | P2 |
| Sub-ticket UI | LOW | LOW | P3 |
| Kanban/board views | LOW | MEDIUM | P3 |
| Email ingestion | MEDIUM | HIGH | P3 |

**Priority key:**
- P1: Must have for v0.2 — the ontology, the two workflows, counting-for-billing, absences, read-only capacity, polish
- P2: Should have, add when possible — SLA deadline, ticket-time linkage, estimates
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Jira | Linear | Zendesk/Freshdesk | Clockify | Float | Our Approach |
|---------|------|--------|--------------------|----------|-------|--------------|
| Ticket kinds | Work types (Task/Story/Bug/Epic/Subtask) | Issue types per team | `type` field (Question/Incident/Problem/Feature Request/Refund) | n/a | n/a | `kind` CHECK (task/helpdesk) — fixed, per-kind status sets |
| Statuses | Configurable workflows (statuses + transitions) | Per-team statuses + Triage | New/Open/Pending/On-Hold/Solved/Closed (Z); Open=2/Pending=3/Resolved=4/Closed=5 (F) | n/a | n/a | task: todo→in_progress→done; helpdesk: new→open→pending→resolved→closed (no per-team config) |
| External requests | JSM request types | Customer Requests (customers w/ revenue/tier/size; count per customer) | Requester→contact/company; portal | n/a | n/a | requester_customer_id (incl. internal customers); customer-role surface; per-customer counts → contracts |
| SLA | Due dates, SLA via add-ons | Rules → deadlines (Urgent→24h, High→1wk); fire-icon breach indicator | SLA policies (first reply/resolution targets, business hours) | n/a | n/a | Priority-derived `due_at` + breach badge (Linear-style, verified) — no rule engine |
| Assignment | Assignee, automatic assignee, watchers | Assignee + delegate; Triage inbox (accept/decline/duplicate/snooze) | Groups, agents, watchers, merge | n/a | n/a | Assignee + watchers; unassigned queue; manual triage |
| Comments/audit | Comments + activity types | Activity log + comments | Public comments + internal notes (private:true); ticket events | n/a | n/a | `ticket_comments` (internal flag) + append-only `ticket_events` |
| Hierarchy | project→epic→story/task→subtask | initiative→project→issue→sub-issue (inherits team/priority/project) | Attach to company; no project nesting | n/a | n/a | activity_id (tree) + one parent_ticket_id level |
| Absences | n/a | n/a | n/a | Policies: types, approval selector, accrual, units, holidays, manager approval, export | Time off on schedule | availability_windows (exists): kind, declared→confirmed, hr + certificate_ref; request/approve + calendars |
| Capacity | Individual capacity plans (hours/days/%, 40h baseline, People/Work views) | n/a | n/a | Capacity/utilization via planning | Schedule timeline: hours/day or % capacity, overbooking flags | Read-only per-activity/WG view: availability − confirmed absences; workload added via ticket estimates/time entries (P2) |
| Billing per customer | n/a (add-ons) | Customer revenue/tier attributes | Reports on ticket volume by org; billing external | Client billing on tracked time | n/a | Native: per-customer monthly request counts + export, feeding existing contracts |

## Sources

- Linear Docs (linear.app/docs — direct fetch, current): Priority, SLAs, Triage, Customer Requests, Parent and Sub-issues, Projects, Due dates, Estimates — **MEDIUM-HIGH** (primary vendor docs, fetched 2026-08-01)
- Atlassian Jira Cloud Support (support.atlassian.com — direct fetch): What are Jira workflows?, Set up work types in team-managed spaces, What are the different types of activity on an issue?, Plan individual capacity for your team — **MEDIUM-HIGH** (primary vendor docs, fetched 2026-08-01)
- Freshdesk API documentation (developers.freshdesk.com/api — direct fetch, full server-rendered docs): ticket status enum (Open=2/Pending=3/Resolved=4/Closed=5), priority enum (Low=1..Urgent=4), type field, watchers, merge, custom fields — **MEDIUM-HIGH** (primary vendor docs, fetched 2026-08-01)
- Clockify Time Off feature page (clockify.me/features/time-off — direct fetch): policies, approval selector, accrual, units, holidays, manager approvals, export — **MEDIUM** (vendor marketing/feature page)
- Float Help Center (support.float.com — direct fetch): Quick-start (capacity = hours/day or % capacity; 8h/day Mon–Fri; schedule timeline; overbooking) — **MEDIUM** (vendor help center)
- Nielsen Norman Group, 10 Usability Heuristics for User Interface Design (nngroup.com — direct fetch, last reviewed Jan 2024): the page-by-page audit framework for the UX polish pass — **MEDIUM-HIGH**
- Wikipedia, Issue tracking system (direct fetch): ticket = unique reference, one formal assignee, urgency, per-issue history, duplicate amalgamation — **MEDIUM** (encyclopedic, secondary)
- Training-level knowledge (not directly re-verified this session — treat as LOW-MEDIUM, verify during phase research): Zendesk ticket status list and SLA policy types (help center is JS-rendered and blocked scraping); BambooHR/Personio leave approval chains, quotas, sick-note handling; ServiceNow task-superclass incident/request/change model; customer portal details (Zendesk Help Center/Freshdesk portal)

---
*Feature research for: Hourglass v0.2 — tickets + availability + UX polish*
*Researched: 2026-08-01*
