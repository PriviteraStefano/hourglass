# Pg-1: Foundation — PostgreSQL schema, pool, docker-compose

**Gathered:** 2026-06-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Create the PostgreSQL foundation: full schema migration, pgx connection pool, demo seed, docker-compose update. No application code porting (that's Pg-2).

Output:
- `migrations/002_full_schema.up.sql` (+ down.sql) — all tables
- `internal/db/pgpool.go` — pgxpool singleton
- `migrations/003_seed.up.sql` (+ down.sql) — MVP demo data
- Updated `docker-compose.yml` — Postgres default, SurrealDB optional
- Updated `cmd/migrate/main.go` — `-all` flag
- Updated `cmd/server/main.go` — init pgpool
- Updated `internal/db/` — remove SurrealDB singleton, add pgpool

</domain>

<decisions>
## Implementation Decisions

### Schema
- **D-01:** All tables `CREATE TABLE IF NOT EXISTS` for idempotent migrations
- **D-02:** UUID primary keys with `gen_random_uuid()` default
- **D-03:** Foreign keys with `ON DELETE CASCADE` where SurrealDB cascaded, `ON DELETE RESTRICT` where it blocked
- **D-04:** `TIMESTAMPTZ` for all datetime fields
- **D-05:** JSONB for flexible/object fields (financial_cutoff_config, audit_log changes)
- **D-06:** CHECK constraints for enum-like fields (role, status, category, etc.)
- **D-07:** Indexes matching current SurrealDB INDEX definitions

### Connection
- **D-08:** `github.com/jackc/pgx/v5/pgxpool` with `pool.MaxConns: 25`
- **D-09:** Reads `DATABASE_URL` env var (already exists)
- **D-10:** Singleton via `sync.Once`, same pattern as current `NewSurrealDB()`
- **D-11:** Health check on pool creation (`pool.Acquire(ctx)` with 5s timeout)

### Migration
- **D-12:** `002_full_schema.up.sql` — DDL for all 18+ tables
- **D-13:** `002_full_schema.down.sql` — DROP TABLE IF EXISTS CASCADE for all
- **D-14:** `003_seed.up.sql` — INSERT with `ON CONFLICT DO NOTHING` for idempotent seeding
- **D-15:** `003_seed.down.sql` — DELETE seeded records (by known UUIDs)
- **D-16:** Passwords pre-hashed with bcrypt, hardcoded in seed (same as current SurrealDB seed)
- **D-17:** `cmd/migrate -all` applies all pending migrations in order, then runs seed migrations
- **D-25:** Port `cmd/migrate` from `database/sql` + `lib/pq` to `pgx` (single driver, no `lib/pq` dep post-migration). This means `internal/db/db.go` and `internal/db/migrate.go` are also ported to `pgx` or replaced by `pgpool.go`.

### Docker
- **D-18:** PostgreSQL is default service (removed from `--profile postgres`)
- **D-19:** SurrealDB moved to `--profile surrealdb` (optional, for reference)
- **D-20:** Health check waits for `pg_isready`
- **D-21:** `DATABASE_URL` exposed to `app` service

### Server wiring
- **D-22:** Server starts pool in `main.go`, passes `*pgxpool.Pool` to adapters (same pattern as current SurrealDB passing)
- **D-23:** Server fails fast if pool can't connect (5s timeout)
- **D-24:** `SURREALDB_*` env vars still accepted but logged as deprecated

### Docker-compose
- **D-26:** `app` service gets `DATABASE_URL: postgres://hourglass:hourglass@postgres:5432/hourglass?sslmode=disable` env var

</decisions>

<canonical_refs>
## Canonical References

- `schema/001_schema.surql` — Current SurrealDB schema (source of truth for table structure)
- `schema/003_seed_demo.surql` — Current seed data (source of truth for demo data values)
- `internal/db/surreal.go` — Current DB singleton pattern (to replicate for pgpool)
- `cmd/migrate/main.go` — Existing migration CLI (to extend with -all flag)
- `docker-compose.yml` — Current compose file (to modify)
- `cmd/server/main.go` — Server entry point (to add pgpool init)
- `internal/core/domain/unit/unit.go` — Reference for domain model field types
- `internal/adapters/secondary/surrealdb/models.go` — Reference for field mappings
- `docs/superpowers/specs/2026-06-07-postgresql-migration-design.md` — Full design doc
</canonical_refs>

<codebase_context>
## Existing Code Insights

### Reusable Assets
- `cmd/migrate/main.go` — Migrations framework, reads `DATABASE_URL`, applies `.up.sql`/`.down.sql`
- `internal/db/surreal.go` — Singleton `sync.Once` pattern for connection management
- `docker-compose.yml` — Postgres already present (profile-gated), same credentials
- `migrations/001_init.up.sql` — Existing initial migration (can reference for style)

### Known Schema (from schema/001_schema.surql)
- 18+ tables: organizations, users, organization_memberships, units, unit_memberships, customers, projects, subprojects, contracts, project_managers, contract_adoptions, working_groups, wg_members, time_entries, expenses, audit_logs, invitations, password_resets, refresh_tokens, verification_tokens, financial_cutoff_periods, budget_caps

### Seed Data Values (from schema/003_seed_demo.surql)
- 6 users with UUIDs, 3 roles (manager, finance, employee)
- 8 units with hierarchy
- 3 contracts, 6 projects, 1 customer
- 15 time entries, 6 expenses

### Pain Points to Fix
- `contracts` table was schemaless in SurrealDB — now gets proper schema
- UUID format was inconsistent (string IDs for units, UUIDs for users) — now all UUID
- `parent_unit_id` was optional string — now proper UUID FK with self-reference

</codebase_context>
