# Operations

## Configuration

### Environment Variables

| Variable       | Default                                                                   | Description                                                                              |
|----------------|---------------------------------------------------------------------------|------------------------------------------------------------------------------------------|
| `DATABASE_URL` | `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` | PostgreSQL connection string                                                             |
| `JWT_SECRET`   | `dev-secret-change-in-production`                                         | JWT signing secret. **Required in production/staging** — app will `FATAL` exit if unset. |
| `PORT`         | `8080`                                                                    | HTTP server listen port                                                                  |
| `GO_ENV`       | (empty)                                                                   | Set to `production` or `staging` to enforce JWT_SECRET requirement                       |
| `RATE_LIMIT`   | `5`                                                                       | Max login/register attempts per window                                                   |
| `VITE_API_URL` | `/api`                                                                    | Frontend API base URL                                                                    |

### JWT_SECRET Enforcement

In `cmd/server/main.go`, the app checks `GO_ENV` at startup:

```go
if jwtSecret == "" {
    if env == "production" || env == "staging" {
        log.Fatal("FATAL: JWT_SECRET is required in production/staging environments")
    }
    log.Println("WARNING: Using default JWT_SECRET. Set JWT_SECRET in production.")
    jwtSecret = "dev-secret-change-in-production"
}
```

---

## Database

### Schema

The full schema is in a single migration file:
`/migrations/000_full_schema.up.sql` (23,628 bytes, 24 tables).

**Major tables:**

- `organizations`, `users`, `organization_memberships`
- `units`, `unit_memberships`
- `working_groups`, `wg_members`
- `customers`
- `contracts`, `contract_adoptions`
- `projects`, `project_adoptions`, `project_managers`
- `time_entries`, `time_entry_items`, `time_entry_approvals`
- `expenses`, `expense_items`, `expense_receipts`, `expense_approvals`
- `refresh_tokens`, `verification_tokens`, `password_reset_tokens`
- `backup_approvers`

### Migrations

The `cmd/migrate` CLI tool applies SQL migrations:

```bash
go run ./cmd/migrate -up -dir migrations    # Apply pending up migrations
go run ./cmd/migrate -down -dir migrations  # Rollback last migration
go run ./cmd/migrate -all -dir migrations   # Apply all + seed data
```

Migrations follow the `NNN_name.up.sql` / `NNN_name.down.sql` naming convention.

### Seed Data

`/migrations/003_seed.up.sql` inserts demo data for development:

- Sample organizations, users, memberships
- Sample projects, contracts, customers
- Sample time entries and expenses

### Connection Pool

Configured in `/internal/db/pgpool.go`:

| Setting                 | Value      |
|-------------------------|------------|
| Max connections         | 25         |
| Max connection lifetime | 30 minutes |
| Max idle time           | 5 minutes  |
| Connection timeout      | 5 seconds  |

Uses `sync.Once` to ensure a single pool instance per process lifetime.

---

## Docker

### docker-compose.yml

Starts two containers:

- **postgres** (`postgres:15-alpine`) on port 5432
- **app** (local build) on port 8080

```bash
docker-compose up -d       # Start in background
docker-compose down        # Stop
```

### Dockerfile

Multi-stage build:

1. **Build stage**: `golang:1.26.1-alpine` compiles the binary
2. **Runtime stage**: `alpine` with the compiled binary + trusted CA certs

```bash
docker build -t hourglass:latest .
```

---

## Makefile Targets

| Target         | Command                                      |
|----------------|----------------------------------------------|
| `build`        | `go build -o bin/hourglass ./cmd/server`     |
| `run`          | `go run ./cmd/server`                        |
| `test`         | `go test -v ./...`                           |
| `setup`        | `go run ./cmd/migrate -all` (migrate + seed) |
| `migrate-up`   | `go run ./cmd/migrate -up -dir migrations`   |
| `migrate-down` | `go run ./cmd/migrate -down -dir migrations` |
| `migrate-all`  | `go run ./cmd/migrate -all -dir migrations`  |
| `docker-build` | `docker build -t hourglass:latest .`         |
| `docker-up`    | `docker-compose up -d`                       |
| `docker-down`  | `docker-compose down`                        |
| `db-init`      | Direct SQL execution via `docker exec`       |
| `clean`        | `rm -rf bin/`                                |

---

## Frontend Dev Server

The Vite dev server proxies `/api` requests to the Go backend on `:8080`:

```ts
// web/vite.config.ts
server: {
  port: 3000,
    proxy
:
  {
    '/api'
  :
    {
      target: 'http://localhost:8080',
        changeOrigin
    :
      true,
        rewrite
    :
      (path) => path.replace(/^\/api/, ''),
    }
  ,
  }
,
}
```

---

## File Uploads

Receipt uploads for expenses are stored in the `./uploads/` directory (mounted
as a Docker volume). The backend reads `MIME type` from the uploaded file and
stores the file path + binary data in the `expense_receipts` table.

---

## Key Source Files

| Area               | File                                 |
|--------------------|--------------------------------------|
| Server entry + DI  | `/cmd/server/main.go`                |
| DB connection pool | `/internal/db/pgpool.go`             |
| Migration CLI      | `/cmd/migrate/main.go`               |
| Full schema        | `/migrations/000_full_schema.up.sql` |
| Seed data          | `/migrations/003_seed.up.sql`        |
| Docker compose     | `/docker-compose.yml`                |
| Dockerfile         | `/Dockerfile`                        |
| Makefile           | `/Makefile`                          |
| Frontend config    | `/web/vite.config.ts`                |
