# Coding Conventions

**Analysis Date:** 2026-08-12

## Naming Patterns

**Files:**
- Go: snake_case matching the primary type, suffixed by layer — `user_repository.go`, `time_entry_repository.go`, `auth.go`, `handler_test_helper.go` (`internal/adapters/secondary/postgres/`, `internal/adapters/primary/http/`, `internal/core/services/`)
- Frontend: kebab-case — `status-badge.tsx`, `app-shell.tsx`, `entries-filters.tsx`, `time-entries-list.tsx` (`web/src/components/`, `web/src/routes/_authenticated/.../-components/`)
- shadcn primitives live in `web/src/components/ui/` (lowercase single-word files: `button.tsx`, `dialog.tsx`, `table.tsx`)
- Test files: `*_test.go` co-located with source (Go); `*.test.ts`/`*.test.tsx` inside a `__tests__/` folder next to the source (frontend)

**Functions:**
- Go: exported methods on `*Handler`/`*Repository` receivers in PascalCase (`(h *AuthHandler) Register`, `(r *UserRepository) GetByEmail`); constructors `NewAuthHandler(...)`, `NewUserRepository(pool)`; doc comment on every exported function: `// GetByEmail returns the first user matching the given email, or ErrUserNotFound.`
- Go test helpers always call `t.Helper()` first (see `internal/adapters/secondary/postgres/exported_test_helpers.go`, `internal/adapters/primary/http/handler_test_helper.go`)
- Frontend: exported named function components (`export function StatusBadge(...)`) — never default exports; hooks prefixed `use` (`use-mobile.ts`); React Query option factories named `xxxQueryOpts` / `xxxMutationOpts` and aggregated under a PascalCase `XxxApis` namespace object (`web/src/api/auth.ts` exports `AuthApis.profileQueryOpts`, `loginMutationOpts`, ...)

**Variables:**
- Go: short camelCase; receivers `h`, `r`, `f` (fixture), `svc`
- Frontend: camelCase locals; request/response field names use snake_case to mirror the Go JSON contract (`organization_name`, `is_active`, `created_at` in `web/src/types/api.ts`); `const PREFIX = \`auth_${Date.now()}\`` style for e2e uniqueness

**Types:**
- Go: domain structs in `internal/core/domain/*` (`auth.User`, `time_entry.TimeEntry`); service request/response types named `<Verb>Request`/`<Verb>Response` or `RegisterRequest`/`LoginResponse` (`internal/core/services/auth/auth.go`); JSON tags snake_case with `omitempty` for optionals: `json:"activated_at,omitempty"`
- Frontend: `interface` for API shapes in `web/src/types/api.ts` and `web/src/types/models.ts`, re-exported through the barrel `web/src/types/index.ts`; components declare `export interface XxxProps`; `import type` for type-only imports (`.oxlintrc` + codebase practice)

## Code Style

**Formatting:**
- Go: `gofmt` standard (no `.golangci.yml`, no format config file); go.mod requires Go 1.26.1; imports grouped stdlib first, then third-party (`internal/core/services/auth/auth_test.go`)
- Frontend: oxfmt (`web/.oxfmtrc.json`) — printWidth 80, tabWidth 2, semicolons, **double quotes**, trailing commas (es5). Run with `bun run fmt` / check `bun run fmt:check`
- Frontend imports use explicit `.ts`/`.tsx` extensions for relative and `@/` imports (`allowImportingTsExtensions: true` in `web/tsconfig.json`): `import { Body } from "@/components/layout/body.tsx"`

**Linting:**
- Frontend: oxlint (`web/.oxlintrc.json`) — plugins `typescript`, `react`, `jsx-a11y`, `unicorn`, `import`; `typeAware: true`; correctness category at `warn`; ignores `dist`, `node_modules`, `src/routeTree.gen.ts`, `e2e`, `**/*.gen.ts`. Run: `bun run lint`, auto-fix `bun run lint:fix`
- Go: static analysis via Qodana CI (`qodana.yaml`, `qodana.starter` profile, `jetbrains/qodana-jvm-community:2026.1`, GitHub workflow `.github/workflows/qodana_code_quality.yml`)

## Import Organization

**Order:**
1. React/external packages (`react`, `@tanstack/react-query`, `lucide-react`)
2. `@/` aliased internal modules (components, api, lib, types)
3. Relative imports within feature folders

**Path Aliases:**
- `@/*` → `./src/*` (`web/tsconfig.json`, mirrored in `web/vitest.config.ts` and `web/vite.config.ts`); use `@/lib/api.ts`, `@/components/ui/button`, `@/types` — never deep relative paths

## Error Handling

**Backend (Go):**
- Sentinel errors declared per domain/service package: `var ErrTimeEntryNotFound = errors.New("time entry not found")` in `internal/core/domain/time_entry/time_entry.go`, service-level in `internal/core/services/auth/auth.go` (`ErrEmailExists`, `ErrInvalidCreds`, ...)
- Services return sentinel errors; handlers map them to HTTP status via `switch err` — `case auth.ErrEmailExists: api.RespondWithError(w, http.StatusConflict, "email already registered")` (`internal/adapters/primary/http/auth.go:65-74`)
- Repositories wrap with context: `return wrapPGError(err, "add user")` (`internal/adapters/secondary/postgres/user_repository.go:31`, helper in `internal/adapters/secondary/postgres/postgres.go`); not-found sentinel errors live on ports interfaces (`internal/core/ports/user_repository.go:11`)
- Shared response envelope from `pkg/api/response.go`: `RespondWithJSON(w, status, payload)` → `{"data": ...}`; `RespondWithError(w, status, msg)` → `{"error": ...}`
- Handler validation before service call: `validateStringLengths(w, lengthField("email", req.Email, MaxEmailLength))` (`internal/adapters/primary/http/auth.go:45-54`)

**Frontend (TypeScript):**
- Single HTTP client `api<T>()` in `web/src/lib/api.ts` throws `Error(message)` from the `{error}` envelope; custom `UnauthorizedError` for expired sessions; 401 triggers one single-flight refresh attempt (`refreshPromise`) then retries once; auth paths never trigger refresh
- **The HTTP client never navigates** — route guards catch `UnauthorizedError` and call `redirect({ to: "/login" })` (`web/src/routes/_authenticated.tsx`)
- Route loader/query failures render `RouteError` (`web/src/components/layout/route-error.tsx`) with `router.invalidate()` retry; leaf-level `errorComponent` per route (`web/src/routes/_authenticated/time-entries/index.tsx:33`)
- Mutations set cache in `onSuccess` (`client.setQueryData`) rather than refetch; auth mutations never hard-redirect (documented at `web/src/api/auth.ts:79-84`)

## Logging

**Framework:** Go standard library `log` — no slog/logrus. Request logging middleware in `internal/middleware/middleware.go:145-153`: `log.Printf("%s %s %d %dms", r.Method, r.URL.Path, rw.statusCode, duration.Milliseconds())`. Tests capture output via `log.SetOutput(&buf)` (`internal/middleware/logging_test.go`).

**Frontend:** `console.error` only inside dev-gated blocks (`if (import.meta.env.DEV) console.error("[route-error] loader/query failed:", error)` in `web/src/components/layout/route-error.tsx`); `console.log` not used in src.

## Comments

**When to Comment:**
- Go: exported declarations get `//` doc comments; section separators in tests: `// ---------------------------------------------------------------------------` / `// TestService_Register` banners (`internal/core/services/auth/auth_test.go`)
- Frontend: JSDoc `/** */` blocks explain *why* and cite plan/ADR references — e.g. `web/src/lib/api.ts:5-10` (refresh loop root cause), `web/src/components/layout/route-error.tsx:14-24` (invalidate vs reset), `web/src/routes/_authenticated/-components/today-page.tsx:40-49` (ADR-P-011). Plan codes are cited inline: `(Plan 10-04)`, `(ADR-FE-017)`, `(D-13-18..23)`

## Function Design

**Size:** Handlers stay thin — decode → validate → call service → respond (`internal/adapters/primary/http/auth.go`). Business logic lives in `internal/core/services/*`, never in handlers.

**Parameters:** Go methods take `ctx context.Context` first, then domain values; repositories take `uuid.UUID` IDs. Frontend component props are `interface XxxProps` objects; generic components use generics (`EntriesTable<T>` in `web/src/components/shared/entries-table.tsx`).

**Return Values:** Services return `(*T, error)`; not-found returned as typed sentinel error (no nil-checks by callers beyond `errors.Is`). Frontend query options return `queryOptions<T>({ queryKey, queryFn })`; mutations return `mutationOptions` with `mutationFn` receiving the request payload typed (`web/src/api/auth.ts`).

## Module Design

**Exports:** Go — one constructor + receiver methods per package; frontend — named exports only; `XxxApis` object aggregates all options per domain (`AuthApis`, `TimeEntriesApis`, `ActivitiesApis`, `ExpensesApis`, `ContractsApis`, `CustomersApis` in `web/src/api/*.ts`).

**Barrel Files:** `web/src/types/index.ts` re-exports `api.ts` + `models.ts` types (import as `import type { TimeEntry } from "@/types"`); `web/src/components/layout/index.ts` exists; feature pages import from explicit file paths rather than index barrels.

## Frontend State & Data Conventions

- All server state via TanStack Query; shared client in `web/src/lib/query-client.ts` (`retry: false`, `staleTime: 30000`, `refetchOnWindowFocus: false`), passed to both `QueryClientProvider` and router context (`context.client` in `web/src/main.tsx`)
- Route loaders use `client.ensureQueryData(XxxApis.xxxQueryOpts(...))` inside `Promise.all` (`web/src/routes/_authenticated/time-entries/index.tsx:19-29`); component-level data uses `useQuery` (non-suspense) with `Skeleton`/locked error states (`web/src/routes/_authenticated/-components/today-page.tsx`)
- URL state: `validateSearch` with zod schemas (`z.object({ date: z.coerce.date().default(new Date()), ... })`), `loaderDeps: ({ search }) => search` (`web/src/routes/_authenticated/time-entries/index.tsx:10-18`)
- Validation schemas: zod; shared schemas exported from `web/src/types/unit.ts` (`CreateUnitRequestSchema`), route-local schemas recreated inline in tests (`web/src/lib/__tests__/validation.test.ts`)
- Form validation: react-hook-form with zod resolver; UI primitives from `web/src/components/ui/` (shadcn-style, configured in `web/components.json`)
- Styling: Tailwind CSS v4 (`@tailwindcss/vite`), `cn()` helper from `web/src/lib/utils.ts` (clsx + tailwind-merge); icons from `lucide-react` with `Icon` suffix (`LoaderIcon`, `ChevronRightIcon`)

---

*Convention analysis: 2026-08-12*
