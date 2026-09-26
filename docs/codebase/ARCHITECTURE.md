<!-- refreshed: 2026-08-25 -->
# Architecture

**Analysis Date:** 2026-08-25

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (web/)                          │
│   React 19 + TanStack Router v1 + TanStack React Query v5     │
│   `web/src/routes/` (file-based) · `web/src/api/` · `web/src/lib/api.ts` │
└───────────────────────────┬─────────────────────────────────┘
                             │  fetch with credentials: 'include'
                             │  auto-retry POST /auth/refresh on 401
                             ▼
┌─────────────────────────────────────────────────────────────┐
│              Go HTTP Server (cmd/server/main.go)               │
│   net/http ServeMux, Go 1.22+ pattern routing                  │
│   Middleware chain: TryAuth → RateLimit → Logging →           │
│                    APIVersion → CORS → mux                     │
└───────┬──────────────────────────────────────┬───────────────┘
        │                                       │
        ▼                                       ▼
┌───────────────────────────┐     ┌──────────────────────────────────────┐
│ Primary HTTP Adapters      │     │  Auth middleware `internal/middleware` │
│ `internal/adapters/primary/http/` │  (JWT validation, role checks)        │
│ Thin handlers: parse → svc │     └──────────────────────────────────────┘
└───────────┬───────────────┘
            │ call
            ▼
┌─────────────────────────────────────────────────────────────┐
│              Application Core (hexagonal)                      │
│  `internal/core/services/*`  — business logic                  │
│  `internal/core/ports/*`      — repository/token interfaces    │
│  `internal/core/domain/*`     — entities, DTOs, errors         │
└───────────┬───────────────────────────────────────┬───────────┘
            │ implements                          │ depends on
            ▼                                     ▼
┌─────────────────────────────────┐   ┌──────────────────────────────────┐
│ Secondary Postgres Adapters       │   │ `internal/auth` (JWT, bcrypt)      │
│ `internal/adapters/secondary/postgres/` │ `internal/db` (pgx pool)          │
└─────────────────┬───────────────┘   └──────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────┐
│              PostgreSQL (migrations/*.up.sql)                  │
│  `DATABASE_URL` connection, 19 migrations applied by cmd/migrate │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Server entry / wiring | Instantiates repos, services, handlers; registers routes; builds middleware chain | `cmd/server/main.go` |
| Migration CLI | Applies/seeds SQL migrations against PostgreSQL | `cmd/migrate/main.go` |
| Auth middleware | Validates `auth_token` cookie, injects userID/orgID/role/email into context; role gating | `internal/middleware/middleware.go` |
| JWT / password | Token signing/validation, bcrypt hashing | `internal/auth/auth.go`, `internal/auth/token_service.go`, `internal/auth/password_hasher.go` |
| DB pool | PostgreSQL connection pool (`pgx`), close | `internal/db/db.go` |
| API envelope | JSON `{data}` / `{error}` response writers | `pkg/api/response.go` |
| Primary handlers | Thin HTTP adapters translating requests to service calls | `internal/adapters/primary/http/*.go` |
| Application services | Business logic per domain (time-entry, activity, coverage, direction, etc.) | `internal/core/services/*/service.go` |
| Ports (interfaces) | Repository & token/hasher contracts | `internal/core/ports/*.go` |
| Domain | Entities, request/response DTOs, sentinel errors | `internal/core/domain/*/*.go` |
| Secondary repos | PostgreSQL implementations of ports (SQL via `pgx`) | `internal/adapters/secondary/postgres/*.go` |
| Models/constants | Cross-cutting constants (Role, Status, Governance) | `internal/models/models.go` |

## Pattern Overview

**Overall:** Hexagonal architecture (ports & adapters), per `AGENTS.md` and `plans/hexagonal-migration.md`.

**Key Characteristics:**
- Business logic lives in `internal/core/services/`, never in handlers.
- Dependencies are inverted through interfaces in `internal/core/ports/` (e.g., `ports.ActivityRepository`) so services are agnostic of PostgreSQL.
- Handlers are thin: parse JSON → call service → map sentinel errors to HTTP status via `pkg/api` envelope.
- Wiring (composition root) is centralized in `cmd/server/main.go`.
- Shared services passed by reference to avoid duplicate instances (notably `routingSvc` — `routing.NewService`, passed to time-entry, activity, ticket, coverage, direction services to keep routing parity; see comments at `cmd/server/main.go:131-186`).

## Layers

**Primary HTTP Adapters (`internal/adapters/primary/http/`):**
- Purpose: Translate HTTP into service calls; request validation/parsing only.
- Location: `internal/adapters/primary/http/`
- Contains: Per-domain handler files (`activity_handler.go`, `time_entry.go`, `coverage_handler.go`, etc.); `_test.go` co-located.
- Depends on: `internal/core/services/*`, `internal/core/ports`, `internal/models`, `pkg/api`, `internal/middleware`.
- Used by: `cmd/server/main.go` (route registration).

**Application Core (`internal/core/`):**
- Purpose: All business logic and domain rules.
- Location: `internal/core/services/`, `internal/core/ports/`, `internal/core/domain/`
- Contains: 18 service packages, 23 port interfaces, 17 domain packages.
- Depends on: its own ports (abstractions), `internal/models`, `internal/auth` (token/hasher ports).
- Used by: primary adapters.

**Secondary Postgres Adapters (`internal/adapters/secondary/postgres/`):**
- Purpose: Implement repository ports against PostgreSQL using `pgx`.
- Location: `internal/adapters/secondary/postgres/`
- Contains: 55 Go files (repositories + tests); `postgres.go` for shared helpers; `test_setup.go` for test DB.
- Depends on: `internal/core/ports` (implements), `internal/db` (pool), `internal/core/domain`.
- Used by: composition root in `cmd/server/main.go`.

**Frontend App Layer (`web/src/`):**
- Purpose: SPA consuming the JSON API via TanStack Router + React Query.
- Location: `web/src/`
- Contains: `routes/` (file-based routing), `api/` (query/mutation options), `lib/` (HTTP client, query client), `components/`, `hooks/`, `types/`.
- Depends on: backend `/api` JSON contract.
- Used by: browser.

## Data Flow

### Primary Request Path (authenticated GET/POST)

1. Request hits server, outer middleware applies in order (`cmd/server/main.go:356`):
   `TryAuth` → `RateLimiter.Middleware` → `Logging` → `APIVersion` → `CORS` → `ServeMux`.
2. Route matches e.g. `POST /time-entries`, wrapped by `middleware.Auth(authService, handler)` (`cmd/server/main.go:309`), which validates the `auth_token` cookie and injects claims into `context` (`internal/middleware/middleware.go:23`).
3. Handler parses JSON, calls the service (`internal/adapters/primary/http/time_entry.go`), which executes business rules and calls a repository port.
4. Repository port implemented by `internal/adapters/secondary/postgres` executes SQL on the `pgx` pool (`internal/db/db.go`).
5. Response serialized through `pkg/api.RespondWithJSON` as `{"data": ...}` (`pkg/api/response.go:14`).

### Auth / Login Flow

1. `POST /auth/login` (rate-limited, no `Auth` wrapper) → `internal/adapters/primary/http/auth.go` → `authsvc.Service` → validates password via `auth.NewPasswordHasher`, issues JWTs via `auth.NewTokenService`, persists refresh token via `postgres.NewRefreshTokenRepository`.
2. JWTs set as HttpOnly `auth_token` / `refresh_token` cookies (`internal/cookies/cookies.go`).
3. Frontend `web/src/api/auth.ts` `loginMutationOpts` caches profile; `web/src/routes/_authenticated.tsx` hydrates `GET /auth/me` in `beforeLoad`.

### Frontend 401 Refresh Flow

1. `web/src/lib/api.ts` `api<T>()` sends `credentials: 'include'`; on `401` (and not an auth path) it calls `POST /auth/refresh` once (deduped via `refreshPromise`) then retries the original request (`web/src/lib/api.ts:46-80`).
2. If refresh fails → throws `UnauthorizedError`; route guards redirect to `/login` (`web/src/routes/_authenticated.tsx:12-17`).

### Approval Workflow (time entry / expense)

Entries transition `draft → submitted → pending_manager → pending_finance → approved/rejected`, tracked immutably in `*_approvals` tables; `currentApproverRole` drives routing. Services in `internal/core/services/time_entry/` and `internal/core/services/expense/` orchestrate these transitions.

## Key Abstractions

**Repository Port (`internal/core/ports`):**
- Purpose: Persistence boundary decoupled from PostgreSQL. Example: `ports.ActivityRepository` (`internal/core/ports/activity_repository.go`) defines CRUD plus derived reads (`ResolveCommercialContext`, `ResolveBillability`).
- Pattern: interface consumed by services, implemented in `internal/adapters/secondary/postgres/`.

**Service (`internal/core/services/*/service.go`):**
- Purpose: Use-case orchestration. Constructor takes the repos/ports it needs. Example: `activitysvc.NewService(...)` (`cmd/server/main.go:154`).

**Domain Entity / DTO (`internal/core/domain/*`):**
- Purpose: Define entities (e.g., `activity.Activity` in `internal/core/domain/activity/activity.go`), request/response DTOs (`CreateActivityRequest`), and sentinel errors (`ErrActivityNotFound`).

**Shared Routing Service (`internal/core/services/routing`):**
- Purpose: Single source of manager-stage resolution (`routing.NewService`, `cmd/server/main.go:136`) consumed by entry/activity/ticket/coverage/direction services to prevent routing drift.

## Entry Points

**Backend HTTP server:**
- Location: `cmd/server/main.go`
- Triggers: `go run ./cmd/server` (default port `:8080`, overridable via `PORT`).
- Responsibilities: build pool, construct all services/handlers, register routes, start listener (`cmd/server/main.go:35-360`).

**Backend migration CLI:**
- Location: `cmd/migrate/main.go`
- Triggers: `go run ./cmd/migrate -up -dir migrations` (or `-all` to apply all + seed).

**Frontend SPA:**
- Location: `web/src/main.tsx`
- Triggers: `bun run dev` (Vite, port `:3000`, proxies `/api` → `http://localhost:8080` per `web/vite.config.ts:21-25`).
- Responsibilities: mount `QueryClientProvider` + `RouterProvider` with `queryClient` in router context as `client` (`web/src/main.tsx:8-25`).

## Architectural Constraints

- **Threading:** Go's single-threaded-per-request goroutine model via `net/http`; `pgx` pool handles concurrent DB access. No explicit worker pools.
- **Global state:** Composition root in `cmd/server/main.go` is the only place wiring happens; services/repos are constructed once and shared. `refreshPromise` in `web/src/lib/api.ts:34` is a module-level singleton for refresh de-duplication.
- **Auth context propagation:** user/org/role/email passed via `context.Context` keys (`internal/middleware/middleware.go:14-21`), not globals. Extract via `GetUserID(ctx)` etc.
- **Circular imports:** Avoided by keeping ports in `internal/core/ports` depending only on `domain` and stdlib; services import ports + domain; adapters import ports + domain. No upward dependencies.
- **Middleware order is significant:** `TryAuth` is outermost so public routes still get context when a valid cookie is present; `Auth` (hard reject) is applied per-route inside the mux (`cmd/server/main.go:96-327`).
- **ServeMux route precedence:** most-specific pattern wins; literal `GET /organizations/settings` coexists with wildcard `GET /organizations/{id}/settings` (`cmd/server/main.go:236-241`).
- **Frontend coupling:** Routes depend on `api/*Apis` option objects and the `client` (queryClient) from router context — not raw fetch.

## Anti-Patterns

### Business logic in handlers
**What happens:** Handler does validation, joins, or status transitions inline.
**Why it's wrong:** Breaks hexagonal rule; logic can't be unit-tested without HTTP; duplicates across adapters.
**Do this instead:** Keep handlers thin (parse → call service → map error). See `internal/adapters/primary/http/activity_handler.go:17-34` for the documented thin-handler contract.

### Navigation inside the HTTP client
**What happens:** `web/src/lib/api.ts` (older version) called `window.location.href` on 401, fighting the router.
**Why it's wrong:** Infinite redirect loops with TanStack Router.
**Do this instead:** Throw `UnauthorizedError` and let route guards (`beforeLoad`) redirect (`web/src/lib/api.ts:11-16`, `web/src/routes/_authenticated.tsx:12-17`).

### Duplicate service instances for shared logic
**What happens:** Constructing a second `routing.Service` per consumer.
**Why it's wrong:** Entry and proposal routing can drift (violates ADR-BE-014).
**Do this instead:** Build the shared service once in `cmd/server/main.go` (e.g., `routingSvc`, `orgSettingsService`) and inject the same instance (`cmd/server/main.go:136,178,186`).

## Error Handling

**Strategy:** Sentinel errors in domain packages; ports/services return them; primary adapters map to HTTP status via `pkg/api.RespondWithError` (`pkg/api/response.go:25`).

**Patterns:**
- Domain sentinel errors (e.g., `ErrActivityNotFound`, `ErrForbidden` in `internal/core/domain/activity/activity.go:11-36`).
- Handlers translate `errors.Is`/type checks into 4xx/5xx with a JSON `{error}` body.
- Frontend throws `Error(error.message || error.error)` from `api<T>()` (`web/src/lib/api.ts:82-87`); React Query surfaces via `errorComponent: RouteError`.

## Cross-Cutting Concerns

**Logging:** `middleware.Logging` logs method/path/status/ms for every request (`internal/middleware/middleware.go:145-154`). Backend uses `log` package; frontend uses console via React Query devtools.
**Validation:** Backend validation in services (e.g., origin immutability guard `internal/core/domain/activity/activity.go:32-35`); frontend uses Zod schemas in `loader`/`validateSearch` (e.g., `web/src/routes/_authenticated/time-entries/index.tsx:10-17`).
**Authentication:** JWT in HttpOnly cookies; `middleware.Auth` hard-gate and `middleware.TryAuth` soft-gate (`internal/middleware/middleware.go:23,90`); role checks via `RequireRole`.
**API versioning:** `middleware.APIVersion` wraps the mux (`cmd/server/main.go:356`) — version header/behavior in `internal/middleware/version.go`.

---

*Architecture analysis: 2026-08-25*
