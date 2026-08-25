---
status: complete
quick_id: 260825-migration-ledger
date: 2026-08-25
---

# Summary — 260825-migration-ledger

**Concern:** CONCERNS.md #1 Migration version ledger missing.

**Changes:**
- `cmd/migrate/main.go` — added `ensureSchemaMigrations` (CREATE TABLE IF NOT EXISTS
  `schema_migrations(version text PRIMARY KEY, applied_at timestamptz)`). `migrateUp` now skips
  already-applied versions (checked via ledger, not error-string matching) and wraps each
  migration's SQL + ledger insert in one `sql.Tx` so a partial failure rolls back cleanly.
  `migrateDown` applies only versions present in the ledger and deletes the row on success.
- `internal/db/migrate.go` — identical ledger logic applied to `MigrateUp`/`MigrateDown`.

**Result:** Re-running migrations is now idempotent and recoverable; no more driver/locale-dependent
`"already exists"` / `"does not exist"` string matching; drift detection is possible via the ledger.

**Verification:** `go build` + `go vet` pass for both packages.
