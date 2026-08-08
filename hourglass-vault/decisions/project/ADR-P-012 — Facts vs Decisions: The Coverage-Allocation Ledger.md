---
tags: ["adr","idea-layer","capture","structure","insight","coverage"]
---
# ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger

**Status:** Accepted
**Date:** 2026-08-02
**Operationalizes:** [[VISION]] §5 (Control) · §6 V5 (pricing analytics) · **Basis:** [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]] (Parts 1, 5–7, 10–13) · **Extends:** [[ADR-P-007 — Activity Ontology]] (D-7 billability) · **Decided by:** D-F, D-I, D-K … D-P of the research note · **Implemented by:** [[ADR-BE-017 — Coverage Encoding]] (coverage encoding — schema, proposal computation, snapshot mechanics; F per Part 13)
**Accepted:** 2026-08-07

---

## Context

The organisation's real process: effort is compiled in Excel by month end; managers + finance spend **week 1 of the following month** deciding how those hours are *covered* — billed to the customer's contract, drawn from a support bucket, absorbed internally, or moved to another project. Today that decision is applied by **editing or duplicating the time records themselves** — e.g. an 8-hour job becomes "4h billed + 4h elsewhere". The practice preserves the money story by destroying the effort story.

Stefano's framing: *"the 8 real hours the employee spent must be tracked, so that when the same activity is proposed again — same customer or another — the analytics say how long it actually takes. First time I underestimated and sold 4h; next time I sell 16, not 4."* If coverage rewrites effort, [[ADR-P-005 — Insight Pillar Roadmap|V5 pricing analytics]] mines corrupted data forever.

The accepted [[ADR-P-007 — Activity Ontology]] (D-7) already says *whether a customer pays* is a property of the work (billability). What it does not say is what **actually happened** with the money after the work was done — including billable work that ended up partly absorbed (underestimate, warranty, goodwill).

## Decision

### The cardinal principle

> **Captured effort is a fact. Coverage is a decision. The decision never rewrites the fact.**

Two planes, one invariant:

| Plane | Content | Written by | Mutability |
|-------|---------|-----------|------------|
| **Facts** | time entries (who, what activity, when, how long) | employee | immutable after approval (existing F11 rule) — corrections only via compensating entries |
| **Decisions** | coverage allocations (which funding source pays which share) | manager (D-L) | **editable indefinitely** (D-1) — past states preserved via period-close snapshots (D-4) |

### D-1 — Coverage is a per-entry allocation ledger

Every time entry is covered by **1..N `coverage_allocations`**, each pointing at one [[ADR-P-014 — Funding Sources & Beneficiary Unit|funding source]]. The hard invariant:

\[\sum \text{allocations}(entry) = \text{entry hours}\]

An approved entry whose allocations don't sum to its hours sits in an explicit **to-cover queue** — uncovered work is a visible state, never an implicit gap (and never a block on anything else).

* **Billability (P-007 D-7) becomes the default-source rule:** `billable` → its contract's budget; `non-billable` → internal absorption. The property of the work drives the *default*; the ledger records what *actually* happened.
* **Allocations never lock (D-F).** "Realism over enforcement": the monthly ritual has a soft target (a mid-month nag on the to-cover queue) and no hard cutoff. The decision never rewrites the fact — and neither does the calendar.
* **Audit-first:** every allocation write is audit-logged per [[ADR-BE-012 — Audit Log Writes]].

### D-2 — One confirmer: the manager

Allocation confirmation is a **single step, by the manager** (D-L). Finance keeps overview and control **through the reports and the audit trail** — not through a second confirm step. Rationale: the hours were already approved twice (manager → finance, BE-014); allocation is money-labeling, not new truth.

### D-3 — Allocation proposals are computed on read, never stored

The week-1 screen derives each proposed allocation **live** from entry + activity chain + funding configuration; only the manager's **confirmed** allocation is ever persisted (D-I). No proposal table, no staleness window, no batch job. The 90% default case is one click: confirm the proposal.

### D-4 — Period close is a snapshot, not a lock

`financial_cutoff_periods` (existing mechanism) locks **facts only** at period close. Because allocations stay editable forever, "what we reported for March" must survive later edits: the period close produces a **reporting snapshot** (billing, bucket levels, per-unit report). Implementation choice — stored snapshot vs as-of-close reconstruction from the BE-012 audit log — is backend-only (Part 13-F); the guarantee is the same under both: **a reported period never changes retroactively.**

### D-5 — Schema is polymorphic-ready, v0.2 allows time entries only

`coverage_allocations` references a polymorphic entry (`entry_type` + `entry_id`), with `time` the only allowed type in v0.2 (D-K). ⚠️ **Documented tension:** generality carried ahead of a demonstrated expense-splitting need ([[F12-Expenses]]), accepted consciously as cheap insurance against a corner-painting migration — one extra validation branch. The BE ADR should cost it honestly and revisit if it proves dead weight.

### D-6 — The 8h example, modelled

Sold estimate was 4h; actual work was 8h on activity A:

| Plane | Record | Value |
|-------|--------|-------|
| Fact | time entry | 8h on activity A |
| Decision | allocation 1 | 4h → contract budget (billed) |
| Decision | allocation 2 | 4h → internal absorption `{ reason: UnderEstimate, unit: beneficiary unit }` |
| Insight | V5 analytics | actual 8h vs [[ADR-P-014 — Funding Sources & Beneficiary Unit|sold]] 4h → accuracy 0.5 → next quote: 16h |

Both truths survive: the customer is billed 4h, the cost of the underestimate is visible on the beneficiary unit, and the effort fact stays 8h.

## What this feeds (Insight)

* **Estimate accuracy** per activity kind/customer — actual vs estimated, from uncorrupted facts (V5's "similar requests cost us X hours" becomes honest)
* **Warranty/goodwill cost per customer** — how much "not charging them" actually costs
* **Support bucket consumption** — remaining, burn rate, saturation per period
* **Cross-project transfers** — explicit and measurable instead of invisible
* **Per-unit non-billed cost report** — the financial resoconto for internal work

## Consequences

* **The corrupt practice becomes impossible to perform silently.** Hours can no longer be moved between projects by editing entries; a transfer is an explicit allocation with mandatory justification. The system changes the organisation's data hygiene by construction, not by policy.
* **Two queues structure the manager's month:** the to-cover queue (money, week 1 of M+1) and — one plane up, per [[ADR-P-015 — Direction: The Plan Plane|P-015]] — the direction-coverage queue (capacity, before M starts). Same doctrine: visible states, soft nags, snapshots not locks.
* **V5's prerequisite is now contractual:** analytics read facts for effort and the ledger for money, and may never join them by rewriting either.
* ⚠️ **Approval chains stay untouched:** allocation confirmation (one manager step) is deliberately *not* part of BE-014's two-stage entry chain; finance oversight is report- and audit-based (D-2).
* ⚠️ **D-K's polymorphism is the single scope-risk** — held to one validation branch by the BE ADR.

## Related

* [[VISION]] §5 (Control), §6 (V5 pricing)
* [[ADR-P-003 — Tickets as the Second Capture Layer]] (dismissal guard, D-M) · [[ADR-P-007 — Activity Ontology]] (D-7 billability → default source) · [[ADR-P-005 — Insight Pillar Roadmap]] (data-maturity gate)
* [[ADR-P-013 — Origins on Activities]] · [[ADR-P-014 — Funding Sources & Beneficiary Unit]] · [[ADR-P-015 — Direction: The Plan Plane]]
* [[ADR-BE-012 — Audit Log Writes]] · [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]]
* [[F09-Contracts]] · [[F11-Time-Entries]] · [[research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research]]

## Acceptance

**Accepted:** 2026-08-07 — Phase 12 (Coverage Backend — The Allocation Loop) operationalizes D-1..D-6 via [[ADR-BE-017 — Coverage Encoding]], satisfying COV-01..COV-05 and the D-01..D-12 locked decisions of the phase context:

* **D-1** (per-entry allocation ledger with the Σ invariant + to-cover queue) → the `coverage_allocations` ledger, the replace-set write with in-tx Σ validation under a `FOR UPDATE` entry-row lock, and the to-cover queue read-model (COV-01).
* **D-2** (one confirmer: the manager; finance read-only) → the D-08 manager gate via `routing.ResolveManagerStage`, with finance read-only access to reports and the audit trail (COV-03).
* **D-3** (proposals computed on read, never stored) → the chain-driven default-source decision function (D-04) with the ticket-kind extension seam (D-05); no proposal table exists.
* **D-4** (period close is a snapshot, not a lock) → the snapshot-not-lock choice (Q10 amendment) is implemented as the **frozen period-close snapshot** (D-10/D-11/D-12): `POST /coverage/close` writes immutable entry-level rows into `coverage_snapshot_rows`; reports read the frozen copy; live allocations stay editable indefinitely; `financial_cutoff_periods` stays facts-only (COV-04).
* **D-5** (polymorphic entry, `time` only in v0.2) → `entry_type` + `entry_id` with the schema CHECK `('time')` and the service branch rejecting `entry_type != 'time'` — the D-K polymorphic validation cost is costed honestly in ADR-BE-017 (COV-05).
* **D-6** (the 8h example, modelled) → realized by the ledger + absorption allocations with mandatory `reason` and the derived bucket balance (COV-02).

Status flipped from Proposed per the vault append-only rule: the status cell and this Acceptance section were appended; no content was rewritten.
