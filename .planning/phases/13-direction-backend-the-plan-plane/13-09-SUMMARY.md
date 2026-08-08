---
phase: 13-direction-backend-the-plan-plane
plan: 09
subsystem: api
tags: [direction, audit, go, postgres, tdd, gap-closure]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane
    provides: executed 13-01..08 direction service/repo/mock/handler code (13-VERIFICATION.md gap context)
provides:
  - supersede-on-create writes the 'superseded' audit row in the same tx (CR-02, DIR-02)
  - Service.Claim nil-WgID guard: user-targeted rows → 404, never a panic (CR-01, DIR-03)
  - wholeCent DECIMAL(8,2) ceiling: absurd est_hours → ErrInvalidHours → 400 (WR-02)
  - MockWorkingGroupRepo.ListMembersFn per-method override seam
affects: [Phase 19 history filters, direction claim surface, direction create/claim validation]

actuals:
  tokens: 2217
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN per gap fix: failing regression at service-unit AND HTTP-boundary level before the fix"
    - "Service mirrors the repo's lock predicate (wg_id IS NOT NULL) at the pool boundary (CR-01)"

key-files:
  created: []
  modified:
    - internal/core/services/direction/direction.go
    - internal/core/services/direction/direction_test.go
    - internal/core/services/testdata/mock_direction_repo.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/primary/http/direction_handler_test.go

key-decisions:
  - "CR-01 handler regression activates the user-targeted row first: the nil-guard sits after the status fast-fail, so the 404 contract is only reachable for an ACTIVE user row (plan's own unit behavior pinned 'status active'; its abbreviated handler text omitted the step)"
  - "wrapPGError untouched: with the service ceiling the 22003 path is unreachable for client input; a global 22003 mapping would alter unrelated repos (time entries) — per plan"
  - "999999.99 ceiling-acceptance unit test is a guard-rail (passes before AND after the fix by design — proves the ceiling never rejects the max valid value)"

patterns-established:
  - "Audit-first via BE-012 restored on supersede: the service builds the full audits slice (created + superseded) and hands it to the repo tx — the repo never synthesizes rows"
  - "Per-method Fn override seam (ListMembersFn) follows the ClaimFn pattern: nil = default, backward compatible"

requirements-completed: [DIR-01, DIR-02, DIR-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Supersede-on-create writes BOTH audit rows in one tx — 'created' on the new row + 'superseded' on the flipped target (CR-02, DIR-02 audit-first)"
    requirement: DIR-02
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/supersede-on-create_writes_created_+_superseded_audit_rows_(CR-02)"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/supersede-on-create_writes_BOTH_audit_rows_in_the_tx_(CR-02)"
        status: pass
    human_judgment: false
  - id: D2
    description: "POST /direction/claims with a user-targeted row id returns 404 with the error envelope — no panic, no dropped connection; mock mirrors the repo wg_id IS NOT NULL predicate (CR-01, DIR-03)"
    requirement: DIR-03
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Claim/claim_on_a_user-targeted_row_returns_ErrDirectionNotFound_without_touching_ListMembers_(CR-01)"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/claims_on_a_user-targeted_row_are_404_—_no_panic,_connection_stays_up_(CR-01)"
        status: pass
    human_judgment: false
  - id: D3
    description: "est_hours above the DECIMAL(8,2) ceiling (999999.99) is rejected with ErrInvalidHours -> 400 at create AND claim; the ceiling value stays accepted; no 22003 path from client input (WR-02, T-13-32)"
    requirement: DIR-01
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/est_hours_0_/_negative_/_sub-cent_/_absurd_rejected_with_ErrInvalidHours"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/est_hours_at_the_DECIMAL(8,2)_ceiling_succeeds_(WR-02)"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Claim/claim_with_non-positive,_sub-cent_or_absurd_hours_is_rejected_with_ErrInvalidHours"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/est_hours_above_the_DECIMAL(8,2)_ceiling_is_400,_never_500_(WR-02)"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 09: Verification-Gap Closure (CR-02 / CR-01 / WR-02) Summary

**Supersede-on-create now writes both audit rows in one tx, claims on user-targeted rows return 404 instead of panicking, and est_hours over the DECIMAL(8,2) ceiling maps to 400 — each gap closed with TDD regression tests at the service-unit and HTTP boundaries**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-08T15:49:29Z
- **Completed:** 2026-08-08T15:55:06Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- CR-02 closed: `Service.Create` builds a named audits slice — 'created' on the new row always, 'superseded' on the flipped target when `SupersedesID != nil` — handed to the repo tx, which iterates the slice in-tx (direction_repository.go:242-246). The audit-first BE-012 contract (port doc direction_repository.go:27-29, ADR-BE-018 §3) is restored at the service boundary; Phase 19 history filters on 'superseded' now find real events. Proven at unit level (2-row slice with created-then-superseded ordering, entity ids, actor, nil payloads) and at the HTTP boundary (both `audit_logs` rows present via pool queries).
- CR-01 closed: `Service.Claim` guards `wg.WgID == nil` → `ErrDirectionNotFound` between the status fast-fail and `ListMembers` — mirroring the repo's lock predicate `wg_id IS NOT NULL` (direction_repository.go:441). `MockDirectionRepo.Claim` mirrors the guard so a future unit test can never silently claim a user row through the mock. The HTTP regression proves the connection stays up (404 envelope decoded; pre-fix the server panicked and dropped the connection with EOF).
- WR-02 closed: `const maxEstHours = 999999.99` + `wholeCent` upper bound — client input above the DECIMAL(8,2) ceiling is rejected with `ErrInvalidHours` → 400 at both create and claim; 999999.99 stays accepted (float64 `*100` rounds exactly to 99999999, verified). The PG 22003 → 500 path is unreachable for client input; `wrapPGError` untouched per plan.
- Mock seam: `MockWorkingGroupRepo.ListMembersFn` (ClaimFn-style, nil = default hard-coded lookup, backward compatible — routing-service call sites unaffected) added to make the ListMembers trap expressible.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1: CR-02 — supersede audit rows** - `5e38e61` (test) + `90fc12e` (feat)
2. **Task 2: CR-01 — Claim nil guard + mock mirror** - `232690a` (test) + `418fa72` (feat)
3. **Task 3: WR-02 — wholeCent ceiling** - `a93b138` (test) + `91f296b` (feat)

**Plan metadata:** pending (docs commit after SUMMARY)

## Files Created/Modified

- `internal/core/services/direction/direction.go` - audits slice (2 rows on supersede), nil-WgID guard in Claim, `maxEstHours` ceiling in `wholeCent`
- `internal/core/services/direction/direction_test.go` - supersede-audit subtest, absurd-hours + ceiling subtests, Claim user-row 404 subtest, `1000000` added to both invalid-hours loops
- `internal/core/services/testdata/mock_direction_repo.go` - `MockDirectionRepo.Claim` user-row guard (mirrors repo predicate)
- `internal/core/services/testdata/mocks.go` - `ListMembersFn` override on `MockWorkingGroupRepo` (the Claim trap seam)
- `internal/adapters/primary/http/direction_handler_test.go` - superseded-audit DB check, claims-on-user-row 404, est_hours 1000000 → 400 subtests

## Decisions Made

- **CR-01 handler regression activates the user-targeted row first.** The plan's handler-test text ("create a self-direction row (200), then POST /direction/claims") omitted the activate step, but its own unit behavior pins "status active" — the guard sits after the status fast-fail, so only an ACTIVE user row reaches the deref and yields 404. Without activation the subtest would observe 409 (draft) and never exercise the guard. The test includes the activate step; the 404 contract is exactly as planned.
- **wrapPGError untouched** (per plan): the service ceiling makes 22003 unreachable for client input; a global 22003 mapping would change unrelated repos (time entries).
- **The 999999.99 ceiling-acceptance test is a guard-rail by design** — it passes before AND after the fix, proving the ceiling never rejects the maximum valid value. The RED failures for Task 3 come from the 1000000 cases (unit: row created with no error; HTTP: 500 instead of 400).

## Deviations from Plan

None - plan executed exactly as written. (The activate-step detail above is a faithful reading of the plan's own pinned unit behavior, not a behavior change.)

## TDD Gate Compliance

All three tasks show RED (`test(13-09)`) before GREEN (`feat(13-09)`):

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| 1 (CR-02) | 5e38e61 | 90fc12e | — | Pass |
| 2 (CR-01) | 232690a | 418fa72 | — | Pass |
| 3 (WR-02) | a93b138 | 91f296b | — | Pass |

RED failures were for the right reasons: Task 1 — audits slice length 1 vs 2 (unit) and superseded COUNT 0 (HTTP); Task 2 — nil-deref panic at direction.go:478 (unit) and EOF dropped connection (HTTP); Task 3 — row created without error (unit) and 500 instead of 400 (HTTP). No REFACTOR commits — implementations were minimal per plan.

## Issues Encountered

None. The plan's float-math assumption (999999.99*100 rounds exactly in float64) was verified before writing tests and holds.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three verification-gap defects (CR-02, CR-01, WR-02) closed with dual-layer regressions; full direction suite green (service unit, HTTP testcontainers, repo mutator suite).
- Plan 13-10 remains: T-13g-04 (directed_to cross-org refs → orgRepo membership check) and T-13g-05 (Unclaim audit action 'cancelled' → pinned 'unclaimed').
- Phase 19 history filters can now rely on 'superseded' audit events.

## Self-Check: PASSED

All 5 files exist on disk; all 6 task commits (5e38e61, 90fc12e, 232690a, 418fa72, a93b138, 91f296b) present in git history.

---
*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*
