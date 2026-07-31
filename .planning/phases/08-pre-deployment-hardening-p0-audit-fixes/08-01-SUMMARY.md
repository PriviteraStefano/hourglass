---
phase: 08-pre-deployment-hardening-p0-audit-fixes
plan: 08-01
subsystem: backend-security
tags: [go, auth, refresh-tokens, pgx, postgres, migrations, input-validation]

# Dependency graph
requires:
  - phase: 00-01
    provides: Fixed auth service behavior (refresh rotation) — the rotation this plan layers reuse detection on
provides:
  - Refresh-token reuse detection: family_id/rotated_at tombstone model, atomic rotation, replay → family revocation → ErrTokenReuse → 401 + cookie clear
  - Handler-boundary string length caps (audit S3) across auth, password_reset, customer, contract, project, time_entry, expense, unit, working_group
affects: [08-03 (password-reset hardening may reuse the rotation tx pattern), 09-activity-ontology (rewrites these same handlers), verifier UAT]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Atomic repo-level rotation via pgx BeginTx + SELECT ... FOR UPDATE (single tx: mark rotated + insert successor)
    - Sentinel error mapping chain: ports.ErrTokenReuse → auth.ErrTokenReuse → HTTP 401 + ClearAuthCookies
    - Rune-count length caps at the adapter boundary (validate.go), domain value objects untouched

key-files:
  created:
    - migrations/010_refresh_token_reuse_detection.up.sql
    - migrations/010_refresh_token_reuse_detection.down.sql
    - internal/core/services/auth/errors.go
    - internal/adapters/primary/http/validate.go
    - internal/adapters/primary/http/validate_test.go
    - internal/adapters/secondary/postgres/refresh_token_rotate_test.go
  modified:
    - internal/core/ports/refresh_token_repo.go
    - internal/adapters/secondary/postgres/refresh_token_repo.go
    - internal/core/services/auth/auth.go
    - internal/core/services/auth/auth_test.go
    - internal/core/services/auth/auth_integration_test.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/primary/http/auth.go
    - internal/adapters/primary/http/auth_integration_test.go
    - internal/adapters/primary/http/{contract,customer,expense,password_reset,project,time_entry,unit,working_group}.go

key-decisions:
  - "Replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5)"
  - "Rotation is a single repo-level transaction (pgx BeginTx + FOR UPDATE); no crash window between revoke and reissue"
  - "Concurrent-refresh race: the second simultaneous refresh of the same token is treated as replay and revokes the family — documented in repo/test comments; client-fingerprint disambiguation is out of scope (audit T9)"
  - "FindByHash now returns only non-rotated, non-revoked tokens (semantic shift from revoke-model to tombstone-model)"
  - "Length caps live at the handler boundary (rune counts) — domain value objects keep format validation only, per hexagonal layering"
  - "Plan listed a 'profile update' handler for S3 caps — no such endpoint exists (only GET /auth/me); register/login cover the auth surface"

patterns-established:
  - "Refresh-token rotation = repo-owned atomic Rotate(ctx, hash, newHash, expiresAt) returning (token, nil | ports.ErrTokenReuse); service maps to domain sentinel; handler maps to 401 + cookie clear"
  - "Length caps via validateStringLengths(w, lengthField(name, value, cap)...) inserted immediately after JSON decode in every write handler"

requirements-completed: [P0-5, S3]

# Metrics
duration: 40min
completed: 2026-07-31
---

# Phase 8 Plan 1: Backend Security Hardening — Refresh-Token Reuse Detection + Input Caps Summary

**Refresh-token reuse detection with family revocation (atomic single-transaction rotation, replay → 401 + both cookies cleared) and handler-boundary string length caps (audit S3) across all write handlers**

## Performance

- **Duration:** 40 min
- **Started:** 2026-07-31T10:08:00Z
- **Completed:** 2026-07-31T10:47:57Z
- **Tasks:** 2
- **Files modified:** 21 (9 created, 12 modified)

## Accomplishments

- Migration `010_refresh_token_reuse_detection` adds `family_id` (backfilled one-family-per-row, DEFAULT `gen_random_uuid()` for new rows), `rotated_at`, and a family index; verifies apply → rollback → re-apply cleanly against a fresh testcontainer
- `RefreshTokenRepository.Rotate` performs rotation in ONE pgx transaction (`FOR UPDATE` lock, mark `rotated_at`, insert successor with same `family_id`); a replayed (rotated or revoked) token tombstones the whole family and surfaces `ports.ErrTokenReuse`
- `Service.Refresh` now delegates to `Rotate` and maps reuse to the new `auth.ErrTokenReuse` sentinel (`internal/core/services/auth/errors.go`); the Refresh handler responds 401 and clears both `auth_token` and `refresh_token` cookies
- `internal/adapters/primary/http/validate.go` enforces per-field rune-count caps (email 320, name 200, description/notes 4000, address 500, VAT 50, phone 50, password 128, short strings 500) in auth, password_reset, customer, contract, project, time_entry, expense, unit, and working_group create/update/reject handlers — 400 with field-level message, domain value objects untouched
- Regression coverage: mock service tests (replay of rotated/revoked token → `ErrTokenReuse`, family revocation), service integration test against real PG (replay → `ErrTokenReuse`, successor dies), handler integration test (replay → 401 + both cookies cleared + successor 401), repo tests (Rotate happy path, replay revokes family, RevokeFamily, migration up/down), and 5 over-limit handler tests across different domains all returning 400

## Task Commits

Each task was committed atomically:

1. **Task 1: Refresh-token reuse detection with family revocation** - `e4b7932` (feat)
2. **Task 2: Request-string length caps at handler boundary** - `96da472` (feat)

**Plan metadata:** pending (docs: complete plan — created with this summary)

## Files Created/Modified

- `migrations/010_refresh_token_reuse_detection.up.sql` / `.down.sql` - family_id + rotated_at columns, backfill, family index (+ matching rollback)
- `internal/core/ports/refresh_token_repo.go` - `RefreshToken` gains `FamilyID`/`RotatedAt`/`RevokedAt`; interface gains `Rotate` + `RevokeFamily`; `ports.ErrTokenReuse` sentinel
- `internal/adapters/secondary/postgres/refresh_token_repo.go` - atomic `Rotate` (BeginTx + FOR UPDATE), `RevokeFamily`, `FindByHash` now excludes rotated+revoked, `Add` assigns fresh family
- `internal/core/services/auth/errors.go` - new `ErrTokenReuse` sentinel (per plan; the package had no errors.go, sentinels previously lived in auth.go)
- `internal/core/services/auth/auth.go` - `Refresh` rewritten to use `Rotate`; login/register/bootstrap/switch-org issue paths unchanged (each Add starts a fresh family)
- `internal/adapters/primary/http/auth.go` - Refresh maps `ErrTokenReuse` → 401 + `cookies.ClearAuthCookies`
- `internal/adapters/primary/http/validate.go` - shared length-cap helper (rune counts, 400 + field message)
- `internal/adapters/primary/http/{customer,contract,project,time_entry,expense,password_reset,unit,working_group}.go` - caps wired into all write handlers
- Test files: `refresh_token_rotate_test.go` (repo + migration), `validate_test.go` (5 over-limit + 1 pass-through), `auth_test.go` (mock reuse cases + family test), service + handler `auth_integration_test.go` (reuse/revocation end-to-end), `testdata/mocks.go` (mock Rotate/RevokeFamily with real semantics)

## Decisions Made

- **Strict reuse model:** replay of any rotated *or* revoked token revokes the whole family — per audit P0-5 ("replay of a rotated token → `ErrTokenReuse` + family revocation"). `Logout`'s `RevokeByHash` tombstones remain consistent with this: a logged-out token's replay also kills its family.
- **Concurrent-refresh race semantics (documented in tests):** with `FOR UPDATE`, the second simultaneous refresh of the same token is indistinguishable from an attacker replay and revokes the family (login again). Client-fingerprint disambiguation is audit item T9, out of scope — noted in `Rotate` comments and the integration test.
- **FindByHash semantic shift:** now returns only non-rotated AND non-revoked tokens. Required so the pre-existing `RefreshTokenRotation` assertion ("old hash no longer findable") holds under the tombstone model where rotation marks `rotated_at` instead of `revoked_at`.
- **Caps at the adapter boundary:** rune-count based (user-facing character semantics), no domain value object changes (per hexagonal layering in the plan).
- **No 'profile update' endpoint exists:** the plan's S3 list mentioned it; auth handler coverage = register + login (GET-only profile).

## Deviations from Plan

None - plan executed as written. Two plan details refined against ground truth (documented above, not scope changes): the auth service package had no `errors.go` (created it per plan), and no profile-update handler exists (register/login covered instead).

## Issues Encountered

- **Docker daemon down at start:** testcontainers integration tests could not run initially; started OrbStack, then all integration tests (service + handler + postgres packages) ran green.
- **Migration 010 down-test ordering:** the rollback test drops columns mid-suite; it re-applies 010 up before finishing so sibling tests in the package always see the full schema.

## User Setup Required

None - no external service configuration required. Requires `go run ./cmd/migrate -up -dir migrations` on any existing database to apply migration 010.

## Next Phase Readiness

- P0-5 and S3 closed: backend security hardening complete; `go build ./...` clean and full `go test ./...` suite green (testcontainers-backed integration tests included)
- Ready for plans 08-02 (frontend), 08-03, 08-04 — the auth flow they touch now has reuse detection + family revocation + cookie clearing on replay
- The migration must be applied to any live DB before deploy (`go run ./cmd/migrate -up`)

---
*Phase: 08-pre-deployment-hardening-p0-audit-fixes*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 7 key created files verified on disk: ✓
- Task commits `e4b7932` + `96da472` present in git log: ✓
- `go build ./...` clean: ✓
- Full `go test -count=1 -timeout 1800s ./...` green (incl. testcontainers integration suites): ✓
