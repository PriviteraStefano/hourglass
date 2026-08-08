---
phase: 13-direction-backend-the-plan-plane
plan: 10
subsystem: api
tags: [direction, audit, go, postgres, tdd, gap-closure, security]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane
    provides: executed 13-01..09 direction service/repo/mock/handler code + CR-01/CR-02/WR-02 hardening (13-VERIFICATION.md gap context)
provides:
  - directed_to same-org active-membership gate in Service.Create (WR-01, T-13g-04): cross-org/inactive targets -> 400 before any mode/routing decision
  - Service.Unclaim writes the pinned 'unclaimed' audit action (WR-03, T-13g-05): unclaim distinguishable from cancel in Phase 19 history filters
  - AuditActionUnclaimed is now a LIVE code path — no dead vocabulary constant
affects: [Phase 19 history filters, direction create validation, ADR-BE-018 §3 compliance]

actuals:
  tokens: 2317
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN per gap fix: failing regression at service-unit AND HTTP/repo boundary before the fix"
    - "Same-org membership gate at the service boundary (no existence oracle): nil / inactive / cross-org are indistinguishable -> one sentinel"

key-files:
  created: []
  modified:
    - internal/core/services/direction/direction.go
    - internal/core/services/direction/direction_test.go
    - internal/core/ports/direction_repository.go
    - internal/adapters/secondary/postgres/direction_repository_test.go
    - internal/adapters/primary/http/direction_handler_test.go

key-decisions:
  - "WR-03 decision (reverses the 13-05 note): Unclaim now writes AuditActionUnclaimed ('unclaimed') — ADR-BE-018 §3 already pins the vocabulary, the code drifted; aligning code to the ADR keeps unclaim events distinguishable from cancels for Phase 19 history filters and makes the exported constant live (T-13g-05)"
  - "Membership gate sentinel is ErrInvalidRequest (400) with NO existence oracle: a nonexistent user, an inactive member, and a cross-org user are indistinguishable at the service boundary (house style, coverage precedent)"
  - "Repo Unclaim is a pass-through: it writes whatever audit action it is handed — no repo logic change; the repo test documents the pinned contract instead"

patterns-established:
  - "Org containment of direction rows decided at the service, not the FK: FK users(id) alone cannot enforce org containment; GetMembership precedes ResolvePlanningMode so cross-org refs die before any mode/routing decision"

requirements-completed: [DIR-01, DIR-02, DIR-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Service.Create rejects directed_to users with no membership or an inactive membership in orgID with ErrInvalidRequest -> 400, before the mode gate; active members pass through unchanged (WR-01, T-13g-04)"
    requirement: DIR-01
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/directed_to_with_no_membership_rejected_with_ErrInvalidRequest_(WR-01)"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/directed_to_with_an_inactive_membership_rejected_with_ErrInvalidRequest_(WR-01)"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create/directed_to_who_is_an_ACTIVE_member_passes_through_to_the_mode_gate_(WR-01)"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/direction_handler_test.go#TestDirectionHandler/directed_to_a_user_outside_the_org_is_400_(WR-01)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Service.Unclaim writes the pinned 'unclaimed' audit action with {reason} end-to-end; port doc and repo test aligned to ADR-BE-018 §3; AuditActionUnclaimed is a live code path (WR-03, T-13g-05)"
    requirement: DIR-02
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Unclaim/claimant_unclaims_their_own_claim_row_with_a_reason"
        status: pass
      - kind: integration
        ref: "internal/adapters/secondary/postgres/direction_repository_test.go#TestDirectionRepository_Unclaim"
        status: pass
    human_judgment: false

# Metrics
duration: 18min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 10: Gap-Closure (WR-01 directed_to membership gate / WR-03 unclaimed audit action) Summary

**Service.Create now rejects directed_to users who are not active members of the org with 400 before any mode/routing decision, and Service.Unclaim writes the ADR-BE-018 §3-pinned 'unclaimed' audit action — closing both remaining Phase 13 verification gaps with dual-layer TDD regressions**

## Performance

- **Duration:** 18 min
- **Started:** 2026-08-08T17:52:00Z
- **Completed:** 2026-08-08T18:10:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- **WR-01 closed (T-13g-04):** `Service.Create` opens the `DirectedTo != nil` branch with `s.orgRepo.GetMembership(ctx, *req.DirectedTo, orgID)` — nil or `!IsActive` membership → `ErrInvalidRequest` (400), strictly BEFORE `ResolvePlanningMode`. A direction row can no longer reference a user outside the org (or an inactive member): the FK `users(id)` alone cannot enforce org containment, and the cross-org reference previously sailed through to the mode gate (unit RED: ErrForbidden where 400 was required; HTTP RED: 200 insert where 400 was required). The WG branch is untouched (same-org already validated via `wgRepo.GetByID`).
- **WR-03 closed (T-13g-05):** `Service.Unclaim`'s audit literal now writes `AuditActionUnclaimed` ('unclaimed') with the unchanged `{reason}` payload, entity/actor/created-at fields — the exported constant is live code, no longer a trap. The service doc comment and the port's Unclaim contract doc ("One 'unclaimed' audit row in the same tx (ADR-BE-018 §3)") were aligned in the same pass. ADR-BE-018 §3 needed NO amendment — it always pinned 'unclaimed'; the drift was in the code (this reverses the 13-05 note "AuditActionUnclaimed stays a pinned vocabulary constant, never written by the unclaim path"). A Phase 19 history filter on 'unclaimed' now returns real unclaim events, distinguishable from cancels.
- The repo needed NO logic change (pass-through — it writes whatever action it is handed); `TestDirectionRepository_Unclaim` now documents the pinned contract via `directionAudit` + `countDirectionAudits` on `AuditActionUnclaimed`.
- Phase 13 stays backend-only: no server-emitted contract changed (warning objects, error envelope, required-reason destructive writes), so no 13-UI-SPEC UI Consideration applies.

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1: WR-01 — directed_to membership gate** - `ddb028f` (test) + `037faa5` (feat)
2. **Task 2: WR-03 — unclaimed audit action** - `b637f93` (test) + `b3f6b92` (feat)

**Plan metadata:** `docs(13-10)` commit follows this SUMMARY.

## Files Created/Modified

- `internal/core/services/direction/direction.go` - membership gate at the top of the `DirectedTo` branch (14 lines, before `ResolvePlanningMode`); Unclaim audit literal `AuditActionCancelled` → `AuditActionUnclaimed` + Unclaim doc comment updated
- `internal/core/services/direction/direction_test.go` - 3 new `TestService_Create` subtests (no-membership / inactive / active pass-through); claimant-unclaim assertion pinned to `AuditActionUnclaimed`
- `internal/adapters/primary/http/direction_handler_test.go` - cross-org `directed_to` → 400 subtest (foreign-org user registered via `registerAndLogin`, id looked up on the pool)
- `internal/core/ports/direction_repository.go` - Unclaim contract doc: "One 'unclaimed' audit row in the same tx (ADR-BE-018 §3)"
- `internal/adapters/secondary/postgres/direction_repository_test.go` - `TestDirectionRepository_Unclaim` helper + count switched to `AuditActionUnclaimed`

## Decisions Made

- **WR-03 reversal (decision_record honored):** Unclaim writes `AuditActionUnclaimed`. The alternative (delete the constant + amend ADR-BE-018 §3) would churn the record of truth to match a less-distinguishing behavior. Reversibility: one line + docs. This reverses the 13-05 decision recorded in STATE.md.
- **No existence oracle (house style):** nil membership, inactive membership, and cross-org users collapse into one sentinel (`ErrInvalidRequest` → 400) — an oracle would leak user existence across org boundaries.
- **Repo stays a pass-through:** no `cancelWithGuard` change; the port doc + repo test carry the pinned vocabulary instead.

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance

Both tasks show RED (`test(13-10)`) before GREEN (`feat(13-10)`):

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| 1 (WR-01) | ddb028f | 037faa5 | — | Pass |
| 2 (WR-03) | b637f93 | b3f6b92 | — | Pass |

RED failures were for the right reasons: Task 1 — unit: ErrForbidden instead of ErrInvalidRequest (the foreign target fell through to the self_planned mode gate); HTTP: 200 insert where 400 was required (the cross-org row was silently created). Task 2 — service: audit action 'cancelled' written where 'unclaimed' expected (the repo test passed in RED by design — the repo is a pass-through and the test documents the pinned contract, exactly as the plan's behavior block states). No REFACTOR commits — implementations were minimal per plan.

## Issues Encountered

One executor-side edit mishap: a first edit of the Unclaim audit literal over-matched and briefly deleted the `ActorID`/`Payload`/`CreatedAt` lines plus the Claim section header. Detected by immediate re-read, restored in the same pass, `go build ./...` green before commit — no net effect on the committed diff (verified against the final file). Not a plan deviation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both remaining Phase 13 verification gaps (WR-01, WR-03) closed with dual-layer regressions; the full direction suite is green (service unit, HTTP testcontainers, repo mutator suite) plus `go build ./...` and `go vet` on all three packages.
- Phase 13 gap-closure is complete: CR-01/CR-02/WR-01/WR-02/WR-03 all closed across 13-09 + 13-10.
- Phase 19 history filters can rely on the full pinned vocabulary: created / activated / cancelled / superseded / claimed / unclaimed, all written by live code paths.

## Self-Check: PASSED

All 5 files exist on disk; all 4 task commits (ddb028f, 037faa5, b637f93, b3f6b92) present in git history; all plan-level `<verification>` commands exit 0.

---
*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*
