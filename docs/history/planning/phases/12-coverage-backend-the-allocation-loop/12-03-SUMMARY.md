---
phase: 12-coverage-backend-the-allocation-loop
plan: 03
subsystem: api
tags: [go, activity, beneficiary-unit, coverage, hexagonal, postgres, cte, coa-05]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop
    provides: 018/019 schema beneficiary_unit_id column (12-01), ADR-BE-017 funding vocabulary (12-02)
provides:
  - "beneficiary_unit_id end-to-end on the activity stack: domain field + request DTOs, port methods, postgres read/write (baseActivityQuery/GetAncestry/scanActivity/Update), service same-org validation, handler boundary parse"
  - "ResolveBeneficiaryUnit CTE — nearest ancestor's unit walking upward while NULL (COV-05 absorption default)"
  - "ResolveFundingContext CTE — nearest ancestor contract + contract_type/sold_hours via contracts JOIN (D-04 decision input)"
  - "FundingContext chain-data type in domain/activity consumed by plan 12-05's coverage service"
affects: [12-05, 12-06, phase-17-surfaces]

# Actuals (#2632) — pairs with the plan's `estimate` (25000) on the same scale.
actuals:
  tokens: 10760    # chars/4 over the realized diff (43042 chars / 4)
  tasks: 2         # tasks completed
  commits: 3       # commits made (2 task commits + 1 docs metadata)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Recursive CTE resolver shape (ResolveCommercialContext analog): seed row, walk parent_id upward with a NULL guard, return first non-null — one resolver per walked column"
    - "Derived chain-data type in domain/activity (FundingContext) returned through the port as the coverage decision input — never stored"
    - "Editable non-origin field: beneficiary_unit_id gets an Update SET branch and re-validated same-org on every write, while hasOriginFields stays origin-only"
    - "Same-org fetch-and-compare via unitRepo.GetByID then u.OrgID == orgID — the expense-service pattern, applied at both service entry points (T-12-06)"

key-files:
  created:
    - internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go
    - internal/core/services/activity/activity_beneficiary_unit_test.go
  modified:
    - internal/core/domain/activity/activity.go
    - internal/core/ports/activity_repository.go
    - internal/adapters/secondary/postgres/activity_repository.go
    - internal/core/services/activity/activity.go
    - internal/adapters/primary/http/activity_handler.go
    - internal/core/services/testdata/mocks.go

key-decisions:
  - "FundingContext lives in domain/activity (chain data belongs to the activity domain, per plan assumption) — ContractID/ContractType/SoldHours all pointer, all nil when the chain has no contract; coverage service (12-05) consumes it via the port"
  - "Two separate CTE resolvers, each with its own NULL-walk guard: ResolveBeneficiaryUnit walks beneficiary_unit_id (absorption default), ResolveFundingContext walks contract_id with the contracts JOIN for contract_type/sold_hours (D-04 input) — never merged into one walk"
  - "beneficiary_unit_id is editable on Update (unlike origin refs): the repo SET builder gains a branch mirroring ContractID and the service re-runs the same-org check on every write; hasOriginFields untouched"
  - "Same-org rejection uses ErrInvalidRequest on OrgID mismatch (caller maps to 400), surfacing the unitRepo fetch error as-is — the flagged_assumption pattern from the plan"

patterns-established:
  - "Pitfall-5 discipline: a column added to baseActivityQuery/GetAncestry lands in all three explicit SELECTs + scanActivity in one commit, proven by an inheritance integration test"
  - "Integration test self-seeds inline (seedOrg/seedActivityKind/seedUnit/seedActivity + direct SQL contract insert) — never seed_demo.sql"

requirements-completed: [COV-05]

coverage:
  - id: D1
    description: "beneficiary_unit_id round-trip through the activity stack — domain field + Create/Update request DTOs, port method additions, postgres baseActivityQuery/GetAncestry/scanActivity/Update touch points, service same-org validation on Create+Update, handler *string → *uuid.UUID boundary parse with 400 on malformed input, MockActivityRepo support"
    requirement: COV-05
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go#TestActivityBeneficiaryUnit_CreateGetReadBack, TestActivityBeneficiaryUnit_UpdateEditable (PASS, go test -count=1)"
        status: pass
      - kind: unit
        ref: "internal/core/services/activity/activity_beneficiary_unit_test.go#TestService_Create_BeneficiaryUnit, TestService_Update_BeneficiaryUnit (PASS)"
        status: pass
      - kind: other
        ref: "go build ./... (exit 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "ResolveBeneficiaryUnit CTE — nearest ancestor's beneficiary_unit_id walking parent_id upward while NULL; nil when the chain has none (COV-05 absorption default)"
    requirement: COV-05
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go#TestResolveBeneficiaryUnit_Inheritance, TestResolveBeneficiaryUnit_NoUnitNil (PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ResolveFundingContext CTE — nearest ancestor contract_id plus contract_type and sold_hours via the contracts JOIN; all-NULL when the chain has no contract (the D-04 decision input for 12-05)"
    verification:
      - kind: integration
        ref: "internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go#TestResolveFundingContext_ContractAttrs, TestResolveFundingContext_NoContractNil (PASS)"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-08-08
status: complete
---

# Phase 12: Plan 03 — Beneficiary Unit Tracer Summary

**beneficiary_unit_id end-to-end on the activity stack (COV-05): nullable domain field + request DTOs, port-resolver pair (ResolveBeneficiaryUnit for the absorption default, ResolveFundingContext with contract_type/sold_hours for D-04), postgres read/write across all five touch points including GetAncestry's explicit column list (Pitfall 5 closed), same-org service validation on Create+Update, handler boundary parse, and 8 new integration/unit tests proving round-trip, editability, and downward inheritance.**

## Performance

- **Duration:** 12 min (bookkeeping run; implementation commits 11:16–11:20Z prior, truncated before summary)
- **Started:** 2026-08-08T09:04:52Z (commit timestamps 11:16:58+0200 / 11:20:52+0200)
- **Completed:** 2026-08-08T09:24:11Z
- **Tasks:** 2
- **Files modified:** 8 (6 modified, 2 created)

## Accomplishments

- Beneficiary unit field wired through the entire activity stack: `Activity.BeneficiaryUnitID *uuid.UUID` (+ Create/Update request pointers, `omitempty`), mirroring contract_id's nullable-inheritance semantics (D-B, COV-05)
- Both coverage-service resolvers live with proven semantics: `ResolveBeneficiaryUnit` (nearest ancestor's unit, nil when none — absorption default) and `ResolveFundingContext` (nearest ancestor contract + contract_type/sold_hours via contracts JOIN — the D-04 decision input)
- Pitfall 5 closed: `beneficiary_unit_id` landed in all three explicit SELECTs of GetAncestry's column list plus baseActivityQuery and scanActivity in a single commit, with an inheritance integration test guarding against silent-nil absorption proposals
- Same-org security gate (T-12-06): fetch-and-compare via `unitRepo.GetByID` + `u.OrgID == orgID` on both Create and Update; handler parses `*string` → `*uuid.UUID` with 400 `invalid beneficiary_unit_id`
- Beneficiary unit stays editable (unlike origin refs): Update SET builder branch mirrors ContractID; `hasOriginFields` untouched

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): End-to-end beneficiary unit — column field, port resolvers, repo read/write + CTE walks** - `86434f8` (feat)
2. **Task 2 (auto): Expansion: service same-org validation + handler DTO parse + mock field + tests** - `d30f336` (feat)

**Plan metadata:** pending (this bookkeeping commit)

## Files Created/Modified

- `internal/core/domain/activity/activity.go` - `BeneficiaryUnitID *uuid.UUID` on Activity + Create/UpdateActivityRequest (omitempty, nullable = internal work inheritance); new `FundingContext{ContractID, ContractType, SoldHours}` struct with doc comment
- `internal/core/ports/activity_repository.go` - `ResolveBeneficiaryUnit(ctx, activityID) (*uuid.UUID, error)` and `ResolveFundingContext(ctx, activityID) (*activitydomain.FundingContext, error)` with doc comments stating chain semantics
- `internal/adapters/secondary/postgres/activity_repository.go` - `a.beneficiary_unit_id` in baseActivityQuery; GetAncestry seed/recursive/final SELECTs; scanActivity + scanActivityResponse scan slots; Update dynamic SET branch; two WITH RECURSIVE CTE resolvers (NULL-walk guards + contracts LEFT JOIN)
- `internal/core/services/activity/activity.go` - same-org fetch-and-compare blocks in Create (after ContractID check) and Update (before repo update); hasOriginFields unchanged
- `internal/adapters/primary/http/activity_handler.go` - `BeneficiaryUnitID *string` on both DTOs; parse blocks returning 400 `invalid beneficiary_unit_id` on malformed UUID
- `internal/core/services/testdata/mocks.go` - MockActivityRepo `BeneficiaryUnitID` field honored on create/update for service-test round-trips
- `internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go` - 6 integration tests (testcontainers, self-seed inline): create/get read-back, update editable, inheritance walk, nil-when-none, funding-context attrs via direct-SQL contract insert, no-contract nil
- `internal/core/services/activity/activity_beneficiary_unit_test.go` - 2 service tests exercising same-org validation paths through the mock

## Decisions Made

- `FundingContext` defined in `domain/activity` — chain data belongs to the activity domain; the coverage service (12-05) consumes it through the port, never stored
- Two separate CTE resolvers with independent NULL-walk guards — resolving the beneficiary unit never short-circuits the contract walk and vice versa (plan flagged_assumption honored)
- Same-org violation returns `ErrInvalidRequest` (→ 400); the unitRepo fetch error surfaces as-is for the caller to map — the plan's flagged pattern
- Editable semantics: Update SET branch + re-validation on every write; `hasOriginFields` stays origin-only (plan prohibition honored)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - the prior executor completed both commits and all acceptance criteria; this run only performed verification (build + targeted + full suite) and bookkeeping.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `ResolveBeneficiaryUnit` + `ResolveFundingContext` port signatures are now fixed — plan 12-05 (coverage service) compiles against them for the absorption default and the D-04 contract-vs-bucket-vs-service-request decision
- COV-05 shared-ID note: plan 12-03 declares COV-05 and its SUMMARY now exists, but the shared-ID gate (sibling plans declaring COV-05 without a SUMMARY) may still hold mark-complete — do not force; the last declaring sibling's completion flips it
- Integration tests prove the CTE shapes against a live Postgres (testcontainers) — 12-05/12-06 repo work can build on the same patterns

## Self-Check: PASSED

- [x] `internal/core/domain/activity/activity.go` exists — BeneficiaryUnitID + FundingContext confirmed in diff
- [x] `internal/core/ports/activity_repository.go` exists — both resolver declarations confirmed in diff
- [x] `internal/adapters/secondary/postgres/activity_beneficiary_unit_test.go` exists (227 lines, 6 test funcs)
- [x] `internal/core/services/activity/activity_beneficiary_unit_test.go` exists (106 lines, 2 test funcs)
- [x] Commit `86434f8` present in git log
- [x] Commit `d30f336` present in git log
- [x] `go build ./...` exit 0
- [x] `go test ./internal/core/services/activity/ ./internal/adapters/primary/http/ -count=1` ok
- [x] `go test ./internal/adapters/secondary/postgres/ -run 'TestActivityBeneficiaryUnit|TestResolveBeneficiaryUnit|TestResolveFundingContext' -count=1` — 6/6 PASS
- [x] `go test ./... -count=1` — full suite green

---
*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*
