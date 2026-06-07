# Phase Pg-1: Foundation — PostgreSQL schema, pool, docker-compose - Pattern Map

**Mapped:** 2026-06-07
**Files analyzed:** 13
**Analogs found:** 12 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `migrations/002_full_schema.up.sql` | migration (DDL) | schema | `migrations/001_init.up.sql` | exact |
| `migrations/002_full_schema.down.sql` | migration (DDL) | schema-rollback | `migrations/001_init.down.sql` | exact |
| `migrations/003_seed.up.sql` | migration (data) | CRUD-insert | `schema/003_seed_demo.surql` | role-match |
| `migrations/003_seed.down.sql` | migration (data) | CRUD-delete | `migrations/001_init.down.sql` | role-match |
| `internal/db/pgpool.go` | utility | infrastructure (pool) | `internal/db/surreal.go` | exact |
| `cmd/migrate/main.go` | CLI | request-response (migrate) | self (current `cmd/migrate/main.go`) | refactor |
| `docker-compose.yml` | config | infrastructure | self (current `docker-compose.yml`) | refactor |
| `cmd/server/main.go` | entry | request-response (init) | self (current `cmd/server/main.go`) | refactor |
| `Makefile` | config | build-tooling | self (current `Makefile`) | refactor |
| `AGENTS.md` | documentation | N/A | self (current `AGENTS.md`) | refactor |
| `go.mod` | config | module-definition | self (current `go.mod`) | refactor |
| `internal/db/db.go` | utility | infrastructure | N/A (dead code, removed) | N/A |
| `internal/db/migrate.go` | utility | migration | N/A (dead code, removed) | N/A |

## Pattern Assignments

### `migrations/002_full_schema.up.sql` (migration DDL, schema)

**Analog:** `migrations/001_init.up.sql` (lines 1-40)

**DDL pattern** (lines 1-12):
```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

**FK + CHECK constraint pattern** (lines 23-35):
```sql
-- Organization memberships table
CREATE TABLE organization_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('employee', 'manager', 'finance', 'customer')),
    is_active BOOLEAN DEFAULT TRUE,
    invited_by UUID REFERENCES users(id),
    invited_at TIMESTAMP WITH TIME ZONE,
    activated_at TIMESTAMP WITH TIME ZONE,
    activation_token VARCHAR(255) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);
```

**Index pattern** (lines 37-40):
```sql
-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_organization_memberships_user_id ON organization_memberships(user_id);
CREATE INDEX idx_organization_memberships_organization_id ON organization_memberships(organization_id);
```

**Key decisions for 002_full_schema.up.sql:**
- Follow `CREATE TABLE IF NOT EXISTS` for all 24 tables (D-01)
- UUID PK with `gen_random_uuid()` default on all tables (D-02)
- `ON DELETE CASCADE` where SurrealDB cascaded, `ON DELETE RESTRICT` where it blocked (D-03)
- `TIMESTAMPTZ` for all datetime fields, NOT NULL DEFAULT NOW() (D-04)
- JSONB for `financial_cutoff_config`, `receipt_ocr_data`, `audit_log changes` fields (D-05)
- CHECK constraints for all enum-like fields: role, status, category, governance_model, project_type, period, action (D-06)
- Indexes matching SurrealDB INDEX definitions from `schema/001_schema.surql` (D-07)

---

### `migrations/002_full_schema.down.sql` (migration DDL, schema-rollback)

**Analog:** `migrations/001_init.down.sql` (lines 1-12)

**Rollback pattern** (lines 1-12):
```sql
-- Drop indexes
DROP INDEX IF EXISTS idx_organization_memberships_organization_id;
DROP INDEX IF EXISTS idx_organization_memberships_user_id;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables (reverse FK dependency order)
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;

-- Drop extension
DROP EXTENSION IF EXISTS pgcrypto;
```

**Key decisions for 002_full_schema.down.sql:**
- DROP INDEX IF EXISTS for all indexes before their tables
- DROP TABLE IF EXISTS CASCADE for all 24 tables in **reverse FK dependency order**
- DROP EXTENSION IF EXISTS pgcrypto at the end

---

### `migrations/003_seed.up.sql` (migration data, CRUD-insert)

**Analog:** `schema/003_seed_demo.surql` (lines 1-1052)

**Idempotent insert pattern** (derived from SurrealDB seed, transposed to PostgreSQL):
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

**Insert order respects FK dependencies** (from SurrealDB seed line 9):
```
-- Entity dependency order:
--   organizations → users → customers → units →
--   organization_memberships → unit_memberships → contracts →
--   projects → subprojects → working_groups → wg_members →
--   time_entries → expenses
```

**Password hash pattern** (from SurrealDB seed line 44):
```sql
password_hash = '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6'
```

**Key decisions for 003_seed.up.sql:**
- All INSERTs use `ON CONFLICT (id) DO NOTHING` for idempotent re-runs (D-14)
- Passwords pre-hashed with bcrypt, same hash as current SurrealDB seed (D-16)
- Hardcoded UUIDs for all seed entities (not SurrealDB string IDs)
- Uses same 6 users, 8 units, 6 projects, 6 subprojects, 6 working groups, 12 time entries, 6 expenses as the SurrealDB seed

---

### `migrations/003_seed.down.sql` (migration data, CRUD-delete)

**Analog:** `migrations/001_init.down.sql` (lines 1-12) for structure

**Seed rollback pattern:**
```sql
-- Delete in reverse FK dependency order
DELETE FROM expenses WHERE id IN ('exp_uuid_1', 'exp_uuid_2', ...);
DELETE FROM time_entries WHERE id IN ('te_uuid_1', 'te_uuid_2', ...);
DELETE FROM wg_members WHERE id IN ('wgm_uuid_1', ...);
DELETE FROM working_groups WHERE id IN ('wg_uuid_1', ...);
DELETE FROM subprojects WHERE id IN ('subproj_uuid_1', ...);
DELETE FROM project_managers WHERE ...
DELETE FROM projects WHERE id IN ('proj_uuid_1', ...);
DELETE FROM contract_adoptions WHERE ...;
DELETE FROM contracts WHERE id IN ('contract_uuid_1', ...);
DELETE FROM unit_memberships WHERE ...;
DELETE FROM organization_memberships WHERE ...;
DELETE FROM units WHERE id IN ('unit_uuid_1', ...);
DELETE FROM customers WHERE id IN ('cust_uuid_1', ...);
DELETE FROM users WHERE id IN ('user_uuid_1', ...);
DELETE FROM organizations WHERE id = '019df8b0-0001-7000-8000-000000000001';
```

**Key decisions:**
- DELETE by known seed UUIDs (not TRUNCATE — would wipe production data)
- Reverse FK dependency order

---

### `internal/db/pgpool.go` (utility, infrastructure-pool)

**Analog:** `internal/db/surreal.go` (lines 1-72)

**Import pattern** (derived from `surreal.go` lines 1-10):
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
```

**Singleton pattern** (mirrors `surreal.go` lines 16-57):
```go
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

        config, err := pgxpool.ParseConfig(databaseURL)
        if err != nil {
            initErr = fmt.Errorf("failed to parse pool config: %w", err)
            return
        }
        config.MaxConns = 25
        config.MaxConnLifetime = 30 * time.Minute
        config.MaxConnIdleTime = 5 * time.Minute

        pool, err := pgxpool.NewWithConfig(ctx, config)
        if err != nil {
            initErr = fmt.Errorf("failed to create pool: %w", err)
            return
        }

        // Health check
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

**Key match points with surreal.go:**
- Same `sync.Once` pattern for singleton initialization (lines 16-19)
- Same package `db`, same error flow pattern (`var initErr error; once.Do(...); if initErr != nil { return nil, initErr }`)
- Same `getEnvOrDefault` helper pattern (can reuse existing `getEnvOrDefault` from `surreal.go`, already in same package)
- Health check analogous to `db.SignIn` + `db.Use` verification — both verify the connection works before returning

---

### `cmd/migrate/main.go` (CLI, request-response) — PORT to pgx

**Analog:** Current `cmd/migrate/main.go` (lines 1-124) + `cmd/schema/main.go` (lines 1-79)

**Import pattern (target)** — replacing `database/sql` + `lib/pq` with `pgx`:
```go
import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/jackc/pgx/v5"
)
```

**Connection pattern (target)** — replacing `sql.Open` + `db.Ping`:
```go
// Current (line 21-28):
db, err := sql.Open("postgres", databaseURL)
if err != nil { log.Fatalf(...) }
defer db.Close()
if err := db.Ping(); err != nil { log.Fatalf(...) }

// Target:
ctx := context.Background()
conn, err := pgx.Connect(ctx, databaseURL)
if err != nil { log.Fatalf(...) }
defer conn.Close(ctx)
if err := conn.Ping(ctx); err != nil { log.Fatalf(...) }
```

**Exec pattern (target)** — replacing `db.Exec(string(content))`:
```go
// Current (line 86):
db.Exec(string(content))

// Target:
conn.Exec(ctx, string(content))
```

**-all flag logic** — new addition (from RESEARCH.md lines 708-719):
```go
func isSeedFile(name string) bool {
    return strings.Contains(name, "seed")
}

func migrateAll(conn *pgx.Conn, dir string) error {
    files, _ := filepath.Glob(filepath.Join(dir, "*.up.sql"))
    sort.Strings(files)

    // Apply migration files first (init, full_schema)
    for _, file := range files {
        if isSeedFile(file) { continue }
        // ... apply
    }

    // Then apply seed files
    for _, file := range files {
        if !isSeedFile(file) { continue }
        // ... apply
    }
    return nil
}
```

**Key changes:**
- Replace `database/sql` + `_ "github.com/lib/pq"` with `"github.com/jackc/pgx/v5"`
- Add `"context"` import
- `sql.Open` → `pgx.Connect(ctx, databaseURL)`
- `db.Ping()` → `conn.Ping(ctx)`
- `db.Exec(string(content))` → `conn.Exec(ctx, string(content))`
- `db.Close()` → `conn.Close(ctx)`
- Add `-all` flag parsing
- `getCommand` function extended to also match `-all`
- Keep error matching via `strings.Contains(err.Error(), "already exists")` — same pattern

---

### `docker-compose.yml` (config, infrastructure)

**Analog:** Current `docker-compose.yml` (lines 1-57)

**Key changes:**

1. **Postgres — remove profiles (lines 33-34):**
```yaml
# Remove these lines to make Postgres default:
#    profiles:
#      - postgres
```

2. **SurrealDB — add profiles (after line 14):**
```yaml
    profiles:
      - surrealdb
```

3. **App — change depends_on (lines 50-52):**
```yaml
    # Current:
    depends_on:
      surrealdb:
        condition: service_healthy

    # Target:
    depends_on:
      postgres:
        condition: service_healthy
```

4. **App — SURREALDB_* env vars kept but deprecated (logged warning in server)**
   - SURREALDB_* env vars remain in compose for reference
   - DATABASE_URL already present (line 45)
   - SurrealDB connection vars kept for `--profile surrealdb` usage

---

### `cmd/server/main.go` (entry, request-response-init)

**Analog:** Current `cmd/server/main.go` (lines 1-245)

**Init pattern** (inserted after line 41, before `sdbConn, err := db.NewSurrealDB()`):
```go
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

**Key decisions:**
- `pgPool` is initialized but unused by application code in Pg-1 (adapters still use SurrealDB)
- `defer db.ClosePool()` for cleanup
- Server fails fast with `log.Fatalf` if pool can't connect (5s timeout via NewPool)
- SurrealDB connection code kept; SURREALDB_* env vars logged as deprecated (D-24)
- In Pg-2, adapters will receive `*pgxpool.Pool` — same pattern as current `sdbConn.DB()` passing

---

### `Makefile` (config, build-tooling)

**Analog:** Current `Makefile` (lines 1-34)

**Add migrate-all target** (after line 16):
```makefile
migrate-all:
	go run ./cmd/migrate -all -dir $(MIGRATIONS_DIR)
```

**Update db-init target** (lines 33-34) — keep as-is or update for new schema.

---

### `AGENTS.md` (documentation)

**Analog:** Current `AGENTS.md` (lines 1-217)

**Updates needed:**
- Line 10: Update database description: `PostgreSQL for application data (primary), SurrealDB still available via docker-compose --profile surrealdb`
- Line 78: Update docker-compose command: `docker-compose up --profile surrealdb` for SurrealDB
- Line 154-157: Update database initialization to reflect PostgreSQL as primary
- Line 161-165: Update env vars section: add pgxpool config, note SURREALDB_* as deprecated

---

### `go.mod` (config, module-definition)

**Analog:** Current `go.mod` (lines 1-24)

**Changes:**
```go
// Add:
require (
    // ... existing ...
    github.com/jackc/pgx/v5 v5.7.x
    // ... existing ...
)

// Remove:
// github.com/lib/pq v1.10.9  (after porting cmd/migrate)
```

Run:
```bash
go get github.com/jackc/pgx/v5@latest
go mod tidy

# After porting migrate:
go mod edit -droprequire github.com/lib/pq
go mod tidy
```

---

### `internal/db/db.go` and `internal/db/migrate.go` (removed, dead code)

**Verification:** Grep confirms nothing imports `db.DB`, `db.MigrateUp`, or `db.MigrateDown`:
- All SurrealDB adapters import `github.com/surrealdb/surrealdb.go` directly (aliased as `sdb`)
- Only `internal/db/surreal.go` is imported by `cmd/server/main.go`
- `db.go` and `migrate.go` are confirmed dead code

**Action:** These files can be deleted in Pg-1 since:
1. `db.go` — replaced by `pgpool.go` (new pgxpool singleton)
2. `migrate.go` — replaced by `cmd/migrate/main.go` (ported to pgx)

---

## Shared Patterns

### Singleton via sync.Once
**Source:** `internal/db/surreal.go` lines 16-57
**Apply to:** `internal/db/pgpool.go`

```go
var (
    poolInstance *pgxpool.Pool
    poolOnce     sync.Once
)

func NewPool() (*pgxpool.Pool, error) {
    var initErr error
    poolOnce.Do(func() {
        // ... init with health check ...
    })
    if initErr != nil {
        return nil, initErr
    }
    if poolInstance == nil {
        return nil, fmt.Errorf("pool not initialized")
    }
    return poolInstance, nil
}
```

### getEnvOrDefault Helper
**Source:** `internal/db/surreal.go` lines 67-72
**Apply to:** `internal/db/pgpool.go` (already in same package — reuse directly)

```go
func getEnvOrDefault(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

### Idempotent Migration Pattern
**Source:** `migrations/001_init.up.sql` (all tables use `CREATE TABLE IF NOT EXISTS`)
**Apply to:** `migrations/002_full_schema.up.sql` (all 24 tables), `migrations/003_seed.up.sql` (all INSERTs use `ON CONFLICT DO NOTHING`)

```sql
CREATE TABLE IF NOT EXISTS ...
INSERT INTO ... VALUES (...) ON CONFLICT (id) DO NOTHING;
```

### Error Handling: "already exists" Skipping
**Source:** `cmd/migrate/main.go` lines 86-92
**Apply to:** Ported `cmd/migrate/main.go` (keep same pattern, works with pgx too)

```go
if _, err := conn.Exec(ctx, string(content)); err != nil {
    if strings.Contains(err.Error(), "already exists") {
        log.Printf("Migration %s already applied, skipping", file)
        continue
    }
    return fmt.Errorf("failed to apply migration %s: %w", file, err)
}
```

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `go.mod` | config | module-definition | No analog needed — standard Go tooling |

## Metadata

**Analog search scope:** `migrations/`, `internal/db/`, `cmd/migrate/`, `cmd/server/`, `cmd/schema/`, `schema/`, `docker-compose.yml`, `Makefile`, `AGENTS.md`, `go.mod`
**Files scanned:** 13
**Pattern extraction date:** 2026-06-07
