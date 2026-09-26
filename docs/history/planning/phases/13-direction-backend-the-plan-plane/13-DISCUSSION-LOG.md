# Phase 13: Direction Backend — The Plan Plane - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-07
**Phase:** 13-Direction Backend — The Plan Plane
**Areas discussed:** Schema & est_hours semantics, Lifecycle & supersede chaining, WG claim model, Org policy storage & mode, Coverage read-model & capacity, Absence warnings & origin fallback

---

## Schema & est_hours semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Single direction table | One table: directed_by, directed_to/wg_id, activity_id, planned_date (NULL = queued), est_hours, priority, due_date, status, supersedes_id, origin_direction_id; mode derived per D-R | ✓ |
| Split scheduled/queued tables | Separate tables per mode — breaks D-R's one-entity model | |
| est_hours required scheduled / optional queued | Required > 0 on scheduled; optional budget on queued (D-AA) | ✓ |
| est_hours required on both | Queued rows always carry a budget | |
| Scheduled only | No est_hours on queued rows | |
| Hard per-row, soft per-day | Reject est_hours <= 0 at write; Σ over capacity = soft warning, never block | ✓ |
| Hard per-day too | Reject rows pushing day over capacity | |
| Read-side only | No write-time validation | |
| Immutable rows + supersede | Replanning = new row with supersedes_id; old flips to superseded in same tx; no in-place edits | ✓ |
| Editable while draft | Edits allowed in draft, frozen after activation | |
| In-place edits + audit | Edit endpoints with audit rows | |
| Budget = total estimate | Queued est_hours = total budget; scheduler pre-fills per-day from capacity | ✓ |
| Priority/due_date only | Queued rows carry no hours | |
| DB CHECK XOR | (directed_to IS NULL AND wg_id IS NOT NULL) OR inverse, house style | ✓ |
| Service-level only | XOR enforced in service only | |
| INT + due_date, both nullable | priority INT (lower = higher) + due_date; ordering priority then due_date | ✓ |
| Enum priority | high/medium/low | |
| due_date only | No priority field | |

**User's choice:** All recommendations accepted.

---

## Lifecycle & supersede chaining

| Option | Description | Selected |
|--------|-------------|----------|
| 4 transitions + derived states | draft → active → superseded/cancelled matrix with audit rows; done/lapsed/claimed derived | ✓ |
| Create = active | No draft state | |
| Ticket-parity state machine | Full in-tx re-validation under FOR UPDATE (CR-01 pattern) — folded into the matrix choice | |
| Implicit via create | Creating a row with supersedes_id flips the old one in the same transaction | ✓ |
| Explicit endpoint | POST /direction/{id}/supersede | |
| Computed on read | done (activity terminal) / lapsed (past date, no hours) / claimed (claim row exists) in service | ✓ |
| Materialized | Generated columns or nightly jobs | |
| Reason required, both states | Cancel requires reason; draft + active cancellable; terminal | ✓ |
| Active-only cancel | Drafts dropped silently | |
| Draft delete + cancel | Hard delete for drafts | |

**User's choice:** All recommendations accepted.

---

## WG claim model

| Option | Description | Selected |
|--------|-------------|----------|
| Claim = derived row, by=manager | directed_by = WG row creator, directed_to = claimant, origin_direction_id = WG row | ✓ |
| Claim = self-direction | directed_by = claimant | |
| WG members only | Membership checked at claim time | ✓ |
| Any org member | No membership check | |
| Members + managers | WG members + anchored-unit managers | |
| Hours-based split claims | Each claim carries its own est_hours; Σ claimed ≤ WG est_hours under tx lock; single claim = all, multiple = split | ✓ (user override) |
| Single claim consumes all | First-wins single claim | |
| No cap when budget absent | Claims uncapped when WG est_hours null; consumed never derives | ✓ |
| Mandatory WG budget | est_hours required on WG rows | |
| Derived spectrum | not_claimed → partially_claimed → fully_claimed (when budget set) | ✓ |
| Binary claimed flag | claimed/unclaimed only | |
| Cancel claim row | Unclaim = cancel with reason; Σ-derived consumption returns hours | ✓ |
| Explicit unclaim endpoint | Separate unclaim action | |
| Queued-only + reachable activity | planned_date NULL CHECK when wg_id set; activity within WG scope | ✓ |
| WG rows schedulable | Violates D-T | |

**User's choice:** Hours-based split claims (free-text override): "a task should be consumed by its allocated hours, so that we are open to: claiming alone (just use all the allocated hours) or multiple claims (different users claim an amount of hours from it), let's double check this". All other recommendations accepted.

---

## Org policy storage & mode

| Option | Description | Selected |
|--------|-------------|----------|
| Settings table (key/value JSONB) | org_settings(org_id, key, value JSONB, PK(org_id,key)) — configs growing | ✓ (user override of typed-column lean) |
| Typed columns on org | deadline/horizon/mode columns + CHECK | |
| JSONB on org row | Single settings column | |
| Org default + per-employee override | Nullable override on memberships falls back to org default | ✓ |
| Org-wide only | All employees share mode | |
| Store + permission gating only | No deadline blocking; mode gates row creation; block-vs-nag deferred to UI prototyping | ✓ |
| Enforce deadline too | Reject creates after deadline | |
| Stored, not enforced | Horizon = UI cadence + policy (D-W) | ✓ |
| Stored + read-model gating | Horizon shapes read-model periods | |
| Key/value JSONB | org_settings table with JSONB values, domain validates | ✓ |
| Typed columns + key | Typed per known setting + extensibility | |
| Audit every change | BE-012 audit_logs with before/after payload | ✓ |
| No audit | Settings are operational | |
| Generic settings endpoints | GET/PUT /organizations/settings {key: value}, manager+, additive keys | ✓ (with JWT-resolved org) |
| Org path param | /organizations/{orgID}/settings | |

**User's choice:** Settings table (free-text): "I think that we should keep a settings table since configurations are getting bigger and bigger". Org scoping (free-text): "settings are org wide, so we should at least define an org for them, a path param should do" — clarified to JWT-resolved org (house style) on follow-up.

---

## Coverage read-model & capacity

| Option | Description | Selected |
|--------|-------------|----------|
| Daily-hours setting | org_settings key (e.g. planning_daily_hours, default 8); capacity = daily hours − confirmed absence hours | ✓ |
| Per-employee field | Hours on memberships | |
| One endpoint + params | GET /direction/coverage?scope=employee|unit|wg&scope_id=&period= | ✓ |
| Separate endpoints | Per employee/unit/WG | |
| Gap list, absence-aware | Day with Σ est_hours < capacity surfaced as (employee, date, capacity, planned, gap); fully-absent days excluded | ✓ |
| Raw delta | capacity − planned regardless of absences | |
| Combined in one service | Read-model + lifecycle in one direction service | ✓ |
| Separate services | Dedicated read-model service | |

**User's choice:** Read-model shape asked for elaboration; after explanation chose one endpoint + params. All others recommended.

---

## Absence warnings & origin fallback

| Option | Description | Selected |
|--------|-------------|----------|
| Overlay on read paths | Coverage/plan read-model overlays absence + validity warnings; advisory, never blocks | ✓ |
| Create-time only | Warnings only in create responses | |
| Read 012 + both statuses | declared+confirmed count until Phase 14 tightens | ✓ |
| Confirmed only | Only confirmed windows | |
| Activity read-path fallback | Service-layer lookup of first direction record when refs empty | ✓ |
| Separate resolution endpoint | Explicit origin-resolution endpoint | |
| Earliest non-cancelled, manager-shape only | First = earliest created_at among non-cancelled; fills assigned_by/assigned_to only | ✓ |
| All rows count | Cancelled/superseded included | |
| Validity-aware surfacing | Outside validity → warning, never flagged uncovered | ✓ |
| Warning-only | Validity only affects warnings | |
| Third warning type | away / partial / over-capacity | ✓ |
| Read-model only | Over-capacity only in gaps | |
| Read-only derivation | Never written back; stored refs authoritative (R4) | ✓ |
| Backfill | Write refs into activities | |

**User's choice:** All recommendations accepted.

---

## the agent's Discretion

- Exact direction CRUD/transition endpoint list and URL shapes
- est_hours hour granularity (mirror time entries' DECIMAL)
- Draft → active activation mechanics (explicit endpoint vs first-action activation)
- Read-model response envelope shape
- Test layout for direction/org_settings packages

## Deferred Ideas

- Block-vs-nag soft policy → UI prototyping (Phase 19)
- Absence confirm/reject tightening → Phase 14
- Plan-adherence analytics (D-U) → V5, aggregate-only per-period
