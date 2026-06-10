---
phase: 03-customers
plan: 01
subsystem: backend
tags: [customers, is-internal, migration, search, org-creation]
requires: []
provides:
  - "is_internal column on customers table"
  - "Internal customer records for existing organizations"
  - "ILIKE search on GET /customers?search="
  - "CreateInternalCustomer method (no role check)"
  - "Auto-created internal customer on org creation"
  - "Company_name lock for internal customers on update"
affects:
  - internal/core/services/organization/organization.go (new dependency on customer service)
tech-stack:
  added: []
  patterns:
    - "Internal customer auto-creation via service orchestration (org → customer)"
    - "Company_name lock pattern: fetch current, check IsInternal, skip update"
key-files:
  created:
    - migrations/009_add_is_internal_to_customers.up.sql
    - migrations/009_add_is_internal_to_customers.down.sql
  modified:
    - internal/core/domain/customer/customer.go
    - internal/core/ports/customer_repository.go
    - internal/adapters/secondary/postgres/customer_repository.go
    - internal/core/services/customer/customer.go
    - internal/adapters/primary/http/customer.go
    - internal/core/services/organization/organization.go
    - internal/core/services/testdata/mocks.go
    - cmd/server/main.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main_test.go
    - internal/adapters/secondary/postgres/customer_repository_test.go
    - internal/core/services/customer/customer_test.go
    - internal/core/services/customer/customer_integration_test.go
    - internal/core/services/organization/organization_test.go
    - internal/core/services/organization/organization_integration_test.go
decisions:
  - "Company_name is locked for internal customers at the service layer (D-05)"
  - "CreateInternalCustomer has no role check — internal orchestration call only"
  - "Org creation failure to create internal customer is logged, not blocking (D-03)"
  - "Seed uses WHERE NOT EXISTS for idempotency (no unique constraint to conflict on)"
metrics:
  duration: ~4.3 min
  completed: 2026-06-10
---

# Phase 3 Plan 01: Backend foundation — is_internal migration + search + internal customer support

**One-liner:** DB migration + ILIKE search + CreateInternalCustomer + company_name lock + auto-creation on org creation — full backend foundation for Phase 3 internal customer support.

## Accomplishments

### T1: `is_internal` column migration + seed existing orgs
- Created `migrations/009_add_is_internal_to_customers.up.sql`: adds `is_internal BOOLEAN NOT NULL DEFAULT false` and seeds internal customer records for all existing organizations (idempotent via `WHERE NOT EXISTS`).
- Created `migrations/009_add_is_internal_to_customers.down.sql`: drops the column.
- Migration applies cleanly — verified against running PostgreSQL instance.
- 70 existing orgs had internal customer records created; NovaTech external customer preserved with `is_internal=false`.

### T2: Backend domain + port + repository + service + handler changes
- **Domain:** Added `IsInternal bool` with `json:"is_internal"` to the `Customer` struct.
- **Port:** Changed `ListByOrg` signature to accept `search string`. Added `CreateInternal(ctx, orgID, companyName)` method.
- **Repository:** Added `is_internal` to all SELECT, RETURNING, and scan operations. `ListByOrg` now builds an ILIKE filter on `name`, `contact_name`, `email` when `search != ""`. Added `CreateInternal` method that creates with all optional fields empty.
- **Service:** `List` passes search to repo. Added `CreateInternalCustomer(ctx, orgID, companyName)` with no role check. `Update` locks `company_name` for internal customers (preserves original if internal).
- **Handler:** `List` parses `?search=` query param from URL and passes to service.

### T3: Auto-create internal customer on org creation
- Modified `internal/core/services/organization/organization.go`: added `customerService` dependency, updated `NewService`, after `CreateOrganization` calls `s.customerService.CreateInternalCustomer`.
- Error is logged but does not block org creation.
- Updated all callers: `main.go`, `handler_test_helper.go`, `main_test.go`, `organization_test.go`, `organization_integration_test.go`.
- `customerService` is nil-safe — tests passing `nil` work correctly.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| T1   | Add is_internal column migration + seed existing orgs | `5bc7451` | `009_add_is_internal_to_customers.up.sql`, `009_add_is_internal_to_customers.down.sql` |
| T2   | Backend domain model + port + repository + service + handler changes | `0a95f4c` | 9 files (domain, port, repo, service, handler, mock, tests) |
| T3   | Auto-create internal customer on org creation | `0f8a21b` | 6 files (org service, main.go, test files) |

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ Compiled cleanly |
| `go vet ./internal/...` | ✅ Passed |
| `go run ./cmd/migrate -up -dir migrations` | ✅ Applied (schema + seed) |
| `is_internal` column exists | ✅ Verified via `\d customers` |
| Internal customers seeded per org | ✅ 70 rows, no duplicates (0 orgs with >1 internal) |
| `go test ./internal/core/services/customer/...` | ✅ Ok |
| `go test ./internal/core/services/organization/...` | ✅ Ok |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — all threat mitigations from the plan's threat model were implemented:
- **T-03-01** (Tampering PUT /customers/{id}): company_name locked for internal customers in service layer ✓
- **T-03-02** (Information Disclosure GET /customers?search=): search scoped to org_id via middleware ✓
- **T-03-03** (Elevation of Privilege CreateInternalCustomer): only called from org service, not exposed via HTTP ✓

## Self-Check: PASSED

- `migrations/009_add_is_internal_to_customers.up.sql` exists: ✓
- `migrations/009_add_is_internal_to_customers.down.sql` exists: ✓
- `internal/core/domain/customer/customer.go` has `IsInternal bool`: ✓
- `internal/core/ports/customer_repository.go` has `search string` param + `CreateInternal`: ✓
- `go build ./...`: ✓
- `go vet ./internal/...`: ✓
- 3 commits for plan 03-01 exist: ✓
