---
phase: 14-availability-backend-absences-capacity
plan: 11
subsystem: api
tags: [go, postgres, error-handling, availability, contract-types, sqlstate]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity
    provides: contract_types DECIMAL(5,2) column (migration 024), wrapPGError 23505/23503 switch, availability writeError sentinel map
provides:
  - validateContractType 999.99 ceiling — hours_per_period ≥ 1000 400s at the service boundary
  - wrapPGError SQLSTATE 22003 → ports.ErrInvalidRequest (adapter boundary)
  - availability writeError maps ports.ErrInvalidRequest → 400 (last line)
affects: [availability-backend-absences-capacity, direction, coverage, time-entries]

# Actuals (#2632) — pairs with the plan's estimate (14000 tokens)
actuals:
  tokens: 4429     # chars/4 over the realized diff (17715 chars)
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SQLSTATE-class mapping at the adapter boundary: numeric_value_out_of_range (22003) → ports.ErrInvalidRequest (the 23505/23503 sentinel family extended)"
    - "Service-side ceiling mirrors the column's DECIMAL max exactly (999.99) so client input 400s before PG can overflow (WindowHoursValid 99.99 precedent)"

key-files:
  created:
    - internal/adapters/secondary/postgres/postgres_test.go
  modified:
    - internal/core/services/availability/contract_types.go
    - internal/core/services/availability/contract_types_test.go
    - internal/adapters/primary/http/contract_types_handler_test.go
    - internal/core/ports/errors.go
    - internal/adapters/secondary/postgres/postgres.go
    - internal/adapters/primary/http/availability_handler.go

key-decisions:
  - "WrapPGError extended with a 22003 → ports.ErrInvalidRequest case despite the Phase 13 13-09 scope note ('wrapPGError untouched') — the verifier (WR-03) mandates the mapping; effect on unrelated repos (time entries, coverage) is strictly 500→400 on numeric overflow, the house rule's direction; no existing test asserts a 500 on 22003"
  - "Boundary value 999.99 accepted (not rejected): the ceiling check is strictly greater-than, mirroring the DECIMAL(5,2) column max exactly"

patterns-established:
  - "Client-input numeric ceilings are validated in the service (fast-fail before any repo call), and the adapter still maps the overflow SQLSTATE as belt-and-braces so the surface can never 500"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "validateContractType rejects hours_per_period above the DECIMAL(5,2) ceiling (1000 → ErrInvalidRequest) and accepts the 999.99 boundary"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: internal/core/services/availability/contract_types_test.go#TestContractTypes_Validation/hours_above_the_DECIMAL(5,2)_ceiling
        status: pass
      - kind: unit
        ref: internal/core/services/availability/contract_types_test.go#TestContractTypes_Validation/ceiling_boundary_accepted
        status: pass
    human_judgment: false
  - id: D2
    description: "POST /availability/contract-types with hours_per_period 1000 returns HTTP 400, never 500"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: internal/adapters/primary/http/contract_types_handler_test.go#TestContractTypesHandler/hours_above_the_DECIMAL(5,2)_ceiling_returns_400_(WR-03)
        status: pass
    human_judgment: false
  - id: D3
    description: "wrapPGError maps SQLSTATE 22003 to ports.ErrInvalidRequest; unknown SQLSTATE (23514) passes through unmapped"
    requirement: AVAIL-02
    verification:
      - kind: unit
        ref: internal/adapters/secondary/postgres/postgres_test.go#TestWrapPGError_22003
        status: pass
    human_judgment: false
  - id: D4
    description: "availability writeError maps ports.ErrInvalidRequest to HTTP 400 — the belt-and-braces last line"
    verification:
      - kind: unit
        ref: go build ./... (availability_handler.go ports import + 400 case compiles)
        status: pass
    human_judgment: false

# Metrics
duration: 32min
completed: 2026-08-12
status: complete
---

# Phase 14 Plan 11: WR-03 Gap Closure — No 500 for Client Input on /availability Summary

**DECIMAL(5,2) ceiling in validateContractType (1000 → 400 at service + HTTP layers), SQLSTATE 22003 → ports.ErrInvalidRequest adapter mapping, and the availability writeError 400 case — the last known 500 path on the /availability surface is closed**

## Performance

- **Duration:** 32 min
- **Started:** 2026-08-12T10:58:00Z
- **Completed:** 2026-08-12T11:30:00Z
- **Tasks:** 2 (both TDD: RED → GREEN)
- **Files modified:** 7 (1 created, 6 modified)

## Accomplishments

- `validateContractType` now rejects `hours_per_period > 999.99` with `ErrInvalidRequest` — the migration 024 `DECIMAL(5,2)` max mirrored exactly at the service boundary; client input 400s before PG is ever reached (T-14g-30 mitigated)
- HTTP regression: `POST /availability/contract-types` with `hours_per_period: 1000` returns 400, never 500 (was 500 via unwrapped PG numeric overflow)
- `wrapPGError` gains the `case "22003"` → `ports.ErrInvalidRequest` mapping (T-14g-31 mitigated) with a pure-unit test (`TestWrapPGError_22003`, no testcontainer: 22003 mapped, unknown 23514 passes through)
- `ports.ErrInvalidRequest` sentinel added (additive); the availability handler's `writeError` maps it to 400 — the surface can no longer 500 on numeric overflow
- Boundary acceptance pinned: 999.99 accepted on an otherwise-valid type

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1: validateContractType 999.99 ceiling + service/HTTP regressions (WR-03 first half)**
   - `24fbb91` (test): add failing test for DECIMAL(5,2) ceiling in contract types
   - `a7e9ed9` (feat): enforce DECIMAL(5,2) ceiling in validateContractType
2. **Task 2: wrapPGError 22003 mapping + writeError case (WR-03 second half, belt-and-braces)**
   - `65de685` (test): add failing test for wrapPGError 22003 mapping
   - `c1e5f3e` (feat): map SQLSTATE 22003 to 400 via ports sentinel

**Plan metadata:** (pending final commit)

## Files Created/Modified

- `internal/core/services/availability/contract_types.go` - `validateContractType` +999.99 ceiling check with WR-03 doc comment
- `internal/core/services/availability/contract_types_test.go` - `TestContractTypes_Validation` table gains `expected error` field, the 1000-reject case, and the 999.99 boundary-acceptance case
- `internal/adapters/primary/http/contract_types_handler_test.go` - new subtest: hr POST with `hours_per_period: 1000` → 400 (WR-03)
- `internal/core/ports/errors.go` - `ErrInvalidRequest` sentinel added (additive)
- `internal/adapters/secondary/postgres/postgres.go` - `wrapPGError` +`case "22003"` → `ports.ErrInvalidRequest`; doc comment extended
- `internal/adapters/secondary/postgres/postgres_test.go` - NEW: `TestWrapPGError_22003` (pure unit: 22003 mapped, 23514 passthrough, no container)
- `internal/adapters/primary/http/availability_handler.go` - `writeError` 400 branch +`ports.ErrInvalidRequest`; ports import; sentinel-map doc comment updated

## Decisions Made

- **Extended `wrapPGError` beyond the Phase 13 13-09 scope note** ("wrapPGError untouched — the 22003 path was unreachable for client input there"). The verifier mandates the mapping; the effect on unrelated repos (time entries, coverage) is strictly 500→400 on numeric overflow — the house rule's direction — and no existing test asserts a 500 on 22003. Flagged and recorded per the plan's NOTE.
- **Boundary 999.99 accepted** — the ceiling check is strictly greater-than, mirroring the column's max exactly (consistent with the `WindowHoursValid` 99.99-ceiling precedent for day hours).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## TDD Gate Compliance

RED/GREEN sequence verified in git log:

| Gate | Commit | Status |
|------|--------|--------|
| RED (Task 1) | `24fbb91` test(14-11): add failing test for DECIMAL(5,2) ceiling in contract types | ✓ failed for the right reason (service accepted 1000; HTTP returned 500) |
| GREEN (Task 1) | `a7e9ed9` feat(14-11): enforce DECIMAL(5,2) ceiling in validateContractType | ✓ both service + HTTP tests pass |
| RED (Task 2) | `65de685` test(14-11): add failing test for wrapPGError 22003 mapping | ✓ failed for the right reason (sentinels/mapping absent) |
| GREEN (Task 2) | `c1e5f3e` feat(14-11): map SQLSTATE 22003 to 400 via ports sentinel | ✓ unit test + handler batteries + build pass |

## Verification Results

- `go test ./internal/core/services/availability/ -run TestContractTypes -count=1` → **PASS** (ceiling table cases)
- `go test ./internal/adapters/primary/http/ -run 'TestContractTypesHandler|TestAvailabilityHandler' -count=1` → **PASS** (HTTP 400 regression + full surface battery)
- `go test ./internal/adapters/secondary/postgres/ -run TestWrapPGError -count=1` → **PASS** (adapter mapping unit test)
- `go build ./...` → **PASS** (repo-wide compile)
- Full postgres package suite → **PASS** (47s, wrapPGError change regression-free)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- WR-03 fully closed: the /availability surface has no remaining known 500 path for client input (service ceiling + adapter 22003 mapping + handler sentinel)
- T-14g-30 (DoS via numeric overflow) and T-14g-31 (availability on overflow) both mitigated per the plan's threat model
- Phase 14 remaining plans: 14-10 (in progress elsewhere)
- Available for phase verification (14-VERIFICATION.md WR-03 row can flip to VERIFIED)

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-12*
