---
phase: 01-authorization
plan: 01
subsystem: auth
tags: [go, jwt, cookies, password-reset, crypto]

# Dependency graph
requires:
  - phase: 00-testing-foundation
    provides: Fixed auth service behavior (refresh rotation, cookie helpers)
provides:
  - Register endpoint sets auth_token and refresh_token cookies (matching Login/Bootstrap)
  - RegisterResponse includes Token, RefreshToken, ExpiresAt fields
  - Password reset codes use unbiased crypto/rand.Int distribution
affects:
  - 01-02 (frontend auth integration — register flow now receives cookies)

# Tech tracking
tech-stack:
  added: [math/big]
  patterns:
    - Token generation in Register follows Bootstrap token generation pattern
    - Cookie setting on Register follows Login/Bootstrap 3-line pattern

key-files:
  created: []
  modified:
    - internal/core/services/auth/auth.go — RegisterResponse extended, Register generates tokens
    - internal/adapters/primary/http/auth.go — Register handler sets cookies, returns 200
    - internal/core/services/password_reset/password_reset.go — generateResetCode uses unbiased distribution

key-decisions:
  - "Guard token generation behind orgID != uuid.Nil to avoid FK violation on refresh_tokens when registering without an organization"
  - "Added crypto/rand.Int with math/big for unbiased password reset code distribution (replacing modulo-biased rand.Read)"
  - "Register returns 200 instead of 201 to match Login/Bootstrap convention (response now includes tokens)"

requirements-completed: [AUTH-01, AUTH-02]

# Metrics
duration: ~3 min
completed: 2026-06-09
---

# Phase 1 Plan 1: Backend Auth Fixes Summary

**Register endpoint now sets auth_token/refresh_token cookies matching Login/Bootstrap, password reset codes use unbiased crypto/rand.Int distribution**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-06-09T23:29:05Z
- **Completed:** 2026-06-09T23:31:26Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Extended RegisterResponse with Token, RefreshToken, ExpiresAt fields matching LoginResponse shape
- Register now generates JWT and refresh tokens on successful registration (following Bootstrap pattern)
- Register handler sets auth_token and refresh_token HttpOnly cookies (matching Login/Bootstrap)
- Changed Register HTTP status from 201 to 200 (response now includes session tokens)
- Fixed password reset code generation to use unbiased crypto/rand.Int distribution (eliminates modulo bias)
- D-08 verified: password reset code is already discarded with `_` in the handler (no change needed)

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend Register service to generate and return auth tokens** - `d901a3e` (feat)
2. **Task 2: Fix Register handler cookies + password reset code entropy** - `aa2cbb9` (feat)

## Files Modified

- `internal/core/services/auth/auth.go` - RegisterResponse extended with Token/RefreshToken/ExpiresAt; Register method generates and stores tokens; token generation guarded behind orgID != uuid.Nil
- `internal/adapters/primary/http/auth.go` - Register handler adds 3-line cookie pattern (SetAccessTokenCookie + SetRefreshTokenCookie), returns 200 instead of 201
- `internal/core/services/password_reset/password_reset.go` - generateResetCode replaced with unbiased crypto/rand.Int distribution; added math/big import

## Decisions Made

- **Token guard for no-org registration:** Token generation only runs when `orgID != uuid.Nil`. The refresh_tokens table has a FK constraint on organization_id, and registering without an org would violate it. When no org is provided, RegisterResponse returns empty token fields.
- **Unbiased distribution:** Using `crypto/rand.Int(rand.Reader, big.NewInt(N))` per character eliminates the modulo bias from `int(v) % 62` where 256 % 62 != 0. Each of the 8 code characters has equal probability.
- **200 vs 201:** Register now returns 200 OK to match Login/Bootstrap convention since the response includes session tokens (semantic shift from "resource created" to "authenticated session established").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Guard token generation behind orgID != uuid.Nil**
- **Found during:** Task 2 (integration test verification)
- **Issue:** TestAuthIntegration/GetProfileWithoutOrg registers a user without an organization. The token generation code called `refreshTokenRepo.Add(ctx, user.ID, uuid.Nil, ...)` which caused a FK violation because `refresh_tokens.organization_id` references `organizations(id)`.
- **Fix:** Wrapped token generation in `if orgID != uuid.Nil { ... }` guard. When no org is specified, RegisterResponse is returned with empty Token/RefreshToken/ExpiresAt fields.
- **Files modified:** `internal/core/services/auth/auth.go`
- **Verification:** `go test -count=1 -timeout 300s ./internal/core/services/auth/...` passes (all subtests green)
- **Committed in:** `aa2cbb9` (amended into Task 2 commit)

---

**Total deviations:** 1 auto-fixed (bug)
**Impact on plan:** Minor — edge case where user registers without organization. Token generation correctly skipped when no org context exists. No scope creep.

## Issues Encountered

- **Refresh token FK constraint with nil orgID:** The `refresh_tokens` table requires a valid `organization_id` foreign key. When a test/user registers without specifying an organization, `orgID` is `uuid.Nil` which causes a FK violation. Fixed by guarding token generation behind `orgID != uuid.Nil`.

## Verification

- ✅ `go build ./...` passes
- ✅ `go test -count=1 -timeout 300s ./internal/core/services/auth/...` — all subtests pass
- ✅ `go test -count=1 -timeout 300s ./internal/core/services/password_reset/...` — all subtests pass
- ✅ Register handler sets auth_token and refresh_token cookies (D-01 satisfied)
- ✅ RegisterResponse contains token, refresh_token, expires_at fields matching LoginResponse shape
- ✅ Password reset codes generated without modulo bias (crypto/rand.Int)
- ✅ Password reset response does not leak code (D-08 — already compliant, verified)

## Next Phase Readiness

- Backend auth fixes complete — Register endpoint is now consistent with Login/Bootstrap
- Ready for Plan 01-02: frontend auth integration (register flow will now receive cookies from backend)
- Password reset hardening complete (entropy improved, code leak verified)

---

*Phase: 01-authorization*
*Completed: 2026-06-09*
