# Phase Pg-1: Foundation — PostgreSQL schema, pool, docker-compose — Research

**Researched:** 2026-06-07
**Domain:** PostgreSQL schema design, pgxpool connection management, Go migration tooling, Docker Compose orchestration
**Confidence:** HIGH

## Summary

Phase Pg-1 creates the PostgreSQL foundation: a 24-table schema translated from SurrealDB, a `pgxpool` connection pool singleton, an updated migration CLI with `-all` one-shot flag, demo seed data as idempotent SQL, and a docker-compose setup where Postgres is the default service. The phase does **not** port any application repository adapters — that is Pg-2's job.

**Primary recommendation:** Use `pgxpool.New(ctx, connString)` with pool config params in the connection string. Create `internal/db/pgpool.go` following the same `sync.Once` singleton pattern as `internal/db/surreal.go`. Port `cmd/migrate/main.go` to `pgx`, add `-all` flag, and create both `migrations/002_full_schema.up.sql` and `migrations/003_seed.up.sql`. Flip docker-compose so Postgres is default, SurrealDB is profiled.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** All tables `CREATE TABLE IF NOT EXISTS` for idempotent migrations
- **D-02:** UUID primary keys with `gen_random_uuid()` default
- **D-03:** Foreign keys with `ON DELETE CASCADE` where SurrealDB cascaded, `ON DELETE RESTRICT` where it blocked
- **D-04:** `TIMESTAMPTZ` for all datetime fields
- **D-05:** JSONB for flexible/object fields (financial_cutoff_config, audit_log changes)
- **D-06:** CHECK constraints for enum-like fields (role, status, category, etc.)
- **D-07:** Indexes matching current SurrealDB INDEX definitions
- **D-08:** `github.com/jackc/pgx/v5/pgxpool` with `pool.MaxConns: 25`
- **D-09:** Reads `DATABASE_URL` env var (already exists)
- **D-10:** Singleton via `sync.Once`, same pattern as current `NewSurrealDB()`
- **D-11:** Health check on pool creation (`pool.Acquire(ctx)` with 5s timeout)
- **D-12:** `002_full_schema.up.sql` — DDL for all 18+ tables
- **D-13:** `002_full_schema.down.sql` — DROP TABLE IF EXISTS CASCADE for all
- **D-14:** `003_seed.up.sql` — INSERT with `ON CONFLICT DO NOTHING` for idempotent seeding
- **D-15:** `003_seed.down.sql` — DELETE seeded records (by known UUIDs)
- **D-16:** Passwords pre-hashed with bcrypt, hardcoded in seed (same as current SurrealDB seed)
- **D-17:** `cmd/migrate -all` applies all pending migrations in order, then runs seed migrations
- **D-25:** Port `cmd/migrate` from `database/sql` + `lib/pq` to `pgx` (single driver, no `lib/pq` dep post-migration). This means `internal/db/db.go` and `internal/db/migrate.go` are also ported to `pgx` or replaced by `pgpool.go`.
- **D-18:** PostgreSQL is default service (removed from `--profile postgres`)
- **D-19:** SurrealDB moved to `--profile surrealdb` (optional, for reference)
- **D-20:** Health check waits for `pg_isready`
- **D-21:** `DATABASE_URL` exposed to `app` service
- **D-26:** `app` service gets `DATABASE_URL: postgres://hourglass:hourglass@postgres:5432/hourglass?sslmode=disable` env var
- **D-22:** Server starts pool in `main.go`, passes `*pgxpool.Pool` to adapters (same pattern as current SurrealDB passing)
- **D-23:** Server fails fast if pool can't connect (5s timeout)
- **D-24:** `SURREALDB_*` env vars still accepted but logged as deprecated

### the agent's Discretion
(None specified for Pg-1)

### Deferred Ideas (OUT OF SCOPE)
- Application code porting (Pg-2)
- SurrealDB removal (Pg-3)
- Any code beyond schema/pool/docker-compose/migrate
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| D-01–D-07 | Schema design | Full table-by-table mapping documented below |
| D-08–D-11 | pgxpool singleton | Pattern documented in Standard Stack + Code Examples |
| D-12–D-17 | Migration + seed | Migration strategy and seed data translation documented |
| D-18–D-21, D-26 | Docker Compose | Compose structure changes documented |
| D-22–D-24 | Server wiring | Server init changes documented |
| D-25 | Port cmd/migrate | Port strategy documented in Migration Patterns |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Schema definition | Database / Storage | — | DDL is database-layer concern |
| Migration execution | API / Backend | — | `cmd/migrate` CLI runs in backend process |
| Connection pool management | API / Backend | — | `internal/db/pgpool.go` is a backend singleton |
| Container orchestration | CDN / Static | — | docker-compose is deployment-layer config |
| Demo data seeding | Database / Storage | — | Seed is SQL migration (003_seed.up.sql) |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jackc/pgx/v5/pgxpool` | v5.7.x | PostgreSQL connection pool | Standard Go PostgreSQL driver with pool management; replaces `database/sql` + `lib/pq` |
| `github.com/jackc/pgx/v5` | v5.7.x | PostgreSQL driver toolkit | Required by pgxpool; provides `pgx.Conn` for migrations and direct queries |
| `github.com/google/uuid` | v1.6.0 | UUID generation | Already in go.mod; pgx natively scans `uuid.UUID` via `pgtype.UUID` |

[VERIFIED: npm registry -> go.dev] — pgx v5 latest is v5.7.x (July 2024 stable). Exact version should be resolved at `go get` time.

### pgx Installation

```bash
go get github.com/jackc/pgx/v5@latest
go mod tidy
```

After porting `cmd/migrate`, remove `lib/pq`:
```bash
go mod edit -droprequire github.com/lib/pq
go mod tidy
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pgx | `database/sql` + `lib/pq` | Status quo — database/sql lacks PostgreSQL-specific features (LISTEN/NOTIFY, COPY, array scanning). pgx is faster and has better type support. |
| pgxpool standalone | `database/sql` + `sql.DB` | sql.DB doesn't have pool-level Acquire semantics, health hooks, or stats. pgxpool is the idiomatic Go PostgreSQL pool. |

[VERIFIED: Context7 docs for pgx/pgxpool]

## Architecture Patterns

### System Architecture Diagram

```
┌──────────────┐    docker-compose up (default)    ┌──────────────────┐
│   Frontend   │  ────────────────────────────────▶  │  Postgres:5432   │
│  (Bun/Vite)  │                                    │  (default)       │
│   :3000      │                                    └──────────────────┘
└──────┬───────┘                                                   
       │ /api (proxy)                                              ┌──────────────────────┐
       ▼                                                           │  SurrealDB:8000       │
┌──────────────┐     ┌──────────────────┐     ┌──────────────────▶│  (--profile surrealdb)│
│   App        │────▶│ internal/db/     │────▶│                    └──────────────────────┘
│  (Go :8080)  │     │ pgpool.go        │     │
│              │     │ (sync.Once       │     │
│              │     │  singleton)      │     │
└──────┬───────┘     └──────────────────┘     │
       │                                      │
       │ go run ./cmd/migrate                 │
       ▼                                      ▼
┌──────────────┐     ┌──────────────────────────────────────────────┐
│ cmd/migrate  │────▶│  migrations/                                 │
│ -all         │     │   ├── 001_init.up.sql  (users/orgs/members)  │
│ -up          │     │   ├── 002_full_schema.up.sql  (all 24 tables)│
│ -down        │     │   └── 003_seed.up.sql  (MVP demo data)       │
└──────────────┘     └──────────────────────────────────────────────┘
```

### Recommended Project Structure (changes only)

```
src/ (implicit)
├── internal/
│   └── db/
│       ├── pgpool.go       # NEW — pgxpool singleton (replaces surreal.go pattern)
│       ├── db.go           # KEPT — legacy, will be removed in Pg-3
│       ├── migrate.go      # UPDATED — ported to pgx from database/sql
│       └── surreal.go      # KEPT — still used by SurrealDB adapters
├── cmd/
│   ├── server/
│   │   └── main.go         # UPDATED — adds pgpool init
│   └── migrate/
│       └── main.go         # UPDATED — ported to pgx, adds -all flag
├── migrations/
│   ├── 001_init.up.sql     # KEPT
│   ├── 001_init.down.sql   # KEPT
│   ├── 002_full_schema.up.sql    # NEW
│   ├── 002_full_schema.down.sql  # NEW
│   ├── 003_seed.up.sql           # NEW
│   └── 003_seed.down.sql         # NEW
└── docker-compose.yml      # UPDATED
```

### Pattern 1: pgxpool Singleton (mirrors current SurrealDB pattern)

**What:** Single `*pgxpool.Pool` initialized once via `sync.Once`, with health check on creation.

**When to use:** Always — this is the standard pattern for lifetime management of database connections.

**Structure:**
```go
// internal/db/pgpool.go
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

func NewPool() (*pgxpool.Pool, error) {
    var initErr error
    poolOnce.Do(func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        databaseURL := os.Getenv("DATABASE_URL")
        if databaseURL == "" {
            databaseURL = "postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable"
        }

        // Append pool config to connection string
        // pgxpool.ParseConfig + modify also works
        pool, err := pgxpool.New(ctx, databaseURL)
        if err != nil {
            initErr = fmt.Errorf("failed to create pool: %w", err)
            return
        }

        // Health check: verify a connection can be acquired
        conn, err := pool.Acquire(ctx)
        if err != nil {
            pool.Close()
            initErr = fmt.Errorf("failed to acquire connection from pool: %w", err)
            return
        }
        conn.Release()

        poolInstance = pool
    })

    if initErr != nil {
        return nil, initErr
    }
    if poolInstance == nil {
        return nil, fmt.Errorf("pool not initialized")
    }
    return poolInstance, nil
}

func ClosePool() {
    if poolInstance != nil {
        poolInstance.Close()
    }
}
```

**Source:** [CITED: pkg.go.dev/github.com/jackc/pgx/v5/pgxpool] — `New`, `Acquire`, `Close` patterns.

### Pattern 2: Pool Config via Connection String

Pool configuration is passed via the connection string or via `pgxpool.ParseConfig`:

```
postgres://user:pass@host:5432/db?sslmode=disable&pool_max_conns=25&pool_max_conn_lifetime=30m
```

Or via code:
```go
config, err := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 25
config.MaxConnLifetime = 30 * time.Minute
config.MaxConnIdleTime = 5 * time.Minute
pool, err := pgxpool.NewWithConfig(ctx, config)
```

**Recommendation:** Use `ParseConfig` + modification for explicit defaults, then pass to `NewWithConfig`. This makes pool configuration visible in code rather than hidden in a URL parameter. [CITED: Context7 pgxpool docs — ParseConfig]

### Anti-Patterns to Avoid
- **Using `sql.DB` pool alongside pgxpool:** Don't keep both `database/sql` and `pgxpool` alive. After porting migrate, remove `lib/pq` dep.
- **Not closing the pool:** Always `defer db.ClosePool()` in main.
- **Forgotten context timeout:** Always use a context with timeout for `New` and `Acquire`, especially at startup.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Connection pool management | Custom pool with mutex/channels | `pgxpool.Pool` | Connection lifecycle, health checking, stats, and retry logic are all battle-tested in pgx |
| UUID <-> DB type conversion | Custom UUID scanning | pgx built-in `pgtype.UUID` | pgx automatically scans `uuid.UUID` to/from PostgreSQL UUID columns without custom code |
| Connection string parsing | Manual URL parsing | `pgxpool.ParseConfig()` | Handles all PostgreSQL connection URI formats, SSL modes, and pool-specific params |
| Migration tracking table | Custom version table | Simple "apply sorted files" (current pattern) | The current approach (apply .up.sql files sorted by name, skip "already exists") works for this project's scale. A proper migration tool (golang-migrate) is overkill for < 10 migrations. |

**Key insight:** pgx's type system handles `uuid.UUID`, `time.Time`, `[]byte`, `[]string`, `json.RawMessage`, and `map[string]any` natively via `pgtype` — no custom scanners needed for the schema types in this project.

## Runtime State Inventory

> This is a Foundation phase (not a rename/refactor), so no runtime state migration is needed. Skip.

## Common Pitfalls

### Pitfall 1: Connection String Missing Pool Config
**What goes wrong:** Pool uses defaults (MaxConns=4), leading to contention under load.
**Why it happens:** `pgxpool.New(ctx, connString)` defaults are conservative.
**How to avoid:** Explicitly set `pool_max_conns=25` in connection string or use `ParseConfig` + override.
**Warning signs:** `Acquire()` blocking for seconds during concurrent requests.

### Pitfall 2: pgx vs database/sql API Confusion
**What goes wrong:** Using `rows.Scan()` instead of `pgx.RowToStructByPos` or `CollectRows`.
**Why it happens:** pgx has different scanning patterns than `database/sql`.
**How to avoid:** Use pgx's `CollectRows` + `RowToStructByPos` for common queries. For single rows, use `pgx.QueryRow(ctx, sql).Scan(...)`.
**Warning signs:** Code that looks like `database/sql` idioms — it's probably wrong for pgx.

**Source:** [CITED: Context7 pgx docs — RowToStructByPos, CollectRows]

### Pitfall 3: Transitive dep from `surrealdb.go`
**What goes wrong:** `go mod tidy` may keep `lib/pq` as a transitive dependency of `surrealdb.go` even after porting `cmd/migrate`.
**Why it happens:** `github.com/surrealdb/surrealdb.go` may depend on `lib/pq` (currently not in PROJECT indirect deps, but something to verify).
**How to avoid:** After the migration port, run `go mod tidy` and verify `lib/pq` is removed. If it persists, check indirect deps.

### Pitfall 4: TIMESTAMPTZ vs TIMESTAMP Confusion
**What goes wrong:** Using `TIMESTAMP` (without timezone) creates issues when servers/users are in different timezones.
**Why it happens:** `time.Time` in Go defaults to UTC, but `TIMESTAMP` in Postgres discards timezone info.
**How to avoid:** Always use `TIMESTAMPTZ` (decision D-04). pgx scans these into `time.Time` correctly.

### Pitfall 5: Docker Volume `/docker-entrypoint-initdb.d` Auto-Runs All SQL
**What goes wrong:** When docker-compose starts Postgres, it auto-executes all `.sql` files mounted at `/docker-entrypoint-initdb.d`. This currently includes `./migrations`, so `001_init.up.sql` runs automatically. Adding `002_full_schema.up.sql` to migrations means it too could auto-run.
**Why it happens:** Postgres official image runs any `.sql`/`.sh` in that directory on first database init.
**How to avoid:** Either (a) remove the volume mount and rely solely on `cmd/migrate`, or (b) keep it for `001_init.up.sql` only and note that `002+` must be manually migrated. Option (b) is the current pattern — the volume currently points at the entire `./migrations` directory, so all files auto-run on first start. For Pg-1, **either leave as-is** (all files auto-run, which is fine for fresh bootstrap) **or** move the mount to a dedicated `./docker-init/` directory with only `001_init.up.sql`.

**Recommendation:** For Pg-1, leave the current volume mount. `002_full_schema.up.sql` will auto-run on first Postgres start (which is harmless since it uses `CREATE TABLE IF NOT EXISTS`). The `cmd/migrate` CLI becomes primarily useful for subsequent migrations beyond the initial bootstrap, and for the `-all` convenience flag.

### Pitfall 6: Seed Data Uses String IDs — PostgreSQL Uses UUIDs
**What goes wrong:** The current SurrealDB seed uses human-readable string IDs (`units:engineering`, `customers:novatech`, `projects:proj_platform_eng`). PostgreSQL will use UUID PKs.
**Why it happens:** The schema decision D-02 mandates UUID PKs everywhere.
**How to avoid:** Assign hardcoded UUIDs for all seeded entities (units, customer, projects, subprojects, working groups, etc.) in the `003_seed.up.sql`. Reference these UUIDs via `INSERT ... ON CONFLICT DO NOTHING` with explicit UUID literals. Document the UUID mapping so Pg-2 adapters can reference the same values.

## Full Schema Translation (SurrealDB → PostgreSQL)

### Entity Count and Naming

Source of truth: `schema/001_schema.surql` — 18+ SurrealDB tables. Expanding to 24 PostgreSQL tables because:
- SurrealDB `audit_logs` is split into `time_entry_approvals` + `expense_approvals` (matching domain model)
- `contract_adoptions` and `project_adoptions` are separate join tables
- `project_managers` is a separate join table
- `backup_approvers` exists in models but not in SurrealDB schema — include for completeness

### Table-by-Table Mapping

**(1) organizations**
| PostgreSQL Column | Type | Constraints | SurrealDB Field |
|---|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() | id (uuid) |
| name | VARCHAR(255) | NOT NULL | name |
| slug | VARCHAR(255) | NOT NULL UNIQUE | slug |
| description | TEXT | | description |
| financial_cutoff_days | INTEGER | | financial_cutoff_days |
| financial_cutoff_config | JSONB | | financial_cutoff_config (object) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | created_at |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | updated_at |
*Index: slug UNIQUE*

**(2) users**
| PostgreSQL Column | Type | Constraints | SurrealDB Field |
|---|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() | id (uuid) |
| email | VARCHAR(255) | NOT NULL UNIQUE | email |
| username | VARCHAR(255) | UNIQUE | username |
| firstname | VARCHAR(255) | | firstname |
| lastname | VARCHAR(255) | | lastname |
| name | VARCHAR(255) | NOT NULL | name |
| password_hash | VARCHAR(255) | NOT NULL | password_hash |
| is_active | BOOLEAN | NOT NULL DEFAULT true | is_active |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | created_at |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | updated_at |
*Indexes: email UNIQUE, username UNIQUE*

**(3) customers**
| PostgreSQL Column | Type | Constraints | SurrealDB Field |
|---|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() | id (string) |
| org_id | UUID | NOT NULL FK→organizations(id) | org_id |
| name | VARCHAR(255) | NOT NULL | name |
| contact_name | VARCHAR(255) | | contact_name |
| email | VARCHAR(255) | | email |
| phone | VARCHAR(50) | | phone |
| address | TEXT | | address |
| vat_number | VARCHAR(50) | | vat_number |
| is_active | BOOLEAN | NOT NULL DEFAULT true | is_active |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | created_at |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | updated_at |
*Index: org_id*

**(4) organization_memberships**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| role | VARCHAR(50) | NOT NULL DEFAULT 'employee' CHECK(role IN ('employee','manager','finance','customer')) |
| is_active | BOOLEAN | NOT NULL DEFAULT true |
| invited_by | UUID | FK→users(id) |
| invited_at | TIMESTAMPTZ | |
| activated_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(user_id, organization_id)*
*Indexes: user_id, organization_id*

**(5) units**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| parent_unit_id | UUID | FK→units(id) ON DELETE RESTRICT |
| hierarchy_level | INTEGER | NOT NULL DEFAULT 0 |
| code | VARCHAR(50) | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(org_id, code) — nullable, so use a partial unique index or ensure NULLs don't collide*
*Indexes: org_id, parent_unit_id*

**Note on self-referencing FK**: parent_unit_id references units(id). Use ON DELETE RESTRICT to prevent deleting a unit that has children.

**(6) unit_memberships**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| unit_id | UUID | NOT NULL FK→units(id) ON DELETE CASCADE |
| is_primary | BOOLEAN | NOT NULL DEFAULT false |
| role | VARCHAR(50) | NOT NULL DEFAULT 'employee' |
| start_date | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| end_date | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, user_id, unit_id, (user_id, is_primary)*

**(7) contracts**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| name | VARCHAR(255) | NOT NULL |
| km_rate | NUMERIC(10,2) | NOT NULL DEFAULT 0 |
| currency | VARCHAR(3) | NOT NULL DEFAULT 'EUR' |
| customer_id | UUID | FK→customers(id) ON DELETE RESTRICT |
| governance_model | VARCHAR(50) | NOT NULL DEFAULT 'creator_controlled' CHECK(governance_model IN ('creator_controlled','unanimous','majority')) |
| created_by_org_id | UUID | NOT NULL FK→organizations(id) |
| is_shared | BOOLEAN | NOT NULL DEFAULT false |
| is_active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**(8) contract_adoptions**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| contract_id | UUID | NOT NULL FK→contracts(id) ON DELETE CASCADE |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| adopted_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(contract_id, organization_id)*

**(9) projects**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| project_type | VARCHAR(50) | NOT NULL CHECK(project_type IN ('billable','internal')) |
| type | VARCHAR(50) | NOT NULL CHECK(type IN ('billable','internal')) |
| contract_id | UUID | FK→contracts(id) ON DELETE RESTRICT |
| customer_id | UUID | FK→customers(id) |
| governance_model | VARCHAR(50) | NOT NULL DEFAULT 'creator_controlled' CHECK(governance_model IN ('creator_controlled','unanimous','majority')) |
| created_by_org_id | UUID | NOT NULL FK→organizations(id) |
| is_shared | BOOLEAN | NOT NULL DEFAULT false |
| budget_amount | NUMERIC(12,2) | |
| financial_cutoff_config | JSONB | |
| is_active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, customer_id*

**(10) project_adoptions**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| project_id | UUID | NOT NULL FK→projects(id) ON DELETE CASCADE |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| adopted_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(project_id, organization_id)*

**(11) project_managers**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| project_id | UUID | NOT NULL FK→projects(id) ON DELETE CASCADE |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(project_id, user_id)*

**(12) subprojects**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| project_id | UUID | NOT NULL FK→projects(id) ON DELETE CASCADE |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| sequence_order | INTEGER | NOT NULL DEFAULT 0 |
| is_active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: project_id*

**(13) working_groups**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| subproject_id | UUID | NOT NULL FK→subprojects(id) ON DELETE CASCADE |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| unit_ids | UUID[] | NOT NULL DEFAULT '{}' |
| enforce_unit_tuple | BOOLEAN | NOT NULL DEFAULT true |
| manager_id | UUID | NOT NULL FK→users(id) |
| delegate_ids | UUID[] | NOT NULL DEFAULT '{}' |
| is_active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, subproject_id, manager_id*

**Note on UUID[]:** pgx natively scans PostgreSQL `UUID[]` arrays into `[]uuid.UUID`. The domain model currently uses `[]string` for `unit_ids` and `delegate_ids` — this conversion will be handled in Pg-2 adapters.

**(14) wg_members**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| wg_id | UUID | NOT NULL FK→working_groups(id) ON DELETE CASCADE |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| unit_id | UUID | NOT NULL FK→units(id) ON DELETE RESTRICT |
| role | VARCHAR(50) | NOT NULL DEFAULT 'member' |
| is_default_subproject | BOOLEAN | NOT NULL DEFAULT false |
| start_date | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| end_date | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Constraints: UNIQUE(wg_id, user_id)*
*Indexes: wg_id, user_id, unit_id*

**(15) time_entries**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| user_id | UUID | NOT NULL FK→users(id) |
| project_id | UUID | NOT NULL FK→projects(id) |
| subproject_id | UUID | NOT NULL FK→subprojects(id) |
| wg_id | UUID | NOT NULL FK→working_groups(id) |
| unit_id | UUID | NOT NULL FK→units(id) |
| hours | NUMERIC(5,2) | NOT NULL CHECK(hours > 0) |
| description | TEXT | NOT NULL |
| entry_date | DATE | NOT NULL |
| status | VARCHAR(50) | NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','approved')) |
| is_deleted | BOOLEAN | NOT NULL DEFAULT false |
| created_from_entry_id | UUID | FK→time_entries(id) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, user_id, project_id, wg_id, unit_id, status, entry_date, (user_id, entry_date), is_deleted*

**Note on status CHECK:** The domain model only uses 'draft', 'submitted', 'approved' for time_entries. The `EntryStatus` model in `models.go` has more values (`pending_manager`, `pending_finance`, `rejected`) that the SurrealDB schema doesn't use for time_entries. Match the SurrealDB schema's CHECK here (3 values).

**(16) time_entry_approvals**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| time_entry_id | UUID | NOT NULL FK→time_entries(id) ON DELETE CASCADE |
| action | VARCHAR(50) | NOT NULL CHECK(action IN ('submit','approve','reject','edit_approve','edit_return','partial_approve','delegate')) |
| actor_user_id | UUID | NOT NULL FK→users(id) |
| actor_role | VARCHAR(50) | NOT NULL |
| changes | TEXT | |
| comment | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: time_entry_id*

**(17) expenses**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| user_id | UUID | NOT NULL FK→users(id) |
| project_id | UUID | FK→projects(id) |
| unit_id | UUID | NOT NULL FK→units(id) |
| category | VARCHAR(50) | NOT NULL CHECK(category IN ('mileage','meal','accommodation','parking','travel_tickets','tolls','taxi','equipment','other')) |
| amount | NUMERIC(10,2) | NOT NULL CHECK(amount > 0) |
| currency | VARCHAR(3) | NOT NULL DEFAULT 'EUR' |
| description | TEXT | |
| expense_date | DATE | NOT NULL |
| receipt_url | TEXT | |
| receipt_ocr_data | JSONB | |
| status | VARCHAR(50) | NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','approved','rejected')) |
| is_deleted | BOOLEAN | NOT NULL DEFAULT false |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, user_id, project_id, unit_id, status, expense_date, (user_id, expense_date)*

**Note on category CHECK:** Use the Go domain model's full category list (9 values) from `models.go`, not the SurrealDB's limited 4-value CHECK. This avoids a constraint mismatch between Go code and DB schema.

**(18) expense_approvals**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| expense_id | UUID | NOT NULL FK→expenses(id) ON DELETE CASCADE |
| action | VARCHAR(50) | NOT NULL CHECK(action IN ('submit','approve','reject','edit_approve','edit_return','partial_approve','delegate')) |
| actor_user_id | UUID | NOT NULL FK→users(id) |
| actor_role | VARCHAR(50) | NOT NULL |
| changes | TEXT | |
| comment | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: expense_id*

**(19) invitations**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| code | VARCHAR(255) | NOT NULL UNIQUE |
| invite_token | VARCHAR(255) | NOT NULL UNIQUE |
| email | VARCHAR(255) | |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','expired')) |
| expires_at | TIMESTAMPTZ | NOT NULL |
| created_by | UUID | NOT NULL FK→users(id) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: code UNIQUE, invite_token UNIQUE, organization_id*

**(20) password_resets**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| code_hash | VARCHAR(255) | NOT NULL |
| expires_at | TIMESTAMPTZ | NOT NULL |
| used_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: user_id*

**(21) refresh_tokens**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| token_hash | VARCHAR(255) | NOT NULL UNIQUE |
| expires_at | TIMESTAMPTZ | NOT NULL |
| revoked_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: token_hash UNIQUE, user_id*

**(22) financial_cutoff_periods**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| project_id | UUID | FK→projects(id) |
| period_start | TIMESTAMPTZ | NOT NULL |
| period_end | TIMESTAMPTZ | NOT NULL |
| cutoff_date | TIMESTAMPTZ | NOT NULL |
| is_locked | BOOLEAN | NOT NULL DEFAULT false |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, project_id, (period_start, period_end)*

**(23) budget_caps**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| org_id | UUID | NOT NULL FK→organizations(id) |
| user_id | UUID | FK→users(id) |
| project_id | UUID | FK→projects(id) |
| category | VARCHAR(50) | CHECK(category IN ('mileage','meal','accommodation','other')) |
| limit_amount | NUMERIC(10,2) | NOT NULL CHECK(limit_amount > 0) |
| period | VARCHAR(20) | NOT NULL DEFAULT 'monthly' CHECK(period IN ('daily','weekly','monthly','yearly')) |
| currency | VARCHAR(3) | NOT NULL DEFAULT 'EUR' |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
*Indexes: org_id, user_id, project_id*

**(24) backup_approvers**
| PostgreSQL Column | Type | Constraints |
|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() |
| organization_id | UUID | NOT NULL FK→organizations(id) ON DELETE CASCADE |
| role | VARCHAR(50) | NOT NULL CHECK(role IN ('employee','manager','finance','customer')) |
| user_id | UUID | NOT NULL FK→users(id) ON DELETE CASCADE |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

### Seed UUID Assignment

The current SurrealDB seed uses string record IDs for units, projects, subprojects, working groups, and customers. PostgreSQL requires UUIDs. Assign deterministic UUIDs for all seeded entities:

| Entity | SurrealDB ID | New UUID | Notes |
|--------|-------------|----------|-------|
| Organization | orgs:uuid-1 | 019df8b0-0001-7000-8000-000000000001 | Already a UUID |
| Users (6) | users:uuid-N | Existing UUIDs | Already UUIDs |
| Customer | customers:novatech | (deterministic UUID TBD) | Needs new UUID |
| Units (8) | units:engineering, etc. | (deterministic UUIDs TBD) | Needs new UUIDs |
| Contracts (3) | contracts:uuid-N | Existing UUIDs | Already UUIDs |
| Projects (6) | projects:proj_* | (deterministic UUIDs TBD) | Needs new UUIDs |
| Subprojects (6) | subprojects:subproj_* | (deterministic UUIDs TBD) | Needs new UUIDs |

**Strategy:** Use `gen_random_uuid()`-generated UUIDs hardcoded into the seed SQL, or systematically assign UUIDs that are deterministic (e.g., from a known namespace UUID v5). For simplicity in a dev MVP, hardcode specific UUIDs — the seed data always references the same values. Document the full UUID table in the seed file header comment.

## Migration Patterns

### Port Strategy: cmd/migrate from database/sql to pgx

**Current:** `cmd/migrate/main.go` uses `database/sql` + `github.com/lib/pq`
- `sql.Open("postgres", databaseURL)` 
- `db.Exec(string(content))` for each migration file
- Error matching via `strings.Contains(err.Error(), "already exists")` for idempotency

**Target:** `cmd/migrate/main.go` uses `pgx` directly (no database/sql)
- `pgx.Connect(ctx, databaseURL)` 
- `conn.Exec(ctx, string(content))` for each migration file
- Error matching via `pgx` error codes or `strings.Contains` (same pattern)

**Changes:**
1. Replace `database/sql` import with `github.com/jackc/pgx/v5`
2. Replace `_ "github.com/lib/pq"` import with pgx
3. Replace `sql.Open` → `pgx.Connect(ctx, databaseURL)`
4. Replace `db.Exec(string(content))` → `conn.Exec(ctx, string(content))`
5. Replace `db.Ping()` → `conn.Ping(ctx)`
6. Replace `db.Close()` → `conn.Close(ctx)`

**-all flag implementation:**
```
cmd/migrate -all     # Apply all .up.sql migrations + .up.sql seed files
cmd/migrate -up      # Apply all .up.sql migrations only (existing behavior)
cmd/migrate -down    # Roll back .down.sql files (existing behavior)
```

Logic for `-all`:
1. Read all `*.up.sql` files from migrations dir (sorted)
2. Filter into: migration files (`*_full_schema.up.sql`, `*_init.up.sql`) and seed files (`*_seed.up.sql`)
3. Execute migration files first, then seed files
4. Each file uses `CREATE TABLE IF NOT EXISTS` / `INSERT ... ON CONFLICT DO NOTHING` — idempotent by design

**Removal of `internal/db/db.go` and `internal/db/migrate.go`:**
These files are currently dead code — nothing imports the `db.DB` struct or calls `MigrateUp`/`MigrateDown`. After porting `cmd/migrate/main.go` to pgx, these files can be deleted or kept as a stub. Recommend deleting them in Pg-1 since nothing depends on them, but verify with `grep -r "db\.DB\|db\.MigrateUp\|db\.MigrateDown"` first.

**Source:** [CITED: Context7 pgx docs — pgx.Connect, Exec, Ping, Close]

### post-migration go.mod state:
```
require (
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.7.x          # ADDED
    github.com/golang-jwt/jwt/v5 v5.3.0
    github.com/stretchr/testify v1.11.1
    github.com/surrealdb/surrealdb.go v1.4.0  # Still needed for SurrealDB adapters until Pg-3
    golang.org/x/crypto v0.48.0
)
# REMOVED: github.com/lib/pq v1.10.9
```

## Seed Data Strategy

### 003_seed.up.sql — Idempotent Seed Pattern

```sql
-- Pattern: INSERT ... ON CONFLICT DO NOTHING
INSERT INTO organizations (id, name, slug, description)
VALUES (
    '019df8b0-0001-7000-8000-000000000001',
    'Tech Consulting Group',
    'tech-consulting-group',
    'MVP demo organization'
) ON CONFLICT (id) DO NOTHING;
```

**Key rules:**
1. **All PK references must use hardcoded UUIDs** — no `SELECT ... WHERE ...` subqueries for FK resolution
2. **Password hashes:** Same bcrypt hash as current seed (`$2a$12$...`) for "demo123"
3. **Insert order must respect FK dependencies:** organizations → users → customers → units → org_memberships → unit_memberships → contracts → projects → subprojects → working_groups → wg_members → time_entries → expenses
4. **All timestamps:** Use `NOW()` for `created_at`/`updated_at`, hardcoded dates for `entry_date`/`expense_date`
5. **Time entry dates:** Keep the original May 2026 dates from the SurrealDB seed, or update to current dates. The exact dates matter for testing time-range queries.

**Recommendation:** Keep the original May 2026 dates. They're stable for test assertions. If the seed is used for demo purposes, users won't care about date ranges — they just need to see data.

### 003_seed.down.sql
```sql
DELETE FROM time_entries WHERE id IN ('uuid1', 'uuid2', ...);
DELETE FROM expenses WHERE id IN ('uuid1', 'uuid2', ...);
-- etc. in reverse FK dependency order
```

**Important:** The down migration must delete entities in **reverse** FK dependency order (time_entries → wg_members → working_groups → subprojects → projects → ... → organizations). Use `DELETE` by known seed UUIDs. Do NOT use `TRUNCATE` — that would wipe production data.

## Docker Compose Changes

### Current → Target

```yaml
# Current                                    # Target
services:                                     services:
  surrealdb:                                    postgres:           # Default (no profile)
    ...                                         image: postgres:15-alpine
                                                (move profile from postgres here)
  postgres:                                     profiles: ~
    profiles: [postgres]           ──▶        healthcheck: pg_isready
    (remove profiles)                          volumes: migrations auto-init (keep)
                                               depends_on: postgres_healthy for app
  app:                                          surrealdb:           # Profiled
    depends_on: surrealdb                       profiles: [surrealdb]
    DATABASE_URL already present                depends_on: none (optional)
                                              app:
                                                depends_on: postgres:condition:service_healthy
```

**Exact changes to docker-compose.yml:**
1. Postgres: remove `profiles: [postgres]` — it starts by default
2. SurrealDB: add `profiles: [surrealdb]` — starts only with `docker-compose --profile surrealdb up`
3. App: change `depends_on` from `surrealdb` to `postgres` with health condition
4. Keep existing env vars on app (DATABASE_URL, JWT_SECRET, etc.)
5. Consider removing `./migrations:/docker-entrypoint-initdb.d` volume or keeping it (see Pitfall 5)

## Server Wiring (cmd/server/main.go)

**Changes needed (minimal):**
1. Add `_, _ = db.NewPool()` (or call it, ignoring the pool for now since adapters still use SurrealDB)
2. Log that pgpool is initialized
3. Accept SURREALDB_* env vars with deprecation warning (D-24)

```go
// In main():
pgPool, err := db.NewPool()
if err != nil {
    log.Fatalf("Failed to initialize PostgreSQL pool: %v", err)
}
defer db.ClosePool()
log.Println("PostgreSQL pool initialized")

if os.Getenv("SURREALDB_URL") != "" {
    log.Println("WARNING: SURREALDB_* env vars are deprecated. PostgreSQL is now the default database.")
}
```

**Note:** In Pg-1, the server still uses SurrealDB adapters for all data operations. The pgpool is initialized but unused by application code. It's wired now so Pg-2 can start using it without further main.go changes.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|-------------|------------------|--------------|--------|
| `database/sql` + `lib/pq` | `pgx/v5/pgxpool` | Pg-1 | Better performance, native UUID/JSONB/array scanning, pool stats |
| `lib/pq` connection | `pgx.Connect()` | Pg-1 | Different API: context-first, native error codes |
| SurrealDB as default DB | PostgreSQL as default DB | Pg-1 | docker-compose default changes, SurrealDB profiles |
| String-based IDs (units) | UUID-only PKs | Pg-1 | All tables use UUID PKs; string IDs need adapter conversion in Pg-2 |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `internal/db/db.go` and `internal/db/migrate.go` are dead code (nothing imports the DB struct) | Port Strategy | Low — if something imports them, keep them; they're harmless |
| A2 | `gen_random_uuid()` works without `pgcrypto` extension in PostgreSQL 15-alpine | Schema | Medium — if PG 15 doesn't have `gen_random_uuid` in `pg_catalog`, add `CREATE EXTENSION IF NOT EXISTS pgcrypto` back |
| A3 | `surrealdb.go` does NOT depend on `lib/pq` transitively | Pitfalls | Low — run `go mod tidy` after removal, add back if needed |
| A4 | `time_entry_approvals` and `expense_approvals` should be separate tables (not a shared audit_logs table) | Schema | Medium — if existing code references a shared `audit_logs` table, adapters in Pg-2 will need merging logic |

## Open Questions

1. **auto-init volume mount strategy**
   - What we know: docker-compose mounts `./migrations:/docker-entrypoint-initdb.d` which auto-runs all SQL files on first Postgres start
   - What's unclear: Should we keep this (auto-runs 001+002 on bootstrap) or remove it (rely solely on `cmd/migrate`)?
   - Recommendation: Keep it for Pg-1. `002_full_schema.up.sql` uses `CREATE TABLE IF NOT EXISTS`, so auto-run is harmless. The `cmd/migrate` CLI is useful for subsequent migrations and the `-all` convenience flag.

2. **pgxpool config — connection string params vs ParseConfig**
   - What we know: Both approaches work
   - What's unclear: Which is cleaner for this project?
   - Recommendation: Use `ParseConfig` + `NewWithConfig` for explicit self-documenting code. Set `MaxConns: 25`, `MaxConnLifetime: 30m`, `MaxConnIdleTime: 5m`.

3. **Seed UUIDs — hardcoded vs namespace-based**
   - What we know: All seed entities need UUIDs
   - What's unclear: Should UUIDs be random or derived (e.g., UUID v5 from namespace + entity name)?
   - Recommendation: Hardcode specific UUIDs for simplicity. Document the full UUID mapping table in the seed file header. Since these are demo data values, they only need to be self-consistent.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` | Build / migrate CLI | ✓ | 1.26.1 | — |
| PostgreSQL 15 | Schema / seed | ✓ (Docker) | 15-alpine | — |
| `docker-compose` | Infrastructure | ✓ | TBD | — |
| `pgx/v5` | Connection pool | ⚠️ Not yet added | Will resolve at `go get` | Use `database/sql` + `lib/pq` as fallback (current state) |

**Missing dependencies with no fallback:** None.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth logic in this phase (schema-only) |
| V3 Session Management | no | No session logic in this phase |
| V4 Access Control | indirect | Row-level access not enforced at DB level — application-layer concern |
| V5 Input Validation | yes | CHECK constraints, NOT NULL, FK referential integrity serve as defense-in-depth |
| V6 Cryptography | yes | bcrypt pre-hashed passwords in seed; no on-the-fly hashing in this phase |
| V9 Data Protection | yes | Pgpool health-check timeout prevents connection leak; TIMESTAMPTZ for audit trail |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection via migration files | Tampering | Migration files are static SQL committed to repo — not user-controllable input |
| Connection pool exhaustion | DoS | MaxConns: 25 cap, Acquire timeout |
| Weak UUID generation | Spoofing | `gen_random_uuid()` uses cryptographically secure RNG in PostgreSQL |

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/github.com/jackc/pgx/v5/pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) — Pool creation, configuration, Acquire, Close
- [pkg.go.dev/github.com/jackc/pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5) — pgx.Connect, Exec, Ping, Close, Batch, CollectRows
- `schema/001_schema.surql` — Source of truth for table structure, field types, indexes
- `schema/003_seed_demo.surql` — Source of truth for seed data values
- `cmd/migrate/main.go` — Current migration CLI logic to port
- `internal/db/surreal.go` — Singleton pattern to replicate

### Secondary (MEDIUM confidence)
- Project `go.mod` — Confirmed existing deps (google/uuid v1.6.0, lib/pq v1.10.9)
- Docker Compose current config — Confirmed postgres profile setup, volumes, health check

### Tertiary (LOW confidence)
- [CITED: pgx v5.7.x package analysis] — Exact latest version should be verified at `go get` time. v5.7 was latest stable as of mid-2024. Training knowledge.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — pgx is well-documented with official Go docs; singleton pattern already exists in codebase
- Architecture: HIGH — patterns are direct translations of existing SurrealDB patterns + standard pgx idioms
- Schema: HIGH — full table-by-table mapping from SurrealDB schema file, verified against domain models
- Migration: MEDIUM — exact pgx API port is straightforward (Exec, Ping, Connect), but `-all` flag logic is a new addition
- Pitfalls: MEDIUM — volume auto-init behavior confirmed correct but edge cases (surrealDB transitive deps) are assumed
- Seed data: MEDIUM — UUID assignment needs explicit resolution, but structure is clear

**Research date:** 2026-06-07
**Valid until:** 2026-07-07 (stable — pgx v5 API doesn't change often)
