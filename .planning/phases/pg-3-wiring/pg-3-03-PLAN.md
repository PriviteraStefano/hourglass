---
phase: pg-3-wiring
plan: 03
type: execute
wave: 2
depends_on: [pg-3-01]
files_modified:
  - cmd/server/main_test.go
autonomous: false
requirements: [D-15, D-21, D-22]
must_haves:
  truths:
    - `cmd/server/main_test.go` exists with TestSmoke function
    - Smoke test uses `postgres.TestPool()` (skips if DATABASE_URL not set, D-22)
    - Smoke test verifies `GET /health` returns 200
    - Smoke test verifies login (POST /auth/login) with seed user returns 200 + cookies
    - Smoke test verifies authenticated `GET /units` returns 200 with data
    - Manual verification: full D-15 flow passes (docker-compose → migrate → server → login → CRUD)
  artifacts:
    - path: cmd/server/main_test.go
      provides: Automated smoke test for postgres-wired server
      min_lines: 60
      contains: ["TestSmoke", "postgres.TestPool"]
  key_links:
    - from: cmd/server/main_test.go
      to: internal/adapters/secondary/postgres/exported_test_helpers.go
      via: postgres.TestPool(), SetupTestSchema(), TeardownTestSchema()
    - from: cmd/server/main_test.go
      to: cmd/server/main.go
      via: integration test against the full server wiring
---

<objective>
Create an automated smoke test and execute the full manual verification flow to confirm the Postgres-wired server works end-to-end.

**Purpose:** The smoke test provides CI-grade automated confidence that the server starts, serves health checks, and authenticates requests. The manual verification confirms real-world function — login as each role, see data, perform CRUD.

**Output:**
- `cmd/server/main_test.go` — Smoke test (integrated, DATABASE_URL-gated)
- Manual verification completed (D-15 flow)
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-3-wiring/pg-3-RESEARCH.md (smoke test example code on lines 371-389)

# Source of truth for test infrastructure
@internal/adapters/secondary/postgres/exported_test_helpers.go (TestPool, SetupTestSchema, TeardownTestSchema)

# POST /auth/login endpoint handler
@internal/adapters/primary/http/auth.go (Login handler — expects {identifier, password}, returns 200 + cookies)

# Health endpoint
@internal/handlers/health.go (returns {"status":"ok"} on GET /health)

# Seed user credentials from seed data
@migrations/003_seed.up.sql (alex.rivera / demo123 — manager user for manual verification)

# Server entry point
@cmd/server/main.go (server startup wiring — verified compiles in Plan 1)

<interfaces>
From internal/adapters/secondary/postgres/exported_test_helpers.go:
```go
func TestPool(t *testing.T) *pgxpool.Pool       // returns pool, skips if DATABASE_URL not set
func SetupTestSchema(t *testing.T, pool *pgxpool.Pool)  // applies migrations
func TeardownTestSchema(t *testing.T, pool *pgxpool.Pool) // drops all tables
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create automated smoke test at cmd/server/main_test.go</name>

  <files>
    cmd/server/main_test.go
  </files>

  <read_first>
    @internal/adapters/secondary/postgres/exported_test_helpers.go (TestPool, SetupTestSchema, TeardownTestSchema patterns)
    @internal/handlers/health.go (returns {"status":"ok"} on GET /health)
    @internal/adapters/primary/http/auth.go (Login handler signature)
    @cmd/server/main.go (server setup code to call in test)
  </read_first>

  <action>
    Create `cmd/server/main_test.go` in package `main`.

    **Imports:**
    - `context`
    - `encoding/json`
    - `net/http`
    - `net/http/cookiejar`
    - `strings`
    - `testing`
    - `github.com/stefanoprivitera/hourglass/internal/adapters/primary/http` (for NewAuthHandler etc, or we can use the server's own mux setup)
    - `github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres`
    - `github.com/stefanoprivitera/hourglass/internal/auth`
    - `authsvc "github.com/stefanoprivitera/hourglass/internal/core/services/auth"`
    - `invitationsvc ...` (and other service packages as needed for server startup)
    - `github.com/stefanoprivitera/hourglass/internal/middleware`

    **TestSmoke function:**

    1. **Setup**: Call `postgres.TestPool(t)` to get pool. Call `postgres.SetupTestSchema(t, pool)` to apply migrations. `t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })`.

    2. **Server initialization**: Follow the same wiring pattern as main.go — set up all repos, services, handlers, mux, and middleware chain. Use the pool from step 1. Use `jwtSecret := "test-secret"` for the auth service. Use a `httptest.NewServer(mux)` to create a test server.

       Repo wiring (same constructors as new main.go after Plan 1):
       - `postgres.NewTimeEntryRepository(pool)`, `postgres.NewAuditLogRepository(pool)`, etc.
       - `auth.NewPasswordHasher()`, `auth.NewTokenService(authService)`
       - `postgres.NewUserRepository(pool)`, `postgres.NewOrganizationRepository(pool)`, `postgres.NewRefreshTokenRepository(pool)`
       - `postgres.NewInvitationRepository(pool)`, `postgres.NewPasswordResetRepository(pool)`, `postgres.NewUserFinder(pool)`
       - `postgres.NewUnitRepository(pool)`, `postgres.NewWorkingGroupRepository(pool)`, `postgres.NewCustomerRepository(pool)`
       - `postgres.NewOrganizationManagementRepository(pool)`, `postgres.NewProjectRepository(pool)`, `postgres.NewContractRepository(pool)`, `postgres.NewExportRepository(pool)`

       The full wiring inline is acceptable for a test file — matching main.go's constructor pattern.

    3. **Test `/health`**: GET `http://testserver/health`, expect 200, decode body as `map[string]interface{}`, verify `result["status"] == "ok"`.

    4. **Test authenticated endpoint with Register→Login→/units**:
       - Register: POST `/auth/register` with `{email, username, password, organization_name}`. Expect 201.
       - Login: POST `/auth/login` with `{identifier, password}`. Use the same email/password. Expect 200. Extract auth_token from response cookies.
       - Authenticated GET `/units`: Create HTTP request with cookie jar (or manually set `auth_token` cookie). Expect 200. Decode body, verify response has `data` field.

    5. **Let's keep the test focused** — it doesn't need to test every endpoint, just verify:
       - Health endpoint works (unauthenticated)
       - Auth flow works (register + login + cookie-based access to a protected endpoint)
       - Data access works (GET /units returns a valid response)

    **Skip behavior**: If `DATABASE_URL` is not set, `postgres.TestPool(t)` calls `t.Skip()`, so the test invisibly passes in CI without a database.

    The smoke test doesn't need to reuse the exact `main()` function — it creates its own server with the same wiring pattern. This gives cleaner test isolation than manipulating package-level globals.
  </action>

  <verify>
    <automated>go vet ./cmd/server/... && go build ./cmd/server/...</automated>
  </verify>

  <done>
    1. `cmd/server/main_test.go` exists with `func TestSmoke(t *testing.T)`
    2. Smoke test uses `postgres.TestPool(t)` (skips if DATABASE_URL not set per D-22)
    3. Smoke test verifies `GET /health` returns 200 with `{"status":"ok"}`
    4. Smoke test registers a user, logs in, and hits `GET /units` with auth cookie
    5. `go vet ./cmd/server/...` and `go build ./cmd/server/...` pass
    6. If DATABASE_URL is set: `go test ./cmd/server/... -run TestSmoke -v -count=1` passes
  </done>
</task>

<task type="checkpoint:human-verify">
  <name>Task 2: Manual end-to-end verification (D-15)</name>

  <files>
    (no files — verification-only task)
  </files>

  <read_first>
    @migrations/003_seed.up.sql (seed credentials: alex.rivera / demo123)
    @.planning/phases/pg-3-wiring/pg-3-RESEARCH.md (D-15 flow steps)
  </read_first>

  <action>
    Execute the D-15 manual verification flow. This confirms the server works end-to-end with PostgreSQL from a fresh bootstrap.

    **Verification steps:**

    1. Start PostgreSQL:
       ```bash
       docker compose up -d postgres
       ```
       Wait for Postgres health check to pass.

    2. Run migrations + seed:
       ```bash
       go run ./cmd/migrate -all
       ```
       This applies all schema migrations and seeds demo data.
       Expect: No errors, migration confirmation prints.

    3. Start the server:
       ```bash
       go run ./cmd/server
       ```
       Expect: Server starts on `:8080`. Logs show "PostgreSQL pool initialized" and "Server starting on port 8080".
       No SurrealDB-related messages should appear.
       The server should NOT crash or print SurrealDB warnings.

    4. Login as demo manager (alex.rivera / demo123):
       ```bash
       curl -X POST http://localhost:8080/auth/login \
         -H "Content-Type: application/json" \
         -d '{"identifier":"alex.rivera@hourglass.test","password":"demo123"}' \
         -v
       ```
       Expect: HTTP 200 with `auth_token` and `refresh_token` Set-Cookie headers.
       Response body contains `{"data":{"user":{...},"membership":{...}}}`.

    5. Check authenticated endpoint (GET /units):
       ```bash
       curl -X GET http://localhost:8080/units -v \
         -H "Cookie: auth_token=<token from login>"
       ```
       Expect: HTTP 200 with JSON array of units. If not empty, verify seed data renders.

    6. Stop the server (Ctrl+C in server terminal).

    **Acceptance criteria:**
    - Server starts without SurrealDB errors
    - Login with seed demo credentials (alex.rivera / demo123) succeeds
    - Authenticated API call returns data
    - All steps complete without errors
  </action>

  <what-built>
    The full PostgreSQL-wired server stack — all repository adapters swapped, SurrealDB removed, verification that everything works end-to-end.
  </what-built>

  <how-to-verify>
    Follow the 6-step D-15 verification flow above exactly. Key outcomes at each step:

    1. `docker compose up -d postgres` → Postgres container healthy
    2. `go run ./cmd/migrate -all` → schema + seed applied successfully
    3. `go run ./cmd/server` → server starts on :8080, no SurrealDB output
    4. `curl login` → 200 + auth_token/refresh_token cookies
    5. `curl /units` with cookie → 200 + data in response
    6. Server stops cleanly (Ctrl+C)
  </how-to-verify>

  <resume-signal>
    Type "manual verification passed" to confirm all 6 steps completed successfully. If any step failed, describe the error.
  </resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Smoke test → PostgreSQL | Integration test connects to live database (gated by DATABASE_URL) |
| Manual verification → Production-like setup | Full bootstrap flow on local machine |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg3-07 | Tampering | Smoke test server startup | mitigate | Uses `httptest.NewServer` — no OS-level port binding. Clean per-test isolation. |
| T-pg3-08 | Information Disclosure | Test credentials in smoke test | accept | Test uses ephemeral registration (unique email/org). No persistent seed credentials exposed. |
| T-pg3-09 | Spoofing | Manual verification flow | accept | Local-only flow. Seed credentials (`alex.rivera / demo123`) are dev-only, documented in AGENTS.md. |
</threat_model>

<verification>
```bash
# Automated smoke test (requires running PostgreSQL)
# DATABASE_URL=postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable \
#   go test ./cmd/server/... -run TestSmoke -v -count=1

# Build verification (no DB needed)
go vet ./cmd/server/...
go build ./cmd/server/...
```

Manual verification via D-15 flow (see Task 2).
</verification>

<success_criteria>
1. `cmd/server/main_test.go` exists with TestSmoke — auto-gated by DATABASE_URL
2. Smoke test compiles and passes `go vet`/`go build`
3. Manual verification flow (D-15) passes: docker → migrate → server → login → CRUD
4. No SurrealDB errors during server startup
5. Login with seed demo credentials succeeds
6. Authenticated API calls return data
</success_criteria>

<output>
After completion, create `.planning/phases/pg-3-wiring/pg-3-03-SUMMARY.md`
</output>
