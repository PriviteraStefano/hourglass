---
phase: 11-foundations-schema-origins-tickets-backend
verified: 2026-08-07T15:30:00Z
status: gaps_found
score: 9/11 must-haves verified
overrides_applied: 0
gaps:
  - truth: "TICK-02: The lifecycle works — open → triage → planned → in_progress → resolved → closed, plus reopen resolved → in_progress; invalid edges return errors; resolved blocks while any linked activity has non-terminal entries"
    status: failed
    reason: "CR-01 (code review): every matrix/guard decision (CanTransition, HasNonTerminalActivities) is evaluated at pool level OUTSIDE the mutator transaction. UpdateState writes `UPDATE tickets SET status=$1 WHERE id=$3 AND org_id=$4` with no status precondition and no row lock; Triage takes `SELECT ... FOR UPDATE` but never re-checks the captured currentStatus against the matrix. Two concurrent requests can violate the pinned matrix: (1) dismissed-ticket resurrection — Triage (passes CanTransition(triage,planned) at pool level) commits after Dismiss, flipping a terminal 'dismissed' ticket to 'planned' and inserting activities; (2) illegal planned → dismissed — Dismiss's lock-free UPDATE lands on a 'planned' ticket. The resolved-block has the same shape (HasNonTerminalActivities checked before UpdateState, no in-tx re-check)."
    artifacts:
      - path: "internal/core/services/ticket/ticket.go"
        issue: "CanTransition / HasNonTerminalActivities / LoggedHours all read via repo.Get at pool level before the mutator call (lines 191-233 Transition, 259-287 Dismiss, 325-379 Triage)"
      - path: "internal/adapters/secondary/postgres/ticket_repository.go"
        issue: "UpdateState (220-247) and Dismiss (253-281) UPDATE without FOR UPDATE and without a status precondition; Triage (304-314) locks FOR UPDATE but the scanned currentStatus is never validated against the matrix (race 1)"
    missing:
      - "Re-run the state-machine validation inside the tx under a `FOR UPDATE` ticket row lock (mirroring the triage pattern) for UpdateState/Dismiss, and re-check currentStatus → planned in Triage; optionally add a SQL status precondition (`AND status=$N`) as backstop"
  - truth: "TICK-04: dismissal is blocked while any linked activity carries logged hours (submitted+approved, not deleted); the dismissed ticket carries the 'dismissed with N h logged' note"
    status: failed
    reason: "CR-01 race 3: LoggedHours is computed at pool level (ticket_repository.go:559-570, no lock shared with the time-entry submit path), then repo.Dismiss commits the dismissal in a separate tx without re-checking the Σ inside the tx. An entry submitted between the Σ read and the dismiss commit bypasses the guard — the ticket commits with dismissed_hours=0 while logged hours > 0 exist. This is the phase's headline security control (T-11-07) and it is check-then-act. Additionally (IN-02), the 'dismissed with N h logged' note is never rendered server-side — only the raw dismissed_hours field is exposed in the JSON; the note exists only in doc comments."
    artifacts:
      - path: "internal/core/services/ticket/ticket.go"
        issue: "Dismiss (270-276) reads hours via repo.LoggedHours at pool level, then calls repo.Dismiss — no in-tx re-validation"
      - path: "internal/adapters/secondary/postgres/ticket_repository.go"
        issue: "LoggedHours (559-570) is a pool-level read; Dismiss (253-281) writes with no FOR UPDATE and no in-tx Σ re-check"
    missing:
      - "Re-run the Σ inside the dismiss tx (SELECT COALESCE(SUM(hours),0) ... via tx) before the UPDATE, under a ticket row lock; reject with ErrDismissalBlocked if > 0"
      - "Render the 'dismissed with N h logged' note on read (derived field) or drop the claim from the acceptance text (IN-02)"
---

# Phase 11: Foundations — Schema + Origins + Tickets Backend Verification Report

**Phase Goal:** The three-plane ontology takes its first shape server-side: activities carry origin (type + reference set, FND-01/02/04), contracts carry sold_hours (FND-03), and the ticket entity exists with lifecycle + triage + dismissal guard + immutable event stream (TICK-01..05). ADR-P-003 revision and ADR-P-013 drafted and recorded.
**Verified:** 2026-08-07T15:30:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | FND-01: Activity created via API carries an origin (type + reference set per D-D); refs set once at creation and immutable | ✓ VERIFIED | `OriginType` + 5 ref fields on Activity/requests (domain/activity/activity.go:76-81); `validateOrigin` role/same-org gates (services/activity/activity.go:130-194); `ErrOriginImmutable` guard on Update (210-230); origin cols in repo INSERT (activity_repository.go); DB CHECK `activities_origin_refs_check` backstop (migration 015); activity tests pass |
| 2 | FND-02: Employee can propose an activity; approval routes via BE-014 routing; lifecycle events land in activity state/audit, never origin | ✓ VERIFIED | Proposals forced `is_active=false` (services/activity/activity.go:168-169); `ApproveProposal` resolves approver via `routing.ResolveManagerStage` (300-317), flips is_active, writes synchronous `proposal_approved` audit row (319-335); routing extraction verified (11-03): shared package consumed by time_entry service |
| 3 | FND-03: Contracts expose sold_hours read/write; D-08 semantics; legacy NULL contract_type valid | ✓ VERIFIED | ContractType/SoldHours/SoldPeriod on domain + requests; `validateSoldConfig` (services/contract/contract.go:104-127); repo baseQuery/Create/dynamic SET Update; handler read/write; `contracts_sold_check` backstop; contract tests pass |
| 4 | FND-04: Origin refs stored on activities; empty-ref fallback to first direction record documented, not built (Phase 13) | ✓ VERIFIED | Refs stored directly and read back in responses; fallback explicitly documented as Phase-13 additive in ADR-P-013 §FND-04 ("an additive read-path fallback, not a D-D revision") |
| 5 | TICK-01: Any employee can create a ticket with a closed-set kind; customer role rejected | ✓ VERIFIED | `closedKindSet` 4 kinds (services/ticket/ticket.go:54-59); customer gate (79-81); kind CHECK in migration 014; `TestTicketCreate` + handler tests pass |
| 6 | TICK-02: Lifecycle works (7 statuses + reopen + guarded dismissal); invalid edges return errors; resolved blocks on non-terminal activities | ✗ FAILED | Matrix pinned in domain (ticket.go:83-102) and sequential behavior tested — but CR-01: matrix + resolved-block checks run OUTSIDE the mutator tx; terminal states violable under concurrency (dismissed resurrection via Triage; illegal planned→dismissed). See gaps. |
| 7 | TICK-03: Triage atomically converts ticket into 1..N customer_ticket-origin activities (all-or-nothing) | ✓ VERIFIED | Single tx: `FOR UPDATE` lock + in-tx plan validation (kind/parent/contract via tx) + kind/status flip + activity inserts + both audit rows (ticket_repository.go:297-440); `TestTicketTriage`/`TestDismissalGuard`/`TestTicketAudit` integration tests pass |
| 8 | TICK-04: Dismissal blocked while linked activities carry logged hours; dismissed ticket carries the hours note | ✗ FAILED | Sequential guard works (service 270-276, `TestTicketDismissalGuard`) — but the Σ is a pool-level read and Dismiss commits with no in-tx re-check (CR-01 race 3): a late-submitted entry can bypass the guard. Note (IN-02) never rendered server-side. See gaps. |
| 9 | TICK-05: Ticket events are append-only via audit trail; no update/delete endpoints | ✓ VERIFIED | Every ticket mutator writes its audit row in the SAME tx (`insertTicketAudit` inside tx in Create/UpdateDetails/UpdateState/Dismiss/Triage/AddComment); 9 routes registered in main.go:220-228 with deliberately NO DELETE; `TestTicketRepository_NoAuditMutation` asserts no audit mutation path; history read ordered stream |
| 10 | Migrations 014-017 apply up/down/up (ADR-BE-004 cycle); legacy NULL rows pass 3VL checks; teardown list extended | ✓ VERIFIED | All 8 files exist with up/down pairs; cycle tests `TestMigration014..017_UpDownUpCycle` (ontology_extension_migrations_test.go:26,91,176,241) + pre-existing 011/012 green; `origin_type IS NULL OR ...` and `contract_type IS NULL OR ...` 3VL guards; teardown includes tickets/ticket_comments/audit_logs (exported_test_helpers.go:101-103) |
| 11 | ADR-P-003 revision + ADR-P-013 (+ ADR-BE-016) recorded in vault; indexes updated | ✓ VERIFIED | ADR-P-003 marked `Revised: 2026-08-03` with v0.2 lifecycle + guarded dismissal + hard boundary list; ADR-P-013 records origin decisions incl. FND-04 fallback; ADR-BE-016 records schema encoding; both `_index.md` files list the ADRs |

**Score:** 9/11 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| migrations/014_ticket_schema.{up,down}.sql | tickets + ticket_comments, kind/status CHECK vocabulary, dismissed_hours, indexes | ✓ VERIFIED | Substantive (51 lines up / 4 down); cycle test green |
| migrations/015_activity_origins.{up,down}.sql | origin_type discriminator + 5 nullable refs + refs-to-type CHECK + ticket_id FK + index | ✓ VERIFIED | `activities_origin_refs_check` 3VL guard; cycle test green |
| migrations/016_contract_sold_hours.{up,down}.sql | contract_type/sold_hours/sold_period + type-consistency CHECK | ✓ VERIFIED | `contracts_sold_check`; cycle test green |
| migrations/017_audit_logs.{up,down}.sql | general audit_logs + (entity_type, entity_id, created_at) index | ✓ VERIFIED | Cycle test green |
| internal/core/services/routing/routing.go | managerResolution + ResolveManagerStage + ResolveUnitManager | ✓ VERIFIED | Exports used by time_entry + activity services; semantics preserved (RoleGated/SkipToFinance/ErrActivityNotLoggable) |
| internal/core/domain/contract/contract.go + service + repo + handler | sold_hours read/write | ✓ VERIFIED | D-08 semantics validated; tests green |
| internal/core/domain/activity/activity.go + service + repo | OriginType + 5 refs; gates; immutability; ApproveProposal | ✓ VERIFIED | ErrOriginImmutable enforced; tests green |
| internal/core/domain/ticket/ticket.go | entity, kind/status vocabulary, locked transition matrix, sentinels | ✓ VERIFIED | Matrix pins reopen + open|triage→dismissed; terminal states closed/dismissed |
| internal/core/services/ticket/ticket.go | Create/List/Get/UpdateDetails/Transition/Dismiss/Triage/AddComment/ListHistory | ⚠️ ORPHANED-gap | All functions present and wired; matrix/guard checks are check-then-act (CR-01) |
| internal/adapters/primary/http/ticket_handler.go | HTTP adapter, 9 routes, no DELETE | ✓ VERIFIED | Registered in main.go:220-228 |
| internal/adapters/secondary/postgres/ticket_repository.go | tx-backed mutators + LoggedHours + HasNonTerminalActivities + ListHistory | ⚠️ PARTIAL | In-tx audit discipline real; but UpdateState/Dismiss lack FOR UPDATE + status precondition; Triage never uses currentStatus |
| internal/core/ports/ticket_repository.go | full TicketRepository interface | ✓ VERIFIED | Includes LoggedHours/HasNonTerminalActivities |
| hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md | revised v0.2 lifecycle | ✓ VERIFIED | Revised 2026-08-03, delta section, hard boundary list preserved |
| hourglass-vault/decisions/project/ADR-P-013 — Origins.md | origin axis decision record | ✓ VERIFIED | D-01..D-04, D-12, FND-04 fallback |
| hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md | BE encoding ADR | ✓ VERIFIED | Migrations 014-017, 3VL rule, in-tx audit writes |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| services/ticket/ticket.go | domain/ticket/ticket.go | CanTransition matrix gates every Transition/Dismiss/Triage call | WIRED (sequential) | Present at service level — but executed at pool level, not re-validated in tx (CR-01) |
| postgres/ticket_repository.go | time_entries | LoggedHours Σ (submitted+approved, is_deleted=false, via ticket_id + origin_type) | WIRED (sequential) | COALESCE(SUM(hours),0) query correct — but pool-level read, no tx lock shared with entry submit path (CR-01 race 3) |
| services/ticket/ticket.go | cmd/server/main.go | 9 routes incl. triage/dismiss/transition/comments/history; no DELETE | WIRED | main.go:220-228; comment documents deliberate absence of delete paths |
| services/activity/activity.go | services/routing | ApproveProposal resolves approvers via routing.ResolveManagerStage | WIRED | activity.go:300 |
| services/activity/activity.go | postgres/ticket_repository.go | customer_ticket origin validation calls ticketRepo.Get (same-org + open|triage check) | WIRED | activity.go:179-188 |
| migrations/014 | migrations/015 | activities.ticket_id FK resolves against tickets | WIRED | FK `ticket_id UUID REFERENCES tickets(id)`; ordering fixed by numbering 014 before 015 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| ticket ListHistory | audit rows | real INSERTs in ticket mutator txs (audit_logs) | Yes — in-tx writes, never fire-and-forget | ✓ FLOWING |
| ticket LoggedHours | Σ hours | real SQL over time_entries | Yes — real query; but pool-level read (race window) | ⚠️ FLOWING-with-race |
| activity origin refs | origin_type/assigned_by/... | POST /activities body → INSERT → SELECT | Yes — stored and read back; DB CHECK backstop | ✓ FLOWING |
| contract sold_hours | sold_hours/sold_period | POST/PUT /contracts → validateSoldConfig → INSERT/UPDATE | Yes — real validation + persistence | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full backend build | `go build ./...` | exit 0 | ✓ PASS |
| Ticket service suite (lifecycle, dismissal guard, triage pre-validation, history) | `go test ./internal/core/services/ticket/...` | ok (3.9s) | ✓ PASS |
| Postgres adapter suite (incl. migration cycles 014-017, NoAuditMutation, triage audit rows) | `go test ./internal/adapters/secondary/postgres/...` | ok (22.7s) | ✓ PASS |
| Activity / contract / routing suites | `go test ./internal/core/services/activity/... ./internal/core/services/contract/... ./internal/core/services/routing/...` | ok | ✓ PASS |
| HTTP adapter suite (incl. ticket API contract test) | `go test ./internal/adapters/primary/http/...` | ok (34.8s) | ✓ PASS |
| Full workspace suite | `go test ./...` | exit 0 (no failures) | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — no probe scripts exist (`scripts/*/tests/probe-*.sh` not found) and no probes declared in any 11-0x-PLAN.md.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| FND-01 | 11-01, 11-02, 11-05 | Activity origin type + reference set, set once, immutable | ✓ SATISFIED | Truth #1; domain/service/repo + CHECK backstop |
| FND-02 | 11-02, 11-03, 11-05 | Employee proposal with routing-based approval | ✓ SATISFIED | Truth #2; ApproveProposal via routing.ResolveManagerStage + audit row |
| FND-03 | 11-01, 11-04 | Contract sold_hours read/write for V5 mining | ✓ SATISFIED | Truth #3; D-08 validation + 016 CHECK |
| FND-04 | 11-01, 11-02, 11-05 | Origin refs stored on activities; Phase-13 fallback additive | ✓ SATISFIED | Truth #4; stored; fallback documented in ADR-P-013 |
| TICK-01 | 11-01, 11-02, 11-05, 11-06 | Internal ticket with closed-set kind | ✓ SATISFIED | Truth #5; service + DB CHECK + tests |
| TICK-02 | 11-01, 11-02, 11-06 | Lifecycle + reopen + resolved-blocks + invalid edges rejected | ✗ BLOCKED | Truth #6; sequential matrix works, concurrency-violable (CR-01) |
| TICK-03 | 11-02, 11-06 | Triage converts ticket to 1..N activities; ticket → activity → entries chain only | ✓ SATISFIED | Truth #7; atomic in-tx triage |
| TICK-04 | 11-01, 11-02, 11-06 | Dismissal guard + dismissed note | ✗ BLOCKED | Truth #8; Σ guard check-then-act (CR-01); note not rendered (IN-02) |
| TICK-05 | 11-01, 11-02, 11-05, 11-06 | Immutable event stream via BE-012 audit trail; no update/delete | ✓ SATISFIED | Truth #9; in-tx audit rows; no DELETE routes; NoAuditMutation test |

Orphaned requirements check: all 9 IDs (FND-01..04, TICK-01..05) are claimed by plans and verified above — none orphaned.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| postgres/ticket_repository.go | 220-247, 253-281 | Check-then-act across tx boundary (no FOR UPDATE, no status precondition on UPDATE) | 🛑 BLOCKER (CR-01) | Terminal-state invariant + dismissal guard violable under concurrency |
| postgres/ticket_repository.go | 304-314 | `FOR UPDATE` lock taken but `currentStatus` never validated against matrix | 🛑 BLOCKER (CR-01) | Dismissed-ticket resurrection to 'planned' |
| services/activity/activity.go | 319-335 | Audit row in separate tx from is_active flip | ⚠️ WARNING (WR-01) | Proposal can be approved without its proposal_approved event |
| primary/http/contract.go | 180-189 | `ErrInvalidSoldConfig` not mapped in Update switch → 500 | ⚠️ WARNING (WR-02) | Client validation error surfaces as 500 |
| services/contract/contract.go | 104-127 | sold_period vocab check skipped when contract_type nil | ⚠️ WARNING (WR-02) | `{sold_period:"bogus"}` persisted when type absent |
| services/activity/activity.go | 130-194 | Forbidden origin refs not rejected in validateOrigin | ⚠️ WARNING (WR-03) | Mixed-ref requests hit DB CHECK → 500 |
| primary/http/ticket_handler.go | 96-192 | No string-length validation on title/description | ⚠️ WARNING (WR-04) | >255-char title → 500 |
| postgres/activity_repository.go | 108-119 | `Get` WHERE clause lacks org predicate (orgID only for is_adopted subquery) | ⚠️ WARNING (WR-05) | Cross-org read in authorization chain (no current exploit) |
| services/ticket/ticket.go | 254, 27 | "dismissed with N h logged" note only in comments, never rendered | ℹ️ INFO (IN-02) | Note missing from read model |
| services/ticket/ticket.go | 150-183 | UpdateDetails accepts empty title | ℹ️ INFO (IN-01) | Ticket can be renamed to "" |
| postgres/ticket_repository.go | 514-520 | `ORDER BY created_at` without id tiebreaker | ℹ️ INFO (IN-03) | Same-ms stream order unspecified |

No TBD/FIXME/XXX markers found in phase-modified files.

### Human Verification Required

1. **CR-01 concurrency race confirmation (post-fix)** — After the TOCTOU fix lands, fire concurrent dismiss + triage (and dismiss + entry-submit) against a real DB and confirm the matrix/dismissal-guard invariants hold.
   **Test:** Two goroutines: `POST /tickets/{id}/triage` and `POST /tickets/{id}/dismiss` on the same `triage`-status ticket, plus dismiss racing an entry submit on a linked activity.
   **Expected:** Exactly one operation succeeds; a dismissed ticket can never reach `planned`; dismissal never commits with `dismissed_hours=0` while submitted/approved hours exist.
   **Why human:** Requires concurrent load against a live Postgres; the sequential test suite cannot expose the race.
2. **"dismissed with N h logged" note rendering** — Decide whether the note is rendered client-side (Phase 18 tickets surface) or should be server-derived.
   **Test:** Dismiss a ticket and read `GET /tickets/{id}`.
   **Expected:** The dismissal note/hours are visible to the user.
   **Why human:** Frontend rendering is out of scope for this backend phase; the data (`dismissed_hours`) is exposed.

### Gaps Summary

Two must-have truths fail, both from the same root cause (code-review CR-01):

1. **TICK-02 (lifecycle integrity under concurrency)** — the state machine is enforced check-then-act: `CanTransition` and `HasNonTerminalActivities` are read at pool level before the mutator tx; `UpdateState`/`Dismiss` write without a row lock or status precondition, and Triage's `FOR UPDATE` lock never validates the captured status. Realistic concurrent requests can resurrect a dismissed (terminal) ticket to `planned` and land an illegal `planned → dismissed`.
2. **TICK-04 (dismissal guard)** — the `LoggedHours` Σ is a pool-level read; the dismiss commit neither locks the ticket nor re-checks the Σ inside the tx, so a late-submitted entry bypasses the guard (dismissal commits with `dismissed_hours=0` while logged hours exist). The T-11-07 security control is check-then-act. Secondary: the "dismissed with N h logged" note is never rendered (IN-02).

Both are fixable in the repository layer by mirroring the existing triage pattern (in-tx `SELECT ... FOR UPDATE`, re-validate matrix/Σ inside the tx, optionally a SQL status precondition). The warnings (WR-01..05) are quality issues that should ride along but do not block the must-haves they touch (FND-02/FND-03 functional behavior is present and tested).

**Deferred items:** None — no later phase (12-26) addresses the TOCTOU fix or the guard hardening.

---

_Verified: 2026-08-07T15:30:00Z_
_Verifier: the agent (gsd-verifier)_
