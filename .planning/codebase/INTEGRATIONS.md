# External Integrations

**Analysis Date:** 2026-06-08

## APIs & External Services

**None detected.** The application operates as a self-contained system with no external HTTP APIs, SaaS services, or third-party platform integrations. All business logic, authentication, data storage, and export capabilities are implemented in-house.

## Data Storage

**Databases:**
- **PostgreSQL 15** — Single database for all application data (users, organizations, time entries, contracts, projects, invitations, customers, work units, working groups, audit logs, approvals, refresh tokens, password reset tokens).
  - Connection: `DATABASE_URL` environment variable.
  - Client: `github.com/jackc/pgx/v5` with `pgxpool` connection pooling (`internal/db/pgpool.go`).
  - Pool config: 25 max connections, 30min max lifetime, 5min max idle time.
  - Migrations: Raw SQL files in `migrations/` directory, applied via `cmd/migrate/main.go`.
  - Seed data: `migrations/003_seed.up.sql`.

**File Storage:**
- **Local filesystem only** — Receipt/file uploads stored at `/app/uploads/receipts` (created in Dockerfile at line 25). No cloud storage (S3, GCS, etc.) configured.

**Caching:**
- **None** — No Redis, Memcached, or any caching layer. React Query provides client-side caching (30s stale time, `staleTime: 30000` in `web/src/lib/query-client.ts`). Rate limiting state is in-memory (`internal/middleware/ratelimit.go`).

## Authentication & Identity

**Auth Provider:**
- **Custom JWT-based authentication** — No external identity provider (no Google, GitHub, Auth0, Clerk, etc.).
  - Access token: 15-minute expiry, HS256-signed JWT with claims (UserID, OrganizationID, Role, Email) stored in HttpOnly `auth_token` cookie.
  - Refresh token: 7-day expiry, opaque UUID stored as HttpOnly `refresh_token` cookie; hashed with SHA-256 in database.
  - Password hashing: bcrypt with cost factor 12 (`golang.org/x/crypto`).
  - Implementation: `internal/auth/auth.go` (Service), `internal/auth/token_service.go` (port adapter), `internal/auth/password_hasher.go` (bcrypt wrapper).

**Password Reset:**
- **Custom code-based flow** — No email service integration. Reset requests go through:
  - `POST /auth/password-reset/request` — Generates a reset code stored in `password_resets` table.
  - `POST /auth/password-reset/verify` — Validates code and updates password.
  - Service: `internal/core/services/password_reset/service.go`
  - Repository: `internal/adapters/secondary/postgres/password_reset_repository.go`

**Invitations:**
- **Self-managed invitation system** — No external invitation provider. Organizations invite users via:
  - `POST /invitations` — Creates invitation record with code and token.
  - `GET /invitations/validate/code/{code}` / `validate/token/{token}` — Validates pending invitations.
  - `POST /invitations/accept` — Accepts invitation, creates membership.
  - Service: `internal/core/services/invitation/service.go`
  - Repository: `internal/adapters/secondary/postgres/invitation_repository.go`

**Bootstrap:**
- `POST /auth/bootstrap` — First-time setup endpoint for creating initial organization and admin user. Checked via `GET /auth/bootstrap-check`.

## Monitoring & Observability

**Error Tracking:**
- **None** — No Sentry, Datadog, or similar error monitoring integration.

**Logs:**
- **Standard library `log` package** — Structured-ish logging in `internal/middleware/middleware.go` (method, path, status, duration in ms). All other logging uses `log.Printf` / `log.Fatal`. No structured logging library (zerolog, slog) detected.

## CI/CD & Deployment

**Hosting:**
- **Docker-based** — Multi-stage `Dockerfile` produces an Alpine-based image exposing port 8080. Suitable for any container runtime (Docker, Kubernetes, ECS, etc.). No specific cloud provider targeting detected.

**CI Pipeline:**
- **None detected** — No `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`, or similar CI configuration found. No GitHub Actions workflows.

## Environment Configuration

**Required env vars:**
| Variable | Default | Required In | Set By |
|----------|---------|-------------|--------|
| `DATABASE_URL` | `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable` | Always | `cmd/server/main.go` |
| `JWT_SECRET` | `dev-secret-change-in-production` | Production/Staging | `cmd/server/main.go` |
| `PORT` | `8080` | When not 8080 | `cmd/server/main.go` |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | When not localhost | `cmd/server/main.go` |
| `GO_ENV` | empty | Production validation | `cmd/server/main.go` |
| `VITE_API_URL` | `/api` | When proxy not used | `web/src/lib/api.ts` |

**Secrets location:**
- `JWT_SECRET` — Passed via environment variable (referenced in `docker-compose.yml` for dev).
- Database credentials — In `DATABASE_URL` environment variable.
- No `.env` files committed; no secrets management (Vault, AWS Secrets Manager, etc.) configured.

## Webhooks & Callbacks

**Incoming:**
- **None** — No webhook endpoints registered.

**Outgoing:**
- **None** — No outgoing webhooks to external systems.

## Email Service

**None detected.** The invitation and password reset systems generate codes/tokens stored in the database but have no email dispatch mechanism. No SMTP configuration, email templates, or email provider SDKs (SendGrid, Postmark, Resend, Mailgun) found in any source file or dependency.

---

## Deployment Architecture (Docker Compose)

```
┌──────────────┐     TCP:8080      ┌──────────────┐
│              │ ◄──────────────── │              │
│  Go Backend  │                   │  Frontend    │
│  (port 8080) │                   │  (port 3000) │
│              │ ────────────────► │  Vite Dev    │
└──────┬───────┘   proxy /api      └──────────────┘
       │
       │ TCP:5432
       ▼
┌──────────────┐
│  PostgreSQL  │
│  (port 5432) │
│  postgres:15 │
└──────────────┘
```

The application has no external service dependencies beyond PostgreSQL. All auth, storage, and business logic is self-contained.

---

*Integration audit: 2026-06-08*
