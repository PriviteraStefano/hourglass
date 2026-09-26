---
phase: 13-direction-backend-the-plan-plane
plan: 05
subsystem: database
tags: [postgres, pgx, direction, tx, for-update, cr-01, audit]

requires:
  - phase: 13-01
    provides: migrations 021/022 — direction_rows table (CHECK vocabularies, XOR/queued/scheduled guards, self-FKs) + org_settings
  - phase: 13-03
    provides: direction domain (entity, lifecycle matrix, errors, audit vocabulary) + ports.DirectionRepository pin + MockDirectionRepo
  - phase: 13-04
    provides: org_settings_repository.go (untouched — this plan does not modify it)
provides:
  - direction_repository.go mutator half: Create (plain + supersede-on-create in one tx), Activate, Cancel, Claim (Σ guard under WG-row FOR UPDATE lock), Unclaim; helpers scanDirectionRow / insertDirectionAudit / getByIDTx + pool-level Get
  - direction_repository_test.go integration suite incl. TestDirectionClaim_Concurrent battery and TestDirectionClaim_SupersedeCancelReclaim chain test
affects: [13-06 (extends this file with ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs + the full-interface assertion), 13-07 (service consumes Get/Create/Activate/Cancel/Claim/Unclaim), 13-08]

actuals:
  tokens: 16356
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Mutator tx closure (CR-01): BeginTx + defer Rollback + SELECT ... FOR UPDATE in-org + in-tx re-validation (matrix/status/Σ/membership) + status-precondition UPDATE backstop + in-tx audit rows (BE-012) + getByIDTx re-read + Commit — the ticket_repository.go Dismiss/UpdateState skeleton applied to the direction plane"
    - "Σ consumption in cents: math.Round(h*100) compare over the predicate origin_direction_id = wgRowID AND status IN ('draft','active') — superseded/cancelled claim rows never consume (coverage ReplaceAllocations precedent)"
    - "Tx-variant pair: pool-level Get + getByIDTx (loggedHoursTx / hasNonTerminalActivitiesTx house pattern)"

key-files:
  created:
    - internal/adapters/secondary/postgres/direction_repository.go
    - internal/adapters/secondary/postgres/direction_repository_test.go
  modified: []

key-decisions:
  - "Claim audit entity_id pinning: the repo generates the claim row id (the port signature takes no id), so Claim pins the audit row's entity_id to the claim row it creates when the caller passed uuid.Nil — entity_id = the direction row id per ADR-BE-018 §3."
  - "Full-interface assertion deferred to 13-06: ports.DirectionRepository declares the read-model methods 13-06 owns, so the var _ ports.DirectionRepository assertion cannot compile on the mutator-only half; Get ships here, the assertion lands with 13-06."
  - "Membership re-check uses the real schema table wg_members (000_full_schema) — the plan text named 'working_group_members', which does not exist; columns wg_id/user_id with UNIQUE(wg_id, user_id)."

patterns-established:
  - "Supersede-of-claim-row carry (ADR-BE-018 §5): Create locks the target FOR UPDATE, re-checks draft|active in-tx, inherits origin_direction_id onto the superseding row (WG-shaped superseding row -> ErrInvalidTarget), flips the target with a status-precondition backstop — the Σ predicate keeps the budget double-count-free across the chain"

requirements-completed: [DIR-01, DIR-02, DIR-03]

coverage:
  - id: D1
    description: "Create mutator — plain insert (draft default) + supersede-on-create in ONE tx: target locked FOR UPDATE, re-checked draft|active in-tx (CR-01), new row carries supersedes_id, target flips to superseded with status-precondition backstop, created+superseded audit rows in-tx; claim-row targets carry origin_direction_id (user-targeted only, ErrInvalidTarget on WG shape); chain rewrite / cancelled targets rejected (ErrInvalidTransition); cross-org targets ErrDirectionNotFound"
    requirement: DIR-01
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Create"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Create_Supersede"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Create_SupersedeClaimRow"
        status: pass
  - id: D2
    description: "Activate + Cancel txs — matrix re-validated against the FOR UPDATE locked row with a status-precondition UPDATE backstop; cancel requires a non-empty reason (ErrCancelReasonRequired fast-fail, DB CHECK second line), reason persisted + 'cancelled' audit with reason payload in-tx; terminal rows reject further transitions; cross-org ids ErrDirectionNotFound"
    requirement: DIR-02
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Activate"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Cancel"
        status: pass
  - id: D3
    description: "Claim tx — WG row locked FOR UPDATE in-org (wg_id IS NOT NULL shape pin), in-tx re-checks: active-only (ErrWgRowNotActive), wg_members membership (ErrNotWgMember), Σ claims in cents over draft|active claim rows vs budget (ErrClaimOverBudget 409, uncapped when budget NULL); claim row user-targeted/queued/draft with attribution + priority/due_date copied from the WG row, 'claimed' audit in-tx. Unclaim = cancel-with-guard (claim rows only, ErrInvalidRequest otherwise; reason required) — hours return via Σ"
    requirement: DIR-03
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Claim"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Unclaim"
        status: pass
  - id: D4
    description: "Concurrency + chain invariants: TestDirectionClaim_Concurrent (5 members x 2.00h vs 8.00 budget — exactly 4 commit + 1 ErrClaimOverBudget, Σ == 8.00, 4 claim rows: over-subscription never commits, CR-01 closure); TestDirectionClaim_SupersedeCancelReclaim (claim -> supersede keeps Σ == 8.00, no strand/double-count -> cancel releases hours Σ == 0 -> re-claim succeeds)"
    requirement: DIR-03
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionClaim_Concurrent"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionClaim_SupersedeCancelReclaim"
        status: pass
---

# Phase 13 Plan 5: Direction Repository Mutators — Supersede-on-Create, Lifecycle, Σ-Guarded Claims Summary

The postgres correctness layer of the direction plane: `direction_repository.go` implements the mutator half of `ports.DirectionRepository` — Create (plain insert + supersede-on-create in ONE tx with the CR-01 FOR UPDATE re-validation closure), Activate, Cancel, Claim (WG-row lock + in-cents Σ-consumption guard + in-tx membership re-check), Unclaim (cancel-with-claim-row-guard) — every mutator writing its audit rows in the same transaction (BE-012), with the full integration test battery including the concurrent-claims race and the supersede-chain hours-return contract.

## Accomplishments

- **Create (plain + supersede-on-create, one tx)**: target row locked `SELECT ... FOR UPDATE` in-org, re-checked `draft|active` in-tx (CR-01 — a second supersede of the same target, or a cancelled target, returns `ErrInvalidTransition`; chain rewrite blocked, Pitfall 4); new row inserted carrying `supersedes_id`; target flipped to `superseded` with a status-precondition UPDATE backstop; `created` + `superseded` audit rows written in the same tx. Claim-row targets inherit `origin_direction_id` onto the superseding row and must stay user-targeted (`wg_id` set → `ErrInvalidTarget`) per ADR-BE-018 §5 — claim hours move along the chain, the WG-budget Σ stays unchanged (proven: superseded target drops out, new draft row counts in).
- **Activate / Cancel**: matrix re-validated against the FOR UPDATE locked row (draft→active; draft|active→cancelled) with the status-precondition backstop; cancel requires a non-empty reason (fast-fail at the repo boundary — DB CHECK second line), persists the reason, and writes the `cancelled` audit with the reason payload in-tx; terminal rows reject further transitions; cross-org ids read as `ErrDirectionNotFound` (org-scoped locks, no existence oracle).
- **Claim (Σ guard under lock)**: WG row locked `FOR UPDATE` in-org; in-tx re-checks: active-only (`ErrWgRowNotActive`), `wg_members` membership (`ErrNotWgMember`), Σ claimed in CENTS (`math.Round(h*100)`) over `origin_direction_id = wgRowID AND status IN ('draft','active')` vs budget — superseded/cancelled claim rows never consume; over budget → `ErrClaimOverBudget` (409); uncapped when budget NULL (D-13-14). Claim row: user-targeted, queued, draft, `directed_by` = WG creator (attribution), `priority`/`due_date` copied (A8), `claimed` audit in-tx (entity_id pinned to the generated claim row id).
- **Unclaim** = cancel-of-claim-row via the shared cancel tx internals with the claim-row guard (`origin_direction_id` NULL → `ErrInvalidRequest`), reason required (D-13-16) — hours return to the WG budget automatically (Σ-derived).
- **Test battery**: 24 integration tests across the two files — create/supersede (incl. chain-rewrite, cancelled-target, cross-org, claim-row carry + WG-shape guard), activate/cancel matrix, claim shape/membership/draft-row/over-budget/uncapped, unclaim + re-claim, the `TestDirectionClaim_Concurrent` battery (5 members × 2.00h vs 8.00 budget: exactly 4 commit + 1 `ErrClaimOverBudget`, Σ == 8.00, 4 claim rows — over-subscription never commits), and `TestDirectionClaim_SupersedeCancelReclaim` (claim → supersede (Σ unchanged) → cancel (Σ == 0) → re-claim (Σ == 8.00)).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Full-interface assertion deferred to 13-06; pool-level Get implemented**
- **Found during:** Task 1 (acceptance criterion (a))
- **Issue:** The plan requires `var _ ports.DirectionRepository = (*DirectionRepository)(nil)` to compile, but `ports.DirectionRepository` declares four read-model methods (ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs) that the plan's own scope (and the orchestrator briefing) assigns to 13-06 — the assertion cannot compile on the mutator-only half. Criterion (a) as written is unsatisfiable within the declared scope.
- **Fix:** Implemented the mutators plus the minimal pool-level `Get` (trivial single-row read required by the interface and by 13-07's service fast-fails; not a read-model — the in-tx twin `getByIDTx` was already planned). Skipped the full-interface assertion; it lands with 13-06 when the file's last interface method exists. No shared contract changed.
- **Files modified:** internal/adapters/secondary/postgres/direction_repository.go
- **Commit:** a2e508e

**2. [Rule 3 - Blocking issue] Membership table name in the plan text is wrong**
- **Found during:** Task 3 (read_first: migrations)
- **Issue:** The plan's Claim tx references `working_group_members`; no such table exists. The real schema table is `wg_members` (migrations/000_full_schema.up.sql:250) with columns `wg_id`/`user_id` and `UNIQUE(wg_id, user_id)` (teardown list agrees).
- **Fix:** Membership re-check uses `SELECT EXISTS (SELECT 1 FROM wg_members WHERE wg_id = $1 AND user_id = $2)` — the plan's intent (authoritative in-tx membership check, D-13-12) unchanged.
- **Files modified:** internal/adapters/secondary/postgres/direction_repository.go
- **Commit:** b21278e

**3. [Rule 1 - Bug] Claim audit row would carry entity_id = uuid.Nil**
- **Found during:** Task 3 (TestDirectionRepository_Claim failed on the audit assertion)
- **Issue:** The claim row's id is generated inside `Claim` (the port signature takes no id), so a caller-built audit row cannot know it — the audit row was written with `entity_id = uuid.Nil`, breaking the ADR-BE-018 §3 trail (entity_id = the direction row id).
- **Fix:** `Claim` pins `auditLog.EntityID = claimID` when the caller passed `uuid.Nil` (documented in the method comment).
- **Files modified:** internal/adapters/secondary/postgres/direction_repository.go
- **Commit:** b21278e

**Total deviations:** 3 auto-fixed (1 missing-functionality, 1 blocking, 1 bug). **Impact:** none on the plan's invariants — all three keep the plan's intent intact; the first is a scope-boundary note for 13-06.

## Self-Check

- [x] `internal/adapters/secondary/postgres/direction_repository.go` exists (created a2e508e)
- [x] `internal/adapters/secondary/postgres/direction_repository_test.go` exists (created a2e508e)
- [x] Commit a2e508e exists (feat: repo foundation + Create)
- [x] Commit 6587239 exists (feat: Activate + Cancel)
- [x] Commit b21278e exists (feat: Claim + Unclaim + batteries)
- [x] `go test ./internal/adapters/secondary/postgres/ -run 'TestDirection' -count=1` exits 0 (plan verification 1)
- [x] `go build ./...` compiles (plan verification 2)
- [x] `go test ./...` full suite green (plan verification 3 — wave-merge equivalent)
- [x] `go vet ./...` clean
- [x] Grep guard: only status-flip UPDATEs (`superseded`/`active`/`cancelled` with preconditions), zero DELETEs — no plan-fact rewrite surface (success criterion 3)

## Verification Results

| Check | Result |
|-------|--------|
| `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionRepository_Create\|TestDirectionRepository_Supersede' -count=1` | ok (Task 1 verify) |
| `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionRepository_Activate\|TestDirectionRepository_Cancel' -count=1` | ok (Task 2 verify) |
| `go test ./internal/adapters/secondary/postgres/ -run 'TestDirectionClaim_Concurrent\|TestDirectionRepository_Claim\|TestDirectionRepository_Unclaim\|TestDirectionClaim_SupersedeCancelReclaim' -count=1` | ok (Task 3 verify) |
| `go test ./internal/adapters/secondary/postgres/ -run 'TestDirection' -count=1` | ok |
| `go build ./...` / `go vet ./...` | pass |
| `go test ./...` (full suite) | all packages ok, no FAIL |

## Notes for 13-06 (read-models plan, extends this file)

- `direction_repository.go` currently implements: `Get`, `Create`, `Activate`, `Cancel`, `Claim`, `Unclaim` (repo-only method — NOT on the port) + helpers. The four read-model methods (`ListPlan`, `Coverage`, `AbsenceWindows`, `FirstDirectionRefs`) are NOT yet implemented — add them AND the `var _ ports.DirectionRepository = (*DirectionRepository)(nil)` assertion in 13-06 (it becomes compilable only once the interface is fully implemented). Do not remove `Unclaim` (13-07's service consumes it as a repo-level method).
- `insertDirectionAudit` writes `entity_type` from `log.EntityType` (caller-controlled) — the service pins `direction.AuditEntityDirection`.
- Requirement marking: DIR-01/02/03 are declared by sibling plans 13-06/07/08 (no SUMMARYs yet), so the shared-ID gate kept them Pending; they flip to Complete when the last declaring plan finishes.

## Threat Flags

None — the security-relevant surface (in-tx locks, org-scoped predicates, audit-in-tx, status preconditions) is exactly the plan's `<threat_model>` surface (T-13-14/15/16/17/18 mitigations), no new endpoints/auth paths/schema changes beyond the declared files.

## Known Stubs

None — no placeholder values, no TODO/FIXME, no empty data sources. The four unimplemented read-model methods are a planned scope split (13-06), not stubs.
