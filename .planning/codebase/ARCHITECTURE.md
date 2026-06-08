<!-- refreshed: 2026-06-08 -->
# Architecture

**Analysis Date:** 2026-06-08

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                         FRONTEND (React SPA)                            │
│  ┌───────────┐  ┌──────────────┐  ┌────────────────┐  ┌─────────────┐  │
│  │  Routes   │  │  API Modules │  │  Components    │  │   Types     │  │
│  │ TanStack  │  │  TanStack    │  │  shadcn/ui     │  │  models.ts  │  │
│  │ Router    │  │  React Query │  │  Tailwind      │  │  api.ts     │  │
│  └─────┬─────┘  └──────┬───────┘  └────────────────┘  └─────────────┘  │
│        │               │                                                │
│        └───────┬───────┘                                                │
│                │                                                        │
│        ┌───────▼────────┐                                               │
│        │  lib/api.ts    │  ← HTTP client with cookie auth + 401 retry  │
│        └───────┬────────┘                                               │
└────────────────┼────────────────────────────────────────────────────────┘
                 │ HTTP/JSON (HttpOnly cookies for auth)
                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      BACKEND (Go 1.26.1 HTTP Server)                    │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │              MIDDLEWARE STACK (outer → inner)                    │   │
│  │  TryAuth → RateLimiter → Logging → APIVersion → CORS → mux      │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │                                       │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │           PRIMARY ADAPTERS (HTTP Handlers)                       │   │
│  │  internal/adapters/primary/http/                                 │   │
│  │  auth.go  project.go  contract.go  customer.go  organization.go │   │
│  │  time_entry.go  unit.go  working_group.go  export.go  ...       │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │                                       │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │              CORE SERVICES (Business Logic)                     │   │
│  │  internal/core/services/                                        │   │
│  │  auth/  project/  contract/  customer/  organization/           │   │
│  │  time_entry/  unit/  working_group/  export/  invitation/       │   │
│  │  password_reset/  expense/                                      │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │ depends on ports (interfaces)        │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │              PORTS (Interface Contracts)                        │   │
│  │  internal/core/ports/                                           │   │
│  │  *Repository interfaces, TokenService, PasswordHasher, etc.    │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │                                       │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │              DOMAIN (Pure Business Entities)                    │   │
│  │  internal/core/domain/                                          │   │
│  │  auth/  contract/  customer/  organization/  project/           │   │
│  │  time_entry/  unit/  working_group/  invitation/  ...          │   │
│  │  Zero external dependencies. Value objects, errors, entities.  │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │                                       │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │           SECONDARY ADAPTERS (PostgreSQL Implementations)        │   │
│  │  internal/adapters/secondary/postgres/                           │   │
│  │  *Repository implementations, pgxpool-based queries             │   │
│  └──────────────────────────────┬──────────────────────────────────┘   │
│                                 │                                       │
│  ┌──────────────────────────────▼──────────────────────────────────┐   │
│  │              INFRASTRUCTURE                                     │   │
│  │  internal/db/pgpool.go  - PostgreSQL connection pool (singleton)│   │
│  │  internal/auth/         - JWT + bcrypt                          │   │
│  │  internal/middleware/   - Auth, CORS, logging, ratelimit        │   │
│  │  internal/cookies/     - HttpOnly cookie helpers               │   │
│  │  internal/models/     - Shared data structures and constants   │   │
│  │  pkg/api/             - JSON response envelope                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Architecture Style: Hexagonal (Ports & Adapters)

The project follows a **hexagonal (ports & adapters) architecture** for the main application flow. Some handlers are still being migrated from a legacy handler-based pattern.

**Key characteristic:** Business logic lives in `internal/core/services/`, depends only on `internal/core/ports/` interfaces, and is completely unaware of HTTP or PostgreSQL.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Server wiring | Wires dependencies, registers routes | `cmd/server/main.go` |
| Migration CLI | Applies SQL migrations to Postgres | `cmd/migrate/main.go` |
| Auth JWT | JWT token creation/validation + bcrypt hashing | `internal/auth/auth.go` |
| Password Hasher | bcrypt interface | `internal/auth/password_hasher.go` |
| Token Service | JWT service wrapper | `internal/auth/token_service.go` |
| DB Pool | Postgres pgxpool singleton | `internal/db/pgpool.go` |
| Auth Middleware | JWT validation into context | `internal/middleware/middleware.go` |
| CORS Middleware | CORS headers | `internal/middleware/cors.go` |
| Logging Middleware | Request logging | `internal/middleware/middleware.go` |
| Rate Limiter | Per-IP/per-user rate limiting | `internal/middleware/ratelimit.go` |
| API Versioning | Version header parsing | `internal/middleware/version.go` |
| Health Handler | Health check endpoint | `internal/handlers/health_handler.go` |
| API Response | JSON envelope {data, error} | `pkg/api/response.go` |
| Models | Shared types, enums, constants | `internal/models/models.go` |
| Cookies | HttpOnly cookie helpers | `internal/cookies/cookies.go` |

### Primary Adapters (HTTP Handlers)

| Handler | File | Service |
|---------|------|---------|
| Auth | `internal/adapters/primary/http/auth.go` | `auth.Service` |
| Time Entry | `internal/adapters/primary/http/time_entry.go` | `time_entry.Service` |
| Project | `internal/adapters/primary/http/project.go` | `project.Service` |
| Contract | `internal/adapters/primary/http/contract.go` | `contract.Service` |
| Customer | `internal/adapters/primary/http/customer.go` | `customer.Service` |
| Organization | `internal/adapters/primary/http/organization.go` | `organization.Service` |
| Unit | `internal/adapters/primary/http/unit.go` | `unit.Service` |
| Working Group | `internal/adapters/primary/http/working_group.go` | `working_group.Service` |
| Export | `internal/adapters/primary/http/export.go` | `export.Service` |
| Invitation | `internal/adapters/primary/http/invitation.go` | `invitation.Service` |
| Password Reset | `internal/adapters/primary/http/password_reset.go` | `password_reset.Service` |

### Services (Business Logic)

| Service | File | Depends On |
|---------|------|------------|
| Auth | `internal/core/services/auth/auth.go` | UserRepository, OrgRepository, TokenService, PasswordHasher, RefreshTokenRepo |
| Time Entry | `internal/core/services/time_entry/time_entry.go` | TimeEntryRepository, AuditLogRepository |
| Project | `internal/core/services/project/project.go` | ProjectRepository |
| Contract | `internal/core/services/contract/contract.go` | ContractRepository |
| Customer | `internal/core/services/customer/customer.go` | CustomerRepository |
| Organization | `internal/core/services/organization/organization.go` | OrganizationManagementRepository |
| Unit | `internal/core/services/unit/unit.go` | UnitRepository |
| Working Group | `internal/core/services/working_group/working_group.go` | WorkingGroupRepository |
| Export | `internal/core/services/export/export.go` | ExportRepository |
| Invitation | `internal/core/services/invitation/invitation.go` | InvitationRepository |
| Password Reset | `internal/core/services/password_reset/password_reset.go` | PasswordResetRepo, UserRepo, UserFinder, PasswordHasher, TokenService, RefreshTokenRepo |

### Domain Entities (per domain subdirectory in `internal/core/domain/`)

| Domain | Files | Key Types |
|--------|-------|-----------|
| auth | `user.go`, `credentials.go`, `errors.go`, `membership.go`, `organization.go` | User, Email, Password, Username, OrganizationMembership |
| time_entry | `time_entry.go` | TimeEntry, AuditLog, CreateTimeEntryRequest, UpdateTimeEntryRequest |
| contract | `contract.go` | Contract, ContractResponse, ContractAdoption |
| customer | `customer.go` | Customer, ContractSummary |
| organization | `organization.go` | Organization, Settings, Member |
| project | `project.go` | Project, ProjectResponse, ProjectAdoption, ProjectManager |
| unit | `unit.go` | Unit, UnitTreeNode, UnitMember |
| working_group | `working_group.go` | WorkingGroup, WorkingGroupMember |
| invitation | `invitation.go` | Invitation |
| password_reset | `password_reset.go` | PasswordReset |
| expense | (empty directory) | |

### Secondary Adapters (PostgreSQL)

| Repository | File |
|------------|------|
| User | `internal/adapters/secondary/postgres/user_repository.go` |
| Refresh Token | `internal/adapters/secondary/postgres/refresh_token_repo.go` |
| Time Entry | `internal/adapters/secondary/postgres/time_entry_repository.go` |
| Project | `internal/adapters/secondary/postgres/project_repository.go` |
| Contract | `internal/adapters/secondary/postgres/contract_repository.go` |
| Customer | `internal/adapters/secondary/postgres/customer_repository.go` |
| Organization | `internal/adapters/secondary/postgres/organization_repo.go` |
| Organization Membership | `internal/adapters/secondary/postgres/organization_membership_repo.go` |
| Organization Management | `internal/adapters/secondary/postgres/organization_management_repo.go` |
| Unit | `internal/adapters/secondary/postgres/unit_repository.go` |
| Unit Member | `internal/adapters/secondary/postgres/unit_member_repository.go` |
| Working Group | `internal/adapters/secondary/postgres/working_group_repository.go` |
| WG Member | `internal/adapters/secondary/postgres/wg_member_repository.go` |
| Invitation | `internal/adapters/secondary/postgres/invitation_repository.go` |
| Password Reset | `internal/adapters/secondary/postgres/password_reset_repository.go` |
| Export | `internal/adapters/secondary/postgres/export_repository.go` |
| Expense | `internal/adapters/secondary/postgres/expense_repository.go` |
| Subproject | `internal/adapters/secondary/postgres/subproject_repository.go` |
| User Finder | `internal/adapters/secondary/postgres/user_finder.go` |

### Ports (Interface Contracts)

All in `internal/core/ports/`:
- `errors.go` — Sentinel errors: `ErrNotFound`, `ErrConflict`, `ErrForeignKey`
- `*_repository.go` — Repository interfaces per domain
- `token_service.go` — `TokenService` interface (GenerateToken, ValidateToken)
- `password_hasher.go` — `PasswordHasher` interface (Hash, Check)
- `user_finder.go` — `UserFinder` interface (FindByEmail, FindByUsername)

## Layers

**Domain Layer:**
- Purpose: Pure business entities and value objects with zero external dependencies
- Location: `internal/core/domain/{domain}/`
- Contains: Struct definitions, factory functions, validation methods, sentinel errors
- Depends on: Go standard library, `github.com/google/uuid`
- Used by: Services layer (imports domain types)

**Ports Layer:**
- Purpose: Interface contracts that define what the application needs from the outside world
- Location: `internal/core/ports/`
- Contains: Repository interfaces, service interfaces, shared filter types
- Depends on: Domain types (for return/parameter types)
- Used by: Services depend on ports; adapters implement ports

**Services Layer:**
- Purpose: Application business logic, use case orchestration
- Location: `internal/core/services/{service}/`
- Contains: Service struct with business methods, domain logic orchestration
- Depends on: Domain entities + ports (interfaces)
- Used by: Primary adapters (HTTP handlers)

**Primary Adapters Layer (Driving):**
- Purpose: Accept external input, delegate to services, format responses
- Location: `internal/adapters/primary/http/`
- Contains: Thin HTTP handlers (parse request → call service → format response)
- Depends on: Services, Middleware, pkg/api
- Used by: HTTP server (middleware chain)

**Secondary Adapters Layer (Driven):**
- Purpose: Implement port interfaces for specific technologies
- Location: `internal/adapters/secondary/postgres/`
- Contains: PostgreSQL repository implementations using pgxpool
- Depends on: Ports interfaces, Domain entities, db/pgpool
- Used by: Injected into services at wiring time (`cmd/server/main.go`)

**Infrastructure Layer:**
- Purpose: Cross-cutting concerns and foundation code
- Location: `internal/auth/`, `internal/middleware/`, `internal/db/`, `internal/cookies/`, `pkg/api/`, `internal/models/`
- Contains: JWT service, middleware stack, DB pool singleton, cookie helpers, API response envelope, shared data types
- Depends on: External packages (golang-jwt, pgxpool, bcrypt)

## Data Flow

### Primary Request Path (e.g., Create Time Entry)

1. **HTTP Request** arrives at an endpoint registered in `cmd/server/main.go` (e.g., `POST /time-entries`)
2. **Middleware Chain** processes the request:
   - `TryAuth` — optional JWT parsing (doesn't block)
   - `RateLimiter` — per-IP/per-user rate limit check
   - `Logging` — request log with duration
   - `APIVersion` — version header detection
   - `CORS` — CORS header enforcement
3. **Auth Middleware** (`middleware.Auth(authService, handlerFunc)`) validates JWT cookie, injects `userID`, `organizationID`, `role`, `email` into context
4. **HTTP Handler** (e.g., `TimeEntryHandler.Create` in `internal/adapters/primary/http/time_entry.go`):
   - Parses request body via `json.NewDecoder`
   - Extracts user/org from context via `middleware.GetUserID(ctx)`, `middleware.GetOrganizationID(ctx)`
   - Maps request to domain types
   - Calls `service.Create(ctx, req)`
   - Handles errors with specific HTTP status mapping
   - Formats response via `api.RespondWithJSON`
5. **Service** (e.g., `time_entry.Service.Create` in `internal/core/services/time_entry/time_entry.go`):
   - Validates business rules (period locking, ownership, status)
   - Creates domain entities with `uuid.New()`
   - Calls `repo.Create(ctx, entry)` through port interface
6. **Repository** (e.g., `TimeEntryRepository.Create` in `internal/adapters/secondary/postgres/time_entry_repository.go`):
   - Executes SQL via pgxpool
   - Maps rows to domain entities
   - Wraps pgx errors to port sentinel errors via `wrapPGError`
7. **Response** flows back through the same chain with JSON `{ data: ... }` or `{ error: ... }` envelope

### Authentication Flow

1. User registers/logs in → `POST /auth/login` hits `AuthHandler.Login` (`internal/adapters/primary/http/auth.go`)
2. Handler calls `authService.Login(ctx, req)` from `internal/core/services/auth/auth.go`
3. Auth service validates credentials via `PasswordHasher.Check()`, generates JWT via `TokenService`, creates refresh token
4. Handler sets `auth_token` (15min) and `refresh_token` (7d) HttpOnly cookies
5. Frontend's `web/src/lib/api.ts` sends all requests with `credentials: 'include'`
6. On 401, frontend auto-calls `POST /auth/refresh` once, retries the original request
7. Protected routes in `web/src/routes/_authenticated.tsx` use `beforeLoad` to hydrate auth state via `fetchQuery(AuthApis.profileQueryOpts)`

### Frontend Data Flow (React Query + TanStack Router)

1. **Route definition** (e.g., `web/src/routes/_authenticated/time-entries/index.tsx`) declares `loaderDeps`, `loader`, `validateSearch`
2. **Route loader** uses `client.ensureQueryData(...)` to prefetch data via React Query
3. **API module** (e.g., `web/src/api/time-entries.ts`) defines `queryOptions` and `mutationOptions` with query keys
4. **HTTP client** (`web/src/lib/api.ts`) sends requests with cookie auth, handles 401 refresh loop
5. **Component** (e.g., `time-entries-page.tsx`) uses React Query hooks like `useSuspenseQuery`, `useMutation`
6. **Mutations** invalidate related queries on success via `queryClient.invalidateQueries({ queryKey: [...] })`

## Key Abstractions

**Service Struct Pattern:**
- Purpose: Encapsulates business logic for a bounded context
- Examples: `Service` structs in `internal/core/services/*/`
- Pattern: Private struct with port interface fields → constructor → public methods
```go
type Service struct {
    repo ports.TimeEntryRepository
}
func NewService(repo ports.TimeEntryRepository) *Service { ... }
func (s *Service) Create(ctx context.Context, req *time_entry.CreateTimeEntryRequest) (*time_entry.TimeEntry, error) { ... }
```

**Handler Struct Pattern:**
- Purpose: Thin HTTP adapter that delegates to a service
- Examples: `*Handler` in `internal/adapters/primary/http/*.go`
- Pattern: Struct with service dependency → constructor → handler methods with `(w, r)` signature
```go
type ProjectHandler struct {
    service *projectsvc.Service
}
func NewProjectHandler(service *projectsvc.Service) *ProjectHandler { ... }
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) { ... }
```

**Repository Struct Pattern:**
- Purpose: PostgreSQL implementation of a port interface
- Examples: `*Repository` in `internal/adapters/secondary/postgres/*.go`
- Pattern: Struct with `*pgxpool.Pool` → constructor → methods that match port interface
```go
type TimeEntryRepository struct {
    pool *pgxpool.Pool
}
func NewTimeEntryRepository(pool *pgxpool.Pool) *TimeEntryRepository { ... }
```

**Port Interface Pattern:**
- Purpose: Contract between service and adapter
- Examples: `internal/core/ports/*.go`
- Pattern: Go interface with domain types in signatures
```go
type TimeEntryRepository interface {
    List(ctx context.Context, orgID uuid.UUID, filters ListFilters) ([]time_entry.TimeEntry, error)
    GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error)
    ...
}
```

**Domain Entity Pattern:**
- Purpose: Pure data + behavior with zero external deps
- Examples: `internal/core/domain/*/*.go`
- Pattern: Struct + factory functions + method receivers (validation, state checks) + sentinel errors
```go
type TimeEntry struct { ... }
func (e *TimeEntry) IsOwner(userID uuid.UUID) bool { ... }
func (e *TimeEntry) CanEdit() bool { ... }
```

**DB Pool Singleton:**
- Location: `internal/db/pgpool.go`
- Pattern: `sync.Once` initialization of `*pgxpool.Pool`, shared across all repositories
- Initialized in `cmd/server/main.go` before wiring

**Middleware Chain:**
- Pattern: `func(http.Handler) http.Handler` composable middleware
- Applied from outer to inner: `TryAuth` → `RateLimiter` → `Logging` → `APIVersion` → `CORS` → mux

**Frontend API Module Pattern:**
- Location: `web/src/api/*.ts`
- Pattern: Named exports with `queryOptions()` and `mutationOptions()` wrappers
- All API modules import `api<T>()` from `web/src/lib/api.ts`
- Query keys use `['domain', ...qualifiers]` convention

## Entry Points

**Server Entry Point:**
- Location: `cmd/server/main.go`
- Triggers: `go run ./cmd/server` or compiled binary
- Responsibilities: Initializes DB pool, JWT service, all repositories, all services, all handlers; registers routes on `http.ServeMux` with Go 1.22+ method patterns (`"POST /auth/login"`); applies middleware wrapper chain; starts HTTP server on `PORT` (default 8080)

**Migration Entry Point:**
- Location: `cmd/migrate/main.go`
- Triggers: `go run ./cmd/migrate -up|-down|-all`
- Responsibilities: Connects directly to PostgreSQL, reads `*.up.sql`/`*.down.sql` from migrations directory, applies in sorted order

**Frontend Entry Point:**
- Location: `web/src/main.tsx`
- Triggers: `bun run dev` (Vite dev server)
- Responsibilities: Creates TanStack Router with route tree, wraps in `QueryClientProvider`, renders to DOM

## Architectural Constraints

- **Threading:** Go's standard goroutine-per-request model. Services and repositories are stateless (except for the rate limiter's in-memory map with `sync.RWMutex`). DB pool is goroutine-safe.
- **Global state:** `internal/db/pgpool.go` uses a package-level `poolInstance` with `sync.Once` — a singleton database connection pool. The rate limiter in `internal/middleware/ratelimit.go` holds a `map[string]*clientInfo` with `sync.RWMutex` — stateful in-memory.
- **Circular imports:** Not detected. Domain → Ports ← Services ← Primary Adapters is a strict acyclic dependency graph.
- **Handler naming:** Each handler file in `internal/adapters/primary/http/` is named after the domain (e.g., `time_entry.go`, `project.go`) — not prefixed with `handler_`.
- **Service naming:** Each service directory in `internal/core/services/` is named after the domain (e.g., `time_entry/`, `project/`).

## Error Handling

**Strategy:** Domain sentinel errors + port layer wrapping + handler HTTP status mapping

**Patterns:**
- Domain errors defined as `var ErrXxx = errors.New("...")` at the bottom of domain or service files
- Port-level errors: `internal/core/ports/errors.go` defines `ErrNotFound`, `ErrConflict`, `ErrForeignKey`
- PostgreSQL adapter wraps pgx errors to port errors via `wrapPGError()` in `internal/adapters/secondary/postgres/postgres.go`
- HTTP handlers switch on domain errors to return appropriate HTTP status codes
- The `pkg/api/response.go` provides `RespondWithJSON` and `RespondWithError` helpers

## Cross-Cutting Concerns

**Logging:** Standard library `log.Printf` in middleware (`internal/middleware/middleware.go`) — not a structured logger. Logs method, path, status code, duration in milliseconds.

**Validation:** Dual-level — frontend uses zod schemas for client-side validation; backend services validate domain invariants; HTTP handlers do basic input format checks (non-empty, regex patterns).

**Authentication:** JWT-based with HttpOnly cookies (`auth_token` + `refresh_token`). Two middleware modes:
- `middleware.TryAuth` — optional auth (sets context if valid token, doesn't block)
- `middleware.Auth` — required auth (returns 401 if missing/invalid)
- `middleware.RequireRole` — role-based authorization on top of auth

**CORS:** Configurable via `ALLOWED_ORIGINS` env var (defaults to `http://localhost:3000`). Supports `credentials: true` for cookie auth. Handles preflight OPTIONS requests.

**Rate Limiting:** In-memory sliding window per-IP (anonymous) or per-user (authenticated). Configurable limits: 20/min anonymous, 100/min authenticated.

**API Versioning:** Accept header based (`Accept: application/vnd.hourglass+json version=1`). Currently only v1.

---

*Architecture analysis: 2026-06-08*
