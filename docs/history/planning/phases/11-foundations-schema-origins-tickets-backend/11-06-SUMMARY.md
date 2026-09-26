---
phase: 11-foundations-schema-origins-tickets-backend
plan: 06
subsystem: api
tags: [go, hexagonal, tickets, state-machine, audit-logs, triage, dismissal-guard, postgres, txn]

# Dependency graph
requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: ticket domain (entity, transition matrix, sentinels) + audit domain + base TicketRepository.Get (11-05) + origin axis on activities (customer_ticket refs, OQ5)
provides:
  - ticket lifecycle service: Create/List/Get/UpdateDetails/Transition/Dismiss/Triage/AddComment/ListHistory with D-15/D-11 gates and the closed-set kind/status vocabularies
  - 12-method TicketRepository port + tx-backed postgres implementation: every mutator writes its audit_logs row in the SAME transaction (Pitfall 2, ADR-BE-016)
  - atomic triage: kind reclassification + status='planned' + 1..N customer_ticket-origin activities + 'triaged'/'activities_created' audit rows in one tx, with in-tx SELECT EXISTS validation (Pitfall 7, no TOCTOU)
  - guarded dismissal (D-13): server-side LoggedHours Σ (submitted+approved, not deleted) blocks dismissal when > 0; dismissed_hours snapshot persisted
  - HTTP surface: 9 routes (POST/GET /tickets, GET/PUT /tickets/{id}, POST triage/dismiss/transition/comments, GET history) with the full sentinel→status map; deliberately NO DELETE route (TICK-05)
affects: [12-funding-sources (LoggedHours stable signature for the Phase-12 computation swap), 13-direction, frontend ticket surfaces]

# Tech tracking
tech-stack:
  added: [none — pure Go on the existing pgx/testify/testcontainers stack]
  patterns:
    - "State writes + audit rows commit in one tx: the repo takes the audit row(s) to persist, the mutator opens BeginTx, writes both, commits (Pitfall 2, ADR-BE-016) — no fire-and-forget"
    - "Authoritative in-tx validation for triage plans (SELECT EXISTS on activity_kinds/activities/contracts inside the tx) with service-level fast-fail as optional UX only — no TOCTOU window (Pitfall 7, T-11-06)"
    - "Guarded dismissal: the D-13 hours Σ is computed server-side (LoggedHours) and the pinned matrix edges open→dismissed/triage→dismissed are consumed ONLY by Dismiss — Transition rejects them, so the guard cannot be bypassed (T-11-07)"
    - "Append-only by construction: no UPDATE/DELETE paths on audit_logs or ticket_comments in the repo; a grep-level test enforces it (TICK-05)"

key-files:
  created:
    - internal/core/services/ticket/ticket.go
    - internal/core/services/ticket/ticket_test.go
    - internal/core/services/ticket/ticket_integration_test.go
    - internal/adapters/primary/http/ticket_handler.go
    - internal/adapters/primary/http/ticket_handler_test.go
  modified:
    - internal/core/ports/ticket_repository.go
    - internal/adapters/secondary/postgres/ticket_repository.go
    - internal/core/services/testdata/mock_ticket_repo.go
    - internal/adapters/secondary/postgres/ticket_repository_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main.go

key-decisions:
  - "Transition rejects to==dismissed: the pinned matrix's dismissal edges are consumed ONLY by Dismiss (the guarded path with the D-11 role gate + D-13 hours guard + dismissed_hours snapshot). Allowing them in Transition would let an owner/assignee bypass the guard (T-11-07)"
  - "Triage implements the service-level fast-fail (KindExists/parent/contract same-org) as optional UX exactly as planned — the repo's in-tx SELECT EXISTS checks remain the correctness guarantee with DB FK/CHECK constraints as the third line"
  - "Dismissal-guard contract test links the activity via the OQ5 customer_ticket path on an open ticket (the exact shape the guard must catch) rather than via triage — triage moves the ticket to planned, from which dismissal is illegal per the matrix"

patterns-established:
  - "Audit-in-tx mutator shape: BeginTx → state write → insertTicketAudit (entity_type 'ticket', JSONB payload, nullable actor/comment) → Commit; defer Rollback on error"
  - "Handler sentinel map mirrors the ErrHasChildren → 409 house style: ErrDismissalBlocked/ErrActivityNotTerminal → 409 Conflict"
  - "Service-level triage fast-fail helpers (activityKindExists/activityExists/contractExists) keep the optional UX checks in one place; the authoritative checks live in the repo tx"

requirements-completed: [TICK-01, TICK-02, TICK-03, TICK-04, TICK-05]

# Metrics
duration: 44min
completed: 2026-08-07
---

# Phase 11 Plan 6: Ticket Lifecycle Service + Guarded Dismissal + Atomic Triage + HTTP Surface Summary

**The ticket lifecycle lands end-to-end (TICK-01..05): any employee creates with a closed-set kind (customer rejected), the D-14 state machine runs with reopen and a server-side guarded dismissal (D-13 raw Σ blocks dismissal while linked activities carry logged hours), triage atomically converts a ticket into 1..N customer_ticket-origin activities in one transaction with in-tx plan validation (Pitfall 7), comments/transitions/dismissals write their audit_logs rows in the SAME tx as the state change, and the whole surface is exposed through 9 HTTP routes with no DELETE path (TICK-05)**

## Performance

- **Duration:** 44 min
- **Started:** 2026-08-07T12:53:28Z (first commit 25af406)
- **Completed:** 2026-08-07T13:37:19Z
- **Tasks:** 3
- **Files modified:** 11 (5 created, 6 modified)

## Accomplishments

- **Ticket lifecycle service (Task 1, verified + committed):** the pre-existing uncommitted work (service, full 12-method port, tx-backed repo, extended mock) was verified against the plan's acceptance criteria, one missing test helper (`strPtr`) was added, and the chunk was committed. Service gates per D-15/D-11: any employee creates (customer → ErrForbidden), owner/assignee/manager+ update+transition+comment, manager|finance triage+dismiss; closed-set kind/status vocabularies validated everywhere.
- **Guarded dismissal + atomic triage + comments + history (Task 2):** `Dismiss` (D-11 gate → CanTransition to dismissed → `LoggedHours` Σ > 0 → ErrDismissalBlocked → repo.Dismiss with the server-side hours snapshot + 'dismissed' audit), `Triage` (structural plan checks + optional fast-fail kind/parent/contract same-org, converting plans to CreateActivityRequest and handing the repo two audit rows), `AddComment` (D-15 gate, body non-empty, comment + 'comment_added' audit in one tx), `ListHistory` (customer rejected). **Security fix:** `Transition` now rejects `to == "dismissed"` so the D-13 hours guard cannot be bypassed via the transition endpoint (T-11-07). Integration tests prove triage tx atomicity (a failing plan rolls back everything), origin/ticket_id on created activities, the dismissal guard on submitted vs draft vs deleted entries, and the ordered append-only stream.
- **HTTP surface + contract test (Task 3):** `TicketHandler` with 9 methods (Create/List/Get/Update/Triage/Dismiss/Transition/AddComment/History), string IDs parsed at the boundary, and the full sentinel map (404/400/403/409/500). 9 routes registered in `cmd/server/main.go` (all auth-wrapped), ticketService wired with ticketRepo + activityRepo + contractRepo + orgRepo, and the test fixture mirrors the wiring. `TestTicketAPI` drives the whole contract against a testcontainer: employee create 201 → manager lifecycle incl. atomic triage and reopen → non-owner employee 403 → customer create 403 → dismiss-with-hours 409 → dismiss-0-hours 200 with dismissed_hours + 'dismissed' in history → comments with 'comment_added' in history → ordered history → **DELETE /tickets/{id} → 405 (no DELETE route, TICK-05)**.

## Task Commits

Each task was committed atomically:

1. **Task 1: Ticket service state machine + permissions + audit-in-tx (verified uncommitted work + `strPtr` fix)** - `25af406` (feat)
2. **Task 2: Atomic triage + dismissal guard + comments + history** - `5809a4a` (feat)
3. **Task 3: Ticket HTTP handler + 9 route registration + API contract test** - `c3858e5` (feat)

**Plan metadata:** (docs commit below)

## Files Created/Modified

- `internal/core/services/ticket/ticket.go` - NEW: Service with Create/List/Get/UpdateDetails/Transition/Dismiss/Triage/AddComment/ListHistory, closed vocabularies, canUpdate gate, triage fast-fail helpers
- `internal/core/services/ticket/ticket_test.go` - NEW: create gates, lifecycle matrix (reopen, resolved-block, permissions), update gates, list/get, dismissal guard, triage pre-validation, comment gating, history
- `internal/core/services/ticket/ticket_integration_test.go` - NEW: TestTicketTriage (tx atomicity + origin + 2 audit rows), TestDismissalGuard (submitted/draft/deleted), TestTicketAudit (ordered stream + actor capture)
- `internal/adapters/primary/http/ticket_handler.go` - NEW: 9 handler methods + DTOs + sentinel map
- `internal/adapters/primary/http/ticket_handler_test.go` - NEW: TestTicketAPI full contract incl. no-DELETE-route assertion
- `internal/core/ports/ticket_repository.go` - EXTENDED to the 12-method interface (Create/ListByOrg/UpdateDetails/UpdateState/Dismiss/Triage/AddComment/ListComments/ListHistory/LoggedHours/HasNonTerminalActivities + Get)
- `internal/adapters/secondary/postgres/ticket_repository.go` - EXTENDED: tx-backed mutators with in-tx audit INSERTs, ListByOrg filters, ListComments (org-scoped join), ListHistory (payload JSONB unmarshal), LoggedHours (D-13 exact query), HasNonTerminalActivities (recursive CTE), Triage with in-tx SELECT EXISTS validation + RETURNING activity inserts
- `internal/core/services/testdata/mock_ticket_repo.go` - EXTENDED to the full interface with audit capture
- `internal/adapters/secondary/postgres/ticket_repository_test.go` - EXTENDED: Create round-trip + audit entity_type 'ticket', UpdateState same-tx audit, triage audit rows, append-only grep guard
- `internal/adapters/primary/http/handler_test_helper.go` - MODIFIED: ticket service + handler + 9 routes wired into the fixture
- `cmd/server/main.go` - MODIFIED: ticketService construction + 9 route registrations

## Decisions Made

- **Dismissal is unreachable via Transition:** the pinned matrix (ADR-BE-016) includes open→dismissed and triage→dismissed, but those edges are consumed only by `Dismiss` — the guarded path (D-11 role gate, D-13 hours guard, dismissed_hours snapshot). `Transition` returns ErrInvalidTransition for `to == "dismissed"`, otherwise an owner/assignee could dismiss a ticket with logged hours and bypass T-11-07. Unit + contract tests pin this.
- **Triage fast-fail is present and layered:** the service runs KindExists/parent/contract same-org checks as optional UX (exactly the plan's "MAY additionally fast-fail"), the repo's in-tx SELECT EXISTS checks are authoritative (Pitfall 7, no TOCTOU), and the DB FK/CHECK constraints backstop as the third line.
- **Contract test links the guard activity via OQ5:** the dismiss-with-hours scenario creates the linked activity through the activity API with `origin_type: customer_ticket` on an *open* ticket — the exact state the guard must protect. (Triage would move the ticket to planned, from which dismissal is illegal per the matrix.)
- **No DELETE surface (TICK-05):** the mux serves exactly the 9 registered ticket routes; the contract test asserts `DELETE /tickets/{id}` answers 405 (Go 1.22 ServeMux method-not-allowed), proving no delete route exists.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Transition could bypass the dismissal guard**
- **Found during:** Task 2 (adding the Dismiss service method)
- **Issue:** The pinned matrix contains open→dismissed and triage→dismissed. Task 1's `Transition` spec checked only `CanTransition` (plus the resolved-terminal check), so `Transition(open → dismissed)` would succeed for an owner/assignee — bypassing the D-13 hours guard (T-11-07) and never writing dismissed_hours.
- **Fix:** `Transition` now rejects `to == "dismissed"` with ErrInvalidTransition; dismissal is only reachable through the guarded `Dismiss` path. Documented in the service, pinned by a unit test ("transition endpoint cannot reach dismissed") and the contract test's lifecycle.
- **Files modified:** internal/core/services/ticket/ticket.go (+ unit test in ticket_test.go)
- **Verification:** `go test ./internal/core/services/ticket/ -count=1` green
- **Committed in:** 5809a4a (Task 2 commit)

**2. [Rule 1 - Bug] Missing `strPtr` test helper in the pre-existing uncommitted work**
- **Found during:** Task 1 verification (first test run of the inherited work)
- **Issue:** `ticket_test.go` referenced `strPtr` (used in lifecycle/update tests) but the ticket package had no definition — the service test build failed.
- **Fix:** Added the `strPtr` helper to ticket_test.go (same shape as time_entry/expense packages).
- **Files modified:** internal/core/services/ticket/ticket_test.go
- **Verification:** `go test ./internal/core/services/ticket/ -run 'TestTicketCreate|TestTicketLifecycle' -count=1` exits 0
- **Committed in:** 25af406 (Task 1 commit)

**3. [Rule 3 - Blocking] Test fixture needed the ticket wiring to run the contract test**
- **Found during:** Task 3 (TestTicketAPI against the shared fixture)
- **Issue:** `handler_test_helper.go` (the integration fixture) had no ticket service/handler/routes — the contract test drove a mux without /tickets. The plan's file list covered main.go only.
- **Fix:** Wired `ticketsvc.NewService(ticketRepo, activityRepo, contractRepo, orgRepo)` + `NewTicketHandler` + the 9 routes into the fixture, mirroring main.go exactly (same pattern as 11-05 deviation 3).
- **Files modified:** internal/adapters/primary/http/handler_test_helper.go
- **Verification:** TestTicketAPI green against the fixture
- **Committed in:** c3858e5 (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (1 Rule 2 security, 1 Rule 1 bug, 1 Rule 3 blocking)
**Impact on plan:** All three were required for correctness/security (guard bypass prevention) or test completeness (inherited helper gap, fixture wiring). No scope creep — the Transition dismissal block is the only behavior beyond the plan's letter, and it enforces the plan's own must-have (TICK-04 dismissal guard).

## Issues Encountered

- **Multi-membership login lands on the manager's own org:** the contract test's role switches use register→insert-membership→switch-organization; restoring the manager session required login + switch-organization (login picks the first active membership, which is the manager's own org). Resolved with a `restoreManager` helper — no product-code change.
- **Dismissal-with-hours scenario shape:** initially wired via triage (which creates the linked activity), but triage moves the ticket to planned — where dismissal is illegal per the matrix, so the 409 could never fire. Corrected to link the activity via the OQ5 customer_ticket path on an open ticket (see Decisions).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 12 (funding sources) can build directly on:** `LoggedHours(ctx, ticketID) (float64, error)` — the D-13-stable signature the plan pinned for the net-of-compensations computation swap (no signature change needed).
- **Tickets are API-complete (TICK-01..05):** full lifecycle, atomic triage with validated activities, guarded dismissal with the hours snapshot, append-only comments/history — all endpoint-level and unit/integration tested; `make test` 0 failures, `go vet ./...` clean.
- **Frontend surfaces can consume:** GET /tickets (status/kind filters), GET /tickets/{id} (ticket + comments), GET /tickets/{id}/history (ordered audit stream) — envelope `{data: ...}` as everywhere.

## Self-Check: PASSED

- Created files verified on disk: ticket.go, ticket_test.go, ticket_integration_test.go, ticket_handler.go, ticket_handler_test.go, 11-06-SUMMARY.md
- Commits verified in git log: 25af406 (Task 1), 5809a4a (Task 2), c3858e5 (Task 3)
- Verification commands: `go test ./internal/core/services/ticket/ -count=1` ok; `go test ./internal/core/services/ticket/ ./internal/adapters/secondary/postgres/ -run 'TestDismissalGuard|TestTicketTriage|TestTicketAudit' -count=1` ok; `go test ./internal/adapters/primary/http/ -run TestTicket -count=1` ok; `go build ./...` ok; `go vet ./...` exit 0; `make test` 0 failures
- Vault artifacts (.obsidian/workspace.json + the two untracked vault files) excluded from every commit

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
