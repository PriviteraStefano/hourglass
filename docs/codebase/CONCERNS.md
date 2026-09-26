# Codebase Concerns

**Analysis Date:** 2026-08-25

## Tech Debt

**[Migration version ledger missing]**
- Issue: The migration runner applies every `*.up.sql` / `*.down.sql` file and has no `schema_migrations` (or equivalent) table to record applied versions. Idempotency is approximated by substring-matching error text (`"already exists"` / `"does not exist"`) rather than by tracking state.
- Files: `cmd/migrate/main.go:72-97` (`migrateUp`), `cmd/migrate/main.go:99-124` (`migrateDown`), `internal/db/migrate.go:10-38` (`MigrateUp`/`MigrateDown`)
- Impact: Re-running migrations is brittle — a migration file with multiple statements that fails partway leaves the schema in an inconsistent state with no recovery path; error-message matching is driver/locale-dependent; there is no drift detection, no "apply only pending" capability, and CI cannot verify a clean target state. Rollbacks have the same fragility.
- Fix approach: Introduce a `schema_migrations (version, applied_at)` ledger; record each applied file and apply only versions above the current max in a single transaction per file. Replace string-match skips with explicit reconciliation.

**[Manual dependency wiring in `main.go`]**
- Issue: All ~30 services, repositories, and handlers are wired by hand in one 360-line function, violating the hexagonal ideal of a composed graph and making every new feature a manual edit to the entry point.
- Files: `cmd/server/main.go:35-359`
- Impact: High friction for new endpoints; easy to wire the wrong repository or forget a dependency; no compile-time graph validation.
- Fix approach: Adopt a small DI composition root (e.g., a `wire`-style factory or explicit `buildGraph()` returning a struct) so handlers/services are assembled in one typed place with clear boundaries.

**[Duplicated repository passed as two ports]**
- Issue: `tesvc.NewService(timeEntryRepo, timeEntryRepo, ...)` passes the same concrete `*TimeEntryRepository` twice to satisfy two different port interfaces (`timeEntryRepo` used as both entry repo and a second dependency).
- Files: `cmd/server/main.go:137`
- Impact: Semantically confusing; suggests an over-broad port or a missing abstraction; future refactors may not notice the two roles are the same instance.
- Fix approach: Split the port interface so a single dependency expresses both roles, or inject a dedicated repository for the second role.

**[Hand-maintained test mocks]**
- Issue: `internal/core/services/testdata/mocks.go` (1,376 lines) is a manually written mock suite. Any change to a port interface requires manual mock updates.
- Files: `internal/core/services/testdata/mocks.go`
- Impact: Test fragility — interface drift silently breaks or requires large mock edits; high maintenance cost as the port surface grows.
- Fix approach: Generate mocks (e.g., `mockgen`/interface-based generators) in a `make` target, or use lightweight fakes per test.

**[Committed generated route tree]**
- Issue: `web/src/routeTree.gen.ts` (TanStack Router generated file) is committed to the repository.
- Files: `web/src/routeTree.gen.ts`
- Impact: Low — regenerated on build; can produce noisy diffs if contributors forget to regenerate.
- Fix approach: Add to `.gitignore` or enforce regeneration in pre-commit/CI.

## Known Bugs / Fragile Areas

**[Internal error messages leaked to clients]**
- Issue: Handler error branches return the raw `err.Error()` string to the client instead of a generic message.
- Files: `internal/adapters/primary/http/auth.go:72` (`api.RespondWithError(w, http.StatusBadRequest, err.Error())`), `internal/adapters/primary/http/auth.go:220` (`"bootstrap failed: "+err.Error()`)
- Impact: Leaks internal implementation details (SQL errors, constraint names, stack context) to unauthenticated callers; aids attackers and confuses API consumers.
- Fix approach: Return stable, generic error codes/messages to clients; log the detailed error server-side only. Establish a convention that services never return user-facing error strings.

**[Export endpoints buffer all rows in memory]**
- Issue: Timesheet/expense/combined exports fetch the entire date range for an org into memory and build the full XLSX/CSV before writing. The code itself notes streaming was considered but not implemented.
- Files: `internal/adapters/primary/http/export.go:260-279` (`writeCSV` comment acknowledging non-streaming), `internal/adapters/primary/http/export.go:140-181` (`writeXLSX`), `internal/core/services/export/export.go:38` (in-memory `sort`)
- Impact: For orgs with long history or many employees, exports can exhaust memory or exceed the (unset) HTTP write timeout; no pagination/cursor.
- Fix approach: Stream rows directly from the DB cursor to `csv.Writer`/`excelize` streaming writer; cap the date range or enforce server-side pagination.

## Security Considerations

**[No request-body size limit on JSON endpoints]**
- Issue: Only the receipt-upload handler wraps the body in `http.MaxBytesReader`; all other JSON endpoints decode `r.Body` directly with no cap.
- Files: `internal/adapters/primary/http/expense.go:508` (the ONLY `MaxBytesReader`, 10 MB, for receipt upload); auth/register/login and all CRUD handlers lack it (e.g., `internal/adapters/primary/http/auth.go:40`, `:93`)
- Impact: An attacker can POST an arbitrarily large body to `/auth/register`, `/auth/login`, `/time-entries`, etc., exhausting server memory (DoS). This is the highest-impact unauthenticated abuse vector.
- Fix approach: Apply `http.MaxBytesReader(w, r.Body, limit)` (e.g., 1 MB) at the `middleware.Logging`/`APIVersion`/`CORS` layer or per handler before decode; return `413` on overflow.

**[Rate-limiter map never evicted — memory leak]**
- Issue: `RateLimiter.requests` is an in-memory `map[string]*clientInfo` with no TTL eviction, no max-size, and no cleanup of expired windows.
- Files: `internal/middleware/ratelimit.go:13-32` (struct), `internal/middleware/ratelimit.go:81-106` (`allow` only inserts/updates, never deletes)
- Impact: Every distinct client IP or user ID accumulates a permanent map entry; over months of uptime the map grows unbounded, increasing memory and map-lookup cost. Combined with the body-size gap, this weakens the server's resilience.
- Fix approach: Use a TTL cache (e.g., `golang-lru`/`ristretto` or a periodic sweeper that drops `now.After(windowEnd)` entries under a mutex), or move to a shared store (Redis) for multi-instance correctness.

**[Rate limiter is per-process only]**
- Issue: The limiter state lives in a single server instance's memory.
- Files: `internal/middleware/ratelimit.go`, wiring at `cmd/server/main.go:89-90,353`
- Impact: Behind a load balancer with N instances, each instance enforces only its own slice of traffic; effective limits are N× higher than configured; trivially bypassed by spreading requests across instances.
- Fix approach: Centralize counters in Redis (or a shared memory store) keyed by client; or enforce rate limiting at the gateway/ingress layer.

**[Dev JWT secret still boots the server]**
- Issue: When `JWT_SECRET` is unset and `GO_ENV` is not `production`/`staging`, the server logs a warning and proceeds with the hardcoded `dev-secret-change-in-production`.
- Files: `cmd/server/main.go:38-44`
- Impact: If `GO_ENV` is misconfigured (or unset) in production, tokens are signed with a publicly known key → full auth bypass via forged JWTs.
- Fix approach: Require `JWT_SECRET` unconditionally in any non-dev environment, or fail closed (refuse to boot) unless an explicit `ALLOW_INSECURE_AUTH=1` dev flag is set.

**[Refresh-token `Secure` flag trusts `X-Forwarded-Proto`]**
- Issue: `IsSecureRequest` treats the cookie as secure only when `r.TLS != nil` OR `X-Forwarded-Proto == "https"`. The header is attacker-controllable unless the edge proxy strictly overrides it.
- Files: `internal/cookies/cookies.go` (`IsSecureRequest`, `SetAccessTokenCookie`, `SetRefreshTokenCookie`)
- Impact: If deployed behind a proxy that does not sanitize `X-Forwarded-Proto`, a client can force an insecure (HTTP) `Secure=false` cookie, exposing tokens to network sniffing. Note: cookies are correctly `HttpOnly` and `SameSite=Strict`, which mitigates CSRF.
- Fix approach: Derive secure flag from the actual listener TLS state / a deployment flag rather than a client-influenced header, or have the proxy strip the header.

**[No centralized panic-recovery middleware]**
- Issue: There is no `recover()`-based middleware returning a clean 500; only tests use `recover()`.
- Files: no `recover()` in non-test `.go` code (confirmed via grep); handler chain at `cmd/server/main.go:356`
- Impact: An unexpected panic in a handler yields Go's default connection-closing behavior with a stack trace in logs and no structured 500; partial writes can corrupt the JSON envelope for the client.
- Fix approach: Add a `Recovery` middleware wrapping the mux that converts panics into `500 {error:"internal"}` and logs the stack.

## Performance Bottlenecks

**[HTTP server has no timeouts and no graceful shutdown]**
- Issue: `stdhttp.ListenAndServe(":"+port, handler)` is used with no `http.Server` configuration (no `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) and no signal handling for `SIGTERM`/`SIGINT`.
- Files: `cmd/server/main.go:355-359`
- Impact: Vulnerable to Slowloris (slow-reading clients hold connections indefinitely); on deploy/restart, in-flight requests are dropped rather than drained.
- Fix approach: Construct `&http.Server{ReadTimeout, WriteTimeout, IdleTimeout, Handler}` and run `Shutdown(ctx)` on signal; consider `graceful`/`http.Server` shutdown pattern.

**[DB connection pool size not configured]**
- Issue: `NewPool` builds a `pgxpool` from `DATABASE_URL` without setting `MaxConns`; pgxpool defaults to 4 when unspecified.
- Files: `internal/db/db.go:41-72`
- Impact: Only 4 DB connections are available to serve all authenticated traffic; under concurrent load this serializes queries and can exhaust quickly, causing latency spikes and request queueing. (Note: the legacy `db.New` path sets `SetMaxOpenConns(25)` at `internal/db/db.go:24`, but the server uses `NewPool`, not `New`.)
- Fix approach: Set `connConfig.MaxConns` from an env knob (e.g., `DB_MAX_CONNS`, default ~10–20) and validate under load.

**[In-memory export sorting/aggregation]**
- Issue: Export service sorts and assembles all rows in Go memory before the handler writes them.
- Files: `internal/core/services/export/export.go:38` (`sort.Slice` over full result set), `internal/adapters/primary/http/export.go`
- Impact: CPU/memory spikes on large orgs; compounds the no-timeout issue above.
- Fix approach: Push ordering/filtering into SQL and stream results (see export item above).

## Repo Hygiene

**[Build binaries committed to git]**
- Issue: Compiled binaries `server` (~23.7 MB) and `migrate` (~7.5 MB) are tracked in the repository. `.gitignore` excludes `main` and `bin/` but not `server`/`migrate`.
- Files: `server` (root), `migrate` (root) — both present in `git ls-files`
- Impact: Bloats the repository history and clones; risk of stale/secret-leaking binaries; CI should build, not commit, artifacts.
- Fix approach: Add `server` and `migrate` to `.gitignore` and `git rm --cached` them (and rewrite history if desired).

## Test Coverage Gaps

**[Frontend test coverage is thin and concentrated]**
- Issue: The web app has 146 `.tsx` files but only 16 test files (`.test.tsx`/`.test.ts`), mostly covering a few pages (`approvals-page`, `today-page`, `register-form`).
- Files: `web/src/**` (146 `.tsx` vs 16 `*_test.ts(x)`)
- Impact: Most route components, dialogs, and hook logic are untested; regressions in approval flows, forms, and data fetching would go unnoticed. The backend, by contrast, has unit+integration tests for every service and many repositories.
- Priority: Medium — add component/hook tests for the highest-traffic pages (time-entries, expenses, contracts, org-hierarchy) and the React Query `api<T>()` client + refresh-on-401 path.

**[No end-to-end coverage evidence in CI config]**
- Issue: Playwright e2e (`web/e2e`) is referenced in AGENTS.md but not verified present/run in `.github` workflows.
- Files: `web/e2e` (referenced), `.github/` workflows
- Impact: No guaranteed automated verification of the full auth→data flow across deploys.
- Fix approach: Wire Playwright into CI and assert the login → time-entry → approve happy path.

## Missing Critical Features / Fragile

**[API versioning middleware is decorative]**
- Issue: `middleware.APIVersion` parses an `Accept` version and stores it in context, but no handler reads `VersionKey` to alter behavior; there is a single `v1` and the version is never enforced or routed.
- Files: `internal/middleware/version.go:9-45`
- Impact: Low — dead/aspirational code; suggests an intended contract-versioning strategy that is not implemented, which could mislead future work.
- Fix approach: Either implement version-based routing/behavior or remove the middleware to avoid implying a guarantee.

---

*Concerns audit: 2026-08-25*
