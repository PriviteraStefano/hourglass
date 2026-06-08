# Codebase Concerns

**Analysis Date:** 2026-06-08

## Tech Debt

### Dual Model Layer — Duplicated Type Definitions

**Issue:** Models are defined in three overlapping layers (`internal/models/models.go`, `internal/models/surreal_models.go`, and `internal/core/domain/*/*.go`), causing duplication and drift risk. The `surreal_models.go` file (251 lines) is a remnant from a SurrealDB prototype and contains types (e.g., `SurrOrganization`, `SurrTimeEntry`, `SurrExpense`) that are never referenced by the current PostgreSQL implementation. Some domain-specific types and request structs in `models.go` (e.g., `PendingEntryGroup`, `BatchApproveRequest`) are also duplicated by domain models in `internal/core/domain/`.

**Files:**
- `internal/models/models.go` (476 lines)
- `internal/models/surreal_models.go` (251 lines)
- `internal/core/domain/*/*.go` (14 domain packages)

**Impact:** When a field is added to one model but not its counterpart, silent data corruption or deserialization errors occur. The SurrealDB types are dead code that increases maintenance surface.

**Fix approach:** Remove `internal/models/surreal_models.go`. Consolidate all domain types into `internal/core/domain/` and remove overlapping types from `internal/models/models.go`. Migrate the few unique request types (e.g., `CreateUnitRequest`, `BatchApproveRequest`) to their respective domain packages or keep only in appropriate handler packages if truly handler-specific.

### SurrealDB Migration Remnant — `surreal_models.go`

**Issue:** The entire `internal/models/surreal_models.go` file (251 lines) contains SurrealDB-specific type definitions (string-based RecordIDs, simplified status constants `SurrStatus*`, entry/action/actor constants). These types are never imported by any production code path.

**Files:** `internal/models/surreal_models.go`

**Impact:** Dead code that misleads developers about the current data model. Constants like `ActorRoleWGManager = "wg_manager"` in one file differ from role strings in the middleware layer.

**Fix approach:** Delete the file after verifying no references exist. Domain-specific constants should live in their respective domain packages.

### Duplicate Project Type Column in Schema

**Issue:** The `projects` table in `migrations/000_full_schema.up.sql` has both a `project_type` column (line 164) and a `type` column (line 165), both with the same CHECK constraint `('billable', 'internal')`. This is a schema design error where two columns store the same concept.

**Files:** `migrations/000_full_schema.up.sql` (lines 164-165)

**Impact:** Ambiguity in which column to use. The domain model (`internal/core/domain/project/project.go`) uses `Type` but the repository could be reading/writing either column.

**Fix approach:** Consolidate to a single column. Create a migration to drop the redundant column after verifying which one the repositories actually query.

### In-Memory Only Rate Limiting

**Issue:** The rate limiter at `internal/middleware/ratelimit.go` uses an in-memory `map[string]*clientInfo` with a `sync.RWMutex`. It counts requests per minute per IP/user but resets to zero on server restart. There is no sharing across multiple server instances.

**Files:** `internal/middleware/ratelimit.go` (86 lines)

**Impact:** Ineffective in multi-instance deployments (e.g., Docker orchestration). An attacker can exhaust rate limit budget per instance. Also, the map grows unbounded as unique IPs accumulate, though entries expire after 1 minute.

**Fix approach:** Use a Redis-backed or database-backed rate limiter for production deployments.

### Refresh Token Lacks Rotation

**Issue:** When a refresh token is used to obtain new access tokens (via `POST /auth/refresh`), the old refresh token is not rotated (not revoked and replaced). The `Refresh` method in `internal/core/services/auth/auth.go` (line 317) only reads the token hash but never issues a new refresh token or revokes the old one.

**Files:** `internal/core/services/auth/auth.go` (lines 317-355, `Refresh` method), `internal/adapters/primary/http/auth.go` (lines 172-196)

**Impact:** If a refresh token is stolen, it can be used indefinitely (up to its 7-day TTL) without detection. There is no token family or reuse detection.

**Fix approach:** Rotate refresh tokens on each refresh: revoke the old hash, generate a new refresh token, store only the new hash. Optionally implement refresh token reuse detection per NIST guidelines.

### Cookie Name Mismatch Between Code Paths

**Issue:** `internal/cookies/cookies.go` defines cookie name constants `AccessTokenCookieName = "access_token"` and `RefreshTokenCookieName = "refresh_token"`, but `internal/adapters/primary/http/auth.go` hard-codes cookie names as `"auth_token"` and `"refresh_token"` directly. The refresh token name happens to match, but the access token name does not (`"access_token"` vs `"auth_token"`).

**Files:**
- `internal/cookies/cookies.go` (line 8-9)
- `internal/adapters/primary/http/auth.go` (lines 123, 133, 151, 161)

**Impact:** The helper functions in `cookies.go` (`SetAccessTokenCookie`, `ClearAuthCookies`, etc.) are unused dead code, or if used in future code, they will set a differently-named cookie than the one the middleware reads in `middleware.go` (line 25: `r.Cookie("auth_token")`).

**Fix approach:** Either unify all handlers to use the `cookies.go` helpers and remove the hardcoded strings, or update `cookies.go` constants and delete the helper functions if they are deliberately unused.

### Fire-and-Forget Audit Log Goroutine

**Issue:** `internal/core/services/time_entry/time_entry.go` line 199 spawns `go s.auditRepo.Create(ctx, auditLog)` to write audit logs asynchronously. The goroutine captures the request context, which may be cancelled after the HTTP response is sent. Errors from `auditRepo.Create` are silently discarded.

**Files:** `internal/core/services/time_entry/time_entry.go` (line 199)

**Impact:** Audit log entries are silently dropped on database write failures. Using a cancelled context will cause the write to fail with `context.Canceled`. There is no retry, back-pressure, or fallback logging mechanism.

**Fix approach:** Use a proper background task queue with a context that outlives the request (e.g., `context.Background()`), add error logging, and consider a retry mechanism for critical audit data.

### Password Reset Code Leaked in Response Body

**Issue:** The password reset endpoint at `internal/adapters/primary/http/password_reset.go` line 54-58 returns the generated reset code directly in the response body: `"code": code`. The reset code is a 3-digit numeric string generated via `crypto/rand` in `internal/core/services/password_reset/password_reset.go` (line 101-108).

**Files:**
- `internal/adapters/primary/http/password_reset.go` (lines 54-58)
- `internal/core/services/password_reset/password_reset.go` (lines 101-108)

**Impact:** Exposing the reset code in the response body means anyone who can read the API response (or who intercepts the response) can reset the user's password. Even in development, this is a bad pattern that should be replaced — the code should only be delivered out-of-band (e.g., email). Additionally, a 3-digit numeric code (1000 combinations) is trivially brute-forceable.

**Fix approach:** Never return the reset code in the response body. Deliver it via email/SMS only. Increase code entropy to 6+ alphanumeric characters and rate-limit verification attempts.

### `/auth/refresh` Lacks Auth Middleware

**Issue:** `POST /auth/refresh` at `cmd/server/main.go` line 80 is registered without `middleware.Auth()`. This is correct because refresh needs to work with a potentially expired access token. However, the handler reads the refresh token from the cookie and does not verify any additional CSRF protection or client fingerprint.

**Files:** `cmd/server/main.go` (line 80), `internal/adapters/primary/http/auth.go` (lines 172-196)

**Impact:** The refresh endpoint is a single-point-of-failure for session security. Without client fingerprinting or rotation, a stolen refresh cookie grants permanent access.

**Fix approach:** Implement refresh token rotation (see above) and optionally bind refresh tokens to client fingerprints (e.g., `User-Agent` + IP hash).

## Known Bugs

### Time Entry DB Status Constraint Mismatch

**Symptoms:** The database schema in `migrations/000_full_schema.up.sql` line 281 defines the status column constraint as: `CHECK (status IN ('draft', 'submitted', 'approved'))`. However, `internal/models/models.go` lines 166-172 defines `EntryStatus` constants including `StatusPendingManager = "pending_manager"`, `StatusPendingFinance = "pending_finance"`, `StatusRejected = "rejected"`. The service layer in `internal/core/services/time_entry/time_entry.go` line 176 sets rejected entries back to `StatusDraft` rather than a `Rejected` state.

**Files:**
- `migrations/000_full_schema.up.sql` (line 281)
- `internal/models/models.go` (lines 166-172)
- `internal/core/services/time_entry/time_entry.go` (line 176)

**Trigger:** Attempting to set a time entry status to `pending_manager`, `pending_finance`, or `rejected` via direct DB updates or future workflow steps will fail with a CHECK constraint violation.

**Workaround:** The `Reject` method sets status to `draft` rather than `rejected`, masking the issue. The approval workflow with `pending_manager`/`pending_finance` states exists in the model layer but cannot be persisted to the database.

### Bogus Error on Register with Bad OrgID

**Symptoms:** In `internal/core/services/auth/auth.go` line 183, `uuid.Parse(req.OrgID)` silently ignores parse errors — the parsed UUID is the zero value `uuid.Nil` when `uuid.Parse()` fails. This means an invalid invite code will not create an org (because `orgID == uuid.Nil` check at line 194) but also does not return an error.

**Files:** `internal/core/services/auth/auth.go` (lines 182-184)

**Trigger:** Registering with a malformed `OrgID` (invite code) silently falls through to no-org behavior without telling the user.

**Workaround:** Pass a valid UUID as invite code.

### Unhandled JSON Decode Error in Time Entry Reject Handler

**Symptoms:** `internal/adapters/primary/http/time_entry.go` line 423 calls `json.NewDecoder(r.Body).Decode(&req)` and discards the error return value. If the body contains malformed JSON, `req.Reason` will be an empty string, and the reject proceeds without a reason.

**Files:** `internal/adapters/primary/http/time_entry.go` (line 423)

**Trigger:** Sending malformed JSON to `POST /time-entries/{id}/reject`.

**Workaround:** The rejection succeeds but without a reason comment.

### `MockOrgRepo.GetMembership` Always Returns nil

**Symptoms:** In `internal/core/services/testdata/mocks.go` lines 252-254, `MockOrgRepo.GetMembership` always returns `(nil, nil)`. This means any test using this mock that exercises the `ErrMembershipNotFound` path cannot do so, and the `SwitchOrganization` flow is never tested with a valid membership.

**Files:** `internal/core/services/testdata/mocks.go` (lines 252-254)

**Trigger:** Any test path that requires an existing organization membership through `MockOrgRepo` will silently get nil.

**Workaround:** Tests must set up membership through a different mock if they need realistic membership responses.

## Security Considerations

### Development-Only Default JWT Secret

**Risk:** `cmd/server/main.go` lines 31-37 fall back to `"dev-secret-change-in-production"` when `JWT_SECRET` is not set and the environment is not production/staging. If a staging server is accidentally started without the env var, JWTs can be forged.

**Files:** `cmd/server/main.go` (lines 31-37), `internal/auth/auth.go` (line 38)

**Current mitigation:** The env var check rejects `GO_ENV=production` or `staging` without a JWT_SECRET. The warning log is printed for other environments.

**Recommendations:** Log a warning regardless of environment, and require explicit opt-in (e.g., `JWT_SECRET=dev`) for accepting the default secret.

### CORS Allows Wildcard Origin

**Risk:** `internal/middleware/cors.go` lines 15-16 treat `"*"` as a valid CORS origin in the allowed list. If `ALLOWED_ORIGINS=*` is set, any website can make credentialed requests to the API.

**Files:** `internal/middleware/cors.go` (lines 15-16)

**Current mitigation:** The default config only allows `http://localhost:3000`. The risk is only in production misconfiguration.

**Recommendations:** Remove the `"*"` wildcard match from the CORS middleware. If any origin is needed, use explicit validation against a whitelist only.

### No Input Length Validation on String Fields

**Risk:** Several handlers accept string fields that map to `VARCHAR(255)` database columns without length validation. Truncation or SQL errors can occur. The Register handler in `internal/adapters/primary/http/auth.go` does not validate `FirstName`, `LastName`, `OrganizationName` lengths.

**Files:**
- `internal/adapters/primary/http/auth.go` (type `RegisterRequest`, lines 27-35)
- `internal/adapters/primary/http/unit.go` 
- `internal/adapters/primary/http/working_group.go` (type `CreateWorkingGroupRequest`, lines 22-31)
- `internal/adapters/primary/http/customer.go`

**Current mitigation:** Domain value objects like `authdomain.NewEmail`, `authdomain.NewPassword`, `authdomain.NewUsername` validate format and emptiness, but not maximum length.

**Recommendations:** Add explicit max-length checks in all handler validation before sending data to the service layer.

### No Rate Limiting on Auth Endpoints

**Risk:** The rate limiter middleware is applied globally in `cmd/server/main.go` line 212 as the outermost layer, but the limits (20 anonymous, 100 authenticated per minute) apply uniformly across all endpoints. Auth-specific endpoints like login and password reset need stricter limits but are not differentiated.

**Files:** `cmd/server/main.go` (line 212), `internal/middleware/ratelimit.go` (lines 25-30)

**Current mitigation:** Global rate limiting exists but does not differentiate auth endpoints.

**Recommendations:** Apply a separate, stricter rate limiter (e.g., 5 req/min for login, 3 req/min for password reset) in front of auth routes, or check path patterns in the rate limiter middleware.

## Performance Bottlenecks

### Inefficient Mock Repositories with Mutex on Every Method

**Problem:** Every mock repository method in `internal/core/services/testdata/mocks.go` (772 lines) acquires and releases `sync.Mutex` even for pure read operations. Test suites running these mocks have unnecessary lock contention.

**Files:** `internal/core/services/testdata/mocks.go` (all mock methods use `mu.Lock()`)

**Cause:** All mock repository methods use `m.mu.Lock()` instead of `m.mu.RLock()` for read-only operations.

**Improvement path:** Use `RLock()`/`RUnlock()` for read-only mock methods (e.g., `List`, `GetByID`, `GetByEmail`, `GetByUsername`).

### No Pagination on List Endpoints

**Problem:** Several list endpoints return all results without pagination: `GET /time-entries`, `GET /units`, `GET /working-groups`, `GET /customers`. As data grows, these requests will consume increasing memory and response time.

**Files:**
- `internal/adapters/primary/http/time_entry.go` (line 81, `List`)
- `internal/adapters/primary/http/unit.go` (handler)
- `internal/adapters/primary/http/working_group.go` (line 62, `List`)
- `internal/adapters/primary/http/customer.go` (handler)

**Cause:** Repositories query without `LIMIT`/`OFFSET` clauses and return all matching rows.

**Improvement path:** Add pagination parameters (page/limit or cursor-based) to list handlers and repositories.

### Unbounded Rate Limiter Map

**Problem:** `internal/middleware/ratelimit.go` stores every unique client key in an in-memory map without eviction. While entries expire after 1 minute (no longer checked after the window), the map grows with every unique IP that makes a request.

**Files:** `internal/middleware/ratelimit.go` (lines 29-30, 76-79)

**Cause:** No cleanup goroutine removes expired entries from `rl.requests`.

**Improvement path:** Add a periodic cleanup goroutine that removes entries whose `windowEnd` has passed, or use a fixed-size LRU cache.

## Fragile Areas

### Working Group Creation: OrgID from Request Body Instead of Middleware

**Files:** `internal/adapters/primary/http/working_group.go` (lines 113-117)

**Why fragile:** The `Create` handler reads `OrgID` from the request body (`req.OrgID`) instead of from the JWT claims via `middleware.GetOrganizationID(ctx)`. A user could potentially specify a different `org_id` in the body and create a working group in a different organization. Although the middleware auth wrapper ensures the user is authenticated, there is no server-side check that the caller belongs to the target org.

**Safe modification:** Always use `middleware.GetOrganizationID(ctx)` for the org context and remove `org_id` from the request body. Validate the user has a membership in the target org.

### Unused Legacy Handler Pattern Coexistence

**Files:** `internal/handlers/health_handler.go`, `internal/middleware/logging_test.go`, `internal/middleware/version.go`

**Why fragile:** The codebase has a mix of old handler patterns (e.g., `healthHandler.ServeHTTP(w, r)` at `cmd/server/main.go` line 52) and the hexagonal adapter pattern (`hexTEHandler.List(w, r)`). Some legacy files (like `internal/handlers/` and `internal/models/` structs) remain as glue. New developers may accidentally add new code to the wrong layer.

**Test coverage:** The health handler has a test (`internal/handlers/health_test.go`) but no integration tests verify the correct wiring.

### `MockPasswordResetRepo` Returns Not Found for Any User

**Files:** `internal/core/services/testdata/mocks.go` (lines 688-690)

**Why fragile:** `MockPasswordResetRepo.FindActiveByUserID` always returns `nil, pwdomain.ErrResetNotFound`. This means all password reset flow tests that need an existing reset will fail unless they bypass the mock entirely.

**Test coverage:** The password reset service test (`internal/core/services/password_reset/password_reset_test.go`) exists but coverage of the `Verify` path is limited by this mock behavior.

### Mock Repositories Return Null for Most Nested Queries

**Files:** `internal/core/services/testdata/mocks.go`

**Why fragile:** Several mock methods return `nil, nil` unconditionally, making them unsuitable for testing any real business logic path:
- `MockOrgRepo.GetMembership` → always `(nil, nil)`
- `MockUnitRepo.GetDescendants` → always `(nil, nil)`
- `MockUnitRepo.ListMembers` → always `(nil, nil)`
- `MockWorkingGroupRepo.ListMembers` → always `(nil, nil)`

## Scaling Limits

### Single-Database Connection Pool

**Current capacity:** `internal/db/pgpool.go` configures `MaxConns = 25` with `MaxConnLifetime = 30min`. The pool uses a global singleton via `sync.Once`.

**Limit:** At 25 concurrent database connections, the application will queue requests once exhausted. With a single PostgreSQL instance, read replicas cannot be used without application changes.

**Scaling path:** Add read/write connection splitting, connection pool per tenant, or horizontal read replicas. The `pgxpool.Pool` setup supports multiple pools but the app currently creates only one.

### No Caching Layer

**Current capacity:** All data is fetched from PostgreSQL on every request. No Redis or in-memory cache exists.

**Limit:** Frequently accessed data (user profile, memberships, organization settings) is queried on nearly every request. The `GetProfile` handler at `GET /auth/me` queries the database on every call.

**Scaling path:** Add a caching layer (e.g., Redis) for auth data and organization metadata. Use cache-aside or write-through patterns.

## Dependencies at Risk

### `golang-jwt/jwt/v5`

**Risk:** This is the JWT library used by the auth system. If a vulnerability is found in the JWT parsing, all tokens are compromised.

**Impact:** Complete authentication bypass.

**Migration plan:** Pin the version in `go.mod` and monitor for advisories. The library is well-maintained but is a critical security dependency.

### `@xyflow/react` in Org Hierarchy Page

**Risk:** The org hierarchy flow visualization uses `@xyflow/react` (formerly react-flow-renderer) which is a large external dependency. Version updates may break the custom `BUNode` component or layout algorithm in `dagre-layout.ts`.

**Files:** `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx`

**Impact:** Org hierarchy UI breaks on upgrade.

**Migration plan:** Maintain a pinned version. Write integration tests for key interactions (node click, drag, expand/collapse).

## Missing Critical Features

### Expense Workflow Is Incomplete

**Problem:** The codebase has expense model definitions (`internal/models/models.go` lines 277-368, `migrations/000_full_schema.up.sql` lines 316-342), expense approval types (lines 392-401), mock repository (`internal/adapters/secondary/postgres/expense_repository_test.go`), and export repo interface (`internal/core/ports/expense_repository.go`). However, there is no expense HTTP handler, no expense service, and no expense frontend pages.

**Blocks:** Users cannot create, submit, approve, or manage expenses through the application.

**Files:**
- `internal/core/ports/expense_repository.go` — Interface exists
- `internal/adapters/secondary/postgres/expense_repository_test.go` — Tests exist
- But no expense handler or service implementation
- `web/src/routes/_authenticated/time-entries/` — No expense routes

### Subproject Management Has No Frontend

**Problem:** Subprojects exist in the database schema (`migrations/000_full_schema.up.sql` lines 212-224) and are referenced by working groups, but there is no handler or frontend for CRUD operations on subprojects.

**Blocks:** Users cannot create, edit, or delete subprojects through the UI.

### No Webhook or Integration System

**Problem:** There is no webhook system for external integrations. Notifications for approvals, rejections, or submissions must be handled in-band.

**Blocks:** External systems cannot react to time entry or expense events in real time.

## Test Coverage Gaps

### Frontend Component Tests Missing

**What's not tested:** The frontend has 7 API-layer tests (`web/src/api/__tests__/`) and 6 Playwright e2e specs (`web/e2e/`). There are zero component-level tests for the 80+ UI components in `web/src/components/ui/` and `web/src/routes/*/-components/`. The complex org hierarchy page (342 lines) with ReactFlow nodes, dialogs, and state management has no component tests.

**Files:**
- `web/src/components/ui/*.tsx` (50+ shadcn-style components) — No tests
- `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` — No tests
- `web/src/routes/_authenticated/time-entries/-components/entry-row.tsx` — No tests

**Risk:** UI regressions in the approval flow, org hierarchy, and time entry forms will not be caught.

**Priority:** Medium

### Integration Tests Skip Real Database

**What's not tested:** All 51 Go test files use in-memory mocks (`internal/core/services/testdata/mocks.go`). There are no integration tests that test against an actual PostgreSQL instance (e.g., using testcontainers or a test database).

**Files:** All `*_test.go` files

**Risk:** SQL query errors, constraint violations, and repository-layer logic bugs are only caught at runtime.

**Priority:** Medium

### Auth Refresh Rotation Not Tested

**What's not tested:** The refresh token flow (`POST /auth/refresh`) has no test coverage for token reuse detection, expiry, or revocation scenarios.

**Files:** `internal/adapters/primary/http/auth_test.go` — Tests login and register but not refresh behavior

**Risk:** A refresh token theft vulnerability could go undetected.

**Priority:** High

---

*Concerns audit: 2026-06-08*
