# Architecture

## Hexagonal (Ports & Adapters) Backend

Hourglass's Go backend follows the hexagonal architecture pattern. Business
logic lives in the **inner core** (domain + services), while external concerns
(HHTP, database) are handled by **adapters** that implement **port** interfaces.

```
┌─────────────────────────────────────────────────────┐
│                    Primary Adapters                   │
│              internal/adapters/primary/http/           │
│  (Thin HTTP handlers — parse request, delegate to      │
│   service, format response)                           │
└──────────┬──────────────────────────────────────────┘
           │ calls
           ▼
┌─────────────────────────────────────────────────────┐
│                  Core / Application                   │
│                                                       │
│  internal/core/domain/    internal/core/ports/        │
│  (pure domain models)    (service & repo interfaces)  │
│                                                       │
│  internal/core/services/                              │
│  (business logic implementations)                     │
└──────────┬──────────────────────────────────────────┘
           │ implements
           ▼
┌─────────────────────────────────────────────────────┐
│                   Secondary Adapters                  │
│            internal/adapters/secondary/postgres/       │
│  (PostgreSQL repository implementations)              │
└─────────────────────────────────────────────────────┘
```

### Layer Breakdown

#### Domain (`internal/core/domain/`)

One subdirectory per bounded context:

- `auth/` — Authentication domain models
- `contract/` — Contract-related models
- `customer/` — Customer models
- `expense/` — Expense domain models
- `invitation/` — Invitation models
- `organization/` — Organization models
- `password_reset/` — Password reset models
- `activity/` — Activity models (single recursive work entity; replaced the old project/subproject domains)
- `time_entry/` — Time entry models
- `unit/` — Unit (org hierarchy) models
- `working_group/` — Working group models

Each domain directory contains pure Go structs without external dependencies.

#### Ports (`internal/core/ports/`)

Interface definitions that the core requires from the outside world:

- `*_repository.go` — Persistence interfaces (e.g., `TimeEntryRepository`, `ExpenseRepository`, `UserRepository`,
  `ProjectRepository`, `ContractRepository`)
- `token_service.go` — Token generation/validation
- `password_hasher.go` — Password hashing interface
- `user_finder.go` — User lookup interface
- `errors.go` — Common error types (`ErrNotFound`, `ErrConflict`)

#### Services (`internal/core/services/`)

Business logic implementations, one directory per domain:

- `auth/` — Registration, login, logout, refresh, bootstrap, org switching
- `time_entry/` — Time entry CRUD, submission, approval workflow
- `expense/` — Expense CRUD, submission, approval workflow
- `contract/` — Contract CRUD, adoption, mileage recalculation
- `project/` — Project CRUD, adoption, manager management
- `customer/` — Customer CRUD
- `organization/` — Org CRUD, invitations, member management, settings
- `unit/` — Unit CRUD, hierarchy tree, members
- `working_group/` — Working group CRUD, members
- `export/` — Date-range CSV/XLSX export of approved entries
- `invitation/` — Invitation CRUD, validation, acceptance
- `password_reset/` — Password reset request/verify flow

Each service follows this pattern:

```go
type Service struct {
    repo ports.SomeRepository
}

func NewService(repo ports.SomeRepository) *Service { ... }
func (s *Service) DoSomething(ctx context.Context, params ...) (Result, error) { ... }
```

#### Primary HTTP Adapters (`internal/adapters/primary/http/`)

Thin HTTP handlers that:

1. Parse the request (path params, query params, JSON body)
2. Extract auth context (user ID, org ID, role) from middleware
3. Call the service layer
4. Format the response using the shared envelope (`pkg/api`)

Each handler file follows this pattern:

```go
type SomeHandler struct {
    service *some.Service
}

func NewSomeHandler(service *some.Service) *SomeHandler { ... }
func (h *SomeHandler) List(w http.ResponseWriter, r *http.Request) { ... }
```

#### Secondary PostgreSQL Adapters (`internal/adapters/secondary/postgres/`)

Repository implementations that connect to PostgreSQL via `pgx/v5` pool. Each
repository implements the corresponding port interface.

#### Shared Models (`internal/models/models.go`)

Shared data structures used across the application:

- **Enums with validation**: `Role`, `GovernanceModel`, `ProjectType`, `EntryStatus`, `ExpenseCategory`,
  `ApprovalAction`
- **Core entities**: `User`, `Organization`, `OrganizationMembership`, `Contract`, `Customer`, `Project`, `TimeEntry`,
  `Expense`, `Unit`, `WorkingGroup`
- **Aggregates**: `UserWithMembership`, `TimeEntryMonthlySummary`, `PendingEntryGroup`
- **Request DTOs**: `CreateProjectRequest`, `UpdateTimeEntryRequest`, `SubmitRequest`, `RejectRequest`, etc.

### Dependency Injection

Wiring happens in `cmd/server/main.go`. The pattern:

```go
repo := postgres.NewSomeRepository(pool)
service := some.NewService(repo)
handler := http.NewSomeHandler(service)
mux.HandleFunc("GET /some-resource", middleware.Auth(authService, handler.List))
```

### Middleware

- **Auth** (`internal/middleware/middleware.go`): Validates JWT from `auth_token` cookie, injects `UserID`,
  `OrganizationID`, `Role`, `Email` into request context
- **TryAuth**: Optional auth — sets context if token present, continues without if missing
- **RequireRole**: Role-check middleware for fine-grained access control
- **CORS** (`internal/middleware/cors.go`): Configurable CORS headers
- **Rate Limiting** (`internal/middleware/ratelimit.go`): Per-route token bucket rate limiter
- **Version** (`internal/middleware/version.go`): App version header
- **Logging** (`internal/middleware/logging_test.go`): Request logging

### Response Envelope

All API responses follow the format in `pkg/api/response.go`:

```json
// Success
{ "data": { ... } }

// Error
{ "error": "message" }
```

---

## Frontend Architecture

### Stack

- **React 19** with TypeScript
- **TanStack Router v1** — File-based routing with auto-generated `routeTree.gen.ts`
- **TanStack React Query v5** — Server state management with query/mutation options
- **Vite** — Build tool with dev server proxying `/api` → `:8080`
- **Tailwind CSS v4** — Utility-first styling
- **shadcn/ui** — Component library (radix-based primitives)
- **Zustand** — Client state (org switching)

### Data Flow

```
Page Component
    │
    ├── Route `loader` calls client.ensureQueryData(queryOpts)
    │   (pre-fetches data before render)
    │
    ├── Component uses useSuspenseQuery(queryOpts)
    │   (consumes cached/queried data)
    │
    └── useMutation(mutationOpts) + onSuccess invalidation
        (triggers writes, auto-refreshes related queries)
```

### API Client (`web/src/lib/api.ts`)

- Thin `api<T>(path, options)` wrapper around `fetch`
- Automatic 401 → token refresh via `POST /auth/refresh`
- Unwraps `{ data: T }` response envelopes
- Guards against refresh loops on auth endpoints
- Throws `UnauthorizedError` when refresh fails permanently

### Query Options Pattern (`web/src/api/*.ts`)

Each resource exports:

```typescript
export const SomeApis = {
  listQueryOpts: queryOptions({ ... }),
  getQueryOpts: (id: string) => queryOptions({ ... }),
  createMutationOpts: mutationOptions({ ... }),
  updateMutationOpts: mutationOptions({ ... }),
  deleteMutationOpts: mutationOptions({ ... }),
}
```

### Route Layout Hierarchy

```
__root (ThemeProvider > TooltipProvider > Outlet > Toaster)
├── _auth (unauthenticated-only — redirects to / if logged in)
│   ├── /login
│   ├── /register
│   ├── /password-reset
│   ├── /invite
│   └── /bootstrap
└── _authenticated (requires auth — redirects to /login on 401)
    └── AppShell (SidebarProvider > AppSidebar > Outlet)
        ├── / (Today landing page — read-only)
        ├── /time-entries (List | Calendar | Export tabs)
        ├── /expenses (Calendar + detail side-by-side)
        ├── /activities, /activities/$id
        ├── /working-groups
        ├── /approvals (Manager | Finance stage tabs)
        ├── /contracts, /contracts/$id
        ├── /customers, /customers/$id
        ├── /org-hierarchy
        └── /exports
```

### Sidebar & Role-Scoped Visibility

The app sidebar (`web/src/components/layout/sidebar.tsx`) groups navigation into
pillar groups — **Today** (ungrouped), **Track**, **Work**, **People**,
**Economics**, **Review**, **Reports**, **Admin** — per ADR-P-011. Group
visibility is scoped by role via `web/src/lib/role-visibility.ts`:

- **Review** (Approvals) renders only for users holding an approval stage —
  org-role `manager`/`finance` plus anyone who is a working-group manager or
  delegate; HR never sees it.
- **Economics** is hidden from `employee` and `customer`.
- **Admin** is hidden from every v0.1 role (no org-admin role exists yet).

These helpers are UX scoping only; every role-restricted surface stays
backend-gated. Authenticated pages render inside the shared `Header` + `Body`
shell (`web/src/components/layout/index.ts`).

### Auth Guard Pattern

The `_authenticated.tsx` layout's `beforeLoad` fetches the user profile via
`AuthApis.profileQueryOpts`. On failure, it clears the auth cache and
redirects to `/login`. The `_auth.tsx` layout does the opposite — if profile
fetch succeeds, it redirects to `/`.

### Component Organization

- **Shared UI** (`web/src/components/ui/`): shadcn/ui primitives (button, dialog, card, etc.)
- **Layout** (`web/src/components/layout/`): AppShell, sidebar, header, body, route error boundary
- **Shared entries** (`web/src/components/shared/`): `EntriesTable`, `StatusBadge`, `EntriesFilters` (used by the time-entries and expenses list views)
- **Approval** (`web/src/components/approval/`): `ApprovalButtons`, `ApprovalHistory`
- **Exports** (`web/src/components/exports/`): `ExportForm` (shared between time-entries and expenses)
- **Route-local** (`web/src/routes/_authenticated/time-entries/-components/`): Components co-located with their routes

---

## Key Source Files

| Purpose                 | File                                         |
|-------------------------|----------------------------------------------|
| Server entry + DI       | `/cmd/server/main.go`                        |
| All port interfaces     | `/internal/core/ports/*.go`                  |
| Service implementations | `/internal/core/services/*/services.go`      |
| HTTP handlers           | `/internal/adapters/primary/http/*.go`       |
| PostgreSQL repos        | `/internal/adapters/secondary/postgres/*.go` |
| JWT auth                | `/internal/auth/auth.go`                     |
| Auth middleware         | `/internal/middleware/middleware.go`         |
| Models + enums          | `/internal/models/models.go`                 |
| Response envelope       | `/pkg/api/response.go`                       |
| DB connection pool      | `/internal/db/pgpool.go`                     |
| Full schema             | `/migrations/000_full_schema.up.sql`         |
| Frontend entry          | `/web/src/main.tsx`                          |
| HTTP client             | `/web/src/lib/api.ts`                        |
| API modules             | `/web/src/api/*.ts`                          |
| Route tree              | `/web/src/routes/`                           |
