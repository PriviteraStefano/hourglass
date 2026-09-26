# Technology Stack

**Analysis Date:** 2026-08-25

## Languages

**Primary:**
- Go 1.26.1 - Backend API server, services, repositories (`cmd/server/main.go`, `internal/core/services/*`, `internal/adapters/*`)
- TypeScript 7.0.2 (compiled via `tsc -b`) - Frontend SPA (`web/src/**`)

**Secondary:**
- SQL (PostgreSQL dialect) - Schema migrations (`migrations/*.up.sql`)
- Bash - Build/seed scripts (`Makefile`, `scripts/seed_demo.sql`)

## Runtime

**Environment:**
- Go 1.26.1 (backend), running as a single static binary (`bin/hourglass`, built in `Dockerfile`)
- Bun (package manager + dev server) for frontend; Node 24 typings
- Browser runtime for the React SPA (served by Vite dev at `:3000`, or static build behind a proxy)

**Package Manager:**
- Go modules (`go.mod` / `go.sum`) — backend
- Bun (`bun.lockb` implied by `bun install` in AGENTS.md; `package.json` present) — frontend
- Lockfile: `go.sum` present; `web/bun.lockb` (per AGENTS.md workflow)

## Frameworks

**Core (Backend):**
- Standard library `net/http` + `http.ServeMux` (Go 1.22+ method/path patterns) — HTTP routing (`cmd/server/main.go`)
- `github.com/golang-jwt/jwt/v5` - JWT auth (`internal/auth/auth.go`)
- `github.com/jackc/pgx/v5` - PostgreSQL connection pool (`internal/db/db.go`)
- `github.com/google/uuid` - UUID identifiers
- `golang.org/x/crypto` - bcrypt password hashing (`internal/auth/password_hasher.go`)

**Core (Frontend):**
- React 19.2 (`react`, `react-dom`) - UI
- TanStack Router v1 (`@tanstack/react-router`, `@tanstack/router-plugin`) - file-based routing (`web/src/routes/`)
- TanStack React Query v5 (`@tanstack/react-query`) - server state (`web/src/lib/query-client.ts`)
- TanStack React Form v1 + React Hook Form v7 + Zod v4 - forms/validation
- @base-ui/react (Base UI) + shadcn 4 - component primitives (`web/src/components/`)
- Tailwind CSS v4 (`@tailwindcss/vite`) - styling
- @xyflow/react + dagre - graph/flow visualizations
- recharts 3 - charts
- zustand 5 - client state

**Testing:**
- Go: `github.com/stretchr/testify` + `github.com/testcontainers/testcontainers-go/modules/postgres` (integration tests spin up real Postgres in Docker)
- Frontend: Vitest 4 + @testing-library/react + jsdom + msw (Mock Service Worker) + Playwright 1.62 (e2e in `web/e2e`)

**Build/Dev:**
- Vite 8 (`vite`, `@vitejs/plugin-react`) - frontend bundler/dev server
- oxlint 1.76 + oxfmt 0.61 - lint/format (replaces ESLint/Prettier)
- `tsc -b` - type-check
- Make - task runner (`Makefile`)
- Docker / docker-compose - containerization

## Key Dependencies

**Critical:**
- `github.com/jackc/pgx/v5` v5.10.0 - The only database driver; connection pooling and all queries go through `pgxpool` (`internal/db/db.go`, `internal/adapters/secondary/postgres/*`)
- `github.com/golang-jwt/jwt/v5` v5.3.0 - Access/refresh token signing/validation (`internal/auth/auth.go`)
- `golang.org/x/crypto` v0.53.0 - bcrypt hashing for passwords (`internal/auth/password_hasher.go`)
- `github.com/xuri/excelize/v2` v2.11.0 - Excel/CSV export generation (`internal/adapters/primary/http/export.go`)
- `github.com/google/uuid` v1.6.0 - Entity IDs and refresh-token generation

**Infrastructure:**
- `github.com/testcontainers/testcontainers-go` v0.42.0 - Spins up ephemeral Postgres for Go integration tests
- `github.com/lib/pq` v1.10.9 - Registered `postgres` driver (used by `internal/db/db.go`; primary driver is pgx)

## Configuration

**Environment:**
- Backend reads env vars at startup in `cmd/server/main.go` and `internal/db/db.go`:
  - `DATABASE_URL` - PostgreSQL connection string (default `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`)
  - `JWT_SECRET` - HS256 signing key (required when `GO_ENV=production`/`staging`, else warns + defaults to `dev-secret-change-in-production`)
  - `GO_ENV` - environment flag (production/staging triggers stricter JWT check)
  - `PORT` - listen port (default `8080`)
  - `ALLOWED_ORIGINS` - comma-separated CORS allowlist (default `http://localhost:3000`)
  - `RATE_LIMIT` - per-window auth rate limit (default 5)
  - `ANONYMOUS_RATE_LIMIT` - per-minute anonymous rate limit (default 20)
- Frontend: `VITE_API_URL` (optional; defaults to `/api`, proxied to `http://localhost:8080` in `web/vite.config.ts`)
- `.env` files: NOT present in repo (note: absence). Secrets supplied via shell/docker-compose env only.

**Build:**
- `go.mod` / `go.sum` - Go module definition
- `web/package.json` - frontend deps and scripts
- `web/vite.config.ts` - Vite config (aliases `@`→`./src`, React, Tailwind, TanStack Router plugin, `/api` proxy)
- `Dockerfile` - multi-stage Go build (golang:1.26.1-alpine → alpine:latest), exposes `8080`, expects `migrations/` + `uploads/receipts/`
- `docker-compose.yml` - `postgres:15-alpine` + `app` service
- `Makefile` - build/run/migrate/test/seed/docker targets

## Platform Requirements

**Development:**
- Go 1.26.1 toolchain
- Bun + Node 24 toolchain for `web/`
- PostgreSQL 15 (local or via `docker-compose up`)
- Docker (for Go integration tests via testcontainers, and for full-stack compose)

**Production:**
- Single Linux amd64 binary (CGO disabled) in a minimal Alpine image (`Dockerfile`)
- PostgreSQL 15 as the only external datastore
- Receipt uploads persisted to a mounted volume at `uploads/receipts/` (Docker volume `./uploads` in compose)

---

*Stack analysis: 2026-08-25*
