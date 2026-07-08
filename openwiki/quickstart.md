# Hourglass — Quickstart

**Time entry and expense tracking with approval workflows for organizations.**

Hourglass is a full-stack application that lets organizations track employee time
entries and expenses through a configurable, role-based approval workflow
(employee → manager → finance). It ships with a Go backend (hexagonal
architecture) and a React frontend.

---

## Tech Stack

| Layer     | Technology                                                              |
|-----------|-------------------------------------------------------------------------|
| Backend   | Go 1.26.1, standard library `net/http`, hexagonal (ports & adapters)    |
| Frontend  | React 19, TanStack Router v1, TanStack React Query v5, Vite, TypeScript |
| Styling   | Tailwind CSS v4, shadcn/ui                                              |
| Database  | PostgreSQL 15                                                           |
| Auth      | JWT (`golang-jwt/jwt/v5`), bcrypt (`golang.org/x/crypto`)               |
| Testing   | `stretchr/testify`, `testcontainers-go`, Vitest, Playwright             |
| Container | Docker (multi-stage), docker-compose                                    |

---

## Getting Started

### Prerequisites

- **Go** >= 1.26.1
- **Node.js / Bun** (for the frontend)
- **PostgreSQL** >= 15 (or use the bundled docker-compose service)
- **Docker** + **docker-compose** (optional, for containerized runs)

### Docker (easiest)

```bash
docker-compose up
```

This starts PostgreSQL 15 and the Hourglass app on port 8080.

### Manual

**1. Backend**

```bash
go run ./cmd/server
```

Runs on `:8080`. Requires `DATABASE_URL` pointing to a PostgreSQL instance.

**2. Database migrations**

```bash
go run ./cmd/migrate -all -dir migrations
```

Applies all `.up.sql` migrations then seeds demo data.

**3. Frontend**

```bash
cd web
bun install
bun run dev
```

Runs on `:3000`, proxies `/api` to `:8080`.

---

## Repository Structure

```
hourglass/
├── cmd/
│   ├── server/              # HTTP server entry + route wiring
│   ├── migrate/             # PostgreSQL migration CLI
│   └── schema/              # Schema tooling
├── internal/
│   ├── core/
│   │   ├── domain/          # Domain models per bounded context
│   │   ├── ports/           # Repository and service interfaces
│   │   └── services/        # Application services (business logic)
│   ├── adapters/
│   │   ├── primary/http/    # HTTP handlers (thin, delegate to services)
│   │   └── secondary/postgres/ # PostgreSQL repository implementations
│   ├── auth/                # JWT, password hashing
│   ├── cookies/             # Cookie helpers
│   ├── db/                  # PostgreSQL connection pool
│   ├── handlers/            # Health handler
│   ├── middleware/          # Auth, CORS, logging, rate limiting
│   └── models/              # Shared data structures and constants
├── pkg/api/                 # JSON response envelope
├── migrations/              # SQL migrations
├── web/                     # React frontend
│   └── src/
│       ├── api/             # TanStack Query query/mutation options
│       ├── routes/          # TanStack Router file-based pages
│       ├── components/      # Shared UI components (shadcn-based)
│       ├── hooks/           # Shared hooks
│       ├── types/           # TypeScript type definitions
│       └── lib/             # HTTP client, query client
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## Documentation Map

| Section                                | Description                                                                                                                            |
|----------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| [Architecture](architecture/README.md) | Hexagonal backend, frontend structure, data flow, routing                                                                              |
| [Domain Concepts](domain/README.md)    | Organizations, users, roles, time entries, expenses, approval workflow, contracts, projects, units, working groups, customers, exports |
| [Operations](operations/README.md)     | Setup, config, database, Docker, Makefile targets                                                                                      |
| [Testing](testing/README.md)           | Backend tests, frontend tests, e2e tests, test patterns                                                                                |

---

## Key Commands

```bash
make build           # Build Go binary to bin/hourglass
make run             # Run backend on :8080
make test            # Run all Go tests
make setup           # Run all migrations + seed
make migrate-up      # Run pending migrations
make migrate-down    # Rollback last migration
make docker-up       # docker-compose up -d
make docker-down     # docker-compose down

cd web && bun run dev      # Frontend dev server on :3000
cd web && bun run build    # Type-check + production build
cd web && bun run test     # Vitest unit tests
cd web && bunx playwright test  # E2E tests
```

---

## Next Steps

1. Read the [Architecture](architecture/README.md) page to understand how the backend and frontend are organized.
2. Read the [Domain Concepts](domain/README.md) page to understand the core business models: organizations, time
   entries, expenses, approval workflows, and the contract/project hierarchy.
3. Read the [Operations](operations/README.md) page for configuration details and operational runbooks.
4. Read the [Testing](testing/README.md) page to understand the test infrastructure and how to write and run tests.
