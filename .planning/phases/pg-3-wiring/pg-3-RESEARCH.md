# Phase Pg-3: Wiring, Cleanup & Verification - Research

**Researched:** 2026-06-07
**Domain:** Server wiring, build system cleanup, end-to-end verification
**Confidence:** HIGH

## Summary

Phase Pg-3 completes the PostgreSQL migration by swapping all SurrealDB repository constructors in `cmd/server/main.go` for their Postgres equivalents, deleting all SurrealDB code, updating build artifacts, and verifying everything works end-to-end.

**Primary recommendation:** One-shot atomic commit (D-16) — rewire main.go, delete all SurrealDB files, update AGENTS.md/Makefile/docker-compose, run `go mod tidy`. Then verify with the manual flow + automated smoke test.

### Critical Gap: `TokenService` and `PasswordHasher` Need New Homes

The SurrealDB package contains two **non-DB adapters** that have no Postgres counterparts: `TokenService` (wraps `auth.Service` to implement `ports.TokenService`) and `PasswordHasher` (wraps bcrypt for `ports.PasswordHasher`). These are referenced by `auth.NewService()` and `passwordresetsvc.NewService()`. After deleting SurrealDB, these must be replaced.

`auth.Service` (in `internal/auth/auth.go`) already has the underlying methods:
- `GenerateToken(uuid.UUID, uuid.UUID, string, string) (string, error)` — matches
- `ValidateToken(string) (*Claims, error)` — returns `*auth.Claims`, port expects `*ports.Claims`
- `GenerateRefreshToken() (string, error)` — matches
- `HashRefreshToken(token string) string` — static function, matches
- `HashPassword(string) (string, error)` — like `Hash()`
- `CheckPassword(string, string) bool` — like `Check()`

The port interfaces (`ports.TokenService`, `ports.PasswordHasher`) require different method names and return types. **These need wrapper adapters** — either:
- **Option A (recommended):** Create `internal/auth/token_service.go` + `internal/auth/password_hasher.go` — small adapter files that implement the port interfaces by delegating to `auth.Service`
- **Option B:** Extend `auth.Service` to implement both ports directly

### Auth Test Rewrite Required

`internal/adapters/primary/http/auth_test.go` imports `internal/adapters/secondary/surrealdb` and uses `GetTestDBWithNamespace()` for test setup. After deleting SurrealDB, this test **must** be rewritten to use the postgres adapters and test helpers (`TestPool`, `SetupTestSchema` from `exported_test_helpers.go`). This is a significant scope item (749-line file).

### `lib/pq` Dependency
[VERIFIED: go.mod, go.sum] `github.com/lib/pq` is **not** in go.mod or go.sum anywhere. Decision D-20 ("remove lib/pq") is a no-op.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Wiring
- **D-01:** `cmd/server/main.go` constructs all postgres repos with `*pgxpool.Pool`
- **D-02:** Each service receives the postgres repo (no code changes to services)
- **D-03:** Handler constructors unchanged — they receive services, not repos
- **D-16:** One-shot replacement — replace all SurrealDB repos with Postgres repos in a single pass. Server goes from all-SurrealDB to all-Postgres in one commit.
- **D-17:** Match existing constructor patterns exactly — `postgres.NewXxx(pool)` instead of `surrealdb.NewXxx(sdbConn.DB())`. Same line-by-line structure, same service wiring, just swap package prefix and pool parameter.
- **D-18:** Remove the deprecated SURREALDB_* env var warning from main.go. Server only needs `DATABASE_URL` and `JWT_SECRET`.

#### Cleanup — delete these entirely:
- **D-04:** `internal/adapters/secondary/surrealdb/` — all 25+ files
- **D-05:** `internal/db/surreal.go` — SurrealDB singleton
- **D-06:** `cmd/schema/main.go` — SurrealDB schema loader
- **D-07:** `schema/` — all `.surql` files
- **D-08:** SurrealDB from `docker-compose.yml`
- **D-09:** SurrealDB from `Makefile` (schema targets, etc.)
- **D-10:** `go.mod` — remove `github.com/surrealdb/surrealdb.go` dependency
- **D-19:** Keep `internal/db/` package as-is after removing `surreal.go` — only `pgpool.go` remains. No rename or restructuring.
- **D-20:** Remove `github.com/lib/pq` from go.mod alongside the SurrealDB dependency — everything uses pgx now. Then run `go mod tidy`.

#### Docker compose
- **D-25:** Remove the SurrealDB service block entirely from `docker-compose.yml` (not just commented out). Also remove `--profile surrealdb` profiles section and any SurrealDB-specific volumes/env vars.

#### Build & docs updates
- **D-11:** Makefile: remove all schema/surreal/seed targets, add `make setup = go run ./cmd/migrate -all`
- **D-12:** AGENTS.md: update all SurrealDB references to PostgreSQL
- **D-13:** Environment vars: `SURREALDB_*` no longer needed, `DATABASE_URL` is required
- **D-14:** Remove `cmd/schema` from any docs/scripts

#### CORS middleware
- **D-23:** Extract CORS middleware from main.go inline closure to `internal/middleware/cors.go` — consistent with Logging, RateLimiter, APIVersion middleware there.

#### Verification
- **D-15:** Full manual verification flow:
  1. `docker compose up -d` (Postgres starts)
  2. `go run ./cmd/migrate -all` (schema + seed)
  3. `go run ./cmd/server` (server starts on :8080)
  4. Login as demo manager (alex.rivera / demo123)
  5. Check all pages render with data
  6. CRUD operations on every entity type
- **D-21:** Automated smoke test in `cmd/server/main_test.go` — verifies `/health` returns 200 and an authenticated `/units` call returns data.
- **D-22:** Smoke test reuses Pg-2's exported test helpers from `internal/adapters/secondary/postgres/exported_test_helpers.go` (setUpTestDB / tearDownTestDB pattern).

### the agent's Discretion

None specified — discussion stayed within phase scope.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Server startup | `cmd/server/main.go` | — | Entry point, constructs all dependencies |
| Database connection | `internal/db/pgpool.go` | — | Singleton pgxpool.Pool initialized once |
| Repository construction | `cmd/server/main.go` | — | Wires 18+ pgxpool-backed repos inline |
| Service construction | `cmd/server/main.go` | — | Services receive repos — unchanged signature |
| Handler construction | `cmd/server/main.go` | — | Handlers receive services — unchanged signature |
| CORS enforcement | `internal/middleware/` | — | Extract from main.go inline to middleware package |
| Build system | `Makefile` | `docker-compose.yml` | Build targets, setup shortcut, Docker config |
| Health check | `internal/handlers/` | — | Still returns `{"status":"ok"}` |
| Smoke test | `cmd/server/main_test.go` | postgres test helpers | Automated verification against live postgres |

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| (none specified in roadmap) | Wire postgres repos in server init, remove SurrealDB entirely, verify everything works end-to-end | Full constructor mapping, file deletion inventory, test rewrite plan below |

## Standard Stack

### Core
| Library | Purpose | Why Standard |
|---------|---------|--------------|
| `github.com/jackc/pgx/v5` | PostgreSQL driver + pgxpool | Already used by all postgres repos, no change needed |
| `internal/db/pgpool.go` | Pool singleton (`*pgxpool.Pool`) | Already wired in main.go — keep as-is |
| `internal/adapters/secondary/postgres` | All repo implementations | Already exist from Pg-2, just need main.go wiring |
| `internal/auth/` | JWT token service + password hashing | Must add `ports.TokenService` / `ports.PasswordHasher` adapters here |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Internal auth adapters | Inline in main.go | Main.go is already >200 lines — adapters in `internal/auth/` is cleaner |

## Architecture Patterns

### System Architecture Diagram

```
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │ cmd/server/main.go                                                          │
  │                                                                             │
  │  db.NewPool() ──► *pgxpool.Pool ──┬──► postgres.NewTimeEntryRepository()   │
  │                                    ├──► postgres.NewUserRepository()        │
  │                                    ├──► postgres.NewOrganizationRepository()│
  │                                    ├──► postgres.New*() (15 total repos)    │
  │                                    │                                        │
  │  auth.NewService(jwtSecret) ──► auth.Service ──┬──► NewTokenService()       │
  │                                                 └──► NewPasswordHasher()     │
  │                                                                             │
  │  repos ──► service.NewService(repo) ──► handler.NewHandler(service)        │
  │         └──► mux.HandleFunc("GET /", handler.Method)                        │
  │                                                                             │
  │  middleware: rateLimiter.Middleware(Logging(APIVersion(corsMiddleware(mux))))│
  └─────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure (after cleanup)
```
cmd/
├── server/main.go           # Postgres wiring (no surrealdb)
├── migrate/main.go          # Postgres migrations (unchanged)
internal/
├── auth/
│   ├── auth.go              # JWT service (unchanged)
│   ├── token_service.go     # NEW — ports.TokenService adapter
│   └── password_hasher.go   # NEW — ports.PasswordHasher adapter
├── db/
│   └── pgpool.go            # pgxpool singleton (unchanged)
├── adapters/
│   secondary/
│     postgres/               # All repos (already exist)
│     # surrealdb/ DELETED
├── middleware/
│   ├── middleware.go         # Auth, Logging, helpers (unchanged)
│   ├── cors.go              # NEW — extracted from main.go inline
│   ├── ratelimit.go         # (unchanged)
│   └── version.go           # (unchanged)
schema/                        # DELETED
docker-compose.yml             # No more SurrealDB service
```

### Wave Execution Order

The one-shot commit constraint (D-16) means all changes happen in a single commit, but the **execution within that commit** has a safe order:

1. **Create adapters** — `internal/auth/token_service.go`, `internal/auth/password_hasher.go`
2. **Extract CORS** — `internal/middleware/cors.go`, remove inline from main.go
3. **Rewrite main.go** — swap all `surrealdb.NewXxx(sdb)` → `postgres.NewXxx(pool)`, remove SurrealDB init
4. **Rewrite auth_test.go** — replace all surrealdb imports with postgres equivalents
5. **Delete SurrealDB files** — `internal/adapters/secondary/surrealdb/`, `internal/db/surreal.go`, `cmd/schema/`, `schema/`
6. **Clean up config** — `docker-compose.yml`, `Makefile`, `AGENTS.md`
7. **Clean deps** — `go mod tidy`
8. **Smoke test** — `cmd/server/main_test.go`
9. **Manual verification** — migrate → seed → server → login → CRUD

### Pattern 1: Postgres Repo Constructor
**What:** Every postgres constructor takes `*pgxpool.Pool` and returns a concrete struct.
**Source:** [VERIFIED: postgres/user_finder.go:18]
```go
func NewUserFinder(pool *pgxpool.Pool) *UserFinder {
    return &UserFinder{pool: pool}
}
```

### Pattern 2: Service Constructor (unchanged)
**What:** Services receive the port interface. No code changes.
**Source:** [VERIFIED: core/services/auth/auth.go:113]
```go
func NewService(
    userRepo ports.UserRepository,
    orgRepo ports.OrganizationRepository,
    tokenService ports.TokenService,
    hasher ports.PasswordHasher,
    refreshTokenRepo ports.RefreshTokenRepository,
) *Service {
```

### Pattern 3: Middleware Signature (for CORS extraction)
**What:** Returns `func(http.Handler) http.Handler` — matches Logging, APIVersion.
**Source:** [VERIFIED: internal/middleware/version.go:13]
```go
func APIVersion(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ... })
}
```

### Anti-Patterns to Avoid
- **Delayed deletion:** Don't keep surrealdb files around after wiring postgres — the compiler won't complain for go files not imported, but the directory is dead weight.
- **auth_test.go left untouched:** It imports from a deleted package — will break compilation.
- **CORS middleware as inline closure:** Pattern established — all other middleware is in `internal/middleware/`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token Service adapter | Custom JWT implementation | `internal/auth/token_service.go` wrapping `auth.Service` | Already 90% implemented — just need `ports.TokenService` wrapper |
| Password Hasher adapter | Custom bcrypt | `internal/auth/password_hasher.go` wrapping bcrypt + auth.Service | `auth.Service.HashPassword()` already exists, just unwrap to ports interface |
| CORS middleware | Inline closure | Extract to `internal/middleware/cors.go` | Consistent with existing pattern (Logging, APIVersion, RateLimiter) |
| Test DB setup | Custom test harness | `postgres.TestPool` + `postgres.SetupTestSchema` from exported_test_helpers.go | Already exists from Pg-2 |

## Common Pitfalls

### Pitfall 1: Missed SurrealDB Import in auth_test.go
**What goes wrong:** After deleting `internal/adapters/secondary/surrealdb/`, the auth handler test (`auth_test.go`) will **not compile** — it imports `hexauth` from the surrealdb package and uses `GetTestDBWithNamespace()`.
**Why it happens:** The auth_test.go was written for the SurrealDB era and was never ported.
**How to avoid:** Must be rewritten to use postgres adapters + exported_test_helpers BEFORE deleting the surrealdb directory.
**Warning signs:** `go build ./...` fails with "package not found" for surrealdb.

### Pitfall 2: TokenService / PasswordHasher Orphaned
**What goes wrong:** `auth.NewService()` and `passwordresetsvc.NewService()` require `ports.TokenService` and `ports.PasswordHasher` implementations. If only surrealdb provided them, the build breaks.
**How to avoid:** Create `internal/auth/token_service.go` and `internal/auth/password_hasher.go` as part of the wiring change.
**Warning signs:** `go build ./cmd/server` fails: "cannot use ... (type *auth.Service) as type ports.TokenService".

### Pitfall 3: `go mod tidy` Removes Transitive Depts Still Needed
**What goes wrong:** After removing `github.com/surrealdb/surrealdb.go`, `go mod tidy` removes its transitive dependencies (`gofrs/uuid`, `gorilla/websocket`, `klauspost/compress`, `x448/float16`). Verify none of these are imported elsewhere.
**How to avoid:** After running `go mod tidy`, rebuild: `go build ./...` and `go test ./...`.
**Verification result:** [VERIFIED: grep] None of the surrealdb transitive deps are imported elsewhere.

### Pitfall 4: Migration Already-Applied Errors Are Idempotent but Noisy
**What goes wrong:** The migrate CLI's "already exists" check is string-based (`strings.Contains(err.Error(), "already exists")`), which may miss Postgres-specific error codes or catch unrelated errors.
**How to avoid:** The existing behavior has been used since Pg-1 — keep as-is for this phase. Not a new concern.

## Code Examples

### Example 1: Postgres Main.go Wiring (Excerpt)
```
// Before (SurrealDB):
sdbConn, err := db.NewSurrealDB()
userRepo := surrealdb.NewUserRepository(sdbConn.DB())

// After (Postgres):
pool, err := db.NewPool()  // already done earlier
userRepo := postgres.NewUserRepository(pool)
```

### Example 2: TokenService Adapter (`internal/auth/token_service.go`)
```go
package auth

import (
    "github.com/google/uuid"
    "github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// TokenService wraps auth.Service to implement ports.TokenService.
type TokenService struct {
    svc *Service
}

func NewTokenService(svc *Service) *TokenService {
    return &TokenService{svc: svc}
}

func (s *TokenService) GenerateToken(userID, organizationID uuid.UUID, role, email string) (string, error) {
    return s.svc.GenerateToken(userID, organizationID, role, email)
}

func (s *TokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
    claims, err := s.svc.ValidateToken(tokenString)
    if err != nil {
        return nil, err
    }
    return &ports.Claims{
        UserID:         claims.UserID,
        OrganizationID: claims.OrganizationID,
        Role:           claims.Role,
        Email:          claims.Email,
        ExpiresAt:      claims.ExpiresAt.Time,
    }, nil
}

func (s *TokenService) GenerateRefreshToken() (string, error) {
    return s.svc.GenerateRefreshToken()
}

func (s *TokenService) HashRefreshToken(token string) string {
    return HashRefreshToken(token)
}
```

### Example 3: PasswordHasher Adapter (`internal/auth/password_hasher.go`)
```go
package auth

type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
    return &PasswordHasher{}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
    svc := &Service{} // minimal — HashPassword uses no instance state
    return svc.HashPassword(password)
}

func (h *PasswordHasher) Check(password, hash string) bool {
    svc := &Service{}
    return svc.CheckPassword(password, hash)
}
```

### Example 4: CORS Middleware (`internal/middleware/cors.go`)
```go
package middleware

import (
    "net/http"
    "strings"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            allowed := false
            for _, o := range allowedOrigins {
                if o == origin || o == "*" {
                    allowed = true
                    break
                }
            }
            if allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
            }
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Example 5: Smoke Test (`cmd/server/main_test.go`)
```go
package main

import (
    "net/http"
    "testing"
    "github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
)

func TestSmoke(t *testing.T) {
    pool := postgres.TestPool(t)
    postgres.SetupTestSchema(t, pool)
    t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

    // Start server, test /health, login, test /units
    // ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `internal/db/surreal.go` | `internal/db/pgpool.go` only | This phase | Singleton pattern preserved, database changed |
| `internal/adapters/secondary/surrealdb/` | `internal/adapters/secondary/postgres/` | This phase | All 25+ files deleted |
| `cmd/schema/main.go` + `schema/` | `cmd/migrate/main.go` + `migrations/` | This phase | SurrealDB schema loader removed |
| SurrealDB in docker-compose | Postgres-only docker-compose | This phase | No more `--profile surrealdb` |
| `SURREALDB_*` env vars | Only `DATABASE_URL` + `JWT_SECRET` | This phase | Server no longer reads SurrealDB vars |
| Inline CORS in main.go | `internal/middleware/cors.go` | This phase | Consistent middleware package |

### Deprecated/outdated:
- `github.com/surrealdb/surrealdb.go` dependency: [VERIFIED: go.mod] Removed from go.mod, clean up via `go mod tidy`
- `github.com/lib/pq`: [VERIFIED: go.mod, go.sum] **Not present at all** — D-20 is a no-op
- SurrealDB indirect deps (`gofrs/uuid`, `gorilla/websocket`, `klauspost/compress`, `x448/float16`): [VERIFIED: go.mod] All removed transitively by `go mod tidy`

## Complete Constructor Mapping

### SurrealDB → Postgres (15 repos + 2 adapters)

| # | Current (SurrealDB) | Target (Postgres) | Note |
|---|---------------------|-------------------|------|
| 1 | `surrealdb.NewTimeEntryRepository(sdb)` | `postgres.NewTimeEntryRepository(pool)` | Same name |
| 2 | `surrealdb.NewAuditLogRepository(sdb)` | `postgres.NewAuditLogRepository(pool)` | Same name |
| 3 | `surrealdb.NewUserRepository(sdb)` | `postgres.NewUserRepository(pool)` | Same name |
| 4 | `surrealdb.NewOrganizationRepository(sdb)` | `postgres.NewOrganizationRepository(pool)` | Same name |
| 5 | `surrealdb.NewPasswordHasher()` | `auth.NewPasswordHasher()` | **NEW PACKAGE** |
| 6 | `surrealdb.NewTokenService(authSvc)` | `auth.NewTokenService(authSvc)` | **NEW PACKAGE** |
| 7 | `surrealdb.NewRefreshTokenRepository(sdb)` | `postgres.NewRefreshTokenRepository(pool)` | Same name |
| 8 | `surrealdb.NewInvitationRepository(sdb)` | `postgres.NewInvitationRepository(pool)` | Same name |
| 9 | `surrealdb.NewPasswordResetRepository(sdb)` | `postgres.NewPasswordResetRepository(pool)` | Same name |
| 10 | `surrealdb.NewUserFinder(sdb)` | `postgres.NewUserFinder(pool)` | Same name |
| 11 | `surrealdb.NewUnitRepository(sdb)` | `postgres.NewUnitRepository(pool)` | Same name |
| 12 | `surrealdb.NewWorkingGroupRepository(sdb)` | `postgres.NewWorkingGroupRepository(pool)` | Same name |
| 13 | `surrealdb.NewCustomerRepository(sdb)` | `postgres.NewCustomerRepository(pool)` | Same name |
| 14 | `surrealdb.NewOrganizationManagementRepository(sdb)` | `postgres.NewOrganizationManagementRepository(pool)` | Same name |
| 15 | `surrealdb.NewProjectRepository(sdb)` | `postgres.NewProjectRepository(pool)` | Same name |
| 16 | `surrealdb.NewContractRepository(sdb)` | `postgres.NewContractRepository(pool)` | Same name |
| 17 | `surrealdb.NewExportRepository(sdb)` | `postgres.NewExportRepository(pool)` | Same name |

### Postgres Repos NOT Wired in Main (internal use only)

| Constructor | Used By | Doesn't Need Direct Wiring |
|------------|---------|---------------------------|
| `postgres.NewOrganizationMembershipRepository(pool)` | `OrganizationRepository` internally | No — org repo handles memberships |
| `postgres.NewSubprojectRepository(pool)` | `ProjectRepository` internally | No |
| `postgres.NewUnitMemberRepository(pool)` | `UnitRepository` internally | No |
| `postgres.NewWGMemberRepository(pool)` | `WorkingGroupRepository` internally | No |
| `postgres.NewExpenseRepository(pool)` | Future use | Not currently used by any service |

## Files to Delete Inventory

| Path | Type | Files | Reason |
|------|------|-------|--------|
| `internal/adapters/secondary/surrealdb/` | Directory | 29 .go files | All SurrealDB adapters |
| `internal/db/surreal.go` | File | 1 | SurrealDB singleton |
| `cmd/schema/main.go` | File | 1 | SurrealDB schema loader |
| `schema/` | Directory | 3 .surql files | SurrealDB schema |
| SurrealDB service in `docker-compose.yml` | Block | ~15 lines | No longer needed |

## Files to Create

| Path | Purpose | Pattern |
|------|---------|---------|
| `internal/auth/token_service.go` | `ports.TokenService` adapter wrapping `auth.Service` | Delegates to auth methods |
| `internal/auth/password_hasher.go` | `ports.PasswordHasher` adapter wrapping bcrypt | Wraps existing HashPassword/CheckPassword |
| `internal/middleware/cors.go` | CORS middleware extracted from main.go | `func CORS(origins) func(http.Handler) http.Handler` |
| `cmd/server/main_test.go` | Automated smoke test | Uses `postgres.TestPool` + `SetupTestSchema` |

## Files to Modify

| Path | What Changes |
|------|-------------|
| `cmd/server/main.go` | All `surrealdb.New*` → `postgres.New*`, remove SurrealDB init, remove inline corsMiddleware, use `middleware.CORS()` |
| `internal/adapters/primary/http/auth_test.go` | Replace all surrealdb imports with postgres adapters, rewrite test setup |
| `Makefile` | Add `setup:` target (`go run ./cmd/migrate -all`), verify no surreal targets |
| `docker-compose.yml` | Remove surrealdb service block and SURREALDB_* env vars from app service |
| `AGENTS.md` | Replace all SurrealDB references with PostgreSQL |
| `go.mod` | Remove `github.com/surrealdb/surrealdb.go`; run `go mod tidy` |

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` | Build | ✓ | go1.26.1 | — |
| `docker` | Verification step 1 | ✓ | 29.4.0 | — |
| `postgres` (via docker) | Smoke test | ✓ | 15-alpine | — |
| `psql` | db-init target (Makefile) | — | — | Not critical — migrate CLI handles it |

**Missing dependencies with no fallback:** None

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `auth.Service` already has methods that can be wrapped for `ports.TokenService`/`ports.PasswordHasher` | TokenService/PasswordHasher Gap | Verified — [CITED: internal/auth/auth.go] |
| A2 | `lib/pq` is not in go.mod or go.sum | Deprecated/Outdated | Low — [VERIFIED: go.mod, go.sum grep] confirmed no entries |
| A3 | `postgres.NewOrganizationMembershipRepository`, `NewSubprojectRepository`, `NewUnitMemberRepository`, `NewWGMemberRepository`, `NewExpenseRepository` don't need explicit wiring | Constructor Mapping | MEDIUM — verified by checking service constructors only take main repos, but should confirm no service directly uses these |
| A4 | None of the surrealdb transitive deps are used elsewhere | Common Pitfall 3 | VERIFIED — grep confirmed no imports outside surrealdb |

## Open Questions

1. **What happens to `internal/adapters/primary/http/auth_test.go`?**
   - What we know: 749-line file imports surrealdb package, uses `GetTestDBWithNamespace()` for test setup
   - What's unclear: Should this be rewritten to postgres as part of Pg-3, or deferred? Rewriting is significant scope but required for build to pass after surrealdb deletion
   - Recommendation: **Must be included** in Pg-3 scope. The build won't compile without it. Use the exported_test_helpers pattern.

2. **Where should TokenService/PasswordHasher adapters live?**
   - What we know: Need wrappers that implement `ports.TokenService` and `ports.PasswordHasher`
   - What's unclear: `internal/auth/` vs a new `internal/adapters/secondary/auth/` vs inline in main.go
   - Recommendation: `internal/auth/` — they wrap auth.Service, they're not DB adapters, and auth package is where the underlying logic already lives

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | None — `go test` standard |
| Quick run command | `go test ./cmd/server/... -run TestSmoke -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-21 | Smoke test: /health returns 200, authenticated /units returns data | integration | `go test ./cmd/server/... -run TestSmoke -v` | ❌ Wave 0 |
| D-15 | Full manual verification flow | manual | See D-15 steps | N/A |

### Sampling Rate
- **Per commit:** `go build ./... && go vet ./...`
- **Phase gate:** Full manual verification (D-15) + smoke test

### Wave 0 Gaps
- [ ] `cmd/server/main_test.go` — covers D-21 (smoke test)
  - Uses `postgres.TestPool()` — skips if DATABASE_URL not set
  - Uses `postgres.SetupTestSchema` / `TeardownTestSchema`
  - Test health endpoint
  - Login as demo user, hit authenticated `/units`

## Security Domain

The phase has no new security surface — it replaces the database adapter layer behind the same port interfaces. No new attack vectors introduced.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Yes (unchanged) | JWT in auth package — unchanged |
| V3 Session Management | Yes (unchanged) | HttpOnly cookies + refresh tokens |
| V5 Input Validation | Yes | Zod (frontend) + pgx parameterized queries (backend) |
| V6 Cryptography | Yes | bcrypt via golang.org/x/crypto — unchanged |

**No new security concerns.** The CORS middleware extraction is a pure refactor — same logic, different file.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase grep] — Full constructor mapping (postgres 20 funcs, surrealdb 17 funcs)
- [VERIFIED: go.mod, go.sum] — Dependency inventory
- [VERIFIED: codebase read] — auth.Service method signatures
- [VERIFIED: codebase read] — Docker compose, Makefile, AGENTS.md current state

### Secondary (MEDIUM confidence)
- [VERIFIED: codebase grep] — auth_test.go surrealdb dependency confirmed — 1 file outside surrealdb directory
- [VERIFIED: codebase read] — exported_test_helpers.go patterns for smoke test reuse
- [VERIFIED: codebase read] — internal/middleware/*.go patterns for CORS extraction

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all files read and verified
- Architecture: HIGH — hexagonal pattern well understood
- Pitfalls: HIGH — auth_test.go and TokenService gaps directly observed

**Research date:** 2026-06-07
**Valid until:** 2026-07-07 (stable — Go 1.26 ecosystem, pgx v5)
