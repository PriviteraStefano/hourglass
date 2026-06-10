# Plan 01-03: End-to-End Auth Verification

**Phase:** 01-authorization
**Duration:** ~5 min
**Tasks:** 2

## Task Results

### Task 1: Automated Backend Verification — PASS

All 11 curl-based checks passed:
- All 6 seed users login successfully with Set-Cookie headers ✓
- Register returns 200 with Set-Cookie (auto-login) ✓
- Login validation errors return correct 4xx ✓
- Cookie refresh flow (POST /auth/refresh) returns new tokens ✓
- GET /auth/me returns profile with role and org_id ✓
- GET /auth/memberships returns array without panic ✓
- POST /auth/password-reset/request does NOT leak code ✓

### Task 2: Manual Frontend Verification — PASS

User confirmed all frontend flows work:
- Login as seed user redirects to /time-entries ✓
- OrgSwitcher shows real org names ✓
- Register + auto-login works ✓
- Login form validation shows errors ✓
- Logout redirects to /login ✓

## Deviation: Org Creator Role

**Found during:** Human verification (Task 2)

**Issue:** Org creator was assigned `"employee"` role, preventing full access.

**Fix:** Register and Bootstrap now assign `"manager"` role when the user creates a new organization (vs `"employee"` when joining an existing org). Affected files:
- `internal/core/services/auth/auth.go` — Register and Bootstrap methods
- `internal/core/services/auth/auth_integration_test.go` — Updated test expectations

**Committed in:** `f162830`

## Verification Report

See `01-03-VERIFICATION.md` for detailed curl command results.
