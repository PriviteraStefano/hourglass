---
phase: 00-testing-foundation
plan: 01
subsystem: auth
tags: [go, testcontainers, auth, jwt, cookies, rate-limit, password-reset]

# Dependency graph
requires:
  - phase: 00-testing-foundation
    plan: 02
    provides: testcontainer infrastructure (SetupPackageContainer, SetupTestSchema)
provides:
  - Handler bug fixes (nil pointer guards, UUID validation, org context checks)
  - Service-level membership-not-found error for GetProfile
  - Refresh token rotation (revoke old hash, issue new token)
  - Unified cookie names (auth_token via cookies.go helpers)
  - 8-char alphanumeric password reset codes (not in response body)
  - Auth-specific rate limiters (5/min login, 3/min password-reset)
  - Configurable mock implementations (Memberships, FindActiveResets maps)
  - Testcontainer-backed auth integration tests (6 test cases)
  - Fixed TeardownTestSchema (verification_tables table added)
affects: [01-authorization, 03-service-tests, 04-handler-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Data-wrapped JSON responses decoded via {Data: *Type} wrapper structs in integration tests"
    - "Per-package container lifecycle via sync.Once + TestMain-style single parent test"

key-files:
  created:
    - internal/adapters/primary/http/auth_integration_test.go
  modified:
    - internal/adapters/primary/http/auth.go
    - internal/adapters/primary/http/unit.go
    - internal/adapters/primary/http/organization.go
    - internal/adapters/primary/http/password_reset.go
    - internal/core/services/auth/auth.go
    - internal/cookies/cookies.go
    - internal/core/services/password_reset/password_reset.go
    - internal/core/services/testdata/mocks.go
    - cmd/server/main.go
    - internal/adapters/secondary/postgres/exported_test_helpers.go

key-decisions:
  - "Bug fixes and auth security cleanup done in the same plan (D-04 + D-16 scope)"
  - "Integration tests use a single parent test with t.Run sub-tests to avoid container lifecycle issues with sync.Once + t.Cleanup"

patterns-established:
  - "Integration tests decode JSON via {Data: *Type} wrapper struct to unwrap RespondWithJSON envelope"
  - "Multiple integration tests in a package use one parent test function to share the testcontainer pool"

requirements-completed: [TEST-01]

# Metrics
duration: 131min
completed: 2026-06-08
---

# Phase 00 Plan 01: Auth Bug Fixes + Cleanup Summary

**Full auth cleanup: 4 known bugs fixed, refresh token rotation, cookie name unification, password reset hardening, rate limiting, mock repairs, and 6 testcontainer-backed integration tests**

## Performance

- **Duration:** 2h 11min
- **Started:** 2026-06-08T22:19:03Z
- **Completed:** 2026-06-09T00:30:00Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- **Bug 1 (GetMemberships nil pointer):** Added `if org == nil { continue }` guard in the memberships loop — prevents nil pointer dereference when an org can't be found
- **Bug 2 (GetProfile empty role/org_id):** Service returns `ErrMembershipNotFound` when membership is nil with non-nil orgID; handler returns 404 for this error
- **Bug 3 (ListMembers UUID parse 500):** Added UUID format validation in the handler, returns 400 for invalid UUIDs instead of 500
- **Bug 4 (ListMembers orgID 500):** Added `uuid.Nil` guard returning 400 when no organization context is available
- **Refresh token rotation:** Revoke old hash via `RevokeByHash`, issue new refresh token on each `POST /auth/refresh`
- **Cookie name unification:** `AccessTokenCookieName = "auth_token"` in cookies.go; all handlers use cookie helpers instead of hardcoded `http.SetCookie`
- **Password reset hardening:** 8-char alphanumeric codes (was 3-digit), removed `"code"` from response body
- **Auth-specific rate limiting:** login (5/min), password-reset (3/min) via separate `authRateLimiter` and `passwordResetRateLimiter` instances
- **Mock repairs:** All 5 broken mock methods (`MockOrgRepo.GetMembership`, `MockPasswordResetRepo.FindActiveByUserID`, `MockUnitRepo.GetDescendants/ListMembers`, `MockWorkingGroupRepo.ListMembers`) now use configurable map fields
- **6 integration tests** covering all bug scenarios, refresh rotation, password reset, and cookie verification
- **Teardown fix:** Added `verification_tokens` to the table drop list in `TeardownTestSchema`

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix 4 known auth bugs + fix broken mock implementations** - `994dabe` (fix)
2. **Task 2: Implement auth security cleanup** - `480ae57` (feat)
3. **Task 3: Write testcontainer-backed integration tests** - `e2a9334` (test)

**Plan metadata:** (next commit)

## Files Created/Modified

- `internal/adapters/primary/http/auth.go` - Bug 1 (GetMemberships nil check), Bug 2 handler fix (404 for ErrMembershipNotFound), all handlers use cookie helpers, Refresh handler sets refresh token cookie
- `internal/adapters/primary/http/unit.go` - Bug 3 (UUID parse validation in ListMembers)
- `internal/adapters/primary/http/organization.go` - Bug 4 (orgID nil guard in ListMembers)
- `internal/core/services/auth/auth.go` - Bug 2 service fix (ErrMembershipNotFound), refresh token rotation, RefreshResponse.RefreshToken field
- `internal/cookies/cookies.go` - AccessTokenCookieName = "auth_token"
- `internal/core/services/password_reset/password_reset.go` - 8-char alphanumeric reset code
- `internal/adapters/primary/http/password_reset.go` - Removed "code" from response body, removed unused `strconv` import
- `cmd/server/main.go` - authRateLimiter, passwordResetRateLimiter instances applied to auth/reset routes
- `internal/core/services/testdata/mocks.go` - Configurable map fields on all 5 broken mocks
- `internal/adapters/primary/http/auth_integration_test.go` - 6 testcontainer-backed integration tests
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - Added verification_tokens to teardown list

## Decisions Made

- Bug fixes and auth security cleanup done in the same plan (pre-split in ROADMAP but merged per D-04 + D-16)
- Integration tests use a single parent `TestAuthIntegration(t)` with `t.Run` sub-tests to work around `SetupPackageContainer`'s `sync.Once` + `t.Cleanup` lifecycle (pool closes when first test's cleanup fires)
- RespondWithJSON wraps all responses in `{"data": ...}` envelope — test helpers decode via `{Data *Type}` wrapper structs

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Fixed TeardownTestSchema missing verification_tokens table**
- **Found during:** Task 3 (Integration test execution)
- **Issue:** `TeardownTestSchema` didn't drop the `verification_tokens` table, causing `ERROR: relation "idx_verification_tokens_user_id" already exists` on re-migration in subsequent sub-tests
- **Fix:** Added `"verification_tokens"` to the table drop list
- **Files modified:** `internal/adapters/secondary/postgres/exported_test_helpers.go`
- **Verification:** All 6 integration tests pass with fresh schema per sub-test
- **Committed in:** `e2a9334` (Task 3)

**2. [Rule 2 - Missing Critical] Integration tests needed data-envelope JSON decoding**
- **Found during:** Task 3 (Integration test execution)
- **Issue:** `RespondWithJSON` wraps responses in `{"data": payload}`, but test helpers decoded directly into response types, resulting in empty fields (token, refresh_token, etc.)
- **Fix:** All JSON decoders in test helpers now use `{Data *ResponseType}` wrapper structs
- **Files modified:** `internal/adapters/primary/http/auth_integration_test.go`
- **Verification:** All 6 integration tests pass with correct field values
- **Committed in:** `e2a9334` (Task 3)

**3. [Rule 2 - Missing Critical] Registration doesn't set auth cookies**
- **Found during:** Task 3 (Integration test execution)
- **Issue:** The register endpoint returns user data but doesn't set auth cookies. Authenticated routes (memberships, profile) require cookies from login. Tests calling register then immediately making authenticated requests got 401.
- **Fix:** Added `registerAndLogin` helper that registers THEN logs in to populate the cookie jar
- **Files modified:** `internal/adapters/primary/http/auth_integration_test.go`
- **Verification:** Authenticated endpoint tests pass
- **Committed in:** `e2a9334` (Task 3)

**4. [Rule 3 - Blocking] Integration tests need shared container lifecycle**
- **Found during:** Task 3 (Integration test design)
- **Issue:** `SetupPackageContainer` registers `t.Cleanup` on the first test's `t` via `sync.Once`. Subsequent tests get a closed pool. Separate `TestAuthIntegration_*` functions failed with "closed pool".
- **Fix:** Restructured all tests as `t.Run` sub-tests under a single `TestAuthIntegration` parent test
- **Files modified:** `internal/adapters/primary/http/auth_integration_test.go`
- **Verification:** All sub-tests run sequentially sharing the same container pool
- **Committed in:** `e2a9334` (Task 3)

---

**Total deviations:** 4 auto-fixed (3 missing critical, 1 blocking)
**Impact on plan:** All auto-fixes were necessary for test correctness and stability. No scope creep. The test file structure (single parent test) is a pattern that should be documented for future integration test packages.

## Issues Encountered

- `http.Client.Jar` in `net/http/cookiejar` does handle `Set-Cookie` headers from `httptest.Server` responses correctly. Cookies set with `cookies.SetAccessTokenCookie` (HttpOnly, SameSite=Strict, Secure=auto) work correctly in test environments where `secure=false`.
- `cookies.GetRefreshTokenFromCookie` was used in the Logout handler instead of `r.Cookie("refresh_token")` — this is part of the consolidation.
- Rate limiter instances created at handler registration time in `main.go` need `Handle` (not `HandleFunc`) because `Middleware()` returns `http.Handler`, not `func(ResponseWriter, *Request)`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Auth bugs fixed and verified with integration tests
- Auth security hardening complete (rotation, cookies, rate limiting, password reset)
- Mock implementations now configurable for downstream service tests
- Ready for Plan 03 (Service test rewrite) and Plan 04 (Handler test rewrite)
- Next: `.planning/phases/00-testing-foundation/00-03-PLAN.md`

## Self-Check: PASSED

- [x] SUMMARY.md created
- [x] All 3 tasks committed (994dabe, 480ae57, e2a9334)
- [x] Integration test file created (auth_integration_test.go)
- [x] All 11 modified files exist
- [x] `go build ./...` passes
- [x] All integration tests pass (6/6)
- [x] All unit tests pass
- [x] Smoke test passes

---

*Phase: 00-testing-foundation*
*Completed: 2026-06-08*
