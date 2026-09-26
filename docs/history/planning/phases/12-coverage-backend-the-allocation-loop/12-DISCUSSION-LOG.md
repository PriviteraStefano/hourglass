# Phase 12: Coverage Backend — The Allocation Loop - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-07
**Phase:** 12-Coverage Backend — The Allocation Loop
**Areas discussed:** Funding-source storage shape, Proposal eligibility rules, Write semantics & permission, Snapshot mechanics

---

## Funding-source storage shape

| Option | Description | Selected |
|--------|-------------|----------|
| Tagged union on the allocation row | coverage_allocations carries source_type + nullable refs (contract_id, unit_id) with CHECK — Phase 11 origin pattern; ledger self-contained | ✓ |
| Separate funding_sources hub table | Pre-declared source configuration rows allocations FK to | |

**User's choice:** Tagged union on the allocation row
**Notes:** Consistent with Phase 11 D-01 origins pattern.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Derived from allocations | Balance = Σ sold_hours − Σ allocations, computed on read; carry-over falls out naturally | ✓ |
| Stored counter on the bucket | remaining_hours column updated in the allocation tx; write-path coupling | |

**User's choice:** Derived from allocations
**Notes:** Consistent with D-I (never store what can be computed).

---

| Option | Description | Selected |
|--------|-------------|----------|
| Allow, visible in report | Negative balance visible in bucket report; D-C no anti-abuse control | ✓ |
| Block overdraw at write time | Service-level rejection when balance would go below zero | |

**User's choice:** Allow, visible in report
**Notes:** The report is the control (D-C).

---

## Proposal eligibility rules

| Option | Description | Selected |
|--------|-------------|----------|
| Single chain-driven rule | One decision function over the activity chain: project → contract budget, support → bucket, zero-value → contract draw, none → absorption w/ beneficiary unit | ✓ |
| Billability first, then chain | Billable/non-billable flag branches first, then finer rules | |

**User's choice:** Single chain-driven rule
**Notes:** Mirrors D-7 billability resolution via the same CTE.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Extension point only | Kind→source matrix deferred; one seam in the proposal function | ✓ |
| Full kind→source matrix now | Pin question/bug → bucket, change/evolution → contract now | |

**User's choice:** Extension point only
**Notes:** No matrix was ever defined; proposals are computed on read so later rules are code-only.

---

| Option | Description | Selected |
|--------|-------------|----------|
| In the queue, flagged | No-source entries appear in the same queue with "no eligible source" + reason | ✓ |
| Separate section | No-source entries listed separately from proposable ones | |

**User's choice:** In the queue, flagged (after asking for a simpler explanation with an example)
**Notes:** User requested simpler questions with examples from this point forward.

---

## Write semantics & permission

| Option | Description | Selected |
|--------|-------------|----------|
| One save, replace all | PUT full allocation set per entry, Σ validated in-tx, atomic replace | ✓ |
| Change rows one at a time | Incremental allocation CRUD with re-checks after each write | |

**User's choice:** One save, replace all
**Notes:** Presented with the 4+4 split example at user's request.

---

| Option | Description | Selected |
|--------|-------------|----------|
| The entry's own manager | BE-014 activity-chain resolution (activity → WG → manager) | ✓ |
| Any org manager | Simpler gate, no relation to who approved the entry | |

**User's choice:** The entry's own manager
**Notes:** Same person who approved the entry; finance read-only per D-L.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Skip corrections entirely | Coverage sees only approved entries, hours > 0; no correction logic | ✓ |
| Skip + leave a note | Same, plus a CONTEXT note for future | |

**User's choice:** Skip corrections entirely (via free-text: "corrections like these shouldn't be possible, activities can't be negative")
**Notes:** User rejected the negative-hour correction scenario — the schema CHECK (hours > 0) forbids it. Compensating entries don't exist in the codebase; D-13 net-of-compensations swap is vacuous. Left as a Deferred Idea.

---

## Snapshot mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Frozen copy at close | Immutable snapshot written at close; reports read the frozen copy | ✓ |
| Replay the audit log | No snapshot table; reconstruct as-of-close from audit_logs | |

**User's choice:** Frozen copy at close (after asking for advice on the payment/invoice concern)
**Notes:** User raised: "since the payment already came we shouldn't be able to modify allocations, we can in the mean time. This is the sort of logical bugs that come out when there isn't a clear system." Advice given: v0.2 has no invoicing; snapshot is the seam; billing layer is where a real lock attaches later. User accepted.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Entry-level rows | Snapshot carries per-entry allocation state; aggregates computed on read | ✓ |
| Entry-level + aggregates | Also pre-aggregate totals per contract/bucket/unit at close | |

**User's choice:** Entry-level rows
**Notes:** Keeps past snapshots re-shapeable for Phase 17 reports.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Hook onto financial_cutoff_periods | Existing cutoff table as the close trigger | |
| New close endpoint | Coverage-specific close endpoint scoped to org + period | ✓ |

**User's choice:** New close endpoint
**Notes:** financial_cutoff_periods stays separate; how they relate is planner discretion.

---

## the agent's Discretion

- Endpoint list/URL shapes for coverage routes
- Proposal read-path exposure shape
- Chain-walk CTE reuse for default-source derivation
- Snapshot table shape and naming
- Whether close endpoint returns snapshot data in one call (implied by user's choice — confirm in planning)
- Audit-write mechanics mirror Phase 11 in-tx pattern
- source_type CHECK vocabulary; justification vs reason columns for transfer/absorption
- Test layout for the coverage domain package

## Deferred Ideas

- Correction/compensation entries (created_from_entry_id schema-only; design coverage interaction when/if they land)
- Billing/invoicing lock (v2+): invoices read frozen snapshot; real allocation lock attaches at billing layer
