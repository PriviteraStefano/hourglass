# Hourglass — Developer Reference

This page is the technical companion to the [README](../README.md). It carries the
full developer reference that the README deliberately keeps condensed: project
structure, environment configuration, Makefile targets, testing commands, the HTTP
API surface, and the domain model.

> **Regenerated from source on 2026-08-01.** Every table on this page was checked
> against the code (Makefile, `cmd/server/main.go`, `internal/models`, `web/vite.config.ts`,
> `web/package.json`) rather than copied from the README. When in doubt, trust the code.

---

## Architecture overview

Hourglass is a full-stack time entry and expense tracking system with approval
workflows for organizations:

- **Backend** — Go 1.26.1, standard library `net/http`, hexagonal (ports & adapters)
  architecture. HTTP handlers live in `internal/adapters/primary/http/` and stay
  thin; business logic lives in `internal/core/services/*`; PostgreSQL adapters
  live in `internal/adapters/secondary/postgres/`.
- **Frontend** — React 19, TanStack Router v1, TanStack React Query v5, Vite,
  TypeScript, Tailwind CSS v4, shadcn/ui.
- **Database** — PostgreSQL 15 for all application data.
- **Auth** — JWT (`golang-jwt/jwt/v5`) with HttpOnly `auth_token` / `refresh_token`
  cookies and bcrypt password hashing.

## Project structure

```
hourglass/
├── cmd/
│   ├── server/          # HTTP server entry point + route wiring
│   └── migrate/         # PostgreSQL migration CLI (-up / -down)
├── internal/
│   ├── core/            # Domain, ports, and application services
│   │   └── services/    # auth, activity, contract, customer, expense, export,
│   │                    # invitation, organization, password_reset, time_entry,
│   │                    # unit, working_group
│   ├── adapters/
│   │   ├── primary/http/        # Thin HTTP adapters (auth, activity, expense…)
│   │   └── secondary/postgres/  # PostgreSQL repositories (driven adapters)
│   ├── auth/            # JWT, password hashing, token service
│   ├── cookies/         # Cookie helpers
│   ├── db/              # PostgreSQL connection pool (pgxpool)
│   ├── handlers/        # Health handler + legacy glue
│   ├── middleware/      # Auth, rate limiting, logging, CORS, API versioning
│   └── models/          # Data structures and constants (Role, Status, Governance)
├── pkg/api/             # Shared { data | error } response envelope
├── migrations/          # SQL migrations (*.up.sql / *.down.sql)
├── web/                 # React frontend (Vite + TanStack)
│   ├── e2e/             # Playwright end-to-end specs
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

## Prerequisites

- **Go** >= 1.26.1
- **Node.js** + **Bun** (for the frontend)
- **PostgreSQL** >= 15 (or the bundled docker-compose service)
- **Docker** + **docker-compose** (optional, for containerized runs)

## Local development

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
make docker-up          # start postgres (or point DATABASE_URL at your own)

make migrate-up         # apply migrations (go run ./cmd/migrate -up -dir migrations)
make run                # http://localhost:8080
```

**Frontend** (terminal 2):

```bash
cd web
bun install
bun run dev             # http://localhost:3000, proxies /api → http://localhost:8080
```

> The Vite dev server rewrites `/api/*` → `http://localhost:8080/*` (the `/api`
> prefix is stripped), so the frontend can call relative paths like `/auth/me`.

## Configuration

### Backend environment variables

| Variable              | Description                                       | Default                                                     |
|-----------------------|---------------------------------------------------|-------------------------------------------------------------|
| `DATABASE_URL`        | PostgreSQL connection string                      | `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` |
| `JWT_SECRET`          | JWT signing key — **fatal if empty in production/staging** | `dev-secret-change-in-production`                  |
| `GO_ENV`              | Runtime environment (`production`/`staging` tighten JWT_SECRET handling) | *(empty)*                        |
| `PORT`                | HTTP listen port                                  | `8080`                                                      |
| `ALLOWED_ORIGINS`     | Comma-separated CORS allowlist                    | `http://localhost:3000`                                     |
| `RATE_LIMIT`          | Requests/min for auth endpoints (register/login, password reset) | `5` (password reset: `3`)                  |
| `ANONYMOUS_RATE_LIMIT`| Requests/min for anonymous clients on all routes; authenticated clients get 100 | `20`        |

### Frontend environment variables

| Variable       | Description      | Default |
|----------------|------------------|---------|
| `VITE_API_URL` | Backend base URL | `/api`  |

## Makefile targets

| Target          | Recipe                                                | Description                          |
|-----------------|-------------------------------------------------------|--------------------------------------|
| `make build`    | `go build -o bin/hourglass ./cmd/server`              | Compile Go binary to `bin/hourglass` |
| `make run`      | `go run ./cmd/server`                                 | Run the server on :8080              |
| `make test`     | `go test -v ./...`                                    | Run all Go tests                     |
| `make migrate-up`   | `go run ./cmd/migrate -up -dir migrations`        | Apply pending migrations             |
| `make migrate-down` | `go run ./cmd/migrate -down -dir migrations`      | Roll back last migration             |
| `make migrate-all`  | `go run ./cmd/migrate -all -dir migrations`       | Apply all migrations then seed       |
| `make setup`    | `go run ./cmd/migrate -all`                           | One-shot migrate + seed              |
| `make clean`    | `rm -rf bin/`                                         | Remove build output                  |
| `make docker-build` | `docker build -t hourglass:latest .`              | Build the multi-stage Docker image   |
| `make docker-up`    | `docker-compose up -d`                            | Start postgres + app                 |
| `make docker-down`  | `docker-compose down`                            | Stop containers                      |
| `make db-init`  | `docker exec -i hourglass-postgres psql -U hourglass -d hourglass < migrations/001_init.up.sql` | Initialize the schema into the running postgres container |

> **Known drift:** `cmd/migrate` currently implements only `-up` and `-down`
> (`getCommand` in `cmd/migrate/main.go`); the `-all` flag referenced by
> `make setup` and `make migrate-all` falls through to the usage error path.
> Until that lands, use `make migrate-up` or `make db-init` to initialize a
> fresh database.

## Testing and quality

```bash
# Backend
make test                                # go test -v ./...

# Frontend unit tests
cd web && bun run test                   # vitest run

# Frontend lint (oxlint)
cd web && bun run lint                   # oxlint --type-aware
cd web && bun run typecheck              # tsc -b

# End-to-end tests (Playwright)
cd web && bunx playwright test           # specs in web/e2e/
```

## HTTP API surface

All routes except `/health` and the auth endpoints are protected by the auth
middleware. Responses use the shared envelope from `pkg/api`: `{ "data": … }` on
success, `{ "error": … }` on failure. 401 responses trigger a cookie refresh via
`POST /auth/refresh` in `web/src/lib/api.ts`.

| Group              | Routes (method + pattern)                                                                                                |
|--------------------|---------------------------------------------------------------------------------------------------------------------------|
| Health             | `GET /health`                                                                                                              |
| Auth               | `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `POST /auth/refresh`, `GET /auth/me`, `POST /auth/bootstrap`, `GET /auth/bootstrap-check`, `POST /auth/switch-organization`, `GET /auth/memberships` |
| Password reset     | `POST /auth/password-reset/request`, `POST /auth/password-reset/verify`                                                    |
| Invitations        | `POST /invitations`, `GET /invitations/validate/code/{code}`, `GET /invitations/validate/token/{token}`, `POST /invitations/accept` |
| Units              | `GET/POST /units`, `GET/PUT/DELETE /units/{id}`, `GET /units/tree`, `GET /units/{id}/descendants`, `GET/POST /units/{id}/members`, `PUT/DELETE /units/{id}/members/{membership_id}`, `GET /units/members/batch` |
| Working groups     | `GET/POST /working-groups`, `GET/PUT/DELETE /working-groups/{id}`, `GET/POST /working-groups/{id}/members`, `DELETE /working-groups/{id}/members/{member_id}` |
| Customers          | `GET/POST /customers`, `GET/PUT/DELETE /customers/{id}`                                                                    |
| Organizations      | `POST /organizations`, `GET /organizations/{id}`, `POST /organizations/invite`, `POST /organizations/invite-customer`, `GET/PUT /organizations/{id}/settings`, `GET /organizations/members`, `PUT /organizations/members/{member_id}/roles`, `DELETE /organizations/members/{member_id}` |
| Activities         | `GET/POST /activities`, `GET/PUT/DELETE /activities/{id}`, `GET /activities/{id}/children`, `GET /activity-kinds`           |
| Contracts          | `GET/POST /contracts`, `GET/PUT/DELETE /contracts/{id}`, `POST /contracts/{id}/adopt`, `POST /contracts/{id}/recalculate-mileage` |
| Exports            | `GET /exports/timesheets`, `GET /exports/expenses`, `GET /exports/combined`, plus `/count` variants for each                |
| Time entries       | `GET/POST /time-entries`, `GET/PUT/DELETE /time-entries/{id}`, `POST /time-entries/{id}/submit|approve|reject`, `GET /time-entries/pending` |
| Expenses           | `GET/POST /expenses`, `GET/PUT/DELETE /expenses/{id}`, `POST /expenses/{id}/submit|approve|reject`, `POST /expenses/{id}/receipt`, `GET /expenses/pending` |

## Domain model reference

**Roles**: `employee`, `manager`, `finance`, `customer`

**Entry status**: `draft` → `submitted` → `pending_manager` → `pending_finance` → `approved` / `rejected`

**Approval actions**: `submit`, `approve`, `reject`, `edit_approve`, `edit_return`, `partial_approve`, `delegate`

**Governance models**: `creator_controlled`, `unanimous`, `majority`

**Project types**: `billable`, `internal`

**Expense categories**: `mileage`, `meal`, `accommodation`, `parking`, `travel_tickets`, `tolls`, `taxi`, `equipment`, `other`

**Time entry**: `TimeEntry` (header, contains `Status` + `CurrentApproverRole`)
contains `[]TimeEntryItem` (line items with hours per activity). Approval history
is immutable in the `*_approvals` tables (`TimeEntryApproval`, `ExpenseApproval`),
each row recording the `Action`, the acting `ActorRole`, and a timestamp.

---

_Source: regenerated from the repository on 2026-08-01 — Makefile, `cmd/server/main.go`,
`cmd/migrate/main.go`, `internal/models/models.go`, `web/vite.config.ts`,
`web/src/lib/api.ts`, `web/package.json`._
