---
phase: 02-org-hierarchy
plan: 01
subsystem: api, backend
tags: [go, postgres, pgx, unit, unit-members, delete-protection]

# Dependency graph
requires:
  - phase: 01-authorization
    provides: JWT auth middleware for protected endpoints
provides:
  - PUT /units/{id}/members/{membershipId} endpoint for primary unit designation (ORG-03)
  - GET /units/members/batch endpoint for batch member fetching (ORG-03)
  - Delete protection enforcement — root unit / children / members checks (ORG-05)
  - One-primary-per-user enforcement at service layer (ORG-03)
affects:
  - 02-02 (frontend primary unit UI can consume new endpoint)
  - 02-03 (frontend batch members UI can consume batch endpoint)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - EXISTS query pattern for HasChildren (identical to HasMembers)
    - UPDATE RETURNING with inline scan (no JOIN columns) for Update method
    - ANY($1) with UUID array for batch member queries
    - One-primary-per-user enforcement using ListMembershipsForUser

key-files:
  created: []
  modified:
    - internal/core/domain/unit/unit.go — sentinel errors
    - internal/core/ports/unit_repository.go — port interface methods
    - internal/core/services/unit/unit.go — service methods
    - internal/adapters/secondary/postgres/unit_repository.go — HasChildren + delegation
    - internal/adapters/secondary/postgres/unit_member_repository.go — Update, ListByUnitIDs, ListMembershipsForUser
    - internal/adapters/primary/http/unit.go — UpdateMember, ListMembersBatch handlers
    - cmd/server/main.go — route registration
    - internal/core/services/unit/unit_integration_test.go — integration subtests
    - internal/core/services/unit/unit_test.go — fixed mock test for root unit check
    - internal/core/services/testdata/mocks.go — MockUnitRepo methods
    - internal/adapters/secondary/postgres/unit_member_repository_test.go — repo tests

key-decisions:
  - "PUT /units/{id}/members/{membershipId} with is_primary/end_date fields and one-primary-per-user enforcement"
  - "Root unit check before children check — hierarchy_level===0 gates first"
  - "HasChildren uses EXISTS pattern matching HasMembers pattern"
  - "ListByUnitIDs uses ANY($1) with UUID array for batch query"
  - "UpdateMember UPDATE RETURNING uses inline scan (no JOIN columns in RETURNING)"

patterns-established:
  - "Delete service checks root unit -> children -> members in priority order"
  - "One-primary-per-user: ListMembershipsForUser + unset other primaries"

requirements-completed: [ORG-03, ORG-05]

# Metrics
duration: 25min
completed: 2026-06-10
---

# Phase 02: Org Hierarchy — Plan 01 Summary

**Backend foundation for primary unit designation (PUT /units/{id}/members/{membershipId}) and delete protection enforcement (root/children/members) with all supporting port, repository, service, and handler layers plus integration tests**

## Performance

- **Duration:** 25 min
- **Started:** 2026-06-10T23:45:00Z
- **Completed:** 2026-06-10T23:45:00Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments
- 2 new sentinel errors: `ErrCannotDeleteRootUnit`, `ErrCannotDeleteWithChildren`
- 4 new port interface methods: `UpdateMember`, `HasChildren`, `ListMembersByUnitIDs`, `ListMembershipsForUser`
- Updated `Service.Delete` with root unit (hierarchy_level===0), children (HasChildren), and members (HasMembers) checks
- New `Service.UpdateMember` with one-primary-per-user enforcement (unset other primaries for same user)
- New `Service.ListMembersByUnitIDs` delegation to repo
- Repository: `HasChildren` (EXISTS), `Update` (UPDATE RETURNING), `ListByUnitIDs` (ANY($1)), `ListMembershipsForUser` (WHERE user_id)
- HTTP handlers: `UpdateMember` (PUT) and `ListMembersBatch` (GET) with sentinel error mapping
- Updated `Delete` handler with root unit and children error responses
- 2 new routes in main.go: `PUT /units/{id}/members/{membership_id}`, `GET /units/members/batch`
- 4 integration subtests: `UpdateMember`, `Delete_RootUnit`, `Delete_HasChildren`, `Delete_HasMembers`
- 2 repository tests: `TestUnitMemberRepository_Update`, `TestUnitMemberRepository_ListByUnitIDs`

## Task Commits

Each task was committed atomically:

1. **Task 1: Domain sentinel errors + Port interface + Service methods** - `1f55135` (feat)
2. **Task 2: Repository implementations** - `697766c` (feat)
3. **Task 3: HTTP handlers + route registration + integration tests** - `1212ffa` (feat)

## Files Created/Modified
- `internal/core/domain/unit/unit.go` — Added `ErrCannotDeleteRootUnit`, `ErrCannotDeleteWithChildren`
- `internal/core/ports/unit_repository.go` — Added `UpdateMember`, `HasChildren`, `ListMembersByUnitIDs`, `ListMembershipsForUser`
- `internal/core/services/unit/unit.go` — Updated `Delete` with root/children checks; added `UpdateMember` (one-primary-per-user), `ListMembersByUnitIDs`
- `internal/adapters/secondary/postgres/unit_repository.go` — Added `HasChildren` (EXISTS), delegation methods for Update, ListByUnitIDs, ListMembershipsForUser
- `internal/adapters/secondary/postgres/unit_member_repository.go` — Added `Update` (UPDATE RETURNING), `ListByUnitIDs` (ANY($1)), `ListMembershipsForUser`
- `internal/adapters/primary/http/unit.go` — Added `UpdateUnitMemberRequest`, `UpdateMember`, `ListMembersBatch` handlers; updated `Delete` sentinel mappings
- `cmd/server/main.go` — Registered `PUT /units/{id}/members/{membership_id}`, `GET /units/members/batch`
- `internal/core/services/unit/unit_integration_test.go` — Added `UpdateMember`, `Delete_RootUnit`, `Delete_HasChildren`, `Delete_HasMembers` subtests
- `internal/core/services/unit/unit_test.go` — Fixed mock `seedUnit` for root unit check
- `internal/core/services/testdata/mocks.go` — Added `HasChildren`, `UpdateMember`, `ListMembersByUnitIDs`, `ListMembershipsForUser` to MockUnitRepo
- `internal/adapters/secondary/postgres/unit_member_repository_test.go` — Added `TestUnitMemberRepository_Update`, `TestUnitMemberRepository_ListByUnitIDs`

## Decisions Made
- **D-02:** PUT endpoint with `is_primary` and `end_date` fields — frontend will only use `is_primary` for v0.1
- **D-03:** One primary per user enforced at service layer via `ListMembershipsForUser` + unset other primaries
- **D-05/D-06:** Delete protection checks root unit (hierarchy_level===0) first, then children, then members — each returns specific 400-level error
- **HasChildren uses EXISTS:** Identical pattern to HasMembers for consistency
- **ListByUnitIDs uses ANY($1):** pgx/v5 supports UUID arrays with ANY($1) for batch queries
- **UpdateMember UPDATE RETURNING uses inline scan:** RETURNING does not include JOIN columns (user_name, user_email), so uses inline field scan matching the Add method's pattern

## Deviations from Plan
None - plan executed exactly as written.

### Deviations from Plan (Plan-as-Authority)

**1. [Rule 1 - Bug] Existing TestUnitIntegration/Delete test failed after root unit check**
- **Found during:** Task 3 (integration tests)
- **Issue:** The existing `Delete` subtest created a unit at hierarchy_level 0 (root), which the new `Service.Delete` rejected with `ErrCannotDeleteRootUnit`. Similarly, `Delete_HasChildren` and `Delete_HasMembers` subtests created test hierarchies where the target unit was root.
- **Fix:** Updated existing `Delete` test to create a child unit (under a root) for deletion. Updated `Delete_HasChildren` to use grandparent→parent→child chain. Updated `Delete_HasMembers` to create parent under root. Also updated mock `seedUnit` to set `HierarchyLevel: 1`.
- **Files modified:** `unit_integration_test.go`, `unit_test.go`
- **Verification:** All 13 integration subtests pass
- **Committed in:** `1212ffa` (Task 3 commit)

**2. [Rule 1 - Bug] MockUnitRepo missing 4 new interface methods**
- **Found during:** Task 3 (building test)
- **Issue:** `MockUnitRepo` in `testdata/mocks.go` didn't implement the 4 new `ports.UnitRepository` methods (`HasChildren`, `UpdateMember`, `ListMembersByUnitIDs`, `ListMembershipsForUser`), causing build failure for mock tests.
- **Fix:** Added stub implementations for all 4 methods.
- **Files modified:** `mocks.go`
- **Verification:** `go test ./internal/core/services/unit/...` passes
- **Committed in:** `1212ffa` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes were necessary for test correctness. No scope creep.

## Issues Encountered
- Pre-existing auth handler integration tests failing with register→200 instead of 201 (Phase 1 change, unrelated to this plan)

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Backend foundation complete for ORG-03 primary unit designation and ORG-05 delete protection
- Ready for Plan 02-02: Frontend "Make Primary" button in side panel + ReparentConfirmDialog wiring + pendingEdgeConnect removal + subtree members section + batch endpoint frontend integration
- New endpoint `PUT /units/{id}/members/{membership_id}` ready for frontend consumption
- New batch endpoint `GET /units/members/batch` ready for frontend consumption

---
*Phase: 02-org-hierarchy*
*Completed: 2026-06-10*
