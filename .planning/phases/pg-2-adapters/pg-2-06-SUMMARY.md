---
phase: pg-2-adapters
plan: 06
type: execution
subsystem: postgres-repositories
tags: [time-entry, expense, export, audit-log]
requires: []
provides: [TimeEntryRepository, AuditLogRepository, ExpenseRepository, ExportRepository]
affects: [internal/core/ports, internal/models]
tech-stack:
  added: [pgx/v5, json.RawMessage]
  patterns: [dynamic WHERE building, full FK chain JOINs, scan helpers, soft delete]
key-files:
  created:
    - internal/adapters/secondary/postgres/time_entry_repository.go
    - internal/adapters/secondary/postgres/time_entry_repository_test.go
    - internal/adapters/secondary/postgres/expense_repository.go
    - internal/adapters/secondary/postgres/expense_repository_test.go
    - internal/adapters/secondary/postgres/export_repository.go
    - internal/adapters/secondary/postgres/export_repository_test.go
  modified:
    - internal/models/models.go (added UnitID to Expense struct)
metrics:
  duration: ~25 min
  completed: 2026-06-07
---

# Phase pg-2-adapters Plan 06: Complex Repos (TimeEntry, AuditLog, Expense, Export) Summary

PostgreSQL implementation of four complex repositories with full integration tests: TimeEntry (7 methods + dynamic query builder), AuditLog (1 method), Expense (5 methods), and Export (2 methods with FK chain JOINs).

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TimeEntryRepository + AuditLogRepository + tests | `26bff17` | `time_entry_repository.go`, `time_entry_repository_test.go` |
| 2 | ExpenseRepository + tests | `211303a` | `expense_repository.go`, `expense_repository_test.go`, `models.go` |
| 3 | ExportRepository + tests | `7cbc9bb` | `export_repository.go`, `export_repository_test.go` |

## Detailed Implementation

### Task 1 — TimeEntryRepository & AuditLogRepository

**TimeEntryRepository** implements `ports.TimeEntryRepository` with 7 methods:

- **List** — Uses `buildTimeEntryListQuery` helper that dynamically builds WHERE clauses with numbered PostgreSQL placeholders. Always includes `org_id = $1` and `is_deleted = $2`, then appends filters for Date, Month, Year, Status, WGID, ProjectID, UserID.
- **GetByID** — Simple SELECT by PK. Maps `pgx.ErrNoRows` → `time_entry.ErrTimeEntryNotFound`.
- **Create** — INSERT with `RETURNING *`. Generates UUID server-side. Supports nullable `created_from_entry_id`.
- **Update** — Full-field dynamic UPDATE with `RETURNING *`. Maps `pgx.ErrNoRows` → `ErrTimeEntryNotFound`.
- **Delete** — Soft delete: `SET is_deleted = true, updated_at = NOW()`.
- **IsPeriodLocked** — `SELECT EXISTS` on `financial_cutoff_periods` with date range overlap check and `is_locked = true`.
- **ListPending** — Filters for `status = 'submitted' AND is_deleted = false`. Optional `wg_manager` role filter scopes to working groups where user is manager or delegate.

**AuditLogRepository** implements `ports.AuditLogRepository`:
- **Create** — Inserts into `time_entry_approvals` with JSON-marshalled Changes as the `comment` column. Parses `log.EntryID` (string) → UUID.

**Tests:** Full FK chain seeding (org → user → customer → contract → project → subproject → working group → time entry). Tests cover Create→GetByID (with nullable `CreatedFromEntryID`), List with/without filters, Update, Delete (soft), IsPeriodLocked (locked/unlocked periods), ListPending (all/wg_manager filter), and WG-level access filtering.

### Task 2 — ExpenseRepository

**ExpenseRepository** implements `ports.ExpenseRepository` with 5 methods:

- **Create** — INSERT with RETURNING. Maps `Type` (ExpenseCategory) ↔ DB `category` column.
- **GetByID** — Simple SELECT. Returns `ports.ErrNotFound` on no rows.
- **ListByOrg** — Paginated with `ORDER BY expense_date DESC`, excludes soft-deleted.
- **Update** — Full-field UPDATE with RETURNING, includes `unit_id`.
- **Delete** — Soft delete: `SET is_deleted = true, updated_at = NOW()`.

**Model fix:** Added `UnitID *uuid.UUID` to `models.Expense` struct (missing from model but required by DB schema).

**Tests:** Create→GetByID (verifies KmDistance round-trip and nullable fields), ListByOrg (pagination, org isolation), Update (changes amount, category, KmDistance), Delete (soft, excludes from ListByOrg, error on non-existent).

### Task 3 — ExportRepository

**ExportRepository** implements `ports.ExportRepository` with 2 methods using 4-level LEFT JOIN chains:

- **Timesheets** — `time_entries → users → projects → contracts → customers`. Returns unified `ExportRow` with `NULL::decimal` for amount/km_distance (expense-specific fields).
- **Expenses** — `expenses → users → projects → contracts → customers`. Returns unified `ExportRow` with `NULL::decimal` for hours.
- **roleFilter** helper — Generates SQL for employee (filter by user_id) and manager (filter by user_id OR project_manager membership) roles.

**Tests:** Timesheets with FK chain, Expenses with FK chain, employee role filter, empty results.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 — Critical] Added UnitID to Expense struct**
- **Found during:** Task 2
- **Issue:** `models.Expense` was missing `UnitID` field, but the DB schema (`002_full_schema.up.sql`) has `unit_id UUID NOT NULL` on the expenses table. Without this field, the repository cannot create or query expenses.
- **Fix:** Added `UnitID *uuid.UUID` field to `models.Expense`
- **Files modified:** `internal/models/models.go`
- **Commit:** `211303a`

### Table Schema Notes

The `002_full_schema.up.sql` migration (full schema) uses a richer table definition than the individual migration files. The actual schema used by tests includes:
- `time_entries`: org_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, is_deleted, created_from_entry_id, etc.
- `expenses`: unit_id, category, amount, km_distance, currency, receipt_url, receipt_ocr_data, is_deleted, etc. (plus customer_id and type from phase2)
- `time_entry_approvals`: time_entry_id, user_id, action, comment, created_at

## Verification

- `go vet ./internal/adapters/secondary/postgres/` — PASS
- `go build ./internal/...` — PASS
- `go build ./internal/adapters/secondary/postgres/` — PASS

## Decisions Made

- Used `pgx.Rows` typed variables in ExportRepository instead of anonymous interfaces for cleaner code and direct compatibility with `pgxpool.Pool.Query()` return type
- Used `COALESCE` for nullable string fields in export queries to avoid pgx scanning NULL into non-pointer `string` types
- Soft delete pattern (is_deleted boolean + updated_at timestamp) consistent with full_schema design
- `buildTimeEntryListQuery` uses numbered placeholders ($1, $2...) with orgID and isDeleted as fixed first two parameters

## Key Learnings

- Dynamic query building with `fmt.Sprintf("...$%d...")` requires careful parameter tracking
- PostgreSQL `CREATE TABLE IF NOT EXISTS` does NOT add missing columns — the full_schema vs individual migration schema mismatch is resolved by the full_schema running first alphabetically
- pgx scans NULL into `*float64` as `nil` correctly, but scanning NULL into `string` requires COALESCE

## Self-Check: PASSED

- [x] All 3 tasks committed individually
- [x] All 6 new files exist
- [x] `go vet` and `go build` pass
- [x] SUMMARY.md written
