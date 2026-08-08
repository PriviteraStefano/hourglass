# Codebase Concerns

**Analysis Date:** 2026-08-08

## Tech Debt

**SurrealDB legacy dead code:**
- Issue: `internal/models/surreal_models.go` (251 lines) contains SurrealDB-era record types, status constants, and org schema structs. Nothing in the repo imports it (`rg "surreal" internal/` returns zero non-test hits). The SurrealDB migration is complete (PostgreSQL via pgxpool is the only storage path), yet the file remains.
- Files: `internal/models/surreal_models.go`
- Impact: Confusion for agents/engineers grepping for canonical types; risk of someone accidentally wiring legacy types into new code.
- Fix approach: Delete the file (plus tracked root artifacts `.graphify_cached.json`, `.graphify_semantic_new.json`, `.graphify_uncached.txt`, `scratch.surql`, and stale plans `plans/surrealdb-handler-migration.md`, `plans/old/surrealdb-implementation-plan.md`, `hourglass-vault/LEGACY/old/design/files/surrealdb-schema.sql`).

**Dead legacy DB layer (`database/sql` + lib/pq):**
- Issue: `internal/db/db.go` defines `DB` (wrapping `*sql.DB`), `New()`, and `Close()` built on `github.com/lib/pq`. `New()` has zero callers — every repository uses the pgxpool path (`NewPool()`), which is the only one wired in `cmd/server/main.go:46`. `lib/pq` remains in `go.mod` as a direct dependency solely for this dead path.
- Files: `internal/db/db.go`, `go.mod`
- Impact: Dead code + an unnecessary direct dependency; `internal/db` has no test files.
- Fix approach: Remove `DB`, `New()`, `Close()`, the `database/sql` + `lib/pq` imports, and drop `github.com/lib/pq` from `go.mod` (`go mod tidy`).

**Committed build binaries at repo root:**
- Issue: `server` (24 MB) and `migrate` (7.8 MB) compiled binaries are tracked in git. `.gitignore` covers `main` and `bin/` but not these two.
- Files: `server`, `migrate`, `.gitignore`
- Impact: 30+ MB of binary churn in history, merge noise, and a pattern that encourages committing build output.
- Fix approach: `git rm --cached server migrate`, add them to `.gitignore`, add a Makefile clean target.

**`uploads/` directory not gitignored:**
- Issue: Receipt uploads are written to `uploads/receipts/{org_id}/{expense_id}/` (`internal/adapters/primary/http/expense.go:514-522`), but `uploads/` is absent from `.gitignore`. Any developer running the server and uploading a receipt will see the file in `git status` — risk of committing real user documents (receipts) to the repo.
- Files: `uploads/`, `.gitignore`, `internal/adapters/primary/http/expense.go`
- Impact: Privacy incident if a receipt is committed; repo bloat.
- Fix approach: Add `uploads/` (with a `.gitkeep`) to `.gitignore`.

**Dual model layers (transitional):**
- Issue: Services import both `internal/models` (legacy constants/structs — e.g., `models.RoleFinance`, `models.StatusSubmitted` in `internal/core/services/activity/activity.go:173`) and `internal/core/domain/*` (new per-aggregate domain types). The same concepts (roles, statuses) exist in both worlds; new features must know which one is canonical.
- Files: `internal/models/models.go`, `internal/core/domain/*`, `internal/core/services/*`
- Impact: Constant drift risk (two sources of truth for role/status strings); unclear guidance for new code.
- Fix approach: Track role/status constants into `internal/core/domain/` and re-export (or migrate call sites) from `internal/models` until the legacy file is emptied.

**`cmd/server/main.go` manual wiring monolith:**
- Issue: 318 lines of hand-rolled dependency construction and `mux.HandleFunc` registration for ~70 routes across 13 feature areas. Every new feature appends to it; there is no route registry or per-feature wiring table.
- Files: `cmd/server/main.go`
- Impact: Merge conflicts, registration drift (a route added in the handler but forgotten in `main.go` fails silently), and 0% test coverage of the wiring.
- Fix approach: Extract per-feature `RegisterRoutes(mux, deps)` functions or a small `wire` package; add a smoke test that asserts every handler is registered.

**Broken `make db-init` target + migration numbering gaps:**
- Issue: `Makefile` `db-init` runs `psql < migrations/001_init.up.sql`, but no `001_init.up.sql` exists — migration numbers jump from `000_full_schema` to `004_...` (001, 002, 003, 007 were removed/renamed during development). The only schema source is `000_full_schema.up.sql` plus 17 patch migrations.
- Files: `Makefile`, `migrations/`
- Impact: `make db-init` fails; anyone reading migration numbering could think migrations are missing.
- Fix approach: Delete the stale `db-init` target or point it at `000_full_schema.up.sql`.

**Unused frontend dependency:**
- Issue: `@tanstack/react-form` is declared in `web/package.json` but never imported — all forms use `react-hook-form` (8 files) or `@tanstack/react-form` is absent (zero matches).
- Files: `web/package.json`
- Impact: Bundle-size surface and dependency confusion (two form libraries listed).
- Fix approach: Remove `@tanstack/react-form`; keep `react-hook-form` + `zod` as the form stack.

**Ignored errors in export writer:**
- Issue: `internal/adapters/primary/http/export.go:277-281` discards csv.Writer errors (`_ = writer.Write(...)`, `_ = writer.Write(header)`) and `writeXLSX` ignores `excelize` errors — a mid-stream write failure produces a truncated download that looks successful.
- Files: `internal/adapters/primary/http/export.go`
- Impact: Silent data loss on partial downloads (disk full, client disconnect).
- Fix approach: Check `writer.Error()` after `Flush()` and surface a 500 when non-nil.

## Known Bugs

**Register "join with invite code" flow is broken end-to-end:**
- Symptoms: A user choosing "Join an existing organization with an invite code" on the register form gets an account with NO organization and NO session. The UI navigates to `/` (`web/src/routes/_auth/register/-components/register-form.tsx` onSubmit → `navigate({ to: "/", replace: true })`), the `/auth/me` guard 401s, and the user is bounced to `/login`, where login succeeds but there is no membership — a dead-end account. No error is ever surfaced.
- Files: `internal/adapters/primary/http/auth.go:62` (`OrgID: req.InviteCode`), `internal/core/services/auth/auth.go:170-171` (`parsed, _ := uuid.Parse(req.OrgID)` — error swallowed), `internal/core/services/invitation/invitation.go:98-105` (`generateInviteCode` returns 6-char alphanumeric, never a UUID), `web/src/routes/_auth/register/-components/register-form.tsx:35-108`
- Trigger: Register with `invite_code` set. `uuid.Parse` always fails → `orgID = uuid.Nil` → membership skipped, token/refresh never issued.
- Workaround: None via UI. The separate `POST /invitations/accept` endpoint exists (`internal/adapters/primary/http/invitation.go`) but is not wired into the register form.
- Fix approach: Pass `InviteCode` through to a real invite validation path (or remove the "join" branch from the UI until the backend supports it). At minimum: validate the invite in `Register` and attach the user to the invited org with the invited role.

**Second 401 after refresh throws the wrong error type:**
- Symptoms: In `web/src/lib/api.ts`, after a successful `/auth/refresh`, the retried request that still returns 401 falls into the generic `!res.ok` branch and throws a plain `Error`, not `UnauthorizedError`. Route guards catch `UnauthorizedError` to redirect to `/login`; a plain `Error` surfaces via the error boundary instead.
- Files: `web/src/lib/api.ts:72-87`
- Trigger: Access token refreshed, then the retried call still 401s (e.g., user deactivated between calls).
- Workaround: None; user must manually navigate to `/login`.
- Fix approach: Re-check `res.status === 401` after the retry and throw `UnauthorizedError`.

**Bootstrap TOCTOU race:**
- Symptoms: `POST /auth/bootstrap` (unauthenticated, unrate-limited, `cmd/server/main.go:95`) checks `AnyExists` in the service (`internal/core/services/auth/auth.go:411`) and fails only if a user exists at check time. Two concurrent bootstraps can both pass the check and create two "bootstrap" orgs/admin users.
- Files: `internal/core/services/auth/auth.go:404-417`, `internal/adapters/primary/http/auth.go:199-229`
- Trigger: Two simultaneous bootstrap requests against a fresh DB.
- Impact: Duplicate bootstrap orgs — confusing initial state; low severity (deployment-time only).
- Workaround: Bootstrappers are typically run once; acceptable for now.
- Fix approach: Serialize with a DB unique constraint on the first user or an advisory lock (same pattern as WR-03 period-close lock).

## Security Considerations

**Access tokens outlive membership/deactivation changes:**
- Risk: `middleware.Auth` (`internal/middleware/middleware.go:23-44`) validates only JWT signature + expiry; it never checks the user's active flag or membership status. A deactivated user keeps calling APIs for up to 15 minutes, and role changes only take effect after token refresh. The refresh path DOES re-check membership (`internal/core/services/auth/auth.go:377-391`) — so the hole is bounded to the access-token window, but every org/role-sensitive check ultimately relies on 15-minute-old claims.
- Files: `internal/middleware/middleware.go`, `internal/core/services/auth/auth.go`
- Current mitigation: 15-minute access token TTL (`internal/auth/auth.go:15`); refresh-time membership revalidation.
- Recommendations: For high-sensitivity actions (approve/reject, role changes, deactivate), re-fetch member status in the service; or add an `is_active` + membership-version claim check in `Auth` when a DB lookup is acceptable.

**Default JWT secret ships in compose:**
- Risk: `cmd/server/main.go:40-41` falls back to `"dev-secret-change-in-production"`, and `docker-compose.yml:27` sets `JWT_SECRET` to the same value. Anyone deploying via `docker-compose up` without overriding the env var runs with a publicly known signing key — attacker can forge access tokens for any user/role.
- Files: `cmd/server/main.go`, `docker-compose.yml`, `deploy/demo/docker-compose.yml`
- Current mitigation: Hard fatal when `GO_ENV=production|staging` and `JWT_SECRET` is empty (`cmd/server/main.go:36-39`).
- Recommendations: Make `docker-compose.yml` (and `deploy/demo/docker-compose.yml`) read `JWT_SECRET` from the environment with no default; fail loudly at startup regardless of `GO_ENV`.

**Rate limiter state never evicted (unbounded memory):**
- Risk: `internal/middleware/ratelimit.go` keeps one `clientInfo` per unique IP/user key forever (`rl.requests` map); entries are only created, never deleted. An attacker rotating IPs (or simply organic traffic over time) grows the map without bound. A single global mutex also serializes every request's bookkeeping.
- Files: `internal/middleware/ratelimit.go`
- Current mitigation: None — no eviction, no cap.
- Recommendations: Evict entries whose `windowEnd` is in the past during `allow()` (opportunistic sweep every N requests), cap map size, and shard by key hash.

**Receipt uploads: extension-only validation, no lifecycle:**
- Risk: `ReceiptUpload` (`internal/adapters/primary/http/expense.go:479-530`) validates only the file extension (`.pdf/.jpg/.jpeg/.png`), not magic bytes/MIME; no virus scanning; files are stored on the app filesystem and never deleted when the expense is deleted (check `Delete` in `internal/core/services/expense/expense.go` — no upload cleanup path).
- Files: `internal/adapters/primary/http/expense.go`, `internal/core/services/expense/expense.go`
- Current mitigation: 10 MB `MaxBytesReader`, allowlisted extensions, UUID-renamed files, org-scoped directories, path components are server-generated UUIDs (no traversal).
- Recommendations: Verify content magic bytes, delete the directory on expense delete, and consider offloading to object storage.

**Open self-registration:**
- Risk: `POST /auth/register` is public and lets anyone create an org (spam vector, org-name squatting). Invite-gated onboarding exists (`internal/adapters/primary/http/invitation.go`) but is not required — registration bypasses invitations entirely.
- Files: `cmd/server/main.go:90`, `internal/core/services/auth/auth.go:133-151`
- Current mitigation: Per-IP rate limit 5/min on `/auth/register` and `/auth/login` (`cmd/server/main.go:87`).
- Recommendations: Acceptable for the current demo-stage product; flag for when the product opens up.

**`/auth/bootstrap` exposed without auth or rate limit:**
- Files: `cmd/server/main.go:95-96`
- Risk: Low (gated by `AnyExists`), but a publicly reachable bootstrap endpoint invites probing; no rate limit means unlimited attempts.
- Recommendation: Move bootstrap behind an env-var token (`BOOTSTRAP_TOKEN`) or a startup-only flag.

## Performance Bottlenecks

**Exports buffer everything in memory, and the date range is unlimited:**
- Problem: `ExportHandler.writeCSV` (`internal/adapters/primary/http/export.go:266-282`) and `writeXLSX` (135-181) fetch ALL rows into `[]ports.ExportRow`, convert to `[]csvRow`, and only then stream. `parseExportRange` (`export.go:284-299`) accepts any `from`/`to` — a 5-year range over a large org loads the whole result set into RAM (times 2 for the XLSX workbook, which excelize builds fully in memory).
- Files: `internal/adapters/primary/http/export.go`, `internal/adapters/secondary/postgres/export_repository.go` (`Timesheets`/`Expenses`/`Combined` return full slices)
- Cause: Repository APIs return `[]ports.ExportRow`; no cursor/iterator interface.
- Improvement path: Add a streaming repository method (`QueryRow`-based iteration) or hard-cap the export range (e.g., max 1 year per request) and stream rows with a chunked writer. The `/exports/*/count` endpoints exist precisely because clients should paginate — recommend the frontend use them.

**Rate limiter contention:**
- Problem: Every request takes the global `rl.mu` lock twice (key lookup + `allow`). Fine at current scale; a bottleneck under heavy traffic behind a shared IP.
- Files: `internal/middleware/ratelimit.go`
- Improvement path: Shard the map by key hash (16-64 shards) when the app needs it.

**No N+1 in hot paths (verified):**
- Repository methods use recursive CTEs (`GetAncestry` in `internal/adapters/secondary/postgres/activity_repository.go:196-205`) and batch queries; service loops iterate already-fetched slices (e.g., `internal/core/services/expense/expense.go:390-394`). This is a strength, not a concern — keeping it as a documented invariant.

## Fragile Areas

**`activity_repository.go` / `ticket_repository.go` (835 / 861 lines):**
- Files: `internal/adapters/secondary/postgres/activity_repository.go`, `internal/adapters/secondary/postgres/ticket_repository.go`
- Why fragile: Each bundles 20+ hand-written SQL statements (multi-join, CTEs, upserts) for the most complex aggregates; single-file edits risk subtle SQL regressions. Activity repo also owns the synchronous audit-log-in-transaction writes (ADR-BE-016 invariant).
- Safe modification: Add tests per query; keep the transactional audit-log invariant (state + event commit or roll back together — see `internal/core/services/activity/activity.go:407-419`).
- Test coverage: 73.9% package-wide for postgres — decent, but the deepest SQL (ancestry joins, proposal approval transaction) deserves targeted coverage.

**Shared routing service invariant (D-08/D-G):**
- Files: `internal/core/services/routing/routing.go`, `cmd/server/main.go:134`
- Why fragile: One shared `routingSvc` instance is passed to time-entry, activity, coverage (and planned proposals). Comments in `main.go:129-136` document that a second instance would let entry and proposal routing drift. Any refactor that changes resolution semantics ripples across three services.
- Safe modification: Only change routing via the documented resolution contract; add service-level regression tests (there is 1 test file for routing).

**Coverage period-close serialization:**
- Files: `internal/core/services/coverage/coverage.go` (WR-03 advisory xact lock), `internal/adapters/secondary/postgres/coverage_repository.go`
- Why fragile: Correctness depends on advisory locking + UTC session pinning (`internal/db/db.go:64-66` timezone pin — WR-06). These are subtle cross-cutting invariants that a "small" change can silently break (the git log shows WR-01..WR-06 fixes landed recently for exactly this).
- Safe modification: Keep the UTC-timezone pool pin; write regression tests for concurrent closes and non-UTC server timezones.

**Migration history with gaps:**
- Files: `migrations/` (000 + 17 pairs; 001-003, 007 removed), `scripts/seed_demo.sql`
- Why fragile: Fresh-DB correctness depends on `000_full_schema.up.sql` staying in sync with the 17 patches; seed scripts (Makefile `seed`, `seed-demo`) must match current schema. `make db-init` already references a missing file.
- Safe modification: For any schema change, add a new numbered pair (`021_...`); never edit applied migrations; verify `make migrate-all` + `make seed` on a clean database.

**No CI test pipeline:**
- Files: `.github/workflows/` — only `docs-check.yml` (warnings allowed, `continue-on-error: true`) and `qodana_code_quality.yml` (static analysis only).
- Why fragile: 79 Go test files, 16 Vitest files, and 11 Playwright specs exist but run nowhere automated on PRs. `go test`, `bun run lint`, `bun run build`, `bunx playwright test` are all manual.
- Safe modification: Add a workflow running `go test ./...`, `go vet`, `cd web && bun install && bun run lint && bun run build`, and the e2e suite on PRs to `main`/`develop`.

## Scaling Limits

**Rate limiter map:** unbounded entries per unique IP/user key → memory growth proportional to distinct client addresses (see Security above). Current capacity: fine at demo scale; breaks under address rotation or long uptimes.
**Exports:** full-result-set buffering → OOM risk once a large org exports a wide date range (see Performance). No row-count cap today.
**Monolithic `main.go` wiring:** ~70 routes in one file; each new feature adds ~15 lines of wiring. Not a runtime limit, but a maintainability ceiling.

## Dependencies at Risk

**`github.com/lib/pq` (v1.10.9):**
- Risk: Used only by the dead `database/sql` path in `internal/db/db.go`.
- Impact: None today; a second Postgres driver confuses audits.
- Migration plan: Delete usage + dependency (`go mod tidy`).

**`github.com/xuri/excelize/v2` (v2.11.0):**
- Risk: Heavy dependency (mscfb, efp, nfp transitive) used only for export; in-memory workbook building.
- Impact: Export memory (see Performance); large binary size in the Go image.
- Migration plan: Keep if XLSX export is required; stream via CSV or a lighter writer otherwise.

**Bleeding-edge frontend toolchain:**
- Files: `web/package.json` — TypeScript 7.0.2, Vite ^8.1.5, Vitest ^4.1.10, React 19.2, oxlint ^1.76.
- Risk: Fast-moving majors; plugin/ecosystem lag possible; TS 7 (Go-based) is a major rewrite with tooling ripple risk.
- Impact: Upgrade breakage on minor pins; `recharts` is already pinned at exactly `3.8.0`, suggesting a known compatibility issue.
- Mitigation: Keep lockfile (`bun.lock`) committed; pin aggressively for the build chain.

**`testcontainers` (v0.42.0):**
- Risk: Pulls ~30 indirect deps (docker, otel, gopsutil) into `go.mod`; only used by integration tests.
- Impact: Build/CI environment weight; docker-in-CI requirement for postgres integration tests.
- Mitigation: Acceptable; ensure CI (when added) exposes a Docker daemon.

## Missing Critical Features

**No automated regression pipeline (highest-impact gap):** the codebase has exceptional test volume but zero CI enforcement — see Fragile Areas. Blocks: safe refactoring of the large repositories; catching `make db-init`-class breakage.

**Expense receipt lifecycle:** uploads exist but there is no download-by-URL endpoint? (uploads are served by the app?) — verify serving path; no delete-on-expense-delete, no storage abstraction. Blocks: moving to object storage, compliance retention policies.

**User-initiated session revocation:** logout revokes the current refresh-token family, but there is no "log out all sessions" or per-device session listing for users; deactivation (`DELETE /organizations/members/{id}`) revokes refresh tokens via `RevokeAllByUser` (`internal/adapters/secondary/postgres/refresh_token_repo.go:63-69`) — verify this is actually called by the deactivate flow; access tokens still live 15 minutes.

## Test Coverage Gaps

**Lowest-coverage Go packages:**
- `internal/core/services/organization` — 41.9% (org settings, membership role management, invite flows)
- `internal/core/services/unit` — 47.4% (tree building, member management)
- `internal/adapters/primary/http` — 47.0% overall (handlers exist but many branches untested)
- `internal/core/services/expense` — 68.7% (approval transitions, receipt upload untested)
- `internal/middleware` — 69.2%
- `cmd/server` — 0.0% (wiring untested); `cmd/migrate` — 15.0%
- Priority: High — the register/invite bug above sits exactly in a 47% area; handler-level tests would have caught the `InviteCode→OrgID` mapping.

**Frontend:**
- 16 Vitest files vs 71 route files; `__tests__` exist only for time-entries, approvals, today-page, and layout components. Auth forms (login/register/password-reset/invite/bootstrap) have no component tests — the broken register "join" branch is untested.
- 11 Playwright specs cover the main flows (`web/e2e/`) but run only manually.

---

*Concerns audit: 2026-08-08*
