---
phase: 00-testing-foundation
plan: 02
subsystem: testing
tags: [testcontainers-go, postgres, integration-testing, test-infrastructure]

# Dependency graph
requires:
  - phase: pg-3
    provides: PostgreSQL backend with migrations
provides:
  - testcontainers-go v0.42 infrastructure for isolated PostgreSQL per-package
  - Container-backed TestPool replacing DATABASE_URL dependency
  - Module-root-relative migration path resolution
  - Fully self-contained smoke test (no external DB)
affects:
  - 00-01 (auth bug fixes — needs testcontainers to run)
  - 00-03 (service test rewrite)
  - 00-04 (handler test rewrite)
  - All future integration test phases

# Tech tracking
tech-stack:
  added:
    - github.com/testcontainers/testcontainers-go v0.42.0
    - github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
  patterns:
    - One PostgreSQL container per package with per-test schema lifecycle
    - sync.Once-based container lifecycle (SetupPackageContainer)
    - Module-root-relative path resolution via findProjectRoot

key-files:
  created:
    - internal/adapters/secondary/postgres/test_setup.go
  modified:
    - go.mod
    - go.sum
    - internal/adapters/secondary/postgres/exported_test_helpers.go
    - cmd/server/main_test.go

key-decisions:
  - "testcontainers-go v0.42.0 selected (latest stable at time of implementation)"
  - "PostgreSQL 16-alpine image for container (smallest Postgres 16 image)"
  - "Resolve migration paths relative to Go module root (by walking up from CWD for go.mod) instead of using fragile relative glob paths"
  - "TestPool delegates entirely to SetupPackageContainer — no more DATABASE_URL env var"

patterns-established:
  - "Test infrastructure: call postgres.SetupPackageContainer(t) in TestMain or first test"
  - "Migration path resolution: findProjectRoot walks up from CWD to locate go.mod"
  - "Cleanup: t.Cleanup() closes pool and terminates container"
  - "Docker skip: t.Skip('Docker not available...') if container fails to start"

requirements-completed: [TEST-02]

# Metrics
duration: 4 min
completed: 2026-06-08
---

# Phase 00 Testing Foundation — Plan 02 Summary

**testcontainers-go infrastructure — fully self-contained PostgreSQL container per package, replacing DATABASE_URL-dependent TestPool**

## Performance

- **Duration:** 4 min
- **Started:** 2026-06-08T22:12:40Z
- **Completed:** 2026-06-08T22:16:06Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- testcontainers-go v0.42.0 and modules/postgres v0.42.0 added as Go module dependencies
- `test_setup.go` created with `SetupPackageContainer` — one PostgreSQL 16-alpine container per package using `sync.Once`, auto-cleanup via `t.Cleanup()`, Docker-availability skip via `t.Skip()`
- `TestPool` in `exported_test_helpers.go` now delegates to `SetupPackageContainer` — no more `DATABASE_URL` env var dependency
- `SetupTestSchema` enhanced with module-root-relative path resolution — works from any package's test directory, not just `internal/adapters/secondary/postgres/`
- Smoke test (`TestSmoke`) passes fully with container-backed PostgreSQL — health (200), register (201), login (200 + auth_token), authenticated units (200 + data)
- Test completes in ~4–5 seconds with container already cached locally

## Task Commits

Each task was committed atomically:

1. **Task 1: Add testcontainers-go dependency and create test_setup.go** - `8d6becb` (feat)
2. **Task 2: Replace TestPool with container-backed helper** - `cbbe024` (feat)
3. **Task 3: Update smoke test and fix migration path** - `0c18f6d` (feat)

**Plan metadata:** Will be committed with final state update.

## Files Created/Modified

- `internal/adapters/secondary/postgres/test_setup.go` — New file: `SetupPackageContainer`, `PackageTestPool`, `sync.Once` container lifecycle
- `go.mod` / `go.sum` — Added testcontainers-go v0.42.0 and modules/postgres v0.42.0
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — Modified: `TestPool` now delegates to `SetupPackageContainer`; `SetupTestSchema` uses `findProjectRoot` for CWD-independent path resolution; added `findProjectRoot`, `migrationGlob` helpers
- `cmd/server/main_test.go` — Modified: updated doc comments for testcontainers behavior

## Decisions Made

- **testcontainers-go v0.42.0** selected (latest stable).
- **PostgreSQL 16-alpine** image used (balanced size vs compatibility).
- **Module-root-relative migration paths** via `findProjectRoot` walking up from CWD for `go.mod` — avoids fragile relative glob patterns that break when tests run from different package directories.
- **`t.Skip()` on Docker unavailability** rather than `t.Fatalf()` — allows test suites to run on environments without Docker (CI machines, etc.).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration path resolution broken when running from non-postgres packages**
- **Found during:** Task 3 (smoke test execution)
- **Issue:** `SetupTestSchema` used `filepath.Glob("../../../../migrations/*.up.sql")` which only resolved correctly when CWD was `internal/adapters/secondary/postgres/`. From `cmd/server/` (where the smoke test runs), the path went too many levels up. This was masked before because the smoke test connected to a pre-configured external PostgreSQL via DATABASE_URL, not a fresh container needing migrations.
- **Fix:** Added `findProjectRoot(t)` helper that walks up from CWD to find `go.mod`, and `migrationGlob(t)` that constructs an absolute path to `migrations/*.up.sql` relative to the module root. Schema resolution now works from any package directory.
- **Files modified:** `internal/adapters/secondary/postgres/exported_test_helpers.go`
- **Verification:** Smoke test passes (migrations apply correctly from cmd/server/), repo tests still work (migration path is CWD-independent).
- **Committed in:** `0c18f6d` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking issue)
**Impact on plan:** Fix was necessary for the smoke test to work with testcontainers. No scope creep — the migration path resolution was a pre-existing bug revealed by the testcontainers migration.

## Issues Encountered

- **Migration path resolution**: `SetupTestSchema` used a relative glob path that only worked from the postgres package directory. Fixed with `findProjectRoot`/`migrationGlob` helpers.
- **Smoke test timing**: First run took ~19s due to container creation and image pull. Second run took ~4.5s (image cached). The plan's 120s timeout is generous enough.

## User Setup Required

None — no external service configuration required. Docker must be running (Orbstack or Docker Desktop).

## Next Phase Readiness

- Testcontainers infrastructure is ready for all subsequent phases
- Plan 01 (auth bug fixes) can now use `postgres.SetupPackageContainer` for testcontainer-backed integration tests
- Plans 03 (service tests) and 04 (handler tests) can rewrite against real PostgreSQL
- Smoke test provides a regression baseline for the entire server wiring

## Self-Check: PASSED

- ✅ All 3 task commits found in git log
- ✅ `internal/adapters/secondary/postgres/test_setup.go` exists on disk
- ✅ `go.mod` modified with testcontainers-go dependency
- ✅ `internal/adapters/secondary/postgres/exported_test_helpers.go` modified
- ✅ `cmd/server/main_test.go` modified
- ✅ `go build ./internal/adapters/secondary/postgres/...` compiles successfully
- ✅ `go test -v -run TestSmoke ./cmd/server/...` passes (2 runs confirmed)
- ✅ No DATABASE_URL references in new code path

---
*Phase: 00-testing-foundation*
*Completed: 2026-06-08*
