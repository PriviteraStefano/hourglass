---
phase: 04-contracts
plan: 01
subsystem: backend
tags: [contracts, customer, CRUD, delete-protection, backend]
requires:
  - phase: 03-01
    provides: Customer domain, internal customer concept
  - phase: 03-02
    provides: Customer frontend API, detail page
provides:
  - customer_id on CreateContractRequest (domain, handler, PG INSERT)
  - HasProjects on ContractRepository (interface, mock, PG implementation)
  - Delete protection: HasProjects check in Delete service method
  - ErrHasActiveProjects sentinel error (domain + HTTP 409 handler)
affects:
  - 04-02 (frontend: customer combobox in create dialog + tests)
tech-stack:
  added: []
  patterns:
    - HTTP handler parses *string → *uuid.UUID for optional foreign key
    - Delete protection with multiple sentinel errors for specific HTTP 409 responses
key-files:
  created: []
  modified:
    - internal/core/domain/contract/contract.go
    - internal/core/ports/contract_repository.go
    - internal/core/services/contract/contract.go
    - internal/core/services/contract/contract_test.go
    - internal/adapters/primary/http/contract.go
    - internal/adapters/secondary/postgres/contract_repository.go
decisions:
  - "CustomerID on CreateContractRequest uses *uuid.UUID (nullable pointer) for domain, *string for HTTP handler (JSON-native), parsed at handler boundary"
  - "HasProjects counts ALL projects (not just active) — consistent with ON DELETE RESTRICT FK constraint"
  - "HasProjects check runs after HasTimeEntries check in Delete service method"
metrics:
  duration: 2m
  completed: 2026-06-11
---

# Phase 4 Plan 01: Backend Contract CRUD with Customer & Delete Protection

**Completed:** 2026-06-11

One-liner: "Add `customer_id` to contract create flow (domain → handler → PG INSERT) and harden delete protection with `HasProjects` check returning specific 409 error"

## Implementation

### Task 1: Domain model (`internal/core/domain/contract/contract.go`)
- Added `CustomerID *uuid.UUID \`json:"customer_id,omitempty"\`` to `CreateContractRequest` struct
- Added `ErrHasActiveProjects = errors.New("contract has active projects")` sentinel error

### Task 2: Repository interface + mock (`internal/core/ports/contract_repository.go`, `internal/core/services/testdata/mocks.go`)
- Added `HasProjects(ctx, contractID) (int, error)` to `ContractRepository` interface
- Mock already had `HasProjectsFn` field + `HasProjects` method (pre-existing)

### Task 3: PostgreSQL repository (`internal/adapters/secondary/postgres/contract_repository.go`)
- Updated `Create` INSERT to include `customer_id` column with `$5` placeholder → `req.CustomerID`
- Added `HasProjects(ctx, contractID)` method — `SELECT COUNT(*) FROM projects WHERE contract_id = $1`

### Task 4: HTTP handler + Service (`internal/adapters/primary/http/contract.go`, `internal/core/services/contract/contract.go`)
- Added `CustomerID *string \`json:"customer_id,omitempty"\`` to handler `CreateContractRequest`
- Added `customer_id` parsing in `Create` handler: `*string` → `*uuid.UUID` (validates, allows null/empty)
- Added `HasProjects` check in `Delete` service method (after existing `HasTimeEntries`)
- Added `ErrHasActiveProjects` case in HTTP Delete error switch → returns 409 "contract has projects and cannot be deleted"

### Task 5: Unit tests (`internal/core/services/contract/contract_test.go`)
- Added "valid contract with customer" test case to `TestService_Create`
- Added "blocked by projects" test case to `TestService_Delete`

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ Passed |
| `go vet ./internal/...` | ✅ Passed |
| `go test -run TestService_ ./internal/core/services/contract/...` | ✅ Passed |
| `TestService_Create/valid_contract_with_customer` | ✅ Passed |
| `TestService_Delete/blocked_by_projects` | ✅ Passed |

## Requirements Satisfied

- **CTRT-02:** Customer dropdown on contract create — backend now stores `customer_id` from `CreateContractRequest`
- **CTRT-04:** Delete protection checks `HasProjects` before deletion, returns specific 409 "contract has projects and cannot be deleted"

## Deviations from Plan

None — plan executed exactly as written.

## Pre-existing Issues (Not Related to This Plan)

The following integration test failures are pre-existing (register endpoint returns 200 instead of 201, known from Phase 1 auth changes) and affect all handler integration tests:
- `TestAuthHandlerIntegration` — 9 subtests
- `TestAuthIntegration` — 5 subtests
- `TestUnitHandlerIntegration` — 3 subtests
- `TestOrganizationHandlerIntegration` — 3 subtests
- `TestProjectHandlerIntegration` — 4 subtests
- `TestContractHandlerIntegration` — 2 subtests
- `TestCustomerHandlerIntegration` — 2 subtests
- `TestTimeEntryHandlerIntegration` — 2 subtests
- `TestWorkingGroupHandlerIntegration` — 2 subtests

All service-level and postgres repository tests pass (11 service packages + 1 postgres package: OK).

## Self-Check: PASSED

- All 5 commits for plan 04-01 exist: ✓
- `go build ./...` compiles: ✓
- `go vet ./internal/...` passes: ✓
- Contract service unit tests pass: ✓
- All 7 modified files verified on disk: ✓
