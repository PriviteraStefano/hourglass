# Testing

Hourglass has test coverage at three levels: Go unit/integration tests,
frontend API unit tests, and end-to-end Playwright tests.

---

## Backend Tests (Go)

### Test Fixture Pattern

Backend tests use a shared test infrastructure built on `testcontainers-go`.

**Package-level PostgreSQL container** (`/internal/adapters/secondary/postgres/test_setup.go`):

```go
pool := postgres.SetupPackageContainer(t)
```

- Uses `sync.Once` to start **one PostgreSQL 16 Alpine container** per Go package
- Shared across all tests in the package (Ryuk reaper handles cleanup)
- Returns a `*pgxpool.Pool` connected to the container
- Skips the test if Docker is unavailable

**HTTP handler test fixture** (`/internal/adapters/primary/http/handler_test_helper.go`):

```go
f := newHandlerFixture(t, pool)
loginResp := f.registerAndLogin(t, email, username, password, orgName)
```

- Wires the **full production stack**: all repos, services, handlers, and middleware
- Uses `httptest.NewServer` for an in-memory HTTP server
- The `http.Client` has a cookie jar that captures auth cookies automatically
- `registerAndLogin()` creates users through the real HTTP API

**Schema setup per test case:**

```go
postgres.SetupTestSchema(t, pool)
t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })
```

### Test Files

| Directory                                        | Description                                       |
|--------------------------------------------------|---------------------------------------------------|
| `internal/adapters/primary/http/*_test.go`       | HTTP handler integration tests with real Postgres |
| `internal/adapters/secondary/postgres/*_test.go` | Repository integration tests with real Postgres   |
| `internal/core/services/*/export_test.go`        | Service-level unit tests using mocks              |
| `internal/core/services/testdata/mocks.go`       | Mock implementations for service tests            |
| `internal/models/*_test.go`                      | Model/data structure tests                        |
| `internal/middleware/*_test.go`                  | Middleware tests                                  |

### Service Tests

Services use mock repositories defined in `internal/core/services/testdata/mocks.go`:

```go
type MockTimeEntryRepository struct {
    mock.Mock
    // implements ports.TimeEntryRepository
}
```

### Running Tests

```bash
make test              # go test -v ./... (all Go tests)

# Or specific packages
go test -v ./internal/core/services/export/...
go test -v ./internal/adapters/primary/http/...
```

---

## Frontend Tests (Vitest + MSW)

### Configuration (`/web/vitest.config.ts`)

| Setting      | Value                                |
|--------------|--------------------------------------|
| Framework    | Vitest 4.x                           |
| Environment  | `jsdom`                              |
| Globals      | Enabled (`describe`, `it`, `expect`) |
| Mock restore | Automatic (`restoreMocks: true`)     |
| Path alias   | `@` → `./src`                        |
| Excludes     | `e2e/**`                             |

### API Unit Tests

Frontend API modules are tested with MSW (Mock Service Worker):

```ts
// web/src/api/__tests__/time-entries.test.ts
const server = setupServer(
  http.get('*/time-entries', () => HttpResponse.json({data: [...]})),
)
```

Each test file:

1. Registers MSW handlers for the expected endpoints
2. Calls the exported `queryOptions` / `mutationOptions` directly
3. Asserts correct HTTP method, path, query params, and response data

**Test files:** `auth.test.ts`, `contracts.test.ts`, `customers.test.ts`, `projects.test.ts`, `time-entries.test.ts`

### Running Frontend Tests

```bash
cd web && bun run test           # Vitest run
cd web && bun run test:watch     # Watch mode
```

### Gaps

No component-level unit tests (e.g., with React Testing Library). The
`ApprovalButtons`, `ApprovalHistory`, `EntryDetail`, and `ExpenseDetail`
components are not unit-tested.

---

## End-to-End Tests (Playwright)

### Configuration (`/web/playwright.config.ts`)

| Setting        | Value                          |
|----------------|--------------------------------|
| Test directory | `./e2e`                        |
| Browser        | Desktop Chrome (Chromium)      |
| Parallel       | Fully parallel, 1 worker on CI |
| Retries        | 2× on CI                       |
| Base URL       | `http://localhost:3000`        |
| Trace          | On first retry only            |

Built-in web server config: runs `bun run dev`, waits for `http://localhost:3000`.

### E2E Test Files

| File                    | Coverage                                                           |
|-------------------------|--------------------------------------------------------------------|
| `auth.spec.ts`          | Register, login/logout, protected route redirects, form validation |
| `contracts.spec.ts`     | Contract CRUD                                                      |
| `customers.spec.ts`     | Customer CRUD                                                      |
| `projects.spec.ts`      | Project CRUD                                                       |
| `org-hierarchy.spec.ts` | Organization hierarchy UI                                          |
| `time-entries.spec.ts`  | Time entry CRUD with approval                                      |

### E2E Pattern

```ts
test.describe.configure({mode: 'serial'})

test.beforeAll(async ({request}) => {
  // Register a fresh user via API
})

test.beforeEach(async ({page}) => {
  // Log in via browser form
})
```

Tests use `waitForLoadState('networkidle')` for async rendering and interact
with real UI elements (fill inputs, click buttons, assert visibility).

### Running E2E Tests

```bash
cd web && bunx playwright test           # Run all
cd web && bunx playwright test e2e/auth.spec.ts  # Single file
cd web && bunx playwright show-report    # View HTML report
```

**Requires a running backend** (defaults to `http://localhost:8080`).

---

## Integration Test File List

### Go Handler Integration Tests

- `/internal/adapters/primary/http/auth_integration_test.go`
- `/internal/adapters/primary/http/auth_test.go`
- `/internal/adapters/primary/http/contract_test.go`
- `/internal/adapters/primary/http/customer_test.go`
- `/internal/adapters/primary/http/export_test.go`
- `/internal/adapters/primary/http/handler_integration_test.go`
- `/internal/adapters/primary/http/invitation_test.go`
- `/internal/adapters/primary/http/organization_test.go`
- `/internal/adapters/primary/http/password_reset_test.go`
- `/internal/adapters/primary/http/project_test.go`
- `/internal/adapters/primary/http/time_entry_test.go`
- `/internal/adapters/primary/http/unit_test.go`
- `/internal/adapters/primary/http/working_group_test.go`

### Go Repository Integration Tests

- `/internal/adapters/secondary/postgres/*_test.go` (all repository files)

### Go Service Unit Tests

- `/internal/core/services/export/export_test.go`

### Frontend MSW Tests

- `/web/src/api/__tests__/auth.test.ts`
- `/web/src/api/__tests__/contracts.test.ts`
- `/web/src/api/__tests__/customers.test.ts`
- `/web/src/api/__tests__/projects.test.ts`
- `/web/src/api/__tests__/time-entries.test.ts`

### E2E Tests

- `/web/e2e/auth.spec.ts`
- `/web/e2e/contracts.spec.ts`
- `/web/e2e/customers.spec.ts`
- `/web/e2e/org-hierarchy.spec.ts`
- `/web/e2e/projects.spec.ts`
- `/web/e2e/time-entries.spec.ts`
