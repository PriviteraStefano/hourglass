---
phase: pg-2-adapters
plan: 02
name: UserRepository + Organization repos
subsystem: adapters/secondary/postgres
tags:
  - postgres
  - user-repository
  - organization-repository
  - repository-pattern
depends_on:
  - pg-2-01
provides:
  - UserRepository (PostgreSQL)
  - OrganizationRepository (PostgreSQL)
  - OrganizationMembershipRepository (PostgreSQL)
  - OrganizationManagementRepository (PostgreSQL)
key-files:
  created:
    - internal/adapters/secondary/postgres/user_repository.go
    - internal/adapters/secondary/postgres/user_repository_test.go
    - internal/adapters/secondary/postgres/organization_repo.go
    - internal/adapters/secondary/postgres/organization_repo_test.go
    - internal/adapters/secondary/postgres/organization_membership_repo.go
    - internal/adapters/secondary/postgres/organization_membership_repo_test.go
    - internal/adapters/secondary/postgres/organization_management_repo.go
    - internal/adapters/secondary/postgres/organization_management_repo_test.go
  modified: []
decisions:
  - Use pgxpool.QueryRow for single-row, pool.Query for list, pool.Exec for writes, pool.Begin for transactions
  - scanUser helper wraps pgx.ErrNoRows to ports.ErrUserNotFound for user lookups
  - scanMemberships shared between both OrganizationMembershipRepository methods
  - GetByID/GetMembership return nil,nil (not error) for not-found per existing surrealdb pattern
  - JSONB financial_cutoff_config uses json.Marshal/json.Unmarshal with []byte intermediary
  - OrganizationRepository uses auth.Organization domain; OrganizationManagementRepository uses orgdomain.Organization
duration: 12m
completed_date: 2026-06-07
---

# Phase pg-2-adapters Plan 02: UserRepository + Organization repos Summary

**One-liner:** PostgreSQL implementations of UserRepository (11 methods), OrganizationRepository (4 methods), OrganizationMembershipRepository (2 methods), and OrganizationManagementRepository (10 methods) with full test coverage.

## Key Decisions

- **scanUser helper:** Extracted a shared scanner function that maps `pgx.ErrNoRows` to `ports.ErrUserNotFound`, used by `GetByEmail`, `GetByUsername`, `GetByID`
- **scanMemberships helper:** Shared row iterator for `ListByUser` and `ListByOrg` in `OrganizationMembershipRepository`
- **not-found pattern for OrganizationRepository:** Returns `nil, nil` (not a sentinel error) for `GetByID` and `GetMembership`, matching the existing surrealdb adapter behavior
- **JSONB handling:** `financial_cutoff_config` column marshaled from `map[string]interface{}` via `json.Marshal` → `[]byte`, scanned back via `json.RawMessage` → `[]byte` → `json.Unmarshal`
- **Domain separation:** `OrganizationRepository` operates on `auth.Organization` (full domain model with financial config); `OrganizationManagementRepository` operates on `orgdomain.Organization` (minimal view with id/name/slug/createdAt)
- **TTL-independent pool:** All methods use `context.Context` from caller; no server-side TTL assumptions

## Created Files

### `user_repository.go` — 11 methods

| Method | SQL Pattern | Notes |
|--------|-----------|-------|
| Add | INSERT (9 cols) | wraps unique_violation → ErrConflict |
| GetByEmail | SELECT WHERE email=$1 | ErrNoRows → ErrUserNotFound |
| GetByUsername | SELECT WHERE username=$1 | same |
| GetByID | SELECT WHERE id=$1 | same |
| EmailExists | SELECT EXISTS(...) | bool return |
| UsernameExists | SELECT EXISTS(...) | bool return |
| AnyExists | SELECT EXISTS(SELECT 1) | bool return |
| UpdatePassword | UPDATE SET password_hash | sets updated_at=NOW() |
| GetMemberships | SELECT FROM org_memberships | returns [] (not nil) on empty |
| AddWithMembership | BEGIN→INSERT user→INSERT membership→COMMIT | transactional |
| AddWithOrgAndMembership | BEGIN→INSERT org→INSERT user→INSERT membership→COMMIT | transactional, JSONB for config |

### `organization_repo.go` — 4 methods

- `Add` — INSERT with JSONB financial_cutoff_config
- `GetByID` — SELECT, returns nil,nil on not-found
- `GetMembership` — SELECT WHERE user_id+org_id, returns nil,nil on not-found
- `AddMembership` — INSERT with nullable invited_by/invited_at/activated_at

### `organization_membership_repo.go` — 2 methods

- `ListByUser` — SELECT WHERE user_id, uses shared `scanMemberships`
- `ListByOrg` — SELECT WHERE org_id, uses shared `scanMemberships`

### `organization_management_repo.go` — 10 methods

- `CreateOrganization` — BEGIN→INSERT org→INSERT owner membership→COMMIT
- `GetOrganization` — SELECT id, name, slug, created_at
- `InviteMember` — INSERT with user_id=NULL for invited-before-registration
- `GetSettings` — SELECT from organization_settings
- `UpdateSettings` — UPDATE with COALESCE logic, RETURNING new row
- `ListMembers` — LEFT JOIN users for user_name/user_email
- `UpdateMemberRole` — UPDATE role WHERE id+org_id
- `DeactivateMember` — UPDATE is_active=false
- `CountActiveFinance` — SELECT COUNT(*) WHERE role=finance AND active
- `GetMemberRole` — SELECT role, returns ErrMemberNotFound on miss

## Test Coverage

- **user_repository_test.go** (11 tests): Add→GetByID round-trip, GetByEmail/Username/ID not-found, Email/Username/AnyExists, UpdatePassword, GetMemberships (with and without results), AddWithMembership, AddWithOrgAndMembership, duplicate email conflict
- **organization_repo_test.go** (4 tests): Add→GetByID with JSONB round-trip, not-found, AddMembership→GetMembership, duplicate membership
- **organization_membership_repo_test.go** (2 tests): ListByUser with 2 memberships, ListByOrg with 2 memberships, empty slice for no results
- **organization_management_repo_test.go** (9 tests): CreateOrganization (with settings auto-creation), GetOrganization not-found, GetSettings (defaults), GetSettings not-found, UpdateSettings (all fields), ListMembers (with LEFT JOIN user_name), UpdateMemberRole, DeactivateMember, CountActiveFinance (active vs inactive), InviteMember (user_id=NULL), GetMemberRole not-found

All tests use `TestPool` + `SetupTestSchema` + `TeardownTestSchema` in cleanup for isolation.

## Deviations from Plan

None — plan executed exactly as written.

## Threat Flags

None — all files are backend repository implementations with no new network endpoints, auth paths, or file access patterns.

## Known Stubs

None.

## Self-Check

- [x] `internal/adapters/secondary/postgres/user_repository.go` exists
- [x] `internal/adapters/secondary/postgres/user_repository_test.go` exists
- [x] `internal/adapters/secondary/postgres/organization_repo.go` exists
- [x] `internal/adapters/secondary/postgres/organization_repo_test.go` exists
- [x] `internal/adapters/secondary/postgres/organization_membership_repo.go` exists
- [x] `internal/adapters/secondary/postgres/organization_membership_repo_test.go` exists
- [x] `internal/adapters/secondary/postgres/organization_management_repo.go` exists
- [x] `internal/adapters/secondary/postgres/organization_management_repo_test.go` exists
- [x] Commit a8b6212 — `feat(pg-2-02): UserRepository with 11 methods and tests`
- [x] Commit c85107a — `feat(pg-2-02): organization repositories and tests`
- [x] `go build ./internal/adapters/secondary/postgres/` passes
- [x] `go vet ./internal/adapters/secondary/postgres/` passes
