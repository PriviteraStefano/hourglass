---
phase: pg-2-adapters
plan: 01
subsystem: database
tags: postgres, pgx, ports, sentinel-errors, user-finder

requires:
  - phase: pg-1-foundation
    provides: PostgreSQL schema (002_full_schema.up.sql), pgxpool singleton
provides:
  - Schema correction (remove users.name, add km_distance to expenses)
  - Shared sentinel errors (ErrNotFound, ErrConflict, ErrForeignKey)
  - ExpenseRepository port interface with 5 CRUD methods
  - SubprojectRepository port interface with 5 CRUD methods
  - wrapPGError helper for pgx→domain error translation
  - Test infrastructure (TestPool, SetupTestSchema, TeardownTestSchema, unique helpers)
  - UserFinder implementation (FindByIdentifier via pgx)
affects: pg-2-02, pg-2-03, pg-2-04, pg-2-05, pg-2-06

tech-stack:
  added: []
  patterns:
    - wrapPGError sentinel translation pattern
    - TestPool + SetupTestSchema integration test pattern
    - pgxpool.QueryRow + Scan for single-row queries

key-files:
  created:
    - internal/core/ports/errors.go
    - internal/core/ports/expense_repository.go
    - internal/core/ports/subproject_repository.go
    - internal/adapters/secondary/postgres/postgres.go
    - internal/adapters/secondary/postgres/exported_test_helpers.go
    - internal/adapters/secondary/postgres/user_finder.go
    - internal/adapters/secondary/postgres/user_finder_test.go
  modified:
    - migrations/002_full_schema.up.sql

key-decisions:
  - "ErrNotFound, ErrConflict, ErrForeignKey as generic sentinels in ports/errors.go (D-22)"
  - "ExpenseRepository and SubprojectRepository port interfaces created per D-20, D-21"

patterns-established:
  - "wrapPGError: single function translating pgx.ErrNoRows (23505, 23503) to domain sentinels"
  - "TestPool: skip-if-no-DATABASE_URL, then db.NewPool() — single source of test pool"
  - "SetupTestSchema: apply all non-seed migrations sorted alphabetically"
  - "FindByIdentifier: SELECT id FROM users WHERE email=$1 OR username=$1 LIMIT 1"

requirements-completed: []

duration: 5min
completed: 2026-06-07
---

# Phase Pg-2 Plan 01: Foundation Summary

**PostgreSQL adapter foundation — schema correction, shared sentinel errors, new port interfaces, wrapPGError helper, test infrastructure (TestPool/SetupTestSchema), and UserFinder implementation via pgx**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-07T14:55:52Z
- **Completed:** 2026-06-07T15:00:00Z
- **Tasks:** 2
- **Files modified:** 8 (7 created, 1 modified)

## Accomplishments

- Removed `name VARCHAR(255) NOT NULL` from users table (redundant with firstname+lastname per D-18)
- Added `km_distance DECIMAL(10,2)` to expenses table after `amount` (per D-19)
- Created `internal/core/ports/errors.go` with `ErrNotFound`, `ErrConflict`, `ErrForeignKey` sentinel errors
- Created `ExpenseRepository` port interface (5 CRUD methods) in ports/
- Created `SubprojectRepository` port interface (5 CRUD methods) in ports/
- Created `wrapPGError` helper translating pgx.ErrNoRows/23505/23503 to domain sentinels
- Created `TestPool` (skip-if-no-DATABASE_URL pattern) and `SetupTestSchema` (sorted migration apply)
- Created `TeardownTestSchema` (CASCADE table drop) and unique helpers (email, username, code)
- Implemented `UserFinder.FindByIdentifier` via pgx `QueryRow` — returns userID for email or username match
- Wrote 3 integration tests: by email, by username, not found (ErrUserNotFound)

## Task Commits

Each task was committed atomically:

1. **Task 1: Schema correction, sentinel errors, new port interfaces** - `1cb6418` (feat)
2. **Task 2: Package helpers, UserFinder, and tests** - `d44077f` (feat)

**Plan metadata:** (committed as part of final SUMMARY commit)

## Files Created/Modified

- `migrations/002_full_schema.up.sql` - Removed users.name, added km_distance to expenses
- `internal/core/ports/errors.go` - ErrNotFound, ErrConflict, ErrForeignKey sentinel errors
- `internal/core/ports/expense_repository.go` - ExpenseRepository port interface
- `internal/core/ports/subproject_repository.go` - SubprojectRepository port interface
- `internal/adapters/secondary/postgres/postgres.go` - wrapPGError error translator
- `internal/adapters/secondary/postgres/exported_test_helpers.go` - TestPool, SetupTestSchema, TeardownTestSchema, unique helpers
- `internal/adapters/secondary/postgres/user_finder.go` - UserFinder via pgx (implements ports.UserFinder)
- `internal/adapters/secondary/postgres/user_finder_test.go` - 3 integration tests

## Decisions Made

- Followed D-18, D-19, D-20, D-21, D-22 from context — all schema/port decisions executed as planned
- Used `errors.Is(err, pgx.ErrNoRows)` for row-not-found rather than string matching (T-pg2-03 mitigation)
- Used `pgconn.PgError.Code` for structured error detection per threat model T-pg2-01

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## Next Phase Readiness

- Foundation for all 6 PostgreSQL adapter plans is complete
- Downstream plans (pg-2-02 through pg-2-06) can import `wrapPGError`, use `TestPool`/`SetupTestSchema`, and implement interfaces from the new port files
- Schema corrections applied — downstream plans work with corrected column definitions
- Ready for pg-2-02 (UserRepository + Organization repositories)

## Self-Check: PASSED

- All 8 created/modified files verified on disk
- All 3 commits verified in git log:
  - `1cb6418` feat(pg-2-01): schema correction, sentinel errors, new port interfaces
  - `d44077f` feat(pg-2-01): postgres package helpers, UserFinder, and tests
  - `e0139f3` docs(pg-2-01): complete PostgreSQL adapter foundation plan
- `go build ./internal/core/ports/` — PASS
- `go vet ./internal/core/ports/` — PASS
- `go build ./internal/adapters/secondary/postgres/` — PASS
- `go vet ./internal/adapters/secondary/postgres/` — PASS

---
*Phase: pg-2-adapters*
*Completed: 2026-06-07*
