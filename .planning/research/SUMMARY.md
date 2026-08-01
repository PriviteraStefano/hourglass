# Project Research Summary

**Project:** Hourglass v0.2 — UX Polish + Tickets + Availability
**Domain:** Time/expense tracking SaaS with approval workflows, adding demand tracking (tickets), staffing (availability/capacity), and page-by-page UX polish
**Researched:** 2026-08-01
**Confidence:** HIGH (stack versions, internal architecture) / MEDIUM (features, pitfalls)

## Executive Summary

Hourglass v0.2 adds three things to a healthy v0.1: a **ticket ontology** (one unified ticket entity with task/helpdesk kinds, scoped under the existing activity tree, with per-customer request counting feeding contracts/billing), an **availability + capacity layer** (absence request/approval on the already-shipped `availability_windows` schema, plus read-only capacity views), and a **page-by-page sketch-driven UX polish pass** that folds in v0.1's UAT debt. The single strongest finding across all four research files: **v0.2 is almost purely additive work on existing foundations.** The activity tree, customers (incl. internal), WG/manager routing, immutable `*_approvals` pattern, and the entire staffing schema (migration 012) already exist — tickets and availability are new services over existing tables, needing **zero new Go dependencies and at most two new frontend packages** (`@tanstack/react-table` 8.21.3 for ticket/capacity grids; `@dnd-kit` only if a sketch lands on a kanban board). Capacity/resource calendars must be custom-built (date-fns + Tailwind) because every calendar library paywalls resource views (FullCalendar Premium $480+).

The load-bearing architecture decision: **tickets anchor to `activity_id` (any depth), never `customer_id`** — customer/contract/billability are derived through the existing recursive CTE, exactly like time entries. Per-customer request counting is therefore a pure derived query, and there is no billing engine, no stored counters. The ticket entity follows the industry pattern "one table, `kind` CHECK, one shared status vocabulary, per-kind transition matrix in the service" (JSM request-type pattern), with immutable append-only `ticket_events` mirroring the shipped `*_approvals` tables. The biggest risks are scope creep into ITSM (SLA engines, portals, escalation — all anti-features), status machine sprawl across kinds, ambiguous request-count semantics for billing, append-only migration traps, and UX churn during polish. Mitigations are structural: an anti-feature list enforced at requirements time, one status machine with customer-facing label projection, one counting rule written into an ADR at schema time, up/down/up cycle tests on every migration, and a tokens-first design-system phase before any per-page polish.

The recommended phase structure is: **UX foundation (tokens + shared components) → tickets backend → tickets frontend → availability backend → availability frontend → per-page UX polish phases.** UX polish phases run on top of everything and each adopts its own v0.1 UAT debt; the token/component foundation must land first so no page — old or new — invents ad-hoc styles. Two decisions must be made deliberately during planning, not discovered mid-phase: the ticket attribution model (activity-anchor vs customer-column — research strongly recommends activity-anchor) and the per-person weekly-hours baseline for capacity math (recommended: `weekly_hours` column on `organization_memberships`).

## Key Findings

### Recommended Stack

v0.2 needs almost no new infrastructure. Verified against npm dist-tags, peerDeps, and first-party pricing pages. Full detail: [STACK.md](STACK.md)

**Core technologies:**
- `@tanstack/react-table` 8.21.3 (add): headless tables for ticket queues and capacity grids — stable v8, renders into existing shadcn markup; v9 is beta, do not touch
- `react-day-picker` ^9.14.0 (keep, **do not bump**): absence range picking; v10 renames to `@daypicker/react` and breaks the shadcn `base-mira` Calendar wrapper — plan v10 as a standalone later task
- `date-fns` ^4.4.0 (keep): all capacity-grid date math — custom grid, no calendar library
- PostgreSQL tables (no new engine): `ticket_events` append-only (proven by `*_approvals`), `ticket_tags` join table (PG docs: "arrays are not sets")
- Go stdlib + `pgx/v5` (keep): tickets/availability are the same hexagonal service + pgx repo + thin HTTP adapter shape as v0.1 — **zero new Go imports**
- `@dnd-kit/*` (optional, only if a sketch lands on a board view): Base UI 1.6.0 has no DnD component (verified via exports map)

**What NOT to use:** FullCalendar and schedule-x (resource views paywalled at $480+), TanStack Table v9 (beta), react-day-picker v10 (renamed package, mid-milestone breakage), message queues/websockets (plain SQL + React Query `refetchInterval` polling suffice; SSE later), rich-text editors (plain Textarea matches existing notes), `@tanstack/react-virtual` (premature under ~1000 rows).

### Expected Features

Full detail: [FEATURES.md](FEATURES.md)

**Must have (table stakes / P1):**
- Unified `tickets` table: `kind` CHECK (task/helpdesk), per-kind status workflow, priority (4 levels), assignee + watchers, `activity_id` scoping, optional one-level `parent_ticket_id`
- Comments with public/internal split + immutable `ticket_events` activity history (ADR-P-003: resolution note, not a conversation)
- Helpdesk requester = customer (incl. internal customers); minimal customer-visible ticket status (in-app, customer role — no public portal)
- Per-customer request counting (monthly, by kind/status) → customer view + Reports + CSV export — the billing input
- Absence request → manager approve flow on existing `availability_windows` (declared → confirmed; HR confirmation with `certificate_ref` for medical)
- Absence calendar views (personal + per-WG) and read-only capacity view per activity/WG (People timeline + Work summary)
- UX polish pass: one phase per page, folding in the 25 open UAT scenarios + 3 human reviews from STATE.md

**Should have (differentiators):**
- Approval-routed ticket lifecycle (extends the core value; only defined transitions, not every status change)
- Time entries linked to tickets (`ticket_id` FK, P2) — makes capacity reflect reality
- SLA as a priority-derived `due_at` + breach badge (Linear-style), not a rule engine (P2)
- Internal customers as first-class helpdesk requesters; HR-role absence confirmation with certificate refs

**Defer (v2+ / anti-features):** SLA engines, configurable workflows, customer portal + KB, email ingestion, auto-routing, leave quota/accrual, resource auto-scheduling, kanban boards, deep ticket hierarchies (one `parent_ticket_id` level max).

### Architecture Approach

Full detail: [ARCHITECTURE.md](ARCHITECTURE.md) — Confidence HIGH (internal ADRs P-003/P-007/P-008/P-011, BE-014, schema, code).

v0.2 adds a **fifth concern — Demand (Ticket)** — to the v0.1 ontology (Commercial, Work, Execution, Capture). It is not a new activity-tree level, not a WG type, not an approval object.

**Major components:**
1. **Ticket service/repo/handler (NEW)** — create/assign/transition/note with per-kind transition matrix; `activity_id NOT NULL` anchor; `ticket_events` append-only history; derived-customer CTE for request counts; `ErrTicketRequiresContract` sentinel for helpdesk tickets without a derived commercial context
2. **Staffing service/repo/handler (NEW)** — `availability_windows` CRUD over the shipped 012 schema; holiday confirmation routing (unit manager + one WG manager per P-008 D-1a); capacity = weekly hours − confirmed absences; workload = submitted+approved time entries on the activity subtree; assignment-time warnings (never blocks)
3. **Frontend: `api/tickets.ts` + `api/staffing.ts`** — existing queryOptions/mutationOptions convention; `/tickets` under Track pillar, `/availability` + capacity under People pillar (navStructure with pure role predicates, ADR-P-011)
4. **UX layer (NEW/extracted)** — semantic status tokens in `index.css` (promote ad-hoc `status-badge.tsx` classes), frozen shared components (PageHeader, FilterBar, DataTable, StatusBadge variants, EmptyState), then per-page passes

**Key patterns:** derived-not-stored commercial context (D-3 CTE walk to nearest contract ancestor); immutable append-only event history (`*_approvals` precedent); kind → per-kind state machine (one CHECK, service-owned transition matrix); hexagonal feature slice (house style); semantic token + shared component layering; role-scoped surfaces via pure predicates.

**Explicitly NOT built:** boards/sprints, subtrees, comment threads, estimates, ticket approval chains, customer accounts/portal, leave balances, capacity blocks on time entry, per-ticket invoicing.

### Critical Pitfalls

Top 5 from [PITFALLS.md](PITFALLS.md) (MEDIUM-HIGH confidence; corroborated by NN/g, UWaterloo, Productive, Fowler, PG docs, and the v0.1 retrospective):

1. **Unified-ticket modeling trap (polymorphic FKs, kind-specific tables)** — prevents a clean `ticket_id` link to entries and breaks billing counts. Avoid: one `tickets` table, `kind` CHECK, nullable kind-specific columns, never `ticket_kind` next to a FK.
2. **Status machine sprawl** — two drifting status machines per kind breaks reports and counts. Avoid: one shared machine sized like the existing entry workflow, internal status vs customer-facing label projection, ADR-governed status additions.
3. **Scope creep into full ITSM (SLA timers, escalation, portal, notifications)** — doubles the milestone. Avoid: anti-feature list in REQUIREMENTS.md, enforced at roadmap creation; escalation = existing approval path or priority bump.
4. **Request-counting traps for billing** — reopen semantics, re-attribution, internal follow-ups, counting window all silently change invoices. Avoid: ONE counting rule in an ADR at schema time (creation-month, immutable customer attribution, exclude internal follow-ups), derived query never a stored counter, org-scoped tests.
5. **Append-only migration traps** — `ALTER TYPE ... ADD VALUE` can't be used in the same transaction; CHECKs need drop+recreate; NOT NULL on populated tables locks. Avoid: new numbered migrations from max+1, values referenced only in later migrations, **up→down→up cycle tests on every schema migration** (v0.1 lesson), watch the `availability_windows.kind` vs ticket `kind` naming collision.

Also critical: **capacity over-promise** (plan against availability = capacity − confirmed absences, never raw headcount; derived-never-stored) and **UX-polish traps** (redesign churn — 2–3 sketch options, one revision round, explicit agree criteria; token drift — design-system phase first; e2e regressions — Playwright specs are the contract, change test + behavior in the same plan).

## Implications for Roadmap

Based on combined research, suggested phase structure:

### Phase 1: UX Foundation — Design Tokens + Shared Components
**Rationale:** Smallest phase, unblocks everything user-facing. Every new page (tickets/availability) must be built on the token/component layer from day one; every polish phase must consume it. Cheapest time to fix inconsistencies is before any page work.
**Delivers:** Semantic status tokens in `index.css` (rule: a color used by ≥2 pages becomes a token before the second use), frozen shared component set (PageHeader, FilterBar, DataTable, StatusBadge variants, EmptyState, ConfirmDialog), sketch loop contract (2–3 options, one revision round, explicit agree criterion).
**Addresses:** UX polish foundation from FEATURES.md; Pattern 5 from ARCHITECTURE.md.
**Avoids:** Pitfall 7 (token drift, redesign churn) and Pitfall 8 (unbounded sketch loops) — the loop contract is stated once here, inherited by every page phase.

### Phase 2: Ticket Ontology — Backend
**Rationale:** The centerpiece; depends on nothing new (activity tree, WG members, contracts all exist — pure additive migration). API contract must land before any ticket UI. Schema locks the counting rule, the status machine, and the attribution model — the decisions everything else depends on.
**Delivers:** Migration 014 (`tickets`, `ticket_events`; optional `organization_memberships.weekly_hours`), ticket domain (kind transition matrix, sentinel errors), port, repo (derived-customer CTE), service, thin HTTP handler, `/tickets/counts` endpoint, testcontainers suites + **up/down/up cycle tests**.
**Uses:** STACK.md — zero new Go deps; pgx/v5 hand-written SQL; append-only pattern from `*_approvals`.
**Implements:** ARCHITECTURE.md §2 — demand concern; Patterns 1, 2, 3, 4.
**Avoids:** Pitfalls 1 (polymorphic FKs), 2 (status sprawl), 4 (counting semantics), 5 (migration traps — cycle tests).
**⚠️ Research flag:** HIGH — needs user validation on the attribution model (activity-anchor vs customer column — research strongly recommends activity-anchor per ADR-P-003/D-3 doctrine) and the exact counting rule sentence (creation-month, immutable attribution, internal-follow-up exclusion).

### Phase 3: Ticket Frontend + Today Composition
**Rationale:** Depends on Phase 2's API contract. Ships the user-visible ontology: list/filter/sort per pillar, detail with event timeline, create/assign/resolve dialogs.
**Delivers:** `web/src/api/tickets.ts`, `/tickets` route (Track pillar), ticket list (TanStack Table 8.21.3), detail timeline from `ticket_events`, customer "Requests" counts section on customer detail page, Today page "my open tickets by priority" block (read-only, P-004 rules), sidebar Tickets entry via `navStructure` (D-5 predicates).
**Uses:** STACK.md — `@tanstack/react-table` 8.21.3; existing `api<T>()` client + queryOptions conventions.
**Avoids:** Pitfall 3 (no portal — customer surface is role-scoped in-app only), anti-pattern 3 (no boards — list first).
**Research flag:** LOW — standard patterns; skip research-phase. Sketch rounds for ticket surfaces should happen within this phase's discuss step (gsd-sketch per PROJECT.md).

### Phase 4: Staffing — Backend (Availability + Capacity)
**Rationale:** The schema exists (migration 012); this is the service implementation pass. Must land before or with capacity views — capacity over `declared`-only windows shows phantom availability. Depends on WG-member access; independent of tickets (kept sequential per GSD convention).
**Delivers:** Staffing domain/service (windows CRUD, holiday confirmation routing per P-008 D-1a — unit manager + one WG manager; HR curates medical with `certificate_ref`), capacity service (capacity = weekly hours − confirmed absences; workload = submitted+approved entries on subtree; utilization), expiry queues, assignment-time warnings (never blocks), `/availability` + `/capacity` endpoints, cycle tests.
**Uses:** STACK.md — zero new Go deps; date range queries via pgx.
**Implements:** ARCHITECTURE.md §3; Pattern 1 (derived availability) and Pattern 4.
**Avoids:** Pitfall 6 (capacity over-promise — derived-never-stored; absence ≠ availability), Pitfall 5 (CHECK rework on `availability_windows` kind/status, `kind` naming collision with tickets).
**⚠️ Research flag:** HIGH — needs the **`weekly_hours` source decision** (constant 8h×weekdays vs `organization_memberships.weekly_hours` column — research recommends the column) and the holiday-confirmation UX (who confirms what, where it renders).

### Phase 5: Availability + Capacity Frontend
**Rationale:** Depends on Phase 4. Surfaces the People pillar: absence request/approval flows, calendars, capacity grids.
**Delivers:** `web/src/api/staffing.ts`, `/availability` (personal + WG calendars; declare/confirm/curate per role; HR expiry queues), capacity view (People timeline + Work summary, custom date-fns + Tailwind grid — **not** a calendar library), sidebar Availability entry via `navStructure`, assignment-time warnings on WG surfaces.
**Uses:** STACK.md — react-day-picker ^9.14 (range mode, `export-form.tsx` pattern), date-fns v4, recharts if utilization charts land.
**Avoids:** Pitfall 6 (show derived availability with absence rows visible), UX pitfall "Today landing not updated" (Today gains ticket block in Phase 3; availability stays out per P-004).
**Research flag:** LOW — standard patterns; skip research-phase.

### Phase 6: Per-Page UX Polish Phases (one phase per page)
**Rationale:** Runs last because polish phases consume the token layer (Phase 1) and their UAT debt assumes stable surfaces. Pitfalls research adds one ordering nuance: pages touching ticket/availability surfaces (time-entry pickers, activity selectors, status badges) should be polished **after** the schema phases (2 and 4) to avoid double work; polish before settled data models is churn.
**Delivers:** One phase per page, each: sketch 2–3 options → agree (explicit criterion) → implement → verify, **folding in that page's v0.1 UAT scenarios + human verification reviews** (25 + 3 filed per-page in STATE.md — adopt them, never defer to a trailing "verification debt" phase). Suggested order by traffic: Today/Time-Entries → Activities → Working Groups/Approvals → Customers/Contracts → Exports/Org; new ticket/availability pages consume tokens from birth (Phase 1) so they need far less polish.
**Addresses:** UX polish pass from FEATURES.md; ARCHITECTURE.md §4 (content-region-only changes; shell/navStructure untouched).
**Avoids:** Pitfall 7 (churn, token drift, e2e breakage — Playwright specs are the contract, change test + behavior in the same plan), Pitfall 8 (loop contract, feedback triage: visual → this phase, behavior → small or new requirement, data-model → escalate to ADR, never implement on the fly).
**Research flag:** none needed — each page needs a sketch round (gsd-sketch standard), not research. Polish phases must be **UI-only** (no new API endpoints — scope-mixing warning sign).

### Phase Ordering Rationale

- **Phase 1 before everything user-facing:** new pages must be built on frozen tokens/components (cheapest time to fix; pitfalls research is emphatic: token/design-system phase first, or every later phase reworks styling).
- **Phase 2 before Phase 3:** API contract first — never build UI on an uncommitted API shape.
- **Phases 2 and 4 independent** except shared frontend primitives from Phase 1; kept sequential per GSD convention. Both are additive over existing tables — the single biggest cost saver of the milestone.
- **Sidebar navStructure touched exactly twice** (Phase 3: Tickets, Phase 5: Availability) to keep D-5 predicates pure.
- **Phase 6 last:** polish consumes everything above; UAT debt assumes stable surfaces. Per-page phases may interleave after Phases 2/4 where pages don't touch shared ticket surfaces, but the roadmap should not serialize polish behind all feature work — folding UAT debt per-page is the milestone's explicit plan.
- **Capacity after absence approval (Phase 4 before Phase 5):** capacity over unconfirmed windows misleads managers (Pitfall 6) — the approval flow lands with the backend phase.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (ticket ontology):** HIGH — validate the activity-anchor vs customer-column attribution decision and the exact counting rule with the user; these lock the ADR. Architecture research flags this as *the* decision of the milestone.
- **Phase 4 (staffing):** HIGH — `weekly_hours` baseline source decision; holiday-confirmation routing UX (dual-confirmation, HR/medical specifics).

Phases with well-documented patterns (skip research-phase):
- **Phase 1 (UX foundation):** token system already exists and is correct (verified against shadcn theming contract) — extend, don't rebuild.
- **Phase 3, 5 (frontends):** standard house patterns (api.ts modules, React Query, shadcn markup).
- **Phase 6 (polish pages):** sketch-driven per gsd-sketch; research says no deeper research needed, just process discipline.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions/licensing verified against npm registry dist-tags/peerDeps + first-party pricing pages (2026-08-01); pattern recommendations MEDIUM but low-risk (headless table, existing deps) |
| Features | MEDIUM | Vendor docs verified directly (Linear, Atlassian, Freshdesk, Clockify, Float, NN/g); Zendesk/BambooHR/Personio details are training-level and flagged LOW-MEDIUM |
| Architecture | HIGH | Internal evidence — ADRs P-003/P-007/P-008/P-011, BE-014, schema, and code read directly; external ecosystem MEDIUM (official vendor docs) |
| Pitfalls | MEDIUM-HIGH | Multiple corroborating industry sources (NN/g, UWaterloo, Productive, Fowler, PG docs, Karwin) + project-specific grounding (v0.1 retrospective, migrations 011–013, ADR-BE-004); a few sources LOW (schillig.uk snippet, training-level) |

**Overall confidence:** HIGH for the recommended structure — the four research files converge independently on the same shape: additive schema over existing tables, activity-anchored tickets with derived billing counts, one status machine, tokens-first UX pass.

### Gaps to Address

- **Weekly-hours baseline for capacity:** no per-person working-hours source exists anywhere. Research recommends `organization_memberships.weekly_hours` (one column, matches D-2, payroll view wants it anyway) over a constant 8h. **Must be a deliberate plan-phase decision** (Phase 4).
- **Customer-facing ticket surface:** external customers have no Hourglass login; the in-app customer-role ticket view is an open ADR candidate per P-011. Decide the minimal projection (status + reply thread) in Phase 2 planning.
- **Time-entry ↔ ticket link:** deferred to P2, but the schema shape matters — do not overload `activity_id`; if linking lands, it's a proper `ticket_id` FK with a stated "one ticket, many entries, one activity" rule. Decide the rule when Phase 2's migration is drafted so nothing blocks it later.
- **Counting semantics edge cases** (reopen, re-attribution, internal follow-ups, test tickets): research lists the traps but the final rule sentence is a user decision for Phase 2 planning.
- **Zendesk/BambooHR specifics** (SLA policy internals, leave approval chains): LOW confidence, irrelevant for v0.2 scope — only relevant if anti-feature lines get challenged.
- **Kanban board variant:** `@dnd-kit` only if a ticket sketch lands on a board view; default is TanStack Table list. Leave the install decision to Phase 3 sketch rounds.
- **react-day-picker v10:** known future migration (package rename breaks the shadcn wrapper) — schedule it as a standalone task, explicitly not in v0.2.

## Sources

### Primary (HIGH confidence)
- Project internal: ADR-P-003, ADR-P-007, ADR-P-008, ADR-P-011, ADR-BE-004, ADR-BE-012, ADR-BE-014; `migrations/011–013`; `web/src/index.css`, `web/src/components/layout/sidebar.tsx`; `go.mod`, `web/package.json`; v0.1 RETROSPECTIVE.md; STATE.md (25 UAT + 3 human reviews)
- npm registry (dist-tags + peerDependencies, 2026-08-01) — authoritative package versions
- PostgreSQL 18 docs — `ALTER TYPE` restrictions; "Arrays are not sets"; Wiki "Don't Do This"
- fullcalendar.io/pricing — Timeline/Resource views Premium from $480
- Nielsen Norman Group (radical vs incremental redesign; 10 usability heuristics) — fetched

### Secondary (MEDIUM confidence)
- Official vendor docs, fetched 2026-08-01: Linear (priority, SLAs, customer requests, triage), Atlassian (workflows, work types, JSM request types, capacity plans), Freshdesk API (status/priority enums, watchers, private notes), Clockify Time Off, Float Help Center (capacity model, allocations + time off), shadcn theming docs
- UWaterloo Atlassian (workflow sprawl), Productive (capacity vs availability), SparrowDesk (status simplicity), HappyFox (ticketing mistakes), XB Software / Halo Lab (redesign pitfalls)
- Martin Fowler P of EAA (Class Table Inheritance), Bill Karwin SQL Antipatterns (polymorphic associations)
- Context7: `/tanstack/table`, `/mui/base-ui`, `/gpbl/react-day-picker`, `/schedule-x/schedule-x`

### Tertiary (LOW confidence — needs validation if used)
- Zendesk SLA policy internals, BambooHR/Personio leave approval chains (training-level, JS-rendered pages blocked scraping)
- schillig.uk workflow-sprawl claims (404 at fetch, snippet only)
- `@dnd-kit/react` 0.5.0 (new-generation DnD, June 2026 — too young, revisit later)

---
*Research completed: 2026-08-01*
*Ready for roadmap: yes*
