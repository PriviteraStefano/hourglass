---
status: needs-work
phase: 11-foundations-schema-origins-tickets-backend
generated: 2026-08-07
depth: standard
files_reviewed: 43
findings:
  critical: 1
  warning: 5
  info: 7
  total: 13
---

# Phase 11 Code Review Report — Foundations: Schema, Origins, Tickets (Backend)

**Reviewed:** 2026-08-07
**Depth:** standard
**Files Reviewed:** 43 (migrations 014–017, ticket/activity/contract/audit domain + services + repos, routing package, HTTP handlers, tests, vault index docs)
**Status:** needs-work

## Summary

The phase delivers four clean additive migrations with up/down symmetry and three-valued-logic CHECK guards, a faithful verbatim extraction of the BE-014 routing package, a well-tested origin axis on activities (role gates, same-org validation, service+repo immutability double-enforcement), contract sold-hours wiring, and a ticket lifecycle whose audit-in-tx discipline (Pitfall 2) and atomic triage with in-tx plan validation (Pitfall 7) are genuinely well executed. Tests are strong: the transition matrix, dismissal guard (submitted/draft/deleted variants), triage rollback atomicity, and the no-DELETE contract test all assert real acceptance criteria.

The dominant defect is a **check-then-act TOCTOU across the ticket state machine**: every matrix/guard decision (CanTransition, LoggedHours, HasNonTerminalActivities) is evaluated at pool level *outside* the mutator transaction, and the mutators neither lock nor re-validate inside the tx (except triage, which locks with `FOR UPDATE` but never re-checks status). Concurrent legitimate requests can therefore violate the pinned matrix (dismissed ticket resurrected to `planned`, `planned` ticket dismissed) and bypass the T-11-07 dismissal guard (entry submitted between the Σ and the write). Secondary issues: proposal approval is not atomic with its audit row, several validation errors surface as 500 (contract sold config, mixed origin refs, oversized titles), and `ErrInvalidSoldConfig` is never mapped in the contract Update handler.

## Critical Issues

### CR-01: State-machine and dismissal-guard checks run outside the transaction — concurrent requests break the matrix and bypass the guard

**Files:**
- `internal/core/services/ticket/ticket.go:191-233` (Transition), `:259-287` (Dismiss), `:325-379` (Triage)
- `internal/adapters/secondary/postgres/ticket_repository.go:220-247` (UpdateState), `:253-281` (Dismiss), `:297-440` (Triage)

**Issue:** `CanTransition`, `LoggedHours` and `HasNonTerminalActivities` are all read from the pool *before* the mutator tx opens. The mutators then write status without any locking (UpdateState, Dismiss) and without re-validating the matrix inside the tx. Triage is the exception that proves the gap: it takes `SELECT … FOR UPDATE` on the ticket row (ticket_repository.go:305-314) but never re-checks the captured `currentStatus` against the matrix before flipping to `planned`. Three concrete race outcomes:

1. **Dismissed-ticket resurrection (terminal-state invariant broken, D-14):** ticket is in `triage`. T1 (Triage) passes `CanTransition(triage, planned)`; T2 (Dismiss) passes `CanTransition(triage, dismissed)` and commits first (no lock on Dismiss's path). T1's `FOR UPDATE` then reads status `dismissed` — a terminal state — and proceeds to set `planned` and insert activities.
2. **Illegal `planned → dismissed`:** the mirror image — Triage commits first, then Dismiss's lock-free `UPDATE … SET status='dismissed'` lands on a `planned` ticket (dismissal is only legal from `open|triage` per the pinned matrix).
3. **Dismissal-guard bypass (T-11-07):** `LoggedHours` reads 0 at pool level; an employee submits an entry on a linked activity (the time-entry path shares no lock with the ticket); `repo.Dismiss` then commits with `dismissed_hours=0` while logged hours > 0 exist. The server-side Σ guard — the phase's headline security control — is check-then-act.

The resolved-block has the same shape: `HasNonTerminalActivities` is checked before `UpdateState`, so `resolved` can land while a draft entry is concurrently created on the subtree (OQ2).

**Fix:** Move the authoritative checks inside the tx, under a ticket row lock, mirroring the triage pattern everywhere:

```go
// UpdateState (and Dismiss):
tx, _ := r.pool.BeginTx(ctx, pgx.TxOptions{})
defer tx.Rollback(ctx)
var currentStatus string
err := tx.QueryRow(ctx, `SELECT status FROM tickets WHERE id=$1 AND org_id=$2 FOR UPDATE`, ticketID, orgID).Scan(&currentStatus)
// …re-validate the matrix (service passes expected-from, or the service
// passes a re-validate callback) and — for Dismiss — re-run the Σ inside
// the tx (`SELECT COALESCE(SUM(hours),0) FROM time_entries …` via tx),
// then UPDATE + audit INSERT + Commit.
```

Triage additionally needs `if !ticketdomain.CanTransition(currentStatus, ticketdomain.StatusPlanned) { return ErrInvalidTransition }` right after the `FOR UPDATE` read. Alternatively, gate the writes with a status precondition in the SQL: `UPDATE tickets SET status=$1 WHERE id=$2 AND org_id=$3 AND status=$4` and map 0 rows-affected to `ErrInvalidTransition` — a cheap belt-and-braces that makes the matrix race-proof even without the lock.

## Warnings

### WR-01: Proposal approval is not atomic — is_active flip and `proposal_approved` audit row commit in separate transactions

**File:** `internal/core/services/activity/activity.go:319-335`

**Issue:** `ApproveProposal` calls `s.activityRepo.Update(...)` (own tx, commits) and then `s.auditRepo.Create(...)` (second tx). If the audit insert fails after the flip committed (transient DB error), the proposal is approved with no `proposal_approved` row — exactly the repudiation the phase's own T-11-08 mitigation exists to prevent, and inconsistent with the ticket path's in-tx discipline (Pitfall 2, ADR-BE-016). The ticket service got the in-tx treatment; the proposal path is the same invariant and should get it too.

**Fix:** Either (a) add an audit row to the activity `Update` tx (extend `UpdateActivityRequest`/repo with an optional audit param written in the same tx), or (b) flip `is_active` and insert the audit in a single dedicated tx inside the repo (e.g., `ApproveProposal(ctx, orgID, activityID, actorID)` on the repo), so the flip is not durable without its event.

### WR-02: Contract sold-config validation gaps surface as 500s — `ErrInvalidSoldConfig` never mapped; `sold_period` vocabulary bypassed when `contract_type` absent

**Files:**
- `internal/adapters/primary/http/contract.go:180-189` (Update switch)
- `internal/core/services/contract/contract.go:104-127` (validateSoldConfig)

**Issue:** Two related defects. (1) The new `ErrInvalidSoldConfig` sentinel (this phase's addition) falls through the Update handler's `switch err` to the default → **500 Internal Server Error** for a plain client validation error (`support` without hours, `project` with period, bogus period). (2) `validateSoldConfig` returns early when `contractType == nil`, so the `sold_period` closed-set check is skipped — `{sold_period: "bogus"}` with no `contract_type` passes service validation and is persisted (the 016 CHECK's 3VL guard lets `contract_type IS NULL` rows pass). Additionally, clearing `sold_period: ""` on an existing `support` contract (without switching to `project` in the same request) sails through the service and then hits `contracts_sold_check` (23514), which `wrapPGError` does not map → 500.

**Fix:**
```go
// handler: add the missing case
case errors.Is(err, contractdomain.ErrInvalidSoldConfig):
    api.RespondWithError(w, http.StatusBadRequest, "invalid sold hours configuration")
// service: hoist the period vocabulary check out of the contractType branch
if soldPeriod != nil {
    switch *soldPeriod { /* month|quarter|year else ErrInvalidSoldConfig */ }
}
if contractType == nil { return nil }
```
And in `Update`, when the request clears `sold_period`, reject unless the persisted `contract_type` is `project`/NULL (load the row before validating, or let the service check `contract_type` from the DB state).

### WR-03: Origin ref-set exclusivity is not validated in the service — mixed-ref requests hit the DB CHECK and surface as 500

**File:** `internal/core/services/activity/activity.go:130-194` (validateOrigin)

**Issue:** For each origin type the service validates the *required* refs but never the *forbidden* ones: e.g. `origin_type=employee_proposal` with `assigned_by` set, or `manager_assignment` with `proposed_by` set, passes `validateOrigin` and reaches the repo, where `activities_origin_refs_check` (migration 015) rejects with SQLSTATE 23514. `wrapPGError` (postgres.go:16-33) maps only 23505/23503/ErrNoRows, so the client gets a **500** for malformed input instead of the clean `ErrInvalidRequest` → 400 the phase's sentinel style promises. The DB backstop works (integrity holds), but the boundary contract is broken and the error is indistinguishable from an internal fault.

**Fix:** In `validateOrigin`, reject any ref outside the type's set, e.g. for `manager_assignment` add `if req.ProposedBy != nil || req.ReviewedBy != nil || req.TicketID != nil { return ErrInvalidRequest }` (and the analogous exclusions for the other two types), so the CHECK never fires on well-formed requests. Optionally extend `wrapPGError` with a 23514 → `ErrInvalidRequest` mapping as a second line.

### WR-04: Ticket Create/Update handlers lack string-length validation — oversized titles surface as 500

**File:** `internal/adapters/primary/http/ticket_handler.go:96-125, 162-192`

**Issue:** `tickets.title` is `VARCHAR(255)`, but neither `Create` nor `Update` runs the house `validateStringLengths` check used by the activity and contract handlers. A title > 255 chars raises `pgconn.PgError` 22001 (string_data_right_truncation), which `wrapPGError` does not map → 500. Same class of boundary gap the phase fixed elsewhere; the ticket handler is the odd one out.

**Fix:** In `Create`/`Update`, mirror `activity_handler.go:137-141`:
```go
if !validateStringLengths(w,
    lengthField("title", req.Title, MaxNameLength),
    lengthField("description", req.Description, MaxDescriptionLength),
) { return }
```

### WR-05: `ApproveProposal` authorization reads through a non-org-scoped `ActivityRepository.Get`

**Files:**
- `internal/core/services/activity/activity.go:264-281` (ApproveProposal gates)
- `internal/adapters/secondary/postgres/activity_repository.go:108-119` (Get — `WHERE a.id = $2`, orgID used only for the `is_adopted` subquery)

**Issue:** The phase's new proposal-approval path makes authorization decisions (origin type, is_active, proposed_by, self-approval) on a row fetched without an org predicate. A caller of org A passing org B's activity ID gets org B's row back; the gates evaluate against it, and only the final `repo.Update`'s `created_by_org_id = $2` predicate stops the write (surfacing as `ErrActivityNotFound`). So there is no exploit today — but the authorization chain is one repo-query tweak away from being exploitable, and the cross-org read (also via `GET /activities/{id}`, pre-existing since phase 09) leaks other-org rows. Phase 11 built the most security-sensitive handler on top of this read without scoping it. Note the ticket service defensively checks `a.OrgID == orgID` after the same call (ticket.go:446-449) — the activity service should do the same or fix `Get`.

**Fix:** Scope `Get` to the org: `WHERE a.id = $2 AND (a.org_id = $3 OR a.is_shared = true)` (shared activities are legitimately visible cross-org), or at minimum add `if existing.OrgID != orgID { return nil, activitydomain.ErrActivityNotFound }` before the gates in `ApproveProposal` (and validate in `validateOrigin`/`validateParent` call sites that rely on the same read).

## Info

### IN-01: `UpdateDetails` accepts an empty title

**File:** `internal/core/services/ticket/ticket.go:150-183`

`Create` rejects empty titles (TICK-01), but `UpdateDetails` has no non-empty check — `PUT /tickets/{id}` with `{"title": ""}` renames the ticket to an empty string. Add the same `title == ""` rejection in Update.

### IN-02: The "dismissed with N h logged" note is never rendered

**File:** `internal/core/services/ticket/ticket.go:254`

The TICK-04 note is described in comments and the plan ("the note renders from DismissedHours on read") but no code produces it — only `dismissed_hours` is exposed in the JSON. Harmless if the frontend renders it, but the phase's own acceptance text implies server-side rendering. Either render it (e.g., a derived field in the response) or drop the claim from the doc comments.

### IN-03: `ListHistory` ordering is unstable for same-timestamp rows

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:514-520`

`ORDER BY created_at` alone leaves the relative order of same-ms rows (e.g., the triage pair `triaged`/`activities_created` written with the same `now`, or rapid transitions) unspecified. `ORDER BY created_at, id` would make the append-only stream deterministic.

### IN-04: Triage `RETURNING` scans the same variable twice

**File:** `internal/adapters/secondary/postgres/ticket_repository.go:393-395`

`org_id` and `created_by_org_id` both scan into `orgIDOut`. It works (both equal `orgID`) but is fragile — a future column reorder silently corrupts the response. Scan into two locals.

### IN-05: `MockTicketRepo.ListComments` always returns nil

**File:** `internal/core/services/testdata/mock_ticket_repo.go:168-170`

`Get`'s comment list can never be exercised in unit tests; the mock returns `nil, nil` unconditionally (and `Get` never returns comments either). Add a `Comments map[uuid.UUID][]ticketdomain.TicketComment` field so `Get`/`ListComments` can be configured — the missing behavior silently passes tests that would otherwise catch a broken `Get`-with-comments path.

### IN-06: Migration 014 header comment misdescribes the reopen edge

**File:** `migrations/014_ticket_schema.up.sql:4`

Comment says "reopen via 'open'", but the pinned matrix reopens `resolved → in_progress` (domain/ticket/ticket.go:98-101). Fix the comment to match the matrix.

### IN-07: `contracts_sold_check` allows `project` contracts with NULL `sold_hours`

**File:** `migrations/016_contract_sold_hours.up.sql:25-29`

Per D-08/D-09 the project commitment is the total sold hours, but both the CHECK and `validateSoldConfig` accept `contract_type='project'` with `sold_hours` NULL — indistinguishable from a legacy row. If a project contract is *supposed* to carry hours, require `sold_hours IS NOT NULL` for `project` in the CHECK and the service; otherwise document that NULL hours is a valid project state.

---

## Verdict

**Needs work before ship.** The phase is architecturally sound and unusually well tested for its size — migrations, routing extraction, origin axis, and the in-tx audit discipline are all high quality. But the ticket state machine's core guarantee (the pinned matrix + the D-13 dismissal guard, T-11-07) is enforced check-then-act across a transaction boundary, which a modestly timed concurrent request can violate: a dismissed terminal ticket can be resurrected to `planned`, a `planned` ticket can be dismissed, and a dismissal can commit with `dismissed_hours=0` while logged hours exist. That single defect (CR-01) is the reason for the verdict; the warnings (proposal audit atomicity, 500-mapped validation errors, un-scoped authorization reads) are all cheap to fix and should ride along. Once CR-01 is closed (in-tx re-validation under `FOR UPDATE`, plus the SQL status-precondition as backstop), the phase is ready.

_Reviewed: 2026-08-07_
_Reviewer: gsd-code-reviewer (adversarial)_
_Depth: standard_
