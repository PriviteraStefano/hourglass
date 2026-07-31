# ADR-P-001 — Units vs. Working Groups: Accountability vs. Execution

---
tags: ["adr", "idea-layer", "structure"]

---

# ADR-P-001 — Units vs. Working Groups: Accountability vs. Execution

**Status:** Accepted
**Date:** 2026-07-28 (proposed) · 2026-07-28 (accepted after feedback)
**Operationalizes:** [[VISION]] §5 · **Blocks:** V3 (knowledge-based team formation)

---

## Context

The model has two groupings of people — **units** (org tree) and **working groups** (subproject teams). The docs never said why both exist. Reading the code revealed the system had made an implicit choice — approval flows through **execution** (WG/project managers), never through the **unit tree** — but made it inconsistently, and left accountability half-wired.

## What the code showed

* A working group is anchored to a `subproject_id` — "the team executing one subproject" — with `unit_ids`, a `manager_id`, `delegate_ids`, and `enforce_unit_tuple`.
* WG members carry a `unit_id`; time entries carry **both** `wg_id` and `unit_id`.
* Approval routing was **inconsistent**: time entries routed by **WG manager/delegate**, expenses by **project manager**. Units were never consulted.
* The D-11 skip (WG manager == creator → skip manager stage) was specced but **not implemented**.
* `enforce_unit_tuple` was stored, defaulted `TRUE`, and **never read**.

## Decision

**Units = accountability.** Stable org tree, reporting lines, **visibility scoping**, and the home of the V3 employee-knowledge profile. **No approval routing.**

**Working groups = execution.** The team on a subproject. **Approval routes through execution** — the WG manager or a delegate is the "manager" approval stage.

The four open questions, **resolved per your feedback**:

| # | Question | Decision |
|---|----------|----------|
| 1 | Expense routing source? | **Matches time entries.** Both entry types route to the **WG manager/delegate** on the entry's working group. Project managers are **no longer** a separate expense-approval source — expense `ListPending` is changed to use the WG rule (same as time entries). Project managers keep their governance meaning ([[LEGACY/12-Contracts-Projects]]) but are not an approval queue. |
| 2 | Unit-tree visibility gating? | **Yes — a unit manager sees only their subtree's entries.** Implemented via `unit_memberships.role = 'manager'` + the recursive-CTE subtree pattern already in the units repository. An org-role `manager` / `finance` sees the whole org. |
| 3 | `enforce_unit_tuple`? | **Remove the column.** (Rationale below.) |
| 4 | D-11 skip applies to delegates? | **Yes, delegates too.** If the entry's WG manager *or any delegate* is the entry owner, `submitted` goes straight to `pending_finance`. |

### On removing `enforce_unit_tuple` — the plain-English version

The column was a leftover toggle that was supposed to mean: *"must every member of this working group also belong to one of the units listed in `unit_ids`?"* (`TRUE` = enforce that link). It was **never read by any code** — so today it does literally nothing.

Here's the thing: **the schema already enforces the link for free.** Every `wg_member` row *requires* a `unit_id` (NOT NULL), and every time entry *requires* a `unit_id` too. So the rule "a WG member has a unit" is already guaranteed by the database, no toggle needed.

What the toggle *would* add is only: *"...and that unit must be one of the specific `unit_ids` on the WG."* That's a **strictness option**, not a safety requirement. Since we have no use case for letting a WG bypass its own unit list, keeping a dead toggle only invites confusion ("why is this FALSE? is something broken?").

**So: delete the column and, if we want the strictness, make it a real validation** ("a WG member's `unit_id` must be in the WG's `unit_ids`") as a proper constraint — not a boolean flag. This is a schema migration (drop column) + an optional follow-up validation ADR.

## Consequences

* **One approval story:** "my WG lead (or a delegate) approves, finance confirms." Time and expenses behave identically.
* **Units stop pretending to govern approvals** and become the org map + V3 profile home + **visibility scope** (subtree gating).
* **V1's "pending approvals" queue** becomes a single, predictable source (the WG on the entry).
* ⚠️ **Expense routing changes behavior.** Historical expenses approved "by project manager" were valid under the old rule; going forward they use WG routing. This is a behavior change to note in the deployment, not a data migration.
* ⚠️ **The D-11 skip must not become true self-approval.** The skip fires only when the *approver role* (WG manager/delegate) coincides with the entry *owner* — an employee can never approve their own entry. The implementation must keep that distinction explicit.
* ⚠️ **New work unlocked (backend):**
  * Expense `ListPending` → WG routing.
  * Subtree visibility gating in time/expense list queries.
  * D-11 skip in `Submit()`/`Approve()` for time + expense.
  * Migration to drop `enforce_unit_tuple` (+ optional unit-membership validation).

## Resolved-by (implementation)

These consequences are implemented in a backend ADR, not here. **Next:** ADR-BE-014 — Approval-Routing Precedence & Visibility Scoping (encodes #1, #2, #4 and the `enforce_unit_tuple` removal from #3).

## Related

* [[VISION]] §4 (Structure), §5, V3
* [[ADR-P-002 — Four Pillars & Feature Purposes]] (Control pillar)
* [[LEGACY/12-Contracts-Projects]] (governance models — project managers' remaining meaning)
* Code: `internal/core/services/{time_entry,expense}/`, `internal/adapters/secondary/postgres/{time_entry,expense}_repository.go` (ListPending), `internal/adapters/secondary/postgres/unit_repository.go` (subtree CTE), `migrations/000_full_schema.up.sql` §4, §13–15
