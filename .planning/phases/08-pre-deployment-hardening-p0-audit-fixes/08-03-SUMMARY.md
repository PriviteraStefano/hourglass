---
phase: 08-pre-deployment-hardening-p0-audit-fixes
plan: 08-03
subsystem: testing
tags: [go, auth, refresh-tokens, reuse-detection, testcontainers, postgres, playwright, e2e, rate-limiting]

# Dependency graph
requires:
  - phase: 08-01
    provides: Refresh-token reuse detection (family_id/rotated_at tombstone model, atomic Rotate, ErrTokenReuse) and handler-boundary string length caps — the layers these regression suites prove dead
provides:
  - Deterministic reuse-detection regression coverage: rotate happy path, family revocation scoping, mid-rotate rollback atomicity, and concurrent-race semantics (exactly one winner, loser = replay path) at unit, repo, service and handler levels against real PostgreSQL
  - S3 length-cap boundary table (every limit class at N-1/N/N+1), rune-count semantics, 7 endpoint-level over-limit cases, and no-false-positive proofs (pass-through + real-PG boundary-length registration)
  - E2E auth spec proving the cookie-rotation contract in the browser: 401-driven silent refresh keeps the session and rotates the refresh_token cookie; replay of the pre-rotation cookie → 401
  - ANONYMOUS_RATE_LIMIT deployment knob for the outer route rate limiter (default 20/min unchanged), unblocking full e2e suite runs
affects: [08-04, verifier UAT, 09-activity-ontology (rewrites the same auth surface), deployment runbooks (rate-limit env knobs)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Deterministic concurrency testing without wall-clock timing: start-barrier goroutines + FOR UPDATE row-lock serialization produce a fixed outcome set (exactly one success + one ErrTokenReuse)
    - E2E rate-limit budgeting: register+login once via API in beforeAll, inject session cookies per test (customers-suite pattern) so anonymous requests stay under backend limits
    - expect.poll for post-navigation side effects (the silent refresh lands after the router's load event, not before)

key-files:
  created: []
  modified:
    - internal/core/services/auth/auth_test.go
    - internal/core/services/auth/auth_integration_test.go
    - internal/adapters/secondary/postgres/refresh_token_rotate_test.go
    - internal/adapters/primary/http/auth_test.go
    - internal/adapters/primary/http/validate_test.go
    - internal/core/services/testdata/mocks.go
    - web/e2e/auth.spec.ts
    - cmd/server/main.go

key-decisions:
  - "Race-loser semantics kept as locked in 08-01 (strict reuse model): the second simultaneous refresh of the same token is indistinguishable from an attacker replay and revokes the family; the race tests assert exactly-one-success + ErrTokenReuse loser and document that distinguishing a legitimate multi-tab loser from an attacker (T9) is out of scope"
  - "ANONYMOUS_RATE_LIMIT env knob for the outer route limiter (default 20/min): mirrors the existing RATE_LIMIT knob; prod behavior unchanged unless explicitly set"
  - "Repo-level rotate tests placed in refresh_token_rotate_test.go alongside the existing Rotate suites (plan listed refresh_token_repo_test.go — same package, coherent grouping)"
  - "E2E suite registers+logs in once via API in beforeAll and injects session cookies (customers-suite pattern) — the app's auth hydration on anonymous page loads burns ~2 requests each, which cannot fit the hardcoded anonymous budgets"

patterns-established:
  - "Regression-test shape for the reuse-detection layer: unit (mocked repo, incl. injectable mid-rotate failure) -> repo (real PG rollback + race) -> service (real PG rotation chain + race) -> handler (cookie jar rotation) -> E2E (browser-observable rotation + replay 401)"

requirements-completed: [P0-5, S3]

# Metrics
duration: 29min
completed: 2026-07-31
---

# Phase 8 Plan 3: Backend Regression Tests Summary

**Reuse-detection regression suites (unit + real-PG repo/service/handler, incl. a deterministic concurrent-race test) proving the P0-5 layer stays dead, an S3 length-cap boundary table over every limit class, and an E2E auth spec proving the silent-refresh cookie rotation is observable in the browser with replay → 401**

## Performance

- **Duration:** 29 min
- **Started:** 2026-07-31T12:37:00Z
- **Completed:** 2026-07-31T13:05:55Z
- **Tasks:** 3
- **Files modified:** 8 (0 created, 8 modified)

## Accomplishments

- **Task 1 — Reuse-detection regression (P0-5):** service unit tests for the rotate happy path (old row rotated, successor inherits `family_id`, new pair returned), replay revoking the whole family including the replayed token itself (with a different-family token untouched — right-family scoping), and an injectable mid-rotate failure proving the service surfaces the error with **no partial state** (old token neither rotated nor revoked, no successor). Real-PG repo tests prove the rotation transaction **rolls back atomically** when the successor insert fails (duplicate `token_hash` UNIQUE violation) and that a concurrent race of the same token yields **exactly one success + one `ErrTokenReuse`** (deterministic via `SELECT … FOR UPDATE` serialization + start-barrier goroutines; semantics documented inline). Service integration proves the login → refresh → refresh chain issues three distinct tokens while staying in one family; the concurrent-refresh race asserts the winner's successor is revoked by the loser (strict-reuse semantics, T9 out of scope). Handler integration extends the rotation test to refresh → refresh with cookie-jar rotation assertions.
- **Task 2 — S3 length caps:** helper-level boundary table covering **every limit class** (email 320, name 200, description 4000, address 500, VAT 50, phone 50, password 128, short 500) at N-1/N (accepted, no response written) and N+1 (400 with field-level message), plus a rune-count semantics test (200 multi-byte chars pass a 200-rune cap, 201 fail). Endpoint over-limit cases grew from 5 to 7 domains (added expense description 5k, register firstname 10k). No-false-positive proofs: expense pass-through reaches its own required-field validation, and a real-PG registration with a boundary-length (200-rune) name succeeds end-to-end.
- **Task 3 — E2E auth rotation:** new spec logs in via API, corrupts the access token, navigates to a protected route — the 401-driven silent refresh keeps the session and **rotates the `refresh_token` cookie** (asserted via `expect.poll` because the refresh lands after the router's load event), then replays the pre-rotation cookie → 401 (reuse detection observable end-to-end). Suite restructured to the customers pattern (API register+login once in `beforeAll`, inject cookies per test) to fit backend anonymous rate limits; stale `201` expectation (register returns 200) and stale `/` URL assertions (landing redirects to `/time-entries`) fixed.

## Task Commits

Each task was committed atomically:

1. **Task 1: Refresh-token reuse-detection regression suite** - `27eeae5` (test) — 5 files, +377
2. **Task 2: S3 length-cap boundary table + endpoint cases** - `98e8a80` (test) — 1 file, +122
3. **Task 3: E2E auth silent-refresh rotation spec** - `5647506` (test) — 2 files, +136/−43

**Plan metadata:** `pending` (docs: complete plan)

## Files Created/Modified

- `internal/core/services/auth/auth_test.go` - unit tests: rotate happy path (family inheritance), mid-rotate failure no-partial-state, strengthened family-revocation scoping (replayed token tombstoned, other-family token untouched)
- `internal/core/services/auth/auth_integration_test.go` - real-PG: login→refresh→refresh chain (3 distinct tokens, one family), concurrent-refresh race (exactly one success, loser ErrTokenReuse, family consistently dead)
- `internal/adapters/secondary/postgres/refresh_token_rotate_test.go` - real-PG: mid-failure rollback (duplicate-hash insert aborts the tx, old token untouched), concurrent race deterministic outcome set
- `internal/adapters/primary/http/auth_test.go` - handler: refresh→refresh rotation with cookie-jar assertions after each rotation
- `internal/adapters/primary/http/validate_test.go` - boundary table (8 limit classes × N-1/N/N+1), rune-count test, expense-description + register-firstname over-limit cases, expense pass-through, real-PG boundary-length register success
- `internal/core/services/testdata/mocks.go` - `MockRefreshTokenRepo.RotateErr` injectable failure (no mutation on error)
- `web/e2e/auth.spec.ts` - silent-refresh rotation E2E + suite restructure to API-login/cookie-injection pattern; fixed stale 201 and `/` URL assertions
- `cmd/server/main.go` - `ANONYMOUS_RATE_LIMIT` env knob for the outer route limiter (default 20/min unchanged)

## Decisions Made

- **Race-loser semantics unchanged (documented in tests):** the plan's "loser gets 401 without corrupting the family" was resolved against the locked 08-01 strict-reuse decision — the concurrent loser is deliberately **not** distinguished from an attacker replay (client-fingerprint disambiguation is audit T9, out of scope), so the family is revoked after a race. Both race tests assert exactly-one-success + `ErrTokenReuse` loser and document this; the post-race family is consistently dead, never partially rotated.
- **`ANONYMOUS_RATE_LIMIT` knob:** the outer route limiter was hardcoded at 20 anonymous req/min; the app's auth hydration on anonymous page loads (each unprotected navigation burns `/auth/me` 401 + failed refresh = 2 anonymous calls) makes a full auth e2e suite structurally exceed it. Added an env knob (default 20, mirroring the existing `RATE_LIMIT` knob) so e2e runs can raise it without weakening prod defaults.
- **E2E rate-limit budgeting:** register + login once via API in `beforeAll` and inject session cookies per test (customers-suite pattern) — keeps anonymous calls ~12-15 per run instead of ~27.
- **Cookie assertion via `expect.poll`:** `page.goto` resolves at the `load` event, before the router guard's refresh lands; reading cookies immediately yields the pre-rotation value. Polling for the change is deterministic, no arbitrary sleep.
- **Repo test placement:** new Rotate tests added to `refresh_token_rotate_test.go` (where the existing Rotate tests live) rather than the plan-listed `refresh_token_repo_test.go` — same package, coherent grouping.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Outer route rate limiter hardcoded at 20/min made the full auth e2e spec impossible**
- **Found during:** Task 3 (E2E auth spec)
- **Issue:** `cmd/server/main.go` wrapped all routes in `NewRateLimiter(20, 100)` with no env knob. The app hydrates auth on every page load, and an anonymous navigation burns 2 requests (`/auth/me` 401 + failed `/auth/refresh`). A full auth spec (register UI, register validation, login ×2, logout, protected route, rotation) makes ~22-27 anonymous calls per run — the rotation test's register reliably hit 429.
- **Fix:** Added `ANONYMOUS_RATE_LIMIT` env parsing (default 20, identical pattern to the existing `RATE_LIMIT` auth-endpoint knob). E2E runs start the backend with `RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500`. Prod defaults unchanged unless explicitly configured.
- **Files modified:** cmd/server/main.go
- **Verification:** `npx playwright test auth` → 7/7 passing twice consecutively
- **Committed in:** 5647506 (Task 3 commit)

**2. [Rule 1 - Bug] Stale e2e assertions contradicted current backend/app behavior**
- **Found during:** Task 3 (E2E auth spec)
- **Issue:** `auth.spec.ts` asserted `register` returns **201** (the handler returns 200 — the `Register_WithNewOrg_Returns201WithUserData` Go test even asserts 200), and asserted the URL is exactly `/` after register/login (the landing route redirects to `/time-entries`). Both failed against the current app.
- **Fix:** `toBe(201)` → `toBe(200)`; `toHaveURL('/')` → `toHaveURL(/\/time-entries/)` (two places).
- **Files modified:** web/e2e/auth.spec.ts
- **Verification:** full auth spec green
- **Committed in:** 5647506 (Task 3 commit)

**3. [Rule 1 - Bug] Cookie-rotation race in the E2E assertion**
- **Found during:** Task 3 (E2E auth spec)
- **Issue:** `page.goto` resolves at the `load` event, before the route guard's silent refresh completes — reading `context.cookies()` immediately after `toHaveURL` returned the pre-rotation refresh token (reproduced standalone).
- **Fix:** Assert the rotation with `expect.poll(...).not.toBe(refreshBefore)` (timeout 15s) — deterministic, no wall-clock sleep.
- **Files modified:** web/e2e/auth.spec.ts
- **Verification:** 7/7 passing twice
- **Committed in:** 5647506 (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 3 blocking)
**Impact on plan:** All fixes were required for the plan's acceptance criteria ("spec passes locally and in CI"). The `ANONYMOUS_RATE_LIMIT` knob is the only production-code touch — minimal, default-preserving, and consistent with the existing `RATE_LIMIT` knob. No scope creep beyond unblocking the e2e gate.

## Issues Encountered

- **Self-inflicted stash mishap (recovered):** while capturing a coverage baseline I ran a `git stash -q -- internal/core/services/auth/` inside the verification command, which stashed the just-written Task 1 service tests (and, unexpectedly, pre-staged phase-09 planning files) and reset the auth files to HEAD. Recovered by extracting the two auth test files from the stash commit via `git checkout stash@{0} -- <paths>`, re-verified the tests, and dropped the entry — the pre-existing older stash was never touched. All auth service tests re-ran green after recovery.
- **Staged phase-09 planning files swept into commits twice:** the index contained pre-staged `09-*.md` files (from planning work before this session). Two `git commit` calls swept them in; both commits were rewritten via `git reset --soft HEAD~1` + selective unstage to restore task-scoped commits, and the planning files were re-staged to their original state. Final commits contain only task files (verified with `git show --stat`).
- **Backend rate-limit discovery:** empirically mapped the two in-memory limiters (outer 20/min on all routes; auth 5/min on register/login, both per-IP 60s windows). The e2e suite cannot run under defaults — local e2e runs require `RATE_LIMIT` + `ANONYMOUS_RATE_LIMIT` raised (documented in the spec header).
- **Duplicate-cookie probe:** initially suspected the corrupted `auth_token` injection losing to the valid token; the backend log proved the garbage token won and the refresh rotated server-side — the failure was purely the cookie-read race fixed via `expect.poll`.

## User Setup Required

None - no external service configuration required. Local e2e runs of the full auth spec need the backend started with raised rate-limit knobs:

```
RATE_LIMIT=500 ANONYMOUS_RATE_LIMIT=500 go run ./cmd/server
```

(The spec header documents this; production deployments keep defaults.)

## Next Phase Readiness

- P0-5 and S3 regression coverage complete at every layer (unit → repo → service → handler → E2E); full `go test ./...` green (19 packages, incl. testcontainers integration suites); `npx playwright test auth` green (7/7, stable across consecutive runs)
- Ready for plan 08-04 (remaining phase-08 work) and the phase verifier UAT; phase 09 (activity ontology) will rewrite the same auth surface with regression suites already in place
- Deployment note: the `ANONYMOUS_RATE_LIMIT` knob joins `RATE_LIMIT` in the ops env surface (both default to current behavior)

---

*Phase: 08-pre-deployment-hardening-p0-audit-fixes*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 8 modified files verified on disk: ✓
- Task commits `27eeae5` (Task 1), `98e8a80` (Task 2), `5647506` (Task 3) present in git log: ✓
- Full `go test -count=1 -timeout 1800s ./...` green (19 packages, exit 0): ✓
- `npx playwright test auth` green: 7/7, stable across two consecutive runs: ✓
