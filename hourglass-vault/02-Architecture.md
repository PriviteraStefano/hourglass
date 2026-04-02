# Architecture

## Tech Stack

### Backend
- **Language**: Go 1.26.1
- **HTTP Server**: Go standard library (`net/http`)
- **Database**: PostgreSQL with custom migration system
- **Auth**: JWT with bcrypt password hashing
- **Response Format**: JSON envelope with `{ data, error }` structure

### Frontend
- **Framework**: React 19
- **Router**: TanStack Router v1 (file-based routing)
- **Data Fetching**: TanStack React Query v5
- **Build Tool**: Vite
- **Language**: TypeScript
- **Styling**: Tailwind CSS + shadcn/ui components

### Database
- **System**: PostgreSQL 13+
- **Connection**: Pool-based via `pq` driver
- **Migrations**: Numbered SQL files (001_init.up.sql, etc.)
- **Schema**: UUID primary keys with foreign key constraints
- **Default Credentials** (dev): user=`hourglass`, password=`hourglass`

## System Architecture Diagram

```
┌─────────────────────────────────────────┐
│         Frontend (React/Vite)           │
├─────────────────────────────────────────┤
│  Routes (TanStack Router)               │
│  - /login, /register                    │
│  - /_authenticated/* (protected)        │
│  - /time-entries, /expenses, /reports   │
├─────────────────────────────────────────┤
│  API Client (lib/api.ts)                │
│  - Auto Bearer token injection          │
│  - 401 redirect to login                │
├─────────────────────────────────────────┤
│  React Query (QueryClient)              │
│  - Query caching, auto-refetch          │
│  - Mutation invalidation on changes     │
└──────────────┬──────────────────────────┘
               │ HTTP
               │ /api/* proxied to :8080
┌──────────────▼──────────────────────────┐
│      Backend (Go HTTP Server)           │
├─────────────────────────────────────────┤
│  Middleware                             │
│  - JWT Auth validation                  │
│  - API versioning (headers)             │
├─────────────────────────────────────────┤
│  Handlers (internal/handlers/)          │
│  - UserHandler      (auth, profile)     │
│  - OrgHandler       (org management)    │
│  - ContractHandler  (contracts)         │
│  - ProjectHandler   (projects)          │
│  - TimeEntryHandler (time tracking)     │
│  - ExpenseHandler   (expense mgmt)      │
│  - ApprovalHandler  (approvals)         │
│  - ExportHandler    (CSV exports)       │
│  - CustomerHandler  (contacts)          │
├─────────────────────────────────────────┤
│  Models & Services                      │
│  - Auth Service (JWT, password hashing) │
│  - Database pooling & connection mgmt   │
└──────────────┬──────────────────────────┘
               │ SQL
┌──────────────▼──────────────────────────┐
│    PostgreSQL Database                  │
├─────────────────────────────────────────┤
│  Tables (see 03-Database-Schema)        │
│  - users, organizations, memberships    │
│  - contracts, projects, adoptions       │
│  - time_entries, time_entry_items       │
│  - expenses, approvals                  │
│  - customers, organization_settings     │
└─────────────────────────────────────────┘
```

## Component Breakdown

### Backend Structure

**File Layout:**
```
cmd/
  server/main.go          # Entry point, route registration
internal/
  auth/
    service.go            # JWT token generation/validation
  db/
    connection.go         # PostgreSQL pooling
  handlers/
    user.go              # Auth & user mgmt endpoints
    organization.go      # Org settings & memberships
    time_entry.go        # Time entry CRUD
    expense.go           # Expense CRUD
    approval.go          # Approval workflows
    contract.go          # Contract management
    project.go           # Project management
    customer.go          # Customer contacts
    export.go            # CSV exports
    health.go            # Health check
  models/
    models.go            # Data structures & constants
  middleware/
    auth.go              # JWT validation wrapper
pkg/
  api/
    response.go          # Response envelope format
migrations/
  00X_*.up.sql          # Schema migrations
  00X_*.down.sql        # Rollback scripts
```

### Frontend Structure

**File Layout:**
```
web/src/
  main.tsx              # Entry point, QueryClient setup
  routes/
    __root.tsx          # Root layout
    _authenticated.tsx   # Protected route guard
    (auth)/             # Login/register pages
      login.tsx
      register.tsx
    _authenticated/     # Protected pages
      time-entries/
        index.tsx       # List view
        $id.tsx         # Detail/edit view
      expenses/
      contracts/
      projects/
      reports/
  api/
    auth.ts             # Auth queries/mutations
    time-entries.ts     # Time entry queries/mutations
    expenses.ts
    contracts.ts
    projects.ts
  lib/
    api.ts              # HTTP client with auth
    query-client.ts     # React Query defaults
  hooks/
    useProjects.ts      # Domain-specific hooks
    useTimeEntries.ts
    useExpenses.ts
  components/
    app-shell.tsx       # Main layout
    time-entry-form.tsx # Reusable forms
    status-badge.tsx    # Status displays
    (shadcn-ui components)
  types/
    api.ts              # API response types
```

## Request/Response Flow

### Typical Authenticated Request

```
Frontend:
  1. GET /api/time-entries
  2. lib/api.ts adds: Authorization: Bearer <JWT>
  3. Vite dev server proxies to http://localhost:8080

Backend:
  1. middleware.Auth validates JWT token
  2. Extracts user ID, adds to request context
  3. Handler processes request
  4. Queries database
  5. Returns: { data: [...], error: null }

Frontend:
  1. React Query receives response
  2. Caches data, triggers component re-render
  3. UI displays time entries
```

### Failed Auth

```
Backend returns: 401 Unauthorized
Frontend (api.ts):
  - Catches 401 status
  - Calls throw redirect({ to: '/login' })
  - User redirected to login page
```

## Database Connection Model

```
database.New(connStr)
  ↓
sql.Open("postgres", connStr)
  ↓
SetMaxOpenConns(25)
SetMaxIdleConns(5)
SetConnMaxLifetime(5 minutes)
  ↓
Returns *sql.DB (used by all handlers)
```

All handlers receive `*sql.DB` via dependency injection, enabling connection pooling.

## Auth Flow

See [[05-Auth-System]] for details. High-level:

```
1. User registers: POST /auth/register
2. Backend hashes password with bcrypt
3. User logs in: POST /auth/login
4. Backend validates, generates JWT
5. Frontend stores JWT in localStorage
6. All API requests include Bearer token
7. Backend validates token signature on every request
8. Token expires, frontend uses refresh endpoint
```

## Approval Workflow Architecture

See [[10-Time-Entries]] and [[11-Expenses]] for detailed flows.

**Key Design:**
- Each entry has immutable `status` field
- Approval history stored separately in `*_approvals` tables
- Status advances via handler methods (Submit, Approve, Reject, etc.)
- `currentApproverRole` tracks next reviewer

## API Design Principles

1. **RESTful routes**: `/time-entries`, `/contracts`, `/expenses`
2. **Standard methods**: GET (list/detail), POST (create), PUT (update)
3. **Consistent response format**: `{ data, error }` envelope
4. **Role-based filtering**: Queries exclude unapproved entries for employees
5. **Org-scoped**: All queries filter by organization context

## Shared Resource Model

Contracts and Projects can be "shared" (adopted by other orgs):

```
Org A creates Contract A
  ↓
Org B adopts Contract A
  ↓
Both can use Contract A and its Projects
  ↓
Linked via contract_adoptions, project_adoptions tables
```

This enables standardized contract/project templates across the company.

---

**Next**: [[03-Database-Schema]] for data model details, or [[04-Backend-Patterns]] for development patterns.
