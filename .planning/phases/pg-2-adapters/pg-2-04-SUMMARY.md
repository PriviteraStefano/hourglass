---
phase: pg-2-adapters
plan: 04
subsystem: "PostgreSQL Adapters — Project, Contract, Customer Repositories"
tags: [postgres, repository, project, subproject, contract, customer]
dependency:
  requires: [pg-2-01, pg-2-02, pg-2-03]
  provides: [ProjectRepository, SubprojectRepository, ContractRepository, CustomerRepository]
  affects: [pg-2-05, pg-2-06]
tech-stack:
  added: []
  patterns: [pgxpool.QueryRow/Query, positional Scan, dynamic SET building, scope-based filtering]
key-files:
  created:
    - internal/adapters/secondary/postgres/project_repository.go
    - internal/adapters/secondary/postgres/subproject_repository.go
    - internal/adapters/secondary/postgres/contract_repository.go
    - internal/adapters/secondary/postgres/customer_repository.go
    - internal/adapters/secondary/postgres/project_repository_test.go
    - internal/adapters/secondary/postgres/subproject_repository_test.go
    - internal/adapters/secondary/postgres/contract_repository_test.go
    - internal/adapters/secondary/postgres/customer_repository_test.go
decisions:
  - "Scope-based project/contract List uses dynamic WHERE clause building with PostgreSQL parameterized args"
  - "Subproject domain model uses string IDs (models.Subproject) — UUID parsing done at repository boundary"
  - "Contract Update uses dynamic SET clause from non-zero fields, fetches full response via Get after update"
  - "RecalculateMileage fetches contract km_rate before updating expenses"
  - "Delete methods check RowsAffected() to return domain ErrNotFound when no rows matched"
  - "Customer SQL columns org_id→OrganizationID, name→CompanyName mapped in scan function"
metrics:
  duration: "~25 min"
  completed: "2026-06-07"
  files_created: 8
  commits: 3
  tests: 19
---

# Phase Pg-2 Plan 04: Project/Contract/Customer Repositories Summary

Completed 3 tasks: ProjectRepository + SubprojectRepository, ContractRepository, and CustomerRepository with full integration test suites.

## Tasks

### Task 1: ProjectRepository + SubprojectRepository

**ProjectRepository** (7 methods):
- `List` — Scope-based filtering (default/adopted/all) with optional contractID filter, LEFT JOINs for contract/organization names, subqueries for adoption count and is_adopted flag
- `Get` — Same query with `WHERE p.id = $2`, returns `ErrProjectNotFound` on miss
- `Create` — INSERT with `RETURNING *`, then Get for full response
- `Adopt` — EXISTS check before INSERT, returns `ErrAlreadyAdopted` if duplicate
- `ListManagers` — LEFT JOIN users for user_name and email
- `AddManager` — INSERT into project_managers, then fetch user info
- `RemoveManager` — DELETE, returns `ErrProjectNotFound` if no rows affected

**SubprojectRepository** (5 methods):
- `ListByProject` — SELECT ordered by sequence_order, name
- `GetByID` — Parses string ID to uuid.UUID, returns nil,nil on ErrNoRows
- `Create` — INSERT with RETURNING, generates ID via uuid.New()
- `Update` — UPDATE with RETURNING
- `Delete` — DELETE by ID

**Tests:** Create→Get, List with all 3 scopes, Adopt→ErrAlreadyAdopted, Manager lifecycle, Subproject CRUD

### Task 2: ContractRepository

**ContractRepository** (8 methods):
- `List` — Scope-based filtering with optional isActive filter, aggregates for adoption count, is_adopted, customer_name, time_entries_count
- `Create` — INSERT, then Get for full ContractResponse with aggregates
- `Get` — Full query with LEFT JOINs and 5 subquery aggregates
- `Adopt` — EXISTS check, returns `ErrAlreadyAdopted` on duplicate
- `Update` — Dynamic SET clause from non-zero/non-nil fields, verifies RowsAffected, returns (ContractResponse, 0, error)
- `RecalculateMileage` — Fetches contract km_rate, updates expense amounts
- `Delete` — WHERE id + created_by_org_id, returns `ErrContractNotFound` if no rows
- `HasTimeEntries` — COUNT(*) subquery across projects under the contract

**Tests:** Create→Get with aggregates, List scope filtering, Update with partial fields, Adopt→ErrAlreadyAdopted, Delete with wrong org, HasTimeEntries

### Task 3: CustomerRepository

**CustomerRepository** (7 methods):
- `ListByOrg` — SELECT with LIMIT/OFFSET pagination
- `Create` — INSERT with RETURNING, generates ID
- `GetByID` — Returns `ErrCustomerNotFound` on miss
- `Update` — Full-field UPDATE with RETURNING
- `Deactivate` — Sets is_active=false, returns `ErrCustomerNotFound` if no rows
- `ListContractsByCustomer` — SELECT from contracts with customer_id filter
- `CountContractsByCustomer` — COUNT(*) with customer_id filter

**Tests:** ListByOrg, Create→GetByID, Update (rename), Deactivate→NotFound, contract listing with Update to set customer_id

## Verification

- `go build ./internal/...` — PASS
- `go vet ./internal/adapters/secondary/postgres/...` — PASS
- `go test -c ./internal/adapters/secondary/postgres/` — compiles (tests skip when DATABASE_URL is unset)

## Deviations from Plan

None — executed exactly as specified.

## Known Stubs

None.

## Threat Flags

None — all repositories are internal infrastructure with no new network endpoints or auth paths.

## Self-Check: PASSED

All 8 files created and verified. Commits confirmed:
- `be8ec16` — ProjectRepository and SubprojectRepository with tests
- `9b35787` — ContractRepository with tests
- `6fe4a30` — CustomerRepository with tests
