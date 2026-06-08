# Phase 0: Testing foundation - Context

**Gathered:** 2026-06-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend test infrastructure reboot using testcontainers-go for fully self-contained PostgreSQL-backed integration tests. Fix all known auth bugs and related auth concerns. Establishes test foundation that all feature phases depend on.

**No frontend work.** All work is Go backend: test infrastructure, auth bug fixes, test rewrites.

</domain>

<decisions>
## Implementation Decisions

### Plan Ordering
- **D-01:** Testcontainers setup BEFORE auth bug fixes (reverse ROADMAP order). Infrastructure must exist before tests can verify fixes.
- **D-02:** Auth bug fixes are test-driven — write failing testcontainer-backed test that reproduces the bug, then fix, then watch test pass.
- **D-03:** Testcontainers setup and auth bug fixes remain separate plans. Clean dependency chain.
- **D-04:** All 4 known auth bugs fixed in one plan (not split per-bug).
- **D-05:** Post-auth order: service-layer test rewrite → handler integration test rewrite → E2E verification.
- **D-06:** 05-bug-buffer kept as a separate plan (not merged into auth fix plan).

### Test Architecture
- **D-07:** testcontainers-go replaces TestPool entirely. No more `DATABASE_URL` dependency for tests. Fully self-contained test suite.
- **D-08:** Hybrid container strategy: one testcontainers PostgreSQL instance per package, each test function gets its own schema via existing SetupTestSchema/TeardownTestSchema pattern.
- **D-09:** ALL tests use testcontainers: service tests, handler tests, repository tests, smoke test (`cmd/server/main_test.go`), and Playwright E2E.

### Mock Tests Fate
- **D-10:** Hybrid approach — keep existing in-memory mock tests for pure business logic (validation, error mapping). Use testcontainers for tests that exercise repository interaction.
- **D-11:** Fix broken mock implementations now (MockOrgRepo.GetMembership returning nil, MockPasswordResetRepo.FindActiveByUserID returning not found, etc.).
- **D-12:** New testcontainers-based integration tests go in separate `_integration_test.go` files (e.g., `auth_integration_test.go`). Keeps mock unit tests and integration tests clearly separated.

### Bug Loop Mechanics
- **D-13:** Minor bugs discovered during test rewrite: fix immediately inline.
- **D-14:** Major/complex bugs: document, t.Skip() the failing test with a reference to the logged bug, batch to 05-PLAN.
- **D-15:** 05-PLAN is a dedicated bug-fixing buffer that includes a human-in-the-middle review loop for major bugs.

### Auth Scope
- **D-16:** Full auth cleanup in Phase 0 — not just the 4 known bugs:
  - Fix `/auth/memberships` nil pointer
  - Fix `/auth/me` returning empty role/org_id
  - Fix `/units/{id}/members` 500
  - Fix `/organizations/members` 500
  - Implement refresh token rotation (revoke old hash, issue new)
  - Fix cookie name mismatch (access_token vs auth_token)
  - Fix password reset code leak (don't return in body, increase entropy)
  - Add stricter rate limiting for auth endpoints (login, password reset)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project overview, key decisions, constraints
- `.planning/REQUIREMENTS.md` — Requirements (TEST-01 through TEST-06)
- `.planning/ROADMAP.md` — Phase definitions and dependency graph
- `.planning/STATE.md` — Current phase state and session info

### Codebase Maps
- `.planning/codebase/TESTING.md` — Existing test patterns, structure, database setup
- `.planning/codebase/CONCERNS.md` — Known bugs, auth concerns, test coverage gaps
- `.planning/codebase/CONVENTIONS.md` — Code style, naming, import organization
- `.planning/codebase/STRUCTURE.md` — Directory layout, where to add new code
- `.planning/codebase/ARCHITECTURE.md` — Hexagonal architecture patterns

### Key Source Files
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — TestPool, SetupTestSchema, TeardownTestSchema, seed helpers
- `internal/core/services/testdata/mocks_test.go` — Mock repository implementations
- `internal/core/services/auth/auth.go` — Auth business logic (bugs live here)
- `internal/adapters/primary/http/auth.go` — Auth HTTP handler
- `internal/cookies/cookies.go` — Cookie name constants
- `internal/middleware/ratelimit.go` — Rate limiter
- `internal/core/services/password_reset/password_reset.go` — Password reset logic
- `cmd/server/main.go` — Route wiring, middleware registration
- `cmd/server/main_test.go` — Smoke test

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `TestPool` + `SetupTestSchema`/`TeardownTestSchema` in `exported_test_helpers.go` — Existing schema lifecycle pattern, testcontainers integration just replaces the pool source
- Seed functions (`seedOrg`, `seedUser`, `seedProject`, etc.) — Ready to use in new integration tests
- Table-driven test pattern in all service tests — Consistent structure to follow for new tests
- `postgres.go` (`wrapPGError`) — Error wrapping for sentinel error matching in integration tests

### Established Patterns
- Table-driven tests with testify (`assert`/`require`) — Standard for all Go tests
- Co-located test files (`auth_test.go` next to `auth.go`)
- Handler integration tests use `httptest.NewServer` + real `http.ServeMux` — Pattern to extend for testcontainers

### Integration Points
- `exported_test_helpers.go` — Single file to modify for testcontainers integration (replace `TestPool`)
- `cmd/server/main.go` — Route registration must be verified in smoke test
- `internal/middleware/middleware.go` — Auth middleware reads `auth_token` cookie — must match cookie name fix

</code_context>

<specifics>
## Specific Ideas

- All auth concerns in CONCERNS.md addressed in this phase — full auth cleanup before feature phases begin

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 0-Testing foundation*
*Context gathered: 2026-06-08*
