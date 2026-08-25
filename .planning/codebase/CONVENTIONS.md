# Coding Conventions

**Analysis Date:** 2026-08-25

This document covers both the Go backend (`./`) and the TypeScript React frontend (`./web`). They use different toolchains and conventions.

## Languages

**Primary (Backend):**
- Go 1.26.1 — `go.mod`. All backend code under `cmd/`, `internal/`, `pkg/`.

**Primary (Frontend):**
- TypeScript (strict) — `web/tsconfig.json`. React 19 with JSX (`"jsx": "react-jsx"`).

## Backend Conventions (Go)

### Naming Patterns

**Files:**
- `snake_case.go` for all source and test files (e.g., `auth.go`, `user_repository_test.go`).
- Tests are siblings: `auth_test.go`, `auth_integration_test.go`. Integration tests use the `*_integration_test.go` suffix.

**Packages:**
- Lowercase single-word package names matching the directory (e.g., `package http`, `package auth`, `package postgres`).
- Hexagonal layers: `internal/core/services/<domain>` (business logic), `internal/adapters/primary/http` (HTTP handlers), `internal/adapters/secondary/postgres` (repositories).

**Types / Structs:**
- `PascalCase` exported types, `camelCase` unexported fields.
- Handler structs use the `*Handler` suffix: `AuthHandler` (`internal/adapters/primary/http/auth.go`).
- Constructors are `NewXxx(...)` and return a pointer: `NewAuthHandler(...)` (`internal/adapters/primary/http/auth.go:22`).

**Functions:**
- Receivers use a short, consistent abbreviation of the type (`h *AuthHandler`, `m *MockUserRepo`).
- HTTP handler methods: `func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)`.

**Errors:**
- Sentinel errors defined as package-level `var`/`const` and compared with `errors.Is` / `switch err`: `auth.ErrEmailExists`, `auth.ErrUsernameExists` (`internal/adapters/primary/http/auth.go:67-70`).
- Port-level not-found errors live in `internal/core/ports`: `ports.ErrUserNotFound` (`internal/adapters/secondary/postgres/user_repository_test.go:44`).

### Code Style

**Formatting:** `gofmt` / `go fmt` is the de-facto standard (no custom formatter configured). No Makefile or CI step for `gofmt` is present, but Go idioms are followed.

**Linting:** No `golangci-lint` config (`.golangci.yml` absent). CI runs Qodana only (`.github/workflows/qodana_code_quality.yml`).

**Imports:** Grouped standard-library first, then third-party (`github.com/...`), then internal `github.com/stefanoprivitera/hourglass/...`.

### Error Handling

- Handlers decode JSON, validate string lengths via `validateStringLengths(w, lengthField(...))` helpers, then delegate to a service. On error they map domain sentinels to HTTP status via `api.RespondWithError(w, status, msg)` (`internal/adapters/primary/http/auth.go:41-74`).
- Success via `api.RespondWithJSON(w, http.StatusOK, payload)` using the envelope in `pkg/api` (`Data`/`Error` fields).
- Services return `error` as the last return value; callers use `require.NoError` / `require.ErrorIs` in tests.
- No `panic` for control flow; errors propagate up to handlers.

### Module / API Design

- Handlers stay thin — all business logic is in `internal/core/services/*` (hexagonal architecture; see `plans/hexagonal-migration.md`).
- Request/response DTOs are declared as local structs inside the handler file (e.g., `RegisterRequest` in `internal/adapters/primary/http/auth.go:26`).
- Domain services depend on repository interfaces defined in `internal/core/ports`, not concrete implementations.

## Frontend Conventions (TypeScript / React)

### Naming Patterns

**Files:**
- `kebab-case.ts(x)` for component and utility files (e.g., `entries-table.tsx`, `status-badge.tsx`, `api.ts`). See `AGENTS.md`.
- Test files are co-located in a `__tests__/` directory beside the source and named `*.test.ts(x)` (e.g., `web/src/api/__tests__/auth.test.ts`, `web/src/components/shared/__tests__/entries-table.test.tsx`).

**React Components:**
- `PascalCase` component names exported from `kebab-case` files.

**Types:**
- Shared API types live in `web/src/types/` (e.g., `web/src/types/unit.ts`, `web/src/types/api.ts`) and are imported via `@/types` path alias.
- Domain schemas are Zod schemas (`z.object(...)`, `z.enum(...)`) — see `web/src/lib/__tests__/validation.test.ts:12-44`.

**Variables / Functions:**
- `camelCase` for variables and functions; `PascalCase` for types/classes.
- TanStack Query options are exported as `const`: `profileQueryOpts`, `loginMutationOpts` (`web/src/api/auth.ts:14-50`).

### Code Style

**Formatting:** `oxfmt` — config in `web/.oxfmtrc.json`: `printWidth: 80`, `tabWidth: 2`, spaces (no tabs), `semi: true`, `singleQuote: false`, `trailingComma: "es5"`. Run via `bun run fmt` / `bun run fmt:check`.

**Linting:** `oxlint` (type-aware) — config in `web/.oxlintrc.json`. Plugins: `typescript`, `react`, `jsx-a11y`, `unicorn`, `import`. `correctness` category is `warn`. Env: `browser`. Ignores `dist`, `node_modules`, `src/routeTree.gen.ts`, `e2e`, `test-results`, `**/*.gen.ts`. Run via `bun run lint` / `bun run lint:fix`.

**Type checking:** `tsc -b` (strict). `noUnusedLocals`/`noUnusedParameters` are `false` but `strict: true` is on. `erasableSyntaxOnly: true`. Path alias `@/*` -> `./src/*` (`web/tsconfig.json`).

**Import organization:** Internal imports use the `@/` alias (e.g., `import { api } from "@/lib/api.ts"`). `verbatimModuleSyntax`-style type imports use `import type`.

### Error Handling

- Central HTTP client `api<T>()` in `web/src/lib/api.ts:36` throws `UnauthorizedError` when auth refresh fails (`web/src/lib/api.ts:11-16`), otherwise `new Error(error.message || error.error || "Request failed")` (`web/src/lib/api.ts:86`).
- `api()` auto-retries once through `POST /auth/refresh` on 401 unless the path is an auth path (`AUTH_PATHS`, `web/src/lib/api.ts:23-32`). Prevents recursion/redirect loops.
- 204 No Content returns `undefined` (avoids JSON parse error on DELETE).
- React Query `queryOptions`/`mutationOptions` defined in `web/src/api/*` modules; `onSuccess` handlers update the query cache via `client.setQueryData(["auth", "me"], ...)` (`web/src/api/auth.ts:27-35`).

### Module Design

- API calls are centralized as TanStack Query `queryOptions`/`mutationOptions` in `web/src/api/*.ts` (one file per domain: `auth.ts`, `time-entries.ts`, `customers.ts`, etc.).
- Shared client `web/src/lib/api.ts` provides the `api<T>()` helper used by all query/mutation fns.
- `web/src/routes/` uses TanStack Router file-based routing; protected routes use `beforeLoad` to hydrate auth (`web/src/routes/_authenticated.tsx`).
- No barrel `index.ts` re-export pattern is enforced; imports target specific files.

---

*Convention analysis: 2026-08-25*
