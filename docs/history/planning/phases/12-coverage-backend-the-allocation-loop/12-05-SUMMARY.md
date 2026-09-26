---
phase: 12-coverage-backend-the-allocation-loop
plan: 05
subsystem: api
tags: [go, coverage, service, d-04, d-08, d-07, allocation-ledger, hexagonal]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-03 ResolveFundingContext/ResolveBeneficiaryUnit CTEs + FundingContext (activity domain)
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-04 coverage domain package (entity/read-models/sentinels) + 7-method CoverageRepository port + MockCoverageRepo
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 12-06 postgres CoverageRepository (in-tx replace + close) — compile anchor both ways
provides:
  - "coverage service: pure D-04 chain-driven DefaultSource decision function (bucket/service-request/budget draws, absorption, no-source flag) with the D-05 ticket-kind extension seam"
  - "Read paths (Propose/ToCoverQueue/BucketBalance/GetSnapshot/ListHistory) gated manager|finance, queue rows enriched with proposals, no-source flagged (D-06)"
  - "ReplaceAllocations: seven-step invariant (entry scope D-09 → D-K branch → cents Σ fast-fail → mandatory D-08 gate → per-row vocabulary/ref-pinning → org-visible ref resolution → in-tx audit contract) — every violation maps to a sentinel, never 500"
  - "ClosePeriod orchestration: manager-only gate, service-generated close id, coverage-closed audit, repo single-tx close with 409 overlap semantics (A6)"
affects: [12-07, phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (28000) on the same scale.
actuals:
  tokens: 12959    # chars/4 over the realized diff (51835 chars / 4)
  tasks: 2         # tasks completed
  commits: 4       # commits made (RED + GREEN + Task-2 feat + docs metadata)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure decision helper + service wrapper: DefaultSource is repo-free and ctx-free (table-driven unit-tested); the service resolves the chain (FundingContext + beneficiary attach) and wraps with fetch + gate + audit construction"
    - "Mandatory D-08 gate shape: structural self-barrier (owner) → routing.ResolveManagerStage → ApproverIDs path (any role claim) XOR RoleGated && role == 'manager' — no optional role path exists"
    - "Service-side ref visibility gate: contractRepo.Get is not org-scoped (id-only filter, is_adopted subquery), so CreatedByOrgID == orgID || (IsShared && IsAdopted) is the actual org-visibility check — identical predicate to 12-06 BucketBalance"
    - "Cents arithmetic for the Σ invariant (math.Round(h*100)) — the fast-fail path; the repo's in-tx FOR UPDATE re-check stays authoritative (CR-01)"

key-files:
  created:
    - internal/core/services/coverage/coverage.go
    - internal/core/services/coverage/coverage_test.go
  modified:
    - internal/core/domain/activity/activity.go

key-decisions:
  - "FundingContext gained BeneficiaryUnitID *uuid.UUID (Rule 3): the plan-pinned DefaultSource signature takes *activitydomain.FundingContext, but the 12-03 struct carried only contract data — the absorption branch input was missing. The service attaches the resolved unit (ResolveBeneficiaryUnit) to the chain when ContractID is nil; DefaultSource stays pure"
  - "Service signatures follow the plan verbatim: ReplaceAllocations takes userID, role string (parsed to uuid internally — middleware.GetUserID returns uuid.UUID); ClosePeriod takes userID uuid.UUID, role string"
  - "Close audit payload carries the period (period_start/period_end); the plan's 'row count' is unknowable at service time — the repo writes the log as given (12-06 marshals log.Payload as-is) and the row count exists only inside the close tx"
  - "Validation order follows the plan's steps: Σ fast-fail precedes per-row checks, so a lone zero-hours row surfaces ErrAllocationSumMismatch (still 400); the row-level hours>0 check is exercised with a compensating-row test"

patterns-established:
  - "Six-dependency service ctor (repo + activityRepo + contractRepo + unitRepo + entryRepo + *routing.Service) — the routing instance is shared BE-014, never re-implemented"
  - "resolveChain helper: ResolveFundingContext → ResolveBeneficiaryUnit attach when contractless — one assembly point feeding DefaultSource from both Propose and ToCoverQueue"

requirements-completed: [COV-01, COV-02, COV-03, COV-04]

coverage:
  - id: D1
    description: "Pure DefaultSource decision function — the D-04 matrix on all six cases: support contract → bucket draw, project+sold>0 → budget draw, project+sold 0/nil → service-request draw (A3), no contract+beneficiary unit → absorption, neither → no-source flag 'no eligible source — needs a unit or contract' (D-06); D-05 ticket-kind extension seam documented"
    requirement: COV-02
    verification:
      - kind: unit
        ref: "internal/core/services/coverage/coverage_test.go#TestDefaultSource (6 table-driven cases, PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Read paths — Propose (computed proposal + current allocations, cross-org → ErrEntryNotCoverable), ToCoverQueue (proposal enrichment with uncovered hours, no-source rows flagged never dropped), BucketBalance (negative balances returned unchanged, D-03), GetSnapshot/ListHistory passthrough; manager|finance read gate rejects employee/customer with coverage.ErrForbidden"
    requirement: COV-01
    verification:
      - kind: unit
        ref: "internal/core/services/coverage/coverage_test.go#TestService_Propose, TestService_ToCoverQueue, TestService_BucketBalance, TestService_GetSnapshot, TestService_ListHistory (PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ReplaceAllocations seven-step write path — entry fetch + org scope + approved/not-deleted (D-09), D-K entry_type branch, cents Σ fast-fail (empty set → ErrAllocationSumMismatch), mandatory D-08 gate (owner structurally forbidden; ApproverIDs any-role; RoleGated && role == 'manager'; ErrActivityNotLoggable → forbidden), per-row vocabulary/ref-pinning, contract org-visibility (CreatedByOrgID == orgID || IsShared && IsAdopted) + unit same-org refs, full-set audit handed to the repo (BE-016)"
    requirement: COV-03
    verification:
      - kind: unit
        ref: "internal/core/services/coverage/coverage_test.go#TestService_ReplaceAllocations (gate matrix, Σ, D-K, 8-row malformed matrix, 4-case visibility, absorption/transfer refs, audit capture — PASS)"
        status: pass
    human_judgment: false
  - id: D4
    description: "ClosePeriod orchestration — manager-only gate (employee/finance → ErrForbidden, no optional branch), service-generated close id (uuid.New()), coverage-closed audit (entity coverage_allocation, payload = period), repo single-tx close returning the frozen PeriodClose; overlapping close propagates coverage.ErrPeriodAlreadyClosed (409)"
    requirement: COV-04
    verification:
      - kind: unit
        ref: "internal/core/services/coverage/coverage_test.go#TestService_ClosePeriod (PASS)"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 05: Coverage Service — The Allocation Loop Summary

**The coverage service: the pure D-04 chain-driven proposal decision function (bucket/service-request/budget draws, absorption, no-source flag), manager|finance read paths (propose, to-cover queue enrichment, bucket balance, snapshot, history), the seven-step ReplaceAllocations write with the cents Σ invariant + per-row vocabulary/ref-pinning + the mandatory D-08 gate (ApproverIDs OR RoleGated && role == "manager"), and the manager-only ClosePeriod orchestration — all unit-tested against the hermetic mock with the audit contract asserted by capture**

## Performance

- **Duration:** 7 min
- **Started:** 2026-08-08T09:43:20Z
- **Completed:** 2026-08-08T09:50:08Z
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- **`DefaultSource` — the D-04 heart, pure and table-driven**: one chain-driven function maps the entry's funding chain to a proposal with no billability-first branch (Pitfall 3 closed): support contract → bucket draw; project with sold > 0 → budget draw; project with sold nil-or-0 (A3, `IS NOT DISTINCT FROM 0`) → service-request draw; no contract + beneficiary unit → absorption; neither → the D-06 flag `"no eligible source — needs a unit or contract"`. The D-05 ticket-kind extension seam is documented as the single future switch point — no kind→source matrix implemented. The service keeps it pure via `resolveChain` (ResolveFundingContext + ResolveBeneficiaryUnit attach when contractless).
- **Read paths (manager|finance, D-L)**: `Propose` returns the computed proposal + the entry's current allocations (cross-org → `ErrEntryNotCoverable`); `ToCoverQueue` enriches every row with its D-04 proposal (hours = uncovered hours) and flags — never drops — no-source rows; `BucketBalance` passes through the derived balance with negatives returned as-is (D-03); `GetSnapshot`/`ListHistory` passthrough. Employee/customer read attempts → `coverage.ErrForbidden`.
- **`ReplaceAllocations` — every user-observable invariant decided here**: entry scope + approved/not-deleted (D-09) → D-K `entry_type` branch (the costed polymorphic validation) → cents Σ fast-fail (empty set included) → the mandatory D-08 gate (structural self-barrier first; then ApproverIDs path in any role claim exactly as Approve; the RoleGated terminal branch REQUIRES the org role claim `"manager"` — no optional role path exists, T-12-09) → per-row vocabulary/ref-pinning mapped to `ErrInvalidRequest` (400 — check_violation 23514 is unmapped in wrapPGError, so the 500 path is unreachable from the service, T-12-14) → ref resolution with the contract org-visibility predicate (`CreatedByOrgID == orgID || (IsShared && IsAdopted)` — Get itself is not org-scoped, T-12-10) and unit same-org compare → full-set audit handed to the repo for the in-tx write (BE-016, T-12-13). Fast-fail only: the repo's FOR UPDATE in-tx re-check (12-06) stays authoritative (CR-01, T-12-11).
- **`ClosePeriod` — snapshot orchestration, not lock**: manager-only write gate (the period is org-scoped with no entry chain — pure role claim, no resolution branch), service-generated close id, coverage-closed audit payload with the period, and the repo's single-tx close returning the frozen `PeriodClose` incl. rows (OQ4). Overlap → `coverage.ErrPeriodAlreadyClosed` (409 at the handler, A6). Allocations stay editable indefinitely (D-F snapshot-not-lock).

## Task Commits

Each task was committed atomically:

1. **Task 1 (TDD RED): add failing tests for D-04 default source and read paths** - `75f1d8e` (test)
2. **Task 1 (TDD GREEN): implement D-04 default source and coverage read paths** - `d30464e` (feat)
3. **Task 2: implement replace-allocation write gate and close orchestration** - `663ae68` (feat)

**Plan metadata:** pending (this docs commit)

## TDD Gate Compliance

Task 1 carried `tdd="true"` (plan type `execute` — plan-level TDD gate does not apply). The task-level RED/GREEN sequence is intact: `75f1d8e` (test) precedes `d30464e` (feat). RED failed for the right reason — the `coveragesvc` package did not exist (undefined symbols) plus the missing `FundingContext.BeneficiaryUnitID` field. GREEN passes the full package. No REFACTOR commit — the GREEN implementation needed no cleanup.

## Files Created/Modified

- `internal/core/services/coverage/coverage.go` - Service struct + NewService (six deps incl. shared `*routing.Service`); pure `DefaultSource` (D-04 + D-05 seam); `resolveChain` helper; Propose/ToCoverQueue/BucketBalance/GetSnapshot/ListHistory (manager|finance read gate); ReplaceAllocations (seven steps); ClosePeriod; `validAbsorptionReason` + `contains` helpers
- `internal/core/services/coverage/coverage_test.go` - fixture wiring the service against MockCoverageRepo/MockActivityRepo/MockContractRepo/MockUnitRepo/MockTimeEntryRepo + real routing.Service over the mock repos; TestDefaultSource (6 cases), 5 read-path tests, gate-matrix/Σ/D-K/malformed-row/visibility/audit tests for ReplaceAllocations, ClosePeriod gate + audit + 409 tests
- `internal/core/domain/activity/activity.go` - `FundingContext` gained `BeneficiaryUnitID *uuid.UUID` (Rule 3 deviation, below)

## Decisions Made

- **FundingContext carries the beneficiary unit** — the plan-pinned `DefaultSource(chain *activitydomain.FundingContext)` needs the absorption input; extending the 12-03 struct (additive pointer field, nil when absent) keeps the decision function pure while the service attaches the resolved unit service-side. The postgres adapter's `ResolveFundingContext` (12-03) is untouched — it never sets the new field; the service fills it from `ResolveBeneficiaryUnit`.
- **Write-gate shape per Pattern 6, verbatim**: `!res.RoleGated && !contains(res.ApproverIDs, userID)` → forbidden; `res.RoleGated && role != "manager"` → forbidden. The ApproverIDs path accepts the WG manager/delegate in any role claim — exactly as they may approve — while the RoleGated branch (org root without a unit manager) is strictly the org role claim `"manager"`.
- **Validation order = plan step order**: Σ fast-fail (step 3) precedes per-row checks (step 5); both map to 400-class sentinels so no ordering can reach the unmapped 23514 500 path.
- **Close audit payload = the period** — row count is computed inside the repo's close tx and cannot be known when the service builds the log; 12-06 marshals the log as given (verified `coverage_repository.go` insertCoverageAudit).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] FundingContext lacked the beneficiary-unit field the pinned DefaultSource signature requires**
- **Found during:** Task 1 RED run (compile: `unknown field BeneficiaryUnitID in struct literal of type activity.FundingContext`)
- **Issue:** The plan pins `DefaultSource(chain *activitydomain.FundingContext)` and requires the absorption branch ("no contract: BeneficiaryUnitID set → absorption draw with that unit"), but the 12-03 `FundingContext` struct carries only `ContractID/ContractType/SoldHours` — the absorption decision input was missing from the chain type.
- **Fix:** Added `BeneficiaryUnitID *uuid.UUID` (omitempty, additive, nil-safe) to `domain/activity.FundingContext`; the service's `resolveChain` attaches the `ResolveBeneficiaryUnit` result to the chain when `ContractID == nil`. DefaultSource stays pure and the six-case matrix is table-driven.
- **Files modified:** `internal/core/domain/activity/activity.go`
- **Verification:** `go test ./internal/core/services/coverage/ -count=1` (6/6 DefaultSource cases incl. absorption) + `go build ./...` green.
- **Committed in:** `d30464e` (Task 1 GREEN commit)

**2. [Rule 1 - Bug] Close audit "row count" is unknowable at service time**
- **Found during:** Task 2 (ClosePeriod implementation)
- **Issue:** The plan's ClosePeriod text says the audit payload is "period + row count", but the audit log must be built and handed to the repo BEFORE the close tx runs; the row count is computed inside the tx and 12-06's `insertCoverageAudit` writes the payload as given (no augmentation — verified `coverage_repository.go:48-65`).
- **Fix:** Payload carries the closed period (`period_start`/`period_end`, `2006-01-02` DATE format). The snapshot's own `Rows` return (OQ4) still surfaces the row count to the caller.
- **Files modified:** `internal/core/services/coverage/coverage.go`
- **Verification:** ClosePeriod audit-capture test asserts entity/action/actor/period payload.
- **Committed in:** `663ae68` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug-adjustment)
**Impact on plan:** Both were required to make the pinned contracts work — the FundingContext field is the decision input the absorption branch needs; the payload adjustment is a plan detail the 12-06 repo contract makes impossible (log built pre-tx). No scope creep; all six must-have truths and threat mitigations (T-12-09/10/13/14) are implemented and tested.

## Issues Encountered

- **Malformed-row test ordering nuance**: the plan's step order puts the cents Σ fast-fail before per-row validation, so a lone `hours=0` row surfaces `ErrAllocationSumMismatch` (still a 400) rather than `ErrInvalidRequest`. The row-level `hours > 0` check is exercised with a compensating second row so Σ stays equal — both sentinels are asserted, no 500 path exists either way.
- **Mock ClosePeriod id semantics**: the mock's primed `ClosePeriodResult` keeps its own ID while the service generates a fresh close id, so the audit-EntityID assertion uses the mock's default path (snapshot keyed by the passed closeID) — which is exactly the contract 12-06 implements (header id = caller-supplied closeID).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The coverage service is complete and unit-tested against the mock: 12-07 (HTTP handlers) compiles against `coveragesvc.NewService` and maps the sentinels (404/400/403/409) to the envelope — the service surface and gate semantics are pinned by this plan's tests.
- 12-06's postgres repo already implements the seven-port contract; the only integration risk left is the handler wiring in `cmd/server/main.go` (12-07), which reuses the shared `routing.Service` instance.
- `FundingContext.BeneficiaryUnitID` is an additive contract extension — no consumer of the 12-03 CTEs is affected (the postgres adapter never sets it; the service fills it).

## Self-Check: PASSED

- [x] `internal/core/services/coverage/coverage.go` exists — Service + NewService + DefaultSource (pinned 5-value signature) + 7 methods + D-05 seam doc
- [x] `internal/core/services/coverage/coverage_test.go` exists (730+ lines, 10 test funcs)
- [x] `internal/core/domain/activity/activity.go` — BeneficiaryUnitID field on FundingContext confirmed
- [x] Commit `75f1d8e` (test RED), `d30464e` (feat GREEN), `663ae68` (feat Task 2) present in git log
- [x] `go test ./internal/core/services/coverage/ -count=1` — ok
- [x] Task verifies: `-run 'TestDefaultSource|TestService_Propose|TestService_ToCoverQueue|TestService_BucketBalance'` and `-run 'TestService_ReplaceAllocations|TestService_ClosePeriod'` — all PASS
- [x] `go build ./...` exit 0; `go vet ./internal/core/services/coverage/ ./internal/core/domain/activity/` clean
- [x] `make test` exit 0 (full suite incl. testcontainers)

---
*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*
