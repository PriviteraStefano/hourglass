---
phase: pg-3-wiring
plan: 01
subsystem: wiring
tags: postgres, surrealdb, migration, middleware, go

# Dependency graph
requires:
  - phase: pg-2-postgres-migration
    provides: postgres repository implementations, exported test helpers, pgpool singleton
provides:
  - TokenService and PasswordHasher adapters in internal/auth/ replacing surrealdb counterparts
  - CORS middleware extracted to internal/middleware/cors.go
  - Postgres-wired cmd/server/main.go with zero SurrealDB imports
  - Auth handler tests rewritten to use PostgreSQL adapters
affects: pg-3-cleanup, pg-3-verify

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Adapter pattern: internal/auth/ package wraps auth.Service to implement ports interfaces"
    - "Middleware extraction: inline closures extracted to internal/middleware/ package"

key-files:
  created:
    - internal/auth/token_service.go
    - internal/auth/password_hasher.go
    - internal/middleware/cors.go
  modified:
    - cmd/server/main.go
    - internal/adapters/primary/http/auth_test.go

key-decisions:
  - "TokenService adapter lives in internal/auth/ package (same package as auth.Service it wraps)"
  - "PasswordHasher wraps bcrypt directly (no auth.Service dependency needed)"
  - "CORS middleware follows func(http.Handler) http.Handler signature matching Logging/APIVersion"

patterns-established:
  - "Non-DB adapters for port interfaces live alongside their implementation in internal/auth/"

requirements-completed: [D-01, D-02, D-03, D-16, D-17, D-18, D-23]

# Metrics
duration: 3min
completed: 2026-06-07
---

# Phase Pg-3 Plan 01: Core Wiring Summary

**Postgres repo wiring in cmd/server/main.go, adapter files for TokenService/PasswordHasher, CORS middleware extraction, and auth test rewrite — zero SurrealDB imports remain in the server entrypoint.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-06-07T16:38:15Z
- **Completed:** 2026-06-07T16:41:06Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- Created `internal/auth/token_service.go` — `ports.TokenService` adapter wrapping `auth.Service` with `auth.Claims` → `ports.Claims` conversion in `ValidateToken`
- Created `internal/auth/password_hasher.go` — `ports.PasswordHasher` adapter wrapping bcrypt directly
- Created `internal/middleware/cors.go` — extracted from inline closure in main.go, follows `func(http.Handler) http.Handler` pattern
- Rewired `cmd/server/main.go` — all 17 SurrealDB repo constructors replaced with Postgres equivalents, SurrealDB init block and SURREALDB_* warning removed, inline `corsMiddleware` replaced with `middleware.CORS()`
- Rewrote `internal/adapters/primary/http/auth_test.go` — SurrealDB test setup replaced with `postgres.TestPool()`, `SetupTestSchema()`, `TeardownTestSchema()`, all 13 test functions preserved with identical assertion logic

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TokenService + PasswordHasher adapter files** - `d551680` (feat)
2. **Task 2: Create CORS middleware + rewrite cmd/server/main.go** - `fefd1a3` (feat)
3. **Task 3: Rewrite auth_test.go to use PostgreSQL adapters** - `6d9c238` (feat)

## Files Created/Modified

### Created
- `internal/auth/token_service.go` - 46 lines, ports.TokenService adapter wrapping auth.Service (4 methods)
- `internal/auth/password_hasher.go` - 23 lines, ports.PasswordHasher adapter wrapping bcrypt (2 methods)
- `internal/middleware/cors.go` - 37 lines, CORS middleware extracted from main.go inline closure

### Modified
- `cmd/server/main.go` - 216 lines (was 255), surrealdb→postgres rewire, CORS import change, ~40 lines removed
- `internal/adapters/primary/http/auth_test.go` - 749 lines preserved, imports/test setup rewritten (~40 lines changed)

## Decisions Made
- **TokenService in `internal/auth/`** rather than a separate adapter package — the wrapper lives alongside the `auth.Service` it delegates to, consistent with the pattern of co-locating adapters with their underlying implementation
- **PasswordHasher uses bcrypt directly** rather than delegating through `auth.Service` — bcrypt is the only dependency (same approach as the surrealdb adapter), avoiding unnecessary indirection through `auth.Service` methods that themselves just call bcrypt
- **CORS middleware as `func(allowedOrigins) func(http.Handler) http.Handler`** — matches the exact signature pattern of `middleware.APIVersion`, `middleware.Logging`, making it composable in the handler chain

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `internal/middleware/cors.go` initially had an unused `strings` import (carried over from the pattern reference in RESEARCH.md) — removed in the same commit after `go vet` caught it. Zero impact.

## Next Phase Readiness
- Ready for **Plan 2 (Cleanup)**: delete `internal/adapters/secondary/surrealdb/`, `internal/db/surreal.go`, `cmd/schema/main.go`, `schema/` directory, SurrealDB from docker-compose/Makefile/AGENTS.md, and run `go mod tidy`
- Ready for **Plan 3 (Verify)**: automated smoke test and manual verification flow

## Self-Check: PASSED

All 5 files exist on disk, all 3 commits verified in git log, zero surrealdb/hexauth references remaining in modified files.

---
*Phase: pg-3-wiring*
*Completed: 2026-06-07*
