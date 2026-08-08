# Technology Stack

**Analysis Date:** 2026-08-08

## Languages

**Primary:**
- Go 1.26.1 - Backend server, migration CLI, all business logic (`go.mod`, `cmd/`, `internal/`)
- TypeScript 7.0.2 - Frontend application (`web/package.json`, `web/src/`)

**Secondary:**
- SQL (PostgreSQL dialect) - Schema migrations in `migrations/*.up.sql` / `*.down.sql`, seed scripts (`scripts/seed_demo.sql`)
- HTML/CSS - Vite SPA shell (`web/index.html`), Tailwind CSS v4 styles (`web/src/index.css`)

## Runtime

**Environment:**
- Backend: Go 1.26.1 standard library `net/http` server (no web framework), single binary built from `cmd/server/main.go`
- Frontend: Bun 1.3.13 runtime/package manager (lockfiles: root `bun.lock`, `web/bun.lock`)

**Package Manager:**
- Go modules (`go.mod`, `go.sum`) - backend
- Bun (`web/package.json`, `web/bun.lock`) - frontend
- Lockfile: present for both

## Frameworks

**Core:**
- Backend: No framework - hexagonal (ports & adapters) architecture with stdlib `net/http` and Go 1.22+ `mux.HandleFunc("METHOD /path", h)` patterns (`cmd/server/main.go`)
- React 19.2 - UI library (`web/package.json`)
- TanStack Router v1.170 - File-based routing with auto-generated `web/src/routeTree.gen.ts` (`web/vite.config.ts` plugin `@tanstack/router-plugin`)
- TanStack React Query v5.101 - Server state, query/mutation options in `web/src/api/*.ts`, shared client in `web/src/lib/query-client.ts`
- Tailwind CSS v4.3 - Styling via `@tailwindcss/vite` plugin (`web/vite.config.ts`, `web/src/index.css`)
- shadcn/ui (style: "base-mira") - Component library config in `web/components.json`, built on `@base-ui/react` (not Radix)

**Testing:**
- Backend: `stretchr/testify` v1.11.1 + `testcontainers-go` postgres module v0.42.0 for integration tests (`internal/adapters/primary/http/*_test.go`, `internal/adapters/secondary/postgres/test_setup.go`)
- Frontend: Vitest v4 + jsdom, `@testing-library/react`, `msw` v2 for API mocking (`web/vitest.config.ts`)
- E2E: Playwright v1.62 (`web/playwright.config.ts`, `web/e2e/*.spec.ts`)

**Build/Dev:**
- Vite 8.1 - Dev server on :3000 with `/api` proxy to :8080 (`web/vite.config.ts`)
- oxlint v1.76 - Linting (`web/.oxlintrc.json`); oxfmt v0.61 - Formatting (`web/.oxfmtrc.json`)
- Makefile - Build/run/test/docker targets; Docker multi-stage build (`Dockerfile`)
- Qodana - JetBrains static analysis (`qodana.yaml`, `.github/workflows/qodana_code_quality.yml`)

## Key Dependencies

**Critical:**
- `github.com/jackc/pgx/v5` v5.10.0 - pgxpool connection pool for PostgreSQL (`internal/db/db.go`)
- `github.com/golang-jwt/jwt/v5` v5.3.0 - HS256 JWT access tokens (`internal/auth/auth.go`)
- `golang.org/x/crypto` v0.53.0 - bcrypt password hashing (cost 12) (`internal/auth/auth.go`)
- `github.com/google/uuid` v1.6.0 - UUID generation for IDs and refresh/activation tokens
- `github.com/xuri/excelize/v2` v2.11.0 - XLSX export generation (`internal/adapters/primary/http/export.go`)
- `github.com/lib/pq` v1.10.9 - `database/sql` postgres driver used by `internal/db.New()` and `cmd/migrate/main.go`

**Frontend (key):**
- `zod` v4.4 - Schema validation (`@hookform/resolvers`)
- `react-hook-form` v7.83 + `@tanstack/react-form` v1.33 - Form handling
- `zustand` v5 - Client state (e.g., sidebar state)
- `recharts` 3.8 - Charts; `@xyflow/react` v12 + `dagre` - workflow/org graph rendering
- `lucide-react` v1.27 - Icons; `sonner` - toasts; `vaul` - drawers; `cmdk` - command menus; `date-fns` v4 - date utilities
- `@fontsource-variable/inter` - Inter variable font

**Infrastructure:**
- `testcontainers-go/modules/postgres` v0.42.0 - Spin-up ephemeral Postgres for integration tests
- `msw` v2.15 - Mock Service Worker for frontend unit tests

## Configuration

**Environment:**
- Configured via environment variables, no committed `.env` files (no `.env`/`.env.example` in repo)
- Backend (`cmd/server/main.go`, `cmd/migrate/main.go`, `internal/db/db.go`):
  - `DATABASE_URL` - PostgreSQL connection string (defaults to `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`; REQUIRED for migrate)
  - `JWT_SECRET` - Token signing key (defaults to `dev-secret-change-in-production`; hard-fails in `GO_ENV=production|staging` if unset)
  - `PORT` - HTTP port (default `8080`)
  - `ALLOWED_ORIGINS` - Comma-separated CORS allowlist (defaults to `http://localhost:3000`)
  - `RATE_LIMIT` - Auth-endpoint rate limit requests/100s (default 5)
  - `ANONYMOUS_RATE_LIMIT` - Global anonymous rate limit requests/100s (default 20)
  - `GO_ENV` - Env selector (production/staging enforce JWT_SECRET); `TZ` used in demo deploy
- Frontend (`web/src/lib/api.ts`, `web/src/api/exports.ts`, `web/src/lib/use-download.ts`):
  - `VITE_API_URL` - Backend base URL (defaults to `/api`, proxied to `http://localhost:8080` in dev)

**Build:**
- `web/tsconfig.json` - strict mode, path alias `@/* → ./src/*`, `erasableSyntaxOnly`
- `web/vite.config.ts` - TanStack Router plugin (auto code splitting), React plugin, Tailwind plugin, `/api` dev proxy
- `web/.oxlintrc.json` - oxlint with typescript/react/jsx-a11y/unicorn/import plugins, type-aware
- `web/.oxfmtrc.json` - printWidth 80, 2-space indent, semicolons, double quotes
- `Dockerfile` - multi-stage: `golang:1.26.1-alpine` builder → `alpine` runtime, copies migrations + `uploads/receipts` dir
- `docker-compose.yml` - postgres:15-alpine + app

## Platform Requirements

**Development:**
- Go >= 1.26.1, Bun (>= 1.3), PostgreSQL >= 15 (or docker-compose), Docker (optional)
- Playwright Chromium browser for e2e

**Production:**
- Docker images built from `Dockerfile` / `deploy/demo/Dockerfile`
- Demo deployment: `deploy/demo/docker-compose.yml` (postgres:17-alpine, Go app, Caddy static SPA server, cloudflared tunnel) - see `openwiki/operations/demo-deployment.md`
- CORS/Cookie note: cookies are Secure + SameSite=Strict when request is TLS (`internal/cookies/cookies.go`)

---

*Stack analysis: 2026-08-08*
