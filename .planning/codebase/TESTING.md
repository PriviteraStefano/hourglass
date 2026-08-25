# Testing Patterns

**Analysis Date:** 2026-08-25

Two independent test stacks: Go (`go test`) for the backend and Vitest + Playwright for the `web/` frontend.

## Backend (Go)

### Test Framework

**Runner:** Go standard `testing` package. Run all with `make test` (`go test -v ./...`) — `Makefile:30`.

**Assertion Library:** `github.com/stretchr/testify` v1.11.1 (`assert` and `require` subpackages). `require` is used for fatal assertions, `assert` for non-fatal. (`internal/core/services/auth/auth_test.go:10-11`).

**Run commands:**
```bash
make test                  # go test -v ./...  (all backend tests)
go test ./internal/core/services/auth/...   # single package
go test -run TestService_Register ./internal/core/services/auth/   # single test
```

**Note:** 87 backend `*_test.go` files exist (excluding `.gsd-worktrees`).

### Test File Organization

**Location:** Co-located with source, same package (white-box). Two flavors per package:
- `*_test.go` — unit tests (mock-based)
- `*_integration_test.go` — integration tests (real PostgreSQL via testcontainers)

Examples: `internal/core/services/auth/auth_test.go` + `auth_integration_test.go`; `internal/adapters/secondary/postgres/user_repository_test.go`.

**Naming:** `TestXxx` functions; subtests via `t.Run(tt.name, func(t *testing.T){...})`. Table-driven tests are the norm (`internal/core/services/auth/auth_test.go:21-90`).

### Test Structure

**Unit test (table-driven):**
```go
func TestService_Register(t *testing.T) {
    tests := []struct {
        name    string
        req     RegisterRequest
        setup   func(*testdata.MockUserRepo)
        wantErr error
    }{ /* cases */ }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            userRepo := &testdata.MockUserRepo{}
            // ... build service from mocks
            resp, err := svc.Register(ctx, tt.req)
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            }
        })
    }
}
```
(`internal/core/services/auth/auth_test.go:21-90`).

**Integration test (real DB):**
```go
func TestAuthIntegration(t *testing.T) {
    pool := postgres.SetupPackageContainer(t)   // testcontainers, skips if no Docker
    t.Run("RegisterWithRealDB", func(t *testing.T) {
        svc := realRepoFixture(t, pool)          // SetupTestSchema + per-subtest teardown
        resp, err := svc.Register(ctx, req)
        require.NoError(t, err)
        require.Equal(t, "manager", resp.Membership.Role)
    })
}
```
(`internal/core/services/auth/auth_integration_test.go:18-60`).

### Mocking

**No mock framework** (no `testify/mock`, no generated mocks). Mocks are hand-written structs in `internal/core/services/testdata/` (package `testdata`), one file per repository/dependency: `mocks.go`, `mock_user_repo.go`, `mock_org_settings_repo.go`, `mock_ticket_repo.go`, etc. (`./internal/core/services/testdata/`).

- Mocks implement the same interface as the real repository (e.g., `MockUserRepo` satisfies the `ports` repo interface).
- Methods return zero values / empty results by default; some mocks expose state setters like `SetPlanRows(...)` for query stubs (`mock_direction_repo.go:64`).
- Tests inject mocks into service constructors: `NewService(userRepo, orgRepo, tokenSvc, pwHasher, refreshRepo)` (`internal/core/services/auth/auth_integration_test.go:25-32`).
- A smoke test, `TestMocks_Instantiate`, asserts all mocks are non-nil (`internal/core/services/testdata/mocks_test.go:7-29`).

**What to mock:** repository ports, token service, password hasher — anything the service depends on. **What NOT to mock:** the DB in `*_integration_test.go` (use the real container).

### Database Integration Tests (testcontainers)

- `internal/adapters/secondary/postgres/test_setup.go:21` — `SetupPackageContainer(t)` spins up a single `postgres:16-alpine` container per package (`sync.Once`). **If Docker is unavailable the test calls `t.Skip(...)`** — integration tests are not run in CI without Docker.
- `internal/adapters/secondary/postgres/exported_test_helpers.go` provides `TestPool`, `SetupTestSchema`, `TeardownTestSchema` (per-subtest schema, registered with `t.Cleanup`), and `uniqueEmail()`/`uniqueUsername()` helpers.
- Repository tests follow: `pool := TestPool(t); SetupTestSchema(t, pool); t.Cleanup(...)` then exercise real SQL (`internal/adapters/secondary/postgres/user_repository_test.go:14-45`).

### Coverage

- **No enforced coverage threshold.** Qodana config `qodana.yaml` has `testCoverageThresholds` commented out. `go test` runs without `-cover` in `Makefile`.
- View coverage manually: `go test -cover ./...` or `go test -coverprofile=cover.out ./... && go tool cover -html=cover.out`.

### Test Types

- **Unit:** service-layer logic with hand-written mocks (`*_test.go`).
- **Integration:** real PostgreSQL via testcontainers (`*_integration_test.go`).
- **E2E (backend):** none in Go; E2E is handled by the frontend Playwright suite.

## Frontend (web/)

### Test Framework

**Runner:** Vitest v4 (`vitest`). Config: `web/vitest.config.ts`.

**Environment:** `jsdom` (browser-DOM simulation for React component tests).

**Globals:** `globals: true` — `describe`/`it`/`expect`/`beforeAll`/`afterEach` available without import, though tests also import them explicitly (`web/src/api/__tests__/auth.test.ts:1`).

**Setup file:** `./src/lib/__tests__/setup.ts` — imports `@testing-library/jest-dom/vitest`, polyfills `window.matchMedia` (jsdom lacks it; needed by shadcn sidebar), and calls `cleanup()` after each test.

**Restore mocks:** `restoreMocks: true`.

**Excludes:** `e2e/**` and `node_modules/**`; `passWithNoTests: true`.

**Run commands:**
```bash
cd web
bun run test            # vitest run (all unit/component tests)
bun run test:watch      # vitest (watch mode)
bun run typecheck       # tsc -b (type-only, not a test runner)
```

### Path Alias

- `@/` -> `./src/` (set in both `vitest.config.ts` and `vite.config.ts` and `tsconfig.json`). Tests import source as `@/lib/api.ts`, `@/types/unit`, etc.

### Test File Organization

**Location:** Co-located in `__tests__/` directories beside the module under test. Naming: `*.test.ts` (logic) and `*.test.tsx` (React components).

Examples:
- `web/src/api/__tests__/auth.test.ts` (API option tests)
- `web/src/lib/__tests__/api.test.ts`, `validation.test.ts`, `role-visibility.test.ts`
- `web/src/components/shared/__tests__/entries-table.test.tsx`, `status-badge.test.tsx`
- `web/src/routes/_authenticated/.../-components/__tests__/*.test.tsx`

**Count:** 16 Vitest test files under `web/src` (e2e excluded).

### Test Structure

**API-layer test (MSW + Vitest):**
```typescript
import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { AuthApis } from "../auth";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("AuthApis", () => {
  it("profileQueryOpts calls GET /auth/me and returns user with membership", async () => {
    server.use(http.get("/api/auth/me", () => HttpResponse.json({ data: mockData })));
    const result = await AuthApis.profileQueryOpts.queryFn!(undefined as any);
    expect(result).toEqual(mockData);
  });
});
```
(`web/src/api/__tests__/auth.test.ts:1-43`).

**Component test (Testing Library):**
```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { EntriesTable } from "../entries-table";
// render(<EntriesTable .../>); screen.getByRole(...); fireEvent.click(...)
```
(`web/src/components/shared/__tests__/entries-table.test.tsx:1-70`).

### Mocking

**HTTP mocking:** MSW (`msw` + `msw/node`) intercepts `fetch` to the API. `setupServer()` is started in `beforeAll`, handlers reset in `afterEach`, closed in `afterAll`. Request bodies are captured to assert payloads (`web/src/api/__tests__/auth.test.ts:76-86`).

**Module/function mocking:** Vitest `vi` (e.g., `vi.fn()`, `vi.mock(...)`) available via globals; `restoreMocks: true` resets between tests.

**DOM:** `@testing-library/react` (`render`, `screen`, `fireEvent`, `within`) + `@testing-library/jest-dom` matchers (`toBeInTheDocument`).

**What to mock:** the backend API (via MSW), `matchMedia` (polyfilled in setup). **What NOT to mock:** React Query cache behavior should be exercised through real `queryFn`/`mutationFn` calls.

### Fixtures

- No shared fixture library; test data is built inline as plain objects (e.g., `mockData` literal in `web/src/api/__tests__/auth.test.ts:14-36`).
- Zod schemas are sometimes re-declared inline in tests for isolation (`web/src/lib/__tests__/validation.test.ts:12-44`).

### Coverage

- **Not enforced.** No `coverage` config in `vitest.config.ts`. View manually: `bun run test --coverage` (Vitest supports `--coverage` if a provider is added; currently none configured).

### Test Types

- **Unit:** pure logic — Zod schemas, API option builders, utility functions (`web/src/lib/__tests__/`).
- **Component:** React component rendering/interaction via Testing Library (`web/src/**/__tests__/*.test.tsx`).
- **E2E:** Playwright (see below).

## End-to-End (Playwright)

**Location:** `web/e2e/` (separate from Vitest; excluded from `vitest.config.ts`). Config: `web/playwright.config.ts`.

**Runner:** `@playwright/test` v1.62. Run: `bunx playwright test` (per `AGENTS.md`), or `cd web && bun run build` then the Playwright runner.

**Config highlights:**
- `testDir: "./e2e"`, `fullyParallel: true`, `retries: CI ? 2 : 0`, `reporter: "html"`.
- `baseURL: "http://localhost:3000"`; `webServer` runs `bun run dev`, waits for `http://localhost:3000`, `reuseExistingServer` unless CI, timeout 120s.
- Single project `chromium` (Desktop Chrome).
- `trace: "on-first-retry"`.

**Test files:** `*.spec.ts` — `auth.spec.ts`, `time-entries.spec.ts`, `expenses.spec.ts`, `approvals.spec.ts`, `contracts.spec.ts`, `customers.spec.ts`, `activities.spec.ts`, `working-groups.spec.ts`, `org-hierarchy.spec.ts`, `error-boundary.spec.ts`.

**Helpers:** `web/e2e/helpers.ts` provides `psql(sql)` (runs SQL via `docker exec hourglass-postgres psql ...`), `registerUser(request, prefix)` (API-based registration), and session-cookie injection. E2E suites register/login ONCE in `beforeAll` and inject cookies into each context; datasets are seeded via direct Postgres inserts (`web/e2e/helpers.ts:1-40`). Note: rate limits should be raised for e2e (`RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server`).

---

*Testing analysis: 2026-08-25*
