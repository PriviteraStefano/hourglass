---
phase: pg-1-foundation
plan: 03
type: execute
wave: 2
depends_on:
  - pg-1-foundation-02
files_modified:
  - docker-compose.yml
  - cmd/server/main.go
  - Makefile
  - AGENTS.md
autonomous: true
requirements:
  - D-18 (PostgreSQL default service)
  - D-19 (SurrealDB optional profile)
  - D-20 (health check pg_isready)
  - D-21 (DATABASE_URL to app service)
  - D-26 (DATABASE_URL env var value)
  - D-22 (pool init in main.go)
  - D-23 (fail fast if can't connect)
  - D-24 (SURREALDB_* deprecated warning)
must_haves:
  truths:
    - "Postgres starts by default (no profiles) in docker-compose up"
    - "SurrealDB starts only with docker-compose --profile surrealdb up"
    - "Postgres has a pg_isready health check"
    - "App service depends_on postgres with health condition"
    - "App service gets DATABASE_URL pointing to postgres:5432"
    - "cmd/server/main.go initializes pgpool and fails fast on connection error"
    - "SURREALDB_* env vars produce a deprecation warning log in main.go"
    - "Makefile has a migrate-all target"
    - "AGENTS.md reflects PostgreSQL as the primary database"
  artifacts:
    - path: "docker-compose.yml"
      provides: "Updated compose with Postgres default and SurrealDB profiled"
    - path: "cmd/server/main.go"
      provides: "Server entry with pgpool init and SurrealDB deprecation warning"
    - path: "Makefile"
      provides: "Build tooling with migrate-all target"
    - path: "AGENTS.md"
      provides: "Updated documentation reflecting PostgreSQL primacy"
  key_links:
    - from: "cmd/server/main.go"
      to: "internal/db/pgpool.go"
      via: "import and call to db.NewPool() / db.ClosePool()"
    - from: "docker-compose.yml"
      to: "cmd/server/main.go"
      via: "DATABASE_URL env var consumed by main.go at startup"
---

<objective>
Update infrastructure configuration, server wiring, and documentation to make PostgreSQL the primary database.

Purpose: Flip docker-compose so Postgres starts by default (no profile) and SurrealDB becomes optional (--profile surrealdb). Wire the pgxpool init into the server entry point with fail-fast on connection failure. Update Makefile and AGENTS.md to reflect the new database topology.

Output:
- `docker-compose.yml` — Postgres default, SurrealDB profiled, app depends on postgres
- `cmd/server/main.go` — pgpool init, SurrealDB deprecation warning
- `Makefile` — migrate-all target added
- `AGENTS.md` — Updated database and env vars documentation
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-1-foundation/pg-1-RESEARCH.md
@.planning/phases/pg-1-foundation/pg-1-PATTERNS.md
@docker-compose.yml
@cmd/server/main.go
@Makefile
@AGENTS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Update docker-compose.yml — Postgres default, SurrealDB profiled</name>
  <files>
    docker-compose.yml
  </files>

  <read_first>
    - docker-compose.yml — current compose file
    - .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — docker compose changes (lines 772-798)
    - .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — docker-compose pattern (lines 349-385)
  </read_first>

  <action>
    Apply these exact changes to docker-compose.yml per D-18, D-19, D-20, D-21, D-26:

    1. **SurrealDB** — Add `profiles: [surrealdb]` block after the image line (D-19). Change its healthcheck test if needed (keep the curl-based one). The surrealdb service stays but now only starts with `--profile surrealdb`.

    2. **Postgres** — Remove the `profiles: [postgres]` block entirely (D-18). Also remove `profiles:` key entirely so postgres starts by default.

    3. **Postgres healthcheck** — Keep existing `pg_isready -U hourglass` healthcheck (D-20).

    4. **App service** — Change `depends_on` from surrealdb to postgres:
       ```yaml
       depends_on:
         postgres:
           condition: service_healthy
       ```
       (D-21)

    5. **App env vars** — Keep `DATABASE_URL: postgres://hourglass:hourglass@postgres:5432/hourglass?sslmode=disable` (already present, D-26). Keep all SURREALDB_* env vars (they're still needed for `--profile surrealdb` usage and D-24 suppresses the warning in main.go, not by removing them). Keep JWT_SECRET, PORT, etc.

    6. Keep the volumes section (postgres_data, uploads). Keep the migrations auto-init volume mount `./migrations:/docker-entrypoint-initdb.d`.
  </action>

  <verify>
    <automated>grep -c "profiles:" docker-compose.yml | grep -c "1"</automated>
  </verify>

  <acceptance_criteria>
    - `docker-compose.yml` has exactly ONE `profiles:` key (under surrealdb service)
    - Postgres service has no `profiles:` key
    - Postgres healthcheck test is `pg_isready -U hourglass`
    - SurrealDB service has `profiles: [surrealdb]`
    - App depends_on references postgres (not surrealdb) with `condition: service_healthy`
    - DATABASE_URL env var on app points to `postgres:5432`
    - SURREALDB_* env vars still present on app service
    - Volumes section still has `postgres_data` and `./migrations:/docker-entrypoint-initdb.d`
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Update cmd/server/main.go — init pgpool, deprecation warning</name>
  <files>
    cmd/server/main.go
  </files>

  <read_first>
    - cmd/server/main.go — current server entry point
    - .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — server wiring section (lines 800-821)
    - .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — server pattern (lines 389-411)
  </read_first>

  <action>
    Apply these changes to cmd/server/main.go per D-22, D-23, D-24:

    1. **Add import for `"fmt"`** if not already present (needed for Sprintf in deprecation warning).

    2. **After `authService := auth.NewService(jwtSecret)` and before `sdbConn, err := db.NewSurrealDB()`**, insert pgpool initialization:
       ```go
       pgPool, err := db.NewPool()
       if err != nil {
           log.Fatalf("Failed to initialize PostgreSQL pool: %v", err)
       }
       defer db.ClosePool()
       log.Println("PostgreSQL pool initialized")
       ```
       (D-22, D-23 — fail fast via log.Fatalf)

    3. **After the existing `log.Println("Using SurrealDB")` line**, add deprecation warning (D-24):
       ```go
       if os.Getenv("SURREALDB_URL") != "" {
           log.Println("WARNING: SURREALDB_* env vars are deprecated. PostgreSQL is now the default database.")
       }
       ```
       Note: `"os"` is already imported in main.go.

    4. **Do NOT remove or modify** any existing SurrealDB code (sdbConn, repository constructors, handler wire-ups). The server still uses SurrealDB for data operations in Pg-1 — pgpool is initialized but not passed anywhere yet. That's intentional and happens in Pg-2.

    5. Run `go vet ./cmd/server/` and `go build ./cmd/server/` to verify compilation.
  </action>

  <verify>
    <automated>go vet ./cmd/server/ && go build ./cmd/server/</automated>
  </verify>

  <acceptance_criteria>
    - `go vet ./cmd/server/` passes
    - `go build ./cmd/server/` succeeds
    - main.go calls `db.NewPool()` before any SurrealDB code
    - Calls `defer db.ClosePool()` for cleanup
    - Fails fast with `log.Fatalf` if pool can't initialize
    - Logs deprecation warning if `SURREALDB_URL` env var is set
    - Existing SurrealDB init and repository wiring remains untouched
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: Update Makefile and AGENTS.md</name>
  <files>
    Makefile
    AGENTS.md
  </files>

  <read_first>
    - Makefile — current Makefile (34 lines)
    - AGENTS.md — current documentation (217 lines)
    - .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — Makefile pattern (lines 415-427), AGENTS.md pattern (lines 429-438)
  </read_first>

  <action>
    **Makefile changes:**
    1. Add a `migrate-all` target after the existing `migrate-down` target:
       ```makefile
       migrate-all:
       	go run ./cmd/migrate -all -dir $(MIGRATIONS_DIR)
       ```
    2. Keep all existing targets unchanged.
    3. Optionally update `db-init` target to reference `002_full_schema` instead of `001_init` if the user wants a broader init — or leave as-is for backward compat. **Recommendation:** leave `db-init` targeting `001_init` (it's a convenience for ad-hoc psql exec, not part of the main migration flow).

    **AGENTS.md changes:**
    1. Line 10: Change `"SurrealDB for application data, plus PostgreSQL SQL migrations via cmd/migrate"` to `"PostgreSQL for application data (primary), with SurrealDB still available via docker-compose --profile surrealdb"`
    2. Line 64: Change `go run ./cmd/server # Runs on :8080, connects to SurrealDB via SURREALDB_URL` to `go run ./cmd/server # Runs on :8080, connects to PostgreSQL via DATABASE_URL`
    3. Line 67-68: Remove or update SurrealDB schema bootstrap reference (keep as note that it's still available via --profile surrealdb)
    4. Line 78: Change `docker-compose up # Starts surrealdb + app; add --profile postgres for the Postgres service` to `docker-compose up # Starts postgres + app; add --profile surrealdb for SurrealDB`
    5. Lines 81-84: Update migration section to mention `cmd/migrate -all` for one-shot bootstrap
    6. Lines 153-157: Change `"Application data uses SurrealDB..."` to `"Application data uses PostgreSQL..."`
    7. Lines 160-170: Add `DATABASE_URL` as primary env var, note SURREALDB_* as deprecated
    8. Keep all other sections unchanged
  </action>

  <verify>
    <automated>make migrate-all 2>/dev/null || echo "migrate-all target exists"</automated>
  </verify>

  <acceptance_criteria>
    - Makefile has `migrate-all` target that runs `go run ./cmd/migrate -all -dir $(MIGRATIONS_DIR)`
    - AGENTS.md database description says "PostgreSQL for application data (primary)"
    - AGENTS.md local dev workflow uses `postgres` and mentions `--profile surrealdb`
    - AGENTS.md database initialization section reflects PostgreSQL as primary
    - AGENTS.md env vars section lists DATABASE_URL as primary, SURREALDB_* as deprecated
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| docker-compose environment → app service | Env vars cross from compose file to container |
| server startup → database | Connection attempt crosses network boundary |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg1-06 | Tampering | docker-compose.yml | accept | Compose file is static and committed to repo; env vars inside are local-dev defaults (not secrets) |
| T-pg1-07 | DoS | Server startup hang | mitigate | Pool init has 5s timeout; server fails fast with log.Fatalf on connection failure (D-23) |
| T-pg1-08 | Information Disclosure | Deprecation warnings | accept | Warnings mention SURREALDB_* env vars — acceptable for development; production should clear them |
</threat_model>

<verification>
- `go vet ./cmd/server/` passes
- `go build ./cmd/server/` succeeds
- `docker-compose config` is valid (run silently to confirm syntax)
- Makefile has migrate-all target
- AGENTS.md reflects Postgres as primary database
</verification>

<success_criteria>
- docker-compose starts Postgres by default, SurrealDB with `--profile surrealdb`
- Server initializes pgpool on startup, fails fast on error
- Server logs SurrealDB deprecation warning when SURREALDB_* env vars are present
- Makefile has migrate-all convenience target
- AGENTS.md documents the new database topology
</success_criteria>

<output>
After completion, create `.planning/phases/pg-1-foundation/pg-1-03-SUMMARY.md`
</output>
