# Testing Patterns

**Analysis Date:** 2026-08-08

Three levels of testing: Go unit + integration tests (backend), Vitest unit tests with MSW (frontend), Playwright e2e (browser). Pattern guidance is documented in `openwiki/testing/README.md` — note that its "Gaps" section (no component unit tests) is stale; component tests now exist (see below).

## Go Backend Tests

### Test Framework

**Runner:** standard `testing` package (Go 1.26.1)
**Assertions:** `github.com/stretchr/testify` — `assert` for non-fatal checks, `require` for fatal ones
**Testcontainers:** `github.com/testcontainers/testcontainers-go/modules/postgres` for integration tests

**Run Commands:**
```bash
make test                    # go test -v ./... (all packages)
go test -v ./internal/core/services/activity/...   # single package
go test -v -run TestAuthHandlerIntegration ./internal/adapters/primary/http/
```

### Test File Organization

**Location:** co-located with source, same package (white-box): `internal/core/services/activity/activity_test.go`, `internal/adapters/secondary/postgres/time_entry_repository_test.go`, `internal/adapters/primary/http/auth_test.go`

**Naming:** `<source_file>_test.go` (e.g., `activity_test.go`), plus purpose-named files (`refresh_token_rotate_test.go`, `ontology_extension_migrations_test.go`). Shared helpers live in dedicated non-test files when exported: `internal/adapters/secondary/postgres/exported_test_helpers.go`, `internal/adapters/primary/http/handler_test_helper.go`.

**Mock/fixture library:** `internal/core/services/testdata/` — `mocks.go`, `mock_<domain>_repo.go` (hand-written in-memory mocks), `factories.go` (entity factories), `mocks_test.go` (instantiation smoke test).

### Unit Test Pattern (services — mocks, no DB)

Hand-written in-memory mocks (mutex-protected maps), NOT testify `mock.Mock`. Example (`internal/core/services/testdata/mocks.go:26`):

```go
type MockTimeEntryRepo struct {
	mu           sync.Mutex
	Entries      map[uuid.UUID]*time_entry.TimeEntry
	PeriodLocked bool
}
```

Factories use functional overrides (`internal/core/services/testdata/factories.go`):

```go
func NewTimeEntry(overrides ...func(*time_entry.TimeEntry)) time_entry.TimeEntry {
	e := time_entry.TimeEntry{ /* defaults */ }
	for _, o := range overrides {
		o(&e)
	}
	return e
}
```

Service tests follow a consistent shape (`internal/core/services/activity/activity_test.go`):

```go
func setupService(t *testing.T) (*Service, *testdata.MockActivityRepo, *testdata.MockContractRepo, *testdata.MockUnitRepo) {
	t.Helper()
	activityRepo := &testdata.MockActivityRepo{}
	// ...
	svc := NewService(activityRepo, contractRepo, unitRepo, orgRepo, ticketRepo, routingSvc)
	return svc, activityRepo, contractRepo, unitRepo
}

func TestService_Create(t *testing.T) {
	t.Run("valid engagement creates", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), validCreateReq())
		require.NoError(t, err)
		assert.Equal(t, "Engagement Alpha", created.Name)
	})
	t.Run("missing name rejected", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		_, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)  // sentinel comparison, always
	})
}
```

**Rules:**
- Top-level test named `TestService_<Method>`; subtests are sentence-style ("valid engagement creates") or `Action_Outcome_Status` style for handlers ("Register_WithNewOrg_Returns201WithUserData")
- Assert errors with `assert.ErrorIs(t, err, domain.ErrX)` — never error strings
- Helper functions declare `t.Helper()`
- One `setupService(t)` per package, one `validCreateReq()`/`seedX` helper per suite
- Banner comment separators (`// ------...`) between suites

### Integration Test Pattern (testcontainers + real Postgres)

**One Postgres 16-alpine container per Go package** via `sync.Once` (`internal/adapters/secondary/postgres/test_setup.go`):

```go
pool := postgres.SetupPackageContainer(t)   // skips with t.Skip if Docker unavailable
postgres.SetupTestSchema(t, pool)           // create schema per subtest
t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })
```

- `SetupPackageContainer` returns a `*pgxpool.Pool`; do NOT register `t.Cleanup` on it (container lifetime is Ryuk-managed; first caller's cleanup would break later tests — see comment at `test_setup.go:52`)
- `SetupTestSchema`/`TeardownTestSchema` live in `internal/adapters/secondary/postgres/exported_test_helpers.go`; schema-per-subtest gives perfect isolation

**Handler integration fixture** (`internal/adapters/primary/http/handler_test_helper.go`) — wires the FULL production stack (all repos, services, handlers, middleware) exactly like `cmd/server/main.go` onto `httptest.NewServer`, with a cookie-jar client:

```go
f := newHandlerFixture(t, pool)
f.registerAndLogin(t, email, "user", "TestPass123!", "OrgName")  // real HTTP register+login, cookies captured
resp, err := f.Client.Post(f.ServerURL+"/units", "application/json", strings.NewReader(body))
```

**Rules:**
- Unique data per subtest via helpers: `uniqueEmail()`, `uniqueOrgName()` (`internal/adapters/primary/http/auth_test.go:18-28`) or `uuid.New().String()[:8]` suffixes
- Assert on decoded envelope: `var wrapped struct { Data map[string]interface{} `json:"data"` }`
- Keep the fixture's route table in sync when routes are added in `cmd/server/main.go`
- Repository integration tests follow the same container+per-test-schema pattern (`internal/adapters/secondary/postgres/*_test.go`)

### Mocking

**What to mock:** repository ports (`ports.XRepository`), token services, password hashers — any boundary service unit tests must not touch.

**What NOT to mock:** handler integration tests use the real repositories and real Postgres (not mocked); middleware tests are pure unit tests over `httptest` requests.

### Test Data

- Entity factories in `internal/core/services/testdata/factories.go`: `NewUser`, `NewOrganization`, `NewTimeEntry`, `NewContract`, `NewActivity` (all with override-function pattern)
- Unique-ID/email helpers: `UniqueID()`, `UniqueEmail()` in the same file
- Service tests seed mocks directly: `repo.Activities[a.ID] = &a` via `seedActivity(repo, overrides...)` helpers

## Frontend Unit Tests (Vitest)

### Test Framework

**Runner:** Vitest 4.x
**Config:** `web/vitest.config.ts` — `environment: "jsdom"`, `globals: true`, `setupFiles: ["./src/lib/__tests__/setup.ts"]`, `restoreMocks: true`, excludes `e2e/**`, `passWithNoTests: true`
**HTTP mocking:** MSW 2.x (`msw/node` `setupServer`)

**Run Commands:**
```bash
cd web && bun run test          # vitest run (single pass)
cd web && bun run test:watch    # watch mode
```

### Test File Organization

**Location:** sibling `__tests__/` directories — `web/src/api/__tests__/`, `web/src/lib/__tests__/`, `web/src/components/layout/__tests__/`, `web/src/components/shared/__tests__/`, and `web/src/routes/_authenticated/*/-components/__tests__/`

**Naming:** `<source>.test.ts` / `<source>.test.tsx`

**Setup file** (`web/src/lib/__tests__/setup.ts`): jest-dom matchers, `window.matchMedia` polyfill (jsdom gap for shadcn layout components), `afterEach(cleanup)`.

### API Layer Tests (MSW)

Standard per-file MSW lifecycle + envelope-shaped fixtures (`web/src/api/__tests__/auth.test.ts`):

```typescript
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

it("profileQueryOpts calls GET /auth/me ...", async () => {
  server.use(http.get("/api/auth/me", () => HttpResponse.json({ data: mockData })));
  const result = await AuthApis.profileQueryOpts.queryFn!(undefined as any);
  expect(result).toEqual(mockData);
});
```

**Pattern:** invoke the exported `queryOptions.queryFn` / `mutationOptions.mutationFn` directly, assert request method/path/body via captured request objects, assert unwrapped response data. HTTP client behavior (refresh-on-401 single-flight, envelope unwrap, 204, `UnauthorizedError`) is covered in `web/src/lib/__tests__/api.test.ts`.

### Component Tests (React Testing Library)

Two styles:

**1. Shallow render tests** (`web/src/components/shared/__tests__/entries-filters.test.tsx`): render component directly with `vi.fn()` props, assert label text/visibility.

**2. Full-stack render tests** (`web/src/components/layout/__tests__/route-error.test.tsx`): mount the real router + query client + MSW handlers:

```tsx
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 } } });
const router = createRouter({ routeTree, context: { client: queryClient }, history: createMemoryHistory({ initialEntries }) });
render(
  <QueryClientProvider client={queryClient}>
    <RouterProvider router={router} />
  </QueryClientProvider>
);
```

- MSW handlers return the `{ data }` envelope for auth/me, memberships, activities, time-entries etc.
- Mutable module flags (`let failTimeEntries = true`) simulate transient outages; reset in `afterEach`
- Assert with `waitFor` + `getByRole`/`getByText`

### Coverage

**Not enforced** — no coverage thresholds in `vitest.config.ts`, no coverage script in `web/package.json`, none in `qodana.yaml` (failureConditions commented out). Run ad hoc with `bunx vitest run --coverage` if needed.

## End-to-End Tests (Playwright)

**Config:** `web/playwright.config.ts` — testDir `./e2e`, Chromium only, `fullyParallel: true` (1 worker on CI), retries 2 on CI, `baseURL: http://localhost:3000`, `trace: "on-first-retry"`, built-in webServer `bun run dev`

**Run Commands:**
```bash
cd web && bunx playwright test                     # all
cd web && bunx playwright test e2e/auth.spec.ts    # one file
cd web && bunx playwright show-report
```

**Requires:** backend on `:8080` (start with `RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server` — see `web/e2e/helpers.ts:5-12`) and the dockerized Postgres (`docker-compose up`).

**Core pattern** (`web/e2e/auth.spec.ts`, `web/e2e/helpers.ts`):
- `test.describe.configure({ mode: "serial" })` per suite
- Register + login ONCE via `request.post("http://localhost:8080/...")` in `beforeAll`; inject session cookies into each test's `BrowserContext` (`loginOnce`, `useSession`) — avoids burning the anonymous rate limit
- Direct-Postgres seeding via `psql()` (docker exec) in `helpers.ts`: `seedBaseEntities`, `seedTimeEntries`, `seedExpenses`, `seedCustomers`, `promoteToFinance` — the public API cannot produce all workflow states in one call
- Assert with role-based locators (`getByRole("heading", { name: "Today" })`), `toHaveURL`, `toContainText`; `waitForLoadState("networkidle")` where needed

**Suites:** `auth.spec.ts`, `contracts.spec.ts`, `customers.spec.ts`, `activities.spec.ts`, `expenses.spec.ts`, `time-entries.spec.ts`, `approvals.spec.ts`, `working-groups.spec.ts`, `org-hierarchy.spec.ts`, `error-boundary.spec.ts`

## Test Types Summary

| Type | Location | Backing | Runs with |
|------|----------|---------|-----------|
| Go service unit | `internal/core/services/*/*_test.go` | testdata mocks | `make test` (no Docker) |
| Go handler integration | `internal/adapters/primary/http/*_test.go` | testcontainers Postgres | `make test` (Docker required, else skip) |
| Go repository integration | `internal/adapters/secondary/postgres/*_test.go` | testcontainers Postgres | `make test` (Docker required) |
| Go middleware/model/migrate | `internal/middleware/*`, `internal/models/*`, `cmd/migrate/*` | httptest / pure unit | `make test` |
| Frontend API unit | `web/src/api/__tests__/`, `web/src/lib/__tests__/` | MSW + jsdom | `bun run test` |
| Frontend component | `web/src/components/*/__tests__/`, `*-components/__tests__/` | RTL + jsdom + MSW | `bun run test` |
| E2E | `web/e2e/*.spec.ts` | Playwright + real stack | `bunx playwright test` |

## Common Patterns

**Async/error testing (frontend):**
```typescript
await expect(api("/test")).rejects.toThrow("bad request");       // error path
const result = await api<{ success: boolean }>("/protected");    // refresh-and-retry path
```

**Sentinel error assertions (Go):**
```go
assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
assert.Nil(t, created)
```

**Test gaps to be aware of:**
- No component tests for many route-level page components (`time-entries-page.tsx`, `entry-detail.tsx` etc.) — only filters/table/route-error/sidebar have coverage
- No coverage thresholds enforced anywhere
- E2E requires local Docker + backend; CI runs Go tests via `make test` (docs-check and qodana workflows exist in `.github/workflows/`; no Playwright CI workflow found)

---

*Testing analysis: 2026-08-08*
