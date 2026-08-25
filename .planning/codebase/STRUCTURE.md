# Codebase Structure

**Analysis Date:** 2026-08-25

## Directory Layout

```
hourglass/
├── cmd/
│   ├── server/main.go          # Backend HTTP server entry + route wiring (composition root)
│   └── migrate/main.go         # PostgreSQL migration/seeding CLI
├── internal/
│   ├── adapters/
│   │   ├── primary/http/       # Thin HTTP handlers (one file per domain + co-located _test.go)
│   │   └── secondary/postgres/ # PostgreSQL repository implementations of core ports
│   ├── auth/                   # JWT service, token service, password hasher
│   ├── cookies/                # HttpOnly cookie helpers (auth_token/refresh_token)
│   ├── core/
│   │   ├── domain/             # Entities, request/response DTOs, sentinel errors (per domain)
│   │   ├── ports/              # Repository + token/hasher interfaces (abstractions)
│   │   └── services/           # Application business logic (18 service packages)
│   ├── db/                     # PostgreSQL pool (pgx) + migrate helper
│   ├── handlers/               # Cross-cutting handlers (health)
│   ├── middleware/             # Auth, TryAuth, RateLimiter, Logging, CORS, APIVersion
│   └── models/                 # Shared constants (Role, Status, Governance) + data structs
├── pkg/
│   └── api/response.go         # JSON envelope writers ({data}/{error})
├── migrations/                 # 19 SQL up/down migrations (cmd/migrate applies)
├── web/                        # React 19 + TanStack frontend (see below)
├── openwiki/                   # Repo documentation (quickstart, architecture notes)
├── graphify-out/               # Knowledge-graph output
├── plans/                      # GSD phase plans + hexagonal-migration.md
└── Makefile / Dockerfile / docker-compose.yml
```

### Frontend (`web/`) layout
```
web/
├── src/
│   ├── main.tsx                # React root: QueryClientProvider + RouterProvider
│   ├── routeTree.gen.ts       # Generated TanStack Router tree (do not edit by hand)
│   ├── routes/                 # File-based routing
│   │   ├── __root.tsx          # Root layout
│   │   ├── _authenticated.tsx  # Protected layout guard (hydrates /auth/me)
│   │   ├── _auth/              # Public auth routes (login, register, invite, password-reset, bootstrap)
│   │   └── _authenticated/     # Protected feature routes (time-entries, activities, expenses, contracts, customers, working-groups, approvals, org-hierarchy, exports)
│   ├── api/                    # Per-domain query/mutation option objects (auth.ts, time-entries.ts, activities.ts, ...)
│   ├── lib/                    # api.ts (HTTP client w/ refresh), query-client.ts, list-filters.ts, utils.ts, role-visibility.ts, use-download.ts
│   ├── hooks/                  # Shared hooks (use-mobile.ts)
│   ├── components/             # app/, layout/, shared/, approval/, exports/, ui/ (shadcn), theme-provider.tsx
│   ├── types/                  # Shared TS types (api.ts mirrors backend models)
│   └── assets/
├── vite.config.ts              # Dev server + /api proxy → http://localhost:8080
├── package.json                # bun + vite + tanstack deps
└── e2e/                        # Playwright tests
```

## Directory Purposes

**`cmd/server/`**
- Purpose: Composition root — wiring of repos, services, handlers, middleware, and route registration.
- Contains: `main.go` (360 lines).
- Key files: `cmd/server/main.go`

**`internal/core/services/`**
- Purpose: All business logic, one package per domain (activity, time_entry, expense, coverage, direction, contract, customer, organization, working_group, unit, ticket, invitation, password_reset, orgsettings, auth, export, routing).
- Contains: `service.go` (or `*.go` split) + co-located `_test.go` + `testdata/`.
- Key files: `internal/core/services/activity/service.go`, `internal/core/services/time_entry/service.go`, `internal/core/services/routing/service.go`

**`internal/core/ports/`**
- Purpose: Interface definitions (repository ports, `TokenService`, `PasswordHasher`, `UserFinder`). The dependency-inversion seam.
- Key files: `internal/core/ports/activity_repository.go`, `internal/core/ports/errors.go`

**`internal/core/domain/`**
- Purpose: Entities, DTOs, and sentinel errors per domain. No I/O.
- Key files: `internal/core/domain/activity/activity.go`, `internal/core/domain/auth/`, `internal/core/domain/coverage/`

**`internal/adapters/primary/http/`**
- Purpose: Thin request/response translation layer mapped to service calls.
- Contains: `<domain>_handler.go` + `<domain>_handler_test.go`; shared `handler_test_helper.go`, `validate.go`.
- Key files: `internal/adapters/primary/http/activity_handler.go`, `time_entry.go`, `auth.go`

**`internal/adapters/secondary/postgres/`**
- Purpose: PostgreSQL implementations of the core ports (`pgx`), plus `*_repository_test.go` and `test_setup.go`.
- Key files: `internal/adapters/secondary/postgres/activity_repository.go`, `postgres.go`, `test_setup.go`

**`internal/middleware/`**
- Purpose: HTTP cross-cutting concerns.
- Key files: `middleware.go` (Auth/TryAuth/RequireRole/Logging), `ratelimit.go`, `cors.go`, `version.go`

**`internal/auth/`**
- Purpose: JWT creation/validation and bcrypt password hashing.
- Key files: `auth.go`, `token_service.go`, `password_hasher.go`

**`pkg/api/`**
- Purpose: Shared JSON response envelope (`{data}` / `{error}`).
- Key files: `pkg/api/response.go`

**`migrations/`**
- Purpose: Versioned SQL schema (`*.up.sql` / `*.down.sql`), applied by `cmd/migrate`.
- Count: 19 up migrations.

**`web/src/`**
- Purpose: Frontend SPA source.
- Key files: `web/src/main.tsx`, `web/src/lib/api.ts`, `web/src/api/auth.ts`, `web/src/routes/_authenticated.tsx`

## Key File Locations

**Entry Points:**
- Backend server: `cmd/server/main.go`
- Migration CLI: `cmd/migrate/main.go`
- Frontend root: `web/src/main.tsx`

**Configuration:**
- `go.mod` / `go.sum` (Go 1.26.1 module)
- `web/package.json`, `web/vite.config.ts`, `web/tsconfig.json`
- `Makefile`, `Dockerfile`, `docker-compose.yml`
- `.mcp.json`, `qodana.yaml`

**Core Logic:**
- Services: `internal/core/services/*/service.go`
- Ports: `internal/core/ports/*.go`
- Domain: `internal/core/domain/*/*.go`

**Testing:**
- Backend: `*_test.go` co-located in `internal/adapters/primary/http/` and `internal/adapters/secondary/postgres/`; `internal/core/services/testdata/`
- Frontend: `web/e2e/` (Playwright), `*-tests*` dirs inside `web/src/components/**` and `web/src/lib/__tests__`, `web/src/api/__tests__`

## Naming Conventions

**Backend files:**
- Handlers: `<domain>_handler.go` (e.g., `activity_handler.go`), receiver `(h *XxxHandler)`; constructors `NewXxxHandler(...)`.
- Services: `service.go` within `internal/core/services/<domain>/`, type `Service`, constructor `NewService(...)`.
- Ports: `<domain>_repository.go` defining `type <Domain>Repository interface`.
- Repositories: `internal/adapters/secondary/postgres/<domain>_repository.go`, constructors `New<Domain>Repository(pool)`.
- Domain: `internal/core/domain/<domain>/<domain>.go`; sentinel errors `ErrXxx`; DTOs `<Op><Domain>Request` / `<Domain>Response`.

**Backend symbols:**
- Routes kebab-case: `/time-entries`, `/auth/me`, `/organizations/settings`.
- IDs use `uuid.UUID`; timestamps `time.Time` (timezone-aware).
- Constants in `internal/models/models.go` (Role, Status, GovernanceModel).

**Frontend files:**
- Routes: TanStack file-based; folder routes use `index.tsx`; dynamic segments `$id`; route groups `_authenticated`, `_auth`; feature subfolders use `-components`, `-context`, `-components/dialogs`.
- Components: primarily kebab-case (`app-shell.tsx`, `status-badge.tsx`).
- API modules: kebab-case per domain (`auth.ts`, `time-entries.ts`), exporting a `<Domain>Apis` object of `queryOptions`/`mutationOptions`.
- Path alias: `@/` → `web/src/` (used in imports like `@/api/auth.ts`, `@/lib/api.ts`).

## Where to Add New Code

**New backend domain feature:**
1. Define entity/DTOs + sentinel errors in `internal/core/domain/<domain>/<domain>.go`.
2. Define repository port in `internal/core/ports/<domain>_repository.go`.
3. Implement business logic in `internal/core/services/<domain>/service.go` (constructor `NewService(...)`).
4. Implement repository in `internal/adapters/secondary/postgres/<domain>_repository.go` (`New<Domain>Repository(pool)`).
5. Add a thin handler in `internal/adapters/primary/http/<domain>_handler.go`.
6. Wire repos → service → handler and register routes in `cmd/server/main.go` (use `middleware.Auth(authService, handler)` for protected routes).
7. Add SQL migration in `migrations/` (`NNN_<name>.up.sql` + `.down.sql`).

**New HTTP route on existing domain:**
- Add the handler method in the existing `internal/adapters/primary/http/<domain>_handler.go` and register it in `cmd/server/main.go`. Keep handler thin.

**New frontend page:**
- Add a route file under `web/src/routes/_authenticated/<feature>/index.tsx` (or `$id/index.tsx` for detail). Use `createFileRoute`, `loader` with `client.ensureQueryData(<Domain>Apis.xxxQueryOpts)`, and `errorComponent: RouteError`.
- Add API options in `web/src/api/<feature>.ts` following `web/src/api/auth.ts` shape (`queryOptions`/`mutationOptions` calling `api<T>(...)`).
- Shared UI goes in `web/src/components/<area>/`; reuse `web/src/components/ui/` (shadcn).

**New shared util / hook:**
- `web/src/lib/` for utilities (`utils.ts`, `list-filters.ts`); `web/src/hooks/` for React hooks.

**New cross-cutting middleware:**
- `internal/middleware/` (e.g., `version.go`); compose it in the chain at `cmd/server/main.go:356`.

## Special Directories

**`web/src/routeTree.gen.ts`**
- Purpose: Auto-generated TanStack Router route tree.
- Generated: Yes (regenerate via TanStack CLI; do not hand-edit).
- Committed: Yes.

**`migrations/`**
- Purpose: Immutable SQL schema history.
- Generated: No (hand-written).
- Committed: Yes (applied by `cmd/migrate`).

**`graphify-out/` and `.planning/`**
- Purpose: Knowledge graph and GSD planning artifacts.
- Generated: Yes.
- Committed: `.planning/` typically yes; `graphify-out/` is analysis output.

**`openwiki/` and `plans/` and `wiki/`**
- Purpose: Human/AI documentation and GSD phase plans (`plans/hexagonal-migration.md` defines the architecture target).
- Generated: No.
- Committed: Yes.

**`web/e2e/`**
- Purpose: Playwright end-to-end specs; e2e suites raise `ANONYMOUS_RATE_LIMIT` to run full specs (`cmd/server/main.go:343-352`).
- Generated: No.

---

*Structure analysis: 2026-08-25*
