---
status: issues_found
phase: 11-foundations-schema-origins-tickets-backend
generated: 2026-08-07T00:00:00Z
depth: standard
files_reviewed: 55
files_reviewed_list:
  - cmd/server/main_test.go
  - cmd/server/main.go
  - hourglass-vault/decisions/backend/_index.md
  - hourglass-vault/decisions/project/_index.md
  - internal/adapters/primary/http/activity_handler.go
  - internal/adapters/primary/http/contract.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/ticket_handler_test.go
  - internal/adapters/primary/http/ticket_handler.go
  - internal/adapters/primary/http/time_entry_test.go
  - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
  - internal/adapters/secondary/postgres/activity_repository_test.go
  - internal/adapters/secondary/postgres/activity_repository.go
  - internal/adapters/secondary/postgres/audit_log_repository.go
  - internal/adapters/secondary/postgres/contract_repository_test.go
  - internal/adapters/secondary/postgres/contract_repository.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/ontology_extension_migrations_test.go
  - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  - internal/adapters/secondary/postgres/ticket_repository_test.go
  - internal/adapters/secondary/postgres/ticket_repository.go
  - internal/core/domain/activity/activity.go
  - internal/core/domain/audit/audit.go
  - internal/core/domain/contract/contract.go
  - internal/core/domain/ticket/ticket.go
  - internal/core/ports/audit_log_repository.go
  - internal/core/ports/ticket_repository.go
  - internal/core/services/activity/activity_origin_test.go
  - internal/core/services/activity/activity_proposal_test.go
  - internal/core/services/activity/activity_test.go
  - internal/core/services/activity/activity.go
  - internal/core/services/contract/contract_test.go
  - internal/core/services/contract/contract.go
  - internal/core/services/routing/routing_test.go
  - internal/core/services/routing/routing.go
  - internal/core/services/testdata/mock_audit_log_repo.go
  - internal/core/services/testdata/mock_ticket_repo.go
  - internal/core/services/testdata/mocks.go
  - internal/core/services/ticket/ticket_integration_test.go
  - internal/core/services/ticket/ticket_test.go
  - internal/core/services/ticket/ticket.go
  - internal/core/services/time_entry/time_entry_test.go
  - internal/core/services/time_entry/time_entry.go
  - migrations/014_ticket_schema.down.sql
  - migrations/014_ticket_schema.up.sql
  - migrations/015_activity_origins.down.sql
  - migrations/015_activity_origins.up.sql
  - migrations/016_contract_sold_hours.down.sql
  - migrations/016_contract_sold_hours.up.sql
  - migrations/017_audit_logs.down.sql
  - migrations/017_audit_logs.up.sql
findings:
  critical: 0
  warning: 6
  info: 5
  total: 11
---

# Phase 11: Code Review Report

**Reviewed:** 2026-08-07T00:00:00Z
**Depth:** standard
**Files Reviewed:** 55
**Status:** issues_found

## Summary

Reviewed the full Phase 11 surface: ticket schema (014), activity origins (015), contract sold-hours (016), audit logs (017), the ticket service/repository/handler with the CR-01 hardening (in-tx re-validation, FOR UPDATE locking), activity origin/proposal flows, approval-routing extraction, and the end-to-end tests. `go build ./...` and `go vet ./...` pass.

**What is solid:** The CR-01 fixes are real — Dismiss re-locks the ticket row, re-checks the matrix in-tx, locks linked activities FOR UPDATE, and recomputes the Σ inside the tx; the race battery (`TestDismissalGuard_RaceWithPendingSubmit`, `TestDismissalRace_VsTriage`, `TestDismiss_RejectsPlanned`) pins the deterministic outcomes. Triage's in-tx EXISTS re-validation closes the kind/parent/contract TOCTOU. Migrations 014–017 are correct (3VL CHECK guards, dependency ordering 014→015, parameterized SQL everywhere). Authz gates are consistently applied per route and role.

**Key concerns:** (1) the resolved-transition guard has the same check-then-act shape the phase fixed for Dismiss — it never locks the linked activities, so a concurrently submitted entry can slip in after the in-tx check and leave a `resolved` ticket with non-terminal entries; (2) the D-13 dismissal Σ is not recursive while the OQ2 resolved gate is, so hours logged on descendant activities bypass the dismissal guard; (3) support→project contract conversion is a functional dead-end that 500s; (4) proposal approval dead-ends when the proposer is the WG manager; (5) ApproveProposal commits the state change before the audit row (Pitfall 2 violation); (6) the TICK-04 dismissed-hours snapshot can only ever be 0 by construction.

No Critical findings in the current state — the prior CR-01 criticals are genuinely fixed.

## Warnings

### WR-01: 'resolved' guard still check-then-act against concurrent time-entry submits

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:268-276`
**Issue:** `UpdateState`'s authoritative OQ2 check (`hasNonTerminalActivitiesTx`) re-runs after the ticket row FOR UPDATE lock — but that lock does **not** serialize against time-entry inserts. The Dismiss path explicitly locks the linked activities FOR UPDATE (line 344-361) because an in-flight `time_entries` INSERT holds a FK KEY SHARE on the activity row; the resolved path locks nothing. In READ COMMITTED, the CTE check reads the pre-insert snapshot, and a submitted/draft entry can commit between the check and the transition commit → ticket lands `resolved` with non-terminal entries. The comment at line 751-753 ("commit-adjacent, never check-then-act") is not true for the entries axis — this is the same CR-01 race class, only with a re-openable outcome (OQ2 violation, not dismissal bypass).
**Fix:**
```go
// in UpdateState, before hasNonTerminalActivitiesTx, mirror Dismiss:
if to == ticketdomain.StatusResolved {
    activityRows, err := tx.Query(ctx,
        `SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket' FOR UPDATE`,
        ticketID)
    // ... drain/close rows (same pattern as Dismiss) ...
    nonTerminal, err := r.hasNonTerminalActivitiesTx(ctx, tx, ticketID)
    // ...
}
```
Add a race test mirroring `TestDismissalGuard_RaceWithPendingSubmit` for the resolved edge.

### WR-02: Dismissal guard Σ and activity lock are non-recursive — descendant hours bypass D-13

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:688-717, 344-361`
**Issue:** `LoggedHours`/`loggedHoursTx` sum entries on **direct** linked activities only, while `HasNonTerminalActivities` (OQ2, the same phase) treats the whole subtree. A user can create a child activity under a customer_ticket activity and log/submit hours there; `Dismiss` then sees Σ=0, the FOR UPDATE lock doesn't cover the child either, and the ticket is dismissed with real logged hours — the note then reads "dismissed with 0 h logged". This defeats the D-13 guard and is inconsistent with the resolved gate's subtree semantics. (Secondary vector: `activity.Service.Create` with a customer_ticket origin validates ticket status open|triage as a pool-level fast-fail only — a manager/finance actor can attach an activity to a ticket that is dismissed between the check and the insert, giving a dismissed ticket new loggable work.)
**Fix:** Recurse the subtree in `LoggedHours`/`loggedHoursTx` (same recursive CTE as `HasNonTerminalActivities`) and lock the subtree in `Dismiss`. If direct-only is the intended D-13 semantics, document the discrepancy explicitly — as written, the two guards disagree about what "linked" means.

### WR-03: support→project contract conversion is a dead-end that returns 500

**File:** `internal/core/services/contract/contract.go:45-53`, `internal/adapters/primary/http/contract.go:180-190`, `internal/adapters/secondary/postgres/contract_repository.go:193-213`
**Issue:** Three stacked problems. (1) Updating `contract_type` to `'project'` on a support contract leaves `sold_period` set in the row → migration 016's `contracts_sold_check` fires → `wrapPGError` → 500. (2) There is no way to clear `sold_period` in the same request: sending `sold_period: ""` is rejected by `validateSoldConfig` (project branch rejects any non-nil `soldPeriod`), so the repo's nullable-clear path (line 203-207) is unreachable for project contracts. (3) The handler's `switch err` doesn't map `ErrInvalidSoldConfig` at all, so even valid-rejection cases surface as 500 instead of 400.
**Fix:** In `Service.Update`, load the existing contract and validate the **merged** config (current row + request deltas) before delegating, and allow `sold_period: ""` as an explicit clear when switching to project; map `ErrInvalidSoldConfig` → 400 in the handler (and add `errors.Is` fallbacks for wrapped repo errors).

### WR-04: Proposal approval dead-ends when the proposer is the WG manager

**File:** `internal/core/services/activity/activity.go:305-317`
**Issue:** When the proposer is the WG manager (or a delegate), `ResolveManagerStage` returns `SkipToFinance: true` (routing.go:77) — for entries that means "skip the manager stage". `ApproveProposal` maps `SkipToFinance` to an unconditional `ErrForbidden` *before* the approver-set membership check. If the WG has delegates, the approver set is `{proposer, delegate}` and the delegate is a legitimate approver — but the delegate is rejected too, because the skip short-circuits first. The proposal becomes unapprovable. The comment "the proposer IS the only approver" is only true when the set has exactly one member.
**Fix:** Only reject when the approver set is exactly `{proposer}`; otherwise fall through to the membership check:
```go
case res.SkipToFinance:
    if len(res.ApproverIDs) == 1 { // set == {proposer}
        return nil, activitydomain.ErrForbidden
    }
    // fall through to membership check below
```
or distinguish "owner in set" from "set == {owner}" in routing. Add a test with proposer-as-manager + delegate.

### WR-05: ApproveProposal commits the state change before the audit row (Pitfall 2 violation)

**File:** `internal/core/services/activity/activity.go:319-335`
**Issue:** `s.activityRepo.Update(...)` commits `is_active=true`, and only then `s.auditRepo.Create(...)` runs in a separate, non-transactional statement. If the audit insert fails (transient DB error), the proposal is approved with no `proposal_approved` audit row, and the API returns 500 while the mutation is already durable — exactly the partial-commit the phase's own ADR-BE-016 Pitfall 2 ("state write and its audit row in the SAME transaction") was written to prevent, and which the ticket repo enforces. Also `CreatedAt: time.Now()` is local time, unlike the UTC convention elsewhere.
**Fix:** Add a repo method that flips `is_active` and inserts the audit row in one tx (mirror `TicketRepository.UpdateState`), or accept a compensating delete/rollback on audit failure. Use `time.Now().UTC()`.

### WR-06: dismissed_hours snapshot is always 0 by construction — TICK-04 note is dead functionality

**File:** `internal/core/services/ticket/ticket.go:277-305`, `internal/adapters/secondary/postgres/ticket_repository.go:363-373`
**Issue:** The D-13 guard blocks dismissal whenever Σ(submitted+approved) > 0 — at the pool level and again in-tx. Therefore every successful dismissal persists `dismissed_hours = 0`, and `scanTicketRow`'s derived note always renders "dismissed with 0 h logged" (confirmed by the handler test asserting exactly that). The non-zero note path is only reachable via direct SQL inserts in repo tests. The snapshot feature (TICK-04) contradicts the guard: the only hours the Σ counts are exactly the hours that block dismissal. Additionally, draft entries are not in the Σ, so a ticket with draft entries can be dismissed and those drafts can later be submitted — hours get logged on a dismissed ticket after the fact.
**Fix:** Reconcile the spec: either (a) make the Σ the *dismissal-time total across all non-deleted entries* and snapshot it while the guard blocks only in-flight/submittable states, or (b) drop the snapshot + note entirely. At minimum, block subsequent submits on activities of dismissed tickets (entry Submit has no ticket-state check).

## Info

### INF-01: Unused parameters in repo signatures

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:239, 313`
**Issue:** `UpdateState`'s `note *string` is never read in the body (the note only lives in the service-built audit payload), and `Dismiss`'s `hours float64` is never read (the in-tx `sum` supersedes it). Both are misleading: a future caller may assume the passed values are authoritative. Either use them or drop them from the port/implementation and document that the in-tx computation is the source of truth (the audit payload `hours` in `ticket.go:302` likewise always records the pool-level value, which is always 0 on success).

### INF-02: No-op UpdateDetails writes a spurious 'updated' audit row

**File:** `internal/core/services/ticket/ticket.go:152-191`, `ticket_repository.go:183-226`
**Issue:** When the request carries no fields (all pointers nil), the repo still bumps `updated_at` and writes an `updated` audit row with an empty payload. Also, an assignee can never be unassigned: `assignee_id: ""` parses to nil = "absent" (handler `parseOptionalUUIDPtr`), so there is no clear-assignee operation. Consider rejecting empty updates (400) and adding a clear semantics if unassignment is a product need.

### INF-03: Triage plan activity name length is unvalidated

**File:** `internal/core/services/ticket/ticket.go:367`, `ticket_handler.go:225`
**Issue:** The activity endpoint validates name length at the boundary (`MaxNameLength`), but the triage path only checks non-empty. A >255-char plan name (activities.name is VARCHAR(255)) surfaces as a raw 500 from the DB via `wrapPGError`. Low likelihood, but the fix is one line in the service loop.

### INF-04: Local time vs UTC inconsistency

**File:** `internal/core/services/activity/activity.go:332`
**Issue:** `CreatedAt: time.Now()` in `ApproveProposal`'s audit row, while the ticket service consistently uses `time.Now().UTC()`. Mixing local and UTC timestamps in the same audit stream makes ordering/comparisons ambiguous. Use UTC.

### INF-05: `switch err` without `errors.Is` in contract handler

**File:** `internal/adapters/primary/http/contract.go:181, 215, 237`
**Issue:** The contract handler compares errors with `==`; service repo errors are wrapped (`wrapPGError`), so a wrapped `ErrContractNotFound`/FK violation falls to the default 500 branch with a generic message. This predates the phase for most paths, but the new `ErrInvalidSoldConfig` sentinel (WR-03) should at minimum be added to the switch, and converting the switch to `errors.Is` would harden all cases.

---

_Reviewed: 2026-08-07T00:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
