# Testing Patterns

**Analysis Date:** 2026-08-12

## Test Framework

**Frontend Runner:**
- Vitest 4.1.10 (`web/package.json`)
- Config: `web/vitest.config.ts` — jsdom environment, `globals: true`, setup file `./src/lib/__tests__/setup.ts`, `restoreMocks: true`, excludes `e2e/**` and `node_modules/**`, `passWithNoTests: true`, alias `@` → `./src`

**Frontend Assertion Library:**
- Vitest `expect` + `@testing-library/jest-dom/vitest` matchers (loaded in `web/src/lib/__tests__/setup.ts`, which also polyfills `window.matchMedia` for shadcn layout components and calls `cleanup()` after each test)

**Frontend API Mocking:**
- MSW 2.x — `msw/node` `setupServer()` per test file, `server.listen({ onUnhandledRequest: "bypass" })` / `resetHandlers()` / `close()` in beforeAll/afterEach/afterAll

**Backend Runner:**
- Go standard `testing` + testify (`github.com/stretchr/testify` v1.11.1 — `assert`/`require`)
- Integration containers: `github.com/testcontainers/testcontainers-go/modules/postgres` v0.42.0 (postgres:16-alpine)

**Run Commands:**
```bash
cd web && bun run test          # Vitest run (all unit/component tests)
cd web && bun run test:watch    # Vitest watch mode
cd web && bun run lint          # oxlint --type-aware
cd web && bun run typecheck     # tsc -b
cd web && bunx playwright test  # E2E (Playwright)
make test                       # go test -v ./... (whole backend)
```

## Test File Organization

**Location:**
- Go: co-located `*_test.go` with source — `internal/core/services/auth/auth_test.go`, `internal/adapters/secondary/postgres/user_repository_test.go`, `internal/adapters/primary/http/auth_test.go`, `internal/middleware/logging_test.go`
- Frontend: `__tests__/` folder next to the unit under test:
  - `web/src/api/__tests__/*.test.ts` (auth, time-entries, contracts, customers, activities, expenses)
  - `web/src/lib/__tests__/*.test.ts` (api, validation, role-visibility, `setup.ts`)
  - `web/src/components/*/__tests__/*.test.tsx` (entries-filters, entries-table, sidebar-groups, route-error)
  - `web/src/routes/_authenticated/*/-components/__tests__/*.test.tsx` (today-page, time-entries-list, approvals-page, expenses-list)
- E2E: `web/e2e/*.spec.ts` (auth, customers, time-entries, expenses, approvals, contracts, activities, working-groups, org-hierarchy, error-boundary) + shared `web/e2e/helpers.ts`

**Naming:**
- Go tests: `TestService_Register`, `TestAuthHandlerIntegration`, subtests `"Register_WithNewOrg_Returns201WithUserData"`, `"Register_InvalidEmail_Returns400"` (PascalCase with underscores); unique-data helpers `uniqueID()`, `uniqueEmail()`, `uniqueOrgName()` (`internal/adapters/primary/http/auth_test.go:18-28`)
- Frontend: `describe("AuthApis", ...)` / `it("loginMutationOpts sends POST /auth/login with credentials", ...)` — behavioral sentences, not function names

## Test Structure

**Go — service unit tests (table-driven):**
```go
// internal/core/services/auth/auth_test.go
func TestService_Register(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		setup   func(*testdata.MockUserRepo)
		wantErr error
	}{ ... }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &testdata.MockUserRepo{}
			tt.setup(userRepo)
			svc := NewService(userRepo, ...)
			resp, err := svc.Register(context.Background(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```
- Pattern: table cases with a `setup func(mock)` for fixture arrangement; `assert.ErrorIs` for sentinel errors; `require` for fatal assertions, `assert` for continued ones

**Go — HTTP handler integration tests:**
```go
// internal/adapters/primary/http/auth_test.go
func TestAuthHandlerIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)
	t.Run("Register_WithNewOrg_Returns201WithUserData", func(t *testing.T) {
		postgres.SetupTestSchema(t, pool)
		t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })
		f := newHandlerFixture(t, pool)
		// POST to f.ServerURL+"/auth/register" via f.Client (cookie jar)
		// decode into map[string]interface{}, assert data.user.email...
	})
}
```

**Frontend — API module tests (MSW):**
```typescript
// web/src/api/__tests__/auth.test.ts
const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

it("loginMutationOpts sends POST /auth/login with credentials", async () => {
  let capturedBody: unknown = null;
  server.use(http.post("/api/auth/login", async ({ request }) => {
    capturedBody = await request.json();
    return HttpResponse.json({ data: mockResponse });
  }));
  const result = await AuthApis.loginMutationOpts.mutationFn!(creds, undefined as any);
  expect(capturedBody).toEqual(creds);
  expect(result).toEqual(mockResponse);
});
```
- The HTTP client contract (`web/src/lib/__tests__/api.test.ts`) covers: envelope unwrapping, error envelope → thrown Error, 401→refresh→retry (asserts the retried request count), refresh failure → `UnauthorizedError`, Content-Type header, network errors

**Frontend — component tests:**
- Pure render/screen assertions (`web/src/components/shared/__tests__/entries-filters.test.tsx`) with `vi.fn()` handlers
- Full-stack component tests build a real router + QueryClientProvider + MSW (`web/src/components/layout/__tests__/sidebar-groups.test.tsx` uses `createMemoryHistory`, `createRouter`, `RouterProvider`, `routeTree.gen` import) and drive role-dependent UI via mutable per-test MSW state (`let currentRole: Role = "employee"`)
- Data factories are plain functions with optional overrides: `function wg(overrides?: Partial<WorkingGroup>): WorkingGroup { return { ...defaults, ...overrides } }` (`web/src/lib/__tests__/role-visibility.test.ts:38-54`)

## Mocking

**Backend:** Hand-written mocks implementing ports interfaces, in `internal/core/services/testdata/mocks.go` (e.g. `MockUserRepo`, `MockOrgRepo`, `MockTokenService`, `MockPasswordHasher`, `MockRefreshTokenRepo`) plus per-domain mocks (`mock_ticket_repo.go`, `mock_direction_repo.go`, `mock_coverage_repo.go`, `mock_availability_repo.go`, `mock_audit_log_repo.go`, `mock_org_settings_repo.go`) and `factories.go`; `mocks_test.go` smoke-tests that all mocks instantiate. Mocks expose in-memory maps as public fields (`u.Users = map[uuid.UUID]*authdomain.User{...}`) for direct seeding.

**Frontend:** MSW for all network mocking; `vi.fn()` / `vi.mock` for local functions; `restoreMocks: true` resets spies between tests. E2E uses real backend + real Postgres (no mocking).

**What to Mock:** Network boundary (MSW), ports in Go services. E2E helper docs state the public API cannot produce all workflow states, so E2E seeds rows directly via `psql` (`web/e2e/helpers.ts`).

**What NOT to Mock:** The full route wiring — `newHandlerFixture` builds the real mux with real postgres-backed services so handler tests exercise the true stack (`internal/adapters/primary/http/handler_test_helper.go`, mirrors `cmd/server/main.go`).

## Fixtures and Factories

**Go:**
- `postgres.SetupPackageContainer(t)` — one postgres:16-alpine container per package via `sync.Once`; **skips tests with `t.Skip` when Docker is unavailable**; cleanup delegated to testcontainers Ryuk (explicitly NOT `t.Cleanup` — see comment in `internal/adapters/secondary/postgres/test_setup.go`)
- `postgres.SetupTestSchema(t, pool)` applies all `migrations/*.up.sql` (excluding seed) sorted alphabetically; `TeardownTestSchema` drops tables per subtest for isolation (`internal/adapters/secondary/postgres/exported_test_helpers.go`)
- `newHandlerFixture(t, pool)` returns `handlerFixture{Client (cookie jar), Server, ServerURL, Pool, authSvc}` with `registerAndLogin`/`loginUser` helpers

**Frontend E2E:**
- `web/e2e/helpers.ts`: `psql()` (docker exec), `registerUser()`, `loginOnce()` (returns Set-Cookie pairs), `useSession()` (injects cookies into browser context), `promoteToFinance()`, `seedBaseEntities()`, `seedTimeEntries()`, `seedExpenses()`, `seedCustomers()` — direct SQL seeding for list states the API can't produce
- `web/e2e/auth.spec.ts` uses `test.describe.configure({ mode: 'serial' })` and registers/logs in once in `beforeAll`, sharing cookies to stay inside backend rate limits

## Coverage

**Requirements:** None enforced. No thresholds in `web/vitest.config.ts` (no coverage provider configured); `qodana.yaml` has coverage thresholds commented out. Coverage is by convention: every service package has unit + integration tests, every handler has integration tests, every api module has MSW tests, every major feature has an e2e spec.

**View Coverage:**
```bash
cd web && bunx vitest run --coverage   # requires installing @vitest/coverage-v8 (not currently a dep)
go test -cover ./...
```

## Test Types

**Unit Tests (Go):**
- Service-level table-driven tests with `testdata` mocks: `internal/core/services/auth/auth_test.go`, `customer_test.go`, `contract_test.go`, `expense_test.go`, `time_entry_test.go`, `activity_test.go`, `unit_test.go`, `working_group_test.go`, `routing_test.go`, `direction_test.go`, `ticket_test.go`, `availability_test.go`, `coverage_test.go`, `invitation_test.go`, `orgsettings_test.go`, `password_reset_test.go`, `export_test.go`
- Model/domain tests: `internal/models/models_test.go`, `internal/models/models_phase2_test.go`, `internal/core/domain/availability/availability_test.go`
- Middleware: `internal/middleware/middleware_test.go`, `logging_test.go` (captures `log` output), `ratelimit_test.go`, `version_test.go`; cmd tests: `cmd/migrate/migrate_test.go`, `cmd/server/main_test.go`

**Unit Tests (Frontend):**
- Pure logic: `web/src/lib/__tests__/role-visibility.test.ts`, `validation.test.ts`, `use-download` (covered via e2e/error-boundary)
- API option factories via MSW: `web/src/api/__tests__/{auth,time-entries,contracts,customers,activities,expenses}.test.ts`, `contracts.test.ts` + `web/src/lib/__tests__/api.test.ts`

**Integration Tests (Go):** `*_integration_test.go` beside services (`auth_integration_test.go`, `contract_integration_test.go`, `customer_integration_test.go`, `unit_integration_test.go`, `organization_integration_test.go`, `invitation_integration_test.go`, `password_reset_integration_test.go`, `working_group_integration_test.go`, `ticket_integration_test.go`) and handler tests under `internal/adapters/primary/http/*_test.go` — all against real Postgres via testcontainers. Repository tests in `internal/adapters/secondary/postgres/*_test.go` cover migrations (`*_migration_test.go`, `*_migrations_test.go`) and CRUD per repository.

**E2E Tests (Playwright):**
- Framework: `@playwright/test` ^1.62 (`web/playwright.config.ts` — testDir `./e2e`, chromium only, baseURL `http://localhost:3000`, `webServer: "bun run dev"` with `reuseExistingServer`, `fullyParallel: true`, CI: retries 2, workers 1, `forbidOnly`)
- Requires backend on :8080 — documented in `web/e2e/helpers.ts` header: `RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server`
- Suites: auth, customers, time-entries, expenses, approvals, contracts, activities, working-groups, org-hierarchy, error-boundary (`web/e2e/*.spec.ts`)

## Common Patterns

**Async Testing (frontend):**
```typescript
// web/src/components/layout/__tests__/sidebar-groups.test.tsx
await waitFor(() => {
  expect(screen.getByText("...")).toBeInTheDocument();
});
```
Route hydration is awaited with `await router.load()` or `waitFor` after MSW handlers respond; e2e uses `await expect(page.getByRole('heading', ...)).toBeVisible({ timeout: 10000 })`.

**Error Testing (frontend):**
```typescript
// web/src/lib/__tests__/api.test.ts
await expect(api("/protected")).rejects.toThrow("Unauthorized");
```
401-refresh flows are asserted with call counters (`let protectedCalls = 0; expect(protectedCalls).toBe(2)`).

**Error Testing (Go):**
```go
if tt.wantErr != nil {
	assert.ErrorIs(t, err, tt.wantErr)
	assert.Nil(t, resp)
} else {
	require.NoError(t, err)
	require.NotNil(t, resp)
}
```
HTTP status assertions use `assert.Equal(t, http.StatusBadRequest, resp.StatusCode)` with `require.NoError` on request/parse steps.

---

*Testing analysis: 2026-08-12*
