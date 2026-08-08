# Coding Conventions

**Analysis Date:** 2026-08-08

Two distinct stacks with separate conventions: a Go backend (`/internal`, `/cmd`, `/pkg`) and a TypeScript/React frontend (`/web`). Rules below are split by stack.

## Backend (Go)

### Naming Patterns

**Files:**
- Snake case: `time_entry_repository.go`, `handler_test_helper.go`, `exported_test_helpers.go`
- Feature files per bounded context: `internal/adapters/primary/http/auth.go`, `internal/core/services/activity/activity.go`, `internal/core/domain/activity/activity.go`

**Packages:**
- Short lowercase names: `http`, `postgres`, `activity`, `testdata`
- Domain packages aliased at import with a `<context>domain` suffix to disambiguate: `activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"` (see `internal/core/services/activity/activity.go`)
- Service packages use the short service name: `tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"` (see `internal/adapters/primary/http/handler_test_helper.go`)

**Types & functions:**
- Constructors: `NewService(...)`, `NewHandler(...)`, `New<X>Repository(pool)` — always return `*X`
- Handlers: struct `XHandler` with exported method per action: `func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)` (`internal/adapters/primary/http/auth.go`)
- Services: struct `Service` holding `ports.XRepository` interfaces, methods take `ctx context.Context` first, then role/org/user identifiers: `func (s *Service) Create(ctx context.Context, role string, orgID, userID uuid.UUID, req *activitydomain.CreateActivityRequest)` (`internal/core/services/activity/activity.go`)
- Repository interfaces named `<Domain>Repository` in `internal/core/ports/` (`internal/ports/activity_repository.go`)
- Compile-time interface assertions on every repository: `var _ ports.ActivityRepository = (*ActivityRepository)(nil)` (`internal/adapters/secondary/postgres/activity_repository.go:25`)

### Error Handling

**Sentinel errors everywhere** — no typed error hierarchies, no error wrapping chains:
- Domain sentinels: `var ErrInvalidRequest = errors.New("invalid request")` defined per bounded context (`internal/core/domain/activity/activity.go:14`)
- Service-level sentinels in a dedicated `errors.go` file (e.g., `internal/core/services/auth/errors.go`)
- Generic port sentinels: `ErrNotFound`, `ErrConflict`, `ErrForeignKey` in `internal/core/ports/errors.go`

**Propagation pattern:**
1. Service validates and returns `nil, domain.ErrX` for rule violations — never raw `fmt.Errorf`
2. Repositories normalize driver errors to port sentinels (`errors.Is(err, pgx.ErrNoRows)` → `ports.ErrNotFound`) — see `internal/adapters/secondary/postgres/activity_repository.go`
3. Cross-domain sentinels are normalized at the service boundary with `errors.Is`: a missing contract repo sentinel maps to the activity `ErrInvalidRequest` (`internal/core/services/activity/activity.go:113`)
4. Handlers switch on the sentinel to pick the HTTP status (`internal/adapters/primary/http/auth.go:66-73`); anything not matched falls to `http.StatusBadRequest` or a 500 default

**Assertion style:** always `assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)` in tests — never string matching.

### HTTP Response Envelope

Every response goes through `pkg/api/response.go`:
- Success: `api.RespondWithJSON(w, http.StatusOK, payload)` → `{"data": ...}`
- Failure: `api.RespondWithError(w, http.StatusBadRequest, "message")` → `{"error": ...}`
- Handlers never write JSON directly; they set `Content-Type: application/json` only via the `api` package

### Validation

- `internal/adapters/primary/http/validate.go` provides `validateStringLengths(w, lengthField("email", req.Email, MaxEmailLength), ...)` — a variadic field-length validator that writes the 400 response itself and returns false
- Handlers call it immediately after body decode, before service calls (`internal/adapters/primary/http/auth.go:45-54`)
- Length constants (`MaxEmailLength`, `MaxNameLength`, `MaxPasswordLength`, `MaxShortStringLength`) live in `internal/adapters/primary/http/validate.go`
- Deeper rule validation (kinds catalog, governance model, cross-org refs) is service-side, returning domain sentinels

### Comments & Documentation

- Every exported service/repository type and non-trivial method carries a doc comment; comments routinely cite ADR numbers and plan IDs: `// Service implements the activity business rules (ADR-P-007 D-2/D-3/D-5/D-8, ADR-P-013 origins)` (`internal/core/services/activity/activity.go:19`)
- "Why" comments for subtle behavior, e.g. the `sync.Once` container lifetime warning in `internal/adapters/secondary/postgres/test_setup.go:52-56`
- Test files use `// ------...` banner separators around suites (`internal/core/services/activity/activity_test.go:46-48`)
- Comment style: full sentences with the subject named ("UnitRepository is wired per the R-4 visibility axis"), not imperative fragments

### Logging

- Standard library `log` only: `log.Printf`, `log.Println`, `log.Fatal` — no structured logging, no logging library
- Request logging is a middleware: `log.Printf("%s %s %d %dms", r.Method, r.URL.Path, rw.statusCode, duration.Milliseconds())` (`internal/middleware/middleware.go:153`)
- Startup/init logs with severity prefixes: `log.Fatal("FATAL: ...")`, `log.Println("WARNING: ...")` (`cmd/server/main.go:38-40`)
- Services do not log; only adapters/middleware/entry points do

### Imports

- Standard group order enforced by `gofmt`: stdlib → external (`github.com/google/uuid`) → project (`github.com/stefanoprivitera/hourglass/...`)
- Within the project group, internal/core/domain imports precede services imports (see `internal/adapters/primary/http/handler_test_helper.go`)

## Frontend (TypeScript / React)

### Naming Patterns

**Files:**
- kebab-case for components, hooks, libs, api modules: `app-shell.tsx`, `status-badge.tsx`, `use-download.ts`, `query-client.ts`, `time-entries.ts`
- Test files: `<name>.test.ts` / `<name>.test.tsx` (never `.spec.ts` for unit tests) in a sibling `__tests__/` directory
- E2E files: `<name>.spec.ts` under `web/e2e/`
- Route-local components live in `-components/` folders: `web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx`

**Functions & variables:** camelCase; boolean flags `isActive`, `hasX` style.

**Components & types:** PascalCase components (`TimeEntriesPage`); interfaces `ApiResponse<T>`, `UserWithMembership` in `web/src/types/api.ts` and `web/src/types/models.ts`, re-exported from `web/src/types/index.ts`.

**Query option exports:** each domain module in `web/src/api/` exports a single `XApis` object grouping `queryOptions`/`mutationOptions` — `export const AuthApis = { profileQueryOpts, loginMutationOpts, ... }` (`web/src/api/auth.ts:182`). Component files consume `AuthApis.profileQueryOpts`.

### Code Style

**Formatting (oxfmt, `web/.oxfmtrc.json`):**
- printWidth 80, tabWidth 2, spaces (no tabs), semicolons, double quotes, trailingComma es5
- Run: `cd web && bun run fmt` / `bun run fmt:check`

**Linting (oxlint, `web/.oxlintrc.json`):**
- Plugins: typescript, react, jsx-a11y, unicorn, import; type-aware mode on
- `correctness` category set to `warn` (does not fail lint)
- `web/src/routeTree.gen.ts`, `**/*.gen.ts`, `e2e/`, `dist/` are ignored
- Run: `cd web && bun run lint` (type-aware), `bun run lint:fix`
- Typecheck: `bun run typecheck` (`tsc -b`); `bun run build` runs it too

### Import Organization

1. External packages first (react, @tanstack/*, zod)
2. `@/` alias imports (maps to `web/src` via tsconfig/vite/vitest aliases)
3. Relative imports for local siblings

**Critical convention — explicit file extensions on all local imports:** `import { api } from "@/lib/api.ts"`, `import { TimeEntriesPage } from "@/routes/.../-components/time-entries-page.tsx"`, `import type { ... } from "@/types"` (via `types/index.ts` barrel). Follow this even though bundlers tolerate extensionless paths.

**Type-only imports:** `import type { ... } from "..."` used consistently for types (`web/src/api/auth.ts:3`).

### React Query Patterns

- All server state flows through `queryOptions`/`mutationOptions` defined in `web/src/api/*.ts`; route loaders call `client.ensureQueryData(...)`, components use `useQuery`/`useMutation` or `useSuspenseQuery`
- Query keys are kebab-case arrays with scoping segments: `["auth", "me"]`, `["auth", "memberships"]`, `["invitations", "code", code]`, `["time-entries"]`
- Shared client in `web/src/lib/query-client.ts` (`retry: false`, `staleTime: 30000`, `refetchOnWindowFocus: false`); injected into the router as `context.client` in `web/src/main.tsx`
- `onSuccess` writes into the query cache with `client.setQueryData(["auth", "me"], ...)` instead of separate cache-write helpers (`web/src/api/auth.ts:27-35`)
- Mutations invalidate: `queryClient.invalidateQueries({ queryKey: ["time-entries"] })`

### HTTP Client (`web/src/lib/api.ts`)

- Single `api<T>(path, options?)` helper — fetch wrapper with:
  - `credentials: "include"` always
  - `Content-Type: application/json` default
  - 401 → single-flight refresh (`POST /auth/refresh` guarded by module-level `refreshPromise`) then one retry; `AUTH_PATHS` excluded to prevent recursion
  - Unwraps the `{ data }` envelope; throws `Error(error.message || error.error)` on non-OK
  - `204` returns `undefined` (all DELETE handlers)
  - Throws `UnauthorizedError` when refresh fails — route guards catch it and `redirect()` to `/login`. **The client never navigates itself.**
- New API calls must go through `api<T>()`; do not call `fetch` directly elsewhere

### Forms & Validation

- zod v4 for all validation; `validateSearch: z.object({...})` in every route with search params (`web/src/routes/_authenticated/time-entries/index.tsx:10`)
- Shared schema helpers in `web/src/lib/list-filters.ts` (e.g. `listStatusesSchema`, `entryStatusSchema`) reused across routes
- react-hook-form + `@hookform/resolvers` for complex forms; TanStack Form for newer forms

### Error Handling (Frontend)

- Loader errors → route `errorComponent: RouteError` (`web/src/components/layout/route-error.tsx`), registered per-route not just at layout level so the error boundary resets on navigation
- `UnauthorizedError` is the only custom error class (`web/src/lib/api.ts:11`)
- Mutation errors surface via sonner toasts in components; no global toast plumbing
- Deliberately empty `onError` callbacks are documented with an explanatory comment (`web/src/api/auth.ts:82-84`)

### Comments

- JSDoc `/** ... */` for module-level invariants and non-obvious client behavior — see `web/src/lib/api.ts:5-16` and `web/e2e/helpers.ts:3-19`
- Inline `//` comments explain "why", frequently referencing plan/ADR ids: `// Leaf-level boundary (P0-4): the error attaches to THIS match...` (`web/src/routes/_authenticated/time-entries/index.tsx:30`)
- No TODO/FIXME litter; unresolved work is tracked in plan documents, not code comments

### Logging

- No console logging in `web/src/` (no `console.log`/`console.error` usage found). Debugging output should not be committed; use devtools.

## Shared / Cross-Cutting

- **Routing registration** (backend): Go 1.22+ method patterns on `http.ServeMux`: `mux.HandleFunc("POST /time-entries", handler.Create)` in `cmd/server/main.go`; protected routes wrap `middleware.Auth(authSvc, handler.X)`. Keep the fixture in `internal/adapters/primary/http/handler_test_helper.go` in sync when routes change.
- **Route/URL naming:** kebab-case everywhere (`/time-entries`, `/working-groups`), JSON fields snake_case, Go struct fields PascalCase.
- **Auth state:** HttpOnly `auth_token`/`refresh_token` cookies (helpers in `internal/cookies`); the frontend never touches tokens directly.
- **Docs:** conventions are additionally documented in `AGENTS.md` and `openwiki/` (see `openwiki/testing/README.md`, `openwiki/architecture/README.md`); verification scripts in `scripts/verify-*.mjs` check wiki/README claims against the codebase — keep them passing when changing behavior described there.
- **Static analysis (backend):** no golangci-lint config; `qodana.yaml` uses the `qodana.starter` profile, wired into CI via `.github/workflows/qodana_code_quality.yml`. `go vet` passes as part of normal `go test`.

---

*Convention analysis: 2026-08-08*
