---
phase: pg-2-adapters
plan: 03
subsystem: postgres
tags: [repositories, units, working-groups, invitations, pgx]
requires: [pg-2-01, pg-2-02]
provides: [postgres-unit-repo, postgres-wg-repo, postgres-invitation-repo]
affects: []
tech-stack:
  added:
    - pgtype.Array[pgtype.UUID] for UUID[] column scanning
  patterns:
    - String↔UUID conversion at adapter boundary (domain models use string IDs
      for certain fields while PG stores UUID)
    - Dynamic SET building for Update methods (name always + conditional fields)
    - Recursive CTE for hierarchical queries (GetDescendants)
    - UUID[] array round-trip via pgtype.Array[pgtype.UUID]
    - Delegate pattern: UnitRepository→UnitMemberRepository,
      WorkingGroupRepository→WGMemberRepository
key-files:
  created:
    - internal/adapters/secondary/postgres/unit_repository.go
    - internal/adapters/secondary/postgres/unit_member_repository.go
    - internal/adapters/secondary/postgres/working_group_repository.go
    - internal/adapters/secondary/postgres/wg_member_repository.go
    - internal/adapters/secondary/postgres/invitation_repository.go
    - internal/adapters/secondary/postgres/unit_repository_test.go
    - internal/adapters/secondary/postgres/unit_member_repository_test.go
    - internal/adapters/secondary/postgres/working_group_repository_test.go
    - internal/adapters/secondary/postgres/wg_member_repository_test.go
    - internal/adapters/secondary/postgres/invitation_repository_test.go
  modified:
    - internal/adapters/secondary/postgres/exported_test_helpers.go
decisions:
  - "Dynamic SET: UnitRepository.Update always sets name; description, code,
    parent_unit_id only when non-empty (matching UpdateUnitRequest semantics)"
  - "UUID[] scan/encode: pgtype.Array[pgtype.UUID] with toUUIDArray/scanUUIDArray
    helpers for WorkingGroup unit_ids and delegate_ids"
  - "Delegate pattern: UnitRepository/WorkingGroupRepository hold references to
    their member repos and delegate ListMembers/AddMember/RemoveMember"
  - "Invitation status: stored as-is from domain (no mapping); DB CHECK constraint
    accepts 'pending', 'accepted', 'expired' — tests use only these values"
  - "CreatedBy: string in domain, UUID in PG with parse/convert at boundary"
metrics:
  duration: ~20 min
  files_created: 10
  files_modified: 1
  commits: 3
  completed_date: 2026-06-07
---

# Phase pg-2-adapters Plan 03: Unit/WG/Invitation Repositories — Summary

Three PostgreSQL adapter repositories implementing domain port interfaces for unit management (with unit member sub-repo), working group management (with WG member sub-repo), and invitation management. All with integration tests.

## Task Results

### Task 1: UnitRepository + UnitMemberRepository ✅
- **UnitRepository** (11 methods): ListByOrg, GetByID, Create, Update (dynamic SET), Delete, GetDescendants (recursive CTE), HasMembers, GetMemberCountsByOrg, plus ListMembers/AddMember/RemoveMember delegated to UnitMemberRepository
- **UnitMemberRepository** (3 methods): ListByUnit (LEFT JOIN users for user_name/user_email), Add (INSERT RETURNING), Remove (DELETE)
- **String↔UUID conversion**: `Unit.ID` (string→uuid.UUID for query, scan back as uuid.UUID→.String()), `Unit.ParentUnitID` (nullable *uuid.UUID→"" string), `UnitMember.ID`, `UnitMember.UnitID` same pattern
- **Tests**: ListByOrg, Create→GetByID, NotFound, Update, Delete, GetDescendants (parent→child→grandchild), HasMembers, GetMemberCountsByOrg, ListByUnit, Add→ListByUnit, Remove
- **Commit**: `4b37f07`

### Task 2: WorkingGroupRepository + WGMemberRepository ✅
- **WorkingGroupRepository** (9 methods): ListByOrg (with optional subprojectID filter), GetByID, Create, Update (dynamic SET including arrays), Delete, HasMembers, plus ListMembers/AddMember/RemoveMember delegated to WGMemberRepository
- **WGMemberRepository** (3 methods): ListByWG, Add (INSERT RETURNING), Remove (DELETE)
- **UUID[] array handling**: `pgtype.Array[pgtype.UUID]` for `unit_ids` and `delegate_ids` columns → `toUUIDArray()` converts `[]string` to PG array, `scanUUIDArray()` converts back
- **Tests**: ListByOrg (unfiltered + filtered by subproject), Create→GetByID (UUID[] round-trip verification), NotFound, Update, Delete, HasMembers (before/after adding member), WG member CRUD
- **Commit**: `802ada5`

### Task 3: InvitationRepository ✅
- **InvitationRepository** (4 methods): Create, FindByCode, FindByToken, Update
- **CreatedBy conversion**: string (domain) → uuid.UUID (PG) via `uuid.Parse()` on write, `createdBy.String()` on read
- **Error mapping**: `pgx.ErrNoRows` → `invitation.ErrInvitationNotFound`
- **Tests**: Create→FindByCode, Create→FindByToken, NotFound by code and token, Update status
- **Commit**: `39de510`

## Test Helpers Added

- `seedOrg(t, pool, now)` — inserts organization row with unique slug
- `seedUser(t, pool, now)` — inserts user row with unique email/username
- `seedProject(t, pool, orgID, now)` — inserts billable project
- `seedSubproject(t, pool, projectID, now)` — inserts subproject
- `seedUnit(t, pool, orgID, now)` — inserts unit via raw SQL

## Verification

- `go vet ./internal/adapters/secondary/postgres/` — passed (no output)
- `go build ./internal/adapters/secondary/postgres/` — passed
- `go build ./internal/...` — passed

## Deviations from Plan

None — plan executed exactly as written.

## Threat Flags

No threat flags — all created files are repository adapters with no new network endpoints, auth paths, or schema changes.

## Known Stubs

None found.

## Self-Check: PASSED

- All 10 created files confirmed on disk
- All 3 commit hashes verified in git log
- Go build passes for postgres adapter and all internal packages
