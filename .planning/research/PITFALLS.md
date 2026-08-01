# Pitfalls Research

**Domain:** Adding ticket ontology (internal tasks + helpdesk, per-customer request billing) and availability/capacity features to an existing time/expense tracking app, plus a page-by-page UX redesign pass
**Researched:** 2026-08-01
**Confidence:** MEDIUM (multiple corroborating industry sources; project-specific grounding from v0.1 retrospective, ADR-BE-004, migrations 011–013)

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: Unified-ticket modeling trap — polymorphic references and kind-specific field sprawl

**What goes wrong:**
The "unified ticket entity" (one `tickets` table with `kind IN ('task','helpdesk')`) is the right call, but implementation drifts into one of two disasters: (a) polymorphic associations — time entries, expenses, and approval rows reference "the ticket" via a generic `(ticket_id, ticket_kind)` pair or two nullable FKs (`task_id`, `helpdesk_ticket_id`), which destroys referential integrity and forces every query to branch on kind; or (b) over-normalization — separate `tasks` and `helpdesk_tickets` tables plus a shared "ticket" view, so every new surface (list, count, approval queue) needs a UNION and per-kind service logic.

**Why it happens:**
Fowler's P of EAA documents the underlying tension: Single Table Inheritance (one table + kind column) is simplest for shared queries; Class Table Inheritance (one table per kind) is cleaner per-kind but costs joins and polymorphic FK handling. With an existing approval engine (two-stage, immutable history) and billing intent (request counts), teams split the difference reactively: they start unified, then add a kind-specific column, then a kind-specific table, ending in a hybrid that is neither.

**Consequences:**
- Time entries can't FK-reference tickets cleanly → manual integrity or a new join table later (schema rewrite).
- Request counting (billing) queries become kind-branching messes; counts disagree between surfaces.
- The v0.1 lesson repeats: the activity ontology rewrite (projects/subprojects → recursive `activities`) was the costliest phase of v0.1.

**How to avoid:**
- One `tickets` table with `kind` CHECK, shared core columns (title, description, status, priority, requester, org, activity_id), and **nullable kind-specific columns** (`customer_id`, `channel`, `requested_at` for helpdesk; `assignee_role` semantics shared). Mirror the pattern 012 used for staffing: additive, kind-constrained via CHECKs, no polymorphic FKs.
- **Never** put `ticket_kind` next to a FK. If both task and helpdesk tickets must be referenceable by one column, add the join table in the same migration (do it once, up front) — do not ship polymorphic FKs "temporarily".
- Decide the time-entry↔ticket link at schema time: time-entry items keep `activity_id`; tickets additionally get their own optional reference — do NOT overload `activity_id` with ticket semantics.
- Reuse the existing activity scoping: `ticket.activity_id` walks the same recursive tree (CTE ancestry already exists) — do not build a second scoping mechanism.

**Warning signs:**
- Any table with two nullable FKs where exactly one is expected ("xor columns").
- Service methods named `GetTicketOrTask` or branching on kind inside repos.
- A "ticket view" over two tables appearing in design discussions.
- Approval history rows needing a `source_kind` discriminator.

**Phase to address:**
Ticket ontology phase (schema + service design) — this is the phase where the entity shape is locked; get it right before any ticket UI or billing count exists.

---

### Pitfall 2: Status machine sprawl — a new status dimension per ticket kind

**What goes wrong:**
Tickets get their own status machine (`open → in_progress → pending_customer → resolved → closed`), tasks get another (`todo → in_progress → review → done`), and each kind accretes custom states ("waiting on third party", "awaiting manager approval"). Within one milestone you end up with two or more status machines that drift apart, and every ticket list/report/approval view must handle both.

**Why it happens:**
Atlassian ecosystem post-mortems (UWaterloo, schillig.uk) call this workflow sprawl: it "becomes a problem when the amount of variation no longer earns its keep in operational clarity, control or reporting value." It happens because the helpdesk kind arrives with ITSM-shaped instincts (SLA states, pending states) while the task kind inherits dev-shop instincts — and no one declares them the same machine. SparrowDesk: "too many status options make things worse — your team needs a simpler system if they need a cheat sheet to track statuses."

**Consequences:**
- Reports break (counting per status requires per-kind maps).
- Automation/transitions confuse agents; training cost rises.
- Billing counts (request counting) become ambiguous: does a reopened ticket's status reset count?

**How to avoid:**
- **One shared status machine** for all ticket kinds, sized like the existing entry workflow (the app already ships a small, proven machine: draft → submitted → pending_manager → pending_finance → approved/rejected). Adopt a similarly small core: `open → in_progress → pending → resolved → closed` plus `reopened` if needed.
- Separate **internal status** from **customer-facing status** (UWaterloo JSM best practice) — one stored field, two rendered labels, never two machines.
- Do not add a status "per customer" or "per priority". Priority is a field, not a status.
- Any new status requires a one-line ADR entry; govern additions.

**Warning signs:**
- Two `status` CHECK constraints with overlapping but unequal value lists.
- Ticket list filters that need `if kind == task … else …` for status.
- The word "pending" used with two different meanings across kinds.

**Phase to address:**
Ticket ontology phase — status machine is part of the ontology ADR, not something the UI polish phase invents later.

---

### Pitfall 3: Scope creep into full ITSM (SLA timers, escalation chains, customer portal, notifications)

**What goes wrong:**
"External helpdesk tickets" starts as "customers ping, we log a ticket, we count requests per customer for billing." It ends with: SLA response/resolution timers per priority, escalation levels, assignment queues, email notifications, a customer portal, knowledge base, and per-agent workload views. Each is defensible alone; together they are a second product (a helpdesk platform) layered onto a time tracker.

**Why it happens:**
The ticketing category's gravity well is strong — every vendor article (HappyFox, Suptask, Zendesk) lists SLA tracking, escalation, routing, portals, and AI as "must-have features." When the feature exists in the market, it feels like a gap. The existing two-stage approval engine makes "one more workflow" look cheap, so escalation logic sneaks in via the approval path.

**Consequences:**
- v0.2 doubles in size; the milestone's three goals (tickets, availability, UX polish) compete for budget.
- SLA timers drag in business calendars, pause/resume semantics, and breach automation — each a hidden subsystem.
- The availability phase (capacity views) and UX polish pass get starved; polish-before-settling risk rises.

**How to avoid:**
- Write the anti-scope line into the ticket ADR: **no SLA timers, no escalation chains, no customer portal, no notification engine, no KB in v0.2.** The only mandated metric is request-count-per-customer for billing.
- Escalation = the existing two-stage approval path or a simple `priority` bump — nothing new.
- Treat "customer pings" as a manual or future channel: the v0.2 helpdesk ticket is created by an employee on behalf of a customer (the app has no inbound email/webhook).

**Warning signs:**
- Requirement sentences containing "when" clauses ("when SLA breaches…", "when ticket is stale…").
- New tables for escalation rules, schedules, or notification templates appearing in the schema draft.
- A "portal" or "customer view" route in the frontend plan.

**Phase to address:**
Ticket ontology phase (scope gate) + requirements phase — the anti-feature list belongs in REQUIREMENTS.md, enforced at roadmap creation.

---

### Pitfall 4: Request-counting traps — ambiguous count semantics for billing

**What goes wrong:**
Request-count-per-customer for billing sounds trivial (count tickets), but every counting rule has edge cases that silently change invoices: (a) thread vs ticket — one customer thread may span several tickets or one ticket may contain multiple requests; (b) reopen semantics — a reopened ticket counted at each close inflates counts; (c) internal vs customer-initiated — tickets created by employees ("helpdesk" logged internally) counting toward the customer's bill; (d) attribution drift — ticket reassigned from customer A to customer B after creation, retroactively changing the count; (e) counting window — calendar month vs billing period mismatch; (f) test/spam tickets.

**Why it happens:**
Vendor definitions are deliberately loose ("each customer query — be it an email or a phone call — is a ticket" — Freshdesk; tickets originate "from email, Help Center, chat, phone…" — Zendesk). Without an in-house definition, engineers pick "count rows where kind=helpdesk and created in month X," which is defensible until the first customer disputes a bill.

**Consequences:**
- Billing disputes with external customers; internal customers (already in the model!) get miscounted against contracts.
- Fixing counting after data exists means backfill migrations and retroactive invoice corrections.

**How to avoid:**
- Define **one** counting rule at schema time and write it into the ADR: e.g., *"A billable request = a ticket of kind `helpdesk`, attributed to the customer stored on the ticket at creation, counted once in the calendar month of creation, excluding tickets created by employees as internal follow-ups (marked `is_internal=true`)."*
- Store `requester_customer_id` (and org snapshot) at creation, immutable — no silent re-attribution; reassignment requires an explicit audit-visible action.
- Make the count a **derived query** over the immutable ticket row (created_at, kind, customer_id), never a stored counter column.
- Do not count ticket status transitions or updates — only creations.

**Warning signs:**
- Two tickets with the same `created_at` and customer counted differently by different queries.
- A `request_count` column appearing in the schema (stored counts go stale).
- Update endpoints that allow changing `customer_id` after creation.

**Phase to address:**
Ticket ontology phase (billing semantics) — the counting rule is a schema/domain decision, and the exports phase (existing CSV/XLSX surface) will surface the numbers, so get it right before wiring exports.

---

### Pitfall 5: Append-only migration traps — enum expansion, CHECK rework, and backfills on live data

**What goes wrong:**
The v0.2 schema work (tickets table, ticket status/kind enums, extending availability) runs into PostgreSQL append-only migration sharp edges: (a) `ALTER TYPE … ADD VALUE` — the new value **cannot be used within the same transaction** that adds it (official docs: "the new value cannot be used until after the transaction has been committed"), so the migration that adds `'helpdesk'` to an enum can't also insert helpdesk rows; (b) CHECK constraint expansion (e.g., adding `'task'`/`'helpdesk'` to a status CHECK) requires **drop + recreate** — PostgreSQL cannot alter CHECKs in place; (c) adding a NOT NULL column to a populated table without a default fails or locks the table; (d) backfills written into the same migration as the schema change make the migration non-atomic and slow.

**Why it happens:**
The codebase already hit these: migration 012 documents "PostgreSQL cannot alter CHECK constraints in place — drop and recreate" for the `hr` role, and migration 013 was needed as a **forward fix** to relabel activity kinds without violating ADR-BE-004's append-only rule. The habits that cause it: writing migrations as "final schema" instead of "next state," and testing only against empty testcontainers schemas.

**Consequences:**
- A migration that works on empty dev DBs but fails or locks in production.
- Enum values that can't be referenced by the app until the next migration runs (deploy-order bugs: app deploys before migration 2 of 2).
- Violating ADR-BE-004 (editing applied migrations) — the v0.1 retrospective calls cycle tests the deterministic catcher of these defects.

**How to avoid:**
- Per ADR-BE-004: new numbered files from max+1, up/down pairs, and testcontainers **up → down → up cycle tests** for every schema-changing migration (proven pattern from 011/012/013).
- Add enum/status values via `ALTER TYPE ... ADD VALUE` in **their own migration**, and never use the new value in the same migration. For CHECK-based statuses (the codebase uses CHECKs, not enums), drop + recreate the constraint, and reference new values only in later migrations.
- New columns: nullable first, backfill in a second migration, then set NOT NULL (three-step) — or keep nullable if the domain allows.
- Before writing a migration, grep existing data shape: `availability_windows` already exists (012) with `kind` CHECK `('holiday','permit','medical','unavailable')` and `status` CHECK `('declared','confirmed')` — extending these means CHECK rework, and `kind` naming here **collides** with ticket `kind`; name the ticket column carefully (`ticket_kind` or use the table's context).

**Warning signs:**
- A single migration that both changes a type/CHECK and inserts rows using the new value.
- `ALTER TABLE ... ADD COLUMN ... NOT NULL` without `DEFAULT` on a populated table.
- Any edit to an existing `.up.sql` file (append-only rule) — including "small fixes."
- Migration names claiming "fix" of an earlier migration (013 pattern) without the corresponding cycle test.

**Phase to address:**
Ticket ontology phase and availability phase (each schema migration) — every migration plan must include the cycle-test step; this is a per-plan verification item, not a one-off.

---

### Pitfall 6: Capacity-planning over-promise — planning against capacity instead of availability, and absence/availability conflation

**What goes wrong:**
The availability feature ships as "resource/capacity views per activity/WG," and one of these happens: (a) the view shows total capacity (headcount × working days) as "available," ignoring absences — Productive.io: "the practical danger is planning against capacity instead of availability… book against the full figure and the person quietly runs over"; (b) absence data is treated as availability (the `availability_windows` table stores **absences**, not free time — conflating "on holiday" with "available"); (c) double-booking semantics are unspecified — should the system hard-block overlapping work or warn?; (d) availability is computed on the fly per request, so two managers see different numbers.

**Why it happens:**
Capacity/availability is a domain where the terms are loaded differently everywhere (ServiceNow forum: availability = spare capacity + time off; Saviom: availability = capacity − allocations − absences; Productive: availability = capacity − absence − bookings). The schema already encodes the trap: the table is named `availability_windows` but stores **absence windows** (`kind: holiday/permit/medical/unavailable`). Absence is an input; availability is a derived output.

**Consequences:**
- Managers over-commit people who are on holiday → the "quiet overrun" surfaces as missed deadlines (Productive).
- Two views disagree → trust in the feature collapses; it gets abandoned.
- Absence statuses (`declared` vs `confirmed`) leak into capacity math before confirmation — or worse, after.

**How to avoid:**
- Model it as the industry formula, explicitly: **capacity = working days × hours − national holidays; availability = capacity − absences − commitments**. Absences are stored facts (with `declared`/`confirmed` status in `availability_windows`); capacity and availability are **derived, never stored** — matching the codebase's "derived-never-stored" pattern from Phase 9 (GetAncestry/ResolveCommercialContext/ResolveBillability).
- Define double-booking semantics in the ADR: e.g., *"overlapping commitments render a warning, not a hard block"* (hard blocks need transactional conflict handling — a whole extra subsystem).
- One derived computation in the service layer (pure, testable — like the pillar visibility predicates in ADR-P-011), consumed by all views; no per-view ad-hoc math.
- Start with **absence surfacing + capacity view**, not scheduling: the milestone says "resource/capacity views," not booking. Keep it read-mostly.

**Warning signs:**
- A column named `available_hours` or `capacity` in the schema (stored derivation).
- Capacity math that doesn't subtract `availability_windows` rows with status `confirmed`.
- Two UI surfaces showing different free-hours for the same person/period.
- Any write endpoint for "availability" — v0.2 should only write absences.

**Phase to address:**
Availability phase (schema/service design) — the derived-never-stored rule and double-booking policy must be locked before the first capacity view is built.

---

### Pitfall 7: UX-polish-pass traps — redesign churn, breaking existing flows/e2e tests, token inconsistency

**What goes wrong:**
The page-by-page polish pass (sketch-driven, one phase per page) triggers one of: (a) **redesign churn** — each page gets re-sketch loop after loop because "polish" has no success criteria; NN/g: "never make radical changes when minimal adjustments will suffice" and "customers balk at change, even when the new design is clearly better"; (b) **flow breakage** — moving buttons, renaming actions, or restructuring a page breaks the approval flows, keyboard paths, and muscle memory the e2e suites encode (Playwright suites per domain exist: auth, approvals, time-entries, expenses, activities, WGs…); (c) **token inconsistency** — pages polished in different phases drift (spacing, radius, semantic colors), recreating the "incohesive patchwork" NN/g warns about; (d) **polishing before the data model settles** — polishing the time-entry or activity pages while the ticket ontology phase may touch shared components (entry pickers, status badges) means double work.

**Why it happens:**
v0.1 shipped the IA (Phase 10) with a Header+Body shell and pillar sidebar, and 25 UAT scenarios + 3 human verification reviews are still open — the polish pass is partially "make it look finished" pressure. Without token governance and a "lock the design system first" step, each page phase becomes a mini-redesign. XB Software: "redesigns aren't just visual, they're psychological… when you disrupt learned behaviors without clear signposting, even elegant upgrades can fail."

**Consequences:**
- Multi-hundred-minute plans (v0.1's Phase 10 P06 ran 453 min — big UI+API surface plans dominate) balloon further.
- e2e regressions erode the green suite that v0.1's whole test strategy depends on.
- The v0.2 features (tickets, availability) land on a moving design foundation.

**How to avoid:**
- **Design-token/system pass first**: one small phase that locks tokens + shared components (badges, tables, dialogs, empty states) BEFORE per-page phases. Every page phase then consumes tokens instead of inventing styles.
- Per page: sketch **2–3 options max** → agree → implement → verify against existing e2e suites; e2e specs are the contract — if a flow changes, the test changes in the same plan.
- **Fold the 25 UAT scenarios + 3 human reviews into the matching page phases** (the milestone already plans this) — do not create a separate "verification debt" phase at the end.
- Freeze the data model before polishing pages that the ticket/availability phases will touch (time-entry pickers, activity selectors, status badges). Order: ontology/availability schema first, then polish the pages they don't touch, then polish shared surfaces after.
- No "boredom redesigns" (NN/g): every change must map to a UAT scenario, a usability defect, or a token-consistency fix.

**Warning signs:**
- Sketch rounds exceeding 3 per page with no new information (churn).
- Playwright specs edited "because the design changed" without a corresponding behavior assertion change.
- Inline hex colors/arbitrary `rounded-*` values appearing in page components instead of tokens.
- Polish phases touching components that ticket/availability phases also touch (co-scheduling risk).

**Phase to address:**
Every UX polish phase + a lead-in design-system/token phase. The roadmap should place the token/component phase before the first page polish phase.

---

### Pitfall 8: Sketch-driven back-and-forth process traps — unbounded revision loops and unfocused feedback

**What goes wrong:**
The "gsd-sketch options → agree → implement → verify" loop degenerates: (a) the user keeps asking for variations because no round is ever declared "agreed" (no acceptance criterion per round); (b) options overload — 5–6 sketches per round create choice paralysis (NN/g: "Simplicity wins over Abundance of Choice"); (c) sketch feedback mixes levels — visual tweaks, new features, and data-model complaints in one reply, so the implement step can't be planned; (d) the loop is used for feature discovery instead of polish (scope mixing); (e) decisions aren't logged, so the same debate reopens next phase.

**Why it happens:**
Sketch-driven work with an AI produces cheap revisions, so the natural equilibrium is "one more round." NN/g's iterative-design guidance is explicit: iteration is only valuable with user feedback and agreed constraints — unbounded iteration without criteria is churn. The v0.1 retrospective's "lock decisions in ADRs before execution — downstream phases become mechanical" is the same principle applied to design.

**Consequences:**
- A single page consumes an entire phase budget (453-minute-plan pattern repeats).
- Feature creep leaks into polish (a sketch comment spawns a new requirement).
- Inconsistent outcomes across pages because each round's decision context is lost.

**How to avoid:**
- Fix the loop contract per page: **2–3 options per round, one round of revisions max, then agree or pick one**; the "agree" step requires an explicit criterion (matches UAT scenario X, preserves flow Y, uses token Z).
- Sketch feedback must be triaged into three buckets before implementation: visual polish (this phase), behavior change (this phase if small, else new requirement), data-model change (escalate to ontology/availability ADR — never implement on the fly).
- Log every agreed decision in the page's UI-SPEC (the v0.1 pattern) + ADR when it touches structure; reopening a settled page decision requires an ADR entry.
- Route "new feature" requests from sketch rounds into the milestone's requirements process, not the page phase.

**Warning signs:**
- Sketch round notes repeat the same unresolved point (signal: no agreement criterion).
- A page phase's plan contains tasks that add new API endpoints (scope mixing — polish phases should be UI-only).
- More than 3 sketch artifacts produced per page.

**Phase to address:**
Every UX polish phase (process contract) — the roadmap should state the loop contract once, in the design-system phase, and each page phase inherits it.

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Polymorphic FK (`ticket_id` + `kind` columns) to link entries/approvals to tickets | Avoids a join table now | Every query branches on kind; integrity unenforceable; likely rewrite | Never — adds the join table up front in the ticket migration |
| Second status machine for helpdesk tickets | Matches "industry" helpdesk UX | Reports, filters, approvals all fork per kind | Never — one core machine, kind-scoped views |
| Stored `available_hours`/`capacity` snapshot on memberships | Fast dashboards | Goes stale; two views disagree; backfill migrations | Only for read-heavy dashboards with an explicit refresh job — and even then prefer derived |
| Counting requests via "count rows" in the export handler | No schema impact | Rule drifts between surfaces; billing disputes | Never — one derived count function in the service layer |
| Polishing a page before the token pass | Feels productive | Reworked in every later phase; inconsistency | Never — token/design-system phase first |
| Editing an applied migration "to fix a small bug" | Instant local fix | Breaks append-only history (ADR-BE-004); cycle tests fail | Never — new forward migration + cycle test (the 013 pattern) |
| Reusing `activity.kind='task'` semantics for tickets | Familiar vocabulary | Collides with activity kinds (013 relabeled subproject tasks to phase; the term is taken) | Never — use `ticket` vocabulary, distinct from activity kinds |

## Integration Gotchas

Common mistakes when connecting the new features to the existing system.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Tickets ↔ activity tree | Storing a denormalized `activity_name` snapshot and filtering tickets by tree via parent walk per row | Reuse the Phase 9 recursive CTE ancestry; ticket holds `activity_id`, tree scoping is one `GetAncestry` join (same pattern as entries) |
| Tickets ↔ time entries | Adding a nullable `ticket_id` to `time_entry_items` with no integrity semantics | If linking is required, a proper `ticket_id` FK with the "one ticket, many entries, one activity" rule stated in the ADR; skip the link entirely in v0.2 if billing only needs counts |
| Tickets ↔ approval workflow | Building a third approval stage machine for helpdesk tickets | Reuse two-stage pattern only if tickets need approval; otherwise no approval (tickets are tracked work, not entries) — decide in the ADR |
| Availability ↔ org memberships | Using membership `valid_from/valid_until` (012) as availability | Membership validity gates employment; `availability_windows` feeds capacity math; both feed derived availability — keep them distinct inputs |
| Request counts ↔ exports | Computing counts inside the CSV/XLSX handler | One derived count function in the service layer, consumed by exports, reports, and UI (Phase 9 "derived-never-stored" pattern) |
| New pages ↔ pillar sidebar (ADR-P-011) | Adding nav entries ad hoc per phase | Extend the single declarative `navStructure` with role-scoped visibility predicates; tickets under Work pillar, availability under People |
| Frontend API ↔ backend for tickets | Writing new api.ts modules that duplicate the auth-refresh/`api<T>()` envelope handling | Use the existing `api<T>()` helper (cookie auth + refresh-on-401) and `queryOptions`/`mutationOptions` patterns from `web/src/api/auth.ts` |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Counting requests by scanning all tickets per export/report | Export requests slow as ticket volume grows | Index `(created_at, kind, customer_id)`; count in SQL with the derived function; paginate exports | Thousands of tickets per customer |
| Capacity views computing per-person availability with N+1 absence queries | Capacity page latency per activity/WG member | Single SQL aggregate: capacity − sum(absence overlap) − sum(commitments) per person; one query for the whole WG view | ~100 members in a WG view |
| Ticket lists joining ancestry CTE per row | List page slow with deep trees | One query that joins `activities` ancestry once (like entries already do); avoid per-row GetAncestry calls | Deep activity trees + hundreds of tickets |
| Enum ordering with `BEFORE`/`AFTER` additions (PG docs) | Slower enum comparisons on status filters | Append values at end; if ordering matters, use a display-sort column | Only pathological; noted for completeness |
| Availability window overlap math done in Go per user | Sluggish People-pillar pages | Overlap via SQL range conditions on `(org_id, user_id, starts_on, ends_on)` index (already exists from 012) | Many windows per user over long ranges |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Helpdesk tickets exposing internal notes to customer role | Internal analysis, costs, and employee data leak to external customers | `customer` role sees only a customer-facing projection of the ticket (separate rendered field, like the "internal vs customer-facing status" rule); internal notes never leave the API for that role |
| Request-count queries lacking org scoping | Cross-org count leakage in exports | Every count function takes `org_id` from the authenticated context; count tests assert org isolation (the codebase already centralizes this in handlers) |
| Ticket attribution stored mutably | Billing manipulation (re-attribute tickets to avoid/trigger counts) | `customer_id` set at creation, immutable; reassignment is an audit-visible action, never a silent update |
| Availability windows exposing medical certificate refs | `certificate_ref` (012, INPS protocol) leaking to managers beyond HR | HR role only for `medical` kind detail; managers see holiday/permit only — enforce in the service layer, not the UI |
| Capacity views leaking individual availability across orgs | Org-B users' schedules visible to org-A managers | WG/activity scoping reuses the existing role predicate system (ADR-P-011); test cross-org denial |

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Redesign churn per page (no agreement criterion) | Users learn a new layout every release; managers see no progress | 2–3 sketch options, one revision round, explicit agree criterion per page (Pitfall 8) |
| Status vocabulary differs between task and helpdesk tickets | Users need a cheat sheet (SparrowDesk) | One status machine, kind-scoped labels (Pitfall 2) |
| Capacity view showing raw capacity (not availability) | Managers over-commit people on holiday; deadlines missed silently | Show derived availability with absence rows visible and excluded from free hours (Pitfall 6) |
| Customer-facing ticket status showing internal stages | Customers confused about progress | Internal status field, customer-facing label projection |
| Token drift across pages polished in different phases | App feels like a patchwork (NN/g) | Design-token phase first; every page phase consumes tokens (Pitfall 7) |
| "Today" landing not updated for ticket/absence actions | New features invisible in the daily landing | Extend Today surface deliberately in the relevant phases, not as an afterthought |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Ticket ontology:** Often missing the immutable customer attribution + counting rule — verify a single ADR sentence defines exactly what counts as a billable request, and a test asserts it.
- [ ] **Ticket status machine:** Often missing the internal-vs-customer-facing split — verify the customer role's ticket view shows only the projection.
- [ ] **Request counts:** Often missing org scoping and test/exclusion rules — verify counts exclude internal follow-ups and are org-scoped in integration tests.
- [ ] **Migration 014+:** Often missing the up/down/up cycle test — verify every new migration has the testcontainers cycle test (v0.1 lesson).
- [ ] **Availability views:** Often missing absence subtraction — verify the capacity view test includes an employee with a confirmed absence window and asserts reduced availability.
- [ ] **availability_windows:** Often missing the `declared` vs `confirmed` distinction in capacity math — verify only confirmed absences (or both, per ADR) feed availability, consistently.
- [ ] **UX polish page:** Often missing e2e updates in the same plan — verify Playwright specs changed only alongside behavior assertions, and the suite stays green per phase.
- [ ] **UAT debt:** Often deferred again — verify the 25 v0.1 UAT scenarios + 3 human reviews are mapped into page phases, not a trailing "verification debt" phase.
- [ ] **Sidebar:** Often missing role-scoped nav for new surfaces — verify tickets/availability entries are in `navStructure` with pure visibility predicates (ADR-P-011), not hardcoded per-user checks.

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Polymorphic FK shipped for tickets | HIGH (schema rewrite) | Add the join table in a new migration with a data backfill; keep old columns as deprecated reads until queries are migrated; cycle test up/down |
| Two status machines shipped | MEDIUM | Forward migration consolidating status values (013-style relabel), mapping table in the migration, report/UI label changes in same phase |
| Stored availability snapshot drifted | MEDIUM | Delete the snapshot column (or stop writing it); switch views to the derived service function; reconcile displayed values in release notes |
| Counting rule ambiguity discovered post-billing | HIGH (invoice disputes) | Freeze the rule in an ADR retroactively; recompute counts from immutable ticket rows (created_at/kind/customer_id); document the window of ambiguity honestly |
| e2e suite broken by a polish phase | MEDIUM | Revert the page to previous commit, re-apply polish with e2e assertions in the same plan; treat suite green as the phase's done-criteria |
| Sketch loop stalled (no agreement) | LOW | Invoke the loop contract: pick one of the 2–3 options by default, log the decision, move to implement |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Unified-ticket modeling trap | Ticket ontology phase (schema + ADR) | Integration test: entries/approvals reference tickets only via the join/FK, no polymorphic columns |
| Status machine sprawl | Ticket ontology phase (status ADR) | Unit test: single status transition table for both kinds; no per-kind status enums in schema |
| ITSM scope creep | Requirements/roadmap phase (anti-feature list) | Milestone audit: no SLA/escalation/portal tables or routes in the v0.2 diff |
| Request-counting traps | Ticket ontology phase (billing ADR) | Integration test: count function excludes internal tickets, is org-scoped, counts creation-month once |
| Append-only migration traps | Every schema phase (ticket + availability) | Cycle tests green per migration; no edits to applied migrations |
| Capacity over-promise / absence conflation | Availability phase (derived-never-stored ADR) | Unit test: availability = capacity − confirmed absences − commitments, computed in service, no stored availability columns |
| UX polish traps (churn/tokens/e2e) | Design-system/token phase first, then per-page phases | Token audit per page; e2e green per phase; ≤3 sketch rounds per page |
| Sketch-loop process traps | Design-system phase (loop contract) + each page phase | Page plan is UI-only (no new API endpoints); decision log in UI-SPEC |

## Sources

- Nielsen Norman Group — "Radical Redesign or Incremental Change?" (2015-02-08, nngroup.com/articles/radical-incremental-redesign/) — redesign disruption, incremental vs overhaul, "users balk at change". [HIGH for the cited claims — fetched]
- Nielsen Norman Group — "Fresh vs. Familiar: How Aggressively to Redesign" (2018) — switching cost / familiarity. [MEDIUM — search-verified]
- XB Software — "How to Redesign a Legacy UI Without Losing Users" (2025-08-07) — learned-behavior disruption, signposting. [MEDIUM — search-verified]
- Halo Lab — "Product Redesign: Motivations, Pitfalls" (2025-10-23) — well-intentioned updates reducing visibility. [MEDIUM — search-verified]
- University of Waterloo Atlassian — "Don't Let Your Jira Workflows Become a Maze" (2025-07-25) — workflow sprawl causes; JSM status/SLA interplay; internal vs customer-facing statuses. [HIGH for cited claims — fetched]
- SparrowDesk — "Why most support ticket lifecycles fail" (2025-08) — too many status options; lifecycle stages; internal notes vs public replies; premature closure. [MEDIUM — fetched]
- schillig.uk — "Reducing Workflow Sprawl in Jira Service Management" — sprawl earns-its-keep test. [LOW — snippet only, page 404 at fetch]
- Productive — "Resource Availability – A Guide for Professional Services" (2026-07-13) — capacity vs availability formula; planning-against-capacity danger; overbooking/burnout. [HIGH for cited claims — fetched]
- Saviom — "Resource Availability" / "Resource Forecasting vs Capacity Planning" — availability = capacity − allocations − absences; optimism bias ignoring absences. [MEDIUM — search-verified]
- Resource Guru — "Resource capacity planning guide" (2026-03) — booking clashes, waiting list, capacity calculation (contracted hours − vacation − BAU). [MEDIUM — search-verified]
- Planview — "Avoid Common Resource Capacity Planning Pitfalls" (2024-01) — visibility/integration pitfalls. [MEDIUM — search-verified]
- PostgreSQL Documentation 18 — `ALTER TYPE` (ADD VALUE transaction restriction; BEFORE/AFTER ordering perf) — official docs, fetched. [HIGH]
- PostgreSQL Wiki — "Don't Do This" — timestamptz, NOT IN, CHECK habits. [HIGH for cited claims — fetched]
- Freshdesk Support — "What is a ticket?" — ticket-as-query definition. [MEDIUM — search-verified]
- Zendesk Developer Docs / Zendesk blog — ticket channel origins; "from support requests to tickets". [MEDIUM — search-verified]
- HappyFox — "10 Common Ticketing System Mistakes" (2026-02-27) — categories/statuses/priorities, SLAs, reporting, training. [MEDIUM — fetched]
- IT Crash Course — "Do's and Don'ts of Ticketing Systems" (2025-06-29) — every issue ticketed, internal notes discipline, unassigned tickets. [MEDIUM — fetched]
- Martin Fowler, P of EAA — "Class Table Inheritance" (2003-03-05, martinfowler.com) — one-table-per-class vs single-table inheritance tradeoffs. [HIGH for the pattern taxonomy — fetched]
- Bill Karwin — *SQL Antipatterns* — "Polymorphic Associations" antipattern (xor-columns FKs). [MEDIUM — canonical reference from training]
- Project-internal: `.planning/RETROSPECTIVE.md` (v0.1 lessons: cycle tests, REQUIREMENTS traceability, P06 453-min plan), `migrations/011–013` (activity ontology, forward-fix pattern, staffing schema), `hourglass-vault/decisions/backend/ADR-BE-004` (append-only migrations), ADR-P-008/ADR-P-011 (staffing schema, pillar IA). [HIGH — primary source, read directly]

---
*Pitfalls research for: Hourglass v0.2 milestone (ticket ontology + availability + UX polish on an existing app)*
*Researched: 2026-08-01*
