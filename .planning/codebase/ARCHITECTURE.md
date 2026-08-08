<!-- refreshed: 2026-08-08 -->
# Architecture

**Analysis Date:** 2026-08-08

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                     Frontend — React SPA (web/src/)                     │
│   TanStack Router file-based routes → XxxApis query/mutation options     │
│   web/src/lib/api.ts (fetch + 401→refresh) — React Query client          │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │ HTTP /api/* (Vite dev proxy → :8080)
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│          Primary HTTP Adapters — internal/adapters/primary/http/         │
│   Thin handlers (parse → delegate → format { data | error } envelope)    │
│   Middleware chain: CORS → APIVersion → Logging → RateLimiter →          │
│                     TryAuth/Auth (JWT claims into request context)       │
└───────────────┬───────────────────────────────┬─────────────────────────┘
                │ calls                          │ implements (via DI in
                ▼                                │  cmd/server/main.go)
┌─────────────────────────────────────────────────────────────────────────┐
│              Core (hexagonal inner) — internal/core/                     │
│   domain/   — pure models + behavior, one dir per bounded context        │
│   ports/    — repository/service interfaces the core requires            │
│   services/ — application logic; routing/ shared approval resolution     │
└───────────────┬─────────────────────────────────────────────────────────┘
                │ SQL via pgx/v5 pool
                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│           Secondary Adapters — internal/adapters/secondary/postgres/     │
│   Repository implementations (implements ports.*), pgxpool, SQL          │
│                                  │                                       │
│                                  ▼                                       │
│                       PostgreSQL (migrations/*.sql)                      │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Server composition root | Reads env, builds pool, instantiates repos → services → handlers, wires routes + middleware | `cmd/server/main.go` |
| Migration CLI | Applies `migrations/*.up.sql` / `*.down.sql` in order | `cmd/migrate/main.go` |
| Domain models | Pure structs + behavior methods (`CanEdit()`, `IsOwner()`); sentinel errors per context | `internal/core/domain/<context>/*.go` |
| Ports | Interfaces the core needs from the outside (repos, token service, hasher, finder) | `internal/core/ports/*.go` |
| Services | Business logic: CRUD gates, approval workflow, validation against domain rules | `internal/core/services/<context>/*.go` |
| Routing service | Resolves the manager approval stage (WG manager/delegates or unit-tree walk) — shared by entries, proposals, coverage | `internal/core/services/routing/routing.go` |
| HTTP handlers | Parse request, extract auth context, call service, write envelope | `internal/adapters/primary/http/<resource>.go` |
| Postgres repositories | SQL implementations of ports; pgx/v5 pool | `internal/adapters/secondary/postgres/*.go` |
| JWT/auth | Token generation/validation (HS256), bcrypt hashing, refresh-token service | `internal/auth/auth.go`, `internal/auth/token_service.go`, `internal/auth/password_hasher.go` |
| Middleware | Auth/TryAuth/RequireRole, CORS, rate limit, logging, API version header | `internal/middleware/*.go` |
| Shared legacy models | Enums + DTOs still imported by some migrated contexts (see Anti-Patterns) | `internal/models/models.go` |
| Response envelope | `{ "data": ... }` / `{ "error": ... }` JSON contract | `pkg/api/response.go` |
| Frontend HTTP client | `api<T>()` fetch wrapper: credentials, envelope unwrap, single-flight 401 refresh | `web/src/lib/api.ts` |
| Frontend API modules | `XxxApis` objects of `queryOptions`/`mutationOptions` consumed by routes | `web/src/api/<resource>.ts` |

## Pattern Overview

**Overall:** Hexagonal architecture (ports & adapters) on the backend; TanStack Router + React Query data-first composition on the frontend. The backend migration is mid-flight: `internal/core/domain|ports|services` is the target structure (see `plans/hexagonal-migration.md`), with several bounded contexts still referencing `internal/models`.

**Key Characteristics:**
- Manual dependency injection in a single composition root (`cmd/server/main.go`) — repos built from `*pgxpool.Pool`, passed into service constructors, then into handler constructors
- Services depend only on `ports.*` interfaces, never on concrete adapters (except the shared `routing.Service`, injected concretely)
- Handlers are thin: no business rules; they read `middleware.GetUserID/GetOrganizationID/GetRole` from context
- Go 1.22+ stdlib `http.ServeMux` with method+pattern routes (`mux.HandleFunc("GET /time-entries", ...)`); no router framework
- Every route registered per-handler-method; auth-gated via `middleware.Auth(...)` wrapping, role gates inside services
- Frontend server state lives entirely in React Query; route guards hydrate auth before render
- Numbered SQL migrations (`000`–`020+`), each with `.up.sql`/`.down.sql`, applied by `cmd/migrate`

## Layers

**Composition / Entry (`cmd/`):**
- Purpose: Process entry points and all wiring
- Location: `cmd/server/main.go`, `cmd/migrate/main.go`
- Contains: `main()`, env parsing, pool creation, route registration, middleware stack
- Depends on: `internal/adapters/*`, `internal/core/services/*`, `internal/auth`, `internal/db`, `internal/middleware`, `internal/handlers`
- Used by: runtime only

**Domain (`internal/core/domain/`):**
- Purpose: Pure business models, no external deps beyond `google/uuid`
- Location: `internal/core/domain/<bounded-context>/` (14 contexts: `auth`, `organization`, `unit`, `working_group`, `activity`, `contract`, `customer`, `time_entry`, `expense`, `invitation`, `password_reset`, `ticket`, `coverage`, `audit`)
- Contains: structs with JSON tags, status/role constants, behavior methods (e.g. `time_entry.CanEdit()`), sentinel errors (e.g. `ErrPeriodLocked`)
- Depends on: `github.com/google/uuid`, stdlib
- Used by: `internal/core/services/*`, `internal/core/ports/*`

**Ports (`internal/core/ports/`):**
- Purpose: Interfaces the core requires from the outside world
- Location: `internal/core/ports/*.go` (21 files: `*_repository.go`, `token_service.go`, `password_hasher.go`, `user_finder.go`, `errors.go`)
- Contains: repository interfaces, `ListFilters` struct, shared error sentinels (`ErrNotFound`, `ErrConflict`)
- Depends on: domain packages
- Used by: services (consume), postgres adapters (implement)

**Services (`internal/core/services/`):**
- Purpose: Application business logic — all validation, workflow state transitions, authorization
- Location: `internal/core/services/<context>/*.go` (15 packages + `routing/` + `testdata/`)
- Contains: `type Service struct{ repo ports.XRepository; ... }`, `NewService(...)`, methods returning domain types
- Depends on: `internal/core/ports`, `internal/core/domain`, other services (`routing`); **not** adapters
- Used by: HTTP handlers (primary adapters), integration tests

**Primary HTTP Adapters (`internal/adapters/primary/http/`):**
- Purpose: Translate HTTP ↔ service calls
- Location: `internal/adapters/primary/http/<resource>.go` (+ `validate.go` request validation helpers)
- Contains: `XxxHandler` structs, request DTOs with `json:` tags, `func (h *XxxHandler) List/Create/Get/Update/Delete/...`
- Depends on: services, middleware (context getters), `pkg/api` (envelope), domain errors for `errors.Is` mapping
- Used by: `cmd/server/main.go`

**Secondary PostgreSQL Adapters (`internal/adapters/secondary/postgres/`):**
- Purpose: Persistence for every port interface
- Location: `internal/adapters/secondary/postgres/*.go`
- Contains: `NewXxxRepository(pool)` constructors, SQL statements, `pgxpool` queries
- Depends on: `internal/core/ports`, `internal/core/domain` (or `internal/models` — transitional)
- Used by: `cmd/server/main.go` (DI)

**Support packages (non-hexagonal glue):**
- `internal/auth/` — legacy JWT/bcrypt service consumed by middleware and `auth.NewTokenService`
- `internal/middleware/` — `Auth`, `TryAuth`, `RequireRole`, `CORS`, `NewRateLimiter`, `Logging`, `APIVersion`
- `internal/cookies/` — cookie construction for `auth_token` / `refresh_token`
- `internal/db/` — `pgxpool` pool creation + legacy `sql.DB` helper
- `internal/handlers/` — health handler (only remaining legacy handler)
- `internal/models/` — shared legacy models/enums (being migrated into `core/domain`)

## Data Flow

### Primary Request Path (e.g., create a time entry)

1. Browser → Vite dev server proxies `/api/*` → `http://localhost:8080` (`web/vite.config.ts:21-27`); production serves the SPA and API from the same origin
2. Middleware chain runs outside-in: `CORS` → `APIVersion` → `Logging` → `RateLimiter` → `TryAuth`; route-level `middleware.Auth` validates the JWT and injects `UserID`/`OrganizationID`/`Role`/`Email` context keys (`internal/middleware/middleware.go:23-44`)
3. ServeMux matches `POST /time-entries` (`cmd/server/main.go:267`) → `TimeEntryHandler.Create` (`internal/adapters/primary/http/time_entry.go:121`)
4. Handler decodes JSON body, pulls claims from context, calls `teService.Create(ctx, req)` (`internal/core/services/time_entry/time_entry.go:43`)
5. Service enforces domain rules: `IsPeriodLocked` check, parses date, builds `time_entry.TimeEntry` with `StatusDraft`, calls `s.repo.Create` (port)
6. `postgres.TimeEntryRepository.Create` executes INSERT against the pool (`internal/adapters/secondary/postgres/time_entry_repository.go`)
7. Handler writes `{ "data": {...} }` via `api.RespondWithJSON` (`pkg/api/response.go:14-23`)

### Authentication Flow

1. `POST /auth/login` (`cmd/server/main.go:91`, rate-limited 5/min) → `AuthHandler.Login` (`internal/adapters/primary/http/auth.go:89`)
2. Hexagonal `authsvc.Service` validates credentials via `password_hasher`, issues access + refresh tokens, stores refresh token hash (reuse detection, migration `010`)
3. Cookies set via `internal/cookies/cookies.go`; frontend `web/src/lib/api.ts` sends `credentials: "include"` and on 401 performs a single-flight `POST /auth/refresh` (guarded against auth-path recursion), then retries the original request once
4. Route guards hydrate auth: `web/src/routes/_authenticated.tsx:8-18` calls `client.fetchQuery(AuthApis.profileQueryOpts)` in `beforeLoad`; failure → `redirect({ to: "/login" })`. `_auth.tsx` does the inverse (redirect to `/` if profile succeeds)

### Approval Workflow (time entries & expenses)

1. `POST /time-entries/{id}/submit` → `TimeEntryHandler.Submit` (`time_entry.go:305`) → `tesvc.Service.Submit` (`time_entry.go:129`)
2. Service calls `routing.Service.ResolveManagerStage` (`internal/core/services/routing/routing.go:57`): R-1 chain (activity's anchored WG → manager + delegates), D-11 skip (owner is in approver set → straight to `pending_finance`), R-2 fallback (personal activity → unit manager walk up the tree)
3. Entry transitions `draft → pending_manager → pending_finance → approved/rejected`; `currentApproverRole` tracks the stage; each action appends an immutable row to `*_approvals` tables and `audit_logs` (migrations `006`, `017`)
4. Coverage allocation writers (`internal/core/services/coverage/coverage.go`) consume the **same shared** `routing.Service` so entry and allocation routing cannot drift (ADR-BE-014 D-G/D-08)
5. Frontend: approvals page (`web/src/routes/_authenticated/approvals/`) lists pending entries per role stage; mutation `onSuccess` invalidates `["time-entries"]` queries (`web/src/api/time-entries.ts:42-44`)

### Frontend Data Flow

1. `web/src/main.tsx` creates the router with the shared `queryClient` in context (`web/src/lib/query-client.ts`: `retry: false`, `staleTime: 30000`)
2. Route `beforeLoad`/loaders pre-fetch via `client.ensureQueryData/fetchQuery(queryOpts)`; page components consume the same `queryOpts` through `useSuspenseQuery`
3. Writes go through `mutationOptions` (`web/src/api/*.ts`), which invalidate related query keys in `onSuccess` and surface `sonner` toasts

## Key Abstractions

**Port Interfaces:**
- Purpose: Inversion-of-control boundary between services and persistence
- Examples: `ports.TimeEntryRepository` (`internal/core/ports/time_entry_repository.go:10-18`), `ports.ActivityRepository`, `ports.TokenService`
- Pattern: Interface defined by consumer (core), implemented by secondary adapter

**Domain Aggregates with Behavior:**
- Purpose: Encapsulate state transitions so services cannot corrupt workflow invariants
- Examples: `time_entry.TimeEntry` with `CanEdit()`, `IsOwner()` (`internal/core/domain/time_entry/time_entry.go`); status constants `draft → submitted → pending_manager → pending_finance → approved/rejected`
- Pattern: Sentinel errors (`ErrEntryNotDraft`, `ErrPeriodLocked`, ...) returned from domain methods, mapped to HTTP statuses in handlers

**routing.Service:**
- Purpose: Single shared resolver for the manager approval stage (ADR-BE-014)
- Examples: injected into `tesvc.NewService`, `activitysvc.NewService` (proposal approval), `coveragesvc.NewService` (`cmd/server/main.go:134-162`)
- Pattern: Concrete cross-service dependency (not a port) — deliberate single instance (D-G parity)

**Frontend Query Options (frontend):**
- Purpose: Declarative, reusable server-state definitions shared by loaders and components
- Examples: `TimeEntriesApis` (`web/src/api/time-entries.ts:95-105`), `AuthApis.profileQueryOpts` consumed in `_authenticated.tsx`
- Pattern: `queryOptions`/`mutationOptions` factories + namespace export object; `queryKey` helpers as `as const` tuples

**api<T>() HTTP Client (frontend):**
- Purpose: Single place for cookies, envelope unwrapping, and 401-refresh semantics
- Location: `web/src/lib/api.ts:36-95`
- Pattern: Throws `UnauthorizedError` on permanent 401 (route guards redirect); never navigates itself

## Entry Points

**Backend server:**
- Location: `cmd/server/main.go`
- Triggers: `go run ./cmd/server` / Docker / `make run`
- Responsibilities: Env validation (fatal if `JWT_SECRET` missing in production), pool init, full DI graph, route table (~100 routes), middleware stack, `ListenAndServe` on `:8080`

**Migration CLI:**
- Location: `cmd/migrate/main.go`
- Triggers: `go run ./cmd/migrate -up|-down -dir migrations`
- Responsibilities: Apply/rollback numbered `.up.sql`/`.down.sql` files via `database/sql` + `lib/pq`

**Frontend:**
- Location: `web/src/main.tsx`
- Triggers: `bun run dev` / `bun run build` (Vite)
- Responsibilities: Router construction with `routeTree.gen.ts`, QueryClient injection, `RouterProvider` render

## Architectural Constraints

- **Threading:** Single-process Go server, stdlib `net/http` (one goroutine per request); no background workers, queues, or cron jobs. `db.NewPool` defaults: max 25 open / 5 idle conns (`internal/db/db.go`). Frontend: single React Query client, single-flight refresh (`refreshPromise` in `web/src/lib/api.ts:34`).
- **Global state:** `web/src/lib/query-client.ts` exports a module-level `queryClient` singleton (must stay the same instance across `main.tsx` and route context). Module-level `refreshPromise` guards concurrent 401 refreshes.
- **DI discipline:** All object graphs constructed in `cmd/server/main.go`; services receive ports, handlers receive concrete services. No DI framework, no global registries in Go.
- **DB access:** Only `internal/adapters/secondary/postgres/*` touches SQL; services go through `ports.*`. Exception: `activityHandler` holds `activityRepo` directly for derived detail reads (`cmd/server/main.go:147`).
- **Route/auth discipline:** Role gates live in services (D-15/D-11 gates for tickets, coverage), not only in middleware; `RequireRole` middleware exists but route-level role gating is done by services reading context.
- **Migration order:** New migrations must be numbered sequentially after `020` and ship `.up.sql` + `.down.sql` pairs.
- **Append-only streams:** Approval history, ticket comments/history, and audit logs are immutable — no update/delete routes (see comments at `cmd/server/main.go:226-228`).

## Anti-Patterns

### Dual Model Sources (migration in progress)

**What happens:** Several bounded contexts define structs in both `internal/core/domain/<ctx>/` and legacy `internal/models/models.go`; services/adapters for `organization`, `contract`, `activity`, `customer`, `ticket`, `coverage` still import `internal/models` (`grep -l "internal/models" internal/ --include="*.go"` matches those packages).
**Why it's wrong:** Two sources of truth for the same entity drift (JSON tags, validation, enum sets); mixed imports make the hexagonal boundary blurry.
**Do this instead:** Follow the completed contexts (`time_entry`, `unit`, `working_group`, `expense`) as the target pattern: domain struct in `core/domain`, port in `core/ports`, adapter implementing it. Track against `plans/hexagonal-migration.md`.

### Handlers Holding Repositories

**What happens:** `http.NewActivityHandler(activityService, activityRepo)` (`cmd/server/main.go:147`) — the handler reaches past the service for derived reads.
**Why it's wrong:** Breaks the thin-handler rule; handler now depends on both service and adapter, complicating handler unit tests.
**Do this instead:** Add the derived-read method to the service (which already holds the repo), or introduce a read-model port.

### Untyped Filter Struct Field

**What happens:** `ListFilters.OrgID interface{}` (`internal/core/ports/time_entry_repository.go:21`).
**Why it's wrong:** Erases compile-time safety at the core boundary.
**Do this instead:** `OrgID uuid.UUID` (omit where unused).

### Dual Auth Packages

**What happens:** Legacy `internal/auth.Service` (JWT validate for middleware) coexists with hexagonal `internal/core/services/auth` + `ports.TokenService` adapter (`internal/auth/token_service.go`).
**Why it's wrong:** Two paths to the same capability; middleware cannot use the port interface without touching adapters.
**Do this instead:** Migrate middleware to validate via a port-backed service once `internal/auth` is folded into `core` (or accept as documented transitional state — see `plans/hexagonal-migration.md`).

## Error Handling

**Strategy:** Sentinel errors per domain package + `errors.Is` mapping in handlers; single JSON error envelope.

**Patterns:**
- Domain/service errors: package-level `var Err... = errors.New(...)` (`internal/core/domain/time_entry/time_entry.go:10-19`); shared port errors `ports.ErrNotFound`, `ports.ErrConflict` (`internal/core/ports/errors.go`)
- Handler mapping: switch/`errors.Is` → `api.RespondWithError(w, status, msg)`; unexpected errors degrade to 500 with generic message (`internal/adapters/primary/http/time_entry.go:77-80`)
- Validation: `internal/adapters/primary/http/validate.go` for request-shape checks; domain enum validation + DB CHECK constraints for value ranges
- Frontend: `api<T>()` throws `Error(error.message || error.error)` from the envelope (`web/src/lib/api.ts:82-87`); `UnauthorizedError` for unrecoverable auth; `sonner` toasts in mutation options

## Cross-Cutting Concerns

**Logging:** `middleware.Logging` logs `METHOD path status duration` per request (`internal/middleware/middleware.go:145-154`); stdlib `log` elsewhere. Frontend surfaces user feedback via `sonner` toasts, not console.
**Validation:** Request DTOs + `validate.go`; domain-level invariants in services; DB CHECK constraints (statuses, roles); `models`/domain enum constants (`Role`, `EntryStatus`, `GovernanceModel`, `ProjectType`, `ExpenseCategory`, `ApprovalAction`).
**Authentication:** JWT HS256 access token (15 min) + refresh token (7 days) in HttpOnly cookies; refresh-token rotation with reuse detection (migration `010`); bcrypt cost 12; org-switching via `POST /auth/switch-organization`; `TryAuth` for anonymous endpoints; rate limiting on auth routes (5/min login/register, 3/min password-reset, outer 20/min anonymous / 100/min authenticated).

---

*Architecture analysis: 2026-08-08*
