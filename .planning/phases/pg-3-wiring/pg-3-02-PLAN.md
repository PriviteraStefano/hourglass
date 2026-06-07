---
phase: pg-3-wiring
plan: 02
type: execute
wave: 2
depends_on: [pg-3-01]
files_modified:
  - internal/adapters/secondary/surrealdb/
  - internal/db/surreal.go
  - cmd/schema/main.go
  - schema/
  - docker-compose.yml
  - Makefile
  - AGENTS.md
  - go.mod
autonomous: true
requirements: [D-04, D-05, D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13, D-14, D-19, D-20, D-25]
must_haves:
  truths:
    - `internal/adapters/secondary/surrealdb/` directory no longer exists
    - `internal/db/surreal.go` no longer exists
    - `cmd/schema/main.go` no longer exists
    - `schema/` directory no longer exists
    - `docker-compose.yml` has no surrealdb service block, no SURREALDB_* env vars on app service
    - `Makefile` has `setup:` target running `go run ./cmd/migrate -all`
    - `AGENTS.md` no longer references SurrealDB in tech stack, directory structure, or env vars
    - `go.mod` has no `github.com/surrealdb/surrealdb.go` dependency
    - `go build ./...` and `go vet ./...` pass after cleanup
  artifacts:
    - path: internal/adapters/secondary/surrealdb/
      provides: Deleted directory (all 29 .go files removed)
      min_lines: 0
    - path: internal/db/surreal.go
      provides: Deleted file (SurrealDB singleton removed)
      min_lines: 0
    - path: cmd/schema/main.go
      provides: Deleted file (SurrealDB schema loader removed)
      min_lines: 0
    - path: schema/
      provides: Deleted directory (all .surql files removed)
      min_lines: 0
    - path: docker-compose.yml
      provides: SurrealDB-free Docker config
      missing_pattern: "surrealdb"
    - path: Makefile
      provides: Build config with `setup:` target
      contains_pattern: "^setup:"
    - path: AGENTS.md
      provides: Updated docs with PostgreSQL-only references
      missing_pattern: "SurrealDB|surrealdb|surreal.go"
  key_links:
    - from: internal/db/surreal.go (deleted)
      to: internal/db/pgpool.go
      via: SurrealDB singleton removed, only pgpool remains (D-19)
    - from: docker-compose.yml
      to: SurrealDB service (deleted)
      via: No surrealdb container, no --profile surrealdb, no SURREALDB_* env
    - from: Makefile
      to: cmd/migrate
      via: `setup:` target calls `go run ./cmd/migrate -all`
    - from: AGENTS.md
      to: PostgreSQL adapters
      via: Directory structure updated, SurrealDB refs replaced
---

<objective>
Delete all SurrealDB source files, update build configuration and documentation, and clean up dependencies.

**Purpose:** Remove all traces of SurrealDB after Postgres wiring is complete (Plan 1). The build would still compile without this plan (surrealdb package is no longer imported), but the dead code, config, and docs must be cleaned up.

**Output:**
- 4 SurrealDB paths deleted (surrealdb/, surreal.go, cmd/schema/, schema/)
- `docker-compose.yml` — no surrealdb service
- `Makefile` — `setup:` target added
- `AGENTS.md` — all SurrealDB references replaced
- `go.mod` — `surrealdb.go` dependency removed
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-3-wiring/pg-3-RESEARCH.md (file deletion inventory, config change details)

# Config files to modify
@docker-compose.yml (57 lines — has surrealdb service + SURREALDB_* env vars on app)
@Makefile (37 lines — needs setup: target, no surreal targets currently)
@AGENTS.md (216 lines — has ~20 SurrealDB references across directory structure, tech stack, env vars, migrations, docker)

# Module file to clean
@go.mod (has github.com/surrealdb/surrealdb.go v1.4.0 on line 11)

# Files/dirs to delete
@internal/adapters/secondary/surrealdb/ (29 .go files — directory)
@internal/db/surreal.go (72 lines — SurrealDB singleton)
@cmd/schema/main.go (single file)
@schema/ (3 .surql files)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Delete SurrealDB source files and clean docker-compose.yml</name>

  <files>
    internal/adapters/secondary/surrealdb/
    internal/db/surreal.go
    cmd/schema/main.go
    schema/
    docker-compose.yml
  </files>

  <read_first>
    @docker-compose.yml (current state — needs surrealdb service block removed)
  </read_first>

  <action>
    **A) Delete SurrealDB source files (4 paths):**

    1. `internal/adapters/secondary/surrealdb/` — entire directory with all 29 .go files.
       Use: `rm -rf internal/adapters/secondary/surrealdb/`

    2. `internal/db/surreal.go` — single file.
       Use: `rm internal/db/surreal.go`

    3. `cmd/schema/main.go` — single file.
       Use: `rm cmd/schema/main.go`

    4. `schema/` — entire directory with 3 .surql files.
       Use: `rm -rf schema/`

    Note: After Plan 1, no remaining Go file imports the surrealdb package, so these deletions won't break compilation.

    **B) Clean docker-compose.yml:**

    1. Remove the entire `surrealdb:` service block (lines 4-16 — from `surrealdb:` through the healthcheck block, including the `profiles:` section on line 7-8). The block is:
       ```
       surrealdb:
         image: surrealdb/surrealdb:v3.0.5
         container_name: hourglass-surrealdb
         profiles:
           - surrealdb
         command: ["start", "--log", "debug", "--user", "root", "--pass", "root", "memory"]
         ports:
           - "8000:8000"
         healthcheck:
           test: ["CMD-SHELL", "curl -f http://localhost:8000/health || exit 1"]
           interval: 5s
           timeout: 5s
           retries: 10
       ```

    2. Remove SURREALDB_* environment variables from the `app:` service block (lines 40-44):
       Remove these 5 lines:
       ```
           SURREALDB_URL: ws://surrealdb:8000/rpc
           SURREALDB_USER: root
           SURREALDB_PASS: root
           SURREALDB_NS: hourglass
           SURREALDB_DB: main
       ```
       Keep `DATABASE_URL`, `JWT_SECRET`, `PORT` environment variables unchanged.
       Keep `depends_on:` that only references `postgres:`.

    Result: `docker-compose.yml` has only `postgres:` and `app:` services. No SurrealDB references remain.
  </action>

  <verify>
    <automated>
      test ! -d internal/adapters/secondary/surrealdb && \
      test ! -f internal/db/surreal.go && \
      test ! -f cmd/schema/main.go && \
      test ! -d schema && \
      grep -c "surrealdb" docker-compose.yml | grep -q "^0$"
    </automated>
  </verify>

  <done>
    1. `internal/adapters/secondary/surrealdb/` directory deleted (29 files)
    2. `internal/db/surreal.go` deleted
    3. `cmd/schema/main.go` deleted
    4. `schema/` directory deleted (3 .surql files)
    5. `docker-compose.yml` has no `surrealdb:` service block
    6. `docker-compose.yml` app service has no `SURREALDB_*` env vars
    7. `docker-compose.yml` only has `postgres:` and `app:` services
  </done>
</task>

<task type="auto">
  <name>Task 2: Update Makefile, AGENTS.md, go.mod + go mod tidy</name>

  <files>
    Makefile
    AGENTS.md
    go.mod
  </files>

  <read_first>
    @Makefile (current — 37 lines, no surrealdb targets, needs `setup:` target)
    @AGENTS.md (current — 216 lines, ~20 SurrealDB refs to update)
    @go.mod (current — line 11: github.com/surrealdb/surrealdb.go v1.4.0)
  </read_first>

  <action>
    **A) Makefile:**

    Add a `setup:` target just below the existing targets. The target should be:
    ```makefile
    setup:
    	go run ./cmd/migrate -all
    ```
    Place it after the `test:` target (around line 21, after `go test -v ./...`).

    Verify that `PHONY` list includes `setup`. Update the `.PHONY` line (line 1) from:
    `.PHONY: build run migrate test clean docker-build docker-up docker-down`
    to:
    `.PHONY: build run migrate test setup clean docker-build docker-up docker-down`

    No surreal/schema targets currently exist in the Makefile, so no deletions needed (D-09 is already satisfied).

    **B) AGENTS.md — update all SurrealDB references:**

    Replace or remove the following specific references (use exact string matching):

    1. **Tech Stack (line 8):**
       `and SurrealDB adapters in internal/adapters/secondary/surrealdb/*`
       → `and PostgreSQL adapters in internal/adapters/secondary/postgres/*`

    2. **Database (line 10):**
       `Database: PostgreSQL for application data (primary), with SurrealDB still available via docker-compose --profile surrealdb`
       → `Database: PostgreSQL for all application data`

    3. **Directory structure (lines 28-36):**
       - Remove `cmd/schema/main.go` line entirely (line 29)
       - `cmd/server/main.go` description: no change needed
       - `internal/adapters/` section (lines 32-34):
         `primary/http/              # Thin HTTP adapters (auth, project, time-entry, etc.)`
         `secondary/surrealdb/       # SurrealDB repositories and driven adapters`
         →
         `primary/http/              # Thin HTTP adapters (auth, project, time-entry, etc.)`
         `secondary/postgres/        # PostgreSQL repositories and driven adapters`
       - `internal/db/` (line 35):
         `SurrealDB connection plus legacy Postgres DB helpers`
         → `PostgreSQL database connection pool`
       - Remove `schema/` line (line 41)

    4. **Local Development (lines 60-78):**
       - Remove the SurrealDB schema bootstrap block (lines 66-68):
         ```
         # SurrealDB schema bootstrap (separate terminal, only when using --profile surrealdb)
         go run ./cmd/schema           # Applies schema/*.surql to SurrealDB
         ```
       - Docker line (line 78):
         `docker-compose up             # Starts postgres + app; add --profile surrealdb for SurrealDB`
         → `docker-compose up             # Starts postgres + app`

    5. **Migrations section (lines 81-85):**
       - Remove `cmd/schema/main.go` line (line 83):
         `cmd/schema/main.go applies schema/*.surql to SurrealDB via SURREALDB_URL, SURREALDB_USER, SURREALDB_PASS, SURREALDB_NS, and SURREALDB_DB`
       - Update line 84:
         `docker-compose up starts postgres and app; SurrealDB is profile-gated (docker-compose --profile surrealdb up) for schema reference`
         → (remove entirely, D-14)

    6. **Database Initialization (lines 153-157):**
       - Line 154: `Application data uses PostgreSQL (DATABASE_URL); SurrealDB still available via docker-compose --profile surrealdb for schema reference`
         → `Application data uses PostgreSQL (DATABASE_URL)`
       - Line 155: `schema/001_schema.surql is the SurrealDB bootstrap schema applied by cmd/schema`
         → (remove entirely)
       - Line 156: PostgreSQL connection string (no change needed)
       - Line 157: `SurrealDB with root/root; the Postgres service is now the default`
         → `The Postgres service is the default`

    7. **Environment Variables (lines 160-169):**
       - Remove lines 161-165 (SURREALDB_URL, SURREALDB_USER, SURREALDB_PASS, SURREALDB_NS, SURREALDB_DB) and line 166 (SCHEMA_DIR)
       - Keep lines 167-169 (DATABASE_URL, JWT_SECRET, ALLOWED_ORIGINS)
       - Update the section header from:
         `Backend (cmd/server/main.go, cmd/schema/main.go, cmd/migrate/main.go):`
         → `Backend (cmd/server/main.go, cmd/migrate/main.go):`

    **C) go.mod:**

    1. Remove the direct dependency line: `github.com/surrealdb/surrealdb.go v1.4.0`
    2. Remove the indirect dependency lines for surrealdb's transitive deps:
       - `github.com/gofrs/uuid v4.4.0+incompatible // indirect`
       - `github.com/gorilla/websocket v1.5.3 // indirect`
       - `github.com/klauspost/compress v1.18.5 // indirect`
       - `github.com/x448/float16 v0.8.4 // indirect`
    3. Run `go mod tidy` to clean up remaining indirect deps and update go.sum.
    4. Run `go build ./...` to verify compilation.

    Note: D-20 (remove `github.com/lib/pq`) is a no-op — `lib/pq` was verified absent from go.mod/gosum in RESEARCH.md.
  </action>

  <verify>
    <automated>
      grep -q "^setup:" Makefile && \
      ! grep -q "surrealdb" docker-compose.yml && \
      ! grep -q "github.com/surrealdb/surrealdb.go" go.mod && \
      go build ./... && go vet ./...
    </automated>
  </verify>

  <done>
    1. `Makefile` has `setup:` target running `go run ./cmd/migrate -all`
    2. `Makefile` `.PHONY` includes `setup`
    3. `AGENTS.md` has zero references to SurrealDB, surrealdb adapters, cmd/schema, or schema/ directory
    4. `AGENTS.md` directory structure shows `secondary/postgres/` not `secondary/surrealdb/`
    5. `AGENTS.md` env vars section removed SURREALDB_* entries
    6. `go.mod` no longer has `github.com/surrealdb/surrealdb.go`
    7. `docker-compose.yml` has no surrealdb service block or SURREALDB_* env vars
    8. `go build ./...` passes with zero surrealdb references anywhere
    9. `go vet ./...` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| File system → Repository directory | Deleting source directories — one-time destructive operation |
| Build configuration → Compilation | Removing dependencies from go.mod — must verify compilation |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg3-04 | Denial of Service | File deletion | mitigate | Files deleted AFTER Plan 1 removes all imports (depends_on enforces order). Deletion only removes dead code. |
| T-pg3-05 | Tampering | go.mod dependency removal | mitigate | `go mod tidy` auto-removes unused deps. `go build ./...` + `go vet ./...` verify no broken imports. |
| T-pg3-06 | Information Disclosure | AGENTS.md outdated refs | accept | Doc update reflects current architecture. No security-sensitive info in AGENTS.md. |
</threat_model>

<verification>
```bash
# Verify all SurrealDB files deleted
test ! -d internal/adapters/secondary/surrealdb && echo "surrealdb/ deleted"
test ! -f internal/db/surreal.go && echo "surreal.go deleted"
test ! -f cmd/schema/main.go && echo "cmd/schema/main.go deleted"
test ! -d schema && echo "schema/ deleted"

# Verify docker-compose surrealdb-free
grep -c "surrealdb" docker-compose.yml || echo "no surrealdb refs in docker-compose"

# Verify go.mod clean
grep "surrealdb" go.mod || echo "no surrealdb in go.mod"

# Full build verification
go build ./... && go vet ./...

# Verify Makefile setup target
grep "^setup:" Makefile && echo "setup target exists"
</verification>

<success_criteria>
1. All 4 SurrealDB paths deleted (29 .go files in surrealdb/, surreal.go, cmd/schema/, schema/)
2. `docker-compose.yml` has no surrealdb service or SURREALDB_* env vars
3. `Makefile` has `setup:` target
4. `AGENTS.md` has zero SurrealDB references
5. `go.mod` has no `github.com/surrealdb/surrealdb.go`
6. `go build ./...` passes after all deletions and dep cleanup
7. `go vet ./...` passes
</success_criteria>

<output>
After completion, create `.planning/phases/pg-3-wiring/pg-3-02-SUMMARY.md`
</output>
