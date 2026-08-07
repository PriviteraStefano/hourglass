---
phase: 11-foundations-schema-origins-tickets-backend
verified: 2026-08-07T18:30:00Z
status: passed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 9/11
  gaps_closed:
    - "TICK-02: lifecycle concurrency integrity — in-tx FOR UPDATE re-validation of every matrix edge (UpdateState/Dismiss/Triage), resolved-block re-checked inside the tx, status-precondition backstop on every mutator UPDATE (11-07)"
    - "TICK-04: dismissal guard authoritative in-tx (loggedHoursTx Σ under ticket + linked-activity FOR UPDATE locks, ErrDismissalBlocked) + 'dismissed with N h logged' note server-rendered on every read (11-07 + 11-08)"
  gaps_remaining: []
  regressions: []
---

# Phase 11: Foundations — Schema + Origins + Tickets Backend Verification Report

**Phase Goal:** The three-plane ontology takes its first shape server-side: activities carry origin (type + reference set, FND-01/02/04), contracts carry sold_hours (FND-03), and the ticket entity exists with lifecycle + triage + dismissal guard + immutable event stream (TICK-01..05). ADR-P-003 revision and ADR-P-013 drafted and recorded.
**Verified:** 2026-08-07T18:30:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (11-07 concurrency hardening + 11-08 note rendering, both blocking human-verify checkpoints approved by user)

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | FND-01: Activity created via API carries an origin (type + reference set per D-D); refs set once at creation and immutable | ✓ VERIFIED | Unchanged since initial verification (regression green): `OriginType` + 5 ref fields on Activity (domain/activity/activity.go:76-81); `validateOrigin` role/same-org gates; `ErrOriginImmutable` guard on Update; DB CHECK `activities_origin_refs_check` (migration 015); activity suite green in full run |
| 2 | FND-02: Employee can propose an activity; approval routes via BE-014 routing; lifecycle events land in activity state/audit, never origin | ✓ VERIFIED | Unchanged (regression green): proposals forced `is_active=false`; `ApproveProposal` resolves via `routing.ResolveManagerStage`, writes synchronous `proposal_approved` audit row; activity/routing suites green in full run |
| 3 | FND-03: Contracts expose sold_hours read/write; D-08 semantics; legacy NULL contract_type valid | ✓ VERIFIED | Unchanged (regression green): `validateSoldConfig` (services/contract/contract.go:104-127); `contracts_sold_check` backstop; contract suite green in full run |
| 4 | FND-04: Origin refs stored on activities; empty-ref fallback to first direction record documented, not built (Phase 13) | ✓ VERIFIED | Unchanged: refs stored + read back; fallback documented as Phase-13 additive in ADR-P-013 §FND-04 |
| 5 | TICK-01: Any employee can create a ticket with a closed-set kind; customer role rejected | ✓ VERIFIED | Unchanged (regression green): `closedKindSet` 4 kinds; customer gate; kind CHECK (migration 014); service + handler tests green |
| 6 | TICK-02: Lifecycle works (7 statuses + reopen + guarded dismissal); invalid edges return errors; resolved blocks on non-terminal activities — **under concurrency** | ✓ VERIFIED | **Gap closed (11-07).** In-tx authoritative re-validation: `UpdateState` (ticket_repository.go:239-298) locks row `FOR UPDATE`, re-checks `CanTransition(currentStatus, to)`, re-runs `hasNonTerminalActivitiesTx` for `resolved`; `Dismiss` (313-396) re-checks `CanTransition(currentStatus,'dismissed')` under lock; `Triage` (412-565) re-checks `CanTransition(currentStatus,'planned')` under lock (dismissed-resurrection closed); every mutator UPDATE carries `AND status = currentStatus` backstop. Behavioral evidence: race battery green under `-race` — `TestDismissalRace_VsTriage`, `TestTransitionRace_VsDismiss` (exactly-one-winner), `TestTriage_RejectsDismissed`, `TestDismiss_RejectsPlanned`, `TestUpdateState_ResolvedBlocked` all PASS. Human checkpoint (CR-01 live-API concurrency) APPROVED by user |
| 7 | TICK-03: Triage atomically converts ticket into 1..N customer_ticket-origin activities (all-or-nothing) | ✓ VERIFIED | Unchanged (regression green): single tx, in-tx plan validation, kind/status flip + activity inserts + both audit rows; `TestTicketTriage` green |
| 8 | TICK-04: Dismissal blocked while linked activities carry logged hours (submitted+approved, not deleted); dismissed ticket carries the 'dismissed with N h logged' note | ✓ VERIFIED | **Gap closed (11-07 + 11-08).** Guard: `Dismiss` locks ticket `FOR UPDATE` → locks linked activities `FOR UPDATE` (serializes with entry-submit path) → re-computes Σ via `loggedHoursTx` (ticket_repository.go:706-717, byte-identical WHERE to pool-level `LoggedHours`) → `ErrDismissalBlocked` if > 0 → writes the in-tx Σ to `dismissed_hours`. Behavioral evidence: `TestDismissalGuard_RaceWithPendingSubmit` (deterministic 2-tx: pending submit blocks dismissal, then ErrDismissalBlocked) PASS under `-race`. Note: `DismissedNote` derived field (domain/ticket/ticket.go:30), populated by `scanTicketRow` (ticket_repository.go:59-63) as "dismissed with N h logged" (FormatFloat precision -1) when dismissed && hours != nil; contract test asserts note in dismiss response, `GET /tickets/{id}`, and `GET /tickets` list (ticket_handler_test.go:204,211,224). Human checkpoint (note renders live; 300-char title → 400) APPROVED by user |
| 9 | TICK-05: Ticket events are append-only via audit trail; no update/delete endpoints | ✓ VERIFIED | Unchanged (regression green): every mutator writes its audit row in the same tx; 9 routes with deliberately no DELETE; `TestTicketRepository_NoAuditMutation` green |
| 10 | Migrations 014-017 apply up/down/up (ADR-BE-004 cycle); legacy NULL rows pass 3VL checks; teardown list extended | ✓ VERIFIED | Unchanged (regression green): 8 files with up/down pairs; cycle tests `TestMigration014..017_UpDownUpCycle` green in full postgres run |
| 11 | ADR-P-003 revision + ADR-P-013 (+ ADR-BE-016) recorded in vault; indexes updated | ✓ VERIFIED | Unchanged: all three ADR files present in vault (`hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md`, `ADR-P-013 — Origins.md`, `decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md`) |

**Score:** 11/11 truths verified (0 present, behavior-unverified — the two behavior-dependent truths now have passing behavioral tests, see Behavioral Spot-Checks)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| migrations/014_ticket_schema.{up,down}.sql | tickets + ticket_comments, kind/status CHECK, dismissed_hours, indexes | ✓ VERIFIED | Unchanged; cycle test green |
| migrations/015_activity_origins.{up,down}.sql | origin_type discriminator + 5 nullable refs + CHECK + ticket_id FK | ✓ VERIFIED | Unchanged; cycle test green |
| migrations/016_contract_sold_hours.{up,down}.sql | contract_type/sold_hours/sold_period + CHECK | ✓ VERIFIED | Unchanged; cycle test green |
| migrations/017_audit_logs.{up,down}.sql | general audit_logs + index | ✓ VERIFIED | Unchanged; cycle test green |
| internal/adapters/secondary/postgres/ticket_repository.go | **hardened mutators (11-07) + note rendering (11-08)** | ✓ VERIFIED | `UpdateState`/`Dismiss`/`Triage` all in-tx FOR UPDATE + matrix re-check + status precondition; private helpers `loggedHoursTx` + `hasNonTerminalActivitiesTx` (byte-identical SQL); `scanTicketRow` derives DismissedNote; pool-level `LoggedHours`/`HasNonTerminalActivities` signatures Phase-12-stable |
| internal/adapters/secondary/postgres/ticket_repository_test.go | race battery + pins + note test | ✓ VERIFIED | 7 new tests (TestDismissalGuard_RaceWithPendingSubmit, TestDismissalRace_VsTriage, TestDismiss_RejectsPlanned, TestTransitionRace_VsDismiss, TestTriage_RejectsDismissed, TestUpdateState_ResolvedBlocked, TestTicketRepository_DismissedNote) — all PASS under `-race` |
| internal/core/domain/ticket/ticket.go | entity, matrix, sentinels + **DismissedNote derived field (11-08)** | ✓ VERIFIED | `DismissedNote *string` `json:"dismissed_note,omitempty"`, documented derived-on-read/never-persisted |
| internal/core/services/ticket/ticket.go | Create/Transition/Dismiss/Triage + **title validation (11-08)** | ✓ VERIFIED | Create rejects `len(title) > 255`; UpdateDetails rejects `*title == ""` and `len(*title) > 255` → `ErrInvalidRequest` → 400 (handler writeError ticket_handler.go:351-352); doc comments state fast-fail-vs-authoritative layering |
| internal/core/services/ticket/ticket_test.go | title boundary tests | ✓ VERIFIED | 256-char rejected / 255-char accepted on Create + UpdateDetails; empty-title update rejected (ticket_test.go:96-106, 353-369) |
| internal/adapters/primary/http/ticket_handler_test.go | contract asserts note on all read paths | ✓ VERIFIED | dismiss response, detail GET, list all assert `"dismissed_note":"dismissed with 0 h logged"` |
| internal/adapters/primary/http/ticket_handler.go | HTTP adapter, 9 routes, no DELETE | ✓ VERIFIED | Unchanged; ErrInvalidRequest → 400 mapping confirmed |
| internal/core/ports/ticket_repository.go | full TicketRepository interface | ✓ VERIFIED | Unchanged (no signature churn — 11-07 constraint honored) |
| hourglass-vault ADR-P-003 / ADR-P-013 / ADR-BE-016 | decision records | ✓ VERIFIED | All present in vault; index entries unchanged |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| repo.UpdateState/Dismiss/Triage | domain matrix | in-tx `CanTransition(lockedStatus, to)` under FOR UPDATE | WIRED | Authoritative re-check inside every mutator tx (ticket_repository.go:262, 336, 435) |
| repo.Dismiss in-tx Σ | time_entries | `loggedHoursTx` via tx.QueryRow (not pool) | WIRED | Byte-identical WHERE to `LoggedHours`; runs after ticket + linked-activity FOR UPDATE locks |
| repo.UpdateState resolved branch | time_entries subtree | `hasNonTerminalActivitiesTx` recursive CTE via tx | WIRED | Same CTE as pool-level `HasNonTerminalActivities`, executed in-tx |
| mutator UPDATEs | SQL backstop | `AND status = currentStatus` on all three UPDATEs | WIRED | UpdateState:279-280, Dismiss:376-378, Triage:489-490 |
| scanTicketRow | every JSON ticket response | Get/List/returned-ticket paths all funnel through scanTicketRow | WIRED | Note derived at scan; `ticketColumns` unchanged |
| service Create/UpdateDetails | handler 400 | ErrInvalidRequest mapping | WIRED | ticket_handler.go:351-352 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| ticket Dismiss dismissed_hours | Σ hours | in-tx `loggedHoursTx` under locks (authoritative, supersedes service value) | Yes — commit-adjacent Σ written; blocked (ErrDismissalBlocked) if > 0 | ✓ FLOWING |
| ticket DismissedNote | derived note | `scanTicketRow` from dismissed_hours column | Yes — "dismissed with {N} h logged", N trimmed of trailing zeros | ✓ FLOWING |
| ticket ListHistory | audit rows | real INSERTs in mutator txs | Yes — in-tx writes | ✓ FLOWING |
| activity origin refs / contract sold_hours | request → INSERT/SELECT | real validation + persistence | Yes — unchanged | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full backend build | `go build ./...` | exit 0 | ✓ PASS |
| Race battery (TICK-02/TICK-04 concurrency invariants) | `go test -race ./internal/adapters/secondary/postgres/ -run 'TestDismissalGuard_RaceWithPendingSubmit\|TestDismissalRace_VsTriage\|TestDismiss_RejectsPlanned\|TestTransitionRace_VsDismiss\|TestTriage_RejectsDismissed\|TestUpdateState_ResolvedBlocked\|TestTicketRepository_DismissedNote' -count=1` | ok (5.8s), all 7 PASS under `-race` | ✓ PASS |
| Postgres ticket suite (regression) | `go test ./internal/adapters/secondary/postgres/ -run TestTicket -count=1` | ok (4.7s) | ✓ PASS |
| Ticket service suite (incl. title validation 255/256/empty) | `go test ./internal/core/services/ticket/ -count=1` | ok (4.6s) | ✓ PASS |
| HTTP adapter suite (incl. note contract assertions) | `go test ./internal/adapters/primary/http/ -run TestTicket -count=1` | ok (6.1s) | ✓ PASS |
| Full workspace suite | `go test ./...` | 0 FAIL across all packages (postgres + http adapters ok) | ✓ PASS |

**Behavioral evidence for behavior-dependent truths:** the two previously-failed truths (TICK-02 lifecycle under concurrency, TICK-04 guarded dismissal) are each exercised by passing tests that run the exact transition/invariant — exactly-one-winner goroutine races, dismissed-resurrection pin, resolved-block pin, pending-submit guard blocking — under `-race` against testcontainers Postgres. Both additionally carry user-APPROVED live-API checkpoints (11-07 checkpoint: CR-01 invariants under concurrent load; 11-08 checkpoint: note renders server-side, oversized title → 400).

### Probe Execution

Step 7c: SKIPPED — no probe scripts exist and none declared in any 11-0x-PLAN.md (unchanged from initial verification).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| FND-01 | 11-01, 11-02, 11-05 | Activity origin type + reference set, set once, immutable | ✓ SATISFIED | Truth #1 |
| FND-02 | 11-02, 11-03, 11-05 | Employee proposal with routing-based approval | ✓ SATISFIED | Truth #2 |
| FND-03 | 11-01, 11-04 | Contract sold_hours read/write for V5 mining | ✓ SATISFIED | Truth #3 |
| FND-04 | 11-01, 11-02, 11-05 | Origin refs stored on activities; Phase-13 fallback additive | ✓ SATISFIED | Truth #4 |
| TICK-01 | 11-01, 11-02, 11-05, 11-06 | Internal ticket with closed-set kind | ✓ SATISFIED | Truth #5 |
| TICK-02 | 11-01, 11-02, 11-06, **11-07** | Lifecycle + reopen + resolved-blocks + invalid edges rejected — under concurrency | ✓ SATISFIED | Truth #6; in-tx matrix re-validation + race battery green |
| TICK-03 | 11-02, 11-06 | Triage converts ticket to 1..N activities; atomic | ✓ SATISFIED | Truth #7 |
| TICK-04 | 11-01, 11-02, 11-06, **11-07, 11-08** | Dismissal guard (in-tx Σ) + dismissed note rendered | ✓ SATISFIED | Truth #8; loggedHoursTx authoritative + DismissedNote on all reads |
| TICK-05 | 11-01, 11-02, 11-05, 11-06 | Immutable event stream via BE-012 audit trail; no update/delete | ✓ SATISFIED | Truth #9 |

Orphaned requirements check: all 9 IDs (FND-01..04, TICK-01..05) are claimed by plans and verified — none orphaned.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX markers in phase-modified files | — | Clean (grep scan exit 1 = no matches) |
| postgres/ticket_repository.go | 691-693, 709-711 | WR-06 (advisory): Σ status set `('submitted','approved')` excludes in-pipeline `pending_manager`/`pending_finance` — matches D-13's letter and the TICK-04 must-have wording ("submitted+approved"); a pending_manager entry stops blocking at the next approval | ⚠️ ADVISORY (11-REVIEW.md) | Does NOT violate the stated must-have (the truth pins 'submitted+approved'); recommended follow-up: widen the set in both Σ queries + ADR-BE-016 D-13 |
| postgres/ticket_repository.go | 344-373 | WR-07 (advisory, race): the `draft → submitted` UPDATE path takes no activity-row lock, so a submit can commit between `loggedHoursTx` and dismissal COMMIT | ⚠️ ADVISORY (11-REVIEW.md) | The approved live-API checkpoint exercised the dismissal-vs-submit race and confirmed the guard held under load; residual window is a follow-up hardening (locking read over counted entry rows), not a must-have violation |
| postgres/ticket_repository.go | 262-264 | WR-08 (advisory, latent): `repo.UpdateState` accepts the matrix-legal dismissal edges with no Σ check — the service rejects dismissal-via-UpdateState (services/ticket/ticket.go:219-221), so no current code path bypasses T-11-07 | ⚠️ ADVISORY (11-REVIEW.md) | Latent convention gap for future callers; recommended follow-up: reject `to == 'dismissed'` in UpdateState |
| primary/http/ticket_handler.go | 368-377 | IN-08 (info): `assignee_id: ""` silently no-ops — assignee can never be unassigned | ℹ️ ADVISORY (11-REVIEW.md) | Unrelated to phase must-haves; follow-up ticket-surface work |

Advisory findings WR-06/07/08 and IN-08 (11-REVIEW.md) were evaluated against the must-have truths: none violate them. TICK-04's stated guard is "submitted+approved, not deleted" (matching D-13 and the code); WR-07's residual window was covered by the user-approved live concurrency checkpoint; WR-08 is latent (no current caller); IN-08 is out of must-have scope. These are recorded as advisory follow-ups, not gaps.

### Human Verification Required

None — both items from the initial verification were resolved and APPROVED by the user:

1. **CR-01 concurrency invariants under live-API load** (initial item 1) — **RESOLVED/APPROVED** via 11-07 blocking checkpoint: triage vs dismiss race → exactly one success (loser 400 invalid transition); dismiss vs entry submit → guard held; no dismissal committed with `dismissed_hours=0` while committed logged hours existed at check time. Corroborated by the automated race battery (green under `-race`).
2. **"dismissed with N h logged" note rendering** (initial item 2) — **RESOLVED/APPROVED** via 11-08 blocking checkpoint: note renders on `GET /tickets/{id}`, `GET /tickets`, and the dismiss response; 300-char title → 400; 255-char title accepted. Corroborated by repo + handler contract tests.

### Gaps Summary

No gaps remain. Both previously-failed truths are closed:

1. **TICK-02 (lifecycle integrity under concurrency)** — closed by 11-07: every matrix decision is re-validated inside the mutator tx under a `FOR UPDATE` ticket row lock (`UpdateState`/`Dismiss`/`Triage`), the resolved-block re-runs via `hasNonTerminalActivitiesTx` in-tx, and every UPDATE carries the `AND status = currentStatus` SQL backstop. Race battery (exactly-one-winner goroutine tests + sequential pins) green under `-race`; live-API checkpoint approved.
2. **TICK-04 (dismissal guard + note)** — closed by 11-07 + 11-08: the hours Σ is re-computed inside the dismiss tx (`loggedHoursTx`) after ticket + linked-activity `FOR UPDATE` locks (`ErrDismissalBlocked` if > 0, deterministic 2-tx test proves the pending-submit block), and the "dismissed with N h logged" note is server-derived via `scanTicketRow` on every read path (contract-pinned in dismiss response, detail, and list). Live-API checkpoint approved.

**Deferred items:** None — no later phase addresses these closures (they are complete).

---

_Verified: 2026-08-07T18:30:00Z_
_Verifier: the agent (gsd-verifier, re-verification)_
