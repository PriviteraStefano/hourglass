# Codebase Structure

**Analysis Date:** 2026-06-08

## Directory Layout

```
hourglass/
├── cmd/
│   ├── migrate/             # PostgreSQL migration CLI
│   │   └── main.go
│   ├── schema/              # Schema generation (if any)
│   └── server/
│       └── main.go          # Server entry point + route wiring
├── internal/
│   ├── adapters/
│   │   ├── primary/
│   │   │   └── http/        # Thin HTTP handlers (driving adapters)
│   │   └── secondary/
│   │       └── postgres/    # PostgreSQL repositories (driven adapters)
│   ├── auth/                # JWT token service + bcrypt
│   ├── cookies/             # HttpOnly cookie helpers
│   ├── core/
│   │   ├── domain/          # Pure business entities (one dir per domain)
│   │   ├── ports/           # Interface contracts
│   │   └── services/        # Business logic (one dir per service)
│   ├── db/                  # PostgreSQL connection pool (singleton)
│   ├── handlers/            # Health check handler
│   ├── middleware/          # Auth, CORS, logging, ratelimit, versioning
│   └── models/              # Shared data structures and constants
├── pkg/
│   └── api/                 # JSON response envelope {data, error}
├── migrations/              # SQL migration files (.up.sql / .down.sql)
├── web/                     # React SPA frontend
│   └── src/
│       ├── api/             # TanStack Query queryOptions/mutationOptions
│       ├── components/      # Reusable UI components
│       │   ├── app/         # App-level (profile-menu, org-switcher)
│       │   ├── layout/      # Sidebar, app-shell, header
│       │   └── ui/          # shadcn/ui primitives
│       ├── hooks/           # Custom React hooks
│       ├── lib/             # API client, query client, utilities
│       ├── routes/          # TanStack Router file-based routes
│       │   ├── __root.tsx
│       │   ├── _auth.tsx
│       │   ├── _auth/       # Login, register, password-reset
│       │   ├── _authenticated.tsx
│       │   └── _authenticated/  # Protected pages
│       └── types/           # TypeScript type definitions
├── bin/                     # Compiled Go binary
├── uploads/                 # Receipt file uploads directory
├── opens/                    # Not applicable
├── docs/                    # Documentation
├── plans/                   # Architecture plans (hexagonal-migration.md)
├── scripts/                 # Utility scripts
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod / go.sum
├── package.json             # Root package.json (likely misc tools)
└── AGENTS.md                # AI agent guide to codebase
```

## Directory Purposes

### Root Level

**`cmd/`:**
- Purpose: Application entry points
- Contains: Go `main` packages
- Key files:
  - `cmd/server/main.go`: Server entry, dependency wiring, route registration (216 lines)
  - `cmd/migrate/main.go`: Database migration CLI tool (185 lines)

**`internal/`:**
- Purpose: Application core — not importable by external Go packages
- Contains: All backend logic (domain, services, adapters, middleware, etc.)

**`migrations/`:**
- Purpose: Database schema and seed SQL files
- Contains: `.up.sql` and `.down.sql` files executed by `cmd/migrate`
- Key files:
  - `000_full_schema.up.sql` / `.down.sql`: Full schema migration
  - `003_seed.up.sql`: Seed data
  - `008_verification_tokens.up.sql` / `.down.sql`: Verification tokens migration

**`web/`:**
- Purpose: Frontend React SPA
- Contains: Vite + React + TanStack Router + TanStack Query application

### Backend (internal/)

**`internal/core/domain/{domain}/`:**
- Purpose: Pure business entities with zero external dependencies
- Contains: Struct definitions, factory functions, validation methods, sentinel errors
- Subdirectories: `auth/`, `contract/`, `customer/`, `expense/` (empty), `invitation/`, `organization/`, `password_reset/`, `project/`, `time_entry/`, `unit/`, `working_group/`

**`internal/core/ports/`:**
- Purpose: Interface contracts that the core depends on
- Contains: 19 Go files defining repository and service interfaces
- Key files:
  - `errors.go`: Shared sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrForeignKey`)
  - `user_repository.go`: `UserRepository` interface
  - `time_entry_repository.go`: `TimeEntryRepository` + `AuditLogRepository` interfaces
  - `token_service.go`: `TokenService` interface
  - `password_hasher.go`: `PasswordHasher` interface

**`internal/core/services/{service}/`:**
- Purpose: Business logic for each bounded context
- Contains: Service struct, public methods, domain error mapping
- Subdirectories: `auth/`, `contract/`, `customer/`, `expense/`, `export/`, `invitation/`, `organization/`, `password_reset/`, `project/`, `time_entry/`, `unit/`, `working_group/`, `testdata/`

**`internal/adapters/primary/http/`:**
- Purpose: HTTP handlers (driving adapters)
- Contains: 11 handler files (one per domain) + corresponding test files
- Each file defines a `*Handler` struct with a `*Service` dependency
- Key files: `auth.go`, `time_entry.go`, `project.go`, `contract.go`, `customer.go`, `organization.go`, `unit.go`, `working_group.go`, `export.go`, `invitation.go`, `password_reset.go`

**`internal/adapters/secondary/postgres/`:**
- Purpose: PostgreSQL implementations of port interfaces
- Contains: 20 repository files + test files + helpers
- Key files: `postgres.go` (error wrapping), `exported_test_helpers.go` (test DB setup)

**`internal/auth/`:**
- Purpose: JWT and password infrastructure
- Contains:
  - `auth.go`: `Service` struct with `GenerateToken`, `ValidateToken`, `HashPassword`, `CheckPassword`, `GenerateRefreshToken`
  - `password_hasher.go`: `PasswordHasher` struct (wraps auth Service for port interface)
  - `token_service.go`: `TokenService` struct (wraps auth Service for port interface)

**`internal/middleware/`:**
- Purpose: HTTP middleware composable chain
- Contains:
  - `middleware.go`: `TryAuth`, `Auth`, `RequireRole`, context accessors (`GetUserID`, `GetOrganizationID`, `GetRole`, `GetEmail`), `Logging`
  - `cors.go`: `CORS` middleware factory
  - `ratelimit.go`: `RateLimiter` with per-IP/per-user limits
  - `version.go`: `APIVersion` middleware

**`internal/db/`:**
- Purpose: PostgreSQL connection pool
- Contains: `pgpool.go` — singleton `*pgxpool.Pool` using `sync.Once`

**`internal/models/`:**
- Purpose: Shared data structures and domain constants
- Contains: `models.go` — enums (`Role`, `EntryStatus`, `GovernanceModel`, `ProjectType`, `ExpenseCategory`, `ApprovalAction`), entity structs (`User`, `Organization`, `Contract`, `Project`, `TimeEntry`, `Expense`, etc.)

**`internal/handlers/`:**
- Purpose: Legacy handler (not yet migrated)
- Contains: `health_handler.go` — simple health check returning `{"status": "ok"}`

**`internal/cookies/`:**
- Purpose: HTTP cookie helpers for auth tokens
- Contains: `cookies.go` — set/clear cookie functions, secure request detection

### Frontend (web/src/)

**`web/src/api/`:**
- Purpose: TanStack Query definitions for each API domain
- Contains: Named exports with `queryOptions()` and `mutationOptions()` wrappers
- Files: `auth.ts`, `contracts.ts`, `customers.ts`, `projects.ts`, `time-entries.ts`, `units.ts`

**`web/src/routes/`:**
- Purpose: TanStack Router file-based routing
- Structure:
  - `__root.tsx`: Root layout with ThemeProvider, TooltipProvider, Toaster
  - `_auth.tsx`: Auth layout group (login, register, password-reset, invite, bootstrap)
  - `_authenticated.tsx`: Protected route guard — fetches profile, redirects to `/login` on failure
  - `_auth/login/`, `_auth/register/`, `_auth/password-reset/`, `_auth/invite/`, `_auth/bootstrap/`
  - `_authenticated/`: Dashboard, time-entries, projects, contracts, customers, org-hierarchy

**`web/src/components/`:**
- Purpose: Reusable UI components
- Structure:
  - `ui/`: shadcn/ui primitives (button, card, dialog, input, sidebar, etc.)
  - `layout/`: App-shell, sidebar, body, header
  - `app/`: Profile menu, organization switcher

**`web/src/lib/`:**
- Purpose: Shared library code
- Key files:
  - `api.ts`: HTTP client with cookie auth + 401 auto-refresh
  - `query-client.ts`: Shared TanStack Query client (retry: false, staleTime: 30000)
  - `utils.ts`: Utility functions (cn classname merging, etc.)

**`web/src/types/`:**
- Purpose: TypeScript type definitions matching the backend API contract
- Files:
  - `models.ts`: Domain model types (User, Organization, Contract, Project, TimeEntry, etc.)
  - `api.ts`: API request/response types (AuthResponse, LoginRequest, etc.)
  - `unit.ts`: Unit-specific types
  - `index.ts`: Re-exports from models and api

**`web/src/hooks/`:**
- Purpose: Custom React hooks
- Key files: `use-mobile.ts` — responsive breakpoint detection

## Key File Locations

**Entry Points:**
- `cmd/server/main.go`: Backend server entry and route wiring
- `cmd/migrate/main.go`: Database migration CLI
- `web/src/main.tsx`: Frontend entry point (React root, router, QueryClientProvider)

**Configuration:**
- `go.mod`: Go module definition and dependencies
- `web/vite.config.ts`: Vite dev server config (port 3000, proxy /api → :8080, path alias @/)
- `web/vitest.config.ts`: Frontend test config
- `web/playwright.config.ts`: E2E test config
- `Dockerfile`: Multi-stage Docker build (Go builder → alpine runtime)
- `docker-compose.yml`: PostgreSQL + app services
- `Makefile`: Build, test, run, migrate, docker commands
- `package.json`: Root npm scripts (likely for project-level tooling)

**Core Logic:**
- `internal/core/services/auth/auth.go`: All auth business logic (register, login, refresh, logout, bootstrap, profile, memberships)
- `internal/core/services/time_entry/time_entry.go`: Time entry business logic (CRUD, submit, approve, reject)
- `internal/adapters/primary/http/auth.go`: Auth HTTP handler (thin adapter)
- `internal/auth/auth.go`: JWT token service + bcrypt password hashing
- `internal/middleware/middleware.go`: Auth middleware + context helpers

**Testing:**
- `internal/core/services/testdata/`: Test factories and mocks
- `internal/adapters/secondary/postgres/exported_test_helpers.go`: Test DB schema setup
- `web/src/api/__tests__/`: Frontend API module unit tests
- `web/src/lib/__tests__/`: Library unit tests (api, validation)
- `web/e2e/`: Playwright E2E tests

## Naming Conventions

**Files:**
- Go files: `snake_case.go` (e.g., `time_entry.go`, `project_repository.go`, `password_reset.go`)
- TypeScript/React files: `kebab-case.ts` and `kebab-case.tsx` (e.g., `app-shell.tsx`, `status-badge.tsx`, `query-client.ts`)
- SQL migrations: `{number}_{name}.up.sql` and `{number}_{name}.down.sql`

**Go Packages:**
- Package names match directory names (e.g., `package time_entry`, `package postgres`, `package auth`)
- HTTP handlers use the same package as other primary adapters: `package http`
- Domain packages: `package auth` (within `core/domain/auth/`)

**Go Types/Functions:**
- Services: `Service` struct, `NewService(...)` constructor, methods with `(s *Service)` receiver
- Handlers: `*Handler` struct, `New*Handler(...)` constructor, methods with `(h *Handler)` receiver
- Repositories: `*Repository` struct, `New*Repository(...)` constructor, methods with `(r *Repository)` receiver
- Port interfaces: `*Repository` interface name, `TokenService`, `PasswordHasher`

**Routes:**
- Backend API routes: kebab-case (e.g., `/time-entries`, `/working-groups`, `/auth/password-reset/request`)
- Frontend file-based routes: same kebab-case directory names (e.g., `time-entries/`, `org-hierarchy/`)

**Frontend Conventions:**
- Components: PascalCase exports (`AppShell`, `LoginForm`, `TimeEntriesPage`)
- API module files: Named exports with `*Apis` namespace (e.g., `AuthApis`, `TimeEntriesApis`)
- API functions: camelCase query/mutation option functions (`profileQueryOpts`, `loginMutationOpts`)
- Query keys: `['domain', ...qualifiers]` (e.g., `['time-entries', 'monthly', month, year]`)
- Test files: `*.test.ts` or `*.spec.ts` co-located in `__tests__/` directories

## Where to Add New Code

**New Feature (Backend):**
1. Define domain entities in `internal/core/domain/{feature}/`
2. Define port interfaces in `internal/core/ports/{feature}_repository.go`
3. Implement business logic in `internal/core/services/{feature}/`
4. Create PostgreSQL repository in `internal/adapters/secondary/postgres/{feature}_repository.go`
5. Create HTTP handler in `internal/adapters/primary/http/{feature}.go`
6. Wire dependencies and register routes in `cmd/server/main.go`

**New Feature (Frontend):**
1. Define API types in `web/src/types/` (api.ts and/or models.ts)
2. Create API module in `web/src/api/{feature}.ts`
3. Create route directory in `web/src/routes/_authenticated/{feature}/`
4. Create components in `web/src/routes/_authenticated/{feature}/-components/`
5. Add sidebar navigation link in `web/src/components/layout/sidebar.tsx`

**New Component/Module:**
- Backend shared utility: `internal/` (new directory if cross-cutting)
- Frontend reusable UI: `web/src/components/ui/` (shadcn/ui pattern)
- Frontend layout component: `web/src/components/layout/`
- Frontend app-specific component: `web/src/components/app/`

**Utilities:**
- Go shared helpers: `pkg/` or `internal/` with appropriate package
- Frontend helpers: `web/src/lib/`
- Test helpers (Go): `internal/core/services/testdata/` for factories
- Test helpers (Go, Postgres): `internal/adapters/secondary/postgres/exported_test_helpers.go`

**Database:**
- Add `.up.sql` and `.down.sql` files in `migrations/` with sequential numbering
- Run via `go run ./cmd/migrate -up -dir migrations`

## Special Directories

**`uploads/`:**
- Purpose: Receipt file uploads for expense items
- Generated: Runtime data
- Committed: No
- Created in Dockerfile: `RUN mkdir -p /app/uploads/receipts`

**`bin/` and `server`/`main`/`migrate` binaries:**
- Purpose: Compiled Go binaries
- Generated: Yes (via `go build` or `make build`)
- Committed: No (in `.gitignore`)

**`internal/core/domain/expense/`:**
- Purpose: Domain entities for expenses
- Status: Empty directory — expense domain models are defined in `internal/models/models.go` instead

---

*Structure analysis: 2026-06-08*
