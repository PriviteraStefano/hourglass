# Testing Patterns

**Analysis Date:** 2026-06-08

## Test Framework

**Go Backend:**
- **Runner:** Standard `go test` (Go 1.26.1)
- **Command:** `go test -v ./...` (from Makefile)
- **Config:** No additional test config files; uses vendored `gotest` via `go.mod`
- **Assertion Library:** `github.com/stretchr/testify` v1.11.1 (`assert` and `require` packages)

**Frontend:**
- **Runner:** Vitest v4.1.6
- **Config:** `web/vitest.config.ts`
- **Command:**
  ```bash
  cd web && bun run test           # vitest run
  cd web && bun run test:watch     # vitest (watch mode)
  ```
- **Assertion Library:** Vitest built-in (`expect`, `describe`, `it`, etc.) with `globals: true` enabled
- **DOM Environment:** `jsdom` with `@testing-library/jest-dom/vitest` matchers

## Test File Organization

**Go Backend:**
- **Co-located** with source files using `_test.go` suffix — e.g., `auth.go` → `auth_test.go`, `handlers.go` → `handlers_test.go`
- **Test helpers** live in separate exported helper files when shared across packages: `internal/adapters/secondary/postgres/exported_test_helpers.go`
- **Mock implementations** centralized in `internal/core/services/testdata/mocks_test.go`

**Frontend:**
- **Separate `__tests__/` directory** per module — e.g., `web/src/api/__tests__/auth.test.ts`, `web/src/lib/__tests__/api.test.ts`
- **E2E tests** in `web/e2e/` — separate from unit/integration tests
- **Setup files** at `web/src/lib/__tests__/setup.ts`

**Structure:**
```
backend:
  internal/
    core/services/auth/
      auth.go                     # Source
      auth_test.go                # Unit tests (co-located)
    core/services/testdata/
      mocks_test.go               # Shared mocks (co-located in testdata/ subpackage)
    adapters/primary/http/
      auth.go                     # Handler source
      auth_test.go                # Integration test (co-located)
    adapters/secondary/postgres/
      exported_test_helpers.go    # TestPool, seed helpers (co-located)
      user_repository.go          # Repository source
      user_repository_test.go     # Repository test (co-located)

frontend:
  web/src/
    api/
      auth.ts                     # API module source
      __tests__/
        auth.test.ts              # Unit tests (separate dir)
    lib/
      api.ts                      # HTTP client source
      __tests__/
        api.test.ts               # Unit tests
        setup.ts                  # Vitest setup
        validation.test.ts        # Zod validation tests
  web/e2e/
    auth.spec.ts                  # E2E tests
    projects.spec.ts              # E2E tests
```

## Test Structure

**Go Backend — Service Tests (Table-Driven):**
```go
func TestService_Register(t *testing.T) {
    tests := []struct {
        name    string
        req     RegisterRequest
        setup   func(*testdata.MockUserRepo)
        wantErr error
    }{
        {
            name: "valid registration with new org",
            req:  RegisterRequest{Email: "test@example.com", ...},
            setup: func(u *testdata.MockUserRepo) {},
            wantErr: nil,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            userRepo := &testdata.MockUserRepo{}
            orgRepo := &testdata.MockOrgRepo{}
            tt.setup(userRepo)

            svc := NewService(userRepo, orgRepo, ...)
            resp, err := svc.Register(context.Background(), tt.req)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, resp)
            } else {
                require.NoError(t, err)
                require.NotNil(t, resp)
                assert.NotEmpty(t, resp.User.ID)
            }
        })
    }
}
```

**Go Backend — HTTP Handler Tests (Integration):**
```go
type testServer struct {
    handler *AuthHandler
    server  *httptest.Server
    client  *http.Client
    pool    *pgxpool.Pool
}

func setupTestServer(t *testing.T) *testServer {
    pool := postgres.TestPool(t)
    postgres.SetupTestSchema(t, pool)
    t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })
    // ... wire up services, mux, httptest.NewServer
}

func TestRegister_WithNewOrg(t *testing.T) {
    ts := setupTestServer(t)
    body := map[string]string{
        "email": uniqueEmail(),
        "password": "password123",
        "organization_name": uniqueOrgName(),
    }
    jsonBody, _ := json.Marshal(body)
    resp, err := http.Post(ts.server.URL+"/auth/register", "application/json", ...)
    // assert status, decode body, check fields
}
```

**Go Backend — Repository Tests (Database Integration):**
```go
func TestUserRepository_Add_GetByID(t *testing.T) {
    pool := TestPool(t)
    SetupTestSchema(t, pool)
    t.Cleanup(func() { TeardownTestSchema(t, pool) })

    repo := NewUserRepository(pool)
    user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Test", "User", "hash123")
    err := repo.Add(context.Background(), user)
    require.NoError(t, err)

    got, err := repo.GetByID(context.Background(), user.ID)
    require.NoError(t, err)
    require.Equal(t, user.ID, got.ID)
}
```

**Frontend — API Unit Tests:**
```typescript
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('AuthApis', () => {
    it('profileQueryOpts calls GET /auth/me and returns user', async () => {
        server.use(
            http.get('/api/auth/me', () => HttpResponse.json({ data: mockData })),
        )
        const result = await AuthApis.profileQueryOpts.queryFn()
        expect(result).toEqual(mockData)
    })
})
```

## Mocking

**Go Backend:**
- **Framework:** Hand-written mocks in `internal/core/services/testdata/mocks_test.go`
- Mocks implement port interfaces from `internal/core/ports/`
- Mock structs store data in maps: `Users map[uuid.UUID]*authdomain.User`
- Mock services follow interface: `type MockTokenService struct { ... GenerateToken(...) ... }`
- **Pattern:** Injected via constructor into services:
  ```go
  userRepo := &testdata.MockUserRepo{}
  orgRepo := &testdata.MockOrgRepo{}
  svc := NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)
  ```
- **Verification:** Mock instantiation verified in `TestMocks_Instantiate` test

**Frontend:**
- **Framework:** MSW (Mock Service Worker) v2 — `msw` and `msw/node`
- **Pattern:** `setupServer()` from `msw/node` with lifecycle hooks:
  ```typescript
  const server = setupServer()
  beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())
  ```
- **Request handlers** use `http.get()`, `http.post()` with URL patterns
- **Response capture** for verifying request bodies:
  ```typescript
  let capturedBody: unknown = null
  server.use(
      http.post('/api/auth/login', async ({ request }) => {
          capturedBody = await request.json()
          return HttpResponse.json({ data: mockResponse })
      }),
  )
  ```
- **No component mocks** — API tests only test the query/mutation option functions directly
- **No `vi.mock()`** usage detected; all mocking is at network level via MSW

**What to Mock:**
- **Go:** Repository interfaces (ports), token service, password hasher
- **Frontend:** HTTP responses (MSW handlers)

**What NOT to Mock:**
- **Go:** Service business logic is tested with real repository test helpers (seed functions) for repository tests
- **Frontend:** Network layer only; data transformation and validation are tested separately

## Fixtures and Factories

**Go Backend — Seed Functions for Database Tests:**
```go
func seedOrg(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID
func seedUser(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID
func seedProject(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID
func seedContract(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID
func seedUnit(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID
```

- All seed functions call `t.Helper()` and `require.NoError(t, err)` internally
- Defined in `internal/adapters/secondary/postgres/exported_test_helpers.go`
- Use `uuid.New()` for IDs, pass `time.Time` for timestamps

**Frontend — Inline Mock Data:**
- Mock data defined inline in each test function as local variables
- No shared fixture files or factories
- Test data uses `as const` assertions for literal types: `role: 'employee' as const`

## Coverage

**Requirements:**
- No coverage targets enforced. No coverage thresholds detected.
- **Makefile:** `go test -v ./...` without `-cover` flag
- **package.json:** `vitest run` without `--coverage` flag

**View Coverage:**
```bash
go test -cover ./...                              # Go backend
cd web && bun vitest run --coverage                 # Frontend (requires @vitest/coverage)
```

## Test Types

**Unit Tests — Go Services:**
- Test business logic with mocked repositories
- Table-driven pattern with multiple error/success cases
- Located in `internal/core/services/*/`
- Examples: `auth_test.go`, `project_test.go`, `time_entry_test.go`

**Integration Tests — Go HTTP Handlers:**
- Test full HTTP request/response cycle with real database
- Use `httptest.NewServer` + real `http.ServeMux` wiring
- Located in `internal/adapters/primary/http/*_test.go`
- Examples: `auth_test.go` (745 lines), `project_test.go`, `contract_test.go`

**Integration Tests — Go Repositories:**
- Test database CRUD operations against real PostgreSQL
- Use `TestPool(t)` which connects via `DATABASE_URL` (skips if not set)
- Use `SetupTestSchema`/`TeardownTestSchema` for schema lifecycle
- Use seed helper functions for data setup
- Located in `internal/adapters/secondary/postgres/*_test.go`
- Examples: `user_repository_test.go`, `project_repository_test.go`

**E2E Go Smoke Test:**
- `cmd/server/main_test.go` — `TestSmoke` integration test that wires the full server and tests health, registration, login, and authenticated access
- Uses real PostgreSQL (via `postgres.TestPool`) and `httptest.Server`

**Frontend Unit Tests:**
- Test API query/mutation option functions by invoking `queryFn`/`mutationFn` directly
- Test Zod validation schemas with `.safeParse()`
- Located in `web/src/api/__tests__/` and `web/src/lib/__tests__/`

**E2E Tests — Playwright:**
- Framework: `@playwright/test` v1.59.1
- Config: `web/playwright.config.ts`
- Command: `cd web && bunx playwright test`
- Tests full browser flow against running dev server
- Located in `web/e2e/`:
  - `auth.spec.ts` — Registration, login, validation, UI error display
  - `projects.spec.ts`
  - `contracts.spec.ts`
  - `customers.spec.ts`
  - `time-entries.spec.ts`
  - `org-hierarchy.spec.ts`

## Common Patterns

**Async Testing — Frontend:**
```typescript
// Direct invocation of queryFn
const result = await AuthApis.profileQueryOpts.queryFn()
expect(result).toEqual(mockData)

// Testing rejected promises
await expect(api('/test')).rejects.toThrow('bad request')
```

**Error Testing — Go:**
```go
// Sentinel error matching
if tt.wantErr != nil {
    assert.ErrorIs(t, err, tt.wantErr)
    assert.Nil(t, resp)
} else {
    require.NoError(t, err)
    require.NotNil(t, resp)
}

// HTTP status code assertion
if resp.StatusCode != http.StatusCreated {
    t.Errorf("expected status 201, got %d", resp.StatusCode)
}
```

**Error Testing — Frontend:**
```typescript
it('handles error response', async () => {
    server.use(
        http.get('/api/test', () =>
            HttpResponse.json({ error: 'bad request' }, { status: 400 }),
        ),
    )
    await expect(api('/test')).rejects.toThrow('bad request')
})
```

**Validation Testing — Frontend (Zod):**
```typescript
it('CreateUnitRequestSchema accepts valid data', () => {
    const valid = { name: 'Engineering', description: null, parent_unit_id: null, code: 'ENG' }
    expect(CreateUnitRequestSchema.safeParse(valid).success).toBe(true)
})
it('CreateUnitRequestSchema rejects empty name', () => {
    const invalid = { name: '', description: null, parent_unit_id: null, code: 'ENG' }
    expect(CreateUnitRequestSchema.safeParse(invalid).success).toBe(false)
})
```

## Database Test Setup

**Go Backend — Schema Lifecycle:**
```go
pool := postgres.TestPool(t)           // Gets pool, skips if DATABASE_URL unset
postgres.SetupTestSchema(t, pool)       // Applies all migrations/*.up.sql
t.Cleanup(func() {
    postgres.TeardownTestSchema(t, pool) // Drops all tables in dependency order
})
```

- `TestPool` at `internal/adapters/secondary/postgres/exported_test_helpers.go:20`
- `SetupTestSchema` applies migration files sorted alphabetically, excluding seed files
- `TeardownTestSchema` drops all 24 tables in reverse dependency order with `CASCADE`

---

*Testing analysis: 2026-06-08*
