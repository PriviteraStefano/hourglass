# Auth E2E Testing Design

**Date:** 2026-04-28  
**Author:** AI

## Goal

Implement full end-to-end testing for the authentication cycle covering both backend (Go HTTP handlers) and frontend (Playwright E2E tests).

## Current State

- Backend auth implemented in `internal/adapters/primary/http/auth.go` and `internal/core/services/auth/auth.go`
- Frontend auth routes in `web/src/routes/(auth)/` (login, register, etc.)
- No existing auth tests
- SurrealDB running in Docker (in-memory mode)

## Auth Flow to Test

```
Register → Login → GetProfile → Refresh → Logout
```

## Architecture

### Backend: Go HTTP Tests

**File:** `internal/adapters/primary/http/auth_test.go`

Uses Go's `httptest` package with real SurrealDB test instance.

### Frontend: Playwright E2E Tests

**File:** `e2e/auth.spec.ts`

Uses Playwright to test full browser flow.

---

## Backend Tests

### Test Suite: Register

| Test | Input | Expected |
|------|-------|----------|
| `TestRegister_WithNewOrg` | Valid email, password, org name | 201 + user data |
| `TestRegister_WithInviteCode` | Valid invite code | 201 + org from invite |
| `TestRegister_InvalidEmail` | `notanemail` | 400 error |
| `TestRegister_WeakPassword` | `short` | 400 error |
| `TestRegister_DuplicateEmail` | Already registered email | 409 conflict |
| `TestRegister_DuplicateUsername` | Already taken username | 409 conflict |
| `TestRegister_MissingOrgAndInvite` | No org_name or invite_code | 400 error |

### Test Suite: Login

| Test | Input | Expected |
|------|-------|----------|
| `TestLogin_WithEmail_Success` | Valid email + password | 200 + cookies set |
| `TestLogin_WithUsername_Success` | Valid username + password | 200 + cookies set |
| `TestLogin_InvalidPassword` | Wrong password | 401 error |
| `TestLogin_NonExistentUser` | Unknown email | 401 error |
| `TestLogin_DeactivatedAccount` | Inactive user | 403 error |
| `TestLogin_InvalidIdentifierFormat` | Special characters in username | 400 error |

### Test Suite: GetProfile

| Test | Input | Expected |
|------|-------|----------|
| `TestGetProfile_Authenticated` | Valid token in header | 200 + user data |
| `TestGetProfile_Unauthenticated` | No token | 401 error |

### Test Suite: Refresh

| Test | Input | Expected |
|------|-------|----------|
| `TestRefresh_ValidToken` | Valid refresh cookie | 200 + new access token |
| `TestRefresh_InvalidToken` | Bad/expired token | 401 error |
| `TestRefresh_MissingCookie` | No refresh cookie | 400 error |

### Test Suite: Logout

| Test | Input | Expected |
|------|-------|----------|
| `TestLogout_WithRefreshToken` | Has refresh cookie | 204 + cookies cleared + token revoked |
| `TestLogout_WithoutRefreshToken` | No refresh cookie | 204 + cookies cleared |

### Test Suite: Bootstrap

| Test | Input | Expected |
|------|-------|----------|
| `TestBootstrap_FirstUser` | First user (no existing users) | 201 + admin token |
| `TestBootstrap_SubsequentUser` | User already exists | 409 conflict |

---

## Frontend Tests (Playwright)

### Install Dependencies

```bash
cd web
bun add -D @playwright/test
bunx playwright install chromium
```

### Configuration: `playwright.config.ts`

- Base URL: `http://localhost:3000` (vite dev server)
- Auth: Store cookies in `playwright/.auth/`
- Trace: on-first-retry

### Test File: `e2e/auth.spec.ts`

| Test | Flow | Expected |
|------|------|----------|
| `register with new org` | Fill form → Submit | Redirect to login, toast success |
| `register validation` | Submit empty form | Inline error messages |
| `login success` | Valid credentials → Submit | Redirect to dashboard, cookie set |
| `login invalid` | Wrong password | Error toast, stay on page |
| `logout` | Click logout | Redirect to login, cookies cleared |
| `protected route redirect` | Visit `/dashboard` without auth | Redirect to `/login` |
| `session persistence` | Login → Refresh page | Still authenticated |

---

## Test Infrastructure

### Database Setup

- Use same SurrealDB connection (in-memory)
- Test namespace: `hourglass_test`
- Each test should use unique identifiers to avoid conflicts

### Test Utilities

```go
// internal/adapters/primary/http/auth_test.go
func setupTestDB(t *testing.T) *sdb.DB {
    // Connect to test namespace
}

func cleanupTestUser(t *testing.T, db *sdb.DB, email string) {
    // Delete test user after test
}
```

### Test Utilities (Frontend)

```typescript
// e2e/utils.ts
async function loginAsUser(page: Page, credentials: LoginCredentials)
async function registerUser(credentials: RegisterCredentials)
async function logout(page: Page)
```

---

## Implementation Order

1. **Backend auth handler tests** - 13 tests covering all HTTP endpoints
2. **Setup Playwright** - Install, configure, create base config
3. **Frontend E2E auth tests** - 7 tests covering key flows
4. **CI integration** - Add to `make test`

---

## Success Criteria

- [ ] All 13 backend auth handler tests pass
- [ ] All 7 frontend E2E auth tests pass
- [ ] Tests run via `make test` (Go) and `bun playwright test` (Frontend)