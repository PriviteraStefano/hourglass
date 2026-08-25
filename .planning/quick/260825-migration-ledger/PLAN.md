---
quick_id: 260825-migration-ledger
description: Add schema_migrations ledger + per-file transactions to migration runner
date: 2026-08-25
status: complete
---

# Quick Task 260825-migration-ledger

## Plan

Address CONCERNS.md #1 "Migration version ledger missing".

Tasks:
1. `cmd/migrate/main.go`: introduce `schema_migrations(version, applied_at)` table; apply each
   `*.up.sql` only if not already recorded, inside a single transaction (exec + ledger insert).
   Same for `down` (reverse order, delete ledger row).
2. `internal/db/migrate.go`: apply the identical ledger logic to the unused `MigrateUp`/`MigrateDown`
   methods so the bug pattern is fixed at both sites.

## Verify

- `go build ./cmd/migrate ./internal/db` passes
- `go vet ./cmd/migrate ./internal/db` passes
- Manual DB run needed to confirm end-to-end (no local DB in CI here); logic is driver-agnostic.
