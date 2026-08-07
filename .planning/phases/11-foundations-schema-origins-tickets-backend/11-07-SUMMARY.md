---
phase: 11-foundations-schema-origins-tickets-backend
plan: 07
subsystem: database
tags: [postgres, concurrency, transactions, race-tests, tickets, toctou]

requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: "11-06 ticket lifecycle state machine (D-14 matrix), dismissal guard, audit rows"
provides:
  - "In-tx authoritative re-validation of every ticket mutator (Dismiss/UpdateState/Triage) under FOR UPDATE row locks — CR-01 TOCTOU closure"
  - "loggedHoursTx + hasNonTerminalActivitiesTx private helpers (pool-level signatures Phase-12-stable)"
  - "Concurrency race test battery (3 goroutine/deterministic races + 3 sequential pins) green under -race"
affects: [11-08, verify-work, phase-verification]

actuals:
  tokens: 8477
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Pitfall 7 (ADR-BE-016): SELECT ... FOR UPDATE on the tickets row, then re-validate the domain matrix inside the mutator tx — locked-row status is commit-order truth"
    - "Check-and-act in one tx: linked-activity FOR UPDATE + in-tx Σ (loggedHoursTx) so the dismissal guard serializes with entry submits"
    - "SQL-as-private-helper via tx (loggedHoursTx / hasNonTerminalActivitiesTx) — pool-level check APIs unchanged, authoritative copies executed against the open tx"

key-files:
  created:
    - internal/adapters/secondary/postgres/ticket_repository_test.go (new race battery + pins)
  modified:
    - internal/adapters/secondary/postgres/ticket_repository.go
    - internal/core/services/ticket/ticket.go

key-decisions:
  - "Repo layer is authoritative: every matrix/guard decision re-validated inside the mutator tx under the FOR UPDATE lock; service pool-level checks are fast-fail UX only"
  - "Dismiss writes its own in-tx Σ (loggedHoursTx) as dismissed_hours — the service's hours param is superseded by the commit-adjacent value"
  - "All mutator UPDATEs carry the SQL status-precondition backstop (AND status = currentStatus)"

patterns-established:
  - "In-tx re-validation layering: lock → re-check → write → audit → commit"
  - "Race tests follow the house goroutine pattern from refresh_token_rotate_test.go (start channel, results channel, exactly-one-winner)"

requirements-completed: [TICK-02, TICK-04]

coverage:
  - id: D1
    description: "Dismiss hardened — in-tx FOR UPDATE ticket lock, CanTransition re-check (planned→dismissed illegal), linked-activity FOR UPDATE, loggedHoursTx Σ re-check (ErrDismissalBlocked), status-precondition UPDATE"
    requirement: TICK-04
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestDismissalGuard_RaceWithPendingSubmit"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestDismissalRace_VsTriage"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestDismiss_RejectsPlanned"
        status: pass
    human_judgment: false
  - id: D2
    description: "UpdateState + Triage hardened — in-tx matrix re-validation (dismissed resurrection closed), resolved-block re-checked via hasNonTerminalActivitiesTx inside the tx, status precondition on both UPDATEs"
    requirement: TICK-02
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestTransitionRace_VsDismiss"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestTriage_RejectsDismissed"
        status: pass
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestUpdateState_ResolvedBlocked"
        status: pass
    human_judgment: false
  - id: D3
    description: "CR-01 invariants hold under concurrent live-API load (VERIFICATION.md human item 1): exactly-one-winner on racing mutators; dismissal guard 409 on committed submissions; never dismissed_hours=0 with committed logged hours"
    verification: []
    human_judgment: true
    rationale: "Belt-and-suspenders live-API run against a running server with concurrent HTTP requests — requires human observation of real API behavior; approved by user ('approved')"

duration: 15min
completed: 2026-08-07
status: complete
---

# Phase 11 Plan 7: CR-01 Concurrency Hardening of Ticket State Machine + Dismissal Guard Summary

**In-tx FOR UPDATE re-validation of Dismiss/UpdateState/Triage closes the CR-01 TOCTOU races: matrix decisions, the resolved-block, and the dismissal hours Σ are all re-checked inside the mutator transaction under the ticket row lock, with a goroutine race battery (green under `-race`) and a human-approved live-API checkpoint.**

## Performance

- **Duration:** 15 min (implementation commits 15:33:25Z–15:39:53Z; plus checkpoint + final verification)
- **Started:** 2026-08-07T15:33:25Z
- **Completed:** 2026-08-07
- **Tasks:** 3 (2 auto + 1 checkpoint:human-verify — approved)
- **Files modified:** 3

## Accomplishments

- **Dismiss** hardened: `SELECT status ... FOR UPDATE` → `CanTransition(currentStatus, 'dismissed')` (illegal planned→dismissed closed) → linked-activity `FOR UPDATE` (serializes against in-flight time-entry submits) → in-tx `loggedHoursTx` Σ re-check (`ErrDismissalBlocked` if > 0, T-11-07) → status-precondition UPDATE. The Σ written to `dismissed_hours` is the commit-adjacent value, never the service's pool-level fast-fail value.
- **UpdateState** hardened: in-tx lock → matrix re-check → resolved-block re-checked via `hasNonTerminalActivitiesTx` inside the tx (`ErrActivityNotTerminal`) → status-precondition UPDATE.
- **Triage** hardened: `CanTransition(currentStatus, 'planned')` under its existing lock (dismissed-resurrection closed) + status precondition on its UPDATE.
- **Race test battery** (RED before fix, GREEN after): `TestDismissalGuard_RaceWithPendingSubmit` (deterministic 2-tx), `TestDismissalRace_VsTriage` (goroutine), `TestTransitionRace_VsDismiss` (goroutine), plus pins `TestDismiss_RejectsPlanned`, `TestTriage_RejectsDismissed`, `TestUpdateState_ResolvedBlocked`.
- **Live-API checkpoint approved**: triage vs dismiss race → exactly one success; dismiss vs time-entry submit → guard 409 on committed submissions; no dismissal committed with `dismissed_hours=0` while committed logged hours existed at check time.

## Task Commits

Each task was committed atomically (TDD: test → feat):

1. **Task 1: Guarded dismissal re-validated inside the tx (RED race tests + Dismiss hardening)** - `7cfabfb` (test), `ca02d0d` (feat)
2. **Task 2: UpdateState + Triage in-tx matrix re-validation** - `f034075` (test), `446d9e7` (feat)
3. **Checkpoint: confirm CR-01 invariants under concurrent live-API load** - approved by user (no commit)

**Plan metadata:** `docs(11-07): complete plan summary` + `docs(11-07): update tracking`

## Files Created/Modified

- `internal/adapters/secondary/postgres/ticket_repository.go` - Dismiss/UpdateState/Triage hardened with in-tx FOR UPDATE + re-validation; new private helpers `loggedHoursTx` and `hasNonTerminalActivitiesTx` executing the Σ / recursive-CTE SQL against the open tx; status-precondition backstop on all mutator UPDATEs
- `internal/adapters/secondary/postgres/ticket_repository_test.go` - 6 new tests: 3 goroutine/deterministic race tests + 3 sequential pins (433 new lines)
- `internal/core/services/ticket/ticket.go` - doc comments only: Transition/Dismiss/Triage re-worded to fast-fail-vs-authoritative layering (Pitfall 7 wording)

## Decisions Made

- **Repo layer is authoritative**: every matrix/guard decision re-validated inside the mutator tx under the FOR UPDATE lock; service pool-level checks are fast-fail UX only (Pitfall 7, ADR-BE-016).
- **Dismiss writes its own Σ**: `loggedHoursTx` result (not the service's `hours` param) is written to `dismissed_hours` — the authoritative T-11-07 gate per D-13.
- **Status-precondition backstop**: every mutator UPDATE carries `AND status = currentStatus` as a final SQL-level guarantee.
- **Phase-12-stable signatures preserved**: pool-level `LoggedHours`/`HasNonTerminalActivities` unchanged; tx variants are private helpers.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Checkpoint Outcome

**Task 3 (checkpoint:human-verify, gate=blocking): APPROVED** — user responded "approved" to the live-API concurrency verification (VERIFICATION.md human item 1). Observed results: exactly-one-winner on the triage vs dismiss race (loser answered 400 invalid ticket status transition); dismissal guard returned 409 on submissions committed before the check and proceeded legally otherwise; no dismissal ever committed with `dismissed_hours=0` while committed logged hours existed at check time. The automated race battery had already proven the same invariants against testcontainers; this was the belt-and-suspenders live run.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VERIFICATION.md gap 1 (TICK-02 lifecycle integrity under concurrency) and gap 2 (TICK-04 dismissal-guard Σ authoritative in-tx) are closed; 11-VERIFICATION.md gap-closure table for these items can be updated by phase verification.
- Ready for 11-08 (dismissal note rendering on read, TICK-04/IN-02 + title validation ride-alongs WR-04/IN-01), which depends on the hardened guard semantics (dismissed tickets carry `dismissed_hours` written by the in-tx Σ).

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
