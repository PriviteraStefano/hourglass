# Pg-2: PostgreSQL adapters

**Gathered:** 2026-06-07
**Status:** Not started

<domain>
## Phase Boundary

Port all SurrealDB repository implementations to PostgreSQL using `pgx`. Keep the same hexagonal service boundaries — domain models stay unchanged, service layer stays unchanged. Only the `internal/adapters/secondary/` package changes.

Repositories to create in `internal/adapters/secondary/postgres/`:
1. user_repository.go
2. organization_repo.go
3. organization_membership_repo.go
4. unit_repository.go
5. unit_member_repository.go
6. project_repository.go
7. subproject_repository.go
8. contract_repository.go
9. customer_repository.go
10. working_group_repository.go
11. wg_member_repository.go
12. time_entry_repository.go
13. expense_repository.go
14. audit_log_repository.go
15. invitation_repository.go
16. password_reset_repository.go
17. refresh_token_repo.go
18. export_repository.go

</domain>

<decisions>
## Implementation Decisions

### Query pattern
- **D-01:** Hand-written SQL with `pgx` — no ORM, no query builder
- **D-02:** `CollectRows` + `RowToStructByName` for list queries
- **D-03:** `pgxpool.QueryRow` + `Scan` for single-row queries
- **D-04:** `pgxpool.Exec` for INSERT/UPDATE/DELETE with `RETURNING` where needed
- **D-05:** `pgx.Batch` for transactional multi-writes (user+membership creation, etc.)

### JOIN translation
- **D-06:** SurrealDB nested subqueries become proper SQL JOINs
  - Before: `(SELECT VALUE name FROM users WHERE id = user_id LIMIT 1)[0]`
  - After: `LEFT JOIN users u ON u.id = um.user_id`
- **D-07:** Aggregate fields (counts, sums) use SQL aggregate functions directly

### Type mapping
- **D-08:** `uuid.UUID` → `pgtype.UUID` for scanning, native Go `uuid.UUID` for parameters
- **D-09:** `time.Time` → `pgtype.Timestamptz` (keep timezone awareness)
- **D-10:** `map[string]interface{}` → `pgtype.JSONB` (financial_cutoff_config, changes)
- **D-11:** `[]string` → `pgtype.Array[pgtype.Text]` (working_group.delegate_ids, unit_ids)

### File structure
- **D-12:** One file per repository, matching SurrealDB naming
- **D-13:** Each repo takes `*pgxpool.Pool` in constructor
- **D-14:** Same method signatures as current SurrealDB repos (services are unmodified)

### Testing
- **D-15:** Write PostgreSQL repository tests alongside each new repository
- **D-16:** Tests use a test PostgreSQL instance (not SurrealDB mock), matching the existing `*_repository_test.go` patterns in `surrealdb/`
- **D-17:** SurrealDB test files will be deleted in Pg-3 alongside the SurrealDB package

</decisions>

<canonical_refs>
## Canonical References

- `internal/adapters/secondary/surrealdb/*.go` — All 14 repository implementations to port
- `internal/adapters/secondary/surrealdb/models.go` — ToDomain/FromDomain patterns to replicate
- `internal/core/domain/*` — Domain models (types to scan into)
- `internal/core/services/*` — Service layer (interface consumers, stays unchanged)
- `migrations/002_full_schema.up.sql` — PostgreSQL table definitions (from Pg-1)
- `internal/db/pgpool.go` — Connection pool (from Pg-1)
- `docs/superpowers/specs/2026-06-07-postgresql-migration-design.md` — Full design doc

### Key translation patterns
- `sdb.Query("SELECT ...")` → `pool.Query(ctx, "SELECT ...")`
- `sdb.Create()` → `pool.Exec(ctx, "INSERT ... RETURNING *")` + `CollectRows`
- `sdb.Select()` → `pool.QueryRow(ctx, "SELECT ... WHERE id = $1")` + `Scan`
- `sdb.Merge()` → `pool.Exec(ctx, "UPDATE ... SET ... WHERE id = $1")`
- `sdb.Delete()` → `pool.Exec(ctx, "DELETE FROM ... WHERE id = $1")`

</canonical_refs>

<codebase_context>
## Existing Code Insights

### Repository size estimate
- ~3,000 lines total across 14 repository files
- ~1,200 lines in models.go (ToDomain/FromDomain + struct definitions)
- Most complex: time_entry (dynamic WHERE building), export (4-level JOIN), contract (aggregates)
- SurrealDB-specific boilerplate to remove: nil-check chains on Query results, CBOR tag handling, uuidToRecordID/recordIDToUUID

### Simplified patterns
- No more `if results == nil || len(*results) == 0` chains
- No more `recordIDToUUID` or `uuidToRecordID` conversion functions
- No more `cbor.Tag` type assertions
- Direct `row.Scan(&id, &name, ...)` or `CollectRows(rows, RowToStructByName[SurrealX])`
- UUIDs stored natively, no custom marshaling

</codebase_context>
