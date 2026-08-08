# External Integrations

**Analysis Date:** 2026-08-08

## APIs & External Services

**Third-party SaaS APIs:**
- None. The application makes no outbound calls to external APIs. No payment, email, SMS, or cloud-service SDKs are used (verified: no `stripe`, `supabase`, `aws`, `smtp` imports in `cmd/`, `internal/`, or `web/src/`).

**Outbound HTTP:**
- None from backend code. Frontend fetches only the Hourglass backend via `web/src/lib/api.ts` (`api<T>()` helper with `credentials: "include"`).

## Data Storage

**Databases:**
- PostgreSQL (sole data store for all application data)
  - Connection: `DATABASE_URL` env var (`internal/db/db.go`, `cmd/migrate/main.go`)
  - Client: `jackc/pgx/v5` pgxpool for the server; `lib/pq` + `database/sql` for the migrate CLI and legacy `internal/db.New()`
  - Driver config: pool pins `timezone=UTC` on connections for deterministic date handling (`internal/db/db.go:59`)
  - Extension: `pgcrypto` (used in `migrations/000_full_schema.up.sql`)
  - Schema: versioned SQL migrations `migrations/*.up.sql`/`*.down.sql` applied by `cmd/migrate/main.go` (runbook: `make migrate-up`, `make setup`)
  - Local dev: postgres:15-alpine via `docker-compose.yml`; demo: postgres:17-alpine via `deploy/demo/docker-compose.yml`
  - Integration tests: ephemeral Postgres containers via testcontainers-go (`internal/adapters/secondary/postgres/test_setup.go`)

**File Storage:**
- Local filesystem only. Expense receipts uploaded to `uploads/receipts/{org_id}/{expense_id}/{uuid}{ext}` (`internal/adapters/primary/http/expense.go:513`)
  - 10 MB max upload (`http.MaxBytesReader`, `expense.go:492`)
  - Allowed types: `.pdf`, `.jpg`, `.jpeg`, `.png` (extension check, `expense.go:504`)
  - No object storage (S3/GCS) integration
  - Receipt URLs are served by the app (mounted volume `/app/uploads` in `docker-compose.yml`)

**Caching:**
- None (no Redis/memcached). No server-side cache; frontend relies on TanStack Query `staleTime: 30000` (`web/src/lib/query-client.ts`)

## Authentication & Identity

**Auth Provider:**
- Custom in-house implementation (no third-party IdP, no OAuth/OIDC)
  - Implementation: `internal/auth/auth.go` - HS256 JWT access tokens (15 min expiry, claims: `user_id`, `organization_id`, `role`, `email`); bcrypt cost-12 password hashing; refresh tokens as random UUIDs, stored hashed (SHA-256) with rotation + reuse detection (`migrations/010_refresh_token_reuse_detection.up.sql`, `internal/adapters/secondary/postgres/refresh_token_repo.go`)
  - Session transport: HttpOnly `auth_token`/`refresh_token` cookies, SameSite=Strict, Secure on TLS (`internal/cookies/cookies.go`); refresh via `POST /auth/refresh`, auto-retried once on 401 by `web/src/lib/api.ts`
  - Password reset + org invitations: token-based codes delivered **in the API response** (no email/SMS delivery - frontend displays them); endpoints `POST /auth/password-reset/request|verify`, `/invitations/*` in `cmd/server/main.go`
  - Rate limiting on auth endpoints: in-memory token-bucket-style limiter (`internal/middleware/ratelimit.go`), configurable via `RATE_LIMIT`/`ANONYMOUS_RATE_LIMIT`

## Monitoring & Observability

**Error Tracking:**
- None (no Sentry/Bugsnag). Errors surface via `log` + HTTP JSON error envelope

**Logs:**
- Go standard library `log` (`cmd/server/main.go`, `internal/middleware/middleware.go` Logging middleware) - request logging middleware wraps the mux; no structured logging library, no log shipping
- Demo compose uses docker json-file logging with 10m/3-file rotation (`deploy/demo/docker-compose.yml`)

**Metrics:**
- None (no Prometheus/metrics endpoint)

## CI/CD & Deployment

**Hosting:**
- Demo deployment: self-hosted Docker host; public ingress via Cloudflare Quick Tunnel (`cloudflare/cloudflared:2025.7.0`, `deploy/demo/docker-compose.yml`) with Caddy (v2-style Caddyfile, `deploy/demo/Caddyfile`) serving the built SPA and reverse-proxying `/api/*` → `app:8080`
- Local: docker-compose (`docker-compose.yml`) - postgres:15-alpine + app

**CI Pipeline:**
- GitHub Actions (`.github/workflows/`):
  - `docs-check.yml` - documentation completeness checks (docs-check.sh, mermaid validation) on PR/push, warnings only
  - `qodana_code_quality.yml` - Qodana static analysis on PR/push to main (requires `QODANA_TOKEN` secret)
  - No build/test/deploy pipeline for Go or frontend code; no artifact registry

## Environment Configuration

**Required env vars:**
- `DATABASE_URL` - PostgreSQL connection string (server + migrate)
- `JWT_SECRET` - JWT signing secret (REQUIRED when `GO_ENV=production|staging`; server exits if missing)
- Optional: `PORT`, `ALLOWED_ORIGINS`, `RATE_LIMIT`, `ANONYMOUS_RATE_LIMIT`, `GO_ENV`, `TZ` (demo)
- Frontend: `VITE_API_URL`

**Secrets location:**
- No committed secrets. Demo deployment reads `JWT_SECRET`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` from compose `.env` interpolation (`deploy/demo/docker-compose.yml`); dev uses defaults in `docker-compose.yml` (dev-only credentials)
- `.env` files are gitignored; none exist in the repo

## Webhooks & Callbacks

**Incoming:**
- None (no third-party webhook receivers)

**Outgoing:**
- None

## Other Notable Integrations

- **Exports**: CSV (stdlib `encoding/csv`) and XLSX (`xuri/excelize/v2`) downloads generated server-side (`internal/adapters/primary/http/export.go`) - not an external integration, but the only "file artifact" pipeline
- **Migrations CLI**: `cmd/migrate/main.go` applies `.up.sql`/`.down.sql` against PostgreSQL via `DATABASE_URL` (`go run ./cmd/migrate -all -dir migrations`)

---

*Integration audit: 2026-08-08*
