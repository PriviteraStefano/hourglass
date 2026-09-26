---
phase: 11-foundations-schema-origins-tickets-backend
plan: 04
subsystem: api
tags: [contracts, sold-hours, go, postgres, fnd-03]

# Dependency graph
requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: migration 016 (contracts contract_type/sold_hours/sold_period + contracts_sold_check) + baseContractQuery/scanContractResponse/Update dynamic-SET pattern
provides:
  - contract_type/sold_hours/sold_period read/write end-to-end through the existing contract endpoints (FND-03)
  - validateSoldConfig service semantics (D-08) on Create + Update with ErrInvalidSoldConfig sentinel
  - DB CHECK backstop verified at the repo layer (contracts_sold_check 23514)
affects: [12-funding-sources, 13-direction, 24-customers-contracts-polish]

# Tech tracking
tech-stack:
  added: [none — existing Go/pgx/testify/testcontainers stack]
  patterns:
    - "Nullable sold-period clear via empty-string sentinel (customer_id nullable-clear pattern reuse)"
    - "contract_type never NULLed from an absent update field — set-once semantics preserved"
    - "Dynamic SET branches follow the established *float64 block; all values parameterized (ADR-BE-003)"

key-files:
  created: []
  modified:
    - internal/core/domain/contract/contract.go
    - internal/core/services/contract/contract.go
    - internal/core/services/contract/contract_test.go
    - internal/adapters/secondary/postgres/contract_repository.go
    - internal/adapters/secondary/postgres/contract_repository_test.go
    - internal/adapters/primary/http/contract.go

key-decisions:
  - "Sold-period clear branch uses the empty-string sentinel (sold_period: \"\") mirroring the existing customer_id nullable-clear pattern; absent field never emits NULL"
  - "Update validates only fields present in the request; when contract_type='support' arrives without hours/period the service rejects with the clean ErrInvalidSoldConfig before the DB CHECK can fire (house style: surface the sentinel first)"
  - "sold_hours has no update-clear branch (plan-scoped): it can only be set via update, matching the plan's nullable-clear scope of sold_period only"

patterns-established:
  - "Service semantic validation + DB CHECK double-enforcement for sold-hours config (T-11-03)"

requirements-completed: [FND-03]

# Metrics
duration: 7min
completed: 2026-08-07
---

# Phase 11 Plan 4: Contract Sold-Hours Read/Write Summary

**contract_type/sold_hours/sold_period wired end-to-end through the existing contract endpoints (FND-03): domain fields + D-08 service validation on Create/Update + repo SELECT/INSERT/UPDATE persistence + handler pass-through, with the contracts_sold_check DB backstop proven at the repo layer**

## Performance

- **Duration:** 7 min
- **Started:** 2026-08-07T10:15:00Z
- **Completed:** 2026-08-07T10:21:51Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- **Domain carries the three fields** on `Contract`, `CreateContractRequest`, and `UpdateContractRequest` with the constants (`ContractTypeProject`/`ContractTypeSupport`, `SoldPeriodMonth/Quarter/Year`) and `ErrInvalidSoldConfig` sentinel.
- **`validateSoldConfig` enforces D-08 semantics on BOTH Create and Update**: NULL type (legacy, D-16) allowed; `support` requires sold_hours + sold_period; `project` forbids sold_period; closed type/period sets. Called in Create after decode and in Update after the finance role gate.
- **Repository persistence**: `baseContractQuery` SELECT + `scanContractResponse` nullable locals + Create INSERT + Update dynamic SET with the nullable-clear branch for sold_period (empty-string sentinel) and set-once semantics for contract_type. All values parameterized (ADR-BE-003, T-11-09).
- **Handler pass-through**: both `CreateContractRequest` and `UpdateContractRequest` DTOs carry the three fields and forward them to the domain requests.
- **Tests**: 10 new/expanded service unit cases (Create + Update, rejections and valid configs incl. legacy NULL type); 3 new integration tests — support round-trip through Get, NULL→support upgrade in one update, support→project transition clearing sold_period, and raw-insert DB CHECK rejection asserted with SQLSTATE 23514 / `contracts_sold_check` (T-11-03).

## Task Commits

Each task was committed atomically:

1. **Task 1: Contract domain fields + service validation** - `89208be` (feat)
2. **Task 2: Repository persistence + handler pass-through** - `a9353fb` (feat)

## Files Created/Modified

- `internal/core/domain/contract/contract.go` - Contract + both request DTOs carry ContractType/SoldHours/SoldPeriod; ErrInvalidSoldConfig; ContractType/SoldPeriod constants
- `internal/core/services/contract/contract.go` - validateSoldConfig called from Create + Update (D-08 semantics)
- `internal/core/services/contract/contract_test.go` - 10 new cases across Create/Update (support-without-period, project-with-period, unknown type, bad period, legacy NULL type allowed, valid support/project)
- `internal/adapters/secondary/postgres/contract_repository.go` - SELECT/INSERT/UPDATE/scan extended with the three columns; sold_period nullable-clear branch; contract_type set-once
- `internal/adapters/secondary/postgres/contract_repository_test.go` - 3 new integration tests (round-trip, NULL→support upgrade, support→project + period clear, DB CHECK raw-insert rejection)
- `internal/adapters/primary/http/contract.go` - both DTOs + Create/Update pass-through

## Decisions Made

- Sold-period clear uses the empty-string sentinel exactly like the existing customer_id nullable-clear branch — the pattern is already established in this repo, no new mechanism needed.
- Update-side validation operates on fields present in the request only; the support-without-period case surfaces the clean `ErrInvalidSoldConfig` sentinel before the DB CHECK would fire on the row.
- `sold_hours` intentionally has no clear branch (only sold_period) per the plan's explicit scope; hours can only be set on update.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- One compile error in the new test struct (`req` field missing from `TestService_Update` table) — fixed immediately during Task 1 test-writing; no impact.
- `contract_integration_test.go` (pre-existing, untouched by this plan) remains non-gofmt-clean — logged as out-of-scope discovery, not fixed per the scope boundary rule.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Contract read/write carries contract_type/sold_hours/sold_period end-to-end (FND-03) — ready for 11-05 (origins/proposals) and 11-06 (tickets) which compile against the same contract repo/service.
- Phase 12 funding sources can now read sold config via the standard contract read models.
- Full suite green: `make test` passes all 20 packages, 0 failures.

## Self-Check: PASSED

- Modified files verified on disk: all 6 files present with the expected content (domain fields, validateSoldConfig, repo columns, handler pass-through, tests).
- Commits verified in git log: 89208be (Task 1 feat), a9353fb (Task 2 feat).
- Verification commands: `go test ./internal/adapters/secondary/postgres/ -run TestContract -count=1` ok; `go test ./internal/core/services/contract/ -count=1` ok; `make test` 0 failures.

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
