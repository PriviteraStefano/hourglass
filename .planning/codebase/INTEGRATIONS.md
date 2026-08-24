# External Integrations

**Analysis Date:** 2026-08-12

## APIs & External Services

**None in application code.** The backend is fully self-contained: no outbound HTTP calls to third-party APIs, no SaaS SDKs, no email/SMS providers, no payment processors. Grep of `internal/` and `cmd/` finds no `http.Client` usage outside tests (`internal/adapters/primary/http/handler_test_helper.go`).

**Infrastructure-level services:**
- Cloudflare (cloudflared) - HTTPS ingress for the demo deployment via a quick tunnel; pinned `cloudflare/cloudflared:2025.7.0` in `deploy/demo/docker-compose.yml`. No API/token integration used (quick tunnel mode).
- GitHub Actions - CI workflows in `.github/workflows/` (docs-check, Qodana). Dev tooling only.
- Qodana - JetBrains code-quality scanning in CI (`qodana_code_quality.yml`, token from `secrets.QODANA_TOKEN`). Dev tooling only.

## Data Storage

**Databases:**
- PostgreSQL (primary and only datastore)
  - Connection: `DATABASE_URL` env var (default `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`)
  - Client: `github.com/jackc/pgx/v5` `pgxpool` for the server (`internal/db/db.go` → `NewPool()`); session `timezone=UTC` pinned at pool config
  - Legacy path: `database/sql` via `github.com/lib/pq` (`internal/db/db.go` → `New()`)
  - Migrations: 22 up/down pairs in `migrations/`, applied by `cmd/migrate/main.go` (`go run ./cmd/migrate -up -dir migrations`)
  - Dev: postgres:15-alpine in `docker-compose.yml`; demo: postgres:17-alpine in `deploy/demo/docker-compose.yml`
  - Seed: `scripts/seed_demo.sql` (via `make seed` / `make seed-demo`)

**File Storage:**
- Local filesystem only. Expense receipt uploads stored under `uploads/receipts/{org_id}/{expense_id}/` and served as relative URLs (`internal/adapters/primary/http/expense.go:479-544`). Docker volumes mount `./uploads` (dev) and the image pre-creates `/app/uploads/receipts` (demo Dockerfile).

**Caching:**
- None. No Redis/memcached. Only in-memory token-bucket rate limiters (`internal/middleware/ratelimit.go`, per-process state).

## Authentication & Identity

**Auth Provider:**
- Custom, self-implemented JWT auth (no external IdP, no OAuth/OIDC):
  - Implementation: `internal/auth/auth.go` - HS256-signed JWTs, 15-minute access tokens, 7-day refresh tokens; claims: user_id, organization_id, role, email
  - Password hashing: bcrypt cost 12 (`internal/auth/password_hasher.go`)
  - Tokens delivered as HttpOnly cookies `auth_token`/`refresh_token`, SameSite=Strict, Secure flag when configured (`internal/cookies/cookies.go`)
  - Refresh tokens persisted server-side in Postgres (`postgres.NewRefreshTokenRepository(pool)` in `cmd/server/main.go:70`)
  - Session rotation/refresh: `POST /auth/refresh`; frontend auto-refreshes once on 401 (`web/src/lib/api.ts`)
- Password reset: in-app token flow (no email delivery) - `POST /auth/password-reset/request` + `/verify` (`internal/core/services/password_reset/`)
- Invitations: token/code-based flow, validated via `GET /invitations/validate/code/{code}` and `/validate/token/{token}` (`internal/core/services/invitation/`)

## Monitoring & Observability

**Error Tracking:**
- None (no Sentry, Datadog, etc.)

**Logs:**
- Go standard library `log` package; request logging middleware (`internal/middleware/middleware.go` → `Logging`); docker logging with rotation (`max-size: 10m`, `max-file: 3` in `deploy/demo/docker-compose.yml`)

**Health:**
- `GET /health` endpoint (`internal/handlers/`, registered in `cmd/server/main.go:60`)

## CI/CD & Deployment

**Hosting:**
- Self-hosted demo deployment: Docker Compose stack (`deploy/demo/docker-compose.yml`) with `cloudflared` tunnel exposing the web container; no cloud platform (no AWS/GCP/Heroku/Netlify)
- Production image: multi-stage `Dockerfile` (golang:1.26.1-alpine → alpine)

**CI Pipeline:**
- GitHub Actions: `.github/workflows/docs-check.yml` (docs completeness + Mermaid validation, non-blocking warnings) and `.github/workflows/qodana_code_quality.yml` (Qodana scan, requires `QODANA_TOKEN` secret)
- No automated test/build pipeline (no CI runs of `make test` or `bun run build`); `make demo-redeploy` (`git pull` + compose up) is the manual deploy path (`Makefile`)

## Environment Configuration

**Required env vars:**
- `DATABASE_URL` - Postgres connection string (server + migrate + demo compose)
- `JWT_SECRET` - mandatory in production/staging (`GO_ENV` gate in `cmd/server/main.go`)
- `PORT` - server port (default 8080)
- `ALLOWED_ORIGINS` - CORS allowlist (default `http://localhost:3000`)
- `RATE_LIMIT`, `ANONYMOUS_RATE_LIMIT` - rate limiting knobs
- `VITE_API_URL` - frontend API base (default `/api`)
- Demo compose additionally: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `TZ=Europe/Rome`

**Secrets location:**
- `deploy/demo/.env` (demo deployment; `.env.example` committed alongside it in `deploy/demo/`) - never read or commit contents
- `JWT_SECRET` in production is required at startup; no secret manager integration

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None (no email, no SMS, no webhook dispatch; approvals and notifications are in-app only)

---

*Integration audit: 2026-08-12*
