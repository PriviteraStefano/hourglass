# Deferred Items — 260801-o06 (demo postgres auth recovery)

## Pre-existing migration chain is not re-runnable (out of scope, NOT caused by this plan)

**Found during:** Task 1 verification (`docker compose run --rm migrate` re-run)

`docker compose run --rm migrate` exits 0 on a fresh DB, but **exits 1 on re-run**
against an already-migrated DB. The plan's premise "`migrate -all` is idempotent"
is false on two counts:

1. **`-all` was never a supported flag** in `cmd/migrate/main.go` (only `-up`/`-down`,
   confirmed by the only commit touching it, 6e0949f, and by `cmd/migrate/migrate_test.go`).
   The compose `migrate` service entrypoint used `-all`, so the one-shot had *always*
   exited 1 with a usage error — the demo DB had **zero tables** until this plan
   switched the entrypoint to `-up` (Rule 3 blocking fix, committed separately).
2. **Even `-up` is not idempotent**: `000_full_schema.up.sql` and
   `011_activity_ontology.up.sql` cannot re-run. Migration 011 renames/drops columns
   that 000's `CREATE INDEX IF NOT EXISTS` statements still reference (e.g. line 244
   `idx_working_groups_subproject_id ON working_groups(subproject_id)` fails with
   `column "subproject_id" does not exist` after 011 renamed it to `activity_id`).
   011 itself also uses unguarded `RENAME COLUMN` / `DROP COLUMN` / `DROP TABLE`.

**Why not fixed here:** pre-existing defect in `migrations/*` — outside the plan's
`files_modified` list, not caused by this task's changes. Making the chain re-runnable
means rewriting already-applied migration DDL (Rule 4 class: structural change to the
migration chain), which needs its own plan.

**Impact on this plan:** the demo IS recovered (app Up, pool initialized, health
`{"status":"ok"}`, seed exit 0, migrate exit 0 on first run proving auth). The only
unmet item is the migrate *re-run* link of Task 1's automated verify.

**Fix in a future plan:** either
- guard the 000/011 statements (`CREATE INDEX` on a column that may not exist, wrap
  renames/drops in existence checks), or
- give `cmd/migrate` a real applied-migrations tracker (schema_migrations table).

**Related stale docs found (do not propagate):** `AGENTS.md` claims "`cmd/migrate -all`
applies all migrations then seeds" — `-all` does not exist; the `Makefile` `migrate-all`
and `setup` targets pass `-all` and would fail identically.
