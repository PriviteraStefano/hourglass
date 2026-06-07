---
phase: pg-3-wiring
plan: 03
subsystem: testing
tags: [smoke-test, integration-test, postgres, go-testing]
requires:
  - phase: pg-2-adapters
    provides: postgres repository implementations
  - phase: pg-3-wiring
    provides: full server wiring with Postgres repos (pg-3-01)
provides:
  - Automated smoke test for Postgres-wired server startup
  - Health endpoint verification (GET /health)
  - Auth flow verification (register, login, cookie-based access)
  - Authenticated data access verification (GET /units)
affects: []
tech-stack:
  added: []
  patterns:
    - Integration test using httptest.NewServer + postgres.TestPool
    - Full server wiring duplication for test isolation
    - DATABASE_URL-gated test skip pattern
key-files:
  created:
    - cmd/server/main_test.go
  modified: []
key-decisions:
  - "Smoke test duplicates main.go wiring inline rather than calling main() — cleaner test isolation with no package-level globals"
  - "Uses httptest.NewServer for OS-level port-free test server"
  - "Uses cookiejar to automatically capture auth_token cookies from login response"
requirements-completed: [D-21, D-22]
duration: 5min
completed: 2026-06-07
---

# Phase Pg-3 Plan 03: Verification — Smoke Test

**Automated smoke test for Postgres-wired server with health check, auth flow, and authenticated data access verification**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-07T08:45:00Z
- **Completed:** 2026-06-07T08:50:00Z
- **Tasks:** 1 of 2 completed (Task 2 deferred — manual D-15 verification)
- **Files modified:** 1

## Accomplishments

- Created `cmd/server/main_test.go` with `TestSmoke` integration test
- Smoke test verifies `GET /health` returns 200 with `{"status":"ok"}`
- Smoke test registers a new user via `POST /auth/register` → 201
- Smoke test logs in via `POST /auth/login` → 200 + `auth_token` cookie
- Smoke test hits authenticated `GET /units` → 200 with `data` field in response
- Test auto-skips via `postgres.TestPool(t)` when `DATABASE_URL` is not set (D-22)
- Mirrors full main.go wiring pattern — all 15+ repos, services, handlers, and 60+ routes
- `go vet ./cmd/server/...` and `go build ./cmd/server/...` pass clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Create automated smoke test at cmd/server/main_test.go** - `86bcef3` (test)

## Files Created/Modified

- `cmd/server/main_test.go` — Automated smoke test in package `main` with `TestSmoke` function (337 lines)

## Decisions Made

- **Wiring duplication over main() reuse:** The smoke test duplicates the full server wiring inline (same constructor calls as main.go) rather than calling `main()` or using package-level state. This provides cleaner test isolation with no shared globals.
- **httptest.NewServer:** Uses the standard test HTTP server for OS-level port-free isolation.
- **Cookie jar for auth:** Uses `net/http/cookiejar` to automatically capture `auth_token` and `refresh_token` cookies from login responses, enabling seamless authenticated requests without manual cookie extraction.

## Pending Tasks

**Task 2: Manual end-to-end verification (D-15)** — Deferred to orchestrator. This task requires manual execution of the D-15 verification flow (docker → migrate → server → login → CRUD). See the plan for detailed steps.

## Deviations from Plan

None — plan executed exactly as written for Task 1.

## Issues Encountered

None

## Next Phase Readiness

- Smoke test file created and compiles
- Manual D-15 verification flow remains to be executed by the orchestrator
- Once D-15 passes, the PostgreSQL migration is fully verified

---

*Phase: pg-3-wiring*
*Completed: 2026-06-07*
