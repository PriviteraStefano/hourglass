---
phase: 13-direction-backend-the-plan-plane
plan: 07
subsystem: backend
tags: [go, hexagonal, direction, warnings, claims, read-model]

# Dependency graph
requires:
  - phase: 13-04
    provides: orgsettings.Service.ResolvePlanningMode (membership override → org default → manager_planned fallback)
  - phase: 13-05
    provides: postgres DirectionRepository mutators (Create supersede-tx, Activate, Cancel, Claim Σ-guard under FOR UPDATE, concrete Unclaim)
  - phase: 13-06
    provides: ListPlan/Coverage/AbsenceWindows read-models with UTC-midnight day normalization
provides:
  - direction service surface: Create / Activate / Cancel / Claim / Unclaim / ListPlan / Coverage (constructor pinned for 13-08 handler wiring)
  - the gate chain: XOR → hours → same-org → WG-scope → mode → routing (coverage gate shape verbatim, D-G parity)
  - the shared warning pure-function overlay (away | partial | over-capacity | invalid) with 13-UI-SPEC verbatim messages
  - ports.DirectionRepository.Unclaim (additive — the repo in-tx claim-row guard is now reachable via the port)
affects: [13-08 direction handler + cmd/server wiring, Phase 14 (confirmed-only absence tightening), Phase 19 (read-model + warning rendering)]

# Actuals (#2632) — pairs with the plan's estimate (40000 est / 24000 raw) to calibrate future estimates.
actuals:
  tokens: 20290
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pool-level fast-fail UX vs authoritative in-tx re-check (CR-01): service gates are fast-fail, the repo owns the locks"
    - "Shared gate shape: routing.ResolveManagerStage consumed with the coverage gate verbatim (no re-implementation, D-G parity)"
    - "Warning pure-function overlay: single computeWarnings channel for create + read responses, messages pre-rendered server-side"

key-files:
  created:
    - internal/core/services/direction/direction.go
    - internal/core/services/direction/direction_test.go
  modified:
    - internal/core/ports/direction_repository.go
    - internal/core/services/testdata/mock_direction_repo.go

key-decisions:
  - "Unclaim added to ports.DirectionRepository + MockDirectionRepo (additive): the plan's Task 2 pins 'repo.Unclaim'; the postgres repo already implements the in-tx claim-row guard as a concrete method, the port now exposes it"
  - "Unclaim audit = 'cancelled' action with {reason} payload (matches the 13-05 repo tests); the ADR vocabulary constant AuditActionUnclaimed stays pinned but the unclaim path writes cancelled per plan + repo contract"
  - "created/activated audits carry nil payloads (the 13-05 repo test contract); cancelled carries {reason}, claimed carries {wg_row_id, est_hours} with uuid.Nil entity (the repo pins it to the claim row it creates)"
  - "computeWarnings fetches the coverage rows itself (the pinned signature takes only employeeIDs + period): over-capacity derives from the repo read-model, shared by ListPlan/Coverage/create"
  - "Away message: the window's intersection with the queried period, formatted 'Away {2 Jan}' or 'Away {2 Jan}–{3 Jan}' (en dash); invalid messages dedupe to one per employee (day-less message)"
  - "Coverage period totals are computed over the FULL row set (incl. capacity-0 away days — their zero capacity keeps the aggregates truthful); only the rows list excludes them (D-13-26)"

patterns-established:
  - "Mode gate: self_planned self-direction skips routing entirely (D-S); manager_planned routes everyone incl. self (strict A9 reading)"
  - "WG-row create routes on the ANCHORED activity (A10) with uuid.Nil unit/owner — the fallback degrades to the role-gated terminal stage"
  - "Read gates (T-13-26): org-wide plan view manager-only, coverage unit/wg scopes manager-only, employee scope self-view for non-managers"

requirements-completed: [DIR-01, DIR-02, DIR-03, DIR-05, DIR-06]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Create orchestration — XOR/hours/same-org/WG-scope/mode/routing gate chain, supersede fast-fail, warnings-in-response (never blocking)"
    requirement: DIR-01
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Create"
        status: pass
    human_judgment: false
  - id: D2
    description: "Lifecycle orchestration — activate/cancel with pool-level matrix fast-fail before the repo, creator-or-manager gate, audit DTOs"
    requirement: DIR-02
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Activate"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Cancel"
        status: pass
    human_judgment: false
  - id: D3
    description: "WG claim orchestration — active-only + membership + positive whole-cent hours fast-fails feeding the authoritative repo tx; unclaim claim-row guard + claimant/creator/manager gate"
    requirement: DIR-03
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Claim"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Unclaim"
        status: pass
    human_judgment: false
  - id: D4
    description: "Warning overlay — away/partial/over-capacity/invalid with the 13-UI-SPEC verbatim message formats, validity + absence-driven, never blocking"
    requirement: DIR-05
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Warnings"
        status: pass
    human_judgment: false
  - id: D5
    description: "Coverage read-model — scope resolution (employee/unit+descendants/WG), validity split, away-day exclusion, period totals; self-or-manager read gates"
    requirement: DIR-06
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_Coverage"
        status: pass
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go#TestService_ListPlan"
        status: pass
    human_judgment: false

# Metrics
duration: 11min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 7: Direction Service Summary

**The direction service: create gate chain (XOR → hours → same-org → WG-scope → mode → routing with the coverage gate shape verbatim), lifecycle + WG-claim orchestration with pool fast-fails feeding the authoritative repo txs, and the shared warning pure-function overlay + ListPlan/Coverage read-model assembly — all unit-tested against the testdata mocks**

## Performance

- **Duration:** 11 min
- **Started:** 2026-08-08T14:08:52Z
- **Completed:** 2026-08-08T14:17:57Z
- **Tasks:** 3 (each RED→GREEN, tdd_mode on)
- **Files modified:** 4 (2 created, 2 extended)

## Accomplishments

- **Create orchestration** (`Create`): the pinned gate chain in order — XOR target (D-13-05) → est_hours positive whole-cent + scheduled-shape (D-13-02/03) → same-org activity (D-02 pattern) → WG queued-only + scope predicate via `GetAncestry` (D-13-17, A5, Pitfall 9) → mode gate (D-13-19/20, A9: self_planned self-direction skips routing — proven by a role=employee success with nothing seeded; manager_planned routes everyone) → supersede fast-fail (CR-01 UX). WG rows route on the anchored activity (A10). Warnings ride the response, never reject (D-13-03/28).
- **Lifecycle + claims** (`Activate`/`Cancel`/`Claim`/`Unclaim`): pool-level matrix fast-fail BEFORE the repo call (trap-asserted), creator-or-manager permission via the shared BE-014 reach, cancel/unclaim reason requirements (D-13-10/16), claim fast-fails (active-only, WG membership D-13-12, positive whole-cent hours) feeding the repo's authoritative in-tx Σ guard (D-13-13, CR-01). Audit DTOs match ADR-BE-018 §3 (cancelled → {reason}; claimed → {wg_row_id, est_hours} with uuid.Nil entity the repo pins).
- **Warning overlay + read-models** (`computeWarnings`/`ListPlan`/`Coverage`): the shared pure function (away | partial | over-capacity | invalid) with verbatim 13-UI-SPEC messages ("Away 10 Aug–21 Aug" en-dash ranges, "Outside validity period" deduped per employee); validity split drops validity-outside employees from the coverage repo call (D-13-31); capacity-0 days excluded from uncovered rows but kept for warnings + totals (D-13-26); period totals per employee (D-Z); self-or-manager read gates (T-13-26).

## Task Commits

Each task was committed atomically (RED→GREEN per task, tdd_mode):

1. **Task 1: Create orchestration** — `fed0ee7` (test) + `2be2b08` (feat)
2. **Task 2: Lifecycle + claim orchestration** — `d28f40e` (test) + `e290668` (feat)
3. **Task 3: Warning overlay + read-models** — `43491ab` (test) + `0e4dc66` (feat)

**Plan metadata:** (docs commit — after this file)

## Files Created/Modified

- `internal/core/services/direction/direction.go` - the `Service` (deps: directionRepo, activityRepo, wgRepo, unitRepo, orgRepo, orgSettingsSvc, routingSvc — constructor pinned for 13-08) with `CreateDirectionRequest`/`PlanResponse`/`CoverageResponse`/`CoverageTotals`, the gate chain, lifecycle + claim orchestration, `computeWarnings`/`warningsForEmployee`-style pure overlay, scope resolution, and the shared `managerReach` (coverage gate shape verbatim)
- `internal/core/services/direction/direction_test.go` - unit suites: `TestService_Create` (17 subtests), `Activate`/`Cancel`/`Claim`/`Unclaim`, `Warnings`, `ListPlan`, `Coverage` — hermetic mocks, no DB
- `internal/core/ports/direction_repository.go` - additive `Unclaim` method (the plan's Task 2 pins "repo.Unclaim"; the postgres repo already implements the in-tx claim-row guard)
- `internal/core/services/testdata/mock_direction_repo.go` - `Unclaim` implementation + `UnclaimFn` override

## Decisions Made

- **Unclaim via the port (additive, plan-sanctioned):** the plan's Task 2 says "repo.Unclaim" and lists the port in `files_modified`; the postgres repo's concrete `Unclaim` (claim-row guard in-tx) is now a port method, with the mock mirroring the guard. All other port signatures untouched.
- **Unclaim audit action = 'cancelled' + {reason}:** matches both the plan text and the 13-05 repo tests (`AuditActionCancelled` with `{reason}` payload); the `AuditActionUnclaimed` domain constant stays in the ADR vocabulary but is not written by this path.
- **Claim validates whole-cent hours (Rule 2 consistency):** the plan pins "estHours > 0"; sub-cent claims would corrupt the repo's cents-based Σ (rounded Σ ≠ stored DECIMAL(8,2)) — the create-side whole-cent rule (D-13-03) is applied to claims too, same sentinel.
- **Unit scope resolution uses per-unit `ListMembers`** (unit + each descendant) rather than `ListMembersByUnitIDs`: functionally equivalent (A6), and the mock's `ListMembersByUnitIDs` is a nil stub — per-unit calls keep the tests hermetic.
- **created/activated audit payloads are nil** (the 13-05 repo test contract); the PATTERNS-doc's richer created payload was not adopted — the repo tests are the lived contract.

## Deviations from Plan

None - plan executed exactly as written. Two plan-consistent implementation notes above (claim whole-cent validation, per-unit member resolution) were within the plan's stated semantics.

## Issues Encountered

- Mock's `GetAncestry`/`ResolveCommercialContext` defaults derive from the seeded `Activities` map — WG-scope ancestry tests needed the parent chain seeded, which worked as documented.
- Test fixture gaps (unseeded memberships making employees validity-outside) surfaced as failures on first GREEN — seeded valid memberships in the scope-resolution tests; semantics unchanged (nil membership = validity-outside is the conservative reading).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for **13-08** (direction handler + cmd/server wiring): the constructor signature is pinned (`directionsvc.NewService(directionRepo, activityRepo, wgRepo, unitRepo, orgRepo, orgSettingsService, routingSvc)`), the HTTP routes map 1:1 to the service surface, and the handler error switch maps the sentinels to 400/403/404/409 per the 13-08 plan.
- `go build ./...`, `go vet` (touched packages), `go test ./internal/core/services/direction/` and the full `make test` suite (518 passes incl. postgres integration) all green.
- Phase 19 renders the read-model rows + warning messages verbatim; Phase 14 tightens the absence-window read to confirmed-only (the service reads declared+confirmed per D-13-29).

---
*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*

## Self-Check: PASSED

- Created files verified on disk: `internal/core/services/direction/direction.go`, `internal/core/services/direction/direction_test.go`, `13-07-SUMMARY.md`
- All 6 plan commits verified in git log: `fed0ee7`, `2be2b08`, `d28f40e`, `e290668`, `43491ab`, `0e4dc66`
- Task acceptance criteria re-run: `go test ./internal/core/services/direction/ -count=1` → PASS (all 8 suites); `go build ./...` → PASS; `go vet` (direction/testdata/ports) → PASS; full `make test` suite → PASS (518 passes, 0 failures, incl. postgres integration tests)
