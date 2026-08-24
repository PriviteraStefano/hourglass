# Codebase Structure

**Analysis Date:** 2026-08-12

## Directory Layout

```
hourglass/
├── cmd/
│   ├── server/main.go          # Composition root: builds services, wires all routes
│   └── migrate/main.go         # PostgreSQL migration CLI (-up / -down)
├── internal/
│   ├── adapters/
│   │   ├── primary/http/       # Thin HTTP handlers (one file per domain + tests)
│   │   └── secondary/postgres/ # PostgreSQL repository implementations + tests
│   ├── auth/                   # JWT validation, token service, bcrypt hasher
│   ├── cookies/                # HttpOnly cookie helpers (auth_token/refresh_token)
│   ├── core/
│   │   ├── domain/<context>/   # Pure entities, request structs, sentinel errors
│   │   ├── ports/              # Repository/service interfaces (hex ports)
│   │   └── services/<context>/ # Business logic services (+ routing, testdata)
│   ├── db/                     # pgxpool construction from DATABASE_URL
│   ├── handlers/               # Legacy glue: health handler
│   ├── middleware/             # Auth, TryAuth, CORS, rate limit, logging, API version
│   └── models/                 # Legacy shared models/constants (models.go)
├── pkg/
│   └── api/                    # {data,error} response envelope + uploads/
├── migrations/                 # NNN_name.up.sql / NNN_name.down.sql (44 migrations)
├── web/                        # Frontend (React 19 + Vite + TanStack)
│   ├── e2e/                    # Playwright specs + helpers.ts
│   └── src/
│       ├── api/                # Per-domain queryOptions/mutationOptions modules
│       ├── components/         # Shared + feature UI components (ui/, layout/, shared/)
│       ├── hooks/              # React hooks (use-mobile.ts)
│       ├── lib/                # api.ts HTTP client, query-client, filters, utils
│       ├── routes/             # TanStack Router file-based routes
│       ├── types/              # Shared API/domain TypeScript types
│       └── main.tsx            # SPA bootstrap (router + QueryClientProvider)
├── deploy/demo/                # Demo deployment topology
├── scripts/                    # seed_demo.sql, helper scripts
├── plans/                      # Architecture/feature plans (hexagonal-migration.md)
├── openwiki/                   # Generated project wiki (architecture/domain/testing/operations)
├── docs/, wiki/, graphify-out/, hourglass-vault/  # Knowledge bases
└── .planning/                  # GSD planning state (codebase/ holds this file)
```

## Directory Purposes

**`cmd/server/`:**
- Purpose: Server binary — the only place concrete types are wired together
- Key files: `main.go` (route table for the whole API; read it first to see the full surface)

**`cmd/migrate/`:**
- Purpose: Migration runner; applies `migrations/*.up.sql` in sorted order (idempotent skip on "already exists")
- Key files: `main.go`

**`internal/core/domain/`:**
- Purpose: Pure domain layer, one package per bounded context
- Contains: `auth/`, `activity/`, `audit/`, `availability/`, `contract/`, `coverage/`, `customer/`, `direction/`, `expense/`, `invitation/`, `organization/`, `orgsettings/`, `password_reset/`, `ticket/`, `time_entry/`, `unit/`, `working_group/`
- Key files: each package's `<context>.go` (entities + `Err*` sentinels + request structs)

**`internal/core/ports/`:**
- Purpose: Hexagonal ports — interfaces the core needs (`<name>_repository.go`, `token_service.go`, `password_hasher.go`, `user_finder.go`, `refresh_token_repo.go`, `errors.go`)
- Key files: `errors.go` (shared `ErrNotFound`/`ErrConflict`/`ErrForeignKey`)

**`internal/core/services/`:**
- Purpose: Application services, one package per domain; `routing/` is the shared approval-routing service; `testdata/` holds mocks + factories for service tests
- Key files: `<context>/<context>.go` per domain

**`internal/adapters/primary/http/`:**
- Purpose: HTTP boundary — handlers, request DTOs, shared validation
- Contains: `<domain>.go` handler + `<domain>_test.go` (integration tests), `validate.go`, `handler_test_helper.go`
- Key files: `validate.go` (string-length caps), `auth.go` (auth endpoints)

**`internal/adapters/secondary/postgres/`:**
- Purpose: PostgreSQL persistence — one `<name>_repository.go` per port, plus schema/migration tests
- Contains: `postgres.go` (pool/scan base), `test_setup.go` (testcontainers pool), `exported_test_helpers.go`, `*_ontology_migration_test.go`, `*_repository_test.go`
- Key files: `time_entry_repository.go`, `activity_repository.go`, `refresh_token_repo.go` (rotation + reuse detection)

**`internal/middleware/`:**
- Purpose: HTTP middleware + context claim helpers
- Key files: `middleware.go` (`Auth`, `TryAuth`, `RequireRole`, claim getters/setters), `ratelimit.go`, `cors.go`, `version.go`

**`internal/auth/`:**
- Purpose: JWT validation (used by middleware) and token/password infrastructure
- Key files: `auth.go`, `token_service.go`, `password_hasher.go`

**`web/src/api/`:**
- Purpose: TanStack Query options per domain — query keys, staleTime, invalidation, toasts
- Contains: `auth.ts`, `activities.ts`, `contracts.ts`, `customers.ts`, `expenses.ts`, `exports.ts`, `time-entries.ts`, `units.ts`, `working-groups.ts` (+ `__tests__/`)
- Note: no `tickets.ts`/`coverage.ts`/`direction.ts`/`availability.ts` — those v0.2 APIs are backend-only

**`web/src/routes/`:**
- Purpose: File-based routing; `_authenticated/` (guarded, `AppShell`), `_auth/` (login/register/invite/password-reset/bootstrap)
- Contains: `__root.tsx`, `_authenticated.tsx`, `_auth.tsx` guards; feature dirs with `index.tsx`/`$id.tsx` plus `-components/` (route-private components) and `-context/` (feature contexts)

**`web/src/types/`:**
- Purpose: Shared TypeScript types mirroring backend JSON
- Key files: `models.ts` (domain models), `api.ts` (envelope + error types), `unit.ts`, `expense-types.ts`, `index.ts`

**`migrations/`:**
- Purpose: SQL schema; `000_full_schema` baseline + numbered deltas through `025_certificate_attachments`
- Contains: `NNN_name.up.sql` / `NNN_name.down.sql` pairs (44 files)

## Key File Locations

**Entry Points:**
- `cmd/server/main.go` — HTTP server + full route table
- `cmd/migrate/main.go` — migration CLI
- `web/src/main.tsx` — SPA bootstrap
- `web/e2e/*.spec.ts` — Playwright E2E
- `scripts/seed_demo.sql` — demo data (`make seed`)

**Configuration:**
- `go.mod` / `go.sum` — Go modules (Go 1.26.1)
- `web/package.json` + `bun.lock` (root) / `web/bun.lock` — frontend deps
- `web/vite.config.ts` — `@` alias → `./src`, `/api` proxy → `:8080`, TanStack router plugin
- `web/src/lib/query-client.ts` — React Query defaults
- `docker-compose.yml`, `Dockerfile`, `deploy/demo/` — containerized deployment
- `Makefile` — build/test/seed targets

**Core Logic:**
- `internal/core/services/routing/routing.go` — shared approval routing (ADR-BE-014)
- `internal/core/services/time_entry/time_entry.go` — approval workflow orchestrator
- `internal/core/services/coverage/coverage.go` — allocation loop
- `internal/core/services/direction/direction.go`, `availability/availability.go` — v0.2 planes
- `internal/adapters/primary/http/time_entry.go` — reference handler (thin + service delegation)

**Testing:**
- `internal/adapters/secondary/postgres/test_setup.go` — testcontainers per-package pool
- `internal/core/services/testdata/` — mocks + factories for service tests
- `internal/adapters/primary/http/handler_test_helper.go` — handler integration harness
- `web/src/**/__tests__/` — Vitest unit tests (co-located)
- `web/e2e/` — Playwright specs + `helpers.ts`

## Naming Conventions

**Files:**
- Go: snake_case per package; repositories `<name>_repository.go`, handlers `<name>_handler.go` (hex) or `<name>.go` (older handlers like `customer.go`, `expense.go`), domains `<context>.go`, tests `<file>_test.go` co-located
- Migrations: `NNN_snake_name.up.sql` / `.down.sql`
- Frontend: kebab-case — `time-entries/index.tsx`, `status-badge.tsx`, `app-shell.tsx`, `query-client.ts`
- Frontend route dirs: `-components/` and `-context/` prefix with hyphen (non-route dirs in TanStack), feature routes use `index.tsx`; dynamic params as `$id.tsx`

**Go:**
- Handlers: `*Handler` receiver type (e.g., `(h *CustomerHandler)`)
- Services: `*Service` with `NewService(...)` constructors; unexported `managerResolution`-style helpers
- Repositories: `*Repository` with `New<X>Repository(pool)`; constructors take `*pgxpool.Pool`
- Packages: short lowercase names; aliased in `cmd/server/main.go` as `<domain>svc` (e.g., `tesvc`, `wgsvc`, `coveragesvc`)
- IDs: `uuid.UUID` everywhere; `time.Time` for timestamps

**Frontend:**
- API modules: `<Domain>Apis` namespace object; `queryOptions`/`mutationOptions` named `<name>QueryOpts` / `<name>MutationOpts`; query keys as `["resource", id, ...]` tuples
- Route files: `createFileRoute("/path")` with `validateSearch` (zod), `loader`, `errorComponent: RouteError`, `pendingMs`
- Components: PascalCase filenames in `components/ui/` (shadcn), kebab-case elsewhere; hooks `use-*.ts`

## Where to Add New Code

**New backend domain (full hexagonal slice):**
1. Domain: `internal/core/domain/<name>/<name>.go` — entities, request structs, `Err*` sentinels
2. Port: `internal/core/ports/<name>_repository.go` — interface
3. Service: `internal/core/services/<name>/<name>.go` — `Service` + `NewService(...)` (import the ports, not concrete repos)
4. HTTP handler: `internal/adapters/primary/http/<name>_handler.go` (or `<name>.go` to match existing handler style)
5. Postgres repo: `internal/adapters/secondary/postgres/<name>_repository.go`
6. Wire everything + register routes in `cmd/server/main.go` (follow the shared-service rules: reuse `routing`/`orgsettings` instances, never construct second copies)
7. Migration: `migrations/NNN_<name>.up.sql` + `.down.sql` (next number after 025)
8. Tests: service mocks in `internal/core/services/testdata/`, repo test in postgres package, handler test in http package

**New frontend feature:**
- Route: `web/src/routes/_authenticated/<feature>/index.tsx` (+ `$id.tsx` for detail pages) with route-private components in `<feature>/-components/`
- API module: `web/src/api/<feature>.ts` exporting `<Feature>Apis` (query/mutation options using `api<T>()` from `web/src/lib/api.ts`)
- Types: extend `web/src/types/models.ts`
- Shared UI: shadcn components in `web/src/components/ui/`; app-level layout in `web/src/components/layout/`

**New route (no new domain):**
- Register in `cmd/server/main.go` with `mux.HandleFunc("METHOD /path", middleware.Auth(authService, handler.Method))`; remember ServeMux most-specific-wins for literal vs wildcard paths

## Special Directories

**`web/src/routes/_authenticated/<feature>/-components/`** and **`-context/`:**
- Purpose: Route-private components and React contexts; hyphen prefix keeps TanStack Router from treating them as routes
- Generated: No
- Committed: Yes

**`web/src/routeTree.gen.ts`:**
- Purpose: Generated route tree from file-based routes
- Generated: Yes (Vite `tanstackRouter` plugin on `bun run dev`/`build`)
- Committed: Yes

**`web/dist/`:** Build output — generated, not committed (present locally).

**`migrations/`:** SQL migration pairs — authoritative schema, committed.

**`uploads/` + `pkg/api/uploads/receipts/`:** Receipt upload storage (`POST /expenses/{id}/receipt`); local filesystem-backed.

**`.gsd-worktrees/M001/`:** Stale worktree snapshot containing an older duplicate of the repo — do not add code here; planned for removal.

**`openwiki/`, `wiki/`, `graphify-out/`, `hourglass-vault/`:** Generated/curated knowledge bases — documentation only; `openwiki/` regenerates from source docs.

**`internal/handlers/`:** Legacy handler home (currently only `health_handler.go`) — new handlers go in `internal/adapters/primary/http/`.

---

*Structure analysis: 2026-08-12*
