# Hourglass Codebase Guide for AI Agents

**Hourglass** is a time entry and expense tracking system with approval workflows for organizations. It's a full-stack TypeScript/Go application with a React frontend and Go backend.

## Architecture Overview

### Tech Stack
- **Backend**: Go 1.26.1, PostgreSQL, standard library HTTP server
- **Frontend**: React 19, TanStack Router v1, TanStack React Query v5, Vite, TypeScript, Tailwind CSS
- **Database**: PostgreSQL with custom migration system
- **Auth**: JWT-based authentication with bcrypt password hashing

### Key Data Flows

**Authentication Flow**: User registers/logs in → `UserHandler.Register/Login` → `auth.Service.GenerateToken` → JWT token stored in localStorage → Frontend uses Bearer token in API requests.

**Time Entry & Expense Workflow**: Employee creates entries (draft) → Submits → Manager reviews → Finance reviews → Approved/Rejected. Each step tracked in approval history with role-based routing.

**Organization Model**: Users belong to organizations with role-based access (employee, manager, finance, customer). Contracts and Projects are either organization-specific or shared across orgs.

### Directory Structure

```
backend:
  cmd/server/main.go           # Server entry, route definitions
  internal/auth/               # JWT, password hashing
  internal/db/                 # PostgreSQL connection & pooling
  internal/handlers/           # HTTP request handlers (user, org, time-entry, expense, approval)
  internal/models/             # Data structures, constants (Role, Status, Governance)
  internal/middleware/         # Auth middleware wrapper
  pkg/api/                     # Shared response format
  migrations/                  # SQL migrations (numbered sequence)

frontend:
  web/src/
    api/auth.ts               # Auth query/mutation options (profile/login/register/logout)
    routes/                    # TanStack Router file-based routing
      __root.tsx              # Root layout
      _authenticated.tsx       # Protected route guard
      (auth)/                 # Login/register routes
      _authenticated/         # Protected pages (time-entries, etc.)
    lib/api.ts                # HTTP client with auto auth header injection
    hooks/                    # Domain hooks (useProjects, useTimeEntries, etc.)
    components/               # Reusable UI (shadcn-based, Tailwind)
    types/api.ts              # Shared API type definitions
    lib/query-client.ts       # TanStack Query configuration
```

## Critical Developer Workflows

### Local Development
```bash
# Backend
cd /Users/stefanoprivitera/Projects/hourglass
go run ./cmd/server           # Runs on :8080, connects to postgres://localhost:5432

# Frontend (separate terminal)
cd web
npm install
npm run dev                   # Runs on :3000, proxies /api to :8080

# Database (Docker)
docker-compose up             # Starts PostgreSQL on :5432
```

### Migrations
- Migrations are SQL files in `/migrations/` (numbered: 001_init, 002_contracts_projects, etc.)
- Local Docker bootstraps schema by mounting `/migrations` into Postgres init (`docker-entrypoint-initdb.d` in `docker-compose.yml`)
- `make migrate-up`/`make migrate-down` targets call `go run ./cmd/migrate`, but `cmd/migrate` is not present in this repo snapshot
- Each migration has `.up.sql` and `.down.sql` files

### Testing & Building
```bash
make build                    # Compiles Go binary to bin/hourglass
make test                     # Runs go test ./...
make docker-build             # Builds multi-stage Docker image
```

## Project-Specific Patterns

### Handler Pattern (Backend)
Each feature has a `*Handler` struct with dependency injection of `*sql.DB` and service dependencies:
```go
type TimeEntryHandler struct {
  db *sql.DB
}
func NewTimeEntryHandler(db *sql.DB) *TimeEntryHandler { ... }
func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
```
Handlers are registered with http.ServeMux in main.go using the new Go 1.22+ pattern: `mux.HandleFunc("POST /time-entries", handler)`.

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
- `web/src/main.tsx` creates `new QueryClient()` inline and passes it to both `QueryClientProvider` and router context (`context.client`)
- `web/src/api/auth.ts` defines auth calls with `queryOptions`/`mutationOptions` consumed by route loaders and mutations
- `web/src/lib/query-client.ts` exports a preset client (`retry: false`, `staleTime: 30000`, `refetchOnWindowFocus: false`) that is currently not wired in `main.tsx`
- All API calls use the `api<T>()` helper which auto-injects Bearer token
- Mutations invalidate relevant queries: `queryClient.invalidateQueries({ queryKey: ['time-entries'] })`

### Approval Workflow Model (Backend)
Entries have `status` (draft → submitted → pending_manager → pending_finance → approved/rejected) and `currentApproverRole`. Approval history is immutable in `*_approvals` tables. See `TimeEntryApproval`, `ExpenseApproval` in models.go.

## Important Integration Points

### Frontend-Backend Contract
- Backend returns `Content-Type: application/json` with `{ data: ... }` on success and `{ error: ... }` on failures
- 401 status auto-redirects to `/login` (handled in `api.ts`)
- Bearer token in `Authorization: Bearer <token>` header
- API base URL from `VITE_API_URL` env or defaults to `http://localhost:8080`
- `web/src/api/auth.ts` expects `GET /auth/me` for profile hydration, but that route is not currently registered in `cmd/server/main.go`

### Database Initialization
- Uses PostgreSQL with UUID primary keys (`gen_random_uuid()`)
- Connection string format: `postgres://user:password@host:port/database?sslmode=disable`
- Default dev credentials in `docker-compose.yml`: user=`hourglass`, password=`hourglass`
- Indexes created on foreign key columns for queries (e.g., `idx_organization_memberships_user_id`)

### Environment Variables
**Backend** (cmd/server/main.go):
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Token signing key (defaults to "dev-secret-change-in-production")
- `PORT` - Server port (inferred from Docker/main.go setup, defaults :8080)

**Frontend** (web/vite.config.ts):
- `VITE_API_URL` - Backend base URL (proxied via Vite dev server by default)

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

## graphify

This project has a graphify knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `python3 -c "from graphify.watch import _rebuild_code; from pathlib import Path; _rebuild_code(Path('.'))"` to keep the graph current
