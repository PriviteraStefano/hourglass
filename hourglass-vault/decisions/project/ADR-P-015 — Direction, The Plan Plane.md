---
tags: ["adr", "idea-layer", "capture", "structure", "insight", "direction"]
---

# ADR-P-015 — Direction, The Plan Plane

**Status:** Proposed
**Date:** 2026-08-08
**Operationalizes:** [[VISION]] §5 (Control) · §6 V3 (planning surfaces) · **Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 14–15, D-Q … D-AA) · **Extends:** [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] (three-plane ontology, one plane up) · **Decided by:** D-R … D-AA (Part 14), D-13-01..34 (Phase 13 locked) · **Implemented by:** [[ADR-BE-018 — Direction & Org Settings Encoding]] (backend encoding — schema, vocabularies, claim lock)

---

## Context

The ontology round closed the three-plane model: **direction** (the plan — what should you work on, *before* the work), **facts** (time entries — what you worked on, *during*), **coverage** (the label — who pays for it, *after*). [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] established the cardinal principle for the last two: *captured effort is a fact, coverage is a decision, the decision never rewrites the fact.* Direction completes the timeline — the same philosophy applied prospectively:

> **The plan never rewrites the fact.** Deviations are data, not violations.

Stefano pulled direction into the v0.2 ontology round (D-Q): the model (entity, states, refs) is fixed *now* so every ADR drafts against the final three-plane ontology — deferring would mean P-013's origin refs and P-004's Today composition were drafted against a two-plane model and reworked the moment the third plane landed. The build is staged (Phase 13 backend, Phase 19 surfaces), but the record of truth lands with this ADR.

The reality direction must serve is deviation, not perfection. The manager's ideal — "always present and always perfect" — deviates eight ways (under-run, preemption, backlog mode, interruption, over-run, evaporation, partial direction, defiance). Every one of those is **data**, recorded and visible, never a violation flagged by the model.

## Decision

### The third plane: direction → facts → coverage

| Plane | Content | Written by | Mutability |
|-------|---------|-----------|------------|
| **Direction** | the plan (who, which activity, how much, when) | manager or self (D-S) | **mutable as a chain** — supersede-only writes, history preserved (D-13-04/08) |
| **Facts** | time entries | employee | immutable after approval — corrections via compensating entries |
| **Coverage** | coverage allocations (funding sources) | manager (D-L) | editable indefinitely — past states via period-close snapshots (D-F) |

Terminology convention (Part 14): **direction** = the plan; **coverage allocation** = the money label. The bare word "allocation" never substitutes for either. Direction-coverage (D-Z) is the *capacity* mirror of the coverage *money* queue, one plane up.

### D-1 — One direction entity; mode is derived

One entity per D-R, mode **derived from `planned_date`** — set → `scheduled`, null → `queued`. Rows carry `directed_by`, `directed_to` XOR `wg_id`, `activity_id`, `planned_date` (nullable), `est_hours` (nullable), `priority`, `due_date`, `status`, `supersedes_id`, `origin_direction_id`. Two modes of direction, one concept:

* **Scheduled** — per-day direction ("what you do on each specific day"); the ideal case.
* **Queued** — the backlog: expiry (`due_date`) + ordering (`priority`, lower = higher); the employee steers. Backlog mode is a *normal operating mode*, not an exception (Deviation 3).

### D-2 — Storage is per-day rows, always; partial days are first-class

Per D-W, the calendar is the **input**: whatever the scheduler spreads across days lands as one row per (employee, activity, day, est_hours). Month/week totals ("cover August") are **derived sums**, never stored — a period-bulk record shape was rejected: it would force the system to invent a day-spread the manager never made, and the deviations act on *days* (preemption and over-run are day-level events).

Per D-AA, **multiple rows may share one day** (e.g. 4h X + 2h Y, remainder uncovered-visible per D-Z) — partial days are first-class. **No intra-day ordering**: within-day sequence is the employee's discretion; day + hours are stored, never start times (calendar-app territory = meta-work, VISION §2). The dynamic planning horizon (day/week/month) is **UI cadence + policy, not a schema dimension**.

### D-3 — est_hours semantics per mode, hard per-row validation, soft per-day warnings

* **Scheduled rows: `est_hours` required** (the day's directed amount). **Queued rows: `est_hours` optional** — when present it is the row's total budget (D-AA): the total estimate for the activity; the scheduler pre-fills per-day rows from remaining day capacity at drop.
* **Hard per-row validation at write:** `est_hours <= 0` (and absurd values) rejected by the service; `DECIMAL(8,2)` mirroring `time_entries.hours`.
* **Soft per-day warnings:** Σ est_hours over day capacity is a soft warning returned in read-model/create responses, **never a rejection** (D-AA realism, D-F).

### D-4 — Immutable rows + supersede-only writes; the plan is a chain

No in-place edit of `planned_date`/`est_hours` (D-13-04). Replanning always creates a **new row with `supersedes_id`** pointing at the old row, which flips to `superseded` in the same transaction (D-13-08). The plan is mutable **as a chain of rows**, never by rewriting the fact of a prior plan. Supersede is implicit via create — **no dedicated supersede endpoint**. Every transition writes an audit row (BE-012); the superseded row keeps its audit trail as history.

### D-5 — Lifecycle: draft → active → superseded/cancelled; derived states never stored

Ticket-style transition matrix (D-13-07): `draft → active`; `draft|active → cancelled` (reason required, D-13-10); `draft|active → superseded` **reachable only via create-with-supersedes_id** (D-13-08). `superseded`/`cancelled` are terminal. Draft is a real created state — created as draft, activated explicitly (or via first plan action).

Per D-V: `done`, `lapsed`, `claimed` are **derived, computed on read, never stored — no nightly jobs** (D-13-09):

* `done` — the linked activity is terminal (Phase 11's terminal-activity CTE semantics: no non-terminal time entries on the activity subtree).
* `lapsed` — past `planned_date`/`due_date` with no logged hours.
* `claimed` — existence of a claim row (see claim spectrum, D-7).

### D-6 — Who directs: managers and self-direction, both first-class

Per D-S: `directed_by` = user; **self-direction is `directed_by == directed_to`; no approval** for self-direction (planning existing work ≠ proposing new work — D-G untouched). Managers direct within their subtree / WG reach (BE-014 machinery). **User-XOR-WG target** enforced by DB CHECK (D-13-05): `(directed_to IS NULL AND wg_id IS NOT NULL) OR (directed_to IS NOT NULL AND wg_id IS NULL)`, mirroring the origin-refs CHECK pattern (D-01).

### D-7 — WG-direction: queued-only, hours-based split claims with Σ-consumption

Per D-T, a WG-direction row is a squad-level todo ("someone picks this up"): **queued-only** (no `planned_date` — scheduling stays personal), sitting in the squad queue. **Claim model:**

* A member's claim creates a **user-targeted row** through the same machinery: `directed_by` = the WG row's creator (manager attribution preserved), `directed_to` = claimant, `origin_direction_id` = the WG row, same activity. **Claim is a derived row, never a stored flag** (D-13-11). WG members only may claim (D-13-12).
* **Hours-based split claims** (user override of the single-claim lean, D-13-13): a WG row's `est_hours` is its allocated hours; a claim may take all or part of them. **Σ claimed ≤ WG est_hours enforced under a transaction lock** (first-wins/over-subscription race closed like CR-01). One claim = claims all; multiple claims = split. No cap when the WG budget is absent (D-13-14).
* **Claimed state is a derived spectrum** (D-13-15): `not_claimed → partially_claimed → fully_claimed` — fully only when a budget is set and Σ == budget. Never stored.
* **Unclaim = cancel the claim row** (reason required); hours return to the WG row automatically since consumption is Σ-derived, never stored (D-13-16).
* WG rows are queued-only (CHECK) and the activity must be within the WG's scope (same-org, reachable via WG subtree — D-13-17).

### D-8 — Org planning policy: org-configurable, stored not enforced

Per D-X, planning policy is org-configurable on three axes, stored as **first-class data** (org_settings key/value, D-13-18):

1. **Deadline** — the "by the 15th, cover next month" date (`planning_deadline`).
2. **Horizon** — the coverage period, day/week/month (`planning_horizon`; UI cadence + policy — stored, **not enforced**, D-13-21).
3. **Mode per employee** — `manager_planned` (manager spreads the calendar) vs `self_planned` (manager gives a budget/queue, the employee schedules freely); org default + per-employee override on `organization_memberships` (D-13-19).

**Store + permission-gating only — no deadline enforcement** (D-13-20): mode gates *who may create scheduled rows for whom* (manager-planned → managers create for that employee; self-planned → the employee creates own rows). Block-vs-nag soft policy is deferred to UI prototyping (D-X parked). Every settings change is audit-logged (D-13-22) with before/after payload.

### D-9 — The scheduler warns, never blocks

The scheduler is a calendar view (D-Y). **Capacity integration is explicit**: it reads [[ADR-P-008 — Availability & Employment Validity]] absence windows and employment validity, warning at plan time ("away 10–21 Aug") — P-008's "surfaced at assignment time only", now with its second consumption point. **Never blocks** (P-008 D-3: reality is messy).

* Warning types are a closed set (D-13-30): `away` (full absence), `partial` (partial-day permit), `over-capacity` (Σ est_hours > capacity), `invalid` (outside employment validity — employee NOT flagged uncovered, D-13-31). Soft, never blocking; surfaced in read-models and at create-response time.
* **Absence reading is declared+confirmed for now** (D-13-29) — Phase 14 adds confirm/reject and tightens to confirmed-only.
* Daily capacity = org's `planning_daily_hours` (default 8) − confirmed P-008 absence hours that day (partial-day permits reduce by their `hours`, full absences zero the day) (D-13-24).

### D-10 — Direction-coverage read-model: planned vs capacity

Direction coverage is a **first-class read-model** (D-Z, D-13-25): planned hours per employee per period vs capacity — the queue that answers "is August covered?" per employee / unit / WG, plus **uncovered-day surfacing**. Uncovered *capacity* is a visible state, never an implicit gap — the same doctrine as the to-cover queue, one plane up.

* One endpoint, scope params: `GET /direction/coverage?scope=employee|unit|wg&scope_id=&period=`; aggregation differs only in which-employees resolution; unit/WG scopes aggregate employees underneath (D-13-25).
* **Uncovered day = absence-aware gap list** (D-13-26): a day whose Σ est_hours < capacity (including 0 — no direction), surfaced as (employee, date, capacity, planned, gap) plus period totals; fully-absent days excluded from uncovered surfacing.
* The read-model endpoint delivers both the D-Z view **and** the derived states (done/lapsed/claimed) in one direction service (D-13-27).

### D-11 — Origin fallback: the first direction record (R4)

Per the R4 resolution (Part 15) and P-013's FND-04, origin refs are stored directly on activities; when Phase 3 lands, the derivation becomes an **additive read-path fallback** (D-13-32..34): when an activity's origin refs are empty, the read path looks up the **first direction record** for that activity (earliest `created_at` among non-cancelled rows) and derives manager-assignment-shaped refs (`directed_by` → `assigned_by`, `directed_to` → `assigned_to`). If none, refs stay empty. The fallback only fills the manager-assignment shape — never employee_proposal/customer_ticket. **Read-only derivation, never written back** — stored refs stay authoritative for pre-direction activities.

### The three assumption-delta decisions (recorded here with rationale)

Three lean-vs-locked deltas surfaced during Phase 13 planning and are recorded as decisions rather than left to implementers:

1. **Identity: no-change.** The identity noun is the direction row id; "multiple rows may share a day" is a grouping convention, **no UNIQUE constraint** on (employee, activity, day). Rationale: the per-day multiplicity of D-W/D-AA *is* the model — a uniqueness constraint would reject the 4h X + 2h Y case the user explicitly wants.
2. **Origin fallback: add-alongside.** The fallback (D-11) is an add-alongside read-path derivation — stored refs stay authoritative; the accepted debt is transient drift between stored refs and derived refs, mitigated by never writing back and by `origin_type` remaining the discriminator. Rationale: R4's "read path gets smarter, stored data stays authoritative" is preserved verbatim; writing back would break P-013's immutability.
3. **Planning policy: promoted to first-class data.** The D-X axes became a modeling decision (D-13-18/19): `org_settings` key/value rows + a membership `planning_mode` override — not typed columns, not config files. Rationale: the user's "configurations are getting bigger and bigger" — new policy keys are data rows, never migrations.

## What this feeds (Insight)

* **Today view revision (P-004):** direction is Today's first stateful data source — "your plan today" + "your queue".
* **Plan-adherence analytics (D-U):** aggregate-only, per-period — never per-day-per-person (gaming → fact-plane corruption, Part 1). A feedback instrument, not a scoreboard.
* **Estimate baselines:** direction estimates ("directed 15h, took 20h") are a third estimate baseline next to sold-vs-actual (D-N) and V5's activity-kind history.
* **Per-unit reports:** direction-coverage feeds the per-unit report family (Part 9).

## Consequences

* **The three-plane ontology is complete and fixed:** direction (plan) → facts (entries) → coverage (label). Every ADR drafted against it stays valid; no circle-back.
* **The plan never rewrites the fact** — deviations (under-run, preemption, over-run, …) are data, visible on the plan-coverage read-model, never violations.
* **Replanning is history, not erasure:** the supersede chain preserves every prior plan state with its audit trail (BE-012); "what did we plan vs what happened" is answerable.
* **WG work is claimable by the hour:** the split-claim model with Σ-consumption under a transaction lock closes the over-subscription race; unclaim returns hours automatically (Σ-derived, never stored).
* **Org policy is data, and stays soft:** deadline/horizon/mode are org-configurable key/value settings with permission gating only — the backend never blocks on deadline or horizon (block-vs-nag is a Phase 19 UI decision).
* **The scheduler stays advisory:** absence/validity warnings never block writes — reality is messy, and P-008's D-3 doctrine holds.
* ⚠️ **Sequencing:** backend lands in Phase 13 (per the staged build of Part 15); the D-Y calendar, direction queue, and coverage read-model surfaces land in Phase 19/4d — this ADR is the contract they compile against.

## Related

* [[ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger]] — the three-plane doctrine; direction-coverage is the capacity mirror of the to-cover queue
* [[ADR-P-008 — Availability & Employment Validity]] — absence windows + employment validity consumed by the D-9 warning overlay (declared+confirmed until Phase 14)
* [[ADR-P-013 — Origins]] — origin refs stored on activities; the D-11 fallback derives manager-assignment refs from the first direction record (FND-04/R4)
* [[ADR-P-007 — Activity Ontology]] (the directed target) · [[ADR-P-004 — The Today View]] (first stateful data source; revision pending) · [[ADR-P-005 — Insight Pillar Roadmap]] (V3)
* [[ADR-BE-018 — Direction & Org Settings Encoding]] · [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] (manager reach, D-6)
* [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 14–15, D-Q … D-AA — record of truth)
