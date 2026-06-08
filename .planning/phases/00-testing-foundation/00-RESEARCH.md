# Phase 0: Testing Foundation — Research

**Researched:** 2026-06-08
**Domain:** Go backend test infrastructure, auth bug fixes, security hardening
**Confidence:** HIGH

## Summary

Phase 0 reboots the test infrastructure from the failed SurrealDB era to a fully self-contained PostgreSQL-backed test suite using testcontainers-go. It simultaneously fixes all known auth bugs and addresses auth security concerns (refresh token rotation, cookie name consistency, password reset hardening, rate limiting) before any feature phases begin.

The key architectural insight is that the existing `TestPool`/`SetupTestSchema`/`TeardownTestSchema` patterns in `exported_test_helpers.go` are a good foundation — testcontainers integration replaces just the pool source while keeping the schema lifecycle and seed functions. The hybrid container strategy (D-08) balances isolation with performance: one PostgreSQL container per package, each test function gets its own schema via the existing `SetupTestSchema`/`TeardownTestSchema` pattern.

The auth bugs fall into two categories: (1) nil-pointer dereferences when repository returns `(nil, nil)` instead of an error, and (2) missing error handling that causes 500s instead of proper HTTP error responses. The auth cleanup scope extends beyond the 4 known bugs to include refresh token rotation, cookie name unification, password reset code removal from response body, and stricter rate limiting for auth endpoints.

**Primary recommendation:** Implement testcontainers infrastructure FIRST (D-01), then fix auth bugs with testcontainer-backed tests (D-02), then rewrite service/handler integration tests (D-05), then E2E verification (D-06), with a separate bug buffer plan for major issues discovered (D-14/D-15).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Testcontainers setup BEFORE auth bug fixes (reverse ROADMAP order). Infrastructure must exist before tests can verify fixes.
- **D-02:** Auth bug fixes are test-driven — write failing testcontainer-backed test that reproduces the bug, then fix, then watch test pass.
- **D-03:** Testcontainers setup and auth bug fixes remain separate plans. Clean dependency chain.
- **D-04:** All 4 known auth bugs fixed in one plan (not split per-bug).
- **D-05:** Post-auth order: service-layer test rewrite → handler integration test rewrite → E2E verification.
- **D-06:** 05-bug-buffer kept as a separate plan (not merged into auth fix plan).
- **D-07:** testcontainers-go replaces TestPool entirely. No more `DATABASE_URL` dependency for tests. Fully self-contained test suite.
- **D-08:** Hybrid container strategy: one testcontainers PostgreSQL instance per package, each test function gets its own schema via existing SetupTestSchema/TeardownTestSchema pattern.
- **D-09:** ALL tests use testcontainers: service tests, handler tests, repository tests, smoke test (`cmd/server/main_test.go`), and Playwright E2E.
- **D-10:** Hybrid approach — keep existing in-memory mock tests for pure business logic (validation, error mapping). Use testcontainers for tests that exercise repository interaction.
- **D-11:** Fix broken mock implementations now (MockOrgRepo.GetMembership returning nil, MockPasswordResetRepo.FindActiveByUserID returning not found, etc.).
- **D-12:** New testcontainers-based integration tests go in separate `_integration_test.go` files (e.g., `auth_integration_test.go`). Keeps mock unit tests and integration tests clearly separated.
- **D-13:** Minor bugs discovered during test rewrite: fix immediately inline.
- **D-14:** Major/complex bugs: document, t.Skip() the failing test with a reference to the logged bug, batch to 05-PLAN.
- **D-15:** 05-PLAN is a dedicated bug-fixing buffer that includes a human-in-the-middle review loop for major bugs.
- **D-16:** Full auth cleanup in Phase 0 — not just the 4 known bugs:
  - Fix `/auth/memberships` nil pointer
  - Fix `/auth/me` returning empty role/org_id
  - Fix `/units/{id}/members` 500
  - Fix `/organizations/members` 500
  - Implement refresh token rotation (revoke old hash, issue new)
  - Fix cookie name mismatch (access_token vs auth_token)
  - Fix password reset code leak (don't return in body, increase entropy)
  - Add stricter rate limiting for auth endpoints (login, password reset)

### the agent's Discretion
- (None specified in CONTEXT.md)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | Known auth bugs fixed | Root cause analysis documented below for all 4 bugs |
| TEST-02 | testcontainers-go wired for isolated PostgreSQL per test run | Testcontainers postgres module v0.32+ confirmed working; docker available |
| TEST-03 | Service-layer tests rewritten against PostgreSQL | Existing seeder functions reusable; TestPool replacement pattern documented |
| TEST-04 | Handler integration tests rewritten for PostgreSQL | Existing httptest.NewServer pattern confirmed; testcontainer pool injection pattern documented |
| TEST-05 | All bugs discovered during test rewrite fixed | Bug buffer plan (PLAN-05) with human-in-the-middle review loop |
| TEST-06 | Playwright E2E verified against PostgreSQL backend | Backend must serve from testcontainer-managed DB; E2E tests unaffected |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Testcontainers container lifecycle | Test infrastructure | — | `exported_test_helpers.go` owns container start/stop |
| Schema per test function | Test infrastructure | — | `SetupTestSchema`/`TeardownTestSchema` already handle schema lifecycle |
| Auth bug fixes | Backend API | — | Bugs are in service layer and HTTP handler layer |
| Refresh token rotation | Backend service | HTTP handler | Service logic + handler cookie setting |
| Cookie name unification | HTTP handler | Cookie helpers | Handler sets cookies, middleware reads them; both must agree |
| Password reset hardening | Backend service | HTTP handler | Remove code from response body + increase entropy |
| Rate limit tightening | Middleware | Backend API | Global rate limiter needs path-aware auth-specific limits |
| Service test rewrite | Backend service | Test infrastructure | New `_integration_test.go` files per service |
| Handler test rewrite | HTTP handler | Test infrastructure | New `_integration_test.go` files per handler |
| Mock repairs | Test infrastructure | — | Fix in `mocks.go` for broken methods |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| GitHub.com/testcontainers/testcontainers-go | v1.35+ | Docker container lifecycle for Go tests | De facto standard for Go integration testing with databases |
| GitHub.com/testcontainers/testcontainers-go/modules/postgres | v0.35+ | PostgreSQL-specific container module | Provides `postgres.Run()`, `ConnectionString()`, snapshot support |
| GitHub.com/jackc/pgx/v5 | v5.10.0 (existing) | PostgreSQL driver | Already in go.mod; test will use pgxpool.New(ctx, connStr) |
| GitHub.com/stretchr/testify | v1.11.1 (existing) | Test assertions | Already used throughout; `require.NoError`, `assert.Equal` patterns |

**Installation:**
```bash
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
```

**Version verification:**
```bash
go list -m github.com/testcontainers/testcontainers-go
go list -m github.com/testcontainers/testcontainers-go/modules/postgres
```

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testcontainers-go | Embedded PostgreSQL (e.g., `cockroachdb/cockroach-go/v2/testserver`) | Embedded PG must match schema behavior; testcontainers uses real Docker PostgreSQL images |
| One container per package | One container per test | Per-test is too slow (Docker start ~5-10s); per-package with schema-per-test is the documented sweet spot |

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/testcontainers/testcontainers-go` | Go module | 4+ yrs | 150M+ | github.com/testcontainers/testcontainers-go | OK | Approved |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | Go module | 4+ yrs | 50M+ | github.com/testcontainers/testcontainers-go | OK | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### Testcontainers Integration Pattern

```
exported_test_helpers.go
├── TestPool(t) *pgxpool.Pool          ← REPLACED by setupTestContainerPool()
├── SetupTestSchema(t, pool)            ← UNCHANGED (applies migrations)
├── TeardownTestSchema(t, pool)         ← UNCHANGED (drops tables)
└── seed* functions                     ← UNCHANGED (data seeding)
```

New pattern:

```
testcontainers_test_helper.go (new shared file)
├── packageTestPool *pgxpool.Pool       ← sync.Once per package
├── packageTestContainer *postgres.PostgresContainer  ← for cleanup
├── SetupPackageContainer(t testing.TB)  ← started once per TestMain
└── SetupTestSchema(t, pool)            ← called per test function

Each _integration_test.go file:
├── TestMain(m *testing.M)              ← starts container once
├── TestFunction(t *testing.T)          ← SetupTestSchema + TeardownTestSchema
```

**Key files to create/modify:**

1. **NEW:** `internal/adapters/secondary/postgres/test_setup.go` — Package-level container startup helper; exports `SetupPackageContainer` and `PackageTestPool`
2. **MODIFY:** `internal/adapters/secondary/postgres/exported_test_helpers.go` — Replace `TestPool(t)` with `SetupPackageContainer`; keep `SetupTestSchema`/`TeardownTestSchema` and seed functions unchanged
3. **NEW:** Per-package `_integration_test.go` files in each service and handler directory
4. **MODIFY:** `cmd/server/main_test.go` — Use new testcontainer pool instead of `TestPool`

### Per-Package Container Lifecycle (D-08)

```go
// In internal/adapters/secondary/postgres/test_setup.go:

var (
    packagePool     *pgxpool.Pool
    packageContainer *postgres.PostgresContainer
    poolOnce        sync.Once
)

func SetupPackageContainer(t testing.TB) *pgxpool.Pool {
    t.Helper()
    poolOnce.Do(func() {
        ctx := context.Background()
        ctr, err := postgres.Run(ctx,
            "postgres:16-alpine",
            postgres.WithDatabase("hourglass_test"),
            postgres.WithUsername("hourglass"),
            postgres.WithPassword("hourglass"),
            postgres.BasicWaitStrategies(),
        )
        if err != nil {
            t.Fatalf("failed to start postgres container: %v", err)
        }
        packageContainer = ctr

        connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
        if err != nil {
            t.Fatalf("failed to get connection string: %v", err)
        }

        pool, err := pgxpool.New(ctx, connStr)
        if err != nil {
            t.Fatalf("failed to create pool: %v", err)
        }
        packagePool = pool

        // Register cleanup (terminates container at end of package)
        t.Cleanup(func() {
            pool.Close()
            if err := testcontainers.TerminateContainer(packageContainer); err != nil {
                t.Logf("failed to terminate container: %v", err)
            }
        })
    })
    return packagePool
}
```

### Schema Lifecycle per Test Function

```go
func TestSomething_Integration(t *testing.T) {
    pool := postgres.SetupPackageContainer(t)  // starts container once per package
    postgres.SetupTestSchema(t, pool)           // applies migrations fresh
    t.Cleanup(func() {
        postgres.TeardownTestSchema(t, pool)    // drops all tables
    })

    // ... test logic with pool ...
}
```

### Per-Package TestMain

For packages that need testcontainer-backed tests, use `TestMain`:

```go
// internal/core/services/auth/auth_integration_test.go
package auth

import (
    "os"
    "testing"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
    testPool = postgres.SetupPackageContainer(m)
    code := m.Run()
    os.Exit(code)
}
```

### Anti-Patterns to Avoid

- **`t.Parallel()` with shared container + schema teardown:** Parallel tests using the same schema will race. Since D-08 says one container per package with schema-per-test, do NOT use `t.Parallel()` with schema-dependent tests within the same package unless using snapshots.
- **`panic` from nil pointer in handler:** Current bug pattern — `org, _` pattern discards error and doesn't check nil. Always check `if org == nil` before accessing `.ID`.
- **Using `_` to discard repository errors:** Errors from repository calls should be checked, not discarded. The bugs in `/auth/memberships` and `/auth/me` both stem from this pattern.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Postgres container lifecycle | Custom Docker SDK integration | testcontainers-go postgres module | Handles container start, health checks, port mapping, cleanup (Ryuk), and connection string generation |
| Refresh token hashing | Custom hashing | `sha256` from `crypto/sha256` (already used) | Already implemented in `internal/auth/auth.go` as `HashRefreshToken` |
| Password hashing | Custom hasher | bcrypt via `golang.org/x/crypto` | Already used in `internal/auth/auth.go` |

## Common Pitfalls

### Pitfall 1: Nil Pointer on Discarded Repository Errors

**What goes wrong:** Handler calls `service.GetOrgByID(ctx, orgID)` and discards the error with `org, _ := ...`. The repository can return `(nil, nil)` for "not found", causing nil pointer dereference when `org.ID` is accessed.

**Where it happens:**
- `internal/adapters/primary/http/auth.go:357` — `org, _ := h.authService.GetOrgByID(ctx, m.OrganizationID)` then `org.ID` on line 369

**Why it happens:** The repository's `GetByID` method has an ambiguous contract — it can return either `(nil, error)` for real errors or `(nil, nil)` for not-found. Callers must handle both.

**How to avoid:** Always check `if org != nil` before accessing fields, even when discarding the error. Or change the repository contract to return a sentinel error for not-found consistently.

### Pitfall 2: Docker Dependency for Test Execution

**What goes wrong:** Tests fail or are skipped when Docker is not running.

**Why it happens:** testcontainers requires a running Docker daemon. If CI or developer machine doesn't have Docker, all integration tests will be skipped.

**How to avoid:** The `SetupPackageContainer` helper should provide a clean `t.Skip("Docker not available")` message. The Makefile should document the Docker requirement.

**Warning signs:** CI pipeline shows all integration tests as skipped/passed without actually testing against PostgreSQL.

### Pitfall 3: Schema Teardown Collision in Shared Container

**What goes wrong:** Two test functions in the same package both call `TeardownTestSchema`, and the second one fails because tables were already dropped.

**Why it happens:** Using `t.Cleanup` with shared container and per-function schema setup/teardown is correct only if each function creates its own schema. The current `SetupTestSchema` applies to the `public` schema, which means parallel tests will collide.

**How to avoid:** Use per-function schemas via PostgreSQL schema namespacing, or use the existing `SetupTestSchema`/`TeardownTestSchema` with sequential tests only (no `t.Parallel()`).

### Pitfall 4: Refresh Token Not Rotated Leaks Session

**What goes wrong:** A stolen refresh token can be used indefinitely until it expires (7 days).

**Why it happens:** The `Refresh` method (auth.go:317) reads the old hash, generates a new access token, but never revokes the old refresh token or issues a new one.

**How to avoid:** Always revoke the old hash after successful refresh (`s.refreshTokenRepo.RevokeByHash(ctx, hash)`) and issue a new refresh token with a new hash.

### Pitfall 5: Password Reset Code in Response Body

**What goes wrong:** The reset code is returned in the `POST /auth/password-reset/request` response body.

**Why it happens:** The password reset handler (password_reset.go:54-58) explicitly includes `"code": code` in the JSON response for convenience during development.

**How to avoid:** Remove the code from the response body entirely. Only deliver it out-of-band (email/SMS). Increase entropy from 3-digit numeric to 6+ alphanumeric characters.

## Auth Bug Root Cause Analysis

### Bug 1: `/auth/memberships` Nil Pointer

**File:** `internal/adapters/primary/http/auth.go:341-377`
**Root cause:** Handler calls `org, _ := h.authService.GetOrgByID(ctx, m.OrganizationID)` at line 357. The `authService.GetOrgByID` delegates to `orgRepo.GetByID`. If `orgRepo.GetByID` returns `(nil, nil)` (the repo returns nil for not-found without an error), then `org` is nil. Line 369 accesses `org.ID.String()` — nil pointer dereference.

**Fix approach:** Check `if org != nil` before accessing fields, or skip memberships where the org is nil. Also fix the upstream `GetByID` to consistently return sentinel errors.

### Bug 2: `/auth/me` Empty Role/OrgID

**File:** `internal/core/services/auth/auth.go:298-315`
**Root cause:** `GetProfile` calls `s.orgRepo.GetMembership(ctx, userID, orgID)`. If the membership is not found (returns `nil, nil`), the response gets an empty `Membership` struct with empty `Role` (line 361-362 in `buildUserWithMembershipPtr`). The handler returns this as valid data.

**Fix approach:** When `orgID != uuid.Nil` but `GetMembership` returns nil, return an error (e.g., `ErrMembershipNotFound`). This forces the frontend to re-authenticate.

### Bug 3: `/units/{id}/members` 500

**File:** `internal/adapters/primary/http/unit.go:209-225`
**Root cause:** `h.service.ListMembers(ctx, unitID)` delegates to `s.repo.ListMembers(ctx, unitID)` which delegates to `members.ListByUnit(ctx, unitID)`. In `unit_member_repository.go:24-27`, `uuid.Parse(unitID)` fails if the unitID is not a valid UUID string, returning an error. The handler wraps ALL errors as 500 at line 220-222, including UUID parse errors which should be 400.

**Fix approach:** Differentiate between parse errors (400 Bad Request) and actual errors (500) in the handler. Check UUID parse errors early or catch them specifically.

### Bug 4: `/organizations/members` 500

**File:** `internal/adapters/primary/http/organization.go:151-158`
**Root cause:** The handler calls `h.service.ListMembers(r.Context(), middleware.GetOrganizationID(r.Context()))`. If `middleware.GetOrganizationID(r.Context())` returns `uuid.Nil` (because the JWT token has no org_id, or the user isn't associated with an org), the database query succeeds but returns no members. A 500 here could arise from other conditions: a database connection issue, a nil pointer in the org handler wiring, or the `ListMembers` query unexpectedly failing.

The most likely cause: If the JWT was generated without an org_id (user registered via invite code flow with invalid/missing org), `middleware.GetOrganizationID` returns `uuid.Nil`. The handler at line 152 passes this to the service. The `OrganizationManagementRepository.ListMembers` executes a query with `orgID = uuid.Nil` — which returns no rows, creating an empty list. This should succeed (returns `[]orgdomain.Member{}`). The 500 might actually be a transient issue from a missing migration table or from the mock breaking in a different way.

**Fix approach:** Add a guard in the handler: if `orgID == uuid.Nil`, return 400 with "no organization context". This provides a clear error instead of a potentially confusing 500.

## Auth Cleanup Analysis

### Refresh Token Rotation

**Current behavior (auth.go:317-355):** `Refresh` validates the old token hash, reads user/membership data, generates a NEW access token, returns it. The old refresh token remains valid. No new refresh token is issued.

**Desired behavior:**
1. Validate old token hash ✓ (already done)
2. **REVOKE** old hash: `s.refreshTokenRepo.RevokeByHash(ctx, hash)` — new step
3. Generate NEW refresh token: `s.tokenService.GenerateRefreshToken()` ✓ (already exists)
4. Hash and store new refresh token: `s.refreshTokenRepo.Add(ctx, userID, orgID, newHash, expiresAt)` ✓ (already exists)
5. Return both new access token AND new refresh token — update response and handler

**Impact:** The `RefreshResponse` struct needs a new `RefreshToken` field. The handler needs to set a new `refresh_token` cookie. The frontend's cookie jar handles this automatically.

### Cookie Name Unification

**Current state:**
- `internal/cookies/cookies.go:8`: `AccessTokenCookieName = "access_token"`
- `internal/adapters/primary/http/auth.go:123`: `Name: "auth_token"`
- `internal/middleware/middleware.go:25`: `r.Cookie("auth_token")` — middleware reads "auth_token"

**Problem:** The `cookies.go` helper `SetAccessTokenCookie` sets a cookie named `"access_token"`, but the handler and middleware both use `"auth_token"`. The helpers are effectively dead code.

**Fix approach:**
1. Change `AccessTokenCookieName = "auth_token"` in cookies.go
2. Replace all hardcoded `"auth_token"` cookie names in `auth.go` handler with `cookies.SetAccessTokenCookie(w, token, secure)` calls
3. Replace all hardcoded `"refresh_token"` cookie names in `auth.go` handler with `cookies.SetRefreshTokenCookie(w, token, secure)` calls
4. The `Logout` handler should use `cookies.ClearAuthCookies(w)`

### Password Reset Security (D-16)

**Current state (password_reset.go:101-108):**
- `generateResetCode()` generates 3 bytes, converts each to digit 0-9 (3-digit code, 1000 combinations)
- Handler (password_reset.go:54-58) returns `"code": code` in response body

**Fix approach:**

1. **Increase entropy:** Change to 6+ alphanumeric characters:
```go
const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
func generateResetCode() string {
    b := make([]byte, 8)
    rand.Read(b)
    result := make([]byte, len(b))
    for i, v := range b {
        result[i] = charset[int(v)%len(charset)]
    }
    return string(result)
}
```

2. **Remove from response body:**
```go
// password_reset.go handler Request()
// BEFORE:
api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
    "message":    "reset code sent",
    "code":       code,
    "expires_at": expiresAt,
})
// AFTER:
api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
    "message":    "reset code sent",
    "expires_at": expiresAt,
})
```

3. **Rate limit verification attempts** — see rate limiting section below.

### Rate Limiting for Auth Endpoints

**Current state (ratelimit.go):**
- Global rate limiter: 20/min anonymous, 100/min authenticated
- Applied as outermost middleware layer: `TryAuth(authService, rateLimiter.Middleware(loggingCORS))`
- No endpoint-specific differentiation

**Current issue with auth endpoints:**
- `POST /auth/login` — 20/min anonymous limit (shared with all other anonymous endpoints)
- `POST /auth/password-reset/request` — 20/min anonymous limit
- `POST /auth/password-reset/verify` — could be hammered at 20/min

**Fix approach:**
Option A: Create a separate rate limiter instance for auth endpoints and apply it as inner middleware specifically on auth routes.
Option B: Modify the existing rate limiter to check HTTP method+path and apply stricter limits.

**Recommended approach (Option A):**
```go
// In cmd/server/main.go:
authRateLimiter := middleware.NewRateLimiter(5, 100) // 5/min anonymous, 100/min authenticated

// Apply to auth routes specifically:
mux.HandleFunc("POST /auth/login", authRateLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
mux.HandleFunc("POST /auth/password-reset/request", authRateLimiter.Middleware(http.HandlerFunc(passwordResetHandler.Request)))
mux.HandleFunc("POST /auth/password-reset/verify", authRateLimiter.Middleware(http.HandlerFunc(passwordResetHandler.Verify)))
```

This way, auth endpoints get 5 req/min anonymous rate, while the global limiter stays at 20/100 for everything else.

## Broken Mock Implementations (D-11)

The following mock methods always return `nil, nil` or hardcoded values, preventing realistic testing:

| Mock | Method | Returns | Impact |
|------|--------|---------|--------|
| `MockOrgRepo.GetMembership` | line 252 | `(nil, nil)` | Cannot test `ErrMembershipNotFound` or valid membership flows |
| `MockPasswordResetRepo.FindActiveByUserID` | line 688 | `(nil, ErrResetNotFound)` | Password reset tests never find an existing reset |
| `MockUnitRepo.GetDescendants` | line 532 | `(nil, nil)` | Cannot test circular parent detection or hierarchy cascade |
| `MockUnitRepo.ListMembers` | line 540 | `(nil, nil)` | Unit member list always empty |
| `MockWorkingGroupRepo.ListMembers` | line 613 | `(nil, nil)` | WG member list always empty |

**Fix approach for each:**
- `MockOrgRepo.GetMembership`: Add a `Memberships map[string]*auth.OrganizationMembership` field (key = `userID:orgID`); look up in the method
- `MockPasswordResetRepo.FindActiveByUserID`: Add a `ResetsByUserID map[string]*pwdomain.PasswordReset` field; look up by userID
- `MockUnitRepo.GetDescendants`/`ListMembers`: Add corresponding map fields and populate them in test setup
- `MockWorkingGroupRepo.ListMembers`: Same pattern

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SurrealDB-backed tests → `t.Skip` if DB unavailable | testcontainers-go PostgreSQL | Phase 0 | Fully self-contained, no external DB dependency |
| TestPool reads `DATABASE_URL` env var | Container pool from testcontainers | Phase 0 | Tests run anywhere Docker runs |
| Refresh token without rotation | Refresh token rotation (revoke+reissue) | Phase 0 | Prevents stolen token reuse |
| 3-digit numeric reset code | 8-char alphanumeric reset code (out-of-band) | Phase 0 | 62^8 vs 10^3 combinations; not leaked in response |
| Uniform rate limiting | Path-specific rate limiting for auth | Phase 0 | Login/reset endpoints have stricter limits |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Docker daemon is available on all test execution environments | Common Pitfalls | Tests will skip silently — must validate in CI setup |
| A2 | testcontainers-go v1.35 works with Go 1.26.1 | Standard Stack | Version constraint mismatch would require fallback version |
| A3 | A single PostgreSQL container with per-schema isolation is fast enough | Architecture Patterns | Container per package may be slow on resource-constrained machines |

## Research Plan

### Plan 1: Testcontainers Setup
1. Add testcontainers-go dependencies to `go.mod`
2. Create `internal/adapters/secondary/postgres/test_setup.go` with `SetupPackageContainer`
3. Modify `exported_test_helpers.go`: replace `TestPool(t)` with container-backed pool helper
4. Update `cmd/server/main_test.go` to use new container pool
5. Verify existing smoke test passes with testcontainer-backed DB

### Plan 2: Auth Bug Fixes (test-driven)
1. For each bug: write failing testcontainer-backed integration test → fix → verify
2. Fix `/auth/memberships` nil pointer: add nil check for org in `GetMemberships` handler
3. Fix `/auth/me` empty role/org_id: return error when membership not found
4. Fix `/units/{id}/members` 500: differentiate parse errors (400) from real errors (500)
5. Fix `/organizations/members` 500: add orgID nil guard
6. Fix broken mock implementations (D-11)

### Plan 3: Auth Cleanup
1. Implement refresh token rotation in `auth.go` service `Refresh` method
2. Fix cookie name in `cookies.go` and update all handler references
3. Fix password reset code: increase entropy, remove from response body
4. Add auth-specific rate limiter in `cmd/server/main.go`

### Plan 4: Service + Handler Test Rewrites
1. Create `_integration_test.go` files for each service (auth, organization, unit, etc.)
2. Create `_integration_test.go` files for each HTTP handler
3. Keep existing mock-based tests for pure business logic (D-10)
4. Run full test suite to verify all tests pass

### Plan 5: Bug Buffer (separate plan)
1. Document major/complex bugs discovered during rewrite
2. `t.Skip()` failing tests with references
3. Include human-in-the-middle review loop (D-15)

### Plan 6: E2E Verification
1. Start backend with testcontainer-managed PostgreSQL
2. Run Playwright E2E tests against it
3. Verify all E2E specs pass

## Open Questions

1. **Snapshot support for faster test runs?** The testcontainers postgres module supports `Snapshot`/`Restore` operations which could be faster than full schema teardown+setup per test. Investigate if this is worth implementing for performance.

2. **Orbstack compatibility?** Docker is running via Orbstack (confirmed from `docker info`). testcontainers-go has good Orbstack support, but worth verifying the postgres module works correctly.

3. **Port conflicts with parallel package execution?** If multiple packages run tests in parallel (via `go test -p 2 ./...`), each starts its own Docker container on a random port. This should be fine since testcontainers maps to random host ports.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Docker | testcontainers-go | ✓ | 29.4.0 (Orbstack) | — |
| Go | Compilation & tests | ✓ | 1.26.1 | — |
| PostgreSQL (native) | testcontainers image | ✓ (Docker) | 16-alpine | — |
| Internet | Docker image pull | Assumed | — | Pre-pull image; fallback to cached |

**Missing dependencies with no fallback:** None

## Sources

### Primary (HIGH confidence)
- Testcontainers-Go official documentation (postgres module) — `postgres.Run()`, `ConnectionString()`, `CleanupContainer()`, snapshot/restore patterns
- Codebase files: `exported_test_helpers.go`, `auth.go` (service + handler), `cookies.go`, `ratelimit.go`, `password_reset.go`, `mocks.go`, `unit.go`, `organization.go`, `main.go`, `main_test.go`

### Secondary (MEDIUM confidence)
- GitHub: testcontainers/testcontainers-go — release notes for v0.32+ API changes

### Tertiary (LOW confidence)
- None — all findings verified against codebase or official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — testcontainers-go is the established Go standard for database-backed integration tests
- Architecture: HIGH — patterns verified against codebase files and testcontainers docs
- Pitfalls: HIGH — all pitfalls derived from actual bugs discovered in codebase or known testcontainers issues

**Research date:** 2026-06-08
**Valid until:** 2026-07-08 (testcontainers-postgres module is stable; auth code won't change until Phase 0 execution)
