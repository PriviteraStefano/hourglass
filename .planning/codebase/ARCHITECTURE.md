<!-- refreshed: 2026-08-12 -->
# Architecture

**Analysis Date:** 2026-08-12

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend SPA (web/)                           │
│   React 19 · Vite · TanStack Router v1 · TanStack Query v5           │
│   web/src/main.tsx · web/src/routes/ · web/src/api/                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ fetch() credentials:include
                                │ 401 → single-flight POST /auth/refresh
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│              HTTP API (Go stdlib net/http, ServeMux)                │
│   cmd/server/main.go (composition root + ~130 route registrations)  │
│   middleware chain: TryAuth → RateLimiter → Logging → APIVersion →  │
│                     CORS → mux; per-route middleware.Auth           │
├─────────────────────────────────────────────────────────────────────┤
│                    PRIMARY ADAPTER (driving)                         │
│   internal/adapters/primary/http/*.go  — thin handlers              │
│   parse → auth context → delegate → envelope response               │
└───────────────────────────────┬─────────────────────────────────────┘
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      HEXAGONAL CORE                                  │
│   domain  internal/core/domain/<bounded-context>/  (pure entities)  │
│   ports   internal/core/ports/*.go               (interfaces)       │
│   services internal/core/services/<context>/     (business logic,   │
│                                                   depends on ports) │
│   routing internal/core/services/routing/         (SHARED service)  │
└───────────────────────────────┬─────────────────────────────────────┘
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   SECONDARY ADAPTER (driven)                         │
│   internal/adapters/secondary/postgres/*_repository.go              │
│   pgxpool → PostgreSQL (migrations/ NNN_*.up|down.sql)              │
│   internal/auth (JWT + bcrypt) · internal/cookies (HttpOnly)        │
└─────────────────────────────────────────────────────────────────────┘
```

Two feature generations coexist:
- **v0.1 (full-stack):** auth, units, working groups, customers, contracts, activities, time entries, expenses, exports, organizations, invitations, password reset — all have a frontend in `web/src/`.
- **v0.2 (backend-only):** tickets, coverage (allocation), direction (plan plane), availability (absences), orgsettings, contract types — implemented in Go only; no frontend routes consume them yet.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Composition root | Builds every service/handler/repo, registers every route, middleware chain | `cmd/server/main.go` |
| Auth (legacy JWT svc) | Validate/sign JWT for `middleware.Auth` | `internal/auth/auth.go` |
| Token service | Access + refresh token issuance (JWT) | `internal/auth/token_service.go` |
| Password hasher | bcrypt hash/compare | `internal/auth/password_hasher.go` |
| Cookie helpers | HttpOnly `auth_token`/`refresh_token` cookies | `internal/cookies/cookies.go` |
| Auth service | Register/login/logout/refresh/bootstrap/switch-org/memberships | `internal/core/services/auth/auth.go` |
| Time entry service | CRUD, submit, approve, reject, period lock | `internal/core/services/time_entry/time_entry.go` |
| Expense service | CRUD, submit, approve, reject, receipt upload | `internal/core/services/expense/expense.go` |
| Routing service (shared) | Resolves manager approval stage (WG chain / unit-tree walk / role gate) | `internal/core/services/routing/routing.go` |
| Activity service | Recursive activity CRUD, origin axis, ancestry/commercial/billability | `internal/core/services/activity/activity.go` |
| Coverage service | Allocation replace-set, proposals, to-cover queue, bucket balance, period close | `internal/core/services/coverage/coverage.go` |
| Direction service | Plan-plane create/activate/cancel/claim + read models | `internal/core/services/direction/direction.go` |
| Availability service | Window lifecycle, certificate attach, contract-type CRUD, capacity | `internal/core/services/availability/availability.go` |
| Orgsettings service | Org policy key/value store, planning-mode fallback | `internal/core/services/orgsettings/orgsettings.go` |
| Organization service | Org CRUD, invites, member roles, schedule override | `internal/core/services/organization/organization.go` |
| Contract service | CRUD, adopt, mileage recalculation | `internal/core/services/contract/contract.go` |
| Customer / Unit / Working-group / Ticket / Invitation / Password-reset / Export services | CRUD + lifecycle per domain | `internal/core/services/{customer,unit,working_group,ticket,invitation,password_reset,export}/*.go` |
| HTTP handlers (primary) | Parse request, read middleware claims, delegate, respond | `internal/adapters/primary/http/*.go` |
| Postgres repositories (secondary) | Raw SQL via pgxpool, scan into domain types, map `pgx.ErrNoRows` → domain sentinel errors | `internal/adapters/secondary/postgres/*_repository.go` |
| DB pool | `pgxpool.Pool` from `DATABASE_URL` | `internal/db/db.go` |
| Health handler (legacy) | `GET /health` | `internal/handlers/health_handler.go` |
| Response envelope | `{data: ...}` / `{error: ...}` | `pkg/api/response.go` |
| Frontend HTTP client | `api<T>()` fetch wrapper, single-flight refresh, `UnauthorizedError` | `web/src/lib/api.ts` |
| Frontend API modules | `queryOptions`/`mutationOptions` per domain, invalidate + toast | `web/src/api/*.ts` |
| Route guards | Profile hydration + redirect | `web/src/routes/_authenticated.tsx`, `web/src/routes/_auth.tsx` |

## Pattern Overview

**Overall:** Hexagonal (ports & adapters) architecture, per the migration plan in `plans/hexagonal-migration.md`. Domain → ports → services → adapters → composition-root wiring. Only `cmd/server/main.go` knows concrete types; everything below depends on interfaces or concrete ports.

**Key Characteristics:**
- One `internal/core/domain/<context>/` package per bounded context — pure entities, value objects, sentinel errors, zero external dependencies (except `uuid`, `time`, `internal/models`)
- One `ports.<X>Repository` interface per aggregate; implementations in `internal/adapters/secondary/postgres/`
- Services hold only ports and *shared* service singletons (see Constraints)
- Handlers are thin and per-domain; route registration lives entirely in `cmd/server/main.go`
- Frontend mirrors the domain split: `web/src/api/<domain>.ts` + `web/src/routes/_authenticated/<domain>/` + types in `web/src/types/`

## Layers

**Domain:**
- Purpose: Pure business entities, request structs, sentinel errors
- Location: `internal/core/domain/<context>/` (e.g. `internal/core/domain/activity/activity.go`)
- Contains: Entity structs with `json:` tags, `*Request` structs, `Err*` vars, domain constants
- Depends on: `internal/models` (legacy shared constants like `models.RoleFinance`, `models.GovernanceModel`)
- Used by: services, ports, postgres adapters (scan targets)

**Ports:**
- Purpose: Interfaces the core requires from the outside world (persistence, token, hashing)
- Location: `internal/core/ports/` — `*_repository.go`, `token_service.go`, `password_hasher.go`, `user_finder.go`, `refresh_token_repo.go`, `errors.go` (`ErrNotFound`, `ErrConflict`, `ErrForeignKey`)
- Depends on: domain types (method signatures reference them)
- Used by: services (dependency injection)

**Services:**
- Purpose: All business logic — authorization checks, state transitions, workflow orchestration
- Location: `internal/core/services/<context>/`
- Contains: `Service` struct with port + shared-service fields; `NewService(...)` constructor; methods per use case
- Depends on: ports, sibling service singletons (injected), domain
- Used by: HTTP handlers, other services (via shared instances)

**Primary HTTP Adapters:**
- Purpose: HTTP boundary — parse, authorize (from middleware claims), validate length caps, delegate, envelope
- Location: `internal/adapters/primary/http/`
- Contains: `*Handler` structs (`{service *x.Service}`), request DTOs, `validate.go` (shared string-length validation), `handler_test_helper.go`
- Depends on: services, `internal/middleware` (claim getters), `pkg/api`
- Used by: `cmd/server/main.go` route registration

**Secondary Adapters:**
- Purpose: Persistence — raw SQL against PostgreSQL
- Location: `internal/adapters/secondary/postgres/`
- Contains: `*Repository` structs over `*pgxpool.Pool`, `scan*` helpers, `test_setup.go` (testcontainers), `exported_test_helpers.go`
- Depends on: domain types (return them), `pkg/...`, pgx
- Used by: composition root (`cmd/server/main.go`)

**Shared Infrastructure:**
- `internal/auth/` — JWT validate (used by middleware) + bcrypt + token service
- `internal/middleware/` — `Auth` (strict), `TryAuth` (best-effort), `RequireRole`, `CORS`, `RateLimiter`, `Logging`, `APIVersion`, claim getters/setters (`GetUserID`, `GetOrganizationID`, `GetRole`, `GetEmail`)
- `pkg/api/` — response envelope + receipt upload helpers
- `internal/db/` — pgx pool construction

## Data Flow

### Primary Request Path (authenticated CRUD)

1. Browser fetch → `api<T>()` in `web/src/lib/api.ts` with `credentials: "include"`; 401 triggers single-flight `POST /auth/refresh`, then one retry; navigation is never performed here (`web/src/lib/api.ts`)
2. Route loader/mutation → query options from `web/src/api/<domain>.ts`; `beforeLoad` in `web/src/routes/_authenticated.tsx` hydrates `AuthApis.profileQueryOpts` first
3. Go middleware chain: `TryAuth` populates context claims if cookie valid; `middleware.Auth` rejects with 401 otherwise (`internal/middleware/middleware.go`)
4. Handler parses `r.PathValue("id")`, query params, JSON body; validates string lengths (`internal/adapters/primary/http/validate.go`); reads `middleware.GetOrganizationID(ctx)` / `GetRole(ctx)`
5. Service applies business rules and calls ports (e.g. `customer.Service.Create` checks `role != finance → ErrForbidden`, `internal/core/services/customer/customer.go`)
6. Repository runs SQL, maps `pgx.ErrNoRows` → domain sentinel (`internal/adapters/secondary/postgres/customer_repository.go`)
7. Handler switches on sentinel errors → HTTP status via `api.RespondWithError` / `api.RespondWithJSON` (`pkg/api/response.go`); frontend unwraps `{data}` from the envelope

### Time Entry Approval Workflow

1. `POST /time-entries/{id}/submit` → `TimeEntryHandler.Submit` (`internal/adapters/primary/http/time_entry.go`) → `tesvc.Submit` (`internal/core/services/time_entry/time_entry.go`)
2. Service calls the **shared** `routing.Service.ResolveManagerStage` (`internal/core/services/routing/routing.go`): WG-anchored R-1 chain (WG manager + delegates, D-11 skip-to-finance if owner is in approver set) → commercial-without-WG rejection (`ErrActivityNotLoggable`, R-2) → personal-activity unit-manager upward walk → role-gated terminal case
3. Status transitions: `draft → submitted → pending_manager → pending_finance → approved/rejected`; each step appends an immutable row to the `*_approvals` tables; `currentApproverRole` drives who may act
4. Frontend reads status via `TimeEntriesApis` queries; `web/src/lib/role-visibility.ts` + `web/src/components/approval/*` gate UI

### Coverage Allocation (v0.2)

1. `PUT /time-entries/{id}/allocations` → `coverageHandler.PutAllocations` (`internal/adapters/primary/http/coverage_handler.go`) → `coveragesvc` (`internal/core/services/coverage/coverage.go`)
2. Allocation writers resolve the manager stage through the **same** shared `routing` service that approved the entry (D-08 parity — no second instance); period close (`POST /coverage/close`) freezes snapshots
3. Ledger/snapshot tables from migrations `019_coverage_allocations`, `020_coverage_snapshots`

### Auth Hydration & Refresh

1. Login sets HttpOnly `auth_token` (15 min) + `refresh_token` cookies (`internal/cookies/cookies.go`); refresh tokens rotate with reuse detection (`internal/adapters/secondary/postgres/refresh_token_repo.go`)
2. `_authenticated.tsx` `beforeLoad` → `client.fetchQuery(AuthApis.profileQueryOpts)` (`GET /auth/me`) → failure clears `["auth","me"]` cache and `redirect({ to: "/login", replace: true })`
3. `_auth.tsx` best-effort check: logged-in users are redirected to `/`; 401 is swallowed (breaks the historical infinite redirect loop)

**State Management:**
- Backend: stateless requests; identity in JWT claims injected into `context.Context` by middleware; authoritative state in PostgreSQL; no in-memory domain state
- Frontend: TanStack Query server cache (shared `queryClient`, `retry:false`, `staleTime:30000`, `refetchOnWindowFocus:false` in `web/src/lib/query-client.ts`); URL search state via `validateSearch` zod schemas (ADR-FE-017); mutations invalidate `queryKey` prefixes

## Key Abstractions

**Ports (repository interfaces):**
- Purpose: Persistence contracts per aggregate — `internal/core/ports/*_repository.go`
- Examples: `ports.TimeEntryRepository`, `ports.ActivityRepository`, `ports.ExportRepository`
- Pattern: Methods return domain types and domain sentinel errors; implemented by `internal/adapters/secondary/postgres/*_repository.go`

**Shared services (D-G parity):**
- Purpose: Single-instance service singletons consumed by multiple domains so routing/planning rules cannot drift
- Examples: `routing.Service` (shared by time_entry, coverage, activity, direction, availability), `orgsettings.Service` (mode gate shared by direction, availability)
- Pattern: Injected as concrete `*Service` pointers via constructors in `cmd/server/main.go`; the file's comments explicitly forbid second instances (`main.go:135-136, 182-186`)

**Response envelope:**
- Purpose: Uniform JSON contract `{data: ...}` success / `{error: ...}` failure
- Location: `pkg/api/response.go`; unwrapped client-side in `web/src/lib/api.ts` (`(await res.json() as ApiResponse<T>).data`)

**Frontend API option modules:**
- Purpose: Query/mutation definitions with query keys, staleTime, invalidation, toasts
- Examples: `web/src/api/activities.ts` (`activitiesQueryKey`, `activitiesQueryOpts`, `createActivityMutationOpts`, exported as `ActivitiesApis`)
- Pattern: Route loaders call `client.ensureQueryData(...)`; mutations invalidate prefix keys and toast on success/failure

## Entry Points

**`cmd/server/main.go`** — the composition root:
- Triggers: `go run ./cmd/server` (or Docker); listens on `PORT` (default 8080)
- Responsibilities: env validation (`JWT_SECRET` fatal in prod/staging), pool construction, build every repo → service → handler, register all ~130 routes with Go 1.22+ `mux.HandleFunc("METHOD /path", ...)`, middleware chain, CORS allowlist from `ALLOWED_ORIGINS`
- Construction order is load-bearing: `routingSvc` before time_entry/coverage/direction/availability; orgsettings before direction/availability; availability before organization (org schedule validates contract_type through it) — `cmd/server/main.go:135-202`

**`cmd/migrate/main.go`** — migration CLI:
- Triggers: `go run ./cmd/migrate -up|-down [-dir migrations]`
- Responsibilities: applies `migrations/*.up.sql` sorted / rolls back `*.down.sql` reversed; idempotent via "already exists"/"does not exist" skip

**`web/src/main.tsx`** — SPA bootstrap:
- Creates the TanStack Router with `context: { client: queryClient }`, renders `QueryClientProvider` + `RouterProvider`; `web/src/routeTree.gen.ts` is generated by the Vite TanStack plugin

**`web/src/routes/**`** — file-based route tree:
- `__root.tsx` root layout, `_authenticated.tsx` guard + `AppShell`, `_auth.tsx` guard, feature routes with zod-validated search, loaders, `errorComponent: RouteError`, `pendingMs`

**`web/e2e/*.spec.ts`** — Playwright E2E entry points against the running dev stack.

## Architectural Constraints

- **Singleton services / no second instances:** `routing`, `orgsettings`, and per-domain repos must be constructed once and shared (`cmd/server/main.go` comments; D-G parity). The wiring is compile-forced: services take `*routing.Service`, not an interface.
- **Threading:** Standard Go net/http goroutine-per-request; shared state is limited to the in-memory rate limiter maps (`internal/middleware/ratelimit.go`, guarded by `sync.RWMutex`) and the single-flight `refreshPromise` in `web/src/lib/api.ts`. No worker pools or background goroutines in the server process.
- **Rate limiting is in-memory** (`internal/middleware/ratelimit.go`) — assumes a single server instance; multi-instance deploys would need a shared store.
- **ServeMux most-specific-wins:** literal `/organizations/settings` routes coexist with typed `/organizations/{id}/settings` wildcards; the literals must not be removed (`cmd/server/main.go:254-257`, Pitfall 6).
- **Domain purity:** domain packages must not import services/adapters; `internal/models` is the one tolerated legacy import (shared constants).
- **Immutable/append-only histories:** `*_approvals` rows and ticket comments/history are never updated or deleted; tickets deliberately have no `DELETE` route (`cmd/server/main.go:272-274`).
- **No hard navigation from the HTTP client:** redirects only happen in route guards via TanStack `redirect()` — `web/src/lib/api.ts` throws `UnauthorizedError` instead (root cause of the historical infinite loop).
- **API versioning:** `Accept: application/json; version=v1` parsed by `internal/middleware/version.go`; v1 default.

## Anti-Patterns

### Legacy `internal/models` shared constants in hex domain packages

**What happens:** Hex domain/services import `internal/models` for role/status/governance constants (e.g. `models.RoleFinance` in `internal/core/services/customer/customer.go`, `models.GovernanceModel` in `internal/core/domain/customer/customer.go`), while newer domains define their own constants in-domain (`internal/core/domain/activity/activity.go`).
**Why it's wrong:** Two vocabularies for the same concept; `internal/models/models.go` (477 lines) is an unmigrated legacy surface documented in `plans/hexagonal-migration.md` as legacy glue.
**Do this instead:** New domains define constants locally (activity/coverage/direction already do); `internal/models` should shrink as domains migrate.

### Pass-through services

**What happens:** Some services only delegate: `customer.Service.List` → `repo.ListByOrg`, `export.Service.Timesheets` → `repo.Timesheets` (`internal/core/services/customer/customer.go`, `internal/core/services/export/export.go`).
**Why it's wrong:** An extra hop with no logic; the hex boundary's value is the enforcement and orchestration that richer services (time_entry, coverage) demonstrate.
**Do this instead:** Fine while the service is a stable seam for future policy; don't add more layers of delegation when a handler could call a port directly — but keep the pattern consistent while it exists.

### Handler 500 fallthrough for unknown service errors

**What happens:** Handlers `switch` on known sentinels and fall through to `http.StatusInternalServerError` with a generic message for anything else (`internal/adapters/primary/http/customer.go:105-107`).
**Why it's wrong:** Real failures (constraint violations, conflicts) surface to clients as opaque 500s instead of actionable 4xx.
**Do this instead:** Map `ports.ErrNotFound/ErrConflict/ErrForeignKey` (`internal/core/ports/errors.go`) centrally, or return richer domain errors from services.

### Stale worktree snapshot in-tree

**What happens:** `.gsd-worktrees/M001/` contains an older full snapshot of the repo (including its own `.planning/`), `main`, `server`, `migrate` binaries, and `web/dist`/`web/node_modules` are committed/left in place.
**Why it's wrong:** Confuses navigation (duplicate `internal/`, `web/` trees) and bloats the working tree.
**Do this instead:** Remove or archive the worktree directory; rely on git.

## Error Handling

**Strategy:** Domain sentinel errors (`errors.New` vars exported per package, e.g. `customer.ErrForbidden`, `activity.ErrActivityNotLoggable`) + a shared trio in `internal/core/ports/errors.go` (`ErrNotFound`, `ErrConflict`, `ErrForeignKey`). Repositories translate `pgx.ErrNoRows` → `ErrXxxNotFound`. Handlers map sentinels → HTTP status with user-facing messages; unknown errors → 500.

**Patterns:**
- Service methods return `(value, error)`; never panic for expected failures
- Handler error mapping: `switch err { case domain.ErrForbidden: 403; case domain.ErrNotFound: 404; default: 500 }`
- Frontend: `api<T>()` throws `Error` with the server message or `UnauthorizedError` for failed refresh; mutations `toast.error(...)` in `onError`

## Cross-Cutting Concerns

**Logging:** stdlib `log` in the server; `middleware.Logging` logs `METHOD path status durationMs` per request (`internal/middleware/middleware.go:145-155`); no structured logging library.
**Validation:** Boundary-level length caps in `internal/adapters/primary/http/validate.go` (`MaxNameLength` etc.); domain-level format/value checks in services; DB CHECK constraints as the final guard (migrations).
**Authentication:** `middleware.Auth` wraps every protected route; claims (userID, orgID, role, email) flow through `context.Context`; `middleware.TryAuth` is the outermost layer for best-effort identity; role gates are re-checked inside services (handler-level `RequireRole` exists but service-level checks dominate).
**CORS:** Allowlist from `ALLOWED_ORIGINS` with `Access-Control-Allow-Credentials: true` (`internal/middleware/cors.go`).
**Rate limiting:** Three limiter instances: auth endpoints (default 5/min, `RATE_LIMIT`), password reset (3/60s), outer anonymous limiter (default 20/min, `ANONYMOUS_RATE_LIMIT`, authenticated 100/min) (`cmd/server/main.go:84-91, 388-394`).

---

*Architecture analysis: 2026-08-12*
