# Pg-3: Wiring, cleanup & verification

**Gathered:** 2026-06-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire all postgres repos in server init, delete all SurrealDB code, update build system and docs, verify everything works end-to-end.

</domain>

<decisions>
## Implementation Decisions

### Wiring
- **D-01:** `cmd/server/main.go` constructs all postgres repos with `*pgxpool.Pool`
- **D-02:** Each service receives the postgres repo (no code changes to services)
- **D-03:** Handler constructors unchanged — they receive services, not repos
- **D-16:** One-shot replacement — replace all SurrealDB repos with Postgres repos in a single pass. Server goes from all-SurrealDB to all-Postgres in one commit.
- **D-17:** Match existing constructor patterns exactly — `postgres.NewXxx(pool)` instead of `surrealdb.NewXxx(sdbConn.DB())`. Same line-by-line structure, same service wiring, just swap package prefix and pool parameter.
- **D-18:** Remove the deprecated SURREALDB_* env var warning from main.go. Server only needs `DATABASE_URL` and `JWT_SECRET`.

### Cleanup — delete these entirely:
- **D-04:** `internal/adapters/secondary/surrealdb/` — all 25+ files
- **D-05:** `internal/db/surreal.go` — SurrealDB singleton
- **D-06:** `cmd/schema/main.go` — SurrealDB schema loader
- **D-07:** `schema/` — all `.surql` files
- **D-08:** SurrealDB from `docker-compose.yml`
- **D-09:** SurrealDB from `Makefile` (schema targets, etc.)
- **D-10:** `go.mod` — remove `github.com/surrealdb/surrealdb.go` dependency
- **D-19:** Keep `internal/db/` package as-is after removing `surreal.go` — only `pgpool.go` remains. No rename or restructuring.
- **D-20:** Remove `github.com/lib/pq` from go.mod alongside the SurrealDB dependency — everything uses pgx now. Then run `go mod tidy`.

### Docker compose
- **D-25:** Remove the SurrealDB service block entirely from `docker-compose.yml` (not just commented out). Also remove `--profile surrealdb` profiles section and any SurrealDB-specific volumes/env vars.

### Build & docs updates
- **D-11:** Makefile: remove all schema/surreal/seed targets, add `make setup = go run ./cmd/migrate -all`
- **D-12:** AGENTS.md: update all SurrealDB references to PostgreSQL
- **D-13:** Environment vars: `SURREALDB_*` no longer needed, `DATABASE_URL` is required
- **D-14:** Remove `cmd/schema` from any docs/scripts

### CORS middleware
- **D-23:** Extract CORS middleware from main.go inline closure to `internal/middleware/cors.go` — consistent with Logging, RateLimiter, APIVersion middleware there.

### Verification
- **D-15:** Full manual verification flow:
  1. `docker compose up -d` (Postgres starts)
  2. `go run ./cmd/migrate -all` (schema + seed)
  3. `go run ./cmd/server` (server starts on :8080)
  4. Login as demo manager (alex.rivera / demo123)
  5. Check all pages render with data
  6. CRUD operations on every entity type
- **D-21:** Automated smoke test in `cmd/server/main_test.go` — verifies `/health` returns 200 and an authenticated `/units` call returns data.
- **D-22:** Smoke test reuses Pg-2's exported test helpers from `internal/adapters/secondary/postgres/exported_test_helpers.go` (setUpTestDB / tearDownTestDB pattern).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Server wiring
- `cmd/server/main.go` — Current SurrealDB wiring (to replace with Postgres)
- `internal/adapters/primary/http/*.go` — All handlers (unchanged)
- `internal/core/services/*` — All services (unchanged)

### Postgres adapters (to wire)
- `internal/adapters/secondary/postgres/*.go` — All 18+ repository implementations (from Pg-2)
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — Test helpers to reuse for smoke test
- `internal/db/pgpool.go` — pgxpool singleton (from Pg-1)

### Files to delete
- `internal/adapters/secondary/surrealdb/` — All SurrealDB repos
- `internal/db/surreal.go` — SurrealDB singleton
- `cmd/schema/main.go` — Schema loader
- `schema/` — All `.surql` files

### Build & config
- `Makefile` — Build targets (to update)
- `AGENTS.md` — Project docs (to update)
- `docker-compose.yml` — Docker setup (to clean)
- `go.mod` — Dependencies (to clean)

### Reference
- `docs/superpowers/specs/2026-06-07-postgresql-migration-design.md` — Full migration design doc
- `internal/middleware/` — Existing middleware (CORS to join them)

</canonical_refs>

<code_context>
## Existing Code Insights

### State After Pg-2
- All 18+ Postgres repos exist in `internal/adapters/secondary/postgres/` with tests
- `internal/db/` has `pgpool.go` (pool singleton) and `surreal.go` (to delete)
- `cmd/server/main.go` still uses SurrealDB for everything — 18 SurrealDB repo constructors, 1 SurrealDB connection
- SurrealDB 26+ files still in `internal/adapters/secondary/surrealdb/`
- `cmd/schema/main.go` + `schema/*.surql` still exist
- `go.mod` has `github.com/surrealdb/surrealdb.go` and `github.com/lib/pq`

### Reusable Assets
- `internal/db/pgpool.go` — Pool singleton with health check, already wired in main.go
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — Test DB setup/teardown for smoke test
- `internal/middleware/` — Middleware package to receive CORS

### Integration Points
- `cmd/server/main.go` — Single entry point, all wiring happens here
- Postgres pool → repo constructors → service constructors → handler constructors

</code_context>

<specifics>
## Specific Ideas

- One-shot replacement means a single commit that: replaces all repo constructors, removes SurrealDB imports, removes SurrealDB init code. Clear before/after diff.
- CORS middleware follows the same func signature pattern as existing middleware in `internal/middleware/`.
- Smoke test uses the exported `NewTestPool` / test DB helpers from Pg-2's test infrastructure — no new test DB setup needed.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: Pg-3-Wiring*
*Context gathered: 2026-06-07*
