# External Integrations

**Analysis Date:** 2026-05-12

## APIs & External Services

**No external third-party APIs detected.** All APIs are self-hosted HTTP endpoints built with Go.

The backend exposes a REST API (port 8080) consumed by the React frontend via:
- TanStack React Query `api()` helper in `web/src/lib/api.ts`
- Frontend proxies `/api` → `http://localhost:8080` via Vite dev server
- Production: Frontend build served separately; `VITE_API_URL` points to backend

**Key API Endpoints (self-hosted):**
| Prefix | Description |
|---|---|
| `POST /auth/*` | Registration, login, logout, refresh, password reset |
| `GET /auth/me` | Current user profile |
| `GET /auth/memberships` | User's organization memberships |
| `POST /auth/switch-organization` | Switch active org context |
| `POST /auth/bootstrap` | Initial setup (first org/user) |
| `GET/POST /units/*` | Organizational unit management |
| `GET/POST /working-groups/*` | Working group management |
| `GET/POST /customers/*` | Customer management |
| `POST /organizations/*` | Org creation, invite, settings |
| `GET/POST /projects/*` | Project management |
| `GET/POST /contracts/*` | Contract management |
| `GET/POST /time-entries/*` | Time entry CRUD + workflow actions |
| `POST /invitations/*` | Invitation creation and acceptance |
| `GET /exports/*` | Timesheet/expense CSV export |

## Data Storage

**SurrealDB (Primary Application Database):**
- Connection: `SURREALDB_URL` (`ws://localhost:8000/rpc` by default)
- Client: `github.com/surrealdb/surrealdb.go v1.4.0` (RPC protocol)
- Namespace: `SURREALDB_NS` (default: `hourglass`)
- Database: `SURREALDB_DB` (default: `main`)
- Schema: `schema/*.surql` applied via `cmd/schema`
- Purpose: All application data — users, organizations, time entries, projects, contracts, audit logs, etc.

**PostgreSQL 15 (Schema Migrations Only):**
- Connection: `DATABASE_URL` (default: `postgres://hourglass:hourglass@localhost:5432/hourglass`)
- Driver: `github.com/lib/pq`
- Client: `cmd/migrate` CLI utility
- Purpose: Legacy SQL migrations (now unused for active data storage; application moved to SurrealDB)
- Migrations: `migrations/*.up.sql` / `*.down.sql` for `001_init`, `002_contracts_projects`, `003_time_entries`, `004_expenses`, `005_approvals`, `006_refresh_tokens`, `007_phase2_schema`, `008_verification_tokens`

**Local File Storage:**
- `uploads/receipts/` - File storage for expense receipt uploads (Docker volume: `./uploads:/app/uploads`)

**No external cloud storage** (AWS S3, GCS, Cloudflare R2) detected.

## Authentication & Identity

**Auth Provider:** Custom JWT-based authentication

**Implementation:**
- `internal/auth/` - JWT service (sign/verify tokens)
- `golang.org/x/crypto` bcrypt - Password hashing
- Cookie-based: `auth_token` (access) and `refresh_token` (refresh) stored as HttpOnly cookies
- `web/src/lib/api.ts` - HTTP client with `credentials: 'include'` and auto-refresh on 401
- `web/src/api/auth.ts` - TanStack Query query/mutation options for auth
- `web/src/routes/_authenticated.tsx` - Route guard via `AuthApis.profileQueryOpts` (`GET /auth/me`)

**Auth Flow:**
1. User submits credentials to `POST /auth/login`
2. Backend returns `auth_token` + `refresh_token` as HttpOnly cookies
3. All subsequent requests include cookies automatically
4. On 401, frontend retries `POST /auth/refresh` once
5. If refresh fails, redirects to `/login`

## Monitoring & Observability

**Error Tracking:** None (no Sentry, Rollbar, etc.)

**Logs:**
- Backend: Go `log.Println` / `log.Fatalf` (plain text to stdout)
- No structured logging library detected
- No log aggregation or external log service

**No external monitoring** (Datadog, New Relic, Grafana, Prometheus) detected.

## CI/CD & Deployment

**Hosting:** Self-hosted Docker containers

**CI Pipeline:** None (no GitHub Actions, CircleCI, GitLab CI, etc.)

**Deployment:**
- `Dockerfile` - Multi-stage build: Go Alpine builder → Alpine runtime
- `docker-compose.yml` - Orchestrates `surrealdb`, optional `postgres` (profile-gated), and `app` services
- `Makefile` targets: `docker-build`, `docker-up`, `docker-down`
- SurrealDB: `surrealdb/surrealdb:v3.0.5` in-memory mode
- Volume: `postgres_data` for Postgres persistence

## Environment Configuration

**Required env vars for backend:**
- `SURREALDB_URL` - SurrealDB RPC endpoint
- `SURREALDB_USER` - SurrealDB username
- `SURREALDB_PASS` - SurrealDB password
- `SURREALDB_NS` - SurrealDB namespace
- `SURREALDB_DB` - SurrealDB database
- `JWT_SECRET` - Token signing key (**required** in production/staging via `GO_ENV`)
- `PORT` - Server port (default: `8080`)
- `ALLOWED_ORIGINS` - CORS allowlist
- `DATABASE_URL` - PostgreSQL connection (for `cmd/migrate`)
- `SCHEMA_DIR` - `.surql` files directory

**Secrets location:** Environment variables only (no Vault, AWS Secrets Manager, etc.)

## Webhooks & Callbacks

**Incoming webhooks:** None detected

**Outgoing webhooks:** None detected

---

*Integration audit: 2026-05-12*
