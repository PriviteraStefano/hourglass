# Codebase Concerns

**Analysis Date:** 2026-08-12

## Tech Debt

**Dual auth stacks (legacy + hexagonal):**
- Issue: Two independent token/auth implementations coexist. `internal/middleware/middleware.go:23` gates routes with the legacy `internal/auth/auth.go` `Service` (plain JWT), while login/register/refresh run through the hexagonal `internal/core/services/auth/auth.go` Service. `internal/auth/token_service.go` exists purely as an adapter shim (ports.TokenService) over the legacy service.
- Files: `internal/auth/auth.go`, `internal/auth/token_service.go`, `internal/core/services/auth/auth.go`, `internal/middleware/middleware.go`, `cmd/server/main.go:47-81`
- Impact: Token expiry constants are duplicated (`AccessTokenExpiry = 15 * time.Minute` in `internal/auth/auth.go:15` and again in `internal/core/services/auth/auth.go`); the legacy path will drift from the hex path (e.g., refresh-token reuse detection exists only in the hex service). Middleware never sees membership deactivation or role changes until a new token is issued.
- Fix approach: Fold `internal/auth.Service` into the hex service; make `middleware.Auth` validate via `ports.TokenService`; delete `internal/auth/token_service.go` shim.

**Dead legacy SurrealDB code:**
- Issue: `internal/models/surreal_models.go` (251 lines of SurrealDB-era structs: `SurrOrganization`, `RecordID`, etc.) has zero references anywhere in Go code. `scratch.surql` sits at the repo root.
- Files: `internal/models/surreal_models.go`, `scratch.surql`
- Impact: Confusing for new contributors (suggests SurrealDB is still in use — it was replaced by PostgreSQL); wasted maintenance surface.
- Fix approach: Delete the file and `scratch.surql`. Note `internal/models/models.go` itself IS still imported by `internal/core/domain/*` and ports — the dual model world (`internal/models` + `internal/core/domain/*`) is intentional during migration, but `surreal_models.go` is pure dead weight.

**Fragile migration tooling:**
- Issue: `cmd/migrate/main.go:86-92` implements "idempotency" by executing every `*.up.sql` file on every run and swallowing errors whose message contains `"already exists"`. There is no applied-migrations tracking table. Any error that happens to contain "already exists" (constraint/index name collisions in NEW migrations) is silently skipped; conversely `011_activity_ontology.up.sql` (unguarded `RENAME COLUMN`/`DROP COLUMN`) hard-fails on re-run, and `000_full_schema.up.sql` `CREATE INDEX IF NOT EXISTS` statements still reference columns that 011 renamed (e.g. `idx_working_groups_subproject_id` → `subproject_id` no longer exists).
- Files: `cmd/migrate/main.go`, `migrations/000_full_schema.up.sql`, `migrations/011_activity_ontology.up.sql`
- Impact: Fresh-DB works; re-run aborts with exit 1. Documented in `.planning/quick/260801-o06-failed-to-initialize-postgresql-pool-pas/deferred-items.md` as pre-existing and out of scope at the time.
- Fix approach: Add a real `schema_migrations` table to `cmd/migrate`, or guard the 000/011 statements. Also: two migration mechanisms coexist — `docker-compose.yml` mounts `./migrations` into `docker-entrypoint-initdb.d` while `cmd/migrate` is the documented path (`Makefile` `migrate-up`).

**400-line route-wiring monolith with compile-forced invariants:**
- Issue: `cmd/server/main.go` (401 lines) manually constructs 15+ services and registers 100+ routes. Comments encode hard-won invariants as prose ("Pitfall 5", "Pitfall 6 — ServeMux most-specific-wins, do NOT remove the typed wildcard registrations", "D-G parity — no second instances"). Old-style handlers (`time_entry.go`, `expense.go`) coexist with hexagonal handlers; `timeEntryRepo`/`expenseRepo` are passed straight into services.
- Files: `cmd/server/main.go`
- Impact: Any refactor that reorders construction or drops a route can silently break subtle routing/parity behavior; the file is already heavily commented because the structure cannot express its own constraints.
- Fix approach: Route registration table (method, path, handler, middleware) in one slice; service construction via a small composition root per domain; add a test asserting the pinned route set (some coverage exists in handler integration tests).

**Giant hand-maintained mocks and test files:**
- Issue: `internal/core/services/testdata/mocks.go` is 1422 lines of hand-written mocks; several repository test files exceed 1000 lines (`direction_repository_test.go` 1573, `ticket_repository_test.go` 1105, `availability_repository_test.go` 1065).
- Files: `internal/core/services/testdata/mocks.go`, `internal/adapters/secondary/postgres/direction_repository_test.go`, `internal/adapters/secondary/postgres/ticket_repository_test.go`
- Impact: Every port change forces hand edits across the mocks; tests are costly to review and maintain.
- Fix approach: Generate mocks (mockgen/mockery) for `internal/core/ports/*`.

**Testcontainers dependency for repository tests:**
- Issue: 20 test files spin up a Docker Postgres container per package (`internal/adapters/secondary/postgres/test_setup.go` — `sync.Once` per package, `postgres:16-alpine`). `make test` (all `go test ./...`) silently skips these if Docker is unavailable, so a laptop without Docker shows green while CI/other machines fail.
- Files: `internal/adapters/secondary/postgres/test_setup.go`, `Makefile` (`test:` target)
- Impact: Local/CI environment drift; slow test runs.
- Fix approach: Keep, but make the skip loud (`t.Skip` → log warning) or gate with a `-tags=integration` flag; run `go test` in CI with Docker guaranteed.

**Stray build artifacts at repo root:**
- Issue: Compiled binaries `main` (10 MB), `server` (24 MB), `migrate` (7 MB) sit at the repo root; `.gsd-worktrees/M001` is a full duplicate checkout of the codebase inside the repo (gitignored, 0 tracked files, but pollutes every glob/grep and disk usage).
- Files: `/main`, `/server`, `/migrate`, `.gsd-worktrees/M001/`
- Impact: Repo hygiene; tooling scans the duplicate tree unless explicitly excluded.
- Fix approach: Delete the binaries (they are gitignored); configure tooling to skip `.gsd-worktrees/`.

## Known Bugs

**Cross-organization time entry read (IDOR):**
- Symptoms: `GET /time-entries/{id}` returns any entry from any organization. The handler passes only the entry ID (`internal/adapters/primary/http/time_entry.go:85-105` → `service.Get(ctx, entryID)` → `TimeEntryRepository.GetByID` with `WHERE te.id = $1`, `internal/adapters/secondary/postgres/time_entry_repository.go:169-181`). No org or owner filter anywhere on the read path.
- Files: `internal/adapters/primary/http/time_entry.go:85`, `internal/core/services/time_entry/time_entry.go:39`, `internal/adapters/secondary/postgres/time_entry_repository.go:169`
- Trigger: Any authenticated user enumerates UUIDs of other orgs' entries.
- Workaround: None.
- Fix approach: Mirror the expense pattern — `ExpenseHandler.Get` checks `e.OrgID != orgID` in the handler (`internal/adapters/primary/http/expense.go:122-125`) — and/or thread `orgID` into `Service.Get`/`GetByID`.

**Receipt upload missing ownership/org check (IDOR write):**
- Symptoms: `POST /expenses/{id}/receipt` (`internal/adapters/primary/http/expense.go:480-548`) parses an expense ID, uploads a file, then calls `SetReceiptURL` (`internal/core/services/expense/expense.go:404-414`) which does `repo.GetByID(ctx, id)` → `UPDATE` with no org/user validation (`expense_repository.go:176-188`). Any authenticated user can overwrite the receipt URL of any expense in any org and store a 10 MB file keyed to an arbitrary expense ID (unbounded disk usage by a single user across many IDs).
- Files: `internal/adapters/primary/http/expense.go:480`, `internal/core/services/expense/expense.go:404`, `internal/adapters/secondary/postgres/expense_repository.go:176`
- Fix approach: Pass `orgID`/`userID` into `SetReceiptURL`; verify ownership before creating the upload directory; enforce a per-user upload cap.

**Migration 011 cycle test failure (open deferred item):**
- Symptoms: `TestMigration011_ActivityOntology_UpDownUpCycle` fails — `021_direction_rows.up.sql` errors `relation "activities" does not exist` (42P01) because the test's pre-state (000-010) doesn't create `activities`, and the 021/022 files were never added to the test's skip list.
- Files: `internal/adapters/secondary/postgres/activity_ontology_migration_test.go`, `migrations/021_direction_rows.up.sql`, `migrations/022_org_settings.up.sql`
- Trigger: Running the full `go test ./...` suite.
- Status: Open, tracked in `.planning/phases/13-direction-backend-the-plan-plane/deferred-items.md`.
- Fix approach: Add 021/022 to the skip list (precedent `ae7f4a6` for 12-01).

**Rate limiter limit-bump bug:**
- Symptoms: `internal/middleware/ratelimit.go:85-87` — `if limit > info.limit { info.limit = limit }` permanently raises the window's limit to the highest tier seen for that key. A key (IP) that once made an authenticated request keeps the 100/min authenticated budget for anonymous traffic for the rest of the window. The window resets only when the client returns after expiry, so a single authenticated burst raises the anonymous cap for everyone on a shared IP.
- Files: `internal/middleware/ratelimit.go:68-88`
- Fix approach: Store the limit per-request-class or recompute rather than mutating the stored limit.

**Rate limiter ignores proxies — shared-bucket DoS in demo deployment:**
- Symptoms: `ratelimit.go:47-58` keys anonymous clients on `RemoteAddr`. Behind the Caddy/cloudflared reverse proxy (`deploy/demo/docker-compose.yml`) every anonymous visitor resolves to the proxy address and shares ONE 20 requests/minute bucket (outer limiter, `cmd/server/main.go:388-394`). A single client hammering any public route throttles the entire demo site for all users; no `X-Forwarded-For` handling exists.
- Files: `internal/middleware/ratelimit.go:47`, `cmd/server/main.go:388`, `deploy/demo/docker-compose.yml`
- Fix approach: Parse `X-Forwarded-For` (first untrusted hop) or configure per-route limits so the outer limiter doesn't apply site-wide behind a proxy.

**`POST /expenses` reported broken (verify):**
- Symptoms: `web/e2e/helpers.ts` states "`POST /expenses` is still broken backend-side (deferred-items.md)", so the expenses e2e suite seeds rows via direct SQL instead of the API.
- Files: `web/e2e/helpers.ts`
- Impact: If still true, expense creation fails for real users; if stale, the comment misleads future test writers.
- Action: Reproduce `POST /expenses` with the current backend before any expense-related phase.

## Security Considerations

**Default JWT secret reachable in compose-deployed stack:**
- Risk: `cmd/server/main.go:38-45` falls back to `dev-secret-change-in-production` unless `GO_ENV=production/staging` is set. The root `docker-compose.yml` (the documented `make docker-up` path) sets `JWT_SECRET: dev-secret-change-in-production` explicitly and never sets `GO_ENV` — anyone who deploys with the root compose gets forgeable auth tokens.
- Files: `cmd/server/main.go:38`, `docker-compose.yml`
- Current mitigation: `deploy/demo/docker-compose.yml` sets `GO_ENV=production` and `${JWT_SECRET}` — the demo path is correct.
- Recommendations: Make `main.go` refuse to boot with the default secret regardless of env when listening on a non-localhost interface, or add a hard check in `docker-compose.yml` via `${JWT_SECRET:?must be set}`.

**Self-enrollment into any org by knowing its UUID:**
- Risk: `POST /auth/register` maps the client-supplied `invite_code` directly to an org UUID (`internal/adapters/primary/http/auth.go:63`) and creates a membership with no invitation-record validation (`internal/core/services/auth/auth.go:187-205`). There is no check that an invitation exists or was accepted; an attacker who learns an org UUID (via exports, screenshots, leaked URLs, or the `bootstrap`/`memberships` flows) self-joins as `employee`.
- Files: `internal/adapters/primary/http/auth.go:33,63`, `internal/core/services/auth/auth.go:133-243`
- Current mitigation: Org UUIDs are 122-bit random; rate limiter caps registration at 5/min/IP.
- Recommendations: Validate the invite code against the `invitations` table (the `POST /invitations/accept` flow exists — wire it into register), or scope the org-join register path to invitation tokens only.

**Unbounded JSON request bodies:**
- Risk: 29 handler files decode JSON with `json.NewDecoder(r.Body).Decode(...)` and no `http.MaxBytesReader` — only the two upload endpoints cap bodies (`expense.go:492` 10 MB, `availability_handler.go:242` 5 MB). Any authenticated user can send multi-GB bodies to any mutating route and exhaust memory.
- Files: all files under `internal/adapters/primary/http/*.go` (e.g. `time_entry.go:202`, `contract.go`, `unit.go`, `activity_handler.go`)
- Recommendations: Global `http.MaxBytesReader` wrapper in the outer middleware chain (`cmd/server/main.go:397`), e.g. 1 MB default.

**CSV/XLSX formula injection in exports:**
- Risk: `internal/adapters/primary/http/export.go` writes user-controlled fields (`Description`, `Status`, `Employee`) raw into CSV and XLSX via `encoding/csv` and `excelize`. A description starting with `=`, `+`, `-`, `@` executes as a formula when the exported file is opened in Excel/Sheets.
- Files: `internal/adapters/primary/http/export.go:266-282,135-181`
- Recommendations: Prefix dangerous cells with `'` or quote as text (excelize: `SetCellStr`).

**Rate limiter map never evicts:**
- Risk: `internal/middleware/ratelimit.go:15` — `requests` map entries are replaced only when the same key returns after window expiry; distinct IPs accumulate forever → unbounded memory growth over a long-lived server.
- Files: `internal/middleware/ratelimit.go`
- Recommendations: Periodic sweep of expired entries (the window is 1 minute).

**Upload validation is extension-only:**
- Risk: Receipt/certificate uploads check file extension, not content (`internal/adapters/primary/http/expense.go:507`, `availability_handler.go:255`). Files are stored with server-generated UUID names (no path traversal — good), but a polyglot/HTML-with-.pdf file is accepted. Receipt files land on local disk under `uploads/` with no serving route, no retention policy, and no cleanup job — orphaned files accumulate.
- Files: `internal/adapters/primary/http/expense.go:513-535`, `uploads/`
- Recommendations: Sniff magic bytes; either serve receipts through an auth-gated endpoint or remove the dead `receipt_url` plumbing.

**Trust of `X-Forwarded-Proto` for Secure cookies:**
- Risk: `internal/cookies/cookies.go` `IsSecureRequest` trusts the client-supplied `X-Forwarded-Proto` header; if the app is ever directly reachable (root `docker-compose.yml` publishes `8080:8080`), an attacker can make the server set Secure cookies over plain HTTP, causing the browser to drop them — a session-nuking annoyance; in the reverse direction a MITM on plain HTTP can set cookies the browser treats as secure.
- Files: `internal/cookies/cookies.go:62-64`
- Recommendations: Trust the header only from known proxy IPs or gate on `r.TLS != nil` for direct connections.

## Performance Bottlenecks

**Exports buffer everything in memory:**
- Problem: CSV and XLSX exports fetch all rows into a slice before writing (`export.go` `writeCSV`/`writeXLSX`; the code comment at `export.go:261-266` acknowledges this). For orgs with large entry volumes, each export spikes memory; `writeCSV` also ignores `csv.Writer` errors.
- Files: `internal/adapters/primary/http/export.go:266-282,135-181`
- Improvement path: Stream rows from the DB cursor into the writer (the comment outlines this); write an integration test for a large fixture first.

**Single global mutex on every request:**
- Problem: `RateLimiter.allow` takes an exclusive write lock per request (`ratelimit.go:69`) for the whole app; at current scale negligible, becomes a bottleneck under load. Combined with the unbounded map, it's a scaling trap.
- Files: `internal/middleware/ratelimit.go:68`
- Improvement path: Sharded or atomic-window limiter (e.g. per-key last-reset with atomic ints).

**Repository/service complexity hotspots:**
- Problem: `internal/core/services/direction/direction.go` (888 lines), `internal/adapters/secondary/postgres/direction_repository.go` (872), `ticket_repository.go` (861), `activity_repository.go` (835), `availability_repository.go` (774). These are the newest features (direction/availability/coverage) with the richest logic; large single files invite duplicated query fragments (e.g. the `activity_adoptions`/`contract_adoptions` org-scope subqueries appear in both `activity_repository.go:64` and `contract_repository.go:60`).
- Files: `internal/adapters/secondary/postgres/*.go`, `internal/core/services/direction/direction.go`
- Improvement path: Extract shared org-scope predicates; keep new routes as small focused methods.

## Fragile Areas

**`cmd/server/main.go` wiring:**
- Files: `cmd/server/main.go`
- Why fragile: Construction order encodes invariants (orgsettings before direction; availability before org service; shared `routingSvc` instance enforced by comments "no second instances — D-G parity"). ServeMux literal-vs-wildcard coexistence is load-bearing (`GET /organizations/settings` vs `GET /organizations/{id}/settings`, documented "Pitfall 6").
- Safe modification: Change one service at a time; run the full `go test ./...` plus e2e (`web/e2e`) after any wiring change; prefer adding routes to the existing block rather than restructuring.

**Migration chain:**
- Files: `migrations/*.up.sql`, `cmd/migrate/main.go`
- Why fragile: Error-string-based skipping means a new migration whose first statement collides with an existing object is silently treated as "already applied" — the rest of its file never runs. Conversely 011-style renames abort re-runs.
- Safe modification: New migrations must be additive (CREATE IF NOT EXISTS / ALTER guarded); never modify already-applied files (000-022).

**Testcontainers-based repository tests:**
- Files: `internal/adapters/secondary/postgres/test_setup.go`, 20 `*_test.go` files in `internal/adapters/secondary/postgres/`
- Why fragile: Require Docker; per-package `sync.Once` container; if the container image (`postgres:16-alpine`) is unavailable or Docker is misconfigured, tests skip silently.
- Test coverage: Repository behavior itself is well covered; handler integration tests exist (`internal/adapters/primary/http/handler_integration_test.go`).

**Frontend auth guard:**
- Files: `web/src/routes/_authenticated.tsx`, `web/src/routes/_auth.tsx`, `web/src/lib/api.ts`
- Why fragile: The guard fetches `/auth/me` via `client.fetchQuery` on every protected navigation (shared `queryClient`, `staleTime: 30000`, `retry: false` in `web/src/lib/query-client.ts`); `web/src/lib/api.ts` does a single refresh-on-401 with a module-level `refreshPromise` — any regression in the refresh path (e.g. cookie attribute changes) breaks the whole app's auth with confusing redirect loops (the code comments document a past infinite-loop bug).
- Safe modification: Keep the redirect loop regression test (`web/e2e/auth.spec.ts`); don't change cookie names/attributes without syncing `web/src/lib/api.ts` and `internal/cookies/cookies.go`.

**e2e suite runtime requirements:**
- Files: `web/e2e/helpers.ts`, `web/e2e/*.spec.ts`
- Why fragile: Requires a running backend with raised limits (`RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server` per `helpers.ts`), the dockerized Postgres (`hourglass-postgres`) with seed data, and direct-SQL seeding — the public API cannot produce all workflow states. Any schema change breaks the seeding SQL.

## Scaling Limits

**Rate limiter state:** in-memory per-instance (`internal/middleware/ratelimit.go`) — a multi-instance deployment resets limits per instance and multiplies the effective limits; no shared store. Current capacity is single-instance by design.
**Uploads:** local disk `uploads/` with no storage abstraction (`internal/adapters/primary/http/expense.go:513`) — requires volume management per instance; multi-instance deploys would need shared storage (S3/MinIO).

## Dependencies at Risk

**testcontainers-go v0.42.0** (`go.mod`) — pulls a large transitive graph (docker, otel, gopsutil); pinned, but every bump churns the indirect set. Impact: test-only.
**excelize/v2 v2.11.0** — heavy dependency for simple tabular exports; if exports grow (streaming), excelize stays, otherwise a lighter writer (e.g. `github.com/xuri/xgen` not needed — plain `encoding/csv` + a zip writer) could replace it.
**TypeScript 7.0.2 (native) + Vite 8 + oxlint 1.76** (`web/package.json`) — fast-moving frontend toolchain; `tsc -b` output may differ between versions; keep the lockfile and CI versions pinned.

## Missing Critical Features

**Receipts cannot be retrieved:**
- Problem: `receipt_url` is stored on expenses and files land in `uploads/`, but no route serves them (no `http.FileServer` or receipt endpoint in `cmd/server/main.go`). The receipt feature is write-only.
- Files: `internal/adapters/primary/http/expense.go:534-538`, `cmd/server/main.go`
- Blocks: Finance users cannot view receipts during approval; orphaned files accumulate.

## Test Coverage Gaps

**Time-entry GET IDOR is untested:**
- What's not tested: cross-org access on `GET /time-entries/{id}` (and receipt-upload authorization). Existing handler tests assert happy paths and 404/403s within-org only.
- Files: `internal/adapters/primary/http/time_entry.go:85`, `internal/adapters/primary/http/expense.go:480`
- Risk: The IDORs shipped unnoticed — proof the authorization matrix lacks negative cross-org tests.
- Priority: High

**No e2e for direction/coverage/availability:**
- What's not tested: `web/e2e/` covers activities, approvals, auth, contracts, customers, error-boundary, expenses, org-hierarchy, time-entries, working-groups — but no direction, coverage, or availability specs, despite those being the newest backend domains.
- Files: `web/e2e/` directory listing
- Risk: Regressions in the newest workflows pass CI.
- Priority: Medium

**No coverage gate:**
- What's not tested: `make test` runs `go test -v ./...` with no `-cover` target; frontend `vitest run` has no coverage config either. No regression guard against coverage drops.
- Files: `Makefile`, `web/package.json`
- Risk: Silent coverage erosion.
- Priority: Low

**`POST /expenses` create path:**
- What's not tested: expense creation through the API end-to-end (the e2e suite seeds via SQL because of the reported backend breakage — see Known Bugs).
- Files: `web/e2e/expenses.spec.ts`, `web/e2e/helpers.ts`
- Risk: A broken create endpoint with no API-level test.
- Priority: High (after verifying whether the bug still exists)

---

*Concerns audit: 2026-08-12*
