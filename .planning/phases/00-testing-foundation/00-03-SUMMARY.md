---
phase: 00-testing-foundation
plan: 03
subsystem: backend
tags:
  - testing
  - auth
  - time-entry
  - organization
  - middleware
  - models
requires: []
provides:
  - auth-service-tests
  - time-entry-service-tests
  - organization-service-tests
  - middleware-tests
  - model-validation-tests
affects: []
tech-stack:
  added: []
  patterns:
    - "Table-driven Go tests with testify assertions"
    - "Co-located _test.go files next to source"
    - "Mock repos from internal/core/services/testdata package"
key-files:
  created:
    - internal/core/services/auth/auth_test.go
    - internal/core/services/time_entry/time_entry_test.go
    - internal/core/services/organization/organization_test.go
    - internal/middleware/middleware_test.go
    - internal/models/models_test.go
  modified:
    - internal/core/services/testdata/mocks.go
key-decisions:
  - "Extend MockUserRepo with Memberships field for test pre-seeding (was returning nil)"
  - "Extend MockRefreshTokenRepo with Tokens map for FindByHash control (was returning nil)
"
  - "Extend MockTimeEntryRepo with PeriodLocked flag for period-locked test"
  - "Extend MockOrgMgmtRepo with Settings field for GetSettings pre-seeding"
requirements-completed: []
duration: 3 min
completed: 2026-05-18
---

# Phase 0 Plan 3: Service-Layer Tests Summary

Backend service-layer tests for Auth, Organization, Time Entry (full approval state matrix),
Auth middleware, and model validation — 5 test files totaling ~1200 lines of table-driven tests.

## Test Coverage

### Auth Service (`auth_test.go`) — 15 test cases
| Test | Cases |
|------|-------|
| TestService_Register | valid registration with org, duplicate email, weak password, invalid email, registration without org |
| TestService_Login | valid credentials, invalid password, inactive user, nonexistent email, login with username |
| TestService_Refresh | valid refresh token, nonexistent token, user not found for token |
| TestService_Bootstrap | bootstrap when no users exist, bootstrap when users exist |

### Time Entry Service (`time_entry_test.go`) — 28 test cases
| Test | Cases |
|------|-------|
| TestService_List | returns entries for org, empty org returns empty |
| TestService_Get | existing entry, nonexistent returns error |
| TestService_Create | valid entry creates draft, period locked returns error, invalid date format |
| TestService_Submit | owner submits draft, non-owner cannot submit, cannot submit already submitted, cannot submit approved |
| TestService_Approve | wg_manager/admin approve submitted, employee/manager/finance/customer forbidden, cannot approve draft, cannot approve already approved (8 cases) |
| TestService_Reject | wg_manager/admin reject submitted to draft, employee forbidden, cannot reject draft, cannot reject approved (5 cases) |
| TestService_Update | owner updates draft, cannot update submitted, non-owner cannot update |
| TestService_ListPending | returns entries for org and role, empty org returns empty |
| TestService_Delete | owner deletes draft, cannot delete submitted |

### Organization Service (`organization_test.go`) — 10 test cases
| Test | Cases |
|------|-------|
| TestService_Create | valid org, missing name error, generated slug, custom slug |
| TestService_Get | existing org, nonexistent org error |
| TestService_GetSettings | returns settings for org |
| TestService_UpdateSettings | finance allowed, employee forbidden, manager forbidden |

### Auth Middleware (`middleware_test.go`) — 5 test cases
| Test | Verification |
|------|-------------|
| TestAuth_MissingCookie | 401, next handler not called |
| TestAuth_InvalidToken | 401, next handler not called |
| TestAuth_ValidToken | 200, context values match claims |
| TestRequireRole_Allowed | 200 for matching role |
| TestRequireRole_Forbidden | 403 for non-matching role |

### Model Validation (`models_test.go`) — 31 test cases (5 new test functions + 1 existing preserved)
| Test | Valid | Invalid |
|------|-------|---------|
| TestRoleIsValid | employee, manager, finance, customer | admin, superuser, "", ceo |
| TestEntryStatusIsValid | draft, submitted, pending_manager, pending_finance, approved, rejected | deleted, "", pending |
| TestGovernanceModelIsValid | creator_controlled, unanimous, majority | democracy, "", dictatorship |
| TestProjectTypeIsValid | billable, internal | external, "" |
| TestExpenseCategoryIsValid | mileage, meal, accommodation, parking, travel_tickets, tolls, taxi, equipment, other | invalid, "" |

## Deviations from Plan

### [Rule 3 - Blocking] Extended mock repos in testdata for test pre-seeding
- **Found during:** Task 1 (auth tests)
- **Issue:** `MockUserRepo.GetMemberships` always returned nil, `MockRefreshTokenRepo.FindByHash` always returned nil — auth tests (login with memberships, refresh with valid token) could not be written without control over return values
- **Fix:** Added `Memberships` field to `MockUserRepo` and `Tokens` field to `MockRefreshTokenRepo` in `internal/core/services/testdata/mocks.go`. Added `PeriodLocked` flag to `MockTimeEntryRepo` and `Settings` field to `MockOrgMgmtRepo` for analogous testability
- **Files modified:** `internal/core/services/testdata/mocks.go`
- **Commit:** 04e3158

### [Rule 3 - Scope Deviation] List test omitted from organization service
- **Found during:** Task 3
- **Issue:** The plan specified `TestService_List` for organization, but the `organization.Service` has no `List` method — it only has `Create`, `Get`, `GetSettings`, `UpdateSettings`, `ListMembers`, `UpdateMemberRoles`, `DeactivateMember`
- **Resolution:** Tested `GetSettings` and `UpdateSettings` instead, which were the closest available methods with meaningful behavior to test
- **Commit:** 99986bc

### [Rule 3 - Scope Deviation] Time entry approve/reject state machine simplified
- **Found during:** Task 2
- **Issue:** The plan specified multi-step approval states (pending_manager → pending_finance → approved) but the actual `time_entry.Service.Approve` only transitions `submitted → approved` (single-step) with roles `wg_manager` or `admin`. Similarly `Reject` transitions `submitted → draft` (not to `rejected`). The plan's SubmitMonth method also doesn't exist
- **Resolution:** Tests written against actual code behavior. Role matrix covers all 4 non-allowed roles (employee, manager, finance, customer) returning ErrForbidden, and all statuses a wg_manager/admin cannot act on (draft, approved) returning ErrEntryNotSubmitted
- **Commit:** eceba1f

## Verification Results

```
ok  github.com/stefanoprivitera/hourglass/internal/core/services/auth           0.459s
ok  github.com/stefanoprivitera/hourglass/internal/core/services/time_entry      0.609s
ok  github.com/stefanoprivitera/hourglass/internal/core/services/organization    0.852s
ok  github.com/stefanoprivitera/hourglass/internal/middleware                    1.536s
ok  github.com/stefanoprivitera/hourglass/internal/models                        1.260s
```

**Total: 89 test cases (87 new + 2 pre-existing) — all PASS**

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 04e3158 | Auth service tests + middleware tests + mock extensions |
| 2 | eceba1f | Time entry service tests with full approval state matrix |
| 3 | 99986bc | Organization service tests + model validation tests |

## Self-Check: PASSED

- [x] All 5 test files created and compiled (`auth_test.go`, `time_entry_test.go`, `organization_test.go`, `middleware_test.go`, `models_test.go`)
- [x] `go test ./internal/core/services/auth/... ./internal/core/services/time_entry/... ./internal/core/services/organization/... ./internal/middleware/... ./internal/models/... -count=1` passes
- [x] Each task committed individually (3 commits)
- [x] SUMMARY.md created at `.planning/phases/00-testing-foundation/00-03-SUMMARY.md`
- [x] Existing `models_phase2_test.go` preserved (TestPhase2ModelsIncludeFlattenedEntryFields still passes)
