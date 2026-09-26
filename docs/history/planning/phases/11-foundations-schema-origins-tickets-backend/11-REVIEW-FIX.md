---
phase: 11-foundations-schema-origins-tickets-backend
fixed_at: 2026-08-07T21:45:00Z
review_path: .planning/phases/11-foundations-schema-origins-tickets-backend/11-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 11: Code Review Fix Report

**Fixed at:** 2026-08-07T21:45:00Z
**Source review:** .planning/phases/11-foundations-schema-origins-tickets-backend/11-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (all warnings; 0 critical)
- Fixed: 6
- Skipped: 0

All fixes were implemented and verified in an isolated git worktree
(`/tmp/sv-11-reviewfix-c5kxfZ`, branch `gsd-reviewfix/11-49547`, later
fast-forwarded to `main`). `go build ./...`, `go vet ./...`, and the
affected package test suites (`internal/adapters/secondary/postgres`,
`internal/core/services/{ticket,time_entry,activity,contract}`,
`internal/adapters/primary/http`, `cmd/server`) all pass on the final
commit.

## Fixed Issues

### WR-01: 'resolved' guard still check-then-act against concurrent time-entry submits

**Files modified:** `internal/adapters/secondary/postgres/ticket_repository.go`, `internal/adapters/secondary/postgres/ticket_repository_test.go`
**Commit:** `6f018af`
**Applied fix:** `UpdateState` now locks the ticket's linked-activity subtree `FOR UPDATE` before the in-tx OQ2 re-check, exactly mirroring the Dismiss hardening. An in-flight `time_entries` INSERT (FK KEY SHARE on the activity row) now blocks until the entry tx commits, so the resolved transition can no longer commit between the check and the write. The lock and the check share one recursive subtree definition (new `lockTicketActivitySubtree` helper). Added `TestUpdateState_Resolved_RaceWithPendingSubmit`, a deterministic 2-tx race pin mirroring `TestDismissalGuard_RaceWithPendingSubmit`: the resolve must block while a draft submit is pending, then refuse with `ErrActivityNotTerminal` after it commits.

### WR-02: Dismissal guard Σ and activity lock are non-recursive — descendant hours bypass D-13

**Files modified:** `internal/adapters/secondary/postgres/ticket_repository.go`, `internal/adapters/secondary/postgres/ticket_repository_test.go`
**Commit:** `6f018af`
**Applied fix:** `LoggedHours`/`loggedHoursTx` now recurse the linked-activity subtree with the same recursive CTE as `HasNonTerminalActivities`, and the Dismiss (and UpdateState) `FOR UPDATE` lock covers the whole subtree — hours logged on a child activity now count toward the D-13 guard exactly like hours on the direct ticket activity. The lock query applies the locking clause to the base `activities` table via a JOIN (a plain CTE reference would not lock the underlying rows). Added `TestLoggedHours_IncludesDescendants` (submitted entry on a child activity → pool Σ = 6.0 and Dismiss → `ErrDismissalBlocked`).

> Note: WR-01 and WR-02 were committed together in `6f018af` because both are implemented through the same shared `lockTicketActivitySubtree` helper in the same file.

### WR-03: support→project contract conversion is a dead-end that returns 500

**Files modified:** `internal/core/services/contract/contract.go`, `internal/adapters/primary/http/contract.go`, `internal/core/services/contract/contract_test.go`, `internal/adapters/primary/http/contract_test.go`
**Commit:** `836dbcd`
**Applied fix:** Three-part fix: (1) `Service.Update` now loads the existing contract and validates the **merged** sold config (current row + request deltas) via new `validateMergedSoldConfig`, so a support→project conversion that leaves `sold_period` set is rejected as `ErrInvalidSoldConfig` instead of passing request-only validation and then 500ing on the DB CHECK; (2) `sold_period: ""` is now accepted as the explicit clear (maps to NULL in the repo's existing nullable-clear branch), making the conversion legal in one request; (3) the HTTP handler maps `ErrInvalidSoldConfig` to 422 (was a 500) and the Update error switch now uses `errors.Is` so wrapped repo errors resolve correctly. Tests: service-level conversion cases (clear succeeds, no-clear rejected, name-only update stays valid, missing contract → 404) and handler-level 422/200 tests. Note: the review text suggested 400; 422 (Unprocessable Entity) was chosen as the more precise status for a well-formed request that fails semantic validation — either is a client error and the endpoint no longer 500s.

### WR-04: Proposal approval dead-ends when the proposer is the WG manager

**Files modified:** `internal/core/services/activity/activity.go`, `internal/core/services/activity/activity_proposal_test.go`
**Commit:** `86a1bf4`
**Applied fix:** The `SkipToFinance` case in `ApproveProposal` no longer short-circuits to `ErrForbidden` before the approver-set check. It rejects only when the approver set is exactly `{proposer}` (`len(res.ApproverIDs) == 1` — routing cannot self-approve); with delegates in the set, the delegate falls through to the membership check and can approve. `seedWGForActivity` gained a variadic delegate parameter and a regression test seeds the proposer as WG manager plus a delegate, asserting the delegate's approval succeeds.

### WR-05: ApproveProposal commits the state change before the audit row (Pitfall 2 violation)

**Files modified:** `internal/core/ports/activity_repository.go`, `internal/adapters/secondary/postgres/activity_repository.go`, `internal/adapters/secondary/postgres/audit_log_repository.go`, `internal/core/services/activity/activity.go`, `internal/core/services/testdata/mocks.go`, `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/adapters/primary/http/handler_test_helper.go`, `internal/core/services/activity/activity_origin_test.go`, `internal/core/services/activity/activity_test.go`, `internal/core/services/activity/activity_proposal_test.go`, `internal/adapters/secondary/postgres/activity_repository_test.go`
**Commit:** `8dfd9e4`
**Applied fix:** The activity repo now exposes `ApproveProposal`, which flips `is_active=true` AND inserts the `proposal_approved` audit row in ONE transaction (mirroring `TicketRepository.UpdateState`). The service calls it with a single audit log and drops its now-unused `auditRepo` dependency; the audit timestamp is `time.Now().UTC()`. The audit insert SQL was extracted into a shared `insertAuditLogTx` helper (used by both the pool-level `GeneralAuditLogRepository.Create` and the same-tx path). Added `TestActivityRepository_ApproveProposal`, which proves both the happy path (flip + audit row) and the atomicity guarantee (an unmarshalable audit payload fails the whole tx — `is_active` stays false and no partial audit row exists).

### WR-06: dismissed_hours snapshot is always 0 by construction — TICK-04 note is dead functionality

**Files modified:** `internal/adapters/secondary/postgres/ticket_repository.go`, `internal/adapters/secondary/postgres/ticket_repository_test.go`, `internal/adapters/secondary/postgres/activity_repository.go`, `internal/adapters/secondary/postgres/activity_repository_test.go`, `internal/core/ports/activity_repository.go`, `internal/core/domain/time_entry/time_entry.go`, `internal/core/services/time_entry/time_entry.go`, `internal/adapters/primary/http/time_entry.go`, `internal/core/services/time_entry/time_entry_test.go`, `internal/core/services/ticket/ticket_integration_test.go`, `internal/core/services/testdata/mocks.go`
**Commit:** `f65544b`
**Applied fix:** Reconciles the TICK-04 semantics per review option (a): the persisted `dismissed_hours` is now the **dismissal-time total across ALL non-deleted entries** (drafts included, computed in-tx via new `totalHoursTx`), while the D-13 guard still blocks only in-flight/submittable hours (submitted+approved). The derived note is therefore meaningful ("dismissed with N h logged" reflects the real total, e.g. draft hours) instead of always 0; the dismissal audit payload is corrected to the same in-tx value. Second half: entry **Submit** now has a ticket-state gate — new `ActivityRepository.IsLinkedTicketDismissed` walks the activity's ancestry (subtree semantics) and Submit returns the new `time_entry.ErrTicketDismissed` (mapped to 409 in the handler), so drafts on a dismissed ticket's activities can never be submitted after the fact. Tests: repo snapshot test (6.5h draft on a child activity → dismissed_hours 6.5 + note + audit payload), repo ancestry-gate test (direct/descendant/open/no-link), service submit-gate tests, and the existing draft-dismissal integration case updated to assert the reconciled note ("dismissed with 8 h logged").

---

_Fixed: 2026-08-07T21:45:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
