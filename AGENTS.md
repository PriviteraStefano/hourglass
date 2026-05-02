# Hourglass Codebase Guide for AI Agents

**Hourglass** is a time entry and expense tracking system with approval workflows for organizations. It's a full-stack TypeScript/Go application with a React frontend and Go backend.

## Architecture Overview

### Tech Stack
- **Backend**: Go 1.26.1, standard library HTTP server, hexagonal services in `internal/core/services/*`, primary HTTP adapters in `internal/adapters/primary/http/*`, and SurrealDB adapters in `internal/adapters/secondary/surrealdb/*`
- **Frontend**: React 19, TanStack Router v1, TanStack React Query v5, Vite, TypeScript, Tailwind CSS
- **Database**: SurrealDB for application data, plus PostgreSQL SQL migrations via `cmd/migrate`
- **Auth**: JWT-based authentication with HttpOnly `auth_token`/`refresh_token` cookies and bcrypt password hashing

### Key Data Flows

**Authentication Flow**: User registers/logs in → `internal/adapters/primary/http/auth.go` → `internal/core/services/auth.Service` → JWTs stored in HttpOnly `auth_token`/`refresh_token` cookies → `web/src/lib/api.ts` sends credentials and retries `POST /auth/refresh` on 401.

**Auth Hydration**: `web/src/routes/_authenticated.tsx` calls `AuthApis.profileQueryOpts` (`GET /auth/me`) before protected routes render; `web/src/api/auth.ts` also exposes `GET /auth/memberships` for organization switching.

**Time Entry & Expense Workflow**: Employee creates entries (draft) → Submits → Manager reviews → Finance reviews → Approved/Rejected. Each step tracked in approval history with role-based routing.

**Organization Model**: Users belong to organizations with role-based access (employee, manager, finance, customer). Contracts and Projects are either organization-specific or shared across orgs.

### Directory Structure

```
backend:
  cmd/server/main.go           # Server entry and route wiring
  cmd/migrate/main.go          # PostgreSQL migration CLI
  cmd/schema/main.go           # SurrealDB schema loader for schema/*.surql
  internal/auth/               # JWT, password hashing
  internal/core/               # Domain, ports, and application services
  internal/adapters/
    primary/http/              # Thin HTTP adapters (auth, project, time-entry, etc.)
    secondary/surrealdb/       # SurrealDB repositories and driven adapters
  internal/db/                 # SurrealDB connection plus legacy Postgres DB helpers
  internal/handlers/           # Health handler and legacy glue
  internal/models/             # Data structures, constants (Role, Status, Governance)
  internal/middleware/         # Auth middleware wrapper
  pkg/api/                     # Shared response format
  migrations/                  # SQL migrations for cmd/migrate
  schema/                      # SurrealDB schema files (*.surql)

frontend:
  web/src/
    api/auth.ts               # Auth query/mutation options (profile/login/register/logout/refresh/bootstrap)
    routes/                    # TanStack Router file-based routing
      __root.tsx              # Root layout
      _authenticated.tsx       # Protected route guard
      (auth)/                 # Login/register routes
      _authenticated/         # Protected pages (time-entries, etc.)
    lib/api.ts                # HTTP client with cookie auth + refresh-on-401
    hooks/                    # Domain hooks (useProjects, useTimeEntries, etc.)
    components/               # Reusable UI (shadcn-based, Tailwind)
    types/api.ts              # Shared API type definitions
    lib/query-client.ts       # Shared TanStack Query client used in main.tsx
```

## Critical Developer Workflows

### Local Development
```bash
# Backend
cd /Users/stefanoprivitera/Projects/hourglass
go run ./cmd/server           # Runs on :8080, connects to SurrealDB via SURREALDB_URL

# SurrealDB schema bootstrap (separate terminal)
go run ./cmd/schema           # Applies schema/*.surql to SurrealDB

# Postgres migrations (only when needed)
go run ./cmd/migrate -up -dir migrations

# Frontend (separate terminal)
cd web
bun install
bun run dev                   # Runs on :3000, proxies /api to :8080

# Docker
docker-compose up             # Starts surrealdb + app; add --profile postgres for the Postgres service
```

### Migrations
- `cmd/migrate/main.go` applies `migrations/*.up.sql` and `*.down.sql` against PostgreSQL via `DATABASE_URL`
- `cmd/schema/main.go` applies `schema/*.surql` to SurrealDB via `SURREALDB_URL`, `SURREALDB_USER`, `SURREALDB_PASS`, `SURREALDB_NS`, and `SURREALDB_DB`
- `docker-compose up` starts `surrealdb` and `app`; `postgres` is profile-gated (`docker-compose --profile postgres up`) for migration work
- Each SQL migration has `.up.sql` and `.down.sql` files

### Testing & Building
```bash
make build                    # Compiles Go binary to bin/hourglass
make test                     # Runs go test -v ./...
make docker-build             # Builds multi-stage Docker image

cd web && bun run build       # Type-checks and builds the frontend
cd web && bun run lint        # Runs ESLint
cd web && bunx playwright test # Runs Playwright e2e tests in web/e2e
```

## Project-Specific Patterns

### Handler Pattern (Backend)
Feature handlers live in `internal/adapters/primary/http/` and stay thin, delegating business logic to `internal/core/services/*`:
```go
type TimeEntryHandler struct {
  service *tesvc.Service
}
func NewTimeEntryHandler(service *tesvc.Service) *TimeEntryHandler { ... }
func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
```
Handlers are registered with `http.ServeMux` in `cmd/server/main.go` using the Go 1.22+ pattern: `mux.HandleFunc("POST /time-entries", handler)`.

### API Response Format (Backend)
JSON responses use a shared envelope with either `data` (success) or `error` (failure) from `pkg/api/response.go`:
```go
type Response struct {
  Data   interface{} `json:"data,omitempty"`
  Error  string      `json:"error,omitempty"`
  Status int         `json:"-"`
}
```

### Protected Routes (Frontend)
Routes are protected in `web/src/routes/_authenticated.tsx` via `beforeLoad` that hydrates auth state through React Query and redirects on failure:
```typescript
export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context: { client } }) => {
    try {
      await client.fetchQuery(AuthApis.profileQueryOpts)
    } catch {
      throw redirect({ to: '/login' })
    }
  },
})
```

### React Query Patterns (Frontend)
- `web/src/main.tsx` imports the shared `queryClient` from `web/src/lib/query-client.ts` and passes it to both `QueryClientProvider` and router context (`context.client`)
- `web/src/api/auth.ts` defines auth calls with `queryOptions`/`mutationOptions` consumed by route loaders and mutations, including refresh/bootstrap/memberships helpers
- `web/src/lib/query-client.ts` exports the shared client (`retry: false`, `staleTime: 30000`, `refetchOnWindowFocus: false`) used in `main.tsx`
- All API calls use the `api<T>()` helper which includes `credentials: 'include'` and auto-retries once through `POST /auth/refresh` on 401
- Mutations invalidate relevant queries: `queryClient.invalidateQueries({ queryKey: ['time-entries'] })`

### Approval Workflow Model (Backend)
Entries have `status` (draft → submitted → pending_manager → pending_finance → approved/rejected) and `currentApproverRole`. Approval history is immutable in `*_approvals` tables. See `TimeEntryApproval`, `ExpenseApproval` in models.go.

## Important Integration Points

### Frontend-Backend Contract
- Backend returns `Content-Type: application/json` with `{ data: ... }` on success and `{ error: ... }` on failures
- 401 status first triggers cookie refresh in `web/src/lib/api.ts`; if refresh fails, it redirects to `/login`
- `web/src/api/auth.ts` expects `GET /auth/me` and `GET /auth/memberships`, and both routes are registered in `cmd/server/main.go`
- API base URL comes from `VITE_API_URL` or defaults to `/api` (proxied to `http://localhost:8080` in Vite dev)

### Database Initialization
- Application data uses SurrealDB (`SURREALDB_URL`, `SURREALDB_USER`, `SURREALDB_PASS`, `SURREALDB_NS`, `SURREALDB_DB`); the local default is `ws://localhost:8000/rpc`
- `schema/001_schema.surql` is the SurrealDB bootstrap schema applied by `cmd/schema`
- Postgres still exists for `cmd/migrate` and uses `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` by default
- `docker-compose.yml` seeds SurrealDB with `root`/`root`; the Postgres service is profile-gated

### Environment Variables
**Backend** (`cmd/server/main.go`, `cmd/schema/main.go`, `cmd/migrate/main.go`):
- `SURREALDB_URL` - SurrealDB RPC endpoint (defaults to `ws://localhost:8000/rpc`)
- `SURREALDB_USER` - SurrealDB user (defaults to `root`)
- `SURREALDB_PASS` - SurrealDB password (defaults to `root`)
- `SURREALDB_NS` - SurrealDB namespace (defaults to `hourglass`)
- `SURREALDB_DB` - SurrealDB database (defaults to `main`)
- `SCHEMA_DIR` - Directory of `.surql` files for `cmd/schema` (defaults to `schema`)
- `DATABASE_URL` - PostgreSQL connection string for `cmd/migrate` (defaults to local hourglass DB)
- `JWT_SECRET` - Token signing key (defaults to "dev-secret-change-in-production")
- `PORT` - Server port (defaults to `:8080`)
- `ALLOWED_ORIGINS` - Comma-separated CORS allowlist (defaults to `http://localhost:3000`)

**Frontend** (web/vite.config.ts):
- `VITE_API_URL` - Backend base URL (defaults to `/api`, proxied to `http://localhost:8080` in dev)

## Key Models & Constants to Know

**Roles**: `employee`, `manager`, `finance`, `customer` (enforced in DB CHECK constraint)

**Entry Status**: `draft`, `submitted`, `pending_manager`, `pending_finance`, `approved`, `rejected`

**Governance Models**: `creator_controlled`, `unanimous`, `majority` (for approval rules on contracts/projects)

**Project Types**: `billable`, `internal`

**Expense Categories**: `mileage`, `meal`, `accommodation`, `other`

**Time Entry Structure**: `TimeEntry` (header) contains `[]TimeEntryItem` (line items with hours per project)

**Approval Actions**: `submit`, `approve`, `reject`, `edit_approve`, `edit_return`, `partial_approve`, `delegate`

## File Conventions

- Handlers use `*Handler` receiver (e.g., `(h *UserHandler)`)
- Models use `uuid.UUID` for IDs and `time.Time` with timezone awareness
- Routes use kebab-case (e.g., `/time-entries`)
- Frontend uses TanStack Router file-based routing with `index.tsx` for folder routes
- Frontend component files are primarily kebab-case (e.g., `app-shell.tsx`, `status-badge.tsx`)
- Types coexist in `models.go` and `types/api.ts` for frontend

## Hexagonal Architecture

This project is using hexagonal (ports & adapters) architecture for the main application flow. See `plans/hexagonal-migration.md` for details.

**When creating new features or refactoring handlers:**
1. Read `plans/hexagonal-migration.md` for the target structure
2. Follow the migration pattern: domain → ports → service → adapters → wiring
3. Keep business logic in `internal/core/services/`, not in handlers
4. Handlers in `internal/adapters/primary/http/` should be thin

## graphify

This project has a graphify knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `python3 -c "from graphify.watch import _rebuild_code; from pathlib import Path; _rebuild_code(Path('.'))"` to keep the graph current
