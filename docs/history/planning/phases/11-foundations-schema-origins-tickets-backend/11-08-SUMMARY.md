---
phase: 11-foundations-schema-origins-tickets-backend
plan: 08
subsystem: api
tags: [tickets, dismissed-note, derived-field, validation, go, postgres]

requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: "11-07 in-tx authoritative dismissal guard — dismissed tickets carry dismissed_hours written by the commit-adjacent Σ"
provides:
  - "Server-rendered dismissal note (IN-02 closed): every read of a dismissed ticket — GET /tickets/{id}, GET /tickets, POST /tickets/{id}/dismiss response — carries dismissed_note = 'dismissed with N h logged' (N = dismissed_hours, trailing zeros trimmed)"
  - "Title input validation: Create/UpdateDetails reject >255-char titles (WR-04) and empty-title updates (IN-01) with ErrInvalidRequest → 400 — no 500 path remains for title input"
affects: [verify-work, phase-verification, TICK-06 frontend surface phases]

actuals:
  tokens: 49    # chars/4 over the realized diff (189 added + 7 deleted across the 6 code files)
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Derived read-model field: DismissedNote computed in scanTicketRow from Status + DismissedHours — never persisted (no column, no migration), renders from the existing dismissed_hours column per OQ3/A4 and D-13"
    - "strconv.FormatFloat(x, 'f', -1, 64) for note formatting — precision -1 trims trailing zeros (5.00 → '5', 7.50 → '7.5')"
    - "Service-boundary validation mirrors the DB column limit (VARCHAR(255)) so oversized input answers 400 from the service, never a 500 from the column"

key-files:
  created: []
  modified:
    - internal/core/domain/ticket/ticket.go
    - internal/adapters/secondary/postgres/ticket_repository.go
    - internal/adapters/secondary/postgres/ticket_repository_test.go
    - internal/adapters/primary/http/ticket_handler_test.go
    - internal/core/services/ticket/ticket.go
    - internal/core/services/ticket/ticket_test.go

key_links:
  - "scanTicketRow → every JSON ticket response (repo Get/List/returned-ticket paths all scan through scanTicketRow)"
  - "Service.Create/UpdateDetails validation → ErrInvalidRequest → handler 400 mapping (already wired)"

key-decisions:
  - "DismissedNote is a derived read-model field, not a column: computed in scanTicketRow only when Status == 'dismissed' && DismissedHours != nil — OQ3/A4, D-13, no migration"
  - "Note number formatted with FormatFloat precision -1 so the raw Σ reads naturally (5.00 → '5', 7.50 → '7.5')"
  - "Title length validation mirrors migration 014's VARCHAR(255) column exactly: >255 rejected with ErrInvalidRequest in Create and UpdateDetails (WR-04, T-11-16) — eliminates the oversized-input 500 path"
  - "Empty-title updates rejected with ErrInvalidRequest (IN-01, T-11-17) — tickets can no longer be renamed to ''; validation runs before the payload map and repo call"

patterns-established:
  - "Read-model derivations live in scanTicketRow (the single funnel) — derived JSON fields never touch ticketColumns or SQL"

requirements-completed: [TICK-04]

coverage:
  - id: D1
    description: "Dismissal note server-rendered (IN-02): every read of a dismissed ticket — GET /tickets/{id}, GET /tickets list, and the POST /tickets/{id}/dismiss response — carries dismissed_note 'dismissed with N h logged' (N = dismissed_hours, trailing zeros trimmed); non-dismissed tickets and dismissed-with-NULL-hours tickets carry no note"
    requirement: TICK-04
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/ticket_repository_test.go#TestTicketRepository_DismissedNote"
        status: pass
      - kind: unit
        ref: "internal/adapters/primary/http/ticket_handler_test.go#TestTicketAPI dismiss scenario (note in dismiss response, follow-up GET, list)"
        status: pass
      - kind: manual_procedural
        ref: "live-API checkpoint (blocking human-verify — user approved): note renders on GET /tickets/{id}, GET /tickets, dismiss response"
        status: pass
    human_judgment: false
  - id: D2
    description: "Title validation (WR-04/IN-01): Create rejects >255-char titles; UpdateDetails rejects empty and >255-char titles — all ErrInvalidRequest → 400, never a 500"
    verification:
      - kind: unit
        ref: "internal/core/services/ticket/ticket_test.go#TestTicketCreate (255 ok / 256 rejected)"
        status: pass
      - kind: unit
        ref: "internal/core/services/ticket/ticket_test.go#TestTicketUpdateDetails (empty and 256 rejected / 255 ok)"
        status: pass
      - kind: manual_procedural
        ref: "live-API checkpoint (blocking human-verify — user approved): 300-char title answered 400, 255-char accepted"
        status: pass
    human_judgment: false

duration: 3h 50m
completed: 2026-08-07
status: complete
---

# Phase 11 Plan 8: Dismissal-Note Server Rendering + Title Validation Summary

**The "dismissed with N h logged" note is now a server-derived read-model field: every read of a dismissed ticket — detail, list, and the dismiss response itself — carries `dismissed_note` computed from `dismissed_hours` (trailing zeros trimmed), closing VERIFICATION.md gap 2's IN-02; WR-04 (>255-char titles) and IN-01 (empty-title updates) ride along as service-boundary validations returning ErrInvalidRequest → 400, eliminating the oversized-input 500 path.**

## Performance

- **Duration:** 3h 50m (commit window 14:21:08Z–18:10:57Z; includes the blocking checkpoint:human-verify wait between task 1 and task 2)
- **Started:** 2026-08-07T14:21:08Z (first commit; execution began slightly earlier)
- **Completed:** 2026-08-07T18:10:57Z
- **Tasks:** 2 auto (1 tracer + 1 TDD) + 1 checkpoint:human-verify — approved
- **Files modified:** 6

## Accomplishments

- **DismissedNote derived field (IN-02 closed, TICK-04 observable):** `Ticket` gains `DismissedNote *string` (`json:"dismissed_note,omitempty"`), populated by `scanTicketRow` — the single funnel every ticket read passes through — only when `Status == 'dismissed' && DismissedHours != nil`. The note is derived at scan time, never persisted: no column, no migration, `ticketColumns` untouched (OQ3/A4, D-13). Number formatting uses `strconv.FormatFloat(..., 'f', -1, 64)` so the raw Σ reads naturally: 5.00 → "5", 7.50 → "7.5".
- **Contract pinned at every read path:** the TestTicketAPI dismiss scenario now asserts `"dismissed_note":"dismissed with 0 h logged"` in the dismiss response body, the follow-up `GET /tickets/{id}`, and a `GET /tickets` list response — the plan's must-have truth "every read of a dismissed ticket" is a passing test, not a doc comment.
- **Title validation (WR-04/IN-01 ride-alongs):** `Create` rejects `len(title) > 255`; `UpdateDetails` rejects `*title == ""` and `len(*title) > 255` — all `ticketdomain.ErrInvalidRequest`, which the handler already maps to 400. TDD: RED tests (256-char + empty cases) → GREEN implementation. 255-char titles still succeed; no 500 path remains for title input (T-11-16/T-11-17).
- **Live-API checkpoint APPROVED by user:** note renders on `GET /tickets/{id}`, `GET /tickets`, and the dismiss response; 300-char title answered 400; 255-char title accepted.

## Task Commits

Each task was committed atomically (TDD: test → feat):

1. **Task 1 (tracer): 'dismissed with N h logged' note rendered on every read — derived field + scan wiring + contract assertion** - `16aba29` (feat)
2. **Task 2 (TDD): Title input validation rides along — 256-char titles rejected (WR-04), empty-title updates rejected (IN-01)** - `22ddc55` (test RED), `309d16d` (feat GREEN)
3. **Checkpoint: confirm the dismissal note renders server-side on live API reads** - approved by user (no commit)

## Files Created/Modified

All six files pre-existed and were modified (none created — verified via `git log --diff-filter=A`):

- `internal/core/domain/ticket/ticket.go` - `Ticket` struct gains `DismissedNote *string` tagged `json:"dismissed_note,omitempty"` with the derived-on-read / never-persisted contract comment
- `internal/adapters/secondary/postgres/ticket_repository.go` - `scanTicketRow` populates the note via `fmt.Sprintf` + `strconv.FormatFloat` (precision -1) when dismissed with non-nil hours; `ticketColumns` unchanged
- `internal/adapters/secondary/postgres/ticket_repository_test.go` - new `TestTicketRepository_DismissedNote`: dismissed with 5h → note "dismissed with 5 h logged"; dismissed with NULL hours → nil; 'planned' → nil
- `internal/adapters/primary/http/ticket_handler_test.go` - TestTicketAPI dismiss scenario asserts the note in the dismiss response, the follow-up `GET /tickets/{id}`, and a `GET /tickets` list response
- `internal/core/services/ticket/ticket.go` - `Create` rejects `len(title) > 255`; `UpdateDetails` rejects `*title == ""` and `len(*title) > 255` — all `ErrInvalidRequest`
- `internal/core/services/ticket/ticket_test.go` - `TestTicketCreate`/`TestTicketUpdateDetails` extended with 255-ok / 256-rejected / empty-rejected cases (via `strings.Repeat`)

## Decisions Made

- **Derived field, not a column:** `DismissedNote` is computed in `scanTicketRow` from the existing `dismissed_hours` column — no migration, no schema change, `ticketColumns` frozen (OQ3/A4, D-13).
- **Note formatting:** `FormatFloat` with precision -1 trims trailing zeros so the raw Σ number (D-13) renders naturally ("5", "7.5" — never "5.00").
- **Validation mirrors the column:** title limits in the service match migration 014's `VARCHAR(255)` exactly, converting what was a DB-level 500 into a contract-level 400 (WR-04, T-11-16).
- **Validation precedes side effects:** rejected titles short-circuit before the payload map and repo call in `UpdateDetails` (IN-01, T-11-17).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Checkpoint Outcome

**Task 3 (checkpoint:human-verify, gate=blocking): APPROVED** — user responded "approved" to the live-API verification (VERIFICATION.md "Human Verification Required" item 2 — the note-rendering decision, now implemented as server-derived). Executor-verified observations: a dismissed ticket's JSON carries `"dismissed_note": "dismissed with 0 h logged"` alongside `"dismissed_hours": 0` on `GET /tickets/{id}`, in the `GET /tickets` list, and in the `POST /tickets/{id}/dismiss` response; a 300-char title answered 400 (not 500); a 255-char title was accepted. The TICK-04 note claim holds at the API boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 11 complete: 8/8 plans.** VERIFICATION.md gap 2 fully closes: TICK-04's dismissal note is server-rendered (IN-02) on top of the concurrency-safe guard from 11-07; WR-04/IN-01 ride-alongs landed within the ticket subsystem only.
- Phase suite green: `go test ./internal/adapters/secondary/postgres/ -run TestTicket -count=1`, `go test ./internal/adapters/primary/http/ -run TestTicket -count=1`, `go test ./internal/core/services/ticket/ -count=1`, `go build ./...` — all pass.
- Frontend/TICK-06 surface phases can render `dismissed_note` directly from the API — no client-side derivation needed.

## Self-Check: PASSED

- All 6 key files exist on disk: ticket.go, ticket_repository.go, ticket_repository_test.go, ticket_handler_test.go, services/ticket/ticket.go, services/ticket/ticket_test.go
- All commits found in git history: `16aba29` (task 1 feat), `22ddc55` (task 2 RED test), `309d16d` (task 2 GREEN feat)
- Final verification suite green: postgres TestTicket (3.9s), http TestTicket (5.6s), ticket service (4.0s), `go build ./...` exit 0

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
