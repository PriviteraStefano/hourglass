# Coding Conventions

**Analysis Date:** 2026-06-08

## Naming Patterns

**Files:**
- **Go backend:** `snake_case.go` — e.g., `auth.go`, `time_entry.go`, `working_group.go`, `exported_test_helpers.go`
- **Frontend (React/TypeScript):** kebab-case for components and pages — e.g., `app-shell.tsx`, `status-badge.tsx`, `theme-provider.tsx`, `time-entries.tsx`
- **Frontend API modules:** camelCase file names — e.g., `auth.ts`, `time-entries.ts`, `query-client.ts`
- **Frontend types:** camelCase file names — e.g., `api.ts`, `models.ts`, `unit.ts`
- **Go test files:** `*_test.go` co-located with source — e.g., `auth_test.go` next to `auth.go`
- **Frontend test files:** `*.test.ts` in `__tests__/` directories — e.g., `auth.test.ts`, `api.test.ts`
- **E2E tests:** `*.spec.ts` — e.g., `auth.spec.ts`, `time-entries.spec.ts`

**Functions:**
- **Go:** PascalCase for exported functions, camelCase for unexported. Handler methods use `(h *Handler)` receiver pattern:
  ```go
  func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) { ... }
  func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) { ... }
  ```
- **Go service constructors:** `NewService(...)` returning `*Service`
- **Go test functions:** PascalCase with `Test` prefix — `TestService_Register`, `TestRepository_Add_GetByID`
- **Frontend:** camelCase for functions and methods — `api<T>()`, `useIsMobile()`, `setupTestServer()`

**Variables:**
- **Go:** camelCase — `userRepo`, `orgRepo`, `tokenSvc`, `pwHasher`
- **Frontend:** camelCase — `queryClient`, `server`, `mockData`
- **Frontend constants:** UPPER_SNAKE_CASE for env keys — `API_BASE`, `MOBILE_BREAKPOINT`

**Types:**
- **Go:** PascalCase — `Role`, `EntryStatus`, `GovernanceModel`, `TimeEntry`, `UserWithMembership`
- **Frontend (TypeScript):** PascalCase for interfaces and types — `AuthResponse`, `LoginRequest`, `CreateTimeEntryRequest`
- **Frontend Zod schemas:** PascalCase with `Schema` suffix — `CreateUnitRequestSchema`, `UnitSchema`

## Code Style

**Formatting:**
- **Go:** Uses standard `gofmt` (no `.golangci.yml` detected). Standard library `net/http` for HTTP server, `encoding/json` for JSON handling.
- **Frontend:** ESLint via `web/eslint.config.js` with `typescript-eslint` recommended, `eslint-plugin-react-hooks`, and `eslint-plugin-react-refresh`. No Prettier config detected — formatting governed by ESLint only.
- **Frontend:** Single quotes for imports (ESLint default), semicolons used.

**Linting:**
- **Go:** No golangci-lint config found. Standard `go vet` expected from `Makefile` test target.
- **Frontend:** ESLint 9+ flat config (`web/eslint.config.js`):
  ```
  import js from '@eslint/js'
  import globals from 'globals'
  import reactHooks from 'eslint-plugin-react-hooks'
  import reactRefresh from 'eslint-plugin-react-refresh'
  import tseslint from 'typescript-eslint'
  ```
- **Frontend command:** `cd web && bun run lint` (runs `eslint .`)

**TypeScript Configuration:**
- `web/tsconfig.json`: Strict mode enabled, `ES2022` target, `bundler` module resolution, `react-jsx` JSX transform. Path alias `@/` → `./src/*`.
- Key strict settings: `strict: true`, `noFallthroughCasesInSwitch: true`, `isolatedModules: true`
- No unused locals/parameters: `noUnusedLocals: false`, `noUnusedParameters: false` (relaxed)

## Import Organization

**Go:**
1. Standard library packages (e.g., `"encoding/json"`, `"net/http"`, `"testing"`)
2. Third-party packages (e.g., `"github.com/google/uuid"`, `"github.com/stretchr/testify/assert"`)
3. Internal project packages (e.g., `"github.com/stefanoprivitera/hourglass/internal/core/services/auth"`)
Groups separated by blank lines. Example from `internal/adapters/primary/http/auth.go`:
```go
import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/cookies"
	"github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)
```

**Frontend:**
1. React/hooks imports (e.g., `import {StrictMode} from 'react'`)
2. Library/framework imports (e.g., `import {createFileRoute} from '@tanstack/react-router'`)
3. Internal path aliases using `@/` (e.g., `import {api} from "@/lib/api.ts"`)
4. Type imports using `import type { ... }` syntax
Groups separated by blank lines. Example from `web/src/api/auth.ts`:
```typescript
import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {api} from "@/lib/api.ts";
import type {AuthResponse, LoginRequest, ...} from "@/types";
```

**Path Aliases:**
- `@/` maps to `web/src/` — used throughout frontend: `@/lib/api.ts`, `@/types`, `@/api/auth.ts`

## Error Handling

**Go:**
- **Sentinel errors** using `errors.New("...")` declared as package-level `var`:
  ```go
  var (
      ErrEmailExists        = errors.New("email already registered")
      ErrInvalidCreds       = errors.New("invalid credentials")
      ErrAccountDeactivated = errors.New("account is deactivated")
  )
  ```
- **Error matching** uses `errors.Is()` in assertions: `assert.ErrorIs(t, err, tt.wantErr)`
- **Switch on error type** in handlers to map to HTTP status codes:
  ```go
  if err != nil {
      switch err {
      case auth.ErrInvalidCreds:
          api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
      case auth.ErrAccountDeactivated:
          api.RespondWithError(w, http.StatusForbidden, "account is deactivated")
      default:
          api.RespondWithError(w, http.StatusInternalServerError, "login failed")
      }
      return
  }
  ```
- **Error wrapping** not observed (direct `err` comparison used)
- **Response envelope** via `pkg/api/response.go`: `{ data: ... }` on success, `{ error: ... }` on failure

**Frontend:**
- API errors thrown as `new Error(...)` with message from server response:
  ```typescript
  if (!res.ok) {
      const error = await res.json().catch(() => ({message: 'Request failed'})) as ApiError
      throw new Error(error.message || error.error || 'Request failed')
  }
  ```
- Validation uses **Zod schemas** (`web/src/types/unit.ts`, route-level schemas) with `.safeParse()`
- React Query `mutationOptions` `onError` handlers redirect on failure: `location.href = '/login'`
- Catch-all `try/catch` with redirect in auth hydration:
  ```typescript
  try {
      const profile = await client.fetchQuery(AuthApis.profileQueryOpts)
      return { profile }
  } catch {
      throw redirect({ to: '/login' })
  }
  ```

## Logging

**Go:**
- `testing.TB.Logf` used in tests for debug output
- No dedicated logging library detected in source handlers; standard `log` package expected
- Request logging middleware in `internal/middleware/logging_test.go` indicates structured logging middleware exists

**Frontend:**
- No structured logging framework detected. `console` usage not lint-restricted.
- `sonner` library (`^2.0.7`) available in dependencies for toast notifications

## Comments

**Go:**
- **Section marker comments** with dash separators in test files:
  ```go
  // ---------------------------------------------------------------------------
  // TestService_Register
  // ---------------------------------------------------------------------------
  ```
- **t.Helper()** marker comment for test helper functions: `// seedContract creates a test contract linked to an org.`
- **Standard Go doc comments** for exported functions and types

**Frontend:**
- Comments used sparingly. Schema sections annotated with comment dividers:
  ```typescript
  // ── Exported schemas from @/types/unit ──────────────────────────────────
  const tabsSchema = z.enum(['owned', 'adopted', 'all'])
  ```

## Function Design

**Size:**
- **Go handlers:** Thin, typically 30-80 lines. Example from `auth.go` — `Register` is ~30 lines.
- **Go services:** Moderate, typically 50-150 lines per method. `auth.go` service is 539 lines total across all methods.
- **Frontend API modules:** Each `queryOptions`/`mutationOptions` is a standalone 5-15 line declaration.
- **Frontend page components:** Very thin, typically 10-30 lines. Route files use inline arrow functions.

**Parameters:**
- **Go:** Handler functions use `(w http.ResponseWriter, r *http.Request)` signature
- **Go:** Service functions use `ctx context.Context` as first parameter followed by request struct
- **Frontend:** Query/mutation options defined as functions or constants, using TypeScript generics `<T>`

**Return Values:**
- **Go services:** Return `(resp *ResponseType, err error)` tuple
- **Go handlers:** Write response directly via `api.RespondWithJSON(w, status, payload)`
- **Frontend `api<T>()`:** Returns `Promise<T>` — unwraps `{ data: T }` envelope

## Module Design

**Exports:**
- **Go handlers:** Exported `*Handler` struct, unexported fields, constructor `New*Handler()` returns pointer
- **Go services:** Exported `*Service` struct, unexported fields, constructor `NewService()` returns pointer
- **Frontend API modules:** Export an `*Apis` object aggregating all query/mutation options:
  ```typescript
  export const AuthApis = {
      profileQueryOpts,
      loginMutationOpts,
      registerMutationOpts,
      // ...
  }
  ```
- **Frontend types:** Export interfaces/types and aggregated barrel file `web/src/types/index.ts`:
  ```typescript
  export * from './models'
  export * from './api'
  ```

**Barrel Files:**
- `web/src/types/index.ts` re-exports from `models.ts` and `api.ts`
- `web/src/components/ui/` components are individually imported — no barrel file for shadcn UI

## Test Conventions

**Go:**
- **Table-driven tests** are the standard pattern for service tests:
  ```go
  tests := []struct {
      name    string
      req     RegisterRequest
      setup   func(*testdata.MockUserRepo)
      wantErr error
  }{ ... }
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- **Subtests** via `t.Run()` for each test case
- **`require.NoError`** used for fatal assertions, **`assert`** for non-fatal checks
- **`t.Helper()`** called in all helper/seed functions
- **Sentinel error comparison** via `assert.ErrorIs(t, err, tt.wantErr)`

**Frontend:**
- **Vitest + MSW** for HTTP mocking in API tests
- **Describe/it blocks** with BDD-style naming
- **beforeAll/afterAll/afterEach** lifecycle for MSW server management
- **Testing Library** (`@testing-library/react`, `@testing-library/jest-dom`) for component tests

---

*Convention analysis: 2026-06-08*
