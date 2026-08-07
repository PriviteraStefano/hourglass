---
status: issues_found
phase: 11-foundations-schema-origins-tickets-backend
generated: 2026-08-07
depth: standard
files_reviewed: 57
files_reviewed_list:
  - cmd/server/main.go
  - cmd/server/main_test.go
  - hourglass-vault/decisions/backend/_index.md
  - hourglass-vault/decisions/project/_index.md
  - internal/adapters/primary/http/activity_handler.go
  - internal/adapters/primary/http/contract.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/ticket_handler.go
  - internal/adapters/primary/http/ticket_handler_test.go
  - internal/adapters/primary/http/time_entry_test.go
  - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
  - internal/adapters/secondary/postgres/activity_repository.go
  - internal/adapters/secondary/postgres/activity_repository_test.go
  - internal/adapters/secondary/postgres/audit_log_repository.go
  - internal/adapters/secondary/postgres/contract_repository.go
  - internal/adapters/secondary/postgres/contract_repository_test.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/ontology_extension_migrations_test.go
  - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  - internal/adapters/secondary/postgres/ticket_repository.go
  - internal/adapters/secondary/postgres/ticket_repository_test.go
  - internal/core/domain/activity/activity.go
  - internal/core/domain/audit/audit.go
  - internal/core/domain/contract/contract.go
  - internal/core/domain/ticket/ticket.go
  - internal/core/ports/audit_log_repository.go
  - internal/core/ports/ticket_repository.go
  - internal/core/services/activity/activity.go
  - internal/core/services/activity/activity_origin_test.go
  - internal/core/services/activity/activity_proposal_test.go
  - internal/core/services/activity/activity_test.go
  - internal/core/services/contract/contract.go
  - internal/core/services/contract/contract_test.go
  - internal/core/services/routing/routing.go
  - internal/core/services/routing/routing_test.go
  - internal/core/services/testdata/mock_audit_log_repo.go
  - internal/core/services/testdata/mock_ticket_repo.go
  - internal/core/services/testdata/mocks.go
  - internal/core/services/ticket/ticket.go
  - internal/core/services/ticket/ticket_integration_test.go
  - internal/core/services/ticket/ticket_test.go
  - internal/core/services/time_entry/time_entry.go
  - internal/core/services/time_entry/time_entry_test.go
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
  warning: 3
  info: 1
  total: 4
---

# Phase 11 Code Review Report — RE-review (CR-01 / 11-07 / 11-08 verification)

**Reviewed:** 2026-08-07
**Depth:** standard
**Files Reviewed:** 57
**Status:** issues_found

## Summary

This is the re-review of phase 11 after the two gap-closure plans landed (11-07 = CR-01 hardening, 11-08 = note rendering + title validation). **Both gap-closure plans are verified complete and correct:**

- **CR-01 (11-07) — closed as scoped.** `repo.UpdateState` (ticket_repository.go:239-298), `repo.Dismiss` (:313-396) and `repo.Triage` (:412-565) now all re-validate inside the mutator transaction under a `FOR UPDATE` ticket row lock: in-tx `CanTransition` on the locked status (dismissed-resurrection and planned→dismissed are closed), the resolved-edge re-check via the in-tx recursive-CTE helper `hasNonTerminalActivitiesTx`, the dismissal Σ re-check via `loggedHoursTx` under linked-activity `FOR UPDATE` locks (serializing the entry-INSERT path), and every UPDATE carries the `AND status = currentStatus` SQL backstop. The six new race/pin tests (TestDismissalGuard_RaceWithPendingSubmit, TestDismissalRace_VsTriage, TestDismiss_RejectsPlanned, TestTransitionRace_VsDismiss, TestTriage_RejectsDismissed, TestUpdateState_ResolvedBlocked) are well-constructed and pass under `go test -race`. No port signature changes.
- **11-08 — closed.** `DismissedNote` is a derived field rendered in `scanTicketRow` (ticket_repository.go:59-63) with `strconv.FormatFloat` precision -1, asserted at repo level and across the dismiss response / GET detail / GET list in the handler contract test. Title validation (Create `len > 255`, UpdateDetails empty + `len > 255` → `ErrInvalidRequest` → 400) landed in the service with 255/256 boundary tests.
- **Known warnings WR-02 (contract sold-config) and WR-03 (forbidden origin refs) remain open** — verified untouched by the gap-closure commits (git diff 7713072..HEAD touches only ticket-subsystem files), per scope.
- Full build green; service/http/postgres ticket suites green, race battery green.

**New findings (this review):** the dismissal guard still has **one deterministic hole** and **one residual race window**, plus a latent guard-by-convention gap in the repo authority layer:

1. **WR-06 (deterministic):** the dismissal Σ (`LoggedHours`/`loggedHoursTx`) counts only `status IN ('submitted','approved')` — an entry at `pending_manager`/`pending_finance` (already submitted and mid-approval) **stops blocking dismissal the moment the manager approves it**. The more committed the logged hours, the weaker the guard. No concurrency required.
2. **WR-07 (race):** the linked-activity `FOR UPDATE` serialization blocks only entry **INSERTs** (FK KEY SHARE); the real submit flow is a `draft → submitted` **UPDATE** on `time_entries`, which takes no activity-row lock — it can commit in the window between the in-tx Σ read and the dismissal commit, landing submitted hours on a ticket dismissed with `dismissed_hours=0`.
3. **WR-08 (latent):** `repo.UpdateState` still accepts the matrix-legal dismissal edges (`open|triage → dismissed`) with no Σ check and no `dismissed_hours` write — the "repo is authoritative" layer can dismiss without the guard; only the service's ad-hoc rejection protects the invariant.

## Warnings

### WR-06: Dismissal-guard Σ excludes `pending_manager`/`pending_finance` — an entry mid-approval stops blocking dismissal (deterministic T-11-07 bypass)

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:691-693` (LoggedHours) and `:709-711` (loggedHoursTx)

**Issue:** Both Σ queries filter `status IN ('submitted','approved')`. The time_entry lifecycle is `draft → submitted → pending_manager → pending_finance → approved` (time_entry domain constants, time_entry.go:21-26). A linked entry therefore **blocks dismissal while it is `submitted`, but stops blocking the moment the manager approves it (`submitted → pending_finance`)** — and finance approval later lands `approved` hours on a ticket dismissed with `dismissed_hours=0` and the false note "dismissed with 0 h logged". This is a deterministic, sequential bypass of T-11-07 — no race involved — and it contradicts the guard's own sibling check: `HasNonTerminalActivities`/`hasNonTerminalActivitiesTx` (ticket_repository.go:739, 765) *do* treat `pending_manager`/`pending_finance` as non-terminal for the resolved-block. D-13's letter ("submitted + approved") matches the code, so the gap is in the pinned status set vs. the guard's documented purpose ("dismissal can never commit with `dismissed_hours=0` while committed logged hours exist"). Nothing in the 11-07 race battery exercises an entry past `submitted`.

**Fix:** Include the in-pipeline statuses in both Σ queries (and pin the corrected set in ADR-BE-016 D-13):
```sql
WHERE is_deleted = false AND status IN ('submitted','pending_manager','pending_finance','approved')
  AND activity_id IN (SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket')
```
(or `status <> 'draft' AND status <> 'rejected'`). Add a repo test: entry at `pending_finance` → `Dismiss` returns `ErrDismissalBlocked`.

### WR-07: Residual check-then-act window — the `draft → submitted` UPDATE is not serialized by the dismissal locks

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:344-361` (Dismiss activity-lock section) and `:367-373` (Σ re-check)

**Issue:** The 11-07 serialization relies on "an in-flight time_entries INSERT holds a FK KEY SHARE on the activity row, so FOR UPDATE blocks until it commits". That is true for **INSERTs only**. The actual submit flow is `UPDATE time_entries SET status='submitted' WHERE id=$1` (services/time_entry/time_entry.go:162, repo Update) — an UPDATE that does not touch FK columns takes **no lock on the activity row**. A draft entry committed *before* the dismiss tx (the realistic case) can transition to `submitted` in the window between `loggedHoursTx` and the dismissal COMMIT: the Σ (fresh READ COMMITTED snapshot) sees 0, dismissal commits with `dismissed_hours=0`, and a submitted entry exists on the dismissed ticket. The deterministic 2-tx test (TestDismissalGuard_RaceWithPendingSubmit) only covers the INSERT-in-flight case (raw INSERT with status 'submitted'), not the real draft→submitted UPDATE path — so the guard remains check-then-act for the exact flow CR-01 race 3 described.

**Fix:** Lock the counted entry rows inside the dismiss tx so a concurrent status UPDATE blocks until dismissal commits — e.g., replace the aggregate in `loggedHoursTx` with a locking read:
```go
rows, err := tx.Query(ctx, `SELECT hours FROM time_entries
    WHERE is_deleted = false AND status IN ('submitted','pending_manager','pending_finance','approved')
      AND activity_id IN (SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket')
    FOR UPDATE`, ticketID)
// aggregate in Go
```
(Alternatively: reject `Submit` on entries whose activity's ticket is dismissed, service-side.) Add a test mirroring TestDismissalGuard_RaceWithPendingSubmit but with the *submit UPDATE* (draft committed, then `UPDATE ... SET status='submitted'`) in the racing tx.

### WR-08: `repo.UpdateState` still permits the dismissal edges without the T-11-07 guard — guard-by-convention at the authoritative layer

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:262-264` (UpdateState matrix re-check)

**Issue:** The transition matrix (domain/ticket/ticket.go:89-108) legally allows `open → dismissed` and `triage → dismissed`, and the CR-01 layering made the repo the authoritative matrix enforcer. `UpdateState` therefore **succeeds for `to == 'dismissed'`** — no Σ re-check, no `dismissed_hours` write, audit action `status_changed` — producing a dismissed ticket with `dismissed_hours IS NULL` and no note, bypassing T-11-07 entirely. Today only the service's ad-hoc rejection (services/ticket/ticket.go:219-221) protects the invariant; the repo — the layer the phase now calls authoritative — does not. A future caller (Phase 12+ code) using `UpdateState` for dismissal would silently skip the guard, and nothing in the race battery pins this (all tests route dismissal through `repo.Dismiss`).

**Fix:** In `UpdateState`, reject the dismissal edges explicitly (mirroring the service guard), or require `dismissed_hours` to be set and run `loggedHoursTx` when `to == StatusDismissed`:
```go
if to == ticketdomain.StatusDismissed {
    return nil, ticketdomain.ErrInvalidTransition // Dismiss is the only sanctioned path (T-11-07)
}
```
Add a pin test: `repo.UpdateState(orgID, ticketID, 'dismissed', ...)` on an `open` ticket → `ErrInvalidTransition`, no status change.

## Info

### IN-08: A ticket's assignee can never be unassigned — `assignee_id: ""` silently no-ops

**File:** `internal/adapters/primary/http/ticket_handler.go:368-377` (parseOptionalUUIDPtr)

**Issue:** `parseOptionalUUIDPtr` maps `nil` **and `""`** to `nil`, and the service treats `nil` as "field untouched" — so `PUT /tickets/{id}` with `{"assignee_id": ""}` neither clears the assignee nor errors; it is silently ignored. Once assigned, a ticket can only be re-assigned, never unassigned (a plausible ops need — the ticket moves back to the unassigned queue). The contract test doesn't cover the clear path.

**Fix:** Distinguish "absent" from "explicit clear" (e.g., a `*string` in the service that can be dereferenced to an empty UUID-pointer sentinel, or a dedicated `assignee_id: null` → NULL branch in `UpdateDetails`), and write `assignee_id = NULL` when the request explicitly clears it.

---

## Prior-review verification appendix

- **CR-01 closed:** in-tx `FOR UPDATE` re-validation + `CanTransition` on locked status in UpdateState/Dismiss/Triage (ticket_repository.go:248-264, 322-338, 420-437); `AND status = current` backstops on all three UPDATEs (:278-280, :375-378, :488-490); in-tx Σ re-check writing the authoritative value (:367-373); in-tx recursive-CTE resolved-block (:268-276). Race battery green under `-race` (6 tests, run this review).
- **11-08 closed:** `DismissedNote` derived in `scanTicketRow` (:59-63), asserted in the dismiss response, GET detail and GET list (ticket_handler_test.go); title length/emptiness validation in service Create/UpdateDetails (services/ticket/ticket.go:84, :169-171) with 255/256 boundary tests.
- **WR-02 / WR-03 (contract sold-config mapping + sold_period vocab; forbidden origin refs):** remain open by design — out of scope for the gap-closure plans, confirmed untouched.

_Reviewed: 2026-08-07_
_Reviewer: gsd-code-reviewer (adversarial, re-review)_
_Depth: standard_
