# Technology Stack

**Analysis Date:** 2026-05-12

## Languages

**Primary:**
- Go 1.26.1 - Backend server, CLI tools, database migrations
- TypeScript 6.0.3 - Frontend with strict mode

**Secondary:**
- SQL (PostgreSQL 15) - Relational data migrations
- SurrealQL - NoSQL schema definitions

## Runtime

**Backend:**
- Go 1.26.1 (Alpine Linux in Docker, macOS/Linux locally)

**Frontend:**
- Vite 8.0.10 dev server with HMR
- ESBuild for transpilation
- Target: ES2022, Module: ESNext

**Package Manager:**
- Backend: `go mod` / `go mod download`
- Frontend: `bun` (used for `bun install`, `bun run dev`, `bun run build`)
- Lockfile: `go.sum`, `bun.lockb`

## Frameworks

**Core Backend:**
- Go standard library `net/http` - HTTP server (no framework)
- Go 1.22+ `http.ServeMux` with pattern-based routing (`"GET /path", handler`)

**Core Frontend:**
- React 19.2.5 - UI library
- React DOM 19.2.5
- Tailwind CSS 4.2.4 (`@tailwindcss/vite`)
- TanStack Router 1.169.1 - File-based routing (`@tanstack/router-plugin`)
- TanStack React Query 5.100.8 - Server state
- TanStack React Form 1.29.1 - Form handling

**Build/Dev:**
- Vite 8.0.10 - Frontend build tool
- ESLint 9.39.4 + TypeScript ESLint 8.59.1 - Linting
- Playwright 1.59.1 - E2E testing (`web/playwright.config.ts`, `web/e2e/`)

**Testing:**
- Go testing (`go test -v ./...`)
- Playwright E2E tests

## Key Dependencies

**Backend (Go):**
- `github.com/golang-jwt/jwt/v5 v5.3.0` - JWT signing/verification
- `github.com/surrealdb/surrealdb.go v1.4.0` - SurrealDB RPC client
- `github.com/google/uuid v1.6.0` - UUID generation
- `github.com/lib/pq v1.10.9` - PostgreSQL driver (for migrations)
- `golang.org/x/crypto v0.48.0` - bcrypt password hashing
- `github.com/stretchr/testify v1.11.1` - Testing assertions

**Frontend (React/TypeScript):**
- `@tanstack/react-router v1.169.1` - File-based routing
- `@tanstack/react-query v5.100.8` - Data fetching/caching
- `@tanstack/react-form v1.29.1` - Form state
- `@base-ui/react v1.4.1` - Base UI component primitives
- `@xyflow/react v12.10.2` - React Flow for diagram rendering
- `recharts v3.8.0` - Charting library
- `shadcn v4.6.0` - Component registry (shadcn/ui)
- `tailwindcss v4.2.4` - CSS framework
- `react-hook-form v7.75.0` - Form hooks
- `zod v4.4.2` - Schema validation
- `date-fns v4.1.0` - Date utilities
- `lucide-react v1.14.0` - Icon library
- `zustand v5.0.13` - Client state management
- `vaul v1.1.2` - Drawer/dialog primitives
- `cmdk v1.1.1` - Command menu
- `next-themes v0.4.6` - Theme switching
- `embla-carousel-react v8.6.0` - Carousel
- `input-otp v1.4.2` - OTP input
- `sonner v2.0.7` - Toast notifications
- `dagre v0.8.5` + `@types/dagre v0.7.54` - Graph layout for React Flow
- `react-resizable-panels v4.10.0` - Resizable panel layouts
- `react-day-picker v9.14.0` - Calendar date picker

**Frontend Dev:**
- `typescript v6.0.3` - Type checking
- `eslint v9.39.4` - Linting
- `@vitejs/plugin-react v6.0.1` - React HMR in Vite
- `@tailwindcss/vite v4.2.4` - Tailwind Vite plugin

## Configuration

**Backend Environment Variables:**
| Variable | Default | Purpose |
|---|---|---|
| `SURREALDB_URL` | `ws://localhost:8000/rpc` | SurrealDB RPC endpoint |
| `SURREALDB_USER` | `root` | SurrealDB username |
| `SURREALDB_PASS` | `root` | SurrealDB password |
| `SURREALDB_NS` | `hourglass` | SurrealDB namespace |
| `SURREALDB_DB` | `main` | SurrealDB database |
| `JWT_SECRET` | `dev-secret-change-in-production` | Token signing key |
| `PORT` | `8080` | Server listen port |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | CORS allowlist (comma-separated) |
| `GO_ENV` | (none) | Set to `production`/`staging` to enforce JWT_SECRET |
| `DATABASE_URL` | `postgres://hourglass:hourglass@localhost:5432/hourglass` | Postgres for `cmd/migrate` |
| `SCHEMA_DIR` | `schema` | Directory for `.surql` schema files |

**Frontend Environment Variables:**
| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_URL` | `/api` | Backend base URL (proxied to `http://localhost:8080` in Vite) |

**Build Config Files:**
- `web/vite.config.ts` - Vite config with React, Tailwind, TanStack Router plugins; API proxy to `:8080`
- `web/tsconfig.json` - TypeScript target ES2022, path alias `@/*` → `./src/*`
- `web/tsconfig.app.json` - App-specific TS config
- `web/tsconfig.node.json` - Node-specific TS config (Vite config)
- `web/eslint.config.js` - ESLint flat config
- `web/playwright.config.ts` - Playwright E2E config
- `.nvmrc` - Not present; Node version not pinned
- `go.mod` / `go.sum` - Go module lock
- `Makefile` - Build targets: `build`, `run`, `migrate-up/down`, `test`, `docker-build`

## Platform Requirements

**Development:**
- macOS/Linux for local dev
- Go 1.26.1
- Node.js (for frontend with Vite)
- Bun (for package management)
- SurrealDB running on port 8000
- Optional: PostgreSQL for `cmd/migrate`

**Production:**
- Docker (Alpine Linux container for Go app)
- SurrealDB (in-memory or persistent)
- Optional: PostgreSQL 15 for migrations

---

*Stack analysis: 2026-05-12*
