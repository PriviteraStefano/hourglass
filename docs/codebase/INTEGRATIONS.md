# External Integrations

**Analysis Date:** 2026-08-25

## APIs & External Services

**None (no third-party HTTP/cloud APIs).**
The backend makes no outbound calls to external SaaS/REST APIs. The only `http.Client` usage is in test helpers (`internal/adapters/primary/http/handler_test_helper.go:259`) for driving the local test server. No Stripe, SendGrid, AWS SDK, Twilio, Slack, or similar integrations are present.

**Self-hosted HTTP API (the app's own surface):**
- The Go `net/http` server exposes the REST API consumed by the SPA. Route table and wiring in `cmd/server/main.go`. Base path `/api` (proxied by Vite in dev, see `web/vite.config.ts:21-27`).
- Contract format: JSON envelope `{ "data": ... }` / `{ "error": ... }` from `pkg/api/response.go`.

## Data Storage

**Databases:**
- PostgreSQL 15 (provider: local/Docker; image `postgres:15-alpine` in `docker-compose.yml`)
  - Connection: `DATABASE_URL` env var (`internal/db/db.go:42`)
  - Driver/client: `github.com/jackc/pgx/v5/pgxpool` (`internal/db/db.go`), with `github.com/lib/pq` registered as a secondary `postgres` driver (`internal/db/db.go:11`)
  - Pool tuning: max open 25, max idle 5, lifetime 5m (`internal/db/db.go:24-26`); session timezone pinned to UTC (`internal/db/db.go:59`)
  - Repositories: `internal/adapters/secondary/postgres/*` (time entries, expenses, users, orgs, contracts, activities, tickets, coverage, units, working groups, customers, invitations, password resets, refresh tokens, exports, org settings, directions)

**File Storage:**
- Local filesystem for expense receipt uploads. Path layout: `uploads/receipts/{org_id}/{expense_id}/{filename}` (`internal/adapters/primary/http/expense.go:530-551`). Persisted via `os.Create`; the stored relative path is returned as `receiptURL`. In Docker, backed by the `./uploads` volume (`docker-compose.yml:35`). NO object store (S3/GCS) is used.

**Caching:**
- None (no Redis/Memcached). React Query client-side caching only (`web/src/lib/query-client.ts`, `staleTime: 30000`, `retry: false`).

## Authentication & Identity

**Auth Provider:**
- Custom JWT-based auth (no external IdP / OAuth provider).
  - Implementation: HS256 JWTs via `github.com/golang-jwt/jwt/v5` (`internal/auth/auth.go`). Access token 15m, refresh token 7d (`internal/auth/auth.go:14-17`).
  - Passwords hashed with bcrypt cost 12 (`internal/auth/password_hasher.go`, `internal/auth/auth.go:43`).
  - Tokens delivered as HttpOnly cookies `auth_token` / `refresh_token`; middleware `middleware.Auth` / `middleware.TryAuth` validates on protected routes (`cmd/server/main.go:356`).
  - Refresh tokens stored hashed (sha256) in `refresh_token` table via `postgres.NewRefreshTokenRepository` (`cmd/server/main.go:69`).
  - Bootstrap flow (`POST /auth/bootstrap`, `GET /auth/bootstrap-check`) seeds a first org/user when none exist (`cmd/server/main.go:97-98`).

## Monitoring & Observability

**Error Tracking:**
- None (no Sentry/Datadog). Errors returned in JSON `error` envelope and logged via `log` package (stdout).

**Logs:**
- Standard library `log` to stdout, wrapped by `middleware.Logging` (`cmd/server/main.go:356`). No structured/logrus sink in the server path (`logrus` appears only as a transitive testcontainers dependency).

## CI/CD & Deployment

**Hosting:**
- Containerized deployment: single image `hourglass:latest` from `Dockerfile`; compose stacks in repo root (`docker-compose.yml`) and demo env (`deploy/demo/docker-compose.yml`, referenced in `Makefile`).
- Demo deploy automation: `make demo-up` / `demo-migrate` / `demo-seed` / `demo-redeploy` (`Makefile`).

**CI Pipeline:**
- No CI config files present in repo (no `.github/workflows`, no GitLab CI). Tests run locally via `make test` (`go test -v ./...`) and `cd web && bun run test`.

## Environment Configuration

**Required env vars (backend):**
- `DATABASE_URL` - Postgres connection (has safe default)
- `JWT_SECRET` - token key (required in prod/staging, else dev default)
- Optional: `GO_ENV`, `PORT`, `ALLOWED_ORIGINS`, `RATE_LIMIT`, `ANONYMOUS_RATE_LIMIT`

**Secrets location:**
- Supplied at runtime via environment / `docker-compose.yml` (`JWT_SECRET: dev-secret-change-in-production` in compose — dev only). No `.env` file committed. No secrets manager integration.

## Webhooks & Callbacks

**Incoming:**
- None. No webhook receiver endpoints exist.

**Outgoing:**
- None. The application does not call external webhooks.

## Other Integrations

**Excel/CSV Export:**
- `github.com/xuri/excelize/v2` generates `.xlsx`/`.csv` timesheet, expense, and combined reports streamed to the browser (`internal/adapters/primary/http/export.go:136-178`). Output is set with `Content-Disposition: attachment` — no external service involved.

**Test Infrastructure:**
- `github.com/testcontainers/testcontainers-go` launches a throwaway Postgres container for Go integration tests (`*_integration_test.go` across `internal/core/services/*` and `internal/adapters/secondary/postgres/*`). This is a dev-time dependency only, not a runtime integration.

---

*Integration audit: 2026-08-25*
