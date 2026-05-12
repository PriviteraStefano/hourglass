# Architecture

**Analysis Date:** 2026-05-12

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React 19)                    │
│    TanStack Router + TanStack Query + Tailwind + shadcn    │
├─────────────────────────────────────────────────────────────┤
│                    API Gateway (Go HTTP)                     │
│           Middleware: Auth, CORS, RateLimit, Logging        │
├──────────────────┬──────────────────┬─────────────────────────┤
│    HTTP Handlers  │   Services      │    Domain Models      │
│   (Adapters/      │   (Core/        │    (Core/Domain)      │
│    Primary)       │    Services)    │                       │
│                   │                 │                       │
│  - auth.go        │ - auth          │ - time_entry          │
│  - time_entry.go  │ - time_entry    │ - project             │
│  - project.go     │ - project       │ - contract            │
│  - contract.go    │ - contract      │ - organization         │
│  - organization   │ - organization  │ - user                │
│                   │                 │                       │
└─────────┬─────────┴────────┬────────┴──────────┬──────────┘
          │                  │                    │
          ▼                  ▼                    ▼
┌─────────────────────────────────────────────────────────────┐
│                    Ports (Interfaces)                        │
│           (Core/Ports - Abstract Boundaries)                │
│                                                             │
│  - TimeEntryRepository    - UserRepository                  │
│  - ProjectRepository      - OrganizationRepository          │
│  - ContractRepository    - TokenService                     │
└───────────────────────────────┬─────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                 Secondary Adapters (SurrealDB)               │
│                (Adapters/Secondary/Surrealdb)               │
│                                                             │
│  - TimeEntryRepository    - UserRepository                  │
│  - ProjectRepository      - OrganizationRepository          │
│  - TokenService           - PasswordHasher                  │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| HTTP Handlers | Thin adapters that parse HTTP requests, extract auth context, call services, format responses | `internal/adapters/primary/http/*.go` |
| Services | Business logic orchestration, validation, state transitions | `internal/core/services/*/*.go` |
| Domain | Entity definitions, business rules, errors | `internal/core/domain/*/*.go` |
| Ports | Interfaces defining repository capabilities | `internal/core/ports/*.go` |
| SurrealDB Repositories | Database operations, SurrealDB record mapping | `internal/adapters/secondary/surrealdb/*.go` |
| Auth Service | JWT token generation/validation, claims | `internal/auth/auth.go` |
| Middleware | Auth extraction, CORS, rate limiting, logging | `internal/middleware/middleware.go` |

## Pattern Overview

**Overall:** Hexagonal Architecture (Ports & Adapters)

**Key Characteristics:**
- Business logic depends on interfaces (ports), not concrete implementations
- HTTP handlers are thin - they parse requests and delegate to services
- Services contain all business rules and orchestration
- Repository implementations are interchangeable via port interfaces
- Domain models contain business logic (e.g., `CanEdit()`, `IsOwner()`)
- Single entry point: `cmd/server/main.go` wires all components

## Layers

**HTTP Handler Layer:**
- Purpose: Adapter between HTTP protocol and application services
- Location: `internal/adapters/primary/http/`
- Contains: Request parsing, response formatting, auth context extraction
- Depends on: Services, not repositories directly
- Used by: `main.go` via `http.ServeMux`

**Service Layer:**
- Purpose: Business logic orchestration and validation
- Location: `internal/core/services/*/`
- Contains: CRUD operations, state transitions, validation logic
- Depends on: Domain models, ports (repository interfaces)
- Used by: HTTP handlers

**Domain Layer:**
- Purpose: Business entities, rules, and errors
- Location: `internal/core/domain/*/`
- Contains: Struct definitions, validation methods, business errors
- Depends on: None (pure domain)
- Used by: Services, repositories

**Port Layer:**
- Purpose: Abstract interfaces for external dependencies
- Location: `internal/core/ports/`
- Contains: Repository interfaces, service abstractions
- Depends on: Domain models for type signatures
- Used by: Services (as dependencies)

**Secondary Adapter Layer:**
- Purpose: Concrete implementations of ports
- Location: `internal/adapters/secondary/surrealdb/`
- Contains: Database operations, SurrealDB-specific mapping
- Depends on: Ports (implements interfaces), SurrealDB driver
- Used by: `main.go` (injected into services)

## Data Flow

### Primary Request Path

1. **HTTP Request** arrives at `main.go` mux
2. **Auth Middleware** extracts JWT from cookie, validates, populates context (`internal/middleware/middleware.go:23-44`)
3. **Handler** receives request, extracts context (UserID, OrgID, Role), parses body (`internal/adapters/primary/http/time_entry.go:47-88`)
4. **Service** performs business logic, validates rules, may call repository (`internal/core/services/time_entry/time_entry.go:21-62`)
5. **Repository** executes database query via SurrealDB (`internal/adapters/secondary/surrealdb/time_entry_repository.go:63-82`)
6. **Response** flows back: Repository → Service → Handler → Client

### Auth Hydration Flow

1. **Frontend** navigates to protected route (`_authenticated.tsx`)
2. **`beforeLoad`** calls `AuthApis.profileQueryOpts` (`GET /auth/me`)
3. **Backend AuthHandler** validates JWT cookie via middleware
4. **Response** returns user profile + organization context
5. **React Query** caches profile, route renders

### Token Refresh Flow

1. **Frontend API client** receives 401 (`web/src/lib/api.ts:20`)
2. **Refresh promise** calls `POST /auth/refresh`
3. **Backend AuthHandler** validates refresh token, issues new access token
4. **Frontend** retries original request
5. If refresh fails, redirect to `/login`

## Key Abstractions

**Repository Pattern:**
- Purpose: Abstract data access, enable testability and swapping backends
- Examples: `TimeEntryRepository`, `UserRepository`, `ProjectRepository`
- Pattern: Interface in `ports/`, implementation in `adapters/secondary/surrealdb/`

**Service Pattern:**
- Purpose: Encapsulate business logic, coordinate multiple repositories
- Examples: `time_entry.Service`, `auth.Service`, `project.Service`
- Pattern: Constructor takes repository interfaces, business methods orchestrate logic

**Domain Errors:**
- Purpose: Express business rule violations as typed errors
- Examples: `ErrTimeEntryNotFound`, `ErrEntryNotDraft`, `ErrPeriodLocked`
- Pattern: Package-level variables in domain package, checked by handlers

## Entry Points

**Backend Server:**
- Location: `cmd/server/main.go`
- Triggers: `go run ./cmd/server`
- Responsibilities: Initialize SurrealDB, create all repositories/services, register routes, start HTTP server on port 8080

**Schema Bootstrap:**
- Location: `cmd/schema/main.go`
- Triggers: `go run ./cmd/schema`
- Responsibilities: Load and apply `.surql` schema files to SurrealDB

**PostgreSQL Migrations:**
- Location: `cmd/migrate/main.go`
- Triggers: `go run ./cmd/migrate -up -dir migrations`
- Responsibilities: Apply SQL migrations to PostgreSQL

**Frontend Dev Server:**
- Location: `web/vite.config.ts` + `web/src/main.tsx`
- Triggers: `cd web && bun run dev`
- Responsibilities: Vite dev server on port 3000, proxies `/api` to `http://localhost:8080`

## Architectural Constraints

- **Threading:** Go's standard `http.ServeMux` is single-threaded by default (no goroutine-per-request isolation)
- **Global state:** None detected - all dependencies injected via constructors
- **Circular imports:** None - domain has no dependencies, services depend on domain+ports, adapters depend on ports
- **Database choice:** SurrealDB is the primary database; PostgreSQL is legacy for migrations only
- **Auth mechanism:** JWT stored in HttpOnly cookie + refresh token pattern

## Anti-Patterns

### Handler Contains Business Logic

**What happens:** Business validation logic (e.g., checking status, role, ownership) appears in HTTP handlers
**Why it's wrong:** Handlers become complex, harder to test, business rules scatter across layers
**Do this instead:** Keep handlers thin - only parse request, extract context, call service, format response. Example: `internal/adapters/primary/http/time_entry.go`

### Domain Models as Anemic Data Containers

**What happens:** Domain structs contain only data fields with no methods
**Why it's wrong:** Business rules leak into services, making them harder to understand and change
**Do this instead:** Add business methods to domain models. Example: `internal/core/domain/time_entry/time_entry.go:78-87` has `IsOwner()`, `CanEdit()`, `CanSubmit()`

## Error Handling

**Strategy:** Domain errors + HTTP status mapping

**Patterns:**
- Domain errors defined as package-level variables (e.g., `ErrTimeEntryNotFound`)
- Handlers check domain errors and return appropriate HTTP status codes
- API responses use envelope format: `{ data: ... }` or `{ error: ... }` (`pkg/api/response.go`)
- Frontend throws on non-2xx responses, extracts error message (`web/src/lib/api.ts`)

## Cross-Cutting Concerns

**Logging:** Go standard `log` package, middleware logs method/path/status/duration
**Validation:** Handlers validate request format; services validate business rules
**Authentication:** JWT in HttpOnly `auth_token` cookie; refresh via `refresh_token` cookie
**Authorization:** Role extracted from JWT claims, passed via context to handlers
**Rate Limiting:** Token bucket algorithm in `middleware/ratelimit.go`

---

*Architecture analysis: 2026-05-12*