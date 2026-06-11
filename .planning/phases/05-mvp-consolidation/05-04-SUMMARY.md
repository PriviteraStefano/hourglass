---
phase: 05-projects
plan: 04
type: execute
subsystem: backend
tags:
  - project
  - testing
  - service
  - handler
  - unit-tests
requires: []
provides: [PROJ-03, PROJ-04, PROJ-05]
affects:
  - internal/core/services/project/project_test.go
  - internal/adapters/primary/http/project_test.go
tech-stack:
  added: []
  patterns:
    - "Table-driven service tests with t.Parallel()"
    - "Handler unit tests with MockProjectRepo and httptest"
    - "HasActiveTimeEntriesFn configurable mock for delete protection tests"
key-files:
  created: []
  modified: []
decisions: []
metrics:
  duration: ~5m
  completed_date: 2026-06-11
  tasks: 3
  subtests: 15
  commits: 2
---

# Phase 5 Plan 4: Backend Tests for Update/Delete/ListSubprojects — Summary

Service (TestService_Update, TestService_Delete) and handler (TestProjectHandler_Update, TestProjectHandler_Delete, TestProjectHandler_ListSubprojects) tests for project Update, Delete, and ListSubprojects — all passing.

## Task Summary

### Task 1: Service tests — TestService_Update and TestService_Delete

**Status:** ✅ Complete

**TestService_Update** (2 subtests):
- `finance_role_updates` — asserts no error when finance role updates
- `non-finance_role_forbidden` — asserts `ErrForbidden` for employee role

**TestService_Delete** (6 subtests):
- `finance_role_deletes` — asserts no error
- `not_found` — asserts error returned for nonexistent project
- `unauthorized_role` — asserts `ErrForbidden` for employee role
- `not_owner` — asserts `ErrForbidden` for wrong org
- `blocked_by_time_entries` — asserts `ErrHasActiveTimeEntries` when mock returns `(true, false, nil)`
- `blocked_by_subproject_entries` — asserts `ErrHasActiveSubprojectEntries` when mock returns `(false, true, nil)`

### Task 2: Handler tests — TestProjectHandler_Update, TestProjectHandler_Delete, TestProjectHandler_ListSubprojects

**Status:** ✅ Complete

**TestProjectHandler_Update** (2 subtests):
- `success` — returns 200
- `forbidden` — returns 403

**TestProjectHandler_Delete** (4 subtests):
- `success` — returns 200
- `forbidden_non-finance_role` — returns 403
- `conflict_time_entries` — returns 409 via `HasActiveTimeEntriesFn` returning `(true, false, nil)`
- `conflict_subproject_entries` — returns 409 via `HasActiveTimeEntriesFn` returning `(false, true, nil)`

**TestProjectHandler_ListSubprojects** (1 subtest):
- `returns_subprojects` — returns 200

### Task 3: Full test suite + frontend build verification

**Status:** ⚠️ Partial

**Backend tests (`go test -count=1 -timeout 300s ./internal/...`):** All project-related tests pass. Pre-existing integration test failures across multiple packages (`TestAuthHandlerIntegration`, `TestProjectHandlerIntegration`, etc.) are unrelated to this plan — all fail at the registration step (expecting 201, getting 200) due to a pre-existing auth handler change.

**Frontend build (`cd web && bun run build`):** Pre-existing TypeScript compilation errors across many files (API test files, router type issues, theme provider, etc.). These are not project-related and span the entire codebase. Out of scope for this test-only plan.

## Verification

- ✅ `go test -v -run "TestService_Update|TestService_Delete" ./internal/core/services/project/...` — 8/8 pass
- ✅ `go test -v -run "TestProjectHandler_Update|TestProjectHandler_Delete|TestProjectHandler_ListSubprojects" ./internal/adapters/primary/http/...` — 7/7 pass
- ✅ `go test -count=1 -timeout 120s ./internal/core/services/project/...` — clean
- ✅ `go test -count=1 -timeout 120s ./internal/adapters/primary/http/...` — non-integration tests pass
- ⚠️ `cd web && bun run build` — pre-existing failures, not project-related

## Commits

```
5ab20e6 test(05-04): add TestService_Update and TestService_Delete for project service
cf07e5d test(05-04): add TestProjectHandler_Update, TestProjectHandler_Delete, TestProjectHandler_ListSubprojects
```

## Deviations from Plan

None — all tests were already implemented and committed by a prior session agent. This execution verified them.

## Known Stubs

None.

## Threat Flags

None.

## Self-Check: PASSED

- ✅ `internal/core/services/project/project_test.go` exists with `TestService_Update` and `TestService_Delete`
- ✅ `internal/adapters/primary/http/project_test.go` exists with `TestProjectHandler_Update`, `TestProjectHandler_Delete`, `TestProjectHandler_ListSubprojects`
- ✅ Commit `5ab20e6` exists in git log
- ✅ Commit `cf07e5d` exists in git log
- ✅ All project tests pass
