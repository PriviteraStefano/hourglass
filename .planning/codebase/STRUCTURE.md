# Codebase Structure

**Analysis Date:** 2026-05-12

## Directory Layout

```
hourglass/
├── cmd/                    # CLI entry points
│   ├── server/main.go      # HTTP server entry
│   ├── migrate/main.go     # PostgreSQL migrations CLI
│   └── schema/main.go      # SurrealDB schema loader
├── internal/               # Application code
│   ├── auth/               # JWT service
│   ├── cookies/            # Cookie utilities
│   ├── core/               # Hexagonal core
│   │   ├── domain/         # Domain entities and rules
│   │   ├── ports/          # Interface definitions
│   │   └── services/       # Business logic
│   ├── adapters/
│   │   ├── primary/http/   # HTTP handlers
│   │   └── secondary/      # External adapters
│   │       └── surrealdb/ # SurrealDB repositories
│   ├── db/                 # Database connection
│   ├── handlers/          # Health check handler
│   ├── middleware/        # HTTP middleware
│   └── models/            # Shared models/constants
├── pkg/api/               # API response format
├── migrations/            # PostgreSQL migrations
├── schema/                # SurrealDB schema files
├── web/                   # Frontend React app
│   ├── src/
│   │   ├── api/           # API client functions
│   │   ├── components/    # React components
│   │   ├── hooks/         # Custom React hooks
│   │   ├── lib/           # Utilities
│   │   ├── routes/        # TanStack Router routes
│   │   └── types/         # TypeScript types
│   └── package.json
└── plans/                 # Planning documents
```

## Directory Purposes

**`cmd/`:**
- Purpose: Executable entry points
- Contains: `server`, `migrate`, `schema` commands
- Key files: `main.go` files for each command

**`internal/auth/`:**
- Purpose: JWT token service
- Contains: Token generation, validation, claims
- Key files: `auth.go`

**`internal/core/domain/`:**
- Purpose: Pure domain entities
- Contains: Business entities (User, TimeEntry, Project, Contract, etc.)
- Key files: `*/*.go` - one file per aggregate root

**`internal/core/ports/`:**
- Purpose: Interface definitions for external dependencies
- Contains: Repository interfaces (e.g., `TimeEntryRepository`)
- Key files: `*_repository.go`

**`internal/core/services/`:**
- Purpose: Business logic orchestration
- Contains: Service implementations per domain
- Key files: `*/service_name.go`

**`internal/adapters/primary/http/`:**
- Purpose: HTTP request/response handling
- Contains: HTTP handlers for each domain
- Key files: `auth.go`, `time_entry.go`, `project.go`, `contract.go`, etc.

**`internal/adapters/secondary/surrealdb/`:**
- Purpose: SurrealDB repository implementations
- Contains: Concrete implementations of port interfaces
- Key files: `*_repository.go`, `models.go`

**`internal/middleware/`:**
- Purpose: HTTP middleware
- Contains: Auth, CORS, logging, rate limiting
- Key files: `middleware.go`, `ratelimit.go`

**`internal/models/`:**
- Purpose: Shared constants and legacy models
- Contains: Role, Status enums; legacy data structures
- Key files: `models.go`

**`pkg/api/`:**
- Purpose: API response format utilities
- Contains: HTTP response helpers
- Key files: `response.go`

**`schema/`:**
- Purpose: SurrealDB schema definitions
- Contains: `.surql` schema files
- Key files: `001_schema.surql`, `002_seed_tcg.surql`

**`migrations/`:**
- Purpose: PostgreSQL migrations (legacy)
- Contains: `.up.sql` and `.down.sql` pairs
- Key files: `001_init.up.sql`, etc.

**`web/src/`:**
- Purpose: React frontend application
- Contains: All frontend code
- Structure: See Frontend section below

## Key File Locations

**Entry Points:**
- `cmd/server/main.go`: Backend HTTP server
- `cmd/schema/main.go`: SurrealDB schema bootstrap
- `web/src/main.tsx`: Frontend React entry

**Configuration:**
- `go.mod` / `go.sum`: Go dependencies
- `web/package.json`: Node dependencies
- `docker-compose.yml`: Local dev services
- `.env`: Environment variables (never committed)

**Core Logic:**
- Services: `internal/core/services/*/`
- Domain: `internal/core/domain/*/`
- Ports: `internal/core/ports/`

**Testing:**
- Backend tests: `*_test.go` alongside source files
- Frontend tests: `web/e2e/` for Playwright

## Frontend Structure (`web/src/`)

```
web/src/
├── api/                   # API client functions
│   ├── auth.ts           # Auth queries/mutations
│   ├── time-entries.ts   # Time entry API
│   ├── projects.ts       # Project API
│   ├── contracts.ts      # Contract API
│   ├── units.ts          # Unit API
│   └── customers.ts      # Customer API
├── components/
│   ├── ui/               # shadcn/ui components
│   ├── layout/           # AppShell, Header, Sidebar
│   └── app/              # Profile menu, Org switcher
├── hooks/                # Custom React hooks
├── lib/
│   ├── api.ts           # HTTP client with refresh
│   ├── query-client.ts  # TanStack Query client
│   └── utils.ts         # Utility functions
├── routes/               # TanStack Router file-based routing
│   ├── __root.tsx       # Root route
│   ├── _authenticated.tsx   # Protected route guard
│   ├── _auth/           # Login, register, password reset
│   │   ├── login/
│   │   ├── register/
│   │   ├── password-reset/
│   │   ├── invite/
│   │   └── bootstrap/
│   └── _authenticated/  # Protected pages
│       ├── index.tsx    # Dashboard
│       ├── time-entries/
│       ├── projects/
│       ├── contracts/
│       └── org-hierarchy/
├── types/
│   ├── api.ts           # API response types
│   ├── models.ts        # Domain models
│   └── index.ts         # Type exports
└── main.tsx             # Entry point
```

## Naming Conventions

**Files (Backend):**
- Go files: `lowercase.go` (e.g., `time_entry.go`, `auth.go`)
- Test files: `*_test.go`
- Models: `models.go` for shared, `*_repository.go` for repos

**Files (Frontend):**
- TypeScript/TSX: `kebab-case.tsx` (e.g., `time-entries-page.tsx`, `login-form.tsx`)
- Components: PascalCase in code, file-based routing uses kebab-case

**Directories:**
- Backend: `lowercase` with underscores for multi-word (e.g., `time_entry`, `password_reset`)
- Frontend: `kebab-case` (e.g., `time-entries`, `org-hierarchy`)

**Functions/Methods:**
- Go: `PascalCase` exported, `camelCase` unexported
- TypeScript: `camelCase`

**Types/Interfaces:**
- Go: `PascalCase` (e.g., `TimeEntry`, `CreateTimeEntryRequest`)
- TypeScript: `PascalCase` (e.g., `TimeEntry`, `CreateTimeEntryRequest`)

## Where to Add New Code

**New Backend Feature:**
- Domain: `internal/core/domain/{feature}/{feature}.go`
- Ports: `internal/core/ports/{feature}_repository.go`
- Service: `internal/core/services/{feature}/{feature}.go`
- Handler: `internal/adapters/primary/http/{feature}.go`
- Repository: `internal/adapters/secondary/surrealdb/{feature}_repository.go`
- Register in: `cmd/server/main.go`

**New Frontend Feature:**
- API client: `web/src/api/{feature}.ts`
- Routes: `web/src/routes/_authenticated/{feature}/`
- Components: `web/src/components/` or co-located in route folder
- Types: `web/src/types/api.ts`

**New Database Entity:**
- Schema: Add to `schema/001_schema.surql`
- Domain: `internal/core/domain/{entity}/`
- Repository: `internal/adapters/secondary/surrealdb/{entity}_repository.go`

## Special Directories

**`.planning/`:**
- Purpose: GSD planning artifacts
- Generated: Yes
- Committed: Yes (git-tracked planning docs)

**`graphify-out/`:**
- Purpose: Knowledge graph generated by graphify
- Generated: Yes (run via graphify skill)
- Committed: Yes

**`hourglass-vault/`:**
- Purpose: Encrypted secrets storage
- Generated: Yes (git-crypt)
- Committed: Encrypted (not readable in git)

**`bin/`:**
- Purpose: Compiled binaries
- Generated: Yes
- Committed: No (in `.gitignore`)

**`uploads/`:**
- Purpose: User-uploaded files
- Generated: At runtime
- Committed: No

---

*Structure analysis: 2026-05-12*