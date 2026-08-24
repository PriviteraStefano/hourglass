# Technology Stack

**Analysis Date:** 2026-08-12

## Languages

**Primary:**
- Go 1.26.1 - Backend server, CLI tools (`go.mod`, module `github.com/stefanoprivitera/hourglass`). All backend code in `cmd/`, `internal/`, `pkg/`
- TypeScript 7.0.2 - Frontend application (`web/package.json`, `web/tsconfig.json`). Strict mode, `erasableSyntaxOnly`, ES2022 target

**Secondary:**
- SQL (PostgreSQL) - 22 numbered up/down migration pairs in `migrations/` (e.g., `000_full_schema.up.sql`, `004_time_entries_status_check.up.sql`)
- CSS - Tailwind CSS 4 syntax (CSS-first config, `@import "tailwindcss"` in `web/src/index.css`), no `tailwind.config.js` (v4 uses `@tailwindcss/vite` plugin)
- Bash/Make - `Makefile` (build/run/migrate/seed/docker targets), `scripts/*.sh` (docs-check, seed, verification)

## Runtime

**Environment:**
- Backend: compiled Go binary (`go run ./cmd/server`, builds to `bin/hourglass`), stdlib `net/http` server on port 8080 (`cmd/server/main.go`)
- Frontend: Vite 8 dev server on port 3000, proxies `/api` → `http://localhost:8080` (`web/vite.config.ts`)

**Package Manager:**
- Backend: Go modules (`go.mod`, `go.sum`)
- Frontend: Bun (lockfile `web/bun.lock`, root `bun.lock`) - `bun install`, `bun run dev`
- Lockfiles: present for both Go and Bun

## Frameworks

**Core:**
- Backend: Go standard library only for HTTP (no framework) - Go 1.22+ `ServeMux` method+path patterns (`mux.HandleFunc("POST /time-entries", ...)` in `cmd/server/main.go`). Hexagonal (ports & adapters) architecture: `internal/core/services/`, `internal/adapters/primary/http/`, `internal/adapters/secondary/postgres/`
- Frontend: React 19.2.8 (`react`, `react-dom`), TanStack Router v1.170 (file-based routing via `@tanstack/router-plugin/vite`, auto code-splitting), TanStack React Query v5.101, Vite 8.1, Tailwind CSS 4.3 via `@tailwindcss/vite`

**Testing:**
- Backend: `testing` + `github.com/stretchr/testify` v1.11.1; integration tests use `github.com/testcontainers/testcontainers-go/modules/postgres` v0.42.0 (real Postgres in Docker, see `internal/adapters/secondary/postgres/test_setup.go`, `internal/adapters/primary/http/handler_test_helper.go`)
- Frontend unit: Vitest 4.1 + `@testing-library/react` + `@testing-library/jest-dom` + jsdom 29 + `msw` 2.15 (config `web/vitest.config.ts`, setup `web/src/lib/__tests__/setup.ts`)
- Frontend e2e: Playwright 1.62 (`web/playwright.config.ts`, specs in `web/e2e/`)

**Build/Dev:**
- `make build` / `make test` / `make docker-build` (`Makefile`)
- Multi-stage `Dockerfile`: `golang:1.26.1-alpine` builder → `alpine:latest` runtime (CGO_ENABLED=0)
- `docker-compose.yml`: postgres:15-alpine + app for local dev
- Linting/format: oxlint 1.76 (`--type-aware`, `web/.oxlintrc.json`) and oxfmt 0.61 (`web/.oxfmtrc.json`, 80-col, semicolons, double quotes); Qodana (`qodana.yaml`, CI only)

## Key Dependencies

**Critical (backend):**
- `github.com/golang-jwt/jwt/v5` v5.3.0 - JWT creation/validation, HS256 (`internal/auth/auth.go`)
- `golang.org/x/crypto` v0.53.0 - bcrypt password hashing, cost 12 (`internal/auth/password_hasher.go`)
- `github.com/jackc/pgx/v5` v5.10.0 - pgxpool connection pool for the server (`internal/db/db.go` → `NewPool()`)
- `github.com/lib/pq` v1.10.9 - `database/sql` driver (legacy path, `internal/db/db.go` → `New()`; superseded by pgx for server)
- `github.com/google/uuid` v1.6.0 - all entity IDs (`uuid.UUID`)
- `github.com/xuri/excelize/v2` v2.11.0 - XLSX export generation (`internal/adapters/primary/http/export.go`)

**Critical (frontend):**
- `@tanstack/react-query` v5.101 - all data fetching, shared client in `web/src/lib/query-client.ts` (retry: false, staleTime: 30000)
- `@tanstack/react-router` v1.170 + `@tanstack/router-plugin` - file-based routing in `web/src/routes/`
- `zod` v4.4 - validation (with `@hookform/resolvers` for react-hook-form)
- `@base-ui/react` v1.6 - shadcn-style UI primitives (components.json `"style": "base-mira"`, ui alias maps to it)
- `@xyflow/react` + `dagre` - workflow/approval-flow diagrams
- `recharts` 3.8 - charts
- `zustand` v5 - client state
- `sonner`, `vaul`, `cmdk`, `react-day-picker`, `input-otp`, `embla-carousel-react`, `lucide-react`, `next-themes`, `date-fns`, `class-variance-authority`, `clsx`, `tailwind-merge`, `@fontsource-variable/inter` - UI utilities

## Configuration

**Environment:**
- Backend reads env vars in `cmd/server/main.go` and `internal/db/db.go`:
  - `DATABASE_URL` - PostgreSQL connection string (default `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`)
  - `JWT_SECRET` - token signing key (default `dev-secret-change-in-production`; **fatal** if unset when `GO_ENV=production|staging`)
  - `PORT` - HTTP port (default 8080)
  - `ALLOWED_ORIGINS` - comma-separated CORS allowlist (default `http://localhost:3000`)
  - `GO_ENV` - environment gate for JWT_SECRET enforcement
  - `RATE_LIMIT` - auth endpoint rate limit per window (default 5)
  - `ANONYMOUS_RATE_LIMIT` - outer per-IP limit (default 20)
- Frontend: `VITE_API_URL` (default `/api`, proxied in dev) - `web/src/lib/api.ts`
- `.env` files: `deploy/demo/.env` + `deploy/demo/.env.example` present (demo deployment secrets; do not read/commit `.env`)

**Build:**
- Backend: `Makefile` (build, run, migrate-up/down, test, seed, docker-* targets); `Dockerfile` (multi-stage); `deploy/demo/Dockerfile` + `deploy/demo/Dockerfile.web` (demo)
- Frontend: `web/vite.config.ts` (router plugin, react plugin, tailwind plugin, `@` → `./src` alias), `web/tsconfig.json` (strict, path alias `@/*`), `web/components.json` (shadcn)

## Platform Requirements

**Development:**
- Go 1.26.1 toolchain
- Bun (frontend install/dev)
- Docker + docker-compose (Postgres 15 container, integration tests via testcontainers)
- Node types v24 (`@types/node`)

**Production:**
- Docker images (`Dockerfile` — `bin` binary + `migrations/` + `uploads/receipts` dir, EXPOSE 8080)
- Demo deployment: `deploy/demo/docker-compose.yml` — app + web (built from `deploy/demo/Dockerfile*`) + postgres:17-alpine + cloudflared tunnel for HTTPS ingress; one-shot `migrate`/`seed`/`recover-db` services
- Runtime image is `alpine:latest` with ca-certificates

---

*Stack analysis: 2026-08-12*
