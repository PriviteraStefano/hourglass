# Hourglass

**Time entry and expense tracking with approval workflows for organizations.**

Hourglass is a full-stack application that lets organizations track employee time
entries and expenses through a configurable, role-based approval workflow
(employee → manager → finance). It ships with a Go backend (hexagonal
architecture) and a React frontend.

---

## Features

- **Time entries** — employees log hours per project with draft → submitted →
  pending_manager → pending_finance → approved/rejected status flow
- **Expenses** — mileage, meal, accommodation, and other categories, with the
  same two-stage approval workflow
- **Approval workflow** — role-differentiated actions: submit, approve, reject,
  edit_approve, edit_return, partial_approve, delegate
- **Organizations & roles** — `employee`, `manager`, `finance`, `customer` with
  DB-enforced CHECK constraints
- **Contracts & projects** — billable or internal, organization-specific or
  shared, with governance models (`creator_controlled`, `unanimous`, `majority`)
- **Exports** — date-range CSV export of approved entries
- **Auth** — JWT in HttpOnly cookies, bcrypt password hashing, refresh-on-401
- **Multi-stage Docker** — Postgres + app via `docker-compose`

---

## Tech stack

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

## Project structure

```
hourglass/
├── cmd/
│   ├── server/          # HTTP server entry point + route wiring
│   ├── migrate/         # PostgreSQL migration CLI
│   └── schema/          # Schema tooling
├── internal/
│   ├── core/            # Domain, ports, application services
│   ├── adapters/
│   │   ├── primary/http/      # Thin HTTP adapters (auth, project, time-entry, expense…)
│   │   └── secondary/postgres/# PostgreSQL repositories (driven adapters)
│   ├── auth/            # JWT, password hashing
│   ├── cookies/         # Cookie helpers
│   ├── db/              # PostgreSQL connection pool
│   ├── handlers/        # Health handler + legacy glue
│   ├── middleware/      # Auth middleware
│   └── models/          # Data structures, constants (Role, Status, Governance)
├── pkg/api/             # Shared response envelope
├── migrations/          # SQL migrations (*.up.sql / *.down.sql)
├── web/                 # React frontend (Vite + TanStack)
│   └── src/
│       ├── api/         # Query/mutation options (auth, projects, time-entries…)
│       ├── routes/      # TanStack Router file-based routing
│       ├── hooks/       # Domain hooks
│       ├── components/  # Reusable UI (shadcn-based)
│       └── lib/         # HTTP client, query client
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

---

## Requirements

- **Go** >= 1.26.1
- **Node.js** / **Bun** (for the frontend)
- **PostgreSQL** >= 15 (or use the bundled docker-compose service)
- **Docker** + **docker-compose** (optional, for containerized runs)

---

## Getting started

### 1. Clone

```bash
git clone https://github.com/PriviteraStefano/hourglass.git
cd hourglass
```

### 2. Run with Docker (easiest)

```bash
make docker-up          # starts postgres + app on :8080
make docker-down        # stop
```

### 3. Local development

**Backend** (terminal 1):

```bash
# start postgres (or point DATABASE_URL at your own)
make docker-up          # or just the postgres service

# apply migrations + seed
make setup              # runs: go run ./cmd/migrate -all

# run the server
make run                # http://localhost:8080
```

**Frontend** (terminal 2):

```bash
cd web
bun install
bun run dev             # http://localhost:3000 (proxies /api → :8080)
```

---

## Configuration

### Backend environment variables

| Variable          | Description                                  | Default                                             |
|-------------------|----------------------------------------------|-----------------------------------------------------|
| `DATABASE_URL`    | PostgreSQL connection string                 | `postgres://hourglass:hourglass@localhost:5432/...` |
| `JWT_SECRET`      | JWT signing key (**required in production**) | `dev-secret-change-in-production`                   |
| `ALLOWED_ORIGINS` | Comma-separated CORS allowlist               | `http://localhost:3000`                             |
| `PORT`            | HTTP listen port                             | `8080`                                              |

### Frontend environment variables

| Variable       | Description      | Default |
|----------------|------------------|---------|
| `VITE_API_URL` | Backend base URL | `/api`  |

---

## Makefile targets

| Target         | Description                             |
|----------------|-----------------------------------------|
| `make build`   | Compile Go binary to `bin/hourglass`    |
| `make run`     | Run the server (`go run ./cmd/server`)  |
| `make test`    | Run Go tests (`go test -v ./...`)       |
| `make setup`   | Apply all migrations + seed             |
| `migrate-up`   | Apply pending migrations                |
| `migrate-down` | Roll back last migration                |
| `migrate-all`  | Apply all migrations then seed          |
| `docker-build` | Build the multi-stage Docker image      |
| `docker-up`    | Start postgres + app via docker-compose |
| `docker-down`  | Stop containers                         |
| `clean`        | Remove `bin/`                           |

---

## Testing

```bash
# backend
make test                                # go test -v ./...

# frontend
cd web && bun run test                   # vitest
cd web && bun run lint                   # eslint
cd web && bunx playwright test           # e2e
```

---

## Domain model reference

**Roles**: `employee`, `manager`, `finance`, `customer`

**Entry status**: `draft` → `submitted` → `pending_manager` → `pending_finance` → `approved` / `rejected`

**Approval actions**: `submit`, `approve`, `reject`, `edit_approve`, `edit_return`, `partial_approve`, `delegate`

**Governance models**: `creator_controlled`, `unanimous`, `majority`

**Project types**: `billable`, `internal`

**Expense categories**: `mileage`, `meal`, `accommodation`, `other`

**Time entry**: `TimeEntry` (header) contains `[]TimeEntryItem` (line items with hours per project)

---

## License

Proprietary — see [LICENSE](./LICENSE).

This software is the proprietary property of Stefano Privitera. No license to
use, copy, modify, or distribute is granted without prior written permission.
Anyone granted permission to evaluate or use the software **must report usage
and any issues** to the copyright holder. See the [LICENSE](./LICENSE) file for
full terms.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
