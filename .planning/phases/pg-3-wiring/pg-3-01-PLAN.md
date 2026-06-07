---
phase: pg-3-wiring
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/auth/token_service.go
  - internal/auth/password_hasher.go
  - internal/middleware/cors.go
  - cmd/server/main.go
  - internal/adapters/primary/http/auth_test.go
autonomous: true
requirements: [D-01, D-02, D-03, D-16, D-17, D-18, D-23]
must_haves:
  truths:
    - auth.TokenService implements ports.TokenService by delegating to auth.Service
    - auth.PasswordHasher implements ports.PasswordHasher by wrapping auth.Service methods
    - middleware.CORS(allowedOrigins) returns func(http.Handler) http.Handler matching Logging/APIVersion signature
    - cmd/server/main.go constructs all 15+ postgres repos via postgres.New*(pool), not surrealdb.New*(sdb)
    - cmd/server/main.go no longer imports surrealdb, calls db.NewSurrealDB(), or prints SURREALDB_* warning
    - cmd/server/main.go uses middleware.CORS() instead of inline corsMiddleware closure
    - auth_test.go uses postgres adapters + exported_test_helpers, not surrealdb or sdb.DB
    - auth_test.go last test (TestLogin_DeactivatedAccount) uses raw SQL via pool.Exec instead of sdb.Query
    - go build ./cmd/server passes with zero surrealdb imports
  artifacts:
    - path: internal/auth/token_service.go
      provides: ports.TokenService adapter wrapping auth.Service
      min_lines: 30
      exports: ["NewTokenService", "TokenService"]
    - path: internal/auth/password_hasher.go
      provides: ports.PasswordHasher adapter wrapping auth.Service / bcrypt
      min_lines: 15
      exports: ["NewPasswordHasher", "PasswordHasher"]
    - path: internal/middleware/cors.go
      provides: CORS middleware matching existing Logging/APIVersion signature
      min_lines: 25
      exports: ["CORS"]
    - path: cmd/server/main.go
      provides: Postgres-wired server entry point
      contains_pattern: "postgres\\.New"
      missing_pattern: "surrealdb\\.New|sdbConn|middleware\\.CORS"
    - path: internal/adapters/primary/http/auth_test.go
      provides: Auth handler tests using postgres adapters
      missing_pattern: "surrealdb|hexauth"
  key_links:
    - from: internal/auth/token_service.go
      to: internal/auth/auth.go
      via: delegates GenerateToken/ValidateToken/GenerateRefreshToken, calls auth.HashRefreshToken
    - from: internal/auth/password_hasher.go
      to: internal/auth/auth.go
      via: delegates HashPassword/CheckPassword
    - from: internal/middleware/cors.go
      to: internal/middleware/version.go
      via: same func signature func(http.Handler) http.Handler
    - from: cmd/server/main.go
      to: internal/db/pgpool.go
      via: db.NewPool() for *pgxpool.Pool singleton
    - from: cmd/server/main.go
      to: internal/adapters/secondary/postgres/*.go
      via: postgres.New*(pool) constructor calls
    - from: internal/adapters/primary/http/auth_test.go
      to: internal/adapters/secondary/postgres/exported_test_helpers.go
      via: postgres.TestPool(), SetupTestSchema(), TeardownTestSchema()
---

<objective>
Replace all SurrealDB repository wiring in `cmd/server/main.go` with Postgres equivalents, create the two missing adapter files (TokenService, PasswordHasher), extract CORS middleware, and rewrite the auth handler test to use Postgres adapters.

**Purpose:** This is the critical "flip the switch" plan — after this, the server compiles and runs using only PostgreSQL. SurrealDB adapters are no longer imported.

**Output:**
- `internal/auth/token_service.go` — `ports.TokenService` adapter
- `internal/auth/password_hasher.go` — `ports.PasswordHasher` adapter
- `internal/middleware/cors.go` — CORS middleware extracted from main.go
- `cmd/server/main.go` — Rewired with 15+ Postgres repo constructors, no SurrealDB
- `internal/adapters/primary/http/auth_test.go` — Rewritten to use Postgres test helpers
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-3-wiring/pg-3-RESEARCH.md (full constructor mapping, code examples, execution order)

# Source files to modify
@cmd/server/main.go (current SurrealDB wiring — 255 lines)
@internal/adapters/primary/http/auth_test.go (749-line test importing surrealdb — must rewrite)

# Port interfaces the adapters must implement
@internal/core/ports/token_service.go (TokenService: 4 methods)
@internal/core/ports/password_hasher.go (PasswordHasher: 2 methods)

# SurrealDB adapters (reference implementations to replicate in internal/auth/)
@internal/adapters/secondary/surrealdb/token_service.go (wraps auth.Service)
@internal/adapters/secondary/surrealdb/password_hasher.go (wraps bcrypt — 24 lines)

# Service wiring patterns
@internal/core/services/auth/auth.go (NewService signature: userRepo, orgRepo, tokenService, hasher, refreshTokenRepo)
@internal/core/services/password_reset/password_reset.go (NewService expects PasswordHasher + TokenService)

# Test infrastructure to reuse
@internal/adapters/secondary/postgres/exported_test_helpers.go (TestPool, SetupTestSchema, TeardownTestSchema, seed helpers)

# Existing CORS pattern to match
@internal/middleware/version.go (APIVersion — func(http.Handler) http.Handler signature)
@internal/middleware/ratelimit.go (RateLimiter.Middleware — method-based)

<interfaces>
From internal/core/ports/token_service.go:
```go
type TokenService interface {
    GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error)
    ValidateToken(tokenString string) (*Claims, error)
    GenerateRefreshToken() (string, error)
    HashRefreshToken(token string) string
}

type Claims struct {
    UserID, OrganizationID uuid.UUID
    Role, Email string
    ExpiresAt time.Time
}
```

From internal/core/ports/password_hasher.go:
```go
type PasswordHasher interface {
    Hash(password string) (string, error)
    Check(password, hash string) bool
}
```

From internal/auth/auth.go (the underlying service to wrap):
```go
type Service struct { secretKey []byte }
func NewService(secretKey string) *Service
func (s *Service) HashPassword(password string) (string, error)
func (s *Service) CheckPassword(password, hash string) bool
func (s *Service) GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error)
func (s *Service) ValidateToken(tokenString string) (*Claims, error)
func (s *Service) GenerateRefreshToken() (string, error)
func HashRefreshToken(token string) string // static func
```

From internal/adapters/secondary/surrealdb/token_service.go (reference):
```go
type TokenService struct { authService *auth.Service }
func NewTokenService(authService *auth.Service) *TokenService
func (s *TokenService) GenerateToken(...) (string, error)  // delegates to s.authService
func (s *TokenService) ValidateToken(tokenString string) (*ports.Claims, error) // wraps auth.Claims→ports.Claims
func (s *TokenService) GenerateRefreshToken() (string, error) // delegates
func (s *TokenService) HashRefreshToken(token string) string   // calls auth.HashRefreshToken(token)
```

From internal/adapters/secondary/surrealdb/password_hasher.go (reference):
```go
type PasswordHasher struct{}
func NewPasswordHasher() *PasswordHasher
func (h *PasswordHasher) Hash(password string) (string, error)  // bcrypt.GenerateFromPassword
func (h *PasswordHasher) Check(password, hash string) bool      // bcrypt.CompareHashAndPassword
```

From internal/middleware/version.go (pattern to match for CORS):
```go
func APIVersion(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ... })
}
```

From internal/adapters/secondary/postgres/exported_test_helpers.go:
```go
func TestPool(t *testing.T) *pgxpool.Pool  // skips if DATABASE_URL not set
func SetupTestSchema(t *testing.T, pool *pgxpool.Pool)   // applies non-seed migrations
func TeardownTestSchema(t *testing.T, pool *pgxpool.Pool) // drops all tables
func seedOrg(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID
func seedUser(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID
// Also: uniqueEmail(), uniqueUsername(), uniqueCode()
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create TokenService + PasswordHasher adapter files</name>

  <files>
    internal/auth/token_service.go
    internal/auth/password_hasher.go
  </files>

  <read_first>
    @internal/core/ports/token_service.go (interface + Claims struct)
    @internal/core/ports/password_hasher.go (interface)
    @internal/adapters/secondary/surrealdb/token_service.go (41-line reference — same implementation, new package)
    @internal/adapters/secondary/surrealdb/password_hasher.go (24-line reference — same implementation, new package)
    @internal/auth/auth.go (Service methods to delegate to: HashPassword, CheckPassword, GenerateToken, ValidateToken, GenerateRefreshToken; static func HashRefreshToken)
  </read_first>

  <action>
    **A) `internal/auth/token_service.go`** — package `auth`, same adapter as surrealdb/token_service.go but in the auth package.

    Struct `TokenService` with field `authService *auth.Service`. Package-local import of `auth` is not needed since we ARE the auth package — just use `Service` directly.

    Constructor `func NewTokenService(svc *Service) *TokenService`.

    Four methods delegating to the auth.Service:
    - `GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error)` → `s.svc.GenerateToken(...)`
    - `ValidateToken(tokenString string) (*ports.Claims, error)` → call `s.svc.ValidateToken(tokenString)`, wrap `*auth.Claims` into `*ports.Claims` mapping `claims.ExpiresAt.Time` → `claims.ExpiresAt`. Import `ports` package.
    - `GenerateRefreshToken() (string, error)` → `s.svc.GenerateRefreshToken()`
    - `HashRefreshToken(token string) string` → `HashRefreshToken(token)` (calls auth's static func)

    Imports needed: `github.com/google/uuid`, `github.com/stefanoprivitera/hourglass/internal/core/ports`

    **B) `internal/auth/password_hasher.go`** — package `auth`.

    Struct `PasswordHasher{}` — no fields needed since it wraps bcrypt directly.

    Constructor `func NewPasswordHasher() *PasswordHasher`.

    Two methods using direct bcrypt calls (same as surrealdb/password_hasher.go):
    - `Hash(password string) (string, error)` → `bcrypt.GenerateFromPassword([]byte(password), 12)`
    - `Check(password, hash string) bool` → `bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil`

    Import: `golang.org/x/crypto/bcrypt`

    No `auth.Service` dependency — this is a standalone bcrypt wrapper.
  </action>

  <verify>
    <automated>go vet ./internal/auth/ && go build ./internal/auth/</automated>
  </verify>

  <done>
    1. `internal/auth/token_service.go` compiles with 4 methods implementing ports.TokenService
    2. `internal/auth/password_hasher.go` compiles with 2 methods implementing ports.PasswordHasher
    3. `go vet ./internal/auth/` passes (no unused imports, no type errors)
    4. Both files exist with exported types and constructors
  </done>
</task>

<task type="auto">
  <name>Task 2: Create CORS middleware + rewrite cmd/server/main.go</name>

  <files>
    internal/middleware/cors.go
    cmd/server/main.go
  </files>

  <read_first>
    @internal/middleware/version.go (APIVersion — signature: func(http.Handler) http.Handler)
    @cmd/server/main.go (current state — 255 lines, surrealdb + inline corsMiddleware)
    @internal/db/pgpool.go (pool singleton — already called in main.go)
  </read_first>

  <action>
    **A) `internal/middleware/cors.go`** — package `middleware`.

    Function `func CORS(allowedOrigins []string) func(http.Handler) http.Handler`.

    Exact same logic as the current inline `corsMiddleware` in main.go (lines 228-254):
    - Get `Origin` from request header
    - Loop through `allowedOrigins`, check if origin matches (`o == origin || o == "*"`)
    - If allowed, set headers: `Access-Control-Allow-Origin`, `Access-Control-Allow-Credentials: true`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, `Access-Control-Allow-Headers: Content-Type`
    - If `r.Method == "OPTIONS"`, write 200 and return
    - Otherwise call `next.ServeHTTP(w, r)`

    Import `net/http` and `strings`. Signature matches `func(http.Handler) http.Handler` like `middleware.APIVersion`, `middleware.Logging`.

    **B) `cmd/server/main.go`** — rewrite the SurrealDB wiring to Postgres. Per D-16 (one-shot) and D-17 (match patterns exactly).

    Changes:

    1. **Imports** — remove `"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"`, add `"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"`. Keep all 10 service imports. Keep `internal/db`, `internal/auth`, all handlers, middleware.

    2. **Remove SurrealDB init block** (lines 47-52):
       Remove `sdbConn, err := db.NewSurrealDB()`, the fatal check, `defer sdbConn.Close()`, and `log.Println("Using SurrealDB")`.

    3. **Remove SURREALDB_* warning** (lines 54-56):
       Remove the `if os.Getenv("SURREALDB_URL")` block entirely (D-18).

    4. **Fix pool variable usage**: The existing `db.NewPool()` call (line 41) is already called and assigned to `_`. Change to `pool, err := db.NewPool()` and use `pool` for all repo constructors. Keep the fatal check and defer.

    5. **Replace all 15 repo constructors** using the mapping from RESEARCH.md:

       | Old (SurrealDB) | New (Postgres) |
       |---|---|
       | `surrealdb.NewTimeEntryRepository(sdbConn.DB())` | `postgres.NewTimeEntryRepository(pool)` |
       | `surrealdb.NewAuditLogRepository(sdbConn.DB())` | `postgres.NewAuditLogRepository(pool)` |
       | `surrealdb.NewUserRepository(sdbConn.DB())` | `postgres.NewUserRepository(pool)` |
       | `surrealdb.NewPasswordHasher()` | `auth.NewPasswordHasher()` |
       | `surrealdb.NewTokenService(authService)` | `auth.NewTokenService(authService)` |
       | `surrealdb.NewRefreshTokenRepository(sdbConn.DB())` | `postgres.NewRefreshTokenRepository(pool)` |
       | `surrealdb.NewInvitationRepository(sdbConn.DB())` | `postgres.NewInvitationRepository(pool)` |
       | `surrealdb.NewPasswordResetRepository(sdbConn.DB())` | `postgres.NewPasswordResetRepository(pool)` |
       | `surrealdb.NewUserFinder(sdbConn.DB())` | `postgres.NewUserFinder(pool)` |
       | `surrealdb.NewUnitRepository(sdbConn.DB())` | `postgres.NewUnitRepository(pool)` |
       | `surrealdb.NewWorkingGroupRepository(sdbConn.DB())` | `postgres.NewWorkingGroupRepository(pool)` |
       | `surrealdb.NewCustomerRepository(sdbConn.DB())` | `postgres.NewCustomerRepository(pool)` |
       | `surrealdb.NewOrganizationManagementRepository(sdbConn.DB())` | `postgres.NewOrganizationManagementRepository(pool)` |
       | `surrealdb.NewProjectRepository(sdbConn.DB())` | `postgres.NewProjectRepository(pool)` |
       | `surrealdb.NewContractRepository(sdbConn.DB())` | `postgres.NewContractRepository(pool)` |
       | `surrealdb.NewExportRepository(sdbConn.DB())` | `postgres.NewExportRepository(pool)` |

       Also on line 101: `surrealdb.NewTokenService(authService)` → `auth.NewTokenService(authService)` (used inline in passwordResetService constructor).

    6. **Inline corsMiddleware** — replace the local function definition (lines 228-254) with import of `middleware.CORS`:
       - Change `corsMiddleware(allowedOrigins)(mux)` to `middleware.CORS(allowedOrigins)(mux)` in the handler chain (line 222).

    7. **Pool lifecycle**: The pool already has `db.ClosePool()` deferred on line 44. Keep as-is.

    **Keep ALL route registrations and handler wiring unchanged** (lines 62-204). Only the constructor calls and the middleware import change.

    The final handler chain stays: `rateLimiter.Middleware(middleware.Logging(middleware.APIVersion(middleware.CORS(allowedOrigins)(mux))))`
  </action>

  <verify>
    <automated>go vet ./cmd/server/ && go build ./cmd/server/</automated>
  </verify>

  <done>
    1. `internal/middleware/cors.go` exports `CORS` with same logic as removed inline corsMiddleware
    2. `cmd/server/main.go` no longer imports surrealdb package
    3. `cmd/server/main.go` no longer calls `db.NewSurrealDB()` or checks `SURREALDB_URL`
    4. All 16 constructor calls use postgres or auth package (zero surrealdb.New* calls)
    5. Handler chain uses `middleware.CORS(allowedOrigins)` not `corsMiddleware(allowedOrigins)`
    6. `go build ./cmd/server/` compiles successfully
    7. `go vet ./cmd/server/` passes
  </done>
</task>

<task type="auto">
  <name>Task 3: Rewrite auth_test.go to use PostgreSQL adapters</name>

  <files>
    internal/adapters/primary/http/auth_test.go
  </files>

  <read_first>
    @internal/adapters/primary/http/auth_test.go (current 749-line file — imports surrealdb, uses GetTestDBWithNamespace)
    @internal/adapters/secondary/postgres/exported_test_helpers.go (TestPool, SetupTestSchema, TeardownTestSchema, seedOrg, seedUser functions)
    @internal/adapters/secondary/surrealdb/token_service.go (to replicate in auth package)
    @internal/adapters/secondary/surrealdb/password_hasher.go (to replicate in auth package)
  </read_first>

  <action>
    Rewrite `internal/adapters/primary/http/auth_test.go` to use PostgreSQL adapters instead of SurrealDB.

    **Key changes:**

    1. **Imports**: Remove `hexauth "github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"` and `sdb "github.com/surrealdb/surrealdb.go"`. Add `"github.com/jackc/pgx/v5/pgxpool"` and `"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"`.

    2. **testServer struct**: Replace `db *sdb.DB` with `pool *pgxpool.Pool`. Remove `"github.com/surrealdb/surrealdb.go"` based import.

    3. **setupTestServer function** — replace SurrealDB setup with PostgreSQL:
       - Remove `hexauth.GetTestDBWithNamespace(ns, ns)` and `db.Close()`
       - Add: `pool := postgres.TestPool(t)`
       - Add: `postgres.SetupTestSchema(t, pool)`
       - Add: `t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })`
       - Replace all `hexauth.New*(db)` with `postgres.New*(pool)`:
         - `hexauth.NewUserRepository(db)` → `postgres.NewUserRepository(pool)`
         - `hexauth.NewOrganizationRepository(db)` → `postgres.NewOrganizationRepository(pool)`
         - `hexauth.NewPasswordHasher()` → `auth.NewPasswordHasher()`
         - `hexauth.NewTokenService(authService)` → `auth.NewTokenService(authService)`
         - `hexauth.NewRefreshTokenRepository(db)` → `postgres.NewRefreshTokenRepository(pool)`
         - `hexauth.NewInvitationRepository(db)` → `postgres.NewInvitationRepository(pool)`
       - Keep all service and handler constructors unchanged
       - Update `&testServer{...}` to set `pool: pool` instead of `db: db`

    4. **TestLogin_DeactivatedAccount test** (lines 709-748) — replace the SurrealDB query that deactivates a user with a raw SQL via pool.Exec:
       ```go
       _, err = ts.pool.Exec(context.Background(),
           "UPDATE users SET is_active = false WHERE email = $1", email)
       ```
       Remove the `sdb.Query[any](...)` call. The `context` import is already present.

    5. **Preserve** all test functions, helper functions (uniqueID, uniqueEmail, uniqueOrgName, post, get), and assertion logic exactly as-is.

    6. **Imports cleanup**: After changes, run `go vet` to ensure no unused imports remain.
  </action>

  <verify>
    <automated>go vet ./internal/adapters/primary/http/ && go build ./internal/adapters/primary/http/</automated>
  </verify>

  <done>
    1. `auth_test.go` no longer imports surrealdb or hexauth package
    2. `setupTestServer` uses `postgres.TestPool()`, `SetupTestSchema()`, `TeardownTestSchema()`
    3. All surrealdb repo constructors replaced with postgres + auth equivalents
    4. `TestLogin_DeactivatedAccount` uses `ts.pool.Exec()` raw SQL instead of `sdb.Query()`
    5. `go vet ./internal/adapters/primary/http/` passes with zero surrealdb references
    6. All 13 existing test functions preserved with identical assertion logic
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| cmd/server/main.go → PostgreSQL repos | Repository wiring and startup path |
| CORS middleware → HTTP handler chain | Request filtering at middleware boundary |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg3-01 | Tampering | main.go repo wiring | mitigate | Pool is obtained from db.NewPool() — controlled initialization path. No user input in constructor wiring. |
| T-pg3-02 | Spoofing | CORS middleware refactor | mitigate | Same logic as removed inline closure — no behavioral change. Headers set only for allowed origins. |
| T-pg3-03 | Information Disclosure | auth_test.go credentials | accept | Test file uses fixed test JWT secret, never production secrets. Test skips if DATABASE_URL not set. |
</threat_model>

<verification>
```bash
# Build verification
go vet ./internal/auth/
go build ./internal/auth/
go vet ./internal/middleware/
go build ./internal/middleware/
go vet ./cmd/server/
go build ./cmd/server/
go vet ./internal/adapters/primary/http/
go build ./internal/adapters/primary/http/

# Full package compilation (should compile regardless of surrealdb presence)
go build ./...

# Grep verification — zero surrealdb imports outside surrealdb/ itself
grep -rn '"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"' cmd/ internal/ --include='*.go'
# Should return zero matches

# Grep verification — no NewSurrealDB call in main.go
grep -n "NewSurrealDB" cmd/server/main.go
# Should return zero matches
</verification>

<success_criteria>
1. `go build ./cmd/server/` compiles successfully with zero SurrealDB imports
2. `go build ./...` compiles (may still try to build surrealdb/ package which exists until Plan 2)
3. `internal/auth/token_service.go` and `internal/auth/password_hasher.go` exist and compile
4. `internal/middleware/cors.go` exists with `CORS` exported function
5. `cmd/server/main.go` uses `postgres.New*(pool)` for all 15+ repo constructors
6. `cmd/server/main.go` uses `middleware.CORS()` not inline `corsMiddleware`
7. `auth_test.go` uses `postgres.TestPool()` setup and zero surrealdb references
8. All 13 auth test functions preserved in rewritten auth_test.go
</success_criteria>

<output>
After completion, create `.planning/phases/pg-3-wiring/pg-3-01-SUMMARY.md`
</output>
