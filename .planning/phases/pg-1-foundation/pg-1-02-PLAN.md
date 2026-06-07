---
phase: pg-1-foundation
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/db/pgpool.go
  - internal/db/db.go
  - internal/db/migrate.go
  - cmd/migrate/main.go
  - go.mod
  - go.sum
autonomous: true
requirements:
  - D-08 (pgxpool MaxConns:25)
  - D-09 (DATABASE_URL env var)
  - D-10 (sync.Once singleton)
  - D-11 (5s health check)
  - D-17 (migrate -all flag)
  - D-25 (port cmd/migrate to pgx)
must_haves:
  truths:
    - "internal/db/pgpool.go provides a pgxpool.Pool singleton via sync.Once, reading DATABASE_URL"
    - "Pool creation includes a 5s health check via pool.Acquire(ctx)"
    - "cmd/migrate/main.go uses pgx (not database/sql + lib/pq)"
    - "cmd/migrate -all flag applies all migrations then seed files"
    - "internal/db/db.go and internal/db/migrate.go are deleted (dead code)"
    - "go.mod includes github.com/jackc/pgx/v5 and no longer requires github.com/lib/pq"
  artifacts:
    - path: "internal/db/pgpool.go"
      provides: "pgxpool.Pool singleton (NewPool, ClosePool)"
      exports: ["NewPool", "ClosePool"]
    - path: "cmd/migrate/main.go"
      provides: "Migration CLI ported to pgx with -all flag"
      exports: ["main", "getCommand", "getMigrationsDir", "migrateUp", "migrateDown", "migrateAll"]
  key_links:
    - from: "cmd/migrate/main.go"
      to: "github.com/jackc/pgx/v5"
      via: "pgx.Connect, conn.Exec, conn.Ping, conn.Close"
    - from: "internal/db/pgpool.go"
      to: "github.com/jackc/pgx/v5/pgxpool"
      via: "pgxpool.ParseConfig, pgxpool.NewWithConfig, pool.Acquire, pool.Close"
---

<objective>
Create the PostgreSQL connection pool singleton and port the migration CLI to pgx.

Purpose: Replace `database/sql` + `lib/pq` with `github.com/jackc/pgx/v5` across the project. The pool singleton follows the same `sync.Once` pattern as `internal/db/surreal.go`. The migration CLI gets a new `-all` flag for one-shot bootstrap and is fully ported to pgx API. Dead code files are removed.

Output:
- `internal/db/pgpool.go` — New pgxpool.Pool singleton
- `internal/db/db.go` — DELETED (dead code, replaced by pgpool.go)
- `internal/db/migrate.go` — DELETED (dead code, replaced by cmd/migrate/main.go)
- `cmd/migrate/main.go` — Ported to pgx with -all flag
- `go.mod` + `go.sum` — Updated with pgx dependency, lib/pq removed
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-1-foundation/pg-1-RESEARCH.md
@.planning/phases/pg-1-foundation/pg-1-PATTERNS.md
@internal/db/surreal.go
@internal/db/db.go
@internal/db/migrate.go
@cmd/migrate/main.go
@go.mod
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add pgx dependency, create pgpool.go, delete dead code</name>
  <files>
    go.mod
    go.sum
    internal/db/pgpool.go
    internal/db/db.go
    internal/db/migrate.go
  </files>

  <read_first>
    - go.mod — current module file to update
    - internal/db/surreal.go — singleton pattern to replicate (lines 16-57, 67-72)
    - internal/db/db.go — dead code to delete (confirms nothing imports it)
    - internal/db/migrate.go — dead code to delete (confirms nothing imports it)
    - .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — pgxpool pattern (lines 163-257), go.mod changes (lines 726-737)
    - .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — pgpool analog (lines 179-264), go.mod changes (lines 441-467)
  </read_first>

  <action>
    Run `go get github.com/jackc/pgx/v5@latest` to add pgx v5 to go.mod and download it. Then run `go mod tidy`.

    Create internal/db/pgpool.go in the `db` package with these exact signatures and behaviors:

    ```go
    package db

    import (
        "context"
        "fmt"
        "os"
        "sync"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"
    )

    var (
        poolInstance *pgxpool.Pool
        poolOnce     sync.Once
    )

    func NewPool() (*pgxpool.Pool, error)
    func ClosePool()
    ```

    NewPool() implementation per D-08 through D-11:
    - Uses sync.Once for singleton initialization (exact pattern from surreal.go lines 16-57)
    - Reads DATABASE_URL env var (D-09); falls back to `postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable`
    - Uses `pgxpool.ParseConfig(databaseURL)` to create a config (not connection string params)
    - Sets `config.MaxConns = 25` (D-08)
    - Sets `config.MaxConnLifetime = 30 * time.Minute`
    - Sets `config.MaxConnIdleTime = 5 * time.Minute`
    - Creates pool via `pgxpool.NewWithConfig(ctx, config)` with a 5s context timeout (D-11)
    - Health check: `pool.Acquire(ctx)` with 5s timeout, then `conn.Release()` (D-11)
    - Uses the existing `getEnvOrDefault` helper from surreal.go (both are in the same `db` package — no import needed)
    - On error during init, sets `initErr`, closes the pool if it was partially created, and returns nil pool
    - Returns `poolInstance, nil` on success

    ClosePool() implementation:
    - Guards against nil poolInstance
    - Calls `poolInstance.Close()`

    Use the exact `var initErr error` / `poolOnce.Do(...)` / `if initErr != nil { return nil, initErr }` / `if poolInstance == nil { return nil, fmt.Errorf("pool not initialized") }` pattern from surreal.go.

    After pgpool.go is created, delete internal/db/db.go and internal/db/migrate.go (confirmed dead code — nothing imports `db.DB`, `db.MigrateUp`, or `db.MigrateDown`; grep shows all imports go through `sdbConn.DB()` from SurrealDB).

    After deletions, run `go mod tidy` to clean up any orphaned indirect deps.
  </action>

  <verify>
    <automated>go vet ./internal/db/ && go build ./internal/db/</automated>
  </verify>

  <acceptance_criteria>
    - `go mod verify` succeeds
    - go.mod includes `github.com/jackc/pgx/v5` in require block
    - go.mod no longer includes `github.com/lib/pq` in require block
    - internal/db/pgpool.go exists with NewPool() and ClosePool() functions
    - internal/db/pgpool.go compiles: `go vet ./internal/db/` passes
    - internal/db/db.go no longer exists
    - internal/db/migrate.go no longer exists
    - NewPool() uses sync.Once pattern identical to surreal.go
    - NewPool() reads DATABASE_URL env var, falls back to default postgres:// URL
    - Config sets MaxConns to 25
    - Health check uses pool.Acquire with 5s context timeout
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Port cmd/migrate/main.go to pgx and add -all flag</name>
  <files>
    cmd/migrate/main.go
  </files>

  <read_first>
    - cmd/migrate/main.go — current migration CLI (lines 1-124) to port
    - .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — port strategy section (lines 688-721), -all flag logic (lines 708-719)
    - .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — migrate port pattern (lines 267-346), -all logic (lines 311-334)
  </read_first>

  <action>
    Rewrite cmd/migrate/main.go. Replace the current `database/sql` + `lib/pq` implementation with pgx.

    **Import changes:**
    - Remove `"database/sql"` and `_ "github.com/lib/pq"` imports
    - Add `"context"` and `"github.com/jackc/pgx/v5"` imports

    **Connection changes (D-25):**
    - Replace `sql.Open("postgres", databaseURL)` with `pgx.Connect(ctx, databaseURL)` using `context.Background()`
    - Replace `db.Ping()` with `conn.Ping(ctx)`
    - Replace `db.Close()` with `conn.Close(ctx)`
    - Replace `db.Exec(string(content))` with `conn.Exec(ctx, string(content))`

    **API changes:**
    - All function signatures change from `*sql.DB` to `*pgx.Conn`
    - `migrateUp(conn *pgx.Conn, dir string) error`
    - `migrateDown(conn *pgx.Conn, dir string) error`
    - New: `migrateAll(conn *pgx.Conn, dir string) error` (D-17)

    **-all flag implementation (D-17):**
    Enhance `getCommand` to also detect `-all` and return `"all"`.
    Usage strings update to: `"Usage: migrate -up|-down|-all [-dir <migrations_dir>]"`

    `migrateAll` logic:
    1. Read all `*.up.sql` files from migrations dir (sorted)
    2. Separate into migration files (contain "init" or "full_schema") and seed files (contain "seed")
    3. Apply migration files first, then seed files
    4. Use the same `strings.Contains(err.Error(), "already exists")` skip logic for idempotency

    **Keep these behaviors unchanged:**
    - Error matching via `strings.Contains(err.Error(), "already exists")` — works the same with pgx
    - Error matching via `strings.Contains(err.Error(), "does not exist")` for down migrations
    - Reading migration files via `os.ReadFile`
    - `getMigrationsDir` function (unchanged)
    - Sorting patterns (unchanged)
    - All env var handling (DATABASE_URL, default URL)

    After rewriting, run `go build ./cmd/migrate/` to verify compilation.
  </action>

  <verify>
    <automated>go vet ./cmd/migrate/ && go build ./cmd/migrate/</automated>
  </verify>

  <acceptance_criteria>
    - cmd/migrate/main.go compiles: `go vet ./cmd/migrate/` passes
    - No imports of `database/sql` or `lib/pq` in cmd/migrate/main.go
    - Uses `pgx.Connect`, `conn.Exec(ctx, ...)`, `conn.Ping(ctx)`, `conn.Close(ctx)`
    - `-all` flag supported alongside existing `-up` and `-down`
    - `-all` applies migration files first, then seed files
    - Existing `-up` and `-down` behavior unchanged
    - Error skipping via `"already exists"` and `"does not exist"` preserved
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| DATABASE_URL env var → pgxpool | Connection string sourced from environment — must not be hardcoded in production |
| pool.Acquire → database | Health check establishes a real connection at startup |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg1-03 | DoS | pgxpool exhaustion | mitigate | MaxConns: 25 cap prevents runaway connection growth; Acquire timeout prevents indefinite blocking |
| T-pg1-04 | Information Disclosure | DATABASE_URL env var | accept | Standard practice for 12-factor apps; connection string is not logged |
| T-pg1-05 | Spoofing | Migration execution | mitigate | Migration files are static SQL committed to repo — not user-controllable; pgx uses same connection as database/sql would |
</threat_model>

<verification>
- `go vet ./internal/db/` passes
- `go vet ./cmd/migrate/` passes
- `go build ./internal/db/` and `go build ./cmd/migrate/` succeed
- go.mod has pgx/v5, no lib/pq
- pgpool.go exists with NewPool() and ClosePool()
- cmd/migrate/main.go recognizes -all, -up, -down flags
</verification>

<success_criteria>
- pgxpool singleton works with health check
- Migration CLI compiles and runs with pgx
- `cmd/migrate -all` applies all migrations + seed
- Dead code (db.go, migrate.go) removed
- lib/pq removed from dependencies
</success_criteria>

<output>
After completion, create `.planning/phases/pg-1-foundation/pg-1-02-SUMMARY.md`
</output>
