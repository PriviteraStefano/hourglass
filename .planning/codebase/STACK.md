# Technology Stack

**Analysis Date:** 2026-06-08

## Languages

**Primary:**
- **Go 1.26.1** — All backend code (`cmd/`, `internal/`, `pkg/`). Defined in `go.mod`.
- **TypeScript 6.0.3** — All frontend code (`web/src/`, `web/e2e/`). Defined in `web/tsconfig.json`.

**Secondary:**
- **SQL** — PostgreSQL migrations (`migrations/*.sql`), executed via `cmd/migrate/main.go`.
- **HTML/CSS** — `web/index.html`, `web/src/index.css` (Tailwind CSS v4 with shadcn).

## Runtime

**Environment:**
- **Backend:** Compiled Go binary running directly on host or in Docker.
- **Frontend:** Browser runtime (React 19 SPA served by Vite dev server or built `dist/`).

**Package Managers:**
- **Go modules** — `go.mod` / `go.sum` for backend dependencies.
- **Bun** — `web/bun.lock` (lockfileVersion 1) for frontend dependencies.

## Frameworks

**Core Backend:**
- **Go standard library `net/http`** — HTTP server with Go 1.22+ pattern-based `ServeMux` (`mux.HandleFunc("POST /auth/login", handler)`).

**Core Frontend:**
- **React 19** (`^19.2.5`) — UI framework.
- **TanStack React Router v1** (`^1.169.1`) — File-based routing with `createFileRoute`, `beforeLoad` guards, and `createRootRouteWithContext`.
- **TanStack React Query v5** (`^5.100.8`) — Server state management, `queryOptions`/`mutationOptions`, automatic cache invalidation.
- **Tailwind CSS v4** (`^4.2.4`) — Utility-first CSS via Vite plugin `@tailwindcss/vite`.

**Testing:**
- **Backend:** `github.com/stretchr/testify v1.11.1` — Assertions and mocking.
- **Frontend unit:** Vitest v4 (`^4.1.6`) with `jsdom` environment + `@testing-library/react v16`.
- **Frontend e2e:** Playwright v1 (`@playwright/test ^1.59.1`).
- **API mocking:** MSW v2 (`msw ^2.14.6`) — Mock Service Worker for frontend tests.

**Build/Dev:**
- **Backend:** `go build ./cmd/server` — Produces `bin/hourglass` binary.
- **Frontend:** Vite v8 (`^8.0.10`) — Dev server (port 3000 with `/api` proxy to `:8080`) + production build.
- **Docker** — Multi-stage build (`Dockerfile`) with `golang:1.26.1-alpine` builder and `alpine:latest` runtime.

## Key Dependencies

### Backend (`go.mod`)

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/jackc/pgx/v5` | `v5.10.0` | PostgreSQL driver with connection pooling (`pgxpool`) |
| `github.com/golang-jwt/jwt/v5` | `v5.3.0` | JWT token generation, signing (HS256), and validation |
| `github.com/google/uuid` | `v1.6.0` | UUID generation for IDs, tokens, refresh tokens |
| `golang.org/x/crypto` | `v0.48.0` | bcrypt password hashing (`bcrypt.GenerateFromPassword`) |
| `github.com/stretchr/testify` | `v1.11.1` | Test assertions and suite support |

### Frontend (`web/package.json`)

**State & Data:**
| Package | Purpose |
|---------|---------|
| `@tanstack/react-query ^5.100.8` | Server state, cache, mutations |
| `zustand ^5.0.13` | Client-side state management |

**Routing:**
| Package | Purpose |
|---------|---------|
| `@tanstack/react-router ^1.169.1` | File-based routing with route tree generation |
| `@tanstack/router-plugin ^1.167.32` | Vite plugin for route tree auto-generation |

**UI Components:**
| Package | Purpose |
|---------|---------|
| `@base-ui/react ^1.4.1` | Headless UI primitives (shadcn base) |
| `shadcn ^4.6.0` | Component scaffolding CLI |
| `lucide-react ^1.14.0` | Icon library |
| `cmdk ^1.1.1` | Command menu (search palette) |
| `sonner ^2.0.7` | Toast notifications |
| `vaul ^1.1.2` | Drawer component |
| `embla-carousel-react ^8.6.0` | Carousel/slider |
| `react-resizable-panels ^4.10.0` | Resizable split panels |
| `react-day-picker ^9.14.0` | Date picker |
| `input-otp ^1.4.2` | OTP code input |
| `@xyflow/react ^12.10.2` | Flow chart / node-graph (org hierarchy) |
| `dagre ^0.8.5` | Graph layout algorithm (used with xyflow) |
| `recharts 3.8.0` | Charting library |

**Forms & Validation:**
| Package | Purpose |
|---------|---------|
| `react-hook-form ^7.75.0` | Form state management |
| `@hookform/resolvers ^5.2.2` | Zod resolver integration |
| `zod ^4.4.2` | Schema validation |
| `@tanstack/react-form ^1.29.1` | Alternative form library |

**Styling:**
| Package | Purpose |
|---------|---------|
| `tailwindcss ^4.2.4` | Utility-first CSS (v4) |
| `@tailwindcss/vite ^4.2.4` | Vite plugin for Tailwind |
| `tailwind-merge ^3.5.0` | Tailwind class merge utility |
| `tw-animate-css ^1.4.0` | Tailwind animation utilities |
| `class-variance-authority ^0.7.1` | Component variant management |
| `clsx ^2.1.1` | Class name concatenation |

**Other:**
| Package | Purpose |
|---------|---------|
| `date-fns ^4.1.0` | Date formatting/manipulation |
| `next-themes ^0.4.6` | Theme (dark/light) switching |
| `@fontsource-variable/inter ^5.2.8` | Inter variable font |

## Configuration

**Backend Environment Variables:**
- `DATABASE_URL` — PostgreSQL connection string (default: `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`)
- `JWT_SECRET` — HMAC-SHA256 signing key (default: `dev-secret-change-in-production`)
- `ALLOWED_ORIGINS` — Comma-separated CORS origins (default: `http://localhost:3000`)
- `PORT` — Server listen port (default: `8080`)
- `GO_ENV` — Environment name (`production`, `staging`, or unset)

**Frontend Environment Variables:**
- `VITE_API_URL` — Backend API base URL (default: `/api`, proxied to `http://localhost:8080`)

**Build Configuration:**
- `web/vite.config.ts` — Vite config with React plugin, Tailwind plugin, TanStack Router plugin, path alias `@/` → `./src`, dev proxy `/api` → `:8080`
- `web/tsconfig.json` — TypeScript config: target ES2022, strict mode, path alias `@/` → `./src/*`
- `web/eslint.config.js` — ESLint flat config with `typescript-eslint`, `eslint-plugin-react-hooks`, `eslint-plugin-react-refresh`
- `Makefile` — Build, run, migrate, test, docker targets

## Platform Requirements

**Development:**
- Go 1.26.1+
- Bun (for frontend package management and dev server)
- PostgreSQL 15+ (local or Docker)
- Docker & Docker Compose (optional, for containerized dev)

**Production:**
- Linux (Docker image based on `alpine:latest`)
- PostgreSQL 15+ accessible via `DATABASE_URL`
- JWT_SECRET must be set (required in `production`/`staging` mode)

---

*Stack analysis: 2026-06-08*
