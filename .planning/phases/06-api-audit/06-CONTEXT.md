# Phase 6: API Audit - Context

**Gathered:** 2026-05-19
**Status:** Ready for planning

<domain>
## Phase Boundary

System-wide API audit — exercise every REST endpoint against a live server pre-loaded with demo seed data, verify HTTP correctness, response shapes, auth flows, and CORS. Backend-only (no frontend rendering checks). Integration-gap focused (complements Phase 0's isolated httptest.Server tests by catching route wiring, middleware chain, schema compatibility, and seed data issues).

3 waves: (1) core domain audit, (2) auxiliary domain audit, (3) batch-fix discovered bugs.

</domain>

<decisions>
## Implementation Decisions

### Test Tooling
- **D-01:** **Go test suite** — testify/require assertions, not shell scripts. `//go:build audit` tag for separate execution.
- **D-02:** **Live server target** — full bootstrap (Docker Compose → SurrealDB → cmd/schema → server). Not httptest.Server.
- **D-03:** **Existing seed data** (`003_seed_demo.surql`) as test data foundation.
- **D-04:** **One file per domain** — `audit_auth_test.go`, `audit_time_entry_test.go`, etc.
- **D-05:** **New `internal/audit/` package** — separate from Phase 0 handler tests.
- **D-06:** **Seed user login** for auth — login as demo users, reuse cookies. Not context injection.
- **D-07:** **`make audit` target** manages full lifecycle: prereq checks → Docker Compose → schema+seed → server start → go test → teardown.
- **D-08:** **Dynamic port**, dedicated SurrealDB namespace (`audit_<ts>`), full teardown.
- **D-09:** **Ad-hoc runs** — not in CI pipeline.
- **D-10:** **Test output only** (`go test -v`). No structured report file.
- **D-11:** **Stateful sequence tests** for workflow chains (draft→submit→approve→reject).
- **D-12:** **Tagged cleanup** — created test records use `[audit_<ts>]` prefix for easy identification and deletion.
- **D-13:** **Read-only policy** — tests that write revert their changes.

### Verification Scope
- **D-14:** **Integration-gap focused** — not re-testing Phase 0's business logic tests. Focuses on route wiring, middleware, CORS, cookie flow, seed compatibility, server startup.
- **D-15:** **Full CRUD per endpoint** — List (with params), Get, Create (valid + invalid), Update, Delete + GET-after-DELETE verification.
- **D-16:** **JSON response shape verification** — field names, types match frontend expectations.
- **D-17:** **Error envelope check** — all 4xx/5xx responses use `{ error: ... }` format.
- **D-18:** **CORS** — verify OPTIONS preflight and Access-Control headers.
- **D-19:** **Auth context** — both authenticated (per-domain) and unauthenticated (401) tested.
- **D-20:** **List endpoints** tested with key query params (`?status=`, `?org_id=`) and verified against exact seed counts (6 users, 8 units, 3 contracts, 6 projects, 12 time entries, 6 expenses).
- **D-21:** **2-3 validation error cases per endpoint** (malformed JSON, missing fields, wrong types).
- **D-22:** **Performance/smoke tests excluded** — functional correctness only.
- **D-23:** **Test export endpoints** (`/exports/timesheets`, `/exports/expenses`, `/exports/combined`).
- **D-24:** **Test contract/project adoption endpoints.**
- **D-25:** **Partial password reset test** (request + verify code, skip completion).
- **D-26:** **Full invitation flow** (create → validate → accept).
- **D-27:** **Full time-entry approval chain** (employee create→submit, manager approve, finance reject).
- **D-28:** **Quick 429 rate limit check.**

### Auth & Authz
- **D-29:** **Full auth surface** — health, bootstrap-check, register, login, logout, refresh, profile, memberships, switch-org, bootstrap.
- **D-30:** **All 6 seed users login verified** — 2 managers, 1 finance, 3 employees.
- **D-31:** **Seed role assignments verified** (GET /organizations/members or /organizations/{id}).
- **D-32:** **Full cookie flow tested** — login sets `auth_token` + `refresh_token`, cookies sent on subsequent requests, refresh flow on 401, logout clears cookies.
- **D-33:** **Missing RequireRole middleware logged as bugs**, fixed in Wave 3 (batch-fix).
- **D-34:** **Cross-role testing sampled per domain** — focused on write operations (approve/reject for TEs, delete for contracts, manage members for units/WGs). Login as each relevant role.
- **D-35:** **Per-domain 401 check** for unauthenticated access.
- **D-36:** **Hard-coded demo credentials** in test file.

### Bug Handling
- **D-37:** **Log + batch-fix** within phase as Wave 3.
- **D-38:** **Fresh `06-BUGS.md`** in phase directory. Structured entries: endpoint, method, expected behavior, actual behavior, severity.
- **D-39:** **Severity classification** — Critical (data loss/blocker), Major (feature broken), Minor (cosmetic/edge case).
- **D-40:** **Audit first → fix after.** Not fix-as-you-go.
- **D-41:** **Test-first fixes** (red/green). Full re-audit after Wave 3.
- **D-42:** **Phase-scoped bugs only** — not consolidating Phase 0 BUGS.md.
- **D-43:** **Same agent** does audit and fix.

### Known-Broken Endpoints
- **D-44:** `/units/{id}/members` and `/organizations/members` (known 500 from Phase 1): tests document current state (expect 500), expectations updated to 200 after fix in Wave 3.

### Frontend
- **D-45:** **Backend-only audit.** No frontend rendering checks.

### Wave Decomposition
- **Wave 1 (core):** auth, units, contracts, projects, customers, time-entries, expenses
- **Wave 2 (auxiliary):** working-groups, invitations, password-reset, exports, org management members
- **Wave 3 (fix):** batch-fix all bugs from 06-BUGS.md, full re-audit

### Agent's Discretion
- Exact test case selection per domain
- Test file naming within `internal/audit/`
- Makefile recipe details
- Docker Compose service configuration for audit
- Seed data mismatch repair approach
- Tagged cleanup implementation details

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Server & Route Wiring
- `cmd/server/main.go` — All route registrations, middleware wiring, service initialization (source of truth for audit endpoint list)
- `internal/middleware/middleware.go` — Auth middleware, RequireRole (currently unused), Logging, APIVersion
- `internal/middleware/ratelimit.go` — Rate limiter configuration
- `pkg/api/response.go` — Response envelope format

### Seed Data
- `schema/003_seed_demo.surql` — MVP demo seed data (all entities, credentials, IDs)
- `cmd/schema/main.go` — Schema/seed loader

### API Handlers (all 11 domains)
- `internal/adapters/primary/http/auth.go` — Auth endpoints
- `internal/adapters/primary/http/unit.go` — Unit endpoints (including known-broken /units/{id}/members)
- `internal/adapters/primary/http/working_group.go` — Working group endpoints
- `internal/adapters/primary/http/customer.go` — Customer endpoints
- `internal/adapters/primary/http/organization.go` — Organization endpoints (including known-broken /organizations/members)
- `internal/adapters/primary/http/project.go` — Project endpoints
- `internal/adapters/primary/http/contract.go` — Contract endpoints
- `internal/adapters/primary/http/time_entry.go` — Time entry endpoints
- `internal/adapters/primary/http/export.go` — Export endpoints
- `internal/adapters/primary/http/invitation.go` — Invitation endpoints
- `internal/adapters/primary/http/password_reset.go` — Password reset endpoints

### Frontend Types (for response shape verification)
- `web/src/types/api.ts` — API type definitions (expected response shapes)

### Prior Phase Context
- `.planning/phases/00-testing-foundation/00-CONTEXT.md` — Testing approach, patterns, infrastructure
- `.planning/phases/05-mvp-consolidation/05-CONTEXT.md` — Seed data decisions, deferred items
- `.planning/codebase/CONCERNS.md` — Known issues (missing RequireRole, error leakage, etc.)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `schema/003_seed_demo.surql` — Demo seed with 6 users, 8 units, 3 contracts, 6 projects, 12 time entries, 6 expenses
- `internal/adapters/secondary/surrealdb/helpers.go:GetTestDBWithNamespace` — Test DB pattern (can reference for namespace strategy)
- `Makefile` — Existing targets (test, build, docker-build) — new `audit` target added here
- `docker-compose.yml` — SurrealDB service definition (reuse for audit bootstrap)
- `web/src/types/api.ts` — Frontend API types for response shape verification

### Established Patterns
- **Handler test pattern:** Integration-test style with real SurrealDB, httptest.Server, testify assertions
- **Phase 0 test structure:** One file per domain, table-driven tests for service layer
- **BUGS.md pattern:** Phase 0 established structured bug tracking with severity
- **Route pattern:** Go 1.22+ `mux.HandleFunc("METHOD /path", handler)` with `middleware.Auth()` wrapper

### Integration Points
- Audit package (`internal/audit/`) runs as a standalone suite via `go test --tags=audit`
- Tests authenticate as seed users by calling `POST /auth/login` with demo credentials
- Stateful tests chain endpoints sequentially (login → create → submit → approve)
- After tests complete, `make audit` tears down Docker Compose and cleans up
- Wave 3 fixes target the handler/service/repository code directly, then re-run audit to confirm

</code_context>

<specifics>
## Specific Ideas

- Full bootstrap lifecycle in `make audit`: check Docker → docker compose up surrealdb → go run ./cmd/schema → go run ./cmd/server (background, dynamic port) → go test --tags=audit ./internal/audit/... → docker compose down
- Seed credentials hard-coded: alex.rivera@tcg.com / demo123, sarah.chen@tcg.com / demo123, etc.
- Known 500 endpoints tested as "expected failure" initially, switched to "expected success" after fix wave
- 2-wave audit (core then auxiliary) with dedicated bug-fix wave as Wave 3

</specifics>

<deferred>
## Deferred Ideas

- **Frontend smoke test** — verifying pages render with seed data. Deferred: backend-only audit.
- **Phase 0 BUGS.md consolidation** — Phase 0 bugs tracked separately.
- **Go CLI seed command** (`cmd/seed`) — deferred from Phase 5; will build once audit confirms all APIs work.
- **CI integration** — audit remains ad-hoc for now.

</deferred>

---

*Phase: 6-api-audit*
*Context gathered: 2026-05-19*
