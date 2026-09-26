# Architecture Research: v0.2 — Ticket Ontology, Availability, UX Polish

**Domain:** Time/expense capture + demand tracking (tickets) + staffing (availability/capacity) — Go hexagonal backend, PostgreSQL, React 19 + TanStack + shadcn/ui
**Researched:** 2026-08-01
**Confidence:** HIGH (internal evidence — ADRs P-003/P-007/P-008/P-011, BE-014, schema, code) · MEDIUM (external ecosystem — official vendor docs)

---

## 1. Standard Architecture

### 1.1 System Overview (v0.1 as-shipped + v0.2 additions)

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        FRONTEND (React 19 SPA)                             │
│  ┌──────────────┐ ┌───────────────────┐ ┌───────────────┐ ┌──────────────┐  │
│  │ Routes       │ │ API modules       │ │ Components    │ │ Types        │  │
│  │ TanStack Rtr │ │ queryOptions/     │ │ ui/ shared/   │ │ types/api.ts │  │
│  │ file-based   │ │ mutationOptions   │ │ approval/ app │ │              │  │
│  └──────┬───────┘ └─────────┬─────────┘ └───────┬───────┘ └──────────────┘  │
│         │                   │                   │                           │
│  ┌──────▼───────────────────▼───────────────────▼───────────────────────┐  │
│  │  Design tokens: index.css — :root/.dark CSS vars + @theme inline     │  │
│  │  Theme provider, navStructure (ADR-P-011 D-1/D-5 declarative groups) │  │
│  └──────┬───────────────────────────────────────────────────────────────┘  │
└─────────┼──────────────────────────────────────────────────────────────────┘
          │ HTTP/JSON (HttpOnly cookies, 401→refresh retry)
          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                      BACKEND (Go, hexagonal)                               │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ PRIMARY ADAPTERS — internal/adapters/primary/http/                  │  │
│  │ auth org unit customer contract activity working_group              │  │
│  │ time_entry expense export  + NEW: ticket.go availability.go         │  │
│  └──────────────────────────────┬──────────────────────────────────────┘  │
│  ┌──────────────────────────────▼──────────────────────────────────────┐  │
│  │ CORE SERVICES — internal/core/services/                             │  │
│  │ time_entry expense activity working_group contract customer ...     │  │
│  │  + NEW: ticket/ (demand), staffing/ (windows, confirm, capacity)     │  │
│  └──────────────────────────────┬──────────────────────────────────────┘  │
│  ┌──────────────────────────────▼──────────────────────────────────────┐  │
│  │ PORTS — internal/core/ports/  (interfaces; + ticket, staffing)      │  │
│  └──────────────────────────────┬──────────────────────────────────────┘  │
│  ┌──────────────────────────────▼──────────────────────────────────────┐  │
│  │ DOMAIN — internal/core/domain/   (+ ticket/, staffing/)             │  │
│  └──────────────────────────────┬──────────────────────────────────────┘  │
│  ┌──────────────────────────────▼──────────────────────────────────────┐  │
│  │ SECONDARY ADAPTERS — internal/adapters/secondary/postgres/          │  │
│  │ 20 existing repos + NEW: ticket_repository, staffing_repository     │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ POSTGRESQL — 24 tables + 3 migration sets (append-only, ADR-BE-004)        │
│ activities(recursive) · working_groups(manager_id+delegate_ids)            │
│ availability_windows(typed absences) · organization_memberships(+validity) │
│ time_entries/expenses(activity_id FK) · *_approvals (immutable history)    │
│ customers(is_internal) · contracts · financial_cutoff_periods              │
│  + NEW (014+): tickets, ticket_events                                      │
└────────────────────────────────────────────────────────────────────────────┘
```

**The v0.2 load-bearing idea:** the v0.1 ontology separates four concerns — Commercial (Customer/Contract), Work (recursive Activity), Execution (Working Group), Capture (time_entries/expenses + their approvals). v0.2 adds a **fifth concern: Demand (Ticket)**. It is *not* a new level of the activity tree, not a new WG type, and not an approval object. It is the demand side of the capture loop (ADR-P-003: demand → effort → cost). Everything below follows from keeping that separation clean.

### 1.2 Component Responsibilities (existing + new)

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Activity service/repo | Recursive work entity; kind catalog; commercial context + billability derived via recursive CTE (D-3/D-7) | `internal/core/services/activity/`, `postgres/activity_repository.go` |
| Working Group service/repo | Execution structure anchored to activity; `manager_id` + `delegate_ids`; member set (`wg_members`) | `internal/core/services/working_group/` |
| Entry services (time/expense) | Two-stage approval; routing via activity → WG → manager/delegate (BE-014 R-1); `IsWGManager`; unit-tree fallback | `internal/core/services/{time_entry,expense}/` |
| Approval history | Immutable append-only `*_approvals` rows (action, actor, comment, created_at) | tables in `000_full_schema`, ADR-BE-012 |
| **Ticket service (NEW)** | Demand entity: create/update/assign/transition/note; per-kind status machine; list/filter; request-count queries | `internal/core/services/ticket/` |
| **Ticket repo (NEW)** | Persist tickets + ticket_events; derived-customer CTE join for counts | `postgres/ticket_repository.go` |
| **Staffing service (NEW)** | availability_windows CRUD + holiday confirmation routing (P-008 D-1a); validity dates; capacity computations | `internal/core/services/staffing/` |
| **Staffing repo (NEW)** | Windows by org/user/date; membership validity; workload aggregation per person/activity/week | `postgres/staffing_repository.go` |
| **Ticket HTTP handler (NEW)** | Thin adapter for `/tickets` + `/tickets/counts` | `internal/adapters/primary/http/ticket.go` |
| **Staffing HTTP handler (NEW)** | Thin adapter for `/availability`, `/availability/confirm`, `/capacity` | `internal/adapters/primary/http/staffing.go` |
| Frontend: `api/tickets.ts`, `api/staffing.ts` (NEW) | queryOptions/mutationOptions following existing API-module convention | `web/src/api/` |
| Frontend: ticket + availability pages (NEW) | `/tickets` (Track pillar), `/availability` + capacity view (People pillar) per ADR-P-011 D-1 | `web/src/routes/_authenticated/{tickets,availability}/` |
| Frontend: shared page components (NEW/extracted) | PageHeader, FilterBar, status badge variants, DataTable — consumed by every page pass | `web/src/components/shared/` |
| Frontend: Today page (MODIFIED) | Compose owned open tickets by priority once tickets ship (ADR-P-003 consequences; P-011 D-2) | `web/src/routes/_authenticated/-components/today-page.tsx` |
| Frontend: sidebar navStructure (MODIFIED) | Add Tickets under **Track**, Availability under **People**; role predicates per D-5 matrix | `web/src/components/layout/sidebar.tsx` |

---

## 2. Recommended Architecture: Ticket Ontology (the centerpiece)

### 2.1 Where tickets attach — `tickets.activity_id NOT NULL`, commercial context derived

**Decision: every ticket references exactly one activity (`activity_id NOT NULL`), at any depth, exactly like time_entries/expenses (ADR-P-007 D-4).** Customer/contract/billability are *derived* through the same recursive CTE (D-3) — never stored on the ticket.

Why this, and not `customer_id` on the ticket:

- It makes the user's framing literally true: *"Both types are part of a larger scope related to projects, and projects are themselves part of activities."* A ticket is demand on a piece of work; the work lives in the tree.
- It preserves the D-3 doctrine ("derived, not stored") that the whole v0.1 ontology is built on. A denormalized `customer_id` duplicates the commercial chain, drifts when activities are re-parented or contracts change, and strands history — the exact rot D-3/D-7 exist to prevent.
- It makes demand → effort traceable for free: a ticket and the time entries that resolved it share the activity chain. The `ticket_events` history (2.4) records links, no join-table needed in v0.2.
- Request counting per customer becomes a pure derived query: walk each ticket's activity chain to the nearest `contract_id` ancestor → `contracts.customer_id` (CTE join), then `COUNT(*) GROUP BY customer, period`. One query, no denormalization, no sync jobs.

Field sketch (additive migration `014_tickets`):

```sql
CREATE TABLE tickets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id),
    activity_id   UUID NOT NULL REFERENCES activities(id),        -- D-4 anchor, any depth
    kind          VARCHAR(20) NOT NULL CHECK (kind IN ('task','helpdesk')),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    status        VARCHAR(20) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','in_progress','resolved','dismissed')),
    priority      VARCHAR(10) NOT NULL DEFAULT 'normal'
                  CHECK (priority IN ('low','normal','high','urgent')),
    requester_id  UUID REFERENCES users(id),                      -- NULL for external
    assignee_id   UUID REFERENCES users(id),                      -- owner; NULL = unassigned
    external_contact VARCHAR(255),                                -- name/email for external requesters
    created_by    UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at   TIMESTAMPTZ
);
-- kind is a closed CHECK, NOT a catalog table: ADR-P-003 defines exactly two
-- origins (internal/external). If org-extensible ticket kinds ever arrive,
-- mirror activity_kinds then — do not pre-build the catalog (YAGNI).
```

Notes on the fields:

- **`kind` is a CHECK, not a catalog** (contrast with `activity_kinds`). The ontology defines exactly two demand origins — `task` (internal work raised by a coworker) and `helpdesk` (external request raised on behalf of a customer, incl. internal customers). P-003's hard boundary is about *not* becoming a PM tool; a free-form kind catalog invites scope creep. If V4 (change requests as a ticket type) lands later, the CHECK widens to include `change_request` in a new migration — cheap.
- **`external_contact` + nullable `requester_id`:** an external ticket's requester is a customer-side person who is *not a user* of Hourglass. For internal tickets the requester is an org member (user FK). `external_contact` is free text (name/email) — the customer *organization* is derived via the activity chain; we do not create customer-side user accounts in v0.2 (the customer-facing surface is an open ADR candidate per P-011).
- **Internal customers are covered structurally:** an internal customer (`customers.is_internal = TRUE`) still has contracts, so an activity under one of those contracts derives that customer. A helpdesk ticket under an internal-customer contract is "external" in origin but bills the org itself — the counting query treats it identically. No special-casing in code.
- **No `parent_id` on tickets, no subtask trees, no estimates, no sprint fields** — P-003's excluded list, enforced by schema absence.
- **`resolved_at`** is set on transition to `resolved`/`dismissed` — it is the *billing-relevant* timestamp (a request is billable once resolved) and powers aging/SLA-ish views later. Storing it is a deliberate exception to "derived, not stored": it is a fact of the state transition, captured by the service at transition time, not recomputable from other data.

### 2.2 Shared entity vs kind-specific fields — one table, one status vocabulary, per-kind transition matrix

**Decision: one `tickets` table with a shared `status` vocabulary (`open → in_progress → resolved/dismissed`) and a per-kind transition matrix enforced in the service layer — not separate status columns, not separate tables.**

This is the industry-standard shape: Jira Service Management models one issue entity where each *request type* maps to its own workflow; team-managed projects give each request type its own statuses (official Atlassian docs). The pattern is "one entity, kind column, workflow attached per kind" — exactly what a single `tickets` table with kind-scoped transitions gives us, minus the configurable-workflow machinery we don't need.

The two kinds differ only in *who may perform transitions and what the resolution means*:

| Transition | task (internal) | helpdesk (external) |
|------------|-----------------|---------------------|
| `open → in_progress` | assignee (or assigner) | assignee; requires `assignee_id` set |
| `→ resolved` | assignee | assignee; resolution note **required** (answer to the customer) |
| `→ dismissed` | assignee / requester | assignee / requester; note recommended |
| `resolved → open` (reopen) | assignee / requester | requester-side reopening via note; assignee |

Enforcement follows the existing codebase pattern: DB CHECK constrains the *vocabulary* (as migration 004/005 do for entries), the **service owns the transition table** keyed by kind, and handler-level role checks gate who may act (as approvals do). Tests assert the matrix per kind (service tests, matching the BE-009 testcontainers convention).

### 2.3 Assignment reuses the WG model — but assignment ≠ approval

**Decision: `tickets.assignee_id` is a plain user FK, validated at assignment time against the execution structure of the ticket's activity chain: WG members (`wg_members` of the WG anchored at the activity) first, then `activity_managers`, then org members for personal/internal activities.**

The WG manager/delegate model (`working_groups.manager_id` + `delegate_ids`) is the *approval* principal (BE-014 R-1) — it must NOT be overloaded as the assignable set. WG *membership* (`wg_members`) is the execution set; that is what assignment validates against. This keeps the doctrine clean: WG manager/delegate = who approves effort/cost; WG members = who can be assigned demand. Reuse the existing `wg_member_repository` via a port method (`WGMembersForActivity` or reuse the activity→WG join already used by `ListPending`), and the existing `IsWGManager`-style service gates for manager-level actions (reassignment, dismissal).

Fallback ladder (mirrors BE-014 R-2's spirit, not its approval semantics): anchored WG members → `activity_managers` (governance role) → any org member for `task` tickets on personal activities. A helpdesk ticket **requires** a derived commercial context (nearest ancestor contract) at creation — service-level sentinel `ErrTicketRequiresContract` mirroring `ErrActivityNotLoggable` — because "make the customer pay" is meaningless without a customer.

### 2.4 History: immutable append-only `ticket_events` (the `*_approvals` pattern, not comments)

**Decision: every action on a ticket appends an immutable row to `ticket_events` — created, assigned, status_changed, note, reopened, linked. No comment threads, no editing, no deletion.**

ADR-P-003 is explicit: *"No comment threads (a resolution note, not a conversation)."* The codebase already has the canonical pattern for this: `time_entry_approvals`/`expense_approvals` — append-only rows of (action, actor, actor_role, comment, created_at), immutable per ADR-BE-012. Mirror it:

```sql
CREATE TABLE ticket_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id     UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    action        VARCHAR(30) NOT NULL
                  CHECK (action IN ('created','assigned','status_changed','note','reopened','linked')),
    actor_user_id UUID NOT NULL REFERENCES users(id),
    comment       TEXT,                       -- the resolution note lives here (required on helpdesk resolve)
    metadata      JSONB,                      -- {from_status, to_status, from_assignee, to_assignee, ...}
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The `comment` column on the event *is* the resolution note — one note per resolution, visible in history, immutable. `metadata` JSONB carries transition details (status/assignee before/after) so the UI renders a timeline without re-querying; the row stays append-only.

### 2.5 Request-count-per-customer billing data flow

The billing flow is **derived counting, surfaced on the Economics surfaces — never a billing engine** (VISION §8: Hourglass produces trusted data; invoice issuance is out of scope):

```
ticket created (helpdesk kind)
  → service validates derived commercial context (activity → nearest contract ancestor → customer)
  → resolved_at set on resolution
  → GET /tickets/counts?customer_id=X&from=2026-08-01&to=2026-08-31&kind=helpdesk
      → repo: CTE walk per ticket → join contracts → COUNT(*) GROUP BY customer_id, DATE_TRUNC(month, resolved_at)
      → response: { total, resolved, by_kind, by_period }  — pure aggregation, no new tables
  → surfaced on: customer detail page ("Requests" section — Economics pillar), export/report views later
```

Two counters matter and both are cheap derived queries: **requests received** (`created_at` in period — "how many requests came in", the demand volume) and **requests resolved** (`resolved_at` in period — the billable volume). v0.2 ships the queries + a counts endpoint + a customer-detail surface; actual invoicing stays out (P-006/P-008 anchors).

---

## 3. Recommended Architecture: Availability + Capacity

### 3.1 The schema exists; v0.2 adds the service + surfaces

`availability_windows` (typed absences, `status declared/confirmed`, `hours` for partial days), `organization_memberships.valid_from/valid_until/work_permit_expires_at`, and the `hr` role all shipped in migration 012 (ADR-P-008). **There is no repository, service, handler, or frontend surface for them yet** — v0.2 is the implementation pass, exactly as P-011 D-4/D-5 scheduled (`/availability` under People).

New `staffing` bounded context (backend):

- **Windows service**: CRUD (self-declare for own windows; unit manager + HR for others), holiday confirmation routing per P-008 D-1a (**both** the unit manager and one WG manager from the person's active WGs must confirm → `status = 'confirmed'`; permit/medical/unavailable are record-only), expiry queues (memberships/permits expiring in next N days — HR surface).
- **Capacity service**: per person per week `capacity_hours = weekly_default_hours − absence_hours(confirmed windows overlapping the week) − holiday_days`; `workload_hours = SUM(hours)` of submitted+approved time entries on the activity subtree; `utilization = workload / capacity`. Absences **reduce capacity**; entries **fill it** — the Float model (allocations + time off vs capacity; official Float docs).
- **Assignment-time warnings** (P-008 D-3): WG member add / activity assignment endpoints surface "away 10–21 Aug", "membership ends 1 Sep", "permit expires 15 Aug" — warnings only, **never blocks** (D-3: "explicitly not used to block time-entry submission").

⚠️ **Schema gap (flagged): there is no per-person weekly-hours source.** `availability_windows` has `hours` per window, but no default working hours exist anywhere. Capacity needs a baseline. Options: (a) constant 8h × weekdays (simplest, defensible for v0.2), (b) `organization_memberships.weekly_hours` column in the same migration batch (better, still cheap). Recommend (b) — one column, matches the D-2 "employment validity dates" row the staffing ADR already owns, and it is the number the payroll view (D-1c) wants anyway. This must be a deliberate decision in the plan, not discovered mid-phase.

### 3.2 Capacity views composition

- **Per activity/WG view** (`/activities/{id}/capacity` or a dedicated `/capacity` route): for the WG anchored at the activity — members × weeks, each cell = capacity − absences vs booked hours on the activity subtree. Data source: staffing repo workload aggregation (time_entries joined through activity subtree CTE) + windows.
- **Per person view** (`/availability`): the person's windows + validity dates + per-week capacity/load — the "People" pillar surface, role-scoped per P-011 D-5 (employee sees own; manager sees subtree; HR curates org-wide; finance reads).
- The Today view does **not** gain capacity content (P-004: one answer, not a dashboard).

---

## 4. Recommended Architecture: Page-by-Page UX Polish

### 4.1 Tokens first, components second, pages third

The design-token foundation already exists and is correct: `index.css` implements the shadcn/ui theming contract exactly — semantic CSS variables under `:root` and `.dark`, exposed to Tailwind via `@theme inline`, radius scale derived from `--radius` (verified against official shadcn theming docs and the live file). **Do not rebuild the token system.** The polish pass extends it:

1. **Token layer (small, front-loaded):** add semantic tokens the pages will need — a status palette (the six approval states + ticket states currently use ad-hoc Tailwind classes in `status-badge.tsx`; promote them to tokens like `--status-approved`, `--status-rejected`, `--ticket-open`, …), plus any spacing/typography tokens the UI-SPEC work identifies. Rule: *a color used by ≥2 pages becomes a token before the second use* — new pages (tickets/availability) must be built on tokens from day one so they don't create new ad-hoc colors.
2. **Shared component layer (front-loaded):** extract the page-level primitives that every per-page pass will consume — `PageHeader` (title band already standardized in Phase 10-03 shell), `FilterBar` (already exists as `entries-filters.tsx` — generalize), `DataTable` (exists in ui/, standardize usage), `StatusBadge` variants, `EmptyState` (exists), `ConfirmDialog`. Freeze the set early; per-page phases consume, never extend ad hoc (moving target = churn).
3. **Per-page passes:** one phase per page, gsd-sketch-driven (per PROJECT.md), each: sketch options → agree → implement → verify, **folding in that page's v0.1 UAT debt** (25 pending scenarios + 3 human verification reviews are already filed per-page in STATE.md — each page phase must adopt its own).

### 4.2 The IA is the contract — pages change inside the shell, never the shell's logic

The pillar IA (ADR-P-011) is the stability boundary: `navStructure` stays a single declarative structure with pure role predicates; sidebar groups/job-language stay; route names stay (`/activities`, `/approvals`, …). The polish pass touches **content regions inside the existing Body/shell wrappers only** — exactly the boundary Phase 10-03 established. New nav entries are added *once*, when their surfaces ship: **Tickets under Track, Availability under People** (D-1 table), with role visibility per the D-5 matrix (customer column untouched — open ADR candidate).

---

## 5. Patterns to Follow

### Pattern 1: Derived-not-stored commercial context (D-3 CTE)
**What:** commercial facts (contract, customer, billability) are resolved by walking `parent_id` to the nearest ancestor with `contract_id` — never denormalized onto children.
**When:** every new entity that participates in the commercial chain — tickets (2.1) and capacity/workload views included.
**Example (shape, in the ticket repo):**
```sql
-- per ticket: nearest contract ancestor + its customer
WITH RECURSIVE chain AS (
  SELECT id, parent_id, contract_id FROM activities WHERE id = $1
  UNION ALL
  SELECT a.id, a.parent_id, a.contract_id FROM activities a
  JOIN chain c ON a.id = c.parent_id
  WHERE c.contract_id IS NULL
)
SELECT contract_id FROM chain WHERE contract_id IS NOT NULL LIMIT 1;
```

### Pattern 2: Immutable append-only event history
**What:** every lifecycle fact is an insert-only row (action, actor, comment, timestamps); nothing is ever updated/deleted; UI renders a timeline from the rows.
**When:** any auditable trail — approvals today, ticket_events (2.4), and (later) capacity snapshots.
**Why here:** ADR-BE-012 + the shipped `*_approvals` tables are the codebase precedent; ADR-P-003 demands it for tickets (resolution note, not conversation).

### Pattern 3: Kind → per-kind state machine (JSM request-type pattern)
**What:** one entity, a `kind` column, one shared status vocabulary constrained by a DB CHECK, and a per-kind transition matrix owned by the service.
**When:** entities whose kinds differ in workflow, not in shape — tickets (task vs helpdesk). Matches how JSM maps request types to workflows.
**Trade-off:** a single CHECK is looser than per-kind CHECKs; the service matrix + tests carry the real invariant. Acceptable: entries already do exactly this (status CHECK + service transition guards).

### Pattern 4: Hexagonal feature slice (the house style)
**What:** domain types + sentinel errors → port interface → service struct → thin HTTP handler → pgx repository; wired in `cmd/server/main.go`; tested with testcontainers integration suites.
**When:** every new feature — tickets and staffing both follow it mechanically. New domains follow the per-domain folder convention (`domain/ticket/`, `services/ticket/`, `ports/ticket_repository.go`, `postgres/ticket_repository.go`).

### Pattern 5: Semantic token + shared component layering
**What:** CSS-variable semantic tokens (shadcn contract) + a frozen set of shared page components; pages consume both.
**When:** all v0.2 frontend work; the layer is what lets per-page passes stay small and consistent (4.1).

### Pattern 6: Role-scoped surfaces via pure predicates
**What:** sidebar visibility and page gates derive from the membership role through pure, testable predicates on the single declarative `navStructure`; backend stays authoritative.
**When:** the two new nav entries (Tickets, Availability) follow P-011 D-5 exactly; HR sees Availability (curator) but never Tickets-as-work or Review.

---

## 6. Data Flow

### Request flow: create an external helpdesk ticket
```
Customer asks → employee opens /tickets → POST /tickets
  → TicketHandler.Create (thin; parses org/user from context)
  → ticket.Service.Create
      • validates activity_id ∈ org, kind ∈ CHECK
      • helpdesk ⇒ derived commercial context exists (CTE) else ErrTicketRequiresContract
      • assigns assignee_id if provided (WG-membership check)
      • INSERT ticket (status 'open') + INSERT ticket_events ('created')  [same tx]
  → TicketRepository.Create (+ CTE-joined response: activity name, derived customer/contract)
  → { data: ticket }  → React Query cache key ['tickets', ...] invalidation
```

### Request flow: resolve + count for billing
```
POST /tickets/{id}/resolve { note }   → service: kind matrix check, set resolved_at,
                                        append 'status_changed' event with note (required for helpdesk)
GET  /tickets/counts?customer_id=&from=&to=
  → repo: tickets → activity chain CTE → contracts.customer_id → GROUP BY customer, period
  → customer detail page renders "Requests" counts (Economics pillar)
```

### Capacity flow
```
GET /capacity?activity_id=&week_start=
  → staffing.Service.CapacityForActivity
      • members = WG anchored at activity (wg_members)
      • per member: capacity = weekly_hours − Σ confirmed windows in week
      • workload = Σ submitted+approved time_entries.hours on activity subtree
  → staffing repo: two aggregations (windows; entries via subtree CTE)
  → utilization per member per week
```

### State management (frontend)
React Query with `['tickets', ...]` / `['staffing', ...]` keys; mutations invalidate; new pages follow the existing `api.ts` module convention (queryOptions/mutationOptions, `api<T>()` client, 401-refresh retry). No new state layer — the codebase pattern scales.

---

## 7. New vs Modified (integration-point manifest)

### New (backend)
| Artifact | Where | Purpose |
|----------|-------|---------|
| Migration `014_tickets.up.sql` | `migrations/` | tickets + ticket_events (+ optional `organization_memberships.weekly_hours` per 3.1) |
| `internal/core/domain/ticket/` | domain | Ticket, TicketEvent, requests, kind transition matrix, sentinel errors |
| `internal/core/ports/ticket_repository.go` | ports | Create/Get/List/Counts/Events ports |
| `internal/core/services/ticket/` | services | business logic incl. per-kind transitions + assignment validation |
| `internal/adapters/primary/http/ticket.go` | primary | `/tickets`, `/tickets/{id}`, `/tickets/counts` |
| `internal/adapters/secondary/postgres/ticket_repository.go` | secondary | SQL + derived-customer CTE |
| `internal/core/domain/staffing/` + service + port + handlers | all layers | availability windows + confirmation routing + capacity |
| `internal/adapters/secondary/postgres/staffing_repository.go` | secondary | windows/validity/expiry + workload aggregation |
| Tests: ticket + staffing service/handler/repo suites | testcontainers | BE-009 convention; transition-matrix tests per kind |

### Modified (backend)
| Artifact | Change |
|----------|--------|
| `cmd/server/main.go` | wire ticket + staffing services/handlers; register routes |
| `internal/core/ports/working_group_repository.go` | expose WG-members-for-activity lookup (assignment validation); possibly extend entry repos with a ticket-link column later (deferred) |
| `internal/models/models.go` | shared constants if DTOs land there (house style is per-domain now) |

### New (frontend)
| Artifact | Purpose |
|----------|---------|
| `web/src/api/tickets.ts`, `web/src/api/staffing.ts` | query/mutation modules |
| `web/src/routes/_authenticated/tickets/` | list (filters: kind/status/assignee/activity), detail (timeline from ticket_events), create/edit dialogs |
| `web/src/routes/_authenticated/availability/` | windows (declare/confirm/curate per role), expiry queues |
| `web/src/routes/_authenticated/capacity/` (or embedded) | per-activity/WG capacity view |
| `web/src/components/shared/{page-header,filter-bar,status-badge-variants,...}` | extracted primitives from 4.1 |
| Semantic tokens for statuses | `index.css` additions |

### Modified (frontend)
| Artifact | Change |
|----------|--------|
| `sidebar.tsx` navStructure | + Tickets (Track), + Availability (People) per P-011 D-5 matrix |
| `today-page.tsx` | compose owned open tickets by priority (P-003 consequence, P-004 rules: read-only, never blank) |
| Customer detail page | "Requests" counts section (Economics) |
| `status-badge.tsx` + per-page components | consume new semantic tokens |
| All pages | per-page polish phases (content regions only, shell untouched) |

### Explicitly NOT built (anti-feature guardrails)
Kanban/board/sprints · ticket subtrees · comment threads · estimates/velocity · ticket approvals (two-stage chain) · customer-side user accounts/portal · leave balances/accruals · capacity blocks on time-entry submission · per-ticket invoicing.

---

## 8. Suggested Build Order (dependency-respecting)

1. **UX foundation (tokens + shared components)** — smallest phase, unblocks everything user-facing. Addresses: token additions (4.1), shared component extraction, UI-SPEC consistency for the shell. *Avoids:* per-page passes drifting into ad-hoc styling; new pages shipping with new ad-hoc colors.
2. **Tickets backend** — migration 014 + domain + port + repo + service + handlers + tests. The centerpiece, and it depends on nothing new (activity tree, WG members, contracts all exist). *Avoids:* building UI on an uncommitted API shape.
3. **Tickets frontend + Today composition** — `/tickets` pages, sidebar entry (Track), Today's "my tickets" block (P-004 read-only composition rules).
4. **Staffing backend** — repo/service over the shipped 012 schema (+ `weekly_hours` decision), confirmation routing, capacity queries. Depends on 1 (no), 2 (no — parallelizable, but keep sequential per GSD), and on `wg_member` access.
5. **Availability + capacity frontend** — `/availability` (People pillar), capacity view, HR expiry queues, assignment-time warnings on WG surface.
6. **Per-page UX polish phases** (one phase per page) — each page phase adopts its own v0.1 UAT/verification debt (STATE.md list); order by traffic: Today/Time-Entries → Activities → Working Groups/Approvals → Customers/Contracts → Exports/Org. Runs on the 1–5 foundation.

**Ordering rationale:** 1 before 2–6 because every new page must be built on the frozen component/token layer (cheapest time to fix). 2 before 3 (API contract first). 2–5 are independent of each other except shared frontend primitives from 1; 3 and 5 both touch `sidebar.tsx` navStructure — do them once each, in their own phases, to keep D-5 predicates pure. 6 last because polish phases consume everything above and their UAT debt assumes stable surfaces.

**Research flags for phases:** Phase 2 (tickets) — needs the domain-design discussion of 2.1–2.5 validated with the user (activity anchor vs customer column is *the* decision); Phase 4 (staffing) — needs the `weekly_hours` source decision and the holiday-confirmation UX (who confirms what, where it renders); Phase 6 — each page needs its own sketch round (gsd-sketch), no research needed.

---

## 9. Scaling Considerations

| Concern | Now (single org, <1k users) | 10k users | 100k+ |
|---------|------------------------------|-----------|-------|
| Derived commercial CTE (tickets/entries) | Recursive CTE per lookup; depth < 6; `activities(parent_id)` indexed (BE-014 R-6 note) | Same; materialized path column if profiling shows cost — do NOT denormalize contract | Same |
| ticket_events append-only | Trivial | Index `(ticket_id, created_at)`; partition by month if >10M rows | Partition + archive resolved tickets |
| Counts queries (`/tickets/counts`) | CTE join per request, cached by React Query | Precomputed per-period materialized view (nightly) | Same + event-sourced counters |
| Capacity aggregation | Two queries (windows, entries) per view | Index `availability_windows(org_id,user_id,starts_on,ends_on)` (exists, 012); entry aggregation via `time_entries(user_id, entry_date)` (exists) | Materialized weekly capacity cubes |
| Availability windows | Declared intent, no locks | — | — |

**First bottleneck:** the derived commercial CTE join on every ticket read if the tree grows deep or counts queries scan large windows. Fix: `resolved_at` period indexes + the counts query restricted to `kind = 'helpdesk'` (the only kind that bills). **Second:** capacity aggregation re-scanning entry history per week — restrict to submitted/approved (already indexed via `time_entries(activity_id)` + status).

---

## 10. Anti-Patterns to Avoid

### Anti-Pattern 1: Storing `customer_id` on the ticket
**What people do:** "tickets belong to customers, put the FK on the ticket" — the simplest-looking design.
**Why it's wrong:** duplicates the commercial chain (D-3 violation); drifts on re-parenting/contract changes; creates a second source of truth that invoices and reports will disagree about; strands history when the tree moves.
**Instead:** derive customer/contract via the activity-chain CTE, exactly like entries.

### Anti-Pattern 2: Ticket-kind statuses as separate tables/columns
**What people do:** `ticket_task_status`, `ticket_helpdesk_status`, or two tables with disjoint state machines.
**Why it's wrong:** duplicates CRUD/handlers/UI for a 5-state vocabulary; the kinds differ in *transitions*, not vocabulary.
**Instead:** one `status` CHECK + service transition matrix per kind (Pattern 3).

### Anti-Pattern 3: Drifting toward a PM tool (boards, sprints, subtasks, estimates)
**What people do:** "since we have tickets, add a Kanban column…" — the strongest scope magnet in the product after leave-management.
**Why it's wrong:** VISION §8 + ADR-P-003 hard boundary; P-006 steering test fails on sight.
**Instead:** schema absence is the enforcement — no board columns, no parent_id, no estimate fields; revisit only via vision revision (V4 change requests as a *kind*, still not a board).

### Anti-Pattern 4: Availability machinery creep (balances, accruals, "days remaining")
**What people do:** capacity views invite counters — "X has 4 holiday days left".
**Why it's wrong:** ADR-P-008 D-5 rejects the whole accounting family; it is the payroll provider's computation.
**Instead:** windows are declared/confirmed/exported facts; capacity = hours − absences; never counters.

### Anti-Pattern 5: Availability as an enforcement gate
**What people do:** block time-entry submission during declared absence.
**Why it's wrong:** P-008 D-3 explicitly rejects blocking — people work during declared absence; blocks create exceptions and meta-work.
**Instead:** warnings at assignment time only; capture stays unconstrained.

### Anti-Pattern 6: Per-page UX rewrites that touch the shell/nav
**What people do:** "polish" turns into re-layouting the sidebar or renaming routes per page.
**Why it's wrong:** breaks the P-011 IA contract, role-visibility predicates, e2e selectors, and the shell wrapping from Phase 10-03; the shell is the stability boundary.
**Instead:** content-region-only changes; navStructure changes only when a surface ships (twice in v0.2, once each).

### Anti-Pattern 7: Over-normalizing ticket kinds into a catalog table
**What people do:** mirror `activity_kinds` for tickets because "orgs might want custom ticket types".
**Why it's wrong:** P-003 defines exactly two origins; a catalog implies extensibility the boundary forbids, and invites the PM-tool drift (Anti-Pattern 3).
**Instead:** closed CHECK; widen via migration if V4 needs `change_request`.

---

## 11. Sources

- Internal (HIGH): ADR-P-003 (tickets as capture layer), ADR-P-007 (activity ontology D-1…D-8), ADR-P-008 (availability D-1…D-5), ADR-P-011 (IA D-1…D-6), ADR-BE-014 (routing R-1…R-6), ADR-BE-004 (migrations), ADR-BE-012 (audit writes) — `hourglass-vault/decisions/`
- Internal (HIGH): `migrations/011_activity_ontology.up.sql`, `012_staffing_schema.up.sql`, `000_full_schema.up.sql` (`*_approvals`, status CHECKs), `internal/core/domain/{activity,working_group}/`, `internal/core/services/{time_entry,expense}/` (`IsWGManager`, `resolveManagerStage`), `web/src/index.css`, `web/src/components/layout/sidebar.tsx`
- External (MEDIUM — official docs, fetched 2026-08-01): Atlassian Jira Service Management docs — request types map to workflows/issue types (support.atlassian.com/jira-service-management-cloud); Float Help Center — schedule = allocations + time off vs capacity (support.float.com); shadcn/ui Theming — semantic CSS-variable tokens + `@theme inline` (ui.shadcn.com/docs/theming)
- External (MEDIUM, synthesis): helpdesk platforms bill per agent; per-request billing is the agency/managed-services pattern (Zendesk/Freshdesk pricing models) — informing the derived-counts design in 2.5, not a feature to copy
- Confidence per the classify-confidence seam: `websearch`/`webfetch` providers → LOW unverified, MEDIUM where cross-checked against the codebase (token architecture verified against `index.css`; request-type pattern cross-checked against ADR-P-003's kind split; capacity model cross-checked against ADR-P-008 schema). Internal claims → HIGH (read from code and ADRs).

---
*Architecture research for: Hourglass v0.2 milestone (ticket ontology on the activity tree, availability, UX polish)*
*Researched: 2026-08-01*
