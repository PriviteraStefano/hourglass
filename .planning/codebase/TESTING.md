# Testing Patterns

**Analysis Date:** 2026-05-12

## Test Frameworks

### Backend (Go)

**Runner:** Go's built-in `testing` package (`go test`)
- **Config:** No external test runner; uses `go test -v ./...` (from `Makefile`)
- **Assertion:** Standard `if` / `t.Errorf` pattern (no assertion library)

**Run commands:**
```bash
make test                     # go test -v ./...
go test -v ./internal/...     # Run specific package tests
```

### Frontend (E2E)

**Runner:** Playwright (`@playwright/test` v1.59.1)
- **Config:** `web/playwright.config.ts`
- **Reporter:** HTML reporter (`reporter: 'html'`)
- **Browser:** Chromium only (Desktop Chrome device)
- **Test location:** `web/e2e/*.spec.ts`

**Run commands:**
```bash
cd web && bunx playwright test           # Run all e2e tests
cd web && bunx playwright test auth     # Run auth tests
cd web && bunx playwright test --ui    # Interactive UI mode
```

**Playwright config settings:**
```typescript
fullyParallel: true
forbidOnly: !!process.env.CI          // Disallow .skip in CI
retries: process.env.CI ? 2 : 0       // 2 retries in CI
workers: process.env.CI ? 1 : undefined
webServer: {
  command: 'bun run dev',
  url: 'http://localhost:3000',
  reuseExistingServer: !process.env.CI,
  timeout: 120000,
}
```

## Test File Organization

### Backend

**Location:** Co-located with source files using `_test.go` suffix.

**Pattern:**
```
internal/adapters/primary/http/
  auth.go
  auth_test.go          # Handler-level integration tests
  time_entry.go
  time_entry_test.go

internal/adapters/secondary/surrealdb/
  user_repository.go
  user_repository_test.go
  organization_repo.go
  organization_repo_test.go
```

**Structure:** Tests are organized by function, not by `TestSuite`:
```go
func TestRegister_WithNewOrg(t *testing.T) { ... }
func TestRegister_InvalidEmail(t *testing.T) { ... }
func TestRegister_DuplicateEmail(t *testing.T) { ... }
```

### Frontend E2E

**Location:** `web/e2e/` directory (separate from source).

**Naming:** Feature-based spec files:
```
web/e2e/
  auth.spec.ts
  (additional specs as needed)
```

## Test Structure

### Backend (Go) — Handler Integration Tests

**Pattern:** `httptest.NewServer` with `http.ServeMux` wiring, real SurrealDB test instance.

**Setup helper pattern (`auth_test.go`):**
```go
type testServer struct {
    handler *AuthHandler
    server  *httptest.Server
    client  *http.Client
    db      *sdb.DB
}

func setupTestServer(t *testing.T) *testServer {
    ns := "test_" + uniqueID()
    db, err := hexauth.GetTestDBWithNamespace(ns, ns)
    if err != nil {
        t.Skipf("Failed to connect to SurrealDB (set SURREALDB_URL): %v", err)
    }
    t.Cleanup(func() {
        db.Close(context.Background())
    })
    // ... wire dependencies ...
    return &testServer{handler, server, client, db}
}
```

**Test data generation:**
```go
func uniqueID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func uniqueEmail() string { return "test_" + uniqueID() + "@example.com" }
func uniqueOrgName() string { return "Org_" + uniqueID() }
```

**Assertions:** Direct comparison with error on mismatch:
```go
if resp.StatusCode != http.StatusCreated {
    t.Errorf("expected status 201, got %d", resp.StatusCode)
}
```

**Response parsing:**
```go
var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
data, ok := result["data"].(map[string]interface{})
if !ok {
    t.Fatal("expected data object in response")
}
```

### Frontend E2E (Playwright)

**Pattern:** Page-object-style tests with `test.describe` grouping.

**Structure:**
```typescript
test.describe('Auth Flow', () => {
  test('register with new organization', async ({ page }) => {
    await page.goto('/register')
    // ... fill form ...
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
  })
})
```

**Key patterns:**
- `async/await` for all Playwright actions
- `expect().toHaveURL()` for navigation assertions
- `expect().toBeVisible()` for element assertions
- `expect().getByText()` for text content assertions
- `{ timeout: 10000 }` for longer waits on async operations
- `request.post()` API calls for test setup (bypassing UI)

## Mocking

### Backend (Go)

**No mocking framework used.** Tests use real infrastructure:
- Real SurrealDB test database (created per-test with unique namespace)
- Real JWT signing/validation
- Real password hashing

**Test infrastructure:** `internal/adapters/secondary/surrealdb/helpers.go` provides:
- `GetTestDBWithNamespace(ns, dbName string)` — creates isolated test DB with schema
- `applyTestSchema()` — defines tables and indexes for test DB

**What NOT mocked:** All business logic runs against real services and repositories. This is closer to integration testing than unit testing.

### Frontend

**No frontend unit/component tests exist.** Only E2E tests with Playwright.

## Fixtures and Factories

### Backend

**Test data:** Inline factory functions per test file:
```go
func uniqueEmail() string { return "test_" + uniqueID() + "@example.com" }
func uniqueOrgName() string { return "Org_" + uniqueID() }
```

**No shared fixture files.** Each test file has its own helper functions.

### Frontend E2E

**Test data:** Generated inline with `Date.now()` for uniqueness:
```typescript
await page.fill('input[name="email"]', `test_${Date.now()}@example.com`)
```

## Coverage

**No enforced coverage targets.** `go test -v ./...` runs all tests with verbose output.

**Coverage reporting:** Not configured. No `go cover` or codecov integration observed.

## Test Types

### Backend Integration Tests

**Handler-level integration tests** (`internal/adapters/primary/http/*_test.go`):
- Test HTTP layer with real HTTP client (`net/http/httptest`)
- Test full request/response cycle including middleware context injection
- Run against real SurrealDB (isolated per test via unique namespace)
- Cover: success paths, validation failures, auth failures, edge cases

**Examples in `auth_test.go` (749 lines, 24 test functions):**
- `TestRegister_WithNewOrg` — full registration flow
- `TestRegister_InvalidEmail` — validation
- `TestRegister_WeakPassword` — password rules
- `TestRegister_DuplicateEmail` — conflict handling
- `TestLogin_WithEmail_Success` — login flow
- `TestLogin_InvalidPassword` — auth failure
- `TestLogout_WithRefreshToken` — cookie clearing
- `TestRefresh_ValidToken` — token refresh
- `TestGetProfile_Authenticated` — protected endpoint with Bearer token
- `TestGetProfile_Unauthenticated` — missing token

**Examples in `time_entry_test.go` (64 lines, 3 test functions):**
- `TestTimeEntryHandler_Create_InvalidBody` — malformed JSON
- `TestTimeEntryHandler_Create_MissingProjectID` — validation
- `TestTimeEntryHandler_Approve_EmployeeForbidden` — role-based access

**Repository tests** (`internal/adapters/secondary/surrealdb/*_test.go`):
- Test DB read/write operations
- Use same `GetTestDBWithNamespace` helper

**Middleware tests** (`internal/middleware/*_test.go`):
- Auth middleware, rate limiting, logging

### Frontend E2E Tests

**Playwright E2E tests** (`web/e2e/auth.spec.ts`, 93 lines, 6 tests):
- Register flow with form submission
- Register validation (empty form errors)
- Login with valid credentials
- Login with invalid credentials (error message)
- Logout redirects to login
- Protected route redirects to unauthenticated users

**CI considerations:**
- Web server auto-started by Playwright config
- Uses real `bun run dev` dev server
- No visual/screenshot assertions

## Common Patterns

### Async Testing (Go)

```go
// HTTP request with error checking
resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", strings.NewReader(string(jsonBody)))
if err != nil {
    t.Fatalf("register failed: %v", err)
}
defer resp.Body.Close()

// For authenticated requests
req, _ := http.NewRequest("POST", ts.server.URL+"/auth/login", strings.NewReader(string(jsonBody)))
req.Header.Set("Content-Type", "application/json")
resp, err = ts.client.Do(req)
```

### Error Testing (Go)

```go
// Expect specific HTTP status
if resp.StatusCode != http.StatusUnauthorized {
    t.Errorf("expected status 401, got %d", resp.StatusCode)
}

// Expect error in response body
var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
if result["email"] == nil {
    t.Error("expected email in response")
}
```

### E2E Error Assertions (Playwright)

```typescript
// Text content
await expect(page.getByText(/invalid credentials/i)).toBeVisible()

// URL redirect
await expect(page).toHaveURL(/\/login/, { timeout: 10000 })

// Form validation
await expect(page.getByText('Email is required')).toBeVisible()
```

### Middleware Context Injection (Go)

```go
ctx := middleware.SetUserID(req.Context(), uuid.New())
ctx = middleware.SetOrganizationID(ctx, uuid.New())
ctx = middleware.SetRole(ctx, "employee")
req = req.WithContext(ctx)
h.Create(rec, req)
```

## Missing Test Coverage

**Frontend:** No unit tests, component tests, or integration tests for React code. Only E2E Playwright tests exist.

**Backend service layer:** Service-level unit tests are not present. Services are tested indirectly through handler integration tests.

**Backend repository layer:** Some repository tests exist but coverage is incomplete.

**Coverage gaps:**
- No frontend component rendering tests
- No React Query query/mutation logic tests
- No Zod validation schema tests
- No edge cases for complex business rules (approval workflows, period locking)
- No performance/load tests

---

*Testing analysis: 2026-05-12*
