---
phase: 11-foundations-schema-origins-tickets-backend
plan: 02
subsystem: docs
tags: [adr, vault, tickets, origins, audit, schema-encoding]

# Dependency graph
requires:
  - phase: 11-foundations-schema-origins-tickets-backend (plan 01)
    provides: migrations 014-017 (tickets, activity origins, contract sold hours, audit logs) — the encoded shapes this ADR records
  - phase: 10-backend-foundations
    provides: ADR-BE-012 (audit log writes), ADR-BE-004 (migrations rule), ADR-BE-014 (approval routing) — extended/referenced
provides:
  - ADR-P-003 revised (v0.2 ticket lifecycle: seven statuses + reopen + guarded dismissal, closed kind set, internal-only + D-15 permission gates, preserved hard boundary list)
  - ADR-P-013 drafted (origin axis: three types, storage decision, same-org validation, ErrOriginImmutable, creation gates, proposal approval, reviewed_by OQ1 resolution, FND-04 Phase-13 fallback)
  - ADR-BE-016 drafted (migrations 014-017 shapes, 3VL CHECK guard house rule, three-layer ticket model, synchronous in-tx audit writes, terminal-activity definition, LoggedHours signature, transition matrix, sold_hours semantics)
  - both decision _index.md files updated (project + backend)
affects: [11-foundations-schema-origins-tickets-backend plans 04-06, 12-funding-sources, 13-direction]

# Tech tracking
tech-stack:
  added: [none — vault markdown decision documents only]
  patterns:
    - "House ADR format: frontmatter tags + status/date/operationalizes/basis/decided-by/implemented-by header + Context/Decision/Consequences/Related"
    - "Revised ADR convention: keep status, add Revised date + delta section listing changes vs the original draft"

key-files:
  created:
    - hourglass-vault/decisions/project/ADR-P-013 — Origins.md
    - hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md
  modified:
    - hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md (revised to v0.2)
    - hourglass-vault/decisions/project/_index.md
    - hourglass-vault/decisions/backend/_index.md

key-decisions:
  - "reviewed_by stays NULL at creation for employee_proposal origins (OQ1): CHECK requires only proposed_by; the approver is recorded in the proposal_approved audit row; ErrInvalidRequest on non-nil reviewed_by at create"
  - "Ticket audit rows are written synchronously inside the same transaction as the state change (OQ4/A3, Pitfall 2); BE-012 fire-and-forget stays for entry approvals only; outbox documented as the reversible successor of the user-deferred durability choice"
  - "Hard boundary list kept verbatim in the P-003 revision — tickets are demand tracking, not task execution (no kanban / sub-task trees / comment threads as conversation)"
  - "Dismissal guard signature pinned as LoggedHours(ctx, ticketID) (float64, error) on raw Σ (submitted+approved, not deleted) — Phase 12 swaps computation to net-of-compensations without signature change (D-13)"
  - "Terminal activity defined as: no non-terminal time entries (draft/submitted/pending_manager/pending_finance, is_deleted=false) on the linked-activity subtree via recursive CTE (OQ2)"
  - "Transition matrix pinned (A7/OQ6): open→triage, triage→planned, triage→dismissed, planned→in_progress, in_progress→resolved, resolved→closed, resolved→in_progress (reopen), open→dismissed; closed/dismissed terminal; else ErrInvalidTransition"

patterns-established:
  - "Each backend phase drafts its ADR + BE encoding ADR (milestone convention) — ADRs pin semantic resolutions before the code plans that implement them"

requirements-completed: [FND-01, FND-02, FND-04, TICK-01, TICK-02, TICK-03, TICK-04, TICK-05]

# Metrics
duration: 12min
completed: 2026-08-07
---

# Phase 11 Plan 2: Decision Documents — P-003 Revision + P-013 + BE-016 Summary

**ADR-P-003 revised to the v0.2 ticket lifecycle (seven statuses + reopen + guarded dismissal, closed kind set, D-15 permission gates, hard boundaries preserved verbatim), ADR-P-013 drafted pinning the origin axis (three types, ErrOriginImmutable immutability, reviewed_by NULL-at-creation resolution, FND-04 Phase-13 fallback), and ADR-BE-016 drafted encoding migrations 014-017 with the 3VL CHECK guard rule, three-layer ticket model, synchronous in-transaction ticket audit writes, terminal-activity definition, and the LoggedHours dismissal-guard signature — both decision indexes updated**

## Performance

- **Duration:** 12 min
- **Started:** 2026-08-07T09:58:00Z (approx — first context load)
- **Completed:** 2026-08-07T10:11:33Z
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- **ADR-P-003 revised in place** (status stays Proposed; "Revised 2026-08-03 (Phase 11)" + delta section): the three-status sketch becomes the v0.2 state machine — `open → triage → planned → in_progress → resolved → closed` + reopen (`resolved → in_progress`) + guarded `dismissed` from `open|triage` (D-A/D-14/D-M); closed kind set `question · bug · change · evolution` (D-H); internal-only v0.2 (D-E) with the D-15 permission gate table (create/update/comment/triage/dismiss/view; `customer` role rejected); chain pinned ticket → activity → entries (TICK-03); `dismissed_hours` note semantics (TICK-04). The hard boundary list (no kanban / no sub-task trees / no comment threads as conversation) is preserved verbatim.
- **ADR-P-013 "Origins" drafted** (Proposed): the three origin types with their ref sets (manager_assignment → assigned_by/assigned_to, employee_proposal → proposed_by/reviewed_by, customer_ticket → ticket_id, D-D/FND-01); storage decision discriminator + nullable columns with EAV/JSONB rejected by R4 (D-01); same-org validation at service level with DB FKs where possible (D-02); immutability via the `ErrOriginImmutable` sentinel, no DB trigger (D-03); creation gates per origin type (D-04); proposal approval = `is_active=false` + BE-014 routing + audit_logs lifecycle, never origin refs (D-G/D-12); the **OQ1 reviewed_by resolution** (NULL at creation, approver in the `proposal_approved` audit row, `ErrInvalidRequest` on non-nil at create, future phase may pin); the **FND-04 Phase-13 fallback statement** (additive read-path enhancement, no migration, read model not painted into a corner). Cross-references P-003 (customer_ticket refs) and P-012 (three-plane model) — the plan's key_link.
- **ADR-BE-016 "Origins Tickets & Audit Schema Encoding" drafted** (Proposed): each migration's shape (014 tickets/ticket_comments with kind/status CHECK vocabulary + `dismissed_hours`; 015 activity origins with refs-to-type CHECK; 016 contracts `contracts_sold_check`; 017 general append-only `audit_logs`); the **house rule** that multi-column CHECKs carry the `discriminator IS NULL OR (...)` three-valued-logic guard (Pitfall 1, D-16 legacy safety); the **three-layer ticket model** (state / comments / audit, D-06) with no UPDATE/DELETE paths on any of the three (TICK-05); the **synchronous in-transaction audit-write decision** (OQ4/A3 — tickets only; BE-012 fire-and-forget stays for entry approvals; outbox documented as the reversible successor of the user-deferred durability choice, with the context_compliance I-3 transparency note); triage validates activity plans **inside** the transaction (Pitfall 7); semantic resolutions: terminal activity via recursive CTE (OQ2), `LoggedHours(ctx, ticketID)` stable guard signature (D-13), transition matrix edges (A7/OQ6), pre-triage fast path (OQ5), sold_hours semantics (D-07..D-09), BE-012 scope note extended to the general table.
- **Both decision indexes updated**: project `_index.md` (P-003 entry revised with date + summary, P-013 row added) and backend `_index.md` (BE-016 row added, matching house table format).
- **Vault-only plan verified**: `git diff HEAD~3` shows exactly the 5 planned files; no Go code or migrations touched.

## Task Commits

Each task was committed atomically:

1. **Task 1: Revise ADR-P-003 (tickets lifecycle)** - `7616a1d` (docs)
2. **Task 2: Draft ADR-P-013 (origin axis)** - `dbaecc2` (docs)
3. **Task 3: Draft ADR-BE-016 (schema encoding)** - `4285f68` (docs)

## Files Created/Modified

- `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` - revised to v0.2: seven-status lifecycle + reopen + guarded dismissal, closed kind set, internal-only + D-15 gates, dismissed_hours semantics, delta section; hard boundary list verbatim
- `hourglass-vault/decisions/project/ADR-P-013 — Origins.md` - NEW: origin types, storage (discriminator + columns, EAV/JSONB rejected), same-org validation, ErrOriginImmutable, creation gates, proposal approval, reviewed_by OQ1 resolution, FND-04 fallback
- `hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md` - NEW: migrations 014-017 shapes, 3VL CHECK guard rule, three-layer model, synchronous in-tx audit, terminal activity, LoggedHours, transition matrix, fast path, sold_hours semantics, BE-012 scope extension
- `hourglass-vault/decisions/project/_index.md` - P-003 entry revised (date + summary), P-013 row added
- `hourglass-vault/decisions/backend/_index.md` - BE-016 row added

## Decisions Made

- **reviewed_by OQ1 resolution recorded as binding**: NULL at creation, approver in the `proposal_approved` audit row, create with non-nil `reviewed_by` → `ErrInvalidRequest`; a future phase may pin it at creation-time (recorded in P-013; enforced in 11-05 Task 2).
- **Audit durability**: synchronous in-transaction writes for ticket events (Pitfall 2); entry approvals keep ADR-BE-012 fire-and-forget; outbox recorded as the successor of the user-deferred durability choice so it stays reversible (recorded in BE-016; implemented in 11-05/11-06).
- **Terminal activity definition**: no non-terminal time entries on the linked-activity subtree via recursive CTE — the `resolved` transition's precondition (recorded in BE-016; implemented in 11-06 Task 2).
- **Dismissal guard signature**: `LoggedHours(ctx, ticketID)` on raw Σ, stable for the Phase-12 net-of-compensations swap (recorded in BE-016; implemented in 11-06 Task 2).
- **Transition matrix pinned** (OQ6 edges incl. `open→dismissed` superset of A7); closed/dismissed terminal; reopen only from resolved (recorded in BE-016; enforced via `CanTransition` in 11-05 Task 1).
- **Pre-triage fast path allowed**: POST /activities with origin customer_ticket while ticket is open/triage, manager+ gated (OQ5; recorded in BE-016, implemented in 11-05 Task 2).

## Deviations from Plan

None - plan executed exactly as written. All three tasks' acceptance criteria and the plan-level verification (vault-only changes) pass.

## Issues Encountered

- **Pre-existing untracked vault files (out of scope, not caused by this plan):** `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md` and `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` are untracked in git (created in a prior session, never committed). Both are linked/referenced by the ADRs drafted here, so the committed ADR graph would dangle if they never land. Left untouched per scope boundary (not in this plan's `files_modified`); flagged for the orchestrator/user to commit deliberately. Also `hourglass-vault/.obsidian/workspace.json` shows a pre-existing modification (Obsidian UI state) — left untouched.
- **Case-sensitive grep on `kanban`:** the verbatim hard-boundary list capitalizes "Kanban", which would not match the acceptance grep `grep -c 'kanban'`; the lowercase term appears in the delta section, satisfying the verification chain while keeping the boundary list verbatim.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **11-04 (sold_hours)** and **11-05 (origins/proposals)** can now compile their service/domain work against pinned decision records: P-013 is the constraint source for origin validation/immutability/gates; BE-016 pins the migration shapes the code reads.
- **11-06 (tickets)** implements against BE-016's semantic resolutions: transition matrix (`CanTransition`, `ErrInvalidTransition`), terminal-activity check (recursive CTE), dismissal guard (`LoggedHours`), synchronous in-tx audit rows, triage in-tx validation.
- The plan's `key_link` (P-013 → P-003 via `customer_ticket` origin refs) is embedded: P-013's customer_ticket row links the revised P-003, and P-003's Optional links bullet references P-013.
- Suggested follow-up (orchestrator): commit the two untracked vault files (ADR-P-012 + research note) so the decision graph is complete in history.

## Self-Check: PASSED

- Created files verified on disk: ADR-P-013 — Origins.md, ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md (`[ -f ]` both true)
- Commits verified in git log: 7616a1d (docs P-003 rev), dbaecc2 (docs P-013), 4285f68 (docs BE-016)
- All acceptance greps pass per task (statuses ×7 + reopen edge, kind set, kanban/sub-task trees/comment thread, dismissed_hours + guard; manager_assignment/employee_proposal/customer_ticket + ErrOriginImmutable + reviewed_by + FND-04/Phase-13; audit_logs/ticket_comments/state-comments-audit + synchronously/in-transaction + terminal + LoggedHours + outbox)
- Plan-level verification: `git diff --name-only HEAD~3 HEAD` shows only the 5 planned vault files; no `.go` or `migrations/` files touched

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
