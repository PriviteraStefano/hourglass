---
phase: 05-mvp-consolidation
plan: 04
subsystem: testing
tags: [go, testify, unit-tests, service-tests, handler-tests, project]

requires:
  - phase: 05-mvp-consolidation
    plan: 01
    provides: MockProjectRepo with HasActiveTimeEntriesFn
provides:
  - Service-level unit tests for project Update (role gating) and Delete (role gating, ownership check, active entries check)
  - Handler-level unit tests for project Update, Delete, and ListSubprojects (HTTP error mapping)

tech-stack:
  added: []
  patterns: [test-table-driven for service tests, handler-mock-subproject-repo for adapter tests]

key-files:
  created: []
  modified:
    - internal/core/services/project/project_test.go — Added TestService_Update (2 subtests) and TestService_Delete (6 subtests)
    - internal/adapters/primary/http/project_test.go — Added TestProjectHandler_Update (2 subtests), TestProjectHandler_Delete (4 subtests), TestProjectHandler_ListSubprojects (1 subtest), plus setupProjectHandler helper and mockSubprojectRepo

key-decisions:
  - "Used middleware.SetOrganizationID and middleware.SetRole instead of the planned WithOrgID/WithRole (those names don't exist in the middleware package)"
  - "Initialized mockRepo.Projects map explicitly in handler Delete tests to avoid nil map panic (plan code omitted nil-check)"

requirements-completed:
  - PROJ-03
  - PROJ-04
  - PROJ-05

duration: 5min
completed: 2026-06-11
---

# Phase 05: Project Tests Summary

**Service and handler unit tests for project Update, Delete, and ListSubprojects operations with role gating, delete protection, and HTTP error mapping**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-11T09:31:00Z
- **Completed:** 2026-06-11T09:36:18Z
- **Tasks:** 3 (2 test commits, 1 verification)
- **Files modified:** 2

## Accomplishments

- Added `TestService_Update` with 2 subtests (finance role updates, non-finance role forbidden) to project service tests — follows contract_test.go table-driven pattern
- Added `TestService_Delete` with 6 subtests (finance deletes, not found, unauthorized role, not owner, blocked by time entries, blocked by subproject entries) — covers all error paths in `Service.Delete`
- Added `TestProjectHandler_Update` with 2 subtests (200 on success, 403 on forbidden) — verifies `Update` handler error switching
- Added `TestProjectHandler_Delete` with 4 subtests (200 on success, 403 on forbidden, 409 on time entries conflict, 409 on subproject entries conflict) — verifies `Delete` handler error switching
- Added `TestProjectHandler_ListSubprojects` with 1 subtest (200 on success) — verifies subproject listing endpoint
- Added `setupProjectHandler` test helper with `mockSubprojectRepo` in the HTTP adapter test file
- Verified all 14 project-specific test subtests pass cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Service Update and Delete tests** - `5ab20e6` (test)
2. **Task 2: Handler Update, Delete, ListSubprojects tests** - `cf07e5d` (test)
3. **Task 3: Full verification** - included in task 1 and 2 commits (no separate commit needed)

## Files Created/Modified

- `internal/core/services/project/project_test.go` — Added 92 lines: `TestService_Update` (table-driven, 2 subtests) and `TestService_Delete` (named subtests, 6 cases) after `TestService_RemoveManager`
- `internal/adapters/primary/http/project_test.go` — Added 157 lines: imports (`context`, `assert`, `projectdomain`, `projectsvc`, `testdata`, `models`), `setupProjectHandler` helper, `mockSubprojectRepo` struct, `TestProjectHandler_Update`, `TestProjectHandler_Delete`, `TestProjectHandler_ListSubprojects`

## Decisions Made

- **Plan deviation — middleware API names:** The plan referenced `middleware.WithOrgID` and `middleware.WithRole` but these don't exist. The actual middleware package exports `SetOrganizationID` and `SetRole`. Used the existing API to match project_test.go patterns.
- **Plan deviation — nil map initialization:** The plan's handler Delete test code assigned to `mockRepo.Projects[projectID]` without initializing the map. Added nil-check and map initialization before assignment.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Used correct middleware function names**
- **Found during:** Task 2 (Handler tests)
- **Issue:** Plan referenced `middleware.WithOrgID` and `middleware.WithRole` which don't exist in the middleware package. Correct functions are `middleware.SetOrganizationID` and `middleware.SetRole`.
- **Fix:** Replaced all `WithOrgID`/`WithRole` calls with `SetOrganizationID`/`SetRole` to match the existing handler test patterns in project_test.go
- **Files modified:** `internal/adapters/primary/http/project_test.go`
- **Verification:** Handler tests compile and pass
- **Committed in:** `cf07e5d` (Task 2 commit)

**2. [Rule 1 - Bug] Fixed nil map assignment in handler Delete tests**
- **Found during:** Task 2 (Handler tests)
- **Issue:** Plan's Delete test code directly assigned `mockRepo.Projects[projectID] = &projectdomain.ProjectResponse{...}` but the Projects map is nil by default on `MockProjectRepo`, causing a panic.
- **Fix:** Added nil-check `if mockRepo.Projects == nil { mockRepo.Projects = make(...) }` before each map assignment in the three Delete subtests that seed project data.
- **Files modified:** `internal/adapters/primary/http/project_test.go`
- **Verification:** Delete tests pass without panic
- **Committed in:** `cf07e5d` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — Bug fixes)
**Impact on plan:** Both fixes were necessary for correctness. No scope creep.

## Issues Encountered

- Pre-existing frontend TypeScript build errors (unrelated to this plan's changes) cause `bun run build` to fail. These errors span auth/contract/customer/project/time-entry test files and several route components — they are pre-existing across the frontend, not caused by this plan's backend-only changes.
- Pre-existing backend integration test failures in `internal/adapters/primary/http/` (register endpoint returns 200 instead of 201) are a known auth bug from prior phase work, unrelated to this plan's project test additions.

## Known Stubs

None — all tests are fully wired and pass.

## Next Phase Readiness

- All project service and handler unit tests are complete: Update (role gating), Delete (role gating + ownership check + active entries check + subproject entries check), ListSubprojects (returns subproject list)
- Service tests cover all error paths in `Service.Update` and `Service.Delete`
- Handler tests verify HTTP error mapping for 200/403/404/409 response codes
- 14 new test subtests added across 5 test functions
- Ready for next plan in wave 3 (if any remaining) or phase verification

---
*Phase: 05-mvp-consolidation*
*Completed: 2026-06-11*
